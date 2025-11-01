package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
)

func TestHandleGetDeviceHistoryFromClient_Success(t *testing.T) {
	ctx := context.Background()
	device := testDevice(20)

	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})

	// For this test, we'll use a nil handler which will cause isPropertySettable to return false
	// This means we need to query with settableOnly=false to see results
	ws := &WebSocketServer{
		ctx:          ctx,
		historyStore: store,
		handler:      nil,
		deviceResolver: func(d echonet_lite.IPAndEOJ) bool {
			return d.Specifier() == device.Specifier()
		},
	}

	ws.historyStore.Record(DeviceHistoryEntry{
		Timestamp: time.Now().UTC(),
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     protocol.PropertyData{String: "on"},
		Origin:    HistoryOriginSet,
	})

	// Query with settableOnly=false since handler is nil and will return settable=false
	settableOnly := false
	payload := protocol.GetDeviceHistoryPayload{
		Target:       device.Specifier(),
		SettableOnly: &settableOnly,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	result := ws.handleGetDeviceHistoryFromClient(&protocol.Message{
		Type:    protocol.MessageTypeGetDeviceHistory,
		Payload: payloadBytes,
	})

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	var response protocol.DeviceHistoryResponse
	if err := json.Unmarshal(result.Data, &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(response.Entries))
	}

	entry := response.Entries[0]
	if entry.EPC != "80" {
		t.Fatalf("expected EPC 80, got %s", entry.EPC)
	}
	if entry.Origin != protocol.HistoryOriginSet {
		t.Fatalf("expected origin set, got %s", entry.Origin)
	}
	// Since handler is nil, isPropertySettable returns false
	if entry.Settable {
		t.Fatalf("expected settable false (handler is nil)")
	}
	if entry.Value.String != "on" {
		t.Fatalf("expected value 'on', got %s", entry.Value.String)
	}
}

func TestHandleGetDeviceHistoryFromClient_InvalidLimit(t *testing.T) {
	ctx := context.Background()
	device := testDevice(21)

	ws := &WebSocketServer{
		ctx:          ctx,
		historyStore: newMemoryDeviceHistoryStore(DefaultHistoryOptions()),
		deviceResolver: func(d echonet_lite.IPAndEOJ) bool {
			return d.Specifier() == device.Specifier()
		},
	}

	limit := 0
	payload := protocol.GetDeviceHistoryPayload{
		Target: device.Specifier(),
		Limit:  &limit,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	result := ws.handleGetDeviceHistoryFromClient(&protocol.Message{
		Type:    protocol.MessageTypeGetDeviceHistory,
		Payload: payloadBytes,
	})

	if result.Success {
		t.Fatalf("expected failure due to invalid limit")
	}
	if result.Error == nil || result.Error.Code != protocol.ErrorCodeInvalidParameters {
		t.Fatalf("expected invalid parameters error, got %+v", result.Error)
	}
}

func TestHandleGetDeviceHistoryFromClient_SettableCalculation(t *testing.T) {
	ctx := context.Background()
	device := testDevice(22)

	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})

	// Create a real handler and register the device with a Set Property Map
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Register the device
	handlerInstance.GetDataManagementHandler().RegisterDevice(device)

	// Set up a Set Property Map (EPC 0x9E) with EPCs 0x80 and 0xB0 as settable
	// EPCs 0xBB (room temperature) and 0xBE (outdoor temperature) are not in the map (read-only sensors)
	// Property Map format: number of properties (1 byte) + property codes (1 byte each)
	setPropertyMapEDT := []byte{
		2,    // Number of settable properties
		0x80, // Operation status
		0xB0, // Operation mode
	}
	setPropertyMapProperty := echonet_lite.Properties{
		echonet_lite.EPCType(0x9E): {
			EPC: echonet_lite.EPCType(0x9E),
			EDT: setPropertyMapEDT,
		},
	}
	handlerInstance.GetDataManagementHandler().RegisterProperties(device, setPropertyMapProperty)

	ws := &WebSocketServer{
		ctx:          ctx,
		historyStore: store,
		handler:      handlerInstance,
		deviceResolver: func(d echonet_lite.IPAndEOJ) bool {
			return d.Specifier() == device.Specifier()
		},
	}

	// Record history entries: 2 settable properties and 2 read-only sensor values
	ws.historyStore.Record(DeviceHistoryEntry{
		Timestamp: time.Now().UTC().Add(-4 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80), // Settable
		Value:     protocol.PropertyData{String: "on"},
		Origin:    HistoryOriginSet,
	})
	ws.historyStore.Record(DeviceHistoryEntry{
		Timestamp: time.Now().UTC().Add(-3 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0xBB), // Read-only sensor
		Value:     protocol.PropertyData{Number: intPtr(23)},
		Origin:    HistoryOriginNotification,
	})
	ws.historyStore.Record(DeviceHistoryEntry{
		Timestamp: time.Now().UTC().Add(-2 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0xB0), // Settable
		Value:     protocol.PropertyData{String: "cooling"},
		Origin:    HistoryOriginSet,
	})
	ws.historyStore.Record(DeviceHistoryEntry{
		Timestamp: time.Now().UTC().Add(-1 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0xBE), // Read-only sensor
		Value:     protocol.PropertyData{Number: intPtr(30)},
		Origin:    HistoryOriginNotification,
	})

	// Test 1: Query with settableOnly=false - should return all 4 entries
	t.Run("All entries with settableOnly=false", func(t *testing.T) {
		settableOnly := false
		payload := protocol.GetDeviceHistoryPayload{
			Target:       device.Specifier(),
			SettableOnly: &settableOnly,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}

		result := ws.handleGetDeviceHistoryFromClient(&protocol.Message{
			Type:    protocol.MessageTypeGetDeviceHistory,
			Payload: payloadBytes,
		})

		if !result.Success {
			t.Fatalf("expected success, got error: %v", result.Error)
		}

		var response protocol.DeviceHistoryResponse
		if err := json.Unmarshal(result.Data, &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(response.Entries) != 4 {
			t.Fatalf("expected 4 entries, got %d", len(response.Entries))
		}

		// Verify settable flags (newest first)
		expectedSettable := []bool{false, true, false, true} // 0xBE, 0xB0, 0xBB, 0x80
		for i, entry := range response.Entries {
			if entry.Settable != expectedSettable[i] {
				t.Errorf("entry %d (EPC %s): expected settable=%v, got settable=%v",
					i, entry.EPC, expectedSettable[i], entry.Settable)
			}
		}
	})

	// Test 2: Query with settableOnly=true - should return only 2 settable entries
	t.Run("Settable entries only with settableOnly=true", func(t *testing.T) {
		settableOnly := true
		payload := protocol.GetDeviceHistoryPayload{
			Target:       device.Specifier(),
			SettableOnly: &settableOnly,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}

		result := ws.handleGetDeviceHistoryFromClient(&protocol.Message{
			Type:    protocol.MessageTypeGetDeviceHistory,
			Payload: payloadBytes,
		})

		if !result.Success {
			t.Fatalf("expected success, got error: %v", result.Error)
		}

		var response protocol.DeviceHistoryResponse
		if err := json.Unmarshal(result.Data, &response); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(response.Entries) != 2 {
			t.Fatalf("expected 2 settable entries, got %d", len(response.Entries))
		}

		// Verify all returned entries are settable
		for i, entry := range response.Entries {
			if !entry.Settable {
				t.Errorf("entry %d (EPC %s): expected settable=true, got settable=false",
					i, entry.EPC)
			}
		}

		// Verify EPCs are the settable ones (0xB0 and 0x80, newest first)
		if response.Entries[0].EPC != "B0" || response.Entries[1].EPC != "80" {
			t.Errorf("expected EPCs [B0, 80], got [%s, %s]",
				response.Entries[0].EPC, response.Entries[1].EPC)
		}
	})
}
