---
name: property-visibility
description: Adds or tunes conditional property visibility in compact-mode device cards via PROPERTY_VISIBILITY_CONDITIONS. Use when a property should hide based on another property's value (for example, hiding temperature setpoints when an air conditioner runs in fan mode), or when adding a new device class that needs context-sensitive compact display.
---

# 条件付きプロパティ表示

コンパクトモードの device card で、他プロパティの値に応じてプロパティを表示/非表示にする機能。展開モードでは常に全プロパティ表示。

## 設定場所

- `web/src/libs/deviceTypeHelper.ts` の `PROPERTY_VISIBILITY_CONDITIONS`

## 条件指定

- `values`: 一致時に非表示
- `notValues`: 不一致時に非表示
- 文字列エイリアス優先（例: `'auto'`, `'cooling'`, `'fan'`）。数値コード（`0x41` など）を直接書く必要はない

## 実例

Home Air Conditioner: 運転モードに応じて温度・湿度設定を自動で非表示にしている。新デバイスクラスへの追加もこのテーブルにエントリを追加するだけ。

## 注意

- コンパクトモードのみ適用
- 表示判定の値取得優先順位は string > number > EDT (Base64 デコード)
- 関連スキル: プロパティ自体が出てこない場合は `webui-troubleshooting` を先に確認
