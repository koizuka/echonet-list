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

```bash
git clone https://github.com/koizuka/echonet-list.git
cd echonet-list

# Build everything (server + web UI)
./script/build.sh

# Run with Web UI + HTTP proxy
./echonet-list -websocket -http-enabled
```

Open your browser to `http://localhost:8080`.

## Deployment workflow (maintained)

For new installations we recommend the following path:

1. Prepare a systemd-based Linux host with Go 1.23+, Node.js 18+, `mkcert`, and sudo access.
2. Clone this repository into `/opt/echonet-list` (or similar) and run `./script/build.sh server` + `./script/build.sh web`.
3. Generate TLS files with `mkcert` under `certs/` so the Web UI/WebSocket endpoint can run over HTTPS/WSS.
4. Run `sudo ./script/install-systemd.sh` to create the `echonet-list` service and copy binaries/web assets/certificates.
5. Install the mkcert CA (`mkcert -CAROOT`) on every browser/device that should trust the UI.
6. Keep the instance up to date with `./script/auto-update.sh` (optionally wired into a systemd timer).

All of these steps are written out in detail inside [docs/installation.md](docs/installation.md).

## Documentation

- [docs/installation.md](docs/installation.md) — deployment + operations guide described above.
- [docs/websocket_client_protocol.md](docs/websocket_client_protocol.md) — reference for anyone building a custom client.
- Everything else under `docs/` is still relevant but may lag behind recent changes; expect occasional gaps until we finish syncing them back up.

## systemd Management Scripts

For easy setup on Raspberry Pi/Ubuntu systems:

- **Install**: `sudo ./script/install-systemd.sh`
- **Uninstall**: `sudo ./script/uninstall-systemd.sh`  
- **Update**: `sudo ./script/update.sh`

See [script/README.md](script/README.md) for details.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
