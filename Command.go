package main

import (
	"echonet-list/echonet_lite"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
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
)

// プロパティ表示モードを表す型
type PropertyMode int

const (
	PropDefault PropertyMode = iota // デフォルトのプロパティを表示
	PropKnown                       // 既知のプロパティのみ表示
	PropAll                         // 全てのプロパティを表示
)

// DeviceIdentifier は、デバイスを一意に識別するための情報を表す構造体
type DeviceIdentifier struct {
	IPAddress    *string                       // IPアドレス。nilの場合は自動選択
	ClassCode    *echonet_lite.EOJClassCode    // クラスコード
	InstanceCode *echonet_lite.EOJInstanceCode // インスタンスコード
}

// コマンドを表す構造体
type Command struct {
	Type       CommandType
	DeviceID   *DeviceIdentifier       // デバイス識別子
	EPCs       []echonet_lite.EPCType  // devicesコマンドのEPCフィルター用。空の場合は全EPCを表示
	PropMode   PropertyMode            // プロパティ表示モード
	Properties []echonet_lite.Property // set/devicesコマンドのプロパティリスト
	DebugMode  *string                 // debugコマンドのモード ("on" または "off")
	Done       chan struct{}           // コマンド実行完了を通知するチャネル
	Error      error                   // コマンド実行中に発生したエラー
}

// GetIPAddress は、コマンドのIPアドレスを取得する
func (c *Command) GetIPAddress() *string {
	if c.DeviceID == nil {
		return nil
	}
	return c.DeviceID.IPAddress
}

// GetClassCode は、コマンドのクラスコードを取得する
func (c *Command) GetClassCode() *echonet_lite.EOJClassCode {
	if c.DeviceID == nil {
		return nil
	}
	return c.DeviceID.ClassCode
}

// GetInstanceCode は、コマンドのインスタンスコードを取得する
func (c *Command) GetInstanceCode() *echonet_lite.EOJInstanceCode {
	if c.DeviceID == nil {
		return nil
	}
	return c.DeviceID.InstanceCode
}

// IPアドレスをパースして、有効なIPアドレスならその文字列のポインタを返す
// 無効なIPアドレスならnilを返す
func tryParseIPAddress(ipStr string) *string {
	if ip := net.ParseIP(ipStr); ip != nil {
		return &ipStr
	}
	return nil
}

// クラスコードとインスタンスコードをパースする
func parseClassAndInstanceCode(codeStr string) (*echonet_lite.EOJClassCode, *echonet_lite.EOJInstanceCode, error) {
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
	classCode := echonet_lite.EOJClassCode(classCode64)

	// インスタンスコードのパース（存在する場合）
	instanceCode := echonet_lite.EOJInstanceCode(1) // デフォルト値
	if len(codeParts) == 2 {
		instanceCode64, err := strconv.ParseUint(codeParts[1], 10, 8)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid instance code: %s (must be a number between 1-255)", codeParts[1])
		}
		if instanceCode64 == 0 || instanceCode64 > 255 {
			return nil, nil, fmt.Errorf("instance code must be between 1 and 255")
		}
		instanceCode = echonet_lite.EOJInstanceCode(instanceCode64)
	}

	return &classCode, &instanceCode, nil
}

// 単一のEPCをパースする
func parseEPC(epcStr string) (echonet_lite.EPCType, error) {
	if len(epcStr) != 2 {
		return 0, fmt.Errorf("EPC must be 2 hexadecimal digits: %s", epcStr)
	}
	epc64, err := strconv.ParseUint(epcStr, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid EPC: %s (must be 2 hexadecimal digits)", epcStr)
	}
	return echonet_lite.EPCType(epc64), nil
}

// プロパティ文字列をパースする
// propertyStr: プロパティ文字列（"EPC:EDT" 形式または "alias" 形式）
// classCode: クラスコード
// debug: デバッグフラグ
// 戻り値: パースされたプロパティとエラー
func parsePropertyString(propertyStr string, classCode echonet_lite.EOJClassCode, debug bool) (echonet_lite.Property, error) {
	// EPC:EDT の形式をパース
	propParts := strings.Split(propertyStr, ":")
	if len(propParts) == 2 {
		// EPCのパース
		epc, err := parseEPC(propParts[0])
		if err != nil {
			return echonet_lite.Property{}, err
		}

		var edt []byte

		// エイリアスのチェック
		// クラスコードからPropertyInfoを取得
		if propInfo, ok := echonet_lite.GetPropertyInfo(classCode, epc); ok && propInfo.Aliases != nil {
			// エイリアスが定義されているか確認
			if aliasEDT, ok := propInfo.Aliases[propParts[1]]; ok {
				// エイリアスが見つかった場合、そのEDT値を使用
				if debug {
					fmt.Printf("エイリアス '%s' を EDT:%X に展開します\n", propParts[1], aliasEDT)
				}
				edt = aliasEDT
			}
		}

		// エイリアスが見つからなかった場合は通常のEDTパース
		if edt == nil {
			// EDTのパース（2桁の16進数の倍数）
			if len(propParts[1])%2 != 0 {
				return echonet_lite.Property{}, fmt.Errorf("EDT must be a multiple of 2 hexadecimal digits: %s", propParts[1])
			}

			edt = make([]byte, len(propParts[1])/2)
			for j := 0; j < len(propParts[1]); j += 2 {
				b, err := strconv.ParseUint(propParts[1][j:j+2], 16, 8)
				if err != nil {
					return echonet_lite.Property{}, fmt.Errorf("invalid EDT: %s (must be hexadecimal digits)", propParts[1])
				}
				edt[j/2] = byte(b)
			}
		}

		// プロパティを返す
		return echonet_lite.Property{EPC: epc, EDT: edt}, nil
	} else {
		// エイリアスのみの場合（例: "on"）
		alias := propertyStr

		// クラスコードに対応するすべてのプロパティを検索
		if p, ok := echonet_lite.PropertyTables.FindAlias(classCode, alias); ok {
			// エイリアスが見つかった場合、そのEPC:EDTを使用
			if debug {
				fmt.Printf("エイリアス '%s' を EPC:%s, EDT:%X に展開します\n", alias, p.EPC, p.EDT)
			}
			return p, nil
		} else {
			return echonet_lite.Property{}, fmt.Errorf("エイリアス '%s' が見つかりません。EPC:EDT 形式を使用してください", alias)
		}
	}
}

// parseDeviceIdentifier は、コマンド引数から DeviceIdentifier をパースする
// 引数:
//
//	parts: コマンドの引数配列
//	argIndex: パース開始位置
//	requireClassCode: クラスコードが必須かどうか
//
// 戻り値:
//
//	deviceIdentifier: パースされた DeviceIdentifier
//	nextArgIndex: 次の引数のインデックス
//	error: エラー
func parseDeviceIdentifier(parts []string, argIndex int, requireClassCode bool) (*DeviceIdentifier, int, error) {
	if argIndex >= len(parts) {
		if requireClassCode {
			return nil, argIndex, fmt.Errorf("デバイス識別子が必要です")
		}
		return nil, argIndex, nil
	}

	deviceID := &DeviceIdentifier{}

	// IPアドレスのパース（省略可能）- IPv4/IPv6に対応
	if ipAddr := tryParseIPAddress(parts[argIndex]); ipAddr != nil {
		deviceID.IPAddress = ipAddr
		argIndex++

		if argIndex >= len(parts) && requireClassCode {
			return nil, argIndex, fmt.Errorf("クラスコードが必要です")
		} else if argIndex >= len(parts) {
			return deviceID, argIndex, nil
		}
	}

	// クラスコードとインスタンスコードのパース
	classCode, instanceCode, err := parseClassAndInstanceCode(parts[argIndex])
	if err != nil {
		if requireClassCode {
			return nil, argIndex, err
		}
		// クラスコードが必須でない場合は、パースエラーを無視して現在の引数を処理せずに返す
		return deviceID, argIndex, nil
	}

	deviceID.ClassCode = classCode
	deviceID.InstanceCode = instanceCode
	argIndex++

	return deviceID, argIndex, nil
}

// 基本的なコマンドオブジェクトを作成するヘルパー関数
func newCommand(cmdType CommandType) *Command {
	return &Command{
		Done: make(chan struct{}),
		Type: cmdType,
	}
}

// "get" コマンドをパースする
func parseGetCommand(parts []string) (*Command, error) {
	cmd := newCommand(CmdGet)

	// デバイス識別子のパース
	deviceID, argIndex, err := parseDeviceIdentifier(parts, 1, true)
	if err != nil {
		return nil, err
	}
	cmd.DeviceID = deviceID

	// EPCのパース
	if argIndex >= len(parts) {
		return nil, fmt.Errorf("get コマンドには少なくとも1つのEPCが必要です")
	}

	for i := argIndex; i < len(parts); i++ {
		epc, err := parseEPC(parts[i])
		if err != nil {
			return nil, err
		}
		cmd.EPCs = append(cmd.EPCs, epc)
	}

	return cmd, nil
}

// "set" コマンドをパースする
func parseSetCommand(parts []string, debug bool) (*Command, error) {
	cmd := newCommand(CmdSet)

	// デバイス識別子のパース
	deviceID, argIndex, err := parseDeviceIdentifier(parts, 1, true)
	if err != nil {
		return nil, err
	}
	cmd.DeviceID = deviceID

	// プロパティのパース
	if argIndex >= len(parts) {
		// 可能なエイリアス一覧
		aliases := echonet_lite.PropertyTables.AvailableAliases(*cmd.GetClassCode())
		fmt.Printf("利用可能なエイリアス:\n")
		// sort by alias

		sortedAliases := make([]string, 0, len(aliases))
		for alias := range aliases {
			sortedAliases = append(sortedAliases, alias)
		}
		sort.Strings(sortedAliases)
		fmt.Println("sorted names: ", sortedAliases) // DEBUG
		for _, alias := range sortedAliases {
			fmt.Printf("%s: %s\n", alias, aliases[alias])
		}
		return nil, fmt.Errorf("set コマンドには少なくとも1つのプロパティが必要です")
	}

	for i := argIndex; i < len(parts); i++ {
		// EPCのみの場合（エイリアス一覧表示）
		epc, err := parseEPC(parts[i])
		if err == nil {
			// クラスコードからPropertyInfoを取得
			if propInfo, ok := echonet_lite.GetPropertyInfo(*cmd.GetClassCode(), epc); ok && propInfo.Aliases != nil && len(propInfo.Aliases) > 0 {
				fmt.Printf("利用可能なエイリアス for EPC %s (%s):\n", epc, propInfo.EPCs)
				sortedAliases := make([]string, 0, len(propInfo.Aliases))
				for alias := range propInfo.Aliases {
					sortedAliases = append(sortedAliases, alias)
				}
				sort.Strings(sortedAliases)
				for _, alias := range sortedAliases {
					fmt.Printf("  %s -> %X\n", alias, propInfo.Aliases[alias])
				}
			} else {
				fmt.Printf("EPC %s にはエイリアスが定義されていません\n", epc)
			}
			continue
		}

		// プロパティ文字列をパース
		prop, err := parsePropertyString(parts[i], *cmd.GetClassCode(), debug)
		if err != nil {
			return nil, err
		}

		// プロパティを追加
		cmd.Properties = append(cmd.Properties, prop)
	}

	return cmd, nil
}

// "devices" または "list" コマンドをパースする
func parseDevicesCommand(parts []string) (*Command, error) {
	cmd := newCommand(CmdDevices)

	// デバイス識別子のパース
	deviceID, argIndex, err := parseDeviceIdentifier(parts, 1, false)
	if err != nil {
		return nil, err
	}
	cmd.DeviceID = deviceID

	// 残りの引数を解析
	for i := argIndex; i < len(parts); i++ {
		switch parts[i] {
		case "-all":
			cmd.PropMode = PropAll
			continue
		case "-props":
			cmd.PropMode = PropKnown
			continue
		}

		pClassCode := cmd.GetClassCode()
		if pClassCode == nil {
			pClassCode = new(echonet_lite.EOJClassCode)
		}
		props, err := parsePropertyString(parts[i], *pClassCode, false) // corrected from classCode to *pClassCode
		if err == nil {
			cmd.Properties = append(cmd.Properties, props)
			continue
		}

		// EPCのパース（2桁の16進数）
		epc, err := parseEPC(parts[i])
		if err == nil {
			cmd.EPCs = append(cmd.EPCs, epc)
			continue
		}

		// 上記のいずれにも該当しない場合はエラー
		return nil, fmt.Errorf("無効な引数: %s", parts[i])
	}

	return cmd, nil
}

// "debug" コマンドをパースする
func parseDebugCommand(parts []string) (*Command, error) {
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

// "update" コマンドをパースする
func parseUpdateCommand(parts []string) (*Command, error) {
	cmd := newCommand(CmdUpdate)

	// デバイス識別子のパース
	deviceID, argIndex, err := parseDeviceIdentifier(parts, 1, false)
	if err != nil {
		return nil, err
	}
	cmd.DeviceID = deviceID

	// 残りの引数がある場合はエラー
	if argIndex < len(parts) {
		return nil, fmt.Errorf("無効な引数: %s", parts[argIndex])
	}

	return cmd, nil
}

// コマンドをパースする
func ParseCommand(input string, debug bool) (*Command, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil, nil
	}

	var cmd *Command
	var err error
	switch parts[0] {
	case "quit":
		cmd = newCommand(CmdQuit)
	case "discover":
		cmd = newCommand(CmdDiscover)
	case "help":
		cmd = newCommand(CmdHelp)
	case "get":
		cmd, err = parseGetCommand(parts)
	case "set":
		cmd, err = parseSetCommand(parts, debug)
	case "devices", "list":
		cmd, err = parseDevicesCommand(parts)
	case "debug":
		cmd, err = parseDebugCommand(parts)
	case "update":
		cmd, err = parseUpdateCommand(parts)
	default:
		return nil, fmt.Errorf("unknown command: %s", parts[0])
	}

	if err != nil {
		return nil, err
	}
	return cmd, nil
}

// コマンドの使用方法を表示する
func PrintUsage() {
	fmt.Println("ECHONET Lite デバイス検出プログラム")
	fmt.Println("コマンド:")
	fmt.Println("  discover: ECHONET Lite デバイスの検出")
	fmt.Println("  devices, list [ipAddress] [classCode[:instanceCode]] [-all|-props] [epc1 epc2...]: 検出されたECHONET Liteデバイスの一覧表示")
	fmt.Println("    ipAddress: IPアドレスでフィルター（例: 192.168.0.212 または IPv6アドレス）")
	fmt.Println("    classCode: クラスコード（4桁の16進数、例: 0130）")
	fmt.Println("    instanceCode: インスタンスコード（1-255の数字、例: 0130:1）")
	fmt.Println("    -all: 全てのEPCを表示")
	fmt.Println("    -props: 既知のEPCのみを表示")
	fmt.Println("    ※-all と -props が両方指定された場合は後に指定された方が有効")
	fmt.Println("    epc: 2桁の16進数で指定（例: 80）。複数指定可能")
	fmt.Println("  get [ipAddress] classCode[:instanceCode] epc1 [epc2...]: プロパティ値の取得")
	fmt.Println("    ipAddress: 対象デバイスのIPアドレス（省略可能、省略時はクラスコードに一致するデバイスが1つだけの場合に自動選択）")
	fmt.Println("    classCode: クラスコード（4桁の16進数、必須）")
	fmt.Println("    instanceCode: インスタンスコード（1-255の数字、省略時は1）")
	fmt.Println("    epc: 取得するプロパティのEPC（2桁の16進数、例: 80）。複数指定可能")
	fmt.Println("  set [ipAddress] classCode[:instanceCode] property1 [property2...]: プロパティ値の設定")
	fmt.Println("    ipAddress: 対象デバイスのIPアドレス（省略可能、省略時はクラスコードに一致するデバイスが1つだけの場合に自動選択）")
	fmt.Println("    classCode: クラスコード（4桁の16進数、必須）")
	fmt.Println("    instanceCode: インスタンスコード（1-255の数字、省略時は1）")
	fmt.Println("    property: 以下のいずれかの形式")
	fmt.Println("      - EPC:EDT（例: 80:30）")
	fmt.Println("        EPC: 2桁の16進数")
	fmt.Println("        EDT: 2桁の16進数の倍数またはエイリアス名")
	fmt.Println("      - EPC（例: 80）- 利用可能なエイリアスを表示")
	fmt.Println("      - エイリアス名（例: on）- 対応するEPC:EDTに自動展開")
	fmt.Println("      - 80:on（OperationStatus{true}と同等）")
	fmt.Println("      - 80:off（OperationStatus{false}と同等）")
	fmt.Println("      - b0:auto（エアコンの自動モードと同等）")
	fmt.Println("  update [ipAddress] [classCode[:instanceCode]]: デバイスのプロパティキャッシュを更新")
	fmt.Println("    ipAddress: 対象デバイスのIPアドレス（省略可能、省略時は全デバイスが対象）")
	fmt.Println("    classCode: クラスコード（4桁の16進数、省略時は全クラスが対象）")
	fmt.Println("    instanceCode: インスタンスコード（1-255の数字、省略時は指定クラスの全インスタンスが対象）")
	fmt.Println("  debug [on|off]: デバッグモードの表示または切り替え")
	fmt.Println("    引数なし: 現在のデバッグモードを表示")
	fmt.Println("    on: デバッグモードを有効にする")
	fmt.Println("    off: デバッグモードを無効にする")
	fmt.Println("  help: このヘルプを表示")
	fmt.Println("  quit: 終了")
}
