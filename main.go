package main

import (
	"context"
	"echonet-list/client"
	"echonet-list/config"
	"echonet-list/console"
	"echonet-list/server"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	defaultLog = "echonet-list.log" // デフォルトのログファイル名
)

func main() {
	// コマンドライン引数のヘルプメッセージをカスタマイズ
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "使用方法: %s [オプション]\n\nオプション:\n", os.Args[0])
		flag.PrintDefaults()
	}

	// コマンドライン引数の定義
	debugFlag := flag.Bool("debug", false, "デバッグモードを有効にする")
	logFilenameFlag := flag.String("log", defaultLog, "ログファイル名を指定する")

	// 設定ファイル関連のフラグ
	configFileFlag := flag.String("config", "", "TOML設定ファイルのパスを指定する")

	// WebSocket関連のフラグ
	websocketFlag := flag.Bool("websocket", false, "WebSocketサーバーモードを有効にする")
	wsAddrFlag := flag.String("ws-addr", "localhost:8080", "WebSocketサーバーのアドレスを指定する")
	wsClientFlag := flag.Bool("ws-client", false, "WebSocketクライアントモードを有効にする")
	wsClientAddrFlag := flag.String("ws-client-addr", "ws://localhost:8080/ws", "WebSocketクライアントの接続先アドレスを指定する")
	wsBothFlag := flag.Bool("ws-both", false, "WebSocketサーバーとクライアントの両方を有効にする（テスト用）")

	// TLS関連のフラグ
	wsTLSFlag := flag.Bool("ws-tls", false, "WebSocketサーバーでTLSを有効にする")
	wsCertFileFlag := flag.String("ws-cert-file", "", "TLS証明書ファイルのパスを指定する")
	wsKeyFileFlag := flag.String("ws-key-file", "", "TLS秘密鍵ファイルのパスを指定する")
	daemonFlag := flag.Bool("daemon", false, "デーモンモードを有効にする")
	pidFileFlag := flag.String("pidfile", "", "PIDファイルのパスを指定する")

	// コマンドライン引数の解析
	flag.Parse()

	// 設定ファイルの読み込み
	cfg, err := config.LoadConfig(*configFileFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "設定ファイルの読み込みに失敗しました: %v\n", err)
		os.Exit(1)
	}

	// コマンドライン引数を設定に適用
	cmdArgs := config.CommandLineArgs{
		ConfigFile:      *configFileFlag,
		ConfigSpecified: *configFileFlag != "",

		Debug:          *debugFlag,
		DebugSpecified: flag.Lookup("debug").Value.String() != "false",

		LogFilename:          *logFilenameFlag,
		LogFilenameSpecified: flag.Lookup("log").Value.String() != defaultLog,

		WebSocketEnabled:          *websocketFlag,
		WebSocketEnabledSpecified: flag.Lookup("websocket").Value.String() != "false",

		WebSocketAddr:          *wsAddrFlag,
		WebSocketAddrSpecified: flag.Lookup("ws-addr").Value.String() != "localhost:8080",

		WebSocketTLSEnabled:          *wsTLSFlag,
		WebSocketTLSEnabledSpecified: flag.Lookup("ws-tls").Value.String() != "false",

		WebSocketTLSCertFile:          *wsCertFileFlag,
		WebSocketTLSCertFileSpecified: *wsCertFileFlag != "",

		WebSocketTLSKeyFile:          *wsKeyFileFlag,
		WebSocketTLSKeyFileSpecified: *wsKeyFileFlag != "",

		WebSocketClientEnabled:          *wsClientFlag,
		WebSocketClientEnabledSpecified: flag.Lookup("ws-client").Value.String() != "false",

		WebSocketClientAddr:          *wsClientAddrFlag,
		WebSocketClientAddrSpecified: flag.Lookup("ws-client-addr").Value.String() != "ws://localhost:8080/ws",

		WebSocketBoth:          *wsBothFlag,
		WebSocketBothSpecified: flag.Lookup("ws-both").Value.String() != "false",
		DaemonEnabled:          *daemonFlag,
		DaemonEnabledSpecified: flag.Lookup("daemon").Value.String() != "false",
		PIDFile:                *pidFileFlag,
		PIDFileSpecified:       flag.Lookup("pidfile").Value.String() != "",
	}

	cfg.ApplyCommandLineArgs(cmdArgs)

	// Daemon mode pre-checks and PID file handling
	if cfg.Daemon.Enabled {
		if !cfg.WebSocket.Enabled {
			fmt.Fprintln(os.Stderr, "デーモンモードでは WebSocket サーバーを有効にする必要があります。-websocket を指定してください。")
			os.Exit(1)
		}
		if cfg.Daemon.PIDFile == "" {
			fmt.Fprintln(os.Stderr, "デーモンモードでは pidfile を指定する必要があります。-pidfile を指定してください。")
			os.Exit(1)
		}
		pid := os.Getpid()
		if err := os.WriteFile(cfg.Daemon.PIDFile, []byte(fmt.Sprintf("%d\n", pid)), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "PIDファイルの作成に失敗しました: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(cfg.Daemon.PIDFile)
	}

	// 設定値を取得
	debug := cfg.Debug
	logFilename := cfg.Log.Filename
	websocket := cfg.WebSocket.Enabled
	wsAddr := cfg.WebSocket.Addr
	wsClient := cfg.WebSocketClient.Enabled
	if cfg.Daemon.Enabled {
		wsClient = false // Daemon modeではクライアントモードを無効にする
	}
	wsClientAddr := cfg.WebSocketClient.Addr

	// ロガーのセットアップ
	logManager, err := server.NewLogManager(logFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ログ設定エラー: %v\n", err)
		os.Exit(1)
	}

	// デーモンモードのときにのみ、SIGHUP でlog rotate を実行
	if cfg.Daemon.Enabled {
		logManager.AutoRotate()
	}

	// ログファイルを閉じる
	defer func() {
		_ = logManager.Close()
	}()

	// ルートコンテキストの作成
	signals := []os.Signal{os.Interrupt, syscall.SIGTERM}
	if !cfg.Daemon.Enabled {
		// コンソールUIモードではSIGHUPでも終了する
		signals = append(signals, syscall.SIGHUP)
	}
	ctx, stop := signal.NotifyContext(context.Background(), signals...)
	defer stop() // プログラム終了時にコンテキストをキャンセル

	var wg sync.WaitGroup
	var c client.ECHONETListClient

	// WebSocketサーバーモードの場合
	if websocket {
		// ECHONETLiteHandlerの作成
		s, err := server.NewServer(ctx, debug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		defer func() {
			if err := s.Close(); err != nil {
				fmt.Printf("セッションのクローズ中にエラーが発生しました: %v\n", err)
			}
		}()

		// WebSocketサーバーの作成と起動
		wsServer, err := server.NewWebSocketServer(ctx, wsAddr, client.NewECHONETListClientProxy(s.GetHandler()), s.GetHandler())
		if err != nil {
			fmt.Fprintf(os.Stderr, "WebSocketサーバーの作成に失敗しました: %v\n", err)
			os.Exit(1)
		}

		// プログラム終了時にWebSocketサーバーを停止する
		defer func() {
			if err := wsServer.Stop(); err != nil {
				fmt.Printf("WebSocketサーバーの停止に失敗しました: %v\n", err)
			}
		}()

		// 定期更新間隔をパース
		updateIntervalStr := cfg.WebSocket.PeriodicUpdateInterval
		updateInterval, err := time.ParseDuration(updateIntervalStr)
		if err != nil || updateIntervalStr == "" {
			fmt.Printf("警告: 設定ファイル 'websocket.periodic_update_interval' の値 '%s' は無効です。デフォルトの1分を使用します。\n", updateIntervalStr)
			updateInterval = 1 * time.Minute // パース失敗時はデフォルト値
		}

		// TLSと定期更新間隔の設定を準備
		readyChan := make(chan struct{})
		startOptions := server.StartOptions{
			Ready:                  readyChan,
			CertFile:               cfg.WebSocket.TLS.CertFile,
			KeyFile:                cfg.WebSocket.TLS.KeyFile,
			PeriodicUpdateInterval: updateInterval,
		}

		// 設定された定期更新間隔を表示
		if updateInterval > 0 {
			fmt.Printf("WebSocketサーバーの定期更新間隔: %v\n", updateInterval)
		} else {
			fmt.Println("WebSocketサーバーの定期更新は無効です。")
		}

		// TLSが有効かどうかを表示
		if cfg.WebSocket.TLS.Enabled {
			if startOptions.CertFile != "" && startOptions.KeyFile != "" {
				fmt.Printf("TLSが有効です。証明書: %s, 秘密鍵: %s\n", startOptions.CertFile, startOptions.KeyFile)
			} else {
				fmt.Fprintln(os.Stderr, "TLSが有効ですが、証明書または秘密鍵が指定されていません。")
				os.Exit(1)
			}
		}

		// WebSocketサーバーを起動
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := wsServer.Start(startOptions); err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "WebSocketサーバーの起動に失敗しました: %v\n", err)
				os.Exit(1)
			}
		}()

		fmt.Printf("WebSocketサーバーを起動しました: %s\n", wsAddr)

		// WebSocketクライアントモードも有効な場合は、サーバーの Ready チャネルを待機
		if wsClient {
			<-readyChan
		} else {
			// クライアントモードでない場合は、ECHONETListClientProxyを使用
			c = client.NewECHONETListClientProxy(s.GetHandler())
		}
	}

	// WebSocketクライアントの変数
	var wsClientInstance *client.WebSocketClient

	// WebSocketクライアントモードの場合
	if wsClient {
		// TLSが有効な場合は、接続先アドレスを修正
		if cfg.WebSocket.TLS.Enabled && cfg.WebSocket.TLS.CertFile != "" && cfg.WebSocket.TLS.KeyFile != "" {
			// ws:// を wss:// に置き換え
			if strings.HasPrefix(wsClientAddr, "ws://") {
				wsClientAddr = "wss://" + wsClientAddr[5:]
				fmt.Printf("TLSが有効なため、接続先アドレスを %s に変更しました\n", wsClientAddr)
			}
		}

		// WebSocketクライアントの作成
		var err error
		wsClientInstance, err = client.NewWebSocketClient(ctx, wsClientAddr, debug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WebSocketクライアントの作成に失敗しました: %v\n", err)
			os.Exit(1)
		}

		// WebSocketサーバーに接続
		if err := wsClientInstance.Connect(); err != nil {
			fmt.Fprintf(os.Stderr, "WebSocketサーバーへの接続に失敗しました: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("WebSocketサーバーに接続しました: %s\n", wsClientAddr)

		// プログラム終了時にWebSocketクライアントを閉じる
		defer func() {
			if err := wsClientInstance.Close(); err != nil {
				fmt.Printf("WebSocketクライアントのクローズに失敗しました: %v\n", err)
			}
		}()

		// クライアントを設定
		c = wsClientInstance
	}

	// スタンドアロンモードの場合
	if !websocket && !wsClient {
		// ECHONETLiteHandlerの作成
		s, err := server.NewServer(ctx, debug)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		defer func() {
			if err := s.Close(); err != nil {
				fmt.Printf("セッションのクローズ中にエラーが発生しました: %v\n", err)
			}
		}()

		// クライアントを設定
		c = client.NewECHONETListClientProxy(s.GetHandler())
	}

	if !cfg.Daemon.Enabled {
		// コンソールUIモード
		console.ConsoleProcess(ctx, c)
	} else {
		// デーモンモード
		// wg.Wait() または ctx.Done() を待機
		go func() {
			wg.Wait()
			stop()
		}()
		<-ctx.Done()
	}
}
