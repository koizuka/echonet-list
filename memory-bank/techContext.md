# Technical Context

This file provides details about the technical environment and constraints of the project, complementing the core project definition in [projectbrief.md](./projectbrief.md).

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

- Go 1.21 or later
- Git for version control
- Command-line environment for running and testing the application
- Local network with ECHONET Lite devices for full testing

## Technical Constraints

- Must maintain compatibility with the ECHONET Lite protocol specification
- Needs to work with various ECHONET Lite device types
- Requires UDP network access for device discovery and communication
- Needs file system access for persistent storage of discovered devices and configuration.

## WebSocket Communication

- **Purpose**: Enables remote control and monitoring of ECHONET Lite devices via WebSocket.
- **Protocol Definition**: The custom JSON-based protocol is defined in `protocol/protocol.go`.
- **Detailed Documentation**: For comprehensive client development details, including all message types, payloads, data formats, and error codes, refer to the primary specification document: **[`docs/websocket_client_protocol.md`](../docs/websocket_client_protocol.md)**.
- **Server**: Listens for WebSocket connections, interacts with ECHONET Lite devices (using the `echonet_lite` package), manages state (devices, aliases, groups), and broadcasts updates to clients. Implemented in `server/websocket_server.go`.
- **Client**: Connects to the server, sends commands (e.g., get/set properties, manage aliases/groups), and receives state notifications. Implemented in `client/websocket_client.go`.
- **Startup Options**:
  - `-websocket`: Enable WebSocket server mode.
  - `-ws-client`: Enable WebSocket client mode.
  - `-ws-client-addr`: Specify WebSocket server address for the client to connect to (default: `localhost:8080`).
  - `-ws-both`: Enable both WebSocket server and client mode for dogfooding.
  - `-ws-tls`: Enable TLS for WebSocket server.
  - `-ws-cert-file`: Specify TLS certificate file path.
  - `-ws-key-file`: Specify TLS private key file path.
  - `-http-host`: Specify HTTP server host name (default: `localhost`).
  - `-http-port`: Specify HTTP server port (default: `8080`).
- **TLS Support**: WebSocket server can be configured to use TLS (WSS) for secure connections.
  - When TLS is enabled, WebSocket clients should connect using `wss://` instead of `ws://`.
  - The application automatically updates client connection URLs when TLS is enabled.

## Daemon Mode

- **Purpose**: バックグラウンドでWebSocketサーバーを実行し、コンソールUIを起動しないモード
- **Activation**: `-daemon` フラグまたは設定ファイルの `daemon.enabled = true` で有効化
- **PID File**: `-pidfile <path>` または設定ファイルの `daemon.pid_file` で指定
- **Log Rotation**: デーモンモード時のみ SIGHUP シグナルによるログローテーションが有効
- **WebSocket Server**: デーモンモードでは WebSocket サーバーが必須（自動的に有効化）
- **Client Mode**: デーモンモードでは WebSocket クライアントモードは無効

## Configuration File

- **Format**: TOML (Tom's Obvious, Minimal Language)
- **Default Path**: `config.toml` in the current directory
- **Command Line Override**: `-config` option to specify a different file path
- **Settings**:
  - General settings (debug mode)
  - Log settings (filename)
  - WebSocket server settings (enabled, TLS)
  - WebSocket client settings (enabled, address)
  - HTTP server settings:
    - `http_enabled = true/false`
    - `http_host = "localhost"` (example)
    - `http_port = 8080` (example)
    - `http_webroot = "web/bundle"` (example)
- **Priority**: Command line arguments take precedence over configuration file settings
- **Sample File**: `config.toml.sample` is provided as a template

## Web UI Development (Planned)

将来的に計画されているWeb UIの開発に関する技術的な考慮事項とワークフローは以下の通りです。

### Frontend Technology

- **Framework Consideration**: UIの試行錯誤とメンテナンスを容易にするため、React, Vue, SvelteなどのコンポーネントベースのJavaScriptフレームワークの採用を検討します。これらのフレームワークは、UIパーツの再利用、状態管理、開発時のホットリロード機能を提供し、開発効率を高めることが期待されます。

### Development Workflow

1.  **Source Code Location**: Web UIのフロントエンドコード（HTML, CSS, JavaScript/TypeScript, フレームワーク関連ファイル）は、プロジェクトルート直下の `webui/` ディレクトリ（仮称）で管理します。
2.  **Build Process**: `webui/` ディレクトリ内で、選択したフレームワークのビルドコマンド（例: `npm run build`）を実行し、静的なHTML, CSS, JavaScriptファイルを生成します。
3.  **Asset Deployment**: ビルドされた静的ファイルを、GoサーバーがWebコンテンツを提供するために設定されたディレクトリ（`config.toml` の `http_webroot` で指定。例: `server/webroot/`）にコピーします。このプロセスはMakefileやスクリプトで自動化することを検討します。

### Server-side Asset Reload

- **Development Phase**: 開発中は、Web UIの静的ファイルを更新した後、Goサーバーを再起動して変更を反映させるのが最も簡単な方法です。
- **Future Enhancement**: UI更新の頻度が高くなった場合、サーバーを停止せずにUIアセットを再読み込みする機能の導入を検討します。有力な方法として、Console UIに新しいコマンド（例: `reload-webui`）を追加し、実行時にHTTPサーバーに `http_webroot` の内容を再読み込みさせる方式が考えられます。SIGHUPシグナルや専用のWebSocketメッセージも代替案として考慮できますが、Consoleコマンドが既存インターフェースとの親和性が高い可能性があります。

### Client-side Auto-Reload

- **Mechanism**: サーバーがWeb UIアセットの再読み込みを完了した後、接続中の全WebSocketクライアントに新しい通知メッセージ (`ui_updated` など、別途定義) を送信します。
- **Client Action**: Webクライアント（ブラウザのJavaScript）は、この通知を受信したら自動的にページをリロード (`window.location.reload()`) し、最新のUIアセットを取得します。これにより、ユーザーは手動でリロードすることなく、常に最新のUIを利用できます。
