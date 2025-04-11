package protocol

import (
	"echonet-list/echonet_lite"
	"encoding/base64"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestDeviceToProtocol(t *testing.T) {
	// Test cases
	tests := []struct {
		name       string
		ipAndEOJ   echonet_lite.IPAndEOJ
		properties echonet_lite.Properties
		lastSeen   time.Time
		want       Device
	}{
		{
			name: "Basic device conversion",
			ipAndEOJ: echonet_lite.IPAndEOJ{
				IP:  net.ParseIP("192.168.1.10"),
				EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
			},
			properties: echonet_lite.Properties{
				{EPC: 0x80, EDT: []byte{0x30}},
				{EPC: 0x81, EDT: []byte{0x01, 0x02}},
			},
			lastSeen: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			want: Device{
				IP:   "192.168.1.10",
				EOJ:  "0130:1",
				Name: "0130[Home air conditioner]",
				Properties: map[string]string{
					"80": base64.StdEncoding.EncodeToString([]byte{0x30}),
					"81": base64.StdEncoding.EncodeToString([]byte{0x01, 0x02}),
				},
				LastSeen: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Empty properties",
			ipAndEOJ: echonet_lite.IPAndEOJ{
				IP:  net.ParseIP("192.168.1.20"),
				EOJ: echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 1),
			},
			properties: echonet_lite.Properties{},
			lastSeen:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			want: Device{
				IP:         "192.168.1.20",
				EOJ:        "0EF0:1",
				Name:       "0EF0[Node profile]",
				Properties: map[string]string{},
				LastSeen:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeviceToProtocol(tt.ipAndEOJ, tt.properties, tt.lastSeen)

			// Check IP
			if got.IP != tt.want.IP {
				t.Errorf("DeviceToProtocol() IP = %v, want %v", got.IP, tt.want.IP)
			}

			// Check EOJ
			if got.EOJ != tt.want.EOJ {
				t.Errorf("DeviceToProtocol() EOJ = %v, want %v", got.EOJ, tt.want.EOJ)
			}

			// Check Name
			if got.Name != tt.want.Name {
				t.Errorf("DeviceToProtocol() Name = %v, want %v", got.Name, tt.want.Name)
			}

			// Check Properties
			if !reflect.DeepEqual(got.Properties, tt.want.Properties) {
				t.Errorf("DeviceToProtocol() Properties = %v, want %v", got.Properties, tt.want.Properties)
			}

			// Check LastSeen
			if !got.LastSeen.Equal(tt.want.LastSeen) {
				t.Errorf("DeviceToProtocol() LastSeen = %v, want %v", got.LastSeen, tt.want.LastSeen)
			}
		})
	}
}

func TestDeviceFromProtocol(t *testing.T) {
	// Test cases
	tests := []struct {
		name         string
		device       Device
		wantIPAndEOJ echonet_lite.IPAndEOJ
		wantErr      bool
	}{
		{
			name: "Basic device conversion",
			device: Device{
				IP:   "192.168.1.10",
				EOJ:  "0130:1",
				Name: "Home air conditioner",
				Properties: map[string]string{
					"80": base64.StdEncoding.EncodeToString([]byte{0x30}),
					"81": base64.StdEncoding.EncodeToString([]byte{0x01, 0x02}),
				},
				LastSeen: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			wantIPAndEOJ: echonet_lite.IPAndEOJ{
				IP:  net.ParseIP("192.168.1.10"),
				EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
			},
			wantErr: false,
		},
		{
			name: "Empty properties",
			device: Device{
				IP:         "192.168.1.20",
				EOJ:        "0EF0:1",
				Name:       "Node profile",
				Properties: map[string]string{},
				LastSeen:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			wantIPAndEOJ: echonet_lite.IPAndEOJ{
				IP:  net.ParseIP("192.168.1.20"),
				EOJ: echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 1),
			},
			wantErr: false,
		},
		{
			name: "Invalid IP",
			device: Device{
				IP:         "invalid-ip",
				EOJ:        "0130:1",
				Name:       "Home air conditioner",
				Properties: map[string]string{},
				LastSeen:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			wantIPAndEOJ: echonet_lite.IPAndEOJ{},
			wantErr:      true,
		},
		{
			name: "Invalid EOJ",
			device: Device{
				IP:         "192.168.1.10",
				EOJ:        "invalid-eoj",
				Name:       "Home air conditioner",
				Properties: map[string]string{},
				LastSeen:   time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			wantIPAndEOJ: echonet_lite.IPAndEOJ{},
			wantErr:      true,
		},
		{
			name: "Invalid property EPC",
			device: Device{
				IP:   "192.168.1.10",
				EOJ:  "0130:1",
				Name: "Home air conditioner",
				Properties: map[string]string{
					"invalid-epc": base64.StdEncoding.EncodeToString([]byte{0x30}),
				},
				LastSeen: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			wantIPAndEOJ: echonet_lite.IPAndEOJ{},
			wantErr:      true,
		},
		{
			name: "Invalid property EDT",
			device: Device{
				IP:   "192.168.1.10",
				EOJ:  "0130:1",
				Name: "Home air conditioner",
				Properties: map[string]string{
					"80": "invalid-base64",
				},
				LastSeen: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			wantIPAndEOJ: echonet_lite.IPAndEOJ{},
			wantErr:      true,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIPAndEOJ, gotProps, err := DeviceFromProtocol(tt.device)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("DeviceFromProtocol() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Check IPAndEOJ
			if !gotIPAndEOJ.IP.Equal(tt.wantIPAndEOJ.IP) {
				t.Errorf("DeviceFromProtocol() IP = %v, want %v", gotIPAndEOJ.IP, tt.wantIPAndEOJ.IP)
			}
			if gotIPAndEOJ.EOJ != tt.wantIPAndEOJ.EOJ {
				t.Errorf("DeviceFromProtocol() EOJ = %v, want %v", gotIPAndEOJ.EOJ, tt.wantIPAndEOJ.EOJ)
			}

			// Check Properties
			for _, prop := range gotProps {
				epcStr := fmt.Sprintf("%02X", byte(prop.EPC))
				wantEDTBase64 := tt.device.Properties[epcStr]
				wantEDT, _ := base64.StdEncoding.DecodeString(wantEDTBase64)

				if !reflect.DeepEqual(prop.EDT, wantEDT) {
					t.Errorf("DeviceFromProtocol() Properties[%s] = %v, want %v", epcStr, prop.EDT, wantEDT)
				}
			}
		})
	}
}

// Test round-trip conversion
func TestDeviceRoundTrip(t *testing.T) {
	// Create a test device
	ipAndEOJ := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP("192.168.1.10"),
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}
	properties := echonet_lite.Properties{
		{EPC: 0x80, EDT: []byte{0x30}},
		{EPC: 0x81, EDT: []byte{0x01, 0x02}},
	}
	lastSeen := time.Now()

	// Convert to protocol Device
	protoDevice := DeviceToProtocol(ipAndEOJ, properties, lastSeen)

	// Convert back to ECHONET Lite types
	gotIPAndEOJ, gotProps, err := DeviceFromProtocol(protoDevice)

	// Check for errors
	if err != nil {
		t.Errorf("Round-trip conversion failed with error: %v", err)
		return
	}

	// Check IPAndEOJ
	if !gotIPAndEOJ.IP.Equal(ipAndEOJ.IP) {
		t.Errorf("Round-trip IP = %v, want %v", gotIPAndEOJ.IP, ipAndEOJ.IP)
	}
	if gotIPAndEOJ.EOJ != ipAndEOJ.EOJ {
		t.Errorf("Round-trip EOJ = %v, want %v", gotIPAndEOJ.EOJ, ipAndEOJ.EOJ)
	}

	// Check Properties
	for _, prop := range gotProps {
		originalProp, found := properties.FindEPC(prop.EPC)
		if !found {
			t.Errorf("Round-trip Properties contains unexpected EPC: %X", prop.EPC)
			continue
		}

		if !reflect.DeepEqual(prop.EDT, originalProp.EDT) {
			t.Errorf("Round-trip Properties[%X] = %v, want %v", prop.EPC, prop.EDT, originalProp.EDT)
		}
	}
}
