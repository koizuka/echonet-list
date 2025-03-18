# Active Context

## Current Task
最近の作業では、デバイスの追加通知の仕組みを改善しました。`Devices`構造体に独自のイベント通知チャネルを追加し、`ECHONETLiteHandler`との依存関係を解消しました。デバイスの追加検出は`Devices.ensureDeviceExists`メソッド内で行い、そこからイベントを送信するようにしました。`ECHONETLiteHandler`はそのイベントを受け取って`DeviceNotification`に変換し、中継するだけの役割になりました。

## Recent Changes
- `Devices.go`に`DeviceEventType`と`DeviceEvent`型を定義しました
- `Devices`構造体に`EventCh`フィールドを追加しました
- `SetEventChannel`メソッドを追加して、イベントチャネルを設定できるようにしました
- `ensureDeviceExists`メソッド内でデバイス追加時にイベントチャネルに通知を送信するようにしました
- `ECHONETLiteHandler`の`NewECHONETLiteHandler`関数内でデバイスイベント用チャンネルを作成し、`Devices`に設定するようにしました
- デバイスイベントを受け取り、`DeviceNotification`に変換して中継するゴルーチンを実装しました
- `onInfMessage`メソッド内のデバイス追加通知部分を削除し、代わりに`Devices.ensureDeviceExists`からの通知を使用するようにしました

## Next Steps
現在の開発サイクルで計画されていた機能はすべて実装されています。今後の計画は以下の通りです：

1. **アーキテクチャ分割**: ECHONET Liteに関する処理は web(WebSocket) サーバーにして、コンソールUIアプリはそれにアクセスするように分割する
2. **Web UI開発**: 上記分割が済んだら、web UIを作成する

## 現在の作業状況
デバイス追加通知機能の改善が完了し、実際の環境でのテストも行いました。`devices.json`を削除してアプリケーションを起動することで、起動時のdiscover処理によって通知が正しく送信されることを確認しました。次のステップとして、以下の作業が必要です：

1. **通知機能の追加テスト**
   - タイムアウト通知が正しく送信されるかテストする
   - 通知チャネルの動作を確認し、必要に応じて調整する

2. **WebSocketサーバーへの分割準備**
   - 現在のコードの依存関係を整理する
   - 通知機能をWebSocketサーバーに統合するための設計を検討する

この作業が完了したら、Web UIの開発に着手する予定です。

## Cline Commands
以下は、Clineに対して定義されたカスタムコマンドです：

- `行数`: すべてのGoファイルの行数をカウントします
  - 実行コマンド: `find . -name "*.go" -print0 | xargs -0 wc -l`
  - 使用例: "行数を教えて" と言うと、プロジェクト内のすべてのGoファイルの行数が表示されます

- `ドキュメントを更新`: Command.goとREADME.mdのドキュメントを更新します
  - 実行手順:
    1. Command.goファイルを読み、必要に応じてPrintUsage()関数を更新します
    2. README.mdファイルを読み、必要に応じて更新します
  - 使用例: "ドキュメントを更新して" と言うと、最新のコマンド仕様に基づいてドキュメントが更新されます
