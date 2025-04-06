package console

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
			PrintUsage(cmd.DeviceAlias)
		case CmdGet:
			cmd.Error = p.processGetCommand(cmd)
		case CmdSet:
			cmd.Error = p.processSetCommand(cmd)
		case CmdDebug:
			cmd.Error = p.processDebugCommand(cmd)
		case CmdUpdate:
			cmd.Error = p.processUpdateCommand(cmd)
		case CmdAliasList:
			aliases := p.handler.AliasList()
			for _, alias := range aliases {
				d := p.handler.FindDeviceByIDString(alias.ID)
				if d == nil {
					fmt.Printf("%s: not found\n", alias.Alias)
				} else {
					fmt.Printf("%s: %v\n", alias.Alias, *d)
				}
			}
		case CmdAliasSet:
			criteria := client.FilterCriteria{
				Device:         cmd.DeviceSpec,
				PropertyValues: cmd.Properties,
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
		case CmdGroupAdd:
			cmd.Error = p.processGroupAddCommand(cmd)
		case CmdGroupRemove:
			cmd.Error = p.processGroupRemoveCommand(cmd)
		case CmdGroupDelete:
			cmd.Error = p.handler.GroupDelete(*cmd.GroupName)
			if cmd.Error == nil {
				fmt.Printf("グループ %s を削除しました\n", *cmd.GroupName)
			}
		case CmdGroupList:
			cmd.Error = p.processGroupListCommand(cmd)
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
				if !p.handler.IsPropertyDefaultEPC(classCode, epc) {
					continue
				}
			case PropKnown:
				// 既知のプロパティのみ表示
				if _, ok := p.handler.GetPropertyInfo(classCode, epc); !ok {
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

func (p *CommandProcessor) getGroupDevices(cmd *Command) ([]client.IPAndEOJ, error) {
	if cmd.GroupName != nil {
		groupDevices := p.handler.GroupList(cmd.GroupName)
		if len(groupDevices) == 0 {
			return nil, fmt.Errorf("グループ %s が見つからないか、デバイスが登録されていません", *cmd.GroupName)
		}

		var devices []client.IPAndEOJ
		for _, group := range groupDevices {
			devices = append(devices, group.Devices...)
		}
		if len(devices) == 0 {
			return nil, fmt.Errorf("グループ %s にデバイスが登録されていません", *cmd.GroupName)
		}
		return devices, nil
	}
	return nil, nil
}

func (p *CommandProcessor) processGetCommand(cmd *Command) error {
	skipValidation := false
	if cmd.DebugMode != nil && *cmd.DebugMode == "-skip-validation" {
		skipValidation = true
	}

	devices, err := p.getGroupDevices(cmd)
	if err != nil {
		return err
	}
	if devices == nil {
		// 通常のデバイス指定の場合
		device, err := p.getSingleDevice(cmd.DeviceSpec)
		if err != nil {
			if !skipValidation {
				return err
			}
			// -skip-validation が付いている場合、 IPアドレスとclassCodeさえあればデバイスを作成して処理を続行する。タイムアウト動作確認用
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
		}
		devices = append(devices, *device)
	}

	if len(cmd.EPCs) == 0 {
		return errors.New("get コマンドには少なくとも1つのEPCが必要です")
	}

	var lastError error
	for _, device := range devices {
		result, err := p.handler.GetProperties(device, cmd.EPCs, skipValidation)
		if err == nil {
			fmt.Printf("プロパティ取得成功: %v\n", result.Device)
			classCode := result.Device.EOJ.ClassCode()
			for _, p := range result.Properties {
				propStr := p.String(classCode)
				fmt.Printf("  %v\n", propStr)
			}
		} else {
			if lastError != nil {
				fmt.Println(lastError)
			}
			lastError = err
		}
	}
	return lastError
}

func (p *CommandProcessor) processSetCommand(cmd *Command) error {
	devices, err := p.getGroupDevices(cmd)
	if err != nil {
		return err
	}
	if devices == nil {
		// 通常のデバイス指定の場合
		device, err := p.getSingleDevice(cmd.DeviceSpec)
		if err != nil {
			return err
		}
		devices = append(devices, *device)
	}

	if len(cmd.Properties) == 0 {
		return errors.New("set コマンドには少なくとも1つのプロパティが必要です")
	}

	var lastError error
	for _, device := range devices {
		result, err := p.handler.SetProperties(device, cmd.Properties)
		if err == nil {
			fmt.Printf("プロパティ設定成功: %v\n", result.Device)
			classCode := result.Device.EOJ.ClassCode()
			for _, p := range result.Properties {
				propStr := p.String(classCode)
				fmt.Printf("  %v\n", propStr)
			}
		} else {
			if lastError != nil {
				fmt.Println(lastError)
			}
			lastError = err
		}
	}
	return lastError
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

func (p *CommandProcessor) processUpdateCommand(cmd *Command) error {
	// グループが指定されている場合
	if cmd.GroupName != nil {
		// グループ内のデバイスを取得
		groupDevices := p.handler.GroupList(cmd.GroupName)
		if len(groupDevices) == 0 {
			return fmt.Errorf("グループ %s が見つからないか、デバイスが登録されていません", *cmd.GroupName)
		}

		// グループ内の各デバイスに対して処理
		for _, group := range groupDevices {
			for _, device := range group.Devices {
				// デバイスごとにフィルタリング条件を作成
				classCode := device.EOJ.ClassCode()
				instanceCode := device.EOJ.InstanceCode()
				criteria := client.FilterCriteria{
					Device: client.DeviceSpecifier{
						IP:           &device.IP,
						ClassCode:    &classCode,
						InstanceCode: &instanceCode,
					},
				}
				err := p.handler.UpdateProperties(criteria)
				if err != nil {
					fmt.Printf("デバイス %v のプロパティ更新に失敗しました: %v\n", device, err)
				} else {
					fmt.Printf("デバイス %v のプロパティを更新しました\n", device)
				}
			}
		}
	} else {
		// 通常のデバイス指定の場合
		// フィルタリング条件を作成
		criteria := client.FilterCriteria{
			Device: cmd.DeviceSpec,
		}
		return p.handler.UpdateProperties(criteria)
	}
	return nil
}

func (p *CommandProcessor) processGroupAddCommand(cmd *Command) error {
	// DeviceSpecs から IPAndEOJ のスライスに変換
	devices := make([]client.IPAndEOJ, 0, len(cmd.DeviceSpecs))
	for _, spec := range cmd.DeviceSpecs {
		found := p.handler.GetDevices(spec)
		if len(found) == 0 {
			return fmt.Errorf("デバイスが見つかりません: %v", spec)
		}
		devices = append(devices, found...)
	}

	err := p.handler.GroupAdd(*cmd.GroupName, devices)
	if err == nil {
		fmt.Printf("グループ %s にデバイスを追加しました\n", *cmd.GroupName)
	}
	return err
}

func (p *CommandProcessor) processGroupRemoveCommand(cmd *Command) error {
	// DeviceSpecs から IPAndEOJ のスライスに変換
	devices := make([]client.IPAndEOJ, 0, len(cmd.DeviceSpecs))
	for _, spec := range cmd.DeviceSpecs {
		found := p.handler.GetDevices(spec)
		if len(found) == 0 {
			return fmt.Errorf("デバイスが見つかりません: %v", spec)
		}
		devices = append(devices, found...)
	}

	err := p.handler.GroupRemove(*cmd.GroupName, devices)
	if err == nil {
		fmt.Printf("グループ %s からデバイスを削除しました\n", *cmd.GroupName)
	}
	return err
}

func (p *CommandProcessor) processGroupListCommand(cmd *Command) error {
	var groups []client.GroupDevicePair
	if cmd.GroupName != nil {
		groups = p.handler.GroupList(cmd.GroupName)
		if len(groups) == 0 {
			return fmt.Errorf("グループ %s が見つかりません", *cmd.GroupName)
		}
	} else {
		groups = p.handler.GroupList(nil)
		if len(groups) == 0 {
			fmt.Println("グループが登録されていません")
		}
	}

	for _, group := range groups {
		fmt.Printf("%s: %d デバイス\n", group.Group, len(group.Devices))
		for _, device := range group.Devices {
			// エイリアスがあれば表示
			aliases := p.handler.GetAliases(device)
			if len(aliases) > 0 {
				fmt.Printf("  %v (%s)\n", device, strings.Join(aliases, ", "))
			} else {
				fmt.Printf("  %v\n", device)
			}
		}
	}
	return nil
}
