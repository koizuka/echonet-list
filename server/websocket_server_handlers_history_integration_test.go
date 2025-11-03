package server

import (
	"context"
	"encoding/json"
	"fmt"
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

// TestRecordPropertyChange_DuplicateDetection tests the duplicate detection for INF notifications
func TestRecordPropertyChange_DuplicateDetection(t *testing.T) {
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
		ctx:          ctx,
		handler:      liteHandler,
		recentSetOps: make(map[string]setOperationTracker),
	}

	historyStore := ws.GetHistoryStore()
	if historyStore == nil {
		t.Fatal("History store is nil")
	}

	testEPC := echonet_lite.EPCType(0x80)

	t.Run("SpontaneousINFIsRecorded", func(t *testing.T) {
		// Simulate a spontaneous INF notification (remote control, physical button, etc.)
		// This should be recorded because there's no recent SET operation
		change := handler.PropertyChangeNotification{
			Device: testDevice,
			Property: echonet_lite.Property{
				EPC: testEPC,
				EDT: []byte{0x30}, // "on"
			},
		}

		ws.recordPropertyChange(change)

		// Verify it was recorded
		query := handler.HistoryQuery{Limit: 10}
		entries := historyStore.Query(testDevice, query)
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry for spontaneous INF, got %d", len(entries))
		}

		if entries[0].Origin != handler.HistoryOriginNotification {
			t.Errorf("Expected origin Notification, got %s", entries[0].Origin)
		}
	})

	t.Run("SETConfirmationINFIsFiltered", func(t *testing.T) {
		// Clear history
		historyStore.Clear(testDevice)

		// Record a SET operation with value 0x31 ("off")
		testValue := protocol.PropertyData{
			String: "off",
			EDT:    "MQ==", // base64 of 0x31 (49)
		}
		ws.recordSetResult(testDevice, testEPC, testValue)

		// Verify SET was recorded
		query := handler.HistoryQuery{Limit: 10}
		entries := historyStore.Query(testDevice, query)
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry for SET, got %d", len(entries))
		}

		// Simulate an INF confirmation with the same value (within 500ms)
		change := handler.PropertyChangeNotification{
			Device: testDevice,
			Property: echonet_lite.Property{
				EPC: testEPC,
				EDT: []byte{0x31}, // 0x31 = "off" - same as SET
			},
		}

		ws.recordPropertyChange(change)

		// Should still be 1 entry (INF was filtered as duplicate)
		entries = historyStore.Query(testDevice, query)
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry (INF filtered), got %d", len(entries))
		}

		if entries[0].Origin != handler.HistoryOriginSet {
			t.Errorf("Expected origin Set, got %s", entries[0].Origin)
		}
	})

	t.Run("INFWithDifferentValueIsRecorded", func(t *testing.T) {
		// Clear history
		historyStore.Clear(testDevice)

		// Record a SET operation with value 0x31 ("off")
		testValue := protocol.PropertyData{
			String: "off",
			EDT:    "MQ==", // base64 of 0x31
		}
		ws.recordSetResult(testDevice, testEPC, testValue)

		// Simulate an INF with a DIFFERENT value 0x30 ("on")
		change := handler.PropertyChangeNotification{
			Device: testDevice,
			Property: echonet_lite.Property{
				EPC: testEPC,
				EDT: []byte{0x30}, // 0x30 = "on" - different from SET
			},
		}

		ws.recordPropertyChange(change)

		// Should have 2 entries (SET + INF with different value)
		query := handler.HistoryQuery{Limit: 10}
		entries := historyStore.Query(testDevice, query)
		if len(entries) != 2 {
			t.Fatalf("Expected 2 entries (SET + different INF), got %d", len(entries))
		}

		// Newest first: INF (on), SET (off)
		if entries[0].Origin != handler.HistoryOriginNotification {
			t.Errorf("Expected first entry to be Notification, got %s", entries[0].Origin)
		}
		if entries[1].Origin != handler.HistoryOriginSet {
			t.Errorf("Expected second entry to be Set, got %s", entries[1].Origin)
		}
	})

	t.Run("INFAfterTrackingWindowIsRecorded", func(t *testing.T) {
		// Clear history and tracking
		historyStore.Clear(testDevice)
		ws.recentSetOpsMutex.Lock()
		for k := range ws.recentSetOps {
			delete(ws.recentSetOps, k)
		}
		ws.recentSetOpsMutex.Unlock()

		// Record a SET operation with value 0x31 ("off")
		testValue := protocol.PropertyData{
			String: "off",
			EDT:    "MQ==", // base64 of 0x31
		}
		ws.recordSetResult(testDevice, testEPC, testValue)

		// Wait for tracking window to expire (500ms + buffer)
		time.Sleep(600 * time.Millisecond)

		// Simulate an INF with the same value (but after tracking window)
		change := handler.PropertyChangeNotification{
			Device: testDevice,
			Property: echonet_lite.Property{
				EPC: testEPC,
				EDT: []byte{0x31}, // 0x31 = "off" - same as SET
			},
		}

		ws.recordPropertyChange(change)

		// Should have 2 entries (SET + INF after window)
		query := handler.HistoryQuery{Limit: 10}
		entries := historyStore.Query(testDevice, query)
		if len(entries) != 2 {
			t.Fatalf("Expected 2 entries (SET + INF after window), got %d", len(entries))
		}

		if entries[0].Origin != handler.HistoryOriginNotification {
			t.Errorf("Expected first entry to be Notification, got %s", entries[0].Origin)
		}
	})

	t.Run("DifferentDeviceINFIsRecorded", func(t *testing.T) {
		// Clear history
		historyStore.Clear(testDevice)

		// Create second device
		testIP2 := net.ParseIP("192.168.1.101")
		testDevice2 := handler.IPAndEOJ{IP: testIP2, EOJ: testEOJ}
		dataHandler.RegisterDevice(testDevice2)

		// Record a SET operation on device 1 with value 0x31 ("off")
		testValue := protocol.PropertyData{
			String: "off",
			EDT:    "MQ==", // base64 of 0x31
		}
		ws.recordSetResult(testDevice, testEPC, testValue)

		// Simulate an INF from device 2 with the same value
		change := handler.PropertyChangeNotification{
			Device: testDevice2,
			Property: echonet_lite.Property{
				EPC: testEPC,
				EDT: []byte{0x31}, // 0x31 = "off" - same value as device 1 SET
			},
		}

		ws.recordPropertyChange(change)

		// Device 2 should have 1 entry (not filtered because it's a different device)
		query := handler.HistoryQuery{Limit: 10}
		entries := historyStore.Query(testDevice2, query)
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry for device 2, got %d", len(entries))
		}

		if entries[0].Origin != handler.HistoryOriginNotification {
			t.Errorf("Expected origin Notification for device 2, got %s", entries[0].Origin)
		}
	})
}

// TestRecordSetResult_TrackingBeforeHistory verifies that recordSetResult
// adds the SET operation to the tracking map BEFORE recording it to history.
// This ensures that if an INF notification arrives immediately after the SET,
// it can be detected as a duplicate.
//
// This test prevents regression of the bug where tracking was added after
// history recording, causing SET confirmations to be recorded as separate entries.
func TestRecordSetResult_TrackingBeforeHistory(t *testing.T) {
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
		ctx:          ctx,
		handler:      liteHandler,
		recentSetOps: make(map[string]setOperationTracker),
	}

	testEPC := echonet_lite.EPCType(0x80)
	testValue := protocol.PropertyData{
		String: "on",
		EDT:    "MA==", // base64 of 0x30
	}

	// Record a SET operation
	ws.recordSetResult(testDevice, testEPC, testValue)

	// Verify tracking entry exists immediately after recordSetResult
	// The tracking key format is: IP_EOJ_EPC (EPC in uppercase hex)
	expectedKey := fmt.Sprintf("%s_%s_%02X", testDevice.IP.String(), testDevice.EOJ.Specifier(), testEPC)
	ws.recentSetOpsMutex.RLock()
	tracker, exists := ws.recentSetOps[expectedKey]
	ws.recentSetOpsMutex.RUnlock()

	if !exists {
		t.Fatalf("Tracking entry should exist immediately after recordSetResult. Expected key: %s", expectedKey)
	}

	// Verify the tracked value matches
	expectedValue := handler.PropertyValue{
		String: "on",
		EDT:    "MA==",
	}
	if !tracker.Value.Equals(expectedValue) {
		t.Errorf("Tracked value mismatch. Expected %+v, got %+v", expectedValue, tracker.Value)
	}

	// Now simulate an immediate INF notification with the same value
	change := handler.PropertyChangeNotification{
		Device: testDevice,
		Property: echonet_lite.Property{
			EPC: testEPC,
			EDT: []byte{0x30}, // same as SET
		},
	}

	ws.recordPropertyChange(change)

	// Verify that only 1 entry exists in history (INF was filtered)
	historyStore := ws.GetHistoryStore()
	query := handler.HistoryQuery{Limit: 10}
	entries := historyStore.Query(testDevice, query)

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry (INF should be filtered), got %d", len(entries))
	}

	if entries[0].Origin != handler.HistoryOriginSet {
		t.Errorf("Expected origin Set, got %s", entries[0].Origin)
	}
}
