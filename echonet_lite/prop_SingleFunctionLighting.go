package echonet_lite

const (
	// EPC
	EPC_SF_Illuminance EPCType = 0xb0

	EPC_SF_Panasonic_OperationStatus EPCType = 0xf3
	EPC_SF_Panasonic_Illuminance     EPCType = 0xf4
	EPC_SF_Panasonic_UnknownStringFD EPCType = 0xfd
	EPC_SF_Panasonic_UnknownStringFE EPCType = 0xfe
)

func (r PropertyRegistry) SingleFunctionLighting() PropertyTable {
	IlluminanceDesc := NumberDesc{Min: 0, Max: 100, Unit: "%"}

	return PropertyTable{
		ClassCode:   SingleFunctionLighting_ClassCode,
		Description: "Single Function Lighting",
		DescriptionMap: map[string]string{
			"ja": "単機能照明",
		},
		EPCDesc: map[EPCType]PropertyDesc{
			EPC_SF_Illuminance: {
				Name: "Illuminance level",
				NameMap: map[string]string{
					"ja": "照度レベル",
				},
				Aliases: nil,
				Decoder: IlluminanceDesc,
			},
			EPC_SF_Panasonic_OperationStatus: {
				Name: "Panasonic Operation Status",
				NameMap: map[string]string{
					"ja": "パナソニック動作状態",
				},
				Aliases: map[string][]byte{
					"on":  {0x30},
					"off": {0x31},
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"on":  "オン",
						"off": "オフ",
					},
				},
				Decoder: nil,
			},
			EPC_SF_Panasonic_Illuminance: {
				Name: "Panasonic Illuminance",
				NameMap: map[string]string{
					"ja": "パナソニック照度",
				},
				Aliases: nil,
				Decoder: IlluminanceDesc,
			},
			EPC_SF_Panasonic_UnknownStringFD: {
				Name: "Panasonic Unknown String FD",
				NameMap: map[string]string{
					"ja": "パナソニック不明文字列FD",
				},
				Aliases: nil,
				Decoder: StringDesc{MaxEDTLen: 255 /* ? */},
			},
			EPC_SF_Panasonic_UnknownStringFE: {
				Name: "Panasonic Unknown String FE",
				NameMap: map[string]string{
					"ja": "パナソニック不明文字列FE",
				},
				Aliases: nil,
				Decoder: StringDesc{MaxEDTLen: 255 /* ? */},
			},
		},
		DefaultEPCs: []EPCType{
			EPC_SF_Illuminance,
		},
	}
}
