# Progress

This file tracks the implementation progress of the project features defined in [projectbrief.md](./projectbrief.md) and planned in [activeContext.md](./activeContext.md).

## What Works

- ECHONET Lite デバイスの検出と制御
- エイリアス機能（作成、表示、削除、一覧表示、永続化）
- メッセージ再送信機能（タイムアウト時に最大3回再試行）
- ヘルプコマンドの拡張（コマンド名指定で詳細表示）
- WebSocketサーバーとクライアント実装
  - TLS対応
  - 設定ファイルサポート
  - 通知機能（デバイス追加、タイムアウト、プロパティ変更）
- デバイスグループ管理機能
  - グループの作成、表示、削除、一覧表示
  - `@`プレフィックスによるグループ指定
  - グループ内の全デバイスに対する一括操作
- `devices`コマンドのグループ化機能（`-group-by <epc>`オプション）

## What's Left to Build

- **アーキテクチャ分割**: ECHONET Liteに関する処理は web(WebSocket) サーバーにして、コンソールUIアプリはそれにアクセスするように分割する
  - 実装予定: 新しいパッケージ構造の設計と実装
  - 状態: 依存関係の整理中
- **Web UI開発**: 上記分割が済んだら、web UIを作成する
  - 実装予定: フロントエンドの設計と実装
  - 状態: 未着手（アーキテクチャ分割後に開始）
  - **詳細な機能要件:**
    - デバイス一覧を設置場所 (EPC 0x81) でグルーピング表示
    - Web UI から設置場所 (EPC 0x81) を設定・変更
    - デバイスの主要な状態 (ON/OFF, 温度等) を一覧で可視化
    - 複数デバイスのグループ操作機能 (グループ設定はサーバー側/設定ファイルで管理)

## Completed Features

- ✅ **デバイス通知機能**: デバイスの追加通知とタイムアウト通知の実装
- ✅ **プロパティ変化通知機能**: デバイスのプロパティ値変化をリアルタイム通知
- ✅ **WebSocketサーバーのリファクタリング**: テスト可能な構造への改善
- ✅ **WebSocketプロトコルのクライアント開発者向けドキュメント**: 詳細な仕様と実装例の提供
- ✅ **WebSocketサーバーのTLS対応**: 安全な接続（WSS）のサポート
- ✅ **設定ファイルのサポート**: TOML形式の設定ファイル対応
- ✅ **デバイスグループ管理機能**: グループ作成と一括操作の実装
- ✅ **Devices Command Grouping Enhancement**: EPCの値でデバイスをグループ化表示
