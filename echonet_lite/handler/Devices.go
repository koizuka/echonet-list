package handler

import (
	"bytes"
	"echonet-list/echonet_lite"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type EPCPropertyMap map[EPCType]Property

// MarshalJSON は EPCPropertyMap を {"0xEPC": "Base64(EDT)"} 形式のJSONにエンコードします。
func (m EPCPropertyMap) MarshalJSON() ([]byte, error) {
	stringMap := make(map[string]string)
	for epc, prop := range m {
		epcStr := fmt.Sprintf("0x%02x", byte(epc))
		edtBase64 := base64.StdEncoding.EncodeToString(prop.EDT)
		stringMap[epcStr] = edtBase64
	}
	return json.Marshal(stringMap)
}

// UnmarshalJSON は新旧両方のフォーマットから EPCPropertyMap をデコードします。
// 新フォーマット: {"0xEPC": "Base64(EDT)"}
// 旧フォーマット: {"EPC(decimal)": {"EPC": EPC(decimal number), "EDT": "Base64(EDT)"}}
func (m *EPCPropertyMap) UnmarshalJSON(data []byte) error {
	// まず新フォーマット {"0xEPC": "Base64(EDT)"} としてデコードを試みる
	var newFormatMap map[string]string
	if err := json.Unmarshal(data, &newFormatMap); err == nil {
		// キーが "0x" で始まっているかチェック (より厳密な新フォーマット判定)
		isLikelyNewFormat := true
		if len(newFormatMap) > 0 {
			for k := range newFormatMap {
				if !strings.HasPrefix(k, "0x") && !strings.HasPrefix(k, "0X") {
					isLikelyNewFormat = false
					break
				}
			}
		} else {
			// 空のマップの場合はどちらとも言えないが、新フォーマットとして扱う
			isLikelyNewFormat = true
		}

		if isLikelyNewFormat {
			result := make(EPCPropertyMap)
			for epcStr, edtBase64 := range newFormatMap {
				var epc EPCType
				// EPCTypeのUnmarshalJSONは "0x..." と 10進数の両方を扱えるが、ここでは "0x..." のみを期待
				if err := json.Unmarshal([]byte(`"`+epcStr+`"`), &epc); err != nil {
					// 新フォーマットだがキーが不正な場合はエラーとする
					return fmt.Errorf("invalid EPC key format in new format %q: %w", epcStr, err)
				}
				edt, err := base64.StdEncoding.DecodeString(edtBase64)
				if err != nil {
					return fmt.Errorf("invalid base64 EDT for EPC %s: %w", epcStr, err)
				}
				result[epc] = Property{EPC: epc, EDT: edt}
			}
			*m = result
			return nil
		}
		// 新フォーマットのマップとしてデコードできたが、キーが "0x" 形式でなければ旧フォーマットの可能性あり
	}

	// 新フォーマットで失敗した場合、またはキー形式が一致しなかった場合、
	// 旧フォーマット {"EPC(decimal)": {"EPC": number, "EDT": string}} としてデコードを試みる
	// Property.EPC が数値なので、一時的な型を使う
	type oldPropertyFormat struct {
		EPC json.Number `json:"EPC"` // Use json.Number to handle number
		EDT string      `json:"EDT"` // Base64 string
	}
	var oldFormatMap map[string]oldPropertyFormat // キーは10進数文字列

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber() // 数値を json.Number としてデコードする

	if err := decoder.Decode(&oldFormatMap); err != nil {
		// どちらのフォーマットでもデコードできなければエラー
		return fmt.Errorf("failed to unmarshal EPCPropertyMap in new or old format: %w", err)
	}

	result := make(EPCPropertyMap)
	for epcStr, propData := range oldFormatMap {
		var epc EPCType
		// EPCType.UnmarshalJSONは10進数文字列も扱える
		if err := json.Unmarshal([]byte(`"`+epcStr+`"`), &epc); err != nil {
			// 旧フォーマットだがキーが不正な場合はエラーとする
			return fmt.Errorf("invalid EPC key format in old format %q: %w", epcStr, err)
		}

		// 旧フォーマットのEDTをデコード
		edt, err := base64.StdEncoding.DecodeString(propData.EDT)
		if err != nil {
			return fmt.Errorf("invalid base64 EDT in old format for EPC %s: %w", epcStr, err)
		}

		// EPCはキーから取得したものを優先する
		result[epc] = Property{EPC: epc, EDT: edt}
	}
	*m = result
	return nil
}

type DeviceProperties map[EOJ]EPCPropertyMap

// MarshalJSON は DeviceProperties を JSON にエンコードする際に、EOJ キーを文字列形式に変換します
func (d DeviceProperties) MarshalJSON() ([]byte, error) {
	// 文字列キーを使用した一時的なマップを作成
	stringMap := make(map[string]EPCPropertyMap)
	for eoj, props := range d {
		// EOJ.Specifier() を使用して文字列キーを生成
		stringMap[eoj.Specifier()] = props
	}
	// 標準のJSONエンコーダを使用して一時マップをエンコード
	return json.Marshal(stringMap)
}

// UnmarshalJSON は JSON から DeviceProperties をデコードする際に、文字列キーを EOJ に変換します
func (d *DeviceProperties) UnmarshalJSON(data []byte) error {
	// 文字列キーを使用した一時的なマップを作成
	stringMap := make(map[string]EPCPropertyMap)
	if err := json.Unmarshal(data, &stringMap); err != nil {
		return err
	}

	// 新しいマップを作成
	result := make(DeviceProperties)
	for eojStr, props := range stringMap {
		// 文字列キーを EOJ に変換
		var eoj EOJ
		var err error

		// インスタンスコードが含まれているかどうかを確認
		if strings.Contains(eojStr, ":") {
			// "CCCC:I" 形式の場合は ParseEOJString を使用
			eoj, err = ParseEOJString(eojStr)
		} else {
			// "CCCC" 形式の場合は ParseEOJClassCodeString を使用してクラスコードのみを解析
			classCode, err := ParseEOJClassCodeString(eojStr)
			if err != nil {
				return fmt.Errorf("invalid EOJ class code: %v", err)
			}
			// インスタンスコード 0 で EOJ を作成
			eoj = echonet_lite.MakeEOJ(classCode, 0)
		}

		if err != nil {
			return fmt.Errorf("invalid EOJ string: %v", err)
		}

		result[eoj] = props
	}

	*d = result
	return nil
}

// DevicesFileFormat は devices.json ファイルの新しいフォーマットを表します。
type DevicesFileFormat struct {
	Version int                         `json:"version"`
	Data    map[string]DeviceProperties `json:"data"`
}

// currentDevicesFileVersion は現在の devices.json のフォーマットバージョンです。
const currentDevicesFileVersion = 1

// DeviceEventType はデバイスイベントの種類を表す型
type DeviceEventType int

const (
	DeviceEventAdded   DeviceEventType = iota // デバイスが追加された
	DeviceEventRemoved                        // デバイスが削除された
	DeviceEventOffline                        // デバイスがオフラインになった
	DeviceEventOnline                         // デバイスがオンラインに復旧した
)

// DeviceEvent はデバイスに関するイベントを表す構造体
type DeviceEvent struct {
	Device IPAndEOJ        // イベントが発生したデバイス
	Type   DeviceEventType // イベントの種類
}

type DevicesImpl struct {
	mu             sync.RWMutex
	data           map[string]DeviceProperties // key is IP address string
	timestamps     map[string]time.Time        // key is "IP EOJ" format string (IPAndEOJ.Key())
	EventCh        chan DeviceEvent            // デバイスイベント通知用チャンネル
	offlineDevices map[string]struct{}         // オフライン状態のデバイス (key: IPAndEOJ.Key())
	saveMu         sync.Mutex                  // ファイル保存操作の排他制御用
}

type Devices struct {
	*DevicesImpl
}

func NewDevices() Devices {
	return Devices{
		DevicesImpl: &DevicesImpl{
			data:           make(map[string]DeviceProperties),
			timestamps:     make(map[string]time.Time),
			EventCh:        nil, // 初期値はnil、後で設定する
			offlineDevices: make(map[string]struct{}),
		},
	}
}

// SetEventChannel はイベント通知用チャンネルを設定する
func (d *Devices) SetEventChannel(ch chan DeviceEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.EventCh = ch
}

func (d Devices) HasIP(ip net.IP) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.data[ip.String()]
	return ok
}

func (d Devices) IsKnownDevice(device IPAndEOJ) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ipStr := device.IP.String()
	if _, ok := d.data[ipStr]; !ok {
		return false
	}
	if _, ok := d.data[ipStr][device.EOJ]; !ok {
		return false
	}
	return true
}

// isOfflineNoLock は指定したデバイスがオフライン状態かどうかを確認します（ロックなし版）
// 呼び出し元がすでにロックを保持している場合に使用します
func (d Devices) isOfflineNoLock(key string) bool {
	_, exists := d.offlineDevices[key]
	return exists
}

// IsOffline は指定したデバイスがオフラインかどうかを返します
func (d Devices) IsOffline(device IPAndEOJ) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.isOfflineNoLock(device.Key())
}

// SetOffline は指定したデバイスのオフライン状態を設定します
func (d Devices) SetOffline(device IPAndEOJ, offline bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := device.Key()
	if offline {
		// すでにオフラインの場合は何もしない（重複チェック）
		if _, alreadyOffline := d.offlineDevices[key]; alreadyOffline {
			return
		}
		d.offlineDevices[key] = struct{}{}
		slog.Info("デバイスをオフライン状態に設定", "device", device.Specifier())

		// イベントをチャンネルに送信
		if d.EventCh != nil {
			select {
			case d.EventCh <- DeviceEvent{
				Device: device,
				Type:   DeviceEventOffline,
			}:
				// 送信成功
				slog.Info("デバイスオフラインイベントを送信", "device", device.Specifier())
			default:
				// チャンネルがブロックされている場合は無視
				slog.Warn("デバイスオフラインイベントチャンネルがブロックされています", "device", device.Specifier())
			}
		} else {
			slog.Warn("デバイスオフラインイベントチャンネルが設定されていません", "device", device.Specifier())
		}
	} else {
		// すでにオンラインの場合は何もしない（重複チェック）
		if !d.isOfflineNoLock(key) {
			return
		}
		delete(d.offlineDevices, key)

		// オフライン状態からオンラインに変わった
		slog.Info("デバイスをオンライン状態に設定", "device", device.Specifier())

		// イベントをチャンネルに送信
		if d.EventCh != nil {
			select {
			case d.EventCh <- DeviceEvent{
				Device: device,
				Type:   DeviceEventOnline,
			}:
				// 送信成功
				slog.Info("デバイスオンラインイベントを送信", "device", device.Specifier())
			default:
				// チャンネルがブロックされている場合は無視
				slog.Warn("デバイスオンラインイベントチャンネルがブロックされています", "device", device.Specifier())
			}
		} else {
			slog.Warn("デバイスオンラインイベントチャンネルが設定されていません", "device", device.Specifier())
		}
	}
}

// SetOfflineByIP sets the offline state of all devices with the given IP address
func (d Devices) SetOfflineByIP(ip net.IP, offline bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ipStr := ip.String()
	eojMap, exists := d.data[ipStr]
	if !exists {
		// No devices with this IP address
		return
	}

	// Set offline state for all devices with this IP
	for eoj := range eojMap {
		device := IPAndEOJ{IP: ip, EOJ: eoj}
		key := device.Key()

		if offline {
			// Check if already offline
			if _, alreadyOffline := d.offlineDevices[key]; !alreadyOffline {
				d.offlineDevices[key] = struct{}{}
				slog.Info("デバイスをオフライン状態に設定 (IP一括)", "device", device.Specifier())

				// Send offline event
				if d.EventCh != nil {
					select {
					case d.EventCh <- DeviceEvent{
						Device: device,
						Type:   DeviceEventOffline,
					}:
						slog.Info("デバイスオフラインイベントを送信 (IP一括)", "device", device.Specifier())
					default:
						slog.Warn("デバイスオフラインイベントチャンネルがブロックされています (IP一括)", "device", device.Specifier())
					}
				}
			}
		} else {
			// Check if was offline
			wasOffline := d.isOfflineNoLock(key)
			delete(d.offlineDevices, key)

			if wasOffline {
				slog.Info("デバイスをオンライン状態に設定 (IP一括)", "device", device.Specifier())

				// Send online event
				if d.EventCh != nil {
					select {
					case d.EventCh <- DeviceEvent{
						Device: device,
						Type:   DeviceEventOnline,
					}:
						slog.Info("デバイスオンラインイベントを送信 (IP一括)", "device", device.Specifier())
					default:
						slog.Warn("デバイスオンラインイベントチャンネルがブロックされています (IP一括)", "device", device.Specifier())
					}
				}
			}
		}
	}
}

// ensureDeviceExists ensures the map structure exists for the given IP and EOJ
// Caller must hold the lock
func (d *Devices) ensureDeviceExists(device IPAndEOJ) {
	ipStr := device.IP.String()
	if _, ok := d.data[ipStr]; !ok {
		d.data[ipStr] = make(map[EOJ]EPCPropertyMap)
	}
	if _, ok := d.data[ipStr][device.EOJ]; !ok {
		d.data[ipStr][device.EOJ] = make(EPCPropertyMap)

		// デバイス追加イベントをチャンネルに送信
		if d.EventCh != nil {
			select {
			case d.EventCh <- DeviceEvent{
				Device: device,
				Type:   DeviceEventAdded,
			}:
				// 送信成功
			default:
				// チャンネルがブロックされている場合は無視
			}
		}
	}
}

func (d Devices) RegisterDevice(device IPAndEOJ) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureDeviceExists(device)
}

func (d Devices) RegisterProperty(device IPAndEOJ, property Property, now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureDeviceExists(device)
	d.data[device.IP.String()][device.EOJ][property.EPC] = property
	// プロパティが更新されたタイムスタンプを記録
	d.timestamps[device.Key()] = now
}

func (d Devices) RegisterProperties(device IPAndEOJ, properties Properties, now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureDeviceExists(device)
	ipStr := device.IP.String()
	props := d.data[ipStr][device.EOJ]
	for _, p := range properties {
		props[p.EPC] = p
	}
	// プロパティが更新されたタイムスタンプを記録
	d.timestamps[device.Key()] = now
}

// GetLastUpdateTime は、指定されたデバイスの最終更新タイムスタンプを取得します
// タイムスタンプが存在しない場合は time.Time のゼロ値を返します
func (d Devices) GetLastUpdateTime(device IPAndEOJ) time.Time {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ts, ok := d.timestamps[device.Key()]
	if !ok {
		return time.Time{} // ゼロ値を返す
	}
	return ts
}

// DeviceSpecifier は、デバイスを一意に識別するための情報を表す構造体
type DeviceSpecifier struct {
	IP           *net.IP          // IPアドレス。nilの場合は自動選択
	ClassCode    *EOJClassCode    // クラスコード
	InstanceCode *EOJInstanceCode // インスタンスコード
}

func (d DeviceSpecifier) String() string {
	var results []string

	if d.IP != nil {
		results = append(results, d.IP.String())
	}
	if d.ClassCode != nil {
		if d.InstanceCode != nil {
			results = append(results, fmt.Sprintf("%v:%v", *d.ClassCode, *d.InstanceCode))
		} else {
			results = append(results, fmt.Sprintf("%v", *d.ClassCode))
		}
	}
	return strings.Join(results, ", ")
}

// FilterCriteria defines filtering criteria for devices and their properties.
// Device and PropertyValues are used to filter devices.
type FilterCriteria struct {
	Device         DeviceSpecifier // Filters devices by IP address, ClassCode, and InstanceCode
	PropertyValues []Property      // Filters devices by property values (EPC and EDT)
	ExcludeOffline bool            // Excludes offline devices from the result
}

func (c FilterCriteria) String() string {
	var results []string
	results = append(results, c.Device.String())
	if len(c.PropertyValues) > 0 {
		results = append(results, fmt.Sprintf("PropertyValues:%v", c.PropertyValues))
	}
	return strings.Join(results, " ")
}

// Filter returns a new Devices filtered by the given criteria.
// The filtering process works as follows:
// 1. Devices are filtered by Device and PropertyValues
// 2. All properties of matched devices are included in the result
func (d Devices) Filter(criteria FilterCriteria) Devices {
	filtered := NewDevices()
	deviceSpec := criteria.Device

	d.mu.RLock()
	defer d.mu.RUnlock()

	for ip, eojMap := range d.data {
		// IPアドレスフィルタがある場合、マッチしないものはスキップ
		if deviceSpec.IP != nil && ip != deviceSpec.IP.String() {
			continue
		}

		for eoj, props := range eojMap {
			// クラスコードフィルタがある場合、マッチしないものはスキップ
			if deviceSpec.ClassCode != nil && eoj.ClassCode() != *deviceSpec.ClassCode {
				continue
			}

			// インスタンスコードフィルタがある場合、マッチしないものはスキップ
			if deviceSpec.InstanceCode != nil && eoj.InstanceCode() != *deviceSpec.InstanceCode {
				continue
			}

			// オフラインデバイスを除外する場合
			if criteria.ExcludeOffline {
				ipAddr := net.ParseIP(ip)
				if ipAddr != nil {
					device := IPAndEOJ{IP: ipAddr, EOJ: eoj}
					if d.IsOffline(device) {
						continue
					}
				}
			}

			// PropertyValueフィルタがある場合
			if len(criteria.PropertyValues) > 0 {
				propValueMatched := false
				// 指定されたPropertyValue(EPC, EDT)のいずれかにマッチするプロパティを探す
				for _, propValue := range criteria.PropertyValues {
					if prop, ok := props[propValue.EPC]; ok {
						// Check if EDT matches using bytes.Equal
						if bytes.Equal(prop.EDT, propValue.EDT) {
							propValueMatched = true
						}
					}
				}
				// マッチしなかった場合はスキップ
				if !propValueMatched {
					continue
				}
			}

			// 全てのプロパティを結果に含める
			if _, ok := filtered.data[ip]; !ok {
				filtered.data[ip] = make(map[EOJ]EPCPropertyMap)
			}
			filtered.data[ip][eoj] = props
		}
	}

	return filtered
}

// DevicePropertyData は、デバイス（IPAndEOJ）とそのプロパティの組を表します
type DevicePropertyData struct {
	Device     IPAndEOJ
	Properties EPCPropertyMap
}

// ListDevicePropertyData は、デバイス（IPAndEOJ）とそのプロパティの組を表します
func (d Devices) ListDevicePropertyData() []DevicePropertyData {
	startTime := time.Now()
	const lockWarnThreshold = 100 * time.Millisecond
	const totalWarnThreshold = 1 * time.Second

	d.mu.RLock()
	lockAcquiredTime := time.Now()
	lockWaitTime := lockAcquiredTime.Sub(startTime)

	// ロック待機時間が異常に長い場合のみログ出力
	if lockWaitTime > lockWarnThreshold {
		slog.Warn("ListDevicePropertyData: Lock acquisition took too long", "lockWaitTime", lockWaitTime)
	}

	defer func() {
		d.mu.RUnlock()
		totalDuration := time.Since(startTime)
		// 全体の処理時間が異常に長い場合のみログ出力
		if totalDuration > totalWarnThreshold {
			slog.Warn("ListDevicePropertyData: Operation took too long",
				"totalDuration", totalDuration,
				"lockWaitTime", lockWaitTime,
				"deviceCount", len(d.data))
		}
	}()

	// 結果が空の場合は早期リターン
	if len(d.data) == 0 {
		return nil
	}

	// デバイスのリストを取得
	ipAndEOJs := d.ListIPAndEOJ()

	// IPアドレスとEOJでソート
	sort.Slice(ipAndEOJs, func(i, j int) bool {
		// IPアドレスでソート
		if !ipAndEOJs[i].IP.Equal(ipAndEOJs[j].IP) {
			// IPアドレスをバイト値として比較
			return bytes.Compare(ipAndEOJs[i].IP, ipAndEOJs[j].IP) < 0
		}
		// EOJでソート
		return ipAndEOJs[i].EOJ < ipAndEOJs[j].EOJ
	})

	// 結果の構築
	var result []DevicePropertyData

	for _, ipAndEOJ := range ipAndEOJs {
		eoj := ipAndEOJ.EOJ
		ipStr := ipAndEOJ.IP.String()
		allProps := d.data[ipStr][eoj]

		// 表示モードに応じてフィルタリングされたプロパティを保持するマップ
		result = append(result, DevicePropertyData{
			Device:     ipAndEOJ,
			Properties: allProps,
		})
	}

	return result
}

// SaveToFile saves the Devices data to a file in the new JSON format with versioning.
func (d Devices) SaveToFile(filename string) error {
	// ファイル保存操作の排他制御
	d.saveMu.Lock()
	defer d.saveMu.Unlock()

	d.mu.RLock()
	defer d.mu.RUnlock()

	fileData := DevicesFileFormat{
		Version: currentDevicesFileVersion,
		Data:    d.data,
	}

	jsonData, err := json.Marshal(fileData)
	if err != nil {
		return fmt.Errorf("failed to marshal devices data: %w", err)
	}

	// 一時ファイルに書き込み
	tempFilename := filename + ".tmp"
	err = os.WriteFile(tempFilename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to temporary file %s: %w", tempFilename, err)
	}

	// 元のファイルをリネームしてバックアップ（オプション）
	// backupFilename := filename + ".bak"
	// os.Rename(filename, backupFilename) // エラーは無視してもよい

	// 一時ファイルをリネームして本ファイルとする
	err = os.Rename(tempFilename, filename)
	if err != nil {
		// リネーム失敗時は一時ファイルを削除
		_ = os.Remove(tempFilename)
		return fmt.Errorf("failed to rename temporary file %s to %s: %w", tempFilename, filename, err)
	}

	return nil
}

// LoadFromFile loads the Devices data from a file, supporting both new and old JSON formats.
func (d Devices) LoadFromFile(filename string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ファイルが存在しない場合はエラーとしない
		}
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Warn("Error closing file", "filename", filename, "err", err)
		}
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// まずバージョン情報を含むかチェックするために一時的なマップにデコード
	var versionCheck map[string]any
	if err := json.Unmarshal(data, &versionCheck); err != nil {
		// JSONとしてパースできない場合はエラー
		return fmt.Errorf("failed to parse file %s as JSON: %w", filename, err)
	}

	if versionVal, ok := versionCheck["version"]; ok {
		// "version" キーが存在する場合
		if versionFloat, ok := versionVal.(float64); ok && int(versionFloat) == currentDevicesFileVersion {
			// バージョンが一致する場合、新しいフォーマットとしてデコード
			var fileData DevicesFileFormat
			if err := json.Unmarshal(data, &fileData); err != nil {
				return fmt.Errorf("failed to unmarshal file %s with version %d: %w", filename, currentDevicesFileVersion, err)
			}
			d.data = fileData.Data
			// TODO: タイムスタンプの復元ロジックが必要な場合はここに追加
			d.timestamps = make(map[string]time.Time) // タイムスタンプは一旦リセット
			return nil
		}
		// バージョンが不一致の場合はエラーまたはフォールバック処理
		// ここでは古いバージョンとして扱うことにする（下の処理に流れる）
		fmt.Printf("Warning: File %s has version %v, expected %d. Attempting to load as old format.\n", filename, versionVal, currentDevicesFileVersion)
	}

	// "version" キーが存在しない、またはバージョンが不一致の場合、古いフォーマットとしてデコード
	var oldData map[string]DeviceProperties
	if err := json.Unmarshal(data, &oldData); err != nil {
		return fmt.Errorf("failed to unmarshal file %s as old format: %w", filename, err)
	}
	d.data = oldData
	// TODO: タイムスタンプの復元ロジックが必要な場合はここに追加
	d.timestamps = make(map[string]time.Time) // タイムスタンプは一旦リセット

	return nil
}

func (h Devices) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.data)
}

// CountAll は全てのデバイス（IPとEOJの組み合わせ）の総数を返す
func (h Devices) CountAll() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, eojMap := range h.data {
		count += len(eojMap)
	}
	return count
}

func (h Devices) ListIPAndEOJ() []IPAndEOJ {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var devices []IPAndEOJ
	for ipStr, eojMap := range h.data {
		ip := net.ParseIP(ipStr)
		for eoj := range eojMap {
			devices = append(devices, IPAndEOJ{IP: ip, EOJ: eoj})
		}
	}
	return devices
}

// GetProperty returns the property for the given IP, EOJ, and EPC
// If the property does not exist, returns nil and false
func (d Devices) GetProperty(device IPAndEOJ, epc EPCType) (*Property, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ipStr := device.IP.String()
	// Check if the device exists
	if deviceMap, ok := d.data[ipStr]; ok {
		if properties, ok := deviceMap[device.EOJ]; ok {
			// Check if the property exists
			if prop, ok := properties[epc]; ok {
				return &prop, true
			}
		}
	}
	return nil, false
}

// PropertyMapType はプロパティマップの種類を表す型
type PropertyMapType int

const (
	GetPropertyMap PropertyMapType = iota
	SetPropertyMap
	StatusAnnouncementPropertyMap
)

// GetPropertyMap は指定されたプロパティマップを取得する
func (d Devices) GetPropertyMap(device IPAndEOJ, mapType PropertyMapType) PropertyMap {
	var mapEPC EPCType

	switch mapType {
	case GetPropertyMap:
		mapEPC = echonet_lite.EPCGetPropertyMap
	case SetPropertyMap:
		mapEPC = echonet_lite.EPCSetPropertyMap
	case StatusAnnouncementPropertyMap:
		mapEPC = echonet_lite.EPCStatusAnnouncementPropertyMap
	default:
		return nil
	}

	prop, ok := d.GetProperty(device, mapEPC)
	if !ok {
		return nil
	}

	propMap := echonet_lite.DecodePropertyMap(prop.EDT)
	if propMap == nil {
		return nil
	}

	return propMap
}

// HasEPCInPropertyMap は指定されたプロパティマップに EPC が含まれているかどうかを確認する
func (d Devices) HasEPCInPropertyMap(device IPAndEOJ, mapType PropertyMapType, epc EPCType) bool {
	propMap := d.GetPropertyMap(device, mapType)
	if propMap == nil {
		return false
	}
	return propMap.Has(epc)
}

func (d Devices) GetIDString(device IPAndEOJ) IDString {
	// 識別番号はNodeProfileObject から取得する
	npo := IPAndEOJ{
		IP:  device.IP,
		EOJ: echonet_lite.NodeProfileObject,
	}
	prop, ok := d.GetProperty(npo, echonet_lite.EPC_NPO_IDNumber)
	if !ok {
		return ""
	}
	id := echonet_lite.DecodeIdentificationNumber(prop.EDT)
	if id == nil {
		return ""
	}
	// それにEOJを結合してIDStringを作成
	return MakeIDString(device.EOJ, *id)
}

func (d Devices) FindByIDString(id IDString) []IPAndEOJ {
	if id == "" {
		return nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []IPAndEOJ

	for ipStr, eojMap := range d.data {
		ip := net.ParseIP(ipStr)
		for eoj := range eojMap {
			ipAndEOJ := IPAndEOJ{IP: ip, EOJ: eoj}
			idStr := d.GetIDString(ipAndEOJ)
			if idStr == id {
				result = append(result, ipAndEOJ)
			}
		}
	}
	return result
}

func (d DeviceProperties) Set(eoj EOJ, properties ...Property) error {
	if eoj.InstanceCode() == 0 {
		// インスタンスコードが0の場合は設定できない
		return fmt.Errorf("インスタンスコードが0のEOJにはプロパティを設定できません")
	}
	if _, ok := d[eoj]; !ok {
		d[eoj] = make(EPCPropertyMap)
	}
	for _, prop := range properties {
		d[eoj][prop.EPC] = prop
	}
	return nil
}

func (d DeviceProperties) Get(eoj EOJ, epc EPCType) (Property, bool) {
	if epc == echonet_lite.EPCGetPropertyMap {
		// GetPropertyMap は特別なプロパティで、全てのプロパティを含むプロパティマップを返す
		propertyMap := make(PropertyMap)
		for epc := range d[eoj] {
			propertyMap.Set(epc)
		}
		// GetPropertyMapは必ず存在する
		propertyMap.Set(echonet_lite.EPCGetPropertyMap)
		// SetPropertyMapも取得可能なプロパティなので、GetPropertyMapに含める
		propertyMap.Set(echonet_lite.EPCSetPropertyMap)
		propertyMap.Set(echonet_lite.EPCStatusAnnouncementPropertyMap)

		return Property{
			EPC: epc,
			EDT: propertyMap.Encode(),
		}, true
	}

	if epc == echonet_lite.EPCSetPropertyMap {
		// SetPropertyMap は特別なプロパティで、設定可能なプロパティのみを含むプロパティマップを返す
		propertyMap := make(PropertyMap)

		// 設定不可能なプロパティを定義
		readOnlyEPCs := map[EPCType]bool{
			echonet_lite.EPCOperationStatus:                       true, // 動作状態（読み取り専用）
			echonet_lite.EPCGetPropertyMap:                        true, // GetPropertyMap（システム生成）
			echonet_lite.EPCSetPropertyMap:                        true, // SetPropertyMap（システム生成）
			echonet_lite.EPCStatusAnnouncementPropertyMap:         true, // 状態通知プロパティマップ（システム生成）
			echonet_lite.EPCStandardVersion:                       true, // 規格Version情報（読み取り専用）
			echonet_lite.EPCIdentificationNumber:                  true, // 識別番号（読み取り専用）
			echonet_lite.EPCMeasuredInstantaneousPowerConsumption: true, // 瞬時消費電力計測値（読み取り専用）
			echonet_lite.EPCMeasuredCumulativePowerConsumption:    true, // 積算消費電力量計測値（読み取り専用）
			echonet_lite.EPCManufacturerCode:                      true, // メーカコード（読み取り専用）
			echonet_lite.EPCBusinessFacilityCode:                  true, // 事業場コード（読み取り専用）
			echonet_lite.EPCProductCode:                           true, // 商品コード（読み取り専用）
			echonet_lite.EPCProductionNumber:                      true, // 製造番号（読み取り専用）
			echonet_lite.EPCProductionDate:                        true, // 製造年月日（読み取り専用）
		}

		// 設定可能なプロパティのみをマップに追加
		for propEPC := range d[eoj] {
			if !readOnlyEPCs[propEPC] {
				propertyMap.Set(propEPC)
			}
		}

		return Property{
			EPC: epc,
			EDT: propertyMap.Encode(),
		}, true
	}

	if _, ok := d[eoj]; !ok {
		return Property{}, false
	}
	prop, ok := d[eoj][epc]
	return prop, ok
}

// GetProperties は指定されたプロパティの値を取得する。第2返り値はすべての指定されたプロパティが取得できたときにtrue。
// プロパティが存在しない場合は、そのプロパティは EDT が空の状態で返される
func (d DeviceProperties) GetProperties(eoj EOJ, properties Properties) (Properties, bool) {
	result := make([]Property, 0, len(properties))
	success := true

	for _, p := range properties {
		rep := Property{
			EPC: p.EPC,
			EDT: []byte{}, // empty
		}
		prop, ok := d.Get(eoj, p.EPC)
		if !ok {
			success = false
		} else {
			rep.EDT = prop.EDT
		}
		result = append(result, rep)
	}
	return result, success
}

// SetProperties は指定されたプロパティの値を設定する。第2返り値はすべての指定されたプロパティが設定できたときにtrue。
// プロパティが書き込めない場合は、そのプロパティはリクエストの値で返され、成功した場合はEDTが空になる
func (d DeviceProperties) SetProperties(eoj EOJ, properties Properties) (Properties, bool) {
	setPropertyMap := PropertyMap{}
	if p, ok := d.Get(eoj, echonet_lite.EPCSetPropertyMap); ok {
		setPropertyMap = echonet_lite.DecodePropertyMap(p.EDT)
	}

	result := make([]Property, 0, len(properties))
	success := true

	for _, p := range properties {
		rep := p
		if !setPropertyMap.Has(p.EPC) {
			success = false
		} else {
			_ = d.Set(eoj, p)
			rep.EDT = []byte{} // 書き込み成功したら empty
		}
		result = append(result, rep)
	}
	return result, success
}

func (d DeviceProperties) GetInstanceList() []EOJ {
	EOJs := []EOJ{}
	for eoj := range d {
		if eoj.ClassCode() == echonet_lite.NodeProfile_ClassCode {
			continue
		}
		EOJs = append(EOJs, eoj)
	}
	return EOJs
}

func (d DeviceProperties) UpdateProfileObjectProperties() error {
	instanceList := d.GetInstanceList()
	selfNodeInstancesProp, err := echonet_lite.PropertyFromInt(
		echonet_lite.NodeProfile_ClassCode,
		echonet_lite.EPC_NPO_SelfNodeInstances,
		len(instanceList),
	)
	if err != nil {
		return err
	}
	selfNodeInstanceListS := echonet_lite.SelfNodeInstanceListS(instanceList)

	classes := make(map[EOJClassCode]struct{})
	for _, e := range instanceList {
		classes[e.ClassCode()] = struct{}{}
	}
	selfNodeClassesProp, err := echonet_lite.PropertyFromInt(
		echonet_lite.NodeProfile_ClassCode,
		echonet_lite.EPC_NPO_SelfNodeClasses,
		len(classes),
	)
	if err != nil {
		return err
	}
	classArray := make([]EOJClassCode, 0, len(classes))
	for c := range classes {
		classArray = append(classArray, c)
	}
	selfNodeClassListS := echonet_lite.SelfNodeClassListS(classArray)

	eoj := echonet_lite.NodeProfileObject
	return d.Set(eoj,
		*selfNodeInstancesProp,
		*selfNodeInstanceListS.Property(),
		*selfNodeClassesProp,
		*selfNodeClassListS.Property(),
	)
}

func (d DeviceProperties) FindEOJ(deoj EOJ) []EOJ {
	// d に　deoj が含まれるなら true
	if _, ok := d[deoj]; ok {
		return []EOJ{deoj}
	}
	// deoj の instanceCode が 0 の場合、classCode が一致する EOJ を探す
	if deoj.InstanceCode() == 0 {
		result := []EOJ{}
		for eoj := range d {
			if eoj.ClassCode() == deoj.ClassCode() {
				result = append(result, eoj)
			}
		}
		return result
	}
	return nil
}

// IsAnnouncementTarget は指定されたEOJとEPCがStatus Announcement Property Mapに含まれているかどうかを判定する
func (d DeviceProperties) IsAnnouncementTarget(eoj EOJ, epc EPCType) bool {
	// Status Announcement Property Mapを取得
	announcementProp, ok := d.Get(eoj, echonet_lite.EPCStatusAnnouncementPropertyMap)
	if !ok {
		return false
	}

	// PropertyMapをデコード
	propMap := echonet_lite.DecodePropertyMap(announcementProp.EDT)
	if propMap == nil {
		return false
	}

	// 指定されたEPCが含まれているかチェック
	return propMap.Has(epc)
}

// FindIPsWithSameNodeProfileID は同じ識別番号(0x83 EDT)を持つ他のIPを検索する
func (d Devices) FindIPsWithSameNodeProfileID(idEDT []byte, excludeIP string) []string {
	if len(idEDT) == 0 {
		return nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []string
	for ipStr, eojMap := range d.data {
		if ipStr == excludeIP {
			continue
		}
		// NodeProfile の 0x83 EDT を比較
		if props, ok := eojMap[echonet_lite.NodeProfileObject]; ok {
			if prop, ok := props[echonet_lite.EPC_NPO_IDNumber]; ok {
				if bytes.Equal(prop.EDT, idEDT) {
					result = append(result, ipStr)
				}
			}
		}
	}
	return result
}

// RemoveAllDevicesByIP は指定IPの全デバイスを削除し、削除したデバイスのリストを返す
func (d *DevicesImpl) RemoveAllDevicesByIP(ip net.IP) []IPAndEOJ {
	d.mu.Lock()
	defer d.mu.Unlock()

	ipKey := ip.String()
	eojMap, exists := d.data[ipKey]
	if !exists {
		return nil
	}

	var removed []IPAndEOJ
	for eoj := range eojMap {
		device := IPAndEOJ{IP: ip, EOJ: eoj}
		deviceKey := device.Key()

		// タイムスタンプを削除
		delete(d.timestamps, deviceKey)

		// オフライン状態を削除
		delete(d.offlineDevices, deviceKey)

		removed = append(removed, device)

		// DeviceEventRemoved イベント送信
		if d.EventCh != nil {
			select {
			case d.EventCh <- DeviceEvent{
				Device: device,
				Type:   DeviceEventRemoved,
			}:
			default:
				// チャンネルがフルの場合はイベントをドロップ
			}
		}
	}

	// IPエントリ自体を削除
	delete(d.data, ipKey)

	return removed
}

// RemoveDevice は指定されたデバイスをDevicesから削除する
func (d *DevicesImpl) RemoveDevice(device IPAndEOJ) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	ipKey := device.IP.String()
	deviceKey := device.Key()

	// デバイスが実際に存在することを確認
	var deviceRemoved bool
	if deviceProps, exists := d.data[ipKey]; exists {
		if _, eojExists := deviceProps[device.EOJ]; eojExists {
			delete(deviceProps, device.EOJ)
			deviceRemoved = true

			// このIPに他のEOJが残っていない場合、IP情報自体を削除
			if len(deviceProps) == 0 {
				delete(d.data, ipKey)
			}
		}
	}

	// タイムスタンプを削除
	delete(d.timestamps, deviceKey)

	// オフライン状態を削除
	delete(d.offlineDevices, deviceKey)

	// デバイスが実際に削除された場合、イベントを発行
	if deviceRemoved && d.EventCh != nil {
		select {
		case d.EventCh <- DeviceEvent{
			Device: device,
			Type:   DeviceEventRemoved,
		}:
		default:
			// チャンネルがフルの場合はイベントをドロップ（ノンブロッキング）
		}
	}

	return nil
}
