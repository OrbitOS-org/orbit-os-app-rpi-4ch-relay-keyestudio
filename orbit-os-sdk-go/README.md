# orbit-os-sdk-go

Go SDK for **OrbitOS / Gravity** — a high-level gRPC client that wraps every OrbitOS system service and exposes it as a typed Go API.

**Repository:** https://github.com/OrbitOS-org/sdk-go  
**Module:** `github.com/OrbitOS-org/sdk-go/v26`  
**Version:** `26.0.1`  
**API Reference:** [`sdk/api-reference.html`](https://www.orbit-os.org/api-reference.html)

---

## Requirements

- Go **1.25+**
- Access to an OrbitOS device (Unix socket on-device, or TCP+TLS over network)

---

## Installation

```bash
go get github.com/OrbitOS-org/sdk-go/v26
```

To pin a **branch tip** (e.g. `API_26`):

```bash
go get github.com/OrbitOS-org/sdk-go/v26@API_26
```

For **local development** next to your app, use a `replace` directive in your `go.mod`:

```go
require github.com/OrbitOS-org/sdk-go/v26 v0.0.0

replace github.com/OrbitOS-org/sdk-go/v26 => ../orbit-os-sdk-go
```

---

## Connecting

The SDK exposes three constructors. All of them return a `*client.Client` that holds every service manager as a field.

```go
import "github.com/OrbitOS-org/sdk-go/v26/client"
```

### Auto (recommended)

Prefers the Unix socket when running on-device, falls back to TCP+TLS otherwise. Also verifies that the SDK API version matches the device.

```go
c, err := client.NewClientAuto("192.168.1.100")
if err != nil {
    log.Fatal(err)
}
defer c.Close()
```

### Unix domain socket (on-device only)

```go
c, err := client.NewUDSClient()
```

### TCP + TLS (remote / cross-machine)

```go
c, err := client.NewTCPClient("192.168.1.100", 6000)
```

TLS credentials are loaded automatically from `certs/grpc/` under the working directory (or any ancestor). Place your certificates at:

```
your-app/
  certs/grpc/
    ca.crt        # required — CA that signed the server cert
    client.crt    # optional — for mutual TLS
    client.key
```

Set `ORBIT_GRPC_TLS_SERVER_NAME` to pin the server's CN/SAN.

---

## Service Managers

Every service is accessed through a manager field on `*Client`:

| Field | Service | Description |
|-------|---------|-------------|
| `c.AIManager` | AIService | Load ONNX/TFLite models, run inference, stream results |
| `c.AppHubManager` | AppHubService | Register WebUI with the Gravity portal (proxy routing) |
| `c.AuthManager` | AuthService | Login / logout from the device |
| `c.BluetoothManager` | BluetoothService | BLE scan, connect, GATT read/write/notify |
| `c.CameraManager` | CameraService | Enumerate cameras, capture frames, start streams |
| `c.DevelopmentManager` | DevelopmentService | Developer utilities (shell, file transfer, etc.) |
| `c.EthernetManager` | EthernetService | Ethernet interface configuration and status |
| `c.EventManager` | EventService | Subscribe to and publish system-wide events |
| `c.FirewallManager` | FirewallService | Manage zones, traffic rules, apply firewall |
| `c.GpioManager` | GpioService | Read/write GPIO pins, configure direction |
| `c.I2CManager` | I2CService | I2C bus enumeration and transfers (handle pattern) |
| `c.PackageManager` | PackageService | Install, remove, list ORB packages |
| `c.PowerManager` | PowerService | Reboot, shutdown, sleep |
| `c.PwmManager` | PwmService | PWM channel configuration and output |
| `c.SpiManager` | SpiService | SPI device enumeration and transfers (handle pattern) |
| `c.SystemManager` | SystemService | Device info, API version, system stats |
| `c.UartManager` | UartService | UART port configuration and data transfer (handle pattern) |
| `c.UpdateManager` | UpdateService | OTA firmware update management |
| `c.VPNManager` | VPNService | WireGuard / OpenVPN profiles, connect, watch events |
| `c.WiFiManager` | WiFiService | Scan, connect, disconnect Wi-Fi networks |

---

## Handle Pattern

Some managers use a two-step **open → handle** model. `Open()` configures the resource and returns a typed handle; all subsequent operations are called on the handle. This eliminates repeated parameters (bus number, port name, device ID) on every call.

| Manager | `Open()` returns | Resource identity |
|---------|-----------------|-------------------|
| `UartManager` | `*UartPort` | port string (e.g. `"ttyS0"`) |
| `I2CManager` | `*I2CBus` | bus number |
| `SpiManager` | `*SpiDevice` | bus + chip-select |
| `CameraManager` | `*LockedCamera` | device ID + client ID |
| `AIManager` | `*AIModel` | model path |

### Example — I2C

```go
bus, err := c.I2CManager.Open(1, 100_000, false, false)
if err != nil {
    log.Fatal(err)
}

addrs, _ := bus.Scan()
fmt.Println("devices found:", addrs)

data, _ := bus.Transfer(0x48, nil, 2, 0) // read 2 bytes from addr 0x48
```

### Example — SPI

```go
dev, err := c.SpiManager.Open(0, 0, 1_000_000, 8, 0, false)
if err != nil {
    log.Fatal(err)
}

rx, _ := dev.Transfer([]byte{0x9F, 0x00, 0x00}, 3) // full-duplex
```

### Example — UART

```go
port, err := c.UartManager.Open("ttyS0", 115200, 8, "N", 1, "none")
if err != nil {
    log.Fatal(err)
}

port.Write([]byte("hello\n"))
data, _ := port.Read(64)
```

---

## Quick Examples

### GPIO

```go
c.GpioManager.SetDirection("GPIO17", client.GpioDirectionOutput)
c.GpioManager.SetValue("GPIO17", client.GpioLevelHigh)
level, _ := c.GpioManager.GetValue("GPIO17")
```

### WiFi

```go
networks, _ := c.WiFiManager.Scan()
for _, n := range networks {
    fmt.Printf("%s (%d dBm)\n", n.GetSsid(), n.GetSignal())
}
c.WiFiManager.Connect("MyNetwork", "password123")
```

### VPN

```go
config, _ := os.ReadFile("vpn.conf")
profileID, _ := c.VPNManager.ApplyWireGuard("work-vpn", config, false)
sessionID, _ := c.VPNManager.Connect(profileID)
fmt.Println("session:", sessionID)
```

### AI Inference

```go
model, err := c.AIManager.LoadModel("/models/yolov8n.onnx", "")
if err != nil {
    log.Fatal(err)
}
defer model.Unload()

result, _ := model.RunInference(inputTensor)
```

### Events

```go
c.EventManager.Subscribe(ctx, "sensor.temperature", func(evt *eventsvcv26.Event) {
    fmt.Println("temp event:", evt.GetPayload())
})
```

### AppHub — register a WebUI

```go
err := c.AppHubManager.RegisterWebUI("127.0.0.1:9033", "/myapp")
if err != nil {
    log.Fatal(err)
}
defer c.AppHubManager.UnregisterService()
```

---

## Repository Layout

| Path | Purpose |
|------|---------|
| `client/` | High-level gRPC wrappers — one `*Manager` per service |
| `api/` | Generated `*.pb.go` / `*_grpc.pb.go` files per service |
| `logger/` | Structured logging helpers for device apps |
| `metadata/` | SDK metadata (version constants, etc.) |
| `scripts/build_package.sh` | Builds an ORB package from a Go main |
| `api/proto/gen.sh` | Regenerates Go bindings from proto sources |

---

## Building an ORB Package

From the **workspace root** (the directory that contains `orbit-os-sdk-go/`):

```bash
# default: build all, target arch from GOARCH env
./orbit-os-sdk-go/scripts/build_package.sh

# specific sub-package, arm64
./orbit-os-sdk-go/scripts/build_package.sh -path basic -arch arm64
```

Set `ORBIT_PROJECT_ROOT` if the SDK is not checked out as a direct child of the workspace root.

---

## Regenerating Proto Bindings

Requires the `gravity-api-proto` repository as a sibling to this SDK tree.

```bash
cd api/proto
./gen.sh
```

Update `GO_IMPORT_PREFIX` in `gen.sh` if the Go module path changes; it must match `go.mod`.

---

## Building

```bash
go build ./...
```

---

## API Reference

Open [`sdk/api-reference-static.html`](https://www.orbit-os.org/api-reference.html) in a browser for the full interactive API reference, including all service methods, parameters, return types, and usage examples.

---
