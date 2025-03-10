# System Patterns

## Architecture
- The application is built in Go with a modular architecture
- Main components:
  - Command-line interface for user interaction
  - Device discovery and management
  - ECHONET Lite protocol implementation
  - Session management for communication
  - UDP connection handling

## Key Technical Decisions
- Go language for cross-platform compatibility and performance
- UDP for network communication with ECHONET Lite devices
- JSON for persistent storage of discovered devices
- Command-line interface for simplicity and scriptability
- Modular design with separate packages for protocol implementation

## Code Organization
- `main.go`: Entry point and main application logic
- `Command.go`: Command parsing and execution
- `Devices.go`: Device management and storage
- `Session.go`: Session management for ECHONET Lite communication
- `UDPConnection.go`: UDP communication handling
- `echonet_lite/`: Package containing ECHONET Lite protocol implementation
  - `echonet_lite.go`: Core ECHONET Lite message handling
  - `EOJ.go`: ECHONET Object implementation
  - `Property.go`: Property handling
  - `ProfileSuperClass.go`: Base class for profiles
  - Device-specific implementations (e.g., `HomeAirConditioner.go`, `FloorHeating.go`)
