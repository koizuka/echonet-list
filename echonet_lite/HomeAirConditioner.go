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

var HAC_PropertyTable = PropertyTable{
	EPCInfo: map[EPCType]PropertyInfo{
		EPC_HAC_AirVolumeSetting:         {"Air volume setting", Decoder(HAC_DecodeAirVolumeSetting), nil},
		EPC_HAC_AirDirectionSwingSetting: {"Air direction swing setting", Decoder(HAC_DecodeAirDirectionSwingSetting), nil},
		EPC_HAC_OperationModeSetting: {
			EPCs:    "Operation mode setting",
			Decoder: Decoder(HAC_DecodeOperationModeSetting),
			Aliases: map[string][]byte{
				"auto":    {HAC_OperationModeAutomatic},
				"cooling": {HAC_OperationModeCooling},
				"heating": {HAC_OperationModeHeating},
				"dry":     {HAC_OperationModeDehumidification},
				"fan":     {HAC_OperationModeAirCirculator},
				"other":   {HAC_OperationModeOther},
			},
		},
		EPC_HAC_TemperatureSetting:        {"Temperature setting", Decoder(HAC_DecodeTemperature), nil},
		EPC_HAC_RelativeHumiditySetting:   {"Relative humidity setting", Decoder(HAC_DecodeHumidity), nil},
		EPC_HAC_CurrentRoomHumidity:       {"Current room humidity", Decoder(HAC_DecodeHumidity), nil},
		EPC_HAC_CurrentRoomTemperature:    {"Current room temperature", Decoder(HAC_DecodeTemperature), nil},
		EPC_HAC_CurrentOutsideTemperature: {"Current outside temperature", Decoder(HAC_DecodeTemperature), nil},
		EPC_HAC_HumidificationModeSetting: {
			EPCs:    "Humidification mode setting",
			Decoder: Decoder(HAC_DecodeHumidificationModeSetting),
			Aliases: map[string][]byte{
				"on":  {0x41},
				"off": {0x42},
			},
		},
	},
	DefaultEPCs: []EPCType{
		EPC_HAC_OperationModeSetting,
		EPC_HAC_CurrentRoomTemperature,
		EPC_HAC_CurrentRoomHumidity,
		EPC_HAC_CurrentOutsideTemperature,
	},
}

type AirVolume byte

const (
	AirVolumeAuto AirVolume = 0x41
	AirVolume1    AirVolume = 0x31
	AirVolume2    AirVolume = 0x32
	AirVolume3    AirVolume = 0x33
	AirVolume4    AirVolume = 0x34
	AirVolume5    AirVolume = 0x35
	AirVolume6    AirVolume = 0x36
	AirVolume7    AirVolume = 0x37
	AirVolume8    AirVolume = 0x38
)

type HAC_AirVolumeSetting struct {
	Volume AirVolume
}

func HAC_DecodeAirVolumeSetting(EDT []byte) *HAC_AirVolumeSetting {
	if len(EDT) < 1 {
		return nil
	}
	return &HAC_AirVolumeSetting{Volume: AirVolume(EDT[0])}
}

func (s *HAC_AirVolumeSetting) String() string {
	switch s.Volume {
	case AirVolumeAuto:
		return "Auto"
	case AirVolume1:
		return "1/8"
	case AirVolume2:
		return "2/8"
	case AirVolume3:
		return "3/8"
	case AirVolume4:
		return "4/8"
	case AirVolume5:
		return "5/8"
	case AirVolume6:
		return "6/8"
	case AirVolume7:
		return "7/8"
	case AirVolume8:
		return "8/8"
	default:
		return fmt.Sprintf("Unknown(%X)", s.Volume)
	}
}

func (s *HAC_AirVolumeSetting) Property() *Property {
	return &Property{EPC: EPC_HAC_AirVolumeSetting, EDT: []byte{byte(s.Volume)}}
}

type HAC_AirDirectionSwing byte

const (
	// 風向スイング OFF＝0x31、
	//上下＝0x41、左右＝0x42、
	//上下左右＝0x43
	HAC_AirDirectionSwingOff        HAC_AirDirectionSwing = 0x31
	HAC_AirDirectionSwingVertical   HAC_AirDirectionSwing = 0x41
	HAC_AirDirectionSwingHorizontal HAC_AirDirectionSwing = 0x42
	HAC_AirDirectionSwingBoth       HAC_AirDirectionSwing = 0x43
)

type HAC_AirDirectionSwingSetting struct {
	Swing HAC_AirDirectionSwing
}

func HAC_DecodeAirDirectionSwingSetting(EDT []byte) *HAC_AirDirectionSwingSetting {
	if len(EDT) < 1 {
		return nil
	}
	return &HAC_AirDirectionSwingSetting{Swing: HAC_AirDirectionSwing(EDT[0])}
}

func (s *HAC_AirDirectionSwingSetting) String() string {
	switch s.Swing {
	case HAC_AirDirectionSwingOff:
		return "Off"
	case HAC_AirDirectionSwingVertical:
		return "Vertical"
	case HAC_AirDirectionSwingHorizontal:
		return "Horizontal"
	case HAC_AirDirectionSwingBoth:
		return "Both"
	default:
		return fmt.Sprintf("Unknown(%X)", s.Swing)
	}
}

func (s *HAC_AirDirectionSwingSetting) Property() *Property {
	return &Property{EPC: EPC_HAC_AirDirectionSwingSetting, EDT: []byte{byte(s.Swing)}}
}

type HAC_OperationModeSetting struct {
	Mode byte
}

const (
	HAC_OperationModeAutomatic        byte = 0x41
	HAC_OperationModeCooling          byte = 0x42
	HAC_OperationModeHeating          byte = 0x43
	HAC_OperationModeDehumidification byte = 0x44
	HAC_OperationModeAirCirculator    byte = 0x45
	HAC_OperationModeOther            byte = 0x40
)

func HAC_DecodeOperationModeSetting(EDT []byte) *HAC_OperationModeSetting {
	if len(EDT) < 1 {
		return nil
	}
	return &HAC_OperationModeSetting{Mode: EDT[0]}
}

func (s *HAC_OperationModeSetting) String() string {
	// Aliasを通して文字列化されるのでここは未知のときに来る
	return fmt.Sprintf("Unknown(%X)", s.Mode)
}

func (s *HAC_OperationModeSetting) Property() *Property {
	return &Property{EPC: EPC_HAC_OperationModeSetting, EDT: []byte{s.Mode}}
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

type HAC_Temperature int8

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
	return fmt.Sprintf("%d℃", *s)
}

type HAC_HumidificationMode bool

func HAC_DecodeHumidificationModeSetting(EDT []byte) *HAC_HumidificationMode {
	if len(EDT) < 1 {
		return nil
	}
	humidification := HAC_HumidificationMode(EDT[0] == 0x41)
	return &humidification
}

func (s *HAC_HumidificationMode) String() string {
	if s != nil && *s {
		return "On"
	}
	return "Off"
}

func (s *HAC_HumidificationMode) Property() *Property {
	var EDT byte
	if *s {
		EDT = 0x41
	} else {
		EDT = 0x42
	}
	return &Property{EPC: EPC_HAC_HumidificationModeSetting, EDT: []byte{EDT}}
}
