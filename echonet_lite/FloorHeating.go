package echonet_lite

import "fmt"

const (
	// EPC
	EPC_FH_RoomTemperature  EPCType = 0xE2 // 室内温度計測値
	EPC_FH_FloorTemperature EPCType = 0xE3 // 床温度計測値
	EPC_FH_OnTimerEnabled   EPCType = 0x90 // ONタイマ予約設定
	EPC_FH_OnTimerHHMM      EPCType = 0x91 // ONタイマ設定値
	EPC_FH_OffTimerEnabled  EPCType = 0x94 // OFFタイマ予約設定
	EPC_FH_OffTimerHHMM     EPCType = 0x95 // OFFタイマ設定値
)

var FH_PropertyTable = PropertyTable{
	EPCInfo: map[EPCType]PropertyInfo{
		EPC_FH_RoomTemperature:  {"Room temperature", Decoder(FH_DecodeTemperature), nil},
		EPC_FH_FloorTemperature: {"Floor temperature", Decoder(FH_DecodeTemperature), nil},

		EPC_FH_OnTimerEnabled:  {"ON timer enabled", Decoder(FH_DecodeOnOff), nil},
		EPC_FH_OnTimerHHMM:     {"ON timer setting", Decoder(FH_DecodeHHMM), nil},
		EPC_FH_OffTimerEnabled: {"OFF timer enabled", Decoder(FH_DecodeOnOff), nil},
		EPC_FH_OffTimerHHMM:    {"OFF timer setting", Decoder(FH_DecodeHHMM), nil},
	},
}

type Temperature int8

func FH_DecodeTemperature(EDT []byte) *Temperature {
	if len(EDT) < 1 {
		return nil
	}
	temp := Temperature(EDT[0])
	return &temp
}

func (t *Temperature) String() string {
	if t == nil {
		return "nil"
	}
	switch *t {
	case -128:
		return "Underflow"
	case 127:
		return "Overflow"
	case 0x7e:
		return "N/A"
	default:
		return fmt.Sprintf("%d℃", *t)
	}
}

func (t *Temperature) EDT() []byte {
	return []byte{byte(*t)}
}

type FH_OnOff bool

func FH_DecodeOnOff(EDT []byte) *FH_OnOff {
	if len(EDT) < 1 {
		return nil
	}
	temp := FH_OnOff(EDT[0] == 0x41)
	return &temp
}

func (t *FH_OnOff) String() string {
	if t == nil {
		return "nil"
	}
	if *t {
		return "ON"
	}
	return "OFF"
}

func (t *FH_OnOff) EDT() []byte {
	if *t {
		return []byte{0x41}
	}
	return []byte{0x42}
}

type FH_HHMM struct {
	Hour   int
	Minute int
}

func FH_DecodeHHMM(EDT []byte) *FH_HHMM {
	if len(EDT) < 2 {
		return nil
	}
	return &FH_HHMM{
		Hour:   int(EDT[0]),
		Minute: int(EDT[1]),
	}
}

func (t *FH_HHMM) String() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

func (t *FH_HHMM) EDT() []byte {
	return []byte{byte(t.Hour), byte(t.Minute)}
}
