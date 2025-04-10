package echonet_lite

import (
	"fmt"
	"sort"
	"strings"
)

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

const (
	// Manufacturer code
	ManufacturerCodeSharp        ManufacturerCode = 0x000005
	ManufacturerCodeDaikin       ManufacturerCode = 0x000008
	ManufacturerCodePanasonic    ManufacturerCode = 0x00000b
	ManufacturerCodeExperimental ManufacturerCode = 0xffffff
)

var ProfileSuperClass_PropertyTable = PropertyTable{
	EPCInfo: map[EPCType]PropertyInfo{
		EPCOperationStatus: {"Operation status", Decoder(DecodeOperationStatus), map[string][]byte{
			"on":  {0x30},
			"off": {0x31},
		}},
		EPCInstallationLocation: {"Installation location", Decoder(DecodeInstallationLocation),
			InstallationLocationAliases()},
		EPCStandardVersion:                       {"Standard version", Decoder(DecodeStandardVersion), nil},
		EPCIdentificationNumber:                  {"Identification number", Decoder(DecodeIdentificationNumber), nil},
		EPCMeasuredInstantaneousPowerConsumption: {"Measured instantaneous power consumption", Decoder(DecodeInstantaneousPowerConsumption), nil},
		EPCMeasuredCumulativePowerConsumption:    {"Measured cumulative power consumption", Decoder(DecodeCumulativePowerConsumption), nil},
		EPCManufacturerFaultCode:                 {"Manufacturer fault code", nil, nil},
		EPCCurrentLimitSetting:                   {"Current limit setting", nil, nil},
		EPCFaultStatus: {"Fault occurrence status", Decoder(DecodeFaultStatus), map[string][]byte{
			"fault":    {0x41},
			"no_fault": {0x42},
		}},
		EPCFaultDescription: {"Fault description", nil, nil},
		EPCManufacturerCode: {"Manufacturer code", Decoder(DecodeManufacturerCode), map[string][]byte{
			"Sharp":        ManufacturerCodeSharp.EDT(),
			"Daikin":       ManufacturerCodeDaikin.EDT(),
			"Panasonic":    ManufacturerCodePanasonic.EDT(),
			"Experimental": ManufacturerCodeExperimental.EDT(),
		}},
		EPCBusinessFacilityCode:          {"Business facility code", nil, nil},
		EPCProductCode:                   {"Product code", Decoder(DecodeProductCode), nil},
		EPCProductionNumber:              {"Production number", nil, nil},
		EPCProductionDate:                {"Production date", Decoder(DecodeDate), nil},
		EPCPowerSavingOperationSetting:   {"Power saving operation setting", nil, nil},
		EPCRemoteControlSetting:          {"Remote control setting", nil, nil},
		EPCCurrentDate:                   {"Current date", Decoder(DecodeDate), nil},
		EPCStatusAnnouncementPropertyMap: {"Status announcement property map", Decoder(DecodePropertyMap), nil},
		EPCSetPropertyMap:                {"Set property map", Decoder(DecodePropertyMap), nil},
		EPCGetPropertyMap:                {"Get property map", Decoder(DecodePropertyMap), nil},
	},
	DefaultEPCs: []EPCType{
		EPCOperationStatus,
		EPCInstallationLocation,
		EPCManufacturerCode,
		EPCProductCode,
	},
}

type OperationStatus bool

func DecodeOperationStatus(EDT []byte) OperationStatus {
	if len(EDT) < 1 {
		return false
	}
	return EDT[0] == 0x30
}

func (s OperationStatus) String() string {
	return fmt.Sprintf("%t", s)
}

func (s OperationStatus) Property() *Property {
	var EDT byte
	if s {
		EDT = 0x30
	} else {
		EDT = 0x31
	}
	return &Property{EPC: EPCOperationStatus, EDT: []byte{EDT}}
}

type InstallationLocation struct {
	PlaceCode  byte // 0..15 or 0x80..0xff
	RoomNumber byte // 0..7
}

func DecodeInstallationLocation(EDT []byte) *InstallationLocation {
	if len(EDT) < 1 {
		return nil
	}
	location := EDT[0]
	if (location & 0x80) == 0 {
		return &InstallationLocation{
			PlaceCode:  (location & 0x78) >> 3,
			RoomNumber: location & 0x07,
		}
	} else {
		// When bit 7 is set, use the raw value as PlaceCode
		return &InstallationLocation{
			PlaceCode:  location,
			RoomNumber: 0,
		}
	}
}

func (s *InstallationLocation) String() string {
	if s == nil {
		return "nil"
	}
	place := ""
	if (s.PlaceCode & 0x80) == 0 {
		placeCode := s.PlaceCode
		roomNumber := s.RoomNumber
		switch placeCode {
		case 0:
			switch roomNumber {
			case 0:
				place = "unspecified"
			default:
				place = "reserved"
			}

		case 1:
			place = "living"
		case 2:
			place = "dining"
		case 3:
			place = "kitchen"
		case 4:
			place = "bathroom"
		case 5:
			place = "lavatory"
		case 6:
			place = "washroom"
		case 7:
			place = "passageway"
		case 8:
			place = "room"
		case 9:
			place = "storeroom"
		case 10:
			place = "entrance"
		case 11:
			place = "storage"
		case 12:
			place = "garden"
		case 13:
			place = "garage"
		case 14:
			place = "balcony"
		case 15:
			place = "others"
		}
		if roomNumber != 0 {
			place = fmt.Sprintf("%s%d", place, roomNumber)
		}
	} else {
		if s.PlaceCode == 0xff {
			place = "undetermined"
		} else {
			place = fmt.Sprintf("unknown(%X)", s.PlaceCode)
		}
	}
	return place
}

func (s *InstallationLocation) Property() *Property {
	var location byte
	if (s.PlaceCode & 0x80) == 0 {
		// Normal case: combine PlaceCode and RoomNumber
		location = (s.PlaceCode << 3) | (s.RoomNumber & 0x07)
	} else {
		// Special case: use PlaceCode directly
		location = s.PlaceCode
	}
	return &Property{EPC: EPCInstallationLocation, EDT: []byte{location}}
}

func InstallationLocationAliases() map[string][]byte {
	aliases := make(map[string][]byte)
	for b := 0; b < 256; b++ {
		loc := DecodeInstallationLocation([]byte{byte(b)})
		if loc != nil {
			name := loc.String()
			if !strings.HasPrefix(name, "unknown") && !strings.HasPrefix(name, "reserved") {
				aliases[name] = []byte{byte(b)}
			}
		}
	}
	return aliases
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

type IdentificationNumber struct {
	ManufacturerCode ManufacturerCode
	UniqueIdentifier []byte // 13 bytes
}

func DecodeIdentificationNumber(EDT []byte) *IdentificationNumber {
	if len(EDT) < 16 {
		return nil
	}
	return &IdentificationNumber{
		ManufacturerCode: ManufacturerCode(uint32(EDT[0])<<16 | uint32(EDT[1])<<8 | uint32(EDT[2])),
		UniqueIdentifier: EDT[3:16],
	}
}

func (s *IdentificationNumber) String() string {
	return fmt.Sprintf("%v:%X", s.ManufacturerCode, s.UniqueIdentifier)
}

func (s *IdentificationNumber) Property() *Property {
	return &Property{
		EPC: EPCIdentificationNumber,
		EDT: append(s.ManufacturerCode.EDT(), s.UniqueIdentifier...),
	}
}

type InstantaneousPowerConsumption struct {
	Power uint16
}

func DecodeInstantaneousPowerConsumption(EDT []byte) *InstantaneousPowerConsumption {
	if len(EDT) < 2 {
		return nil
	}
	return &InstantaneousPowerConsumption{
		Power: uint16(EDT[0])<<8 | uint16(EDT[1]),
	}
}

func (s *InstantaneousPowerConsumption) String() string {
	return fmt.Sprintf("%d W", s.Power)
}

func (s *InstantaneousPowerConsumption) Property() *Property {
	return &Property{EPC: EPCMeasuredInstantaneousPowerConsumption, EDT: []byte{byte(s.Power >> 8), byte(s.Power & 0xff)}}
}

type CumulativePowerConsumption struct {
	Power uint32
}

func DecodeCumulativePowerConsumption(EDT []byte) *CumulativePowerConsumption {
	if len(EDT) < 4 {
		return nil
	}
	return &CumulativePowerConsumption{
		Power: uint32(EDT[0])<<24 | uint32(EDT[1])<<16 | uint32(EDT[2])<<8 | uint32(EDT[3]),
	}
}

func (s *CumulativePowerConsumption) String() string {
	return fmt.Sprintf("%f kWh", float64(s.Power)/1000.0)
}

func (s *CumulativePowerConsumption) Property() *Property {
	return &Property{EPC: EPCMeasuredCumulativePowerConsumption, EDT: []byte{byte(s.Power >> 24), byte(s.Power >> 16), byte(s.Power >> 8), byte(s.Power & 0xff)}}
}

type FaultStatus struct {
	Fault bool
}

func DecodeFaultStatus(EDT []byte) *FaultStatus {
	if len(EDT) != 1 {
		return nil
	}
	switch EDT[0] {
	case 0x41:
		return &FaultStatus{Fault: true}
	case 0x42:
		return &FaultStatus{Fault: false}
	}
	return nil
}

func (s *FaultStatus) String() string {
	p := s.Property()
	return fmt.Sprintf("%X", p.EDT)
}

func (s *FaultStatus) Property() *Property {
	var EDT byte
	if s.Fault {
		EDT = 0x41
	} else {
		EDT = 0x42
	}
	return &Property{EPC: EPCFaultStatus, EDT: []byte{EDT}}
}

type ManufacturerCode uint32

func DecodeManufacturerCode(EDT []byte) ManufacturerCode {
	if len(EDT) < 3 {
		return 0
	}
	return ManufacturerCode(uint32(EDT[0])<<16 | uint32(EDT[1])<<8 | uint32(EDT[2]))
}

func (c ManufacturerCode) String() string {
	return fmt.Sprintf("%X", uint32(c))
}

func (c ManufacturerCode) EDT() []byte {
	return []byte{byte(c >> 16), byte(c >> 8), byte(c)}
}

func (c ManufacturerCode) Property() *Property {
	return &Property{EPC: EPCManufacturerCode, EDT: c.EDT()}
}

type ProductCode string

func DecodeProductCode(EDT []byte) ProductCode {
	return ProductCode(EDT)
}

func (c ProductCode) String() string {
	return string(c)
}

func (c ProductCode) Property() *Property {
	return &Property{EPC: EPCProductCode, EDT: []byte(c)}
}

type Date struct {
	Year  uint16
	Month uint8
	Day   uint8
}

func DecodeDate(EDT []byte) *Date {
	if len(EDT) < 4 {
		return nil
	}
	return &Date{
		Year:  uint16(EDT[0])<<8 | uint16(EDT[1]),
		Month: EDT[2],
		Day:   EDT[3],
	}
}

func (s *Date) String() string {
	return fmt.Sprintf("%04d/%02d/%02d", s.Year, s.Month, s.Day)
}

func (s *Date) EDT() []byte {
	return []byte{byte(s.Year >> 8), byte(s.Year & 0xff), s.Month, s.Day}
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
