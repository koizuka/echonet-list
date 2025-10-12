package console

import (
	"echonet-list/client"
	"echonet-list/echonet_lite/handler"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c-bata/go-prompt"
	"golang.org/x/exp/slices"
)

// CommandDefinition はコマンドの定義を保持する構造体
type CommandDefinition struct {
	Name              string                                                               // コマンド名
	Aliases           []string                                                             // 別名（例: devicesとlistなど）
	Summary           string                                                               // 概要（短い説明）
	Syntax            string                                                               // 構文
	Description       []string                                                             // 詳細説明（各行が1つの要素）
	ParseFunc         func(p CommandParser, parts []string, debug bool) (*Command, error)  // パース関数
	GetCandidatesFunc func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest // 補完候補生成関数
}

// CommandTable はコマンドの定義を格納するテーブル
// コマンドの使用法に変化があったときは、README.md も更新すること
var CommandTable = []CommandDefinition{
	{
		Name:    "discover",
		Summary: "ECHONET Lite デバイスの検出",
		Syntax:  "discover",
		Description: []string{
			"ネットワーク上のECHONET Liteデバイスを検出します。",
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			return newCommand(CmdDiscover), nil
		},
	},
	{
		Name:    "devices",
		Aliases: []string{"list"},
		Summary: "検出されたECHONET Liteデバイスの一覧表示",
		Syntax:  "devices, list [ipAddress] [classCode[:instanceCode]] [-all|-props] [epc1 epc2...] [-group-by epc]",
		Description: []string{
			"ipAddress: IPアドレスでフィルター（例: 192.168.0.212 または IPv6アドレス）",
			"classCode: クラスコード（4桁の16進数、例: 0130）",
			"instanceCode: インスタンスコード（1-255の数字、例: 0130:1）",
			"-all: 全てのEPCを表示",
			"-props: 既知のEPCのみを表示",
			"epc: 2桁の16進数で指定（例: 80）。複数指定可能",
			"-group-by epc: 指定したEPCの値でデバイスをグループ化して表示（例: -group-by 80）",
			"※-all, -props, epc は最後に指定されたものが有効になります",
		},
		GetCandidatesFunc: func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest {
			words := splitWords(d.TextBeforeCursor())
			if len(words) >= 2 && words[len(words)-2] == "-group-by" {
				// EPCを要求するが、サジェストはしない
				return []prompt.Suggest{}
			}

			suggestions := []prompt.Suggest{
				{Text: "-all", Description: "全てのEPCを表示"},
				{Text: "-props", Description: "既知のEPCのみを表示"},
				{Text: "-group-by", Description: "指定EPCでグループ化"},
			}
			suggestions = append(suggestions, getDeviceCandidates(c)...)
			suggestions = append(suggestions, getPropertyAliasCandidates(c)...)
			return suggestions
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			cmd := newCommand(CmdDevices)

			// デバイス識別子のパース
			deviceSpec, groupName, argIndex, err := p.parseDeviceSpecifierOrGroup(parts, 1, false)
			if err != nil {
				return nil, err
			}
			cmd.DeviceSpec = deviceSpec
			cmd.GroupName = groupName

			// 残りの引数を解析
			for i := argIndex; i < len(parts); i++ {
				switch parts[i] {
				case "-all":
					cmd.PropMode = PropAll
					cmd.EPCs = nil // EPCsをクリア
					continue
				case "-props":
					cmd.PropMode = PropKnown
					cmd.EPCs = nil // EPCsをクリア
					continue
				case "-group-by":
					// -group-by の次の引数がEPCであることを確認
					if i+1 >= len(parts) {
						return nil, fmt.Errorf("-group-by オプションにはEPCが必要です")
					}
					epc, err := parseEPC(parts[i+1])
					if err != nil {
						return nil, fmt.Errorf("-group-by オプションの引数が無効です: %v", err)
					}
					cmd.GroupByEPC = &epc
					i++ // 次の引数（EPC）をスキップ
					continue
				}

				classCode := cmd.GetClassCode()
				props, err := p.parsePropertyString(parts[i], classCode, false)
				if err == nil {
					cmd.Properties = append(cmd.Properties, props)
					continue
				}

				// EPCのパース（2桁の16進数）
				epc, err := parseEPC(parts[i])
				if err == nil {
					cmd.EPCs = append(cmd.EPCs, epc)
					cmd.PropMode = PropEPC
					continue
				}

				// 上記のいずれにも該当しない場合はエラー
				return nil, &InvalidArgument{Argument: parts[i]}
			}

			return cmd, nil
		},
	},
	{
		Name:    "get",
		Summary: "プロパティ値の取得",
		Syntax:  "get [ipAddress] classCode[:instanceCode] epc1 [epc2...] [-skip-validation]",
		Description: []string{
			"ipAddress: 対象デバイスのIPアドレス（省略可能、省略時はクラスコードに一致するデバイスが1つだけの場合に自動選択）",
			"classCode: クラスコード（4桁の16進数、必須）",
			"instanceCode: インスタンスコード（1-255の数字、省略時は1）",
			"epc: 取得するプロパティのEPC（2桁の16進数、例: 80）。複数指定可能",
			"-skip-validation: デバイスの存在チェックをスキップ（タイムアウト動作確認用）",
		},
		GetCandidatesFunc: func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest {
			words := splitWords(d.TextBeforeCursor())
			wordCount := len(words)

			if wordCount <= 2 { // コマンド名 or デバイス指定子
				return getDeviceCandidates(c)
			} else { // EPC or プロパティエイリアス or オプション
				suggestions := getPropertyAliasCandidates(c)
				suggestions = append(suggestions, prompt.Suggest{Text: "-skip-validation"})
				return suggestions
			}
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			cmd := newCommand(CmdGet)

			// デバイス識別子またはグループ名のパース
			deviceSpec, groupName, argIndex, err := p.parseDeviceSpecifierOrGroup(parts, 1, true)
			if err != nil {
				return nil, err
			}
			cmd.DeviceSpec = deviceSpec
			cmd.GroupName = groupName

			// EPCのパース
			if argIndex >= len(parts) {
				return nil, fmt.Errorf("get コマンドには少なくとも1つのEPCが必要です")
			}

			for i := argIndex; i < len(parts); i++ {
				if parts[i] == "-skip-validation" {
					cmd.DebugMode = &parts[i]
					continue
				}
				epc, err := parseEPC(parts[i])
				if err != nil {
					return nil, err
				}
				cmd.EPCs = append(cmd.EPCs, epc)
			}

			return cmd, nil
		},
	},
	{
		Name:    "history",
		Summary: "デバイスの履歴を表示",
		Syntax:  "history [ipAddress] classCode[:instanceCode] [-limit N] [-since RFC3339] [-all]",
		Description: []string{
			"デバイスの操作履歴を新しい順に表示します。",
			"ipAddress/classCode[:instanceCode]: 対象デバイスの指定（エイリアス指定も可）",
			"-limit N: 取得する履歴件数の上限（既定 50）",
			"-since RFC3339: 指定した時刻以降の履歴のみ表示（例: 2024-05-01T12:00:00Z）",
			"-all: Set Property Map に含まれない通知も表示（既定では書き込み可能なプロパティのみ）",
		},
		GetCandidatesFunc: func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest {
			suggestions := []prompt.Suggest{
				{Text: "-limit", Description: "取得件数の上限を指定"},
				{Text: "-since", Description: "この時刻以降の履歴を取得"},
				{Text: "-all", Description: "すべての履歴（センサー値など）を含める"},
			}
			suggestions = append(suggestions, getDeviceCandidates(c)...)
			return suggestions
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			cmd := newCommand(CmdHistory)

			deviceSpec, groupName, argIndex, err := p.parseDeviceSpecifierOrGroup(parts, 1, false)
			if err != nil {
				return nil, err
			}
			if groupName != nil {
				return nil, fmt.Errorf("history コマンドはグループ指定に対応していません")
			}
			cmd.DeviceSpec = deviceSpec

			if deviceSpec.IP == nil && deviceSpec.ClassCode == nil {
				return nil, fmt.Errorf("history コマンドにはデバイスの指定が必要です")
			}

			for argIndex < len(parts) {
				switch parts[argIndex] {
				case "-limit":
					if argIndex+1 >= len(parts) {
						return nil, fmt.Errorf("-limit オプションには数値が必要です")
					}
					value, err := strconv.Atoi(parts[argIndex+1])
					if err != nil || value <= 0 {
						return nil, fmt.Errorf("-limit には1以上の整数を指定してください")
					}
					cmd.HistoryOptions.Limit = value
					argIndex += 2
				case "-since":
					if argIndex+1 >= len(parts) {
						return nil, fmt.Errorf("-since オプションには時刻が必要です")
					}
					sinceStr := parts[argIndex+1]
					timestamp, err := time.Parse(time.RFC3339Nano, sinceStr)
					if err != nil {
						timestamp, err = time.Parse(time.RFC3339, sinceStr)
					}
					if err != nil {
						return nil, fmt.Errorf("-since の形式が不正です (RFC3339): %v", err)
					}
					cmd.HistoryOptions.Since = &timestamp
					argIndex += 2
				case "-all":
					settable := false
					cmd.HistoryOptions.SettableOnly = &settable
					argIndex++
				default:
					return nil, &InvalidArgument{Argument: parts[argIndex]}
				}
			}

			return cmd, nil
		},
	},
	{
		Name:    "set",
		Summary: "プロパティ値の設定",
		Syntax:  "set [ipAddress] classCode[:instanceCode] property1 [property2...]",
		Description: []string{
			"ipAddress: 対象デバイスのIPアドレス（省略可能、省略時はクラスコードに一致するデバイスが1つだけの場合に自動選択）",
			"classCode: クラスコード（4桁の16進数、必須）",
			"instanceCode: インスタンスコード（1-255の数字、省略時は1）",
			"property: 以下のいずれかの形式",
			"  - EPC:EDT（例: 80:30）",
			"    EPC: 2桁の16進数",
			"    EDT: 2桁の16進数の倍数またはエイリアス名",
			"  - EPC（例: 80）- 利用可能なエイリアスを表示",
			"  - エイリアス名（例: on）- 対応するEPC:EDTに自動展開",
			"  - 80:on（OperationStatus{true}と同等）",
			"  - 80:off（OperationStatus{false}と同等）",
			"  - b0:auto（エアコンの自動モードと同等）",
		},
		GetCandidatesFunc: func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest {
			words := splitWords(d.TextBeforeCursor())
			wordCount := len(words)

			if wordCount <= 2 { // コマンド名 or デバイス指定子
				return getDeviceCandidates(c)
			} else { // プロパティ指定 (EPC:EDT or Alias)
				lastWord := words[len(words)-1]
				parts := strings.Split(lastWord, ":")
				if len(parts) == 2 {
					if epc, err := parseEPC(parts[0]); err == nil {
						// epc に使えるaliasを列挙する
						return getPropertyAliasCandidatesForEPC(c, epc, parts[0]+":")
					}
				}
				return getPropertyAliasCandidates(c)
			}
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			cmd := newCommand(CmdSet)

			// デバイス識別子またはグループ名のパース
			deviceSpec, groupName, argIndex, err := p.parseDeviceSpecifierOrGroup(parts, 1, true)
			if err != nil {
				return nil, err
			}
			cmd.DeviceSpec = deviceSpec
			cmd.GroupName = groupName

			// プロパティのパース
			if argIndex >= len(parts) {
				// デバイスが指定されている場合は、プロパティが必要というエラーを返す
				if (cmd.DeviceSpec.IP != nil || cmd.DeviceSpec.ClassCode != nil || cmd.DeviceSpec.InstanceCode != nil) ||
					cmd.GroupName != nil {
					return nil, errors.New("set コマンドには少なくとも1つのプロパティが必要です")
				}

				// デバイスが指定されていない場合は、エイリアス一覧を表示
				return nil, errors.New("デバイスまたはグループが指定されていません")
			}

			for i := argIndex; i < len(parts); i++ {
				// EPCのみの場合（エイリアス一覧表示）
				epc, err := parseEPC(parts[i])
				if err == nil {
					// クラスコードからPropertyDescを取得
					classCode := cmd.GetClassCode()
					if propDesc, ok := p.propertyDescProvider.GetPropertyDesc(classCode, epc); ok && propDesc.Aliases != nil && len(propDesc.Aliases) > 0 {
						return nil, &AvailableAliasesForEPC{EPC: epc, Aliases: propDesc.Aliases}
					} else {
						return nil, &AvailableAliasesForEPC{EPC: epc}
					}
				}

				// プロパティ文字列をパース
				classCode := cmd.GetClassCode()
				prop, err := p.parsePropertyString(parts[i], classCode, debug)
				if err != nil {
					return nil, err
				}

				// プロパティを追加
				cmd.Properties = append(cmd.Properties, prop)
			}

			return cmd, nil
		},
	},
	{
		Name:    "update",
		Summary: "デバイスのプロパティキャッシュを更新",
		Syntax:  "update [ipAddress] [classCode[:instanceCode]] [-force]",
		Description: []string{
			"ipAddress: 対象デバイスのIPアドレス（省略可能、省略時は全デバイスが対象）",
			"classCode: クラスコード（4桁の16進数、省略時は全クラスが対象）",
			"instanceCode: インスタンスコード（1-255の数字、省略時は指定クラスの全インスタンスが対象）",
			"-force: 最終更新時刻に関わらず強制的に更新",
		},
		GetCandidatesFunc: func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest {
			words := splitWords(d.TextBeforeCursor())
			wordCount := len(words)

			suggestions := []prompt.Suggest{}
			if wordCount <= 2 {
				suggestions = append(suggestions, getDeviceCandidates(c)...)
			}
			suggestions = append(suggestions, prompt.Suggest{Text: "-force"})
			return suggestions
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			cmd := newCommand(CmdUpdate)

			// デバイス識別子またはグループ名のパース
			deviceSpec, groupName, argIndex, err := p.parseDeviceSpecifierOrGroup(parts, 1, false)
			if err != nil {
				return nil, err
			}
			cmd.DeviceSpec = deviceSpec
			cmd.GroupName = groupName

			// 残りの引数を解析
			for i := argIndex; i < len(parts); i++ {
				switch parts[i] {
				case "-f", "-force":
					cmd.ForceUpdate = true
				default:
					return nil, &InvalidArgument{Argument: parts[i]}
				}
			}

			return cmd, nil
		},
	},
	{
		Name:    "alias",
		Summary: "デバイスエイリアスの管理",
		Syntax:  "alias [alias] [ipAddress] [classCode[:instanceCode]] [property...] | alias -delete alias",
		Description: []string{
			"引数なし: 登録済みのエイリアス一覧を表示",
			"alias: エイリアス名（例: ac）",
			"-delete: エイリアスを削除（例: alias -delete ac）",
			"ipAddress: 対象デバイスのIPアドレス（省略可能、省略時はクラスコードに一致するデバイスが1つだけの場合に自動選択）",
			"classCode: クラスコード（4桁の16進数）",
			"instanceCode: インスタンスコード（1-255の数字、省略時は1）",
			"property: プロパティ値による絞り込み（例: living1 - 設置場所が'living1'のデバイスを指定）",
			"例: alias ac 192.168.0.3 0130:1 - IPアドレス192.168.0.3、クラスコード0130、インスタンスコード1のデバイスに「ac」というエイリアスを設定",
			"例: alias ac 0130 - クラスコード0130のデバイスに「ac」というエイリアスを設定（デバイスが1つだけの場合）",
			"例: alias aircon1 0130 living1 - クラスコード0130で設置場所が'living1'のデバイスに「aircon1」というエイリアスを設定",
			"例: alias ac - 「ac」というエイリアスの情報を表示",
			"例: alias -delete ac - 「ac」というエイリアスを削除",
		},
		GetCandidatesFunc: func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest {
			words := splitWords(d.TextBeforeCursor())
			wordCount := len(words)
			suggestions := []prompt.Suggest{}

			if wordCount == 2 { // サブコマンド (-delete) or 新しいエイリアス名 or 既存エイリアス名
				suggestions = append(suggestions, prompt.Suggest{Text: "-delete"})
				suggestions = append(suggestions, getDeviceAliasCandidates(c)...)
			} else if wordCount == 3 && words[1] == "-delete" { // alias -delete <alias_to_delete>
				suggestions = append(suggestions, getDeviceAliasCandidates(c)...)
			} else if wordCount >= 3 && words[1] != "-delete" { // alias <name> <device_specifier>...
				suggestions = append(suggestions, getDeviceCandidates(c)...)
				// TODO: プロパティによる絞り込みの補完 (例: 'living1')
			}
			return suggestions
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			cmd := newCommand(CmdAliasList)

			// 引数がない場合はエイリアス一覧を表示する
			if len(parts) == 1 {
				cmd.Type = CmdAliasList
			} else if parts[1] == "-delete" {
				if len(parts) != 3 {
					return nil, fmt.Errorf("エイリアスの削除にはエイリアス名が必要です")
				}
				cmd.Type = CmdAliasDelete
				cmd.DeviceAlias = &parts[2]
				return cmd, nil
			} else if len(parts) == 2 {
				cmd.Type = CmdAliasGet

				// エイリアス名のパース
				alias := parts[1]
				if err := handler.ValidateDeviceAlias(alias); err != nil {
					return nil, err
				}
				cmd.DeviceAlias = &alias
			} else {
				cmd.Type = CmdAliasSet

				// エイリアス名のパース
				alias := parts[1]
				if err := handler.ValidateDeviceAlias(alias); err != nil {
					return nil, err
				}
				cmd.DeviceAlias = &alias

				// デバイス識別子のパース
				deviceSpec, groupName, argIndex, err := p.parseDeviceSpecifierOrGroup(parts, 2, true)
				if err != nil {
					return nil, err
				}
				cmd.DeviceSpec = deviceSpec
				cmd.GroupName = groupName

				// 絞り込みプロパティ値のパース
				var classCode client.EOJClassCode
				if deviceSpec.ClassCode != nil {
					classCode = *deviceSpec.ClassCode
				}
				for {
					if argIndex >= len(parts) {
						break
					}
					props, err := p.parsePropertyString(parts[argIndex], classCode, false)
					if err != nil {
						break
					}
					cmd.Properties = append(cmd.Properties, props)
					argIndex++
				}

				// 残りの引数がある場合はエラー
				if argIndex < len(parts) {
					return nil, &InvalidArgument{Argument: parts[argIndex]}
				}
			}

			return cmd, nil
		},
	},
	{
		Name:    "debug",
		Summary: "デバッグモードの表示または切り替え",
		Syntax:  "debug [on|off]",
		Description: []string{
			"引数なし: 現在のデバッグモードを表示",
			"on: デバッグモードを有効にする",
			"off: デバッグモードを無効にする",
		},
		GetCandidatesFunc: func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest {
			words := splitWords(d.TextBeforeCursor())
			wordCount := len(words)
			if wordCount == 2 {
				return []prompt.Suggest{
					{Text: "on", Description: "デバッグモード有効"},
					{Text: "off", Description: "デバッグモード無効"},
				}
			}
			return []prompt.Suggest{}
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			cmd := newCommand(CmdDebug)

			// 引数がない場合は現在のデバッグモードを表示するためにnilのままにする
			if len(parts) == 1 {
				return cmd, nil
			}

			// 引数を解析
			if len(parts) != 2 || (parts[1] != "on" && parts[1] != "off") {
				return nil, fmt.Errorf("debug コマンドの引数は on または off のみ有効です")
			}
			// on/off の値を DebugMode フィールドに格納する
			value := parts[1]
			cmd.DebugMode = &value

			return cmd, nil
		},
	},
	{
		Name:    "debugoffline",
		Summary: "デバイスのオフライン状態を手動で設定（デバッグ用）",
		Syntax:  "debugoffline [ipAddress] classCode[:instanceCode] [on|off]",
		Description: []string{
			"ipAddress: 対象デバイスのIPアドレス（省略可能、省略時はクラスコードに一致するデバイスが1つだけの場合に自動選択）",
			"classCode: クラスコード（4桁の16進数、必須）",
			"instanceCode: インスタンスコード（1-255の数字、省略時は1）",
			"on: デバイスをオフライン状態に設定",
			"off: デバイスをオンライン状態に設定",
			"引数なし: オフライン状態のトグル",
			"注意: これはデバッグ専用の機能です。実際のデバイス状態は変わりません。",
		},
		GetCandidatesFunc: func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest {
			words := splitWords(d.TextBeforeCursor())
			wordCount := len(words)

			if wordCount <= 2 { // コマンド名 or デバイス指定子
				return getDeviceCandidates(c)
			} else { // on/off
				return []prompt.Suggest{
					{Text: "on", Description: "オフライン状態に設定"},
					{Text: "off", Description: "オンライン状態に設定"},
				}
			}
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			cmd := newCommand(CmdDebugOffline)

			// デバイス識別子のパース
			deviceSpec, groupName, argIndex, err := p.parseDeviceSpecifierOrGroup(parts, 1, true)
			if err != nil {
				return nil, err
			}
			cmd.DeviceSpec = deviceSpec
			cmd.GroupName = groupName

			// on/off の解析（省略可能）
			if argIndex < len(parts) {
				switch parts[argIndex] {
				case "on":
					offlineFlag := "on"
					cmd.DebugMode = &offlineFlag
				case "off":
					offlineFlag := "off"
					cmd.DebugMode = &offlineFlag
				default:
					return nil, fmt.Errorf("debugoffline コマンドの引数は on または off のみ有効です")
				}
			}

			return cmd, nil
		},
	},
	{
		Name:    "help",
		Summary: "ヘルプを表示",
		Syntax:  "help [command]",
		Description: []string{
			"引数なし: 全コマンドの概要を表示",
			"command: 指定したコマンドの詳細を表示",
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			cmd := newCommand(CmdHelp)

			// 引数がある場合は、その特定のコマンドについてのヘルプを表示する
			if len(parts) > 1 {
				cmd.DeviceAlias = &parts[1] // コマンド名を DeviceAlias に格納
			}

			return cmd, nil
		},
	},
	{
		Name:    "group",
		Summary: "デバイスグループの管理",
		Syntax:  "group add|remove|delete|list [@groupName] [deviceId1 deviceId2...]",
		Description: []string{
			"add: グループを作成し、デバイスを追加します",
			"remove: グループからデバイスを削除します",
			"delete: グループを削除します",
			"list: グループの一覧または詳細を表示します",
			"@groupName: グループ名（@で始まる必要があります）",
			"deviceId: デバイスID（IPアドレス、クラスコード、インスタンスコード、またはエイリアス）",
			"例: group add @livingroom 192.168.0.3 0130:1 ac",
			"例: group remove @livingroom 192.168.0.3 0130:1",
			"例: group delete @livingroom",
			"例: group list",
			"例: group list @livingroom",
		},
		GetCandidatesFunc: func(c client.ECHONETListClient, d prompt.Document) []prompt.Suggest {
			words := splitWords(d.TextBeforeCursor())
			wordCount := len(words)
			suggestions := []prompt.Suggest{}

			if wordCount == 2 { // サブコマンド (add, remove, delete, list)
				suggestions = []prompt.Suggest{
					{Text: "add", Description: "グループ作成/デバイス追加"},
					{Text: "remove", Description: "グループからデバイス削除"},
					{Text: "delete", Description: "グループ削除"},
					{Text: "list", Description: "グループ一覧/詳細表示"},
				}
			} else if wordCount == 3 { // グループ名 (@group)
				suggestions = append(suggestions, getGroupCandidates(c)...)
			} else if wordCount >= 4 && (words[1] == "add" || words[1] == "remove") { // デバイス指定子
				suggestions = append(suggestions, getDeviceCandidates(c)...)
			}
			return suggestions
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			if len(parts) < 2 {
				return nil, fmt.Errorf("group コマンドにはサブコマンドが必要です")
			}

			var cmd *Command

			switch parts[1] {
			case "add":
				if len(parts) < 3 {
					return nil, fmt.Errorf("group add コマンドにはグループ名が必要です")
				}
				groupName := parts[2]
				if err := handler.ValidateGroupName(groupName); err != nil {
					return nil, err
				}

				cmd = newCommand(CmdGroupAdd)
				cmd.GroupName = &groupName

				// デバイス指定子のパース
				var deviceSpecs []client.DeviceSpecifier
				argIndex := 3
				for argIndex < len(parts) {
					devs, nextArgIndex, err := p.parseDeviceSpecifiers(parts, argIndex, true)
					if err != nil {
						return nil, err
					}
					deviceSpecs = append(deviceSpecs, devs...)
					argIndex = nextArgIndex
				}

				if len(deviceSpecs) == 0 {
					return nil, fmt.Errorf("group add コマンドには少なくとも1つのデバイスが必要です")
				}

				cmd.DeviceSpecs = deviceSpecs

			case "remove":
				if len(parts) < 3 {
					return nil, fmt.Errorf("group remove コマンドにはグループ名が必要です")
				}
				groupName := parts[2]
				if err := handler.ValidateGroupName(groupName); err != nil {
					return nil, err
				}

				cmd = newCommand(CmdGroupRemove)
				cmd.GroupName = &groupName

				// デバイス指定子のパース
				var deviceSpecs []client.DeviceSpecifier
				argIndex := 3
				for argIndex < len(parts) {
					devs, nextArgIndex, err := p.parseDeviceSpecifiers(parts, argIndex, true)
					if err != nil {
						return nil, err
					}
					deviceSpecs = append(deviceSpecs, devs...)
					argIndex = nextArgIndex
				}

				if len(deviceSpecs) == 0 {
					return nil, fmt.Errorf("group remove コマンドには少なくとも1つのデバイスが必要です")
				}

				cmd.DeviceSpecs = deviceSpecs

			case "delete":
				if len(parts) != 3 {
					return nil, fmt.Errorf("group delete コマンドにはグループ名のみが必要です")
				}
				groupName := parts[2]
				if err := handler.ValidateGroupName(groupName); err != nil {
					return nil, err
				}

				cmd = newCommand(CmdGroupDelete)
				cmd.GroupName = &groupName

			case "list":
				cmd = newCommand(CmdGroupList)
				if len(parts) > 2 {
					groupName := parts[2]
					if err := handler.ValidateGroupName(groupName); err != nil {
						return nil, err
					}
					cmd.GroupName = &groupName
				}

			default:
				return nil, fmt.Errorf("不明なサブコマンド: %s", parts[1])
			}

			return cmd, nil
		},
	},
	{
		Name:    "quit",
		Summary: "終了",
		Syntax:  "quit",
		Description: []string{
			"プログラムを終了します。",
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			return newCommand(CmdQuit), nil
		},
	},
}

// PrintCommandSummary は、全コマンドの簡単なサマリーを表示する
func PrintCommandSummary() {
	fmt.Println("コマンド:")

	// テーブルからサマリーを表示
	for _, cmd := range CommandTable {
		aliases := ""
		if len(cmd.Aliases) > 0 {
			aliases = fmt.Sprintf(", %s", strings.Join(cmd.Aliases, ", "))
		}
		fmt.Printf("  %-10s: %s\n", cmd.Name+aliases, cmd.Summary)
	}

	fmt.Println("")
	fmt.Println("詳細は 'help <コマンド名>' で確認できます。例: 'help get'")
}

// PrintCommandDetail は、特定のコマンドの詳細情報を表示する
func PrintCommandDetail(commandName string) {
	// テーブルから指定されたコマンドを検索
	for _, cmd := range CommandTable {
		if cmd.Name == commandName || slices.Contains(cmd.Aliases, commandName) {
			fmt.Printf("  %s: %s\n", cmd.Name, cmd.Summary)
			fmt.Printf("  構文: %s\n", cmd.Syntax)

			if len(cmd.Description) > 0 {
				fmt.Println("  詳細:")
				for _, line := range cmd.Description {
					fmt.Printf("    %s\n", line)
				}
			}
			return
		}
	}

	// コマンドが見つからなかった場合
	fmt.Printf("不明なコマンド: %s\n", commandName)
	fmt.Println("利用可能なコマンドを確認するには 'help' を入力してください")
}

// コマンドの使用方法を表示する
func PrintUsage(commandName *string) {
	if commandName == nil {
		// 引数無しの場合はタイトルとサマリーを表示
		fmt.Println("ECHONET Lite デバイス検出プログラム")
		PrintCommandSummary()
	} else {
		// 特定のコマンドの詳細を表示（タイトルなし）
		PrintCommandDetail(*commandName)
	}
}
