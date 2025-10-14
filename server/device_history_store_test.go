package server

import (
	"fmt"
	"net"
	"os"
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

func TestMemoryDeviceHistoryStore_SaveToFile(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	device1 := testDevice(1)
	device2 := testDevice(2)
	now := time.Now().UTC()

	// Add some test data
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device1,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     protocol.PropertyData{String: "on"},
		Origin:    HistoryOriginSet,
		Settable:  true,
	})
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Minute),
		Device:    device2,
		EPC:       echonet_lite.EPCType(0xB0),
		Value:     protocol.PropertyData{Number: intPtr(25)},
		Origin:    HistoryOriginNotification,
		Settable:  false,
	})

	// Save to temporary file
	tmpFile := t.TempDir() + "/history_test.json"
	err := store.SaveToFile(tmpFile)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Verify file exists and is valid JSON
	data, err := readTestFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("saved file is empty")
	}

	// Basic JSON structure check
	if !containsString(data, "\"version\"") || !containsString(data, "\"data\"") {
		t.Error("saved file does not contain expected JSON structure")
	}
}

func TestMemoryDeviceHistoryStore_LoadFromFile_BasicLoad(t *testing.T) {
	// Create and populate store
	store1 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	device := testDevice(1)
	now := time.Now().UTC()

	store1.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     protocol.PropertyData{String: "test-value"},
		Origin:    HistoryOriginSet,
		Settable:  true,
	})

	// Save to file
	tmpFile := t.TempDir() + "/history_load_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load into new store
	store2 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	filter := HistoryLoadFilter{
		Since:          24 * time.Hour, // Load everything within 24 hours
		PerDeviceLimit: 100,
	}
	if err := store2.LoadFromFile(tmpFile, filter); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify loaded data
	entries := store2.Query(device, HistoryQuery{})
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after load, got %d", len(entries))
	}

	if entries[0].Value.String != "test-value" {
		t.Errorf("expected value 'test-value', got '%s'", entries[0].Value.String)
	}
	if entries[0].EPC != echonet_lite.EPCType(0x80) {
		t.Errorf("expected EPC 0x80, got 0x%02X", entries[0].EPC)
	}
	if entries[0].Origin != HistoryOriginSet {
		t.Errorf("expected origin 'set', got '%s'", entries[0].Origin)
	}
}

func TestMemoryDeviceHistoryStore_LoadFromFile_TimeFilter(t *testing.T) {
	store1 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	device := testDevice(1)
	now := time.Now().UTC()

	// Add entries at different times
	entries := []struct {
		offset time.Duration
		value  string
	}{
		{-10 * 24 * time.Hour, "very-old"}, // 10 days ago
		{-5 * 24 * time.Hour, "old"},       // 5 days ago
		{-2 * 24 * time.Hour, "recent"},    // 2 days ago
		{-1 * time.Hour, "very-recent"},    // 1 hour ago
	}

	for i, entry := range entries {
		store1.Record(DeviceHistoryEntry{
			Timestamp: now.Add(entry.offset),
			Device:    device,
			EPC:       echonet_lite.EPCType(0x80 + byte(i)),
			Value:     protocol.PropertyData{String: entry.value},
			Origin:    HistoryOriginNotification,
			Settable:  true,
		})
	}

	// Save to file
	tmpFile := t.TempDir() + "/history_time_filter_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load with time filter (only last 3 days)
	store2 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	filter := HistoryLoadFilter{
		Since:          3 * 24 * time.Hour, // Last 3 days
		PerDeviceLimit: 100,
	}
	if err := store2.LoadFromFile(tmpFile, filter); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify only recent entries were loaded
	loadedEntries := store2.Query(device, HistoryQuery{})
	if len(loadedEntries) != 2 {
		t.Fatalf("expected 2 entries (within 3 days), got %d", len(loadedEntries))
	}

	// Check that very-old and old entries were filtered out
	for _, entry := range loadedEntries {
		if entry.Value.String == "very-old" || entry.Value.String == "old" {
			t.Errorf("time filter failed: entry '%s' should have been filtered out", entry.Value.String)
		}
	}
}

func TestMemoryDeviceHistoryStore_LoadFromFile_CountFilter(t *testing.T) {
	store1 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 100})
	device := testDevice(1)
	now := time.Now().UTC()

	// Add 10 entries
	for i := 0; i < 10; i++ {
		store1.Record(DeviceHistoryEntry{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Device:    device,
			EPC:       echonet_lite.EPCType(0x80),
			Value:     protocol.PropertyData{String: fmt.Sprintf("value-%d", i)},
			Origin:    HistoryOriginNotification,
			Settable:  true,
		})
	}

	// Save to file
	tmpFile := t.TempDir() + "/history_count_filter_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load with count filter (only 3 most recent)
	store2 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 100})
	filter := HistoryLoadFilter{
		Since:          24 * time.Hour, // Load everything within 24 hours
		PerDeviceLimit: 3,              // But only 3 per device
	}
	if err := store2.LoadFromFile(tmpFile, filter); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify only 3 most recent entries were loaded
	loadedEntries := store2.Query(device, HistoryQuery{})
	if len(loadedEntries) != 3 {
		t.Fatalf("expected 3 entries (count limit), got %d", len(loadedEntries))
	}

	// Should be newest first: value-9, value-8, value-7
	expectedValues := []string{"value-9", "value-8", "value-7"}
	for i, expected := range expectedValues {
		if loadedEntries[i].Value.String != expected {
			t.Errorf("entry %d: expected '%s', got '%s'", i, expected, loadedEntries[i].Value.String)
		}
	}
}

func TestMemoryDeviceHistoryStore_RoundTrip(t *testing.T) {
	store1 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	device1 := testDevice(1)
	device2 := testDevice(2)
	now := time.Now().UTC()

	// Add diverse data types
	num25 := 25
	store1.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device1,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     protocol.PropertyData{String: "string-value"},
		Origin:    HistoryOriginSet,
		Settable:  true,
	})
	store1.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Minute),
		Device:    device1,
		EPC:       echonet_lite.EPCType(0xB0),
		Value:     protocol.PropertyData{Number: &num25},
		Origin:    HistoryOriginNotification,
		Settable:  false,
	})
	store1.Record(DeviceHistoryEntry{
		Timestamp: now.Add(2 * time.Minute),
		Device:    device2,
		EPC:       echonet_lite.EPCType(0xC0),
		Value:     protocol.PropertyData{EDT: "AQIDBA=="},
		Origin:    HistoryOriginSet,
		Settable:  true,
	})

	// Save
	tmpFile := t.TempDir() + "/history_roundtrip_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load
	store2 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	filter := HistoryLoadFilter{
		Since:          24 * time.Hour,
		PerDeviceLimit: 100,
	}
	if err := store2.LoadFromFile(tmpFile, filter); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify device1 entries
	entries1 := store2.Query(device1, HistoryQuery{})
	if len(entries1) != 2 {
		t.Fatalf("device1: expected 2 entries, got %d", len(entries1))
	}

	// Verify device2 entries
	entries2 := store2.Query(device2, HistoryQuery{})
	if len(entries2) != 1 {
		t.Fatalf("device2: expected 1 entry, got %d", len(entries2))
	}

	// Check string value
	if entries1[1].Value.String != "string-value" {
		t.Errorf("string value mismatch: expected 'string-value', got '%s'", entries1[1].Value.String)
	}

	// Check numeric value
	if entries1[0].Value.Number == nil || *entries1[0].Value.Number != 25 {
		t.Errorf("numeric value mismatch")
	}

	// Check EDT value
	if entries2[0].Value.EDT != "AQIDBA==" {
		t.Errorf("EDT value mismatch: expected 'AQIDBA==', got '%s'", entries2[0].Value.EDT)
	}
}

func TestMemoryDeviceHistoryStore_LoadFromFile_FileNotFound(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	filter := DefaultHistoryLoadFilter()

	// Try to load non-existent file - should not error
	err := store.LoadFromFile("/nonexistent/path/history.json", filter)
	if err != nil {
		t.Errorf("LoadFromFile should not error on non-existent file, got: %v", err)
	}

	// Store should remain empty
	device := testDevice(1)
	entries := store.Query(device, HistoryQuery{})
	if len(entries) != 0 {
		t.Errorf("expected empty store after loading non-existent file, got %d entries", len(entries))
	}
}

func TestMemoryDeviceHistoryStore_LoadFromFile_InvalidJSON(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	filter := DefaultHistoryLoadFilter()

	// Create file with invalid JSON
	tmpFile := t.TempDir() + "/invalid.json"
	if err := writeTestFile(tmpFile, []byte("invalid json content")); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Load should fail with error
	err := store.LoadFromFile(tmpFile, filter)
	if err == nil {
		t.Error("LoadFromFile should error on invalid JSON")
	}
}

func TestMemoryDeviceHistoryStore_LoadFromFile_EmptyFile(t *testing.T) {
	store := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 10})
	filter := DefaultHistoryLoadFilter()

	// Create empty file
	tmpFile := t.TempDir() + "/empty.json"
	if err := writeTestFile(tmpFile, []byte("")); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Load should fail with error
	err := store.LoadFromFile(tmpFile, filter)
	if err == nil {
		t.Error("LoadFromFile should error on empty file")
	}
}

func TestMemoryDeviceHistoryStore_LoadFromFile_MultipleDevices(t *testing.T) {
	store1 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 100})
	now := time.Now().UTC()

	// Add entries for multiple devices
	for deviceID := 1; deviceID <= 5; deviceID++ {
		device := testDevice(deviceID)
		for i := 0; i < 3; i++ {
			store1.Record(DeviceHistoryEntry{
				Timestamp: now.Add(time.Duration(i) * time.Minute),
				Device:    device,
				EPC:       echonet_lite.EPCType(0x80),
				Value:     protocol.PropertyData{String: fmt.Sprintf("device%d-value%d", deviceID, i)},
				Origin:    HistoryOriginNotification,
				Settable:  true,
			})
		}
	}

	// Save
	tmpFile := t.TempDir() + "/history_multi_device_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load
	store2 := newMemoryDeviceHistoryStore(HistoryOptions{PerDeviceLimit: 100})
	filter := HistoryLoadFilter{
		Since:          24 * time.Hour,
		PerDeviceLimit: 100,
	}
	if err := store2.LoadFromFile(tmpFile, filter); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify all devices have their entries
	for deviceID := 1; deviceID <= 5; deviceID++ {
		device := testDevice(deviceID)
		entries := store2.Query(device, HistoryQuery{})
		if len(entries) != 3 {
			t.Errorf("device %d: expected 3 entries, got %d", deviceID, len(entries))
		}
	}
}

// Helper functions for tests

func readTestFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeTestFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
