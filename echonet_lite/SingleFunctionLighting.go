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

				EPC_SF_Panasonic_OperationStatus: {"Panasonic Operation Status", Decoder(SF_Panasonic_DecodeOperationStatus), map[string][]byte{
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

// TODO Manufacturer codeがPanasonicの場合にのみ使うようにする

type SF_Panasonic_OperationStatus uint8 // 0x31:OFF, 0x30:ON

func SF_Panasonic_DecodeOperationStatus(EDT []byte) *SF_Panasonic_OperationStatus {
	if len(EDT) < 1 {
		return nil
	}
	operationStatus := SF_Panasonic_OperationStatus(EDT[0])
	return &operationStatus
}

func (s *SF_Panasonic_OperationStatus) String() string {
	if s == nil {
		return "nil"
	}
	return "Panasonic Operation Status: " + string(*s)
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
