package main

import (
	"bytes"
	"echonet-list/echonet_lite"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
)

type DeviceProperties map[echonet_lite.EOJ]map[echonet_lite.EPCType]echonet_lite.Property

type DevicesImpl struct {
	mu   sync.RWMutex
	data map[string]DeviceProperties
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

func (d Devices) HasIP(IP string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.data[IP]
	return ok
}

func (d Devices) IsKnownDevice(IP string, EOJ echonet_lite.EOJ) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if _, ok := d.data[IP]; !ok {
		return false
	}
	if _, ok := d.data[IP][EOJ]; !ok {
		return false
	}
	return true
}

// ensureDeviceExists ensures the map structure exists for the given IP and EOJ
// Caller must hold the lock
func (d *Devices) ensureDeviceExists(IP string, EOJ echonet_lite.EOJ) {
	if _, ok := d.data[IP]; !ok {
		d.data[IP] = make(map[echonet_lite.EOJ]map[echonet_lite.EPCType]echonet_lite.Property)
	}
	if _, ok := d.data[IP][EOJ]; !ok {
		d.data[IP][EOJ] = make(map[echonet_lite.EPCType]echonet_lite.Property)
	}
}

func (d Devices) RegisterDevice(IP string, EOJ echonet_lite.EOJ) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureDeviceExists(IP, EOJ)
}

func (d Devices) RegisterProperty(IP string, EOJ echonet_lite.EOJ, property echonet_lite.Property) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureDeviceExists(IP, EOJ)
	d.data[IP][EOJ][property.EPC] = property
}

func (d Devices) RegisterProperties(IP string, EOJ echonet_lite.EOJ, properties echonet_lite.Properties) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureDeviceExists(IP, EOJ)
	for _, p := range properties {
		d.data[IP][EOJ][p.EPC] = p
	}
}

// FilterCriteria defines filtering criteria for devices and their properties.
// IPAddress, ClassCode, InstanceCode, and PropertyValues are used to filter devices.
// EPCs is used to filter properties of the matched devices.
type FilterCriteria struct {
	IPAddress      *string                       // Filters devices by IP address
	ClassCode      *echonet_lite.EOJClassCode    // Filters devices by class code
	InstanceCode   *echonet_lite.EOJInstanceCode // Filters devices by instance code
	EPCs           []echonet_lite.EPCType        // Filters properties of matched devices (not devices themselves)
	PropertyValues []echonet_lite.Property       // Filters devices by property values (EPC and EDT)
}

// Filter returns a new Devices filtered by the given criteria.
// The filtering process works as follows:
// 1. Devices are filtered by IPAddress, ClassCode, InstanceCode, and PropertyValues
// 2. For matched devices, if EPCs is specified, only those properties are included in the result
func (d Devices) Filter(criteria FilterCriteria) Devices {
	// ショートカット：フィルタ条件が無い場合は自身を返す
	if criteria.IPAddress == nil && criteria.ClassCode == nil &&
		criteria.InstanceCode == nil && len(criteria.EPCs) == 0 && len(criteria.PropertyValues) == 0 {
		return d
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	filtered := NewDevices()

	for ip, eojMap := range d.data {
		// IPアドレスフィルタがある場合、マッチしないものはスキップ
		if criteria.IPAddress != nil && ip != *criteria.IPAddress {
			continue
		}

		for eoj, props := range eojMap {
			// クラスコードフィルタがある場合、マッチしないものはスキップ
			if criteria.ClassCode != nil && eoj.ClassCode() != *criteria.ClassCode {
				continue
			}

			// インスタンスコードフィルタがある場合、マッチしないものはスキップ
			if criteria.InstanceCode != nil && eoj.InstanceCode() != *criteria.InstanceCode {
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

			// EPCフィルタがある場合
			if len(criteria.EPCs) > 0 {
				matchedProps := make(map[echonet_lite.EPCType]echonet_lite.Property)

				epcMatched := false
				// 指定されたEPCのいずれかにマッチするプロパティを探す
				for _, epc := range criteria.EPCs {
					if prop, ok := props[epc]; ok {
						matchedProps[epc] = prop
						epcMatched = true
					}
				}
				// マッチしなかった場合はスキップ
				if !epcMatched {
					continue
				}

				// 初めて見つかったEOJの場合は、プロパティのマップを初期化
				filtered.ensureDeviceExists(ip, eoj)

				// マッチしたプロパティだけを結果に含める
				for epc, prop := range matchedProps {
					filtered.data[ip][eoj][epc] = prop
				}
			} else {
				// EPCフィルタがなければ、全てのプロパティを結果に含める
				if _, ok := filtered.data[ip]; !ok {
					filtered.data[ip] = make(map[echonet_lite.EOJ]map[echonet_lite.EPCType]echonet_lite.Property)
				}
				filtered.data[ip][eoj] = props
			}
		}
	}

	return filtered
}

func (d Devices) String() string {
	return d.StringWithPropertyMode(PropDefault)
}

// StringWithPropertyMode returns a string representation of devices with the specified property mode
func (d Devices) StringWithPropertyMode(propMode PropertyMode) string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// 結果が空の場合は早期リターン
	if len(d.data) == 0 {
		return ""
	}

	// IPアドレスのスライスを収集してソート
	ips := make([]string, 0, len(d.data))
	for ip := range d.data {
		ips = append(ips, ip)
	}

	// IPアドレスをソート
	sort.Slice(ips, func(i, j int) bool {
		ii, _ := net.ResolveIPAddr("ip", ips[i])
		ij, _ := net.ResolveIPAddr("ip", ips[j])
		return string(ii.IP) < string(ij.IP)
	})

	// 結果の構築
	var results []string

	for _, ip := range ips {
		eojMap := d.data[ip]

		// このIPに対するEOJをスライスに収集してソート
		eojs := make([]echonet_lite.EOJ, 0, len(eojMap))
		for eoj := range eojMap {
			eojs = append(eojs, eoj)
		}

		if len(eojs) == 0 {
			continue
		}

		// EOJをソート
		sort.Slice(eojs, func(i, j int) bool {
			return eojs[i] < eojs[j]
		})

		// 各EOJに対する出力を生成
		for _, eoj := range eojs {
			props := eojMap[eoj]
			results = append(results, fmt.Sprintf("%s, %v:", ip, eoj))

			// 表示するプロパティを選択
			epcsToShow := make([]echonet_lite.EPCType, 0, len(props))

			// 表示モードに応じてフィルタリング
			for epc := range props {
				switch propMode {
				case PropDefault:
					// デフォルトのプロパティのみ表示
					if !echonet_lite.IsPropertyDefaultEPC(eoj.ClassCode(), epc) {
						continue
					}
				case PropKnown:
					// 既知のプロパティのみ表示
					if _, ok := echonet_lite.GetPropertyInfo(eoj.ClassCode(), epc); !ok {
						continue
					}
				}
				epcsToShow = append(epcsToShow, epc)
			}

			// プロパティをソート
			sort.Slice(epcsToShow, func(i, j int) bool {
				return epcsToShow[i] < epcsToShow[j]
			})

			// プロパティの出力を生成
			for _, epc := range epcsToShow {
				p := props[epc]
				results = append(results, fmt.Sprintf("  %v", p.String(eoj.ClassCode())))
			}
		}
	}

	return strings.Join(results, "\n")
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

// DeviceInfo は、デバイスの情報を表す構造体
type DeviceInfo struct {
	IP  string
	EOJ echonet_lite.EOJ
}

// FindDevicesByClassAndInstance は、指定されたクラスコードとインスタンスコードに一致するデバイスを検索します
func (d Devices) FindDevicesByClassAndInstance(classCode *echonet_lite.EOJClassCode, instanceCode *echonet_lite.EOJInstanceCode) []DeviceInfo {
	if classCode == nil {
		return nil
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	var matchingDevices []DeviceInfo
	for ip, eojMap := range d.data {
		for eoj := range eojMap {
			if eoj.ClassCode() == *classCode {
				if instanceCode == nil || eoj.InstanceCode() == *instanceCode {
					matchingDevices = append(matchingDevices, DeviceInfo{IP: ip, EOJ: eoj})
				}
			}
		}
	}

	return matchingDevices
}

// HasPropertyWithValue checks if a property with the expected EPC and EDT exists for the given device
func (d Devices) HasPropertyWithValue(ip string, eoj echonet_lite.EOJ, epc echonet_lite.EPCType, expectedEDT []byte) bool {
	// Create a filter criteria for the specific IP, EOJ, and property value (EPC and EDT)
	ipCopy := ip
	propValue := echonet_lite.Property{
		EPC: epc,
		EDT: expectedEDT,
	}
	criteria := FilterCriteria{
		IPAddress:      &ipCopy,
		PropertyValues: []echonet_lite.Property{propValue},
	}

	// Filter the devices based on the property value criteria
	filtered := d.Filter(criteria)

	// Check if the device and EOJ exist in the filtered result
	return filtered.IsKnownDevice(ip, eoj)
}

// GetProperty returns the property for the given IP, EOJ, and EPC
// If the property does not exist, returns nil and false
func (d Devices) GetProperty(ip string, eoj echonet_lite.EOJ, epc echonet_lite.EPCType) (*echonet_lite.Property, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check if the device exists
	if deviceMap, ok := d.data[ip]; ok {
		if properties, ok := deviceMap[eoj]; ok {
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
func (d Devices) GetPropertyMap(ip string, eoj echonet_lite.EOJ, mapType PropertyMapType) echonet_lite.PropertyMap {
	var mapEPC echonet_lite.EPCType

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

	prop, ok := d.GetProperty(ip, eoj, mapEPC)
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
func (d Devices) HasEPCInPropertyMap(ip string, eoj echonet_lite.EOJ, mapType PropertyMapType, epc echonet_lite.EPCType) bool {
	propMap := d.GetPropertyMap(ip, eoj, mapType)
	if propMap == nil {
		return false
	}
	return propMap.Has(epc)
}

func (d DeviceProperties) Set(eoj echonet_lite.EOJ, property echonet_lite.Property) {
	if _, ok := d[eoj]; !ok {
		d[eoj] = make(map[echonet_lite.EPCType]echonet_lite.Property)
	}
	d[eoj][property.EPC] = property
}

func (d DeviceProperties) Get(eoj echonet_lite.EOJ, epc echonet_lite.EPCType) (echonet_lite.Property, bool) {
	if epc == echonet_lite.EPCGetPropertyMap {
		// GetPropertyMap は特別なプロパティで、全てのプロパティを含むプロパティマップを返す
		propertyMap := make(echonet_lite.PropertyMap)
		for epc := range d[eoj] {
			propertyMap.Set(epc)
		}
		// GetPropertyMapは必ず存在する
		propertyMap.Set(echonet_lite.EPCGetPropertyMap)

		return echonet_lite.Property{
			EPC: epc,
			EDT: propertyMap.Encode(),
		}, true
	}

	if _, ok := d[eoj]; !ok {
		return echonet_lite.Property{}, false
	}
	prop, ok := d[eoj][epc]
	return prop, ok
}

// GetProperties は指定されたプロパティの値を取得する。第2返り値はすべての指定されたプロパティが取得できたときにtrue。
// プロパティが存在しない場合は、そのプロパティは EDT が空の状態で返される
func (d DeviceProperties) GetProperties(eoj echonet_lite.EOJ, properties echonet_lite.Properties) (echonet_lite.Properties, bool) {
	result := make([]echonet_lite.Property, 0, len(properties))
	success := true

	for _, p := range properties {
		rep := echonet_lite.Property{
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
func (d DeviceProperties) SetProperties(eoj echonet_lite.EOJ, properties echonet_lite.Properties) (echonet_lite.Properties, bool) {
	setPropertyMap := echonet_lite.PropertyMap{}
	if p, ok := d.Get(eoj, echonet_lite.EPCSetPropertyMap); ok {
		setPropertyMap = echonet_lite.DecodePropertyMap(p.EDT)
	}

	result := make([]echonet_lite.Property, 0, len(properties))
	success := true

	for _, p := range properties {
		rep := echonet_lite.Property{
			EPC: p.EPC,
			EDT: p.EDT, // 書き込み失敗したらリクエストの値
		}
		if !setPropertyMap.Has(p.EPC) {
			success = false
		} else {
			d.Set(eoj, p)
			rep.EDT = []byte{} // 書き込み成功したら empty
		}
		result = append(result, rep)
	}
	return result, success
}

func (d DeviceProperties) GetInstanceList() []echonet_lite.EOJ {
	EOJs := []echonet_lite.EOJ{}
	for eoj := range d {
		if eoj.ClassCode() == echonet_lite.NodeProfile_ClassCode {
			continue
		}
		EOJs = append(EOJs, eoj)
	}
	return EOJs
}

func (d DeviceProperties) UpdateProfileObjectProperties() {
	instanceList := d.GetInstanceList()
	selfNodeInstances := echonet_lite.SelfNodeInstances(len(instanceList))
	selfNodeInstanceListS := echonet_lite.SelfNodeInstanceListS(instanceList)

	classes := make(map[echonet_lite.EOJClassCode]struct{})
	for _, e := range instanceList {
		classes[e.ClassCode()] = struct{}{}
	}
	selfNodeClasses := echonet_lite.SelfNodeClasses(len(classes))
	classArray := make([]echonet_lite.EOJClassCode, 0, len(classes))
	for c := range classes {
		classArray = append(classArray, c)
	}
	selfNodeClassListS := echonet_lite.SelfNodeClassListS(classArray)

	eoj := NodeProfileObject1
	d.Set(eoj, *selfNodeInstances.Property())
	d.Set(eoj, *selfNodeInstanceListS.Property())
	d.Set(eoj, *selfNodeClasses.Property())
	d.Set(eoj, *selfNodeClassListS.Property())
}
