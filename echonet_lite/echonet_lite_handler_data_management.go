package echonet_lite

import (
	"bytes"
	"echonet-list/echonet_lite/log"
	"errors"
	"fmt"
	"strings"
	"time"
)

// saveDeviceInfo は、デバイス情報をファイルに保存する共通処理
func (h *ECHONETLiteHandler) saveDeviceInfo() {
	if err := h.devices.SaveToFile(DeviceFileName); err != nil {
		if logger := log.GetLogger(); logger != nil {
			logger.Log("警告: デバイス情報の保存に失敗しました: %v", err)
		}
		// 保存に失敗しても処理は継続
	}
}

// registerDevice は、デバイスのプロパティを登録し、追加・変更されたプロパティを返します。
func (h *ECHONETLiteHandler) registerProperties(device IPAndEOJ, properties Properties) []EPCType {
	h.propMutex.Lock()
	defer h.propMutex.Unlock()

	logger := log.GetLogger()

	// 変更されたプロパティを追跡
	var changedProperties []Property

	// 各プロパティについて処理
	for _, prop := range properties {
		// 現在の値を取得
		currentProp, exists := h.devices.GetProperty(device, prop.EPC)

		// プロパティが新規または値が変更された場合
		if !exists || !bytes.Equal(currentProp.EDT, prop.EDT) {
			// 変更されたプロパティとして追加
			changedProperties = append(changedProperties, prop)

			if h.Debug {
				if !exists {
					fmt.Printf("%v: プロパティ追加: %v\n", device, prop.String(device.EOJ.ClassCode()))
				} else {
					fmt.Printf("%v: プロパティ変更: %v -> %v\n", device,
						currentProp.String(device.EOJ.ClassCode()),
						prop.String(device.EOJ.ClassCode()))
				}
			}
		}
	}

	// デバイスのプロパティを登録
	h.devices.RegisterProperties(device, properties, time.Now())

	// 変更されたプロパティについて通知を送信
	for _, prop := range changedProperties {
		select {
		case h.PropertyChangeCh <- PropertyChangeNotification{
			Device:   device,
			Property: prop,
		}:
			// 送信成功
		default:
			// チャンネルがブロックされている場合は無視
			if logger != nil {
				logger.Log("警告: プロパティ変化通知チャネルがブロックされています")
			}
		}
	}

	result := make([]EPCType, 0, len(changedProperties))
	for _, prop := range changedProperties {
		result = append(result, prop.EPC)
	}
	return result
}

type DeviceAndProperties struct {
	Device     IPAndEOJ
	Properties Properties
}

// ListDevices は、検出されたデバイスの一覧を表示する
func (h *ECHONETLiteHandler) ListDevices(criteria FilterCriteria) []DeviceAndProperties {
	// フィルタリングを実行
	filtered := h.devices.Filter(criteria)

	temp := filtered.ListDevicePropertyData()
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
	return result
}

func (h *ECHONETLiteHandler) SaveAliasFile() error {
	err := h.DeviceAliases.SaveToFile(DeviceAliasesFileName)
	if err != nil {
		return fmt.Errorf("エイリアス情報の保存に失敗しました: %w", err)
	}
	return nil
}

func (h *ECHONETLiteHandler) AliasList() []AliasIDStringPair {
	return h.DeviceAliases.List()
}

func (h *ECHONETLiteHandler) GetAliases(device IPAndEOJ) []string {
	ids := h.devices.GetIDString(device)
	if ids == "" {
		return nil
	}
	return h.DeviceAliases.FindAliasesByIDString(ids)
}

func (h *ECHONETLiteHandler) DeviceStringWithAlias(device IPAndEOJ) string {
	names := h.GetAliases(device)
	names = append(names, device.String())
	return (strings.Join(names, " "))
}

func (h *ECHONETLiteHandler) AliasSet(alias *string, criteria FilterCriteria) error {
	devices := h.devices.Filter(criteria)
	if devices.Len() == 0 {
		return fmt.Errorf("デバイスが見つかりません: %v", criteria)
	}
	if devices.Len() > 1 {
		return TooManyDevicesError{devices.ListIPAndEOJ()}
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

func (h *ECHONETLiteHandler) AliasDelete(alias *string) error {
	if alias == nil {
		return errors.New("エイリアス名が指定されていません")
	}
	if err := h.DeviceAliases.DeleteByAlias(*alias); err != nil {
		return fmt.Errorf("エイリアス %s の削除に失敗しました: %w", *alias, err)
	}
	return h.SaveAliasFile()
}

func (h *ECHONETLiteHandler) AliasGet(alias *string) (*IPAndEOJ, error) {
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

func (h *ECHONETLiteHandler) GetDevices(deviceSpec DeviceSpecifier) []IPAndEOJ {
	// フィルタリング条件を作成
	criteria := FilterCriteria{
		Device: deviceSpec,
	}

	// フィルタリング
	return h.devices.Filter(criteria).ListIPAndEOJ()
}

// SaveGroupFile はグループ情報をファイルに保存する
func (h *ECHONETLiteHandler) SaveGroupFile() error {
	err := h.DeviceGroups.SaveToFile(DeviceGroupsFileName)
	if err != nil {
		return fmt.Errorf("グループ情報の保存に失敗しました: %w", err)
	}
	return nil
}

// GroupList はグループのリストを返す
func (h *ECHONETLiteHandler) GroupList(groupName *string) []GroupDevicePair {
	return h.DeviceGroups.GroupList(groupName)
}

// GroupAdd はグループにデバイスを追加する
func (h *ECHONETLiteHandler) GroupAdd(groupName string, devices []IDString) error {
	err := h.DeviceGroups.GroupAdd(groupName, devices)
	if err != nil {
		return err
	}
	return h.SaveGroupFile()
}

// GroupRemove はグループからデバイスを削除する
func (h *ECHONETLiteHandler) GroupRemove(groupName string, devices []IDString) error {
	err := h.DeviceGroups.GroupRemove(groupName, devices)
	if err != nil {
		return err
	}
	return h.SaveGroupFile()
}

// GroupDelete はグループを削除する
func (h *ECHONETLiteHandler) GroupDelete(groupName string) error {
	err := h.DeviceGroups.GroupDelete(groupName)
	if err != nil {
		return err
	}
	return h.SaveGroupFile()
}

// GetDevicesByGroup はグループ名に対応するデバイスリストを返す
func (h *ECHONETLiteHandler) GetDevicesByGroup(groupName string) ([]IDString, bool) {
	return h.DeviceGroups.GetDevicesByGroup(groupName)
}

func (c *ECHONETLiteHandler) FindDeviceByIDString(id IDString) *IPAndEOJ {
	devices := c.devices.FindByIDString(id)
	if len(devices) == 0 {
		return nil
	}
	if len(devices) == 1 {
		return &devices[0]
	}

	// 同一 IDString のデバイスが複数ある場合、 last update time が一番新しいものを選ぶ
	latest := devices[0]
	for _, d := range devices {
		if c.devices.GetLastUpdateTime(d).After(c.devices.GetLastUpdateTime(latest)) {
			latest = d
		}
	}
	return &latest
}

func (c *ECHONETLiteHandler) GetIDString(device IPAndEOJ) IDString {
	return c.devices.GetIDString(device)
}

// GetLastUpdateTime は、指定されたデバイスの最終更新タイムスタンプを取得します
// タイムスタンプが存在しない場合は time.Time のゼロ値を返します
func (c *ECHONETLiteHandler) GetLastUpdateTime(device IPAndEOJ) time.Time {
	return c.devices.GetLastUpdateTime(device)
}
