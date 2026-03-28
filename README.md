# esphome-apiclient — Go ESPHome Native API Client

A pure-Go client library for the [ESPHome Native API](https://esphome.io/components/api.html), enabling direct communication with ESPHome devices over TCP using Protocol Buffers.

## Protocol Overview

ESPHome devices expose a TCP server (default port `6053`) that speaks a lightweight binary protocol:

```
┌──────────┬──────────────────┬──────────────────┬────────────────────────┐
│ 0x00     │ VarInt(msg_size) │ VarInt(msg_type) │ Protobuf-encoded body  │
└──────────┴──────────────────┴──────────────────┴────────────────────────┘
```

Each frame starts with a zero byte, followed by a varint-encoded message size, a varint-encoded message type ID (from `option (id)` in `api.proto`), and then the protobuf payload.

## Connection Lifecycle

```
Client                                          ESPHome Device
  │                                                   │
  │──── TCP Connect ─────────────────────────────────►│
  │                                                   │
  │──── HelloRequest (id=1) ─────────────────────────►│
  │◄─── HelloResponse (id=2) ────────────────────────│
  │     (negotiate API version)                       │
  │                                                   │
  │  ┌─ If Noise encryption is configured ──────────┐ │
  │  │  Noise handshake occurs at the TCP level      │ │
  │  │  before any protobuf messages are exchanged.  │ │
  │  │  Uses Noise_NNpsk0_25519_ChaChaPoly_SHA256    │ │
  │  └──────────────────────────────────────────────┘ │
  │                                                   │
  │──── DeviceInfoRequest (id=9) ────────────────────►│
  │◄─── DeviceInfoResponse (id=10) ──────────────────│
  │                                                   │
  │──── ListEntitiesRequest (id=11) ─────────────────►│
  │◄─── ListEntities*Response (one per entity) ──────│
  │◄─── ListEntitiesDoneResponse (id=19) ────────────│
  │                                                   │
  │──── SubscribeStatesRequest (id=20) ──────────────►│
  │◄─── *StateResponse (streamed as values change) ──│
  │                                                   │
  │──── PingRequest (id=7) ──────────────────────────►│
  │◄─── PingResponse (id=8) ─────────────────────────│
  │                                                   │
  │──── DisconnectRequest (id=5) ────────────────────►│
  │◄─── DisconnectResponse (id=6) ───────────────────│
  │                                                   │
  │──── TCP Close ────────────────────────────────────│
```

> **Note:** Password authentication (`AuthenticationRequest`/`AuthenticationResponse`, IDs 3/4) was removed in ESPHome 2026.1.0. Use Noise encryption instead.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│  (your code: subscribe to sensors, send commands, etc.)      │
├──────────────────────────────────────────────────────────────┤
│                       Client (public API)                    │
│  Connect() / Close() / ListEntities() / SubscribeStates()   │
│  DeviceInfo() / SendCommand() / Ping()                       │
├──────────────────────────────────────────────────────────────┤
│                     Entity Registry                          │
│  Maps entity key → metadata (name, type, device_class, etc.) │
│  Caches latest state for each entity                         │
├──────────────────────────────────────────────────────────────┤
│                     Message Router                           │
│  Dispatches decoded messages by type to handlers/callbacks   │
│  Manages subscription lifecycle                              │
├──────────────────────────────────────────────────────────────┤
│                    Codec (frame layer)                       │
│  Encodes: 0x00 + VarInt(size) + VarInt(type) + proto bytes  │
│  Decodes: reads frames from TCP, resolves type → proto msg   │
├──────────────────────────────────────────────────────────────┤
│                    Transport Layer                            │
│  ┌─────────────────┐  ┌──────────────────────────────────┐  │
│  │   Plain TCP      │  │  Noise Protocol Encryption       │  │
│  │   (no encryption)│  │  Noise_NNpsk0_25519_ChaChaPoly   │  │
│  └─────────────────┘  └──────────────────────────────────┘  │
├──────────────────────────────────────────────────────────────┤
│                     net.Conn (TCP)                            │
└──────────────────────────────────────────────────────────────┘
```

### Package Layout

```
esphome-apiclient/
├── api.proto              # ESPHome API protobuf definitions (source of truth)
├── pb/                    # Generated Go protobuf code (from api.proto)
│   └── api.pb.go
├── transport/
│   ├── transport.go       # Transport interface (Read/Write frames)
│   ├── plain.go           # Plain TCP transport
│   └── noise.go           # Noise-encrypted transport (NNpsk0_25519_ChaChaPoly_SHA256)
├── codec/
│   └── codec.go           # Frame encoding/decoding (0x00 + varint size + varint type + body)
├── client.go              # High-level Client struct and connection lifecycle
├── entities.go            # Entity registry: type-safe wrappers per entity domain
├── router.go              # Message routing and subscription dispatch
└── README.md
```

## Core Components

### 1. Transport Layer

Abstracts the underlying connection so the rest of the client is encryption-agnostic.

```go
// Transport reads and writes raw bytes over the connection.
type Transport interface {
    Read(p []byte) (n int, err error)
    Write(p []byte) (n int, err error)
    Close() error
}
```

Two implementations:
- **PlainTransport** — wraps a bare `net.Conn`.
- **NoiseTransport** — performs the [Noise Protocol](http://noiseprotocol.org/) handshake (`Noise_NNpsk0_25519_ChaChaPoly_SHA256`) using a pre-shared key, then wraps the encrypted channel. Uses `github.com/flynn/noise`.

### 2. Frame Codec

Handles the ESPHome binary framing on top of the transport:

```go
type Frame struct {
    Type uint32 // message type ID from api.proto option (id)
    Data []byte // protobuf-encoded message body
}

func Encode(msg proto.Message, msgType uint32) ([]byte, error)
func Decode(r io.Reader) (*Frame, error)
```

### 3. Message Router

Maintains a registry mapping message type IDs → protobuf message types, and dispatches decoded messages to registered handlers.

```go
type MessageHandler func(msg proto.Message)

type Router struct {
    handlers map[uint32][]MessageHandler
}

func (r *Router) On(msgType uint32, handler MessageHandler)
func (r *Router) Dispatch(frame *Frame) error
```

Supports fan-out: multiple handlers can subscribe to the same message type (e.g., a state cache + a user callback both listening for `SensorStateResponse`).

### 4. Entity Registry

Caches entity metadata from `ListEntities*Response` and latest state from `*StateResponse`. Provides type-safe accessors per entity domain.

```go
type EntityRegistry struct {
    sensors       map[uint32]*SensorEntity
    binarySensors map[uint32]*BinarySensorEntity
    switches      map[uint32]*SwitchEntity
    lights        map[uint32]*LightEntity
    // ... one map per entity domain
}

type SensorEntity struct {
    Key              uint32
    Name             string
    ObjectID         string
    Unit             string
    DeviceClass      string
    StateClass       SensorStateClass
    AccuracyDecimals int32
    State            float32
    Missing          bool
}
```

### 5. Client (Public API)

The main entry point. Orchestrates connection, entity discovery, and subscriptions.

```go
type Client struct {
    transport Transport
    codec     *Codec
    router    *Router
    entities  *EntityRegistry
}

func Dial(address string, opts ...Option) (*Client, error)
func (c *Client) DeviceInfo() (*DeviceInfoResponse, error)
func (c *Client) ListEntities() (*EntityRegistry, error)
func (c *Client) SubscribeStates(handler func(entityKey uint32, state proto.Message))
func (c *Client) SubscribeLogs(level LogLevel, handler func(log *SubscribeLogsResponse))
func (c *Client) SendCommand(cmd proto.Message) error
func (c *Client) Ping() error
func (c *Client) Close() error
```

#### Options

```go
func WithEncryptionKey(key string) Option   // base64-encoded Noise PSK
func WithTimeout(d time.Duration) Option    // connection and read timeout
func WithClientInfo(info string) Option     // client_info in HelloRequest
func WithOnDisconnect(fn func()) Option     // callback on unexpected disconnect
func WithReconnect(interval time.Duration) Option // auto-reconnect with backoff
```

## Supported Entity Types

The client should support all entity domains defined in `api.proto`:

| Domain              | ListEntities Response                   | State Response                   | Command Request                       |
| ------------------- | --------------------------------------- | -------------------------------- | ------------------------------------- |
| Binary Sensor       | `ListEntitiesBinarySensorResponse`      | `BinarySensorStateResponse`      | —                                     |
| Cover               | `ListEntitiesCoverResponse`             | `CoverStateResponse`             | `CoverCommandRequest`                 |
| Fan                 | `ListEntitiesFanResponse`               | `FanStateResponse`               | `FanCommandRequest`                   |
| Light               | `ListEntitiesLightResponse`             | `LightStateResponse`             | `LightCommandRequest`                 |
| Sensor              | `ListEntitiesSensorResponse`            | `SensorStateResponse`            | —                                     |
| Switch              | `ListEntitiesSwitchResponse`            | `SwitchStateResponse`            | `SwitchCommandRequest`                |
| Text Sensor         | `ListEntitiesTextSensorResponse`        | `TextSensorStateResponse`        | —                                     |
| Camera              | `ListEntitiesCameraResponse`            | `CameraImageResponse`            | `CameraImageRequest`                  |
| Climate             | `ListEntitiesClimateResponse`           | `ClimateStateResponse`           | `ClimateCommandRequest`               |
| Water Heater        | `ListEntitiesWaterHeaterResponse`       | `WaterHeaterStateResponse`       | `WaterHeaterCommandRequest`           |
| Number              | `ListEntitiesNumberResponse`            | `NumberStateResponse`            | `NumberCommandRequest`                |
| Select              | `ListEntitiesSelectResponse`            | `SelectStateResponse`            | `SelectCommandRequest`                |
| Siren               | `ListEntitiesSirenResponse`             | `SirenStateResponse`             | `SirenCommandRequest`                 |
| Lock                | `ListEntitiesLockResponse`              | `LockStateResponse`              | `LockCommandRequest`                  |
| Button              | `ListEntitiesButtonResponse`            | —                                | `ButtonCommandRequest`                |
| Media Player        | `ListEntitiesMediaPlayerResponse`       | `MediaPlayerStateResponse`       | `MediaPlayerCommandRequest`           |
| Alarm Control Panel | `ListEntitiesAlarmControlPanelResponse` | `AlarmControlPanelStateResponse` | `AlarmControlPanelCommandRequest`     |
| Text                | `ListEntitiesTextResponse`              | `TextStateResponse`              | `TextCommandRequest`                  |
| Date                | `ListEntitiesDateResponse`              | `DateStateResponse`              | `DateCommandRequest`                  |
| Time                | `ListEntitiesTimeResponse`              | `TimeStateResponse`              | `TimeCommandRequest`                  |
| DateTime            | `ListEntitiesDateTimeResponse`          | `DateTimeStateResponse`          | `DateTimeCommandRequest`              |
| Event               | `ListEntitiesEventResponse`             | `EventResponse`                  | —                                     |
| Valve               | `ListEntitiesValveResponse`             | `ValveStateResponse`             | `ValveCommandRequest`                 |
| Update              | `ListEntitiesUpdateResponse`            | `UpdateStateResponse`            | `UpdateCommandRequest`                |
| Infrared            | `ListEntitiesInfraredResponse`          | `InfraredRFReceiveEvent`         | `InfraredRFTransmitRawTimingsRequest` |

## Additional Features

### Bluetooth Proxy

ESPHome devices can act as BLE proxies. The client should support:

- `SubscribeBluetoothLEAdvertisements` — stream BLE advertisement packets
- `BluetoothDeviceRequest` — connect/disconnect/pair/unpair BLE devices
- `BluetoothGATT*` — read, write, and subscribe to GATT characteristics and descriptors
- `BluetoothConnectionsFree` — query available connection slots
- `BluetoothScannerSetMode` — switch between active/passive scanning

### Voice Assistant

- `SubscribeVoiceAssistant` — register as a voice assistant pipeline
- `VoiceAssistantRequest` / `VoiceAssistantResponse` — handle voice conversations
- `VoiceAssistantAudio` — stream audio data
- `VoiceAssistantConfiguration` — manage wake words

### Z-Wave Proxy

- `ZWaveProxyFrame` — relay Z-Wave frames bidirectionally
- `ZWaveProxyRequest` — subscribe/unsubscribe to Z-Wave events

### Home Assistant Services

- `SubscribeHomeassistantServices` — receive service call requests from the device
- `SubscribeHomeAssistantStates` — the device requests state from HA entities
- `ExecuteService` — invoke custom ESPHome services with typed arguments

### Log Streaming

- `SubscribeLogs` — stream device logs at a configurable log level

### Noise Encryption Key Management

- `NoiseEncryptionSetKey` — remotely provision a new encryption key on the device

## Concurrency Model

```
                    ┌──────────────────┐
                    │   Read Loop      │  (goroutine)
                    │  reads frames    │
                    │  from transport, │
                    │  dispatches via  │
                    │  Router          │
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
    ┌─────────────┐ ┌──────────────┐ ┌───────────┐
    │ State cache │ │ User handler │ │ Ping/pong │
    │ update      │ │ callback     │ │ tracking  │
    └─────────────┘ └──────────────┘ └───────────┘

                    ┌──────────────────┐
                    │   Write path     │  (mutex-protected)
                    │  serializes      │
                    │  proto → frame   │
                    │  writes to       │
                    │  transport       │
                    └──────────────────┘
```

- **Single read goroutine** continuously reads frames and dispatches to handlers. Handlers should not block — offload heavy work to separate goroutines.
- **Write path** is mutex-protected so multiple goroutines can safely call `SendCommand`, `Ping`, etc.
- **Keepalive** — a background goroutine sends periodic `PingRequest` messages and monitors for `PingResponse` to detect dead connections.
- **Reconnect** — optional auto-reconnect with exponential backoff. Re-discovers entities and re-subscribes to states after reconnection.

## Error Handling

- If the Hello handshake fails (version mismatch), close the connection immediately — no `DisconnectRequest`.
- On unexpected TCP close, invoke the disconnect callback and optionally trigger reconnect.
- API version negotiation: the client must check `api_version_major` / `api_version_minor` from `HelloResponse` and adapt or reject incompatible versions.

## Usage Example

```go
client, err := esphome.Dial("sensor-node.local:6053",
    esphome.WithEncryptionKey("base64-noise-psk-here"),
    esphome.WithClientInfo("esphome-apiclient"),
    esphome.WithReconnect(10 * time.Second),
)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

info, _ := client.DeviceInfo()
fmt.Printf("Connected to %s (%s)\n", info.Name, info.EsphomeVersion)

entities, _ := client.ListEntities()
for _, sensor := range entities.Sensors() {
    fmt.Printf("  sensor: %s (%s)\n", sensor.Name, sensor.Unit)
}

client.SubscribeStates(func(key uint32, state proto.Message) {
    switch s := state.(type) {
    case *pb.SensorStateResponse:
        fmt.Printf("sensor %d = %.2f\n", key, s.State)
    case *pb.BinarySensorStateResponse:
        fmt.Printf("binary_sensor %d = %v\n", key, s.State)
    }
})

// Send a command
client.SendCommand(&pb.SwitchCommandRequest{Key: 0x12345678, State: true})

// Block until context is cancelled
<-ctx.Done()
```

## Code Generation

Generate Go protobuf code from `api.proto`:

```sh
protoc --go_out=pb --go_opt=paths=source_relative api.proto
```

Requires `protoc` and `protoc-gen-go`. The generated code lands in `pb/api.pb.go`.

---

## Implementation Guide

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

- [ ] `SubscribeBluetoothAdvertisements(handler func(*BluetoothLEAdvertisementResponse))`
- [ ] `BluetoothConnect(address uint64) error` / `BluetoothDisconnect(address uint64) error`
- [ ] GATT operations: `GATTGetServices`, `GATTRead`, `GATTWrite`, `GATTNotify`
- [ ] Handle `BluetoothGATTErrorResponse` for error propagation
- [ ] Connection slot management via `BluetoothConnectionsFree`
- **🧪 TEST: GATT read** — mock server returns data for a handle, verify client returns it
- **🧪 TEST: GATT error** — mock server returns error response, verify client surfaces the error
- **🧪 TEST: Connection slots** — verify free/limit/allocated parsing

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

## References

- [ESPHome Native API Protocol](https://esphome.io/components/api.html)
- [ESPHome API Protobuf definitions](https://github.com/esphome/esphome/blob/dev/esphome/components/api/api.proto)
- [Noise Protocol Framework](http://noiseprotocol.org/)
- [flynn/noise (Go implementation)](https://github.com/flynn/noise)
- [mycontroller-org/esphome_api](https://github.com/mycontroller-org/esphome_api) — existing Go client (used as reference)
