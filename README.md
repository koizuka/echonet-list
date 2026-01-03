# ECHONET Lite Device Discovery and Control Tool

This is a Go application for discovering and controlling ECHONET Lite devices on a local network. ECHONET Lite is a communication protocol for smart home devices, primarily used in Japan.

**Author**: @koizuka

<img width="1265" height="445" alt="image" src="https://github.com/user-attachments/assets/9267e37c-df0f-4d61-b3fa-6f17815efdc9" />

## Features

- Automatic discovery of ECHONET Lite devices on the local network
- List all discovered devices with their properties
- Get and set property values on devices
- Persistent storage of discovered devices
- Support for various device types (air conditioners, lighting, floor heating, etc.)
- **Modern Web UI**: React-based interface with real-time updates
- **Device Organization**: Tab-based navigation by location and device groups
- **Visual Controls**: Property-specific UI controls (dropdowns, sliders, toggles)
- **Device History**: View device property change history with timeline, hex viewer for raw data, and filtering options
- **Status Indicators**: Visual feedback for device operation and fault states
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Internationalization**: Multi-language support (English, Japanese) for property descriptions and UI
- **Network Monitoring**: Automatic detection of network interface changes for reliable multicast communication
- WebSocket API for custom client development
- TLS support for secure connections
- Daemon mode for running as a system service

## Quick Start

Most users run the server on a target host (e.g., Raspberry Pi) and use `script/` for setup.

```bash
git clone https://github.com/koizuka/echonet-list.git
cd echonet-list

# Build everything (server + web UI)
./script/build.sh

# Run with Web UI + HTTP proxy
./echonet-list -websocket -http-enabled
```

Before opening the UI, enable TLS in `config.toml` and generate certificates
with mkcert (see [docs/installation.md](docs/installation.md)).

Open your browser to `https://<host>:8080`.

### LAN Usage and TLS

TLS is required even on a trusted LAN because modern browsers (especially on
mobile) block non-secure WebSocket connections from secure pages. The systemd
config enables TLS by default so the UI uses HTTPS/WSS. Use `mkcert` and install
the CA on client devices so the browser accepts the certificate.

If you expose the service outside your LAN, put it behind a reverse proxy with
authentication and keep HTTPS/WSS enabled there.

### Requirements

- Go 1.23+
- Node.js 18+ (only needed for the Web UI build/dev)
- Multicast-enabled local network (ECHONET Lite devices must be on the same L2 segment)

### First Run Notes

- Discovery uses multicast; if you see no devices, confirm your host network interface allows multicast.
- The Web UI is bundled to `web/bundle/` and served by the Go binary.
- To override defaults, copy `config.toml.sample` to `config.toml` and edit as needed.
- Browser access requires TLS/WSS; keep `tls.enabled = true` and use mkcert.

### Local Development (no build scripts)

```bash
# Server with debug logging
go run ./main.go -debug -websocket -http-enabled

# Web UI dev server (separate terminal)
cd web
npm install
npm run dev
```

When running the Vite dev server, set `VITE_WS_URL=wss://<host>/ws` and ensure
the Go server is running with TLS and a trusted certificate.

## Documentation

- [docs/installation.md](docs/installation.md) — recommended Raspberry Pi/Linux setup with `script/` (build, install, systemd, TLS, updates).
- [script/README.md](script/README.md) — what each deployment script does and how to run it.
- [docs/websocket_client_protocol.md](docs/websocket_client_protocol.md) — reference for anyone building a custom client.
- Everything else under `docs/` is still relevant but may lag behind recent changes; expect occasional gaps until we finish syncing them back up.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
