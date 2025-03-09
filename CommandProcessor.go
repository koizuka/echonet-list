package main

import (
	"context"
	"fmt"
)

// CommandProcessor は、コマンド処理を担当する構造体
type CommandProcessor struct {
	handler *ECHONETLiteHandler
	cmdChan chan *Command
	done    chan struct{}
	ctx     context.Context    // コンテキスト
	cancel  context.CancelFunc // コンテキストのキャンセル関数
}

// NewCommandProcessor は、CommandProcessor の新しいインスタンスを作成する
func NewCommandProcessor(ctx context.Context, handler *ECHONETLiteHandler) *CommandProcessor {
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
			// フィルタリング条件を作成
			criteria := FilterCriteria{
				IPAddress:      cmd.GetIPAddress(),
				ClassCode:      cmd.GetClassCode(),
				InstanceCode:   cmd.GetInstanceCode(),
				EPCs:           cmd.EPCs,
				PropertyValues: cmd.Properties,
			}
			fmt.Println(p.handler.ListDevices(criteria, cmd.PropMode))
		case CmdHelp:
			PrintUsage()
		case CmdGet:
			ip, eoj, properties, err := p.handler.GetProperties(cmd)
			cmd.Error = err
			if err == nil {
				fmt.Printf("プロパティ取得成功: %s, %v\n", ip, eoj)
				for _, p := range properties {
					propStr := p.String(eoj.ClassCode())
					fmt.Printf("  %v\n", propStr)
				}
			}
		case CmdSet:
			ip, eoj, properties, err := p.handler.SetProperties(cmd)
			cmd.Error = err
			if err == nil {
				fmt.Printf("プロパティ設定成功: %s, %v\n", ip, eoj)
				for _, p := range properties {
					propStr := p.String(eoj.ClassCode())
					fmt.Printf("  %v\n", propStr)
				}
			}
		case CmdDebug:
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
		case CmdUpdate:
			cmd.Error = p.handler.UpdateProperties(cmd)
		default:
			panic("unhandled default case")
		}

		// コマンド実行完了を通知（quit以外の全てのコマンド）
		close(cmd.Done)
	}
}
