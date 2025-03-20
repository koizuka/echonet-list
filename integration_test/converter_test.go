package integration_test

import (
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
	"net"
	"reflect"
	"testing"
)

// TestTypeConversion tests conversion between protocol and ECHONET Lite types
func TestTypeConversion(t *testing.T) {
	t.Run("IPAndEOJ", func(t *testing.T) {
		// Create IP and EOJ for test
		ip := net.ParseIP("192.168.1.100")
		eoj := echonet_lite.MakeEOJ(0x0130, 0x01) // Air conditioner, instance 1
		aliases := []string{"aircon1", "living"}
		
		// Create IPAndEOJ
		ipAndEOJ := echonet_lite.IPAndEOJ{
			IP:  ip,
			EOJ: eoj,
		}
		
		// Convert to DeviceInfo
		deviceInfo := protocol.ConvertIPAndEOJToDeviceInfo(ipAndEOJ, aliases)
		
		// Verify conversion
		if deviceInfo.IP != "192.168.1.100" {
			t.Errorf("IP = %v, want %v", deviceInfo.IP, "192.168.1.100")
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
		
		// Convert back to IPAndEOJ
		convertedIPAndEOJ, err := protocol.ConvertDeviceInfoToIPAndEOJ(deviceInfo)
		if err != nil {
			t.Fatalf("Failed to convert DeviceInfo to IPAndEOJ: %v", err)
		}
		
		// Verify round trip conversion
		if !convertedIPAndEOJ.IP.Equal(ipAndEOJ.IP) {
			t.Errorf("IP = %v, want %v", convertedIPAndEOJ.IP, ipAndEOJ.IP)
		}
		
		if convertedIPAndEOJ.EOJ.ClassCode() != ipAndEOJ.EOJ.ClassCode() {
			t.Errorf("ClassCode = %v, want %v", convertedIPAndEOJ.EOJ.ClassCode(), ipAndEOJ.EOJ.ClassCode())
		}
		
		if convertedIPAndEOJ.EOJ.InstanceCode() != ipAndEOJ.EOJ.InstanceCode() {
			t.Errorf("InstanceCode = %v, want %v", convertedIPAndEOJ.EOJ.InstanceCode(), ipAndEOJ.EOJ.InstanceCode())
		}
	})
	
	t.Run("Property", func(t *testing.T) {
		// Create Property for test
		echonetProp := echonet_lite.Property{
			EPC: 0x80, // Operation status
			EDT: []byte{0x30}, // ON
		}
		
		// Convert to PropertyInfo (using air conditioner class code)
		propInfo := protocol.ConvertPropertyToPropertyInfo(echonetProp, 0x0130)
		
		// Verify conversion
		if propInfo.EPC != 0x80 {
			t.Errorf("EPC = %v, want %v", propInfo.EPC, 0x80)
		}
		
		if !reflect.DeepEqual(propInfo.EDT, protocol.ByteArray{0x30}) {
			t.Errorf("EDT = %v, want %v", propInfo.EDT, protocol.ByteArray{0x30})
		}
		
		// Convert back to Property
		convertedProp := protocol.ConvertPropertyInfoToProperty(propInfo)
		
		// Verify round trip conversion
		if convertedProp.EPC != echonetProp.EPC {
			t.Errorf("EPC = %v, want %v", convertedProp.EPC, echonetProp.EPC)
		}
		
		if !reflect.DeepEqual(convertedProp.EDT, echonetProp.EDT) {
			t.Errorf("EDT = %v, want %v", convertedProp.EDT, echonetProp.EDT)
		}
	})
	
	t.Run("DeviceSpecifier", func(t *testing.T) {
		// Create test values
		ipStr := "192.168.1.100"
		classCode := protocol.ClassCode(0x0130)
		instanceCode := uint8(1)
		
		// Create protocol.DeviceSpecifier
		protocolSpec := protocol.DeviceSpecifier{
			IP:           &ipStr,
			ClassCode:    &classCode,
			InstanceCode: &instanceCode,
		}
		
		// Convert to echonet_lite.DeviceSpecifier
		echonetSpec := protocol.ConvertDeviceSpecifierToEchonetDeviceSpecifier(protocolSpec)
		
		// Verify conversion
		expectedIP := net.ParseIP("192.168.1.100")
		if !echonetSpec.IP.Equal(expectedIP) {
			t.Errorf("IP = %v, want %v", echonetSpec.IP, expectedIP)
		}
		
		if *echonetSpec.ClassCode != 0x0130 {
			t.Errorf("ClassCode = %v, want %v", *echonetSpec.ClassCode, 0x0130)
		}
		
		if *echonetSpec.InstanceCode != 0x01 {
			t.Errorf("InstanceCode = %v, want %v", *echonetSpec.InstanceCode, 0x01)
		}
		
		// Convert back to protocol.DeviceSpecifier
		convertedSpec := protocol.ConvertEchonetDeviceSpecifierToDeviceSpecifier(echonetSpec)
		
		// Verify round trip conversion
		if *convertedSpec.IP != ipStr {
			t.Errorf("IP = %v, want %v", *convertedSpec.IP, ipStr)
		}
		
		if *convertedSpec.ClassCode != classCode {
			t.Errorf("ClassCode = %v, want %v", *convertedSpec.ClassCode, classCode)
		}
		
		if *convertedSpec.InstanceCode != instanceCode {
			t.Errorf("InstanceCode = %v, want %v", *convertedSpec.InstanceCode, instanceCode)
		}
	})
}