# for Claude

## プロジェクト概要

- ECHONET Lite コントローラーを Go で開発しているプロジェクト。家庭 LAN 内動作前提で認証は持たない。
- console UI と Web UI を提供する。Web UI は TypeScript + React 19 で実装され、WebSocket でリアルタイム通信する。
- WebSocket プロトコル仕様や Web UI 開発ガイドは @docs/ にある。

## Server Build & Test Commands

サーバーはプロジェクトルートで作業する。`cd {フルパス}` してから実行:

- Build: `go build`
- Run: `./echonet-list [-debug]`
- Run as daemon: `./echonet-list -daemon -websocket`
- Test: `go test ./...`
- Format: `gofmt -w .`
- Check: `go vet ./...`

コミット前に format / test / build がエラーなく通ることを確認する。

## Web UI Build & Test Commands

Web UI は `web` ディレクトリで作業する。`cd {フルパス}/web` してから実行:

- Build: `npm run build`
- Dev Server: `npm run dev`
  - カスタム WebSocket URL: `VITE_WS_URL=wss://custom-host:8080/ws npm run dev`
- Test: `npm run test`
- Lint: `npm run lint`
- Type Check: `npm run typecheck`

コミット前に lint / typecheck / test / build がエラーなく通ることを確認する。

ビルド結果は `web/bundle/` に出力され、Go サーバーが HTTP で配信する（WebSocket と同一ホスト → CORS 解決）。

## アーキテクチャ概要

- 対応ブラウザ: PC Chrome, iPhone Safari, Android Chrome
- フロントエンド: React 19 + TypeScript + shadcn/ui (Tailwind ベース) + Vite + Vitest
- WebSocket クライアント層は React Hooks にする
- ディレクトリ:
  - `web/{src/{components,hooks,libs},public,bundle}`
- リアルタイムにプロパティ変更を反映。タブは「設置場所」「デバイスグループ」「@グループ名」で切り替え。プロパティは EPC コードではなく人間可読な名前と適切な UI コントロールで表示・編集する。

## コーディング規約

- React は関数型スタイルで書く:

  ```tsx
  import React from 'react';
  import { UIComponent } from 'whatever';

  type ComponentProps = { prop1: number, ...}
  export function SomeComponent(props: ComponentProps) {
    const state = useSomething(...);
    return <>
      <UIComponent prop={prop1} />
    </>;
  }
  ```

- ユニットテストはターゲットと同一ディレクトリに `*.test.ts` で配置する。
- shadcn/ui を優先的に使う。

## 開発手順（テストファースト）

1. 要求仕様に基づくユニットテストを先に書く
2. ターゲット関数を最低限定義してテストがコンパイルエラーにならないようにする
3. テストが失敗することを確認
4. テストが通るよう実装
5. 通るまで反復
6. 通ったらビルド
7. ビルドしたターゲットファイル（例: Web UI なら `web/bundle/`）をサーバーが読み込めるよう、必要な配線（埋め込み・配信パス・ハンドラ等）を修正する
8. lint 警告を解消
9. `./echonet-list` で起動して動作確認をユーザーに依頼

## 関連スキル

条件付きで必要な詳細手順は専用スキルに切り出してある。状況に応じて参照:

- `webui-troubleshooting`: Web UI でプロパティが表示されない/編集できないときの診断
- `property-visibility`: コンパクトモードの条件付き表示 `PROPERTY_VISIBILITY_CONDITIONS`
- `echonet-i18n`: PropertyDesc/PropertyTable と `get_property_description` の多言語対応
- `daemon-operations`: systemd デーモン運用、ログローテーション、リモートログ参照

## 参考ドキュメント

- `docs/websocket_client_protocol.md`: WebSocket プロトコル仕様
- `docs/web_ui_implementation_guide.md`: Web UI 実装ガイド
- `docs/client_ui_development_guide.md`: クライアント UI 開発ガイド
- `docs/react_hooks_usage_guide.md`: React Hooks 使用ガイド
- `docs/console_ui_usage.md`: Console UI コマンド一覧（`debugoffline`, `location` 他）
- `docs/internationalization.md`: 国際化対応の詳細
- `docs/daemon-setup.md`: デーモンセットアップ
- `docs/configuration.md` / `docs/installation.md` / `docs/quick-start.md`
- `docs/troubleshooting.md` / `docs/error_handling_guide.md`
- `docs/integration-testing.md` / `docs/network-monitoring.md` / `docs/server-modes.md`
- `docs/device_types.md`
- `docs/mkcert_setup_guide.md`
- `docs/blocking-issue-investigation.md`

## 環境変数

- `ECHONET_SERVER_HOST`: 実サーバー（デーモン稼働ホスト）の SSH 接続先。`.claude/settings.local.json` で定義。ログ参照は `daemon-operations` スキル参照。
