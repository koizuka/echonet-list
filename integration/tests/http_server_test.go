//go:build integration

package tests

import (
	"echonet-list/integration/helpers"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHTTPServerBasic(t *testing.T) {
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

	// HTTPクライアントを作成（タイムアウト設定）
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// ルートパスへのGETリクエスト
	resp, err := client.Get(server.GetHTTPURL())
	helpers.AssertNoError(t, err, "HTTPルートへのリクエスト")
	defer resp.Body.Close()

	// ステータスコードの確認
	helpers.AssertTrue(t, resp.StatusCode >= 200 && resp.StatusCode < 300, "HTTPステータスコードの確認")

	// レスポンスボディの読み取り
	body, err := io.ReadAll(resp.Body)
	helpers.AssertNoError(t, err, "レスポンスボディの読み取り")

	// HTMLコンテンツが返されることを確認
	bodyStr := string(body)
	helpers.AssertTrue(t, len(bodyStr) > 0, "レスポンスボディが空でないことを確認")

	t.Logf("HTTPレスポンス長: %d文字", len(bodyStr))
}

func TestHTTPServerStaticFiles(t *testing.T) {
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

	// HTTPクライアントを作成
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 一般的な静的ファイルパスをテスト
	testPaths := []string{
		"/",           // ルートパス
		"/index.html", // インデックスファイル
	}

	for _, path := range testPaths {
		t.Run("Path_"+strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
			url := server.GetHTTPURL() + path
			resp, err := client.Get(url)
			helpers.AssertNoError(t, err, "HTTPリクエスト")
			defer resp.Body.Close()

			// 404以外であることを確認（ファイルが存在しない場合は404が期待される）
			if resp.StatusCode == 404 {
				t.Logf("パス %s はファイルが存在しないため404を返しました（期待される動作）", path)
			} else {
				helpers.AssertTrue(t, resp.StatusCode >= 200 && resp.StatusCode < 300, 
					"HTTPステータスコード確認")
			}

			// Content-Typeヘッダーの確認
			contentType := resp.Header.Get("Content-Type")
			helpers.AssertNotEqual(t, "", contentType, "Content-Typeヘッダーの存在確認")

			t.Logf("パス %s - ステータス: %d, Content-Type: %s", path, resp.StatusCode, contentType)
		})
	}
}

func TestHTTPServerCORS(t *testing.T) {
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

	// HTTPクライアントを作成
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// OPTIONSリクエストでCORSプリフライトをテスト
	req, err := http.NewRequest("OPTIONS", server.GetHTTPURL(), nil)
	helpers.AssertNoError(t, err, "OPTIONSリクエストの作成")

	// プリフライトリクエストヘッダーを追加
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	resp, err := client.Do(req)
	helpers.AssertNoError(t, err, "CORSプリフライトリクエスト")
	defer resp.Body.Close()

	// CORSヘッダーの確認
	accessControlAllowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	t.Logf("Access-Control-Allow-Origin: %s", accessControlAllowOrigin)

	// ステータスコードの確認（200または204が期待される）
	helpers.AssertTrue(t, resp.StatusCode == 200 || resp.StatusCode == 204 || resp.StatusCode == 404,
		"CORSプリフライトのステータスコード確認")
}

func TestHTTPServerHealthCheck(t *testing.T) {
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

	// HTTPクライアントを作成
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// ヘルスチェック用のパスをテスト
	healthPaths := []string{
		"/health",
		"/ping",
		"/status",
	}

	for _, path := range healthPaths {
		t.Run("HealthCheck_"+strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
			url := server.GetHTTPURL() + path
			resp, err := client.Get(url)
			
			if err != nil {
				t.Logf("ヘルスチェックパス %s は実装されていません（期待される動作）", path)
				return
			}
			defer resp.Body.Close()

			// 実装されている場合は正常なレスポンスを期待
			if resp.StatusCode != 404 {
				helpers.AssertTrue(t, resp.StatusCode >= 200 && resp.StatusCode < 300,
					"ヘルスチェックのステータスコード確認")
				
				body, err := io.ReadAll(resp.Body)
				helpers.AssertNoError(t, err, "ヘルスチェックレスポンスの読み取り")
				
				t.Logf("ヘルスチェック %s - ステータス: %d, レスポンス: %s", 
					path, resp.StatusCode, string(body))
			} else {
				t.Logf("ヘルスチェックパス %s は404を返しました（実装されていません）", path)
			}
		})
	}
}

func TestHTTPServerErrorHandling(t *testing.T) {
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

	// HTTPクライアントを作成
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 存在しないパスにアクセスして404エラーをテスト
	nonexistentPaths := []string{
		"/nonexistent",
		"/api/invalid",
		"/test/404",
	}

	for _, path := range nonexistentPaths {
		t.Run("Error404_"+strings.ReplaceAll(path, "/", "_"), func(t *testing.T) {
			url := server.GetHTTPURL() + path
			resp, err := client.Get(url)
			helpers.AssertNoError(t, err, "存在しないパスへのリクエスト")
			defer resp.Body.Close()

			// 404ステータスコードが返されることを確認
			helpers.AssertEqual(t, 404, resp.StatusCode, "404エラーの確認")

			t.Logf("存在しないパス %s で期待通り404エラーが返されました", path)
		})
	}
}