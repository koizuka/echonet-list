# Device History Refactoring Progress

## 目的
デバイス履歴管理をserver層からhandler層に移動し、ドメインロジックを適切な層に配置する

## 完了した作業 ✅

### 1. handler層に履歴管理を実装
- ✅ `echonet_lite/handler/DeviceHistory.go` を作成
  - `DeviceHistoryStore` インターフェース
  - `memoryDeviceHistoryStore` 実装
  - `PropertyValue` 型（循環依存回避）
  - `PropertyValueFromEDT` 変換関数
  - settable/non-settable プロパティの別管理
  - ファイル永続化（SaveToFile/LoadFromFile）

- ✅ `echonet_lite/handler/DeviceHistory_test.go` を作成
  - 18個のテストすべて成功
  - 既存のserver層テストを移動・適応

### 2. DataManagementHandlerに統合
- ✅ `DeviceHistory` フィールドを追加
- ✅ `NewDataManagementHandler` に引数追加
- ✅ `ECHONETLiteHandler.go` で履歴ストア初期化

### 3. protocol層に型変換関数を追加
- ✅ `protocol/protocol.go` に変換関数追加:
  - `PropertyData.ToHandlerPropertyValue()`
  - `PropertyDataFromHandlerValue()`

### 4. WebSocketServerの部分的更新
- ✅ `historyStore` と `historyFilePath` フィールドを削除
- ✅ `NewWebSocketServer` から履歴初期化処理を削除
- ✅ `GetHistoryStore()` を handler 経由に変更
- ✅ `recordHistory()` を handler 経由に変更
- ✅ `recordPropertyChange()` を handler 経由に変更
- ✅ `clearHistoryForDevice()` を handler 経由に変更
- ✅ `Shutdown()` での保存処理を一旦削除（TODOコメント追加）

## 完了した作業（続き） ✅

### 5. websocket_server_handlers_history.go の更新 ✅
- ✅ `ws.GetHistoryStore()` を使用するように変更
- ✅ `handler.HistoryQuery` を使用
- ✅ `handler.HistoryOrigin` を使用
- ✅ `protocol.PropertyDataFromHandlerValue()` で値変換

### 6. handler層で履歴ファイルの読み込み・保存を実装 ✅
- ✅ `ECHONETLieHandlerOptions` に `HistoryOptions` フィールド追加
- ✅ `ECHONETLiteHandler` に `historyFilePath` フィールド追加
- ✅ `NewECHONETLiteHandler` で履歴ファイルの読み込み実装
- ✅ `Close` メソッドで履歴ファイルの保存実装
- ✅ `server.go` で config から履歴オプションを設定

### 7. config設定の調整 ✅
- ✅ `config/config.go` は既に実装済み
  - `History.PerDeviceSettableLimit`
  - `History.PerDeviceNonSettableLimit`
  - `DataFiles.HistoryFile`

### 8. 古いファイルの削除 ✅
- ✅ `server/device_history_store.go` 削除
- ✅ `server/device_history_store_test.go` 削除
- ✅ `server/websocket_server_handlers_history_test.go` 削除
- ✅ `server/websocket_server_history_test.go` 削除

### 9. テストと動作確認 ✅
- ✅ `go test ./...` で全テスト成功
- ✅ `go build` で警告なしビルド成功
- ⏳ Web UIでデバイス履歴表示が正常動作（手動テスト必要）
- ⏳ 履歴ファイルの保存・読み込みが正常動作（手動テスト必要）

## 技術的な決定事項

### 循環依存の解決
- `handler.PropertyValue` を定義（`protocol.PropertyData` の代わり）
- `protocol` パッケージが `handler` をインポート（既存）
- 変換関数を `protocol` パッケージに配置

### HistoryOrigin の重複
- `server.HistoryOrigin` と `handler.HistoryOrigin` が両方存在
- server層で変換が必要
- 最終的には `server.HistoryOrigin` を削除予定

### 履歴ファイルパスの管理
- 以前: `WebSocketServer` がパスを管理
- 今後: `ECHONETLiteHandler` がパスを管理
- config から読み込んで handler に渡す

## 次のステップ

1. ✅ **websocket_server_handlers_history.go を更新** - 完了
2. ✅ **handler層に履歴永続化を実装** - 完了
3. ✅ **古いファイルを削除** - 完了
4. ✅ **テストと動作確認** - ビルドとテスト成功
5. **動作確認と コミット**
   - 手動でWeb UI動作確認を推奨
   - 履歴ファイルの保存・読み込みを確認
   - 完成したら変更をコミット

## ファイル変更サマリー

**新規作成:**
- `echonet_lite/handler/DeviceHistory.go` (550行)
- `echonet_lite/handler/DeviceHistory_test.go` (880行)

**変更:**
- `echonet_lite/handler/handler_data_management.go` (+2行)
- `echonet_lite/handler/ECHONETLiteHandler.go` (+50行程度)
  - HistoryOptions フィールド追加
  - historyFilePath フィールド追加
  - 履歴ファイル読み込み・保存処理追加
- `protocol/protocol.go` (+14行)
- `server/server.go` (+8行)
  - 履歴オプション設定追加
- `server/websocket_server.go` (~30行変更)
  - HistoryOrigin を handler.HistoryOrigin に変更
- `server/websocket_server_handlers_history.go` (~10行変更)
  - handler層の型を使用するように変更
- `main.go` (+1行)
  - handler パッケージ import 追加

**削除:**
- `server/device_history_store.go` (543行)
- `server/device_history_store_test.go` (889行)
- `server/websocket_server_handlers_history_test.go`
- `server/websocket_server_history_test.go`

## ブランチ情報
- ブランチ名: `refactor/move-device-history-to-handler`
- 最新コミット: `a991ffe` - "refactor: move device history management from server to handler layer"
