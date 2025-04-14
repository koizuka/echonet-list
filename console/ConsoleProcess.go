package console

import (
	"context"
	"echonet-list/client"
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"golang.org/x/exp/slices"
)

func ConsoleProcess(ctx context.Context, c client.ECHONETListClient) {
	// コマンドプロセッサの作成と開始
	processor := NewCommandProcessor(ctx, c)
	processor.Start()
	// defer processor.Stop() は不要。明示的に呼び出すため

	// コマンドの使用方法を表示
	fmt.Println("help for usage, quit to exit")

	// コマンドパーサーの作成
	p := NewCommandParser(c, c, c)

	// Executor: ユーザーがEnterを押したときに実行される関数
	executor := func(line string) {
		// quit コマンドの特別な処理
		if strings.TrimSpace(line) == "quit" {
			fmt.Println("Exiting...")
			processor.Stop()
			os.Exit(0) // go-prompt を終了させるためにプロセスを終了
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

		// quit コマンドは上で処理済みなので、ここでは考慮不要

		// コマンドを送信し、エラーをチェック
		if err := processor.SendCommand(cmd); err != nil {
			fmt.Printf("エラー: %v\n", err)
		}
		// コマンド完了待機は processor 内部で行われる
	}

	// Completer: 入力中に補完候補を返す関数
	completer := func(d prompt.Document) []prompt.Suggest {
		lineStr := d.TextBeforeCursor()
		lastWord := d.GetWordBeforeCursor()
		if lastWord == "" {
			// 最後の単語が空の場合は補完候補を返さない
			return []prompt.Suggest{}
		}

		words := splitWords(lineStr)
		wordCount := len(words)

		if wordCount <= 1 {
			// コマンド名の候補を生成
			return prompt.FilterHasPrefix(getCandidatesForCommand(), d.GetWordBeforeCursor(), false)
		}

		cmdName := words[0]

		// Special case: help command argument completion
		isHelpCommand := cmdName == "help"
		if isHelpCommand && wordCount == 2 { // help の最初の引数を入力中 (e.g., "help ", "help d")
			// コマンド名の候補を生成 (help の引数として)
			return prompt.FilterHasPrefix(getCandidatesForCommand(), d.GetWordBeforeCursor(), false)
		}

		// Other commands: Delegate to GetCandidatesFunc defined in CommandTable
		for _, cmdDef := range CommandTable {
			if cmdDef.Name == cmdName || slices.Contains(cmdDef.Aliases, cmdName) {
				if cmdDef.GetCandidatesFunc != nil {
					return prompt.FilterHasPrefix(cmdDef.GetCandidatesFunc(c, d), lastWord, false)
				}
				return []prompt.Suggest{}
			}
		}

		// Command not found
		return []prompt.Suggest{}
	}

	// 履歴ファイルのパス設定は go-prompt では直接行わない
	// historyFile := ".echonet_history"
	// if home, err := os.UserHomeDir(); err == nil {
	// 	historyFile = fmt.Sprintf("%s/.echonet_history", home)
	// }
	// TODO: 履歴の永続化が必要な場合は、go-prompt の OptionHistory とファイル I/O を組み合わせる

	// go-prompt の設定と実行
	pt := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("> "),
		prompt.OptionTitle("echonet-list console"),
		// prompt.OptionHistory( /* 履歴機能の設定 */ ), // 必要に応じて履歴を設定
		prompt.OptionCompletionWordSeparator(" "), // 補完の区切り文字
	)

	fmt.Println("Starting interactive console...")
	pt.Run()

	// pt.Run() は通常、Ctrl+C や quit コマンドで終了するまでブロックする
	// プログラム終了処理は executor 内の os.Exit で行う
	fmt.Println("Console finished.") // 通常ここには到達しない
}

// getCandidatesForCommand は、すべてのコマンドの補完候補を取得する
func getCandidatesForCommand() []prompt.Suggest {
	suggestions := make([]prompt.Suggest, 0, len(CommandTable)*2)
	for _, cmdDef := range CommandTable {
		suggestions = append(suggestions, prompt.Suggest{Text: cmdDef.Name, Description: cmdDef.Summary})
		for _, alias := range cmdDef.Aliases {
			suggestions = append(suggestions, prompt.Suggest{Text: alias, Description: cmdDef.Summary + " (alias for " + cmdDef.Name + ")"})
		}
	}
	return suggestions
}
