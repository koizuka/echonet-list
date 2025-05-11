package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"fmt"
	"net"
	"time"
)

// ECHONETLiteHandler は、ECHONET Lite の通信処理を担当する構造体
// 内部的に各機能を担当するハンドラを持ち、ファサードとして機能する
type ECHONETLiteHandler struct {
	core             *HandlerCore                    // コア機能
	comm             *CommunicationHandler           // 通信機能
	data             *DataManagementHandler          // データ管理機能
	NotificationCh   chan DeviceNotification         // デバイス通知用チャネル
	PropertyChangeCh chan PropertyChangeNotification // プロパティ変化通知用チャネル
}

// NewECHONETLiteHandler は、ECHONETLiteHandler の新しいインスタンスを作成する
func NewECHONETLiteHandler(ctx context.Context, ip net.IP, debug bool) (*ECHONETLiteHandler, error) {
	// タイムアウト付きのコンテキストを作成
	handlerCtx, cancel := context.WithCancel(ctx)

	// Controller Object
	seoj := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	// 自ノードのセッションを作成
	session, err := CreateSession(handlerCtx, ip, seoj, debug)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("接続に失敗: %w", err)
	}

	// デバイス情報を管理するオブジェクトを作成
	devices := echonet_lite.NewDevices()

	// デバイスイベント用チャンネルを作成
	deviceEventCh := make(chan echonet_lite.DeviceEvent, 100)
	// Devicesにイベントチャンネルを設定
	devices.SetEventChannel(deviceEventCh)

	// DeviceFileName のファイルが存在するなら読み込む
	err = devices.LoadFromFile(DeviceFileName)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("デバイス情報の読み込みに失敗: %w", err)
	}

	aliases := echonet_lite.NewDeviceAliases()

	// DeviceAliasesFileName のファイルが存在するなら読み込む
	err = aliases.LoadFromFile(DeviceAliasesFileName)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("エイリアス情報の読み込みに失敗: %w", err)
	}

	// デバイスグループを管理するオブジェクトを作成
	groups := echonet_lite.NewDeviceGroups()

	// DeviceGroupsFileName のファイルが存在するなら読み込む
	err = groups.LoadFromFile(DeviceGroupsFileName)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, fmt.Errorf("グループ情報の読み込みに失敗: %w", err)
	}

	localDevices := make(echonet_lite.DeviceProperties)
	operationStatusOn, ok := echonet_lite.ProfileSuperClass_PropertyTable.FindAlias("on")
	if !ok {
		cancel() // エラーの場合はコンテキストをキャンセル
		panic("プロパティテーブルに on が見つかりません")
	}
	manufacturerCodeEDT, ok := echonet_lite.ManufacturerCodeEDTs["Experimental"]
	if !ok {
		cancel() // エラーの場合はコンテキストをキャンセル
		panic("プロパティテーブルに Experimental が見つかりません")
	}

	identificationNumber := echonet_lite.IdentificationNumber{
		ManufacturerCode: manufacturerCodeEDT,
		UniqueIdentifier: make([]byte, 13), // 識別番号未設定は13バイトの0
	}

	commonProps := []echonet_lite.Property{
		operationStatusOn,
		*identificationNumber.Property(),
		{echonet_lite.EPCManufacturerCode, manufacturerCodeEDT},
	}
	npoProps := []echonet_lite.Property{*echonet_lite.ECHONETLite_Version.Property()}
	npoProps = append(npoProps, commonProps...)

	// 自ノードのプロファイルオブジェクトを作成
	err = localDevices.Set(echonet_lite.NodeProfileObject, npoProps...)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, err
	}

	controllerProps := commonProps

	// コントローラのプロパティを設定
	err = localDevices.Set(seoj, controllerProps...)
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

	// セッションのタイムアウト通知チャンネルを作成
	sessionTimeoutCh := make(chan SessionTimeoutEvent, 100)

	// セッションにタイムアウト通知チャンネルを設定
	session.SetTimeoutChannel(sessionTimeoutCh)

	// 各ハンドラを初期化
	core := NewHandlerCore(handlerCtx, cancel, debug)
	data := NewDataManagementHandler(devices, aliases, groups, core)
	comm := NewCommunicationHandler(handlerCtx, session, localDevices, data, core, debug)

	// イベント中継ループを開始
	core.StartEventRelayLoop(deviceEventCh, sessionTimeoutCh)

	// INFメッセージのコールバックを設定
	session.OnInf(comm.onInfMessage)
	session.OnReceive(comm.onReceiveMessage)

	// NotificationCh を中継用にラップし、タイムアウト時にオフライン状態を設定
	wrappedCh := make(chan DeviceNotification, 100)
	go func() {
		for ev := range core.NotificationCh {
			if ev.Type == DeviceTimeout {
				// デバイスをオフラインに設定
				data.SetOffline(ev.Device, true)
			}
			wrappedCh <- ev
		}
		close(wrappedCh)
	}()

	// ECHONETLiteHandlerを作成
	handler := &ECHONETLiteHandler{
		core:             core,
		comm:             comm,
		data:             data,
		NotificationCh:   wrappedCh,
		PropertyChangeCh: core.PropertyChangeCh,
	}

	return handler, nil
}

// Close は、ECHONETLiteHandler のリソースを解放する
func (h *ECHONETLiteHandler) Close() error {
	return h.core.Close()
}

// StartMainLoop は、メインループを開始する
func (h *ECHONETLiteHandler) StartMainLoop() {
	go h.comm.session.MainLoop()
}

// SetDebug は、デバッグモードを設定する
func (h *ECHONETLiteHandler) SetDebug(debug bool) {
	h.core.SetDebug(debug)
	h.comm.SetDebug(debug)
}

// IsDebug は、現在のデバッグモードを返す
func (h *ECHONETLiteHandler) IsDebug() bool {
	if h == nil || h.core == nil {
		return false
	}
	return h.core.IsDebug()
}

// NotifyNodeList は、自ノードのインスタンスリストを通知する
func (h *ECHONETLiteHandler) NotifyNodeList() error {
	return h.comm.NotifyNodeList()
}

// GetSelfNodeInstanceListS は、SelfNodeInstanceListSプロパティを取得する
func (h *ECHONETLiteHandler) GetSelfNodeInstanceListS(ip net.IP, isMulti bool) error {
	return h.comm.GetSelfNodeInstanceListS(ip, isMulti)
}

// GetGetPropertyMap は、GetPropertyMapプロパティを取得する
func (h *ECHONETLiteHandler) GetGetPropertyMap(device echonet_lite.IPAndEOJ) error {
	return h.comm.GetGetPropertyMap(device)
}

// Discover は、ECHONET Liteデバイスを検出する
func (h *ECHONETLiteHandler) Discover() error {
	return h.comm.Discover()
}

// GetProperties は、プロパティ値を取得する
func (h *ECHONETLiteHandler) GetProperties(device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType, skipValidation bool) (DeviceAndProperties, error) {
	return h.comm.GetProperties(device, EPCs, skipValidation)
}

// SetProperties は、プロパティ値を設定する
func (h *ECHONETLiteHandler) SetProperties(device echonet_lite.IPAndEOJ, properties echonet_lite.Properties) (DeviceAndProperties, error) {
	return h.comm.SetProperties(device, properties)
}

// UpdateProperties は、フィルタリングされたデバイスのプロパティキャッシュを更新する
func (h *ECHONETLiteHandler) UpdateProperties(criteria echonet_lite.FilterCriteria, force bool) error {
	return h.comm.UpdateProperties(criteria, force)
}

// ListDevices は、検出されたデバイスの一覧を表示する
func (h *ECHONETLiteHandler) ListDevices(criteria echonet_lite.FilterCriteria) []DeviceAndProperties {
	return h.data.ListDevices(criteria)
}

// SaveAliasFile は、エイリアス情報をファイルに保存する
func (h *ECHONETLiteHandler) SaveAliasFile() error {
	return h.data.SaveAliasFile()
}

// AliasList は、エイリアスのリストを返す
func (h *ECHONETLiteHandler) AliasList() []echonet_lite.AliasIDStringPair {
	return h.data.AliasList()
}

// GetAliases は、指定されたデバイスのエイリアスを取得する
func (h *ECHONETLiteHandler) GetAliases(device echonet_lite.IPAndEOJ) []string {
	return h.data.GetAliases(device)
}

// DeviceStringWithAlias は、デバイスの文字列表現にエイリアスを付加する
func (h *ECHONETLiteHandler) DeviceStringWithAlias(device echonet_lite.IPAndEOJ) string {
	return h.data.DeviceStringWithAlias(device)
}

// AliasSet は、デバイスにエイリアスを設定する
func (h *ECHONETLiteHandler) AliasSet(alias *string, criteria echonet_lite.FilterCriteria) error {
	return h.data.AliasSet(alias, criteria)
}

// AliasDelete は、エイリアスを削除する
func (h *ECHONETLiteHandler) AliasDelete(alias *string) error {
	return h.data.AliasDelete(alias)
}

// AliasGet は、エイリアスからデバイスを取得する
func (h *ECHONETLiteHandler) AliasGet(alias *string) (*echonet_lite.IPAndEOJ, error) {
	return h.data.AliasGet(alias)
}

// GetDevices は、デバイス指定子に一致するデバイスを取得する
func (h *ECHONETLiteHandler) GetDevices(deviceSpec echonet_lite.DeviceSpecifier) []echonet_lite.IPAndEOJ {
	return h.data.GetDevices(deviceSpec)
}

// SaveGroupFile は、グループ情報をファイルに保存する
func (h *ECHONETLiteHandler) SaveGroupFile() error {
	return h.data.SaveGroupFile()
}

// GroupList は、グループのリストを返す
func (h *ECHONETLiteHandler) GroupList(groupName *string) []echonet_lite.GroupDevicePair {
	return h.data.GroupList(groupName)
}

// GroupAdd は、グループにデバイスを追加する
func (h *ECHONETLiteHandler) GroupAdd(groupName string, devices []echonet_lite.IDString) error {
	return h.data.GroupAdd(groupName, devices)
}

// GroupRemove は、グループからデバイスを削除する
func (h *ECHONETLiteHandler) GroupRemove(groupName string, devices []echonet_lite.IDString) error {
	return h.data.GroupRemove(groupName, devices)
}

// GroupDelete は、グループを削除する
func (h *ECHONETLiteHandler) GroupDelete(groupName string) error {
	return h.data.GroupDelete(groupName)
}

// GetDevicesByGroup は、グループ名に対応するデバイスリストを返す
func (h *ECHONETLiteHandler) GetDevicesByGroup(groupName string) ([]echonet_lite.IDString, bool) {
	return h.data.GetDevicesByGroup(groupName)
}

// FindDeviceByIDString は、IDStringからデバイスを検索する
func (h *ECHONETLiteHandler) FindDeviceByIDString(id echonet_lite.IDString) *echonet_lite.IPAndEOJ {
	return h.data.FindDeviceByIDString(id)
}

// GetIDString は、デバイスのIDStringを取得する
func (h *ECHONETLiteHandler) GetIDString(device echonet_lite.IPAndEOJ) echonet_lite.IDString {
	return h.data.GetIDString(device)
}

// GetLastUpdateTime は、指定されたデバイスの最終更新タイムスタンプを取得する
func (h *ECHONETLiteHandler) GetLastUpdateTime(device echonet_lite.IPAndEOJ) time.Time {
	if h == nil || h.data == nil {
		return time.Time{}
	}
	return h.data.GetLastUpdateTime(device)
}
