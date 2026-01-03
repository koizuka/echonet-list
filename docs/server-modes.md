# Server Modes Guide

The ECHONET Lite controller supports multiple server modes to accommodate different use cases. This guide explains each mode and when to use them.

## Overview

The application can run in several modes:

1. **Console Mode** (default) - Interactive command-line interface
2. **WebSocket Server Mode** - API server for web clients
3. **Integrated HTTP/WebSocket Server** - Combined web UI and API server
4. **WebSocket Client Mode** - Connect to another instance
5. **Daemon Mode** - Background service

## Console Mode

The default mode when running without any server options. Provides an interactive terminal UI for device discovery and control.

```bash
./echonet-list
```

Features:

- Interactive device list with real-time updates
- Property viewing and editing
- Device grouping and management
- Direct terminal-based control

See [Console UI Usage Guide](console_ui_usage.md) for detailed usage instructions.

## WebSocket Server Mode

Enables the WebSocket API server, allowing web browsers and other clients to connect and control ECHONET Lite devices.

```bash
./echonet-list -websocket
```

Features:

- WebSocket endpoint at `/ws`
- Real-time bi-directional communication
- JSON-based protocol
- Multiple client support

For protocol details, see [WebSocket Client Protocol](websocket_client_protocol.md).

## Integrated HTTP/WebSocket Server

Combines the WebSocket API server with an HTTP static file server, providing both the API and web UI from a single port.

```bash
./echonet-list -websocket -http-enabled -ws-tls \
  -ws-cert-file=certs/localhost+2.pem \
  -ws-key-file=certs/localhost+2-key.pem
```

Features:

- Single port for both WebSocket and HTTP
- WebSocket API at `/ws`
- Static files served from web root (default: `web/bundle`)
- Eliminates CORS issues
- Simplified deployment

TLS is required for browsers (HTTPS/WSS). Non-TLS is only suitable for
non-browser clients on a trusted network.

URLs:

- Web UI: `https://localhost:8080/`
- WebSocket API: `wss://localhost:8080/ws`

## WebSocket Client Mode

Connects to another instance running in WebSocket server mode. Useful for distributed setups or testing.

```bash
./echonet-list -ws-client -ws-client-addr wss://192.168.1.100:8080/ws
```

Features:

- Remote control of ECHONET Lite devices
- Console UI connected to remote server
- Useful for multi-location setups

## Daemon Mode

Runs the application as a background service without console UI. Requires WebSocket server to be enabled for control.

```bash
./echonet-list -daemon -websocket -http-enabled -ws-tls \
  -ws-cert-file=certs/localhost+2.pem \
  -ws-key-file=certs/localhost+2-key.pem
```

Features:

- Background operation
- PID file management
- Log rotation support (SIGHUP)
- systemd integration ready
- Platform-specific default paths

For detailed daemon setup, see [Daemon Setup Guide](daemon-setup.md).

## Mode Combinations

### Development Setup

For web UI development with hot reloading:

1. Run the server (WebSocket only):

```bash
./echonet-list -websocket -ws-tls \
  -ws-cert-file=certs/localhost+2.pem \
  -ws-key-file=certs/localhost+2-key.pem
```

2. Run Vite dev server:

```bash
cd web && npm run dev
```

### Production Setup

Single integrated server with TLS:

```bash
./echonet-list -daemon -websocket -http-enabled -ws-tls \
  -ws-cert-file=/etc/echonet-list/cert.pem \
  -ws-key-file=/etc/echonet-list/key.pem
```

### Testing Setup

Run both server and client for testing:

```bash
./echonet-list -ws-both
```

## Choosing the Right Mode

### Use Console Mode when

- Direct device control is needed
- Running on a local machine
- No web access required
- Quick testing or debugging

### Use WebSocket Server when

- Building custom clients
- Web UI access is needed
- Multiple users need access
- Remote control is required

### Use Integrated Server when

- Deploying the complete solution
- Serving the official web UI
- Single-port deployment is preferred
- Production environment

### Use Daemon Mode when

- Running as a system service
- Automatic startup is needed
- Long-term operation
- Server/headless environment

## Security Considerations

1. **TLS is strongly recommended** for any network-accessible deployment
2. The application does not include authentication - use network security
3. Bind to localhost only unless external access is required
4. Use firewall rules to restrict access as needed

## Related Documentation

- [Configuration Guide](configuration.md) - Detailed configuration options
- [WebSocket Client Protocol](websocket_client_protocol.md) - API protocol details
- [Web UI Implementation Guide](web_ui_implementation_guide.md) - Web UI details
- [Daemon Setup Guide](daemon-setup.md) - systemd service configuration
