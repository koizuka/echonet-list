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

func testDevice(id int) handler.IPAndEOJ {
	ip := net.ParseIP(fmt.Sprintf("192.0.2.%d", id))
	eoj := echonet_lite.MakeEOJ(echonet_lite.EOJClassCode(0x0130), echonet_lite.EOJInstanceCode(id))
	return handler.IPAndEOJ{
		IP:  ip,
		EOJ: eoj,
	}
}
