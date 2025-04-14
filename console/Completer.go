package console

import (
	"echonet-list/client"

	"github.com/c-bata/go-prompt"
)

// --- 補完候補生成のためのヘルパー関数群 ---
// これらは CommandTable.go 内の GetCandidatesFunc や ConsoleProcess.go の completer から呼び出される

// getDeviceAliasCandidates はデバイスエイリアスの候補を返す
func getDeviceAliasCandidates(c client.ECHONETListClient) []prompt.Suggest {
	aliasList := c.AliasList()
	suggests := make([]prompt.Suggest, 0, len(aliasList))
	for _, pair := range aliasList {
		suggests = append(suggests, prompt.Suggest{
			Text: pair.Alias,
		})
	}
	return suggests
}

// getDeviceCandidates はデバイス指定子の候補（エイリアス、グループ、IP、EOJ）を返す
func getDeviceCandidates(c client.ECHONETListClient) []prompt.Suggest {
	aliases := getDeviceAliasCandidates(c)
	groups := getGroupCandidates(c)

	// IPアドレスとEOJを取得
	deviceSpec := client.DeviceSpecifier{}
	devices := c.GetDevices(deviceSpec)
	ips := make([]prompt.Suggest, 0, len(devices))
	eojs := make([]prompt.Suggest, 0, len(devices))

	uniqueIPs := make(map[string]struct{})
	uniqueEOJs := make(map[string]struct{})

	for _, device := range devices {
		ipStr := device.IP.String()
		if _, exists := uniqueIPs[ipStr]; !exists {
			ips = append(ips, prompt.Suggest{Text: ipStr})
			uniqueIPs[ipStr] = struct{}{}
		}

		eojStr := device.EOJ.Specifier()
		if _, exists := uniqueEOJs[eojStr]; !exists {
			eojs = append(eojs, prompt.Suggest{Text: eojStr})
			uniqueEOJs[eojStr] = struct{}{}
		}
	}

	// 候補を結合
	candidates := make([]prompt.Suggest, 0, len(aliases)+len(groups)+len(ips)+len(eojs))
	candidates = append(candidates, groups...)
	candidates = append(candidates, aliases...)
	candidates = append(candidates, ips...)
	candidates = append(candidates, eojs...)

	return candidates
}

// getPropertyAliasCandidates はプロパティエイリアスの候補を返す
func getPropertyAliasCandidates(c client.ECHONETListClient) []prompt.Suggest {
	aliases := c.GetAllPropertyAliases()
	suggests := make([]prompt.Suggest, 0, len(aliases))
	for _, alias := range aliases {
		classCode := client.EOJClassCode(0) // TODO
		prop, _ := c.FindPropertyAlias(classCode, alias)
		suggests = append(suggests, prompt.Suggest{
			Text:        alias,
			Description: prop.String(classCode),
		})
	}
	return suggests
}

// getGroupCandidates はグループ名の候補を返す
func getGroupCandidates(c client.ECHONETListClient) []prompt.Suggest {
	groups := c.GroupList(nil)
	suggests := make([]prompt.Suggest, 0, len(groups))
	for _, group := range groups {
		suggests = append(suggests, prompt.Suggest{
			Text: group.Group,
		})
	}
	return suggests
}

// splitWords は入力行を単語に分割する補助関数
// go-prompt の Document.GetWordBeforeCursor や Document.TextBeforeCursor と組み合わせて使う
func splitWords(line string) []string {
	// 空の入力の場合は空のスライスを返す
	if line == "" {
		return []string{}
	}

	words := make([]string, 0) // non-nil スライスとして初期化
	var word string
	inQuote := false
	lastWasSpace := true // 最初はスペースとみなす

	for _, r := range line {
		switch r {
		case ' ', '\t':
			if !inQuote {
				if !lastWasSpace && word != "" { // 直前がスペースでなく、単語がある場合のみ追加
					words = append(words, word)
					word = ""
				}
				lastWasSpace = true
			} else { // inQuote
				word += string(r)
				lastWasSpace = false // クォート内ではスペースも単語の一部
			}
		case '"', '\'':
			inQuote = !inQuote
			lastWasSpace = false
		default:
			word += string(r)
			lastWasSpace = false
		}
	}

	// 最後の単語を追加
	if word != "" {
		words = append(words, word)
	}

	// 末尾が空白だった場合、空の単語を1つだけ追加
	if lastWasSpace {
		words = append(words, "")
	}

	return words
}
