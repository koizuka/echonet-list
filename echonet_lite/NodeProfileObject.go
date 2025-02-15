package echonet_lite

import "fmt"

const (
	// EPC
	EPC_NPO_VersionInfo              EPCType = 0x82
	EPC_NPO_IDNumber                 EPCType = 0x83
	EPC_NPO_IndividualID             EPCType = 0xbf
	EPC_NPO_SelfNodeInstances        EPCType = 0xd3
	EPC_NPO_SelfNodeClasses          EPCType = 0xd4
	EPC_NPO_InstanceListNotification EPCType = 0xd5
	EPC_NPO_SelfNodeInstanceListS    EPCType = 0xd6
	EPC_NPO_SelfNodeClassListS       EPCType = 0xd7
)

var NPO_PropertyTable = PropertyTable{
	EPCInfo: map[EPCType]PropertyInfo{
		EPC_NPO_VersionInfo:              {"Version information", Decoder(NPO_DecodeVersionInfo), nil},
		EPC_NPO_IDNumber:                 {"Identification number", nil, nil},
		EPC_NPO_IndividualID:             {"Individual identification information", nil, nil},
		EPC_NPO_SelfNodeInstances:        {"Self-node instances number", nil, nil},
		EPC_NPO_SelfNodeClasses:          {"Self-node classes number", nil, nil},
		EPC_NPO_InstanceListNotification: {"instance list notification", Decoder(DecodeInstanceListNotification), nil},
		EPC_NPO_SelfNodeInstanceListS:    {"Self-node instance list S", Decoder(DecodeSelfNodeInstanceListS), nil},
		EPC_NPO_SelfNodeClassListS:       {"Self-node class list S", nil, nil},
	},
}

type NPO_VersionInfo struct {
	MajorVersion byte
	MinorVersion byte
	Default      bool // 既定電文
	Optional     bool // 任意電文
}

func NPO_DecodeVersionInfo(EDT []byte) *NPO_VersionInfo {
	if len(EDT) < 3 {
		return nil
	}
	return &NPO_VersionInfo{
		MajorVersion: EDT[0],
		MinorVersion: EDT[1],
		Default:      EDT[2]&0x01 != 0,
		Optional:     EDT[2]&0x02 != 0,
	}
}

func (s *NPO_VersionInfo) String() string {
	return fmt.Sprintf("%d.%d Default:%t, Optional:%t",
		s.MajorVersion, s.MinorVersion,
		s.Default, s.Optional,
	)
}

type InstanceList []EOJ

func DecodeInstanceList(EDT []byte) *InstanceList {
	if len(EDT) < 1 {
		return nil
	}
	result := InstanceList{}
	instances := int(EDT[0])
	if len(EDT) < 1+instances*3 {
		return nil
	}
	for i := 0; i < instances; i++ {
		eoj := DecodeEOJ(EDT[1+i*3 : 1+i*3+3])
		result = append(result, eoj)
	}
	return &result
}

func (s *InstanceList) String() string {
	if s == nil {
		return "nil"
	}
	return fmt.Sprintf("%d:%v", len(*s), *s)
}

func (s *InstanceList) EDT() []byte {
	if s == nil {
		return nil
	}
	EDT := make([]byte, 1, 1+len(*s)*3)
	EDT[0] = byte(len(*s))
	for _, eoj := range *s {
		EDT = append(EDT, eoj.Encode()...)
	}
	return EDT
}

type InstanceListNotification InstanceList

func DecodeInstanceListNotification(EDT []byte) *InstanceListNotification {
	l := DecodeInstanceList(EDT)
	if l == nil {
		return nil
	}
	return (*InstanceListNotification)(l)
}

func (s *InstanceListNotification) String() string {
	return (*InstanceList)(s).String()
}

func (s *InstanceListNotification) Property() *Property {
	if s == nil {
		return nil
	}
	return &Property{EPC_NPO_InstanceListNotification, (*InstanceList)(s).EDT()}
}

type SelfNodeInstanceListS InstanceList

func DecodeSelfNodeInstanceListS(EDT []byte) *SelfNodeInstanceListS {
	l := DecodeInstanceList(EDT)
	if l == nil {
		return nil
	}
	return (*SelfNodeInstanceListS)(l)
}

func (s *SelfNodeInstanceListS) String() string {
	return (*InstanceList)(s).String()
}

func (s *SelfNodeInstanceListS) Property() *Property {
	if s == nil {
		return nil
	}
	return &Property{EPC_NPO_SelfNodeInstanceListS, (*InstanceList)(s).EDT()}
}
