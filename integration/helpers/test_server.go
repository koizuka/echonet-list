//go:build integration

package helpers

import (
	"context"
	"echonet-list/client"
	"echonet-list/config"
	"echonet-list/server"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TestServer は統合テスト用のサーバーを管理する
type TestServer struct {
	Server     *server.Server
	WSServer   *server.WebSocketServer
	Config     *config.Config
	Port       int
	mu         sync.Mutex
	running    bool
	logManager *server.LogManager
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewTestServer は新しいテストサーバーを作成する
func NewTestServer() (*TestServer, error) {
	// 利用可能なポートを見つける
	port, err := findFreePort()
	if err != nil {
		return nil, fmt.Errorf("利用可能なポートが見つかりません: %v", err)
	}

	// 一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "echonet-test-*")
	if err != nil {
		return nil, fmt.Errorf("一時ディレクトリの作成に失敗: %v", err)
	}

	// プロジェクトルートを取得
	projectRoot, err := GetAbsolutePath("../../")
	if err != nil {
		return nil, fmt.Errorf("プロジェクトルートの取得に失敗: %v", err)
	}

	// テスト用設定を作成
	cfg := config.NewConfig()
	cfg.Debug = true
	cfg.WebSocket.Enabled = true
	cfg.TLS.Enabled = false
	cfg.HTTPServer.Enabled = true
	cfg.HTTPServer.Port = port
	cfg.HTTPServer.Host = "localhost"
	cfg.HTTPServer.WebRoot = filepath.Join(projectRoot, "web/bundle")
	cfg.Log.Filename = filepath.Join(tempDir, "test-echonet-list.log")

	// データファイルパスは必要に応じて後で設定

	// コンテキスト作成
	ctx, cancel := context.WithCancel(context.Background())

	return &TestServer{
		Config: cfg,
		Port:   port,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// SetTestFixtures はテスト用のフィクスチャファイルを設定する
func (ts *TestServer) SetTestFixtures(devicesFile, aliasesFile, groupsFile string) {
	ts.Config.DataFiles.DevicesFile = devicesFile
	ts.Config.DataFiles.AliasesFile = aliasesFile
	ts.Config.DataFiles.GroupsFile = groupsFile
}

// Start はテストサーバーを起動する
func (ts *TestServer) Start() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.running {
		return fmt.Errorf("サーバーは既に実行中です")
	}

	// ログマネージャーを作成
	logManager, err := server.NewLogManager(ts.Config.Log.Filename, ts.Config.Debug)
	if err != nil {
		return fmt.Errorf("ログマネージャーの作成に失敗: %v", err)
	}
	ts.logManager = logManager

	// ECHONETサーバーを作成
	s, err := server.NewServer(ts.ctx, ts.Config)
	if err != nil {
		return fmt.Errorf("ECHONETサーバーの作成に失敗: %v", err)
	}
	ts.Server = s

	// WebSocketサーバーを作成
	httpAddr := fmt.Sprintf("%s:%d", ts.Config.HTTPServer.Host, ts.Config.HTTPServer.Port)
	serverStartupTime := time.Now().UTC() // テスト用のサーバー起動時刻
	wsServer, err := server.NewWebSocketServer(ts.ctx, httpAddr, client.NewECHONETListClientProxy(s.GetHandler()), s.GetHandler(), serverStartupTime)
	if err != nil {
		return fmt.Errorf("WebSocketサーバーの作成に失敗: %v", err)
	}
	ts.WSServer = wsServer

	// ログブロードキャストを設定
	if err := logManager.SetTransport(wsServer.GetTransport()); err != nil {
		return fmt.Errorf("ログブロードキャスト設定に失敗: %v", err)
	}

	// サーバーを非同期で起動
	readyChan := make(chan struct{})
	startOptions := server.StartOptions{
		Ready:                  readyChan,
		PeriodicUpdateInterval: 0, // テスト時は定期更新を無効化
		HTTPEnabled:            ts.Config.HTTPServer.Enabled,
		HTTPWebRoot:            ts.Config.HTTPServer.WebRoot,
	}

	go func() {
		if err := wsServer.Start(startOptions); err != nil {
			fmt.Printf("WebSocketサーバーの起動に失敗: %v\n", err)
		}
	}()

	// サーバーが起動するまで待機
	select {
	case <-readyChan:
		ts.running = true
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("サーバーの起動がタイムアウトしました")
	}
}

// Stop はテストサーバーを停止する
func (ts *TestServer) Stop() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !ts.running {
		return nil
	}

	var errors []error

	// WebSocketサーバーを停止
	if ts.WSServer != nil {
		if err := ts.WSServer.Stop(); err != nil {
			errors = append(errors, fmt.Errorf("WebSocketサーバーの停止に失敗: %v", err))
		}
	}

	// ECHONETサーバーを停止
	if ts.Server != nil {
		if err := ts.Server.Close(); err != nil {
			errors = append(errors, fmt.Errorf("ECHONETサーバーの停止に失敗: %v", err))
		}
	}

	// ログマネージャーを停止
	if ts.logManager != nil {
		if err := ts.logManager.Close(); err != nil {
			errors = append(errors, fmt.Errorf("ログマネージャーの停止に失敗: %v", err))
		}
	}

	// コンテキストをキャンセル
	ts.cancel()

	ts.running = false

	if len(errors) > 0 {
		return fmt.Errorf("停止中にエラーが発生: %v", errors)
	}

	return nil
}

// GetWebSocketURL はWebSocketのURLを返す
func (ts *TestServer) GetWebSocketURL() string {
	return fmt.Sprintf("ws://%s:%d/ws", ts.Config.HTTPServer.Host, ts.Port)
}

// GetHTTPURL はHTTPのURLを返す
func (ts *TestServer) GetHTTPURL() string {
	return fmt.Sprintf("http://%s:%d", ts.Config.HTTPServer.Host, ts.Port)
}

// IsRunning はサーバーが実行中かどうかを返す
func (ts *TestServer) IsRunning() bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.running
}

// findFreePort は利用可能なポートを見つける
func findFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
