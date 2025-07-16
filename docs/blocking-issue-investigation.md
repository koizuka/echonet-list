# Server Blocking Issue Investigation

## 概要

ECHONETリストサーバーがブロックし、Web UIからの discover と update コマンドが応答しなくなる問題を調査・対処しました。

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

### 1. デバッグログの追加

WebSocketハンドラーと通信ハンドラーに詳細なログを追加：

- 操作開始・完了時刻の記録
- 処理時間の測定
- エラー発生時の詳細情報

### 2. タイムアウト管理システム

`echonet_lite/handler/handler_timeout.go`:
- デバイス検出: 30秒
- プロパティ更新: 60秒
- プロパティ取得: 10秒
- プロパティ設定: 10秒

### 3. システム監視

`echonet_lite/handler/handler_monitoring.go`:
- Goroutine数の監視
- メモリ使用量の監視
- チャンネルバッファ使用率の監視
- 異常値の検出とアラート

### 4. エラーハンドリング改善

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

## 今後の改善案

1. **プロメテウスメトリクス**: より詳細な監視のためのメトリクス出力
2. **ヘルスチェックエンドポイント**: HTTP経由での状態確認
3. **自動復旧機能**: 異常検出時の自動リスタート
4. **分散トレーシング**: リクエストのフロー追跡

## 関連ファイル

- `server/websocket_server_handlers_discovery.go`: デバッグログ追加
- `server/websocket_server_handlers_properties.go`: デバッグログ追加
- `echonet_lite/handler/handler_communication.go`: デバッグログ追加
- `echonet_lite/handler/handler_timeout.go`: タイムアウト管理（新規）
- `echonet_lite/handler/handler_monitoring.go`: システム監視（新規）
- `server/LogManager.go`: ログローテーション改善