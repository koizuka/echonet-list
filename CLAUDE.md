# for Claude

## プロジェクト概要

- このプロジェクトは家電制御規格の ECHONET Lite のコントローラーを Go で開発しています。
  - 当面は家庭 LAN 内で動作させる想定のため、認証などは持ちません。
  - このコントローラーは、console UI と Web UIを提供します。
  - Web UI 本体は TypeScript で実装済みで、WebSocket によるリアルタイム通信が動作しています。
    - WebSocket のプロトコルや Web UI 開発ガイドは @docs/ に記述しています。

## Server Build & Test Commands

- Build: `go build`
- Run: `./echonet-list [-debug]`
- Test: `go test ./...`
- Format: `go fmt ./...`
- Check: `go vet ./...`

## Web UI Build & Test Commands

- Build: `cd web && npm run build`
- Dev Server: `cd web && npm run dev`
- Test: `cd web && npm run test`
- Lint: `cd web && npm run lint`

Web UI のビルド結果は `web/bundle/` に出力され、Go サーバーがHTTPサーバーとして配信します。

## Web UI の実装状況

### 実装済み機能

- **タブベースナビゲーション**: 設置場所とデバイスグループによるタブ分け表示
- **プロパティ表示改善**: EPCコードを人間可読な名前で表示
- **インタラクティブ編集**: プロパティ種類に応じたUIコントロール（ドロップダウン、数値入力）
- **デバイスエイリアス表示**: エイリアス名での分かりやすいデバイス識別
- **レスポンシブデザイン**: モバイル・デスクトップ対応
- **リアルタイム更新**: WebSocketによるプロパティ変更のリアルタイム反映

### 技術詳細

- **フレームワーク**: React 19 + TypeScript
- **UIライブラリ**: shadcn/ui (Tailwind CSS ベース)
- **ビルドツール**: Vite
- **テスト**: Vitest
- **主要ファイル**:
  - `src/App.tsx`: メインアプリケーション
  - `src/components/PropertyEditor.tsx`: プロパティ編集コンポーネント
  - `src/libs/propertyHelper.ts`: プロパティ名・値変換
  - `src/libs/locationHelper.ts`: ロケーション・グループ管理
  - `src/libs/deviceIdHelper.ts`: デバイス識別子処理

## Web UI の仕様

- PC のChrome, iPhone Safari, Android Chromeで動作させます。
- 構成ファイルは、サーバーが http サーバーを持ち、それが配信することで、 WebSocketサーバーと同一ホストで動作させ、CORS 制約を解決します。
- 開発:
  - 構成ファイルのソースコードはリポジトリの web サブディレクトリ内で開発します。
    - node.js, Vite, TypeScript, React 19 を用いて開発します。Vitest でテストを用意します。なるべく新しいスタイルを使います。
    - WebSocket の通信クライアント層を用意し、 React Hooks にします。
    - ディレクトリ構成は、以下の様になります:
      - web/
        - node_modules/
        - bundle/ - ビルド結果が格納される
        - public/
        - src/
          - components/
          - hooks/
          - libs/
    - ユニットテストコードは、ターゲットコードと同一のディレクトリで、 `*.ts` なら `*.test.ts` という名前で配置します。
    - React は関数型スタイルで開発します。コードスタイルは以下の様な感じです:

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

    - UI framework は shadcn/ui を試してみたいです。

- 実行:
  - WebSocketでリアルタイムに情報を更新・反映します。
  - 基本画面ではLAN内の機器を「設置場所」と「デバイスグループ」でタブ分けし、各機器は alias 名でユーザーに分かりやすく識別させ、on/off や動作モード、現在の温度などの状態を表示し、変更を可能とします。
  - 複数の機器をグループ化したグループ画面も実装済みで、"@" プレフィックス付きのグループ名でタブ表示されます。
  - プロパティはEPCコードではなく人間可読な名前で表示され、プロパティの種類に応じて適切なUIコントロール（ドロップダウン、数値入力など）で編集可能です。

## 開発手順

- タスクを開始するときは、テストファーストで行います。
  1. 最初に要求仕様に基づいたユニットテストを書きます。
  2. ユニットテストがコンパイルエラーにならない程度にターゲット関数を定義します。
  3. まず今回作ったユニットテストが失敗することを確認します。
  4. 次に、ユニットテストが通るようにターゲット関数を実装します。
  5. ユニットテストが通るまで修正とテストを繰り返します。
  6. すべてのユニットテストが通ったら、ビルドを行います。
  7. ビルドしたターゲットファイルをサーバーが読み込めるよう、適切な修正を行います。
  8. `npm run lint` で警告が出たら修正します。
  9. サーバーを ./echonet-lite で起動し、ユーザーに動作確認してもらいます。
