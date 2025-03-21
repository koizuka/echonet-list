# Project Brief

## Core Requirements

- Create a Go application for discovering and controlling ECHONET Lite devices on a local network
- Support various ECHONET Lite device types (air conditioners, lighting, floor heating, etc.)
- Provide a command-line interface for user interaction
- Enable automatic discovery of devices on the local network
- Allow users to get and set property values on specific devices
- Maintain persistent storage of discovered devices
- Provide alias functionality for easier reference to devices

## Project Goals

- Create a unified interface for ECHONET Lite device control
- Make it easy for users to interact with smart home devices
- Ensure reliable communication with proper error handling and retransmission
- Design a modular architecture that can be extended to support new device types
- Implement a notification system for real-time state changes
- Prepare for future architecture split with WebSocket server and multiple UI clients

## Scope

- Focus on ECHONET Lite protocol implementation
- Support common device types used in smart homes
- Provide a command-line interface initially, with plans for a web interface
- Implement core functionality for device discovery, property reading/writing, and state management
- Design for cross-platform compatibility (macOS, Linux, Windows)

## Success Criteria

- Successfully discover ECHONET Lite devices on the local network
- Correctly interpret and display device properties
- Reliably send commands to devices and handle responses
- Persist device information between application restarts
- Provide clear error messages and recovery options
- Support common ECHONET Lite device types
- Implement a notification system for device state changes
