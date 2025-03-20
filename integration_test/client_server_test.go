package integration_test

import (
	"echonet-list/protocol"
	"encoding/json"
	"github.com/google/uuid"
	"testing"
)

// TestProtocolMessageMarshaling tests marshaling/unmarshaling of protocol messages
func TestProtocolMessageMarshaling(t *testing.T) {
	t.Run("CommandMessage", func(t *testing.T) {
		// Create a device specifier
		ipStr := "192.168.1.100"
		classCode := protocol.ClassCode(0x0130)
		instanceCode := uint8(1)
		
		deviceSpec := protocol.DeviceSpecifier{
			IP:           &ipStr,
			ClassCode:    &classCode,
			InstanceCode: &instanceCode,
		}
		
		// Create EPCs
		epcs := []protocol.EPCType{0x80, 0xB0}
		
		// Create command message
		cmd := protocol.CommandMessage{
			Message: protocol.Message{
				Type: "command",
				ID:   uuid.New().String(),
			},
			Command:    "get",
			DeviceSpec: deviceSpec,
			EPCs:       epcs,
		}
		
		// Marshal to JSON
		jsonData, err := json.Marshal(cmd)
		if err != nil {
			t.Fatalf("Failed to marshal command message: %v", err)
		}
		
		// Unmarshal from JSON
		var decodedCmd protocol.CommandMessage
		if err := json.Unmarshal(jsonData, &decodedCmd); err != nil {
			t.Fatalf("Failed to unmarshal command message: %v", err)
		}
		
		// Verify fields
		if decodedCmd.Type != "command" {
			t.Errorf("Type = %v, want %v", decodedCmd.Type, "command")
		}
		
		if decodedCmd.Command != "get" {
			t.Errorf("Command = %v, want %v", decodedCmd.Command, "get")
		}
		
		// Verify device spec
		if specMap, ok := decodedCmd.DeviceSpec.(map[string]interface{}); ok {
			// Check IP
			if ip, ok := specMap["ip"].(string); !ok || ip != ipStr {
				t.Errorf("DeviceSpec.IP = %v, want %v", specMap["ip"], ipStr)
			}
			
			// Check ClassCode (comes as string in hex format)
			if cc, ok := specMap["classCode"].(string); !ok || cc != "0130" {
				t.Errorf("DeviceSpec.ClassCode = %v, want %v", specMap["classCode"], "0130")
			}
			
			// Check InstanceCode
			if ic, ok := specMap["instanceCode"].(float64); !ok || int(ic) != int(instanceCode) {
				t.Errorf("DeviceSpec.InstanceCode = %v, want %v", specMap["instanceCode"], instanceCode)
			}
		} else {
			t.Errorf("DeviceSpec is not a map: %T", decodedCmd.DeviceSpec)
		}
		
		// Verify EPCs
		if len(decodedCmd.EPCs) != len(epcs) {
			t.Errorf("EPCs length = %v, want %v", len(decodedCmd.EPCs), len(epcs))
		} else {
			for i, epc := range decodedCmd.EPCs {
				if epc != epcs[i] {
					t.Errorf("EPCs[%d] = %v, want %v", i, epc, epcs[i])
				}
			}
		}
	})
	
	t.Run("ResponseMessage", func(t *testing.T) {
		// Create device info
		deviceInfo := protocol.DeviceInfo{
			IP: "192.168.1.100",
			EOJ: protocol.EOJInfo{
				ClassCode:    0x0130,
				InstanceCode: 0x01,
			},
			Aliases: []string{"aircon1"},
		}
		
		// Create property info
		propInfo := protocol.PropertyInfo{
			EPC:  0x80,
			EDT:  protocol.ByteArray{0x30},
			Name: "Operation status",
			Value: "ON",
		}
		
		// Create device property result
		result := protocol.DevicePropertyResult{
			Device:     deviceInfo,
			Properties: []protocol.PropertyInfo{propInfo},
			Success:    true,
		}
		
		// Create response message
		resp := protocol.ResponseMessage{
			Message: protocol.Message{
				Type: "response",
				ID:   uuid.New().String(),
			},
			Success: true,
			Data:    result,
		}
		
		// Marshal to JSON
		jsonData, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Failed to marshal response message: %v", err)
		}
		
		// Unmarshal from JSON
		var decodedResp protocol.ResponseMessage
		if err := json.Unmarshal(jsonData, &decodedResp); err != nil {
			t.Fatalf("Failed to unmarshal response message: %v", err)
		}
		
		// Verify fields
		if decodedResp.Type != "response" {
			t.Errorf("Type = %v, want %v", decodedResp.Type, "response")
		}
		
		if decodedResp.Success != true {
			t.Errorf("Success = %v, want %v", decodedResp.Success, true)
		}
		
		// Data is a map[string]interface{} after unmarshaling generic JSON
		if decodedResp.Data == nil {
			t.Errorf("Data is nil")
		}
	})
	
	t.Run("NotificationMessage", func(t *testing.T) {
		// Create device info
		deviceInfo := protocol.DeviceInfo{
			IP: "192.168.1.100",
			EOJ: protocol.EOJInfo{
				ClassCode:    0x0130,
				InstanceCode: 0x01,
			},
			Aliases: []string{"aircon1"},
		}
		
		// Create notification message
		notif := protocol.NotificationMessage{
			Message: protocol.Message{
				Type: "notification",
				ID:   uuid.New().String(),
			},
			Event:      "deviceAdded",
			DeviceInfo: deviceInfo,
		}
		
		// Marshal to JSON
		jsonData, err := json.Marshal(notif)
		if err != nil {
			t.Fatalf("Failed to marshal notification message: %v", err)
		}
		
		// Unmarshal from JSON
		var decodedNotif protocol.NotificationMessage
		if err := json.Unmarshal(jsonData, &decodedNotif); err != nil {
			t.Fatalf("Failed to unmarshal notification message: %v", err)
		}
		
		// Verify fields
		if decodedNotif.Type != "notification" {
			t.Errorf("Type = %v, want %v", decodedNotif.Type, "notification")
		}
		
		if decodedNotif.Event != "deviceAdded" {
			t.Errorf("Event = %v, want %v", decodedNotif.Event, "deviceAdded")
		}
		
		// DeviceInfo is a map[string]interface{} after unmarshaling generic JSON
		if decodedNotif.DeviceInfo == nil {
			t.Errorf("DeviceInfo is nil")
		}
	})
	
	t.Run("PropertyEncoding", func(t *testing.T) {
		// Test various property encodings
		propTests := []struct {
			name  string
			epc   protocol.EPCType
			edt   protocol.ByteArray
			value string
		}{
			{"OperationStatus_ON", 0x80, protocol.ByteArray{0x30}, "ON"},
			{"OperationStatus_OFF", 0x80, protocol.ByteArray{0x31}, "OFF"},
			{"Temperature", 0xB0, protocol.ByteArray{0x1E}, "30.0"},
		}
		
		for _, tt := range propTests {
			t.Run(tt.name, func(t *testing.T) {
				// Create property info
				propInfo := protocol.PropertyInfo{
					EPC:   tt.epc,
					EDT:   tt.edt,
					Name:  tt.name,
					Value: tt.value,
				}
				
				// Marshal to JSON
				jsonData, err := json.Marshal(propInfo)
				if err != nil {
					t.Fatalf("Failed to marshal property info: %v", err)
				}
				
				// Unmarshal from JSON
				var decodedProp protocol.PropertyInfo
				if err := json.Unmarshal(jsonData, &decodedProp); err != nil {
					t.Fatalf("Failed to unmarshal property info: %v", err)
				}
				
				// Verify fields
				if decodedProp.EPC != tt.epc {
					t.Errorf("EPC = %v, want %v", decodedProp.EPC, tt.epc)
				}
				
				if len(decodedProp.EDT) != len(tt.edt) {
					t.Errorf("EDT length = %v, want %v", len(decodedProp.EDT), len(tt.edt))
				} else {
					for i, b := range decodedProp.EDT {
						if b != tt.edt[i] {
							t.Errorf("EDT[%d] = %v, want %v", i, b, tt.edt[i])
						}
					}
				}
				
				if decodedProp.Name != tt.name {
					t.Errorf("Name = %v, want %v", decodedProp.Name, tt.name)
				}
				
				if decodedProp.Value != tt.value {
					t.Errorf("Value = %v, want %v", decodedProp.Value, tt.value)
				}
			})
		}
	})
}