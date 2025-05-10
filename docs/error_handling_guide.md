# ECHONET Lite WebSocket クライアント エラーハンドリングガイド

## 1. WebSocket接続のエラーハンドリング

### 1.1 接続エラー

WebSocket接続時に発生する可能性のあるエラーとその対処方法：

- **接続拒否（Connection Refused）**
  - 原因：サーバーが起動していない、ポートが閉じている、ファイアウォールによるブロック
  - 対処：サーバーの状態を確認し、必要に応じて再接続を試みる
  - 推奨実装：指数バックオフを使用した再接続（例：1秒、2秒、4秒、8秒...）

- **TLS証明書エラー**
  - 原因：自己署名証明書やmkcertで生成した証明書が信頼されていない
  - 対処：開発環境では証明書の信頼設定が必要
  - 推奨実装：エラーメッセージに証明書の信頼設定手順を表示

### 1.2 接続切断

WebSocket接続が切断された場合の対処：

- **予期せぬ切断**
  - 原因：ネットワーク不安定、サーバー再起動、タイムアウト
  - 対処：自動再接続の実装
  - 推奨実装：

    ```javascript
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 5;
    const baseDelay = 1000; // 1秒

    function reconnect() {
      if (reconnectAttempts >= maxReconnectAttempts) {
        console.error('最大再接続回数に達しました');
        return;
      }

      const delay = Math.min(baseDelay * Math.pow(2, reconnectAttempts), 30000);
      setTimeout(() => {
        console.log(`再接続を試みます（${reconnectAttempts + 1}回目）`);
        connectWebSocket();
        reconnectAttempts++;
      }, delay);
    }
    ```

- **意図的な切断**
  - 原因：ユーザーによる切断、アプリケーション終了
  - 対処：クリーンアップ処理の実装
  - 推奨実装：

    ```javascript
    function cleanup() {
      // 未送信のリクエストをキャンセル
      // 進行中の操作を中断
      // リソースを解放
      websocket.close();
    }
    ```

## 2. アプリケーションレベルのエラーハンドリング

### 2.1 デバイス通信エラー

- **タイムアウト（ECHONET_TIMEOUT）**
  - 原因：デバイスからの応答がない
  - 対処：ユーザーに通知し、必要に応じて再試行
  - 推奨実装：

    ```javascript
    function handleTimeout(deviceId, operation) {
      showNotification(`デバイス ${deviceId} の ${operation} がタイムアウトしました`);
      // 自動再試行の場合は、一定時間後に再実行
    }
    ```

- **デバイスエラー（ECHONET_DEVICE_ERROR）**
  - 原因：デバイスからのエラー応答
  - 対処：エラーコードに応じた適切なメッセージを表示
  - 推奨実装：エラーコードとメッセージのマッピングテーブルを用意

### 2.2 データ整合性エラー

- **プロパティ値の不整合**
  - 原因：デバイスの状態とUIの表示が一致しない
  - 対処：定期的な状態同期、ユーザーへの通知
  - 推奨実装：

    ```javascript
    function syncDeviceState(deviceId) {
      // デバイスの全プロパティを再取得
      // UIを更新
      // 不整合がある場合はユーザーに通知
    }
    ```

## 3. エラーログとデバッグ

### 3.1 ログ記録

- **エラーログの構造**

  ```javascript
  const errorLog = {
    timestamp: new Date().toISOString(),
    type: 'ERROR_TYPE',
    deviceId: 'DEVICE_ID',
    operation: 'OPERATION',
    details: {
      // エラーの詳細情報
    }
  };
  ```

- **ログレベル**
  - ERROR: アプリケーションの動作に影響する重大なエラー
  - WARN: 一時的な問題や警告
  - INFO: 通常の操作ログ
  - DEBUG: デバッグ情報

### 3.2 デバッグモード

- **デバッグ情報の表示**
  - WebSocket通信の生データ
  - デバイスの状態変化
  - エラーの詳細情報

- **デバッグモードの切り替え**

  ```javascript
  const DEBUG_MODE = localStorage.getItem('debug_mode') === 'true';

  function logDebug(message, data) {
    if (DEBUG_MODE) {
      console.log(`[DEBUG] ${message}`, data);
    }
  }
  ```

## 4. エラー通知とユーザーインターフェース

### 4.1 エラー通知の設計

- **通知の種類**
  - トースト通知：一時的なエラー
  - モーダルダイアログ：重要なエラー
  - インライン表示：フォームのバリデーションエラー

- **通知の優先度**
  1. 重大なエラー（操作不能）
  2. 警告（機能制限）
  3. 情報（一時的な問題）

### 4.2 エラー回復のUI

- **再試行ボタン**
  - 失敗した操作の再実行
  - 自動再試行の有効/無効切り替え

- **状態リセット**
  - デバイスの状態を初期値に戻す
  - 接続を再確立

## 5. ベストプラクティス

1. **エラーの予測と防止**
   - 入力値のバリデーション
   - 操作の前確認
   - 状態の整合性チェック

2. **ユーザーフレンドリーなエラーメッセージ**
   - 技術的な詳細は最小限に
   - 具体的な対処方法を提示
   - 多言語対応

3. **エラー発生時の状態保持**
   - ユーザーの入力データを保持
   - 操作の履歴を記録
   - 自動保存の実装

4. **エラー監視と分析**
   - エラーの発生頻度の追跡
   - パターンの分析
   - 改善点の特定
