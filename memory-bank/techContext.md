# Technical Context

This file provides details about the technical environment and constraints of the project, complementing the core project definition in [projectbrief.md](./projectbrief.md).

## Technologies Used

- **Programming Language**: Go (version 1.20 as specified in go.mod)
- **Dependencies**:
  - github.com/chzyer/readline: For command-line interface
  - golang.org/x/sys: For system-level operations
- **Version Control**: Git (version 2.48.1)
- **Package Management**: Go modules
- **Operating System**: macOS, Linux, Windows
- **Package Manager**: Homebrew (used for updating Git)

## Development Setup

- Go 1.20 or later
- Git for version control
- Command-line environment for running and testing the application
- Local network with ECHONET Lite devices for full testing

## Technical Constraints

- Must maintain compatibility with the ECHONET Lite protocol specification
- Needs to work with various ECHONET Lite device types
- Requires UDP network access for device discovery and communication
- Needs file system access for persistent storage of discovered devices and configuration.

## WebSocket Communication

- **Purpose**: Enables remote control and monitoring of ECHONET Lite devices via WebSocket.
- **Protocol Definition**: The custom JSON-based protocol is defined in `protocol/protocol.go`.
- **Detailed Documentation**: For comprehensive client development details, including all message types, payloads, data formats, and error codes, refer to the primary specification document: **[`docs/websocket_client_protocol.md`](../docs/websocket_client_protocol.md)**.
- **Server**: Listens for WebSocket connections, interacts with ECHONET Lite devices (using the `echonet_lite` package), manages state (devices, aliases, groups), and broadcasts updates to clients. Implemented in `server/websocket_server.go`.
- **Client**: Connects to the server, sends commands (e.g., get/set properties, manage aliases/groups), and receives state notifications. Implemented in `client/websocket_client.go`.
- **Startup Options**:
  - `-websocket`: Enable WebSocket server mode.
  - `-ws-addr`: Specify WebSocket server address (default: `localhost:8080`).
  - `-ws-client`: Enable WebSocket client mode.
  - `-ws-client-addr`: Specify WebSocket server address for the client to connect to (default: `localhost:8080`).
  - `-ws-both`: Enable both WebSocket server and client mode for dogfooding.
  - `-ws-tls`: Enable TLS for WebSocket server.
  - `-ws-cert-file`: Specify TLS certificate file path.
  - `-ws-key-file`: Specify TLS private key file path.
- **TLS Support**: WebSocket server can be configured to use TLS (WSS) for secure connections.
  - When TLS is enabled, WebSocket clients should connect using `wss://` instead of `ws://`.
  - The application automatically updates client connection URLs when TLS is enabled.

## Configuration File

- **Format**: TOML (Tom's Obvious, Minimal Language)
- **Default Path**: `config.toml` in the current directory
- **Command Line Override**: `-config` option to specify a different file path
- **Settings**:
  - General settings (debug mode)
  - Log settings (filename)
  - WebSocket server settings (enabled, address, TLS)
  - WebSocket client settings (enabled, address)
  - HTTP server settings (planned for Web UI):
    - `http_enabled = true/false`
    - `http_port = 8081` (example)
    - `http_webroot = "server/webroot"` (example)
- **Priority**: Command line arguments take precedence over configuration file settings
- **Sample File**: `config.toml.sample` is provided as a template
