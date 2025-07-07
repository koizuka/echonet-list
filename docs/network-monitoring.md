# Network Monitoring Implementation

## Overview

This document describes the network monitoring functionality implemented in ECHONET List to detect network interface changes and maintain accurate local IP address caches for proper self-packet filtering in UDP multicast communication.

## Features

### 1. Network Interface Monitoring

- **Interface Detection**: Monitors network interface state changes (up/down, IP changes)  
- **Local IP Cache**: Maintains up-to-date cache of local IP addresses
- **Self-Packet Filtering**: Enables accurate detection of packets sent by the local node

### 2. Multicast Group Management

- **Automatic Group Join**: Uses `net.ListenMulticastUDP()` for automatic multicast group joining
- **IGMP Compliance**: Relies on OS kernel's IGMP implementation for group membership
- **No Manual Refresh**: Multicast group membership is entirely handled by the OS kernel

### 3. Simplified Configuration

- **Single Setting**: Only `monitor_enabled` is needed
- **Automatic Features**: Network monitoring runs independently when enabled

## Configuration

### TOML Configuration File

```toml
[network]
# Enable network interface monitoring
monitor_enabled = true
```

### Recommended Settings

#### All Network Types

```toml
[network]
monitor_enabled = true
```

## Implementation Details

### Network Layer (UDPConnection)

**File**: `echonet_lite/network/UDPConnection.go`

#### Key Components

- `NetworkMonitor` struct: Manages network interface monitoring state
- `NetworkMonitorConfig` struct: Simple configuration (enabled flag only)
- `networkMonitorLoop()`: Main event loop for network interface monitoring
- `monitorNetworkChanges()`: Detects and handles network interface changes

#### Key Methods

- `CreateUDPConnection()`: Creates UDP connection with optional network monitoring
- `IsNetworkMonitorEnabled()`: Returns current network monitoring status

### Session Layer Integration

**File**: `echonet_lite/handler/Session.go`

- `CreateSession()`: Creates session with network monitoring configuration
- Network monitoring is configured through handler options

### Handler Integration

**File**: `echonet_lite/handler/ECHONETLiteHandler.go`

- Extended `ECHONETLieHandlerOptions` to include `NetworkMonitorConfig`
- Passes configuration through to session layer

### Application Integration

**Files**: `server/server.go`, `main.go`

- `NewServer()`: Creates server with network monitoring configuration
- Parses configuration from TOML file
- Applies network monitoring settings to handler

## Protocol Considerations

### Network Interface Monitoring

- **OS Integration**: Uses standard Go `net` package interfaces
- **No Network Traffic**: Monitoring operates entirely through OS APIs
- **Interface Queries**: Polls network interface state periodically
- **Local Operation**: No network packets are sent for monitoring

### Multicast Group Management

- **IGMP Compliance**: Relies on OS kernel's IGMP implementation for multicast group membership
- **Automatic**: Multicast group joining/leaving handled by `net.ListenMulticastUDP()`
- **No Manual Intervention**: Application does not send IGMP packets directly

### Network Traffic Impact

- **Zero Network Overhead**: Network monitoring generates no network traffic
- **No Protocol Interference**: Doesn't affect ECHONET Lite communication
- **Local Only**: All monitoring operations are local system calls

## Monitoring and Diagnostics

### Log Messages

#### Info Level

```
INFO ネットワーク監視が開始されました
INFO ネットワークインターフェースの変更を検出しました
INFO ネットワーク監視が停止されました
```

#### Debug Level

```
DEBUG ローカルIPアドレスを更新しました count=2
DEBUG ネットワーク監視ループを終了します
```

#### Warning Level

```
WARN ネットワークインターフェース情報の取得に失敗 err="operation not permitted"
WARN ローカルIPアドレスの再取得に失敗 err="no route to host"
```

### Status Monitoring

Use the `IsNetworkMonitorEnabled()` method to check current status:

```go
enabled := connection.IsNetworkMonitorEnabled()
if enabled {
    fmt.Println("Network monitoring is enabled")
}
```

## Troubleshooting

### Common Issues

#### 1. Network Monitoring Not Starting

- **Symptom**: No network monitoring log messages
- **Cause**: `monitor_enabled = false` in configuration
- **Solution**: Enable network monitoring in configuration file

#### 2. Network Change Detection Not Working

- **Symptom**: Network changes not detected
- **Cause**: Insufficient permissions to query network interfaces
- **Solution**: Check application permissions for network interface queries

#### 3. Interface Query Failures

- **Symptom**: Warning messages about interface information retrieval
- **Cause**: OS permissions or network driver issues
- **Solution**: Run with appropriate permissions or check network drivers

#### 4. Monitoring Loop Issues

- **Symptom**: Network monitoring stops unexpectedly
- **Cause**: Context cancellation or system resource limits
- **Solution**: Ensure proper context management and system resources

### Network Requirements

#### Firewall Settings

- **Outbound UDP**: Allow port 3610 to 224.0.23.0
- **Inbound UDP**: Allow port 3610 from multicast group
- **IGMP**: Allow IGMP traffic for multicast group management

#### Network Interface Requirements

- **Multicast Support**: Network interface must support multicast
- **IGMP**: Router must support IGMP for multicast routing
- **Permissions**: Application may need network interface query permissions

## Testing

### Unit Tests

Run network monitoring specific tests:

```bash
go test ./echonet_lite/network -v -run TestNetworkMonitor
```

### Integration Testing

Test with configuration file:

```bash
./echonet-list -config config.toml -debug
```

### Network Simulation

Test network changes:

```bash
# Disable network interface (requires admin privileges)
sudo ifconfig en0 down
sleep 5
sudo ifconfig en0 up
```

## Backward Compatibility

- All existing APIs remain unchanged
- Default behavior: network monitoring disabled
- Configuration is optional and backward compatible
- Network monitoring operates independently when enabled

## Breaking Changes (Recent Updates)

⚠️ **Configuration Changes Required**

If you have an existing configuration file with multicast settings, you must update to the new network section:

```diff
-[multicast]
-keep_alive_enabled = true
+[network]
+monitor_enabled = true
```

The multicast-specific configuration has been replaced with general network monitoring configuration to better reflect the actual functionality.

## Performance Impact

### Memory Usage

- **Additional Memory**: <1KB per connection for network monitoring state
- **Goroutines**: 1 additional goroutine per connection when enabled
- **Timers**: 1 timer per connection (network interface check, 10-second interval)

### CPU Usage

- **Background Processing**: Minimal CPU usage for single timer
- **Network Monitoring**: Low-frequency interface checks (every 10 seconds)
- **Interface Queries**: Minimal overhead for system calls

### Network Impact

- **Bandwidth**: Zero network traffic generated by monitoring
- **No Network Packets**: All monitoring is local system operations
- **No Effect**: On existing ECHONET Lite protocol communications

## Future Enhancements

### Potential Improvements

1. **Network Quality Metrics**: Track interface stability and changes
2. **Advanced Change Detection**: Support for more network scenarios
3. **Monitoring Statistics**: Detailed network change reporting
4. **Configuration Validation**: Automatic network setup verification
5. **Diagnostic Tools**: Network troubleshooting utilities
