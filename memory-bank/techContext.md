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
- Needs file system access for persistent storage of discovered devices

## WebSocket Communication (Planned)

- **Architecture**: Client/Server model over WebSocket.
- **Server**: Handles ECHONET Lite communication, manages device state, and broadcasts updates.
- **Client**: Connects to the server, sends commands, receives updates. Console UI can act as a client.
- **Startup Options**:
  - `-websocket`: Enable WebSocket server mode.
  - `-ws-addr string`: Specify WebSocket server address (default: `localhost:8080`).
  - `-ws-client`: Enable WebSocket client mode.
  - `-ws-client-addr string`: Specify server address for the client (default: `localhost:8080`).
  - `-ws-both`: Enable both server and client for dogfooding.

- **Protocol**: Custom JSON-based protocol.
  - **Base Message Structure**:
    ```json
    {
      "type": "message_type_string",
      "payload": { ... }, // Message-specific payload
      "requestId": "client_request_id_string" // Optional, for request-response matching
    }
    ```
  - **Device Identification**: Devices identified by a string: `"IP_ADDRESS EOJ_CLASS:EOJ_INSTANCE"`, e.g., `"192.168.0.1 0130:01"`. EOJ parts are hex encoded.
  - **Device Representation**:
    ```json
    {
      "ip": "string",
      "eoj": "string", // e.g., "0130:01"
      "name": "string", // Device class name
      "properties": { "EPC_HEX": "EDT_HEX", ... }, // Map of property EPC (hex) to value (hex)
      "lastSeen": "timestamp" // ISO 8601 format
    }
    ```
  - **Aliases Representation**: Map of alias name (string) to device identifier string.
    ```json
    { "alias_name": "device_identifier_string", ... }
    ```
  - **Error Representation**:
    ```json
    {
      "code": "ERROR_CODE_STRING",
      "message": "string"
    }
    ```

  - **Server -> Client Message Types**:
    - `initial_state`: Sends current devices and aliases upon connection.
      - Payload: `{ "devices": [Device, ...], "aliases": Aliases }`
    - `device_added`: New device discovered.
      - Payload: `{ "device": Device }`
    - `device_updated`: Device properties or status changed. Includes `lastSeen`.
      - Payload: `{ "device": Device }`
    - `device_removed`: Device no longer available (timeout, etc.).
      - Payload: `{ "ip": "string", "eoj": "string" }`
    - `alias_changed`: Alias added, updated, or deleted.
      - Payload: `{ "change_type": "added"|"updated"|"deleted", "alias": "string", "target": "device_identifier_string" }`
    - `property_changed`: Specific property value changed.
      - Payload: `{ "ip": "string", "eoj": "string", "epc": "EPC_HEX", "value": "EDT_HEX" }`
    - `timeout_notification`: Communication timeout with a device.
      - Payload: `{ "ip": "string", "eoj": "string", "code": "ECHONET_TIMEOUT", "message": "string" }`
    - `error_notification`: General server-side error.
      - Payload: `{ "code": "ERROR_CODE_STRING", "message": "string" }`
    - `command_result`: Response to a client request.
      - Payload: `{ "success": bool, "data": any, "error": Error }` (data/error depends on success)

  - **Client -> Server Message Types**:
    - `get_properties`: Request property values.
      - Payload: `{ "targets": ["device_identifier_string", ...], "epcs": ["EPC_HEX", ...] }`
    - `set_properties`: Set property values.
      - Payload: `{ "target": "device_identifier_string", "properties": { "EPC_HEX": "EDT_HEX", ... } }`
    - `update_properties`: Request refresh of device properties (e.g., after discovery).
      - Payload: `{ "targets": ["device_identifier_string", ...] }`
    - `manage_alias`: Add or delete an alias.
      - Payload: `{ "action": "add"|"delete", "alias": "string", "target": "device_identifier_string" }` (target needed for add)
    - `discover_devices`: Trigger ECHONET Lite device discovery.
      - Payload: `{}` (empty)

  - **Error Codes (Examples)**:
    - Client Request Related: `INVALID_REQUEST_FORMAT`, `INVALID_PARAMETERS`, `TARGET_NOT_FOUND`, `ALIAS_OPERATION_FAILED`, `ALIAS_ALREADY_EXISTS`, `INVALID_ALIAS_NAME`, `ALIAS_NOT_FOUND`
    - Server/Communication Related: `ECHONET_TIMEOUT`, `ECHONET_DEVICE_ERROR`, `ECHONET_COMMUNICATION_ERROR`, `INTERNAL_SERVER_ERROR`

## WebSocket Communication (Planned)

- **Protocol**: Custom JSON-based protocol over WebSocket (defined in `protocol/protocol.go`)
  - **Message Types**: `initial_state`, `device_added`, `device_updated`, `device_removed`, `alias_changed`, `property_changed`, `timeout_notification`, `error_notification`, `command_result`, `get_properties`, `set_properties`, `update_properties`, `manage_alias`, `discover_devices`
  - **Payloads**: Defined for each message type (e.g., `InitialStatePayload`, `DeviceAddedPayload`)
  - **Device Identification**: Devices are identified using a string format combining IP address and EOJ, e.g., `192.168.0.1 0130:1`.
- **Server**: Listens for WebSocket connections, communicates with ECHONET Lite devices, and broadcasts state changes to clients.
- **Client**: Connects to the WebSocket server, sends commands, and receives state updates.
- **Startup Options**:
  - `-websocket`: Enable WebSocket server mode.
  - `-ws-addr`: Specify WebSocket server address (default: `localhost:8080`).
  - `-ws-client`: Enable WebSocket client mode.
  - `-ws-client-addr`: Specify WebSocket server address for the client to connect to (default: `localhost:8080`).
  - `-ws-both`: Enable both WebSocket server and client mode for dogfooding.
