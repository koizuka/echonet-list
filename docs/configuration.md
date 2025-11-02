# Configuration Guide

This guide covers all configuration options for the ECHONET Lite Device Discovery and Control Tool, including command line options and the configuration file format.

## Configuration File

The application supports a TOML configuration file for persistent settings. By default, it looks for `config.toml` in the current directory.

### Getting Started

1. Copy the sample configuration file:

```bash
cp config.toml.sample config.toml
```

2. Edit `config.toml` to customize your settings
3. The configuration file is excluded from version control by `.gitignore`

### Configuration File Format

```toml
# echonet-list 設定ファイル

# 全般設定
debug = false

# ログ設定
[log]
filename = "echonet-list.log"

# WebSocketサーバー設定
[websocket]
enabled = true
# 定期的なプロパティ更新間隔（例: "1m", "30s", "0" で無効）
periodic_update_interval = "1m"

# TLS設定（HTTPサーバーとWebSocketサーバーで共通）
[tls]
enabled = false
cert_file = "certs/localhost+2.pem"
key_file = "certs/localhost+2-key.pem"

# WebSocketクライアント設定
[websocket_client]
enabled = false
addr = "ws://localhost:8080/ws"  # TLS有効時はwss://を使用

# HTTP Server設定（WebSocketと統合）
[http_server]
enabled = false
host = "localhost"
port = 8080
web_root = "web/bundle"

# デバイス履歴設定
[history]
per_device_settable_limit = 200     # 操作可能プロパティの履歴保持件数
per_device_non_settable_limit = 100 # 通知のみプロパティの履歴保持件数

# ネットワーク監視設定
[network]
monitor_enabled = true  # ネットワークインターフェース変更の監視

# デーモンモード設定
[daemon]
enabled = false
pid_file = ""  # 省略時はプラットフォーム別のデフォルトパスを使用
```

### Configuration Sections

#### General Settings

- `debug`: Enable debug mode for detailed communication logs

#### Log Settings (`[log]`)

- `filename`: Log file path (default: "echonet-list.log")

#### WebSocket Server (`[websocket]`)

- `enabled`: Enable WebSocket server mode
- `periodic_update_interval`: Interval for periodic property updates (e.g., "1m", "30s", "0" to disable)

#### TLS Settings (`[tls]`)

- `enabled`: Enable TLS for both HTTP and WebSocket servers
- `cert_file`: Path to TLS certificate file
- `key_file`: Path to TLS private key file

#### WebSocket Client (`[websocket_client]`)

- `enabled`: Enable WebSocket client mode
- `addr`: WebSocket server address to connect to

#### HTTP Server (`[http_server]`)

- `enabled`: Enable integrated HTTP server
- `host`: Server hostname (default: "localhost")
- `port`: Server port (default: 8080)
- `web_root`: Web root directory for static files (default: "web/bundle")

#### Device History (`[history]`)

- `per_device_settable_limit`: Maximum number of settable property history entries per device (default: 200)
  - Controls history for user-initiated operations (on/off, mode changes, etc.)
- `per_device_non_settable_limit`: Maximum number of non-settable property history entries per device (default: 100)
  - Controls history for sensor notifications (temperature, humidity, etc.)

These separate limits ensure that important operation history is retained even when frequent sensor notifications occur.

#### Network Monitoring (`[network]`)

- `monitor_enabled`: Enable network interface monitoring for reliable multicast communication (default: true)

#### Daemon Mode (`[daemon]`)

- `enabled`: Enable daemon mode
- `pid_file`: PID file path (uses platform defaults if empty)

## Command Line Options

Command line options take precedence over configuration file settings.

### Basic Options

- `-config <path>`: Specify configuration file path (default: `config.toml`)
- `-debug`: Enable debug mode for detailed communication logs
- `-log <filename>`: Specify log file name

### Server Mode Options

#### WebSocket Server

- `-websocket`: Enable WebSocket server mode
- `-ws-both`: Enable both WebSocket server and client modes (for testing)

#### WebSocket Client

- `-ws-client`: Enable WebSocket client mode
- `-ws-client-addr <address>`: WebSocket server address (default: `ws://localhost:8080/ws`)

#### TLS Options

- `-ws-tls`: Enable TLS for the integrated server
- `-ws-cert-file <path>`: TLS certificate file path
- `-ws-key-file <path>`: TLS private key file path

#### HTTP Server

- `-http-enabled`: Enable HTTP server (integrated with WebSocket)
- `-http-host <hostname>`: Server hostname (default: `localhost`)
- `-http-port <port>`: Server port (default: `8080`)
- `-http-webroot <path>`: Web root directory (default: `web/bundle`)

### Daemon Mode

- `-daemon`: Enable daemon mode (requires WebSocket server)
- `-pidfile <path>`: PID file path (uses platform defaults if not specified)

### Platform Default Paths

#### Linux

- PID file: `/var/run/echonet-list.pid`
- Log file: `/var/log/echonet-list.log`

#### macOS

- PID file: `/usr/local/var/run/echonet-list.pid`
- Log file: `/usr/local/var/log/echonet-list.log`

## Usage Examples

### Basic Console Mode

```bash
./echonet-list
```

### Debug Mode

```bash
./echonet-list -debug
```

### Web UI with Integrated Server

```bash
./echonet-list -websocket -http-enabled
```

### Web UI with TLS

```bash
./echonet-list -websocket -http-enabled -ws-tls \
  -ws-cert-file=certs/localhost+2.pem \
  -ws-key-file=certs/localhost+2-key.pem
```

### Daemon Mode with Web UI

```bash
./echonet-list -daemon -websocket -http-enabled
```

### Custom Configuration File

```bash
./echonet-list -config /etc/echonet-list/config.toml
```

### WebSocket Client Mode

```bash
./echonet-list -ws-client -ws-client-addr ws://192.168.1.100:8080/ws
```

## Configuration Priority

Settings are applied in the following priority order (highest to lowest):

1. Command line options
2. Configuration file settings
3. Built-in defaults

For example, if both `-debug` flag and `debug = false` in config file are present, debug mode will be enabled (command line takes precedence).
