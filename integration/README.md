# Integration Tests

This directory contains integration tests for the ECHONET Lite controller project.

## Overview

The integration tests verify that the complete system works correctly by:

- Starting a real server instance with test fixtures
- Testing WebSocket communication
- Verifying device discovery and management
- Testing HTTP server functionality
- Validating full stack integration

## Directory Structure

```
integration/
├── fixtures/           # Test data files
│   ├── test-devices.json     # Sample device data
│   ├── test-aliases.json     # Sample device aliases
│   ├── test-groups.json      # Sample device groups
│   └── test-config.toml      # Test configuration
├── helpers/            # Test utilities
│   ├── test_server.go        # Test server management
│   └── test_utils.go         # Common test utilities
├── tests/              # Integration test files
│   ├── server_startup_test.go       # Server lifecycle tests
│   ├── websocket_connection_test.go # WebSocket communication tests
│   ├── device_discovery_test.go     # Device management tests
│   ├── http_server_test.go          # HTTP server tests
│   └── full_stack_test.go           # End-to-end integration tests
├── Makefile            # Build and test automation
└── README.md           # This file
```

## Running Integration Tests

### Prerequisites

- Go 1.21 or later
- Node.js 20 or later
- npm

### Methods to Run Tests

#### 1. Using the Integration Test Script

```bash
# From project root
./script/integration-test.sh
```

This script automatically:
- Builds the Go server
- Builds the Web UI
- Runs all integration tests

#### 2. Using Make

```bash
# From integration directory
cd integration

# Run all tests
make test

# Run specific test suites
make test-server   # Server startup tests only
make test-ws       # WebSocket tests only

# Build targets
make build-all     # Build both server and web UI
make build-server  # Build Go server only
make build-web     # Build Web UI only

# Clean built artifacts
make clean
```

#### 3. Using Go Test Directly

```bash
# From project root
go build -o echonet-list .
cd web && npm ci && npm run build && cd ..
go test -v -tags=integration -timeout=5m ./integration/tests/...
```

### Running Specific Tests

```bash
# Run only server startup tests
go test -v -tags=integration -run "TestServer" ./integration/tests/...

# Run only WebSocket tests
go test -v -tags=integration -run "TestWebSocket" ./integration/tests/...

# Run with verbose output and race detection
go test -v -race -tags=integration ./integration/tests/...
```

## Test Structure

### Test Server

The `TestServer` helper provides:
- Automatic port allocation to avoid conflicts
- Test fixture loading
- Server lifecycle management
- URL generation for WebSocket and HTTP connections

### Test Fixtures

#### test-devices.json
Contains minimal device data with:
- Air Conditioner (class 0130) 
- Two Single Function Lights (class 0291)
- Base64-encoded property values

#### test-aliases.json
Maps human-readable names to device IDs:
- "Test Air Conditioner" → device ID
- "Test Light 1" → device ID
- "Test Light 2" → device ID

#### test-groups.json
Defines device groups:
- "@Test Lights" - contains both lights
- "@All Test Devices" - contains all devices

#### test-config.toml
Test-specific configuration:
- Debug mode enabled
- TLS disabled for simplicity
- Random port allocation
- Test mode enabled

### Test Categories

#### 1. Server Startup Tests
- Server creation and configuration
- Multiple server instances (port conflict avoidance)
- Configuration loading and validation

#### 2. WebSocket Tests
- Basic connection establishment
- Multiple concurrent connections
- Message exchange and protocol validation
- Connection timeout handling

#### 3. Device Discovery Tests
- Test fixture loading
- Device alias resolution
- Property reading and writing
- Group management

#### 4. HTTP Server Tests
- Static file serving
- CORS handling
- Health checks
- Error responses

#### 5. Full Stack Tests
- Complete workflow testing
- Multi-client synchronization
- Real-time updates
- Error recovery

## Continuous Integration

Integration tests run automatically in GitHub Actions when:
- Go code changes are detected
- Web UI code changes are detected
- Integration test files are modified

The CI workflow:
1. Builds the Go server
2. Builds the Web UI
3. Runs all integration tests
4. Reports results and coverage

## Troubleshooting

### Common Issues

#### Port Conflicts
Tests use random port allocation, but conflicts can still occur:
```bash
# Kill any running instances
pkill -f echonet-list
```

#### Build Failures
Ensure all dependencies are installed:
```bash
# Go dependencies
go mod download

# Node.js dependencies
cd web && npm ci
```

#### Test Timeout
Increase timeout for slow environments:
```bash
go test -v -tags=integration -timeout=10m ./integration/tests/...
```

### Debug Mode

Enable verbose logging in tests:
```bash
# Set debug environment variable
DEBUG=1 go test -v -tags=integration ./integration/tests/...
```

### Log Files

Integration tests create temporary log files:
- Location: `/tmp/echonet-test-*/test-echonet-list.log`
- Cleaned up automatically after tests

## Writing New Tests

### Test File Structure

```go
//go:build integration

package tests

import (
    "echonet-list/integration/helpers"
    "testing"
)

func TestMyNewFeature(t *testing.T) {
    // Create test server
    server, err := helpers.NewTestServer()
    helpers.AssertNoError(t, err, "Test server creation")
    
    // Set test fixtures
    server.SetTestFixtures(devicesFile, aliasesFile, groupsFile)
    
    // Start server
    err = server.Start()
    helpers.AssertNoError(t, err, "Server startup")
    defer server.Stop()
    
    // Your test logic here...
}
```

### Best Practices

1. **Use build tags**: Always include `//go:build integration`
2. **Clean up resources**: Use `defer` for server cleanup
3. **Use helpers**: Leverage existing helper functions
4. **Descriptive names**: Use clear test and assertion messages
5. **Isolated tests**: Each test should be independent
6. **Timeouts**: Set appropriate timeouts for operations

### Helper Functions

Available in `helpers` package:
- `NewTestServer()` - Create test server instance
- `NewWebSocketConnection()` - Create WebSocket client
- `AssertEqual()`, `AssertNoError()` - Test assertions
- `WaitForCondition()` - Wait for async operations
- `CreateTempFile()` - Create temporary test files

## Contributing

When adding new integration tests:

1. Follow the existing test structure
2. Add appropriate build tags
3. Use the helper utilities
4. Update this README if adding new test categories
5. Ensure tests pass in CI environment