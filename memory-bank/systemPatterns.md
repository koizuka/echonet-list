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

## Key Technical Decisions

- Go language for cross-platform compatibility and performance
- UDP for network communication with ECHONET Lite devices
- JSON for persistent storage of discovered devices
- Command-line interface for simplicity and scriptability
- Modular design with separate packages for protocol implementation
- Alias system for easier device reference in commands
- Notification system for loose coupling, allowing frontend components to receive state changes:
  - Device addition notifications inform when new devices are discovered
  - Timeout notifications alert when device communication fails
  - Property change notifications (planned) will enable real-time state updates
- **WebSocket Client/Server Architecture (Planned)**:
  - The application can be split into a server and a client communicating via WebSocket.
  - **Server**: Handles ECHONET Lite communication (discovery, property get/set) and manages device state. It exposes a WebSocket endpoint.
  - **Client**: Connects to the WebSocket server to interact with devices. The console UI can act as a WebSocket client.
  - This architecture allows for separating the core ECHONET Lite logic from the user interface, enabling different types of clients (e.g., console, web UI).

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
- `server/`: Package containing server implementation
  - `server.go`: Server implementation
  - `LogManager.go`: Log management functionality
- `protocol/`: Package containing protocol definitions
  - `protocol.go`: Protocol interface definitions
- `echonet_lite/`: Package containing ECHONET Lite protocol implementation
  - `DeviceAliases.go`: Device alias management and storage
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
