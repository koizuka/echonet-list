package console

import (
	"context"
	"echonet-list/client"
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"golang.org/x/exp/slices"
	"golang.org/x/term"
)

func ConsoleProcess(ctx context.Context, c client.ECHONETListClient) {
	// 現在の端末状態を保存
	orig, _ := term.GetState(int(os.Stdin.Fd()))
	defer term.Restore(int(os.Stdin.Fd()), orig)

	// 履歴ファイルのパスを取得し、履歴を読み込む
	historyFilePath := getHistoryFilePath()
	initialHistory := loadHistory(historyFilePath)

	// コマンドプロセッサの作成と開始
	processor := NewCommandProcessor(ctx, c)
	processor.Start()
	defer processor.Stop()

	// コマンドの使用方法を表示
	fmt.Println("help for usage, quit to exit")

	// コマンドパーサーの作成
	p := NewCommandParser(c, c, c)

	// Executor: ユーザーがEnterを押したときに実行される関数
	executor := func(line string) {
		trimmedLine := strings.TrimSpace(line)
		// 空行は何もしない
		if trimmedLine == "" {
			return
		}

		cmd, err := p.ParseCommand(line, c.IsDebug())
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
			return // エラーがあってもプロンプトは継続
		}
		if cmd == nil {
			return // 何も入力されなかった場合など
		}

		// 履歴に追加 (quit 以外)
		if cmd.Type != CmdQuit {
			initialHistory = append(initialHistory, line)
		}

		// コマンドを送信し、エラーをチェック
		if err := processor.SendCommand(cmd); err != nil {
			fmt.Printf("エラー: %v\n", err)
		}
		// コマンド完了待機は processor 内部で行われる
	}

	// Completer: 入力中に補完候補を返す関数
	completer := func(d prompt.Document) []prompt.Suggest {
		lastWord := d.GetWordBeforeCursor()
		if lastWord == "" {
			return []prompt.Suggest{}
		}

		lineStr := d.TextBeforeCursor()
		words := splitWords(lineStr)
		wordCount := len(words)

		// コマンド名補完 (最初の単語 or help の2番目の単語)
		shouldSuggestCommands := false
		if wordCount <= 1 {
			shouldSuggestCommands = true
		} else if wordCount == 2 {
			cmdName := words[0]
			helpDef := findCommandDefinition("help")
			if cmdName == "help" || (helpDef != nil && slices.Contains(helpDef.Aliases, cmdName)) {
				shouldSuggestCommands = true
			}
		}

		if shouldSuggestCommands {
			suggestions := make([]prompt.Suggest, 0, len(CommandTable)*2)
			for _, cmdDef := range CommandTable {
				suggestions = append(suggestions, prompt.Suggest{Text: cmdDef.Name, Description: cmdDef.Summary})
				for _, alias := range cmdDef.Aliases {
					suggestions = append(suggestions, prompt.Suggest{Text: alias, Description: cmdDef.Summary + " (alias for " + cmdDef.Name + ")"})
				}
			}
			return prompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
		}

		// 引数補完
		cmdName := words[0]
		cmdDef := findCommandDefinition(cmdName)
		if cmdDef != nil && cmdDef.GetCandidatesFunc != nil {
			return prompt.FilterHasPrefix(cmdDef.GetCandidatesFunc(c, d), d.GetWordBeforeCursor(), true)
		}

		// コマンドが見つからないか、補完関数がない場合
		return []prompt.Suggest{}
	}

	// go-prompt の設定と実行
	pt := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("> "),
		prompt.OptionTitle("echonet-list console"),
		prompt.OptionHistory(initialHistory), // 読み込んだ履歴を設定
		prompt.OptionCompletionWordSeparator(" "),
		prompt.OptionLivePrefix(func() (prefix string, useLivePrefix bool) {
			return "> ", true
		}),
		prompt.OptionSetExitCheckerOnInput(func(in string, breakLine bool) bool {
			// quit コマンドを入力した場合、プロンプトを終了
			return strings.TrimSpace(in) == "quit" && breakLine
		}),
	)

	pt.Run()

	saveHistory(historyFilePath, initialHistory)
}

// findCommandDefinition は CommandTable からコマンド定義を検索するヘルパー関数
// (重複定義を避けるため、ConsoleProcess.go 内に保持)
func findCommandDefinition(name string) *CommandDefinition {
	for i := range CommandTable {
		cmdDef := &CommandTable[i] // ポインタを取得してループ内で変更しないようにする
		if cmdDef.Name == name || slices.Contains(cmdDef.Aliases, name) {
			return cmdDef
		}
	}
	return nil
}
