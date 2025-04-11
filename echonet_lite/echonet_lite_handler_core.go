package echonet_lite

import (
	"context"
	"echonet-list/echonet_lite/log"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	DeviceFileName        = "devices.json"
	DeviceAliasesFileName = "aliases.json"
	DeviceGroupsFileName  = "groups.json"

	CommandTimeout = 3 * time.Second // コマンド実行のタイムアウト時間
)

// NotificationType は通知の種類を表す型
type NotificationType int

const (
	DeviceAdded NotificationType = iota
	DeviceTimeout
)

// DeviceNotification はデバイスに関する通知を表す構造体
type DeviceNotification struct {
	Device IPAndEOJ
	Type   NotificationType
	Error  error // タイムアウトの場合はエラー情報
}

// PropertyChangeNotification はプロパティ変化に関する通知を表す構造体
type PropertyChangeNotification struct {
	Device   IPAndEOJ
	Property Property
}

// ECHONETLiteHandler は、ECHONET Lite の通信処理を担当する構造体
type ECHONETLiteHandler struct {
	session          *Session
	devices          Devices
	propMutex        sync.RWMutex // プロパティの排他制御用ミューテックス
	DeviceAliases    *DeviceAliases
	DeviceGroups     *DeviceGroups
	localDevices     DeviceProperties // 自ノードが所有するデバイスのプロパティ
	Debug            bool
	ctx              context.Context                 // コンテキスト
	cancel           context.CancelFunc              // コンテキストのキャンセル関数
	NotificationCh   chan DeviceNotification         // デバイス通知用チャネル
	PropertyChangeCh chan PropertyChangeNotification // プロパティ変化通知用チャネル
}

// NewECHONETLiteHandler は、ECHONETLiteHandler の新しいインスタンスを作成する
func NewECHONETLiteHandler(ctx context.Context, ip net.IP, seoj EOJ, debug bool) (*ECHONETLiteHandler, error) {
	// タイムアウト付きのコンテキストを作成
	handlerCtx, cancel := context.WithCancel(ctx)

	// 自ノードのセッションを作成
	session, err := CreateSession(handlerCtx, ip, seoj, debug)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("接続に失敗: %w", err)
	}

	// デバイス情報を管理するオブジェクトを作成
	devices := NewDevices()

	// デバイスイベント用チャンネルを作成
	deviceEventCh := make(chan DeviceEvent, 100)
	// Devicesにイベントチャンネルを設定
	devices.SetEventChannel(deviceEventCh)

	// DeviceFileName のファイルが存在するなら読み込む
	err = devices.LoadFromFile(DeviceFileName)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("デバイス情報の読み込みに失敗: %w", err)
	}

	aliases := NewDeviceAliases()

	// DeviceAliasesFileName のファイルが存在するなら読み込む
	err = aliases.LoadFromFile(DeviceAliasesFileName)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("エイリアス情報の読み込みに失敗: %w", err)
	}

	// デバイスグループを管理するオブジェクトを作成
	groups := NewDeviceGroups()

	// DeviceGroupsFileName のファイルが存在するなら読み込む
	err = groups.LoadFromFile(DeviceGroupsFileName)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("グループ情報の読み込みに失敗: %w", err)
	}

	localDevices := make(DeviceProperties)
	operationStatusOn, ok := ProfileSuperClass_PropertyTable.FindAlias("on")
	if !ok {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("プロパティテーブルに on が見つかりません")
	}
	manufacturerCode, ok := ProfileSuperClass_PropertyTable.FindAlias("Experimental")
	if !ok {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("プロパティテーブルに Experimental が見つかりません")
	}
	identificationNumber := IdentificationNumber{
		ManufacturerCode: manufacturerCode.EDT,
		UniqueIdentifier: make([]byte, 13), // 識別番号未設定は13バイトの0
	}

	err = localDevices.Set(NodeProfileObject,
		&operationStatusOn,
		&identificationNumber,
		&manufacturerCode,
		&ECHONETLite_Version,
	)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, err
	}

	err = localDevices.Set(seoj,
		&operationStatusOn,
		&identificationNumber,
		&manufacturerCode,
	)
	if err != nil {
		cancel()
		return nil, err
	}

	// 最後にやること
	err = localDevices.UpdateProfileObjectProperties()
	if err != nil {
		cancel()
		return nil, err
	}

	// 通知チャンネルを作成
	notificationCh := make(chan DeviceNotification, 100)           // バッファサイズは100に設定
	propertyChangeCh := make(chan PropertyChangeNotification, 100) // バッファサイズは100に設定

	// セッションのタイムアウト通知チャンネルを作成
	sessionTimeoutCh := make(chan SessionTimeoutEvent, 100)

	// セッションにタイムアウト通知チャンネルを設定
	session.SetTimeoutChannel(sessionTimeoutCh)

	handler := &ECHONETLiteHandler{
		session:          session,
		devices:          devices,
		DeviceAliases:    aliases,
		DeviceGroups:     groups,
		localDevices:     localDevices,
		Debug:            debug,
		ctx:              handlerCtx,
		cancel:           cancel,
		NotificationCh:   notificationCh,
		PropertyChangeCh: propertyChangeCh,
	}

	// デバイスイベントとセッションタイムアウトイベントを通知チャンネルに中継するゴルーチンを起動
	go func() {
		for {
			select {
			case event, ok := <-deviceEventCh:
				if !ok {
					// チャンネルが閉じられた場合は終了
					return
				}
				// DeviceEventをDeviceNotificationに変換して中継
				switch event.Type {
				case DeviceEventAdded:
					select {
					case notificationCh <- DeviceNotification{
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
			case event, ok := <-sessionTimeoutCh:
				if !ok {
					// チャンネルが閉じられた場合は終了
					return
				}
				// SessionTimeoutEventをDeviceNotificationに変換して中継
				select {
				case notificationCh <- DeviceNotification{
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
			case <-handlerCtx.Done():
				// コンテキストがキャンセルされた場合は終了
				return
			}
		}
	}()

	// INFメッセージのコールバックを設定
	session.OnInf(handler.onInfMessage)
	session.OnReceive(handler.onReceiveMessage)

	return handler, nil
}

// Close は、ECHONETLiteHandler のリソースを解放する
func (h *ECHONETLiteHandler) Close() error {
	// コンテキストをキャンセル
	if h.cancel != nil {
		h.cancel()
	}

	// 通知チャネルを閉じる
	if h.NotificationCh != nil {
		close(h.NotificationCh)
	}

	// プロパティ変化通知チャネルを閉じる
	if h.PropertyChangeCh != nil {
		close(h.PropertyChangeCh)
	}

	return h.session.Close()
}

// StartMainLoop は、メインループを開始する
func (h *ECHONETLiteHandler) StartMainLoop() {
	go h.session.MainLoop()
}

// SetDebug は、デバッグモードを設定する
func (h *ECHONETLiteHandler) SetDebug(debug bool) {
	h.Debug = debug
	h.session.Debug = debug
}

// IsDebug は、現在のデバッグモードを返す
func (h *ECHONETLiteHandler) IsDebug() bool {
	return h.Debug
}

// NotifyNodeList は、自ノードのインスタンスリストを通知する
func (h *ECHONETLiteHandler) NotifyNodeList() error {
	list := InstanceListNotification(h.localDevices.GetInstanceList())
	return h.session.Broadcast(NodeProfileObject, ESVINF, Properties{*list.Property()})
}
