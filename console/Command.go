package console

import (
	"echonet-list/client"
	"echonet-list/echonet_lite/handler"
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
	CmdDebugOffline
	CmdUpdate
	CmdAliasSet
	CmdAliasGet
	CmdAliasDelete
	CmdAliasList
	CmdGroupAdd
	CmdGroupRemove
	CmdGroupDelete
	CmdGroupList
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
	DeviceSpec  client.DeviceSpecifier   // デバイス指定子（単一デバイス用）
	DeviceSpecs []client.DeviceSpecifier // 複数デバイス指定子（グループ追加・削除用）
	DeviceAlias *string                  // エイリアス
	GroupName   *string                  // グループ名（グループ操作用およびフィルタリング用）
	EPCs        []client.EPCType         // devicesコマンドのEPCフィルター用。空の場合は全EPCを表示
	PropMode    PropertyMode             // プロパティ表示モード
	Properties  client.Properties        // set/devicesコマンドのプロパティリスト
	GroupByEPC  *client.EPCType          // devicesコマンドのグループ化に使用するEPC
	DebugMode   *string                  // debugコマンドのモード ("on" または "off")
	ForceUpdate bool                     // updateコマンドの強制更新フラグ
	Done        chan struct{}            // コマンド実行完了を通知するチャネル
	Error       error                    // コマンド実行中に発生したエラー
}

// GetIPAddress は、コマンドのIPアドレスを取得する
func (c *Command) GetIPAddress() *net.IP {
	return c.DeviceSpec.IP
}

// GetClassCode は、コマンドのクラスコードを取得する
func (c *Command) GetClassCode() client.EOJClassCode {
	var result client.EOJClassCode
	if c.DeviceSpec.ClassCode != nil {
		result = *c.DeviceSpec.ClassCode
	}
	return result
}

// GetInstanceCode は、コマンドのインスタンスコードを取得する
func (c *Command) GetInstanceCode() *client.EOJInstanceCode {
	return c.DeviceSpec.InstanceCode
}

type CommandParser struct {
	propertyDescProvider client.PropertyDescProvider
	aliasManager         client.AliasManager
	groupManager         client.GroupManager
}

func NewCommandParser(propertyDescProvider client.PropertyDescProvider, aliasManager client.AliasManager, groupManager client.GroupManager) *CommandParser {
	return &CommandParser{
		propertyDescProvider: propertyDescProvider,
		aliasManager:         aliasManager,
		groupManager:         groupManager,
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
		return nil, nil, fmt.Errorf("class code must be 4 hexadecimal digits: %v", codeParts[0])
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

		if info, ok := p.propertyDescProvider.GetPropertyDesc(classCode, epc); ok {
			if valueEDT, ok := info.ToEDT(propParts[1]); ok {
				if debug {
					fmt.Printf("'%s' を EDT:%X に展開します\n", propParts[1], valueEDT)
				}
				edt = valueEDT
			}
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

		if p, ok := p.propertyDescProvider.FindPropertyAlias(classCode, alias); ok {
			if debug {
				fmt.Printf("エイリアス '%s' を EPC:%s, EDT:%X に展開します\n", alias, p.EPC, p.EDT)
			}
			return p, nil
		}
		return client.Property{}, fmt.Errorf("エイリアス '%s' が見つかりません。EPC:EDT 形式を使用してください", alias)
	}
}

// parseDeviceSpecifierOrGroup は、コマンド引数から DeviceSpecifier またはグループ名をパースする
// 引数:
//
//	parts: コマンドの引数配列
//	argIndex: パース開始位置
//	requireClassCode: クラスコードが必須かどうか
//
// 戻り値:
//
//	deviceSpecifier: パースされた DeviceSpecifier
//	groupName: パースされたグループ名（@で始まる場合）
//	nextArgIndex: 次の引数のインデックス
//	error: エラー
func (p CommandParser) parseDeviceSpecifierOrGroup(parts []string, argIndex int, requireClassCode bool) (client.DeviceSpecifier, *string, int, error) {
	var deviceSpec client.DeviceSpecifier

	if argIndex >= len(parts) {
		if requireClassCode {
			return deviceSpec, nil, argIndex, fmt.Errorf("デバイス識別子またはグループ名が必要です")
		}
		return deviceSpec, nil, argIndex, nil
	}

	// @で始まる場合はグループ名
	if strings.HasPrefix(parts[argIndex], "@") {
		group := parts[argIndex]
		return deviceSpec, &group, argIndex + 1, nil
	}

	// エイリアスの取得
	if alias, ok := p.aliasManager.GetDeviceByAlias(parts[argIndex]); ok {
		deviceSpec = handler.DeviceSpecifierFromIPAndEOJ(alias)
		return deviceSpec, nil, argIndex + 1, nil
	}

	// IPアドレスのパース（省略可能）- IPv4/IPv6に対応
	if ipAddr := tryParseIPAddress(parts[argIndex]); ipAddr != nil {
		deviceSpec.IP = ipAddr
		argIndex++

		if argIndex >= len(parts) && requireClassCode {
			return deviceSpec, nil, argIndex, fmt.Errorf("クラスコードが必要です")
		} else if argIndex >= len(parts) {
			return deviceSpec, nil, argIndex, nil
		}
	}

	// クラスコードとインスタンスコードのパース
	classCode, instanceCode, err := parseClassAndInstanceCode(parts[argIndex])
	if err != nil {
		if requireClassCode {
			return deviceSpec, nil, argIndex, err
		}
		// クラスコードが必須でない場合は、パースエラーを無視して現在の引数を処理せずに返す
		return deviceSpec, nil, argIndex, nil
	}

	deviceSpec.ClassCode = classCode
	deviceSpec.InstanceCode = instanceCode
	argIndex++

	return deviceSpec, nil, argIndex, nil
}

func (p CommandParser) parseDeviceSpecifiers(parts []string, argIndex int, requireClassCode bool) ([]client.DeviceSpecifier, int, error) {
	deviceSpecs := make([]client.DeviceSpecifier, 0)

	for argIndex < len(parts) {
		deviceSpec, groupName, nextArgIndex, err := p.parseDeviceSpecifierOrGroup(parts, argIndex, requireClassCode)
		if err != nil {
			return nil, argIndex, err
		}
		if groupName != nil {
			groups := p.groupManager.GroupList(groupName)
			if groups == nil {
				return nil, argIndex, fmt.Errorf("グループ '%s' が見つかりません", *groupName)
			}
			for _, group := range groups {
				for _, ids := range group.Devices {
					if device, ok := p.aliasManager.GetDeviceByAlias(string(ids)); ok {
						deviceSpec := handler.DeviceSpecifierFromIPAndEOJ(device)
						deviceSpecs = append(deviceSpecs, deviceSpec)
					}
				}
			}
		} else {
			deviceSpecs = append(deviceSpecs, deviceSpec)
		}
		argIndex = nextArgIndex
	}

	return deviceSpecs, argIndex, nil
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

type InvalidArgument struct {
	Argument string
}

func (e *InvalidArgument) Error() string {
	return fmt.Sprintf("無効な引数: %s", e.Argument)
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
