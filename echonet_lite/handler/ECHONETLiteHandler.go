package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/network"
	"fmt"
	"log/slog"
	"net"
	"time"
)

// ECHONETLiteHandler は、ECHONET Lite の通信処理を担当する構造体
// 内部的に各機能を担当するハンドラを持ち、ファサードとして機能する
type ECHONETLiteHandler struct {
	core             *HandlerCore                    // コア機能
	comm             *CommunicationHandler           // 通信機能
	data             *DataManagementHandler          // データ管理機能
	historyFilePath  string                          // 履歴ファイルパス
	PropertyChangeCh chan PropertyChangeNotification // プロパティ変化通知用チャネル
}

type ECHONETLieHandlerOptions struct {
	IP                   net.IP                        // 自ノードのIPアドレス, nilの場合はワイルドカード
	Debug                bool                          // デバッグモード
	ManufacturerCode     string                        // echonet_lite.ManufacturerCodeEDT のキーのいずれか。省略時は Experimental
	UniqueIdentifier     []byte                        // 13バイトのユニーク識別子, nilの場合はMACアドレスから生成
	NetworkMonitorConfig *network.NetworkMonitorConfig // ネットワーク監視設定
	// カスタムファイルパス（空文字の場合はデフォルトファイルを使用）
	DevicesFile string // デバイスファイルパス
	AliasesFile string // エイリアスファイルパス
	GroupsFile  string // グループファイルパス
	// 履歴設定
	HistoryOptions HistoryOptions // 履歴ストアのオプション
	// テスト用設定（CI環境での実行時にファイルアクセスやネットワーク通信を避ける）
	TestMode bool // テストモード（ファイル読み込みとネットワーク通信を無効化）
}

// getFileOrDefault は、カスタムファイル名が空文字の場合にデフォルトファイル名を返す
func getFileOrDefault(customFile, defaultFile string) string {
	if customFile == "" {
		return defaultFile
	}
	return customFile
}

// OfflineManager defines the interface for managing device offline states
type OfflineManager interface {
	IsOffline(device IPAndEOJ) bool
	SetOffline(device IPAndEOJ, offline bool)
	SetOfflineByIP(ip net.IP, offline bool)
}

// handleDeviceTimeout processes device timeout events based on device type
// For NodeProfile: marks all devices with the same IP as offline
// For other devices: marks as offline only if the NodeProfile is already offline
func handleDeviceTimeout(device IPAndEOJ, manager OfflineManager) {
	// NodeProfileの場合は、そのIPの全デバイスをオフラインに
	if device.EOJ.ClassCode() == echonet_lite.NodeProfile_ClassCode {
		slog.Info("NodeProfileタイムアウト: IPの全デバイスをオフラインに設定", "ip", device.IP)
		manager.SetOfflineByIP(device.IP, true)
	} else {
		// NodeProfile以外の場合、NodeProfileがオフラインの場合のみオフラインに
		nodeProfile := IPAndEOJ{
			IP:  device.IP,
			EOJ: echonet_lite.NodeProfileObject,
		}
		if manager.IsOffline(nodeProfile) {
			slog.Info("デバイスタイムアウト: NodeProfileがオフラインのため、デバイスをオフラインに設定", "device", device.Specifier())
			manager.SetOffline(device, true)
		} else {
			slog.Info("デバイスタイムアウト: NodeProfileがオンラインのため、オフライン設定をスキップ", "device", device.Specifier())
		}
	}
}

// handleDeviceOnline processes device online events to potentially recover NodeProfile
// For non-NodeProfile devices: if the NodeProfile of the same IP is offline, try to recover it by UpdateProperties
func handleDeviceOnline(device IPAndEOJ, handler *ECHONETLiteHandler) {
	// NodeProfile以外のデバイスがオンラインになった場合
	if device.EOJ.ClassCode() != echonet_lite.NodeProfile_ClassCode {
		nodeProfile := IPAndEOJ{
			IP:  device.IP,
			EOJ: echonet_lite.NodeProfileObject,
		}

		// NodeProfileがオフラインの場合、復活を試みる
		if handler.IsOffline(nodeProfile) {
			slog.Info("デバイスオンライン: NodeProfileがオフラインのため復活を試行", "device", device.Specifier(), "nodeProfile", nodeProfile.Specifier())

			// NodeProfileのプロパティを更新することで生存確認
			nodeProfileClassCode := echonet_lite.NodeProfile_ClassCode
			criteria := FilterCriteria{
				Device: DeviceSpecifier{
					IP:           &device.IP,
					ClassCode:    &nodeProfileClassCode,
					InstanceCode: nil, // 全インスタンス
				},
				ExcludeOffline: false, // オフラインデバイスも対象に含める
			}

			err := handler.UpdateProperties(criteria, true) // forceフラグをtrueで実行
			if err != nil {
				slog.Warn("NodeProfile復活の試行に失敗", "nodeProfile", nodeProfile.Specifier(), "error", err)
			} else {
				slog.Info("NodeProfile復活処理を実行", "nodeProfile", nodeProfile.Specifier())
			}
		}
	}
}

// NewECHONETLiteHandler は、ECHONETLiteHandler の新しいインスタンスを作成する
func NewECHONETLiteHandler(ctx context.Context, options ECHONETLieHandlerOptions) (*ECHONETLiteHandler, error) {
	// タイムアウト付きのコンテキストを作成
	handlerCtx, cancel := context.WithCancel(ctx)

	// Controller Object
	seoj := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	// デバイス情報を管理するオブジェクトを作成
	devices := NewDevices()

	// デバイスイベント用チャンネルを作成
	deviceEventCh := make(chan DeviceEvent, 100)
	// Devicesにイベントチャンネルを設定
	devices.SetEventChannel(deviceEventCh)

	// デバイス情報を読み込む（テストモードでは省略）
	if !options.TestMode {
		devicesFile := getFileOrDefault(options.DevicesFile, DeviceFileName)
		slog.Info("デバイスファイルを使用", "file", devicesFile)
		err := devices.LoadFromFile(devicesFile)
		if err != nil {
			cancel() // エラーの場合はコンテキストをキャンセル
			slog.Error("デバイス情報の読み込みに失敗", "file", devicesFile, "error", err)
			return nil, fmt.Errorf("デバイス情報の読み込みに失敗 (file: %s): %w", devicesFile, err)
		}
		slog.Info("デバイス情報の読み込み完了", "file", devicesFile, "deviceCount", devices.CountAll())
	}

	aliases := NewDeviceAliases()

	// エイリアス情報を読み込む（テストモードでは省略）
	if !options.TestMode {
		aliasesFile := getFileOrDefault(options.AliasesFile, DeviceAliasesFileName)
		slog.Info("エイリアスファイルを使用", "file", aliasesFile)
		err := aliases.LoadFromFile(aliasesFile)
		if err != nil {
			cancel() // エラーの場合はコンテキストをキャンセル
			slog.Error("エイリアス情報の読み込みに失敗", "file", aliasesFile, "error", err)
			return nil, fmt.Errorf("エイリアス情報の読み込みに失敗 (file: %s): %w", aliasesFile, err)
		}
		slog.Info("エイリアス情報の読み込み完了", "file", aliasesFile, "aliasCount", aliases.Count())
	}

	groups := NewDeviceGroups()

	// グループ情報を読み込む（テストモードでは省略）
	if !options.TestMode {
		groupsFile := getFileOrDefault(options.GroupsFile, DeviceGroupsFileName)
		slog.Info("グループファイルを使用", "file", groupsFile)
		err := groups.LoadFromFile(groupsFile)
		if err != nil {
			cancel() // エラーの場合はコンテキストをキャンセル
			slog.Error("グループ情報の読み込みに失敗", "file", groupsFile, "error", err)
			return nil, fmt.Errorf("グループ情報の読み込みに失敗 (file: %s): %w", groupsFile, err)
		}
		slog.Info("グループ情報の読み込み完了", "file", groupsFile, "groupCount", groups.Count())
	}

	// 自ノードのセッションを作成（テストモードでは省略）
	var session *Session
	var err error
	if !options.TestMode {
		session, err = CreateSession(handlerCtx, options.IP, seoj, options.Debug, options.NetworkMonitorConfig, devices.IsOffline)
		if err != nil {
			cancel() // エラーの場合はコンテキストをキャンセル
			return nil, fmt.Errorf("接続に失敗: %w", err)
		}
	}

	localDevices := make(DeviceProperties)
	operationStatusOn, ok := echonet_lite.ProfileSuperClass_PropertyTable.FindAlias("on")
	if !ok {
		cancel() // エラーの場合はコンテキストをキャンセル
		panic("プロパティテーブルに on が見つかりません")
	}
	manufacturerCode := options.ManufacturerCode
	if manufacturerCode == "" {
		manufacturerCode = "Experimental" // デフォルトは Experimental
	}
	manufacturerCodeEDT, ok := echonet_lite.ManufacturerCodeEDTs[manufacturerCode]
	if !ok {
		cancel() // エラーの場合はコンテキストをキャンセル
		panic(fmt.Sprintf("プロパティテーブルに %v が見つかりません", manufacturerCode))
	}

	// UniqueIdentifier を生成
	uniqueIdentifier := make([]byte, 13) // デフォルトは13バイトの0
	if options.UniqueIdentifier != nil {
		// ユーザー指定のユニーク識別子を使用
		copy(uniqueIdentifier, options.UniqueIdentifier[0:13])
	} else {
		genId, err := GenerateUniqueIdentifierFromMACAddress()
		if err != nil {
			cancel() // エラーの場合はコンテキストをキャンセル
			return nil, fmt.Errorf("ユニーク識別子の生成に失敗: %w", err)
		}
		copy(uniqueIdentifier, genId[:])
	}

	identificationNumber := echonet_lite.IdentificationNumber{
		ManufacturerCode: manufacturerCodeEDT,
		UniqueIdentifier: uniqueIdentifier,
	}
	slog.Info("ユニーク識別子", "identificationNumber", identificationNumber.String())

	commonProps := []Property{
		operationStatusOn,
		*identificationNumber.Property(),
		{EPC: echonet_lite.EPCManufacturerCode, EDT: manufacturerCodeEDT},
	}
	npoProps := []Property{*echonet_lite.ECHONETLite_Version.Property()}
	npoProps = append(npoProps, commonProps...)

	// NodeProfileObject用のStatus Announcement Property Mapを作成(現在は空)
	npoAnnouncementMap := make(PropertyMap)

	npoProps = append(npoProps, Property{
		EPC: echonet_lite.EPCStatusAnnouncementPropertyMap,
		EDT: npoAnnouncementMap.Encode(),
	})

	// 自ノードのプロファイルオブジェクトを作成
	err = localDevices.Set(echonet_lite.NodeProfileObject, npoProps...)
	if err != nil {
		cancel() // エラーの場合はコンテキストをキャンセル
		return nil, err
	}

	// コントローラー用のStatus Announcement Property Mapを作成
	// 設置場所の変更をアナウンスするように設定
	controllerAnnouncementMap := make(PropertyMap)
	controllerAnnouncementMap.Set(echonet_lite.EPCInstallationLocation) // 0x81

	controllerProps := commonProps
	controllerProps = append(controllerProps, Property{
		EPC: echonet_lite.EPCStatusAnnouncementPropertyMap,
		EDT: controllerAnnouncementMap.Encode(),
	})
	controllerProps = append(controllerProps, Property{
		EPC: echonet_lite.EPCInstallationLocation,
		EDT: []byte{0x00},
	}) // 設置場所：未設定

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

	// セッションにタイムアウト通知チャンネルを設定（テストモードでは省略）
	if !options.TestMode && session != nil {
		session.SetTimeoutChannel(sessionTimeoutCh)
	}

	// 各ハンドラを初期化
	core := NewHandlerCore(handlerCtx, cancel, options.Debug)

	// 履歴ストアを作成
	historyOpts := options.HistoryOptions
	if historyOpts.PerDeviceSettableLimit == 0 && historyOpts.PerDeviceNonSettableLimit == 0 {
		// オプションが指定されていない場合はデフォルトを使用
		historyOpts = DefaultHistoryOptions()
		// HistoryFilePathだけは引き継ぐ
		historyOpts.HistoryFilePath = options.HistoryOptions.HistoryFilePath
	}
	history := NewMemoryDeviceHistoryStore(historyOpts)

	// 履歴ファイルの読み込み（テストモードでは省略、ファイルパスが指定されている場合のみ）
	if !options.TestMode && historyOpts.HistoryFilePath != "" {
		slog.Info("履歴ファイルを使用", "file", historyOpts.HistoryFilePath)
		// ロード時のフィルター設定
		filter := HistoryLoadFilter{
			PerDeviceSettableLimit:    historyOpts.PerDeviceSettableLimit,
			PerDeviceNonSettableLimit: historyOpts.PerDeviceNonSettableLimit,
		}
		err := history.LoadFromFile(historyOpts.HistoryFilePath, filter)
		if err != nil {
			slog.Warn("履歴ファイルの読み込みに失敗（新規作成します）", "file", historyOpts.HistoryFilePath, "error", err)
		} else {
			slog.Info("履歴ファイルの読み込み完了", "file", historyOpts.HistoryFilePath)
		}
	}

	data := NewDataManagementHandler(devices, aliases, groups, history, core)
	var comm *CommunicationHandler
	if !options.TestMode && session != nil {
		comm = NewCommunicationHandler(handlerCtx, session, localDevices, data, core, options.Debug)
		// プロパティ更新後のフック処理を設定
		data.SetHookProcessor(comm)
	}

	// イベント中継ループを開始（テストモードでは省略）
	if !options.TestMode {
		core.StartEventRelayLoop(deviceEventCh, sessionTimeoutCh)
	}

	// INFメッセージのコールバックを設定（テストモードでは省略）
	if !options.TestMode && session != nil && comm != nil {
		session.OnInf(comm.onInfMessage)
		session.OnReceive(comm.onReceiveMessage)
	}

	// ECHONETLiteHandlerを作成
	handler := &ECHONETLiteHandler{
		core:             core,
		comm:             comm,
		data:             data,
		historyFilePath:  historyOpts.HistoryFilePath,
		PropertyChangeCh: core.PropertyChangeCh,
	}

	// タイムアウト時にオフライン状態を設定するgoroutineを起動
	// SubscribeNotifications を使用して専用チャンネルを取得
	subscribedCh := core.SubscribeNotifications(100)
	go func() {
		for ev := range subscribedCh {
			if ev.Type == DeviceTimeout {
				handleDeviceTimeout(ev.Device, data)
			}
			if ev.Type == DeviceOnline {
				handleDeviceOnline(ev.Device, handler)
			}
		}
	}()

	return handler, nil
}

// Close は、ECHONETLiteHandler のリソースを解放する
// 履歴ファイルの保存に失敗した場合でもエラーを返さず、ログに記録するのみとする。
// これは、履歴データの保存失敗がアプリケーションの正常終了を妨げるべきではないためである。
func (h *ECHONETLiteHandler) Close() error {
	// 履歴ファイルの保存（ファイルパスが指定されている場合のみ）
	if h.historyFilePath != "" && h.data != nil && h.data.DeviceHistory != nil {
		slog.Info("履歴ファイルを保存", "file", h.historyFilePath)
		err := h.data.DeviceHistory.SaveToFile(h.historyFilePath)
		if err != nil {
			// 履歴保存エラーはログに記録するのみで、Close()自体は失敗させない
			// 履歴データは重要だが、保存失敗がシステム終了を妨げるべきではない
			slog.Error("履歴ファイルの保存に失敗", "file", h.historyFilePath, "error", err)
		} else {
			slog.Info("履歴ファイルの保存完了", "file", h.historyFilePath)
		}
	}
	return h.core.Close()
}

// GetCore は、HandlerCoreを取得する
func (h *ECHONETLiteHandler) GetCore() *HandlerCore {
	return h.core
}

// GetDataManagementHandler は、DataManagementHandlerを取得する
func (h *ECHONETLiteHandler) GetDataManagementHandler() *DataManagementHandler {
	return h.data
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

// DebugSetOffline sets the offline state of a device for debugging purposes
func (h *ECHONETLiteHandler) DebugSetOffline(target string, offline bool) error {
	// Parse the target device identifier
	ipAndEOJ, err := ParseDeviceIdentifier(target)
	if err != nil {
		return err
	}

	// Set the device offline/online state directly using DataManagementHandler
	h.data.SetOffline(ipAndEOJ, offline)
	return nil
}

// IsOfflineDevice checks if a device is currently offline
func (h *ECHONETLiteHandler) IsOfflineDevice(device IPAndEOJ) bool {
	return h.data.IsOffline(device)
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
func (h *ECHONETLiteHandler) GetGetPropertyMap(device IPAndEOJ) error {
	return h.comm.GetGetPropertyMap(device)
}

// Discover は、ECHONET Liteデバイスを検出する
func (h *ECHONETLiteHandler) Discover() error {
	return h.comm.Discover()
}

// GetProperties は、プロパティ値を取得する
func (h *ECHONETLiteHandler) GetProperties(device IPAndEOJ, EPCs []EPCType, skipValidation bool) (DeviceAndProperties, error) {
	return h.comm.GetProperties(device, EPCs, skipValidation)
}

// SetProperties は、プロパティ値を設定する
func (h *ECHONETLiteHandler) SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error) {
	return h.comm.SetProperties(device, properties)
}

// UpdateProperties は、フィルタリングされたデバイスのプロパティキャッシュを更新する
func (h *ECHONETLiteHandler) UpdateProperties(criteria FilterCriteria, force bool) error {
	if h.comm == nil {
		// テストモードではCommunicationHandlerが無いため、何も実行しない
		return nil
	}
	return h.comm.UpdateProperties(criteria, force)
}

// ListDevices は、検出されたデバイスの一覧を表示する
func (h *ECHONETLiteHandler) ListDevices(criteria FilterCriteria) []DeviceAndProperties {
	return h.data.ListDevices(criteria)
}

// SaveAliasFile は、エイリアス情報をファイルに保存する
func (h *ECHONETLiteHandler) SaveAliasFile() error {
	return h.data.SaveAliasFile()
}

// AliasList は、エイリアスのリストを返す
func (h *ECHONETLiteHandler) AliasList() []AliasIDStringPair {
	return h.data.AliasList()
}

// GetAliases は、指定されたデバイスのエイリアスを取得する
func (h *ECHONETLiteHandler) GetAliases(device IPAndEOJ) []string {
	return h.data.GetAliases(device)
}

// DeviceStringWithAlias は、デバイスの文字列表現にエイリアスを付加する
func (h *ECHONETLiteHandler) DeviceStringWithAlias(device IPAndEOJ) string {
	return h.data.DeviceStringWithAlias(device)
}

// AliasSet は、デバイスにエイリアスを設定する
func (h *ECHONETLiteHandler) AliasSet(alias *string, criteria FilterCriteria) error {
	return h.data.AliasSet(alias, criteria)
}

// AliasDelete は、エイリアスを削除する
func (h *ECHONETLiteHandler) AliasDelete(alias *string) error {
	return h.data.AliasDelete(alias)
}

// AliasGet は、エイリアスからデバイスを取得する
func (h *ECHONETLiteHandler) AliasGet(alias *string) (*IPAndEOJ, error) {
	return h.data.AliasGet(alias)
}

// GetDeviceByAlias は、エイリアスからデバイスを取得する（client.AliasManagerインターフェース用）
func (h *ECHONETLiteHandler) GetDeviceByAlias(alias string) (IPAndEOJ, bool) {
	device, err := h.data.AliasGet(&alias)
	if err != nil {
		return IPAndEOJ{}, false
	}
	if device == nil {
		return IPAndEOJ{}, false
	}
	return *device, true
}

// GetDevices は、デバイス指定子に一致するデバイスを取得する
func (h *ECHONETLiteHandler) GetDevices(deviceSpec DeviceSpecifier) []IPAndEOJ {
	return h.data.GetDevices(deviceSpec)
}

// SaveGroupFile は、グループ情報をファイルに保存する
func (h *ECHONETLiteHandler) SaveGroupFile() error {
	return h.data.SaveGroupFile()
}

// GroupList は、グループのリストを返す
func (h *ECHONETLiteHandler) GroupList(groupName *string) []GroupDevicePair {
	return h.data.GroupList(groupName)
}

// GroupAdd は、グループにデバイスを追加する
func (h *ECHONETLiteHandler) GroupAdd(groupName string, devices []IDString) error {
	return h.data.GroupAdd(groupName, devices)
}

// GroupRemove は、グループからデバイスを削除する
func (h *ECHONETLiteHandler) GroupRemove(groupName string, devices []IDString) error {
	return h.data.GroupRemove(groupName, devices)
}

// GroupDelete は、グループを削除する
func (h *ECHONETLiteHandler) GroupDelete(groupName string) error {
	return h.data.GroupDelete(groupName)
}

// GetDevicesByGroup は、グループ名に対応するデバイスリストを返す
func (h *ECHONETLiteHandler) GetDevicesByGroup(groupName string) ([]IDString, bool) {
	return h.data.GetDevicesByGroup(groupName)
}

// FindDeviceByIDString は、IDStringからデバイスを検索する
func (h *ECHONETLiteHandler) FindDeviceByIDString(id IDString) *IPAndEOJ {
	return h.data.FindDeviceByIDString(id)
}

// GetIDString は、デバイスのIDStringを取得する
func (h *ECHONETLiteHandler) GetIDString(device IPAndEOJ) IDString {
	return h.data.GetIDString(device)
}

// GetLastUpdateTime は、指定されたデバイスの最終更新タイムスタンプを取得する
func (h *ECHONETLiteHandler) GetLastUpdateTime(device IPAndEOJ) time.Time {
	if h == nil || h.data == nil {
		return time.Time{}
	}
	return h.data.GetLastUpdateTime(device)
}

// PropertyDescProviderインターフェースの実装

// GetAllPropertyAliases は、全てのプロパティエイリアスを取得する
func (h *ECHONETLiteHandler) GetAllPropertyAliases() map[string]echonet_lite.PropertyDescription {
	// 一時的な実装: 空のマップを返す
	return make(map[string]echonet_lite.PropertyDescription)
}

// GetPropertyDesc は、指定されたクラスコードとEPCのプロパティ記述を取得する
func (h *ECHONETLiteHandler) GetPropertyDesc(classCode EOJClassCode, epc EPCType) (*echonet_lite.PropertyDesc, bool) {
	// 一時的な実装: 常にfalseを返す
	return nil, false
}

// IsPropertyDefaultEPC は、指定されたプロパティがデフォルトEPCかどうかを判定する
func (h *ECHONETLiteHandler) IsPropertyDefaultEPC(classCode EOJClassCode, epc EPCType) bool {
	// 一時的な実装: 常にfalseを返す
	return false
}

// FindPropertyAlias は、指定されたエイリアスからプロパティを検索する
func (h *ECHONETLiteHandler) FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool) {
	// 一時的な実装: 常に空のプロパティとfalseを返す
	return Property{}, false
}

// AvailablePropertyAliases は、指定されたクラスコードで利用可能なプロパティエイリアスを取得する
func (h *ECHONETLiteHandler) AvailablePropertyAliases(classCode EOJClassCode) map[string]echonet_lite.PropertyDescription {
	// 一時的な実装: 空のマップを返す
	return make(map[string]echonet_lite.PropertyDescription)
}

// RemoveDevice は、指定されたデバイスをハンドラーから削除する
func (h *ECHONETLiteHandler) RemoveDevice(device IPAndEOJ) error {
	return h.data.RemoveDevice(device)
}

// IsOffline は、指定されたデバイスがオフラインかどうかを返す
func (h *ECHONETLiteHandler) IsOffline(device IPAndEOJ) bool {
	return h.data.IsOffline(device)
}
