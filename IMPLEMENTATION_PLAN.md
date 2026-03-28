# esphome-apiclient Implementation Plan

A phased approach to building the client, from bare TCP to a fully-featured library. Each phase builds on the previous one. Items marked with **🧪 TEST** are important to cover with tests.

### Phase 1 — Protobuf Code Generation & Message Type Registry

> Goal: Have compilable Go types for all messages and a mapping from type ID → proto message.

- [x] Set up `buf` or raw `protoc` toolchain to generate Go code from `api.proto` into `pb/`
- [x] Handle the custom `api_options.proto` options — you need a small proto file that declares the `id`, `source`, `no_delay`, etc. extensions so `protoc` can parse `api.proto`
- [x] Build a **message type registry**: a `map[uint32]func() proto.Message` that maps every `option (id)` value to a factory for the corresponding message type. This can be generated or hand-written
  - Example: `1 → func() proto.Message { return &pb.HelloRequest{} }`
  - **🧪 TEST:** Verify every message ID in the registry round-trips: create → marshal → unmarshal → compare

### Phase 2 — Frame Codec

> Goal: Read and write the ESPHome binary frame format over any `io.ReadWriter`.

- [x] Implement `WriteFrame(w io.Writer, msgType uint32, data []byte) error`
  - Write `0x00`, encode `len(data)` as varint, encode `msgType` as varint, write `data`
- [x] Implement `ReadFrame(r io.Reader) (msgType uint32, data []byte, err error)`
  - Read and validate the leading `0x00` byte
  - Decode size varint, decode type varint, read exactly `size` bytes
- [x] Handle edge cases: zero-length payloads, maximum message size guard (prevent OOM from corrupt frames)
- **🧪 TEST: Frame round-trip** — encode a frame, decode it, assert type and payload match
- **🧪 TEST: Partial reads** — use `iotest.HalfReader` or a drip-feed reader to ensure the decoder handles partial TCP reads correctly
- **🧪 TEST: Invalid frames** — missing zero byte, truncated varint, size exceeding limit → expect clean errors
- **🧪 TEST: Multiple frames** — write N frames to a buffer, read them back in order

### Phase 3 — Plain TCP Transport

> Goal: Dial a TCP connection and send/receive protobuf messages.

- [x] Implement `PlainTransport` wrapping `net.Conn`
- [x] `Dial(address string, timeout time.Duration) (*PlainTransport, error)` — TCP connect with deadline
- [x] Add read/write deadlines support (configurable per-operation or global)
- [x] Wire the codec on top: `SendMessage(msg proto.Message, msgType uint32)` and `RecvMessage() (proto.Message, error)` using the registry + codec
- **🧪 TEST:** Use `net.Pipe()` to create an in-memory connection pair. Send a `HelloRequest` on one end, decode it on the other, verify fields
- **🧪 TEST:** Connection timeout — dial a non-routable address, assert it fails within the deadline

### Phase 4 — Connection Handshake

> Goal: Implement the Hello exchange and connection setup.

- [x] Send `HelloRequest` with `client_info`, `api_version_major=1`, `api_version_minor=10` (or latest)
- [x] Receive and parse `HelloResponse`, extract `api_version_major`, `api_version_minor`, `server_info`, `name`
- [x] **Version validation**: if `api_version_major` doesn't match, return an error and close (no `DisconnectRequest`)
- [x] Store negotiated API version on the client for feature-gating later
- **🧪 TEST: Successful handshake** — mock server sends a valid `HelloResponse`, verify client extracts version and name
- **🧪 TEST: Version mismatch** — mock server returns `api_version_major=99`, verify client returns error and closes connection
- **🧪 TEST: Server sends unexpected message** — send a `PingRequest` instead of `HelloResponse`, verify client errors

### Phase 5 — Read Loop & Message Router

> Goal: A background goroutine that continuously reads frames and dispatches them.

- [x] Start a goroutine after handshake that calls `ReadFrame` in a loop
- [x] Look up the message type in the registry, unmarshal, dispatch to registered handlers
- [x] Support registering multiple handlers per message type (fan-out)
- [x] Handle unknown message types gracefully (log + skip, don't crash)
- [x] Clean shutdown: close the transport → read loop gets EOF → exits goroutine
- [x] Expose a `Done() <-chan struct{}` or similar to detect when the read loop exits
- **🧪 TEST: Dispatch** — register a handler for message type X, feed a frame of type X into the pipe, assert handler fires with correct message
- **🧪 TEST: Fan-out** — register two handlers for the same type, assert both fire
- **🧪 TEST: Unknown type** — feed an unregistered message type, assert no panic and read loop continues
- **🧪 TEST: Clean shutdown** — close transport, assert read loop goroutine exits (use `Done()` channel)

### Phase 6 — Core RPC Methods

> Goal: Implement DeviceInfo, ListEntities, SubscribeStates, Ping, Disconnect.

- [x] `DeviceInfo()` — send request, wait for response (synchronous request-response using a channel/callback)
- [x] `ListEntities()` — send request, collect all `ListEntities*Response` messages until `ListEntitiesDoneResponse` arrives
- [x] `SubscribeStates()` — send request, route incoming `*StateResponse` messages to user-provided callback
- [x] `Ping()` — send `PingRequest`, wait for `PingResponse` with timeout
- [x] `Disconnect()` — send `DisconnectRequest`, wait for `DisconnectResponse`, then close TCP
- [x] Implement request-response correlation: the router needs to support "wait for next message of type X" for synchronous calls
- **🧪 TEST: DeviceInfo round-trip** — mock server responds with known device info, verify all fields
- **🧪 TEST: ListEntities** — mock server sends 3 sensor entities + 2 switch entities + `ListEntitiesDoneResponse`, verify registry contains all 5
- **🧪 TEST: ListEntities timeout** — server never sends `ListEntitiesDoneResponse`, verify client times out
- **🧪 TEST: Ping timeout** — server never sends `PingResponse`, verify error after deadline
- **🧪 TEST: Disconnect sequence** — verify both request and response are exchanged before TCP close

### Phase 7 — Entity Registry

> Goal: Type-safe entity storage with state caching.

- [x] Define Go structs for each entity domain (sensor, binary sensor, switch, light, climate, etc.)
- [x] Populate metadata from `ListEntities*Response` messages (key, name, object_id, device_class, icon, unit, etc.)
- [x] Update state on incoming `*StateResponse` messages (store latest value, handle `missing_state`)
- [x] Provide accessors: `Sensors() []*SensorEntity`, `Switches() []*SwitchEntity`, `ByKey(uint32) Entity`, `ByName(string) Entity`
- [x] Thread-safe reads (RWMutex) since the read loop writes and user code reads concurrently
- **🧪 TEST: Populate + query** — feed list-entities responses, verify `Sensors()` returns correct items
- **🧪 TEST: State update** — feed a `SensorStateResponse`, verify `entity.State` is updated
- **🧪 TEST: Missing state** — feed a response with `missing_state=true`, verify entity reflects this
- **🧪 TEST: Concurrent access** — run reads and writes from multiple goroutines with `-race`

### Phase 8 — Command Sending

> Goal: Send commands to controllable entities.

- [x] `SendCommand(cmd proto.Message) error` — generic: marshal and send any command message
- [x] Type-safe convenience methods:
  - `SetSwitch(key uint32, state bool) error`
  - `SetLight(key uint32, opts LightCommandOpts) error`
  - `SetClimate(key uint32, opts ClimateCommandOpts) error`
  - `SetNumber(key uint32, value float32) error`
  - `SetSelect(key uint32, value string) error`
  - `PressButton(key uint32) error`
  - `SetCoverPosition(key uint32, position float32) error`
  - `SetCover(key uint32, opts CoverCommandOpts) error`
  - `SetFan(key uint32, opts FanCommandOpts) error`
  - `SetSiren(key uint32, opts SirenCommandOpts) error`
  - `SetLock(key uint32, command LockCommand, code string) error`
  - `SetMediaPlayer(key uint32, opts MediaPlayerCommandOpts) error`
- [x] Validate that the entity key exists and is the correct type before sending (optional but nice)
- **🧪 TEST: Switch command** — send `SetSwitch`, capture the bytes written to the pipe, decode and verify the protobuf fields ✅
- **🧪 TEST: Light command with partial fields** — only set brightness (no color), verify `has_*` flags are set correctly in the proto ✅

### Phase 9 — Noise Encryption Transport

> Goal: Support encrypted connections using the Noise protocol.

- [x] Implement the Noise handshake (`Noise_NNpsk0_25519_ChaChaPoly_SHA256`) — the PSK is derived from the base64 `encryption.key` in ESPHome config
- [x] The handshake happens **before** the first protobuf frame (the `0x00` prefix byte is replaced by a different framing during handshake)
- [x] After handshake, wrap the `net.Conn` so reads/writes are transparently encrypted/decrypted
- [x] Study the [ESPHome Noise implementation](https://github.com/esphome/esphome/blob/dev/esphome/components/api/api_noise_context.h) and the [aioesphomeapi Python client](https://github.com/esphome/aioesphomeapi/blob/main/aioesphomeapi/_frame_helper/noise.pyx) for reference
- [x] The Noise frame format differs from plain: `0x01` prefix byte, then 2-byte big-endian length, then encrypted payload
- **🧪 TEST: Noise handshake against real device** — integration test (can be skipped in CI, tagged `//go:build integration`)
- [x] **🧪 TEST: Noise handshake mock** — simulate both sides of the handshake using `flynn/noise`, verify encrypted frames decode correctly
- [x] **🧪 TEST: Wrong PSK** — attempt handshake with wrong key, verify clean error

### Phase 10 — Keepalive & Reconnect

> Goal: Production-grade connection management.

- [x] Background keepalive goroutine: send `PingRequest` every N seconds (configurable, default 20s)
- [x] If no `PingResponse` within timeout, consider connection dead → trigger disconnect callback → attempt reconnect
- [x] Reconnect loop with exponential backoff (use `go-retry` or similar)
- [x] On reconnect: re-run Hello → DeviceInfo → ListEntities → SubscribeStates
- [x] Expose connection state: `Connected() bool`, `OnConnect(func())`, `OnDisconnect(func())`
- [x] Context-aware: all goroutines should exit cleanly when the parent context is cancelled
- [x] **🧪 TEST: Keepalive fires** — use a short interval, mock server, assert `PingRequest` arrives on schedule
- [x] **🧪 TEST: Dead connection detection** — don't respond to ping, verify client triggers disconnect after timeout
- [x] **🧪 TEST: Reconnect** — simulate disconnect, verify client re-establishes connection and re-subscribes
- [x] **🧪 TEST: Context cancellation** — cancel context, verify all goroutines exit (no goroutine leaks — use `goleak`)

### Phase 11 — Log Streaming & Services

> Goal: Support subscribe_logs and execute_service.

- [x] `SubscribeLogs(level LogLevel, handler func(*SubscribeLogsResponse))` — stream device logs
- [x] `ExecuteService(key uint32, args []ServiceArg) error` — call custom ESPHome services
- [x] Parse `ListEntitiesServicesResponse` during entity discovery to know available services and their argument types
- [x] **🧪 TEST: Log streaming** — mock server sends log messages at various levels, verify handler receives only those at or above the requested level
- [x] **🧪 TEST: Service execution** — send a service call with mixed argument types (int, float, string, bool), verify the proto encodes correctly

### Phase 12 - Cli utility

- [x] Take esphome device yaml as input
- [x] Stream logs
- [x] stream sensors
- [x] set switches

### Phase 13 — Bluetooth Proxy (Optional)

> Goal: Support BLE operations through the ESPHome device.

- [x] `SubscribeBluetoothAdvertisements(handler func(proto.Message))`
- [x] `BluetoothConnect(address uint64) error` / `BluetoothDisconnect(address uint64) error`
- [x] GATT operations: `GATTGetServices`, `GATTRead`, `GATTWrite`, `GATTNotify`
- [x] Handle `BluetoothGATTErrorResponse` for error propagation
- [x] Connection slot management via `BluetoothConnectionsFree`
- [x] **🧪 TEST: GATT read** — mock server returns data for a handle, verify client returns it ✅
- [x] **🧪 TEST: GATT error** — mock server returns error response, verify client surfaces the error ✅
- [x] **🧪 TEST: Connection slots** — verify free/limit/allocated parsing ✅

---

### Testing Strategy Summary

| Category                               | Approach                                 | Tools                                 |
| -------------------------------------- | ---------------------------------------- | ------------------------------------- |
| **Unit (codec, registry, router)**     | Pure Go tests, no I/O                    | `testing`, `testify/assert`           |
| **Integration (transport, handshake)** | `net.Pipe()` or mock TCP server          | `testing`, `net.Pipe`                 |
| **Concurrency**                        | Run with `-race`, use `goleak`           | `go test -race`, `go.uber.org/goleak` |
| **Noise encryption**                   | Mock both sides with `flynn/noise`       | `flynn/noise`, `net.Pipe`             |
| **End-to-end (real device)**           | Build-tagged integration tests           | `//go:build integration`              |
| **Fuzz**                               | Fuzz the frame decoder with random bytes | `testing.F`, `go test -fuzz`          |

Key test infrastructure to build early:

```go
// mockTransport implements Transport using net.Pipe() for testing.
type mockTransport struct {
    client net.Conn // client side
    server net.Conn // server side (test controls this)
}

// Helper to send a protobuf message from the "server" side
func (m *mockTransport) ServerSend(msgType uint32, msg proto.Message) error

// Helper to read what the "client" sent
func (m *mockTransport) ServerRecv() (uint32, proto.Message, error)
```

This mock lets you simulate an ESPHome device in tests without any network I/O.
