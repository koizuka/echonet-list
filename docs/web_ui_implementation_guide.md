# ECHONET Lite Web UI 実装ガイド

## 概要

このガイドでは、ECHONET Lite デバイス管理のための Web UI の実装について説明します。React + TypeScript + shadcn/ui を使用した現在の実装内容と、主要な機能について詳しく解説します。

## アーキテクチャ

### 技術スタック

- **フロントエンド**: React 19, TypeScript, Vite
- **UI ライブラリ**: shadcn/ui (Tailwind CSS ベース)
- **状態管理**: React Hooks (useState, useEffect)
- **通信**: WebSocket (ECHONET Lite WebSocket プロトコル)
- **テスト**: Vitest

### ディレクトリ構成

```
web/
├── src/
│   ├── components/        # UI コンポーネント
│   │   ├── PropertyEditor.tsx
│   │   └── ui/           # shadcn/ui コンポーネント
│   ├── hooks/            # React Hooks
│   │   ├── useECHONET.ts
│   │   ├── usePropertyDescriptions.ts
│   │   └── useWebSocketConnection.ts
│   ├── libs/             # ユーティリティ関数
│   │   ├── propertyHelper.ts
│   │   ├── locationHelper.ts
│   │   └── deviceIdHelper.ts
│   └── App.tsx           # メインアプリケーション
├── bundle/               # ビルド結果
└── public/               # 静的ファイル
```

## 主要機能

### 1. タブベースの UI ナビゲーション

#### 実装概要

デバイスを「設置場所」と「デバイスグループ」でタブ分けして表示する機能を実装しています。

#### 設置場所の抽出と表示

`locationHelper.ts` では設置場所の抽出と表示名の翻訳を以下のように処理：

##### 抽出ロジック（`extractLocationFromDevice()`）

以下の優先順位でロケーションを決定：

1. **EPC 0x81 (Installation Location)** プロパティから抽出（生の値）
2. **デバイスエイリアス**から抽出（`"リビング - エアコン"` → `"リビング"`）
3. **デバイス名**から抽出
4. フォールバック: `"Unknown"`

##### 表示名の翻訳（`getLocationDisplayName()`）

タブ表示などで使用される表示名は、サーバー側の翻訳を利用：

```typescript
export function getLocationDisplayName(
  locationId: string,
  devices: Record<string, Device>,
  propertyDescriptions: Record<string, PropertyDescriptionData>,
  lang?: string
): string {
  // そのロケーションIDを持つデバイスを検索
  const devicesInLocation = Object.values(devices).filter(device => {
    const rawLocation = extractRawLocationFromDevice(device);
    return rawLocation === locationId;
  });
  
  if (devicesInLocation.length > 0) {
    const device = devicesInLocation[0];
    const classCode = device.eoj.split(':')[0];
    
    // サーバー側のプロパティ記述子から翻訳を取得
    const descriptor = getPropertyDescriptor('81', propertyDescriptions, classCode, lang);
    const translatedValue = formatPropertyValue(installationLocationProperty, descriptor, lang);
    
    return translatedValue; // 例: "living" → "リビング"
  }
  
  return locationId.charAt(0).toUpperCase() + locationId.slice(1);
}
```

これにより、タブ名やプロパティ表示で一貫して日本語翻訳が使用されます。

#### グループ管理

- グループ名は `@` プレフィックスで識別（例: `@1F床暖房`）
- デバイス識別子の部分マッチングで柔軟なグループ化をサポート

```typescript
// locationHelper.ts の例
export function getAllTabs(
  devices: Record<string, Device>,
  aliases: DeviceAlias,
  groups: DeviceGroup
): string[] {
  const locationTabs = getAllLocations(devices, aliases);
  const groupTabs = Object.keys(groups)
    .filter(groupName => groupName.startsWith('@'))
    .sort();
  
  return [...locationTabs, ...groupTabs];
}
```

### 2. プロパティ表示の改善

#### 人間可読なプロパティ名

`propertyHelper.ts` で EPC コードを人間可読な名前に変換：

```typescript
export function getPropertyName(
  epc: string, 
  propertyDescriptions: Record<string, PropertyDescriptionData>,
  classCode?: string
): string {
  if (classCode && propertyDescriptions[classCode]?.properties[epc]?.description) {
    return propertyDescriptions[classCode].properties[epc].description;
  }
  return `EPC ${epc.toUpperCase()}`;
}
```

#### プロパティ値のフォーマット

エイリアス、数値、文字列を適切に表示し、多言語対応を提供：

```typescript
export function formatPropertyValue(
  value: { EDT?: string; string?: string; number?: number },
  descriptor?: PropertyDescriptor,
  lang?: string
): string {
  const currentLang = lang || getCurrentLocale();

  // 文字列値がある場合は、翻訳を試行
  if (value.string) {
    // aliasTranslationsが利用可能で、英語以外の場合
    if (descriptor?.aliasTranslations && currentLang !== 'en') {
      const translation = descriptor.aliasTranslations[value.string];
      if (translation) {
        return translation; // 例: "living" → "リビング"
      }
    }
    return value.string; // 翻訳がない場合は元の値
  }

  // 数値 + 単位
  if (value.number !== undefined) {
    const unit = descriptor?.numberDesc?.unit || '';
    return `${value.number}${unit}`;
  }

  // EDTから逆引きでエイリアス名を取得
  if (value.EDT && descriptor?.aliases) {
    try {
      const edtBytes = atob(value.EDT);
      for (const [aliasName, aliasEDT] of Object.entries(descriptor.aliases)) {
        if (atob(aliasEDT) === edtBytes) {
          // 翻訳があれば使用
          if (descriptor.aliasTranslations && currentLang !== 'en') {
            const translation = descriptor.aliasTranslations[aliasName];
            if (translation) return translation;
          }
          return aliasName;
        }
      }
    } catch {
      // デコードエラーは無視
    }
  }

  return 'Raw data';
}
```

この関数により、設置場所やその他のプロパティで一貫してサーバー側の翻訳が使用されます。

### 3. インタラクティブなプロパティエディタ

#### PropertyEditor コンポーネント

プロパティの種類に応じて適切な UI コントロールを動的に生成し、多言語表示に対応：

```typescript
// PropertyEditor.tsx の主要ロジック
const renderEditor = () => {
  // エイリアスがある場合: ドロップダウンメニュー（翻訳対応）
  if (descriptor?.aliases && Object.keys(descriptor.aliases).length > 0) {
    return (
      <PropertySelectControl
        value={currentValue.string || ''}
        aliases={descriptor.aliases}
        aliasTranslations={descriptor.aliasTranslations} // 翻訳データを渡す
        onChange={handleAliasSelect}
        disabled={isLoading || !isConnectionActive}
      />
    );
  }
  
  // 数値プロパティの場合: 入力フィールド
  if (descriptor?.numberDesc) {
    return <NumberInputEditor />;
  }
  
  // 文字列プロパティの場合: テキスト入力
  if (descriptor?.stringDesc) {
    return <StringInputEditor />;
  }
  
  return null; // 編集不可
};
```

`PropertySelectControl` では、`aliasTranslations` を使用してドロップダウンのオプションを翻訳表示：

```typescript
// PropertySelectControl.tsx
const getDisplayText = (aliasName: string) => {
  // 翻訳が利用可能で英語以外の場合
  if (aliasTranslations && currentLang !== 'en') {
    const translation = aliasTranslations[aliasName];
    if (translation) {
      return translation; // 例: "living" → "リビング"
    }
  }
  
  return aliasName; // 翻訳がない場合は英語キー
};
```

#### サポートされる編集タイプ

1. **エイリアス選択**: ドロップダウンメニュー（例: ON/OFF, 運転モード）
2. **数値入力**: 範囲制限付き入力フィールド（例: 温度設定）
3. **文字列入力**: 最大長制限付きテキストフィールド

### 4. デバイス識別子の正確な処理

#### 課題と解決

Go バックエンドとの識別子形式の違いを解決：

**問題**:

- Go側: `027B04:000005:00112233445566778899AABBCCDD`
- TypeScript側: `027B04:FE000500112233445566778899AABBCCDD`

**解決策**: `deviceIdHelper.ts` でパース処理を実装

```typescript
function parseIdentificationNumber(hexString: string): string {
  // FE + 6桁manufacturer + 26桁unique → manufacturer:unique 形式に変換
  if (hexString.length !== 34 || !hexString.startsWith('FE')) {
    return hexString;
  }
  
  const manufacturerCode = hexString.substring(2, 8);
  const uniqueIdentifier = hexString.substring(8);
  
  return `${manufacturerCode}:${uniqueIdentifier}`;
}
```

#### EOJベースの部分マッチング

グループデバイスマッチングで、EOJ部分での部分一致をサポート：

```typescript
// locationHelper.ts
const deviceEOJPart = deviceIdentifier.split(':')[0]; // "027B04"
return groupDeviceIds.some(groupId => groupId.startsWith(deviceEOJPart + ':'));
```

### 5. レスポンシブデザイン

#### タブナビゲーション

```css
/* Tailwind CSS クラス例 */
.tab-list {
  @apply w-max min-w-full h-auto p-1 bg-muted flex flex-nowrap justify-start gap-1 sm:flex-wrap sm:w-full;
}

.tab-trigger {
  @apply px-3 py-2 text-sm whitespace-nowrap flex-shrink-0;
}
```

- **モバイル**: 横スクロール、タブ名の短縮表示
- **デスクトップ**: フレックスラップ、完全なタブ名表示

## 実装パターン

### カスタムフックの使用

```typescript
// メインアプリケーションでの使用例
function App() {
  const wsUrl = import.meta.env.DEV 
    ? 'wss://localhost:8080/ws'
    : 'wss://localhost:8080/ws';
  
  const echonet = usePropertyDescriptions(wsUrl);
  
  const handlePropertyChange = async (target: string, epc: string, value: PropertyValue) => {
    try {
      await echonet.setDeviceProperties(target, { [epc]: value });
    } catch (error) {
      console.error('Failed to change property:', error);
    }
  };
  
  // UI レンダリング...
}
```

### エラーハンドリング

- WebSocket 接続エラーの表示
- プロパティ変更失敗時のユーザーフィードバック
- デバイス探索失敗時の適切な状態表示

### パフォーマンス最適化

- プロパティ詳細情報の自動キャッシュ
- 不要な再レンダリングの防止
- 効率的なデバイス状態更新

## 今後の拡張可能性

### 1. 追加可能な機能

- **デバイス操作履歴**: 操作ログの表示
- **スケジュール機能**: 時間指定での自動操作
- **シーン機能**: 複数デバイスの一括制御
- **通知機能**: デバイス状態変化の通知

### 2. UI/UX 改善

- **ダッシュボード**: デバイス状態のサマリー表示
- **カスタマイズ**: ユーザー定義のレイアウト
- **テーマ**: ダーク/ライトモード切り替え
- **多言語対応**: 国際化対応

### 3. 技術的拡張

- **PWA対応**: オフライン機能とプッシュ通知
- **状態管理ライブラリ**: Redux Toolkit や Zustand の導入
- **リアルタイム可視化**: チャートやグラフでの状態表示

## 開発ワークフロー

### ビルドコマンド

```bash
# 開発サーバー起動
npm run dev

# 本番ビルド
npm run build

# テスト実行
npm run test

# リンター実行
npm run lint
```

### デバッグ

- ブラウザ DevTools での WebSocket メッセージ確認
- React DevTools でのコンポーネント状態確認
- コンソールログでのデバイス識別子マッチングデバッグ

## 注意事項

### セキュリティ

- 本実装は家庭LAN内での使用を想定
- 認証機能は現在未実装
- HTTPS/WSS での運用を推奨

### 互換性

- Chrome (PC), Safari (iPhone), Chrome (Android) での動作確認済み
- WebSocket 接続が必要なため、ネットワーク環境に依存

### メンテナンス

- ECHONET Lite 規格更新時のプロパティ定義更新
- Go バックエンドとの API 仕様同期
- UI ライブラリ（shadcn/ui）の定期更新
