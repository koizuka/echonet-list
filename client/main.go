package main

import (
	"context"
	"echonet-list/clientlib"
	"echonet-list/protocol"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
)

const (
	defaultLog = "echonet-client.log" // デフォルトのログファイル名
)

func main() {
	// コマンドライン引数のヘルプメッセージをカスタマイズ
	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "使用方法: %s [オプション]\n\nオプション:\n", os.Args[0])
		flag.PrintDefaults()
	}

	// コマンドライン引数の定義
	debugFlag := flag.Bool("debug", false, "デバッグモードを有効にする")
	serverURLFlag := flag.String("server", "ws://localhost:8080/ws", "WebSocketサーバーのURL")

	// コマンドライン引数の解析
	flag.Parse()

	// フラグの値を取得
	debug := *debugFlag
	serverURL := *serverURLFlag

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

	// WebSocketクライアントの作成
	echonetClient, err := clientlib.NewECHONETLiteClient(ctx, serverURL)
	if err != nil {
		fmt.Printf("WebSocketクライアントの作成に失敗: %v\n", err)
		return
	}
	defer echonetClient.Close()

	// 通知を監視するゴルーチン
	go func() {
		for notification := range echonetClient.GetNotificationChannel() {
			switch notification.Event {
			case "deviceAdded":
				fmt.Printf("新しいデバイスが検出されました: %v\n", notification.DeviceInfo)
			case "deviceTimeout":
				// fmt.Printf("デバイス %v へのリクエストがタイムアウトしました: %v\n",
				// 	notification.DeviceInfo, notification.Data)
			}
		}
	}()

	// コマンドプロセッサの作成と開始
	processor := clientlib.NewCommandProcessor(ctx, echonetClient)
	processor.Start()
	// defer processor.Stop() は不要。明示的に呼び出すため

	// デバッグモードを設定（-debugフラグが指定された場合）
	if debug {
		debugMode := "on"
		echonetClient.DebugMode(&debugMode)
	}

	// コマンドの使用方法を表示
	fmt.Println("help for usage, quit to exit")

	// コマンド入力待ち（readline を使用して履歴機能を追加）
	// 履歴ファイルのパスを設定
	historyFile := ".echonet_history"
	if home, err := os.UserHomeDir(); err == nil {
		historyFile = fmt.Sprintf("%s/.echonet_history", home)
	}

	// コマンド補完用の関数を定義

	// TODO: エイリアス取得とコマンド補完機能の実装
	aliases := []readline.PrefixCompleterInterface{}

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

	p := NewCommandParser()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			break
		}

		cmd, err := p.ParseCommand(line, debug)
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
			continue
		}
		if cmd == nil {
			continue
		}

		if cmd.Type == clientlib.CmdQuit {
			// quitコマンドの場合は、コマンドチャネル経由で送信せず、直接終了処理を行う
			close(cmd.Done) // 完了を通知
			processor.Stop()
			break
		}

		// コマンドを送信し、エラーをチェック
		if err := processor.SendCommand(cmd); err != nil {
			fmt.Printf("エラー: %v\n", err)
		}
	}
}

// CommandParser はコマンド文字列をパースするための構造体
type CommandParser struct{}

// NewCommandParser は新しいCommandParserを作成する
func NewCommandParser() *CommandParser {
	return &CommandParser{}
}

// ParseCommand はコマンド文字列をパースする
func (p *CommandParser) ParseCommand(input string, debug bool) (*clientlib.Command, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil, nil
	}

	var cmd *clientlib.Command
	var err error
	switch parts[0] {
	case "quit":
		cmd = newCommand(clientlib.CmdQuit)
	case "discover":
		cmd = newCommand(clientlib.CmdDiscover)
	case "help":
		cmd = newCommand(clientlib.CmdHelp)
	case "get":
		cmd, err = p.parseGetCommand(parts)
	case "set":
		cmd, err = p.parseSetCommand(parts, debug)
	case "devices", "list":
		cmd, err = p.parseDevicesCommand(parts)
	case "debug":
		cmd, err = p.parseDebugCommand(parts)
	case "update":
		cmd, err = p.parseUpdateCommand(parts)
	case "alias":
		cmd, err = p.parseAliasCommand(parts)
	default:
		return nil, fmt.Errorf("unknown command: %s", parts[0])
	}

	if err != nil {
		return nil, err
	}
	return cmd, nil
}

// 基本的なコマンドオブジェクトを作成するヘルパー関数
func newCommand(cmdType clientlib.CommandType) *clientlib.Command {
	return &clientlib.Command{
		Done: make(chan struct{}),
		Type: cmdType,
	}
}

// IPアドレスをパースして、有効なIPアドレスならそのポインタを返す
// 無効なIPアドレスならnilを返す
func tryParseIPAddress(ipStr string) *string {
	// 実際のIP検証はWebSocketサーバー側で行うので、ここでは簡易的なチェックのみ
	if strings.Contains(ipStr, ".") || strings.Contains(ipStr, ":") {
		return &ipStr
	}
	return nil
}

// クラスコードとインスタンスコードをパースする
func parseClassAndInstanceCode(codeStr string) (*protocol.ClassCode, *uint8, error) {
	codeParts := strings.Split(codeStr, ":")
	if len(codeParts) > 2 {
		return nil, nil, fmt.Errorf("invalid format: %s (use classCode or classCode:instanceCode)", codeStr)
	}

	// クラスコードのパース
	if len(codeParts[0]) != 4 {
		return nil, nil, fmt.Errorf("class code must be 4 hexadecimal digits")
	}
	
	classCodeBytes, err := hex.DecodeString(codeParts[0])
	if err != nil {
		return nil, nil, fmt.Errorf("invalid class code: %s (must be 4 hexadecimal digits)", codeParts[0])
	}
	
	if len(classCodeBytes) != 2 {
		return nil, nil, fmt.Errorf("class code must be 4 hexadecimal digits")
	}
	
	// バイト配列からuint16へ変換
	classCodeValue := uint16(classCodeBytes[0])<<8 | uint16(classCodeBytes[1])
	classCode := protocol.ClassCode(classCodeValue)

	// インスタンスコードのパース（存在する場合）
	var instanceCode *uint8
	if len(codeParts) == 2 {
		instanceCode64, err := strconv.ParseUint(codeParts[1], 10, 8)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid instance code: %s (must be a number between 1-255)", codeParts[1])
		}
		if instanceCode64 == 0 || instanceCode64 > 255 {
			return nil, nil, fmt.Errorf("instance code must be between 1 and 255")
		}
		code := uint8(instanceCode64)
		instanceCode = &code
	}

	return &classCode, instanceCode, nil
}

// 単一のEPCをパースする
func parseEPC(epcStr string) (protocol.EPCType, error) {
	if len(epcStr) != 2 {
		return 0, fmt.Errorf("EPC must be 2 hexadecimal digits: %s", epcStr)
	}
	
	epcBytes, err := hex.DecodeString(epcStr)
	if err != nil {
		return 0, fmt.Errorf("invalid EPC: %s (must be 2 hexadecimal digits)", epcStr)
	}
	
	if len(epcBytes) != 1 {
		return 0, fmt.Errorf("EPC must be 2 hexadecimal digits")
	}
	
	return protocol.EPCType(epcBytes[0]), nil
}

// devicesコマンドまたはlistコマンドをパースする
func (p *CommandParser) parseDevicesCommand(parts []string) (*clientlib.Command, error) {
	cmd := newCommand(clientlib.CmdDevices)
	cmd.PropMode = protocol.PropDefault

	// デバイス指定子と残りの引数を解析
	argIndex := 1
	for argIndex < len(parts) {
		// オプション解析
		if parts[argIndex] == "-all" {
			cmd.PropMode = protocol.PropAll
			argIndex++
			continue
		} else if parts[argIndex] == "-props" {
			cmd.PropMode = protocol.PropKnown
			argIndex++
			continue
		}

		// IPアドレスの解析
		if ip := tryParseIPAddress(parts[argIndex]); ip != nil {
			cmd.DeviceSpec.IP = ip
			argIndex++
			if argIndex >= len(parts) {
				break
			}
		}

		// クラスコードとインスタンスコードの解析
		classCode, instanceCode, err := parseClassAndInstanceCode(parts[argIndex])
		if err == nil {
			cmd.DeviceSpec.ClassCode = classCode
			cmd.DeviceSpec.InstanceCode = instanceCode
			argIndex++
			if argIndex >= len(parts) {
				break
			}
		}

		// EPCの解析
		epc, err := parseEPC(parts[argIndex])
		if err == nil {
			cmd.EPCs = append(cmd.EPCs, epc)
			cmd.PropMode = protocol.PropEPC
			argIndex++
			continue
		}

		// それ以外の引数は無効
		return nil, fmt.Errorf("無効な引数: %s", parts[argIndex])
	}

	return cmd, nil
}

// getコマンドをパースする
func (p *CommandParser) parseGetCommand(parts []string) (*clientlib.Command, error) {
	cmd := newCommand(clientlib.CmdGet)

	// デバイス指定子と残りの引数を解析
	argIndex := 1
	
	// 最低限クラスコードが必要
	if argIndex >= len(parts) {
		return nil, fmt.Errorf("get コマンドにはクラスコードが必要です")
	}
	
	// IPアドレスの解析
	if ip := tryParseIPAddress(parts[argIndex]); ip != nil {
		cmd.DeviceSpec.IP = ip
		argIndex++
		if argIndex >= len(parts) {
			return nil, fmt.Errorf("get コマンドにはクラスコードが必要です")
		}
	}
	
	// クラスコードとインスタンスコードの解析
	classCode, instanceCode, err := parseClassAndInstanceCode(parts[argIndex])
	if err != nil {
		return nil, err
	}
	cmd.DeviceSpec.ClassCode = classCode
	cmd.DeviceSpec.InstanceCode = instanceCode
	argIndex++
	
	// EPCの解析
	for ; argIndex < len(parts); argIndex++ {
		if parts[argIndex] == "-skip-validation" {
			cmd.DebugMode = &parts[argIndex]
			continue
		}
		
		epc, err := parseEPC(parts[argIndex])
		if err != nil {
			return nil, err
		}
		cmd.EPCs = append(cmd.EPCs, epc)
	}
	
	if len(cmd.EPCs) == 0 {
		return nil, fmt.Errorf("get コマンドには少なくとも1つのEPCが必要です")
	}
	
	return cmd, nil
}

// setコマンドをパースする
func (p *CommandParser) parseSetCommand(parts []string, debug bool) (*clientlib.Command, error) {
	cmd := newCommand(clientlib.CmdSet)

	// デバイス指定子と残りの引数を解析
	argIndex := 1
	
	// 最低限クラスコードが必要
	if argIndex >= len(parts) {
		return nil, fmt.Errorf("set コマンドにはクラスコードが必要です")
	}
	
	// IPアドレスの解析
	if ip := tryParseIPAddress(parts[argIndex]); ip != nil {
		cmd.DeviceSpec.IP = ip
		argIndex++
		if argIndex >= len(parts) {
			return nil, fmt.Errorf("set コマンドにはクラスコードが必要です")
		}
	}
	
	// クラスコードとインスタンスコードの解析
	classCode, instanceCode, err := parseClassAndInstanceCode(parts[argIndex])
	if err != nil {
		return nil, err
	}
	cmd.DeviceSpec.ClassCode = classCode
	cmd.DeviceSpec.InstanceCode = instanceCode
	argIndex++
	
	// プロパティの解析
	if argIndex >= len(parts) {
		return nil, fmt.Errorf("set コマンドには少なくとも1つのプロパティが必要です")
	}
	
	// TODO: プロパティの解析実装（現在は簡易的な実装）
	for ; argIndex < len(parts); argIndex++ {
		propParts := strings.Split(parts[argIndex], ":")
		if len(propParts) != 2 {
			return nil, fmt.Errorf("プロパティは EPC:EDT 形式で指定してください: %s", parts[argIndex])
		}
		
		epc, err := parseEPC(propParts[0])
		if err != nil {
			return nil, err
		}
		
		edt, err := hex.DecodeString(propParts[1])
		if err != nil {
			return nil, fmt.Errorf("無効なEDT: %s (16進数で指定してください)", propParts[1])
		}
		
		cmd.Properties = append(cmd.Properties, protocol.Property{
			EPC: epc,
			EDT: protocol.ByteArray(edt),
		})
	}
	
	return cmd, nil
}

// debugコマンドをパースする
func (p *CommandParser) parseDebugCommand(parts []string) (*clientlib.Command, error) {
	cmd := newCommand(clientlib.CmdDebug)

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

// updateコマンドをパースする
func (p *CommandParser) parseUpdateCommand(parts []string) (*clientlib.Command, error) {
	cmd := newCommand(clientlib.CmdUpdate)

	// デバイス指定子の解析
	argIndex := 1
	
	if argIndex < len(parts) {
		// IPアドレスの解析
		if ip := tryParseIPAddress(parts[argIndex]); ip != nil {
			cmd.DeviceSpec.IP = ip
			argIndex++
		}
		
		// クラスコードとインスタンスコードの解析（もしあれば）
		if argIndex < len(parts) {
			classCode, instanceCode, err := parseClassAndInstanceCode(parts[argIndex])
			if err == nil {
				cmd.DeviceSpec.ClassCode = classCode
				cmd.DeviceSpec.InstanceCode = instanceCode
				argIndex++
			}
		}
		
		// 余分な引数があればエラー
		if argIndex < len(parts) {
			return nil, fmt.Errorf("無効な引数: %s", parts[argIndex])
		}
	}
	
	return cmd, nil
}

// aliasコマンドをパースする
func (p *CommandParser) parseAliasCommand(parts []string) (*clientlib.Command, error) {
	cmd := newCommand(clientlib.CmdAliasList)

	// エイリアスコマンドは現在未実装
	return cmd, fmt.Errorf("エイリアス関連コマンドは現在実装中です")
}