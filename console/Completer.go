package console

import (
	"echonet-list/client"
	"strings"

	"golang.org/x/exp/slices"
)

// カスタム補完機能を実装する構造体
type dynamicCompleter struct {
	client client.ECHONETListClient
}

// CompleterInterface を実装していることを確認
var _ CompleterInterface = (*dynamicCompleter)(nil)

// Do メソッドを実装して readline.AutoCompleter インターフェースを満たす
func (dc *dynamicCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	// 現在の入力行を解析して、入力段階を判断する
	lineStr := string(line[:pos])
	words := splitWords(lineStr)
	wordCount := len(words)

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
		candidates = getCandidatesForCommand(dc, cmd, wordCount, words)
	}

	// 最後の単語でフィルタリングして返す
	result := [][]rune{}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, lastWord) {
			result = append(result, []rune(candidate[len(lastWord):]+" "))
		}
	}
	return result, len(lastWord)
}

// コマンド名の候補を返す
func (dc *dynamicCompleter) getCommandCandidates() []string {
	var candidates []string
	for _, cmdDef := range CommandTable {
		candidates = append(candidates, cmdDef.Name)
		candidates = append(candidates, cmdDef.Aliases...)
	}
	return candidates
}

// デバイスエイリアスの候補を返す
func (dc *dynamicCompleter) getDeviceAliasCandidates() []string {
	// aliasList からエイリアスを取得
	aliasList := dc.client.AliasList()
	aliases := make([]string, 0, len(aliasList))
	for _, pair := range aliasList {
		aliases = append(aliases, pair.Alias)
	}
	return aliases
}

// デバイスの候補を返す
func (dc *dynamicCompleter) getDeviceCandidates() []string {
	// aliasList からエイリアスを取得
	aliases := dc.getDeviceAliasCandidates()

	// IPアドレスを取得
	deviceSpec := client.DeviceSpecifier{}
	devices := dc.client.GetDevices(deviceSpec)
	ips := make([]string, 0, len(devices))
	for _, device := range devices {
		ip := device.IP.String()
		if !slices.Contains(ips, ip) {
			ips = append(ips, ip)
		}
	}

	// EOJを取得
	eojs := make([]string, 0, len(devices))
	for _, device := range devices {
		eoj := device.EOJ.Specifier()
		if !slices.Contains(eojs, eoj) {
			eojs = append(eojs, eoj)
		}
	}

	candidates := make([]string, 0, len(aliases)+len(ips)+len(eojs))
	candidates = append(candidates, aliases...)
	candidates = append(candidates, ips...)
	candidates = append(candidates, eojs...)

	return candidates
}

// プロパティエイリアスの候補を返す
func (dc *dynamicCompleter) getPropertyAliasCandidates() []string {
	return dc.client.GetAllPropertyAliases()
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

// getCandidatesForCommand はコマンドと引数位置に応じた候補を返す
func getCandidatesForCommand(dc CompleterInterface, cmd string, wordCount int, words []string) []string {
	// コマンド名に一致するCommandDefinitionを検索
	for _, cmdDef := range CommandTable {
		if cmdDef.Name == cmd || slices.Contains(cmdDef.Aliases, cmd) {
			// 該当するコマンドの補完関数が定義されていれば呼び出す
			if cmdDef.GetCandidatesFunc != nil {
				return cmdDef.GetCandidatesFunc(dc, wordCount, words)
			}
			break
		}
	}
	return []string{} // デフォルトは空リスト
}
