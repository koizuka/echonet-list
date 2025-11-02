package server

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
)

// TestHandleGetDeviceHistoryFromClient_Basic tests basic functionality of history retrieval
func TestHandleGetDeviceHistoryFromClient_Basic(t *testing.T) {
	ctx := context.Background()

	// Create a test handler with history enabled
	handlerOpts := handler.ECHONETLieHandlerOptions{
		TestMode: true,
		HistoryOptions: handler.HistoryOptions{
			PerDeviceSettableLimit:    10,
			PerDeviceNonSettableLimit: 10,
		},
	}
	liteHandler, err := handler.NewECHONETLiteHandler(ctx, handlerOpts)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	defer liteHandler.Close()

	// Create test device
	testIP := net.ParseIP("192.168.1.100")
	testEOJ := echonet_lite.MakeEOJ(0x0291, 1) // Single Function Lighting
	testDevice := handler.IPAndEOJ{IP: testIP, EOJ: testEOJ}

	// Register device in handler
	dataHandler := liteHandler.GetDataManagementHandler()
	dataHandler.RegisterDevice(testDevice)

	// Create WebSocket server
	ws := &WebSocketServer{
		ctx:     ctx,
		handler: liteHandler,
		deviceResolver: func(d handler.IPAndEOJ) bool {
			return d.Key() == testDevice.Key()
		},
	}

	// Record a test history entry
	historyStore := ws.GetHistoryStore()
	if historyStore == nil {
		t.Fatal("History store is nil")
	}

	testValue := handler.PropertyValue{
		String: "on",
		EDT:    "MA==", // base64 of 0x30
	}
	historyStore.Record(handler.DeviceHistoryEntry{
		Timestamp: time.Now().UTC(),
		Device:    testDevice,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     testValue,
		Origin:    handler.HistoryOriginSet,
		Settable:  true,
	})

	// Test 1: Query with settableOnly=false (should get all entries)
	t.Run("QueryAllEntries", func(t *testing.T) {
		settableOnly := false
		payload := protocol.GetDeviceHistoryPayload{
			Target:       testDevice.Specifier(),
			SettableOnly: &settableOnly,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		result := ws.handleGetDeviceHistoryFromClient(&protocol.Message{
			Type:    protocol.MessageTypeGetDeviceHistory,
			Payload: payloadBytes,
		})

		if !result.Success {
			t.Fatalf("Expected success, got error: %v", result.Error)
		}

		var response protocol.DeviceHistoryResponse
		if err := json.Unmarshal(result.Data, &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(response.Entries))
		}

		entry := response.Entries[0]
		if entry.EPC != "80" {
			t.Errorf("Expected EPC 80, got %s", entry.EPC)
		}
		if entry.Origin != protocol.HistoryOriginSet {
			t.Errorf("Expected origin set, got %s", entry.Origin)
		}
		if entry.Value.String != "on" {
			t.Errorf("Expected value 'on', got %s", entry.Value.String)
		}
	})

	// Test 2: Invalid limit
	t.Run("InvalidLimit", func(t *testing.T) {
		limit := 0
		payload := protocol.GetDeviceHistoryPayload{
			Target: testDevice.Specifier(),
			Limit:  &limit,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		result := ws.handleGetDeviceHistoryFromClient(&protocol.Message{
			Type:    protocol.MessageTypeGetDeviceHistory,
			Payload: payloadBytes,
		})

		if result.Success {
			t.Error("Expected error for invalid limit")
		}
		if result.Error == nil || result.Error.Code != protocol.ErrorCodeInvalidParameters {
			t.Errorf("Expected InvalidParameters error, got: %v", result.Error)
		}
	})

	// Test 3: Unknown device
	t.Run("UnknownDevice", func(t *testing.T) {
		payload := protocol.GetDeviceHistoryPayload{
			Target: "192.168.1.200 0291:1",
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		result := ws.handleGetDeviceHistoryFromClient(&protocol.Message{
			Type:    protocol.MessageTypeGetDeviceHistory,
			Payload: payloadBytes,
		})

		if result.Success {
			t.Error("Expected error for unknown device")
		}
		if result.Error == nil || result.Error.Code != protocol.ErrorCodeInvalidParameters {
			t.Errorf("Expected InvalidParameters error, got: %v", result.Error)
		}
	})

	// Test 4: Empty target
	t.Run("EmptyTarget", func(t *testing.T) {
		payload := protocol.GetDeviceHistoryPayload{
			Target: "",
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("Failed to marshal payload: %v", err)
		}

		result := ws.handleGetDeviceHistoryFromClient(&protocol.Message{
			Type:    protocol.MessageTypeGetDeviceHistory,
			Payload: payloadBytes,
		})

		if result.Success {
			t.Error("Expected error for empty target")
		}
		if result.Error == nil || result.Error.Code != protocol.ErrorCodeInvalidParameters {
			t.Errorf("Expected InvalidParameters error, got: %v", result.Error)
		}
	})
}

// TestRecordHistory tests the recordHistory function
func TestRecordHistory(t *testing.T) {
	ctx := context.Background()

	// Create a test handler with history enabled
	handlerOpts := handler.ECHONETLieHandlerOptions{
		TestMode: true,
		HistoryOptions: handler.HistoryOptions{
			PerDeviceSettableLimit:    10,
			PerDeviceNonSettableLimit: 10,
		},
	}
	liteHandler, err := handler.NewECHONETLiteHandler(ctx, handlerOpts)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	defer liteHandler.Close()

	// Create test device
	testIP := net.ParseIP("192.168.1.100")
	testEOJ := echonet_lite.MakeEOJ(0x0291, 1)
	testDevice := handler.IPAndEOJ{IP: testIP, EOJ: testEOJ}

	// Register device in handler
	dataHandler := liteHandler.GetDataManagementHandler()
	dataHandler.RegisterDevice(testDevice)

	ws := &WebSocketServer{
		ctx:     ctx,
		handler: liteHandler,
	}

	// Test recording history
	testValue := protocol.PropertyData{
		String: "on",
		EDT:    "MA==",
	}
	ws.recordHistory(testDevice, echonet_lite.EPCType(0x80), testValue, handler.HistoryOriginSet)

	// Verify the entry was recorded
	historyStore := ws.GetHistoryStore()
	if historyStore == nil {
		t.Fatal("History store is nil")
	}

	query := handler.HistoryQuery{Limit: 10}
	entries := historyStore.Query(testDevice, query)

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.EPC != echonet_lite.EPCType(0x80) {
		t.Errorf("Expected EPC 0x80, got 0x%02X", entry.EPC)
	}
	if entry.Origin != handler.HistoryOriginSet {
		t.Errorf("Expected origin Set, got %s", entry.Origin)
	}
	if entry.Value.String != "on" {
		t.Errorf("Expected value 'on', got %s", entry.Value.String)
	}
	// Note: Without Set Property Map, settable will be false
	// This is expected behavior for test mode
}

// TestClearHistoryForDevice tests the clearHistoryForDevice function
func TestClearHistoryForDevice(t *testing.T) {
	ctx := context.Background()

	// Create a test handler with history enabled
	handlerOpts := handler.ECHONETLieHandlerOptions{
		TestMode: true,
		HistoryOptions: handler.HistoryOptions{
			PerDeviceSettableLimit:    10,
			PerDeviceNonSettableLimit: 10,
		},
	}
	liteHandler, err := handler.NewECHONETLiteHandler(ctx, handlerOpts)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	defer liteHandler.Close()

	// Create test device
	testIP := net.ParseIP("192.168.1.100")
	testEOJ := echonet_lite.MakeEOJ(0x0291, 1)
	testDevice := handler.IPAndEOJ{IP: testIP, EOJ: testEOJ}

	ws := &WebSocketServer{
		ctx:     ctx,
		handler: liteHandler,
	}

	// Record some history
	historyStore := ws.GetHistoryStore()
	if historyStore == nil {
		t.Fatal("History store is nil")
	}

	historyStore.Record(handler.DeviceHistoryEntry{
		Timestamp: time.Now().UTC(),
		Device:    testDevice,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     handler.PropertyValue{String: "on"},
		Origin:    handler.HistoryOriginSet,
		Settable:  true,
	})

	// Verify entry exists
	query := handler.HistoryQuery{Limit: 10}
	entries := historyStore.Query(testDevice, query)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry before clear, got %d", len(entries))
	}

	// Clear history
	ws.clearHistoryForDevice(testDevice)

	// Verify entry was cleared
	entries = historyStore.Query(testDevice, query)
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries after clear, got %d", len(entries))
	}
}
