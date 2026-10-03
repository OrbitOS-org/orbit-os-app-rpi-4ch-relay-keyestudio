<p align="center">
  <img src="https://www.orbit-os.org/images/vscode/orbit-os-logo.png" width="300" alt="Orbit OS">
</p>

<h1 align="center">RPI 4-Channel Relay Controller — Keyestudio KS0212</h1>

<p align="center"><b>Control the Keyestudio KS0212 relay shield on a Raspberry Pi from a web UI, Modbus TCP or MQTT / Home Assistant.</b></p>

An [Orbit OS](https://www.orbit-os.org/?ref=github-rpi4ch) app for the [Keyestudio KS0212](https://docs.keyestudio.com/projects/KS0212/en/latest/) 4-channel relay shield. Install it from the Store, open its page in the Orbit OS AppHub and switch relays from any browser — or integrate them with PLCs, SCADA or Home Assistant.

<a href="https://store.orbit-os.org/app/rpi-4ch?ref=github-rpi4ch"><img src="https://www.orbit-os.org/images/badges/get-it-on-orbit-os-store@3x.png" width="200" alt="Get it on Orbit OS Store"></a>

![RPI 4-Channel Relay Controller in the Orbit OS Store](docs/store-screenshot.png)

## Features

- **Web UI** — control and monitor all 4 relays from a browser, with real-time state (SSE)
- **Modbus TCP server** — FC01 Read Coils, FC03 Read Holding Registers, FC05 Write Single Coil, FC06 Write Single Register, FC0F Write Multiple Coils
- **MQTT / Home Assistant** — auto-discovery, `ON`/`OFF` command and state topics, availability (LWT), retained state
- **Board check** — warns at startup if the device does not look like a Raspberry Pi
- Uses the official [Orbit OS Go SDK](https://github.com/OrbitOS-org/orbit-os-sdk-go) — GPIO access goes through the Orbit OS permission model declared in the app manifest

## Hardware

- Raspberry Pi 3, 4 or 5 running [Orbit OS](https://www.orbit-os.org/getting_started.html?ref=github-rpi4ch)
- [Keyestudio KS0212](https://docs.keyestudio.com/projects/KS0212/en/latest/) 4-channel relay shield

| Channel | GPIO (BCM) | Modbus coil |
|---|---|---|
| CH1 | 4 | 0 |
| CH2 | 22 | 1 |
| CH3 | 6 | 2 |
| CH4 | 26 | 3 |

## Install

**From the Orbit OS Store (recommended):** install [RPI 4-Channel Relay Controller](https://store.orbit-os.org/app/rpi-4ch?ref=github-rpi4ch) on your device in one click. No build required.

**From source — recommended: [Orbit Studio](https://marketplace.visualstudio.com/items?itemName=orbit-os.orbit-studio) (VS Code):**

1. Clone the repository and open the folder in VS Code with the Orbit Studio extension:
   ```bash
   git clone https://github.com/OrbitOS-org/orbit-os-app-rpi-4ch-relay-keyestudio
   code orbit-os-app-rpi-4ch-relay-keyestudio
   ```
2. In the Orbit sidebar, run **Add / Update SDK** and set your device's IP.
3. Use **Run** to try it live against a device in Developer Mode, then **Build + Deploy** to install the signed `.orb`.

**Without Orbit Studio:** `go build ./cmd/rpi_4ch_relay_keyestudio` builds the binary with the published SDK module — use Orbit Studio to package and sign the `.orb`.

## Usage

### Web UI
Open the app from the Orbit OS AppHub (`http://<DEVICE_IP>/rpi-4ch`). Modbus and MQTT are configured from the same page.

### Modbus TCP
Default port **5020** (configurable in the web UI; saved on the device). Coils 0–3 map to CH1–CH4.

### MQTT / Home Assistant
Set the broker address, credentials, topic prefix and device ID in the web UI. Home Assistant discovery is published on connect.

| Topic | Payload |
|---|---|
| `<prefix>/<device_id>/<ch>/state` | `ON` / `OFF` |
| `<prefix>/<device_id>/<ch>/set` | `ON` / `OFF` |
| `<prefix>/<device_id>/availability` | `online` / `offline` |

## Development (Orbit Studio)

This project follows the [Orbit Studio](https://marketplace.visualstudio.com/items?itemName=orbit-os.orbit-studio) layout:

| Path | What |
|---|---|
| `cmd/rpi_4ch_relay_keyestudio/` | app source — `main.go` (relays, web server, Modbus), `mqtt.go`, `webui/` (embedded UI), `metadata.json` (manifest & permissions) |
| `cmd/rpi_4ch_relay_keyestudio/orb/icon.svg` | launcher / Store icon |
| `orbit.project.json` | Orbit Studio project settings (your device IP goes in the git-ignored `orbit.project.local.json`) |

- **Recommended workflow:** open the folder in VS Code with Orbit Studio, **Add / Update SDK** (creates the local `orbit-os-sdk-go/` copy and `go.work`, both git-ignored), then **Run** against a device in Developer Mode, or **Build + Deploy**.
- Without Orbit Studio, `go build` uses the published SDK module [`github.com/OrbitOS-org/orbit-os-sdk-go/v26`](https://pkg.go.dev/github.com/OrbitOS-org/orbit-os-sdk-go/v26).
- Development TLS certificates live in `cmd/certs/grpc/` and are never committed.

## Security

- The web UI is only reachable through the Orbit OS AppHub, behind the Launcher login.
- **Modbus TCP has no authentication** (a limitation of the protocol): when enabled, anyone who can reach port 5020 can switch the relays. Enable it only on trusted networks.
- Relays switch real loads — check your wiring and ratings before connecting mains-powered equipment.

## Links

[App in the Store](https://store.orbit-os.org/app/rpi-4ch?ref=github-rpi4ch) · [Orbit OS](https://www.orbit-os.org/?ref=github-rpi4ch) · [Getting started](https://www.orbit-os.org/getting_started.html?ref=github-rpi4ch) · [SDK reference](https://www.orbit-os.org/api-reference.html?ref=github-rpi4ch) · [Forum](https://forum.orbit-os.org/?ref=github-rpi4ch) · info@orbit-os.org

## License

Apache-2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
