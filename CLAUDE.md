# for Claude

## プロジェクト概要

- このプロジェクトは家電制御規格の ECHONET Lite のコントローラーを Go で開発しています。
  - 当面は家庭 LAN 内で動作させる想定のため、認証などは持ちません。
  - このコントローラーは、console UI と Web UIを提供します。
  - Web UI 本体は TypeScript で実装済みで、WebSocket によるリアルタイム通信が動作しています。
    - WebSocket のプロトコルや Web UI 開発ガイドは @docs/ に記述しています。

## Server Build & Test Commands

- サーバーはプロジェクトルートディレクトリで作業します。 `cd {フルパス}` してから実行してください:
  - Build: `go build`
  - Run: `./echonet-list [-debug]`
  - Run as daemon: `./echonet-list -daemon -websocket`
  - Test: `go test ./...`
  - Format: `gofmt -w .`
  - Check: `go vet ./...`
  - コミット前には、format, test, buildしてエラーがないことを確認してください。

## Web UI Build & Test Commands

- Web UI は `web` ディレクトリ内で作業します。 `cd {フルパス}` してから実行してください:
  - Build: `npm run build`
  - Dev Server: `npm run dev`
  - Dev Server (with custom WebSocket URL): `VITE_WS_URL=wss://custom-host:8080/ws npm run dev`
  - Test: `npm run test`
  - Lint: `npm run lint`
  - Type Check: `npm run typecheck`
  - コミット前には、lint, typecheck, test, buildでエラーがないことを確認してください。

Web UI のビルド結果は `web/bundle/` に出力され、Go サーバーがHTTPサーバーとして配信します。

## Web UI の実装状況

### 実装済み機能

- **タブベースナビゲーション**: 設置場所とデバイスグループによるタブ分け表示
- **ステータスインジケータ**: デバイスとタブの状態を視覚的に表示
  - デバイスカード: 電源状態（緑/グレー）とエラー状態（赤）のインジケータ
  - タブ: 電源ON（緑）とエラー（赤）のインジケータ（操作可能デバイスのみ対象）
- **プロパティ表示改善**: EPCコードを人間可読な名前で表示
- **インタラクティブ編集**: プロパティ種類に応じたUIコントロール（ドロップダウン、数値入力）
- **条件付きプロパティ表示**: コンパクトモードで文脈に応じたプロパティの表示/非表示
  - 文字列エイリアスベースの条件定義（例: `'auto'`, `'fan'`）
  - Home Air Conditioner: 運転モードに応じて温度・湿度設定を自動で非表示
  - `PROPERTY_VISIBILITY_CONDITIONS` で拡張可能
- **デバイスエイリアス表示**: エイリアス名での分かりやすいデバイス識別
- **ダッシュボードタブ**: 全操作可能デバイスをコンパクト表示
  - 設置場所ごとにグループ化（PCは横並び、モバイルは縦積み）
  - 温度ベースの色分け表示（青=涼、オレンジ=暖、赤=高温）
  - デバイスON時は緑枠で表示
  - ホバーツールチップでデバイス名表示（PC）
- **設置場所設定ダイアログ**: Web UIから設置場所を管理
  - エイリアスの追加・削除
  - ドラッグ&ドロップによる表示順変更
  - 入力バリデーション（最大32文字、禁止文字チェック）
- **デバイス履歴表示**: 各デバイスのプロパティ変更履歴を表示
  - 時刻、プロパティ名、値、発生源（操作/通知）を時系列表示
  - サーバー起動マーカーで再起動前後を区別
  - settableOnlyフィルターで操作可能プロパティのみ/全プロパティを切り替え
- **モダンUIデザイン**: Tech Dashboard風のデザイン
- **レスポンシブデザイン**: モバイル・デスクトップ対応
- **リアルタイム更新**: WebSocketによるプロパティ変更のリアルタイム反映
- **国際化対応**: 日本語・英語対応のプロパティ表示（`get_property_description` API の `lang` パラメータ）

### プロパティ表示の仕組み

- **プライマリプロパティ**: `deviceTypeHelper.ts` の `DEVICE_PRIMARY_PROPERTIES` でデバイス種別ごとに定義
  - デバイスカードの常時表示対象（コンパクト表示時も表示）
  - 操作ステータス（0x80）と設置場所（0x81）は全デバイス共通のエッセンシャルプロパティ
  - Single Function Lighting (0291): Illuminance Level (0xB0) など
- **条件付きプロパティ表示**: `PROPERTY_VISIBILITY_CONDITIONS` で文脈に応じた表示制御
  - 文字列エイリアス優先（例: `'auto'`, `'cooling'`）で条件定義
  - 数値コード（例: `0x41`）は不要、直感的な編集が可能
  - コンパクトモードのみ適用、展開モードは全プロパティ表示
  - 値取得優先順位: string > number > EDT (Base64デコード)
- **PropertyEditor**: プロパティの編集可能性と表示方法を制御
  - aliasありプロパティ: Selectドロップダウンで値選択
  - aliasなしプロパティ: 現在値表示 + 編集ボタン（数値入力など）
  - Set Property Map (EPC 0x9E) による編集可能性判定
- **formatPropertyValue**: プロパティ値の表示フォーマット（単位付き数値、alias名など）

### 技術詳細

- **フレームワーク**: React 19 + TypeScript
- **UIライブラリ**: shadcn/ui (Tailwind CSS ベース)
- **ビルドツール**: Vite
- **テスト**: Vitest
- **主要ファイル**:
  - `src/App.tsx`: メインアプリケーション
  - `src/components/PropertyEditor.tsx`: プロパティ編集コンポーネント
  - `src/components/DeviceStatusIndicators.tsx`: デバイス状態インジケータ
  - `src/components/DeviceHistoryDialog.tsx`: デバイス履歴ダイアログ
  - `src/components/DashboardCard.tsx`: ダッシュボードデバイスカード
  - `src/components/DashboardTabContent.tsx`: ダッシュボードタブコンテンツ
  - `src/components/LocationSettingsDialog.tsx`: 設置場所設定ダイアログ
  - `src/hooks/useWebSocketConnection.ts`: WebSocket接続管理
  - `src/hooks/useECHONET.ts`: ECHONETプロトコル状態管理
  - `src/hooks/useDeviceHistory.ts`: デバイス履歴取得
  - `src/hooks/useAutoReconnect.ts`: 自動再接続機能
  - `src/hooks/useLogNotifications.ts`: ログ通知機能
  - `src/hooks/usePersistedTab.ts`: タブ状態永続化
  - `src/hooks/useCardExpansion.ts`: デバイスカード展開状態管理
  - `src/libs/propertyHelper.ts`: プロパティ名・値変換
  - `src/libs/deviceTypeHelper.ts`: デバイス種別・プライマリプロパティ定義
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
  9. サーバーを ./echonet-list で起動し、ユーザーに動作確認してもらいます。

## Web UI トラブルシューティング

### プロパティが表示されない場合

1. **プライマリプロパティの確認**: `web/src/libs/deviceTypeHelper.ts` の `DEVICE_PRIMARY_PROPERTIES` でデバイスクラスコードが正しく定義されているか
   - ECHONET Lite仕様書でクラスコードを確認（例：Single Function Lighting = 0291）
   - EPCコードが正しい形式（大文字の16進数）で定義されているか

2. **サーバー側プロパティ定義の確認**: Go側の `echonet_lite/prop_*.go` ファイルでプロパティが正しく定義されているか
   - プロパティの型（NumberDesc, StringDesc, aliases）
   - DefaultEPCsに含まれているか

3. **PropertyEditor の表示条件**:
   - 編集可能プロパティ: Set Property Map (EPC 0x9E) に含まれているか
   - alias有りプロパティ: Selectドロップダウンが値を表示
   - alias無しプロパティ: 現在値 + 編集ボタンで表示

### デバッグ手順

1. ブラウザの開発者ツールでWebSocketメッセージを確認
2. `formatPropertyValue` 関数の戻り値を確認
3. デバイスの `properties` オブジェクトに該当EPCが含まれているか確認

### 条件付きプロパティ表示の設定

コンパクトモードで特定の条件下でプロパティを非表示にする機能です。

- **設定場所**: `web/src/libs/deviceTypeHelper.ts` の `PROPERTY_VISIBILITY_CONDITIONS`
- **条件**: `values`（一致時非表示）または `notValues`（不一致時非表示）で指定
- 文字列エイリアス優先、コンパクトモードのみ適用

## 国際化対応の実装

### 概要

- PropertyTable と PropertyDesc に多言語対応機能を実装済み
- WebSocket API (`get_property_description`) で `lang` パラメータをサポート
- 現在サポート言語: 英語（デフォルト）、日本語

### 主要実装ファイル

- `echonet_lite/PropertyDesc.go`: PropertyDesc 構造体の言語対応フィールド
- `echonet_lite/Property.go`: PropertyTable 構造体の言語対応フィールド  
- `protocol/protocol.go`: WebSocket プロトコルの言語パラメータ
- `server/websocket_server_handlers_properties.go`: 言語対応レスポンス処理
- `docs/internationalization.md`: 国際化対応の詳細ガイド

### 設計原則

1. **通信では英語キーを使用**: `set_properties` などの操作では `aliases` の英語キーを使用
2. **表示は翻訳を使用**: UI表示では `aliasTranslations` の値を使用  
3. **後方互換性**: 既存コードは変更なしで動作

### 使用例

```json
{
  "type": "get_property_description", 
  "payload": {
    "classCode": "0291",
    "lang": "ja"
  }
}
```

詳細な実装方法とガイドラインは `docs/internationalization.md` を参照してください。

## デーモンモードの実装

### デーモンモードの概要

- バックグラウンドサービスとして動作するためのデーモンモードを実装済み
- Linux/macOSでsystemdサービスとして運用可能
- プラットフォーム別のデフォルトパスを自動設定

### 主要ファイル

- `config/config.go`: プラットフォーム別デフォルトパス設定
  - `getDefaultPIDFile()`: OS別のPIDファイルパスを返す
  - `getDefaultDaemonLogFile()`: OS別のログファイルパスを返す
- `main.go`: デーモンモード処理
  - PIDファイルの作成・削除
  - ログローテーション対応（SIGHUP）
  - コンソールUI無効化
- `systemd/`: systemd関連ファイル
  - `echonet-list.service`: systemdサービス定義
  - `config.toml.systemd`: systemd用設定サンプル
  - `echonet-list.logrotate`: logrotate設定

### デーモンモード時の動作

1. WebSocketサーバーが必須（コンソールUIが使えないため）
2. PIDファイルを作成（デフォルト: `/var/run/echonet-list.pid`）
3. ログファイルパスを自動切り替え（デフォルト: `/var/log/echonet-list.log`）
4. SIGHUPシグナルでログローテーション実行
5. SIGTERM/SIGINTで正常終了

### デバッグ時の注意

- デーモンモードではコンソール出力が無いため、ログファイルを確認
- systemdの場合は `journalctl -u echonet-list -f` でもログ確認可能
- 権限エラーの場合は、書き込み可能なパスを `-pidfile` で指定

### 実サーバーのログ参照

実サーバーのログを参照するには、環境変数 `ECHONET_SERVER_HOST` で定義されたホストに SSH 接続して sudo でログファイルにアクセスします。

```bash
ssh $ECHONET_SERVER_HOST sudo cat /var/log/echonet-list.log
ssh $ECHONET_SERVER_HOST sudo tail -f /var/log/echonet-list.log
```

- 環境変数 `ECHONET_SERVER_HOST` は `.claude/settings.local.json` で定義されています
- ログファイルは root 権限が必要なため `sudo` を使用します

## Console UI のコマンド

### debugoffline - デバイスのオフライン状態設定

テスト用にデバイスのオフライン状態を手動設定するデバッグコマンドです。

```
debugoffline <device_specifier> [on|off]
```

- `<device_specifier>`: デバイス指定子（IP+クラスコード:インスタンス、またはエイリアス）
- `[on|off]`: オフライン状態の設定（省略時はトグル）

### location - 設置場所のエイリアスと表示順の管理

Web UIでの設置場所タブの表示名や順序を管理します。

```
location list                       # 設置場所一覧
location alias add #alias rawValue  # エイリアス追加（#で始まる名前）
location alias delete #alias        # エイリアス削除
location order reset                # 表示順リセット
```

※表示順の変更はWeb UIで編集してください
