//go:build integration

package tests

import (
	"echonet-list/integration/helpers"
	"path/filepath"
	"testing"
	"time"
)

func TestWebSocketConnection(t *testing.T) {
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

	// 接続成功を確認するため、メッセージを送信
	testMessage := map[string]interface{}{
		"type": "ping",
	}

	err = wsConn.SendMessage(testMessage)
	helpers.AssertNoError(t, err, "メッセージの送信")

	// 応答を受信
	response, err := wsConn.ReceiveMessage(5 * time.Second)
	helpers.AssertNoError(t, err, "応答の受信")

	t.Logf("受信したメッセージ: %+v", response)

	// WebSocket接続を閉じる
	err = wsConn.Close()
	helpers.AssertNoError(t, err, "WebSocket接続の切断")
}

func TestMultipleWebSocketConnections(t *testing.T) {
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

	// 複数のWebSocket接続を作成
	numConnections := 3
	connections := make([]*helpers.WebSocketConnection, numConnections)

	for i := 0; i < numConnections; i++ {
		conn, err := helpers.NewWebSocketConnection(server.GetWebSocketURL())
		helpers.AssertNoError(t, err, "WebSocket接続の作成")
		connections[i] = conn
	}

	// 全ての接続を使用してメッセージを送信
	for i, conn := range connections {
		testMessage := map[string]interface{}{
			"type":   "ping",
			"client": i + 1,
		}

		err = conn.SendMessage(testMessage)
		helpers.AssertNoError(t, err, "クライアントのメッセージ送信")
	}

	// 全ての接続から応答を受信
	for i, conn := range connections {
		response, err := conn.ReceiveMessage(5 * time.Second)
		helpers.AssertNoError(t, err, "クライアントの応答受信")

		t.Logf("クライアント%d 受信メッセージ: %+v", i+1, response)
	}

	// 全ての接続を閉じる
	for _, conn := range connections {
		err = conn.Close()
		helpers.AssertNoError(t, err, "WebSocket接続の切断")
	}
}

func TestWebSocketMessageExchange(t *testing.T) {
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

	// デバイス一覧を要求
	listMessage := map[string]interface{}{
		"type": "list_devices",
	}

	err = wsConn.SendMessage(listMessage)
	helpers.AssertNoError(t, err, "デバイス一覧要求の送信")

	// デバイス一覧の応答を待機
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "device_list"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "デバイス一覧の応答受信")

	t.Logf("デバイス一覧: %+v", response)

	// 応答にデバイス情報が含まれていることを確認
	if devices, ok := response["devices"]; ok {
		t.Logf("デバイス数: %d", len(devices.([]interface{})))
	} else {
		t.Error("応答にデバイス情報が含まれていません")
	}
}

func TestWebSocketConnectionTimeout(t *testing.T) {
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

	// 存在しないメッセージタイプを待機してタイムアウトを確認
	_, err = wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "nonexistent_message_type"
		},
		1*time.Second, // 短いタイムアウト
	)

	// エラーが発生することを確認（タイムアウトエラー）
	helpers.AssertError(t, err, "タイムアウトエラーの確認")
	t.Logf("期待されたタイムアウトエラー: %v", err)
}