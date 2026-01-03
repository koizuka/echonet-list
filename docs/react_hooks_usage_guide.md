# ECHONET Lite React Hooks 使用ガイド

## 概要

このガイドでは、ECHONET Lite デバイスとの WebSocket 通信を行うために開発された React Hooks の使用方法について説明します。

## 提供されるフック

### 1. `useECHONET` - メインフック

ECHONET Lite デバイスの管理を行うメインのフックです。WebSocket 接続、デバイス状態管理、操作を統合的に提供します。

### 2. `useWebSocketConnection` - 低レベル WebSocket フック

WebSocket 接続の管理と再接続処理を提供します。通常は `useECHONET` を使用することを推奨しますが、カスタムの WebSocket 処理が必要な場合に直接使用できます。

## 基本的な使用方法

### セットアップ

```tsx
import React from 'react';
import { useECHONET } from './hooks/useECHONET';

function App() {
  const echonet = useECHONET('wss://localhost:8080/ws');

  // 接続状態の確認
  if (echonet.connectionState === 'connecting') {
    return <div>接続中...</div>;
  }

  if (echonet.connectionState === 'error') {
    return <div>接続エラーが発生しました</div>;
  }

  return (
    <div>
      <DeviceList echonet={echonet} />
    </div>
  );
}
```

### 接続先URL

接続先URLの形式：
- 推奨: `wss://hostname:port/ws` (HTTPS/WSS)
- `ws://` はHTTPで提供するローカル開発時のみ利用可能（モバイルは非対応のことがあります）

## デバイス操作

### デバイス一覧の取得と表示

```tsx
function DeviceList({ echonet }: { echonet: ECHONETHook }) {
  const { devices, aliases, groups } = echonet;

  return (
    <div>
      <h2>デバイス一覧</h2>
      {Object.entries(devices).map(([deviceKey, device]) => (
        <DeviceCard key={deviceKey} device={device} echonet={echonet} />
      ))}
    </div>
  );
}
```

### プロパティの取得と表示

```tsx
function DeviceCard({ device, echonet }: { device: Device; echonet: ECHONETHook }) {
  const [loading, setLoading] = useState(false);

  const handleRefresh = async () => {
    setLoading(true);
    try {
      await echonet.updateDeviceProperties([`${device.ip} ${device.eoj}`]);
    } catch (error) {
      console.error('プロパティ更新エラー:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="device-card">
      <h3>{device.name}</h3>
      <p>IP: {device.ip}, EOJ: {device.eoj}</p>
      
      {/* プロパティ表示 */}
      <div>
        {Object.entries(device.properties).map(([epc, value]) => (
          <div key={epc}>
            <strong>EPC {epc}:</strong> 
            {value.string || value.number?.toString() || value.EDT}
          </div>
        ))}
      </div>
      
      <button onClick={handleRefresh} disabled={loading}>
        {loading ? '更新中...' : 'プロパティ更新'}
      </button>
    </div>
  );
}
```

### プロパティの設定

```tsx
function DeviceControl({ device, echonet }: { device: Device; echonet: ECHONETHook }) {
  const [temperature, setTemperature] = useState(25);

  const handleSetProperty = async () => {
    try {
      await echonet.setDeviceProperties(`${device.ip} ${device.eoj}`, {
        'B3': { number: temperature }, // 温度設定
        '80': { string: 'on' },        // 電源ON
      });
    } catch (error) {
      console.error('プロパティ設定エラー:', error);
    }
  };

  return (
    <div>
      <input
        type="number"
        value={temperature}
        onChange={(e) => setTemperature(parseInt(e.target.value))}
        min="16"
        max="30"
      />
      <button onClick={handleSetProperty}>設定</button>
    </div>
  );
}
```

### プロパティ詳細情報の活用

```tsx
function SmartDeviceControl({ device, echonet }: { device: Device; echonet: ECHONETHook }) {
  const [propertyDesc, setPropertyDesc] = useState<PropertyDescriptionData | null>(null);

  useEffect(() => {
    // デバイスのクラスコードからプロパティ詳細を取得
    const classCode = device.eoj.split(':')[0]; // EOJ から ClassCode を抽出
    
    echonet.getPropertyDescription(classCode)
      .then(setPropertyDesc)
      .catch(console.error);
  }, [device.eoj, echonet]);

  const renderPropertyControl = (epc: string, value: PropertyValue) => {
    const propInfo = propertyDesc?.properties[epc];
    if (!propInfo) return null;

    // エイリアスがある場合（選択肢）
    if (propInfo.aliases) {
      return (
        <select
          value={value.string || ''}
          onChange={(e) => handlePropertyChange(epc, { string: e.target.value })}
        >
          {Object.keys(propInfo.aliases).map(alias => (
            <option key={alias} value={alias}>{alias}</option>
          ))}
        </select>
      );
    }

    // 数値の場合（スライダーまたは数値入力）
    if (propInfo.numberDesc) {
      const { min, max, unit } = propInfo.numberDesc;
      return (
        <div>
          <input
            type="range"
            min={min}
            max={max}
            value={value.number || min}
            onChange={(e) => handlePropertyChange(epc, { number: parseInt(e.target.value) })}
          />
          <span>{value.number}{unit}</span>
        </div>
      );
    }

    // 文字列の場合
    if (propInfo.stringDesc) {
      return (
        <input
          type="text"
          value={value.string || ''}
          maxLength={propInfo.stringDesc.maxEDTLen}
          onChange={(e) => handlePropertyChange(epc, { string: e.target.value })}
        />
      );
    }

    return <span>{value.string || value.number || value.EDT}</span>;
  };

  const handlePropertyChange = async (epc: string, newValue: PropertyValue) => {
    try {
      await echonet.setDeviceProperties(`${device.ip} ${device.eoj}`, {
        [epc]: newValue,
      });
    } catch (error) {
      console.error('プロパティ変更エラー:', error);
    }
  };

  return (
    <div>
      {Object.entries(device.properties).map(([epc, value]) => (
        <div key={epc}>
          <label>{propertyDesc?.properties[epc]?.description || `EPC ${epc}`}:</label>
          {renderPropertyControl(epc, value)}
        </div>
      ))}
    </div>
  );
}
```

## エイリアス管理

### エイリアスの追加・削除

```tsx
function AliasManager({ echonet }: { echonet: ECHONETHook }) {
  const [aliasName, setAliasName] = useState('');
  const [selectedDevice, setSelectedDevice] = useState('');

  const handleAddAlias = async () => {
    if (!aliasName || !selectedDevice) return;
    
    try {
      await echonet.addAlias(aliasName, selectedDevice);
      setAliasName('');
      setSelectedDevice('');
    } catch (error) {
      console.error('エイリアス追加エラー:', error);
    }
  };

  const handleDeleteAlias = async (alias: string) => {
    try {
      await echonet.deleteAlias(alias);
    } catch (error) {
      console.error('エイリアス削除エラー:', error);
    }
  };

  return (
    <div>
      <h3>エイリアス管理</h3>
      
      {/* 現在のエイリアス一覧 */}
      <div>
        {Object.entries(echonet.aliases).map(([alias, deviceId]) => (
          <div key={alias}>
            <span>{alias} → {deviceId}</span>
            <button onClick={() => handleDeleteAlias(alias)}>削除</button>
          </div>
        ))}
      </div>

      {/* 新規エイリアス追加 */}
      <div>
        <input
          type="text"
          placeholder="エイリアス名"
          value={aliasName}
          onChange={(e) => setAliasName(e.target.value)}
        />
        <select
          value={selectedDevice}
          onChange={(e) => setSelectedDevice(e.target.value)}
        >
          <option value="">デバイスを選択</option>
          {Object.values(echonet.devices).map(device => (
            <option key={device.id} value={device.id}>
              {device.name} ({device.ip} {device.eoj})
            </option>
          ))}
        </select>
        <button onClick={handleAddAlias}>追加</button>
      </div>
    </div>
  );
}
```

## グループ管理

### グループの作成と管理

```tsx
function GroupManager({ echonet }: { echonet: ECHONETHook }) {
  const [groupName, setGroupName] = useState('');
  const [selectedDevices, setSelectedDevices] = useState<string[]>([]);

  const handleCreateGroup = async () => {
    if (!groupName || selectedDevices.length === 0) return;
    
    try {
      await echonet.addGroup(groupName, selectedDevices);
      setGroupName('');
      setSelectedDevices([]);
    } catch (error) {
      console.error('グループ作成エラー:', error);
    }
  };

  const handleDeleteGroup = async (group: string) => {
    try {
      await echonet.deleteGroup(group);
    } catch (error) {
      console.error('グループ削除エラー:', error);
    }
  };

  return (
    <div>
      <h3>グループ管理</h3>
      
      {/* 現在のグループ一覧 */}
      <div>
        {Object.entries(echonet.groups).map(([group, devices]) => (
          <div key={group}>
            <h4>{group}</h4>
            <ul>
              {devices.map(deviceId => (
                <li key={deviceId}>{deviceId}</li>
              ))}
            </ul>
            <button onClick={() => handleDeleteGroup(group)}>グループ削除</button>
          </div>
        ))}
      </div>

      {/* 新規グループ作成 */}
      <div>
        <input
          type="text"
          placeholder="グループ名（@で開始）"
          value={groupName}
          onChange={(e) => setGroupName(e.target.value)}
        />
        
        <div>
          {Object.values(echonet.devices).map(device => (
            <label key={device.id}>
              <input
                type="checkbox"
                checked={selectedDevices.includes(device.id)}
                onChange={(e) => {
                  if (e.target.checked) {
                    setSelectedDevices([...selectedDevices, device.id]);
                  } else {
                    setSelectedDevices(selectedDevices.filter(id => id !== device.id));
                  }
                }}
              />
              {device.name} ({device.ip} {device.eoj})
            </label>
          ))}
        </div>
        
        <button onClick={handleCreateGroup}>グループ作成</button>
      </div>
    </div>
  );
}
```

## 接続状態の監視

### 接続状態とエラーハンドリング

```tsx
function ConnectionStatus({ echonet }: { echonet: ECHONETHook }) {
  const { connectionState } = echonet;

  const getStatusDisplay = () => {
    switch (connectionState) {
      case 'connecting':
        return { text: '接続中...', color: 'orange' };
      case 'connected':
        return { text: '接続済み', color: 'green' };
      case 'disconnected':
        return { text: '切断', color: 'red' };
      case 'error':
        return { text: 'エラー', color: 'red' };
      default:
        return { text: '不明', color: 'gray' };
    }
  };

  const status = getStatusDisplay();

  return (
    <div style={{ color: status.color }}>
      <strong>接続状態: {status.text}</strong>
      
      {connectionState === 'disconnected' && (
        <button onClick={echonet.connect}>再接続</button>
      )}
    </div>
  );
}
```

## 実装のベストプラクティス

### 1. エラーハンドリング

すべての非同期操作で適切なエラーハンドリングを実装してください：

```tsx
const handleOperation = async () => {
  try {
    await echonet.someOperation();
  } catch (error) {
    // ユーザーにエラーを表示
    setErrorMessage(error.message);
    console.error('操作エラー:', error);
  }
};
```

### 2. ローディング状態の管理

ユーザーに操作の進行状況を示すためにローディング状態を管理してください：

```tsx
const [loading, setLoading] = useState(false);

const handleAsyncOperation = async () => {
  setLoading(true);
  try {
    await echonet.someOperation();
  } finally {
    setLoading(false);
  }
};
```

### 3. デバイス探索の実行タイミング

アプリケーション起動時やユーザーの明示的な要求時にデバイス探索を実行してください：

```tsx
useEffect(() => {
  if (echonet.connectionState === 'connected') {
    echonet.discoverDevices();
  }
}, [echonet.connectionState]);
```

### 4. プロパティ詳細情報のキャッシュ活用

`getPropertyDescription` の結果は自動的にキャッシュされるため、同じクラスコードに対して繰り返し呼び出しても効率的です。

## 型定義

主要な型は `hooks/types.ts` で定義されています：

- `Device`: デバイス情報
- `PropertyValue`: プロパティ値
- `DeviceAlias`: エイリアス情報
- `DeviceGroup`: グループ情報
- `PropertyDescriptionData`: プロパティ詳細情報
- `ECHONETHook`: メインフックの戻り値型

TypeScript を使用することで、型安全性が保証され、開発効率が向上します。
