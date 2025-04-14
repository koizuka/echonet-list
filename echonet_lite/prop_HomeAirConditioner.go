package echonet_lite

import "fmt"

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
	return PropertyRegistryEntry{
		ClassCode: HomeAirConditioner_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Home Air Conditioner",
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_HAC_AirVolumeSetting: {"Air volume setting", nil, HAC_AirVolumeAliases()},
				EPC_HAC_AirDirectionSwingSetting: {"Air direction swing setting", nil, map[string][]byte{
					"off":        {0x31},
					"vertical":   {0x41},
					"horizontal": {0x42},
					"both":       {0x43},
				}},
				EPC_HAC_OperationModeSetting: {"Operation mode setting", nil, map[string][]byte{
					"auto":    {0x41},
					"cooling": {0x42},
					"heating": {0x43},
					"dry":     {0x44},
					"fan":     {0x45},
					"other":   {0x40},
				}},
				EPC_HAC_TemperatureSetting:        {"Temperature setting", Decoder(HAC_DecodeTemperature), nil},
				EPC_HAC_RelativeHumiditySetting:   {"Relative humidity setting", Decoder(HAC_DecodeHumidity), nil},
				EPC_HAC_CurrentRoomHumidity:       {"Current room humidity", Decoder(HAC_DecodeHumidity), nil},
				EPC_HAC_CurrentRoomTemperature:    {"Current room temperature", Decoder(HAC_DecodeTemperature), nil},
				EPC_HAC_CurrentOutsideTemperature: {"Current outside temperature", Decoder(HAC_DecodeTemperature), nil},
				EPC_HAC_HumidificationModeSetting: {"Humidification mode setting", nil, map[string][]byte{
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

func HAC_AirVolumeAliases() map[string][]byte {
	result := make(map[string][]byte)
	result["auto"] = []byte{0x41}
	for i := 1; i <= 8; i++ {
		result[fmt.Sprintf("%d", i)] = []byte{byte(0x30 + i)}
	}
	return result
}

type HAC_Humidity uint8

func HAC_DecodeHumidity(EDT []byte) *HAC_Humidity {
	if len(EDT) < 1 {
		return nil
	}
	humidity := HAC_Humidity(EDT[0])
	return &humidity
}

func (s *HAC_Humidity) String() string {
	if s == nil {
		return "nil"
	}
	return fmt.Sprintf("%d%%", *s)
}

func (s *HAC_Humidity) EDT() []byte {
	if s == nil {
		return nil
	}
	return []byte{byte(*s)}
}

type HAC_Temperature uint8

func HAC_DecodeTemperature(EDT []byte) *HAC_Temperature {
	if len(EDT) < 1 {
		return nil
	}
	temp := HAC_Temperature(EDT[0])
	return &temp
}
func (s *HAC_Temperature) EDT() []byte {
	if s == nil {
		return nil
	}
	return []byte{byte(*s)}
}

func (s *HAC_Temperature) String() string {
	if s == nil {
		return "nil"
	}
	if *s == 0xfd {
		return "unknown"
	}
	return fmt.Sprintf("%d℃", *s)
}
