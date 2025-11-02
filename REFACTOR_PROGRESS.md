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

## 残りの作業 🚧

### 5. websocket_server_handlers_history.go の更新
**現在のエラー:**
```
server/websocket_server_handlers_history.go:16: ws.historyStore undefined
server/websocket_server_handlers_history.go:51: ws.historyStore undefined
server/websocket_server_handlers_history.go:64: ws.historyStore undefined
```

**必要な変更:**
- 16行目: `ws.historyStore == nil` → `ws.GetHistoryStore() == nil`
- 51行目: `ws.historyStore.PerDeviceTotalLimit()` → `ws.GetHistoryStore().PerDeviceTotalLimit()`
- 60-64行目: `HistoryQuery` → `handler.HistoryQuery`
- 64行目: `ws.historyStore.Query()` → `ws.GetHistoryStore().Query()`
- 77-90行目: `HistoryOrigin` → `handler.HistoryOrigin` への変換
- 89行目: `entry.Value` → `protocol.PropertyDataFromHandlerValue(entry.Value)`

### 6. handler層で履歴ファイルの読み込み・保存を実装

**ECHONETLiteHandlerに追加が必要:**
- 履歴ファイルパスの管理
- 起動時の履歴読み込み
- 終了時の履歴保存

**実装箇所:**
- `echonet_lite/handler/ECHONETLiteHandler.go`:
  - コンストラクタで `HistoryOptions` を受け取る
  - 初期化時に `LoadFromFile()` を呼び出す
  - Shutdown メソッドで `SaveToFile()` を呼び出す

### 7. config設定の調整

**必要な変更:**
- `config/config.go`:
  - `HistoryFilePath` を設定可能に
  - `PerDeviceSettableLimit` と `PerDeviceNonSettableLimit` を設定可能に

- サンプルconfig (`systemd/config.toml.systemd`):
  - 履歴設定の追加例を記載

- ドキュメント:
  - `CLAUDE.md` に履歴設定の説明を追加

### 8. 古いファイルの削除

**削除対象:**
- `server/device_history_store.go`
- `server/device_history_store_test.go`
- `server/device_history_store.go` 内の型定義（`HistoryOrigin` など）

### 9. テストと動作確認

- [ ] `go test ./...` で全テスト成功
- [ ] `go build` で警告なしビルド成功
- [ ] Web UIでデバイス履歴表示が正常動作
- [ ] 履歴ファイルの保存・読み込みが正常動作

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

1. **websocket_server_handlers_history.go を更新**
   - 上記エラーをすべて修正
   - ビルドが通ることを確認

2. **handler層に履歴永続化を実装**
   - ECHONETLiteHandler に HistoryOptions 追加
   - 起動時ロード・終了時保存を実装

3. **古いファイルを削除**
   - server層の device_history_store 関連ファイル削除

4. **テストと動作確認**
   - 全体テスト実行
   - Web UI 動作確認

5. **コミット**
   - 完成したら PR 作成

## ファイル変更サマリー

**新規作成:**
- `echonet_lite/handler/DeviceHistory.go` (550行)
- `echonet_lite/handler/DeviceHistory_test.go` (880行)

**変更:**
- `echonet_lite/handler/handler_data_management.go` (+2行)
- `echonet_lite/handler/ECHONETLiteHandler.go` (+2行)
- `protocol/protocol.go` (+14行)
- `server/websocket_server.go` (-48行, +40行)

**削除予定:**
- `server/device_history_store.go` (543行)
- `server/device_history_store_test.go` (889行)

## ブランチ情報
- ブランチ名: `refactor/move-device-history-to-handler`
- 最新コミット: `a991ffe` - "refactor: move device history management from server to handler layer"
