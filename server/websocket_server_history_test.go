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

func TestWebSocketServer_RecordPropertyChange_SkipsDuplicateNotification(t *testing.T) {
	store := newMemoryDeviceHistoryStore(DefaultHistoryOptions())
	ws := &WebSocketServer{
		historyStore: store,
	}

	device := testDevice(13)
	epc := echonet_lite.EPCType(0x80)
	edt := []byte{0x30}

	// Create the value that will be produced by MakePropertyData
	prop := echonet_lite.Property{
		EPC: epc,
		EDT: edt,
	}
	expectedValue := protocol.MakePropertyData(device.EOJ.ClassCode(), prop)

	// First, record a Set operation with the expected value
	ws.recordSetResult(device, epc, expectedValue)

	// Verify the Set operation was recorded
	entries := store.Query(device, HistoryQuery{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after Set operation, got %d", len(entries))
	}
	if entries[0].Origin != HistoryOriginSet {
		t.Fatalf("expected origin %s, got %s", HistoryOriginSet, entries[0].Origin)
	}

	// Now, record a notification with the same value
	change := handler.PropertyChangeNotification{
		Device:   device,
		Property: prop,
	}
	ws.recordPropertyChange(change)

	// Verify that the notification was NOT recorded (still only 1 entry)
	entries = store.Query(device, HistoryQuery{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (notification should be skipped), got %d", len(entries))
	}
	if entries[0].Origin != HistoryOriginSet {
		t.Fatalf("expected only Set origin to remain, got %s", entries[0].Origin)
	}
}

func TestWebSocketServer_RecordPropertyChange_RecordsNonDuplicateNotification(t *testing.T) {
	store := newMemoryDeviceHistoryStore(DefaultHistoryOptions())
	ws := &WebSocketServer{
		historyStore: store,
	}

	device := testDevice(14)
	epc := echonet_lite.EPCType(0x80)

	// First, record a Set operation with value "on" (EDT: 0x30)
	prop1 := echonet_lite.Property{
		EPC: epc,
		EDT: []byte{0x30},
	}
	value1 := protocol.MakePropertyData(device.EOJ.ClassCode(), prop1)
	ws.recordSetResult(device, epc, value1)

	// Verify the Set operation was recorded
	entries := store.Query(device, HistoryQuery{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after Set operation, got %d", len(entries))
	}

	// Now, record a notification with a DIFFERENT value (EDT: 0x31 = "off")
	prop2 := echonet_lite.Property{
		EPC: epc,
		EDT: []byte{0x31},
	}
	change := handler.PropertyChangeNotification{
		Device:   device,
		Property: prop2,
	}
	ws.recordPropertyChange(change)

	// Verify that the notification WAS recorded (2 entries)
	entries = store.Query(device, HistoryQuery{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (notification with different value should be recorded), got %d", len(entries))
	}

	// Verify the order (newest first)
	if entries[0].Origin != HistoryOriginNotification {
		t.Fatalf("expected newest entry to be Notification, got %s", entries[0].Origin)
	}
	if entries[1].Origin != HistoryOriginSet {
		t.Fatalf("expected second entry to be Set, got %s", entries[1].Origin)
	}
}

func TestWebSocketServer_RecordPropertyChange_RecordsNotificationAfterTimeWindow(t *testing.T) {
	store := newMemoryDeviceHistoryStore(DefaultHistoryOptions())
	ws := &WebSocketServer{
		historyStore: store,
	}

	device := testDevice(15)
	epc := echonet_lite.EPCType(0x80)
	edt := []byte{0x30}

	// Create the value that will be produced by MakePropertyData
	prop := echonet_lite.Property{
		EPC: epc,
		EDT: edt,
	}
	value := protocol.MakePropertyData(device.EOJ.ClassCode(), prop)

	// Record a Set operation with a timestamp 3 seconds ago (outside the 2-second window)
	oldEntry := DeviceHistoryEntry{
		Timestamp: time.Now().UTC().Add(-3 * time.Second),
		Device:    device,
		EPC:       epc,
		Value:     value,
		Origin:    HistoryOriginSet,
		Settable:  true,
	}
	store.Record(oldEntry)

	// Verify the Set operation was recorded
	entries := store.Query(device, HistoryQuery{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after Set operation, got %d", len(entries))
	}

	// Now, record a notification with the same value
	change := handler.PropertyChangeNotification{
		Device:   device,
		Property: prop,
	}
	ws.recordPropertyChange(change)

	// Verify that the notification WAS recorded (2 entries) because Set is outside time window
	entries = store.Query(device, HistoryQuery{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (notification outside time window should be recorded), got %d", len(entries))
	}

	// Verify the order (newest first)
	if entries[0].Origin != HistoryOriginNotification {
		t.Fatalf("expected newest entry to be Notification, got %s", entries[0].Origin)
	}
	if entries[1].Origin != HistoryOriginSet {
		t.Fatalf("expected second entry to be Set, got %s", entries[1].Origin)
	}
}
