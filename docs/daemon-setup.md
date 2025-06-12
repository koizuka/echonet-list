# ECHONET Lite Controller デーモンセットアップガイド

このドキュメントでは、ECHONET Lite ControllerをLinuxシステムでデーモンとして設定・運用する方法を説明します。

## 目次

1. [概要](#概要)
2. [前提条件](#前提条件)
3. [インストール手順](#インストール手順)
4. [systemdでの運用](#systemdでの運用)
5. [ログ管理](#ログ管理)
6. [トラブルシューティング](#トラブルシューティング)
7. [セキュリティ考慮事項](#セキュリティ考慮事項)

## 概要

ECHONET Lite Controllerのデーモンモードでは、以下の機能が利用できます：

- バックグラウンドでの常時稼働
- 自動起動設定
- ログローテーション
- PIDファイルによるプロセス管理
- systemdによるサービス管理

## 前提条件

- Linux（Ubuntu 20.04以降、Debian 10以降、CentOS 8以降など）
- systemd対応のディストリビューション
- Go 1.21以降（ビルドする場合）
- root権限またはsudo権限

## インストール手順

### 1. 専用ユーザーの作成

セキュリティのため、専用ユーザーでサービスを実行します：

```bash
# ユーザーとグループの作成
sudo groupadd echonet
sudo useradd -r -g echonet -d /var/lib/echonet-list -s /sbin/nologin echonet

# 必要なディレクトリの作成
sudo mkdir -p /var/lib/echonet-list
sudo mkdir -p /etc/echonet-list
sudo mkdir -p /var/log
sudo mkdir -p /usr/local/share/echonet-list

# 権限設定
sudo chown echonet:echonet /var/lib/echonet-list
sudo chmod 750 /var/lib/echonet-list
```

### 2. バイナリのインストール

```bash
# ビルド（ソースコードがある場合）
go build -o echonet-list

# バイナリをシステムにインストール
sudo cp echonet-list /usr/local/bin/
sudo chmod 755 /usr/local/bin/echonet-list
```

### 3. 設定ファイルの配置

```bash
# 設定ファイルのコピー
sudo cp systemd/config.toml.systemd /etc/echonet-list/config.toml
sudo chown root:echonet /etc/echonet-list/config.toml
sudo chmod 640 /etc/echonet-list/config.toml

# 必要に応じて設定を編集
sudo nano /etc/echonet-list/config.toml
```

### 4. Web UIファイルの配置（必要な場合）

```bash
# Web UIファイルのコピー
sudo cp -r web/bundle/* /usr/local/share/echonet-list/web/bundle/
sudo chown -R root:root /usr/local/share/echonet-list
```

### 5. デバイス設定ファイルの配置

```bash
# 既存の設定ファイルがある場合
sudo cp devices.json /var/lib/echonet-list/
sudo cp aliases.json /var/lib/echonet-list/
sudo cp groups.json /var/lib/echonet-list/
sudo chown echonet:echonet /var/lib/echonet-list/*.json
```

## systemdでの運用

### 1. systemdサービスファイルのインストール

```bash
# サービスファイルのコピー
sudo cp systemd/echonet-list.service /etc/systemd/system/
sudo chmod 644 /etc/systemd/system/echonet-list.service

# systemdデーモンのリロード
sudo systemctl daemon-reload
```

### 2. サービスの管理

```bash
# サービスの開始
sudo systemctl start echonet-list

# サービスの停止
sudo systemctl stop echonet-list

# サービスの再起動
sudo systemctl restart echonet-list

# サービスの状態確認
sudo systemctl status echonet-list

# 起動時の自動開始を有効化
sudo systemctl enable echonet-list

# 起動時の自動開始を無効化
sudo systemctl disable echonet-list
```

### 3. ログの確認

```bash
# systemdジャーナルログの確認
sudo journalctl -u echonet-list -f

# アプリケーションログの確認
sudo tail -f /var/log/echonet-list.log
```

## ログ管理

### logrotateの設定

ログファイルの自動ローテーションを設定します：

```bash
# logrotate設定ファイルのインストール
sudo cp systemd/echonet-list.logrotate /etc/logrotate.d/echonet-list
sudo chmod 644 /etc/logrotate.d/echonet-list

# 設定のテスト
sudo logrotate -d /etc/logrotate.d/echonet-list
```

### 手動でのログローテーション

```bash
# PIDファイルからプロセスIDを取得してSIGHUPを送信
sudo kill -HUP $(cat /var/run/echonet-list.pid)
```

## トラブルシューティング

### サービスが起動しない場合

1. ログを確認：

   ```bash
   sudo journalctl -u echonet-list -n 50
   sudo tail -50 /var/log/echonet-list.log
   ```

2. 設定ファイルの検証：

   ```bash
   sudo -u echonet /usr/local/bin/echonet-list -config /etc/echonet-list/config.toml -debug
   ```

3. 権限の確認：

   ```bash
   ls -la /var/lib/echonet-list/
   ls -la /var/log/echonet-list.log
   ls -la /var/run/echonet-list.pid
   ```

### PIDファイルエラー

PIDファイルのディレクトリが存在しない場合：

```bash
# /var/run配下にディレクトリを作成（再起動後も保持される設定）
sudo mkdir -p /var/run
sudo touch /var/run/echonet-list.pid
sudo chown echonet:echonet /var/run/echonet-list.pid
```

または、systemdのtmpfilesを使用：

```bash
# /etc/tmpfiles.d/echonet-list.conf を作成
echo "d /run/echonet-list 0755 echonet echonet -" | sudo tee /etc/tmpfiles.d/echonet-list.conf
sudo systemd-tmpfiles --create
```

### ポート使用エラー

デフォルトポート（8080）が使用中の場合：

```bash
# 使用中のポートを確認
sudo lsof -i :8080
sudo netstat -tlnp | grep 8080

# 設定ファイルでポートを変更
sudo nano /etc/echonet-list/config.toml
# [http_server] セクションのportを変更
```

## セキュリティ考慮事項

### ファイアウォール設定

必要なポートのみを開放：

```bash
# UFW（Ubuntu/Debian）の場合
sudo ufw allow 8080/tcp comment "ECHONET-List Web UI"

# firewalld（CentOS/RHEL）の場合
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

### TLS/SSL設定

本番環境では、TLSを有効にすることを推奨：

1. 証明書の準備：

   ```bash
   sudo mkdir -p /etc/echonet-list/certs
   # Let's Encryptまたは自己署名証明書を配置
   ```

2. 設定ファイルでTLSを有効化：

   ```toml
   [tls]
   enabled = true
   cert_file = "/etc/echonet-list/certs/server.crt"
   key_file = "/etc/echonet-list/certs/server.key"
   ```

### アクセス制限

必要に応じて、特定のIPアドレスからのみアクセスを許可：

```toml
[http_server]
host = "127.0.0.1"  # ローカルホストのみ
# または
host = "192.168.1.100"  # 特定のインターフェース
```

## 運用のベストプラクティス

1. **定期的なバックアップ**：

   ```bash
   # 設定とデータのバックアップ
   sudo tar -czf echonet-backup-$(date +%Y%m%d).tar.gz \
     /etc/echonet-list/ \
     /var/lib/echonet-list/*.json
   ```

2. **監視の設定**：
   - systemdのRestartオプションで自動再起動
   - 外部監視ツール（Nagios、Zabbixなど）との連携

3. **ログの監視**：
   - エラーログの定期的な確認
   - ディスク容量の監視

4. **アップデート手順**：

   ```bash
   # サービスの停止
   sudo systemctl stop echonet-list
   
   # バイナリの更新
   sudo cp new-echonet-list /usr/local/bin/echonet-list
   
   # サービスの開始
   sudo systemctl start echonet-list
   ```
