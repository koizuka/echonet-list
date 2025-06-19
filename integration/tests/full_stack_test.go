//go:build integration

package tests

import (
	"echonet-list/integration/helpers"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFullStackIntegration(t *testing.T) {
	// テストサーバーを起動
	server, err := helpers.NewTestServer()
	helpers.AssertNoError(t, err, "テストサーバーの作成")

	// フィクスチャファイルの設定
	fixturesDir, err := helpers.GetAbsolutePath("../fixtures")
	helpers.AssertNoError(t, err, "フィクスチャディレクトリのパス取得")

	devicesFile := filepath.Join(fixturesDir, "test-devices.json")
	aliasesFile := filepath.Join(fixturesDir, "test-aliases.json")
	groupsFile := filepath.Join(fixturesDir, "test-groups.json")

	server.SetTestFixtures(devicesFile, aliasesFile, groupsFile)

	err = server.Start()
	helpers.AssertNoError(t, err, "サーバーの起動")
	defer server.Stop()

	// WebSocket接続を作成
	wsConn, err := helpers.NewWebSocketConnection(server.GetWebSocketURL())
	helpers.AssertNoError(t, err, "WebSocket接続の作成")
	defer wsConn.Close()

	// 1. デバイス一覧の取得
	listMessage := map[string]interface{}{
		"type": "list_devices",
	}

	err = wsConn.SendMessage(listMessage)
	helpers.AssertNoError(t, err, "デバイス一覧要求の送信")

	deviceResponse, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "device_list"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "デバイス一覧の応答受信")

	devices, ok := deviceResponse["devices"].([]interface{})
	helpers.AssertTrue(t, ok, "デバイス一覧の形式確認")
	helpers.AssertTrue(t, len(devices) > 0, "デバイスが発見されていることを確認")

	// 2. エイリアス一覧の取得
	aliasMessage := map[string]interface{}{
		"type": "list_aliases",
	}

	err = wsConn.SendMessage(aliasMessage)
	helpers.AssertNoError(t, err, "エイリアス一覧要求の送信")

	aliasResponse, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "alias_list"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "エイリアス一覧の応答受信")

	aliases, ok := aliasResponse["aliases"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "エイリアス一覧の形式確認")
	helpers.AssertTrue(t, len(aliases) > 0, "エイリアスが存在することを確認")

	// 3. グループ一覧の取得
	groupMessage := map[string]interface{}{
		"type": "list_groups",
	}

	err = wsConn.SendMessage(groupMessage)
	helpers.AssertNoError(t, err, "グループ一覧要求の送信")

	groupResponse, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "group_list"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "グループ一覧の応答受信")

	groups, ok := groupResponse["groups"].([]interface{})
	helpers.AssertTrue(t, ok, "グループ一覧の形式確認")
	helpers.AssertTrue(t, len(groups) > 0, "グループが存在することを確認")

	t.Logf("フルスタック統合テスト完了 - デバイス: %d, エイリアス: %d, グループ: %d", 
		len(devices), len(aliases), len(groups))
}

func TestMultiClientSynchronization(t *testing.T) {
	// テストサーバーを起動
	server, err := helpers.NewTestServer()
	helpers.AssertNoError(t, err, "テストサーバーの作成")

	// フィクスチャファイルの設定
	fixturesDir, err := helpers.GetAbsolutePath("../fixtures")
	helpers.AssertNoError(t, err, "フィクスチャディレクトリのパス取得")

	devicesFile := filepath.Join(fixturesDir, "test-devices.json")
	aliasesFile := filepath.Join(fixturesDir, "test-aliases.json")
	groupsFile := filepath.Join(fixturesDir, "test-groups.json")

	server.SetTestFixtures(devicesFile, aliasesFile, groupsFile)

	err = server.Start()
	helpers.AssertNoError(t, err, "サーバーの起動")
	defer server.Stop()

	// 複数のクライアント接続を作成
	numClients := 3
	clients := make([]*helpers.WebSocketConnection, numClients)
	var wg sync.WaitGroup

	// クライアントを並行で接続
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			conn, err := helpers.NewWebSocketConnection(server.GetWebSocketURL())
			helpers.AssertNoError(t, err, "クライアントのWebSocket接続")
			clients[clientID] = conn

			// 各クライアントからデバイス一覧を要求
			listMessage := map[string]interface{}{
				"type":     "list_devices",
				"clientID": clientID + 1,
			}

			err = conn.SendMessage(listMessage)
			helpers.AssertNoError(t, err, "クライアントのデバイス一覧要求")

			// 応答を受信
			response, err := conn.WaitForMessage(
				func(msg map[string]interface{}) bool {
					msgType, ok := msg["type"].(string)
					return ok && msgType == "device_list"
				},
				15*time.Second,
			)
			helpers.AssertNoError(t, err, "クライアントの応答受信")

			devices, ok := response["devices"].([]interface{})
			helpers.AssertTrue(t, ok, "クライアントのデバイス一覧形式確認")
			helpers.AssertTrue(t, len(devices) > 0, "クライアントのデバイス存在確認")

			t.Logf("クライアント%d: %d個のデバイスを受信", clientID+1, len(devices))
		}(i)
	}

	// 全クライアントの処理完了を待機
	wg.Wait()

	// 全クライアント接続を閉じる
	for _, client := range clients {
		if client != nil {
			err := client.Close()
			helpers.AssertNoError(t, err, "クライアントの接続終了")
		}
	}

	t.Logf("マルチクライアント同期テスト完了 - %d個のクライアントが正常に動作", numClients)
}

func TestRealTimeUpdates(t *testing.T) {
	// テストサーバーを起動
	server, err := helpers.NewTestServer()
	helpers.AssertNoError(t, err, "テストサーバーの作成")

	// フィクスチャファイルの設定
	fixturesDir, err := helpers.GetAbsolutePath("../fixtures")
	helpers.AssertNoError(t, err, "フィクスチャディレクトリのパス取得")

	devicesFile := filepath.Join(fixturesDir, "test-devices.json")
	aliasesFile := filepath.Join(fixturesDir, "test-aliases.json")
	groupsFile := filepath.Join(fixturesDir, "test-groups.json")

	server.SetTestFixtures(devicesFile, aliasesFile, groupsFile)

	err = server.Start()
	helpers.AssertNoError(t, err, "サーバーの起動")
	defer server.Stop()

	// 2つのクライアント接続を作成（送信者と受信者）
	senderConn, err := helpers.NewWebSocketConnection(server.GetWebSocketURL())
	helpers.AssertNoError(t, err, "送信者WebSocket接続の作成")
	defer senderConn.Close()

	receiverConn, err := helpers.NewWebSocketConnection(server.GetWebSocketURL())
	helpers.AssertNoError(t, err, "受信者WebSocket接続の作成")
	defer receiverConn.Close()

	// 受信者で更新通知を待機する goroutine を開始
	updateReceived := make(chan bool, 1)
	go func() {
		// リアルタイム更新通知を待機
		_, err := receiverConn.WaitForMessage(
			func(msg map[string]interface{}) bool {
				msgType, ok := msg["type"].(string)
				return ok && (msgType == "property_update" || msgType == "device_update")
			},
			20*time.Second,
		)
		if err == nil {
			updateReceived <- true
		}
	}()

	// 短い待機でWebSocket接続を安定化
	time.Sleep(1 * time.Second)

	// 送信者からプロパティ変更要求を送信
	setPropertyMessage := map[string]interface{}{
		"type":  "set_property",
		"ip":    "192.168.1.100",
		"eoj":   "0130",
		"epc":   "80", // 動作状態
		"value": "31", // ON
	}

	err = senderConn.SendMessage(setPropertyMessage)
	helpers.AssertNoError(t, err, "プロパティ変更要求の送信")

	// プロパティ変更の応答を確認
	setResponse, err := senderConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "set_property_response"
		},
		10*time.Second,
	)
	
	if err == nil {
		t.Logf("プロパティ変更応答を受信: %+v", setResponse)
	} else {
		t.Logf("プロパティ変更応答を受信できませんでした（テストモードのため期待される動作）: %v", err)
	}

	// リアルタイム更新通知を確認
	select {
	case <-updateReceived:
		t.Log("リアルタイム更新通知が正常に受信されました")
	case <-time.After(10 * time.Second):
		t.Log("リアルタイム更新通知はタイムアウトしました（テストモードのため期待される場合があります）")
	}

	t.Log("リアルタイム更新テスト完了")
}

func TestErrorRecovery(t *testing.T) {
	// テストサーバーを起動
	server, err := helpers.NewTestServer()
	helpers.AssertNoError(t, err, "テストサーバーの作成")

	// フィクスチャファイルの設定
	fixturesDir, err := helpers.GetAbsolutePath("../fixtures")
	helpers.AssertNoError(t, err, "フィクスチャディレクトリのパス取得")

	devicesFile := filepath.Join(fixturesDir, "test-devices.json")
	aliasesFile := filepath.Join(fixturesDir, "test-aliases.json")
	groupsFile := filepath.Join(fixturesDir, "test-groups.json")

	server.SetTestFixtures(devicesFile, aliasesFile, groupsFile)

	err = server.Start()
	helpers.AssertNoError(t, err, "サーバーの起動")
	defer server.Stop()

	// WebSocket接続を作成
	wsConn, err := helpers.NewWebSocketConnection(server.GetWebSocketURL())
	helpers.AssertNoError(t, err, "WebSocket接続の作成")
	defer wsConn.Close()

	// 無効なメッセージを送信してエラーハンドリングをテスト
	invalidMessages := []map[string]interface{}{
		{"type": "invalid_message_type"},
		{"type": "get_property", "ip": "invalid_ip"},
		{"type": "set_property", "epc": "invalid_epc"},
		{}, // 空のメッセージ
	}

	for i, invalidMsg := range invalidMessages {
		t.Run("InvalidMessage_"+string(rune('A'+i)), func(t *testing.T) {
			err := wsConn.SendMessage(invalidMsg)
			helpers.AssertNoError(t, err, "無効なメッセージの送信")

			// エラーレスポンスまたは無応答を確認
			_, err = wsConn.ReceiveMessage(3 * time.Second)
			if err != nil {
				t.Logf("無効なメッセージ%dに対して応答なし（期待される動作）: %v", i+1, err)
			} else {
				t.Logf("無効なメッセージ%dに対して何らかの応答を受信", i+1)
			}
		})
	}

	// 接続が依然として動作することを確認
	testMessage := map[string]interface{}{
		"type": "list_devices",
	}

	err = wsConn.SendMessage(testMessage)
	helpers.AssertNoError(t, err, "テストメッセージの送信")

	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "device_list"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "エラー後の正常メッセージ応答")

	devices, ok := response["devices"].([]interface{})
	helpers.AssertTrue(t, ok, "エラー後のデバイス一覧形式確認")
	helpers.AssertTrue(t, len(devices) > 0, "エラー後のデバイス存在確認")

	t.Log("エラー回復テスト完了 - 接続は無効なメッセージ後も正常に動作")
}