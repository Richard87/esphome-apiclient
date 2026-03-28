# ESPHome Protocol Description

This document contains the protocol and connection details used by `esphome-apiclient`.

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
