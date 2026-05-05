---
name: echonet-i18n
description: Implements multi-language support for ECHONET property descriptions using PropertyTable / PropertyDesc and the get_property_description WebSocket API. Use when adding a new property class with translated aliases, tweaking lang-parameter handling, or wiring a new language into the server side.
---

# ECHONET 国際化対応ガイド

PropertyTable / PropertyDesc に多言語対応機能を実装済み。WebSocket API `get_property_description` が `lang` パラメータをサポートする。サポート言語: 英語（デフォルト）、日本語。

## 設計原則

1. **通信では英語キーを使用**: `set_properties` などの操作では `aliases` の英語キーを使用
2. **表示は翻訳を使用**: UI 表示では `aliasTranslations` の値を使用
3. **後方互換性**: 既存コードは変更なしで動作

## 主要実装ファイル

- `echonet_lite/PropertyDesc.go`: `PropertyDesc` の言語対応フィールド
- `echonet_lite/Property.go`: `PropertyTable` の言語対応フィールド
- `protocol/protocol.go`: WebSocket プロトコルの言語パラメータ
- `server/websocket_server_handlers_properties.go`: 言語対応レスポンス処理

## 使用例

```json
{
  "type": "get_property_description",
  "payload": {
    "classCode": "0291",
    "lang": "ja"
  }
}
```

## 詳細

実装ガイドラインの完全版は `docs/internationalization.md` を参照。新言語追加・新プロパティの翻訳追加もこちらに従う。
