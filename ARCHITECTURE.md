# esphome-apiclient Architecture

This document contains the architecture overview for `esphome-apiclient`.

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

## Package Layout

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
├── PROTOCOL.md            # Protocol framing and connection lifecycle
├── ARCHITECTURE.md        # Architecture overview
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

## Code Generation

Generate Go protobuf code from `api.proto`:

```sh
protoc --go_out=pb --go_opt=paths=source_relative api.proto
```

Requires `protoc` and `protoc-gen-go`. The generated code lands in `pb/api.pb.go`.
