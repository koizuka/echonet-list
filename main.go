package main

import (
	"context"
	"echonet-list/client"
	"echonet-list/server"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chzyer/readline"
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

	// コマンドライン引数の解析
	flag.Parse()

	// フラグの値を取得
	debug := *debugFlag
	logFilename := *logFilenameFlag

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

	c := client.NewECHONETListClientProxy(s.GetHandler())

	// コマンドプロセッサの作成と開始
	processor := NewCommandProcessor(ctx, c)
	processor.Start()
	// defer processor.Stop() は不要。明示的に呼び出すため

	// コマンドの使用方法を表示
	fmt.Println("help for usage, quit to exit")

	// コマンド入力待ち（readline を使用して履歴機能を追加）
	// 履歴ファイルのパスを設定
	historyFile := ".echonet_history"
	if home, err := os.UserHomeDir(); err == nil {
		historyFile = fmt.Sprintf("%s/.echonet_history", home)
	}

	// コマンド補完用の関数を定義

	aliases := []readline.PrefixCompleterInterface{}
	for _, alias := range c.GetAllPropertyAliases() {
		aliases = append(aliases, readline.PcItem(alias))
	}

	// devicesとlistコマンド用のオプション
	deviceListOptions := []readline.PrefixCompleterInterface{
		readline.PcItem("-all"),
		readline.PcItem("-props"),
	}
	deviceListOptions = append(deviceListOptions, aliases...)

	completer := readline.NewPrefixCompleter(
		readline.PcItem("quit"),
		readline.PcItem("discover"),
		readline.PcItem("help"),
		readline.PcItem("get", aliases...),
		readline.PcItem("set", aliases...),
		readline.PcItem("devices", deviceListOptions...),
		readline.PcItem("list", deviceListOptions...),
		readline.PcItem("update"),
		readline.PcItem("debug",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
	)

	// readline の設定
	rlConfig := &readline.Config{
		Prompt:          "> ",
		HistoryFile:     historyFile,
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "quit",
	}

	rl, err := readline.NewEx(rlConfig)
	if err != nil {
		fmt.Printf("readline の初期化エラー: %v\n", err)
		return
	}
	defer func(rl *readline.Instance) {
		_ = rl.Close()
	}(rl)

	p := NewCommandParser(c, c)

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			break
		}

		cmd, err := p.ParseCommand(line, c.IsDebug())
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
			continue
		}
		if cmd == nil {
			continue
		}

		if cmd.Type == CmdQuit {
			// quitコマンドの場合は、コマンドチャネル経由で送信せず、直接終了処理を行う
			close(cmd.Done) // 完了を通知
			processor.Stop()
			// handler.Close() は defer で呼ばれるので、ここでは呼ばない
			break
		}

		// コマンドを送信し、エラーをチェック
		if err := processor.SendCommand(cmd); err != nil {
			fmt.Printf("エラー: %v\n", err)
		}
	}
}
