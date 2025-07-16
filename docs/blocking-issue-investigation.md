# Server Blocking Issue Investigation

## 概要

ECHONETリストサーバーがブロックし、Web UIからの discover と update コマンドが応答しなくなる問題を調査・対処しました。

## 改善済み: スマートな操作追跡システム

**従来の問題**: 毎回のupdate操作で大量のログが出力され、長期運用時にログが膨大になる

**新しい解決策**: 監視goroutineによる操作追跡システムを実装

### 新システムの特徴

1. **静かな正常動作**: 正常に完了した操作は最小限のログのみ
2. **異常時の詳細記録**: タイムアウトやエラー時のみ詳細なログを出力
3. **メモリ効率**: 完了した操作の情報は自動的に削除
4. **リアルタイム監視**: 5秒間隔でのタイムアウト検出

## 発見された問題点

### 1. ログローテーションの無限ループ

**問題**: `server/LogManager.go` のログローテーション処理が無限ループになっていた

```go
// 問題のあるコード
go func() {
    for {
        <-rotateSignalCh
        // ログローテーション処理
    }
}()
```

**対処**: selectステートメントとpanicリカバリを追加

### 2. 操作のタイムアウト不足

**問題**: discover と update 操作にタイムアウト制御がなく、デバイスが応答しない場合に永続的にブロック

**対処**: タイムアウト管理システムを実装

### 3. 監視システムの不足

**問題**: goroutineリークやチャンネルバッファの使用状況を監視する仕組みがない

**対処**: 監視システムを実装

## 実装した対策

### 1. スマート操作追跡システム ⭐ **新機能**

`echonet_lite/handler/handler_operation_tracker.go`:
- 操作の開始・完了を監視goroutineで追跡
- 正常完了時: デバッグログのみ（通常は非表示）
- 異常時のみ: 詳細なエラーログとタイムアウト情報
- メモリ効率: 完了した操作情報は自動削除

**タイムアウト設定**:
- デバイス検出: 30秒
- プロパティ更新: 60秒
- プロパティ取得: 10秒
- プロパティ設定: 10秒

### 2. WebSocketハンドラー改善

従来の詳細ログを操作追跡システムに置き換え：
- 正常時: ログ出力を大幅削減
- 異常時: 詳細な診断情報を記録
- フォールバック: 追跡システム無効時は従来方式で動作

### 3. システム監視（既存）

`echonet_lite/handler/handler_monitoring.go`:
- Goroutine数の監視
- メモリ使用量の監視
- チャンネルバッファ使用率の監視
- 異常値の検出とアラート

### 4. エラーハンドリング改善（既存）

- ログローテーション処理にpanicリカバリ追加
- selectステートメントによる適切なシグナル処理

## 使用方法

### 1. デバッグモードでの起動

```bash
./echonet-list -debug
```

### 2. ログ監視

```bash
# 標準出力でログを確認
./echonet-list

# デーモンモードでログファイルを監視
tail -f /var/log/echonet-list.log
```

### 3. システムメトリクス監視

ログで以下の情報を確認：
- `System metrics`: Goroutine数、メモリ使用量
- `High goroutine count detected`: 異常なGoroutine数
- `High channel buffer usage`: チャンネルバッファの高使用率

### 4. 操作追跡システム ⭐ **新機能**

**正常時のログ（デバッグレベル）**:
```
Operation started id=discover_20240101_120000.000 type=discover
Operation completed successfully id=discover_20240101_120000.000 duration=2.5s
```

**異常時のログ（警告・エラーレベル）**:
```
Operation timeout detected id=update_properties_20240101_120000.000 type=update_properties duration=65s timeout=60s
Operation failed id=discover_20240101_120000.000 type=discover duration=15s error=network timeout
```

## 今後の改善案

1. **プロメテウスメトリクス**: より詳細な監視のためのメトリクス出力
2. **ヘルスチェックエンドポイント**: HTTP経由での状態確認
3. **自動復旧機能**: 異常検出時の自動リスタート
4. **分散トレーシング**: リクエストのフロー追跡

## 関連ファイル

### 新規作成
- `echonet_lite/handler/handler_operation_tracker.go`: 操作追跡システム ⭐
- `echonet_lite/handler/handler_operation_tracker_test.go`: 操作追跡システムテスト
- `server/websocket_server_operation_tracker.go`: WebSocket操作追跡インターフェース
- `echonet_lite/handler/handler_timeout.go`: タイムアウト管理
- `echonet_lite/handler/handler_monitoring.go`: システム監視

### 更新
- `server/websocket_server_handlers_discovery.go`: 操作追跡システム統合
- `server/websocket_server_handlers_properties.go`: 操作追跡システム統合
- `echonet_lite/handler/handler_communication.go`: 操作追跡システム統合
- `echonet_lite/handler/handler_core.go`: 操作追跡システム統合
- `echonet_lite/handler/ECHONETLiteHandler.go`: GetCore()メソッド追加
- `server/LogManager.go`: ログローテーション改善