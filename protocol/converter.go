package protocol

import (
	"echonet-list/echonet_lite"
	"net"
)

// ConvertIPAndEOJToDeviceInfo は IPAndEOJ を DeviceInfo に変換する
func ConvertIPAndEOJToDeviceInfo(device echonet_lite.IPAndEOJ, aliases []string) DeviceInfo {
	classCode := device.EOJ.ClassCode()
	instanceCode := device.EOJ.InstanceCode()

	return DeviceInfo{
		IP: device.IP.String(),
		EOJ: EOJInfo{
			ClassCode:    ClassCode(classCode),
			InstanceCode: uint8(instanceCode),
		},
		Aliases: aliases,
	}
}

// ConvertDeviceInfoToIPAndEOJ は DeviceInfo を IPAndEOJ に変換する
func ConvertDeviceInfoToIPAndEOJ(deviceInfo DeviceInfo) (echonet_lite.IPAndEOJ, error) {
	ip := net.ParseIP(deviceInfo.IP)
	if ip == nil {
		return echonet_lite.IPAndEOJ{}, ErrInvalidIP
	}

	eoj := echonet_lite.MakeEOJ(
		echonet_lite.EOJClassCode(deviceInfo.EOJ.ClassCode),
		echonet_lite.EOJInstanceCode(deviceInfo.EOJ.InstanceCode),
	)

	return echonet_lite.IPAndEOJ{
		IP:  ip,
		EOJ: eoj,
	}, nil
}

// ConvertPropertyToPropertyInfo は Property を PropertyInfo に変換する
func ConvertPropertyToPropertyInfo(property echonet_lite.Property, classCode echonet_lite.EOJClassCode) PropertyInfo {
	propInfo := PropertyInfo{
		EPC: EPCType(property.EPC),
		EDT: ByteArray(property.EDT),
	}

	// プロパティ情報が存在する場合、名前と説明を追加
	if info, ok := echonet_lite.GetPropertyInfo(classCode, property.EPC); ok {
		// PropertyInfoにはDecoderがあるので、それを使って値を設定
		if info.Decoder != nil {
			decodedValue := info.Decoder(property.EDT)
			propInfo.Value = decodedValue.String()
		}

		// EPCs（名前）を設定
		propInfo.Name = info.EPCs
	}

	return propInfo
}

// ConvertPropertyInfoToProperty は PropertyInfo を Property に変換する
func ConvertPropertyInfoToProperty(propInfo PropertyInfo) echonet_lite.Property {
	return echonet_lite.Property{
		EPC: echonet_lite.EPCType(propInfo.EPC),
		EDT: []byte(propInfo.EDT),
	}
}

// ConvertDeviceSpecifierToEchonetDeviceSpecifier は DeviceSpecifier を echonet_lite.DeviceSpecifier に変換する
func ConvertDeviceSpecifierToEchonetDeviceSpecifier(spec DeviceSpecifier) echonet_lite.DeviceSpecifier {
	var result echonet_lite.DeviceSpecifier

	if spec.IP != nil {
		ip := net.ParseIP(*spec.IP)
		result.IP = &ip
	}

	if spec.ClassCode != nil {
		classCode := echonet_lite.EOJClassCode(*spec.ClassCode)
		result.ClassCode = &classCode
	}

	if spec.InstanceCode != nil {
		instanceCode := echonet_lite.EOJInstanceCode(*spec.InstanceCode)
		result.InstanceCode = &instanceCode
	}

	// エイリアスは別途処理

	return result
}

// ConvertEchonetDeviceSpecifierToDeviceSpecifier は echonet_lite.DeviceSpecifier を DeviceSpecifier に変換する
func ConvertEchonetDeviceSpecifierToDeviceSpecifier(spec echonet_lite.DeviceSpecifier) DeviceSpecifier {
	var result DeviceSpecifier

	if spec.IP != nil {
		ipStr := spec.IP.String()
		result.IP = &ipStr
	}

	if spec.ClassCode != nil {
		classCode := ClassCode(*spec.ClassCode)
		result.ClassCode = &classCode
	}

	if spec.InstanceCode != nil {
		instanceCode := uint8(*spec.InstanceCode)
		result.InstanceCode = &instanceCode
	}

	return result
}

// ConvertPropertyToEchonetProperty は Property を echonet_lite.Property に変換する
func ConvertPropertyToEchonetProperty(prop Property) echonet_lite.Property {
	return echonet_lite.Property{
		EPC: echonet_lite.EPCType(prop.EPC),
		EDT: []byte(prop.EDT),
	}
}

// ConvertEchonetPropertyToProperty は echonet_lite.Property を Property に変換する
func ConvertEchonetPropertyToProperty(prop echonet_lite.Property) Property {
	return Property{
		EPC: EPCType(prop.EPC),
		EDT: ByteArray(prop.EDT),
	}
}

// エラー定義
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidIP = Error("無効なIPアドレス")
)