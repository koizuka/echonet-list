# ECHONET Lite デバイス検出プログラム

WebSocketを使用したECHONET Liteデバイス制御システム

## 概要

このプロジェクトは、ECHONET Liteプロトコルを使用してホームネットワーク内のデバイスを発見し、デバイスの状態を取得・制御するためのツールです。WebSocketを使用してクライアント・サーバーアーキテクチャを採用しており、コマンドラインクライアントとサーバーコンポーネントに分かれています。

## 構成

- `server/` - WebSocketサーバー（ECHONET Lite通信を担当）
- `client/` - コマンドラインクライアント
- `protocol/` - クライアントとサーバー間で使用される共通プロトコル定義
- `echonet_lite/` - ECHONET Liteプロトコルの実装

## ビルド方法

### 依存関係

- Go 1.18以上

### サーバーのビルド

```shell
go build -o echonet-server ./server
```

### クライアントのビルド

```shell
go build -o echonet-client ./client
```

## 使い方

### サーバーの起動

```shell
./echonet-server -port 8080 -debug
```

オプション:
- `-port`: WebSocketサーバーのポート番号（デフォルト: 8080）
- `-debug`: デバッグモードを有効にする
- `-log`: ログファイル名を指定する（デフォルト: echonet-server.log）

### クライアントの使用

```shell
./echonet-client -server ws://localhost:8080/ws
```

オプション:
- `-server`: WebSocketサーバーのURL（デフォルト: ws://localhost:8080/ws）
- `-debug`: デバッグモードを有効にする

クライアントを起動すると、以下のようなコマンドが使用できます:

- `discover`: ECHONET Liteデバイスの発見
- `devices` or `list`: 検出されたデバイスの一覧表示
- `get`: デバイスのプロパティ値を取得
- `set`: デバイスのプロパティ値を設定
- `update`: デバイスのプロパティキャッシュを更新
- `debug`: デバッグモードの表示/切り替え
- `help`: ヘルプを表示
- `quit`: プログラムを終了

詳細なコマンドの使い方については、`help`コマンドを実行してください。

## サポートされているデバイスタイプ

このアプリケーションは、以下のECHONET Liteデバイスタイプをサポートしています:

- ホームエアコン (0x0130)
- 床暖房 (0x027b)
- 単機能照明 (0x0291)
- 照明システム (0x02a3)
- コントローラー (0x05ff)
- ノードプロファイル (0x0ef0)

## 使用例

### エアコンの発見と制御

1. アプリケーションを起動する
2. デバイスを発見する: `discover`
3. すべてのデバイスを一覧表示する: `devices`
4. エアコンの動作状態を取得する: `get 0130 80`
5. エアコンをオンにする: `set 0130 on`
6. 温度を25°Cに設定する: `set 0130 b3:19` （25°Cの16進数は0x19）

### 照明の制御

1. デバイスを発見する: `discover`
2. すべての照明デバイスを一覧表示する: `devices 0291`
3. 照明をオンにする: `set 0291 on`
4. 照明をオフにする: `set 0291 off`

### デバイスプロパティの更新

1. デバイスを発見する: `discover`
2. すべてのエアコンのすべてのプロパティを更新する: `update 0130`
3. 特定のデバイスのすべてのプロパティを更新する: `update 192.168.0.5 0130:1`
4. 更新されたプロパティを確認する: `devices 0130 -all`

## トラブルシューティング

### 一般的なエラー

#### ポートが既に使用されている

以下のエラーメッセージが表示される場合:
```
listen udp :3610: bind: address already in use
```
これは、別のインスタンスのアプリケーションがすでに実行されていて、UDPポート3610を使用していることを示しています。

**解決方法:**
1. 他のアプリケーションインスタンスを見つけて終了させる:
   - Linux/macOS: `ps aux | grep echonet-server` でプロセスを見つけ、`kill <PID>` で終了させる
   - Windows: タスクマネージャーを使用してプロセスを終了させる
2. 他のインスタンスを停止した後、アプリケーションを再実行する

## 参考文献

- [ECHONET Lite仕様](https://echonet.jp/spec_v114_lite/)
- [ECHONET Liteオブジェクト仕様](https://echonet.jp/spec_object_rr2/)