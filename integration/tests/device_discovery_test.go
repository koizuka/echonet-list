//go:build integration

package tests

import (
	"echonet-list/integration/helpers"
	"path/filepath"
	"testing"
	"time"
)

func TestDeviceDiscovery(t *testing.T) {
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

	// payloadからデバイス情報を取得
	payload, ok := response["payload"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "initial_state payloadの形式確認")

	devicesMap, ok := payload["devices"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "デバイス一覧の形式確認")
	helpers.AssertTrue(t, len(devicesMap) > 0, "デバイスが発見されていることを確認")

	t.Logf("発見されたデバイス数: %d", len(devicesMap))

	// テストフィクスチャから期待される最小デバイス数を確認
	// test-devices.jsonには5つのデバイスが含まれている（デバイス3つ + Node Profile 2つ）
	helpers.AssertTrue(t, len(devicesMap) >= 3, "期待されるデバイス数の確認")
}

func TestDeviceAliases(t *testing.T) {
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

	// initial_state メッセージを受信してエイリアス情報を取得
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "initial_state"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "initial_state メッセージの受信")

	// payloadからエイリアス情報を取得
	payload, ok := response["payload"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "initial_state payloadの形式確認")

	aliases, ok := payload["aliases"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "エイリアス一覧の形式確認")
	helpers.AssertTrue(t, len(aliases) > 0, "エイリアスが存在することを確認")

	t.Logf("エイリアス数: %d", len(aliases))

	// テストフィクスチャに含まれるエイリアスの確認
	expectedAliases := []string{"Test Air Conditioner", "Test Light 1", "Test Light 2"}
	for _, expectedAlias := range expectedAliases {
		_, exists := aliases[expectedAlias]
		helpers.AssertTrue(t, exists, "期待されるエイリアス"+expectedAlias+"の存在確認")
	}
}

func TestDeviceGroups(t *testing.T) {
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

	// initial_state メッセージを受信してグループ情報を取得
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "initial_state"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "initial_state メッセージの受信")

	// payloadからグループ情報を取得
	payload, ok := response["payload"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "initial_state payloadの形式確認")

	groups, ok := payload["groups"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "グループ一覧の形式確認")
	helpers.AssertTrue(t, len(groups) > 0, "グループが存在することを確認")

	t.Logf("グループ数: %d", len(groups))

	// テストフィクスチャに含まれるグループの確認
	expectedGroups := []string{"@Test Lights", "@All Test Devices"}
	for _, expectedGroup := range expectedGroups {
		_, exists := groups[expectedGroup]
		helpers.AssertTrue(t, exists, "期待されるグループ"+expectedGroup+"の存在確認")
	}
}

func TestPropertyOperations(t *testing.T) {
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

	// initial_state メッセージからプロパティ情報を確認
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "initial_state"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "initial_state メッセージの受信")

	// payloadからデバイス情報とプロパティを確認
	payload, ok := response["payload"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "initial_state payloadの形式確認")

	devicesMap, ok := payload["devices"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "デバイス一覧の形式確認")

	// テストデバイスのプロパティを確認
	deviceKey := "192.168.1.100 0130:1"
	device, ok := devicesMap[deviceKey].(map[string]interface{})
	helpers.AssertTrue(t, ok, "テストデバイスの存在確認")

	properties, ok := device["properties"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "プロパティの存在確認")

	// EPC 80（動作状態）のプロパティを確認
	property80, ok := properties["80"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "プロパティ80の存在確認")
	helpers.AssertNotEqual(t, nil, property80, "プロパティ値がnilでないことを確認")

	t.Logf("取得したプロパティ値: %v", property80)
}
