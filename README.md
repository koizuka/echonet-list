# ECHONET Lite Device Discovery and Control Tool

This is a Go application for discovering and controlling ECHONET Lite devices on a local network. ECHONET Lite is a communication protocol for smart home devices, primarily used in Japan.

## Features

- Automatic discovery of ECHONET Lite devices on the local network
- List all discovered devices with their properties
- Get property values from specific devices
- Set property values on specific devices
- Persistent storage of discovered devices in a JSON file
- Support for various device types (air conditioners, lighting, floor heating, etc.)

## Installation

### Prerequisites

- Go 1.21 or later

### Building from Source

1. Clone the repository
2. Build the application:

```bash
go build
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
- `-ws-addr`: Specify WebSocket server address (default: `localhost:8080`)
- `-ws-client`: Enable WebSocket client mode
- `-ws-client-addr`: Specify WebSocket client connection address (default: `ws://localhost:8080/ws`)
- `-ws-both`: Enable both WebSocket server and client modes (for testing)
- `-ws-tls`: Enable TLS for WebSocket server
- `-ws-cert-file`: Specify TLS certificate file path
- `-ws-key-file`: Specify TLS private key file path

Example with debug mode:

```bash
./echonet-list -debug
```

Example with WebSocket server and TLS:

```bash
./echonet-list -websocket -ws-tls -ws-cert-file=cert.pem -ws-key-file=key.pem
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
addr = "localhost:8080"

# TLS設定
[websocket.tls]
enabled = false
cert_file = "/path/to/cert.pem"
key_file = "/path/to/key.pem"

# WebSocketクライアント設定
[websocket_client]
enabled = false
addr = "ws://localhost:8080/ws"  # TLS有効時はwss://を使用
```

Command line options take precedence over configuration file settings.

### WebSocket Support

The application can run in WebSocket server mode, allowing web browsers and other clients to connect and interact with ECHONET Lite devices. It can also run in WebSocket client mode, connecting to another instance of the application running in server mode.

#### WebSocket Protocol Documentation

Detailed documentation for the WebSocket protocol is available in [docs/websocket_client_protocol.md](docs/websocket_client_protocol.md). This document provides comprehensive information for developers who want to implement their own WebSocket clients in various programming languages (JavaScript/TypeScript, Python, Java, C#, etc.) to communicate with the ECHONET Lite WebSocket server.

The documentation includes:
- Protocol overview and communication flow
- Message formats and data types
- Server-to-client notifications
- Client-to-server requests
- Server responses
- Implementation guidelines
- Error handling
- TypeScript example implementation

#### Secure WebSocket (WSS) with TLS

For secure WebSocket connections (WSS), you need to provide a TLS certificate and private key. You can generate these using tools like `mkcert` for development:

```bash
# Install mkcert
brew install mkcert  # macOS with Homebrew
mkcert -install      # Install local CA

# Generate certificate for your domain/IP
mkcert localhost 192.168.1.100  # Replace with your server's hostname/IP

# Move the generated files to the certs directory
mkdir -p certs
mv localhost+1.pem localhost+1-key.pem certs/

# Use the generated files
./echonet-list -websocket -ws-tls -ws-cert-file=certs/localhost+1.pem -ws-key-file=certs/localhost+1-key.pem
```

When TLS is enabled, WebSocket clients should connect using `wss://` instead of `ws://`.

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
> alias ac                               # Show information about alias 'ac'
> alias -delete ac                       # Delete alias 'ac'
> alias                                  # List all aliases
```

Using aliases with other commands:
```bash
> get ac 80                    # Get operation status of device with alias 'ac'
> set ac on                    # Turn on device with alias 'ac'
```

#### Debug Mode

```bash
> debug [on|off]
```

Displays or changes the debug mode:

- No arguments: Display current debug mode
- `on`: Enable debug mode
- `off`: Disable debug mode

#### Help

```bash
> help
```

Displays help information about available commands.

#### Quit

```bash
> quit
```

Exits the application.

## Supported Device Types

The application supports various ECHONET Lite device types, including:

- Home Air Conditioner (0x0130)
- Floor Heating (0x027b)
- Single-Function Lighting (0x0291)
- Lighting System (0x02a3)
- Controller (0x05ff)
- Node Profile (0x0ef0)

## Example Use Cases

### Discovering and Controlling an Air Conditioner

1. Start the application
2. Discover devices: `discover`
3. List all devices: `devices`
4. Get the operation status of an air conditioner: `get 0130 80`
5. Turn on the air conditioner: `set 0130 on`
6. Set the temperature to 25°C: `set 0130 b3:19` (25°C in hexadecimal is 0x19)

### Controlling Lights

1. Discover devices: `discover`
2. List all lighting devices: `devices 0291`
3. Turn on a light: `set 0291 on`
4. Turn off a light: `set 0291 off`

### Updating Device Properties

1. Discover devices: `discover`
2. Update all properties of all air conditioners: `update 0130`
3. Update all properties of a specific device: `update 192.168.0.5 0130:1`
4. Check the updated properties: `devices 0130 -all`

## Troubleshooting

### Common Errors

#### Port Already in Use

If you encounter the error message:
```
listen udp :3610: bind: address already in use
```
This indicates that another instance of the application is already running and using UDP port 3610. 

**Resolution:**
1. Find and terminate the other running instance of the application
   - On Linux/macOS: `ps aux | grep echonet-list` to find the process, then `kill <PID>` to terminate it
   - On Windows: Use Task Manager to end the process
2. After stopping the other instance, try running the application again

## References

- [ECHONET Lite Specification](https://echonet.jp/spec_v114_lite/)
- [ECHONET Lite Object Specification](https://echonet.jp/spec_object_rr2/)
