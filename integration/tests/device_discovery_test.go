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

	// 応答にデバイス情報が含まれていることを確認
	devices, ok := response["devices"].([]interface{})
	helpers.AssertTrue(t, ok, "デバイス一覧の形式確認")
	helpers.AssertTrue(t, len(devices) > 0, "デバイスが発見されていることを確認")

	t.Logf("発見されたデバイス数: %d", len(devices))

	// テストフィクスチャから期待される最小デバイス数を確認
	// test-devices.jsonには3つのデバイスが含まれている
	helpers.AssertTrue(t, len(devices) >= 3, "期待されるデバイス数の確認")
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

	// エイリアス一覧を要求
	aliasMessage := map[string]interface{}{
		"type": "list_aliases",
	}

	err = wsConn.SendMessage(aliasMessage)
	helpers.AssertNoError(t, err, "エイリアス一覧要求の送信")

	// エイリアス一覧の応答を待機
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "alias_list"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "エイリアス一覧の応答受信")

	// 応答にエイリアス情報が含まれていることを確認
	aliases, ok := response["aliases"].(map[string]interface{})
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

	// グループ一覧を要求
	groupMessage := map[string]interface{}{
		"type": "list_groups",
	}

	err = wsConn.SendMessage(groupMessage)
	helpers.AssertNoError(t, err, "グループ一覧要求の送信")

	// グループ一覧の応答を待機
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "group_list"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "グループ一覧の応答受信")

	// 応答にグループ情報が含まれていることを確認
	groups, ok := response["groups"].([]interface{})
	helpers.AssertTrue(t, ok, "グループ一覧の形式確認")
	helpers.AssertTrue(t, len(groups) > 0, "グループが存在することを確認")

	t.Logf("グループ数: %d", len(groups))

	// テストフィクスチャに含まれるグループの確認
	expectedGroups := []string{"@Test Lights", "@All Test Devices"}
	groupNames := make([]string, 0, len(groups))

	for _, group := range groups {
		if groupMap, ok := group.(map[string]interface{}); ok {
			if groupName, ok := groupMap["group"].(string); ok {
				groupNames = append(groupNames, groupName)
			}
		}
	}

	for _, expectedGroup := range expectedGroups {
		found := false
		for _, groupName := range groupNames {
			if groupName == expectedGroup {
				found = true
				break
			}
		}
		helpers.AssertTrue(t, found, "期待されるグループ"+expectedGroup+"の存在確認")
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

	// プロパティ読み取りテスト
	getPropertyMessage := map[string]interface{}{
		"type": "get_property",
		"ip":   "192.168.1.100",
		"eoj":  "0130",
		"epc":  "80", // 動作状態
	}

	err = wsConn.SendMessage(getPropertyMessage)
	helpers.AssertNoError(t, err, "プロパティ取得要求の送信")

	// プロパティ応答を待機
	response, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "property_response"
		},
		10*time.Second,
	)
	helpers.AssertNoError(t, err, "プロパティ応答の受信")

	// 応答にプロパティ値が含まれていることを確認
	value, ok := response["value"]
	helpers.AssertTrue(t, ok, "プロパティ値の存在確認")
	helpers.AssertNotEqual(t, nil, value, "プロパティ値がnilでないことを確認")

	t.Logf("取得したプロパティ値: %v", value)
}
