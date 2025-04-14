package echonet_lite

import (
	"echonet-list/echonet_lite/utils"
	"fmt"
)

type EOJ uint32

type EOJClassCode uint16
type EOJInstanceCode uint8

func (e EOJ) ClassCode() EOJClassCode {
	return EOJClassCode(e >> 8 & 0xffff)
}
func (e EOJ) InstanceCode() EOJInstanceCode {
	return EOJInstanceCode(e)
}

type ClassGroupCodeType byte
type ClassCodeType byte

func (c EOJClassCode) ClassGroupCode() ClassGroupCodeType {
	return ClassGroupCodeType(c >> 8)
}
func (c EOJClassCode) ClassCode() ClassCodeType {
	return ClassCodeType(c)
}
func (c EOJClassCode) Encode() []byte {
	return utils.Uint32ToBytes(uint32(c), 2)
}

func MakeEOJ(classCode EOJClassCode, instanceCode EOJInstanceCode) EOJ {
	return EOJ(uint32(classCode)<<8 | uint32(instanceCode))
}

func DecodeEOJ(data []byte) EOJ {
	if len(data) != 3 {
		return 0
	}
	classCode := EOJClassCode(utils.BytesToUint32(data[0:2]))
	instanceCode := EOJInstanceCode(data[2])
	return MakeEOJ(classCode, instanceCode)
}
func (e EOJ) Encode() []byte {
	return utils.Uint32ToBytes(uint32(e), 3)
}

func (c EOJClassCode) String() string {
	var s string
	if p, ok := PropertyTables[c]; ok {
		s = p.Description
	} else {
		switch c.ClassGroupCode() {
		case 0x00:
			s = "Sensor-related device"
		case 0x01:
			s = "Air conditioner-related device"
		case 0x02:
			s = "Housing/facility-related device"
		case 0x03:
			s = "Cooking/housework-related device"
		case 0x04:
			s = "Health-related device"
		case 0x05:
			s = "Management/control-related device"
		case 0x06:
			s = "Audiovisual-related device"
		case 0x07:
			s = "Network-related device"
		case 0x0e:
			s = "Profile"
		case 0x0f:
			s = "User definition"
		default:
			s = "?"
		}
	}
	return fmt.Sprintf("%04X[%s]", uint16(c), s)
}

func (e EOJ) String() string {
	return fmt.Sprintf("%s:%v", e.ClassCode(), e.InstanceCode())
}

func (e EOJ) IDString() string {
	return fmt.Sprintf("%06X", uint32(e))
}

func (e EOJ) Specifier() string {
	if e.InstanceCode() == 0 {
		return fmt.Sprintf("%04X", uint16(e.ClassCode()))
	}
	return fmt.Sprintf("%04X:%d", uint16(e.ClassCode()), e.InstanceCode())
}
