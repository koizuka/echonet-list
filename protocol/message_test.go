package protocol

import (
	"encoding/json"
	"testing"
)

func TestEPCTypeMarshaling(t *testing.T) {
	tests := []struct {
		name string
		epc  EPCType
		want string
	}{
		{"Zero", EPCType(0x00), `"00"`},
		{"Random", EPCType(0x80), `"80"`},
		{"Max", EPCType(0xFF), `"FF"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			bytes, err := json.Marshal(tt.epc)
			if err != nil {
				t.Fatalf("Failed to marshal EPCType: %v", err)
			}
			got := string(bytes)
			if got != tt.want {
				t.Errorf("MarshalJSON() = %v, want %v", got, tt.want)
			}

			// Unmarshal
			var epc EPCType
			if err := json.Unmarshal([]byte(tt.want), &epc); err != nil {
				t.Fatalf("Failed to unmarshal EPCType: %v", err)
			}
			if epc != tt.epc {
				t.Errorf("UnmarshalJSON() = %v, want %v", epc, tt.epc)
			}
		})
	}
}

func TestByteArrayMarshaling(t *testing.T) {
	tests := []struct {
		name  string
		bytes ByteArray
		want  string
	}{
		{"Empty", ByteArray{}, `""`},
		{"SingleByte", ByteArray{0x01}, `"01"`},
		{"MultiByte", ByteArray{0x01, 0x02, 0x03}, `"010203"`},
		{"Null", nil, `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			bytes, err := json.Marshal(tt.bytes)
			if err != nil {
				t.Fatalf("Failed to marshal ByteArray: %v", err)
			}
			got := string(bytes)
			if got != tt.want {
				t.Errorf("MarshalJSON() = %v, want %v", got, tt.want)
			}

			// Unmarshal
			var ba ByteArray
			if err := json.Unmarshal([]byte(tt.want), &ba); err != nil {
				t.Fatalf("Failed to unmarshal ByteArray: %v", err)
			}
			
			// Compare original and unmarshaled
			if len(ba) != len(tt.bytes) {
				t.Errorf("UnmarshalJSON() length = %v, want %v", len(ba), len(tt.bytes))
				return
			}
			
			for i := range ba {
				if i < len(tt.bytes) && ba[i] != tt.bytes[i] {
					t.Errorf("UnmarshalJSON()[%d] = %v, want %v", i, ba[i], tt.bytes[i])
				}
			}
		})
	}
}

func TestClassCodeMarshaling(t *testing.T) {
	tests := []struct {
		name      string
		classCode ClassCode
		want      string
	}{
		{"Zero", ClassCode(0x0000), `"0000"`},
		{"Random", ClassCode(0x0EF0), `"0EF0"`},
		{"Max", ClassCode(0xFFFF), `"FFFF"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			bytes, err := json.Marshal(tt.classCode)
			if err != nil {
				t.Fatalf("Failed to marshal ClassCode: %v", err)
			}
			got := string(bytes)
			if got != tt.want {
				t.Errorf("MarshalJSON() = %v, want %v", got, tt.want)
			}

			// Unmarshal
			var cc ClassCode
			if err := json.Unmarshal([]byte(tt.want), &cc); err != nil {
				t.Fatalf("Failed to unmarshal ClassCode: %v", err)
			}
			if cc != tt.classCode {
				t.Errorf("UnmarshalJSON() = %v, want %v", cc, tt.classCode)
			}
		})
	}
}

func TestCommandMessageMarshaling(t *testing.T) {
	// Create test EPCs
	epcs := []EPCType{0x80, 0xB0}
	
	// Create a sample device specifier
	alias := "testDevice"
	deviceSpec := DeviceSpecifier{
		Alias: &alias,
	}
	
	// Create command message
	cmd := CommandMessage{
		Message: Message{
			Type: "command",
			ID:   "123456",
		},
		Command:    "get",
		DeviceSpec: deviceSpec,
		EPCs:       epcs,
	}
	
	// Marshal to JSON
	jsonBytes, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("Failed to marshal CommandMessage: %v", err)
	}
	
	// Unmarshal from JSON
	var decodedCmd CommandMessage
	if err := json.Unmarshal(jsonBytes, &decodedCmd); err != nil {
		t.Fatalf("Failed to unmarshal CommandMessage: %v", err)
	}
	
	// Verify basic fields
	if decodedCmd.Type != "command" {
		t.Errorf("Type = %v, want %v", decodedCmd.Type, "command")
	}
	
	if decodedCmd.ID != "123456" {
		t.Errorf("ID = %v, want %v", decodedCmd.ID, "123456")
	}
	
	if decodedCmd.Command != "get" {
		t.Errorf("Command = %v, want %v", decodedCmd.Command, "get")
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
	
	// Verify device spec (needs type assertion)
	if spec, ok := decodedCmd.DeviceSpec.(map[string]interface{}); ok {
		if aliasVal, ok := spec["alias"].(string); !ok || aliasVal != alias {
			t.Errorf("DeviceSpec.Alias = %v, want %v", spec["alias"], alias)
		}
	} else {
		t.Errorf("DeviceSpec type assertion failed, got type %T", decodedCmd.DeviceSpec)
	}
}

func TestResponseMessageMarshaling(t *testing.T) {
	// Create sample property info
	propInfo := PropertyInfo{
		EPC:   0x80,
		EDT:   ByteArray{0x30},
		Name:  "Operation status",
		Value: "ON",
	}
	
	// Create device info
	deviceInfo := DeviceInfo{
		IP: "192.168.1.5",
		EOJ: EOJInfo{
			ClassCode:    0x0130,
			InstanceCode: 0x01,
		},
		Aliases: []string{"aircon1"},
	}
	
	// Create device property result
	result := DevicePropertyResult{
		Device:     deviceInfo,
		Properties: []PropertyInfo{propInfo},
		Success:    true,
	}
	
	// Create response message
	resp := ResponseMessage{
		Message: Message{
			Type: "response",
			ID:   "123456",
		},
		Success: true,
		Data:    result,
	}
	
	// Marshal to JSON
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal ResponseMessage: %v", err)
	}
	
	// Unmarshal from JSON
	var decodedResp ResponseMessage
	if err := json.Unmarshal(jsonBytes, &decodedResp); err != nil {
		t.Fatalf("Failed to unmarshal ResponseMessage: %v", err)
	}
	
	// Verify basic fields
	if decodedResp.Type != "response" {
		t.Errorf("Type = %v, want %v", decodedResp.Type, "response")
	}
	
	if decodedResp.ID != "123456" {
		t.Errorf("ID = %v, want %v", decodedResp.ID, "123456")
	}
	
	if decodedResp.Success != true {
		t.Errorf("Success = %v, want %v", decodedResp.Success, true)
	}
	
	// Data will be a map[string]interface{} due to json unmarshaling
	// Just verify it exists for simplicity
	if decodedResp.Data == nil {
		t.Errorf("Data should not be nil")
	}
}

func TestNotificationMessageMarshaling(t *testing.T) {
	// Create device info
	deviceInfo := DeviceInfo{
		IP: "192.168.1.5",
		EOJ: EOJInfo{
			ClassCode:    0x0130,
			InstanceCode: 0x01,
		},
		Aliases: []string{"aircon1"},
	}
	
	// Create notification message
	notif := NotificationMessage{
		Message: Message{
			Type: "notification",
			ID:   "notify123",
		},
		Event:      "deviceAdded",
		DeviceInfo: deviceInfo,
	}
	
	// Marshal to JSON
	jsonBytes, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("Failed to marshal NotificationMessage: %v", err)
	}
	
	// Unmarshal from JSON
	var decodedNotif NotificationMessage
	if err := json.Unmarshal(jsonBytes, &decodedNotif); err != nil {
		t.Fatalf("Failed to unmarshal NotificationMessage: %v", err)
	}
	
	// Verify basic fields
	if decodedNotif.Type != "notification" {
		t.Errorf("Type = %v, want %v", decodedNotif.Type, "notification")
	}
	
	if decodedNotif.ID != "notify123" {
		t.Errorf("ID = %v, want %v", decodedNotif.ID, "notify123")
	}
	
	if decodedNotif.Event != "deviceAdded" {
		t.Errorf("Event = %v, want %v", decodedNotif.Event, "deviceAdded")
	}
	
	// DeviceInfo will be a map[string]interface{} due to json unmarshaling
	// Just verify it exists for simplicity
	if decodedNotif.DeviceInfo == nil {
		t.Errorf("DeviceInfo should not be nil")
	}
}