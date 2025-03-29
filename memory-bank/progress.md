# Progress

This file tracks the implementation progress of the project features defined in [projectbrief.md](./projectbrief.md) and planned in [activeContext.md](./activeContext.md).

## What Works

- The ECHONET Lite application is functional and can discover and control devices
- The alias command functionality is fully implemented
  - Users can create, view, delete, and list aliases for devices in memory
  - Documentation for the alias command is complete in both PrintUsage and README.md
  - Aliases are persisted to disk in aliases.json
- The DeviceAliases.go file has SaveToFile and LoadFromFile methods implemented
- ECHONETLiteHandler.go calls SaveToFile after alias operations to persist aliases to disk
- LoadFromFile is called at startup to load saved aliases
- FilterCriteria has been improved by removing the EPCs field
  - EPCs filtering is now handled in CommandProcessor.go using Command.EPCs
  - "-all" and "-props" options now clear the EPCs filter
  - PrintUsage documentation has been updated to clarify that the last specified option takes precedence
- Message retransmission functionality has been implemented
  - Session.go now has the ability to retry sending messages up to 3 times when a timeout occurs
  - Added GetPropertiesWithContext and SetPropertiesWithContext methods that handle retransmission
  - ECHONETLiteHandler now uses these new methods for more reliable communication
  - Improved error handling for partial success cases
- Help command has been enhanced
  - When given a command name as an argument, it shows detailed information for that command only
  - Without arguments, it shows a summary of all commands
  - Command information is now stored in a table-driven approach using CommandDefinition structs
  - This makes the help system more maintainable and user-friendly

## What's Left to Build

No immediate tasks remaining.

### 将来の計画 (Future Plans)

- ✅ **デバイス通知機能**: ECHONETLiteHandlerから呼出元に対して、デバイスの追加通知とデバイスのリトライタイムアウト通知を送るチャンネルを作る。mainではそれを受けて表示する。
- **プロパティ変化通知機能**: デバイスのプロパティ値が変化した際に通知を送る機能を実装する。これにより、フロントエンドが状態変化をリアルタイムに受け取れるようになる。この機能は、システムを疎結合にし、将来的なWebSocketサーバーとUI分離のアーキテクチャを実現するための重要な要素となる。
  - 実装予定: プロパティ監視機能とイベント通知の仕組みの設計と実装
  - 状態: 計画中（デバイス通知機能の次に実装予定）
- **アーキテクチャ分割**: ECHONET Liteに関する処理は web(WebSocket) サーバーにして、コンソールUIアプリはそれにアクセスするように分割する
  - 実装予定: 新しいパッケージ構造の設計と実装
  - 状態: 依存関係の整理中
- **Web UI開発**: 上記分割が済んだら、web UIを作成する
  - 実装予定: フロントエンドの設計と実装
  - 状態: 未着手（アーキテクチャ分割後に開始）

## Progress Status

- **Alias Command Documentation in PrintUsage**: ✅ COMPLETED
- **Alias Command Documentation in README.md**: ✅ COMPLETED
- **Alias Command Persistence Implementation**: ✅ COMPLETED
  - SaveToFile is called after alias operations in ECHONETLiteHandler.go
  - LoadFromFile is called at startup in NewECHONETLiteHandler
- **FilterCriteria Improvement**: ✅ COMPLETED
  - Removed EPCs field from FilterCriteria
  - Modified Filter method to not use EPCs field
  - Updated CommandProcessor.go to filter using Command.EPCs
  - Updated Command.go to clear EPCs when "-all" or "-props" is specified
  - Updated PrintUsage documentation
  - Updated Filter_test.go to remove EPCs-related test cases
- **Message Retransmission**: ✅ COMPLETED
  - Added MaxRetries and RetryInterval fields to Session struct
  - Implemented unregisterCallback function for proper cleanup
  - Added CreateSetPropertyMessage function for consistency
  - Implemented sendRequestWithContext for common retry logic
  - Added GetPropertiesWithContext and SetPropertiesWithContext methods
  - Modified ECHONETLiteHandler's GetProperties and SetProperties to use the new methods
  - Updated ECHONETLiteHandler's UpdateProperties to use GetPropertiesWithContext with go routines for parallel processing
  - Improved error handling for partial success cases
- **Device Notification**: ✅ COMPLETED
  - ✅ Added NotificationType and DeviceNotification types
  - ✅ Added NotificationCh to ECHONETLiteHandler struct
  - ✅ Added ErrMaxRetriesReached error type for proper error handling
  - ✅ Modified ECHONETLiteHandler to send notifications for new devices and timeouts
  - ✅ Added notification listener in main.go to display notifications to the user
  - ✅ Improved device addition notification by moving it to Devices.ensureDeviceExists
  - ✅ Added DeviceEventType and DeviceEvent types in Devices.go
  - ✅ Added EventCh to Devices struct for device event notifications
  - ✅ Implemented event forwarding from Devices to ECHONETLiteHandler
  - ✅ Tested device addition notification in real environment
  - ✅ Added unit tests for device notification in Devices_test.go
  - ✅ Testing timeout notification in real environment completed
- **Help Command Enhancement**: ✅ COMPLETED
  - ✅ Created CommandDefinition struct to hold command information
  - ✅ Implemented CommandTable to store all command definitions
  - ✅ Added parseHelpCommand function to handle help command with arguments
  - ✅ Modified PrintUsage to show detailed information for a specific command
  - ✅ Added PrintCommandSummary and PrintCommandDetail functions
  - ✅ Converted ParseCommand to use table-driven approach
  - ✅ Replaced custom contains function with slices.Contains from standard library
- **Console UI Separation**: ✅ COMPLETED
  - ✅ Moved console UI related files to `console/` directory
  - ✅ Organized code into client, server, and protocol packages
  - ✅ Updated imports and dependencies
  - ✅ Tested functionality after reorganization
- **Architecture Split (WebSocket Implementation)**: 🔄 IN PROGRESS
  - ✅ WebSocket server/client implementation has been started. The implemented code (`protocol/protocol.go`, `server/websocket_server.go`, `client/websocket_client.go`) provides a basic WebSocket-based client-server architecture.
  - ✅ Added command-line flags for WebSocket mode: `-websocket`, `-ws-addr`, `-ws-client`, `-ws-client-addr`, `-ws-both`
  - ✅ Implemented WebSocket client that implements the ECHONETListClient interface
  - ✅ Implemented WebSocket server that handles client requests and notifications
  - ✅ Added helper functions in `echonet_lite` package for parsing hex strings to `EOJClassCode`, `EOJInstanceCode`, `EPCType`
  - ✅ Implemented client URL validation using `net/url.Parse`
  - ✅ Added synchronization for `-ws-both` mode
  - ✅ Implemented Base64 encoding/decoding for property values in WebSocket protocol
    - ✅ Modified `DeviceToProtocol` and `DeviceFromProtocol` functions to use Base64 encoding
    - ✅ Updated WebSocket server to use `DeviceToProtocol` function
    - ✅ Updated WebSocket client to properly decode Base64-encoded property values
    - ✅ Removed debug output code for cleaner implementation
  - **Issues Fixed**:
    - ✅ Fixed the `quit` command issue that was causing the application to freeze
    - ✅ Improved error handling and logging in the WebSocket client
    - ✅ Added proper cleanup of WebSocket connections when the application exits
    - ✅ Fixed binary data handling in JSON by implementing Base64 encoding/decoding
    - ✅ Fixed the `list` command in WebSocket client mode
  - **Issues Remaining**:
    - ⚠️ Some WebSocket client commands still need implementation or fixes
    - ⚠️ Need to add more tests for the WebSocket client and server
- **Web UI Development**: 🔄 PLANNED
