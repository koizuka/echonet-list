# Active Context

このファイルでは、現在進行中の作業と最近の変更点をまとめています。基礎ドキュメントは [projectbrief.md](./projectbrief.md) と [systemPatterns.md](./systemPatterns.md) を参照してください。

## Current Work

- プロパティ転送フォーマットを `"EPC": "BASE64"` 形式から  
  `"EPC": { "EDT": "BASE64", "string": "xxx" }` 形式へ変更  
- サーバー→クライアント、クライアント→サーバー双方向の JSON スキーマおよび各ハンドラを修正  
- Go コード（`protocol`, `server`, `client` パッケージ）とテスト、ドキュメント（`docs/websocket_client_protocol.md`）を一貫して更新・補完  
- テストエラー対応や変数スコープ修正など、複数の段階的なリファクタリングを実施  

## Key Technical Concepts

- 新フォーマット用の `PropertyData` 構造体導入  
- `DeviceToProtocol` / `DeviceFromProtocol` の双方向変換ロジック修正  
- EDT の Base64 と文字列表現を保持するためのフィールド追加 (`EDTToString`)  
- Go のテスト駆動開発（テストコードに String フィールドを追加）  
- ドキュメント中のサンプル JSON を新フォーマットに合わせて置換  

## Relevant Files and Code

- **protocol/protocol.go**  
  - `PropertyData` 構造体追加  
  - `DeviceToProtocol` / `DeviceFromProtocol` 修正  
- **protocol/protocol_test.go**  
  - テスト期待値に `string` フィールドを追加  
- **server/websocket_server_handlers_properties.go**  
  - プロパティ送信・応答ハンドラを新フォーマット対応に更新  
- **client/websocket_notifications.go**  
  - `handlePropertyChanged` 内で `EDT`／`string` 両対応のパース処理を実装  
- **docs/websocket_client_protocol.md**  
  - 全サンプル JSON を新フォーマットへ更新  

## Problem Solving

- `TestDeviceToProtocol` の失敗から文字列表現生成ロジックを見直し  
- 変数スコープ（`err`, `ipAndEOJ`, `epc`）の衝突を修正  
- ドキュメントの複数箇所を SEARCH／REPLACE ブロックで段階的に置換  

## Pending Tasks and Next Steps

1. ドキュメント内の他メッセージ例（`property_changed`, `set_properties` リクエストなど）を新フォーマット化  
2. クライアント実装ガイド／コード例にも String フィールド対応を反映  
3. 追加テストケース検討（文字列のみ指定時、Base64 のみ指定時の挙動）  
4. UI／サンプルクライアントへのフォーマットガイド反映  
5. メモリーバンクや README の進捗更新
