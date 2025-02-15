package echonet_lite

import "fmt"

const (
	// EPC
	EPC_SF_Illuminance EPCType = 0xb0
)

var SF_PropertyTable = PropertyTable{
	EPCInfo: map[EPCType]PropertyInfo{
		EPC_SF_Illuminance: {"Illuminance level", Decoder(SF_DecodeIlluminance), nil},
	},
	DefaultEPCs: []EPCType{
		EPC_SF_Illuminance,
	},
}

type Illuminance uint8

func SF_DecodeIlluminance(EDT []byte) *Illuminance {
	if len(EDT) < 1 {
		return nil
	}
	illuminance := Illuminance(EDT[0])
	return &illuminance
}

func (i *Illuminance) String() string {
	if i == nil {
		return "nil"
	}
	return fmt.Sprintf("%d%%", *i)
}
