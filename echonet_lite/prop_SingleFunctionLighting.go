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
		EPCDesc: map[EPCType]PropertyDesc{
			EPC_SF_Illuminance: {"Illuminance level", nil, IlluminanceDesc},
			EPC_SF_Panasonic_OperationStatus: {"Panasonic Operation Status", map[string][]byte{
				"on":  {0x30},
				"off": {0x31},
			}, nil},
			EPC_SF_Panasonic_Illuminance:     {"Panasonic Illuminance", nil, IlluminanceDesc},
			EPC_SF_Panasonic_UnknownStringFD: {"Panasonic Unknown String FD", nil, StringDesc{MaxEDTLen: 255 /* ? */}},
			EPC_SF_Panasonic_UnknownStringFE: {"Panasonic Unknown String FE", nil, StringDesc{MaxEDTLen: 255 /* ? */}},
		},
		DefaultEPCs: []EPCType{
			EPC_SF_Illuminance,
		},
	}
}
