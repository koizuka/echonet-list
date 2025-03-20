package main

import (
	"context"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/log"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// コマンドライン引数の定義
	port := flag.Int("port", 8080, "WebSocketサーバーのポート番号")
	debugFlag := flag.Bool("debug", false, "デバッグモードを有効にする")
	logFilenameFlag := flag.String("log", "echonet-server.log", "ログファイル名を指定する")

	// コマンドライン引数の解析
	flag.Parse()

	// ロガーのセットアップ
	logger, err := log.NewLogger(*logFilenameFlag)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ログ設定エラー: %v\n", err)
		return
	}
	log.SetLogger(logger)

	// ログファイルを閉じる
	defer log.SetLogger(nil)

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

	// ログローテーション用のシグナルハンドリング (SIGHUP)
	rotateSignalCh := make(chan os.Signal, 1)
	signal.Notify(rotateSignalCh, syscall.SIGHUP)
	go func() {
		for {
			<-rotateSignalCh
			fmt.Println("SIGHUPを受信しました。ログファイルをローテーションします...")
			logger := log.GetLogger()
			if err := logger.Rotate(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "ログローテーションエラー: %v\n", err)
			}
		}
	}()

	// Controller Object
	SEOJ := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	// local address （ECHONET Liteの既定ポートを使用）
	var localIP net.IP = nil // nilはすべてのインターフェースをリッスンする

	// ECHONETLiteHandlerの作成
	handler, err := echonet_lite.NewECHONETLiteHandler(ctx, localIP, SEOJ, *debugFlag)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := handler.Close(); err != nil {
			fmt.Printf("セッションのクローズ中にエラーが発生しました: %v\n", err)
		}
	}()

	// メインループの開始
	handler.StartMainLoop()

	// ノードリストの通知
	_ = handler.NotifyNodeList()

	// デバイスの発見
	_ = handler.Discover()

	// ECHONETLiteServerの作成
	server := NewECHONETLiteServer(ctx, handler)
	
	// サーバーの起動
	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("WebSocketサーバーを起動しています: %s\n", addr)
	if err := server.Start(addr); err != nil {
		fmt.Printf("サーバーエラー: %v\n", err)
	}
}