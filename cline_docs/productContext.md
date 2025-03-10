# Product Context

## Project Purpose
This is a Go application for discovering and controlling ECHONET Lite devices on a local network. ECHONET Lite is a communication protocol for smart home devices, primarily used in Japan.

## Problems Solved
- Provides a unified interface for discovering and controlling various ECHONET Lite devices
- Enables automatic discovery of ECHONET Lite devices on the local network
- Allows users to get and set property values on specific devices
- Maintains persistent storage of discovered devices
- Supports various device types (air conditioners, lighting, floor heating, etc.)

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
   - `help`: Display help information
   - `quit`: Exit the application
5. The application supports various device types including air conditioners, lighting, floor heating, etc.
