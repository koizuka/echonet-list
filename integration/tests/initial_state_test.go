//go:build integration

package tests

import (
	"echonet-list/integration/helpers"
	"path/filepath"
	"testing"
	"time"
)

func TestInitialStateMessage(t *testing.T) {
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

	// initial_stateメッセージを待機
	initialStateMsg, err := wsConn.WaitForMessage(
		func(msg map[string]interface{}) bool {
			msgType, ok := msg["type"].(string)
			return ok && msgType == "initial_state"
		},
		15*time.Second,
	)
	helpers.AssertNoError(t, err, "initial_stateメッセージの受信")

	t.Logf("受信したinitial_stateメッセージ: %+v", initialStateMsg)

	// ペイロードを確認
	payload, ok := initialStateMsg["payload"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "payloadの存在確認")

	// デバイスの確認
	devices, ok := payload["devices"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "devicesの形式確認")
	t.Logf("デバイス数: %d", len(devices))

	// フィクスチャファイルには2つのIPアドレスに3つのデバイスがあるはず
	if len(devices) == 0 {
		t.Error("デバイスが空です - フィクスチャファイルの読み込みに失敗している可能性があります")

		// デバッグ情報を出力
		for key, value := range devices {
			t.Logf("デバイスキー: %s, 値: %+v", key, value)
		}
	}

	// エイリアスの確認
	aliases, ok := payload["aliases"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "aliasesの形式確認")
	t.Logf("エイリアス数: %d", len(aliases))

	if len(aliases) == 0 {
		t.Error("エイリアスが空です - フィクスチャファイルの読み込みに失敗している可能性があります")
	}

	// グループの確認
	groups, ok := payload["groups"].(map[string]interface{})
	helpers.AssertTrue(t, ok, "groupsの形式確認")
	t.Logf("グループ数: %d", len(groups))

	if len(groups) == 0 {
		t.Error("グループが空です - フィクスチャファイルの読み込みに失敗している可能性があります")
	}

	// フィクスチャファイルの内容と照合
	// test-aliases.jsonには3つのエイリアスがある
	if len(aliases) < 3 {
		t.Errorf("期待される最小エイリアス数: 3, 実際: %d", len(aliases))
	}

	// test-groups.jsonには2つのグループがある
	if len(groups) < 2 {
		t.Errorf("期待される最小グループ数: 2, 実際: %d", len(groups))
	}
}
