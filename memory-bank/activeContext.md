# Active Context

このファイルでは、現在進行中の作業と最近の変更点をまとめています。基礎ドキュメントは [projectbrief.md](./projectbrief.md) と [systemPatterns.md](./systemPatterns.md) を参照してください。

## Current Work

- プロパティ転送フォーマットを `"EPC": "BASE64"` 形式から
  `"EPC": { "EDT": "BASE64", "string": "xxx", "number": nnn }` 形式へ変更
- `protocol.PropertyData` に `Number` フィールドを追加
- サーバー→クライアント、クライアント→サーバー双方向の JSON スキーマおよび各ハンドラを修正
- `server/websocket_server_handlers_properties.go` で `set_properties` 時に `Number` フィールドを処理するように修正
- `server/websocket_server_handlers_set_properties_test.go` に `Number` フィールドを使用するテストケースと、値が未指定の場合のエラーテストケースを追加
- Go コード（`protocol`, `server`, `client` パッケージ）とテストを一貫して更新・補完
- テストエラー対応や変数スコープ修正など、複数の段階的なリファクタリングを実施
- ドキュメント（`docs/websocket_client_protocol.md`）を新プロパティフォーマットに合わせて更新 (`property_changed` 通知例、クライアント実装例の修正)

## Key Technical Concepts

- 新フォーマット用の `PropertyData` 構造体導入 (`EDT`, `String`, `Number` フィールド)
- `DeviceToProtocol` / `DeviceFromProtocol` の双方向変換ロジック修正
- `set_properties` ハンドラでの `Number` フィールド処理ロジック追加 (数値対応プロパティのみ)
- EDT の Base64 と文字列表現、数値表現を保持するためのフィールド追加 (`EDTToString`, `ToInt`)
- Go のテスト駆動開発（テストコードに `String`, `Number` フィールド、エラーケースを追加）
- ドキュメント中のサンプル JSON を新フォーマットに合わせて置換

## Relevant Files and Code

- **protocol/protocol.go**  
  - `PropertyData` 構造体追加  
  - `DeviceToProtocol` / `DeviceFromProtocol` 修正  
- **protocol/protocol_test.go**
  - テスト期待値に `string`, `number` フィールドを追加
- **server/websocket_server_handlers_properties.go**
  - プロパティ送信・応答ハンドラを新フォーマット対応に更新 (`MakePropertyData`, `handleSetPropertiesFromClient`)
- **server/websocket_server_handlers_set_properties_test.go**
  - `Number` フィールド使用ケース、値未指定エラーケースを追加
- **client/websocket_notifications.go**
  - `handlePropertyChanged` 内で `EDT`／`string`／`number` 両対応のパース処理を実装
- **docs/websocket_client_protocol.md**
  - 全サンプル JSON を新フォーマットへ更新
  - `property_changed` 通知例に `number` フィールドを含むケースを追加
  - クライアント実装例（TypeScript）の `handleNotification` と `setDeviceProperties` を新フォーマット対応に修正

## Problem Solving

- `TestDeviceToProtocol` の失敗から文字列表現生成ロジックを見直し  
- 変数スコープ（`err`, `ipAndEOJ`, `epc`）の衝突を修正  
- ドキュメントの複数箇所を SEARCH／REPLACE ブロックで段階的に置換  

## Pending Tasks and Next Steps

1.  UI／サンプルクライアントへのフォーマットガイド反映 (残タスク)
2.  メモリーバンク (`activeContext.md`, `progress.md`) や README の進捗更新 (この更新作業)
