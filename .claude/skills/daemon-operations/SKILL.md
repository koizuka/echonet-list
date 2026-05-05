---
name: daemon-operations
description: Operates and debugs echonet-list when running as a systemd background daemon, covering PID files, log rotation via SIGHUP, and remote log access through ECHONET_SERVER_HOST. Use when modifying daemon-mode behavior in main.go / config.go, debugging a production server, or investigating live logs on the deployed host.
---

# デーモンモード運用

`./echonet-list -daemon -websocket` でバックグラウンド常駐させたときの構成と運用知識。Linux/macOS で systemd サービスとして動かす想定。

## 動作

1. WebSocket サーバーが必須（コンソール UI が使えないため）
2. PID ファイルを作成（デフォルト: `/var/run/echonet-list.pid`）
3. ログファイルパスを自動切り替え（デフォルト: `/var/log/echonet-list.log`）
4. `SIGHUP` でログローテーション実行
5. `SIGTERM` / `SIGINT` で正常終了

## 主要ファイル

- `config/config.go`
  - `getDefaultPIDFile()`: OS 別 PID ファイルパス
  - `getDefaultDaemonLogFile()`: OS 別ログファイルパス
- `main.go`: PID ファイル作成・削除、SIGHUP ハンドラ、コンソール UI 無効化
- `systemd/echonet-list.service`: systemd サービス定義
- `systemd/config.toml.systemd`: systemd 用設定サンプル
- `systemd/echonet-list.logrotate`: logrotate 設定

詳細セットアップ手順は `docs/daemon-setup.md` を参照。

## デバッグ

- コンソール出力なし → ログファイルで確認
- systemd 経由なら `journalctl -u echonet-list -f` も可
- 権限エラー時は書き込み可能なパスを `-pidfile` で指定

## 実サーバーのログ参照

`ECHONET_SERVER_HOST` 環境変数（`.claude/settings.local.json` 定義）で SSH 接続し、root 権限のログを `sudo` で読む:

```bash
ssh "$ECHONET_SERVER_HOST" sudo cat /var/log/echonet-list.log
ssh "$ECHONET_SERVER_HOST" sudo tail -f /var/log/echonet-list.log
```
