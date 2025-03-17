# Progress

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

## What's Left to Build
- All planned features for the current development cycle have been implemented

### 将来の計画 (Future Plans)
- **メッセージ再送機能**: Session でメッセージを送信したあと、返信を必要としているものについて、返信タイムアウトになったときには同一メッセージを再送する仕組み
  - 実装予定: Session.go の修正が必要
  - 状態: 設計検討中
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
- **Message Retransmission**: 🔄 PLANNED
- **Architecture Split**: 🔄 PLANNED
- **Web UI Development**: 🔄 PLANNED
