# Active Context

This file focuses on the current work and recent changes in the project, building on the foundation defined in [projectbrief.md](./projectbrief.md) and the architectural patterns in [systemPatterns.md](./systemPatterns.md).

## Current Task

最近の作業では、プロパティ変化通知機能を実装しました。また、以前にはWebSocketを通じてプロパティエイリアス情報を取得するための機能を実装し、その通信フォーマットを改善しました。WebSocketサーバーのリファクタリングを行い、テスト可能な構造に改善しました。デバイスグループ管理機能を実装し、WebSocketプロトコルのクライアント開発者向けドキュメントを作成し、WebSocketサーバーのTLS対応と設定ファイルのサポートを実装しました。

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

プロパティ変化通知機能の実装が完了し、WebSocketクライアントとサーバーの基本的な実装も完了しましたが、まだいくつかの課題が残っています：

1. **WebSocketクライアントの機能改善**:
   - `list`や`discover`などのコマンドが正しく動作するように実装を改善する
   - WebSocketクライアントとサーバーのテストを追加する

2. **アーキテクチャ分割の完了**:
   - ECHONET Liteに関する処理は web(WebSocket) サーバーにして、コンソールUIアプリはそれにアクセスするように分割する
   - 残りの機能をWebSocketプロトコル経由で利用できるようにする

3. **Web UI開発**: 上記分割が済んだら、web UIを作成する
   - プロパティ変化通知機能を活用して、リアルタイムに状態を更新するUIを実装する

## 現在の作業状況

プロパティ変化通知機能の実装が完了し、ECHONET Liteデバイスからのプロパティ値変化通知（INF）をWebSocketクライアントにリアルタイムで転送できるようになりました。これにより、フロントエンドアプリケーションがデバイスの状態変化をリアルタイムに検知し、表示を更新することが可能になります。

WebSocketサーバーのリファクタリングが完了し、テスト可能な構造に改善しました。WebSocketTransportインターフェースを導入し、大きなファイルを機能ごとに分割しました。これにより、コードの保守性とテスト容易性が向上しました。

WebSocketクライアントとサーバーの基本実装が完了し、`quit`コマンドの問題も解決しました。アプリケーションは`-ws-both`モードで起動し、WebSocketサーバーとクライアントの両方を同時に実行できるようになりました。

WebSocketプロトコルでのプロパティ値のBase64エンコード/デコード実装を行い、バイナリデータをJSONで安全に送受信できるようになりました。また、デバッグ表示コードを削除してコードをクリーンアップしました。

WebSocketクライアントの`list`コマンドが正しく動作するようになりましたが、他のコマンドはまだ実装中です。今後は残りのコマンドも正しく動作するように実装を進めていく予定です。

以前の作業では、デバイス追加通知機能の改善が完了し、実際の環境でのテストも行いました。`devices.json`を削除してアプリケーションを起動することで、起動時のdiscover処理によって通知が正しく送信されることを確認しました。

また、デバイス通知機能のユニットテストを`Devices_test.go`に追加し、以下の点を検証しました：

1. イベントチャンネルが正しく設定されるか
2. 新しいデバイスが登録されたときにイベントが送信されるか
3. 既に登録済みのデバイスに対しては重複してイベントが送信されないか
4. チャンネルがブロックされている場合（バッファがいっぱいの場合）にも問題なく動作するか

### タイムアウト通知機能のテスト

タイムアウト通知機能のテストを行いました。getコマンドに`-skip-validation`パラメータを追加し、デバイスの存在チェックをスキップしてタイムアウトの動作確認ができるようにしました。

テスト内容:
1. 存在しないIPアドレスに対して`get 192.168.0.254 0130:1 80 -skip-validation`コマンドを実行
2. タイムアウト通知が正しく表示されることを確認:
   ```
   デバイス 192.168.0.254 0130[Home air conditioner]:1 へのリクエストがタイムアウトしました: maximum retries reached (3) for device 192.168.0.254 0130[Home air conditioner]:1
   エラー: プロパティ取得に失敗: maximum retries reached (3) for device 192.168.0.254 0130[Home air conditioner]:1
   ```

この機能により、タイムアウト通知の動作確認が容易になりました。README.mdとCommand.goのヘルプ情報も更新し、この新しいパラメータについての説明を追加しました。

次のステップとして、以下の作業が必要です：

1.  **デバイスグループ管理機能の実装 (最優先)**
    *   **目的:** 複数のデバイスをグループとしてまとめ、一括操作を可能にする。Web UI のグループ操作機能の基盤となる。
    *   **永続化:** グループ定義を `groups.json` ファイルに保存する (`aliases.json` と同様の形式)。
    *   **グループ名のルール:** グループ名は必ず `@` プレフィックスで始まることとする。
    *   **CLI コマンド:**
        *   `group add @<group_name> <device_id1> ...`: グループ作成・デバイス追加 (`@` プレフィックス必須)
        *   `group remove @<group_name> <device_id1> ...`: グループからデバイス削除
        *   `group delete @<group_name>`: グループ削除
        *   `group list [@<group_name>]`: グループ一覧または詳細表示 (デバイスIDとエイリアスを表示)
    *   **既存コマンド拡張:**
        *   `set`, `get`, `update` コマンドの `<target>` 引数で `@` プレフィックス付きのグループ名を指定可能にする。
        *   名前解決ロジック: まず `@` プレフィックスを確認し、あればグループとして処理。なければエイリアス名またはデバイスIDとして処理。
        *   グループ指定時に、グループ内の全デバイスに対してコマンドを実行する。
    *   **実装:**
        *   グループ管理用の Go 構造体と JSON 永続化ロジックを実装。
        *   `console/` パッケージに `group` コマンドを追加 (`@` プレフィックスを考慮)。
        *   `console/CommandProcessor.go` を修正し、既存コマンドでの `@` プレフィックスによるグループ名解決とループ実行をサポート。

2.  **WebSocket プロトコル拡張 (グループ管理)**
    *   CLI でのグループ管理機能実装後、WebSocket 経由でグループを管理・操作するためのメッセージタイプを追加する。

3.  **WebSocketサーバーへの分割準備**
    *   現在のコードの依存関係を整理する。
    *   通知機能とグループ管理機能を WebSocket サーバーに統合するための設計を検討する。

4.  **Web UI 開発 (計画)**
    *   上記分割が完了次第、Web UI の開発に着手する予定です。
    *   **フレームワーク:** SvelteKit + TypeScript を使用する方針です。
    *   **ディレクトリ構成:** プロジェクトルートに `web-client` ディレクトリを作成し、その中で開発を行います。ビルド成果物は `server/webroot` に配置し、Go サーバーから配信します。
    *   **将来的な機能要件:**
        *   デバイス一覧を設置場所 (EPC 0x81) でグルーピング表示
        *   Web UI から設置場所 (EPC 0x81) を設定・変更
        *   デバイスの主要な状態 (ON/OFF, 温度等) を一覧で可視化
        *   複数デバイスのグループ操作機能 (グループ設定はサーバー側/設定ファイルで管理)
