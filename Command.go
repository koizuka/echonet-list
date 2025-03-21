package main

import (
	"echonet-list/client"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

// コマンドの種類を表す型
type CommandType int

const (
	CmdUnknown CommandType = iota
	CmdQuit
	CmdDiscover
	CmdDevices
	CmdHelp
	CmdSet
	CmdGet
	CmdDebug
	CmdUpdate
	CmdAliasSet
	CmdAliasGet
	CmdAliasDelete
	CmdAliasList
)

// プロパティ表示モードを表す型
type PropertyMode int

const (
	PropDefault PropertyMode = iota // デフォルトのプロパティを表示
	PropKnown                       // 既知のプロパティのみ表示
	PropAll                         // 全てのプロパティを表示
	PropEPC                         // 特定のEPCのみ表示
)

// コマンドを表す構造体
type Command struct {
	Type        CommandType
	DeviceSpec  client.DeviceSpecifier // デバイス指定子
	DeviceAlias *string                // エイリアス
	EPCs        []client.EPCType       // devicesコマンドのEPCフィルター用。空の場合は全EPCを表示
	PropMode    PropertyMode           // プロパティ表示モード
	Properties  client.Properties      // set/devicesコマンドのプロパティリスト
	DebugMode   *string                // debugコマンドのモード ("on" または "off")
	Done        chan struct{}          // コマンド実行完了を通知するチャネル
	Error       error                  // コマンド実行中に発生したエラー
}

// GetIPAddress は、コマンドのIPアドレスを取得する
func (c *Command) GetIPAddress() *net.IP {
	return c.DeviceSpec.IP
}

// GetClassCode は、コマンドのクラスコードを取得する
func (c *Command) GetClassCode() *client.EOJClassCode {
	return c.DeviceSpec.ClassCode
}

// GetInstanceCode は、コマンドのインスタンスコードを取得する
func (c *Command) GetInstanceCode() *client.EOJInstanceCode {
	return c.DeviceSpec.InstanceCode
}

type CommandParser struct {
	propertyInfoProvider client.PropertyInfoProvider
	aliasManager         client.AliasManager
}

func NewCommandParser(propertyInfoProvider client.PropertyInfoProvider, aliasManager client.AliasManager) *CommandParser {
	return &CommandParser{
		propertyInfoProvider: propertyInfoProvider,
		aliasManager:         aliasManager,
	}
}

// IPアドレスをパースして、有効なIPアドレスならそのポインタを返す
// 無効なIPアドレスならnilを返す
func tryParseIPAddress(ipStr string) *net.IP {
	if ip := net.ParseIP(ipStr); ip != nil {
		return &ip
	}
	return nil
}

// クラスコードとインスタンスコードをパースする
func parseClassAndInstanceCode(codeStr string) (*client.EOJClassCode, *client.EOJInstanceCode, error) {
	codeParts := strings.Split(codeStr, ":")
	if len(codeParts) > 2 {
		return nil, nil, fmt.Errorf("invalid format: %s (use classCode or classCode:instanceCode)", codeStr)
	}

	// クラスコードのパース
	if len(codeParts[0]) != 4 {
		return nil, nil, fmt.Errorf("class code must be 4 hexadecimal digits")
	}
	classCode64, err := strconv.ParseUint(codeParts[0], 16, 16)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid class code: %s (must be 4 hexadecimal digits)", codeParts[0])
	}
	classCode := client.EOJClassCode(classCode64)

	// インスタンスコードのパース（存在する場合）
	var instanceCode *client.EOJInstanceCode
	if len(codeParts) == 2 {
		instanceCode64, err := strconv.ParseUint(codeParts[1], 10, 8)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid instance code: %s (must be a number between 1-255)", codeParts[1])
		}
		if instanceCode64 == 0 || instanceCode64 > 255 {
			return nil, nil, fmt.Errorf("instance code must be between 1 and 255")
		}
		code := client.EOJInstanceCode(instanceCode64)
		instanceCode = &code
	}

	return &classCode, instanceCode, nil
}

// 単一のEPCをパースする
func parseEPC(epcStr string) (client.EPCType, error) {
	if len(epcStr) != 2 {
		return 0, fmt.Errorf("EPC must be 2 hexadecimal digits: %s", epcStr)
	}
	epc64, err := strconv.ParseUint(epcStr, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid EPC: %s (must be 2 hexadecimal digits)", epcStr)
	}
	return client.EPCType(epc64), nil
}

func parseHexBytes(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		return nil, fmt.Errorf("hex string must be a multiple of 2 characters: %s", hexStr)
	}
	bytes := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		b, err := strconv.ParseUint(hexStr[i:i+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex byte: %s", hexStr[i:i+2])
		}
		bytes[i/2] = byte(b)
	}
	return bytes, nil
}

func (p CommandParser) GetEDTFromAlias(c client.EOJClassCode, e client.EPCType, alias string) ([]byte, bool) {
	if info, ok := p.propertyInfoProvider.GetPropertyInfo(c, e); ok && info.Aliases != nil {
		if aliases, ok := info.Aliases[alias]; ok {
			return aliases, true
		}
	}
	return nil, false
}

// プロパティ文字列をパースする
// propertyStr: プロパティ文字列（"EPC:EDT" 形式または "alias" 形式）
// classCode: クラスコード
// debug: デバッグフラグ
// 戻り値: パースされたプロパティとエラー
func (p CommandParser) parsePropertyString(propertyStr string, classCode client.EOJClassCode, debug bool) (client.Property, error) {
	// EPC:EDT の形式をパース
	propParts := strings.Split(propertyStr, ":")
	if len(propParts) == 2 {
		// EPCのパース
		epc, err := parseEPC(propParts[0])
		if err != nil {
			return client.Property{}, err
		}

		var edt []byte

		if aliasEDT, ok := p.GetEDTFromAlias(classCode, epc, propParts[1]); ok {
			if debug {
				fmt.Printf("エイリアス '%s' を EDT:%X に展開します\n", propParts[1], aliasEDT)
			}
			edt = aliasEDT
		}

		// エイリアスが見つからなかった場合は通常のEDTパース
		if edt == nil {
			edt, err = parseHexBytes(propParts[1])
			if err != nil {
				return client.Property{}, fmt.Errorf("EPC:%s: %v", propParts[0], err)
			}
		}

		return client.Property{EPC: epc, EDT: edt}, nil
	} else {
		// エイリアスのみの場合（例: "on"）
		alias := propertyStr

		if p, ok := p.propertyInfoProvider.FindPropertyAlias(classCode, alias); ok {
			if debug {
				fmt.Printf("エイリアス '%s' を EPC:%s, EDT:%X に展開します\n", alias, p.EPC, p.EDT)
			}
			return p, nil
		} else {
			return client.Property{}, fmt.Errorf("エイリアス '%s' が見つかりません。EPC:EDT 形式を使用してください", alias)
		}
	}
}

// parseDeviceSpecifier は、コマンド引数から DeviceSpecifier をパースする
// 引数:
//
//	parts: コマンドの引数配列
//	argIndex: パース開始位置
//	requireClassCode: クラスコードが必須かどうか
//
// 戻り値:
//
//	deviceSpecifier: パースされた DeviceSpecifier
//	nextArgIndex: 次の引数のインデックス
//	error: エラー
func (p CommandParser) parseDeviceSpecifier(parts []string, argIndex int, requireClassCode bool) (client.DeviceSpecifier, int, error) {
	var deviceSpec client.DeviceSpecifier
	if argIndex >= len(parts) {
		if requireClassCode {
			return deviceSpec, argIndex, fmt.Errorf("デバイス識別子が必要です")
		}
		return deviceSpec, argIndex, nil
	}

	// エイリアスの取得
	if alias, ok := p.aliasManager.GetDeviceByAlias(parts[argIndex]); ok {
		classCode := alias.EOJ.ClassCode()
		instanceCode := alias.EOJ.InstanceCode()
		deviceSpec := client.DeviceSpecifier{
			IP:           &alias.IP,
			ClassCode:    &classCode,
			InstanceCode: &instanceCode,
		}
		return deviceSpec, argIndex + 1, nil
	}

	// IPアドレスのパース（省略可能）- IPv4/IPv6に対応
	if ipAddr := tryParseIPAddress(parts[argIndex]); ipAddr != nil {
		deviceSpec.IP = ipAddr
		argIndex++

		if argIndex >= len(parts) && requireClassCode {
			return deviceSpec, argIndex, fmt.Errorf("クラスコードが必要です")
		} else if argIndex >= len(parts) {
			return deviceSpec, argIndex, nil
		}
	}

	// クラスコードとインスタンスコードのパース
	classCode, instanceCode, err := parseClassAndInstanceCode(parts[argIndex])
	if err != nil {
		if requireClassCode {
			return deviceSpec, argIndex, err
		}
		// クラスコードが必須でない場合は、パースエラーを無視して現在の引数を処理せずに返す
		return deviceSpec, argIndex, nil
	}

	deviceSpec.ClassCode = classCode
	deviceSpec.InstanceCode = instanceCode
	argIndex++

	return deviceSpec, argIndex, nil
}

// 基本的なコマンドオブジェクトを作成するヘルパー関数
func newCommand(cmdType CommandType) *Command {
	return &Command{
		Done: make(chan struct{}),
		Type: cmdType,
	}
}

type AvailableAliasesForAll struct {
	Aliases map[string]string
}

func (e *AvailableAliasesForAll) Error() string {
	messages := make([]string, 0, len(e.Aliases)+1)
	messages = append(messages, "利用可能なエイリアス:")

	// sort by alias
	sortedAliases := make([]string, 0, len(e.Aliases))
	for alias := range e.Aliases {
		sortedAliases = append(sortedAliases, alias)
	}
	sort.Strings(sortedAliases)

	for _, alias := range sortedAliases {
		messages = append(messages, fmt.Sprintf("  %s -> %v", alias, e.Aliases[alias]))
	}
	return strings.Join(messages, "\n")
}

type AvailableAliasesForEPC struct {
	EPC     client.EPCType
	Aliases map[string][]byte
}

func (e *AvailableAliasesForEPC) Error() string {
	if len(e.Aliases) == 0 {
		return fmt.Sprintf("EPC %s にはエイリアスが定義されていません", e.EPC)
	}
	messages := make([]string, 0, len(e.Aliases)+1)
	messages = append(messages, fmt.Sprintf("利用可能なエイリアス for EPC %s:", e.EPC))
	sortedAliases := make([]string, 0, len(e.Aliases))
	for alias := range e.Aliases {
		sortedAliases = append(sortedAliases, alias)
	}
	sort.Strings(sortedAliases)
	for _, alias := range sortedAliases {
		messages = append(messages, fmt.Sprintf("  %s -> %X", alias, e.Aliases[alias]))
	}
	return strings.Join(messages, "\n")
}

// "get" コマンドをパースする
func (p CommandParser) parseGetCommand(parts []string) (*Command, error) {
	cmd := newCommand(CmdGet)

	// デバイス識別子のパース
	deviceSpec, argIndex, err := p.parseDeviceSpecifier(parts, 1, true)
	if err != nil {
		return nil, err
	}
	cmd.DeviceSpec = deviceSpec

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
}

// "set" コマンドをパースする
func (p CommandParser) parseSetCommand(parts []string, debug bool) (*Command, error) {
	cmd := newCommand(CmdSet)

	// デバイス識別子のパース
	deviceSpec, argIndex, err := p.parseDeviceSpecifier(parts, 1, true)
	if err != nil {
		return nil, err
	}
	cmd.DeviceSpec = deviceSpec

	// プロパティのパース
	if argIndex >= len(parts) {
		// 可能なエイリアス一覧
		aliases := p.propertyInfoProvider.AvailablePropertyAliases(*cmd.GetClassCode())
		return nil, &AvailableAliasesForAll{Aliases: aliases}
	}

	for i := argIndex; i < len(parts); i++ {
		// EPCのみの場合（エイリアス一覧表示）
		epc, err := parseEPC(parts[i])
		if err == nil {
			// クラスコードからPropertyInfoを取得
			if propInfo, ok := p.propertyInfoProvider.GetPropertyInfo(*cmd.GetClassCode(), epc); ok && propInfo.Aliases != nil && len(propInfo.Aliases) > 0 {
				return nil, &AvailableAliasesForEPC{EPC: epc, Aliases: propInfo.Aliases}
			} else {
				return nil, &AvailableAliasesForEPC{EPC: epc}
			}
		}

		// プロパティ文字列をパース
		prop, err := p.parsePropertyString(parts[i], *cmd.GetClassCode(), debug)
		if err != nil {
			return nil, err
		}

		// プロパティを追加
		cmd.Properties = append(cmd.Properties, prop)
	}

	return cmd, nil
}

type InvalidArgument struct {
	Argument string
}

func (e *InvalidArgument) Error() string {
	return fmt.Sprintf("無効な引数: %s", e.Argument)
}

// "devices" または "list" コマンドをパースする
func (p CommandParser) parseDevicesCommand(parts []string) (*Command, error) {
	cmd := newCommand(CmdDevices)

	// デバイス識別子のパース
	deviceSpec, argIndex, err := p.parseDeviceSpecifier(parts, 1, false)
	if err != nil {
		return nil, err
	}
	cmd.DeviceSpec = deviceSpec

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
		}

		pClassCode := cmd.GetClassCode()
		if pClassCode == nil {
			pClassCode = new(client.EOJClassCode)
		}
		props, err := p.parsePropertyString(parts[i], *pClassCode, false) // corrected from classCode to *pClassCode
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
}

// "debug" コマンドをパースする
func (p CommandParser) parseDebugCommand(parts []string) (*Command, error) {
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
}

// "help" コマンドをパースする
func (p CommandParser) parseHelpCommand(parts []string) (*Command, error) {
	cmd := newCommand(CmdHelp)

	// 引数がある場合は、その特定のコマンドについてのヘルプを表示する
	if len(parts) > 1 {
		cmd.DeviceAlias = &parts[1] // コマンド名を DeviceAlias に格納
	}

	return cmd, nil
}

// "update" コマンドをパースする
func (p CommandParser) parseUpdateCommand(parts []string) (*Command, error) {
	cmd := newCommand(CmdUpdate)

	// デバイス識別子のパース
	deviceSpec, argIndex, err := p.parseDeviceSpecifier(parts, 1, false)
	if err != nil {
		return nil, err
	}
	cmd.DeviceSpec = deviceSpec

	// 残りの引数がある場合はエラー
	if argIndex < len(parts) {
		return nil, &InvalidArgument{Argument: parts[argIndex]}
	}

	return cmd, nil
}

// "alias" コマンドをパースする
// 登録する場合:
// syntax: alias _alias_ [_ipAddress_] _classCode_[:_instanceCode_]
// 削除する場合:
// syntax: alias -delete _alias_
// 表示する場合:
// syntax: alias _alias_
// 一覧する場合:
// syntax: alias
func (p CommandParser) parseAliasCommand(parts []string) (*Command, error) {
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
		if err := p.aliasManager.ValidateDeviceAlias(alias); err != nil {
			return nil, err
		}
		cmd.DeviceAlias = &alias
	} else {
		cmd.Type = CmdAliasSet

		// エイリアス名のパース
		alias := parts[1]
		if err := p.aliasManager.ValidateDeviceAlias(alias); err != nil {
			return nil, err
		}
		cmd.DeviceAlias = &alias

		// デバイス識別子のパース
		deviceSpec, argIndex, err := p.parseDeviceSpecifier(parts, 2, true)
		if err != nil {
			return nil, err
		}
		cmd.DeviceSpec = deviceSpec

		// 絞り込みプロパティ値のパース
		var classCode client.EOJClassCode
		if deviceSpec.ClassCode != nil {
			classCode = *deviceSpec.ClassCode
		}
		for {
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
}

// コマンドをパースする
func (p CommandParser) ParseCommand(input string, debug bool) (*Command, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil, nil
	}

	commandName := parts[0]

	// テーブルから一致するコマンドを探す
	for _, cmdDef := range CommandTable {
		if cmdDef.Name == commandName || slices.Contains(cmdDef.Aliases, commandName) {
			if cmdDef.ParseFunc != nil {
				return cmdDef.ParseFunc(p, parts, debug)
			}
			// ParseFuncが定義されていない場合はデフォルトのコマンドを返す
			return newCommand(CmdUnknown), nil
		}
	}

	return nil, fmt.Errorf("unknown command: %s", commandName)
}

// CommandDefinition はコマンドの定義を保持する構造体
type CommandDefinition struct {
	Name        string                                                              // コマンド名
	Aliases     []string                                                            // 別名（例: devicesとlistなど）
	Summary     string                                                              // 概要（短い説明）
	Syntax      string                                                              // 構文
	Description []string                                                            // 詳細説明（各行が1つの要素）
	ParseFunc   func(p CommandParser, parts []string, debug bool) (*Command, error) // パース関数
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
		Syntax:  "devices, list [ipAddress] [classCode[:instanceCode]] [-all|-props] [epc1 epc2...]",
		Description: []string{
			"ipAddress: IPアドレスでフィルター（例: 192.168.0.212 または IPv6アドレス）",
			"classCode: クラスコード（4桁の16進数、例: 0130）",
			"instanceCode: インスタンスコード（1-255の数字、例: 0130:1）",
			"-all: 全てのEPCを表示",
			"-props: 既知のEPCのみを表示",
			"epc: 2桁の16進数で指定（例: 80）。複数指定可能",
			"※-all, -props, epc は最後に指定されたものが有効になります",
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			return p.parseDevicesCommand(parts)
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
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			return p.parseGetCommand(parts)
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
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			return p.parseSetCommand(parts, debug)
		},
	},
	{
		Name:    "update",
		Summary: "デバイスのプロパティキャッシュを更新",
		Syntax:  "update [ipAddress] [classCode[:instanceCode]]",
		Description: []string{
			"ipAddress: 対象デバイスのIPアドレス（省略可能、省略時は全デバイスが対象）",
			"classCode: クラスコード（4桁の16進数、省略時は全クラスが対象）",
			"instanceCode: インスタンスコード（1-255の数字、省略時は指定クラスの全インスタンスが対象）",
		},
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			return p.parseUpdateCommand(parts)
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
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			return p.parseAliasCommand(parts)
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
		ParseFunc: func(p CommandParser, parts []string, debug bool) (*Command, error) {
			return p.parseDebugCommand(parts)
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
			return p.parseHelpCommand(parts)
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
