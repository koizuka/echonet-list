# Multicast Keep-Alive Implementation

## Overview

This document describes the multicast keep-alive functionality implemented in ECHONET List to maintain reliable UDP multicast group membership and detect network connectivity issues.

## Features

### 1. Multicast Group Membership Maintenance

- **Automatic Group Join**: Uses `net.ListenMulticastUDP()` for automatic multicast group joining
- **Periodic Heartbeat**: Sends minimal UDP packets to maintain group membership
- **Group Refresh**: Periodically validates and refreshes multicast group membership

### 2. Network Monitoring

- **Interface Monitoring**: Detects network interface changes (up/down, IP changes)
- **Automatic Recovery**: Automatically refreshes connections when network changes are detected
- **IP Address Updates**: Updates local IP address cache when network interfaces change

### 3. Configurable Keep-Alive Settings

- **Heartbeat Interval**: Configurable interval for sending heartbeat packets (default: 30s)
- **Group Refresh Interval**: Configurable interval for group membership validation (default: 5m)
- **Network Monitor**: Enable/disable network interface monitoring (default: enabled)

## Configuration

### TOML Configuration File

```toml
[multicast]
# Enable multicast keep-alive functionality
keep_alive_enabled = true

# Interval for sending heartbeat packets to maintain multicast group membership
# Recommended: 30s-60s for home networks, 10s-30s for unstable networks
heartbeat_interval = "30s"

# Interval for refreshing multicast group membership
# Recommended: 5m-10m for most environments
group_refresh_interval = "5m"

# Enable network interface monitoring for automatic reconnection
# Monitors network changes and automatically refreshes connections
network_monitor_enabled = true
```

### Recommended Settings

#### Home Networks (Stable)

```toml
heartbeat_interval = "60s"
group_refresh_interval = "10m"
network_monitor_enabled = true
```

#### Corporate Networks (Moderate)

```toml
heartbeat_interval = "30s"
group_refresh_interval = "5m"
network_monitor_enabled = true
```

#### Unstable Networks (Mobile/WiFi)

```toml
heartbeat_interval = "15s"
group_refresh_interval = "2m"
network_monitor_enabled = true
```

## Implementation Details

### Network Layer (UDPConnection)

**File**: `echonet_lite/network/UDPConnection.go`

#### Key Components

- `MulticastKeepAlive` struct: Manages keep-alive state and timers
- `KeepAliveConfig` struct: Configuration parameters
- `keepAliveLoop()`: Main keep-alive event loop
- `sendHeartbeat()`: Sends minimal UDP packets for group membership
- `monitorNetworkChanges()`: Monitors network interface changes

#### Key Methods

- `CreateUDPConnectionWithKeepAlive()`: Creates UDP connection with keep-alive
- `TriggerHeartbeat()`: Manually triggers heartbeat
- `TriggerGroupRefresh()`: Manually triggers group refresh
- `GetKeepAliveStatus()`: Returns current keep-alive status

### Session Layer Integration

**File**: `echonet_lite/handler/Session.go`

- `CreateSessionWithKeepAlive()`: Creates session with keep-alive configuration
- Maintains backward compatibility with existing `CreateSession()`

### Handler Integration

**File**: `echonet_lite/handler/ECHONETLiteHandler.go`

- Extended `ECHONETLieHandlerOptions` to include `KeepAliveConfig`
- Passes configuration through to session layer

### Application Integration

**Files**: `server/server.go`, `main.go`

- `NewServerWithConfig()`: Creates server with multicast configuration
- Parses configuration from TOML file
- Applies keep-alive settings to handler

## Protocol Considerations

### Heartbeat Packets

- **Size**: Minimal 1-byte packets (`0x00`)
- **Purpose**: Maintain OS-level multicast group membership
- **Not ECHONET Lite**: These are low-level network keep-alive packets
- **Filtering**: Properly filtered by ECHONET Lite message parser (too small to be valid)

### Network Traffic Impact

- **Minimal Overhead**: 1 byte every 30 seconds (default)
- **Background Traffic**: Runs independently of application traffic
- **No Protocol Interference**: Doesn't affect ECHONET Lite communication

## Monitoring and Diagnostics

### Log Messages

#### Info Level

```
INFO マルチキャストキープアライブが開始されました heartbeatInterval=30s groupRefreshInterval=5m networkMonitorEnabled=true
INFO ネットワークインターフェースの変更を検出しました
INFO マルチキャストキープアライブが停止されました
```

#### Debug Level

```
DEBUG ネットワークキープアライブを送信 multicastIP=224.0.23.0
DEBUG マルチキャストグループのメンバーシップを確認しました multicastIP=224.0.23.0
DEBUG ローカルIPアドレスを更新しました count=2
DEBUG キープアライブループを終了します
```

#### Warning Level

```
WARN ネットワークキープアライブ送信エラー err="network is unreachable"
WARN ネットワークインターフェース情報の取得に失敗 err="operation not permitted"
WARN ローカルIPアドレスの再取得に失敗 err="no route to host"
```

### Status Monitoring

Use the `GetKeepAliveStatus()` method to check current status:

```go
enabled, lastHeartbeat, lastGroupRefresh := connection.GetKeepAliveStatus()
if enabled {
    fmt.Printf("Last heartbeat: %v\n", lastHeartbeat)
    fmt.Printf("Last group refresh: %v\n", lastGroupRefresh)
}
```

## Troubleshooting

### Common Issues

#### 1. Keep-Alive Not Starting

- **Symptom**: No keep-alive log messages
- **Cause**: `keep_alive_enabled = false` or missing multicast IP
- **Solution**: Enable in configuration and verify multicast setup

#### 2. Network Change Detection Not Working

- **Symptom**: Network changes not detected
- **Cause**: `network_monitor_enabled = false` or insufficient permissions
- **Solution**: Enable monitoring and check network interface permissions

#### 3. High Network Traffic

- **Symptom**: Unexpected network usage
- **Cause**: Too frequent heartbeat interval
- **Solution**: Increase `heartbeat_interval` (e.g., to 60s or 120s)

#### 4. Multicast Group Leave Issues

- **Symptom**: Application doesn't properly leave multicast group
- **Cause**: Improper shutdown or context cancellation
- **Solution**: Ensure proper Close() call and context management

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

Run keep-alive specific tests:

```bash
go test ./echonet_lite/network -v -run TestKeepAlive
```

### Integration Testing

Test with configuration file:

```bash
./echonet-list -config config.multicast.toml -debug
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
- Default behavior: keep-alive disabled
- New functionality accessed through `*WithKeepAlive` functions
- Configuration is optional and backward compatible

## Performance Impact

### Memory Usage

- **Additional Memory**: ~1KB per connection for keep-alive state
- **Goroutines**: 1 additional goroutine per connection
- **Timers**: 2-3 timers per connection (heartbeat, group refresh, network monitor)

### CPU Usage

- **Background Processing**: Minimal CPU usage for timers
- **Network Monitoring**: Low-frequency interface checks (every 10 seconds)
- **Heartbeat**: Minimal packet creation and transmission

### Network Impact

- **Bandwidth**: ~3 bytes/minute (1 byte per 30s heartbeat)
- **Packet Rate**: 2 packets/minute additional multicast traffic
- **No Effect**: On existing ECHONET Lite protocol communications

## Future Enhancements

### Potential Improvements

1. **Adaptive Intervals**: Adjust heartbeat frequency based on network stability
2. **Connection Quality Metrics**: Track packet loss and latency
3. **Advanced Network Detection**: Support for more network change scenarios
4. **Keep-Alive Statistics**: Detailed monitoring and reporting
5. **Custom Heartbeat Messages**: Application-specific keep-alive content
