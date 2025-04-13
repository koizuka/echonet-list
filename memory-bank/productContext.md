# Product Context

This file expands on the core project definition in [projectbrief.md](./projectbrief.md) to provide detailed context about the product's purpose and functionality.

## Project Purpose

This is a Go application for discovering and controlling ECHONET Lite devices on a local network. ECHONET Lite is a communication protocol for smart home devices, primarily used in Japan.

## Problems Solved

- Provides a unified interface for discovering and controlling various ECHONET Lite devices
- Enables automatic discovery of ECHONET Lite devices on the local network
- Allows users to get and set property values on specific devices
- Maintains persistent storage of discovered devices
- Supports various device types (air conditioners, lighting, floor heating, etc.)
- Provides alias functionality for easier reference to devices

## How It Works

1. The application discovers ECHONET Lite devices on the local network using UDP broadcast
2. It maintains a list of discovered devices with their properties
3. Users can interact with devices through a command-line interface
4. Commands include:
   - `discover`: Find all ECHONET Lite devices on the network
   - `devices`/`list`: List all discovered devices with filtering options
   - `get`: Get property values from specific devices
   - `set`: Set property values on specific devices
   - `update`: Update all properties of devices
   - `alias`: Manage device aliases (create, view, delete, list)
   - `debug`: Display or change debug mode
   - `help`: Display help information
   - `quit`: Exit the application
5. The application supports various device types including air conditioners, lighting, floor heating, etc.
6. Users can create aliases for devices to reference them more easily in commands
7. The application includes a notification system that enables loose coupling between components:
   - Device discovery notifications inform when new devices are found
   - Timeout notifications alert when device communication fails
   - Property change notifications (planned) will allow frontend components to receive real-time state changes
8. The notification system is designed to support a future architecture where:
   - ECHONET Lite processing will be handled by a WebSocket server
   - Console UI and Web UI will connect to this server to receive state updates
   - This loose coupling enables multiple frontends to react to state changes independently

## Web UI Features (Planned)

将来的に開発予定の Web UI では、以下の機能を目指します：

- **デバイス一覧のグルーピング:** デバイスを設置場所（リビング、寝室など）でグループ化して表示します。設置場所の情報は、ECHONET Lite の Installation Location プロパティ (EPC 0x81) から取得します。
- **設置場所の管理:** Web UI からデバイスの Installation Location プロパティ (EPC 0x81) を設定・変更できるようにします。
- **状態の可視化:** デバイス一覧で、ON/OFF 状態や設定温度などの主要なプロパティを分かりやすく表示します。
- **グループ操作:** ユーザーが定義したグループ（例：「リビングの照明」）に対して、一括で操作（例：すべて消灯）する機能を提供します。
- **グループ設定:** グループの作成・更新・削除・一覧取得は、既存のWebSocketメッセージ `manage_group` を使用してサーバーと通信し、サーバー側で設定を永続化します。
- **リアルタイム更新:** WebSocketを通じてサーバーから送信される状態変化通知（`device_updated`, `property_changed`, `group_changed` など）を受信し、UI表示をリアルタイムに更新します。
