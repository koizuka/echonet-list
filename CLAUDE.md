# ECHONET Lite Project Guidelines

## Build Commands
- Build: `go build`
- Run: `./echonet-list [-debug]`
- Run tests: `go test ./...`
- Run specific test: `go test ./echonet_lite -run TestOperationStatus_Encode`
- Format code: `go fmt ./...`
- Check code: `go vet ./...`

## Development Workflow
- Always run `go fmt ./...` after making changes to Go files
- After significant changes, run `go vet ./...` to check for potential issues

## Code Style Guidelines
- **File Organization**: Package main in root, echonet_lite package for protocol implementation
- **Imports**: Group standard library, then external packages, then local packages
- **Naming**: CamelCase for exported entities, camelCase for unexported entities
- **Comments**: Comments for exported functions start with function name
- **Types**: Define custom types for specialized uses (e.g., `OperationStatus`)
- **Error Handling**: Return errors for recoverable failures, use proper error types
- **Testing**: Table-driven tests with descriptive names for each test case
- **Formatting**: Standard Go formatting with proper indentation
- **Hexadecimal**: Use `0x` prefix for clarity (e.g., `0x80` not just `80`)