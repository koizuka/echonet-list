package echonet_lite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
)

type EPCPropertyMap map[EPCType]Property
type DeviceProperties map[EOJ]EPCPropertyMap

type DevicesImpl struct {
	mu   sync.RWMutex
	data map[string]DeviceProperties // key is IP address string
}

type Devices struct {
	*DevicesImpl
}

func NewDevices() Devices {
	return Devices{
		DevicesImpl: &DevicesImpl{
			data: make(map[string]DeviceProperties),
		},
	}
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

// ensureDeviceExists ensures the map structure exists for the given IP and EOJ
// Caller must hold the lock
func (d *Devices) ensureDeviceExists(device IPAndEOJ) {
	ipStr := device.IP.String()
	if _, ok := d.data[ipStr]; !ok {
		d.data[ipStr] = make(map[EOJ]EPCPropertyMap)
	}
	if _, ok := d.data[ipStr][device.EOJ]; !ok {
		d.data[ipStr][device.EOJ] = make(EPCPropertyMap)
	}
}

func (d Devices) RegisterDevice(device IPAndEOJ) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureDeviceExists(device)
}

func (d Devices) RegisterProperty(device IPAndEOJ, property Property) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureDeviceExists(device)
	d.data[device.IP.String()][device.EOJ][property.EPC] = property
}

func (d Devices) RegisterProperties(device IPAndEOJ, properties Properties) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureDeviceExists(device)
	ipStr := device.IP.String()
	for _, p := range properties {
		d.data[ipStr][device.EOJ][p.EPC] = p
	}
}

// DeviceSpecifier は、デバイスを一意に識別するための情報を表す構造体
type DeviceSpecifier struct {
	IP           *net.IP          // IPアドレス。nilの場合は自動選択
	ClassCode    *EOJClassCode    // クラスコード
	InstanceCode *EOJInstanceCode // インスタンスコード
}

func (d *DeviceSpecifier) String() string {
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
	Device         *DeviceSpecifier // Filters devices by IP address, ClassCode, and InstanceCode
	PropertyValues []Property       // Filters devices by property values (EPC and EDT)
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
	// ショートカット：フィルタ条件が無い場合は自身を返す
	if (criteria.Device == nil ||
		(criteria.Device.IP == nil && criteria.Device.ClassCode == nil && criteria.Device.InstanceCode == nil)) &&
		len(criteria.PropertyValues) == 0 {
		return d
	}
	var deviceSpec DeviceSpecifier
	if criteria.Device != nil {
		deviceSpec = *criteria.Device
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	filtered := NewDevices()

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
	d.mu.RLock()
	defer d.mu.RUnlock()

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

func (m EPCPropertyMap) SortedProperties() Properties {
	// プロパティのEPCをソート
	epcsToShow := make([]EPCType, 0, len(m))
	for epc := range m {
		epcsToShow = append(epcsToShow, epc)
	}
	sort.Slice(epcsToShow, func(i, j int) bool {
		return epcsToShow[i] < epcsToShow[j]
	})

	properties := make(Properties, 0, len(m))
	for _, epc := range epcsToShow {
		properties = append(properties, m[epc])
	}
	return properties
}

// SaveToFile saves the Devices data to a file in JSON format.
func (d Devices) SaveToFile(filename string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("ファイルを閉じる際にエラーが発生しました: %v\n", err)
		}
	}()

	encoder := json.NewEncoder(file)
	return encoder.Encode(d.data)
}

// LoadFromFile loads the Devices data from a file in JSON format.
func (d Devices) LoadFromFile(filename string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("ファイルを閉じる際にエラーが発生しました: %v\n", err)
		}
	}()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&d.data)
}

func (h Devices) Len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.data)
}

func (h Devices) ListIPAndEOJ() []IPAndEOJ {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var devices []IPAndEOJ
	for ipStr, eojMap := range h.data {
		for eoj := range eojMap {
			devices = append(devices, IPAndEOJ{IP: net.ParseIP(ipStr), EOJ: eoj})
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
		mapEPC = EPCGetPropertyMap
	case SetPropertyMap:
		mapEPC = EPCSetPropertyMap
	case StatusAnnouncementPropertyMap:
		mapEPC = EPCStatusAnnouncementPropertyMap
	default:
		return nil
	}

	prop, ok := d.GetProperty(device, mapEPC)
	if !ok {
		return nil
	}

	propMap := DecodePropertyMap(prop.EDT)
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

func (d DeviceProperties) Set(eoj EOJ, properties ...IProperty) error {
	if eoj.InstanceCode() == 0 {
		// インスタンスコードが0の場合は設定できない
		return fmt.Errorf("インスタンスコードが0のEOJにはプロパティを設定できません")
	}
	if _, ok := d[eoj]; !ok {
		d[eoj] = make(EPCPropertyMap)
	}
	for _, prop := range properties {
		p := prop.Property()
		if p == nil {
			continue
		}
		d[eoj][p.EPC] = *p
	}
	return nil
}

func (d DeviceProperties) Get(eoj EOJ, epc EPCType) (Property, bool) {
	if epc == EPCGetPropertyMap {
		// GetPropertyMap は特別なプロパティで、全てのプロパティを含むプロパティマップを返す
		propertyMap := make(PropertyMap)
		for epc := range d[eoj] {
			propertyMap.Set(epc)
		}
		// GetPropertyMapは必ず存在する
		propertyMap.Set(EPCGetPropertyMap)

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
	if p, ok := d.Get(eoj, EPCSetPropertyMap); ok {
		setPropertyMap = DecodePropertyMap(p.EDT)
	}

	result := make([]Property, 0, len(properties))
	success := true

	for _, p := range properties {
		rep := Property{
			EPC: p.EPC,
			EDT: p.EDT, // 書き込み失敗したらリクエストの値
		}
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
		if eoj.ClassCode() == NodeProfile_ClassCode {
			continue
		}
		EOJs = append(EOJs, eoj)
	}
	return EOJs
}

func (d DeviceProperties) UpdateProfileObjectProperties() error {
	instanceList := d.GetInstanceList()
	selfNodeInstances := SelfNodeInstances(len(instanceList))
	selfNodeInstanceListS := SelfNodeInstanceListS(instanceList)

	classes := make(map[EOJClassCode]struct{})
	for _, e := range instanceList {
		classes[e.ClassCode()] = struct{}{}
	}
	selfNodeClasses := SelfNodeClasses(len(classes))
	classArray := make([]EOJClassCode, 0, len(classes))
	for c := range classes {
		classArray = append(classArray, c)
	}
	selfNodeClassListS := SelfNodeClassListS(classArray)

	eoj := NodeProfileObject
	return d.Set(eoj,
		&selfNodeInstances,
		&selfNodeInstanceListS,
		&selfNodeClasses,
		&selfNodeClassListS,
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
