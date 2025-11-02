package handler

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"echonet-list/echonet_lite"
)

func TestMemoryDeviceHistoryStore_RecordEnforcesLimit(t *testing.T) {
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 3})
	device := testDevice(1)
	base := time.Now()

	for i := 0; i < 5; i++ {
		store.Record(DeviceHistoryEntry{
			Timestamp: base.Add(time.Duration(i) * time.Minute),
			Device:    device,
			EPC:       echonet_lite.EPCType(0x80),
			Value: PropertyValue{
				String: fmt.Sprintf("value-%d", i),
			},
			Origin: HistoryOriginNotification,
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

func TestMemoryDeviceHistoryStore_Clear(t *testing.T) {
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 5})
	device := testDevice(3)

	store.Record(DeviceHistoryEntry{
		Timestamp: time.Now(),
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     PropertyValue{String: "value"},
		Origin:    HistoryOriginNotification,
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
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	device := testDevice(4)
	now := time.Now().UTC()

	// Test case 1: No history - should not be duplicate
	isDup := store.IsDuplicateNotification(device, echonet_lite.EPCType(0x80), PropertyValue{String: "on"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate when history is empty")
	}

	// Test case 2: Add a Set operation
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     PropertyValue{String: "on"},
		Origin:    HistoryOriginSet,
	})

	// Test case 3: Same device, EPC, and value within time window - should be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x80), PropertyValue{String: "on"}, 2*time.Second)
	if !isDup {
		t.Error("expected duplicate for same device, EPC, and value within time window")
	}

	// Test case 4: Different value - should not be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x80), PropertyValue{String: "off"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for different value")
	}

	// Test case 5: Different EPC - should not be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x81), PropertyValue{String: "on"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for different EPC")
	}

	// Test case 6: Add an older Set operation (outside time window)
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(-3 * time.Second),
		Device:    device,
		EPC:       echonet_lite.EPCType(0x82),
		Value:     PropertyValue{String: "value1"},
		Origin:    HistoryOriginSet,
	})

	// Test case 7: Outside time window - should not be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x82), PropertyValue{String: "value1"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for Set operation outside time window")
	}

	// Test case 8: Add a Notification (not Set) operation
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Second),
		Device:    device,
		EPC:       echonet_lite.EPCType(0x83),
		Value:     PropertyValue{String: "notif"},
		Origin:    HistoryOriginNotification,
	})

	// Test case 9: Notification origin - should not be considered as duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0x83), PropertyValue{String: "notif"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate when origin is Notification (not Set)")
	}
}

func TestMemoryDeviceHistoryStore_IsDuplicateNotification_NumericValues(t *testing.T) {
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	device := testDevice(5)
	now := time.Now().UTC()

	// Add a Set operation with numeric value
	num25 := 25
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0xB0),
		Value:     PropertyValue{Number: &num25},
		Origin:    HistoryOriginSet,
	})

	// Same numeric value - should be duplicate
	num25Copy := 25
	isDup := store.IsDuplicateNotification(device, echonet_lite.EPCType(0xB0), PropertyValue{Number: &num25Copy}, 2*time.Second)
	if !isDup {
		t.Error("expected duplicate for same numeric value")
	}

	// Different numeric value - should not be duplicate
	num26 := 26
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0xB0), PropertyValue{Number: &num26}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for different numeric value")
	}
}

func TestMemoryDeviceHistoryStore_IsDuplicateNotification_EDTValues(t *testing.T) {
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	device := testDevice(6)
	now := time.Now().UTC()

	// Add a Set operation with EDT value
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0xC0),
		Value:     PropertyValue{EDT: "AQID"}, // base64 encoded bytes
		Origin:    HistoryOriginSet,
	})

	// Same EDT value - should be duplicate
	isDup := store.IsDuplicateNotification(device, echonet_lite.EPCType(0xC0), PropertyValue{EDT: "AQID"}, 2*time.Second)
	if !isDup {
		t.Error("expected duplicate for same EDT value")
	}

	// Different EDT value - should not be duplicate
	isDup = store.IsDuplicateNotification(device, echonet_lite.EPCType(0xC0), PropertyValue{EDT: "BAUG"}, 2*time.Second)
	if isDup {
		t.Error("expected no duplicate for different EDT value")
	}
}

func testDevice(id int) IPAndEOJ {
	ip := net.ParseIP(fmt.Sprintf("192.0.2.%d", id))
	eoj := echonet_lite.MakeEOJ(echonet_lite.EOJClassCode(0x0130), echonet_lite.EOJInstanceCode(id))
	return IPAndEOJ{
		IP:  ip,
		EOJ: eoj,
	}
}

func TestMemoryDeviceHistoryStore_SaveToFile(t *testing.T) {
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	device1 := testDevice(1)
	device2 := testDevice(2)
	now := time.Now().UTC()

	// Add some test data
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device1,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     PropertyValue{String: "on"},
		Origin:    HistoryOriginSet,
	})
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Minute),
		Device:    device2,
		EPC:       echonet_lite.EPCType(0xB0),
		Value:     PropertyValue{Number: intPtr(25)},
		Origin:    HistoryOriginNotification,
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
	store1 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	device := testDevice(1)
	now := time.Now().UTC()

	store1.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     PropertyValue{String: "test-value"},
		Origin:    HistoryOriginSet,
	})

	// Save to file
	tmpFile := t.TempDir() + "/history_load_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load into new store
	store2 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	filter := HistoryLoadFilter{
		PerDeviceNonSettableLimit: 100,
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

func TestMemoryDeviceHistoryStore_LoadFromFile_CountFilter(t *testing.T) {
	store1 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 100})
	device := testDevice(1)
	now := time.Now().UTC()

	// Add 10 entries
	for i := 0; i < 10; i++ {
		store1.Record(DeviceHistoryEntry{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Device:    device,
			EPC:       echonet_lite.EPCType(0x80),
			Value:     PropertyValue{String: fmt.Sprintf("value-%d", i)},
			Origin:    HistoryOriginNotification,
		})
	}

	// Save to file
	tmpFile := t.TempDir() + "/history_count_filter_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load with count filter (only 3 most recent)
	store2 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 100})
	filter := HistoryLoadFilter{
		PerDeviceNonSettableLimit: 3, // Only 3 per device
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
	store1 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	device1 := testDevice(1)
	device2 := testDevice(2)
	now := time.Now().UTC()

	// Add diverse data types
	num25 := 25
	store1.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device1,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     PropertyValue{String: "string-value"},
		Origin:    HistoryOriginSet,
	})
	store1.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Minute),
		Device:    device1,
		EPC:       echonet_lite.EPCType(0xB0),
		Value:     PropertyValue{Number: &num25},
		Origin:    HistoryOriginNotification,
	})
	store1.Record(DeviceHistoryEntry{
		Timestamp: now.Add(2 * time.Minute),
		Device:    device2,
		EPC:       echonet_lite.EPCType(0xC0),
		Value:     PropertyValue{EDT: "AQIDBA=="},
		Origin:    HistoryOriginSet,
	})

	// Save
	tmpFile := t.TempDir() + "/history_roundtrip_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load
	store2 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	filter := HistoryLoadFilter{
		PerDeviceNonSettableLimit: 100,
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
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
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
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
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
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
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
	store1 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 100})
	now := time.Now().UTC()

	// Add entries for multiple devices
	for deviceID := 1; deviceID <= 5; deviceID++ {
		device := testDevice(deviceID)
		for i := 0; i < 3; i++ {
			store1.Record(DeviceHistoryEntry{
				Timestamp: now.Add(time.Duration(i) * time.Minute),
				Device:    device,
				EPC:       echonet_lite.EPCType(0x80),
				Value:     PropertyValue{String: fmt.Sprintf("device%d-value%d", deviceID, i)},
				Origin:    HistoryOriginNotification,
			})
		}
	}

	// Save
	tmpFile := t.TempDir() + "/history_multi_device_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load
	store2 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 100})
	filter := HistoryLoadFilter{
		PerDeviceNonSettableLimit: 100,
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

// TestMemoryDeviceHistoryStore_SettableAndNonSettableLimits tests separate limits for settable and non-settable properties
func TestMemoryDeviceHistoryStore_SettableAndNonSettableLimits(t *testing.T) {
	store := NewMemoryDeviceHistoryStore(HistoryOptions{
		PerDeviceSettableLimit:    3, // settable limit
		PerDeviceNonSettableLimit: 2, // non-settable limit
	})
	device := testDevice(1)
	base := time.Now().UTC()

	// Record 5 settable entries (should keep only newest 3)
	for i := 0; i < 5; i++ {
		store.Record(DeviceHistoryEntry{
			Timestamp: base.Add(time.Duration(i) * time.Minute),
			Device:    device,
			EPC:       echonet_lite.EPCType(0x80),
			Value:     PropertyValue{String: fmt.Sprintf("settable-%d", i)},
			Origin:    HistoryOriginSet,
			Settable:  true,
		})
	}

	// Record 4 non-settable entries (should keep only newest 2)
	for i := 0; i < 4; i++ {
		store.Record(DeviceHistoryEntry{
			Timestamp: base.Add(time.Duration(10+i) * time.Minute),
			Device:    device,
			EPC:       echonet_lite.EPCType(0xB0),
			Value:     PropertyValue{String: fmt.Sprintf("non-settable-%d", i)},
			Origin:    HistoryOriginNotification,
			Settable:  false,
		})
	}

	// Query all entries
	entries := store.Query(device, HistoryQuery{})

	// Should have 3 settable + 2 non-settable = 5 entries total
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries (3 settable + 2 non-settable), got %d", len(entries))
	}

	// Count settable and non-settable entries
	settableCount := 0
	nonSettableCount := 0
	for _, entry := range entries {
		if entry.Settable {
			settableCount++
		} else {
			nonSettableCount++
		}
	}

	if settableCount != 3 {
		t.Errorf("expected 3 settable entries, got %d", settableCount)
	}
	if nonSettableCount != 2 {
		t.Errorf("expected 2 non-settable entries, got %d", nonSettableCount)
	}

	// Verify that oldest entries were dropped
	// Newest settable entries should be: settable-4, settable-3, settable-2
	// Newest non-settable entries should be: non-settable-3, non-settable-2
	settableValues := []string{}
	nonSettableValues := []string{}
	for _, entry := range entries {
		if entry.Settable {
			settableValues = append(settableValues, entry.Value.String)
		} else {
			nonSettableValues = append(nonSettableValues, entry.Value.String)
		}
	}

	expectedSettable := []string{"settable-4", "settable-3", "settable-2"}
	expectedNonSettable := []string{"non-settable-3", "non-settable-2"}

	if !stringSlicesEqual(settableValues, expectedSettable) {
		t.Errorf("settable values mismatch: expected %v, got %v", expectedSettable, settableValues)
	}
	if !stringSlicesEqual(nonSettableValues, expectedNonSettable) {
		t.Errorf("non-settable values mismatch: expected %v, got %v", expectedNonSettable, nonSettableValues)
	}

	// Verify chronological order (newest first)
	// Expected order: settable-4 (4min), settable-3 (3min), settable-2 (2min),
	//                 non-settable-3 (13min), non-settable-2 (12min)
	// However, since merged entries are sorted newest-first, the actual order should be:
	// non-settable-3 (13min), non-settable-2 (12min), settable-4 (4min), settable-3 (3min), settable-2 (2min)
	for i := 0; i < len(entries)-1; i++ {
		if entries[i].Timestamp.Before(entries[i+1].Timestamp) {
			t.Errorf("entries not in chronological order (newest first) at index %d: %v is before %v",
				i, entries[i].Timestamp, entries[i+1].Timestamp)
		}
	}
}

// TestMemoryDeviceHistoryStore_SettableSaveLoad tests that settable flag is preserved in save/load
func TestMemoryDeviceHistoryStore_SettableSaveLoad(t *testing.T) {
	store1 := NewMemoryDeviceHistoryStore(HistoryOptions{
		PerDeviceSettableLimit:    10,
		PerDeviceNonSettableLimit: 10,
	})
	device := testDevice(1)
	now := time.Now().UTC()

	// Add settable and non-settable entries
	store1.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     PropertyValue{String: "settable-value"},
		Origin:    HistoryOriginSet,
		Settable:  true,
	})
	store1.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0xB0),
		Value:     PropertyValue{String: "non-settable-value"},
		Origin:    HistoryOriginNotification,
		Settable:  false,
	})

	// Save to file
	tmpFile := t.TempDir() + "/settable_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load into new store
	store2 := NewMemoryDeviceHistoryStore(HistoryOptions{
		PerDeviceSettableLimit:    10,
		PerDeviceNonSettableLimit: 10,
	})
	filter := HistoryLoadFilter{
		PerDeviceSettableLimit:    100,
		PerDeviceNonSettableLimit: 100,
	}
	if err := store2.LoadFromFile(tmpFile, filter); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify settable flags are preserved
	entries := store2.Query(device, HistoryQuery{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries after load, got %d", len(entries))
	}

	// First entry (newest) should be non-settable
	if entries[0].Settable != false {
		t.Errorf("expected first entry to be non-settable, got settable=%v", entries[0].Settable)
	}
	if entries[0].Value.String != "non-settable-value" {
		t.Errorf("expected 'non-settable-value', got '%s'", entries[0].Value.String)
	}

	// Second entry should be settable
	if entries[1].Settable != true {
		t.Errorf("expected second entry to be settable, got settable=%v", entries[1].Settable)
	}
	if entries[1].Value.String != "settable-value" {
		t.Errorf("expected 'settable-value', got '%s'", entries[1].Value.String)
	}
}

// Helper functions for tests

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

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

// TestMemoryDeviceHistoryStore_RecordEventHistory tests recording of online/offline events
func TestMemoryDeviceHistoryStore_RecordEventHistory(t *testing.T) {
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	device := testDevice(1)
	now := time.Now().UTC()

	// Record an offline event (EPC should be 0 for events)
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0), // No EPC for event entries
		Value:     PropertyValue{},         // Empty value for events
		Origin:    HistoryOriginOffline,
	})

	// Record an online event
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0), // No EPC for event entries
		Value:     PropertyValue{},         // Empty value for events
		Origin:    HistoryOriginOnline,
	})

	// Query all entries (including events)
	entries := store.Query(device, HistoryQuery{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify newest first: online, offline
	if entries[0].Origin != HistoryOriginOnline {
		t.Errorf("expected first entry to be online event, got %s", entries[0].Origin)
	}
	if entries[1].Origin != HistoryOriginOffline {
		t.Errorf("expected second entry to be offline event, got %s", entries[1].Origin)
	}

	// Verify EPC is 0 for events
	for i, entry := range entries {
		if entry.EPC != echonet_lite.EPCType(0) {
			t.Errorf("entry %d: expected EPC to be 0 for event, got 0x%02X", i, entry.EPC)
		}
	}
}

// TestMemoryDeviceHistoryStore_MixedPropertyAndEventHistory tests mixed property changes and events
func TestMemoryDeviceHistoryStore_MixedPropertyAndEventHistory(t *testing.T) {
	store := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	device := testDevice(1)
	now := time.Now().UTC()

	// Record a property change
	store.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     PropertyValue{String: "on"},
		Origin:    HistoryOriginSet,
	})

	// Record an offline event
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0),
		Value:     PropertyValue{},
		Origin:    HistoryOriginOffline,
	})

	// Record an online event
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(2 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0),
		Value:     PropertyValue{},
		Origin:    HistoryOriginOnline,
	})

	// Record another property change
	store.Record(DeviceHistoryEntry{
		Timestamp: now.Add(3 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0x80),
		Value:     PropertyValue{String: "off"},
		Origin:    HistoryOriginNotification,
	})

	// Query all entries (SettableOnly filtering is now done by the handler, not by storage)
	allEntries := store.Query(device, HistoryQuery{})
	if len(allEntries) != 4 {
		t.Fatalf("expected 4 entries total, got %d", len(allEntries))
	}

	// Verify Query returns all entries without filtering (filtering is done by handler layer)
	entriesForFiltering := store.Query(device, HistoryQuery{})
	if len(entriesForFiltering) != 4 {
		t.Fatalf("expected 4 entries (no filtering by Query), got %d", len(entriesForFiltering))
	}

	// Verify that events and property changes are both included
	eventCount := 0
	propertyCount := 0
	for _, entry := range entriesForFiltering {
		if entry.Origin == HistoryOriginOnline || entry.Origin == HistoryOriginOffline {
			eventCount++
		} else {
			propertyCount++
		}
	}
	if eventCount != 2 {
		t.Errorf("expected 2 event entries, got %d", eventCount)
	}
	if propertyCount != 2 {
		t.Errorf("expected 2 property entries, got %d", propertyCount)
	}
}

// TestMemoryDeviceHistoryStore_EventHistorySaveLoad tests saving and loading event history
func TestMemoryDeviceHistoryStore_EventHistorySaveLoad(t *testing.T) {
	store1 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	device := testDevice(1)
	now := time.Now().UTC()

	// Add event history
	store1.Record(DeviceHistoryEntry{
		Timestamp: now,
		Device:    device,
		EPC:       echonet_lite.EPCType(0),
		Value:     PropertyValue{},
		Origin:    HistoryOriginOffline,
	})
	store1.Record(DeviceHistoryEntry{
		Timestamp: now.Add(1 * time.Minute),
		Device:    device,
		EPC:       echonet_lite.EPCType(0),
		Value:     PropertyValue{},
		Origin:    HistoryOriginOnline,
	})

	// Save to file
	tmpFile := t.TempDir() + "/event_history_test.json"
	if err := store1.SaveToFile(tmpFile); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load into new store
	store2 := NewMemoryDeviceHistoryStore(HistoryOptions{PerDeviceNonSettableLimit: 10})
	filter := HistoryLoadFilter{
		PerDeviceNonSettableLimit: 100,
	}
	if err := store2.LoadFromFile(tmpFile, filter); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify loaded event history
	entries := store2.Query(device, HistoryQuery{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 event entries after load, got %d", len(entries))
	}

	// Verify online event (newest first)
	if entries[0].Origin != HistoryOriginOnline {
		t.Errorf("expected online event, got %s", entries[0].Origin)
	}
	if entries[0].EPC != echonet_lite.EPCType(0) {
		t.Errorf("expected EPC 0 for online event, got 0x%02X", entries[0].EPC)
	}

	// Verify offline event
	if entries[1].Origin != HistoryOriginOffline {
		t.Errorf("expected offline event, got %s", entries[1].Origin)
	}
	if entries[1].EPC != echonet_lite.EPCType(0) {
		t.Errorf("expected EPC 0 for offline event, got 0x%02X", entries[1].EPC)
	}
}
