# Progress

本ファイルでは、[projectbrief.md](./projectbrief.md) および [activeContext.md](./activeContext.md) に定義された機能の実装状況を追跡します。

## What Works

- ECHONET Lite デバイスの検出・制御
- エイリアス機能（作成、表示、削除、一覧表示、永続化）
- メッセージ再送信機能（タイムアウト時に最大3回再試行）
- ヘルプコマンドの拡張（コマンド名指定で詳細表示）
- WebSocket サーバー／クライアント実装
  - TLS 対応 (WSS)
  - 設定ファイル (TOML) サポート
  - 通知機能
    - デバイス追加通知
    - タイムアウト通知
    - プロパティ変化通知 (`property_changed`)
- デバイスグループ管理機能
  - グループの作成・表示・削除・一覧表示
  - `@` プレフィックスによるグループ指定
  - グループ内一括操作
- `devices` コマンドの EPC 値でのグループ化表示 (`-group-by <epc>`)
- WebSocket サーバーの定期的プロパティ自動更新機能
  - クライアント接続時、設定ファイルで指定可能な間隔（デフォルト1分）
- デバイスのオフライン状態管理（基本）
  - タイムアウト時の自動マーク
  - `UpdateProperties` でのスキップ
  - WebSocket クライアントへのオフライン通知
- プロパティ転送フォーマット刷新
  - `"EPC": { "EDT": "BASE64", "string": "xxx" }` 形式導入
  - `PropertyData` 構造体および双方向変換ロジック (`DeviceToProtocol` / `DeviceFromProtocol`) の実装
  - 対応テスト・サンプル JSON の更新
  - サーバー／クライアントハンドラの更新
  - ドキュメント (`docs/websocket_client_protocol.md`) の全面置換

## What's Left to Build

- **ドキュメント更新**  
  - `property_changed`, `set_properties` 等のメッセージ例を新フォーマット化  
  - クライアント実装ガイド／コード例への反映  
- **テスト拡充**  
  - 文字列のみ／Base64 のみ指定時の挙動  
  - WebSocket クライアント各コマンドの統合テスト  
- **WebSocket クライアント機能改善**  
  - `discover`, `get`, `set`, `update`, `alias`, `group` コマンドを全て WebSocket 経由で動作させる  
  - オフライン状態デバイスの UI 表示・エラーハンドリング  
- **UI／サンプルクライアントへのガイド反映**  
  - プロパティ新フォーマット対応の例示  
  - グループ管理・オフライン表示サンプル  
- **Web UI 開発準備**  
  - `webui/` ディレクトリ構成  
  - ビルド・配信フロー設計  
  - 初期ページレイアウト・コンポーネント検討  
- **ドキュメント・README の進捗反映**  
  - Memory Bank 更新  
  - README.md の最新コマンド例と設定例を追加
