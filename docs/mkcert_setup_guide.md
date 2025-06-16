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

1. 証明書が正しくインストールされたことを確認：

```bash
mkcert -CAROOT
```

## 4. サーバー証明書の生成

1. 証明書を生成するディレクトリに移動：

```bash
cd /path/to/your/project/certs
```

1. 証明書を生成：

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

1. サーバーを起動する際にTLSを有効化：

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

1. 証明書をメールで送信するか、Webサーバーでホスト

1. iOSデバイスで証明書をインストール：
   - 証明書ファイルをタップ
   - 「設定」アプリで「プロファイルがダウンロードされました」をタップ
   - 「インストール」をタップ

1. ルート証明書を信頼する設定（重要）：
   - 「設定」→「一般」→「情報」→「証明書信頼設定」
   - インストールした証明書の「ルート証明書を全面的に信頼」をオンにする
   - 警告メッセージが表示されたら「続ける」をタップ

**注意**: 手動でインストールした証明書はSSL/TLS通信に対して自動的には信頼されません。必ず上記の手順4を実行して、ルート証明書を信頼する必要があります。

### Android

1. 証明書をエクスポート：

```bash
mkcert -CAROOT
# 表示されたパスから rootCA.pem をコピー
```

1. 証明書をメールで送信するか、Webサーバーでホスト

1. Androidデバイスで証明書をインストール：
   - 証明書ファイルをタップ
   - 証明書名を入力
   - 「VPNとアプリ」を選択
   - 「インストール」をタップ

## 7. mkcertを使わずに証明書をインストールする方法

### Windows

mkcertを使用せずに、既存のrootCA.pemをWindowsにインストールする場合：

1. **証明書の取得**
   - 他の環境でmkcertを使用して生成されたrootCA.pemを取得
   - または、`mkcert -CAROOT`で表示されるパスからrootCA.pemをコピー

2. **証明書マネージャーを使用したインストール**

   方法1: MMCコンソールを使用

   ```cmd
   # 管理者権限でコマンドプロンプトを開く
   mmc
   ```

   - ファイル → スナップインの追加と削除
   - 「証明書」を選択して「追加」
   - 「コンピューター アカウント」を選択
   - 「ローカル コンピューター」を選択
   - 「信頼されたルート証明機関」→「証明書」を右クリック
   - 「すべてのタスク」→「インポート」
   - rootCA.pemを選択してインポート

   方法2: certutilコマンドを使用

   ```cmd
   # 管理者権限でコマンドプロンプトを開く
   certutil -addstore -f "ROOT" rootCA.pem
   ```

3. **PowerShellを使用したインストール**

   ```powershell
   # 管理者権限でPowerShellを開く
   Import-Certificate -FilePath "rootCA.pem" -CertStoreLocation Cert:\LocalMachine\Root
   ```

4. **インストールの確認**

   ```cmd
   certutil -store Root | findstr "mkcert"
   ```

### 注意事項

- 管理者権限が必要です
- ブラウザの再起動が必要な場合があります
- Firefoxは独自の証明書ストアを使用するため、別途インポートが必要な場合があります

## 8. トラブルシューティング

### 証明書が信頼されない場合

1. ローカルCAが正しくインストールされているか確認：

```bash
mkcert -CAROOT
```

1. 証明書の有効期限を確認：

```bash
openssl x509 -in localhost+2.pem -text -noout | grep "Not After"
```

1. 証明書の情報を確認：

```bash
openssl x509 -in localhost+2.pem -text -noout
```

### モバイルデバイスで接続できない場合

1. 証明書が正しくインストールされているか確認
2. デバイスの日時設定が正しいか確認
3. ブラウザのキャッシュをクリア
4. 必要に応じて証明書を再インストール

## 9. セキュリティに関する注意事項

1. 生成された証明書は開発環境専用です
2. 本番環境では、信頼された認証局（CA）から発行された証明書を使用してください
3. 証明書の秘密鍵は安全に保管し、バージョン管理システムにコミットしないでください
4. 定期的に証明書を更新することを推奨します

## 10. 証明書の更新

証明書の有効期限が近づいた場合：

1. 新しい証明書を生成：`mkcert localhost 127.0.0.1 ::1`
1. 古い証明書をバックアップ：`mv localhost+2.pem localhost+2.pem.bak && mv localhost+2-key.pem localhost+2-key.pem.bak`
1. 新しい証明書を配置し、サーバーを再起動
