package protocol

import (
	"echonet-list/echonet_lite"
	"net"
	"reflect"
	"testing"
)

func TestConvertIPAndEOJToDeviceInfo(t *testing.T) {
	// Create a test IPAndEOJ
	ip := net.ParseIP("192.168.1.5")
	eoj := echonet_lite.MakeEOJ(0x0130, 0x01) // Air conditioner, instance 1
	ipAndEOJ := echonet_lite.IPAndEOJ{
		IP:  ip,
		EOJ: eoj,
	}
	aliases := []string{"aircon1", "living_ac"}

	// Convert to DeviceInfo
	deviceInfo := ConvertIPAndEOJToDeviceInfo(ipAndEOJ, aliases)

	// Verify conversion results
	if deviceInfo.IP != "192.168.1.5" {
		t.Errorf("IP = %v, want %v", deviceInfo.IP, "192.168.1.5")
	}

	if deviceInfo.EOJ.ClassCode != 0x0130 {
		t.Errorf("ClassCode = %v, want %v", deviceInfo.EOJ.ClassCode, 0x0130)
	}

	if deviceInfo.EOJ.InstanceCode != 0x01 {
		t.Errorf("InstanceCode = %v, want %v", deviceInfo.EOJ.InstanceCode, 0x01)
	}

	if !reflect.DeepEqual(deviceInfo.Aliases, aliases) {
		t.Errorf("Aliases = %v, want %v", deviceInfo.Aliases, aliases)
	}
}

func TestConvertDeviceInfoToIPAndEOJ(t *testing.T) {
	// Create a test DeviceInfo
	deviceInfo := DeviceInfo{
		IP: "192.168.1.5",
		EOJ: EOJInfo{
			ClassCode:    0x0130,
			InstanceCode: 0x01,
		},
		Aliases: []string{"aircon1", "living_ac"},
	}

	// Convert to IPAndEOJ
	ipAndEOJ, err := ConvertDeviceInfoToIPAndEOJ(deviceInfo)
	if err != nil {
		t.Fatalf("Failed to convert DeviceInfo to IPAndEOJ: %v", err)
	}

	// Verify conversion results
	expectedIP := net.ParseIP("192.168.1.5")
	if !ipAndEOJ.IP.Equal(expectedIP) {
		t.Errorf("IP = %v, want %v", ipAndEOJ.IP, expectedIP)
	}

	if ipAndEOJ.EOJ.ClassCode() != 0x0130 {
		t.Errorf("ClassCode = %v, want %v", ipAndEOJ.EOJ.ClassCode(), 0x0130)
	}

	if ipAndEOJ.EOJ.InstanceCode() != 0x01 {
		t.Errorf("InstanceCode = %v, want %v", ipAndEOJ.EOJ.InstanceCode(), 0x01)
	}
}

func TestConvertDeviceInfoToIPAndEOJ_InvalidIP(t *testing.T) {
	// Create a test DeviceInfo with invalid IP
	deviceInfo := DeviceInfo{
		IP: "invalid-ip",
		EOJ: EOJInfo{
			ClassCode:    0x0130,
			InstanceCode: 0x01,
		},
	}

	// Attempt conversion
	_, err := ConvertDeviceInfoToIPAndEOJ(deviceInfo)
	
	// Should return error
	if err != ErrInvalidIP {
		t.Errorf("Expected error %v, got %v", ErrInvalidIP, err)
	}
}

func TestConvertPropertyToPropertyInfo(t *testing.T) {
	// Create a test Property (e.g. Operation status ON)
	property := echonet_lite.Property{
		EPC: 0x80,
		EDT: []byte{0x30},
	}
	classCode := echonet_lite.EOJClassCode(0x0130) // Air conditioner

	// Convert to PropertyInfo
	propInfo := ConvertPropertyToPropertyInfo(property, classCode)

	// Verify conversion results
	if propInfo.EPC != 0x80 {
		t.Errorf("EPC = %v, want %v", propInfo.EPC, 0x80)
	}

	if !reflect.DeepEqual(propInfo.EDT, ByteArray{0x30}) {
		t.Errorf("EDT = %v, want %v", propInfo.EDT, ByteArray{0x30})
	}

	// Note: Name and Value may be empty if property info is not registered
	// in the ECHONET Lite standard data - this is implementation dependent
}

func TestConvertPropertyInfoToProperty(t *testing.T) {
	// Create a test PropertyInfo
	propInfo := PropertyInfo{
		EPC:  0x80,
		EDT:  ByteArray{0x30},
		Name: "Operation status",
	}

	// Convert to Property
	property := ConvertPropertyInfoToProperty(propInfo)

	// Verify conversion results
	if property.EPC != 0x80 {
		t.Errorf("EPC = %v, want %v", property.EPC, 0x80)
	}

	if !reflect.DeepEqual(property.EDT, []byte{0x30}) {
		t.Errorf("EDT = %v, want %v", property.EDT, []byte{0x30})
	}
}

func TestConvertDeviceSpecifierToEchonetDeviceSpecifier(t *testing.T) {
	// Create test values
	ipStr := "192.168.1.5"
	classCode := ClassCode(0x0130)
	instanceCode := uint8(0x01)

	// Create a test DeviceSpecifier
	deviceSpec := DeviceSpecifier{
		IP:           &ipStr,
		ClassCode:    &classCode,
		InstanceCode: &instanceCode,
	}

	// Convert to echonet_lite.DeviceSpecifier
	echonetSpec := ConvertDeviceSpecifierToEchonetDeviceSpecifier(deviceSpec)

	// Verify conversion results
	expectedIP := net.ParseIP("192.168.1.5")
	if !echonetSpec.IP.Equal(expectedIP) {
		t.Errorf("IP = %v, want %v", echonetSpec.IP, expectedIP)
	}

	if *echonetSpec.ClassCode != 0x0130 {
		t.Errorf("ClassCode = %v, want %v", *echonetSpec.ClassCode, 0x0130)
	}

	if *echonetSpec.InstanceCode != 0x01 {
		t.Errorf("InstanceCode = %v, want %v", *echonetSpec.InstanceCode, 0x01)
	}
}

func TestConvertEchonetDeviceSpecifierToDeviceSpecifier(t *testing.T) {
	// Create test values
	ip := net.ParseIP("192.168.1.5")
	classCode := echonet_lite.EOJClassCode(0x0130)
	instanceCode := echonet_lite.EOJInstanceCode(0x01)

	// Create a test echonet_lite.DeviceSpecifier
	echonetSpec := echonet_lite.DeviceSpecifier{
		IP:           &ip,
		ClassCode:    &classCode,
		InstanceCode: &instanceCode,
	}

	// Convert to DeviceSpecifier
	deviceSpec := ConvertEchonetDeviceSpecifierToDeviceSpecifier(echonetSpec)

	// Verify conversion results
	if *deviceSpec.IP != "192.168.1.5" {
		t.Errorf("IP = %v, want %v", *deviceSpec.IP, "192.168.1.5")
	}

	if *deviceSpec.ClassCode != 0x0130 {
		t.Errorf("ClassCode = %v, want %v", *deviceSpec.ClassCode, 0x0130)
	}

	if *deviceSpec.InstanceCode != 0x01 {
		t.Errorf("InstanceCode = %v, want %v", *deviceSpec.InstanceCode, 0x01)
	}
}

func TestPropertyConversion(t *testing.T) {
	// Create a test Protocol Property
	protoProp := Property{
		EPC: 0x80,
		EDT: ByteArray{0x30},
	}

	// Convert to echonet_lite.Property
	echonetProp := ConvertPropertyToEchonetProperty(protoProp)

	// Verify conversion results
	if echonetProp.EPC != 0x80 {
		t.Errorf("EPC = %v, want %v", echonetProp.EPC, 0x80)
	}

	if !reflect.DeepEqual(echonetProp.EDT, []byte{0x30}) {
		t.Errorf("EDT = %v, want %v", echonetProp.EDT, []byte{0x30})
	}

	// Convert back to Protocol Property
	reconvertedProp := ConvertEchonetPropertyToProperty(echonetProp)

	// Verify round-trip conversion
	if reconvertedProp.EPC != protoProp.EPC {
		t.Errorf("Reconverted EPC = %v, want %v", reconvertedProp.EPC, protoProp.EPC)
	}

	if !reflect.DeepEqual(reconvertedProp.EDT, protoProp.EDT) {
		t.Errorf("Reconverted EDT = %v, want %v", reconvertedProp.EDT, protoProp.EDT)
	}
}