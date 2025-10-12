package server

import (
	"encoding/base64"
	"testing"
	"time"

	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
)

func TestWebSocketServer_RecordPropertyChange(t *testing.T) {
	store := newMemoryDeviceHistoryStore(DefaultHistoryOptions())
	ws := &WebSocketServer{
		historyStore: store,
	}

	device := testDevice(10)
	change := handler.PropertyChangeNotification{
		Device: device,
		Property: echonet_lite.Property{
			EPC: echonet_lite.EPCType(0x80),
			EDT: []byte{0x30},
		},
	}

	ws.recordPropertyChange(change)

	entries := store.Query(device, HistoryQuery{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Origin != HistoryOriginNotification {
		t.Fatalf("expected origin %s, got %s", HistoryOriginNotification, entry.Origin)
	}
	if entry.Value.EDT != base64.StdEncoding.EncodeToString([]byte{0x30}) {
		t.Fatalf("unexpected EDT value %q", entry.Value.EDT)
	}
}

func TestWebSocketServer_RecordSetResult(t *testing.T) {
	store := newMemoryDeviceHistoryStore(DefaultHistoryOptions())
	ws := &WebSocketServer{
		historyStore: store,
	}

	device := testDevice(11)
	value := protocol.PropertyData{
		String: "on",
	}

	ws.recordSetResult(device, echonet_lite.EPCType(0x80), value)

	entries := store.Query(device, HistoryQuery{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Origin != HistoryOriginSet {
		t.Fatalf("expected origin %s, got %s", HistoryOriginSet, entry.Origin)
	}
	if entry.Value.String != "on" {
		t.Fatalf("unexpected value %s", entry.Value.String)
	}
}

func TestWebSocketServer_ClearHistoryForDevice(t *testing.T) {
	store := newMemoryDeviceHistoryStore(DefaultHistoryOptions())
	ws := &WebSocketServer{
		historyStore: store,
	}

	device := testDevice(12)
	store.Record(DeviceHistoryEntry{
		Timestamp: time.Now(),
		Device:    device,
		EPC:       echonet_lite.EPCType(0xA0),
		Value:     protocol.PropertyData{String: "value"},
		Origin:    HistoryOriginNotification,
	})

	ws.clearHistoryForDevice(device)

	if entries := store.Query(device, HistoryQuery{}); len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}
