package echonet_lite

import (
	"echonet-list/echonet_lite/utils"
	"fmt"
)

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

var NodeProfileObject = MakeEOJ(NodeProfile_ClassCode, 1)
var NodeProfileObject_SendOnly = MakeEOJ(NodeProfile_ClassCode, 2)

var ECHONETLite_Version NPO_VersionInfo = NPO_VersionInfo{
	MajorVersion: 1,
	MinorVersion: 14,
	Default:      true,
	Optional:     false,
}

func (r PropertyRegistry) NodeProfileObject() PropertyRegistryEntry {
	return PropertyRegistryEntry{
		ClassCode: NodeProfile_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Node Profile",
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_NPO_VersionInfo:              {Desc: "Version information", Decoder: Decoder(NPO_DecodeVersionInfo)},
				EPC_NPO_IDNumber:                 {Desc: "Identification number"},
				EPC_NPO_IndividualID:             {Desc: "Individual identification information"},
				EPC_NPO_SelfNodeInstances:        {Desc: "Self-node instances number", Decoder: Decoder(DecodeSelfNodeInstances)},
				EPC_NPO_SelfNodeClasses:          {Desc: "Self-node classes number", Decoder: Decoder(DecodeSelfNodeClasses)},
				EPC_NPO_InstanceListNotification: {Desc: "instance list notification", Decoder: Decoder(DecodeInstanceListNotification)},
				EPC_NPO_SelfNodeInstanceListS:    {Desc: "Self-node instance list S", Decoder: Decoder(DecodeSelfNodeInstanceListS)},
				EPC_NPO_SelfNodeClassListS:       {Desc: "Self-node class list S", Decoder: Decoder(DecodeSelfNodeClassListS)},
			},
		},
	}
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

func (s *NPO_VersionInfo) EDT() []byte {
	if s == nil {
		return nil
	}
	var thirdByte byte
	if s.Default {
		thirdByte |= 0x01
	}
	if s.Optional {
		thirdByte |= 0x02
	}
	return []byte{s.MajorVersion, s.MinorVersion, thirdByte}
}

func (s *NPO_VersionInfo) Property() *Property {
	if s == nil {
		return nil
	}
	return &Property{EPC_NPO_VersionInfo, s.EDT()}
}

func (s *NPO_VersionInfo) String() string {
	return fmt.Sprintf("%d.%d Default:%t, Optional:%t",
		s.MajorVersion, s.MinorVersion,
		s.Default, s.Optional,
	)
}

type SelfNodeInstances uint32

func DecodeSelfNodeInstances(EDT []byte) *SelfNodeInstances {
	if len(EDT) != 3 {
		return nil
	}
	result := utils.BytesToUint32(EDT)
	return (*SelfNodeInstances)(&result)
}

func (s *SelfNodeInstances) String() string {
	if s == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *s)
}

func (s *SelfNodeInstances) Property() *Property {
	if s == nil {
		return nil
	}
	return &Property{EPC_NPO_SelfNodeInstances, utils.Uint32ToBytes(uint32(*s), 3)}
}

type SelfNodeClasses uint16

func DecodeSelfNodeClasses(EDT []byte) *SelfNodeClasses {
	if len(EDT) != 2 {
		return nil
	}
	classes := SelfNodeClasses(utils.BytesToUint32(EDT))
	return &classes
}

func (s *SelfNodeClasses) String() string {
	if s == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", *s)
}

func (s *SelfNodeClasses) Property() *Property {
	if s == nil {
		return nil
	}
	return &Property{EPC_NPO_SelfNodeClasses, utils.Uint32ToBytes(uint32(*s), 2)}
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

type SelfNodeClassListS []EOJClassCode

func DecodeSelfNodeClassListS(EDT []byte) *SelfNodeClassListS {
	if len(EDT) < 1 {
		return nil
	}
	result := SelfNodeClassListS{}
	classes := int(EDT[0])
	if len(EDT) < 1+classes*2 {
		return nil
	}
	for i := 0; i < classes; i++ {
		class := EOJClassCode(EDT[1+i*2])
		result = append(result, class)
	}
	return &result
}

func (s *SelfNodeClassListS) String() string {
	if s == nil {
		return "nil"
	}
	return fmt.Sprintf("%d:%v", len(*s), *s)
}

func (s *SelfNodeClassListS) EDT() []byte {
	if s == nil {
		return nil
	}
	EDT := make([]byte, 1, 1+len(*s)*2)
	EDT[0] = byte(len(*s))
	for _, class := range *s {
		EDT = append(EDT, class.Encode()...)
	}
	return EDT
}

func (s *SelfNodeClassListS) Property() *Property {
	if s == nil {
		return nil
	}
	return &Property{EPC_NPO_SelfNodeClassListS, s.EDT()}
}
