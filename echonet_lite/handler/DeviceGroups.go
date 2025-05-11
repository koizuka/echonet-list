package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// DeviceGroups はデバイスグループを管理する構造体
type DeviceGroups struct {
	groups map[string][]IDString // グループ名 -> デバイスリスト
	mutex  sync.RWMutex
}

// NewDeviceGroups は DeviceGroups の新しいインスタンスを作成する
func NewDeviceGroups() *DeviceGroups {
	return &DeviceGroups{
		groups: make(map[string][]IDString),
	}
}

// LoadFromFile はファイルからグループ情報を読み込む
func (g *DeviceGroups) LoadFromFile(filename string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// ファイルが存在しない場合は空のグループリストを作成して終了
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		g.groups = make(map[string][]IDString)
		return nil
	}

	// ファイルを開く
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("グループファイルを開けません: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// JSONデコード用の一時構造体
	type GroupEntry struct {
		Group   string     `json:"group"`
		Devices []IDString `json:"devices"`
	}
	var entries []GroupEntry

	// JSONデコード
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&entries); err != nil {
		return fmt.Errorf("グループファイルの解析に失敗しました: %v", err)
	}

	// グループマップを初期化
	g.groups = make(map[string][]IDString)

	// エントリをグループマップに変換
	for _, entry := range entries {
		g.groups[entry.Group] = entry.Devices
	}

	return nil
}

// SaveToFile はグループ情報をファイルに保存する
func (g *DeviceGroups) SaveToFile(filename string) error {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	// ディレクトリが存在しない場合は作成
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ディレクトリの作成に失敗しました: %v", err)
	}

	// JSONエンコード用の一時構造体
	type GroupEntry struct {
		Group   string     `json:"group"`
		Devices []IDString `json:"devices"`
	}

	// グループマップをエントリのスライスに変換
	entries := make([]GroupEntry, 0, len(g.groups))
	for group, devices := range g.groups {
		entries = append(entries, GroupEntry{
			Group:   group,
			Devices: devices,
		})
	}

	// エントリをグループ名でソート
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Group < entries[j].Group
	})

	// ファイルを作成
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("グループファイルの作成に失敗しました: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// JSONエンコード
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entries); err != nil {
		return fmt.Errorf("グループファイルの書き込みに失敗しました: %v", err)
	}

	return nil
}

// ValidateGroupName はグループ名が有効かどうかを検証する
func ValidateGroupName(groupName string) error {
	if !strings.HasPrefix(groupName, "@") {
		return fmt.Errorf("グループ名は '@' で始まる必要があります: %s", groupName)
	}

	if len(groupName) <= 1 {
		return fmt.Errorf("グループ名は '@' の後に少なくとも1文字必要です: %s", groupName)
	}

	// 空白文字を含まないことを確認
	if strings.ContainsAny(groupName, " \t\n\r") {
		return fmt.Errorf("グループ名に空白文字を含めることはできません: %s", groupName)
	}

	return nil
}

// GroupAdd はグループにデバイスを追加する
func (g *DeviceGroups) GroupAdd(groupName string, devices []IDString) error {
	// グループ名の検証
	if err := ValidateGroupName(groupName); err != nil {
		return err
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	// 既存のグループを取得または新規作成
	existingDevices, exists := g.groups[groupName]
	if !exists {
		existingDevices = make([]IDString, 0)
	}

	// 新しいデバイスを追加（重複チェック）
	for _, device := range devices {
		// 既に存在するかチェック
		found := false
		for _, existing := range existingDevices {
			if existing == device {
				found = true
				break
			}
		}

		// 存在しない場合のみ追加
		if !found {
			existingDevices = append(existingDevices, device)
		}
	}

	// 更新されたデバイスリストを保存
	g.groups[groupName] = existingDevices

	return nil
}

// GroupRemove はグループからデバイスを削除する
func (g *DeviceGroups) GroupRemove(groupName string, devices []IDString) error {
	// グループ名の検証
	if err := ValidateGroupName(groupName); err != nil {
		return err
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	// グループが存在するか確認
	existingDevices, exists := g.groups[groupName]
	if !exists {
		return fmt.Errorf("グループが存在しません: %s", groupName)
	}

	// 削除するデバイスがない場合は何もしない
	if len(devices) == 0 {
		return nil
	}

	// 指定されたデバイスを削除
	newDevices := make([]IDString, 0, len(existingDevices))
	for _, existing := range existingDevices {
		// 削除対象かチェック
		shouldKeep := true
		for _, device := range devices {
			if existing == device {
				shouldKeep = false
				break
			}
		}

		// 削除対象でなければ保持
		if shouldKeep {
			newDevices = append(newDevices, existing)
		}
	}

	// 更新されたデバイスリストを保存
	if len(newDevices) == 0 {
		// デバイスがなくなった場合はグループを削除
		delete(g.groups, groupName)
	} else {
		g.groups[groupName] = newDevices
	}

	return nil
}

// GroupDelete はグループを削除する
func (g *DeviceGroups) GroupDelete(groupName string) error {
	// グループ名の検証
	if err := ValidateGroupName(groupName); err != nil {
		return err
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	// グループが存在するか確認
	if _, exists := g.groups[groupName]; !exists {
		return fmt.Errorf("グループが存在しません: %s", groupName)
	}

	// グループを削除
	delete(g.groups, groupName)

	return nil
}

// GroupList はグループのリストを返す
// groupNameがnilの場合は全グループを返す
// groupNameが指定されている場合は指定されたグループの情報を返す
func (g *DeviceGroups) GroupList(groupName *string) []GroupDevicePair {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	result := make([]GroupDevicePair, 0)

	if groupName != nil {
		// 特定のグループが指定された場合
		if devices, exists := g.groups[*groupName]; exists {
			result = append(result, GroupDevicePair{
				Group:   *groupName,
				Devices: devices,
			})
		}
	} else {
		// 全グループを返す場合
		// グループ名をソート
		groupNames := make([]string, 0, len(g.groups))
		for name := range g.groups {
			groupNames = append(groupNames, name)
		}
		sort.Strings(groupNames)

		// ソートされたグループ名でループ
		for _, name := range groupNames {
			devices := g.groups[name]

			result = append(result, GroupDevicePair{
				Group:   name,
				Devices: devices,
			})
		}
	}

	return result
}

// GetDevicesByGroup はグループ名に対応するデバイスリストを返す
func (g *DeviceGroups) GetDevicesByGroup(groupName string) ([]IDString, bool) {
	// グループ名の検証
	if err := ValidateGroupName(groupName); err != nil {
		return nil, false
	}

	g.mutex.RLock()
	defer g.mutex.RUnlock()

	devices, exists := g.groups[groupName]
	if !exists {
		return nil, false
	}

	// コピーを返す
	result := make([]IDString, len(devices))
	copy(result, devices)
	return result, true
}

// GroupDevicePair はグループとデバイスのペアを表す
type GroupDevicePair struct {
	Group   string
	Devices []IDString
}
