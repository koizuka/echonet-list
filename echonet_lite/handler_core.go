package echonet_lite

import (
	"context"
	"echonet-list/echonet_lite/log"
	"fmt"
)

// HandlerCore は、ECHONETLiteHandlerのコア機能を担当する構造体
type HandlerCore struct {
	ctx              context.Context                 // コンテキスト
	cancel           context.CancelFunc              // コンテキストのキャンセル関数
	NotificationCh   chan DeviceNotification         // デバイス通知用チャネル
	PropertyChangeCh chan PropertyChangeNotification // プロパティ変化通知用チャネル
	Debug            bool                            // デバッグモード
}

// NewHandlerCore は、HandlerCoreの新しいインスタンスを作成する
func NewHandlerCore(ctx context.Context, cancel context.CancelFunc, debug bool) *HandlerCore {
	// 通知チャンネルを作成
	notificationCh := make(chan DeviceNotification, 100)           // バッファサイズは100に設定
	propertyChangeCh := make(chan PropertyChangeNotification, 400) // バッファサイズは400に設定

	return &HandlerCore{
		ctx:              ctx,
		cancel:           cancel,
		NotificationCh:   notificationCh,
		PropertyChangeCh: propertyChangeCh,
		Debug:            debug,
	}
}

// Close は、HandlerCoreのリソースを解放する
func (c *HandlerCore) Close() error {
	// コンテキストをキャンセル
	if c.cancel != nil {
		c.cancel()
	}

	// 通知チャネルを閉じる
	if c.NotificationCh != nil {
		close(c.NotificationCh)
	}

	// プロパティ変化通知チャネルを閉じる
	if c.PropertyChangeCh != nil {
		close(c.PropertyChangeCh)
	}

	return nil
}

// SetDebug は、デバッグモードを設定する
func (c *HandlerCore) SetDebug(debug bool) {
	c.Debug = debug
}

// IsDebug は、現在のデバッグモードを返す
func (c *HandlerCore) IsDebug() bool {
	return c.Debug
}

// RelayDeviceEvent は、DeviceEventをDeviceNotificationに変換して中継する
func (c *HandlerCore) RelayDeviceEvent(event DeviceEvent) {
	// DeviceEventをDeviceNotificationに変換して中継
	switch event.Type {
	case DeviceEventAdded:
		select {
		case c.NotificationCh <- DeviceNotification{
			Device: event.Device,
			Type:   DeviceAdded,
		}:
			// 送信成功
		default:
			// チャンネルがブロックされている場合は無視
			if logger := log.GetLogger(); logger != nil {
				logger.Log("警告: 通知チャネルがブロックされています")
			}
		}
	}
}

// RelaySessionTimeoutEvent は、SessionTimeoutEventをDeviceNotificationに変換して中継する
func (c *HandlerCore) RelaySessionTimeoutEvent(event SessionTimeoutEvent) {
	// SessionTimeoutEventをDeviceNotificationに変換して中継
	select {
	case c.NotificationCh <- DeviceNotification{
		Device: event.Device,
		Type:   DeviceTimeout,
		Error:  event.Error,
	}:
		// 送信成功
	default:
		// チャンネルがブロックされている場合は無視
		if logger := log.GetLogger(); logger != nil {
			logger.Log("警告: 通知チャネルがブロックされています")
		}
	}
}

// RelayPropertyChangeEvent は、プロパティ変更通知を中継する
func (c *HandlerCore) RelayPropertyChangeEvent(device IPAndEOJ, property Property) {
	select {
	case c.PropertyChangeCh <- PropertyChangeNotification{
		Device:   device,
		Property: property,
	}:
		// 送信成功
	default:
		// チャンネルがブロックされている場合は無視
		if logger := log.GetLogger(); logger != nil {
			logger.Log("警告: プロパティ変化通知チャネルがブロックされています")
		}
	}
}

// StartEventRelayLoop は、デバイスイベントとセッションタイムアウトイベントを通知チャンネルに中継するゴルーチンを起動する
func (c *HandlerCore) StartEventRelayLoop(deviceEventCh <-chan DeviceEvent, sessionTimeoutCh <-chan SessionTimeoutEvent) {
	go func() {
		for {
			select {
			case event, ok := <-deviceEventCh:
				if !ok {
					// チャンネルが閉じられた場合は終了
					return
				}
				c.RelayDeviceEvent(event)
			case event, ok := <-sessionTimeoutCh:
				if !ok {
					// チャンネルが閉じられた場合は終了
					return
				}
				c.RelaySessionTimeoutEvent(event)
			case <-c.ctx.Done():
				// コンテキストがキャンセルされた場合は終了
				return
			}
		}
	}()
}

// DebugLog は、デバッグモードが有効な場合にメッセージを出力する
func (c *HandlerCore) DebugLog(format string, args ...interface{}) {
	if c.Debug {
		fmt.Printf(format+"\n", args...)
	}
}
