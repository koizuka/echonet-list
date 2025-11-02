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
  "id": "013001:00000B:ABCDEF0123456789ABCDEF012345", // GetIDString()で生成: EOJ.IDString():IdentificationNumber.String()
  "properties": {
    "80": { "EDT": "MzA=", "string": "on" },  // EPC "80" (OperationStatus)
    "B3": { "EDT": "MjU=", "string": "25", "number": 25 }   // EPC "B3" (温度設定)
  },
  "lastSeen": "2023-04-01T12:34:56Z",
  "isOffline": false // オプション：デバイスがオフライン状態の場合のみ true が設定される
}
```

- `ip`: デバイスのIPアドレス（文字列）
- `eoj`: ECHONET Lite オブジェクト識別子（文字列、形式: "CCCC:I"）
  - CCCC: 4桁の16進数クラスコード（例: "0130" = エアコン）
  - I: 10進数インスタンスコード（例: "1"）
- `name`: デバイスの名前（文字列）
- `id`: デバイス識別子（`GetIDString()`で生成、デバイスエイリアス照合で使用）
  - 形式: `EOJ.IDString():IdentificationNumber.String()`
  - `EOJ.IDString()`: 6桁16進数（例: "013001"）
  - `IdentificationNumber.String()`: 同一IPのNodeProfileObjectのEPC 83から取得（例: "00000B:ABCDEF0123456789ABCDEF012345"）
  - **重要**: エイリアス照合では、デバイス自身のIDではなく、同一IPのNodeProfileObjectのIdentificationNumberを使用
- `properties`: プロパティのマップ
  - キー: 2桁の16進数EPC（プロパティコード）文字列
  - 値: オブジェクト { "EDT": "Base64エンコード文字列", "string": "文字列表現", "number": 数値 }
    - `EDT`: Base64エンコードされたバイト列（必須ではない）
    - `string`: 人間が読める文字列表現（必須ではない）
    - `number`: 数値表現（PropertyDescにNumberDescが含まれる場合のみ使用可能、必須ではない）
- `lastSeen`: デバイスのプロパティが最後に更新された時刻（ISO 8601形式）
- `isOffline`: デバイスのオフライン状態（オプション、`omitempty`）
  - `true`: デバイスがオフライン状態（通信不可）
  - `false` または未設定: デバイスがオンライン状態
  - `initial_state` メッセージでオフラインデバイスも含めて送信される

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
- デバイスのIDStringは `EOJ:ManufacturerCode:UniqueIdentifier` 形式の文字列（例: "013001:00000B:ABCDEF0123456789ABCDEF012345"）で表現されます
  - EOJは6桁の16進数（例: "013001"）
  - ManufacturerCode, UniqueIdentifierは、**同じIPアドレスを持つNodeProfileObject(EOJ=0EF0:1)のEPC=0x83（識別番号）** のプロパティ値（17バイト）から、先頭の1バイト(0xFE)を除いた残り16バイトのうち先頭3バイト(ManufacturerCode)と残り13バイト(UniqueIdentifier)を `:` で区切ってそれぞれ16進数文字列で表現したもの。ManufacturerCode はEPC=0x8A(メーカコード)と同じ(例: "00000B" = Panasonic)

## 4. サーバー -> クライアント メッセージ（通知）

サーバーからクライアントへ非同期に送信されるJSONメッセージです。`requestId` は含まれません。クライアントは `type` フィールドを見て処理を分岐します。

### initial_state

接続確立時に現在のデバイス状態とエイリアス、およびサーバーの起動時刻を通知します。

```json
{
  "type": "initial_state",
  "payload": {
    "devices": {
      "192.168.1.10 0130:1": {
        "ip": "192.168.1.10",
        "eoj": "0130:1",
        "name": "HomeAirConditioner",
        "id": "013001:00000B:ABCDEF0123456789ABCDEF012345", // 例
        "properties": {
          "80": { "EDT": "MzA=", "string": "on" },
          "B3": { "EDT": "MjU=", "string": "25", "number": 25 }
        },
        "lastSeen": "2023-04-01T12:34:56Z"
      },
      "192.168.1.11 0290:1": {
        "ip": "192.168.1.11",
        "eoj": "0290:1",
        "name": "LightingSystem",
        "properties": {
          "80": { "EDT": "MzA=", "string": "on" },
          "B3": { "EDT": "NTA=", "string": "50", "number": 50 }
        },
        "lastSeen": "2023-04-01T12:35:00Z"
      },
      "192.168.1.12 0130:2": {
        "ip": "192.168.1.12",
        "eoj": "0130:2",
        "name": "HomeAirConditioner",
        "properties": {
          "80": { "EDT": "MzE=", "string": "off" },
          "B3": { "EDT": "MjI=", "string": "22", "number": 22 }
        },
        "lastSeen": "2023-04-01T12:30:00Z",
        "isOffline": true // オフラインデバイスも initial_state に含まれる
      }
    },
    "aliases": {
      "living_ac": "013001:00000B:ABCDEF0123456789ABCDEF012345", // 例
      "bedroom_light": "029001:000005:FEDCBA9876543210FEDCBA987654" // 例
    },
    "groups": {
      "@living_room": ["013001:00000B:ABCDEF0123456789ABCDEF012345", "029001:000005:FEDCBA9876543210FEDCBA987654"], // 例
      "@bedroom": ["013001:000008:FEDCBA9876543210ABCDEF012345"] // 例
    },
    "serverStartupTime": "2023-04-01T12:00:00Z" // サーバーの起動時刻（ISO 8601形式）
  }
}
```

### device_added

新しいデバイスが検出されたことを通知します。オンライン復旧時にも同じメッセージが送信されます。

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

**使用ケース:**
- 新規デバイス検出時
- デバイスのオンライン復旧時（`device_offline` で削除されたデバイスの復元）

**クライアント実装時の注意事項:**
- `properties` が空の場合（主にオンライン復旧時）、自動的に `list_devices` を実行してキャッシュされたプロパティを取得することを推奨します
- `get_properties` はネットワーク通信を行うため、復旧直後の不安定な状態では失敗する可能性があります
- `list_devices` はサーバーのキャッシュから安定してデータを取得できるため、復旧時に適しています
- 通常の新規検出時も `properties` は空で送信され、後続のプロパティ取得で情報が充実されます

### alias_changed

デバイスエイリアスが追加・更新・削除されたことを通知します。

```json
{
  "type": "alias_changed",
  "payload": {
    "change_type": "added",  // "added", "updated", "deleted" のいずれか
    "alias": "kitchen_ac",
    "target": "013001:00000B:ABCDEF0123456789ABCDEF012345" // 例
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
    "epc": "80", // OperationStatus
    "value": { "EDT": "MzE=", "string": "off" } // "31" (OFF)
  }
}

// 例2: 温度設定 (B3) が 26 度に変更された場合
{
  "type": "property_changed",
  "payload": {
    "ip": "192.168.1.10",
    "eoj": "0130:1",
    "epc": "B3", // Set temperature value
    "value": { "EDT": "MjY=", "string": "26", "number": 26 } // 26
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

### device_offline

デバイスがオフラインとしてマークされたことを通知します。クライアントはこのデバイスにオフライン状態のマーキング（`isOffline: true`）を適用するべきです。

**注意**: このメッセージは既に接続済みのクライアントにのみ送信されます。新規接続クライアントは `initial_state` でオフラインデバイスを `isOffline: true` フィールド付きで受信します。

```json
{
  "type": "device_offline",
  "payload": {
    "ip": "192.168.1.10",
    "eoj": "0130:1"
  }
}
```

- `ip`: デバイスのIPアドレス（文字列）
- `eoj`: ECHONET Lite オブジェクト識別子（文字列、形式: "CCCC:I"）

### device_online

デバイスがオフライン状態からオンラインに復旧したことを通知します。

```json
{
  "type": "device_online",
  "payload": {
    "ip": "192.168.1.10",
    "eoj": "0130:1"
  }
}
```

- `ip`: デバイスのIPアドレス（文字列）
- `eoj`: ECHONET Lite オブジェクト識別子（文字列、形式: "CCCC:I"）

**実装仕様:**
- サーバー側では、デバイスのオンライン復旧時に `device_online` 通知に続いて `device_added` メッセージが自動的に送信されます
- クライアント側では `device_online` 通知を受信後、続いて受信される `device_added` メッセージでデバイスが自動的に復旧します
- オフライン時に削除されたデバイスは、`device_added` メッセージにより完全なデバイス情報と共に復元されます

**クライアント実装時の注意事項:**
- この通知は主に情報提供用途で、実際のデバイス復旧処理は後続の `device_added` メッセージで自動的に行われます
- 特別な処理は不要ですが、ログ出力やユーザー通知などの目的で利用できます

### device_deleted

デバイスが削除されたことを通知します。クライアントはこのデバイスをUIから削除する必要があります。

```json
{
  "type": "device_deleted",
  "payload": {
    "ip": "192.168.1.10",
    "eoj": "0130:1"
  }
}
```

- `ip`: 削除されたデバイスのIPアドレス（文字列）
- `eoj`: ECHONET Lite オブジェクト識別子（文字列、形式: "CCCC:I"）

**使用ケース:**
- 手動でのデバイス削除時
- NodeProfile削除による同一IPアドレスデバイスの一括削除時

**NodeProfile削除時の動作:**
- NodeProfile（クラスコード0x0ef0）が削除されると、同一IPアドレスのすべてのデバイスについて個別に`device_deleted`通知が送信されます
- クライアントは各通知を受信して対応するデバイスをUIから削除します

### group_changed

デバイスグループが追加・更新・削除されたことを通知します。

```json
{
  "type": "group_changed",
  "payload": {
    "change_type": "added",  // "added", "updated", "deleted" のいずれか
    "group": "@living_room",
    "devices": ["013001:00000B:ABCDEF0123456789ABCDEF012345", "029001:000005:FEDCBA9876543210FEDCBA987654"]  // 例, change_type が "deleted" の場合は省略可能
  }
}
```

- `change_type`: 変更の種類（"added"=追加, "updated"=更新, "deleted"=削除）
- `group`: グループ名（"@" で始まる文字列）
- `devices`: グループに含まれるデバイスIDString文字列の配列（change_type が "deleted" の場合は省略可能）

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

### log_notification

サーバー側でError/Warnレベルのログが発生したことを通知します。デバッグやシステム監視に使用できます。

```json
{
  "type": "log_notification",
  "payload": {
    "level": "ERROR",  // "ERROR" または "WARN"
    "message": "Device communication timeout",
    "time": "2023-04-01T12:34:56Z",
    "attributes": {
      "device": "192.168.1.10 0130:1",
      "err": "timeout after 5s"
    }
  }
}
```

- `level`: ログレベル（"ERROR" または "WARN"）
- `message`: ログメッセージ
- `time`: ログ発生時刻（ISO 8601形式）
- `attributes`: ログに関連する追加情報（key-valueペア）

## 4.1. デバイスオフライン/オンライン復旧フロー

デバイスがオフライン状態になった後、オンライン復旧する際の完全なメッセージフローを説明します。

### オフライン → オンライン復旧の流れ

1. **デバイスオフライン**：
   ```
   device_offline メッセージ送信
   → クライアントはデバイスをUIから削除
   ```

2. **デバイスオンライン復旧検出**：
   ```
   device_online メッセージ送信（情報提供用）
   → device_added メッセージ送信（実際の復旧処理）
   → クライアントはデバイスをUIに復元
   ```

3. **プロパティ自動取得**（推奨実装）：
   ```
   device_added で properties が空の場合
   → update_properties を自動実行
   → プロパティ更新でデバイス情報を充実
   ```

### 実装推奨パターン

```javascript
// device_added ハンドラーの推奨実装
case 'device_added':
  // デバイスを状態に追加
  addDevice(message.payload.device);
  
  // プロパティが空の場合は全プロパティを取得
  if (Object.keys(message.payload.device.properties).length === 0) {
    const deviceId = `${message.payload.device.ip} ${message.payload.device.eoj}`;
    // get_properties で確実に全プロパティを取得
    getProperties([deviceId], []); // 空のEPCsで全プロパティ取得
  }
  break;
```

### プロパティ取得方法の使い分け

**オンライン復旧時には `list_devices` を推奨**

- **`update_properties`**: 差分更新のみ。値が変わっていない場合は `property_changed` 通知が送信されない
- **`get_properties`**: ネットワーク通信でリアルタイム取得。復旧直後は失敗する可能性あり
- **`list_devices`**: キャッシュからの安定取得。復旧時に最適

オンライン復旧時は、デバイスが不安定な状態の可能性があるため、`list_devices` を使用してキャッシュされたプロパティ情報を安定して取得することを推奨します。

### 利点

- **シンプルな実装**: 既存の `device_added` ハンドラーでオンライン復旧も処理
- **確実な復元**: `list_devices` により安定してプロパティが取得される
- **自動復元**: デバイスは完全な情報と共に自動的に復元される
- **一貫性**: 新規検出とオンライン復旧で同じメッセージフローを使用

## 5. クライアント -> サーバー メッセージ（リクエスト）

クライアントからサーバーへ操作を要求するJSONメッセージです。一意な `requestId`（文字列）を含める必要があります。サーバーは対応する `requestId` を持つ `command_result` メッセージで応答します。

### get_properties

指定したデバイスのプロパティ値を取得します（ネットワーク通信を実行）。

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

### list_devices

指定したデバイスのキャッシュされたデータを取得します（ネットワーク通信なし）。

```json
{
  "type": "list_devices",
  "payload": {
    "targets": ["192.168.1.10 0130:1"]  // オプション: 空の場合は全オンラインデバイス
  },
  "requestId": "req-124"
}
```

- `targets`: デバイスID文字列（IP EOJ形式）の配列（オプション）

**使用ケース:**
- オンライン復旧時のプロパティ取得（安定性重視）
- キャッシュされたデバイス情報の取得
- `initial_state` と同等の情報をリクエストベースで取得

**利点:**
- ネットワーク通信を行わないため高速・安定
- デバイスが不安定な状態でも失敗しない
- `initial_state` メッセージと同じデータソースを使用

### set_properties

指定したデバイスのプロパティ値を設定します。

```json
{
  "type": "set_properties",
  "payload": {
    "target": "192.168.1.10 0130:1",
    "properties": {
      "80": { "EDT": "MzA=", "string": "on" },
      "B3": { "EDT": "MjU=", "number": 25 }
    }
  },
  "requestId": "req-124"
}
```

- `target`: デバイスID文字列（IP EOJ形式）
- `properties`: 設定するプロパティのマップ。値は以下のいずれかの形式を許容  
  - `{ "EDT": "Base64文字列" }`  
  - `{ "string": "文字列表現" }`  
  - `{ "number": 数値 }`（PropertyDescにNumberDescが含まれる場合のみ使用可能）  
  - `{ "EDT": "Base64文字列", "string": "文字列表現" }`（`EDT` とそれ以外の二つを指定した時は矛盾がない場合のみ有効、矛盾時はエラー）  
  - `number` と `string` の両方が与えられたらエラーになります

### update_properties

指定したデバイスのプロパティ情報をサーバーに再取得させます。`force: true` でなければ、更新したばかりのデバイスの更新は省略します

```json
{
  "type": "update_properties",
  "payload": {
    "targets": ["192.168.1.10 0130:1", "192.168.1.11 0290:1"], // 省略すると全デバイス更新
    "force": true // オプショナル: 強制更新フラグ
  },
  "requestId": "req-125"
}
```

- `targets`: デバイスID文字列（IP EOJ形式）の配列。**省略した場合、または空の配列 (`[]`) を指定した場合は、検出されている全てのデバイスが更新対象となります。**
- `force`: (オプショナル) `true` の場合、デバイスの最終更新時刻に関わらず強制的にプロパティを更新します。デフォルトは `false` です。

### manage_alias

デバイスエイリアスの追加・削除を行います。

```json
{
  "type": "manage_alias",
  "payload": {
    "action": "add",  // "add" または "delete"
    "alias": "bedroom_ac",
    "target": "013001:00000B:ABCDEF0123456789ABCDEF012345"  // 例, action が "add" の場合必須
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
    "devices": ["013001:00000B:ABCDEF0123456789ABCDEF012345", "029001:000005:FEDCBA9876543210FEDCBA987654"]  // 例, action が "add" または "remove" の場合必須
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
- `devices`: デバイスIDString文字列（EOJ:ManufacturerCode:UniqueIdentifier形式）の配列（`action` が "add" または "remove" の場合必須）

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

### delete_device

指定したデバイスを削除します。NodeProfile（クラスコード0x0ef0）を削除する場合、同一IPアドレスのすべてのデバイスが削除されます。

```json
{
  "type": "delete_device",
  "payload": {
    "target": "192.168.1.10 0130:1"
  },
  "requestId": "req-128"
}
```

- `target`: デバイスID文字列（IP EOJ形式）

### get_device_history

指定したデバイスの最近の履歴を取得します（サーバーのオンメモリ履歴ストアから取得）。

```json
{
  "type": "get_device_history",
  "payload": {
    "target": "192.168.1.10 0130:1",
    "limit": 50,               // オプション: 取得件数の上限（既定値 50, サーバー設定値を超える場合は丸め込み）
    "settableOnly": true       // オプション: true で Set Property Map に含まれる履歴のみ（既定 true）
  },
  "requestId": "req-129"
}
```

- `target`: デバイスID文字列（IP EOJ形式）。必須。
- `limit`: 取得件数の上限。正の整数のみ許容。省略時は 50。
- `settableOnly`: `true` の場合、Set Property Map に含まれるプロパティのみ返します。省略時は `true`。

レスポンスは `command_result` メッセージの `data` フィールドに以下の形式で返されます：

```json
{
  "type": "command_result",
  "payload": {
    "success": true,
    "data": {
      "entries": [
        {
          "timestamp": "2024-05-01T12:34:56.789Z",
          "epc": "80",
          "value": { "string": "on", "EDT": "MzA=" },
          "origin": "set",
          "settable": true
        },
        {
          "timestamp": "2024-05-01T12:35:10.123Z",
          "epc": "B0",
          "value": { "number": 24, "EDT": "Eg==" },
          "origin": "notification",
          "settable": true
        }
      ]
    }
  },
  "requestId": "req-129"
}
```

- `entries`: 履歴の配列。新しい順で返されます。
- `timestamp`: ISO 8601 / RFC3339 形式の時刻 (UTC)。
- `epc`: 履歴対象の EPC（2桁16進数文字列）。
- `value`: プロパティ値 (`PropertyData` と同形式)。
- `origin`: `"set"`（set_properties による更新）または `"notification"`（プロパティ変化通知）。
- `settable`: Set Property Map に含まれるプロパティなら `true`。

デバイスが存在しない場合やパラメータが不正な場合は `success: false` となり、`error` に詳細が入ります。

**重要な動作仕様**:
- **NodeProfile削除時**: 指定したデバイスのクラスコードが`0x0ef0`（NodeProfile）の場合、同一IPアドレスのすべてのデバイスが削除されます
- **通常デバイス削除時**: 指定したデバイスのみが削除されます

**削除通知**: 削除された各デバイスについて、すべてのクライアントに`device_deleted`通知が送信されます。

### get_property_description

指定したクラスコード (`classCode`) に対応する各プロパティ (EPC) について、UI での表示や編集に役立つ詳細情報（説明、値のエイリアス、数値範囲、単位、文字列制限など）を取得します。

ECHONET Lite のプロパティ値 (EDT) はバイト列であり直接扱うのが難しいため、この API はクライアントがプロパティ値を解釈し、適切な UI (選択肢、スライダー、テキスト入力など) を提供するのを助けます。

#### 国際化対応

このAPIは多言語対応を提供しており、`lang` パラメータで言語を指定できます：

```json
{
  "type": "get_property_description",
  "payload": {
    "classCode": "0130", // 例: Home Air Conditioner
    "lang": "ja"         // 言語コード (オプション)
  },
  "requestId": "req-128"
}
```

- `classCode`: 4桁の16進数クラスコード（例: "0130" = エアコン）。**空文字列 (`""`) を指定した場合、共通プロパティ（ProfileSuperClass）の情報のみを返します。**
- `lang`: 言語コード（オプション）。指定可能な値：
  - `"ja"`: 日本語
  - `"en"` または省略: 英語（デフォルト）

応答は `command_result` メッセージで返されます。取得した情報の詳細な意味と、それを利用した UI 実装のガイドラインについては、**[クライアント UI 開発ガイド](./client_ui_development_guide.md)** を参照してください。

## 6. サーバー -> クライアント メッセージ（応答）

クライアントからのリクエストに対する応答JSONメッセージです。リクエストと同じ `requestId` を含みます。

### command_result

各リクエスト操作の結果を返します。

**`get_properties`, `set_properties`, `update_properties`, `manage_alias`, `manage_group`, `discover_devices` の成功時の例:**

```json
{
  "type": "command_result",
  "payload": {
    "success": true,
    "data": { /* リクエストに応じたデータ、または null */ }
  },
  "requestId": "req-123"
}
```

**`get_property_description` の成功時の例:**

応答ペイロードの `data` フィールドには `PropertyDescriptionData` オブジェクトが含まれます。このオブジェクトの詳細な構造と各フィールドの意味、および UI での活用方法については、**[クライアント UI 開発ガイド](./client_ui_development_guide.md)** を参照してください。

```json
// 応答例 (英語版、lang="en" または省略時)
{
  "type": "command_result",
  "payload": {
    "success": true,
    "data": {
      "classCode": "0130", // リクエストされたクラスコード
      "properties": {
        "80": { // EPC (例: 動作状態)
          "description": "Operation status",
          "aliases": { "on": "MzA=", "off": "MzE=" }
          // numberDesc, stringDesc はこのプロパティにはない
        },
        "B3": { // EPC (例: 温度設定)
          "description": "Set temperature value",
          // aliases はない
          "numberDesc": { "min": 0, "max": 50, "unit": "C", ... }
        },
        "8C": { // EPC (例: 商品コード)
           "description": "Product code",
           // aliases, numberDesc はない
           "stringDesc": { "maxEDTLen": 12, ... },
           "stringSettable": true // 文字列で設定可能か (オプショナル)
        }
        // ... 他のプロパティ定義
      }
    }
  },
  "requestId": "req-128" // 対応するリクエストID
}
```

```json
// 応答例 (日本語版、lang="ja"時)
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
        "B3": {
          "description": "温度設定値",
          "numberDesc": { "min": 0, "max": 50, "unit": "C", ... }
        },
        "8C": {
           "description": "商品コード",
           "stringDesc": { "maxEDTLen": 12, ... },
           "stringSettable": true
        }
        // ... 他のプロパティ定義
      }
    }
  },
  "requestId": "req-128"
}
```

#### 国際化応答フィールド

- `description`: プロパティの説明文（指定した言語）
- `aliases`: プロパティのエイリアス値（常に英語キー）
- `aliasTranslations`: エイリアスの翻訳テーブル（指定した言語での表示名）

**重要**: プロパティ設定時は `aliases` の英語キーを使用してください。`aliasTranslations` は表示目的のみです。

**失敗時の例（共通）：**

```json
{
  "type": "command_result",
  "payload": {
    "success": false,
    "error": {
      "code": "TARGET_NOT_FOUND", // または他の ErrorCode
      "message": "Device 192.168.1.99 0130:1 not found" // エラーメッセージ
    }
  },
  "requestId": "req-123" // 対応するリクエストID
}
```

- `success`: 操作が成功したかどうか（boolean）
- `data`: 成功時の追加データ（JSON形式、内容はリクエストによる）。`get_property_description` の場合は `PropertyDescriptionData` オブジェクト（詳細は **[クライアント UI 開発ガイド](./client_ui_development_guide.md)** を参照）。他のリクエストではデバイス情報やグループ情報、または `null`。
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
        // 応答の payload をコールバックに渡す
        callback(message.payload);
        pendingRequests.delete(message.requestId);
      }
    } else {
      // サーバーからの通知 (requestId がない)
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

    case "device_deleted":
      console.log("Device deleted:", payload.ip, payload.eoj);
      // デバイスリストから削除
      const deletedDeviceId = `${payload.ip} ${payload.eoj}`;
      delete devices[deletedDeviceId];
      break;
      
    // 他の通知タイプも同様に処理...
    case "property_changed":
      const propValue = payload.value; // { EDT?: string, string?: string, number?: number }
      const valueStr = propValue.string ?? (propValue.number !== undefined ? propValue.number.toString() : atob(propValue.EDT ?? ""));
      console.log(`Property ${payload.epc} changed for ${payload.ip} ${payload.eoj} to ${valueStr}`);
      // 対応するデバイスのプロパティを更新
      const targetDeviceId = `${payload.ip} ${payload.eoj}`;
      if (devices[targetDeviceId]) {
        // 新しい形式の value オブジェクト全体を保存
        devices[targetDeviceId].properties[payload.epc] = propValue;
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
      
    case "log_notification":
      // ログレベルに応じた処理
      if (payload.level === "ERROR") {
        console.error(`[Server ${payload.level}] ${payload.message}`, payload.attributes);
      } else if (payload.level === "WARN") {
        console.warn(`[Server ${payload.level}] ${payload.message}`, payload.attributes);
      }
      // UI表示やログ保存などの追加処理
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
async function setDeviceProperties(targetDevice: string, properties: Record<string, { EDT?: string; string?: string; number?: number }>) {
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

// デバイス削除
async function deleteDevice(targetDevice: string) {
  try {
    const payload = { target: targetDevice };
    const resultData = await sendRequest("delete_device", payload);
    console.log(`Deleted device ${targetDevice}:`, resultData);
    return resultData;
  } catch (error) {
    console.error(`Failed to delete device ${targetDevice}:`, error);
    throw error;
  }
}

// プロパティ詳細情報取得
async function getPropertyDescription(classCode: string, lang?: string) {
  try {
    const payload = { classCode: classCode };
    if (lang) {
      payload.lang = lang;
    }
    const resultData = await sendRequest("get_property_description", payload);
    console.log(`Property description for class ${classCode} (${lang || 'en'}):`, resultData);
    // resultData は PropertyDescriptionData オブジェクト
    return resultData;
  } catch (error) {
    console.error(`Failed to get property description for class ${classCode}:`, error);
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
// getPropertyDescription("0130"); // エアコンのプロパティ詳細を取得（英語）
// getPropertyDescription("0130", "ja"); // エアコンのプロパティ詳細を取得（日本語）
// getDeviceProperties("192.168.1.10 0130:1", ["80", "B0"]);
// deleteDevice("192.168.1.10 0130:1"); // デバイス削除
// deleteDevice("192.168.1.10 0ef0:1"); // NodeProfile削除（同一IPのすべてのデバイスが削除される）
// addGroup("@living_room", ["013001:00000B:ABCDEF0123456789ABCDEF012345", "029001:000005:FEDCBA9876543210FEDCBA987654"]); // 例
```

このコード例は概念的なものであり、実際の実装では言語やフレームワークに応じた適切なエラーハンドリングやタイプセーフな実装が必要です。
