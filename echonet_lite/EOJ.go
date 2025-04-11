package echonet_lite

import "fmt"

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
	return Uint32ToBytes(uint32(c), 2)
}

func MakeEOJClassCode(classGroupCode ClassGroupCodeType, classCode ClassCodeType) EOJClassCode {
	// Note: This function is kept for compatibility, but direct byte conversion is preferred.
	return EOJClassCode(uint16(classGroupCode)<<8 | uint16(classCode))
}
func MakeEOJ(classCode EOJClassCode, instanceCode EOJInstanceCode) EOJ {
	return EOJ(uint32(classCode)<<8 | uint32(instanceCode))
}

func DecodeEOJ(data []byte) EOJ {
	if len(data) != 3 {
		return 0
	}
	classCode := EOJClassCode(BytesToUint32(data[0:2]))
	instanceCode := EOJInstanceCode(data[2])
	return MakeEOJ(classCode, instanceCode)
}
func (e EOJ) Encode() []byte {
	return Uint32ToBytes(uint32(e), 3)
}

const (
	HomeAirConditioner_ClassCode     EOJClassCode = 0x0130 // 家庭用エアコン
	VentingFan_ClassCode             EOJClassCode = 0x0133 // 換気扇
	FloorHeating_ClassCode           EOJClassCode = 0x027b // 床暖房
	SingleFunctionLighting_ClassCode EOJClassCode = 0x0291 // 単機能照明
	LightingSystem_ClassCode         EOJClassCode = 0x02a3 // 照明システム
	Refrigerator_ClassCode           EOJClassCode = 0x03b7 // 冷凍冷蔵庫
	Switch_ClassCode                 EOJClassCode = 0x05fd // スイッチ
	PortableTerminal_ClassCode       EOJClassCode = 0x05fe // 携帯端末
	Controller_ClassCode             EOJClassCode = 0x05ff // コントローラ
	NodeProfile_ClassCode            EOJClassCode = 0x0ef0 // ノードプロファイル
)

func (c EOJClassCode) String() string {
	var s string
	switch c {
	case HomeAirConditioner_ClassCode:
		s = "Home air conditioner"
	case VentingFan_ClassCode:
		s = "Ventilation fan"
	case FloorHeating_ClassCode:
		// 床暖房
		s = "Floor heating"
	case SingleFunctionLighting_ClassCode:
		// 単機能照明
		s = "Single-function lighting"
	case LightingSystem_ClassCode:
		// 照明システム
		s = "Lighting system"
	case Refrigerator_ClassCode:
		// 冷凍冷蔵庫
		s = "Refrigerator"
	case Switch_ClassCode:
		s = "Switch"
	case PortableTerminal_ClassCode:
		// 携帯端末
		s = "Portable terminal"
	case Controller_ClassCode:
		s = "Controller"
	case NodeProfile_ClassCode:
		s = "Node profile"

	default:
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
