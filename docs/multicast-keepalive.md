# Multicast Keep-Alive Implementation

## Overview

This document describes the multicast keep-alive functionality implemented in ECHONET List to maintain reliable UDP multicast group membership and detect network connectivity issues.

## Features

### 1. Multicast Group Membership Maintenance

- **Automatic Group Join**: Uses `net.ListenMulticastUDP()` for automatic multicast group joining
- **IGMP Compliance**: Relies on OS kernel's IGMP implementation for group membership
- **Group Refresh**: Periodically validates and refreshes multicast group membership

### 2. Network Monitoring

- **Interface Monitoring**: Detects network interface changes (up/down, IP changes)
- **Automatic Recovery**: Automatically refreshes connections when network changes are detected
- **IP Address Updates**: Updates local IP address cache when network interfaces change
- **IGMP-based Keep-Alive**: Uses standard IGMP protocol instead of custom heartbeat packets

### 3. Configurable Keep-Alive Settings

- **Group Refresh Interval**: Configurable interval for group membership validation (default: 5m)
- **Network Monitor**: Enable/disable network interface monitoring (default: enabled)

## Configuration

### TOML Configuration File

```toml
[multicast]
# Enable multicast keep-alive functionality
keep_alive_enabled = true

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
group_refresh_interval = "10m"
network_monitor_enabled = true
```

#### Corporate Networks (Moderate)

```toml
group_refresh_interval = "5m"
network_monitor_enabled = true
```

#### Unstable Networks (Mobile/WiFi)

```toml
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
- ~~`sendHeartbeat()`~~: Removed - now relies on OS kernel's IGMP implementation
- `monitorNetworkChanges()`: Monitors network interface changes

#### Key Methods

- `CreateUDPConnectionWithKeepAlive()`: Creates UDP connection with keep-alive
- ~~`TriggerHeartbeat()`~~: Removed - heartbeat functionality replaced by IGMP
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

### IGMP Compliance (Updated in PR #33)

- **RFC 2236**: Full compliance with IGMPv2 specification
- **Query/Report**: OS kernel handles IGMP Query messages from routers and sends Reports
- **Timeout**: Default 260-second timeout managed by router (configurable)
- **No Custom Packets**: Removed application-level heartbeat packets that violated IGMP spec

### IGMP-based Keep-Alive

- **Protocol**: Uses standard IGMP (Internet Group Management Protocol)
- **OS Integration**: Relies on OS kernel's IGMP implementation
- **No Custom Packets**: No application-level heartbeat packets
- **Standards Compliant**: Follows RFC 2236 (IGMPv2) specifications

### Network Traffic Impact

- **Zero Application Overhead**: No custom keep-alive packets
- **IGMP Only**: Standard IGMP Query/Report messages handled by OS
- **No Protocol Interference**: Doesn't affect ECHONET Lite communication

## Monitoring and Diagnostics

### Log Messages

#### Info Level

```
INFO マルチキャストキープアライブが開始されました groupRefreshInterval=5m networkMonitorEnabled=true
INFO ネットワークインターフェースの変更を検出しました
INFO マルチキャストキープアライブが停止されました
```

#### Debug Level

```
DEBUG マルチキャストグループのメンバーシップを確認しました multicastIP=224.0.23.0
DEBUG ローカルIPアドレスを更新しました count=2
DEBUG キープアライブループを終了します
```

#### Warning Level

```
WARN ネットワークインターフェース情報の取得に失敗 err="operation not permitted"
WARN ローカルIPアドレスの再取得に失敗 err="no route to host"
WARN マルチキャストグループの再参加に失敗しました err="network is unreachable"
```

### Status Monitoring

Use the `GetKeepAliveStatus()` method to check current status:

```go
enabled, lastGroupRefresh := connection.GetKeepAliveStatus()
if enabled {
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

#### 3. IGMP Timeout Issues

- **Symptom**: Multicast group membership timeout
- **Cause**: Network router IGMP configuration or firewall blocking IGMP
- **Solution**: Check router IGMP settings and firewall rules for IGMP traffic

#### 4. Multicast Group Leave Issues

- **Symptom**: Application doesn't properly leave multicast group
- **Cause**: Improper shutdown or context cancellation
- **Solution**: Ensure proper Close() call and context management

#### 5. Synchronization Issues (Fixed in PR #33)

- **Symptom**: Race conditions or goroutine leaks
- **Cause**: Improper synchronization in keep-alive shutdown
- **Solution**: Updated to use `done` channel for proper goroutine termination

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

## Breaking Changes (PR #33)

⚠️ **Configuration Changes Required**

If you have an existing configuration file with `heartbeat_interval`, you must remove this setting:

```diff
[multicast]
keep_alive_enabled = true
-heartbeat_interval = "30s"  # REMOVE THIS LINE
group_refresh_interval = "5m"
network_monitor_enabled = true
```

The heartbeat functionality has been completely removed in favor of IGMP-compliant implementation.

## Performance Impact

### Memory Usage

- **Additional Memory**: <1KB per connection for keep-alive state
- **Goroutines**: 1 additional goroutine per connection
- **Timers**: 1-2 timers per connection (group refresh, network monitor)

### CPU Usage

- **Background Processing**: Minimal CPU usage for timers
- **Network Monitoring**: Low-frequency interface checks (every 10 seconds)
- **No Heartbeat Overhead**: No CPU usage for packet creation/transmission

### Network Impact

- **Bandwidth**: Zero additional application-level traffic
- **Packet Rate**: Only standard IGMP messages (handled by OS)
- **No Effect**: On existing ECHONET Lite protocol communications

## Future Enhancements

### Potential Improvements

1. **Connection Quality Metrics**: Track multicast group membership status
2. **Advanced Network Detection**: Support for more network change scenarios
3. **Keep-Alive Statistics**: Detailed monitoring and reporting
4. **IGMP Version Detection**: Automatic detection of IGMPv2/v3 support
5. **Router Configuration Helper**: Diagnostic tools for IGMP router settings
