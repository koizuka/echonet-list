# ECHONET Lite WebSocket プロトコル クライアント開発ガイド

## 1. 概要

ECHONET Lite WebSocketプロトコルは、ECHONET Liteデバイスの遠隔操作・監視を可能にするためのプロトコルです。このプロトコルを使用することで、クライアントアプリケーションからECHONET Liteデバイス（エアコン、照明、家電など）の状態を取得したり、操作したりすることができます。

WebSocketを使用する主な利点：

- 双方向通信：サーバーからクライアントへのプッシュ通知が可能
- 軽量：HTTPに比べてオーバーヘッドが少ない
- リアルタイム性：デバイスの状態変化をリアルタイムに通知可能
- クロスプラットフォーム：ブラウザ、モバイルアプリ、デスクトップアプリなど様々な環境で利用可能

基本的な通信フローは以下の通りです：

1. クライアントがWebSocketサーバーに接続
2. サーバーが初期状態（`initial_state`）をクライアントに送信
3. クライアントはリクエストを送信し、サーバーは応答を返す
4. サーバーは非同期にデバイスの状態変化を通知
5. クライアントは必要に応じてデバイスの操作や状態取得を行う

## 2. 接続

### サーバーへの接続方法

WebSocketサーバーへの接続は、使用する言語やライブラリによって異なりますが、基本的な手順は同じです。以下は一般的な接続方法です：

#### サーバーURLの形式

```
ws://hostname:port/ws      // 非暗号化接続
wss://hostname:port/ws     // SSL/TLS暗号化接続
```

例：
- `ws://localhost:8080/ws` (ローカル開発環境)
- `wss://echonet.example.com/ws` (本番環境)

#### 接続確立

使用する言語のWebSocketライブラリを使用して接続を確立します。接続が成功すると、サーバーは最初のメッセージとして `initial_state` を送信します。

#### 切断処理

クライアントが明示的に切断する場合や、エラーや接続タイムアウトが発生した場合の処理を実装する必要があります。必要に応じて再接続ロジックも実装します。

## 3. メッセージフォーマット

すべてのメッセージはJSON形式で送受信されます。

### 基本構造

```json
{
  "type": "message_type",
  "payload": { /* メッセージ固有のデータ */ },
  "requestId": "req-123"  // リクエスト時・レスポンス時のみ
}
```

- `type`: メッセージの種類を示す文字列（必須）
- `payload`: メッセージ固有のデータを含むJSONオブジェクト（必須）
- `requestId`: クライアントからのリクエストに対応するID（リクエスト時・レスポンス時に使用、オプショナル）

### データ型

#### Device（デバイス情報）

```json
{
  "ip": "192.168.1.10",
  "eoj": "0130:1",
  "name": "HomeAirConditioner",
  "properties": {
    "80": "MzA=",  // EPC "80" (OperationStatus) の値 "30" (ON) をBase64エンコード
    "B0": "MjU="   // EPC "B0" (温度設定) の値 "25" をBase64エンコード
  },
  "lastSeen": "2023-04-01T12:34:56Z"
}
```

- `ip`: デバイスのIPアドレス（文字列）
- `eoj`: ECHONET Lite オブジェクト識別子（文字列、形式: "CCCC:I"）
  - CCCC: 4桁の16進数クラスコード（例: "0130" = エアコン）
  - I: 10進数インスタンスコード（例: "1"）
- `name`: デバイスの名前（文字列）
- `properties`: プロパティのマップ
  - キー: 2桁の16進数EPC（プロパティコード）文字列
  - 値: EDTのBase64エンコード文字列
- `lastSeen`: デバイスが最後に応答した時刻（ISO 8601形式）

#### Error（エラー情報）

```json
{
  "code": "ECHONET_TIMEOUT",
  "message": "Device did not respond within timeout period"
}
```

- `code`: エラーコード（文字列）
- `message`: エラーの詳細メッセージ（文字列）

#### ErrorCode（エラーコード一覧）

クライアントリクエスト関連：
- `INVALID_REQUEST_FORMAT`: リクエストの形式が不正
- `INVALID_PARAMETERS`: パラメータが不正
- `TARGET_NOT_FOUND`: 対象デバイスが見つからない
- `ALIAS_OPERATION_FAILED`: エイリアス操作に失敗
- `ALIAS_ALREADY_EXISTS`: エイリアスが既に存在する
- `INVALID_ALIAS_NAME`: エイリアス名が不正
- `ALIAS_NOT_FOUND`: エイリアスが見つからない

サーバー/通信関連：
- `ECHONET_TIMEOUT`: ECHONET Liteデバイスからの応答がタイムアウト
- `ECHONET_DEVICE_ERROR`: ECHONET Liteデバイスからのエラー応答
- `ECHONET_COMMUNICATION_ERROR`: ECHONET Lite通信エラー
- `INTERNAL_SERVER_ERROR`: サーバー内部エラー

### 注意事項

- ECHONET Lite の EPC（プロパティコード）は16進数文字列（例: "80"）で表現されます
- EDT（プロパティ値データ）はBase64エンコードされた文字列で表現されます
- デバイス識別子は `IP EOJ` 形式の文字列（例: "192.168.1.10 0130:1"）で表現されます
- デバイスのIDStringは `EOJ:ManufacturerCode:UniqueIdentifier` 形式の文字列（例: "013001:FE0000:08D0C5D3C3E17B000000000000"）で表現されます
  - EOJは6桁の16進数（例: "013001"）
  - ManufacturerCodeはEPC=0x83（識別番号）のプロパティの最初の3バイトを16進数で表現したもの
  - UniqueIdentifierはEPC=0x83（識別番号）のプロパティの残り13バイトを16進数で表現したもの

## 4. サーバー -> クライアント メッセージ（通知）

サーバーからクライアントへ非同期に送信されるJSONメッセージです。`requestId` は含まれません。クライアントは `type` フィールドを見て処理を分岐します。

### initial_state

接続確立時に現在のデバイス状態とエイリアスを通知します。

```json
{
  "type": "initial_state",
  "payload": {
    "devices": {
      "192.168.1.10 0130:1": {
        "ip": "192.168.1.10",
        "eoj": "0130:1",
        "name": "HomeAirConditioner",
        "properties": {
          "80": "MzA=",
          "B0": "MjU="
        },
        "lastSeen": "2023-04-01T12:34:56Z"
      },
      "192.168.1.11 0290:1": {
        "ip": "192.168.1.11",
        "eoj": "0290:1",
        "name": "LightingSystem",
        "properties": {
          "80": "MzA=",
          "B0": "NTA="
        },
        "lastSeen": "2023-04-01T12:35:00Z"
      }
    },
    "aliases": {
      "living_ac": "013001:FE0000:08D0C5D3C3E17B000000000000",
      "bedroom_light": "029001:FFFFFF:9876543210FEDCBA9876543210"
    },
    "groups": {
      "@living_room": ["192.168.1.10 0130:1", "192.168.1.11 0290:1"],
      "@bedroom": ["192.168.1.12 0130:2"]
    }
  }
}
```

### device_added

新しいデバイスが検出されたことを通知します。

```json
{
  "type": "device_added",
  "payload": {
    "device": {
      "ip": "192.168.1.12",
      "eoj": "0130:2",
      "name": "HomeAirConditioner",
      "properties": {},
      "lastSeen": "2023-04-01T12:36:00Z"
    }
  }
}
```

### device_updated

デバイス情報（プロパティなど）が更新されたことを通知します。

```json
{
  "type": "device_updated",
  "payload": {
    "device": {
      "ip": "192.168.1.10",
      "eoj": "0130:1",
      "name": "HomeAirConditioner",
      "properties": {
        "80": "MzA=",
        "B0": "MjY="  // 温度設定が変更された
      },
      "lastSeen": "2023-04-01T12:37:00Z"
    }
  }
}
```

### device_removed

デバイスがネットワークから切断された、またはタイムアウトしたことを通知します。

```json
{
  "type": "device_removed",
  "payload": {
    "ip": "192.168.1.12",
    "eoj": "0130:2"
  }
}
```

### alias_changed

デバイスエイリアスが追加・更新・削除されたことを通知します。

```json
{
  "type": "alias_changed",
  "payload": {
    "change_type": "added",  // "added", "updated", "deleted" のいずれか
    "alias": "kitchen_ac",
    "target": "013001:FE0000:08D0C5D3C3E17B000000000000"
  }
}
```

### property_changed

デバイスのプロパティ値が変化したことを通知します。

```json
{
  "type": "property_changed",
  "payload": {
    "ip": "192.168.1.10",
    "eoj": "0130:1",
    "epc": "80",
    "value": "MzE="  // "31" (OFF) をBase64エンコード
  }
}
```

### timeout_notification

デバイスとの通信でタイムアウトが発生したことを通知します。

```json
{
  "type": "timeout_notification",
  "payload": {
    "ip": "192.168.1.10",
    "eoj": "0130:1",
    "code": "ECHONET_TIMEOUT",
    "message": "Device did not respond within timeout period"
  }
}
```

### group_changed

デバイスグループが追加・更新・削除されたことを通知します。

```json
{
  "type": "group_changed",
  "payload": {
    "change_type": "added",  // "added", "updated", "deleted" のいずれか
    "group": "@living_room",
    "devices": ["192.168.1.10 0130:1", "192.168.1.11 0290:1"]  // change_type が "deleted" の場合は省略可能
  }
}
```

- `change_type`: 変更の種類（"added"=追加, "updated"=更新, "deleted"=削除）
- `group`: グループ名（"@" で始まる文字列）
- `devices`: グループに含まれるデバイスID文字列の配列（change_type が "deleted" の場合は省略可能）

### error_notification

サーバー内部やECHONET Lite通信でエラーが発生したことを通知します。

```json
{
  "type": "error_notification",
  "payload": {
    "code": "INTERNAL_SERVER_ERROR",
    "message": "Failed to process request due to internal error"
  }
}
```

## 5. クライアント -> サーバー メッセージ（リクエスト）

クライアントからサーバーへ操作を要求するJSONメッセージです。一意な `requestId`（文字列）を含める必要があります。サーバーは対応する `requestId` を持つ `command_result` メッセージで応答します。

### get_properties

指定したデバイスのプロパティ値を取得します。

```json
{
  "type": "get_properties",
  "payload": {
    "targets": ["192.168.1.10 0130:1"],
    "epcs": ["80", "B0", "B3"]
  },
  "requestId": "req-123"
}
```

- `targets`: デバイスID文字列（IP EOJ形式）の配列
- `epcs`: EPC文字列（例: "80"）の配列

### set_properties

指定したデバイスのプロパティ値を設定します。

```json
{
  "type": "set_properties",
  "payload": {
    "target": "192.168.1.10 0130:1",
    "properties": {
      "80": "MzA=",  // "30" (ON) をBase64エンコード
      "B0": "MjU="   // "25" (温度設定) をBase64エンコード
    }
  },
  "requestId": "req-124"
}
```

- `target`: デバイスID文字列（IP EOJ形式）
- `properties`: 設定するプロパティのマップ（キー: EPC文字列, 値: EDTのBase64文字列）

### update_properties

指定したデバイスのプロパティ情報をサーバーに再取得させます。

```json
{
  "type": "update_properties",
  "payload": {
    "targets": ["192.168.1.10 0130:1", "192.168.1.11 0290:1"]
  },
  "requestId": "req-125"
}
```

- `targets`: デバイスID文字列（IP EOJ形式）の配列

### manage_alias

デバイスエイリアスの追加・削除を行います。

```json
{
  "type": "manage_alias",
  "payload": {
    "action": "add",  // "add" または "delete"
    "alias": "bedroom_ac",
    "target": "013001:FE0000:08D0C5D3C3E17B000000000000"  // action が "add" の場合必須
  },
  "requestId": "req-126"
}
```

- `action`: "add"（追加）または "delete"（削除）
- `alias`: エイリアス文字列
- `target`: デバイスIDString（EOJ:ManufacturerCode:UniqueIdentifier形式、`action`が"add"の場合必須）

### manage_group

デバイスグループの追加・削除・更新を行います。

```json
{
  "type": "manage_group",
  "payload": {
    "action": "add",  // "add", "remove", "delete", "list" のいずれか
    "group": "@living_room",
    "devices": ["192.168.1.10 0130:1", "192.168.1.11 0290:1"]  // action が "add" または "remove" の場合必須
  },
  "requestId": "req-128"
}
```

- `action`: 操作の種類
  - "add": グループを作成またはデバイスを追加
  - "remove": グループからデバイスを削除
  - "delete": グループを削除
  - "list": グループ一覧または特定グループの情報を取得
- `group`: グループ名（"@" で始まる文字列）
- `devices`: デバイスID文字列（IP EOJ形式）の配列（`action` が "add" または "remove" の場合必須）

### discover_devices

ネットワーク上のECHONET Liteデバイスを再探索します。

```json
{
  "type": "discover_devices",
  "payload": {},
  "requestId": "req-127"
}
```

- `payload`: 空のJSONオブジェクト `{}`

### get_property_aliases

指定したクラスコードに対応するプロパティエイリアス一覧を取得します。

```json
{
  "type": "get_property_aliases",
  "payload": {
    "classCode": "0130"
  },
  "requestId": "req-128"
}
```

- `classCode`: 4桁の16進数クラスコード（例: "0130" = エアコン）

## 6. サーバー -> クライアント メッセージ（応答）

クライアントからのリクエストに対する応答JSONメッセージです。リクエストと同じ `requestId` を含みます。

### property_aliases_result

`get_property_aliases` リクエストに対する応答です。

成功時の例：

```json
{
  "type": "property_aliases_result",
  "payload": {
    "success": true,
    "data": {
      "classCode": "0130",
      "properties": {
        "80": {
          "description": "Operation status",
          "aliases": {
            "on": "MzA=",
            "off": "MzE="
          }
        },
        "B0": {
          "description": "Operation mode setting",
          "aliases": {
            "auto": "NDE=",
            "cooling": "NDI=",
            "heating": "NDM=",
            "dehumidification": "NDQ=",
            "ventilation": "NDU="
          }
        },
        "B3": {
          "description": "Air flow rate setting",
          "aliases": {
            "silent": "NDE=",
            "standard": "NDI=",
            "powerful": "NDM="
          }
        }
      }
    }
  },
  "requestId": "req-128"
}
```

失敗時の例：

```json
{
  "type": "property_aliases_result",
  "payload": {
    "success": false,
    "error": {
      "code": "INVALID_PARAMETERS",
      "message": "Invalid class code: ABCD (must be 4 hexadecimal digits)"
    }
  },
  "requestId": "req-128"
}
```

- `success`: 操作が成功したかどうか（boolean）
- `data`: 成功時のデータ
  - `classCode`: リクエストされたクラスコード
  - `properties`: EPCとその情報のマップ
    - キー: EPC (16進数文字列)（例: "80" = 動作状態）
    - 値: EPCの情報
      - `description`: EPCの説明（例: "Operation status"）
      - `aliases`: エイリアス名とEDT値のマップ
        - キー: エイリアス名（例: "on", "off", "cooling"）
        - 値: EDT (Base64エンコード文字列)（例: "MzA=" = 0x30 = ON）
- `error`: 失敗時のエラー情報（`Error` オブジェクト、成功時は null または undefined）

### command_result

各リクエスト操作の結果を返します。

成功時の例：

```json
{
  "type": "command_result",
  "payload": {
    "success": true,
    "data": {
      "ip": "192.168.1.10",
      "eoj": "0130:1",
      "name": "HomeAirConditioner",
      "properties": {
        "80": "MzA=",
        "B0": "MjU="
      },
      "lastSeen": "2023-04-01T12:38:00Z"
    }
  },
  "requestId": "req-123"
}
```

失敗時の例：

```json
{
  "type": "command_result",
  "payload": {
    "success": false,
    "error": {
      "code": "TARGET_NOT_FOUND",
      "message": "Device 192.168.1.99 0130:1 not found"
    }
  },
  "requestId": "req-123"
}
```

- `success`: 操作が成功したかどうか（boolean）
- `data`: 成功時の追加データ（JSON形式、内容はリクエストによる）
- `error`: 失敗時のエラー情報（`Error` オブジェクト、成功時は null または undefined）

## 7. クライアント実装のポイント（言語非依存）

### WebSocketライブラリの選択

使用する言語に適した標準的なWebSocketクライアントライブラリを選びます：

- JavaScript/TypeScript: ブラウザ標準の `WebSocket` API、Node.jsの `ws` ライブラリ
- Python: `websockets` または `websocket-client` ライブラリ
- Java: `Java WebSocket API (JSR 356)` または `Tyrus`
- C#: `System.Net.WebSockets` または `SignalR`
- Go: `gorilla/websocket` ライブラリ

### 接続管理

- 接続確立: サーバーURLに接続し、WebSocketハンドシェイクを行います
- エラーハンドリング: 接続エラー、切断、タイムアウトなどを処理します
- 再接続ロジック: 接続が切れた場合に自動的に再接続を試みます

### メッセージ送受信

- 送信: 上記「クライアント -> サーバー メッセージ」で定義されたJSONオブジェクトを文字列化して送信します。リクエストごとに一意な `requestId` を生成・付与します。
- 受信: サーバーから受信したメッセージをJSONとしてパースします。

### リクエストと応答のマッチング

- 送信したリクエストの `requestId` と、受信した `command_result` メッセージの `requestId` を照合して、どのリクエストに対する応答かを判断します。
- タイムアウト処理: 一定時間内に応答がない場合はタイムアウトとして処理します。

### 通知の処理

- `requestId` を持たないメッセージ（通知）を受信した場合、`type` に応じてクライアントの状態（デバイスリスト、エイリアスリストなど）を更新します。
- 各通知タイプに対応するハンドラを実装します。

### 状態管理

- サーバーから受信した `initial_state` や各種通知メッセージに基づき、クライアント側でデバイスやエイリアスの状態を管理・保持します。
- デバイスリスト、プロパティ値、エイリアスなどの情報を適切なデータ構造で管理します。

### データ変換

- EPC（16進文字列）や EDT（Base64文字列）をアプリケーションで扱いやすい形式に変換します。
- 例: Base64デコード、数値への変換、列挙型への変換など

## 8. エラーハンドリング

### WebSocket接続自体のエラー

- 接続失敗: サーバーが見つからない、応答がない
- 切断: ネットワーク問題、サーバーシャットダウン
- 再接続戦略: 指数バックオフなどを使用した再接続

### 受信メッセージのパースエラー

- JSONパースエラー: 不正なJSON形式
- スキーマ検証エラー: 必須フィールドの欠落、型の不一致

### command_result メッセージのエラー

- `success` が `false` の場合、`error` フィールドを確認
- エラーコードに応じた適切な処理を実装

### 通知エラー

- `error_notification`: サーバー内部エラー
- `timeout_notification`: デバイス通信タイムアウト

## 9. 実装例（TypeScript - 概念）

```typescript
// WebSocket接続 (例: ブラウザ環境)
const socket = new WebSocket("ws://localhost:8080/ws");

let requestIdCounter = 0;
const pendingRequests = new Map<string, (response: any) => void>();

// デバイス情報を保持する変数
let devices = {};
let aliases = {};
let groups = {};

socket.onopen = () => {
  console.log("WebSocket connected");
  // 接続確立時に必要な処理を実行
  // initial_state メッセージは自動的にサーバーから送信される
  
  // 例: 接続確立後にデバイス探索を開始
  discoverDevices().catch(err => {
    console.error("Failed to discover devices:", err);
  });
};

socket.onmessage = (event) => {
  try {
    const message = JSON.parse(event.data);
    console.log("Received:", message);

    if (message.requestId && pendingRequests.has(message.requestId)) {
      // リクエストへの応答 (command_result)
      const callback = pendingRequests.get(message.requestId);
      if (callback) {
        callback(message.payload); // payload をコールバックに渡す
        pendingRequests.delete(message.requestId);
      }
    } else {
      // サーバーからの通知
      handleNotification(message.type, message.payload);
    }
  } catch (error) {
    console.error("Failed to parse message or handle:", error);
  }
};

socket.onerror = (error) => {
  console.error("WebSocket error:", error);
};

socket.onclose = () => {
  console.log("WebSocket disconnected");
  // 必要であれば再接続処理
};

// 通知処理の例
function handleNotification(type: string, payload: any) {
  switch (type) {
    case "initial_state":
      console.log("Initial state received:", payload.devices, payload.aliases, payload.groups);
      // アプリケーションの状態を初期化
      devices = payload.devices;
      aliases = payload.aliases;
      groups = payload.groups;
      
      // 初期状態を受け取った後の処理
      // 例: UIの更新、デバイスリストの表示など
      break;
      
    case "device_added":
      console.log("Device added:", payload.device);
      // デバイスリストに追加
      const deviceId = `${payload.device.ip} ${payload.device.eoj}`;
      devices[deviceId] = payload.device;
      break;
      
    // 他の通知タイプも同様に処理...
    case "property_changed":
      console.log(`Property ${payload.epc} changed for ${payload.ip} ${payload.eoj} to ${atob(payload.value)}`); // Base64デコード例
      // 対応するデバイスのプロパティを更新
      const targetDeviceId = `${payload.ip} ${payload.eoj}`;
      if (devices[targetDeviceId]) {
        devices[targetDeviceId].properties[payload.epc] = payload.value;
      }
      break;
      
    case "group_changed":
      console.log(`Group ${payload.group} ${payload.change_type}:`, payload.devices);
      // グループの変更を処理
      switch (payload.change_type) {
        case "added":
          groups[payload.group] = payload.devices;
          break;
        case "updated":
          groups[payload.group] = payload.devices;
          break;
        case "deleted":
          delete groups[payload.group];
          break;
      }
      break;
      
    case "error_notification":
      console.error(`Server Error: ${payload.code} - ${payload.message}`);
      break;
  }
}

// リクエスト送信関数 (例)
function sendRequest(type: string, payload: any): Promise<any> {
  return new Promise((resolve, reject) => {
    const requestId = `req-${requestIdCounter++}`;
    const message = {
      type: type,
      payload: payload,
      requestId: requestId,
    };

    // タイムアウト処理
    const timeoutId = setTimeout(() => {
        pendingRequests.delete(requestId);
        reject(new Error(`Request ${requestId} timed out`));
    }, 10000); // 10秒タイムアウト

    pendingRequests.set(requestId, (responsePayload) => {
        clearTimeout(timeoutId);
        if (responsePayload.success) {
            resolve(responsePayload.data); // 成功時の data を返す
        } else {
            reject(responsePayload.error); // 失敗時の error を返す
        }
    });

    console.log("Sending:", message);
    socket.send(JSON.stringify(message));
  });
}

// --- API関数の例 ---

// デバイス探索
async function discoverDevices() {
  const payload = {};
  return sendRequest("discover_devices", payload);
}

// デバイスのプロパティ取得
async function getDeviceProperties(targetDevice: string, epcs: string[]) {
  try {
    const payload = { targets: [targetDevice], epcs: epcs };
    const resultData = await sendRequest("get_properties", payload);
    console.log(`Properties for ${targetDevice}:`, resultData);
    return resultData;
  } catch (error) {
    console.error(`Failed to get properties for ${targetDevice}:`, error);
    throw error;
  }
}

// デバイスのプロパティ設定
async function setDeviceProperties(targetDevice: string, properties: Record<string, string>) {
  try {
    const payload = { target: targetDevice, properties: properties };
    const resultData = await sendRequest("set_properties", payload);
    console.log(`Set properties for ${targetDevice}:`, resultData);
    return resultData;
  } catch (error) {
    console.error(`Failed to set properties for ${targetDevice}:`, error);
    throw error;
  }
}

// エイリアス追加
async function addAlias(alias: string, targetDevice: string) {
  try {
    const payload = { action: "add", alias: alias, target: targetDevice };
    const resultData = await sendRequest("manage_alias", payload);
    console.log(`Added alias ${alias} for ${targetDevice}:`, resultData);
    return resultData;
  } catch (error) {
    console.error(`Failed to add alias ${alias} for ${targetDevice}:`, error);
    throw error;
  }
}

// グループ追加
async function addGroup(groupName: string, devices: string[]) {
  try {
    const payload = { action: "add", group: groupName, devices: devices };
    const resultData = await sendRequest("manage_group", payload);
    console.log(`Added group ${groupName} with devices:`, devices);
    return resultData;
  } catch (error) {
    console.error(`Failed to add group ${groupName}:`, error);
    throw error;
  }
}

// グループからデバイスを削除
async function removeFromGroup(groupName: string, devices: string[]) {
  try {
    const payload = { action: "remove", group: groupName, devices: devices };
    const resultData = await sendRequest("manage_group", payload);
    console.log(`Removed devices from group ${groupName}:`, devices);
    return resultData;
  } catch (error) {
    console.error(`Failed to remove devices from group ${groupName}:`, error);
    throw error;
  }
}

// グループ削除
async function deleteGroup(groupName: string) {
  try {
    const payload = { action: "delete", group: groupName };
    const resultData = await sendRequest("manage_group", payload);
    console.log(`Deleted group ${groupName}`);
    return resultData;
  } catch (error) {
    console.error(`Failed to delete group ${groupName}:`, error);
    throw error;
  }
}

// グループ一覧取得
async function listGroups(groupName?: string) {
  try {
    const payload = { action: "list", group: groupName };
    const resultData = await sendRequest("manage_group", payload);
    console.log(`Group list:`, resultData);
    return resultData;
  } catch (error) {
    console.error(`Failed to get group list:`, error);
    throw error;
  }
}

// 使用例:
// 接続確立後（onopen内）で実行するか、initial_state受信後に実行
// getDeviceProperties("192.168.1.10 0130:1", ["80", "B0"]);
// addGroup("@living_room", ["192.168.1.10 0130:1", "192.168.1.11 0290:1"]);
```

このコード例は概念的なものであり、実際の実装では言語やフレームワークに応じた適切なエラーハンドリングやタイプセーフな実装が必要です。
