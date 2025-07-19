package handler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// HandlerCore は、ECHONETLiteHandlerのコア機能を担当する構造体
type HandlerCore struct {
	ctx              context.Context                 // コンテキスト
	cancel           context.CancelFunc              // コンテキストのキャンセル関数
	NotificationCh   chan DeviceNotification         // デバイス通知用チャネル
	PropertyChangeCh chan PropertyChangeNotification // プロパティ変化通知用チャネル
	Debug            bool                            // デバッグモード
	OperationTracker *OperationTracker               // 操作追跡システム
}

// NewHandlerCore は、HandlerCoreの新しいインスタンスを作成する
func NewHandlerCore(ctx context.Context, cancel context.CancelFunc, debug bool) *HandlerCore {
	// 通知チャンネルを作成
	notificationCh := make(chan DeviceNotification, 100)            // バッファサイズは100に設定
	propertyChangeCh := make(chan PropertyChangeNotification, 2000) // バッファサイズは2000に設定

	// 操作追跡システムを作成
	operationTracker := NewOperationTracker(ctx, 5*time.Second)
	operationTracker.Start()

	return &HandlerCore{
		ctx:              ctx,
		cancel:           cancel,
		NotificationCh:   notificationCh,
		PropertyChangeCh: propertyChangeCh,
		Debug:            debug,
		OperationTracker: operationTracker,
	}
}

// Close は、HandlerCoreのリソースを解放する
func (c *HandlerCore) Close() error {
	// 操作追跡システムを停止
	if c.OperationTracker != nil {
		c.OperationTracker.Stop()
	}

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

func (c *HandlerCore) notify(notification DeviceNotification) {
	select {
	case c.NotificationCh <- notification:
		// 送信成功
	default:
		// チャンネルがブロックされている場合は無視
		slog.Warn("通知チャネルがブロックされています")
	}
}

// RelayDeviceEvent は、DeviceEventをDeviceNotificationに変換して中継する
func (c *HandlerCore) RelayDeviceEvent(event DeviceEvent) {
	// DeviceEventをDeviceNotificationに変換して中継
	switch event.Type {
	case DeviceEventAdded:
		c.notify(DeviceNotification{
			Device: event.Device,
			Type:   DeviceAdded,
		})
	case DeviceEventOffline:
		c.notify(DeviceNotification{
			Device: event.Device,
			Type:   DeviceOffline,
		})
	case DeviceEventOnline:
		c.notify(DeviceNotification{
			Device: event.Device,
			Type:   DeviceOnline,
		})
	}
}

// RelaySessionTimeoutEvent は、SessionTimeoutEventをDeviceNotificationに変換して中継する
func (c *HandlerCore) RelaySessionTimeoutEvent(event SessionTimeoutEvent) {
	// SessionTimeoutEventをDeviceNotificationに変換して中継
	c.notify(DeviceNotification{
		Device: event.Device,
		Type:   DeviceTimeout,
		Error:  event.Error,
	})
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
		slog.Warn("プロパティ変化通知チャネルがブロックされています")
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
