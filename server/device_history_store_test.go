package server

import (
	"fmt"
	"net"
	"testing"
	"time"

	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
)

func TestMemoryDeviceHistoryStore_RecordEnforcesLimit(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 3})
	device := testDevice(1)
	base := time.Now()

	for i := 0; i < 5; i++ {
		store.Record(DeviceHistoryEntry{
			Timestamp: base.Add(time.Duration(i) * time.Minute),
			Device:    device,
			EPC:       echonet_lite.EPCType(0x80),
			Value: protocol.PropertyData{
				String: fmt.Sprintf("value-%d", i),
			},
			Origin:   HistoryOriginNotification,
			Settable: true,
		})
	}

	entries := store.Query(device, HistoryQuery{})

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Should be newest first: indices 4,3,2
	for i, expected := range []string{"value-4", "value-3", "value-2"} {
		if entries[i].Value.String != expected {
			t.Errorf("entry %d expected value %s, got %s", i, expected, entries[i].Value.String)
		}
	}
}

func TestMemoryDeviceHistoryStore_QueryFilters(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	device := testDevice(2)
	base := time.Now()

	entries := []DeviceHistoryEntry{
		{
			Timestamp: base.Add(-3 * time.Hour),
			Device:    device,
			EPC:       echonet_lite.EPCType(0xA0),
			Value:     protocol.PropertyData{String: "old"},
			Origin:    HistoryOriginNotification,
			Settable:  true,
		},
		{
			Timestamp: base.Add(-2 * time.Hour),
			Device:    device,
			EPC:       echonet_lite.EPCType(0xA1),
			Value:     protocol.PropertyData{String: "no-set"},
			Origin:    HistoryOriginNotification,
			Settable:  false,
		},
		{
			Timestamp: base.Add(-1 * time.Hour),
			Device:    device,
			EPC:       echonet_lite.EPCType(0xA2),
			Value:     protocol.PropertyData{String: "recent"},
			Origin:    HistoryOriginSet,
			Settable:  true,
		},
	}

	for _, entry := range entries {
		store.Record(entry)
	}

	result := store.Query(device, HistoryQuery{
		Since:        base.Add(-90 * time.Minute),
		SettableOnly: true,
		Limit:        5,
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}

	if result[0].Value.String != "recent" {
		t.Fatalf("expected 'recent', got %s", result[0].Value.String)
	}
}

func TestMemoryDeviceHistoryStore_Clear(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 5})
	device := testDevice(3)

	store.Record(DeviceHistoryEntry{
		Timestamp: time.Now(),
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     protocol.PropertyData{String: "value"},
		Origin:    HistoryOriginNotification,
		Settable:  true,
	})

	if entries := store.Query(device, HistoryQuery{}); len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	store.Clear(device)

	if entries := store.Query(device, HistoryQuery{}); len(entries) != 0 {
		t.Fatalf("expected 0 entry after clear, got %d", len(entries))
	}
}

func TestMemoryDeviceHistoryStore_IsDuplicateNotification(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	device := testDevice(4)
	now := time.Now().UTC()

	// Test case 1: No history - should not be duplicate
	isDup := store.IsDuplicateNotification(device, echonet_lite.EPCType(0x80), protocol.PropertyData{String: "on"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate when history is empty")
	}

	// Test case 2: Add a Set operation
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     protocol.PropertyData{String: "on"},
		Origin:    HistoryOriginSet,
		Settable:  true,
	})

	// Test case 3: Same device, EPC, and value within time window - should be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x80), protocol.PropertyData{String: "on"}, 2*time.Second)
	if !isDup {
		t.Error("expected duplicate for same device, EPC, and value within time window")
	}

	// Test case 4: Different value - should not be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x80), protocol.PropertyData{String: "off"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for different value")
	}

	// Test case 5: Different EPC - should not be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x81), protocol.PropertyData{String: "on"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for different EPC")
	}

	// Test case 6: Add an older Set operation (outside time window)
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(-3 * time.Second),
		Device:    device,
		EPC:       echonet_lite.EPCType(0x82),
		Value:     protocol.PropertyData{String: "value1"},
		Origin:    HistoryOriginSet,
		Settable:  true,
	})

	// Test case 7: Outside time window - should not be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x82), protocol.PropertyData{String: "value1"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for Set operation outside time window")
	}

	// Test case 8: Add a Notification (not Set) operation
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Second),
		Device:    device,
		EPC:       echonet_lite.EPCType(0x83),
		Value:     protocol.PropertyData{String: "notif"},
		Origin:    HistoryOriginNotification,
		Settable:  true,
	})

	// Test case 9: Notification origin - should not be considered as duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x83), protocol.PropertyData{String: "notif"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate when origin is Notification (not Set)")
	}
}

func TestMemoryDeviceHistoryStore_IsDuplicateNotification_NumericValues(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	device := testDevice(5)
	now := time.Now().UTC()

	// Add a Set operation with numeric value
	num25 := 25
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0xB0),
		Value:     protocol.PropertyData{Number: &num25},
		Origin:    HistoryOriginSet,
		Settable:  true,
	})

	// Same numeric value - should be duplicate
	num25Copy := 25
	isDup := store.IsDuplicateNotification(device, echonet_lite.EPCType(0xB0), protocol.PropertyData{Number: &num25Copy}, 2*time.Second)
	if !isDup {
		t.Error("expected duplicate for same numeric value")
	}

	// Different numeric value - should not be duplicate
	num26 := 26
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0xB0), protocol.PropertyData{Number: &num26}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for different numeric value")
	}
}

func TestMemoryDeviceHistoryStore_IsDuplicateNotification_EDTValues(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	device := testDevice(6)
	now := time.Now().UTC()

	// Add a Set operation with EDT value
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0xC0),
		Value:     protocol.PropertyData{EDT: "AQID"}, // base64 encoded bytes
		Origin:    HistoryOriginSet,
		Settable:  true,
	})

	// Same EDT value - should be duplicate
	isDup := store.IsDuplicateNotification(device, echonet_lite.EPCType(0xC0), protocol.PropertyData{EDT: "AQID"}, 2*time.Second)
	if !isDup {
		t.Error("expected duplicate for same EDT value")
	}

	// Different EDT value - should not be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0xC0), protocol.PropertyData{EDT: "BAUG"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for different EDT value")
	}
}

func testDevice(id int) handler.IPAndEOJ {
	ip := net.ParseIP(fmt.Sprintf("192.0.2.%d", id))
	eoj := echonet_lite.MakeEOJ(echonet_lite.EOJClassCode(0x0130), echonet_lite.EOJInstanceCode(id))
	return handler.IPAndEOJ{
		IP:  ip,
		EOJ: eoj,
	}
}
