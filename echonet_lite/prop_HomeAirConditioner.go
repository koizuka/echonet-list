package echonet_lite

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

func (r PropertyRegistry) HomeAirConditioner() PropertyRegistryEntry {
	TemperatureSetting := NumberValueDesc{Min: 0, Max: 50, Unit: "℃", EDTLen: 1}
	MeasuredTemperature := NumberValueDesc{Min: -127, Max: 125, Unit: "℃", EDTLen: 1}
	ExtraValueAlias := map[string][]byte{
		"unknown":   {0xFD},
		"underflow": {0xFE},
		"overflow":  {0xFF},
	}
	Humidity := NumberValueDesc{Min: 0, Max: 100, Unit: "%", EDTLen: 1}

	return PropertyRegistryEntry{
		ClassCode: HomeAirConditioner_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Home Air Conditioner",
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_HAC_AirVolumeSetting: {"Air volume setting", nil, map[string][]byte{
					"auto": {0x41},
				}, &NumberValueDesc{Min: 1, Max: 8, Offset: 0x30, EDTLen: 1}},
				EPC_HAC_AirDirectionSwingSetting: {"Air direction swing setting", nil, map[string][]byte{
					"off":        {0x31},
					"vertical":   {0x41},
					"horizontal": {0x42},
					"both":       {0x43},
				}, nil},
				EPC_HAC_OperationModeSetting: {"Operation mode setting", nil, map[string][]byte{
					"auto":    {0x41},
					"cooling": {0x42},
					"heating": {0x43},
					"dry":     {0x44},
					"fan":     {0x45},
					"other":   {0x40},
				}, nil},
				EPC_HAC_TemperatureSetting:        {"Temperature setting", nil, ExtraValueAlias, &TemperatureSetting},
				EPC_HAC_RelativeHumiditySetting:   {"Relative humidity setting", nil, ExtraValueAlias, &Humidity},
				EPC_HAC_CurrentRoomHumidity:       {"Current room humidity", nil, ExtraValueAlias, &Humidity},
				EPC_HAC_CurrentRoomTemperature:    {"Current room temperature", nil, ExtraValueAlias, &MeasuredTemperature},
				EPC_HAC_CurrentOutsideTemperature: {"Current outside temperature", nil, ExtraValueAlias, &MeasuredTemperature},
				EPC_HAC_HumidificationModeSetting: {"Humidification mode setting", nil, map[string][]byte{
					"on":  {0x41},
					"off": {0x42},
				}, nil},
			},
			DefaultEPCs: []EPCType{
				EPC_HAC_OperationModeSetting,
				EPC_HAC_TemperatureSetting,
				EPC_HAC_CurrentRoomTemperature,
				EPC_HAC_CurrentRoomHumidity,
				EPC_HAC_CurrentOutsideTemperature,
			},
		},
	}
}
