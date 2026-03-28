# esphome-apiclient — Go ESPHome Native API Client

A pure-Go client library for the [ESPHome Native API](https://esphome.io/components/api.html), enabling direct communication with ESPHome devices over TCP using Protocol Buffers.

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

## CLI Tool

A bundled CLI (`cmd/esphome-cli`) lets you quickly interact with ESPHome devices from the terminal.

### Install

To install the CLI utility, run:

```sh
go install github.com/richard87/esphome-apiclient/cmd/esphome-cli@latest
```

Ensure your `GOBIN` directory (usually `$HOME/go/bin`) is in your `PATH`.

### Usage

The CLI uses subcommands for different operations. Global flags (address, yaml, key, etc.) should be placed before the subcommand or after it (flags are inherited).

```sh
# Scan for devices on the local network
esphome-cli scan

# Connect using an ESPHome YAML config (derives address and encryption key automatically)
esphome-cli --yaml device.yaml info

# Or specify address and key directly
esphome-cli --address mydevice.local:6053 --key "base64-noise-psk" logs

# Available subcommands:
#   scan       — scan for ESPHome devices on local network
#   info       — print device info (name, MAC, ESPHome version, etc.)
#   entities   — list all entities (sensors, switches, lights, services, etc.)
#   sensors    — stream live sensor/switch/binary sensor state updates
#   logs       — stream device logs (--level DEBUG|INFO|WARN|ERROR|VERBOSE)
#   switch     — control/list switches (--switch-key 0x1234 --switch-state on|off)
```

### Examples

```sh
# Show device info
esphome-cli --yaml device.yaml info

# Stream sensor values
esphome-cli --yaml device.yaml sensors

# Stream logs at INFO level
esphome-cli --yaml device.yaml logs --level INFO

# List available switches (omit --switch-key to list them)
esphome-cli --yaml device.yaml switch

# Turn a switch on
esphome-cli --yaml device.yaml switch --switch-key 0xABCD1234 --switch-state on
```

## Library Usage

```go
import (
    "context"
    "fmt"
    "log"
    "time"

    esphome "github.com/richard87/esphome-apiclient"
    "github.com/richard87/esphome-apiclient/pb"
    "google.golang.org/protobuf/proto"
)

func main() {
    ctx := context.Background()

    client, err := esphome.DialWithContext(ctx, "mydevice.local:6053", 5*time.Second,
        esphome.WithEncryptionKey("base64-noise-psk"),
        esphome.WithClientInfo("my-app"),
        esphome.WithReconnect(10*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Get device info
    info, _ := client.DeviceInfo()
    fmt.Printf("Connected to %s (ESPHome %s)\n", info.Name, info.EsphomeVersion)

    // Discover entities
    client.ListEntities()
    for _, s := range client.Entities().Sensors() {
        fmt.Printf("  sensor: %s (%s)\n", s.Name, s.UnitOfMeasurement)
    }

    // Stream state updates
    client.SubscribeStates(func(msg proto.Message) {
        switch m := msg.(type) {
        case *pb.SensorStateResponse:
            entity := client.Entities().ByKey(m.Key)
            fmt.Printf("sensor %s = %.4g\n", entity.GetName(), m.State)
        case *pb.SwitchStateResponse:
            fmt.Printf("switch 0x%08X = %v\n", m.Key, m.State)
        }
    })

    // Send a command
    client.SetSwitch(0x12345678, true)

    // Stream logs
    client.SubscribeLogs(pb.LogLevel_LOG_LEVEL_DEBUG, func(msg *pb.SubscribeLogsResponse) {
        fmt.Printf("[%s] %s\n", msg.Level, msg.Message)
    })

    // Bluetooth Proxy (Phase 13)
    client.SubscribeBluetoothAdvertisements(func(msg proto.Message) {
        if adv, ok := msg.(*pb.BluetoothLERawAdvertisementsResponse); ok {
            fmt.Printf("BLE adv: %d devices\n", len(adv.Advertisements))
        }
    })

    <-ctx.Done()
}
```

## Bluetooth Proxy (Experimental)

ESPHome devices can act as Bluetooth proxies. The client supports:

- **Advertisements**: `SubscribeBluetoothAdvertisements(handler)`
- **Connections**: `BluetoothConnect(address)`, `BluetoothDisconnect(address)`
- **GATT**: `BluetoothGATTGetServices(address)`, `BluetoothGATTRead(address, handle)`, `BluetoothGATTWrite(address, handle, data, response)`, `BluetoothGATTNotify(address, handle, enable, handler)`
- **Scanner**: `BluetoothScannerSetMode(mode)`
- **Capacity**: `SubscribeBluetoothConnectionsFree(handler)`

> Experimental:
> I dont have any use for Bluetooth Proxy at the moment, so this is experimental until further notice. If you use it, and it works for you (or not), please let me know!

## Protocol

Protocol framing and lifecycle details have been moved to [PROTOCOL.md](PROTOCOL.md).

## Architecture

Architecture overview and package layout have been moved to [ARCHITECTURE.md](ARCHITECTURE.md).

## Implementation Guide

The phased implementation plan and testing strategy have been moved to [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md).

## References

- [ESPHome Native API Protocol](https://esphome.io/components/api.html)
- [ESPHome API Protobuf definitions](https://github.com/esphome/esphome/blob/dev/esphome/components/api/api.proto)
- [Noise Protocol Framework](http://noiseprotocol.org/)
- [flynn/noise (Go implementation)](https://github.com/flynn/noise)
- [mycontroller-org/esphome_api](https://github.com/mycontroller-org/esphome_api) — existing Go client (used as reference)
