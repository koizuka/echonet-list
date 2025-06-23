# Integration Testing Guide

This guide explains how to run and write integration tests for the ECHONET Lite controller project.

## Overview

Integration tests verify that all components of the system work correctly together:
- Server startup and configuration
- WebSocket communication between server and clients
- Device discovery and management
- HTTP server functionality
- Full stack operation with Web UI

## Running Integration Tests

### Using the Script (Recommended)

```bash
./script/integration-test.sh
```

### Using Go Command Directly

```bash
# Build server first (if needed)
go build

# Build web UI (if needed)
cd web && npm install && npm run build && cd ..

# Run integration tests
go test -v -tags=integration -timeout=5m ./integration/tests/...
```

## Test Structure

Integration tests are located in `/integration/`:

```
integration/
├── README.md           # Integration test overview
├── fixtures/           # Test data files
│   ├── test-devices.json
│   ├── test-aliases.json
│   ├── test-groups.json
│   └── test-config.toml
├── tests/              # Test files
│   ├── server_startup_test.go
│   ├── websocket_connection_test.go
│   ├── device_discovery_test.go
│   ├── http_server_test.go
│   └── full_stack_test.go
└── helpers/            # Test utilities
    ├── test_server.go
    └── test_utils.go
```

## Test Categories

### 1. Server Startup Tests
- Server initialization
- Configuration loading
- Log file creation
- Graceful shutdown

### 2. WebSocket Tests
- Connection establishment
- Message protocol
- Multiple clients
- Reconnection

### 3. Device Discovery Tests
- Loading from fixtures
- Alias resolution
- Group management
- Property updates

### 4. HTTP Server Tests
- Static file serving
- WebSocket upgrade
- CORS handling
- Error responses

### 5. Full Stack Tests
- Complete system flow
- Multi-client synchronization
- Real-time updates
- Error handling

## Writing Integration Tests

### 1. Use the Integration Build Tag

```go
//go:build integration

package tests
```

### 2. Use Test Helpers

```go
func TestMyFeature(t *testing.T) {
    // Start test server
    server := helpers.StartTestServer(t)
    defer server.Stop()
    
    // Connect WebSocket
    conn := helpers.ConnectWebSocket(t, server.WebSocketURL())
    defer conn.Close()
    
    // Your test logic here
}
```

### 3. Clean Up Resources

Always clean up resources using defer:
```go
defer server.Stop()
defer conn.Close()
defer helpers.CleanupTestFiles(t, tempFile)
```

## Test Fixtures

Test fixtures simulate real device data but with minimal content:

- `test-devices.json`: Contains 2 test devices (air conditioner and light)
- `test-aliases.json`: Maps human-readable names to device IDs
- `test-groups.json`: Defines a test group with both devices
- `test-config.toml`: Server configuration for testing

## Debugging Integration Tests

### Enable Debug Output

```bash
go test -v -tags=integration ./integration/tests/... -run TestName
```

### Check Test Logs

Test server logs are written to the temp directory:
```bash
# During test execution, logs are in:
/tmp/TestXXXXXX/test-echonet-list.log
```

### Run Specific Test

```bash
go test -v -tags=integration ./integration/tests/server_startup_test.go -run TestServerStartup
```

## CI/CD Integration

Integration tests run automatically in GitHub Actions:
- On push to main/master/develop branches
- On pull requests
- When server or web code changes

The CI workflow:
1. Builds the server
2. Builds the web UI
3. Runs integration tests with test fixtures
4. Reports results

## Troubleshooting

### Test Timeout
If tests timeout, increase the timeout:
```bash
go test -v -tags=integration -timeout=10m ./integration/tests/...
```

### Port Conflicts
Tests automatically allocate free ports. If you still have issues:
- Check for processes using ports
- Kill any stray test processes

### Missing Web Bundle
If you see "Web bundle not found":
```bash
cd web && npm install && npm run build
```

### Server Build Errors
Ensure you have Go 1.21+ installed:
```bash
go version
```

## Best Practices

1. **Isolate Tests**: Each test should be independent
2. **Use Test Fixtures**: Don't rely on real network devices
3. **Clean Up**: Always clean up resources
4. **Check Errors**: Don't ignore errors in tests
5. **Be Descriptive**: Use clear test names and error messages
6. **Test Edge Cases**: Include error scenarios
7. **Keep Tests Fast**: Use shorter timeouts in test configs