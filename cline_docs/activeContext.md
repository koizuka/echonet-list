# Active Context

## Current Task
最近の作業では、Session.goにメッセージ再送機能を実装し、タイムアウト時に最大3回まで再送する仕組みを追加しました。また、ECHONETLiteHandlerのGetPropertiesとSetPropertiesメソッドを修正して、この新しい機能を利用するようにしました。

## Recent Changes
- `Session` 構造体に `MaxRetries` と `RetryInterval` フィールドを追加しました
- `unregisterCallback` 関数を実装して、コールバックの登録解除を適切に行えるようにしました
- `CreateSetPropertyMessage` 関数を追加して、`CreateGetPropertyMessage` との一貫性を持たせました
- `sendRequestWithContext` 関数を実装して、タイムアウト検出と再送処理の共通ロジックを提供しました
- `GetPropertiesWithContext` と `SetPropertiesWithContext` メソッドを追加しました
- `ECHONETLiteHandler` の `GetProperties` と `SetProperties` メソッドを修正して、新しいWithContextメソッドを使用するようにしました
- `ECHONETLiteHandler` の `UpdateProperties` メソッドを修正して、`GetPropertiesWithContext` を使用し、go routineによる並列処理を実装しました
- 部分的な成功の場合のエラーハンドリングを改善しました

## Next Steps
現在の開発サイクルで計画されていた機能はすべて実装されています。今後の計画は以下の通りです：

1. **アーキテクチャ分割**: ECHONET Liteに関する処理は web(WebSocket) サーバーにして、コンソールUIアプリはそれにアクセスするように分割する
2. **Web UI開発**: 上記分割が済んだら、web UIを作成する

## 現在の作業状況
メッセージ再送機能の実装が完了しました。次のステップとして、WebSocketサーバーへの分割に向けて、現在のコードの依存関係を整理しています。この分割が完了したら、Web UIの開発に着手する予定です。

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
