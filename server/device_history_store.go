package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
)

// HistoryOrigin identifies how a history entry was produced.
type HistoryOrigin string

const (
	// HistoryOriginNotification indicates that the entry came from a property change notification.
	HistoryOriginNotification HistoryOrigin = "notification"
	// HistoryOriginSet indicates that the entry came from a successful set_properties operation.
	HistoryOriginSet HistoryOrigin = "set"
	// HistoryOriginOnline indicates that the entry came from a device online event.
	HistoryOriginOnline HistoryOrigin = "online"
	// HistoryOriginOffline indicates that the entry came from a device offline event.
	HistoryOriginOffline HistoryOrigin = "offline"
)

// DeviceHistoryEntry represents a single history item for a device.
type DeviceHistoryEntry struct {
	Timestamp time.Time
	Device    handler.IPAndEOJ
	EPC       echonet_lite.EPCType
	Value     protocol.PropertyData
	Origin    HistoryOrigin
}

// HistoryQuery specifies filters applied when fetching history entries.
type HistoryQuery struct {
	Since        time.Time
	Limit        int
	SettableOnly bool
}

// DeviceHistoryStore defines behaviour required from a history backend.
type DeviceHistoryStore interface {
	Record(entry DeviceHistoryEntry)
	Query(device handler.IPAndEOJ, query HistoryQuery) []DeviceHistoryEntry
	Clear(device handler.IPAndEOJ)
	PerDeviceLimit() int
	IsDuplicateNotification(device handler.IPAndEOJ, epc echonet_lite.EPCType, value protocol.PropertyData, within time.Duration) bool
}

// HistoryOptions configures the behaviour of the history store.
type HistoryOptions struct {
	PerDeviceLimit  int
	HistoryFilePath string // Path to history file for persistence (empty = disabled)
}

// DefaultHistoryOptions returns the default options used when none are provided.
func DefaultHistoryOptions() HistoryOptions {
	return HistoryOptions{
		PerDeviceLimit:  500,
		HistoryFilePath: "", // Disabled by default
	}
}

// newMemoryDeviceHistoryStore constructs an in-memory store.
func newMemoryDeviceHistoryStore(opts HistoryOptions) *memoryDeviceHistoryStore {
	options := DefaultHistoryOptions()
	if opts.PerDeviceLimit > 0 {
		options.PerDeviceLimit = opts.PerDeviceLimit
	}

	return &memoryDeviceHistoryStore{
		perDeviceLimit: options.PerDeviceLimit,
		data:           make(map[string][]DeviceHistoryEntry),
	}
}

type memoryDeviceHistoryStore struct {
	mu             sync.RWMutex
	perDeviceLimit int
	data           map[string][]DeviceHistoryEntry
}

func (s *memoryDeviceHistoryStore) Record(entry DeviceHistoryEntry) {
	if entry.Device.IP == nil {
		return
	}

	key := entry.Device.Key()

	s.mu.Lock()
	defer s.mu.Unlock()

	entries := append(s.data[key], entry)

	if limit := s.perDeviceLimit; limit > 0 && len(entries) > limit {
		// Drop oldest entries to enforce the cap.
		entries = entries[len(entries)-limit:]
	}

	s.data[key] = entries
}

func (s *memoryDeviceHistoryStore) Query(device handler.IPAndEOJ, query HistoryQuery) []DeviceHistoryEntry {
	key := device.Key()

	s.mu.RLock()
	entries, ok := s.data[key]
	s.mu.RUnlock()
	if !ok || len(entries) == 0 {
		return nil
	}

	limit := query.Limit
	if limit <= 0 {
		limit = len(entries)
	}

	result := make([]DeviceHistoryEntry, 0, min(limit, len(entries)))
	since := query.Since

	// Iterate from newest to oldest so the result is ordered newest-first.
	// Note: SettableOnly filtering is done by the caller after calculating settable flags
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		if !since.IsZero() && entry.Timestamp.Before(since) {
			continue
		}

		result = append(result, entry)
		if len(result) >= limit {
			break
		}
	}

	return result
}

func (s *memoryDeviceHistoryStore) Clear(device handler.IPAndEOJ) {
	key := device.Key()

	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
}

func (s *memoryDeviceHistoryStore) PerDeviceLimit() int {
	return s.perDeviceLimit
}

// IsDuplicateNotification checks if there's a recent Set operation for the same device, EPC, and value.
// This is used to avoid recording duplicate history entries when a notification follows a set operation.
func (s *memoryDeviceHistoryStore) IsDuplicateNotification(device handler.IPAndEOJ, epc echonet_lite.EPCType, value protocol.PropertyData, within time.Duration) bool {
	key := device.Key()

	s.mu.RLock()
	entries, ok := s.data[key]
	s.mu.RUnlock()

	if !ok || len(entries) == 0 {
		return false
	}

	now := time.Now().UTC()
	cutoff := now.Add(-within)

	// Check recent entries from newest to oldest
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		// Stop checking if we've gone past the time window
		if entry.Timestamp.Before(cutoff) {
			break
		}

		// Check if this is a Set operation for the same EPC
		if entry.Origin == HistoryOriginSet && entry.EPC == epc {
			// Check if the values match
			equal := propertyDataEqual(entry.Value, value)
			// Debug logging
			if !equal {
				slog.Debug("PropertyData comparison mismatch",
					"epc", fmt.Sprintf("0x%02X", epc),
					"setEDT", entry.Value.EDT,
					"notifEDT", value.EDT,
					"setString", entry.Value.String,
					"notifString", value.String,
					"setNumber", entry.Value.Number,
					"notifNumber", value.Number)
			}
			if equal {
				return true
			}
		}
	}

	return false
}

// propertyDataEqual compares two PropertyData values for equality
func propertyDataEqual(a, b protocol.PropertyData) bool {
	// Compare EDT (base64 encoded bytes)
	if a.EDT != b.EDT {
		return false
	}
	// Compare String (for alias-based properties)
	if a.String != b.String {
		return false
	}
	// Compare Number (for numeric properties)
	if (a.Number == nil) != (b.Number == nil) {
		return false
	}
	if a.Number != nil && b.Number != nil && *a.Number != *b.Number {
		return false
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// HistoryLoadFilter specifies filters applied when loading history from file
type HistoryLoadFilter struct {
	Since          time.Duration // Load entries from this duration ago (e.g., 7 * 24 * time.Hour for 1 week)
	PerDeviceLimit int           // Maximum number of entries per device to load
}

// DefaultHistoryLoadFilter returns the default filter settings for loading history
func DefaultHistoryLoadFilter() HistoryLoadFilter {
	return HistoryLoadFilter{
		Since:          7 * 24 * time.Hour, // 1 week
		PerDeviceLimit: 100,                // 100 entries per device
	}
}

// historyFileFormat represents the JSON structure for persisting history data
type historyFileFormat struct {
	Version int                                 `json:"version"`
	Data    map[string][]jsonDeviceHistoryEntry `json:"data"`
}

// jsonDeviceHistoryEntry is used for JSON marshaling/unmarshaling of DeviceHistoryEntry
type jsonDeviceHistoryEntry struct {
	Timestamp time.Time             `json:"timestamp"`
	Device    jsonIPAndEOJ          `json:"device"`
	EPC       string                `json:"epc"`
	Value     protocol.PropertyData `json:"value"`
	Origin    HistoryOrigin         `json:"origin"`
}

// jsonIPAndEOJ is used for JSON marshaling/unmarshaling of handler.IPAndEOJ
type jsonIPAndEOJ struct {
	IP  string `json:"ip"`
	EOJ string `json:"eoj"`
}

const currentHistoryFileVersion = 1

// SaveToFile saves the history data to a JSON file
func (s *memoryDeviceHistoryStore) SaveToFile(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Convert data to JSON-serializable format
	jsonData := make(map[string][]jsonDeviceHistoryEntry)
	for deviceKey, entries := range s.data {
		jsonEntries := make([]jsonDeviceHistoryEntry, 0, len(entries))
		for _, entry := range entries {
			jsonEntries = append(jsonEntries, jsonDeviceHistoryEntry{
				Timestamp: entry.Timestamp,
				Device: jsonIPAndEOJ{
					IP:  entry.Device.IP.String(),
					EOJ: entry.Device.EOJ.Specifier(),
				},
				EPC:    fmt.Sprintf("0x%02X", byte(entry.EPC)),
				Value:  entry.Value,
				Origin: entry.Origin,
			})
		}
		jsonData[deviceKey] = jsonEntries
	}

	fileData := historyFileFormat{
		Version: currentHistoryFileVersion,
		Data:    jsonData,
	}

	// Marshal to JSON
	data, err := json.Marshal(fileData)
	if err != nil {
		return fmt.Errorf("failed to marshal history data: %w", err)
	}

	// Write to temporary file
	tempFilename := filename + ".tmp"
	if err := os.WriteFile(tempFilename, data, 0644); err != nil {
		return fmt.Errorf("failed to write to temporary file %s: %w", tempFilename, err)
	}

	// Rename temporary file to actual file (atomic operation)
	if err := os.Rename(tempFilename, filename); err != nil {
		// Clean up temporary file on error
		_ = os.Remove(tempFilename)
		return fmt.Errorf("failed to rename temporary file %s to %s: %w", tempFilename, filename, err)
	}

	slog.Info("History data saved successfully", "filename", filename, "deviceCount", len(jsonData))
	return nil
}

// LoadFromFile loads the history data from a JSON file with filtering
func (s *memoryDeviceHistoryStore) LoadFromFile(filename string, filter HistoryLoadFilter) error {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		slog.Info("History file does not exist, starting with empty history", "filename", filename)
		return nil
	}

	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read history file %s: %w", filename, err)
	}

	// Parse JSON
	var fileData historyFileFormat
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to unmarshal history file %s: %w", filename, err)
	}

	// Check version
	if fileData.Version != currentHistoryFileVersion {
		slog.Warn("History file version mismatch, attempting to load anyway",
			"filename", filename,
			"fileVersion", fileData.Version,
			"expectedVersion", currentHistoryFileVersion)
	}

	// Calculate cutoff time for filtering
	cutoffTime := time.Now().Add(-filter.Since)

	// Load and filter data
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing data
	s.data = make(map[string][]DeviceHistoryEntry)

	totalLoaded := 0
	totalFiltered := 0

	for deviceKey, jsonEntries := range fileData.Data {
		// Convert JSON entries back to DeviceHistoryEntry
		// Filter by time and limit per device
		filtered := make([]DeviceHistoryEntry, 0, min(len(jsonEntries), filter.PerDeviceLimit))

		// Process entries from newest to oldest
		for i := len(jsonEntries) - 1; i >= 0 && len(filtered) < filter.PerDeviceLimit; i-- {
			jsonEntry := jsonEntries[i]

			// Skip entries older than cutoff time
			if jsonEntry.Timestamp.Before(cutoffTime) {
				totalFiltered++
				continue
			}

			// Parse IP address
			ip := net.ParseIP(jsonEntry.Device.IP)
			if ip == nil {
				slog.Warn("Invalid IP address in history entry, skipping",
					"deviceKey", deviceKey,
					"ip", jsonEntry.Device.IP)
				totalFiltered++
				continue
			}

			// Parse EOJ
			eoj, err := handler.ParseEOJString(jsonEntry.Device.EOJ)
			if err != nil {
				slog.Warn("Invalid EOJ in history entry, skipping",
					"deviceKey", deviceKey,
					"eoj", jsonEntry.Device.EOJ,
					"error", err)
				totalFiltered++
				continue
			}

			// Parse EPC
			var epc echonet_lite.EPCType
			if _, err := fmt.Sscanf(jsonEntry.EPC, "0x%02X", (*byte)(&epc)); err != nil {
				slog.Warn("Invalid EPC in history entry, skipping",
					"deviceKey", deviceKey,
					"epc", jsonEntry.EPC,
					"error", err)
				totalFiltered++
				continue
			}

			// Create entry
			entry := DeviceHistoryEntry{
				Timestamp: jsonEntry.Timestamp,
				Device: handler.IPAndEOJ{
					IP:  ip,
					EOJ: eoj,
				},
				EPC:    epc,
				Value:  jsonEntry.Value,
				Origin: jsonEntry.Origin,
			}

			filtered = append(filtered, entry)
			totalLoaded++
		}

		// Reverse filtered entries to restore chronological order (oldest first)
		for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
			filtered[i], filtered[j] = filtered[j], filtered[i]
		}

		if len(filtered) > 0 {
			s.data[deviceKey] = filtered
		}
	}

	slog.Info("History data loaded successfully",
		"filename", filename,
		"totalLoaded", totalLoaded,
		"totalFiltered", totalFiltered,
		"deviceCount", len(s.data))

	return nil
}
