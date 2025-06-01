import WebSocket from 'ws';
import https from 'https';

// SSL証明書の検証を無効にする (開発環境用)
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';

const WS_URL = 'wss://localhost:8080/ws';

console.log('WebSocket接続テストを開始します...');
console.log('接続先:', WS_URL);

// WebSocketクライアントを作成
const ws = new WebSocket(WS_URL, {
  // SSL証明書の検証をスキップ
  rejectUnauthorized: false,
  // カスタムエージェントを使用
  agent: new https.Agent({
    rejectUnauthorized: false
  })
});

// タイムアウト設定
const TIMEOUT = 10000; // 10秒
const timeoutId = setTimeout(() => {
  console.error('❌ 接続タイムアウト');
  ws.close();
  process.exit(1);
}, TIMEOUT);

// 接続成功
ws.on('open', () => {
  clearTimeout(timeoutId);
  console.log('✅ WebSocket接続成功');
  
  // 5秒後に接続を閉じる
  setTimeout(() => {
    console.log('🔄 接続を閉じます...');
    ws.close(1000, 'Test completed');
  }, 5000);
});

// メッセージ受信
ws.on('message', (data) => {
  try {
    const message = JSON.parse(data.toString());
    console.log('📨 受信メッセージ:', JSON.stringify(message, null, 2));
    
    // initial_state メッセージの場合、デバイス数を表示
    if (message.type === 'initial_state') {
      const deviceCount = Object.keys(message.payload.devices || {}).length;
      const aliasCount = Object.keys(message.payload.aliases || {}).length;
      const groupCount = Object.keys(message.payload.groups || {}).length;
      
      console.log(`📊 初期状態受信: デバイス${deviceCount}個, エイリアス${aliasCount}個, グループ${groupCount}個`);
    }
  } catch (error) {
    console.log('📨 受信データ (JSON以外):', data.toString());
  }
});

// エラーハンドリング
ws.on('error', (error) => {
  clearTimeout(timeoutId);
  console.error('❌ WebSocketエラー:', error.message);
  
  // より詳細なエラー情報
  if (error.code) {
    console.error('エラーコード:', error.code);
  }
  if (error.errno) {
    console.error('errno:', error.errno);
  }
  if (error.syscall) {
    console.error('syscall:', error.syscall);
  }
});

// 接続終了
ws.on('close', (code, reason) => {
  clearTimeout(timeoutId);
  console.log(`🔌 接続終了: コード=${code}, 理由="${reason}"`);
  
  // 終了コードの説明
  const codeDescriptions = {
    1000: 'Normal Closure - 正常終了',
    1001: 'Going Away - サーバーまたはクライアントが離脱',
    1002: 'Protocol Error - プロトコルエラー',
    1003: 'Unsupported Data - サポートされていないデータ',
    1006: 'Abnormal Closure - 異常終了 (通常はネットワークエラー)',
    1007: 'Invalid frame payload data - 無効なペイロード',
    1008: 'Policy Violation - ポリシー違反',
    1009: 'Message Too Big - メッセージが大きすぎる',
    1011: 'Internal Server Error - サーバー内部エラー'
  };
  
  const description = codeDescriptions[code] || '不明な終了コード';
  console.log(`終了コードの説明: ${description}`);
  
  if (code === 1000) {
    console.log('✅ テスト完了');
    process.exit(0);
  } else {
    console.log('❌ テスト失敗');
    process.exit(1);
  }
});

// プロセス終了時の処理
process.on('SIGINT', () => {
  console.log('\n🛑 テストを中断します...');
  clearTimeout(timeoutId);
  ws.close();
  process.exit(0);
});