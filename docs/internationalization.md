# 国際化（i18n）対応ガイド

## 概要

ECHONET Lite WebSocket サーバーは、多言語対応機能を提供しています。この機能により、プロパティの説明文やエイリアスの表示を、クライアントの言語設定に応じて日本語や英語で表示することができます。

## サポート言語

- **英語 (`en`)**: デフォルト言語
- **日本語 (`ja`)**: サポート言語

## 対応API

### get_property_description

`get_property_description` APIでは、`lang` パラメータを使用して言語を指定できます。

```json
{
  "type": "get_property_description",
  "payload": {
    "classCode": "0130",
    "lang": "ja"
  },
  "requestId": "req-001"
}
```

#### 応答例

**英語版 (lang="en" または省略時):**

```json
{
  "type": "command_result",
  "payload": {
    "success": true,
    "data": {
      "classCode": "0130",
      "properties": {
        "80": {
          "description": "Operation status",
          "aliases": { "on": "MzA=", "off": "MzE=" }
        },
        "B0": {
          "description": "Illuminance level",
          "numberDesc": { "min": 0, "max": 100, "unit": "%" }
        }
      }
    }
  },
  "requestId": "req-001"
}
```

**日本語版 (lang="ja"):**

```json
{
  "type": "command_result",
  "payload": {
    "success": true,
    "data": {
      "classCode": "0130",
      "properties": {
        "80": {
          "description": "動作状態",
          "aliases": { "on": "MzA=", "off": "MzE=" },
          "aliasTranslations": { "on": "オン", "off": "オフ" }
        },
        "B0": {
          "description": "照度レベル",
          "numberDesc": { "min": 0, "max": 100, "unit": "%" }
        }
      }
    }
  },
  "requestId": "req-001"
}
```

## フィールド説明

### 国際化対応フィールド

- **`description`**: プロパティの説明文（指定した言語で表示）
- **`aliases`**: プロパティのエイリアス値（常に英語キー、通信で使用）
- **`aliasTranslations`**: エイリアスの翻訳テーブル（指定した言語での表示名）

### 重要な設計原則

1. **通信では英語キーを使用**: `set_properties` などの操作では、`aliases` の英語キーを使用
2. **表示は翻訳を使用**: UI表示では `aliasTranslations` の値を使用
3. **後方互換性**: 既存の英語のみ対応クライアントは変更なしで動作

## 実装例

### JavaScript/TypeScript での実装

```typescript
interface PropertyDescResponse {
  classCode: string;
  properties: {
    [epc: string]: {
      description: string;
      aliases?: { [key: string]: string };
      aliasTranslations?: { [key: string]: string };
      numberDesc?: NumberDesc;
      stringDesc?: StringDesc;
      stringSettable?: boolean;
    };
  };
}

// プロパティ説明を取得（言語指定）
async function getPropertyDescription(
  classCode: string, 
  lang: string = "en"
): Promise<PropertyDescResponse> {
  const response = await sendRequest("get_property_description", {
    classCode,
    lang
  });
  return response;
}

// プロパティ値を表示用文字列に変換
function formatPropertyValue(
  epcData: any, 
  value: string, 
  lang: string = "en"
): string {
  // aliases から該当する英語キーを見つける
  let englishKey: string | null = null;
  if (epcData.aliases) {
    for (const [alias, encodedValue] of Object.entries(epcData.aliases)) {
      if (encodedValue === value) {
        englishKey = alias;
        break;
      }
    }
  }
  
  // 翻訳が利用可能な場合は使用
  if (englishKey && epcData.aliasTranslations && epcData.aliasTranslations[englishKey]) {
    return epcData.aliasTranslations[englishKey];
  }
  
  // 英語キーが見つかった場合はそれを返す
  if (englishKey) {
    return englishKey;
  }
  
  return value;
}

// プロパティ設定用のキーを取得（常に英語）
function getPropertyKey(epcData: any, displayValue: string): string | null {
  // 翻訳から英語キーを逆引き
  if (epcData.aliasTranslations) {
    for (const [englishKey, translatedValue] of Object.entries(epcData.aliasTranslations)) {
      if (translatedValue === displayValue) {
        return englishKey;
      }
    }
  }
  
  // 直接的な英語キーマッチ
  if (epcData.aliases && epcData.aliases[displayValue]) {
    return displayValue;
  }
  
  return null;
}
```

### React での実装例

```tsx
import React, { useState, useEffect } from 'react';

interface PropertyControlProps {
  classCode: string;
  epc: string;
  currentValue: string;
  lang: string;
  onValueChange: (value: string) => void;
}

const PropertyControl: React.FC<PropertyControlProps> = ({
  classCode,
  epc,
  currentValue,
  lang,
  onValueChange
}) => {
  const [propertyDesc, setPropertyDesc] = useState<any>(null);

  useEffect(() => {
    getPropertyDescription(classCode, lang)
      .then(desc => setPropertyDesc(desc.properties[epc]));
  }, [classCode, epc, lang]);

  if (!propertyDesc) return <div>Loading...</div>;

  const handleChange = (displayValue: string) => {
    // 表示値から英語キーに変換
    const englishKey = getPropertyKey(propertyDesc, displayValue);
    if (englishKey) {
      onValueChange(englishKey);
    }
  };

  // 現在の値（Base64エンコードされたEDT）から表示値を取得
  const currentEncodedValue = propertyDesc.aliases?.[currentValue] || currentValue;
  const displayValue = formatPropertyValue(propertyDesc, currentEncodedValue, lang);

  return (
    <div>
      <label>{propertyDesc.description}</label>
      {propertyDesc.aliases && (
        <select 
          value={displayValue} 
          onChange={e => handleChange(e.target.value)}
        >
          {Object.entries(propertyDesc.aliasTranslations || propertyDesc.aliases)
            .map(([key, translatedValue]) => (
              <option key={key} value={translatedValue}>
                {translatedValue}
              </option>
            ))}
        </select>
      )}
    </div>
  );
};
```

## サーバー側の言語データ定義

### PropertyDesc構造体

```go
type PropertyDesc struct {
    Name              string                            // 英語の説明
    NameMap           map[string]string                 // 言語別の説明
    Aliases           map[string][]byte                 // 英語エイリアス
    AliasTranslations map[string]map[string]string      // 言語別エイリアス翻訳
    Decoder           PropertyDecoder                   // デコーダ
}
```

### 定義例

```go
EPC_SF_Panasonic_OperationStatus: {
    Name: "Panasonic Operation Status",
    NameMap: map[string]string{
        "ja": "パナソニック動作状態",
    },
    Aliases: map[string][]byte{
        "on":  {0x30},
        "off": {0x31},
    },
    AliasTranslations: map[string]map[string]string{
        "ja": {
            "on":  "オン",
            "off": "オフ",
        },
    },
    Decoder: nil,
}
```

## ベストプラクティス

1. **デフォルト言語の提供**: 常に英語のフォールバックを提供
2. **一貫性のあるキー**: 通信では常に英語キーを使用
3. **UI表示の分離**: 表示ロジックと通信ロジックを分離
4. **言語設定の保存**: ユーザーの言語設定をローカルに保存
5. **段階的な翻訳**: 新しいプロパティは英語のみから開始し、段階的に翻訳を追加

## トラブルシューティング

### 翻訳が表示されない

1. `lang` パラメータが正しく指定されているか確認
2. サーバー側で該当言語の翻訳が定義されているか確認
3. フォールバック処理が正しく実装されているか確認

### プロパティ設定が失敗する

1. 英語キーを使用しているか確認
2. `aliasTranslations` ではなく `aliases` のキーを使用しているか確認
3. キーの大文字小文字が正確か確認

### パフォーマンス考慮事項

1. 言語データのキャッシュ
2. 不要な API 呼び出しの削減
3. 翻訳データの遅延読み込み
