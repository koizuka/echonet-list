package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"echonet-list/echonet_lite"
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
