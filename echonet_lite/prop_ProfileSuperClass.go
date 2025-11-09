package echonet_lite

import (
	"echonet-list/echonet_lite/utils"
	"fmt"
	"time"
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
	EPCSetPropertyMap                        EPCType = 0x9e // Set プロパティマップ
	EPCGetPropertyMap                        EPCType = 0x9f // Get プロパティマップ
)

var ManufacturerCodeEDTs = map[string][]byte{
	"Sharp":        {0x00, 0x00, 0x05},
	"Daikin":       {0x00, 0x00, 0x08},
	"Panasonic":    {0x00, 0x00, 0x0b},
	"Experimental": {0xff, 0xff, 0xff},
}

var ProfileSuperClass_PropertyTable = PropertyTable{
	Description: "Profile Super Class",
	DescriptionTranslations: map[string]string{
		"ja": "プロファイルスーパークラス",
	},
	EPCDesc: map[EPCType]PropertyDesc{
		EPCOperationStatus: {
			Name: "Operation status",
			NameTranslations: map[string]string{
				"ja": "動作状態",
			},
			Aliases: map[string][]byte{
				"on":  {0x30},
				"off": {0x31},
			},
			AliasTranslations: map[string]map[string]string{
				"ja": {
					"on":  "on",
					"off": "off",
				},
			},
			Decoder: nil,
		},
		EPCInstallationLocation: {
			Name: "Installation location",
			NameTranslations: map[string]string{
				"ja": "設置場所",
			},
			Aliases:           InstallationLocationAliases(),
			AliasTranslations: InstallationLocationAliasTranslations(),
			Decoder:           nil,
		},
		EPCStandardVersion: {
			Name: "Standard version",
			NameTranslations: map[string]string{
				"ja": "規格Version情報",
			},
			Aliases: nil,
			Decoder: StandardVersionDesc{},
		},
		EPCIdentificationNumber: {
			Name: "Identification number",
			NameTranslations: map[string]string{
				"ja": "識別番号",
			},
			Aliases: nil,
			Decoder: IdentificationNumberDesc{},
		},
		EPCMeasuredInstantaneousPowerConsumption: {
			Name: "Measured instantaneous power consumption",
			NameTranslations: map[string]string{
				"ja": "瞬時電力計測値",
			},
			ShortName: "Instantaneous power",
			ShortNameTranslations: map[string]string{
				"ja": "瞬時電力",
			},
			Aliases: nil,
			Decoder: NumberDesc{EDTLen: 2, Max: 65533, Unit: "W"},
		},
		EPCMeasuredCumulativePowerConsumption: {
			Name: "Measured cumulative power consumption",
			NameTranslations: map[string]string{
				"ja": "積算電力量計測値",
			},
			ShortName: "Cumulative power",
			ShortNameTranslations: map[string]string{
				"ja": "積算電力量",
			},
			Aliases: nil,
			Decoder: CumulativePowerConsumptionDesc{},
		},
		EPCManufacturerFaultCode: {
			Name: "Manufacturer fault code",
			NameTranslations: map[string]string{
				"ja": "メーカ異常コード",
			},
			Aliases: nil,
			Decoder: nil,
		},
		EPCCurrentLimitSetting: {
			Name: "Current limit setting",
			NameTranslations: map[string]string{
				"ja": "電流制限設定",
			},
			Aliases: nil,
			Decoder: nil,
		},
		EPCFaultStatus: {
			Name: "Fault occurrence status",
			NameTranslations: map[string]string{
				"ja": "異常発生状況",
			},
			Aliases: map[string][]byte{
				"fault":    {0x41},
				"no_fault": {0x42},
			},
			AliasTranslations: map[string]map[string]string{
				"ja": {
					"fault":    "異常あり",
					"no_fault": "異常なし",
				},
			},
			Decoder: nil,
		},
		EPCFaultDescription: {
			Name: "Fault description",
			NameTranslations: map[string]string{
				"ja": "異常内容",
			},
			Aliases: FaultDescriptionAliases(),
			AliasTranslations: map[string]map[string]string{
				"ja": {
					"no_fault":                         "異常なし",
					"recoverable_power_cycle":          "復帰可能: 電源入れ直し",
					"recoverable_reset_button":         "復帰可能: リセットボタン",
					"recoverable_improper_setting":     "復帰可能: セット不良",
					"recoverable_supply":               "復帰可能: 補給",
					"recoverable_cleaning":             "復帰可能: 掃除",
					"recoverable_battery_change":       "復帰可能: 電池交換",
					"recoverable_no_action":            "復帰可能: 復帰操作不要",
					"repair_safety_device":             "要修理: 安全装置作動",
					"repair_switch_malfunction":        "要修理: スイッチ異常",
					"repair_sensor_malfunction":        "要修理: センサー異常",
					"repair_component_malfunction":     "要修理: 機能部品異常",
					"repair_control_board_malfunction": "要修理: 制御基板異常",
					"fault_unknown":                    "異常内容不明",
				},
			},
			Decoder: nil,
		},
		EPCManufacturerCode: {
			Name: "Manufacturer code",
			NameTranslations: map[string]string{
				"ja": "メーカコード",
			},
			Aliases: ManufacturerCodeEDTs,
			Decoder: nil,
		},
		EPCBusinessFacilityCode: {
			Name: "Business facility code",
			NameTranslations: map[string]string{
				"ja": "事業場コード",
			},
			Aliases: nil,
			Decoder: nil,
		},
		EPCProductCode: {
			Name: "Product code",
			NameTranslations: map[string]string{
				"ja": "商品コード",
			},
			Aliases: nil,
			Decoder: StringDesc{MinEDTLen: 12, MaxEDTLen: 12},
		},
		EPCProductionNumber: {
			Name: "Production number",
			NameTranslations: map[string]string{
				"ja": "製造番号",
			},
			Aliases: nil,
			Decoder: StringDesc{MinEDTLen: 12, MaxEDTLen: 12},
		},
		EPCProductionDate: {
			Name: "Production date",
			NameTranslations: map[string]string{
				"ja": "製造年月日",
			},
			Aliases: nil,
			Decoder: DateDesc{},
		},
		EPCPowerSavingOperationSetting: {
			Name: "Power saving operation setting",
			NameTranslations: map[string]string{
				"ja": "節電動作設定",
			},
			Aliases: map[string][]byte{
				"power_saving": {0x41},
				"normal":       {0x42},
			},
			AliasTranslations: map[string]map[string]string{
				"ja": {
					"power_saving": "節電中",
					"normal":       "通常",
				},
			},
			Decoder: nil,
		},
		EPCRemoteControlSetting: {
			Name: "Remote control setting",
			NameTranslations: map[string]string{
				"ja": "遠隔操作設定",
			},
			Aliases: map[string][]byte{
				"not_public_line":       {0x41}, // 公衆回線を経由しない制御
				"public_line":           {0x42}, // 公衆回線経由の制御
				"not_pubic_line_normal": {0x61}, // 通信回線正常（公衆回線経由の操作不可）
				"public_line_normal":    {0x62}, // 通信回線正常（公衆回線経由の操作可能）
			},
			AliasTranslations: map[string]map[string]string{
				"ja": {
					"not_public_line":       "公衆回線経由不可",
					"public_line":           "公衆回線経由可",
					"not_pubic_line_normal": "回線正常（遠隔不可）",
					"public_line_normal":    "回線正常（遠隔可）",
				},
			},
			Decoder: nil,
		},
		EPCCurrentDate: {
			Name: "Current date",
			NameTranslations: map[string]string{
				"ja": "現在年月日",
			},
			Aliases: nil,
			Decoder: DateDesc{},
		},
		EPCStatusAnnouncementPropertyMap: {
			Name: "Status announcement property map",
			NameTranslations: map[string]string{
				"ja": "状変アナウンスプロパティマップ",
			},
			Aliases: nil,
			Decoder: PropertyMapDesc{},
		},
		EPCSetPropertyMap: {
			Name: "Set property map",
			NameTranslations: map[string]string{
				"ja": "Setプロパティマップ",
			},
			ShortName: "Set map",
			ShortNameTranslations: map[string]string{
				"ja": "Set一覧",
			},
			Aliases: nil,
			Decoder: PropertyMapDesc{},
		},
		EPCGetPropertyMap: {
			Name: "Get property map",
			NameTranslations: map[string]string{
				"ja": "Getプロパティマップ",
			},
			ShortName: "Get map",
			ShortNameTranslations: map[string]string{
				"ja": "Get一覧",
			},
			Aliases: nil,
			Decoder: PropertyMapDesc{},
		},
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
		9: "staircase", 10: "entrance", 11: "storage", 12: "garden",
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

// InstallationLocationAliasTranslations は、設置場所のエイリアス翻訳マップを生成します。
func InstallationLocationAliasTranslations() map[string]map[string]string {
	translations := make(map[string]map[string]string)

	// 日本語翻訳
	jaTranslations := map[string]string{
		"living": "リビング", "dining": "ダイニング", "kitchen": "キッチン",
		"bathroom": "浴室", "lavatory": "トイレ", "washroom": "洗面所",
		"passageway": "廊下", "room": "部屋", "staircase": "階段室",
		"entrance": "玄関", "storage": "納戸", "garden": "庭",
		"garage": "ガレージ", "balcony": "バルコニー", "others": "その他",
		"unspecified": "未指定", "undetermined": "未定",
	}

	// 番号付きの場所も生成（例: living1, living2...）
	for i := 1; i <= 7; i++ {
		for enKey, jaValue := range jaTranslations {
			if enKey != "unspecified" && enKey != "undetermined" {
				keyWithNum := fmt.Sprintf("%s%d", enKey, i)
				jaTranslations[keyWithNum] = fmt.Sprintf("%s %d", jaValue, i)
			}
		}
	}

	translations["ja"] = jaTranslations
	return translations
}

// FaultDescriptionAliases は、異常内容コードに対応するエイリアス文字列とEDTのマップを生成します。
func FaultDescriptionAliases() map[string][]byte {
	aliases := make(map[string][]byte)

	// 0x0000: 異常なし
	aliases["no_fault"] = []byte{0x00, 0x00}

	// 0x0001〜0x0009: 復帰可能な異常
	aliases["recoverable_power_cycle"] = []byte{0x00, 0x01}      // 電源入れ直し
	aliases["recoverable_reset_button"] = []byte{0x00, 0x02}     // リセットボタン
	aliases["recoverable_improper_setting"] = []byte{0x00, 0x03} // セット不良
	aliases["recoverable_supply"] = []byte{0x00, 0x04}           // 補給
	aliases["recoverable_cleaning"] = []byte{0x00, 0x05}         // 掃除
	aliases["recoverable_battery_change"] = []byte{0x00, 0x06}   // 電池交換
	aliases["recoverable_no_action"] = []byte{0x00, 0x07}        // 復帰操作不要

	// 0x000a〜0x00e9: 要修理
	// 0x000a〜0x0013: 安全装置作動
	aliases["repair_safety_device"] = []byte{0x00, 0x0a} // 安全装置作動（代表値）

	// 0x0014〜0x001d: スイッチ異常
	aliases["repair_switch_malfunction"] = []byte{0x00, 0x14} // スイッチ異常（代表値）

	// 0x001e〜0x003b: センサー異常
	aliases["repair_sensor_malfunction"] = []byte{0x00, 0x1e} // センサー異常（代表値）

	// 0x003c〜0x0059: 機能部品異常
	aliases["repair_component_malfunction"] = []byte{0x00, 0x3c} // 機能部品異常（代表値）

	// 0x005a〜0x006e: 制御基板異常
	aliases["repair_control_board_malfunction"] = []byte{0x00, 0x5a} // 制御基板異常（代表値）

	// 0x03ff: 異常内容不明
	aliases["fault_unknown"] = []byte{0x03, 0xff}

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
	if s == nil {
		return nil
	}
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
	if len(EDT) != 4 {
		return "", false
	}
	year := uint16(utils.BytesToUint32(EDT[0:2]))
	month := EDT[2]
	day := EDT[3]
	return fmt.Sprintf("%04d/%02d/%02d", year, month, day), true
}

func (d DateDesc) FromString(s string) ([]byte, bool) {
	date, err := time.Parse("2006/1/2", s)
	if err != nil {
		return nil, false
	}
	year := date.Year()
	month := date.Month()
	day := date.Day()
	return []byte{byte(year >> 8), byte(year & 0xff), byte(month), byte(day)}, true
}
