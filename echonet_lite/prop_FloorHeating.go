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

func (r PropertyRegistry) FloorHeating() PropertyTable {
	var FH_OnOffAlias = map[string][]byte{
		"on":  {0x41},
		"off": {0x42},
	}
	var FH_OnOffAliasTranslations = map[string]map[string]string{
		"ja": {
			"on":  "入",
			"off": "切",
		},
	}
	MeasuredTemperatureDesc := NumberDesc{Min: -127, Max: 125, Unit: "℃"}
	ExtraValueAlias := map[string][]byte{
		"N/A":       {0x7e},
		"overflow":  {0x7f},
		"underflow": {0x80},
	}
	ExtraValueAliasTranslations := map[string]map[string]string{
		"ja": {
			"N/A":       "N/A",
			"overflow":  "オーバーフロー",
			"underflow": "アンダーフロー",
		},
	}

	return PropertyTable{
		ClassCode:   FloorHeating_ClassCode,
		Description: "Floor Heating",
		DescriptionMap: map[string]string{
			"ja": "床暖房",
		},
		EPCDesc: map[EPCType]PropertyDesc{
			EPC_FH_TemperatureLevel: {
				Name: "Temperature setting(level)",
				NameMap: map[string]string{
					"ja": "温度設定値",
				},
				Aliases: map[string][]byte{
					"auto": {0x41},
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"auto": "自動",
					},
				},
				Decoder: NumberDesc{Min: 1, Max: 15, Offset: 0x30},
			},
			EPC_FH_RoomTemperature: {
				Name: "Room temperature",
				NameMap: map[string]string{
					"ja": "室内温度計測値",
				},
				Aliases:           ExtraValueAlias,
				AliasTranslations: ExtraValueAliasTranslations,
				Decoder:           MeasuredTemperatureDesc,
			},
			EPC_FH_FloorTemperature: {
				Name: "Floor temperature",
				NameMap: map[string]string{
					"ja": "床温度計測値",
				},
				Aliases:           ExtraValueAlias,
				AliasTranslations: ExtraValueAliasTranslations,
				Decoder:           MeasuredTemperatureDesc,
			},
			EPC_FH_SpecialMode: {
				Name: "Special mode",
				NameMap: map[string]string{
					"ja": "特別運転設定",
				},
				Aliases: map[string][]byte{
					"normal": {0x41}, // 通常運転
					"low":    {0x42}, // ひかえめ運転
					"high":   {0x43}, // ハイパワー運転
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"normal": "通常運転",
						"low":    "ひかえめ運転",
						"high":   "ハイパワー運転",
					},
				},
				Decoder: nil,
			},
			EPC_FH_DailyTimerEnabled: {
				Name: "Daily timer enabled",
				NameMap: map[string]string{
					"ja": "デイリータイマー設定",
				},
				Aliases: map[string][]byte{
					"off":         {0x40},
					"dailyTimer1": {0x41},
					"dailyTimer2": {0x42},
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"off":         "オフ",
						"dailyTimer1": "デイリータイマー1",
						"dailyTimer2": "デイリータイマー2",
					},
				},
				Decoder: nil,
			},
			EPC_FH_DailyTimer1: {
				Name: "Daily timer1",
				NameMap: map[string]string{
					"ja": "デイリータイマー1設定値",
				},
				Aliases: nil,
				Decoder: FH_DailyTimerDesc{},
			},
			EPC_FH_DailyTimer2: {
				Name: "Daily timer2",
				NameMap: map[string]string{
					"ja": "デイリータイマー2設定値",
				},
				Aliases: nil,
				Decoder: FH_DailyTimerDesc{},
			},

			EPC_FH_OnTimerEnabled: {
				Name: "ON timer enabled",
				NameMap: map[string]string{
					"ja": "ONタイマ予約設定",
				},
				Aliases:           FH_OnOffAlias,
				AliasTranslations: FH_OnOffAliasTranslations,
				Decoder:           nil,
			},
			EPC_FH_OnTimerHHMM: {
				Name: "ON timer setting",
				NameMap: map[string]string{
					"ja": "ONタイマ設定値",
				},
				Aliases: nil,
				Decoder: FH_HHMMDesc{},
			},
			EPC_FH_OffTimerEnabled: {
				Name: "OFF timer enabled",
				NameMap: map[string]string{
					"ja": "OFFタイマ予約設定",
				},
				Aliases:           FH_OnOffAlias,
				AliasTranslations: FH_OnOffAliasTranslations,
				Decoder:           nil,
			},
			EPC_FH_OffTimerHHMM: {
				Name: "OFF timer setting",
				NameMap: map[string]string{
					"ja": "OFFタイマ設定値",
				},
				Aliases: nil,
				Decoder: FH_HHMMDesc{},
			},

			EPC_FH_Temperature1: {
				Name: "Temperature sensor 1",
				NameMap: map[string]string{
					"ja": "温度センサ1",
				},
				Aliases:           ExtraValueAlias,
				AliasTranslations: ExtraValueAliasTranslations,
				Decoder:           MeasuredTemperatureDesc,
			},
			EPC_FH_Temperature2: {
				Name: "Temperature sensor 2",
				NameMap: map[string]string{
					"ja": "温度センサ2",
				},
				Aliases:           ExtraValueAlias,
				AliasTranslations: ExtraValueAliasTranslations,
				Decoder:           MeasuredTemperatureDesc,
			},
		},
		DefaultEPCs: []EPCType{
			EPC_FH_TemperatureLevel,
			EPC_FH_RoomTemperature,
			EPC_FH_SpecialMode,
			EPC_FH_Temperature1,
			EPC_FH_Temperature2,
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
