package handler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// HandlerCore は、ECHONETLiteHandlerのコア機能を担当する構造体
type HandlerCore struct {
	ctx                     context.Context                 // コンテキスト
	cancel                  context.CancelFunc              // コンテキストのキャンセル関数
	NotificationCh          chan DeviceNotification         // デバイス通知用チャネル（内部用）
	PropertyChangeCh        chan PropertyChangeNotification // プロパティ変化通知用チャネル
	Debug                   bool                            // デバッグモード
	OperationTracker        *OperationTracker               // 操作追跡システム
	notificationSubscribers []chan DeviceNotification       // 通知購読者のリスト
	subscribersMutex        sync.RWMutex                    // 購読者リストの保護
}

// NewHandlerCore は、HandlerCoreの新しいインスタンスを作成する
func NewHandlerCore(ctx context.Context, cancel context.CancelFunc, debug bool) *HandlerCore {
	// 通知チャンネルを作成
	notificationCh := make(chan DeviceNotification, 100)            // バッファサイズは100に設定
	propertyChangeCh := make(chan PropertyChangeNotification, 2000) // バッファサイズは2000に設定

	// 操作追跡システムを作成
	operationTracker := NewOperationTracker(ctx, 5*time.Second)
	operationTracker.Start()

	core := &HandlerCore{
		ctx:                     ctx,
		cancel:                  cancel,
		NotificationCh:          notificationCh,
		PropertyChangeCh:        propertyChangeCh,
		Debug:                   debug,
		OperationTracker:        operationTracker,
		notificationSubscribers: make([]chan DeviceNotification, 0),
	}

	// ファンアウト処理を開始
	go core.fanoutNotifications()

	return core
}

// Close は、HandlerCoreのリソースを解放する
func (c *HandlerCore) Close() error {
	// 操作追跡システムを停止
	if c.OperationTracker != nil {
		c.OperationTracker.Stop()
	}

	// コンテキストをキャンセルしてfanoutNotifications()の終了をシグナル
	if c.cancel != nil {
		c.cancel()
	}

	// 通知チャネルを閉じる（これによりfanoutNotifications()が終了する）
	if c.NotificationCh != nil {
		close(c.NotificationCh)
	}

	// fanoutNotifications()の終了を少し待つ
	time.Sleep(10 * time.Millisecond)

	// プロパティ変化通知チャネルを閉じる
	if c.PropertyChangeCh != nil {
		close(c.PropertyChangeCh)
	}

	// 購読者チャンネルを閉じる（fanoutNotifications()終了後なので安全）
	c.subscribersMutex.Lock()
	for _, subscriber := range c.notificationSubscribers {
		close(subscriber)
	}
	c.notificationSubscribers = nil
	c.subscribersMutex.Unlock()

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
		slog.Warn("HandlerCore.notify: 通知チャネルがブロックされています", "notificationType", notification.Type, "device", notification.Device.Specifier())
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
	case DeviceEventRemoved:
		c.notify(DeviceNotification{
			Device: event.Device,
			Type:   DeviceRemoved,
		})
	case DeviceEventOffline:
		c.notify(DeviceNotification{
			Device: event.Device,
			Type:   DeviceOffline,
		})
	case DeviceEventOnline:
		// オンライン復旧時はデバイスオンライン通知を送信
		c.notify(DeviceNotification{
			Device: event.Device,
			Type:   DeviceOnline,
		})
	default:
		slog.Warn("未知のDeviceEventType", "eventType", event.Type, "device", event.Device.Specifier())
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

// SubscribeNotifications は、通知を受信するためのチャンネルを作成して返す
func (c *HandlerCore) SubscribeNotifications(bufferSize int) <-chan DeviceNotification {
	ch := make(chan DeviceNotification, bufferSize)
	c.subscribersMutex.Lock()
	c.notificationSubscribers = append(c.notificationSubscribers, ch)
	c.subscribersMutex.Unlock()
	return ch
}

// fanoutNotifications は、内部NotificationChから購読者へ通知を配信する
func (c *HandlerCore) fanoutNotifications() {
	for {
		select {
		case notification, ok := <-c.NotificationCh:
			if !ok {
				// チャンネルが閉じられた場合は終了
				return
			}

			// 全購読者に通知を配信（バッファフルの購読者は切断）
			c.subscribersMutex.Lock()
			activeSubscribers := make([]chan DeviceNotification, 0, len(c.notificationSubscribers))
			for _, subscriber := range c.notificationSubscribers {
				select {
				case subscriber <- notification:
					// 送信成功
					activeSubscribers = append(activeSubscribers, subscriber)
				default:
					// バッファがフルの購読者は切断
					slog.Warn("通知購読者のバッファがフルのため切断します", "notificationType", notification.Type, "device", notification.Device.Specifier())
					close(subscriber)
				}
			}
			c.notificationSubscribers = activeSubscribers
			c.subscribersMutex.Unlock()

		case <-c.ctx.Done():
			// コンテキストがキャンセルされた場合は終了
			return
		}
	}
}
