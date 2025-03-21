package main

import (
	"context"
	"echonet-list/client"
	"errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/exp/slices"
)

// CommandProcessor は、コマンド処理を担当する構造体
type CommandProcessor struct {
	handler client.ECHONETListClient
	cmdChan chan *Command
	done    chan struct{}
	ctx     context.Context    // コンテキスト
	cancel  context.CancelFunc // コンテキストのキャンセル関数
}

// NewCommandProcessor は、CommandProcessor の新しいインスタンスを作成する
func NewCommandProcessor(ctx context.Context, handler client.ECHONETListClient) *CommandProcessor {
	// コマンドプロセッサ用のコンテキストを作成
	processorCtx, cancel := context.WithCancel(ctx)

	return &CommandProcessor{
		handler: handler,
		cmdChan: make(chan *Command),
		done:    make(chan struct{}),
		ctx:     processorCtx,
		cancel:  cancel,
	}
}

// Start は、コマンド処理を開始する
func (p *CommandProcessor) Start() {
	go p.processCommands()
}

// Stop は、コマンド処理を停止する
func (p *CommandProcessor) Stop() {
	// コンテキストをキャンセル
	if p.cancel != nil {
		p.cancel()
	}

	// チャネルが既に閉じられていないことを確認
	select {
	case <-p.done:
		// 既に終了している場合は何もしない
		return
	default:
		// まだ終了していない場合は閉じる
		close(p.cmdChan)
		<-p.done // コマンド処理goroutineの終了を待つ
	}
}

// SendCommand は、コマンドを送信し、結果のエラーを返す
func (p *CommandProcessor) SendCommand(cmd *Command) error {
	p.cmdChan <- cmd
	<-cmd.Done       // コマンドの実行が完了するまで待つ
	return cmd.Error // コマンド実行中のエラーを返す
}

// processCommands は、コマンドを処理するgoroutine
func (p *CommandProcessor) processCommands() {
	defer close(p.done)

	// コンテキストのキャンセルを監視するgoroutineを起動
	go func() {
		<-p.ctx.Done()
		// コンテキストがキャンセルされた場合の処理
		// cmdChanは閉じない（Stop()メソッドで閉じるため）
	}()

	for cmd := range p.cmdChan {
		// コンテキストがキャンセルされていないか確認
		select {
		case <-p.ctx.Done():
			// コンテキストがキャンセルされた場合は終了
			return
		default:
			// 継続
		}

		switch cmd.Type {
		case CmdQuit:
			close(cmd.Done) // 終了コマンドの場合は即座に完了を通知して終了
			return
		case CmdDiscover:
			cmd.Error = p.handler.Discover()
		case CmdDevices:
			cmd.Error = p.processDevicesCommand(cmd)

		case CmdHelp:
			PrintUsage()
		case CmdGet:
			cmd.Error = p.processGetCommand(cmd)
		case CmdSet:
			cmd.Error = p.processSetCommand(cmd)
		case CmdDebug:
			cmd.Error = p.processDebugCommand(cmd)
		case CmdUpdate:
			// フィルタリング条件を作成
			criteria := client.FilterCriteria{
				Device: cmd.DeviceSpec,
			}
			cmd.Error = p.handler.UpdateProperties(criteria)
		case CmdAliasList:
			aliases := p.handler.AliasList()
			for _, alias := range aliases {
				fmt.Println(alias)
			}
		case CmdAliasSet:
			criteria := client.FilterCriteria{
				Device: cmd.DeviceSpec,
			}
			cmd.Error = p.handler.AliasSet(cmd.DeviceAlias, criteria)
		case CmdAliasDelete:
			cmd.Error = p.handler.AliasDelete(cmd.DeviceAlias)
		case CmdAliasGet:
			device, err := p.handler.AliasGet(cmd.DeviceAlias)
			cmd.Error = err
			if err == nil {
				fmt.Printf("%s: %v\n", *cmd.DeviceAlias, device)
			}
		default:
			panic("unhandled default case")
		}

		// コマンド実行完了を通知（quit以外の全てのコマンド）
		close(cmd.Done)
	}
}

type DeviceClassNotFoundError struct {
	ClassCode client.EOJClassCode
}

func (e DeviceClassNotFoundError) Error() string {
	return fmt.Sprintf("クラスコード %v のデバイスが見つかりません", e.ClassCode)
}

type TooManyDevicesError struct {
	ClassCode client.EOJClassCode
	Devices   []client.IPAndEOJ
}

func (e TooManyDevicesError) Error() string {
	errMsg := []string{
		fmt.Sprintf("クラスコード %v のデバイスが複数見つかりました。IPアドレスを指定してください", e.ClassCode),
	}
	for _, device := range e.Devices {
		errMsg = append(errMsg, fmt.Sprintf("  %v", device))
	}
	return strings.Join(errMsg, "\n")
}

func (p *CommandProcessor) getSingleDevice(deviceSpec client.DeviceSpecifier) (*client.IPAndEOJ, error) {
	// フィルタリング
	filtered := p.handler.GetDevices(deviceSpec)

	// マッチするデバイスが1つだけでない場合はエラー
	if len(filtered) != 1 {
		var classCode client.EOJClassCode
		if deviceSpec.ClassCode != nil {
			classCode = *deviceSpec.ClassCode
		}
		if len(filtered) == 0 {
			return nil, DeviceClassNotFoundError{ClassCode: classCode}
		}
		return nil, TooManyDevicesError{ClassCode: classCode, Devices: filtered}
	}

	// マッチしたデバイスを返す
	return &filtered[0], nil
}

func sortProperties(p client.Properties) client.Properties {
	// プロパティをEPCでソート
	epcsToShow := make([]client.EPCType, 0, len(p))
	for _, prop := range p {
		epc := prop.EPC
		epcsToShow = append(epcsToShow, epc)
	}
	sort.Slice(p, func(i, j int) bool {
		return p[i].EPC < p[j].EPC
	})
	return p
}

func (p *CommandProcessor) processDevicesCommand(cmd *Command) error {
	// フィルタリング条件を作成
	criteria := client.FilterCriteria{
		Device:         cmd.DeviceSpec,
		PropertyValues: cmd.Properties,
	}
	result := p.handler.ListDevices(criteria)
	for _, d := range result {
		device := d.Device
		properties := d.Properties
		classCode := device.EOJ.ClassCode()

		// プロパティ表示モードに応じてフィルタリング
		filteredProps := make(client.Properties, 0, len(properties))
		for _, prop := range properties {
			epc := prop.EPC
			switch cmd.PropMode {
			case PropDefault:
				// デフォルトのプロパティのみ表示
				if !client.IsPropertyDefaultEPC(classCode, epc) {
					continue
				}
			case PropKnown:
				// 既知のプロパティのみ表示
				if _, ok := client.GetPropertyInfo(classCode, epc); !ok {
					continue
				}
			case PropEPC:
				// cmd.EPCsにあるもののみ表示
				if !slices.Contains(cmd.EPCs, epc) {
					continue
				}
			}
			filteredProps = append(filteredProps, prop)
		}

		if len(filteredProps) == 0 {
			continue
		}

		names := p.handler.GetAliases(device)
		names = append(names, device.String())
		fmt.Println(strings.Join(names, " "))

		for _, prop := range sortProperties(filteredProps) {
			fmt.Printf("  %v\n", prop.String(classCode))
		}
	}
	return nil
}

func (p *CommandProcessor) processGetCommand(cmd *Command) error {
	skipValidation := false
	if cmd.DebugMode != nil && *cmd.DebugMode == "-skip-validation" {
		skipValidation = true
	}

	device, err := p.getSingleDevice(cmd.DeviceSpec)
	if err != nil {
		// -skip-validation が付いている場合、 IPアドレスとclassCodeさえあればデバイスを作成して処理を続行する。タイムアウト動作確認用
		if skipValidation {
			if cmd.DeviceSpec.IP == nil || cmd.DeviceSpec.ClassCode == nil {
				return errors.New("get コマンドにはIPアドレスとクラスコードが必要です")
			}
			instanceCode := client.EOJInstanceCode(0)
			if cmd.DeviceSpec.InstanceCode != nil {
				instanceCode = *cmd.DeviceSpec.InstanceCode
			}
			device = &client.IPAndEOJ{
				IP:  *cmd.DeviceSpec.IP,
				EOJ: client.MakeEOJ(*cmd.DeviceSpec.ClassCode, instanceCode),
			}
		} else {
			return err
		}
	}
	if len(cmd.EPCs) == 0 {
		return errors.New("get コマンドには少なくとも1つのEPCが必要です")
	}
	result, err := p.handler.GetProperties(*device, cmd.EPCs, skipValidation)
	if err == nil {
		fmt.Printf("プロパティ取得成功: %v\n", result.Device)
		classCode := result.Device.EOJ.ClassCode()
		for _, p := range result.Properties {
			propStr := p.String(classCode)
			fmt.Printf("  %v\n", propStr)
		}
	}
	return err
}

func (p *CommandProcessor) processSetCommand(cmd *Command) error {
	device, err := p.getSingleDevice(cmd.DeviceSpec)
	if err != nil {
		return err
	}
	if len(cmd.Properties) == 0 {
		return errors.New("set コマンドには少なくとも1つのプロパティが必要です")
	}
	result, err := p.handler.SetProperties(*device, cmd.Properties)
	if err == nil {
		fmt.Printf("プロパティ設定成功: %v\n", result.Device)
		classCode := result.Device.EOJ.ClassCode()
		for _, p := range result.Properties {
			propStr := p.String(classCode)
			fmt.Printf("  %v\n", propStr)
		}
	}
	return err
}

func (p *CommandProcessor) processDebugCommand(cmd *Command) error {
	// デバッグモードの表示または切り替え
	if cmd.DebugMode != nil {
		// 引数がある場合はデバッグモードを切り替え
		debugMode := *cmd.DebugMode == "on"
		p.handler.SetDebug(debugMode)
		if debugMode {
			fmt.Println("デバッグモードを有効にしました")
		} else {
			fmt.Println("デバッグモードを無効にしました")
		}
	} else {
		// 引数がない場合は現在のデバッグモードを表示
		if p.handler.IsDebug() {
			fmt.Println("現在のデバッグモード: 有効")
		} else {
			fmt.Println("現在のデバッグモード: 無効")
		}
	}
	return nil
}
