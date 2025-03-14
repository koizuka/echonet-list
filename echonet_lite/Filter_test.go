package echonet_lite

import (
	"bytes"
	"net"
	"testing"
)

func TestFilter(t *testing.T) {
	// Define IP addresses
	ip1 := net.ParseIP("192.168.1.1")
	ip2 := net.ParseIP("192.168.1.2")

	// Define constants for test data
	const (
		// Instance codes
		instanceCode1 = 0x01
		instanceCode2 = 0x02 // 別のインスタンスコード（フィルタリングテスト用）

		// EPC codes
		epcOperationStatus      = 0x80 // Operation status
		epcInstallationLocation = 0x81 // Installation location
		epcOperationMode        = 0xB0 // Operation mode setting (for air conditioner)
		epcLightLevel           = 0xB6 // Light level setting (for lighting)
	)

	// EDT values
	var (
		edtOn          = []byte{0x30} // ON
		edtOff         = []byte{0x31} // OFF
		edtLivingRoom  = []byte{0x01} // Living room
		edtDiningRoom  = []byte{0x02} // Dining room
		edtCooling     = []byte{0x42} // Cooling
		edt100Percent  = []byte{0x64} // 100%
		edtNonExistent = []byte{0xFF} // 存在しないEDT値（除外テスト用）
	)

	// Create a Devices instance with test data
	devices := NewDevices()

	// Create test EOJs and Properties
	eoj1 := MakeEOJ(HomeAirConditioner_ClassCode, EOJInstanceCode(instanceCode1))
	eoj2 := MakeEOJ(SingleFunctionLighting_ClassCode, EOJInstanceCode(instanceCode1))
	eoj3 := MakeEOJ(HomeAirConditioner_ClassCode, EOJInstanceCode(instanceCode2)) // インスタンスコード2のエアコン
	epc1 := EPCType(epcOperationStatus)
	epc2 := EPCType(epcInstallationLocation)
	epc3 := EPCType(epcOperationMode)
	epc4 := EPCType(epcLightLevel)

	// Properties for eoj1 (Air Conditioner)
	ac_property1 := Property{
		EPC: epc1,
		EDT: edtOn,
	}
	ac_property2 := Property{
		EPC: epc2,
		EDT: edtLivingRoom,
	}
	ac_property3 := Property{
		EPC: epc3,
		EDT: edtCooling,
	}

	// Properties for eoj2 (Lighting)
	light_property1 := Property{
		EPC: epc1,
		EDT: edtOff,
	}
	light_property2 := Property{
		EPC: epc2,
		EDT: edtDiningRoom,
	}
	light_property4 := Property{
		EPC: epc4,
		EDT: edt100Percent,
	}

	// Register the test properties
	devices.RegisterProperties(IPAndEOJ{ip1, eoj1}, []Property{ac_property1, ac_property2, ac_property3})
	devices.RegisterProperties(IPAndEOJ{ip1, eoj2}, []Property{light_property1, light_property2, light_property4})
	devices.RegisterProperties(IPAndEOJ{ip1, eoj3}, []Property{ac_property1, ac_property2, ac_property3}) // インスタンスコード2のデバイスも登録
	devices.RegisterProperties(IPAndEOJ{ip2, eoj1}, []Property{ac_property1, ac_property2, ac_property3})

	// For string representation in expected results
	ip1Str := ip1.String()
	ip2Str := ip2.String()

	// Define test cases
	tests := []struct {
		name            string
		criteria        FilterCriteria
		expectedDevices map[string][]EOJ             // 期待されるデバイス
		expectedProps   map[string]map[EOJ][]EPCType // 期待されるプロパティ
	}{
		{
			name: "Filter by EPCs only",
			criteria: FilterCriteria{
				EPCs: []EPCType{epc1},
			},
			expectedDevices: map[string][]EOJ{
				ip1Str: {eoj1, eoj2, eoj3},
				ip2Str: {eoj1},
			},
			expectedProps: map[string]map[EOJ][]EPCType{
				ip1Str: {
					eoj1: {epc1},
					eoj2: {epc1},
					eoj3: {epc1},
				},
				ip2Str: {
					eoj1: {epc1},
				},
			},
		},
		{
			name: "Filter by PropertyValues only",
			criteria: FilterCriteria{
				PropertyValues: []Property{
					{
						EPC: epc1,
						EDT: edtOn,
					},
				},
			},
			expectedDevices: map[string][]EOJ{
				ip1Str: {eoj1, eoj3},
				ip2Str: {eoj1},
			},
			expectedProps: map[string]map[EOJ][]EPCType{
				ip1Str: {
					eoj1: {epc1, epc2, epc3},
					eoj3: {epc1, epc2, epc3},
				},
				ip2Str: {
					eoj1: {epc1, epc2, epc3},
				},
			},
		},
		{
			name: "Filter by PropertyValues with non-existent EDT",
			criteria: FilterCriteria{
				PropertyValues: []Property{
					{
						EPC: epc1,
						EDT: edtNonExistent, // 存在しないEDT値
					},
				},
			},
			expectedDevices: map[string][]EOJ{},
			expectedProps:   map[string]map[EOJ][]EPCType{},
		},
		{
			name: "Filter by both EPCs and PropertyValues",
			criteria: FilterCriteria{
				EPCs: []EPCType{epc1, epc2},
				PropertyValues: []Property{
					{
						EPC: epc1,
						EDT: edtOn,
					},
				},
			},
			expectedDevices: map[string][]EOJ{
				ip1Str: {eoj1, eoj3},
				ip2Str: {eoj1},
			},
			expectedProps: map[string]map[EOJ][]EPCType{
				ip1Str: {
					eoj1: {epc1, epc2},
					eoj3: {epc1, epc2},
				},
				ip2Str: {
					eoj1: {epc1, epc2},
				},
			},
		},
		{
			name: "Filter by IP address",
			criteria: FilterCriteria{
				Device: &DeviceSpecifier{IPAddress: &ip1},
			},
			expectedDevices: map[string][]EOJ{
				ip1Str: {eoj1, eoj2, eoj3},
			},
			expectedProps: map[string]map[EOJ][]EPCType{
				ip1Str: {
					eoj1: {epc1, epc2, epc3},
					eoj2: {epc1, epc2, epc4},
					eoj3: {epc1, epc2, epc3},
				},
			},
		},
		{
			name: "Filter by class code",
			criteria: FilterCriteria{
				Device: &DeviceSpecifier{ClassCode: ptr(HomeAirConditioner_ClassCode)},
			},
			expectedDevices: map[string][]EOJ{
				ip1Str: {eoj1, eoj3},
				ip2Str: {eoj1},
			},
			expectedProps: map[string]map[EOJ][]EPCType{
				ip1Str: {
					eoj1: {epc1, epc2, epc3},
					eoj3: {epc1, epc2, epc3},
				},
				ip2Str: {
					eoj1: {epc1, epc2, epc3},
				},
			},
		},
		{
			name: "Filter by instance code",
			criteria: FilterCriteria{
				Device: &DeviceSpecifier{InstanceCode: ptr(EOJInstanceCode(instanceCode1))},
			},
			expectedDevices: map[string][]EOJ{
				ip1Str: {eoj1, eoj2}, // eoj3はinstanceCode2なので含まれない
				ip2Str: {eoj1},
			},
			expectedProps: map[string]map[EOJ][]EPCType{
				ip1Str: {
					eoj1: {epc1, epc2, epc3},
					eoj2: {epc1, epc2, epc4},
				},
				ip2Str: {
					eoj1: {epc1, epc2, epc3},
				},
			},
		},
		{
			name: "Filter by instance code 2",
			criteria: FilterCriteria{
				Device: &DeviceSpecifier{InstanceCode: ptr(EOJInstanceCode(instanceCode2))},
			},
			expectedDevices: map[string][]EOJ{
				ip1Str: {eoj3}, // instanceCode2のデバイスのみ
			},
			expectedProps: map[string]map[EOJ][]EPCType{
				ip1Str: {
					eoj3: {epc1, epc2, epc3},
				},
			},
		},
		{
			name: "Complex filter: IP + Class + EPCs",
			criteria: FilterCriteria{
				Device: &DeviceSpecifier{
					IPAddress: &ip1,
					ClassCode: ptr(SingleFunctionLighting_ClassCode),
				},
				EPCs: []EPCType{epc1, epc4},
			},
			expectedDevices: map[string][]EOJ{
				ip1Str: {eoj2},
			},
			expectedProps: map[string]map[EOJ][]EPCType{
				ip1Str: {
					eoj2: {epc1, epc4},
				},
			},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := devices.Filter(tt.criteria)

			// Check if all expected devices are in the filtered result
			for ip, expectedEOJs := range tt.expectedDevices {
				ipAddr := net.ParseIP(ip)
				for _, eoj := range expectedEOJs {
					device := IPAndEOJ{ipAddr, eoj}
					if !filtered.IsKnownDevice(device) {
						t.Errorf("Expected device %v to exist in filtered result, but it doesn't", device)
					}

					// Check if all expected properties are in the filtered result
					if expectedProps, ok := tt.expectedProps[ip][eoj]; ok {
						for _, epc := range expectedProps {
							prop, exists := filtered.GetProperty(device, epc)
							if !exists {
								t.Errorf("Expected property with EPC %v to exist for device %v, but it doesn't", epc, device)
							}

							// Get the original property to compare
							originalProp, _ := devices.GetProperty(device, epc)
							if originalProp != nil && prop != nil && !bytes.Equal(prop.EDT, originalProp.EDT) {
								t.Errorf("Property EDT mismatch for device %v, EPC %v: got %v, want %v",
									device, epc, prop.EDT, originalProp.EDT)
							}
						}

						// Check if there are no unexpected properties
						for epc := range filtered.data[ip][eoj] {
							found := false
							for _, expectedEPC := range expectedProps {
								if epc == expectedEPC {
									found = true
									break
								}
							}
							if !found {
								t.Errorf("Unexpected property with EPC %v found for device %v", epc, device)
							}
						}
					}
				}
			}

			// Check if there are no unexpected devices
			for ip, eojMap := range filtered.data {
				for eoj := range eojMap {
					found := false
					if expectedEOJs, ok := tt.expectedDevices[ip]; ok {
						for _, expectedEOJ := range expectedEOJs {
							if eoj == expectedEOJ {
								found = true
								break
							}
						}
					}
					if !found {
						t.Errorf("Unexpected device with IP %s and EOJ %v found in filtered result", ip, eoj)
					}
				}
			}

		})
	}
}

// Helper function for creating pointers
func ptr[T any](v T) *T {
	return &v
}
