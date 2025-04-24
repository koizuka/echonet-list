package echonet_lite

const (
	// EPC
	EPC_SF_Illuminance EPCType = 0xb0

	EPC_SF_Panasonic_OperationStatus EPCType = 0xf3
	EPC_SF_Panasonic_Illuminance     EPCType = 0xf4
	EPC_SF_Panasonic_UnknownStringFD EPCType = 0xfd
	EPC_SF_Panasonic_UnknownStringFE EPCType = 0xfe
)

func (r PropertyRegistry) SingleFunctionLighting() PropertyRegistryEntry {
	Illuminance := NumberValueDesc{Min: 0, Max: 100, Unit: "%"}

	return PropertyRegistryEntry{
		ClassCode: SingleFunctionLighting_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Single Function Lighting",
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_SF_Illuminance: {Desc: "Illuminance level", Number: &Illuminance},
				EPC_SF_Panasonic_OperationStatus: {Desc: "Panasonic Operation Status", Aliases: map[string][]byte{
					"on":  {0x30},
					"off": {0x31},
				}},
				EPC_SF_Panasonic_Illuminance:     {Desc: "Panasonic Illuminance", Number: &Illuminance},
				EPC_SF_Panasonic_UnknownStringFD: {Desc: "Panasonic Unknown String FD", Decoder: Decoder(SF_Panasonic_DecodeUnknownString)},
				EPC_SF_Panasonic_UnknownStringFE: {Desc: "Panasonic Unknown String FE", Decoder: Decoder(SF_Panasonic_DecodeUnknownString)},
			},
			DefaultEPCs: []EPCType{
				EPC_SF_Illuminance,
			},
		},
	}
}

type SF_Panasonic_UnknownString string

func SF_Panasonic_DecodeUnknownString(EDT []byte) *SF_Panasonic_UnknownString {
	if len(EDT) < 1 {
		return nil
	}
	result := SF_Panasonic_UnknownString(string(EDT))
	return &result
}

func (s *SF_Panasonic_UnknownString) String() string {
	if s == nil {
		return "nil"
	}
	return string(*s)
}
