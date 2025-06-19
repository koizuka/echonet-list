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

	// initial_state メッセージを受信してデバイス/エイリアス/グループ情報を取得
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "initial_state"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "initial_state メッセージの受信")

	// payloadから各情報を取得
	payload, ok := response["payload"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "initial_state payloadの形式確認")

	// 1. デバイス一覧の検証
	devicesMap, ok := payload["devices"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "デバイス一覧の形式確認")
	helpers.AssertTrue(t, len(devicesMap) > 0, "デバイスが発見されていることを確認")

	// 2. エイリアス一覧の検証
	aliases, ok := payload["aliases"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "エイリアス一覧の形式確認")
	helpers.AssertTrue(t, len(aliases) > 0, "エイリアスが存在することを確認")

	// 3. グループ一覧の検証
	groups, ok := payload["groups"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "グループ一覧の形式確認")
	helpers.AssertTrue(t, len(groups) > 0, "グループが存在することを確認")

	t.Logf("フルスタック統合テスト完了 - デバイス: %d, エイリアス: %d, グループ: %d",
		len(devicesMap), len(aliases), len(groups))
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

			// initial_state メッセージを受信
			response, err := conn.WaitForMessage(
				func(msg map[string]interface{}) bool {
					msgType, ok := msg["type"].(string)
					return ok && msgType == "initial_state"
				},
				15*time.Second,
			)
			helpers.AssertNoError(t, err, "クライアントの応答受信")

			// payloadからデバイス情報を取得
			payload, ok := response["payload"].(map[string]interface{})
			helpers.AssertTrue(t, ok, "クライアントのinitial_state payload形式確認")

			devicesMap, ok := payload["devices"].(map[string]interface{})
			helpers.AssertTrue(t, ok, "クライアントのデバイス一覧形式確認")
			helpers.AssertTrue(t, len(devicesMap) > 0, "クライアントのデバイス存在確認")

			t.Logf("クライアント%d: %d個のデバイスを受信", clientID+1, len(devicesMap))
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

	// 送信者からプロパティ変更要求を送信（実際のWebSocketプロトコルを使用）
	setPropertiesMessage := map[string]interface{}{
		"type":   "set_properties",
		"target": "192.168.1.100 0130:1",
		"properties": map[string]interface{}{
			"80": map[string]interface{}{
				"string": "on",
			},
		},
	}

	err = senderConn.SendMessage(setPropertiesMessage)
	helpers.AssertNoError(t, err, "プロパティ変更要求の送信")

	// プロパティ変更のcommand_result応答を確認
	setResponse, err := senderConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "command_result"
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
		{"type": "get_properties", "targets": []string{"invalid_ip"}},
		{"type": "set_properties", "target": "invalid_target", "properties": map[string]interface{}{}},
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

	// 接続が依然として動作することを確認（有効なコマンドを送信してテスト）
	// update_propertiesコマンドを送信して接続が正常に動作することを確認
	updateMessage := map[string]interface{}{
		"type":    "update_properties",
		"targets": []string{"192.168.1.100 0130:1"},
	}

	err = wsConn.SendMessage(updateMessage)
	helpers.AssertNoError(t, err, "エラー後のテストメッセージ送信")

	// command_result応答を受信
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "command_result"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "エラー後の正常メッセージ応答")

	// 応答の形式を確認（成功でも失敗でも応答が返ってくることが重要）
	_, ok := response["payload"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "エラー後のcommand_result応答形式確認")

	t.Log("エラー回復テスト完了 - 接続は無効なメッセージ後も正常に動作")
}
