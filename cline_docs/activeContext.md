# Active Context

## Current Task
Devices.Filter の FilterCriteria から EPCs を廃止し、代わりに Command の PropertyMode に指定した EPCs を使用するように変更しました。

## Recent Changes
- `FilterCriteria` 構造体から `EPCs` フィールドを削除しました
- `Filter` メソッドを修正して、EPCs フィールドを使わないようにしました
- `CommandProcessor.go` の `processDevicesCommand` メソッドを修正して、Command の EPCs を使って結果をフィルタリングするようにしました
- `Command.go` の `parseDevicesCommand` メソッドを修正して、"-all" または "-props" オプションが指定された場合に EPCs をクリアするようにしました
- `PrintUsage` 関数の説明を更新して、"-all", "-props", "epc" は最後に指定されたものが有効になることを明記しました
- `Filter_test.go` の EPCs フィールドを使用するテストケースを削除しました

## Next Steps
1. ✅ `FilterCriteria` 構造体から `EPCs` フィールドを削除 (COMPLETED)
2. ✅ `Filter` メソッドを修正して、EPCs フィールドを使わないようにする (COMPLETED)
3. ✅ `CommandProcessor.go` の `processDevicesCommand` メソッドを修正 (COMPLETED)
4. ✅ `Command.go` の `parseDevicesCommand` メソッドを修正 (COMPLETED)
5. ✅ `PrintUsage` 関数の説明を更新 (COMPLETED)
6. ✅ `Filter_test.go` のテストケースを修正 (COMPLETED)

## 将来の計画 (Future Plans)
1. Session でメッセージを送信したあと、返信を必要としているものについて、返信タイムアウトになったときには同一メッセージを再送する仕組みを実装する
2. ECHONET Liteに関する処理は web(WebSocket) サーバーにして、コンソールUIアプリはそれにアクセスするように分割する
3. 上記分割が済んだら、web UIを作成する
