# Scripts

このディレクトリには、ECHONET Lite Controllerの開発・運用のためのユーティリティスクリプトが含まれています。

## 含まれるスクリプト

### auto-update.sh

Raspberry Pi での運用を想定した自動更新スクリプトです。git pull の結果に基づいて自動的にビルドとサービス更新を行います。

**機能:**

- git pull の実行と変更ファイル分析
- 変更内容に応じた自動ビルド判定
  - Go ファイル変更 → サーバービルド
  - Web UI ファイル変更 → Web UI ビルド
  - 両方の変更 → 全体ビルド
- 自動サービス更新（systemd 対応）
- ドライランモード（実際の操作なしで動作確認）
- スクリプト自己更新の検出と再実行促進

**使用方法:**

```bash
# 通常実行（自動でsudoに昇格）
./script/auto-update.sh

# ドライランモード（git pull せずに動作確認）
./script/auto-update.sh -d
./script/auto-update.sh --dry-run

# ヘルプ表示
./script/auto-update.sh -h
./script/auto-update.sh --help
```

**動作フロー:**

1. git pull でリモートの変更を取得
2. 変更されたファイルを分析してビルド対象を判定
3. 必要に応じて `./script/build.sh` を実行
4. ビルドが行われた場合、`./script/update.sh` でサービス更新

**ドライランモードの利点:**

- git fetch のみ実行（リポジトリ状態を変更しない）
- 実際のリモート差分を表示
- 何度でも同じ結果でテスト可能

**前提条件:**

- Git リポジトリ内で実行
- systemd サービスが既にインストール済み（update.sh 実行時）
- リモートリポジトリへのアクセス権限

### build.sh

サーバーとWeb UIのビルドを行います。引数により部分ビルドも可能です。

**機能:**

- サーバーバイナリのビルド
- Web UI の依存関係インストールとビルド
- 引数による部分ビルド対応

**使用方法:**

```bash
# 全てビルド（デフォルト）
./script/build.sh
./script/build.sh all

# サーバーのみビルド
./script/build.sh server

# Web UIのみビルド
./script/build.sh web
./script/build.sh web-ui
```

**出力:**

- サーバーバイナリ: `./echonet-list`
- Web UI バンドル: `./web/bundle/`

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

- `./script/build.sh` でバイナリとWeb UIがビルド済み

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
- `./script/build.sh` でファイルがビルド済み

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
./script/build.sh

# 2. systemdサービスのインストール
sudo ./script/install-systemd.sh

# 3. サービス状態確認
sudo systemctl status echonet-list
```

### アップデート

**自動更新（推奨）:**

```bash
# 全自動で git pull → ビルド → サービス更新
./script/auto-update.sh

# 事前確認（ドライラン）
./script/auto-update.sh -d
```

**手動更新（従来の方法）:**

```bash
# 1. 新しいバージョンをビルド
./script/build.sh

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
