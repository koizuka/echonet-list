# ECHONET Lite Device Discovery and Control Tool

This is a Go application for discovering and controlling ECHONET Lite devices on a local network. ECHONET Lite is a communication protocol for smart home devices, primarily used in Japan.

## Features

- Automatic discovery of ECHONET Lite devices on the local network
- List all discovered devices with their properties
- Get property values from specific devices
- Set property values on specific devices
- Persistent storage of discovered devices in a JSON file
- Support for various device types (air conditioners, lighting, floor heating, etc.)
- **Modern Web UI**: Interactive React-based web interface with real-time updates
- **Tab-based Navigation**: Devices organized by installation location and device groups
- **Property Editing**: Visual controls for different property types (dropdowns, numeric inputs)
- **Device Status Indicators**: Visual indicators for operation status and fault conditions
- **Responsive Design**: Mobile-friendly interface that works on phones, tablets, and desktops
- Integrated WebSocket and HTTP server for web UI
- TLS support for secure connections

## Documentation

- [WebSocket Client Protocol](docs/websocket_client_protocol.md) - WebSocketプロトコルの詳細仕様
- [Client UI Development Guide](docs/client_ui_development_guide.md) - WebSocketクライアントUI開発ガイド
- [Web UI Implementation Guide](docs/web_ui_implementation_guide.md) - Web UI実装ガイド
- [React Hooks Usage Guide](docs/react_hooks_usage_guide.md) - React Hooks使用ガイド
- [Error Handling Guide](docs/error_handling_guide.md) - エラーハンドリングガイド
- [mkcert Setup Guide](docs/mkcert_setup_guide.md) - 開発環境の証明書セットアップガイド
- [Device Types and Examples](docs/device_types.md) - サポートされているデバイスタイプと使用例
- [Troubleshooting Guide](docs/troubleshooting.md) - トラブルシューティングガイド
- [Daemon Setup Guide](docs/daemon-setup.md) - デーモンモードのセットアップガイド

## Installation

### Prerequisites

- Go 1.23 or later
- Node.js 18+ and npm (for Web UI development)

### Building from Source

1. Clone the repository
2. Build the Go application:

```bash
go build
```

3. Build the Web UI (optional, pre-built version included):

```bash
cd web
npm install
npm run build
```

## Usage

Run the application:

```bash
./echonet-list [options]
```

### Command Line Options

- `-debug`: Enable debug mode to display detailed communication logs (packet contents, hex dumps, etc.)
- `-log`: Specify a log file name
- `-config`: Specify a TOML configuration file path (default: `config.toml` in the current directory)
- `-websocket`: Enable WebSocket server mode
- `-ws-client`: Enable WebSocket client mode
- `-ws-client-addr`: Specify WebSocket client connection address (default: `ws://localhost:8080/ws`)
- `-ws-both`: Enable both WebSocket server and client modes (for testing)
- `-ws-tls`: Enable TLS for the integrated server
- `-ws-cert-file`: Specify TLS certificate file path
- `-ws-key-file`: Specify TLS private key file path
- `-http-enabled`: Enable HTTP server (integrated with WebSocket server)
- `-http-host`: Specify server host name (default: `localhost`)
- `-http-port`: Specify server port (default: `8080`)
- `-http-webroot`: Specify web root directory (default: `web/bundle`)
- `-daemon`: Enable daemon mode (requires WebSocket server to be enabled)
- `-pidfile`: Specify PID file path (default depends on platform)

Example with debug mode:

```bash
./echonet-list -debug
```

Example with Web UI (integrated server):

```bash
./echonet-list -websocket -http-enabled
```

Example with Web UI and TLS:

```bash
./echonet-list -websocket -http-enabled -ws-tls -ws-cert-file=certs/localhost+2.pem -ws-key-file=certs/localhost+2-key.pem
```

Example as daemon (background service):

```bash
./echonet-list -daemon -websocket -http-enabled
```

### Configuration File

The application supports a TOML configuration file for persistent settings. By default, it looks for `config.toml` in the current directory. You can specify a different file using the `-config` option.

To get started, copy the sample configuration file:

```bash
cp config.toml.sample config.toml
```

Then edit `config.toml` to customize your settings. The configuration file is excluded from version control by `.gitignore`.

Example configuration file (`config.toml.sample`):

```toml
# echonet-list 設定ファイル

# 全般設定
debug = false

# ログ設定
[log]
filename = "echonet-list.log"

# WebSocketサーバー設定
[websocket]
enabled = true
# 定期的なプロパティ更新間隔（例: "1m", "30s", "0" で無効）
periodic_update_interval = "1m"

# TLS設定（HTTPサーバーとWebSocketサーバーで共通）
[tls]
enabled = false
cert_file = "certs/localhost+2.pem"
key_file = "certs/localhost+2-key.pem"

# WebSocketクライアント設定
[websocket_client]
enabled = false
addr = "ws://localhost:8080/ws"  # TLS有効時はwss://を使用

# HTTP Server設定（WebSocketと統合）
[http_server]
enabled = false
host = "localhost"
port = 8080
web_root = "web/bundle"

# デーモンモード設定
[daemon]
enabled = false
pid_file = ""  # 省略時はプラットフォーム別のデフォルトパスを使用
```

Command line options take precedence over configuration file settings.

### WebSocket Support

The application can run in WebSocket server mode, allowing web browsers and other clients to connect and interact with ECHONET Lite devices. It can also run in WebSocket client mode, connecting to another instance of the application running in server mode.

For detailed information about the WebSocket protocol and client development, please refer to:

- [WebSocket Client Protocol](docs/websocket_client_protocol.md)
- [Client UI Development Guide](docs/client_ui_development_guide.md)
- [Error Handling Guide](docs/error_handling_guide.md)

For setting up TLS certificates in development environment, see:

- [mkcert Setup Guide](docs/mkcert_setup_guide.md)

### Integrated Server Support

The application includes an integrated HTTP and WebSocket server that provides both the ECHONET Lite WebSocket API and web UI from a single port. This eliminates port conflicts and simplifies deployment.

-   **Single Port**: Both WebSocket (`/ws`) and HTTP static files are served from the same port
-   **Web Root**: Static files are served from the directory specified by `-http-webroot` or `http_server.web_root` (default: `web/bundle`)
-   **Host**: The server binds to the host specified by `-http-host` or `http_server.host` (default: `localhost`)
-   **Port**: The server listens on the port specified by `-http-port` or `http_server.port` (default: `8080`)
-   **TLS**: If TLS is enabled (`-ws-tls` or `tls.enabled`), both WebSocket and HTTP are served over TLS using the same certificate

**URLs**:
- WebSocket API: `wss://localhost:8080/ws` (with TLS) or `ws://localhost:8080/ws` (without TLS)
- Web UI: `https://localhost:8080/` (with TLS) or `http://localhost:8080/` (without TLS)

**Development Workflow**: During web UI development, you can run the Vite development server independently (`npm run dev` in the `web/` directory) for faster iteration. For integration testing and deployment, enable both WebSocket and HTTP servers in the Go application.

### Daemon Mode

The application supports running as a daemon (background service) on Linux and macOS systems. This is useful for running the ECHONET Lite controller as a persistent service that starts automatically on system boot.

#### Daemon Mode Features

- Runs in the background without console UI
- PID file management for process control
- Automatic log rotation on SIGHUP signal
- Platform-specific default paths:
  - **Linux**: PID file at `/var/run/echonet-list.pid`, logs at `/var/log/echonet-list.log`
  - **macOS**: PID file at `/usr/local/var/run/echonet-list.pid`, logs at `/usr/local/var/log/echonet-list.log`

#### Quick Start with Daemon Mode

```bash
# Run as daemon with Web UI
./echonet-list -daemon -websocket -http-enabled

# Run as daemon with custom PID file
./echonet-list -daemon -websocket -pidfile /tmp/echonet-list.pid
```

#### systemd Service (Linux)

For production deployment on Linux systems with systemd:

1. Install the service file:
```bash
sudo cp systemd/echonet-list.service /etc/systemd/system/
sudo systemctl daemon-reload
```

2. Start and enable the service:
```bash
sudo systemctl start echonet-list
sudo systemctl enable echonet-list
```

3. Check service status:
```bash
sudo systemctl status echonet-list
sudo journalctl -u echonet-list -f
```

For detailed daemon setup instructions, including security configuration, log rotation, and troubleshooting, see the [Daemon Setup Guide](docs/daemon-setup.md).

## Web UI

The application includes a modern, responsive web interface built with React 19 and TypeScript. The Web UI provides an intuitive way to monitor and control ECHONET Lite devices through your web browser.

### Web UI Features

- **Real-time Updates**: Device properties update automatically via WebSocket connection
- **Tab-based Organization**: Devices are organized by installation location and device groups
- **Property Editing**: Interactive controls for different property types:
  - Select controls for enumerated values (operation modes, etc.)
  - Numeric inputs with validation for temperature, power settings
  - Toggle switches for on/off controls
- **Device Status Indicators**: Visual dots showing operation status and fault conditions
- **Compact/Expanded Views**: Cards can be collapsed to show only essential properties
- **Tab Persistence**: Selected tab is remembered across page reloads
- **Device Sorting**: Consistent ordering by device type and installation location
- **Responsive Design**: Works seamlessly on desktop, tablet, and mobile devices
- **Device Aliases**: Friendly names for easier device identification

### Web UI Technology Stack

- **Frontend**: React 19 with TypeScript
- **UI Framework**: shadcn/ui components with Tailwind CSS
- **Build Tool**: Vite with hot module replacement
- **Testing**: Vitest with React Testing Library
- **State Management**: React Hooks (useState, useEffect, useCallback)
- **Real-time Communication**: WebSocket API for live updates

### Web UI Development

For Web UI development, see the detailed guides:

- [Web UI Implementation Guide](docs/web_ui_implementation_guide.md) - Comprehensive implementation details
- [React Hooks Usage Guide](docs/react_hooks_usage_guide.md) - Custom hooks for ECHONET Lite integration

#### Quick Start

1. Start the Go server with WebSocket and HTTP enabled:

```bash
./echonet-list -websocket -http-enabled
```

2. Open your web browser and navigate to `http://localhost:8080`

#### Development Mode

For faster web UI development with hot reloading:

1. Start the Go server (WebSocket only):

```bash
./echonet-list -websocket
```

2. In a separate terminal, start the Vite development server:

```bash
cd web
npm run dev
```

3. Open `http://localhost:5173` for the development version with hot reloading

#### Custom WebSocket URL for Development

When developing the Web UI, you can connect to a different host's ECHONET server by setting the `VITE_WS_URL` environment variable:

```bash
cd web
VITE_WS_URL=wss://192.168.1.100:8080/ws npm run dev
```

This is useful for testing the Web UI against a server running on a different machine or IP address.

#### Building for Production

```bash
cd web
npm run build
```

The built files are output to `web/bundle/` and served by the Go HTTP server.

### Commands

The application provides a command-line interface with the following commands:

#### Discover Devices

```bash
> discover
```

This command broadcasts a discovery message to find all ECHONET Lite devices on the network.

#### List Devices

```bash
> devices
> list
```

Lists all discovered devices. You can filter the results:

```bash
> devices [ipAddress] [classCode[:instanceCode]] [-all|-props] [EPC1 EPC2 ...]
```

Options:

- `ipAddress`: Filter by IP address (e.g., 192.168.0.212)
- `classCode`: Filter by class code (4 hexadecimal digits, e.g., 0130)
- `instanceCode`: Filter by instance code (1-255, e.g., 0130:1)
- `-all`: Show all properties
- `-props`: Show only known properties
- `EPC`: Show only specific properties (2 hexadecimal digits, e.g., 80)

#### Get Property Values

```bash
> get [ipAddress] classCode[:instanceCode] epc1 [epc2...] [-skip-validation]
```

Gets property values from a specific device:

- `ipAddress`: Target device IP address (optional if only one device matches the class code)
- `classCode`: Class code (4 hexadecimal digits, required)
- `instanceCode`: Instance code (1-255, defaults to 1 if omitted)
- `epc`: Property code to get (2 hexadecimal digits, e.g., 80)
- `-skip-validation`: Skip device existence validation (useful for testing timeout behavior)

#### Set Property Values

```bash
> set [ipAddress] classCode[:instanceCode] property1 [property2...]
```

Sets property values on a specific device:

- `ipAddress`: Target device IP address (optional if only one device matches the class code)
- `classCode`: Class code (4 hexadecimal digits, required)
- `instanceCode`: Instance code (1-255, defaults to 1 if omitted)
- `property`: Property to set, in one of these formats:
  - `EPC:EDT` (e.g., 80:30)
  - `EPC` (e.g., 80) - displays available aliases for this EPC
  - Alias name (e.g., `on`) - automatically expanded to the corresponding EPC:EDT
  - Examples:
    - `on` (equivalent to setting operation status to ON)
    - `off` (equivalent to setting operation status to OFF)
    - `80:on` (equivalent to setting operation status to ON)
    - `b0:auto` (equivalent to setting air conditioner to auto mode)

#### Update Device Properties

```bash
> update [ipAddress] [classCode[:instanceCode]]
```

Updates all properties of devices that match the specified criteria:

- `ipAddress`: Filter by IP address (e.g., 192.168.0.212)
- `classCode`: Filter by class code (4 hexadecimal digits, e.g., 0130)
- `instanceCode`: Filter by instance code (1-255, e.g., 0130:1)

This command retrieves all properties listed in the device's GetPropertyMap and updates the local cache. It can be used to refresh the property values of one or multiple devices.

#### Device Aliases

```bash
> alias
> alias <aliasName>
> alias <aliasName> [ipAddress] classCode[:instanceCode] [property1 property2...]
> alias -delete <aliasName>
```

Manages device aliases for easier reference:

- No arguments: Lists all registered aliases
- `<aliasName>`: Shows information about the specified alias
- `<aliasName> [ipAddress] classCode[:instanceCode] [property1 property2...]`: Creates or updates an alias for a device
- `-delete <aliasName>`: Deletes the specified alias

Examples:

```bash
> alias ac 192.168.0.3 0130:1           # Create alias 'ac' for air conditioner at 192.168.0.3
> alias ac 0130                          # Create alias 'ac' for the only air conditioner (if only one exists)
> alias aircon1 0130 living1             # Create alias 'aircon1' for air conditioner with installation location 'living1'
> alias aircon2 0130 on kitchen1         # Create alias 'aircon2' for powered-on air conditioner in the kitchen1
```

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
