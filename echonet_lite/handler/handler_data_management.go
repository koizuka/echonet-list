package handler

import (
	"bytes"
	"echonet-list/echonet_lite"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

// DataManagementHandler は、データ管理機能を担当する構造体
type DataManagementHandler struct {
	devices          Devices                     // デバイス情報
	DeviceAliases    *DeviceAliases              // デバイスエイリアス
	DeviceGroups     *DeviceGroups               // デバイスグループ
	LocationSettings *LocationSettings           // ロケーション設定
	DeviceHistory    DeviceHistoryStore          // デバイス履歴
	propMutex        sync.RWMutex                // プロパティの排他制御用ミューテックス
	notifier         NotificationRelay           // 通知中継
	hookProcessor    PropertyUpdateHookProcessor // プロパティ更新後処理
}

// NewDataManagementHandler は、DataManagementHandlerの新しいインスタンスを作成する
func NewDataManagementHandler(devices Devices, aliases *DeviceAliases, groups *DeviceGroups, locationSettings *LocationSettings, history DeviceHistoryStore, notifier NotificationRelay) *DataManagementHandler {
	return &DataManagementHandler{
		devices:          devices,
		DeviceAliases:    aliases,
		DeviceGroups:     groups,
		LocationSettings: locationSettings,
		DeviceHistory:    history,
		notifier:         notifier,
	}
}

// SetHookProcessor は、プロパティ更新後の追加処理を実行するプロセッサーを設定する
func (h *DataManagementHandler) SetHookProcessor(processor PropertyUpdateHookProcessor) {
	h.hookProcessor = processor
}

// SaveDeviceInfo は、デバイス情報をファイルに保存する
func (h *DataManagementHandler) SaveDeviceInfo() {
	if err := h.devices.SaveToFile(DeviceFileName); err != nil {
		slog.Warn("デバイス情報の保存に失敗しました", "err", err)
		// 保存に失敗しても処理は継続
	}
}

// detectAndRegisterPropertyChanges は、プロパティの変更を検出し、登録と通知を行う
func (h *DataManagementHandler) detectAndRegisterPropertyChanges(device IPAndEOJ, properties Properties) []ChangedProperty {
	// 変更されたプロパティを追跡
	var changedProperties []ChangedProperty

	// 先にデバイスのロックを取得してから、propMutexをロック（ロック順序を統一）
	// 各プロパティについて処理
	for _, prop := range properties {
		// 現在の値を取得（propMutexのロックなしで呼び出し）
		currentProp, exists := h.devices.GetProperty(device, prop.EPC)

		// プロパティが新規または値が変更された場合
		if !exists || !bytes.Equal(currentProp.EDT, prop.EDT) {
			if !exists {
				currentProp = &Property{EPC: prop.EPC, EDT: []byte{}}
			}
			before := currentProp.EDTString(device.EOJ.ClassCode())
			after := prop.EDTString(device.EOJ.ClassCode())
			if before != after {
				// 変更されたプロパティとして追加
				changedProperties = append(changedProperties, ChangedProperty{
					EPC:       prop.EPC,
					beforeEDT: currentProp.EDT,
					afterEDT:  prop.EDT,
				})
			}
		}
	}

	if len(changedProperties) > 0 {
		classCode := device.EOJ.ClassCode()
		changes := make([]string, len(changedProperties))
		for i, p := range changedProperties {
			changes[i] = p.StringForClass(classCode)
		}
		slog.Info("プロパティ更新", "device", h.DeviceStringWithAlias(device), "count", len(changedProperties), "changes", strings.Join(changes, ", "))
	}

	// デバイスのプロパティを登録（propMutexのロックなしで呼び出し）
	h.devices.RegisterProperties(device, properties, time.Now())

	// 変更されたプロパティについて通知を送信
	// propMutexのロックを取得して通知処理
	h.propMutex.Lock()
	defer h.propMutex.Unlock()
	for _, prop := range changedProperties {
		h.notifier.RelayPropertyChangeEvent(device, prop.After())
	}

	return changedProperties
}

// RegisterProperties は、デバイスのプロパティを登録し、追加・変更されたプロパティを返す
func (h *DataManagementHandler) RegisterProperties(device IPAndEOJ, properties Properties) []ChangedProperty {
	// プロパティの変更検出、登録、通知を実行
	changedProperties := h.detectAndRegisterPropertyChanges(device, properties)

	// propMutexのロック外でフック処理を実行（デッドロック防止）
	// プロパティ更新後の追加処理を実行
	if h.hookProcessor != nil {
		if err := h.hookProcessor.ProcessPropertyUpdateHooks(device, properties); err != nil {
			slog.Warn("プロパティ更新後の追加処理でエラー", "device", device, "err", err)
		}
	}

	return changedProperties
}

// ListDevices は、検出されたデバイスの一覧を表示する
func (h *DataManagementHandler) ListDevices(criteria FilterCriteria) []DeviceAndProperties {
	// フィルタリングを実行
	startTime := time.Now()

	// 処理時間の閾値（正常時はこの時間以内に完了すべき）
	const warnThreshold = 1 * time.Second
	const errorThreshold = 5 * time.Second

	filtered := h.devices.Filter(criteria)
	filterDuration := time.Since(startTime)

	// フィルタリングが異常に遅い場合のみログ出力
	if filterDuration > errorThreshold {
		slog.Error("ListDevices: Filter operation took too long", "duration", filterDuration, "criteria", criteria.String())
	} else if filterDuration > warnThreshold {
		slog.Warn("ListDevices: Filter operation is slow", "duration", filterDuration, "criteria", criteria.String())
	}

	// デバイスプロパティデータの取得
	listStartTime := time.Now()
	temp := filtered.ListDevicePropertyData()
	listDuration := time.Since(listStartTime)

	// ListDevicePropertyDataが異常に遅い場合のみログ出力
	if listDuration > errorThreshold {
		slog.Error("ListDevices: ListDevicePropertyData took too long", "duration", listDuration, "deviceCount", len(temp))
	} else if listDuration > warnThreshold {
		slog.Warn("ListDevices: ListDevicePropertyData is slow", "duration", listDuration, "deviceCount", len(temp))
	}

	// 結果の変換
	convertStartTime := time.Now()
	result := make([]DeviceAndProperties, 0, len(temp))
	for _, d := range temp {
		p := make(Properties, 0, len(d.Properties))
		for _, prop := range d.Properties {
			p = append(p, prop)
		}
		result = append(result, DeviceAndProperties{
			Device:     d.Device,
			Properties: p,
		})
	}
	convertDuration := time.Since(convertStartTime)
	totalDuration := time.Since(startTime)

	// 全体の処理時間が異常に長い場合のみログ出力
	if totalDuration > errorThreshold {
		slog.Error("ListDevices: Operation took too long",
			"totalDuration", totalDuration,
			"filterDuration", filterDuration,
			"listDuration", listDuration,
			"convertDuration", convertDuration,
			"resultCount", len(result))
	} else if totalDuration > warnThreshold {
		slog.Warn("ListDevices: Operation is slow",
			"totalDuration", totalDuration,
			"filterDuration", filterDuration,
			"listDuration", listDuration,
			"convertDuration", convertDuration,
			"resultCount", len(result))
	}

	return result
}

// SaveAliasFile は、エイリアス情報をファイルに保存する
func (h *DataManagementHandler) SaveAliasFile() error {
	err := h.DeviceAliases.SaveToFile(DeviceAliasesFileName)
	if err != nil {
		return fmt.Errorf("エイリアス情報の保存に失敗しました: %w", err)
	}
	return nil
}

// AliasList は、エイリアスのリストを返す
func (h *DataManagementHandler) AliasList() []AliasIDStringPair {
	return h.DeviceAliases.List()
}

// GetAliases は、指定されたデバイスのエイリアスを取得する
func (h *DataManagementHandler) GetAliases(device IPAndEOJ) []string {
	ids := h.devices.GetIDString(device)
	if ids == "" {
		return nil
	}
	return h.DeviceAliases.FindAliasesByIDString(ids)
}

// DeviceStringWithAlias は、デバイスの文字列表現にエイリアスを付加する
func (h *DataManagementHandler) DeviceStringWithAlias(device IPAndEOJ) string {
	names := h.GetAliases(device)
	names = append(names, device.String())
	return (strings.Join(names, " "))
}

func (h *DataManagementHandler) IsOffline(device IPAndEOJ) bool {
	return h.devices.IsOffline(device)
}

func (h *DataManagementHandler) SetOffline(device IPAndEOJ, offline bool) {
	h.devices.SetOffline(device, offline)
}

// SetOfflineByIP sets the offline state of all devices with the given IP address
func (h *DataManagementHandler) SetOfflineByIP(ip net.IP, offline bool) {
	h.devices.SetOfflineByIP(ip, offline)
}

// AliasSet は、デバイスにエイリアスを設定する
func (h *DataManagementHandler) AliasSet(alias *string, criteria FilterCriteria) error {
	devices := h.devices.Filter(criteria)
	if devices.Len() == 0 {
		return fmt.Errorf("デバイスが見つかりません: %v", criteria)
	}
	if devices.Len() > 1 {
		return echonet_lite.TooManyDevicesError{Devices: devices.ListIPAndEOJ()}
	}
	found := devices.ListIPAndEOJ()[0]

	ids := h.devices.GetIDString(found)
	if ids == "" {
		return fmt.Errorf("デバイスのIDが見つかりません: %v", found)
	}

	err := h.DeviceAliases.Register(*alias, ids)
	if err != nil {
		return fmt.Errorf("エイリアスを設定できませんでした: %w", err)
	}
	return h.SaveAliasFile()
}

// AliasDelete は、エイリアスを削除する
func (h *DataManagementHandler) AliasDelete(alias *string) error {
	if alias == nil {
		return errors.New("エイリアス名が指定されていません")
	}
	if err := h.DeviceAliases.DeleteByAlias(*alias); err != nil {
		return fmt.Errorf("エイリアス %s の削除に失敗しました: %w", *alias, err)
	}
	return h.SaveAliasFile()
}

// AliasGet は、エイリアスからデバイスを取得する
func (h *DataManagementHandler) AliasGet(alias *string) (*IPAndEOJ, error) {
	if alias == nil {
		return nil, errors.New("エイリアス名が指定されていません")
	}
	ids, ok := h.DeviceAliases.FindByAlias(*alias)
	if !ok {
		return nil, fmt.Errorf("エイリアス %s が見つかりません", *alias)
	}
	devices := h.devices.FindByIDString(ids)
	if len(devices) == 0 {
		return nil, fmt.Errorf("エイリアス %s に紐付いたデバイス %s が見つかりません", *alias, ids)
	}
	device := devices[0]
	return &device, nil
}

// GetDevices は、デバイス指定子に一致するデバイスを取得する
func (h *DataManagementHandler) GetDevices(deviceSpec DeviceSpecifier) []IPAndEOJ {
	// フィルタリング条件を作成
	criteria := FilterCriteria{
		Device: deviceSpec,
	}

	// フィルタリング
	return h.devices.Filter(criteria).ListIPAndEOJ()
}

// SaveGroupFile は、グループ情報をファイルに保存する
func (h *DataManagementHandler) SaveGroupFile() error {
	err := h.DeviceGroups.SaveToFile(DeviceGroupsFileName)
	if err != nil {
		return fmt.Errorf("グループ情報の保存に失敗しました: %w", err)
	}
	return nil
}

// GroupList は、グループのリストを返す
func (h *DataManagementHandler) GroupList(groupName *string) []GroupDevicePair {
	return h.DeviceGroups.GroupList(groupName)
}

// GroupAdd は、グループにデバイスを追加する
func (h *DataManagementHandler) GroupAdd(groupName string, devices []IDString) error {
	err := h.DeviceGroups.GroupAdd(groupName, devices)
	if err != nil {
		return err
	}
	return h.SaveGroupFile()
}

// GroupRemove は、グループからデバイスを削除する
func (h *DataManagementHandler) GroupRemove(groupName string, devices []IDString) error {
	err := h.DeviceGroups.GroupRemove(groupName, devices)
	if err != nil {
		return err
	}
	return h.SaveGroupFile()
}

// GroupDelete は、グループを削除する
func (h *DataManagementHandler) GroupDelete(groupName string) error {
	err := h.DeviceGroups.GroupDelete(groupName)
	if err != nil {
		return err
	}
	return h.SaveGroupFile()
}

// GetDevicesByGroup は、グループ名に対応するデバイスリストを返す
func (h *DataManagementHandler) GetDevicesByGroup(groupName string) ([]IDString, bool) {
	return h.DeviceGroups.GetDevicesByGroup(groupName)
}

// FindDeviceByIDString は、IDStringからデバイスを検索する
func (h *DataManagementHandler) FindDeviceByIDString(id IDString) *IPAndEOJ {
	devices := h.devices.FindByIDString(id)
	if len(devices) == 0 {
		return nil
	}
	if len(devices) == 1 {
		return &devices[0]
	}

	// 同一 IDString のデバイスが複数ある場合、 last update time が一番新しいものを選ぶ
	latest := devices[0]
	for _, d := range devices {
		if h.devices.GetLastUpdateTime(d).After(h.devices.GetLastUpdateTime(latest)) {
			latest = d
		}
	}
	return &latest
}

// GetIDString は、デバイスのIDStringを取得する
func (h *DataManagementHandler) GetIDString(device IPAndEOJ) IDString {
	return h.devices.GetIDString(device)
}

// GetLastUpdateTime は、指定されたデバイスの最終更新タイムスタンプを取得する
func (h *DataManagementHandler) GetLastUpdateTime(device IPAndEOJ) time.Time {
	return h.devices.GetLastUpdateTime(device)
}

// IsKnownDevice は、デバイスが既知かどうかを確認する
func (h *DataManagementHandler) IsKnownDevice(device IPAndEOJ) bool {
	return h.devices.IsKnownDevice(device)
}

// HasEPCInPropertyMap は、指定されたEPCがプロパティマップに含まれているかを確認する
func (h *DataManagementHandler) HasEPCInPropertyMap(device IPAndEOJ, mapType PropertyMapType, epc EPCType) bool {
	return h.devices.HasEPCInPropertyMap(device, mapType, epc)
}

// GetPropertyMap は、指定されたデバイスのプロパティマップを取得する
func (h *DataManagementHandler) GetPropertyMap(device IPAndEOJ, mapType PropertyMapType) PropertyMap {
	return h.devices.GetPropertyMap(device, mapType)
}

// GetProperty は、指定されたデバイスのプロパティを取得する
func (h *DataManagementHandler) GetProperty(device IPAndEOJ, epc EPCType) (*Property, bool) {
	return h.devices.GetProperty(device, epc)
}

// Filter は、条件に一致するデバイスをフィルタリングする
func (h *DataManagementHandler) Filter(criteria FilterCriteria) Devices {
	return h.devices.Filter(criteria)
}

// RegisterDevice は、デバイスを登録する
func (h *DataManagementHandler) RegisterDevice(device IPAndEOJ) {
	h.devices.RegisterDevice(device)
}

// HasIP は、指定されたIPアドレスを持つデバイスが存在するかを確認する
func (h *DataManagementHandler) HasIP(ip net.IP) bool {
	return h.devices.HasIP(ip)
}

// FindByIDString は、IDStringからデバイスを検索する
func (h *DataManagementHandler) FindByIDString(id IDString) []IPAndEOJ {
	return h.devices.FindByIDString(id)
}

// ValidateEPCsInPropertyMap は、指定されたEPCがプロパティマップに含まれているかを確認する
func (h *DataManagementHandler) ValidateEPCsInPropertyMap(device IPAndEOJ, epcs []EPCType, mapType PropertyMapType) (bool, []EPCType, error) {
	invalidEPCs := []EPCType{}

	// デバイスが存在するか確認
	if !h.IsKnownDevice(device) {
		return false, invalidEPCs, fmt.Errorf("デバイスが見つかりません: %v", device)
	}

	// 各EPCがプロパティマップに含まれているか確認
	for _, epc := range epcs {
		if !h.HasEPCInPropertyMap(device, mapType, epc) {
			invalidEPCs = append(invalidEPCs, epc)
		}
	}

	return len(invalidEPCs) == 0, invalidEPCs, nil
}

// FindIPsWithSameNodeProfileID は同じ識別番号(0x83 EDT)を持つ他のIPを検索する
func (h *DataManagementHandler) FindIPsWithSameNodeProfileID(idEDT []byte, excludeIP string) []string {
	return h.devices.FindIPsWithSameNodeProfileID(idEDT, excludeIP)
}

// RemoveAllDevicesByIP は指定IPの全デバイスを削除し、削除したデバイスのリストを返す
func (h *DataManagementHandler) RemoveAllDevicesByIP(ip net.IP) []IPAndEOJ {
	return h.devices.RemoveAllDevicesByIP(ip)
}

// RemoveDevice は、指定されたデバイスをハンドラーから削除する
func (h *DataManagementHandler) RemoveDevice(device IPAndEOJ) error {
	// デバイスが存在するか確認
	if !h.IsKnownDevice(device) {
		return fmt.Errorf("デバイスが見つかりません: %v", device)
	}

	// Devicesからデバイスを削除
	return h.devices.RemoveDevice(device)
}

// SaveLocationSettingsFile は、ロケーション設定をファイルに保存する
func (h *DataManagementHandler) SaveLocationSettingsFile() error {
	if h.LocationSettings == nil {
		return nil
	}
	err := h.LocationSettings.SaveToFile(LocationSettingsFileName)
	if err != nil {
		return fmt.Errorf("ロケーション設定の保存に失敗しました: %w", err)
	}
	return nil
}

// LocationAliasAdd は、ロケーションエイリアスを追加する
func (h *DataManagementHandler) LocationAliasAdd(alias, value string) error {
	if h.LocationSettings == nil {
		return errors.New("LocationSettings is not initialized")
	}
	if err := h.LocationSettings.Aliases.Add(alias, value); err != nil {
		return err
	}
	return h.SaveLocationSettingsFile()
}

// LocationAliasUpdate は、ロケーションエイリアスを更新する
func (h *DataManagementHandler) LocationAliasUpdate(alias, value string) error {
	if h.LocationSettings == nil {
		return errors.New("LocationSettings is not initialized")
	}
	if err := h.LocationSettings.Aliases.Update(alias, value); err != nil {
		return err
	}
	return h.SaveLocationSettingsFile()
}

// LocationAliasDelete は、ロケーションエイリアスを削除する
func (h *DataManagementHandler) LocationAliasDelete(alias string) error {
	if h.LocationSettings == nil {
		return errors.New("LocationSettings is not initialized")
	}
	if err := h.LocationSettings.Aliases.Delete(alias); err != nil {
		return err
	}
	return h.SaveLocationSettingsFile()
}

// GetLocationSettings は、ロケーション設定を取得する
func (h *DataManagementHandler) GetLocationSettings() (map[string]string, []string) {
	if h.LocationSettings == nil {
		return nil, nil
	}
	return h.LocationSettings.Aliases.GetAll(), h.LocationSettings.Order.Get()
}

// SetLocationOrder は、ロケーションの表示順を設定する
func (h *DataManagementHandler) SetLocationOrder(order []string) error {
	if h.LocationSettings == nil {
		return errors.New("LocationSettings is not initialized")
	}
	h.LocationSettings.Order.Set(order)
	return h.SaveLocationSettingsFile()
}

// EnsureLocationInOrder は、指定されたロケーションが順序リストに含まれていることを保証する
func (h *DataManagementHandler) EnsureLocationInOrder(location string) (bool, error) {
	if h.LocationSettings == nil {
		return false, errors.New("LocationSettings is not initialized")
	}
	added := h.LocationSettings.Order.EnsureLocation(location)
	if added {
		if err := h.SaveLocationSettingsFile(); err != nil {
			return added, err
		}
	}
	return added, nil
}
