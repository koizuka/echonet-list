package echonet_lite

import (
	"echonet-list/echonet_lite/props"
)

const (
	// EPC
	EPC_SF_Illuminance EPCType = 0xb0

	EPC_SF_Panasonic_OperationStatus EPCType = 0xf3
	EPC_SF_Panasonic_Illuminance     EPCType = 0xf4
	EPC_SF_Panasonic_UnknownStringFD EPCType = 0xfd
	EPC_SF_Panasonic_UnknownStringFE EPCType = 0xfe
)

func (r PropertyRegistry) SingleFunctionLighting() PropertyRegistryEntry {
	return PropertyRegistryEntry{
		ClassCode: SingleFunctionLighting_ClassCode,
		PropertyTable: PropertyTable{
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_SF_Illuminance: {"Illuminance level", Decoder(props.DecodeIlluminance), nil},

				EPC_SF_Panasonic_OperationStatus: {"Panasonic Operation Status", nil, map[string][]byte{
					"on":  {0x30},
					"off": {0x31},
				}},
				EPC_SF_Panasonic_Illuminance:     {"Panasonic Illuminance", Decoder(props.DecodeIlluminance), nil},
				EPC_SF_Panasonic_UnknownStringFD: {"Panasonic Unknown String FD", Decoder(SF_Panasonic_DecodeUnknownString), nil},
				EPC_SF_Panasonic_UnknownStringFE: {"Panasonic Unknown String FE", Decoder(SF_Panasonic_DecodeUnknownString), nil},
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
