# System Patterns

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

## Code Organization
- `main.go`: Entry point and main application logic
- `Command.go`: Command parsing and execution
- `CommandProcessor.go`: Command processing and execution
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
