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

- **Documentation**: Detailed client developer documentation available in `docs/websocket_client_protocol.md`.
- **Protocol**: Custom JSON-based protocol over WebSocket (defined in `protocol/protocol.go`).
  - **Base Message Structure**:

    ```json
    {
      "type": "message_type_string",
      "payload": { ... }, // Message-specific payload
      "requestId": "client_request_id_string" // Optional, for request-response matching
    }
    ```

  - **Device Identification**: Devices are identified using a string format combining IP address and EOJ (Class Group Code:Class Code:Instance Code), e.g., `"192.168.0.1 0130:01"`. The EOJ parts are hex encoded. The full identifier string (including IP) is also available in the `id` field of the `Device` object.
  - **Device Representation (`Device`)**:

    ```json
    {
      "ip": "string", // IP Address
      "eoj": "string", // EOJ (e.g., "0130:01")
      "name": "string", // Device class name (e.g., "HomeAirConditioner")
      "id": "string", // Full device identifier (e.g., "192.168.0.1 0130:01")
      "properties": { "EPC_HEX": "EDT_BASE64", ... }, // Map of property EPC (hex) to value (Base64 encoded string)
      "lastSeen": "timestamp" // ISO 8601 format (e.g., "2023-10-27T10:00:00Z")
    }
    ```

  - **Server -> Client Message Types**:
    - `initial_state`: Sends current devices, aliases, and groups upon connection. Payload: `InitialStatePayload` (`{ "devices": { "device_id": Device, ... }, "aliases": { "alias_name": "device_id", ... }, "groups": { "group_name": ["device_id", ...], ... } }`)
    - `device_added`: New device discovered. Payload: `DeviceAddedPayload` (`{ "device": Device }`)
    - `device_updated`: Device properties or status changed. Includes `lastSeen`. Payload: `DeviceUpdatedPayload` (`{ "device": Device }`)
    - `device_removed`: Device no longer available (timeout, etc.). Payload: `DeviceRemovedPayload` (`{ "ip": "string", "eoj": "string" }`)
    - `alias_changed`: Alias added, updated, or deleted. Payload: `AliasChangedPayload` (`{ "change_type": "added"|"updated"|"deleted", "alias": "string", "target": "device_id" }`)
    - `group_changed`: Group created, updated (members added/removed), or deleted. Payload: `GroupChangedPayload` (`{ "change_type": "added"|"updated"|"deleted", "group": "string", "devices": ["device_id", ...] }`)
    - `property_changed`: Specific property value changed. Payload: `PropertyChangedPayload` (`{ "ip": "string", "eoj": "string", "epc": "EPC_HEX", "value": "EDT_BASE64" }`)
    - `timeout_notification`: Communication timeout with a device. Payload: `TimeoutNotificationPayload` (`{ "ip": "string", "eoj": "string", "code": "ECHONET_TIMEOUT", "message": "string" }`)
    - `error_notification`: General server-side error. Payload: `ErrorNotificationPayload` (`{ "code": "ERROR_CODE_STRING", "message": "string" }`)
    - `command_result`: Response to a client request. Payload: `CommandResultPayload` (`{ "success": bool, "data": any, "error": Error }`) (data/error depends on success)
    - `property_aliases_result`: Response to `get_property_aliases`. Payload: `PropertyAliasesResultPayload` (`{ "success": bool, "data": PropertyAliasesData, "error": Error }`)
  - **Client -> Server Message Types**:
    - `get_properties`: Request property values. Payload: `GetPropertiesPayload` (`{ "targets": ["device_id_or_alias_or_group", ...], "epcs": ["EPC_HEX", ...] }`)
    - `set_properties`: Set property values. Payload: `SetPropertiesPayload` (`{ "target": "device_id_or_alias_or_group", "properties": { "EPC_HEX": "EDT_BASE64", ... } }`)
    - `update_properties`: Request refresh of device properties. Payload: `UpdatePropertiesPayload` (`{ "targets": ["device_id_or_alias_or_group", ...] }`)
    - `manage_alias`: Add or delete an alias. Payload: `ManageAliasPayload` (`{ "action": "add"|"delete", "alias": "string", "target": "device_id" }`) (target needed for add)
    - `manage_group`: Add/remove devices from a group, or delete a group. Payload: `ManageGroupPayload` (`{ "action": "add"|"remove"|"delete"|"list", "group": "string", "devices": ["device_id_or_alias", ...] }`) (devices needed for add/remove)
    - `discover_devices`: Trigger ECHONET Lite device discovery. Payload: `DiscoverDevicesPayload` (`{}`)
    - `get_property_aliases`: Request property aliases for a device class. Payload: `GetPropertyAliasesPayload` (`{ "classCode": "CLASS_CODE_HEX" }`) (e.g., "0130")
  - **Error Codes**: Defined in `protocol/protocol.go` (e.g., `INVALID_REQUEST_FORMAT`, `TARGET_NOT_FOUND`, `ECHONET_TIMEOUT`).
- **Server**: Listens for WebSocket connections, communicates with ECHONET Lite devices via the `echonet_lite` package, manages device state (including aliases and groups), and broadcasts state changes to connected clients.
- **Client**: Connects to the WebSocket server, sends command messages (like `get_properties`, `set_properties`, `manage_alias`, `manage_group`), and receives notification messages (like `device_added`, `property_changed`, `group_changed`).
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
