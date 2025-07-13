package echonet_lite

import "time"

const (
	// EPC
	EPC_HAC_AirVolumeSetting          EPCType = 0xA0 // 風量設定
	EPC_HAC_AirDirectionSwingSetting  EPCType = 0xA3 // 風向スイング設定
	EPC_HAC_OperationModeSetting      EPCType = 0xB0 // 運転モード設定
	EPC_HAC_TemperatureSetting        EPCType = 0xB3 // 温度設定値
	EPC_HAC_RelativeHumiditySetting   EPCType = 0xB4 // 除湿モード時相対湿度設定値
	EPC_HAC_CurrentRoomHumidity       EPCType = 0xBA
	EPC_HAC_CurrentRoomTemperature    EPCType = 0xBB
	EPC_HAC_CurrentOutsideTemperature EPCType = 0xBE
	EPC_HAC_HumidificationModeSetting EPCType = 0xC1 // 加湿モード設定
)

func (r PropertyRegistry) HomeAirConditioner() PropertyTable {
	TemperatureSettingDesc := NumberDesc{Min: 0, Max: 50, Unit: "℃"}
	MeasuredTemperatureDesc := NumberDesc{Min: -127, Max: 125, Unit: "℃"}
	ExtraValueAlias := map[string][]byte{
		"unknown":   {0xFD},
		"underflow": {0xFE},
		"overflow":  {0xFF},
	}
	ExtraValueAliasTranslations := map[string]map[string]string{
		"ja": {
			"unknown":   "不明",
			"underflow": "アンダーフロー",
			"overflow":  "オーバーフロー",
		},
	}
	HumidityDesc := NumberDesc{Min: 0, Max: 100, Unit: "%"}

	return PropertyTable{
		ClassCode:   HomeAirConditioner_ClassCode,
		Description: "Home Air Conditioner",
		DescriptionTranslations: map[string]string{
			"ja": "家庭用エアコン",
		},
		EPCDesc: map[EPCType]PropertyDesc{
			EPC_HAC_AirVolumeSetting: {
				Name: "Air volume setting",
				NameTranslations: map[string]string{
					"ja": "風量設定",
				},
				Aliases: map[string][]byte{
					"auto": {0x41},
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"auto": "自動",
					},
				},
				Decoder: NumberDesc{Min: 1, Max: 8, Offset: 0x30},
			},
			EPC_HAC_AirDirectionSwingSetting: {
				Name: "Air direction swing setting",
				NameTranslations: map[string]string{
					"ja": "風向スイング設定",
				},
				Aliases: map[string][]byte{
					"off":        {0x31},
					"vertical":   {0x41},
					"horizontal": {0x42},
					"both":       {0x43},
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"off":        "停止",
						"vertical":   "上下",
						"horizontal": "左右",
						"both":       "上下左右",
					},
				},
				Decoder: nil,
			},
			EPC_HAC_OperationModeSetting: {
				Name: "Operation mode setting",
				NameTranslations: map[string]string{
					"ja": "運転モード設定",
				},
				Aliases: map[string][]byte{
					"auto":    {0x41},
					"cooling": {0x42},
					"heating": {0x43},
					"dry":     {0x44},
					"fan":     {0x45},
					"other":   {0x40},
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"auto":    "自動",
						"cooling": "冷房",
						"heating": "暖房",
						"dry":     "除湿",
						"fan":     "送風",
						"other":   "その他",
					},
				},
				Decoder: nil,
				// 運転モードが変更されたら、温度設定値などを再取得する
				TriggerUpdate: true,
				UpdateDelay:   2 * time.Second,
				UpdateTargets: []EPCType{
					EPC_HAC_TemperatureSetting,
					EPC_HAC_RelativeHumiditySetting,
				},
			},
			EPC_HAC_TemperatureSetting: {
				Name: "Temperature setting",
				NameTranslations: map[string]string{
					"ja": "温度設定値",
				},
				Aliases:           ExtraValueAlias,
				AliasTranslations: ExtraValueAliasTranslations,
				Decoder:           TemperatureSettingDesc,
			},
			EPC_HAC_RelativeHumiditySetting: {
				Name: "Relative humidity setting",
				NameTranslations: map[string]string{
					"ja": "除湿モード時相対湿度設定値",
				},
				Aliases:           ExtraValueAlias,
				AliasTranslations: ExtraValueAliasTranslations,
				Decoder:           HumidityDesc,
			},
			EPC_HAC_CurrentRoomHumidity: {
				Name: "Current room humidity",
				NameTranslations: map[string]string{
					"ja": "室内相対湿度計測値",
				},
				Aliases:           ExtraValueAlias,
				AliasTranslations: ExtraValueAliasTranslations,
				Decoder:           HumidityDesc,
			},
			EPC_HAC_CurrentRoomTemperature: {
				Name: "Current room temperature",
				NameTranslations: map[string]string{
					"ja": "室内温度計測値",
				},
				Aliases:           ExtraValueAlias,
				AliasTranslations: ExtraValueAliasTranslations,
				Decoder:           MeasuredTemperatureDesc,
			},
			EPC_HAC_CurrentOutsideTemperature: {
				Name: "Current outside temperature",
				NameTranslations: map[string]string{
					"ja": "屋外温度計測値",
				},
				Aliases:           ExtraValueAlias,
				AliasTranslations: ExtraValueAliasTranslations,
				Decoder:           MeasuredTemperatureDesc,
			},
			EPC_HAC_HumidificationModeSetting: {
				Name: "Humidification mode setting",
				NameTranslations: map[string]string{
					"ja": "加湿モード設定",
				},
				Aliases: map[string][]byte{
					"on":  {0x41},
					"off": {0x42},
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"on":  "入",
						"off": "切",
					},
				},
				Decoder: nil,
			},
		},
		DefaultEPCs: []EPCType{
			EPC_HAC_OperationModeSetting,
			EPC_HAC_TemperatureSetting,
			EPC_HAC_CurrentRoomTemperature,
			EPC_HAC_CurrentRoomHumidity,
			EPC_HAC_CurrentOutsideTemperature,
		},
	}
}
