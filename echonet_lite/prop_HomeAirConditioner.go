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
	TemperatureSetting := NumberValueDesc{Min: 0, Max: 50, Unit: "℃"}
	MeasuredTemperature := NumberValueDesc{Min: -127, Max: 125, Unit: "℃"}
	ExtraValueAlias := map[string][]byte{
		"unknown":   {0xFD},
		"underflow": {0xFE},
		"overflow":  {0xFF},
	}
	Humidity := NumberValueDesc{Min: 0, Max: 100, Unit: "%"}

	return PropertyRegistryEntry{
		ClassCode: HomeAirConditioner_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Home Air Conditioner",
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_HAC_AirVolumeSetting: {Desc: "Air volume setting", Aliases: map[string][]byte{
					"auto": {0x41},
				}, Number: &NumberValueDesc{Min: 1, Max: 8, Offset: 0x30}},
				EPC_HAC_AirDirectionSwingSetting: {Desc: "Air direction swing setting", Aliases: map[string][]byte{
					"off":        {0x31},
					"vertical":   {0x41},
					"horizontal": {0x42},
					"both":       {0x43},
				}},
				EPC_HAC_OperationModeSetting: {Desc: "Operation mode setting", Aliases: map[string][]byte{
					"auto":    {0x41},
					"cooling": {0x42},
					"heating": {0x43},
					"dry":     {0x44},
					"fan":     {0x45},
					"other":   {0x40},
				}},
				EPC_HAC_TemperatureSetting:        {Desc: "Temperature setting", Aliases: ExtraValueAlias, Number: &TemperatureSetting},
				EPC_HAC_RelativeHumiditySetting:   {Desc: "Relative humidity setting", Aliases: ExtraValueAlias, Number: &Humidity},
				EPC_HAC_CurrentRoomHumidity:       {Desc: "Current room humidity", Aliases: ExtraValueAlias, Number: &Humidity},
				EPC_HAC_CurrentRoomTemperature:    {Desc: "Current room temperature", Aliases: ExtraValueAlias, Number: &MeasuredTemperature},
				EPC_HAC_CurrentOutsideTemperature: {Desc: "Current outside temperature", Aliases: ExtraValueAlias, Number: &MeasuredTemperature},
				EPC_HAC_HumidificationModeSetting: {Desc: "Humidification mode setting", Aliases: map[string][]byte{
					"on":  {0x41},
					"off": {0x42},
				}},
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
