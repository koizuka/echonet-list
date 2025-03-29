package main

import (
	"context"
	"echonet-list/client"
	"echonet-list/console"
	"echonet-list/server"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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

	// WebSocket関連のフラグ
	websocketFlag := flag.Bool("websocket", false, "WebSocketサーバーモードを有効にする")
	wsAddrFlag := flag.String("ws-addr", "localhost:8080", "WebSocketサーバーのアドレスを指定する")
	wsClientFlag := flag.Bool("ws-client", false, "WebSocketクライアントモードを有効にする")
	wsClientAddrFlag := flag.String("ws-client-addr", "ws://localhost:8080/ws", "WebSocketクライアントの接続先アドレスを指定する")
	wsBothFlag := flag.Bool("ws-both", false, "WebSocketサーバーとクライアントの両方を有効にする（テスト用）")

	// コマンドライン引数の解析
	flag.Parse()

	// フラグの値を取得
	debug := *debugFlag
	logFilename := *logFilenameFlag
	websocket := *websocketFlag
	wsAddr := *wsAddrFlag
	wsClient := *wsClientFlag
	wsClientAddr := *wsClientAddrFlag
	wsBoth := *wsBothFlag

	// -ws-both が指定された場合は、サーバーとクライアントの両方を有効にする
	if wsBoth {
		websocket = true
		wsClient = true
	}

	// ロガーのセットアップ
	logger, err := server.NewLogManager(logFilename)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ログ設定エラー: %v\n", err)
		return
	}

	// ログファイルを閉じる
	defer func() {
		_ = logger.Close()
	}()

	// ルートコンテキストの作成
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // プログラム終了時にコンテキストをキャンセル

	// シグナルハンドリングの設定 (SIGINT, SIGTERM)
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalCh
		fmt.Println("\nシグナルを受信しました。終了します...")
		cancel() // シグナル受信時にコンテキストをキャンセル
	}()

	var wg sync.WaitGroup
	var c client.ECHONETListClient

	// WebSocketサーバーモードの場合
	if websocket {
		// ECHONETLiteHandlerの作成
		s, err := server.NewServer(ctx, debug)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer func() {
			if err := s.Close(); err != nil {
				fmt.Printf("セッションのクローズ中にエラーが発生しました: %v\n", err)
			}
		}()

		// WebSocketサーバーの作成と起動
		wsServer, err := server.NewWebSocketServer(ctx, wsAddr, client.NewECHONETListClientProxy(s.GetHandler()), s.GetHandler())
		if err != nil {
			fmt.Printf("WebSocketサーバーの作成に失敗しました: %v\n", err)
			return
		}

		// プログラム終了時にWebSocketサーバーを停止する
		defer func() {
			if err := wsServer.Stop(); err != nil {
				fmt.Printf("WebSocketサーバーの停止に失敗しました: %v\n", err)
			}
		}()

		// WebSocketサーバーを起動
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := wsServer.Start(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("WebSocketサーバーの起動に失敗しました: %v\n", err)
			}
		}()

		fmt.Printf("WebSocketサーバーを起動しました: %s\n", wsAddr)

		// WebSocketクライアントモードも有効な場合は、少し待ってからクライアントを起動
		if wsClient {
			// サーバーが起動するまで少し待つ
			time.Sleep(500 * time.Millisecond)
		} else {
			// クライアントモードでない場合は、ECHONETListClientProxyを使用
			c = client.NewECHONETListClientProxy(s.GetHandler())
		}
	}

	// WebSocketクライアントの変数
	var wsClientInstance *client.WebSocketClient

	// WebSocketクライアントモードの場合
	if wsClient {
		// WebSocketクライアントの作成
		var err error
		wsClientInstance, err = client.NewWebSocketClient(ctx, wsClientAddr, debug)
		if err != nil {
			fmt.Printf("WebSocketクライアントの作成に失敗しました: %v\n", err)
			return
		}

		// WebSocketサーバーに接続
		if err := wsClientInstance.Connect(); err != nil {
			fmt.Printf("WebSocketサーバーへの接続に失敗しました: %v\n", err)
			return
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
			fmt.Println(err)
			return
		}
		defer func() {
			if err := s.Close(); err != nil {
				fmt.Printf("セッションのクローズ中にエラーが発生しました: %v\n", err)
			}
		}()

		// クライアントを設定
		c = client.NewECHONETListClientProxy(s.GetHandler())
	}

	// コンソールUIの終了を通知するチャネル
	consoleDone := make(chan struct{})

	// コンソールUIを開始
	go func() {
		console.ConsoleProcess(ctx, c)
		close(consoleDone) // コンソールUIが終了したことを通知
	}()

	// コンソールUIの終了を待つ
	<-consoleDone
}
