package echonet_lite

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// EPC
	EPC_FH_TemperatureLevel  EPCType = 0xE1 // 温度設定値
	EPC_FH_RoomTemperature   EPCType = 0xE2 // 室内温度計測値
	EPC_FH_FloorTemperature  EPCType = 0xE3 // 床温度計測値
	EPC_FH_SpecialMode       EPCType = 0xE5 // 特別運転設定
	EPC_FH_DailyTimerEnabled EPCType = 0xE6 // デイリータイマー設定
	EPC_FH_DailyTimer1       EPCType = 0xE7 // デイリータイマー1設定値
	EPC_FH_DailyTimer2       EPCType = 0xE8 // デイリータイマー2設定値
	EPC_FH_OnTimerEnabled    EPCType = 0x90 // ONタイマ予約設定
	EPC_FH_OnTimerHHMM       EPCType = 0x91 // ONタイマ設定値
	EPC_FH_OffTimerEnabled   EPCType = 0x94 // OFFタイマ予約設定
	EPC_FH_OffTimerHHMM      EPCType = 0x95 // OFFタイマ設定値
	EPC_FH_Temperature1      EPCType = 0xf3 // Daikin: 温度センサ1(行きの水温?)
	EPC_FH_Temperature2      EPCType = 0xf4 // Daikin: 温度センサ2(戻りの水温?)
)

func (r PropertyRegistry) FloorHeating() PropertyRegistryEntry {
	var FH_OnOffAlias = map[string][]byte{
		"on":  {0x41},
		"off": {0x42},
	}
	MeasuredTemperatureDesc := NumberDesc{Min: -127, Max: 125, Unit: "℃"}
	ExtraValueAlias := map[string][]byte{
		"N/A":       {0x7e},
		"overflow":  {0x7f},
		"underflow": {0x80},
	}

	return PropertyRegistryEntry{
		ClassCode: FloorHeating_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Floor Heating",
			EPCDesc: map[EPCType]PropertyDesc{
				EPC_FH_TemperatureLevel: {"Temperature setting(level)", map[string][]byte{
					"auto": {0x41},
				}, NumberDesc{Min: 1, Max: 15, Offset: 0x30}},
				EPC_FH_RoomTemperature:  {"Room temperature", ExtraValueAlias, MeasuredTemperatureDesc},
				EPC_FH_FloorTemperature: {"Floor temperature", ExtraValueAlias, MeasuredTemperatureDesc},
				EPC_FH_SpecialMode: {"Special mode", map[string][]byte{
					"normal": {0x41}, // 通常運転
					"low":    {0x42}, // ひかえめ運転
					"high":   {0x43}, // ハイパワー運転
				}, nil},
				EPC_FH_DailyTimerEnabled: {"Daily timer enabled", map[string][]byte{
					"off":         {0x40},
					"dailyTimer1": {0x41},
					"dailyTimer2": {0x42},
				}, nil},
				EPC_FH_DailyTimer1: {"Daily timer1", nil, FH_DailyTimerDesc{}},
				EPC_FH_DailyTimer2: {"Daily timer2", nil, FH_DailyTimerDesc{}},

				EPC_FH_OnTimerEnabled:  {"ON timer enabled", FH_OnOffAlias, nil},
				EPC_FH_OnTimerHHMM:     {"ON timer setting", nil, FH_HHMMDesc{}},
				EPC_FH_OffTimerEnabled: {"OFF timer enabled", FH_OnOffAlias, nil},
				EPC_FH_OffTimerHHMM:    {"OFF timer setting", nil, FH_HHMMDesc{}},

				EPC_FH_Temperature1: {"Temperature sensor 1", ExtraValueAlias, MeasuredTemperatureDesc},
				EPC_FH_Temperature2: {"Temperature sensor 2", ExtraValueAlias, MeasuredTemperatureDesc},
			},
			DefaultEPCs: []EPCType{
				EPC_FH_TemperatureLevel,
				EPC_FH_RoomTemperature,
				EPC_FH_SpecialMode,
				EPC_FH_Temperature1,
				EPC_FH_Temperature2,
			},
		},
	}
}

type FH_HHMMDesc struct{}

func (d FH_HHMMDesc) ToString(EDT []byte) (string, bool) {
	if len(EDT) != 2 {
		return "", false
	}
	hour := int(EDT[0])
	minute := int(EDT[1])
	return fmt.Sprintf("%02d:%02d", hour, minute), true
}

func (d FH_HHMMDesc) FromString(s string) ([]byte, bool) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return nil, false
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, false
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, false
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return nil, false
	}
	return []byte{byte(hour), byte(minute)}, true
}

type FH_DailyTimerDesc struct{}

func (d FH_DailyTimerDesc) ToString(EDT []byte) (string, bool) {
	timer := FH_DecodeDailyTimer(EDT)
	if timer == nil {
		return "", false
	}
	return timer.String(), true
}

// デイリータイマー設定値: 各ビットが30分で24時間を表す(6バイト, 0x01=0:0-0:30) -> [0-47]bool
type FH_DailyTimer [48]bool

func FH_DecodeDailyTimer(EDT []byte) *FH_DailyTimer {
	if len(EDT) < 6 {
		return nil
	}
	timer := FH_DailyTimer{}
	for i := 0; i < 6; i++ {
		for j := 0; j < 8; j++ {
			timer[i*8+j] = (EDT[i]>>j)&0x01 == 1
		}
	}
	return &timer
}
func (t *FH_DailyTimer) String() string {
	if t == nil {
		return "nil"
	}

	type range_t struct{ start, end int }
	var ranges []range_t
	for i := 0; i < len(*t); i++ {
		if (*t)[i] {
			start := i
			for i < len(*t) && (*t)[i] {
				i++
			}
			end := i
			ranges = append(ranges, range_t{start, end})
		}
	}

	s := make([]string, 0, len(ranges))
	for _, r := range ranges {
		s = append(s, fmt.Sprintf("%02d:%02d-%02d:%02d",
			r.start/2, (r.start%2)*30,
			r.end/2, (r.end%2)*30,
		))
	}

	return strings.Join([]string{"[", strings.Join(s, ", "), "]"}, "")
}
func (t *FH_DailyTimer) EDT() []byte {
	if t == nil {
		return nil
	}
	EDT := make([]byte, 6)
	for i := 0; i < 6; i++ {
		for j := 0; j < 8; j++ {
			if (*t)[i*8+j] {
				EDT[i] |= (1 << j)
			}
		}
	}
	return EDT
}
