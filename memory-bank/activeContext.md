# Active Context

This file focuses on the current work and recent changes in the project, building on the foundation defined in [projectbrief.md](./projectbrief.md) and the architectural patterns in [systemPatterns.md](./systemPatterns.md).

## Current Task

最近の作業では、`devices` コマンドに `-group-by <epc>` オプションを追加し、指定したEPCの値でデバイスをグループ化して表示する機能を実装しました。また、以前にはプロパティ変化通知機能を実装し、WebSocketを通じてプロパティエイリアス情報を取得するための機能を実装し、その通信フォーマットを改善しました。WebSocketサーバーのリファクタリングを行い、テスト可能な構造に改善しました。デバイスグループ管理機能を実装し、WebSocketプロトコルのクライアント開発者向けドキュメントを作成し、WebSocketサーバーのTLS対応と設定ファイルのサポートを実装しました。

### WebSocketを通じたプロパティエイリアス情報の提供機能と通信フォーマットの改善

WebSocketクライアントがクラスコードを指定して、そのクラスに対応するプロパティエイリアス情報を取得できる機能を実装しました。これにより、WebSocketを通してしかアクセスできない別言語のクライアントでもPropertyAliasの情報にアクセスできるようになりました。

さらに、プロパティエイリアス情報の通信フォーマットを改善し、EPCごとにグループ化された形式に変更しました。これにより、クライアント側でEPCに対する選択肢（エイリアスとその値）をより扱いやすくなりました。

実装内容：

1. `protocol/protocol.go`に`EPCInfo`構造体を新たに定義し、EPCの説明テキストとエイリアスマップを保持するようにしました。
2. `PropertyAliasesData`構造体を修正し、`aliases`フィールドを`properties`フィールドに変更しました。
3. `server/websocket_server_handlers_properties.go`ファイルの`handleGetPropertyAliasesFromClient`メソッドを修正し、`AvailablePropertyAliases`から取得したデータを新しいプロトコルフォーマットに変換するロジックを実装しました。
4. EPCとその説明テキスト、エイリアス名、EDT値を正しく解析し、EPCごとにグループ化するようにしました。
5. `server/websocket_server_handlers_properties_test.go`のテストケースを修正し、新しいレスポンスフォーマットの期待値に合わせてアサーションを更新しました。
6. `docs/websocket_client_protocol.md`を更新し、`property_aliases_result`メッセージのフォーマットに関する記述を修正しました。

新しいフォーマットでは、以下のような構造になります：

```json
{
  "classCode": "0130",
  "properties": {
    "80": {
      "description": "Operation status",
      "aliases": {
        "on": "MzA=",
        "off": "MzE="
      }
    },
    "B0": {
      "description": "Operation mode setting",
      "aliases": {
        "auto": "NDE=",
        "cooling": "NDI=",
        "heating": "NDM="
      }
    }
  }
}
```

この変更により、クライアント側でEPCごとに選択肢を表示したり、特定のEPCに対する操作をグループ化したりすることが容易になります。また、EPCの説明テキストも含まれるようになったため、ユーザーインターフェースでより分かりやすい表示が可能になりました。

### WebSocketサーバーのリファクタリング

WebSocketサーバーのコードをリファクタリングし、テスト可能な構造に改善しました。具体的には以下の変更を行いました：

1. **インターフェースの導入**: `WebSocketTransport` インターフェースを導入し、WebSocketサーバーのネットワーク層を抽象化しました。これにより、テスト時にmockに差し替えることが可能になりました。

2. **コードの分割**: 大きなファイルを機能ごとに分割し、コードの管理がしやすくなりました：
   - `websocket_server.go` - 基本的な構造と主要なメソッド
   - `websocket_server_handlers_properties.go` - プロパティ関連のハンドラメソッド
   - `websocket_server_handlers_management.go` - エイリアスやグループ管理関連のハンドラメソッド
   - `websocket_server_handlers_discovery.go` - デバイス検出関連のハンドラメソッド

3. **テスト容易性の向上**: インターフェースを使用することで、単体テストが書きやすくなりました。テスト時には、実際のWebSocketサーバーの代わりにモックを使用できます。

### デバイスグループ管理機能の実装

デバイスグループ管理機能を実装し、複数のデバイスをグループとしてまとめて一括操作できるようにしました。この機能は、Web UIのグループ操作機能の基盤となります。

実装内容：

1. `client/interfaces.go` に `GroupManager` インターフェースを追加
2. `echonet_lite/DeviceGroups.go` にグループ管理機能を実装
3. `console/Command.go` に `group` コマンドを追加
4. `console/CommandProcessor.go` を修正し、既存コマンドでの `@` プレフィックスによるグループ名解決とループ実行をサポート
5. `protocol/protocol.go` にグループ関連のプロトコルを追加
6. `server/websocket_server.go` と `client/websocket_client.go` にグループ管理機能を実装
7. `docs/websocket_client_protocol.md` にグループ関連の記述を追加

グループ名は必ず `@` プレフィックスで始まることとし、以下のコマンドを実装しました：

- `group add @<group_name> <device_id1> ...`: グループ作成・デバイス追加
- `group remove @<group_name> <device_id1> ...`: グループからデバイス削除
- `group delete @<group_name>`: グループ削除
- `group list [@<group_name>]`: グループ一覧または詳細表示

`set`、`get`、`update` コマンドの `<target>` 引数でグループ名（`@` プレフィックス付き）を指定できるようになり、グループ内の全デバイスに対して一括操作できるようになりました。例えば、`set @1F床暖房 on` と入力すると、グループ「@1F床暖房」に登録されている全デバイスの電源をONにすることができます。

### WebSocketプロトコルのクライアント開発者向けドキュメント

WebSocketプロトコルを使用してECHONET Liteデバイスと通信するクライアントアプリケーションの開発者向けに、詳細なドキュメントを作成しました。このドキュメントは、TypeScriptなど様々な言語でクライアントを実装する開発者をサポートするために作成されました。

ドキュメントの内容：

1. プロトコルの概要と基本的な通信フロー
2. WebSocketサーバーへの接続方法
3. メッセージフォーマットとデータ型の詳細
4. サーバーからクライアントへのメッセージ（通知）の種類と形式
5. クライアントからサーバーへのメッセージ（リクエスト）の種類と形式
6. サーバーからクライアントへのメッセージ（応答）の形式
7. クライアント実装のポイント（言語非依存）
8. エラーハンドリングの方法
9. TypeScriptでの実装例

ドキュメントは `docs/websocket_client_protocol.md` に保存され、将来的にクライアントアプリケーションを開発する際の参考資料として利用できます。

### WebSocketサーバーのTLS対応

WebSocketサーバーをTLS対応にし、ブラウザからの安全な接続（WSS）を可能にしました。具体的には以下の変更を行いました：

1. `server/websocket_server.go` に `StartOptions` 構造体を追加し、TLS証明書と秘密鍵のパスを指定できるようにしました
2. `Start()` メソッドを修正して、TLS証明書と秘密鍵を使用してサーバーを起動できるようにしました
3. WebSocketクライアントの接続先アドレスを修正し、TLSが有効な場合は `ws://` ではなく `wss://` を使用するようにしました

### 設定ファイルのサポート

TOML形式の設定ファイルをサポートし、コマンドライン引数と設定ファイルの両方から設定を読み込めるようにしました。具体的には以下の変更を行いました：

1. `config/config.go` パッケージを作成して、TOML設定ファイルの読み込みと、コマンドライン引数の解析を実装しました
2. `main.go` を修正して、設定ファイルの読み込みと、コマンドライン引数の適用を実装しました
3. サンプル設定ファイル `config.toml.sample` を作成し、`.gitignore` を更新して `config.toml` を除外しました

### 開発環境用の証明書作成と整理

`mkcert` を使用して開発環境用の証明書を作成し、TLS対応のWebSocketサーバーをテストできるようにしました。

1. `mkcert` をインストールし、ローカルCAをインストールしました
2. localhost の証明書を作成しました（有効期限: 2027年6月30日）
3. 証明書ファイル用の `certs` ディレクトリを作成し、証明書ファイルを移動しました
4. `config.toml` と `config.toml.sample` を更新して、証明書ファイルのパスを修正しました
5. `.gitignore` を更新して、localhost用の証明書はリポジトリに含め、それ以外の証明書は除外するようにしました

これらの変更により、WebSocketサーバーがTLS対応になり、設定ファイルのサポートが追加されました。また、開発環境でのテストが容易になりました。

以前の作業では、`DeviceFromProtocol` 関数の改善を行いました。この関数は、WebSocketプロトコルで使用される `protocol.Device` 型から ECHONET Lite の型に変換する役割を持っています。

具体的には以下の変更を行いました：

1. 戻り値の型を `map[string]string` から `echonet_lite.Properties` に変更しました
2. 関数内部で、プロパティの変換処理を修正し、文字列形式のプロパティを `echonet_lite.Property` 型に直接変換するようにしました
3. クライアントコードを修正し、新しい戻り値の型を使用するようにしました

これらの変更により、プロパティの処理が簡素化され、型変換のコードが削減されました。また、WebSocketクライアントとサーバー間の通信がより効率的になりました。

以前の作業では、WebSocketプロトコルでのプロパティ値のBase64エンコード/デコード実装を行いました。ECHONET Liteのプロパティ値（EDT）はバイナリデータであり、JSONでそのまま送受信することができないため、Base64エンコードを使用してテキスト形式に変換する必要がありました。

具体的には以下の変更を行いました：

1. `protocol/protocol.go`の`DeviceToProtocol`関数を修正し、プロパティ値をBase64エンコードするようにしました
2. `protocol/protocol.go`の`DeviceFromProtocol`関数を修正し、プロパティ値をBase64デコードするようにしました
3. `server/websocket_server.go`の各関数を修正し、`DeviceToProtocol`関数を使用するようにしました
4. `client/websocket_client.go`の各関数を修正し、Base64エンコードされたプロパティ値を正しくデコードするようにしました
5. デバッグ用の表示コードを削除し、コードをクリーンアップしました

これらの変更により、バイナリデータをJSONで安全に送受信できるようになり、データの整合性が保たれるようになりました。

以前の作業では、WebSocketクライアントとサーバーの実装を進め、特に`quit`コマンド実行時のアプリケーション終了処理を改善しました。以前は`quit`コマンドを実行するとアプリケーションがフリーズし、強制終了が必要でしたが、適切なリソース解放処理を実装することでこの問題を解決しました。

以前の作業では、サーバー化のための構成変更の準備として、コンソールUIに関わる部分を `console/` ディレクトリに移動しました。これにより、UIとバックエンドの責務がより明確に分離され、将来的なWebSocketサーバーとUI分離のアーキテクチャへの移行が容易になります。

この変更に伴い、以下のファイルが `console/` ディレクトリに移動されました：

- `Command.go`：コマンドの基本構造と解析処理
- `CommandTable.go`：コマンド定義テーブルとヘルプ表示機能
- `CommandProcessor.go`：コマンド処理と実行
- `Completer.go`：コマンドライン補完機能
- `Completer_test.go`：コマンドライン補完のテスト
- `ConsoleProcess.go`：メインのコンソールUIプロセス

また、クライアント・サーバーモデルへの移行準備として、以下のパッケージも整理されました：

- `client/`：クライアント実装
- `server/`：サーバー実装
- `protocol/`：プロトコル定義

以前の作業では、helpコマンドの機能を拡張しました。コマンドの使用方法の表示がだいぶ長くなってきたため、helpコマンドの引数にコマンド名を与えると、そのコマンドの情報だけ絞り込んで表示し、引数無しだと概要と全コマンドのサマリーだけが出るように改善しました。

この変更により、ユーザーは必要な情報だけを簡単に参照できるようになり、コマンドラインインターフェースの使いやすさが向上しました。

その前の作業では、デバイスの追加通知の仕組みを改善しました。`Devices`構造体に独自のイベント通知チャネルを追加し、`ECHONETLiteHandler`との依存関係を解消しました。デバイスの追加検出は`Devices.ensureDeviceExists`メソッド内で行い、そこからイベントを送信するようにしました。`ECHONETLiteHandler`はそのイベントを受け取って`DeviceNotification`に変換し、中継するだけの役割になりました。

この通知系の追加は、システムを疎結合にし、フロントエンドが状態変化をリアルタイムに受け取れるようにするためのものです。これは将来的なアーキテクチャ分割（ECHONET Lite処理をWebSocketサーバーに分離し、コンソールUIやWeb UIがそれに接続する形態）を見据えた設計です。今後はプロパティ変化通知なども実装していく予定で、これによりフロントエンドコンポーネントがデバイスの状態変化をリアルタイムに検知できるようになります。

### プロパティ変化通知機能の実装

ECHONET Liteデバイスからのプロパティ値変化通知（INF）を受信した際に、WebSocketクライアントにリアルタイムで通知する機能を実装しました。これにより、フロントエンドアプリケーションがデバイスの状態変化をリアルタイムに検知し、表示を更新することが可能になります。

実装内容：

1. `echonet_lite/ECHONETLiteHandler.go` に `PropertyChangeNotification` 構造体を定義し、変化があったデバイス (`IPAndEOJ`) とプロパティ (`Property`) を含めるようにしました。
2. `ECHONETLiteHandler` 構造体に `PropertyChangeCh` フィールドを追加し、プロパティ変化通知用のチャネルとして使用します。
3. `NewECHONETLiteHandler` 関数で `PropertyChangeCh` を初期化し、バッファサイズを100に設定しました。
4. `onInfMessage` 関数内で、ECHONET Liteデバイスからプロパティ変化通知（INF）を受信し、`h.registerProperties` で内部状態を更新した後、変更があった各プロパティについて `PropertyChangeNotification` を作成し、`PropertyChangeCh` に送信するようにしました。
5. `server/websocket_server.go` の `listenForNotifications` 関数を拡張し、`handler.PropertyChangeCh` からの通知も監視するようにしました。
6. プロパティ変化通知を受信したら、その情報を `protocol.PropertyChangedPayload` 形式に変換し（IP, EOJ, EPCを文字列化し、EDTをBase64エンコード）、接続している全てのWebSocketクライアントに `property_changed` メッセージとして送信するようにしました。

この機能により、以下のようなメリットがあります：

- フロントエンドアプリケーションがデバイスの状態変化をリアルタイムに検知できる
- ポーリングによる定期的な状態確認が不要になり、ネットワークトラフィックが削減される
- ユーザーインターフェースの応答性が向上し、ユーザー体験が改善される
- システムが疎結合になり、将来的なWebSocketサーバーとUI分離のアーキテクチャが実現しやすくなる

この機能は、既に実装済みのデバイス追加通知とデバイスタイムアウト通知の仕組みを拡張したものであり、同様のパターンを使用しています。WebSocketクライアント側では、`client/websocket_notifications.go` の `handlePropertyChanged` 関数で通知を受け取り、内部状態を更新します。

## Recent Changes

- デーモンモード機能を実装しました
  - `-daemon` フラグおよび `daemon` セクションでデーモンモードを有効化
  - `-pidfile` フラグおよび `daemon.pid_file` で PID ファイル指定
  - デーモンモード時のみ SIGHUP による `logger.AutoRotate()` を有効化
  - デーモンモード時はコンソール UI を起動せず、バックグラウンドで WebSocket サーバーを起動し終了まで待機
  - PID ファイルの作成・削除を実装
  - デーモンモード時は WebSocket クライアントモードを無効化

- デバイスのオフライン状態管理機能を追加しました
  - `echonet_lite/Devices.go` に `IsOffline` と `SetOffline` メソッドを追加しました。
  - `echonet_lite/ECHONETLiteHandler.go` で `Session` からのタイムアウト通知 (`DeviceTimeout`) を受け取った際に、`DataManagementHandler.SetOffline` を呼び出して該当デバイスをオフラインとしてマークするようにしました。

- WebSocketサーバーの定期的なプロパティ自動更新機能を設定ファイルで指定可能にしました
  - `config/config.go` の `Config.WebSocket` に `PeriodicUpdateInterval` (string) を追加し、デフォルト値を "1m" に設定しました。
  - `main.go` で設定ファイルからこの値を読み込み、`time.ParseDuration` でパースして `server.StartOptions` に渡すように修正しました (パース失敗時はデフォルト1分)。
  - `server/websocket_server.go` の `StartOptions` に `PeriodicUpdateInterval` (time.Duration) フィールドを追加しました。
  - `server/websocket_server.go` の `Start` および `Stop` メソッドを修正し、Tickerの開始・停止を `PeriodicUpdateInterval` の値に基づいて制御するようにしました。
  - これにより、`config.toml` で `websocket.periodic_update_interval = "30s"` のように更新間隔を指定できます (0以下または無効な値で無効化)。

- プロパティ変化通知機能を実装・修正しました
  - `echonet_lite/ECHONETLiteHandler.go` に `PropertyChangeNotification` 構造体を定義しました
  - `ECHONETLiteHandler` 構造体に `PropertyChangeCh` フィールドを追加しました
  - `onInfMessage` 関数内でプロパティ変化通知を送信するようにしました
  - `server/websocket_server.go` の `listenForNotifications` 関数を拡張し、プロパティ変化通知を処理するようにしました
  - WebSocketクライアントへのプロパティ変化通知のブロードキャスト機能を実装しました
  - WebSocketクライアントのプロパティ変化通知処理を修正しました
    - `client/websocket_notifications.go` の `handleInitialState`, `handleDeviceAdded`, `handleDeviceUpdated`, `handleDeviceRemoved`, `handlePropertyChanged` 関数を修正しました
    - デバイスマップのキーとして `ipAndEOJ.String()` ではなく `ipAndEOJ.Specifier()` を使用するように変更しました
    - これにより、デバイス識別子の一貫性が保たれ、プロパティ変更通知が正しく処理されるようになりました

- WebSocketサーバーのリファクタリングを行いました
  - `WebSocketTransport` インターフェースを導入し、WebSocketサーバーのネットワーク層を抽象化しました
  - 大きなファイルを機能ごとに分割し、コードの管理がしやすくなりました
    - `websocket_server.go` - 基本的な構造と主要なメソッド
    - `websocket_server_handlers_properties.go` - プロパティ関連のハンドラメソッド
    - `websocket_server_handlers_management.go` - エイリアスやグループ管理関連のハンドラメソッド
    - `websocket_server_handlers_discovery.go` - デバイス検出関連のハンドラメソッド
  - インターフェースを使用することで、単体テストが書きやすくなりました

- WebSocketプロトコルのクライアント開発者向けドキュメントを作成しました
  - `docs/websocket_client_protocol.md` ファイルを作成し、WebSocketプロトコルの詳細な仕様と使用方法を記述しました
  - プロトコルのメッセージ形式、データ型、通知、リクエスト、応答などを詳細に説明しました
  - TypeScriptでの実装例を含め、言語に依存しない形でクライアント実装のポイントを解説しました
  - このドキュメントにより、様々な言語でWebSocketクライアントを実装する開発者がプロトコルを理解しやすくなります

- `client/websocket_client.go` の `ListDevices` メソッドを修正し、IPアドレスのソート順を改善しました
  - WebSocketクライアントで表示されるデバイスリストのIPアドレスが文字列順ではなく数値順でソートされるようにしました
  - `sort.Slice` と `bytes.Compare` を使用して、IPv4/IPv6両対応のソート処理を実装しました
  - これにより、「192.168.0.9」と「192.168.0.10」のような場合に、正しく「192.168.0.9」の後に「192.168.0.10」が表示されるようになりました
  - 元の `echonet_lite/Devices.go` の `ListDevicePropertyData` メソッドと同様のソートロジックを使用しています

- `protocol/protocol.go` の `DeviceToProtocol` 関数を改善しました
  - 引数の型を `map[echonet_lite.EPCType][]byte` から `echonet_lite.Properties` に変更しました
  - 関数内部で、プロパティの処理ロジックを `echonet_lite.Properties` スライスをループするように修正しました
  - `server/websocket_server.go` と `protocol/protocol_test.go` も修正し、新しい引数型に対応させました
  - これにより、プロパティの処理が一貫性を持ち、型変換のコードがさらに削減されました

- `protocol/protocol.go` の `DeviceFromProtocol` 関数を改善しました
  - 戻り値の型を `map[string]string` から `echonet_lite.Properties` に変更しました
  - 関数内部で、プロパティの変換処理を修正し、文字列形式のプロパティを `echonet_lite.Property` 型に直接変換するようにしました
  - クライアントコードを修正し、新しい戻り値の型を使用するようにしました
  - これにより、プロパティの処理が簡素化され、型変換のコードが削減されました
- WebSocketサーバーの`handleGetProperties`関数を修正し、クライアントが期待する形式（配列ではなく単一のデバイスオブジェクト）でレスポンスを返すようにしました
  - 問題：クライアントは単一の`protocol.Device`オブジェクトを期待していましたが、サーバーは`[]protocol.Device`配列を返していました
  - 修正：サーバーが結果の最初のデバイスだけをJSONにシリアライズして返すようにしました
  - これにより、`get`コマンド実行時の「json: cannot unmarshal array into Go value of type protocol.Device」エラーが解消されました
- WebSocketプロトコルでのプロパティ値のBase64エンコード/デコード実装を行いました
- `protocol/protocol.go`の`DeviceToProtocol`関数と`DeviceFromProtocol`関数を修正し、プロパティ値をBase64エンコード/デコードするようにしました
- WebSocketサーバーとクライアントのコードを修正し、Base64エンコードされたプロパティ値を正しく処理するようにしました
- デバッグ用の表示コードを削除し、コードをクリーンアップしました
- WebSocketクライアントの`list`コマンドが正しく動作するようになりました
- `Command.go`を`Command.go`（コマンドの基本構造と解析処理）と`CommandTable.go`（コマンド定義テーブルとヘルプ表示機能）に分割し、コードの責務をより明確に分離しました
- `Devices.go`に`DeviceEventType`と`DeviceEvent`型を定義しました
- `Devices`構造体に`EventCh`フィールドを追加しました
- `SetEventChannel`メソッドを追加して、イベントチャネルを設定できるようにしました
- `ensureDeviceExists`メソッド内でデバイス追加時にイベントチャネルに通知を送信するようにしました
- `ECHONETLiteHandler`の`NewECHONETLiteHandler`関数内でデバイスイベント用チャンネルを作成し、`Devices`に設定するようにしました
- デバイスイベントを受け取り、`DeviceNotification`に変換して中継するゴルーチンを実装しました
- `onInfMessage`メソッド内のデバイス追加通知部分を削除し、代わりに`Devices.ensureDeviceExists`からの通知を使用するようにしました

## Next Steps

主要な機能（プロパティ変化通知、グループ管理、WebSocketプロトコル定義、サーバーリファクタリング、TLS対応、設定ファイル）の実装は完了しました。今後の主な作業は以下の通りです。

1.  **デバイスのオフライン状態管理 (最優先)**:
    *   デバイスとの通信がタイムアウトした場合に、そのデバイスを「オフライン」状態としてマークする機能を実装します。
    *   オフライン状態のデバイスに対する挙動を定義します。例えば：
        *   `get`/`set`/`update` コマンド実行時にエラーを返すか、警告を表示するか。
        *   `list`/`devices` コマンドでオフライン状態を明示的に表示する。
        *   WebSocketクライアント（コンソールUI、Web UI）での表示方法を検討する。
        *   オフライン状態から復帰する条件（例：再度の通信成功、手動リセット）を定義する。
    *   関連する通知（`device_updated` など）やデータ構造（`echonet_lite.Device` など）への影響を考慮します。

2.  **WebSocketクライアントの機能改善**:
    *   コンソールUI（WebSocketクライアント）で、`list`以外のコマンド（`discover`, `get`, `set`, `update`, `alias`, `group`など）がWebSocket経由で正しく動作するように実装を完了させます。
    *   WebSocketクライアントとサーバー間の通信に関するテストを追加します。
    *   オフライン状態のデバイスを適切に扱えるようにクライアント側の実装を修正します。

3.  **Web UI 開発 (計画)**:
    *   デバイスのオフライン状態管理機能の実装後、Web UIの開発に着手します。
    *   詳細は後述の「Web UI 開発 (計画)」セクションを参照してください。

**注記:** 当初の計画にあった「アーキテクチャ分割（CLIクライアントの分離）」は、現時点では優先度を下げ、実装を省略します。WebSocketサーバー機能は引き続き維持・改善しますが、コンソールUIを完全に別プロセスにする作業は見送ります。

## 現在の作業状況

デバイスグループ管理機能の実装が完了し、CLIおよびWebSocket経由でのグループ操作が可能になりました。プロパティ変化通知機能も実装済みで、WebSocketクライアントはデバイスの状態変化をリアルタイムに受信できます。WebSocketサーバーのリファクタリング、TLS対応、設定ファイルサポートも完了しています。WebSocketプロトコルのドキュメント (`docs/websocket_client_protocol.md`) も整備済みです。

現在は、デバイスのオフライン状態管理機能の設計と実装、およびコンソールUI（WebSocketクライアント）の機能実装を進める必要があります。

### タイムアウト通知機能のテスト

タイムアウト通知機能のテストを行いました。getコマンドに`-skip-validation`パラメータを追加し、デバイスの存在チェックをスキップしてタイムアウトの動作確認ができるようにしました。

テスト内容:

1. 存在しないIPアドレスに対して`get 192.168.0.254 0130:1 80 -skip-validation`コマンドを実行
2. タイムアウト通知が正しく表示されることを確認:

   ```bash
   デバイス 192.168.0.254 0130[Home air conditioner]:1 へのリクエストがタイムアウトしました: maximum retries reached (3) for device 192.168.0.254 0130[Home air conditioner]:1
   エラー: プロパティ取得に失敗: maximum retries reached (3) for device 192.168.0.254 0130[Home air conditioner]:1
   ```

この機能により、タイムアウト通知の動作確認が容易になりました。README.mdとCommand.goのヘルプ情報も更新し、この新しいパラメータについての説明を追加しました。

## Web UI 開発 (計画)

アーキテクチャ分割完了後、以下の計画でWeb UIの開発に着手します。

-   **目的:** ECHONET LiteデバイスをWebブラウザ経由で視覚的に監視・操作するインターフェースを提供します。
-   **アーキテクチャ:** WebSocketサーバー (`server/websocket_server.go`) と通信するシングルページアプリケーション (SPA) として実装します。
-   **通信プロトコル:** `docs/websocket_client_protocol.md` で定義されたJSONベースのWebSocketプロトコルを使用します。
-   **フロントエンド技術:**
    -   **Framework Consideration**: UIの試行錯誤とメンテナンスを容易にするため、React, Vue, SvelteなどのコンポーネントベースのJavaScriptフレームワークの採用を検討します。これらのフレームワークは、UIパーツの再利用、状態管理、開発時のホットリロード機能を提供し、開発効率を高めることが期待されます。
-   **配信方法:**
    -   `echonet-list` アプリケーション自体にHTTPサーバー機能を追加し、ビルドされたWeb UIの静的ファイル（HTML/CSS/JS）を配信します。
    -   Go標準の `net/http` パッケージを使用し、`http.FileServer` で静的ファイルを配信します。
    -   HTTPサーバー用のポートは設定ファイル (`config.toml` の `http_port`) で指定可能にします（WebSocketポートとは別）。
    -   これにより、WebSocketサーバーと同一オリジンからの配信となり、CORSの問題を回避できます。
-   **開発ワークフロー:**
    1.  **Source Code Location**: Web UIのフロントエンドコードは、プロジェクトルート直下の `webui/` ディレクトリ（仮称）で管理します。
    2.  **Build Process**: `webui/` ディレクトリ内で、選択したフレームワークのビルドコマンド（例: `npm run build`）を実行し、静的なHTML, CSS, JavaScriptファイルを生成します。
    3.  **Asset Deployment**: ビルドされた静的ファイルを、GoサーバーがWebコンテンツを提供するために設定されたディレクトリ（`config.toml` の `http_webroot` で指定。例: `server/webroot/`）にコピーします。このプロセスはMakefileやスクリプトで自動化することを検討します。
-   **サーバー側アセット再読み込み:**
    -   **Development Phase**: 開発中は、Web UIの静的ファイルを更新した後、Goサーバーを再起動して変更を反映させます。
    -   **Future Enhancement**: UI更新の頻度が高くなった場合、サーバーを停止せずにUIアセットを再読み込みする機能の導入を検討します。有力な方法として、Console UIに新しいコマンド（例: `reload-webui`）を追加し、実行時にHTTPサーバーに `http_webroot` の内容を再読み込みさせる方式が考えられます。
-   **クライアント側自動リロード:**
    -   **Mechanism**: サーバーがWeb UIアセットの再読み込みを完了した後、接続中の全WebSocketクライアントに新しい通知メッセージ (`ui_updated` など、別途定義) を送信します。
    -   **Client Action**: Webクライアント（ブラウザのJavaScript）は、この通知を受信したら自動的にページをリロード (`window.location.reload()`) し、最新のUIアセットを取得します。
-   **機能要件:**
    -   デバイス一覧を設置場所 (EPC 0x81) でグルーピング表示
    -   Web UI から設置場所 (EPC 0x81) を設定・変更
    -   デバイスの主要な状態 (ON/OFF, 温度等) を一覧で可視化
    -   複数デバイスのグループ操作機能 (既存の `manage_group` WebSocket API を利用)
    -   リアルタイム状態更新 (WebSocket通知を利用)
