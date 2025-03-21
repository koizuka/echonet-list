# Product Context

This file expands on the core project definition in [projectbrief.md](./projectbrief.md) to provide detailed context about the product's purpose and functionality.

## Project Purpose

This is a Go application for discovering and controlling ECHONET Lite devices on a local network. ECHONET Lite is a communication protocol for smart home devices, primarily used in Japan.

## Problems Solved

- Provides a unified interface for discovering and controlling various ECHONET Lite devices
- Enables automatic discovery of ECHONET Lite devices on the local network
- Allows users to get and set property values on specific devices
- Maintains persistent storage of discovered devices
- Supports various device types (air conditioners, lighting, floor heating, etc.)
- Provides alias functionality for easier reference to devices

## How It Works

1. The application discovers ECHONET Lite devices on the local network using UDP broadcast
2. It maintains a list of discovered devices with their properties
3. Users can interact with devices through a command-line interface
4. Commands include:
   - `discover`: Find all ECHONET Lite devices on the network
   - `devices`/`list`: List all discovered devices with filtering options
   - `get`: Get property values from specific devices
   - `set`: Set property values on specific devices
   - `update`: Update all properties of devices
   - `alias`: Manage device aliases (create, view, delete, list)
   - `debug`: Display or change debug mode
   - `help`: Display help information
   - `quit`: Exit the application
5. The application supports various device types including air conditioners, lighting, floor heating, etc.
6. Users can create aliases for devices to reference them more easily in commands
7. The application includes a notification system that enables loose coupling between components:
   - Device discovery notifications inform when new devices are found
   - Timeout notifications alert when device communication fails
   - Property change notifications (planned) will allow frontend components to receive real-time state changes
8. The notification system is designed to support a future architecture where:
   - ECHONET Lite processing will be handled by a WebSocket server
   - Console UI and Web UI will connect to this server to receive state updates
   - This loose coupling enables multiple frontends to react to state changes independently
