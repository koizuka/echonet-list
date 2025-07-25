package handler

import (
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/network"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ParseEOJString parses a string in the format "CCCC:I" where CCCC is a 4-digit hex class code
// and I is a decimal instance code. Returns the parsed EOJ.
// Examples: "0130:1", "0EF0:1"
func ParseEOJString(eojStr string) (EOJ, error) {
	parts := strings.Split(eojStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid EOJ format: %s (expected format: CCCC:I)", eojStr)
	}

	classCode, err := ParseEOJClassCodeString(parts[0])
	if err != nil {
		return 0, err
	}

	instanceCode, err := ParseEOJInstanceCodeString(parts[1])
	if err != nil {
		return 0, err
	}

	return echonet_lite.MakeEOJ(classCode, instanceCode), nil
}

// ParseEOJClassCodeString parses a 4-digit hex string into an EOJClassCode.
// Example: "0130" -> HomeAirConditioner_ClassCode
func ParseEOJClassCodeString(classCodeStr string) (EOJClassCode, error) {
	if len(classCodeStr) != 4 {
		return 0, fmt.Errorf("class code must be 4 hexadecimal digits: %s", classCodeStr)
	}

	classCode64, err := strconv.ParseUint(classCodeStr, 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid class code: %s (must be 4 hexadecimal digits)", classCodeStr)
	}

	return EOJClassCode(classCode64), nil
}

// ParseEOJInstanceCodeString parses a decimal string into an EOJInstanceCode.
// Example: "1" -> EOJInstanceCode(1)
func ParseEOJInstanceCodeString(instanceCodeStr string) (EOJInstanceCode, error) {
	instanceCode64, err := strconv.ParseUint(instanceCodeStr, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid instance code: %s (must be a number between 1-255)", instanceCodeStr)
	}

	if instanceCode64 == 0 || instanceCode64 > 255 {
		return 0, fmt.Errorf("instance code must be between 1 and 255")
	}

	return EOJInstanceCode(instanceCode64), nil
}

// ParseEPCString parses a 2-digit hex string into an EPCType.
// Example: "80" -> EPCType(0x80)
func ParseEPCString(epcStr string) (EPCType, error) {
	if len(epcStr) != 2 {
		return 0, fmt.Errorf("EPC must be 2 hexadecimal digits: %s", epcStr)
	}

	epc64, err := strconv.ParseUint(epcStr, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid EPC: %s (must be 2 hexadecimal digits)", epcStr)
	}

	return EPCType(epc64), nil
}

// ParseHexString parses a hex string into a byte array.
// Example: "30" -> []byte{0x30}
func ParseHexString(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		return nil, fmt.Errorf("hex string must be a multiple of 2 characters: %s", hexStr)
	}

	bytes := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		b, err := strconv.ParseUint(hexStr[i:i+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex byte: %s", hexStr[i:i+2])
		}
		bytes[i/2] = byte(b)
	}

	return bytes, nil
}

// ParseDeviceIdentifier parses a device identifier string in the format "IP EOJ"
// where IP is an IP address and EOJ is in the format "CCCC:I".
// Example: "192.168.0.1 0130:1"
func ParseDeviceIdentifier(deviceIdStr string) (IPAndEOJ, error) {
	parts := strings.Fields(deviceIdStr)
	if len(parts) != 2 {
		return IPAndEOJ{}, fmt.Errorf("invalid device identifier format: %#v (expected format: IP EOJ)", deviceIdStr)
	}

	ip := net.ParseIP(parts[0])
	if ip == nil {
		return IPAndEOJ{}, fmt.Errorf("invalid IP address: %s", parts[0])
	}

	eoj, err := ParseEOJString(parts[1])
	if err != nil {
		return IPAndEOJ{}, err
	}

	return IPAndEOJ{IP: ip, EOJ: eoj}, nil
}

// FindPropertyAlias finds a property by its alias name for a given class code.
// This is a wrapper around PropertyTables.FindAlias.
func FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool) {
	return echonet_lite.PropertyTables.FindAlias(classCode, alias)
}

// AvailablePropertyAliases returns a map of available property aliases for a given class code.
// This is a wrapper around PropertyTables.AvailableAliases.
func AvailablePropertyAliases(classCode EOJClassCode) map[string]echonet_lite.PropertyDescription {
	return echonet_lite.PropertyTables.AvailableAliases(classCode)
}

// IsPropertyDefaultEPC checks if a property is a default property for a given class code.
// This is already implemented in Property.go, but included here for completeness.
// func IsPropertyDefaultEPC(classCode EOJClassCode, epc EPCType) bool {
//     return IsPropertyDefaultEPC(classCode, epc)
// }

// IDString は EOJ と IdentificationNumber の組み合わせを表す文字列型です
type IDString string

// MakeIDString は EOJ と IdentificationNumber から IDString を生成します
// 形式は "${EOJ.IDString()}:${identificationNumber.String()}" です
func MakeIDString(eoj EOJ, id echonet_lite.IdentificationNumber) IDString {
	return IDString(fmt.Sprintf("%s:%s", eoj.IDString(), id.String()))
}

func DeviceSpecifierFromIPAndEOJ(device IPAndEOJ) DeviceSpecifier {
	classCode := device.EOJ.ClassCode()
	instanceCode := device.EOJ.InstanceCode()
	return DeviceSpecifier{
		IP:           &device.IP,
		ClassCode:    &classCode,
		InstanceCode: &instanceCode,
	}
}

func GenerateUniqueIdentifierFromMACAddress() ([]byte, error) {
	uniqueIdentifier := make([]byte, 13) // 13バイトのユニーク識別子を生成

	localIPs, err := network.GetLocalIPv4s()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IPv4 address: %v", err)
	}
	if len(localIPs) > 0 {
		// 最初のローカルIPアドレスを使用
		macAddr, macErr := network.GetMACAddressByIP(localIPs[0])
		if macErr != nil {
			return nil, fmt.Errorf("failed to get MAC address for %s: %v", localIPs[0].String(), macErr)
		}
		if len(macAddr) == 6 {
			// MACアドレスが6バイトの場合、先頭6バイトにコピー
			copy(uniqueIdentifier, macAddr)
			// 残りの7バイトは0のまま
		} else {
			return nil, fmt.Errorf("unexpected MAC address length (%d bytes) for %s: %v", len(macAddr), localIPs[0].String(), macAddr)
		}
	}
	return uniqueIdentifier, nil
}
