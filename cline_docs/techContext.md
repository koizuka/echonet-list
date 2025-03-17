# Technical Context

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
