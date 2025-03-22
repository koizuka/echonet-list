package main

import (
	"context"
	"echonet-list/client"
	"echonet-list/server"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
)

// カスタム補完機能を実装する構造体
type dynamicCompleter struct {
	client client.ECHONETListClient
}

// Do メソッドを実装して readline.AutoCompleter インターフェースを満たす
func (dc *dynamicCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	// 現在の入力行を解析して、入力段階を判断する
	lineStr := string(line[:pos])
	words := splitWords(lineStr)
	wordCount := len(words)

	// デバッグ出力
	// fmt.Fprintf(os.Stderr, "DEBUG: line=%q, pos=%d, words=%v, wordCount=%d\n", lineStr, pos, words, wordCount)

	// 最後の単語を取得
	lastWord := ""
	if wordCount > 0 {
		lastWord = words[wordCount-1]
	}

	// 候補を取得
	var candidates []string
	if wordCount <= 1 {
		// コマンド名の補完
		candidates = dc.getCommandCandidates()
	} else {
		// コマンド引数の補完
		cmd := words[0]
		candidates = dc.getCandidatesForCommand(cmd, wordCount, words)
	}

	// fmt.Printf("DEBUG: lastWord=%v, candidates=%v\n", lastWord, candidates)

	// 最後の単語でフィルタリングして返す
	result := [][]rune{}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, lastWord) {
			result = append(result, []rune(candidate[len(lastWord):]))
		}
	}
	return result, len(lastWord)
}

// コマンド名の候補を返す
func (dc *dynamicCompleter) getCommandCandidates() []string {
	var candidates []string
	for _, cmdDef := range GetCommandTable() {
		candidates = append(candidates, cmdDef.Name)
		candidates = append(candidates, cmdDef.Aliases...)
	}
	return candidates
}

// デバイスエイリアスの候補を返す
func (dc *dynamicCompleter) getDeviceAliasCandidates() []string {
	var aliases []string
	for _, pair := range dc.client.AliasList() {
		aliases = append(aliases, pair.Alias)
	}
	return aliases
}

// プロパティエイリアスの候補を返す
func (dc *dynamicCompleter) getPropertyAliasCandidates() []string {
	return dc.client.GetAllPropertyAliases()
}

// コマンドと引数位置に応じた候補を返す
func (dc *dynamicCompleter) getCandidatesForCommand(cmd string, wordCount int, words []string) []string {
	switch cmd {
	case "get", "set":
		if wordCount == 2 {
			// デバイスエイリアスのみを表示
			return dc.getDeviceAliasCandidates()
		} else if wordCount >= 3 {
			// プロパティエイリアスのみを表示
			return dc.getPropertyAliasCandidates()
		}

	case "devices", "list":
		// オプションとエイリアスを表示
		options := []string{"-all", "-props"}
		return append(options, dc.getDeviceAliasCandidates()...)

	case "update":
		if wordCount == 2 {
			// デバイスエイリアスのみを表示
			return dc.getDeviceAliasCandidates()
		}

	case "debug":
		if wordCount == 2 {
			// on/off オプションを表示
			return []string{"on", "off"}
		}

	case "help":
		if wordCount == 2 {
			// コマンド名を表示
			return dc.getCommandCandidates()
		}

	case "alias":
		if wordCount == 2 {
			// -delete オプションとエイリアス名を表示
			return append([]string{"-delete"}, dc.getDeviceAliasCandidates()...)
		} else if wordCount == 3 && words[1] == "-delete" {
			// alias -delete の後にはエイリアス名
			return dc.getDeviceAliasCandidates()
		} else if wordCount >= 3 {
			// alias <name> の後にはデバイス指定子（IPアドレスやクラスコード）
			// ここではIPアドレスの補完は難しいので、デバイスエイリアスのみ提供
			return dc.getDeviceAliasCandidates()
		}
	}

	return []string{} // デフォルトは空リスト
}

// 入力行を単語に分割する補助関数
func splitWords(line string) []string {
	// 空の入力の場合は空のスライスを返す
	if line == "" {
		return []string{}
	}

	var words []string
	var word string
	inQuote := false
	lastWasSpace := false

	for _, r := range line {
		switch r {
		case ' ', '\t':
			if !inQuote {
				if word != "" {
					words = append(words, word)
					word = ""
				}
				lastWasSpace = true
			} else if inQuote {
				word += string(r)
			}
		case '"', '\'':
			inQuote = !inQuote
			lastWasSpace = false
		default:
			word += string(r)
			lastWasSpace = false
		}
	}

	if word != "" {
		words = append(words, word)
	}

	// 末尾が空白だった場合、空の単語を1つだけ追加
	if lastWasSpace {
		words = append(words, "")
	}

	return words
}

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

	// 動的補完機能を使用
	completer := &dynamicCompleter{client: c}

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
