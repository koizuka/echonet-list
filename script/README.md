# ECHONET Lite Controller systemd管理スクリプト

このディレクトリには、Raspberry Pi (Ubuntu) 環境でECHONET Lite Controllerをsystemdサービスとして管理するためのスクリプトが含まれています。

## 含まれるスクリプト

### install-systemd.sh

systemdサービスとしてのインストールを行います。

**機能:**

- 専用ユーザー・グループの作成
- バイナリとWeb UIファイルの適切な場所への配置
- 設定ファイルとデータディレクトリのセットアップ
- systemdサービスの登録・起動
- logrotate設定のインストール

**使用方法:**

```bash
sudo ./script/install-systemd.sh
```

**前提条件:**

- `go build` でバイナリがビルド済み
- `cd web && npm run build` でWeb UIがビルド済み

### uninstall-systemd.sh

systemdサービスのアンインストールを行います。

**機能:**

- サービスの停止・無効化・削除
- インストールされたファイルの削除
- ユーザー・グループの削除
- データと設定の保持/削除を選択可能

**使用方法:**

```bash
sudo ./script/uninstall-systemd.sh
```

### update.sh

稼働中のサービスを新しいバイナリとWeb UIで更新します。

**機能:**

- サービスの安全な停止・更新・再起動
- 自動バックアップ作成
- 失敗時の自動ロールバック
- 設定ファイルとデータファイルの差分更新

**使用方法:**

```bash
sudo ./script/update.sh
```

**前提条件:**

- サービスが既にインストール済み
- `go build` と `cd web && npm run build` でファイルがビルド済み

## インストール先パス

| 項目 | パス |
|------|------|
| バイナリ | `/usr/local/bin/echonet-list` |
| 設定ファイル | `/etc/echonet-list/config.toml` |
| データディレクトリ | `/var/lib/echonet-list/` |
| Web UIファイル | `/usr/local/share/echonet-list/web/` |
| ログファイル | `/var/log/echonet-list.log` |
| PIDファイル | `/var/run/echonet-list.pid` |

## 使用例

### 初回インストール

```bash
# 1. プロジェクトのビルド
go build
cd web && npm run build && cd ..

# 2. systemdサービスのインストール
sudo ./script/install-systemd.sh

# 3. サービス状態確認
sudo systemctl status echonet-list
```

### アップデート

```bash
# 1. 新しいバージョンをビルド
go build
cd web && npm run build && cd ..

# 2. サービスを更新
sudo ./script/update.sh
```

### アンインストール

```bash
sudo ./script/uninstall-systemd.sh
```

## 管理コマンド

```bash
# サービス状態確認
sudo systemctl status echonet-list

# サービス停止/開始/再起動
sudo systemctl stop echonet-list
sudo systemctl start echonet-list
sudo systemctl restart echonet-list

# ログ確認
sudo journalctl -u echonet-list -f
sudo tail -f /var/log/echonet-list.log

# 自動起動の有効化/無効化
sudo systemctl enable echonet-list
sudo systemctl disable echonet-list
```

## トラブルシューティング

### サービスが起動しない場合

1. ログを確認

```bash
sudo journalctl -u echonet-list -n 50
sudo tail -50 /var/log/echonet-list.log
```

2. 設定ファイルの確認

```bash
sudo -u echonet /usr/local/bin/echonet-list -config /etc/echonet-list/config.toml -debug
```

3. 権限の確認

```bash
ls -la /var/lib/echonet-list/
ls -la /var/log/echonet-list.log
```

### ポート使用エラー

```bash
# 使用中のポートを確認
sudo lsof -i :8080
sudo netstat -tlnp | grep 8080

# 設定ファイルでポートを変更
sudo nano /etc/echonet-list/config.toml
```

## セキュリティ設定

スクリプトは以下のセキュリティ設定を自動で適用します:

- 専用の非特権ユーザー(`echonet`)での実行
- systemdによるプロセス分離とリソース制限
- 適切なファイル権限の設定
- 最小権限の原則に基づいたディレクトリアクセス

詳細なセキュリティ設定については、`@docs/daemon-setup.md` を参照してください。

## 注意事項

- これらのスクリプトはUbuntu/Debian系のLinux環境で動作確認されています
- 実行にはroot権限が必要です
- バックアップは自動削除されないため、定期的な手動削除をお勧めします
- macOS環境ではLinuxコンテナなどでのテストが必要です
