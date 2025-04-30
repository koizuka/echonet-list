package echonet_lite

import (
	"echonet-list/echonet_lite/utils"
	"fmt"
	"sort"
)

// 機器オブジェクトスーパークラス
const (
	// EPC
	// profile super class
	EPCOperationStatus                       EPCType = 0x80 // 動作状態
	EPCInstallationLocation                  EPCType = 0x81 // 設置場所
	EPCStandardVersion                       EPCType = 0x82 // 規格Version情報
	EPCIdentificationNumber                  EPCType = 0x83 // 識別番号
	EPCMeasuredInstantaneousPowerConsumption EPCType = 0x84 // 瞬時消費電力計測値
	EPCMeasuredCumulativePowerConsumption    EPCType = 0x85 // 積算消費電力量計測値
	EPCManufacturerFaultCode                 EPCType = 0x86 // メーカ異常コード
	EPCCurrentLimitSetting                   EPCType = 0x87 // 電流制限設定値
	EPCFaultStatus                           EPCType = 0x88 // 異常発生状態
	EPCFaultDescription                      EPCType = 0x89 // 異常内容
	EPCManufacturerCode                      EPCType = 0x8a // メーカコード
	EPCBusinessFacilityCode                  EPCType = 0x8b // 事業場コード
	EPCProductCode                           EPCType = 0x8c // 商品コード
	EPCProductionNumber                      EPCType = 0x8d // 製造番号
	EPCProductionDate                        EPCType = 0x8e // 製造年月日
	EPCPowerSavingOperationSetting           EPCType = 0x8f // 節電動作設定
	EPCRemoteControlSetting                  EPCType = 0x93 // 遠隔操作設定
	EPCCurrentDate                           EPCType = 0x98 // 現在日時
	EPCStatusAnnouncementPropertyMap         EPCType = 0x9d // 状態アナウンスプロパティマップ
	EPCSetPropertyMap                        EPCType = 0x9e //Set プロパティマップ
	EPCGetPropertyMap                        EPCType = 0x9f // Get プロパティマップ
)

var ProfileSuperClass_PropertyTable = PropertyTable{
	Description: "Profile Super Class",
	EPCDesc: map[EPCType]PropertyDesc{
		EPCOperationStatus: {"Operation status", map[string][]byte{
			"on":  {0x30},
			"off": {0x31},
		}, nil},
		EPCInstallationLocation:                  {"Installation location", InstallationLocationAliases(), nil},
		EPCStandardVersion:                       {"Standard version", nil, StandardVersionDesc{}},
		EPCIdentificationNumber:                  {"Identification number", nil, IdentificationNumberDesc{}},
		EPCMeasuredInstantaneousPowerConsumption: {"Measured instantaneous power consumption", nil, NumberDesc{EDTLen: 2, Max: 65533, Unit: "W"}},
		EPCMeasuredCumulativePowerConsumption:    {"Measured cumulative power consumption", nil, CumulativePowerConsumptionDesc{}},
		EPCManufacturerFaultCode:                 {"Manufacturer fault code", nil, nil},
		EPCCurrentLimitSetting:                   {"Current limit setting", nil, nil},
		EPCFaultStatus: {"Fault occurrence status", map[string][]byte{
			"fault":    {0x41},
			"no_fault": {0x42},
		}, nil},
		EPCFaultDescription: {"Fault description", nil, nil},
		EPCManufacturerCode: {"Manufacturer code", map[string][]byte{
			"Sharp":        {0x00, 0x00, 0x05},
			"Daikin":       {0x00, 0x00, 0x08},
			"Panasonic":    {0x00, 0x00, 0x0b},
			"Experimental": {0xff, 0xff, 0xff},
		}, nil},
		EPCBusinessFacilityCode: {"Business facility code", nil, nil},
		EPCProductCode:          {"Product code", nil, StringDesc{MinEDTLen: 12, MaxEDTLen: 12}},
		EPCProductionNumber:     {"Production number", nil, StringDesc{MinEDTLen: 12, MaxEDTLen: 12}},
		EPCProductionDate:       {"Production date", nil, DateDesc{}},
		EPCPowerSavingOperationSetting: {"Power saving operation setting", map[string][]byte{
			"power_saving": {0x41},
			"normal":       {0x42},
		}, nil},
		EPCRemoteControlSetting: {"Remote control setting", map[string][]byte{
			"not_public_line":       {0x41}, // 公衆回線を経由しない制御
			"public_line":           {0x42}, // 公衆回線経由の制御
			"not_pubic_line_normal": {0x61}, // 通信回線正常（公衆回線経由の操作不可）
			"public_line_normal":    {0x62}, // 通信回線正常（公衆回線経由の操作可能）
		}, nil},
		EPCCurrentDate:                   {"Current date", nil, DateDesc{}},
		EPCStatusAnnouncementPropertyMap: {"Status announcement property map", nil, PropertyMapDesc{}},
		EPCSetPropertyMap:                {"Set property map", nil, PropertyMapDesc{}},
		EPCGetPropertyMap:                {"Get property map", nil, PropertyMapDesc{}},
	},
	DefaultEPCs: []EPCType{
		EPCOperationStatus,
		EPCInstallationLocation,
		EPCManufacturerCode,
		EPCProductCode,
	},
}

// InstallationLocationAliases は、設置場所コードに対応するエイリアス文字列とEDTのマップを生成します。
func InstallationLocationAliases() map[string][]byte {
	aliases := make(map[string][]byte)
	placeNames := map[byte]string{
		1: "living", 2: "dining", 3: "kitchen", 4: "bathroom",
		5: "lavatory", 6: "washroom", 7: "passageway", 8: "room",
		9: "storeroom", 10: "entrance", 11: "storage", 12: "garden",
		13: "garage", 14: "balcony", 15: "others",
	}

	for b := range 256 {
		locationByte := byte(b)
		var place string

		if (locationByte & 0x80) == 0 { // 通常の場所コード (ビット7が0)
			placeCode := (locationByte & 0x78) >> 3
			roomNumber := locationByte & 0x07

			if placeCode == 0 {
				if roomNumber == 0 {
					place = "unspecified"
				}
				// roomNumber > 0 は予約済みなので何もしない
			} else if baseName, ok := placeNames[placeCode]; ok {
				place = baseName
				if roomNumber != 0 {
					place = fmt.Sprintf("%s%d", place, roomNumber)
				}
			}
		} else { // 特殊な場所コード (ビット7が1)
			if locationByte == 0xff {
				place = "undetermined"
			}
			// その他の特殊コード (0x80-0xfe) はエイリアスに含めない
		}

		// 有効な名前が生成された場合のみエイリアスに追加
		if place != "" {
			aliases[place] = []byte{locationByte}
		}
	}
	return aliases
}

type StandardVersionDesc struct{}

func (d StandardVersionDesc) ToString(EDT []byte) (string, bool) {
	s := DecodeStandardVersion(EDT)
	if s == nil {
		return "", false
	}
	return s.String(), true
}

type StandardVersion struct {
	Reserved1 byte
	Reserved2 byte
	Release   byte
	Revision  byte
}

func DecodeStandardVersion(EDT []byte) *StandardVersion {
	if len(EDT) < 4 {
		return nil
	}
	return &StandardVersion{
		Reserved1: EDT[0],
		Reserved2: EDT[1],
		Release:   EDT[2],
		Revision:  EDT[3],
	}
}

func (s *StandardVersion) String() string {
	return fmt.Sprintf("Release %c Rev.%d", s.Release, s.Revision)
}

func (s *StandardVersion) Property() *Property {
	return &Property{EPC: EPCStandardVersion, EDT: []byte{s.Reserved1, s.Reserved2, s.Release, s.Revision}}
}

type IdentificationNumberDesc struct{}

func (d IdentificationNumberDesc) ToString(EDT []byte) (string, bool) {
	i := DecodeIdentificationNumber(EDT)
	if i == nil {
		return "", false
	}
	return i.String(), true
}

type IdentificationNumber struct {
	ManufacturerCode []byte // 3 bytes
	UniqueIdentifier []byte // 13 bytes
}

func DecodeIdentificationNumber(EDT []byte) *IdentificationNumber {
	if len(EDT) != 17 {
		return nil
	}
	if EDT[0] != 0xfe {
		// unknown ID type
		return nil
	}
	return &IdentificationNumber{
		ManufacturerCode: EDT[1:4],
		UniqueIdentifier: EDT[4:17],
	}
}

func (s *IdentificationNumber) String() string {
	return fmt.Sprintf("%X:%X", s.ManufacturerCode, s.UniqueIdentifier)
}

func (s *IdentificationNumber) Property() *Property {
	EDT := make([]byte, 0, 17)
	EDT = append(EDT, byte(0xfe))
	EDT = append(EDT, s.ManufacturerCode...)
	EDT = append(EDT, s.UniqueIdentifier...)
	return &Property{
		EPC: EPCIdentificationNumber,
		EDT: EDT,
	}
}

type CumulativePowerConsumptionDesc struct{}

func (d CumulativePowerConsumptionDesc) ToString(EDT []byte) (string, bool) {
	if len(EDT) != 4 {
		return "", false
	}
	power := utils.BytesToUint32(EDT)
	return fmt.Sprintf("%.3f kWh", float64(power)/1000.0), true
}

type DateDesc struct{}

func (d DateDesc) ToString(EDT []byte) (string, bool) {
	dt := DecodeDate(EDT)
	if dt == nil {
		return "", false
	}
	return dt.String(), true
}

type Date struct {
	Year  uint16
	Month uint8
	Day   uint8
}

func DecodeDate(EDT []byte) *Date {
	if len(EDT) != 4 {
		return nil
	}
	return &Date{
		Year:  uint16(utils.BytesToUint32(EDT[0:2])),
		Month: EDT[2],
		Day:   EDT[3],
	}
}

func (s *Date) String() string {
	return fmt.Sprintf("%04d/%02d/%02d", s.Year, s.Month, s.Day)
}

func (s *Date) EDT() []byte {
	yearBytes := utils.Uint32ToBytes(uint32(s.Year), 2)
	return []byte{yearBytes[0], yearBytes[1], s.Month, s.Day}
}

type PropertyMap map[EPCType]struct{}

func (m PropertyMap) Has(epc EPCType) bool {
	_, ok := m[epc]
	return ok
}

func (m PropertyMap) Set(epc EPCType) {
	m[epc] = struct{}{}
}

func (m PropertyMap) Delete(epc EPCType) {
	delete(m, epc)
}

func (m PropertyMap) EPCs() []EPCType {
	epcs := make([]EPCType, 0, len(m))
	for epc := range m {
		epcs = append(epcs, epc)
	}
	return epcs
}

type ErrInvalidPropertyMap struct {
	EDT []byte
}

func (e ErrInvalidPropertyMap) Error() string {
	return fmt.Sprintf("invalid property map: %X", e.EDT)
}

type PropertyMapDesc struct{}

func (d PropertyMapDesc) ToString(EDT []byte) (string, bool) {
	p := DecodePropertyMap(EDT)
	if p == nil {
		return "", false
	}
	return p.String(), true
}

// プロパティマップ記述形式
// プロパティマップは、EPC(0x80〜0xff)の有無の集合。
//
// 1. プロパティの個数が16未満の場合 (1+プロパティの個数バイト)
//   1バイト目: プロパティの個数
//   2バイト目以降: EPC がそのまま列挙される
//
// 2. プロパティの個数が16以上の場合 (17バイト)
//   1バイト目: プロパティの個数
//   2〜17バイト目: プロパティコードのビットマップ。8*16=128ビット。EPCは0x80〜0xff
//     ビットの場所は bytes[(EPC & 0x0f)] & (1 << ((EPC >> 4) - 8)) で表す。

func (m PropertyMap) Encode() []byte {
	if len(m) < 16 {
		bytes := make([]byte, 1, 1+len(m))
		bytes[0] = byte(len(m))
		for epc := range m {
			bytes = append(bytes, byte(epc))
		}
		return bytes
	}

	bytes := make([]byte, 17)
	bytes[0] = byte(len(m))
	for epc := range m {
		bytes[epc&0x0f+1] |= 1 << (epc>>4 - 8)
	}
	return bytes
}

func DecodePropertyMap(bytes []byte) PropertyMap {
	m := make(PropertyMap)
	if len(bytes) < 1 {
		return m
	}

	n := int(bytes[0])
	if n < 16 {
		if len(bytes) != n+1 {
			return nil
		}
		for _, epc := range bytes[1:] {
			m[EPCType(epc)] = struct{}{}
		}
	} else {
		if len(bytes) != 17 {
			return nil
		}
		for i, b := range bytes[1:] {
			for j := 0; j < 8; j++ {
				if b&(1<<j) != 0 {
					m[EPCType(i+j<<4+0x80)] = struct{}{}
				}
			}
		}
	}
	return m
}

func (m PropertyMap) String() string {
	var arr []EPCType
	for epc := range m {
		arr = append(arr, epc)
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i] < arr[j]
	})
	return fmt.Sprint(arr)
}
