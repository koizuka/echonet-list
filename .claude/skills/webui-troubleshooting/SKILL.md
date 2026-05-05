---
name: webui-troubleshooting
description: Diagnoses why an ECHONET device property does not appear or behaves incorrectly in the Web UI. Use when working in web/ and a property is missing from a device card, the editor shows the wrong control, formatPropertyValue output looks wrong, or a newly added device class shows nothing useful.
---

# Web UI トラブルシューティング

`web/` の Web UI でプロパティ表示・編集系の不具合を観測したときに使う。

## プロパティが表示されない場合

1. **プライマリプロパティの確認**: `web/src/libs/deviceTypeHelper.ts` の `DEVICE_PRIMARY_PROPERTIES` でデバイスクラスコードが正しく定義されているか
   - ECHONET Lite 仕様書でクラスコードを確認（例: Single Function Lighting = `0291`）
   - EPC コードが大文字 16 進数の形式で定義されているか

2. **サーバー側プロパティ定義の確認**: Go 側の `echonet_lite/prop_*.go` でプロパティが定義されているか
   - 型（`NumberDesc`, `StringDesc`, aliases）が適切か
   - `DefaultEPCs` に含まれているか

3. **PropertyEditor の表示条件**:
   - 編集可能プロパティ: Set Property Map (EPC `0x9E`) に含まれているか
   - alias ありプロパティ: Select ドロップダウンが値を表示
   - alias なしプロパティ: 現在値 + 編集ボタンで表示

## プロパティ表示の仕組み

- **プライマリプロパティ**: `deviceTypeHelper.ts` の `DEVICE_PRIMARY_PROPERTIES` でデバイス種別ごとに定義
  - デバイスカードの常時表示対象（コンパクト表示時も表示）
  - 操作ステータス (`0x80`) と設置場所 (`0x81`) は全デバイス共通
- **値取得優先順位**: `string` > `number` > EDT (Base64 デコード)
- **PropertyEditor**: `web/src/components/PropertyEditor.tsx`
- **formatPropertyValue**: `web/src/libs/propertyHelper.ts` で表示フォーマット（単位付き数値、alias 名など）

## デバッグ手順

1. ブラウザの開発者ツールで WebSocket メッセージを確認
2. `formatPropertyValue` 関数の戻り値を確認
3. デバイスの `properties` オブジェクトに該当 EPC が含まれているか確認

関連: 条件付きで非表示になっている疑いがあるなら `property-visibility` スキルを参照。
