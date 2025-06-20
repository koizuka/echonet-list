//go:build integration

package tests

import (
	"echonet-list/integration/helpers"
	"path/filepath"
	"testing"
	"time"
)

func TestServerStartupAndShutdown(t *testing.T) {
	// テストサーバーを作成
	server, err := helpers.NewTestServer()
	helpers.AssertNoError(t, err, "テストサーバーの作成")

	// フィクスチャファイルのパス設定
	fixturesDir, err := helpers.GetAbsolutePath("../fixtures")
	helpers.AssertNoError(t, err, "フィクスチャディレクトリのパス取得")

	devicesFile := filepath.Join(fixturesDir, "test-devices.json")
	aliasesFile := filepath.Join(fixturesDir, "test-aliases.json")
	groupsFile := filepath.Join(fixturesDir, "test-groups.json")

	server.SetTestFixtures(devicesFile, aliasesFile, groupsFile)

	// サーバー開始前は実行中でないことを確認
	helpers.AssertFalse(t, server.IsRunning(), "サーバー開始前の状態確認")

	// サーバーを起動
	err = server.Start()
	helpers.AssertNoError(t, err, "サーバーの起動")

	// サーバーが実行中であることを確認
	helpers.AssertTrue(t, server.IsRunning(), "サーバー起動後の状態確認")

	// URLが正しく設定されていることを確認
	wsURL := server.GetWebSocketURL()
	httpURL := server.GetHTTPURL()

	helpers.AssertNotEqual(t, "", wsURL, "WebSocket URLの設定確認")
	helpers.AssertNotEqual(t, "", httpURL, "HTTP URLの設定確認")

	t.Logf("WebSocket URL: %s", wsURL)
	t.Logf("HTTP URL: %s", httpURL)

	// サーバーが応答するまで待機
	condition := func() bool {
		return server.IsRunning()
	}
	success := helpers.WaitForCondition(condition, 5*time.Second, 100*time.Millisecond)
	helpers.AssertTrue(t, success, "サーバーの応答確認")

	// サーバーを停止
	err = server.Stop()
	helpers.AssertNoError(t, err, "サーバーの停止")

	// サーバーが停止していることを確認
	helpers.AssertFalse(t, server.IsRunning(), "サーバー停止後の状態確認")
}

func TestMultipleServerInstances(t *testing.T) {
	// 複数のサーバーインスタンスを作成してポート競合がないことを確認
	servers := make([]*helpers.TestServer, 3)

	// サーバーを順次作成・起動
	for i := 0; i < 3; i++ {
		server, err := helpers.NewTestServer()
		helpers.AssertNoError(t, err, "テストサーバーの作成")
		servers[i] = server

		// フィクスチャファイルのパス設定
		fixturesDir, err := helpers.GetAbsolutePath("../fixtures")
		helpers.AssertNoError(t, err, "フィクスチャディレクトリのパス取得")

		devicesFile := filepath.Join(fixturesDir, "test-devices.json")
		aliasesFile := filepath.Join(fixturesDir, "test-aliases.json")
		groupsFile := filepath.Join(fixturesDir, "test-groups.json")

		server.SetTestFixtures(devicesFile, aliasesFile, groupsFile)

		err = server.Start()
		helpers.AssertNoError(t, err, "テストサーバーの起動")

		t.Logf("サーバー%d - Port: %d", i+1, server.Port)
	}

	// 全サーバーが異なるポートを使用していることを確認
	ports := make(map[int]bool)
	for _, server := range servers {
		helpers.AssertTrue(t, server.IsRunning(), "サーバーの実行状態")

		if ports[server.Port] {
			t.Errorf("ポート%dが重複しています", server.Port)
		}
		ports[server.Port] = true
	}

	// 全サーバーを停止
	for _, server := range servers {
		err := server.Stop()
		helpers.AssertNoError(t, err, "テストサーバーの停止")
		helpers.AssertFalse(t, server.IsRunning(), "サーバー停止後の状態")
	}
}

func TestConfigurationLoading(t *testing.T) {
	// テストサーバーを作成
	server, err := helpers.NewTestServer()
	helpers.AssertNoError(t, err, "テストサーバーの作成")

	// 設定が正しく読み込まれていることを確認
	cfg := server.Config
	helpers.AssertTrue(t, cfg.Debug, "デバッグモードの確認")
	helpers.AssertTrue(t, cfg.WebSocket.Enabled, "WebSocketサーバーの有効化確認")
	helpers.AssertFalse(t, cfg.TLS.Enabled, "TLSの無効化確認")
	helpers.AssertTrue(t, cfg.HTTPServer.Enabled, "HTTPサーバーの有効化確認")
	// データファイルパス設定は後で行うため、ここでは確認しない

	// フィクスチャファイルを設定
	fixturesDir, err := helpers.GetAbsolutePath("../fixtures")
	helpers.AssertNoError(t, err, "フィクスチャディレクトリのパス取得")

	devicesFile := filepath.Join(fixturesDir, "test-devices.json")
	aliasesFile := filepath.Join(fixturesDir, "test-aliases.json")
	groupsFile := filepath.Join(fixturesDir, "test-groups.json")

	server.SetTestFixtures(devicesFile, aliasesFile, groupsFile)

	// フィクスチャファイルパスが正しく設定されていることを確認
	helpers.AssertEqual(t, devicesFile, cfg.DataFiles.DevicesFile, "デバイスファイルパスの確認")
	helpers.AssertEqual(t, aliasesFile, cfg.DataFiles.AliasesFile, "エイリアスファイルパスの確認")
	helpers.AssertEqual(t, groupsFile, cfg.DataFiles.GroupsFile, "グループファイルパスの確認")

	// フィクスチャファイルが存在することを確認
	helpers.AssertTrue(t, helpers.FileExists(devicesFile), "デバイスファイルの存在確認")
	helpers.AssertTrue(t, helpers.FileExists(aliasesFile), "エイリアスファイルの存在確認")
	helpers.AssertTrue(t, helpers.FileExists(groupsFile), "グループファイルの存在確認")

	// テスト後のクリーンアップは不要（サーバーを起動していないため）
}
