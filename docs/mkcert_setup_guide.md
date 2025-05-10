# mkcertを使用した開発環境の証明書セットアップガイド

## 1. はじめに

このガイドでは、ECHONET Lite WebSocketサーバーでTLSを使用するための証明書を、mkcertを使用してセットアップする方法を説明します。mkcertは、ローカル開発環境用の信頼された証明書を簡単に作成できるツールです。

## 2. mkcertのインストール

### macOS

```bash
brew install mkcert
brew install nss  # Firefox用
```

### Windows

```bash
choco install mkcert
```

### Linux (Ubuntu/Debian)

```bash
sudo apt install libnss3-tools
sudo apt install mkcert
```

## 3. ローカル認証局（CA）のセットアップ

1. mkcertをインストールした後、ローカルCAをセットアップします：

```bash
mkcert -install
```

2. 証明書が正しくインストールされたことを確認：

```bash
mkcert -CAROOT
```

## 4. サーバー証明書の生成

1. 証明書を生成するディレクトリに移動：

```bash
cd /path/to/your/project/certs
```

2. 証明書を生成：

```bash
mkcert localhost 127.0.0.1 ::1
```

これにより、以下のファイルが生成されます：

- `localhost+2.pem` (証明書)
- `localhost+2-key.pem` (秘密鍵)

## 5. 証明書の設定

1. 生成された証明書をECHONET Lite WebSocketサーバーの設定に反映：

```toml
[websocket.tls]
enabled = true
cert_file = "certs/localhost+2.pem"
key_file = "certs/localhost+2-key.pem"
```

2. サーバーを起動する際にTLSを有効化：

```bash
./echonet-list -websocket -ws-tls -ws-cert-file certs/localhost+2.pem -ws-key-file certs/localhost+2-key.pem
```

## 6. モバイルデバイスでの使用

### iOS

1. 証明書をエクスポート：

```bash
mkcert -CAROOT
# 表示されたパスから rootCA.pem をコピー
```

2. 証明書をメールで送信するか、Webサーバーでホスト

3. iOSデバイスで証明書をインストール：
   - 証明書ファイルをタップ
   - 「設定」アプリで「プロファイルがダウンロードされました」をタップ
   - 「インストール」をタップ
   - 「設定」→「一般」→「プロファイル」で証明書を信頼

### Android

1. 証明書をエクスポート：

```bash
mkcert -CAROOT
# 表示されたパスから rootCA.pem をコピー
```

2. 証明書をメールで送信するか、Webサーバーでホスト

3. Androidデバイスで証明書をインストール：
   - 証明書ファイルをタップ
   - 証明書名を入力
   - 「VPNとアプリ」を選択
   - 「インストール」をタップ

## 7. トラブルシューティング

### 証明書が信頼されない場合

1. ローカルCAが正しくインストールされているか確認：

```bash
mkcert -CAROOT
```

2. 証明書の有効期限を確認：

```bash
openssl x509 -in localhost+2.pem -text -noout | grep "Not After"
```

3. 証明書の情報を確認：

```bash
openssl x509 -in localhost+2.pem -text -noout
```

### モバイルデバイスで接続できない場合

1. 証明書が正しくインストールされているか確認
2. デバイスの日時設定が正しいか確認
3. ブラウザのキャッシュをクリア
4. 必要に応じて証明書を再インストール

## 8. セキュリティに関する注意事項

1. 生成された証明書は開発環境専用です
2. 本番環境では、信頼された認証局（CA）から発行された証明書を使用してください
3. 証明書の秘密鍵は安全に保管し、バージョン管理システムにコミットしないでください
4. 定期的に証明書を更新することを推奨します

## 9. 証明書の更新

証明書の有効期限が近づいた場合：

1. 新しい証明書を生成：`mkcert localhost 127.0.0.1 ::1`
1. 古い証明書をバックアップ：`mv localhost+2.pem localhost+2.pem.bak && mv localhost+2-key.pem localhost+2-key.pem.bak`
1. 新しい証明書を配置し、サーバーを再起動
