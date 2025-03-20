package clientlib

import (
	"context"
	"echonet-list/protocol"
	"fmt"
	"sync"
)

// Command はクライアント側のコマンドを表す構造体
type Command struct {
	Type        CommandType
	DeviceSpec  protocol.DeviceSpecifier // デバイス指定子
	DeviceAlias *string                  // エイリアス
	EPCs        []protocol.EPCType       // devicesコマンドのEPCフィルター用。空の場合は全EPCを表示
	PropMode    protocol.PropertyMode    // プロパティ表示モード
	Properties  []protocol.Property      // set/devicesコマンドのプロパティリスト
	DebugMode   *string                  // debugコマンドのモード ("on" または "off")
	Done        chan struct{}            // コマンド実行完了を通知するチャネル
	Error       error                    // コマンド実行中に発生したエラー
}

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

// CommandProcessor はコマンドの処理を担当する構造体
type CommandProcessor struct {
	client    *ECHONETLiteClient
	cmdChan   chan *Command
	done      chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	waitGroup sync.WaitGroup
}

// NewCommandProcessor は新しいCommandProcessorを作成する
func NewCommandProcessor(ctx context.Context, client *ECHONETLiteClient) *CommandProcessor {
	processorCtx, cancel := context.WithCancel(ctx)

	return &CommandProcessor{
		client:  client,
		cmdChan: make(chan *Command),
		done:    make(chan struct{}),
		ctx:     processorCtx,
		cancel:  cancel,
	}
}

// Start はコマンド処理を開始する
func (p *CommandProcessor) Start() {
	go p.processCommands()
}

// Stop はコマンド処理を停止する
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

// SendCommand はコマンドを送信し、結果のエラーを返す
func (p *CommandProcessor) SendCommand(cmd *Command) error {
	p.cmdChan <- cmd
	<-cmd.Done       // コマンドの実行が完了するまで待つ
	return cmd.Error // コマンド実行中のエラーを返す
}

// processCommands はコマンドを処理するgoroutine
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
			cmd.Error = p.client.Discover()
		case CmdDevices:
			p.waitGroup.Add(1)
			go func() {
				defer p.waitGroup.Done()
				results, err := p.client.ListDevices(cmd.DeviceSpec, cmd.PropMode, cmd.EPCs, cmd.Properties)
				cmd.Error = err
				if err == nil {
					DisplayDeviceList(results)
				}
				close(cmd.Done)
			}()
			continue
		case CmdHelp:
			PrintUsage()
		case CmdGet:
			p.waitGroup.Add(1)
			go func() {
				defer p.waitGroup.Done()
				result, err := p.client.GetProperties(cmd.DeviceSpec, cmd.EPCs, cmd.DebugMode != nil && *cmd.DebugMode == "-skip-validation")
				cmd.Error = err
				if err == nil && result != nil {
					DisplayDeviceProperties(*result)
				}
				close(cmd.Done)
			}()
			continue
		case CmdSet:
			p.waitGroup.Add(1)
			go func() {
				defer p.waitGroup.Done()
				result, err := p.client.SetProperties(cmd.DeviceSpec, cmd.Properties)
				cmd.Error = err
				if err == nil && result != nil {
					DisplayDeviceProperties(*result)
				}
				close(cmd.Done)
			}()
			continue
		case CmdDebug:
			p.waitGroup.Add(1)
			go func() {
				defer p.waitGroup.Done()
				debug, err := p.client.DebugMode(cmd.DebugMode)
				cmd.Error = err
				if err == nil {
					if cmd.DebugMode != nil {
						if *cmd.DebugMode == "on" {
							fmt.Println("デバッグモードを有効にしました")
						} else {
							fmt.Println("デバッグモードを無効にしました")
						}
					} else {
						if debug {
							fmt.Println("現在のデバッグモード: 有効")
						} else {
							fmt.Println("現在のデバッグモード: 無効")
						}
					}
				}
				close(cmd.Done)
			}()
			continue
		case CmdUpdate:
			cmd.Error = p.client.UpdateProperties(cmd.DeviceSpec)
		case CmdAliasList:
			// TODO: 未実装
			cmd.Error = fmt.Errorf("エイリアス関連コマンドは現在実装中です")
		case CmdAliasSet:
			// TODO: 未実装
			cmd.Error = fmt.Errorf("エイリアス関連コマンドは現在実装中です")
		case CmdAliasDelete:
			// TODO: 未実装
			cmd.Error = fmt.Errorf("エイリアス関連コマンドは現在実装中です")
		case CmdAliasGet:
			// TODO: 未実装
			cmd.Error = fmt.Errorf("エイリアス関連コマンドは現在実装中です")
		default:
			panic("未処理のコマンドタイプです")
		}

		// コマンド実行完了を通知（quit以外の全てのコマンド）
		close(cmd.Done)
	}

	// 全ての非同期コマンドが完了するのを待つ
	p.waitGroup.Wait()
}

// DisplayDeviceList は取得したデバイス情報を表示する
func DisplayDeviceList(results []protocol.DevicePropertyResult) {
	for _, result := range results {
		// デバイス名を表示
		names := append(result.Device.Aliases, fmt.Sprintf("%s %s:%d", 
			result.Device.IP, 
			result.Device.EOJ.ClassCode, 
			result.Device.EOJ.InstanceCode))
		
		fmt.Println(names[0])

		// プロパティを表示
		for _, prop := range result.Properties {
			var name string
			if prop.Name != "" {
				name = fmt.Sprintf(" (%s)", prop.Name)
			}
			fmt.Printf("  %s%s: %s\n", prop.EPC, name, prop.EDT)
			
			// 変換された値がある場合は表示
			if prop.Value != nil {
				fmt.Printf("    値: %v\n", prop.Value)
			}
		}
	}
}

// DisplayDeviceProperties はデバイスプロパティを表示する
func DisplayDeviceProperties(result protocol.DevicePropertyResult) {
	// デバイス情報を表示
	fmt.Printf("デバイス: %s %s:%d\n", 
		result.Device.IP, 
		result.Device.EOJ.ClassCode, 
		result.Device.EOJ.InstanceCode)
	
	// エイリアスがあれば表示
	if len(result.Device.Aliases) > 0 {
		fmt.Printf("エイリアス: %v\n", result.Device.Aliases)
	}
	
	// プロパティを表示
	for _, prop := range result.Properties {
		var name string
		if prop.Name != "" {
			name = fmt.Sprintf(" (%s)", prop.Name)
		}
		fmt.Printf("  %s%s: %s\n", prop.EPC, name, prop.EDT)
		
		// 変換された値がある場合は表示
		if prop.Value != nil {
			fmt.Printf("    値: %v\n", prop.Value)
		}
	}
}

// PrintUsage は使用方法を表示する
func PrintUsage() {
	fmt.Println("ECHONET Lite デバイス検出プログラム (WebSocketクライアント)")
	fmt.Println("コマンド:")
	fmt.Println("  discover: ECHONET Lite デバイスの検出")
	fmt.Println("  devices, list [ipAddress] [classCode[:instanceCode]] [-all|-props] [epc1 epc2...]: 検出されたECHONET Liteデバイスの一覧表示")
	fmt.Println("    ipAddress: IPアドレスでフィルター（例: 192.168.0.212 または IPv6アドレス）")
	fmt.Println("    classCode: クラスコード（4桁の16進数、例: 0130）")
	fmt.Println("    instanceCode: インスタンスコード（1-255の数字、例: 0130:1）")
	fmt.Println("    -all: 全てのEPCを表示")
	fmt.Println("    -props: 既知のEPCのみを表示")
	fmt.Println("    epc: 2桁の16進数で指定（例: 80）。複数指定可能")
	fmt.Println("    ※-all, -props, epc は最後に指定されたものが有効になります")
	fmt.Println("  get [ipAddress] classCode[:instanceCode] epc1 [epc2...] [-skip-validation]: プロパティ値の取得")
	fmt.Println("    ipAddress: 対象デバイスのIPアドレス（省略可能、省略時はクラスコードに一致するデバイスが1つだけの場合に自動選択）")
	fmt.Println("    classCode: クラスコード（4桁の16進数、必須）")
	fmt.Println("    instanceCode: インスタンスコード（1-255の数字、省略時は1）")
	fmt.Println("    epc: 取得するプロパティのEPC（2桁の16進数、例: 80）。複数指定可能")
	fmt.Println("    -skip-validation: デバイスの存在チェックをスキップ（タイムアウト動作確認用）")
	fmt.Println("  set [ipAddress] classCode[:instanceCode] property1 [property2...]: プロパティ値の設定")
	fmt.Println("    ipAddress: 対象デバイスのIPアドレス（省略可能、省略時はクラスコードに一致するデバイスが1つだけの場合に自動選択）")
	fmt.Println("    classCode: クラスコード（4桁の16進数、必須）")
	fmt.Println("    instanceCode: インスタンスコード（1-255の数字、省略時は1）")
	fmt.Println("    property: プロパティ指定")
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