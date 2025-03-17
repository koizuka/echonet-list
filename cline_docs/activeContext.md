# Active Context

## Current Task
最近の作業では、Devices.Filter の FilterCriteria から EPCs を廃止し、代わりに Command の PropertyMode に指定した EPCs を使用するように変更しました。この変更は完了し、すべてのテストが通過しています。

## Recent Changes
- `FilterCriteria` 構造体から `EPCs` フィールドを削除しました
- `Filter` メソッドを修正して、EPCs フィールドを使わないようにしました
- `CommandProcessor.go` の `processDevicesCommand` メソッドを修正して、Command の EPCs を使って結果をフィルタリングするようにしました
- `Command.go` の `parseDevicesCommand` メソッドを修正して、"-all" または "-props" オプションが指定された場合に EPCs をクリアするようにしました
- `PrintUsage` 関数の説明を更新して、"-all", "-props", "epc" は最後に指定されたものが有効になることを明記しました
- `Filter_test.go` の EPCs フィールドを使用するテストケースを削除しました

## Next Steps
現在の開発サイクルで計画されていた機能はすべて実装されています。今後の計画は以下の通りです：

1. **メッセージ再送機能の実装**: Session でメッセージを送信したあと、返信を必要としているものについて、返信タイムアウトになったときには同一メッセージを再送する仕組みを実装する
2. **アーキテクチャ分割**: ECHONET Liteに関する処理は web(WebSocket) サーバーにして、コンソールUIアプリはそれにアクセスするように分割する
3. **Web UI開発**: 上記分割が済んだら、web UIを作成する

## 現在の作業状況
現在は、将来の計画に向けた準備段階にあります。メッセージ再送機能の設計を検討中で、Session.go ファイルの修正が必要になります。また、WebSocketサーバーへの分割に向けて、現在のコードの依存関係を整理しています。

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
