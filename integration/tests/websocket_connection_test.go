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

	// 接続成功を確認するため、initial_state メッセージを待機
	// WebSocket接続すると自動的に initial_state が送信される
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "initial_state"
		},
		5*time.Second,
	)
	helpers.AssertNoError(t, err, "initial_state メッセージの受信")

	t.Logf("受信したinitial_state: %+v", response)

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

	// 全ての接続から initial_state メッセージを受信
	for i, conn := range connections {
		response, err := conn.WaitForMessage(
			func(msg map[string]interface{}) bool {
				msgType, ok := msg["type"].(string)
				return ok && msgType == "initial_state"
			},
			5*time.Second,
		)
		helpers.AssertNoError(t, err, "クライアントのinitial_state受信")

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

	// initial_state メッセージを受信してデバイス情報を取得
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "initial_state"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "initial_state メッセージの受信")

	t.Logf("initial_state: %+v", response)

	// 応答にデバイス情報が含まれていることを確認
	if payload, ok := response["payload"]; ok {
		if payloadMap, ok := payload.(map[string]interface{}); ok {
			if devices, ok := payloadMap["devices"]; ok {
				if devicesMap, ok := devices.(map[string]interface{}); ok {
					t.Logf("デバイス数: %d", len(devicesMap))
				} else {
					t.Error("デバイス情報の形式が正しくありません")
				}
			} else {
				t.Error("payloadにデバイス情報が含まれていません")
			}
		} else {
			t.Error("payloadの形式が正しくありません")
		}
	} else {
		t.Error("応答にpayloadが含まれていません")
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

	// まず initial_state を受信して、その後存在しないメッセージタイプを待機してタイムアウトを確認
	_, err = wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "initial_state"
		},
		5*time.Second,
	)
	helpers.AssertNoError(t, err, "initial_state の受信")

	// 次に存在しないメッセージタイプを待機してタイムアウトを確認
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
