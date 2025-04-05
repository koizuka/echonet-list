# System Patterns

This file describes the system architecture and code organization patterns used in the project, building on the foundation defined in [projectbrief.md](./projectbrief.md).

## Architecture

- The application is built in Go with a modular architecture
- Main components:
  - Command-line interface for user interaction
  - Device discovery and management
  - ECHONET Lite protocol implementation
  - Session management for communication
  - UDP connection handling
  - Device alias management
  - Device group management
  - WebSocket server for remote client access

## Key Technical Decisions

- Go language for cross-platform compatibility and performance
- UDP for network communication with ECHONET Lite devices
- JSON for persistent storage of discovered devices
- Command-line interface for simplicity and scriptability
- Modular design with separate packages for protocol implementation
- Alias system for easier device reference in commands
- Group system for batch operations on multiple devices
- Notification system for loose coupling, allowing frontend components to receive state changes:
  - Device addition notifications inform when new devices are discovered
  - Timeout notifications alert when device communication fails
  - Property change notifications (planned) will enable real-time state updates
- **WebSocket Client/Server Architecture**:
  - The application can be split into a server and a client communicating via WebSocket.
  - **Server**: Handles ECHONET Lite communication (discovery, property get/set) and manages device state. It exposes a WebSocket endpoint.
  - **Client**: Connects to the WebSocket server to interact with devices. The console UI can act as a WebSocket client.
  - This architecture allows for separating the core ECHONET Lite logic from the user interface, enabling different types of clients (e.g., console, web UI).
  - **WebSocketTransport Interface**: Abstracts the WebSocket server's network layer, making it testable by allowing mock implementations for testing.

## Code Organization

- `main.go`: Entry point and main application logic
- `console/`: Package containing console UI implementation
  - `Command.go`: Command parsing and execution
  - `CommandTable.go`: Command definition table and help display functionality
  - `CommandProcessor.go`: Command processing and execution
  - `Completer.go`: Command line completion functionality
  - `Completer_test.go`: Tests for command line completion
  - `ConsoleProcess.go`: Main console UI process
- `client/`: Package containing client implementation
  - `client.go`: Client interface definitions
  - `ECHONETListClientProxy.go`: Client proxy implementation
  - `interfaces.go`: Interface definitions
  - `websocket_client.go`: WebSocket client implementation
- `server/`: Package containing server implementation
  - `server.go`: Server implementation
  - `LogManager.go`: Log management functionality
  - `transport.go`: WebSocket transport interface
  - `websocket_server.go`: WebSocket server implementation
  - `websocket_server_handlers_properties.go`: Property-related handlers
  - `websocket_server_handlers_management.go`: Alias and group management handlers
  - `websocket_server_handlers_discovery.go`: Device discovery handlers
- `protocol/`: Package containing protocol definitions
  - `protocol.go`: Protocol interface definitions
  - `protocol_test.go`: Tests for protocol functionality
- `config/`: Package containing configuration functionality
  - `config.go`: Configuration loading and parsing
- `echonet_lite/`: Package containing ECHONET Lite protocol implementation
  - `DeviceAliases.go`: Device alias management and storage
  - `DeviceGroups.go`: Device group management and storage
  - `Devices.go`: Device management and storage
  - `Devices_test.go`: Tests for device management
  - `echonet_lite.go`: Core ECHONET Lite message handling
  - `ECHONETLiteHandler.go`: Main handler for ECHONET Lite protocol
  - `EOJ.go`: ECHONET Object implementation
  - `Filter_test.go`: Tests for filtering functionality
  - `FloorHeating.go`: Floor heating device implementation
  - `HomeAirConditioner.go`: Air conditioner device implementation
  - `IPAndEOJ.go`: IP address and EOJ handling
  - `NodeProfileObject.go`: Node profile object implementation
  - `ProfileSuperClass.go`: Base class for profiles
  - `ProfileSuperClass_test.go`: Tests for profile super class
  - `Property.go`: Property handling
  - `Session.go`: Session management for ECHONET Lite communication
  - `SingleFunctionLighting.go`: Lighting device implementation
  - `log/`: Logging functionality
    - `logger.go`: Logger implementation
  - `network/`: Network communication
    - `network.go`: Network utility functions
    - `UDPConnection.go`: UDP communication handling
- `docs/`: Documentation
  - `websocket_client_protocol.md`: WebSocket protocol documentation for client developers
- `certs/`: TLS certificates for WebSocket server
  - `localhost+2.pem`: Certificate file
  - `localhost+2-key.pem`: Private key file

## WebSocket Server Architecture

The WebSocket server is designed with a modular architecture to improve maintainability and testability:

1. **Interface-Based Design**:
   - `WebSocketTransport` interface abstracts the WebSocket server's network layer
   - This allows for mock implementations during testing
   - The real implementation uses Gorilla WebSocket library

2. **File Organization**:
   - `websocket_server.go`: Core server structure and main methods
   - `websocket_server_handlers_properties.go`: Property-related handlers (get, set, update)
   - `websocket_server_handlers_management.go`: Alias and group management handlers
   - `websocket_server_handlers_discovery.go`: Device discovery handlers

3. **Message Handling**:
   - Each message type has a dedicated handler method
   - Messages are parsed and validated before processing
   - Responses are sent back to the client using a common message format
   - Property aliases information can be retrieved via WebSocket using the `get_property_aliases` message type

4. **Notification System**:
   - Server listens for notifications from the ECHONET Lite handler
   - Notifications are broadcast to all connected clients
   - Supported notifications: device added, device timeout, property changed

5. **Security**:
   - TLS support for secure WebSocket connections (WSS)
   - Certificate and private key paths configurable via options

## WebSocket Protocol Message Types

The WebSocket protocol supports the following message types:

### Client -> Server (Requests)
- `get_properties`: Get property values for specified devices and EPCs
- `set_properties`: Set property values for a specified device
- `update_properties`: Request the server to update property information for specified devices
- `manage_alias`: Add or delete device aliases
- `manage_group`: Add, remove, delete, or list device groups
- `discover_devices`: Trigger device discovery
- `get_property_aliases`: Get property aliases for a specified class code

### Server -> Client (Responses and Notifications)
- `command_result`: Response to client requests
- `property_aliases_result`: Response to `get_property_aliases` request (EPCごとにグループ化されたプロパティエイリアス情報)
- `initial_state`: Initial state sent when a client connects
- `device_added`: Notification when a new device is discovered
- `device_updated`: Notification when device information is updated
- `device_removed`: Notification when a device is removed
- `alias_changed`: Notification when a device alias is changed
- `group_changed`: Notification when a device group is changed
- `property_changed`: Notification when a property value changes
- `timeout_notification`: Notification when a device communication times out
- `error_notification`: Notification of an error

This architecture allows for better separation of concerns, easier testing, and improved maintainability.
