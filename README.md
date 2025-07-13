# ECHONET Lite Device Discovery and Control Tool

This is a Go application for discovering and controlling ECHONET Lite devices on a local network. ECHONET Lite is a communication protocol for smart home devices, primarily used in Japan.

**Author**: @koizuka

<img width="1282" height="524" alt="image" src="https://github.com/user-attachments/assets/31c6acbd-c9f3-4e78-b1e9-0573d08fb9a2" />

## Features

- Automatic discovery of ECHONET Lite devices on the local network
- List all discovered devices with their properties
- Get and set property values on devices
- Persistent storage of discovered devices
- Support for various device types (air conditioners, lighting, floor heating, etc.)
- **Modern Web UI**: React-based interface with real-time updates
- **Device Organization**: Tab-based navigation by location and device groups
- **Visual Controls**: Property-specific UI controls (dropdowns, sliders, toggles)
- **Status Indicators**: Visual feedback for device operation and fault states
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Internationalization**: Multi-language support (English, Japanese) for property descriptions and UI
- **Network Monitoring**: Automatic detection of network interface changes for reliable multicast communication
- WebSocket API for custom client development
- TLS support for secure connections
- Daemon mode for running as a system service

## Quick Start

```bash
# Clone and build
git clone https://github.com/koizuka/echonet-list.git
cd echonet-list

# Build everything (server + web UI)
./script/build.sh

# Or build components separately:
# ./script/build.sh server    # Server only
# ./script/build.sh web       # Web UI only

# Run with Web UI
./echonet-list -websocket -http-enabled
```

Open your browser to `http://localhost:8080`

For a complete getting started guide, see [Quick Start Guide](docs/quick-start.md).

## Documentation

### Getting Started

- [Quick Start Guide](docs/quick-start.md) - Get up and running quickly
- [Installation Guide](docs/installation.md) - Prerequisites, building, and setup
- [Configuration Guide](docs/configuration.md) - All configuration options

### Usage Guides

- [Console UI Usage Guide](docs/console_ui_usage.md) - Terminal interface operation
- [Server Modes Guide](docs/server-modes.md) - Understanding different operation modes
- [Daemon Setup Guide](docs/daemon-setup.md) - Running as a system service

### Web UI & API

- [Web UI Implementation Guide](docs/web_ui_implementation_guide.md) - Web interface details
- [WebSocket Client Protocol](docs/websocket_client_protocol.md) - API protocol specification
- [Client UI Development Guide](docs/client_ui_development_guide.md) - Building custom clients
- [React Hooks Usage Guide](docs/react_hooks_usage_guide.md) - React integration guide
- [Internationalization Guide](docs/internationalization.md) - Multi-language support implementation
- [Error Handling Guide](docs/error_handling_guide.md) - Error handling patterns

### Reference

- [Device Types and Examples](docs/device_types.md) - Supported devices and usage
- [Network Monitoring Guide](docs/network-monitoring.md) - Network interface monitoring and multicast management
- [mkcert Setup Guide](docs/mkcert_setup_guide.md) - TLS certificate setup
- [Troubleshooting Guide](docs/troubleshooting.md) - Common issues and solutions

### Development Resources

For ECHONET Lite specifications (Japanese):

- [ECHONET Lite Protocol Specification](https://echonet.jp/spec_v114_lite/)
- [ECHONET Lite Device Object Specifications](https://echonet.jp/spec_object_rr2/)

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
