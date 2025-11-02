package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"echonet-list/echonet_lite"
)

// PropertyValue represents the value of a property in history.
// This is similar to protocol.PropertyData but defined in handler package to avoid import cycles.
type PropertyValue struct {
	EDT    string `json:"EDT,omitempty"`    // Base64 encoded EDT, omitted if empty
	String string `json:"string,omitempty"` // String representation of EDT, omitted if empty
	Number *int   `json:"number,omitempty"` // Numeric value, omitted if nil
}

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
	Device    IPAndEOJ
	EPC       echonet_lite.EPCType
	Value     PropertyValue
	Origin    HistoryOrigin
	Settable  bool // Whether the property is settable (writable)
}

// HistoryQuery specifies filters applied when fetching history entries.
type HistoryQuery struct {
	Limit int
}

// DeviceHistoryStore defines behaviour required from a history backend.
type DeviceHistoryStore interface {
	Record(entry DeviceHistoryEntry)
	Query(device IPAndEOJ, query HistoryQuery) []DeviceHistoryEntry
	Clear(device IPAndEOJ)
	// PerDeviceTotalLimit returns the maximum total number of history entries per device.
	// For stores with separate settable/non-settable limits, this returns the sum of both limits.
	PerDeviceTotalLimit() int
	IsDuplicateNotification(device IPAndEOJ, epc echonet_lite.EPCType, value PropertyValue, within time.Duration) bool
	SaveToFile(filename string) error
	LoadFromFile(filename string, filter HistoryLoadFilter) error
}

// HistoryOptions configures the behaviour of the history store.
type HistoryOptions struct {
	PerDeviceSettableLimit    int    // Maximum number of settable property history per device
	PerDeviceNonSettableLimit int    // Maximum number of non-settable property history per device
	HistoryFilePath           string // Path to history file for persistence (empty = disabled)
}

// DefaultHistoryOptions returns the default options used when none are provided.
func DefaultHistoryOptions() HistoryOptions {
	return HistoryOptions{
		PerDeviceSettableLimit:    200, // Settable properties (operations)
		PerDeviceNonSettableLimit: 100, // Non-settable properties (notifications)
		HistoryFilePath:           "",  // Disabled by default
	}
}

// NewMemoryDeviceHistoryStore constructs an in-memory store.
func NewMemoryDeviceHistoryStore(opts HistoryOptions) DeviceHistoryStore {
	options := DefaultHistoryOptions()
	if opts.PerDeviceSettableLimit > 0 {
		options.PerDeviceSettableLimit = opts.PerDeviceSettableLimit
	}
	if opts.PerDeviceNonSettableLimit > 0 {
		options.PerDeviceNonSettableLimit = opts.PerDeviceNonSettableLimit
	}

	return &memoryDeviceHistoryStore{
		perDeviceSettableLimit: options.PerDeviceSettableLimit,
		perDeviceLimit:         options.PerDeviceNonSettableLimit,
		settableData:           make(map[string][]DeviceHistoryEntry),
		nonSettableData:        make(map[string][]DeviceHistoryEntry),
	}
}

type memoryDeviceHistoryStore struct {
	mu                     sync.RWMutex
	perDeviceSettableLimit int
	perDeviceLimit         int
	settableData           map[string][]DeviceHistoryEntry
	nonSettableData        map[string][]DeviceHistoryEntry
}

func (s *memoryDeviceHistoryStore) Record(entry DeviceHistoryEntry) {
	if entry.Device.IP == nil {
		return
	}

	key := entry.Device.Key()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to appropriate map based on settable flag
	if entry.Settable {
		entries := append(s.settableData[key], entry)
		entries = s.trimToLimit(entries, s.perDeviceSettableLimit)
		s.settableData[key] = entries
	} else {
		entries := append(s.nonSettableData[key], entry)
		entries = s.trimToLimit(entries, s.perDeviceLimit)
		s.nonSettableData[key] = entries
	}
}

func (s *memoryDeviceHistoryStore) Query(device IPAndEOJ, query HistoryQuery) []DeviceHistoryEntry {
	key := device.Key()

	s.mu.RLock()
	settableEntries := s.settableData[key]
	nonSettableEntries := s.nonSettableData[key]
	s.mu.RUnlock()

	// Both slices are already sorted (oldest first), so merge them efficiently
	allEntries := mergeEntriesByTimestamp(settableEntries, nonSettableEntries)

	if len(allEntries) == 0 {
		return nil
	}

	limit := query.Limit
	if limit <= 0 {
		limit = len(allEntries)
	}

	result := make([]DeviceHistoryEntry, 0, min(limit, len(allEntries)))

	// Iterate from newest to oldest so the result is ordered newest-first
	for i := len(allEntries) - 1; i >= 0; i-- {
		entry := allEntries[i]

		result = append(result, entry)
		if len(result) >= limit {
			break
		}
	}

	return result
}

func (s *memoryDeviceHistoryStore) Clear(device IPAndEOJ) {
	key := device.Key()

	s.mu.Lock()
	delete(s.settableData, key)
	delete(s.nonSettableData, key)
	s.mu.Unlock()
}

// PerDeviceTotalLimit returns the maximum total number of history entries per device.
// This is the sum of settable and non-settable limits.
func (s *memoryDeviceHistoryStore) PerDeviceTotalLimit() int {
	return s.perDeviceSettableLimit + s.perDeviceLimit
}

// IsDuplicateNotification checks if there's a recent Set operation for the same device, EPC, and value.
// This is used to avoid recording duplicate history entries when a notification follows a set operation.
func (s *memoryDeviceHistoryStore) IsDuplicateNotification(device IPAndEOJ, epc echonet_lite.EPCType, value PropertyValue, within time.Duration) bool {
	key := device.Key()

	s.mu.RLock()
	settableEntries := s.settableData[key]
	nonSettableEntries := s.nonSettableData[key]
	s.mu.RUnlock()

	// Both slices are already sorted (oldest first), so merge them efficiently
	allEntries := mergeEntriesByTimestamp(settableEntries, nonSettableEntries)

	if len(allEntries) == 0 {
		return false
	}

	now := time.Now().UTC()
	cutoff := now.Add(-within)

	// Check recent entries from newest to oldest
	for i := len(allEntries) - 1; i >= 0; i-- {
		entry := allEntries[i]

		// Stop checking if we've gone past the time window
		if entry.Timestamp.Before(cutoff) {
			break
		}

		// Check if this is a Set operation for the same EPC
		if entry.Origin == HistoryOriginSet && entry.EPC == epc {
			// Check if the values match
			equal := propertyValueEqual(entry.Value, value)
			// Debug logging
			if !equal {
				slog.Debug("PropertyValue comparison mismatch",
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

// propertyValueEqual compares two PropertyValue values for equality
func propertyValueEqual(a, b PropertyValue) bool {
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

// mergeEntriesByTimestamp merges two already-sorted slices into a single sorted slice.
// Both input slices must be sorted in ascending order (oldest first).
// This is O(n) which is more efficient than insertion sort O(n²) for merging sorted data.
//
// Note: The returned slice contains the same DeviceHistoryEntry structs as the input slices
// (not copies). This is safe because DeviceHistoryEntry instances are never modified after
// creation (immutable by convention).
func mergeEntriesByTimestamp(a, b []DeviceHistoryEntry) []DeviceHistoryEntry {
	result := make([]DeviceHistoryEntry, 0, len(a)+len(b))
	i, j := 0, 0

	// Merge while both slices have elements
	for i < len(a) && j < len(b) {
		if a[i].Timestamp.Before(b[j].Timestamp) || a[i].Timestamp.Equal(b[j].Timestamp) {
			result = append(result, a[i])
			i++
		} else {
			result = append(result, b[j])
			j++
		}
	}

	// Append remaining elements from either slice
	result = append(result, a[i:]...)
	result = append(result, b[j:]...)
	return result
}

// reverseEntries reverses a slice of history entries in place.
func reverseEntries(entries []DeviceHistoryEntry) {
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
}

// trimToLimit returns a slice of entries limited to the configured per-device limit.
// If customLimit is provided and > 0, it overrides the store's perDeviceLimit.
// If limit > 0 and len(entries) > limit, oldest entries are dropped to enforce the cap.
func (s *memoryDeviceHistoryStore) trimToLimit(entries []DeviceHistoryEntry, customLimit ...int) []DeviceHistoryEntry {
	limit := s.perDeviceLimit
	if len(customLimit) > 0 && customLimit[0] > 0 {
		limit = customLimit[0]
	}
	if limit > 0 && len(entries) > limit {
		// Drop oldest entries to enforce the cap
		return entries[len(entries)-limit:]
	}
	return entries
}

// HistoryLoadFilter specifies filters applied when loading history from file
type HistoryLoadFilter struct {
	PerDeviceSettableLimit    int // Maximum number of settable entries per device to load
	PerDeviceNonSettableLimit int // Maximum number of non-settable entries per device to load
}

// DefaultHistoryLoadFilter returns the default filter settings for loading history
func DefaultHistoryLoadFilter() HistoryLoadFilter {
	opts := DefaultHistoryOptions()
	return HistoryLoadFilter{
		PerDeviceSettableLimit:    opts.PerDeviceSettableLimit,
		PerDeviceNonSettableLimit: opts.PerDeviceNonSettableLimit,
	}
}

// historyFileFormat represents the JSON structure for persisting history data
type historyFileFormat struct {
	Version int                                 `json:"version"`
	Data    map[string][]jsonDeviceHistoryEntry `json:"data"`
}

// jsonDeviceHistoryEntry is used for JSON marshaling/unmarshaling of DeviceHistoryEntry
type jsonDeviceHistoryEntry struct {
	Timestamp time.Time     `json:"timestamp"`
	Device    jsonIPAndEOJ  `json:"device"`
	EPC       string        `json:"epc"`
	Value     PropertyValue `json:"value"`
	Origin    HistoryOrigin `json:"origin"`
	Settable  bool          `json:"settable,omitempty"` // Whether the property is settable
}

// jsonIPAndEOJ is used for JSON marshaling/unmarshaling of IPAndEOJ
type jsonIPAndEOJ struct {
	IP  string `json:"ip"`
	EOJ string `json:"eoj"`
}

const currentHistoryFileVersion = 1

// SaveToFile saves the history data to a JSON file
func (s *memoryDeviceHistoryStore) SaveToFile(filename string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Merge settable and non-settable data for each device
	allDeviceKeys := make(map[string]bool)
	for key := range s.settableData {
		allDeviceKeys[key] = true
	}
	for key := range s.nonSettableData {
		allDeviceKeys[key] = true
	}

	// Convert data to JSON-serializable format
	jsonData := make(map[string][]jsonDeviceHistoryEntry)
	for deviceKey := range allDeviceKeys {
		settableEntries := s.settableData[deviceKey]
		nonSettableEntries := s.nonSettableData[deviceKey]

		// Both slices are already sorted (oldest first), so merge them efficiently
		allEntries := mergeEntriesByTimestamp(settableEntries, nonSettableEntries)

		jsonEntries := make([]jsonDeviceHistoryEntry, 0, len(allEntries))
		for _, entry := range allEntries {
			jsonEntries = append(jsonEntries, jsonDeviceHistoryEntry{
				Timestamp: entry.Timestamp,
				Device: jsonIPAndEOJ{
					IP:  entry.Device.IP.String(),
					EOJ: entry.Device.EOJ.Specifier(),
				},
				EPC:      fmt.Sprintf("0x%02X", byte(entry.EPC)),
				Value:    entry.Value,
				Origin:   entry.Origin,
				Settable: entry.Settable,
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

	// Load and filter data
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing data
	s.settableData = make(map[string][]DeviceHistoryEntry)
	s.nonSettableData = make(map[string][]DeviceHistoryEntry)

	totalLoaded := 0
	totalFiltered := 0

	for deviceKey, jsonEntries := range fileData.Data {
		// Separate settable and non-settable entries
		settableFiltered := make([]DeviceHistoryEntry, 0)
		nonSettableFiltered := make([]DeviceHistoryEntry, 0)

		// Process entries from newest to oldest
		for i := len(jsonEntries) - 1; i >= 0; i-- {
			jsonEntry := jsonEntries[i]

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
			eoj, err := ParseEOJString(jsonEntry.Device.EOJ)
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
				Device: IPAndEOJ{
					IP:  ip,
					EOJ: eoj,
				},
				EPC:      epc,
				Value:    jsonEntry.Value,
				Origin:   jsonEntry.Origin,
				Settable: jsonEntry.Settable,
			}

			// Separate by settable flag
			if entry.Settable {
				settableFiltered = append(settableFiltered, entry)
			} else {
				nonSettableFiltered = append(nonSettableFiltered, entry)
			}
			totalLoaded++
		}

		// Reverse to chronological order (oldest first) before trimming
		reverseEntries(settableFiltered)
		reverseEntries(nonSettableFiltered)

		// Apply per-device limits separately
		settableFiltered = s.trimToLimit(settableFiltered, filter.PerDeviceSettableLimit)
		nonSettableFiltered = s.trimToLimit(nonSettableFiltered, filter.PerDeviceNonSettableLimit)

		if len(settableFiltered) > 0 {
			s.settableData[deviceKey] = settableFiltered
		}
		if len(nonSettableFiltered) > 0 {
			s.nonSettableData[deviceKey] = nonSettableFiltered
		}
	}

	// Count unique devices across both maps
	allDeviceKeys := make(map[string]bool)
	for key := range s.settableData {
		allDeviceKeys[key] = true
	}
	for key := range s.nonSettableData {
		allDeviceKeys[key] = true
	}

	slog.Info("History data loaded successfully",
		"filename", filename,
		"totalLoaded", totalLoaded,
		"totalFiltered", totalFiltered,
		"deviceCount", len(allDeviceKeys))

	return nil
}

// PropertyValueFromEDT creates a PropertyValue from EDT bytes
func PropertyValueFromEDT(edt []byte, epc echonet_lite.EPCType, classCode echonet_lite.EOJClassCode) PropertyValue {
	// Get property description
	desc, _ := echonet_lite.GetPropertyDesc(classCode, epc)
	if desc == nil {
		// No description, store as base64 EDT
		return PropertyValue{
			EDT: encodeEDTToBase64(edt),
		}
	}

	// Try to get string alias
	if len(desc.Aliases) > 0 {
		// Find the reverse mapping (EDT bytes -> alias string)
		for alias, aliasEDT := range desc.Aliases {
			if bytes.Equal(edt, aliasEDT) {
				return PropertyValue{
					EDT:    encodeEDTToBase64(edt),
					String: alias,
				}
			}
		}
	}

	// Try to decode as number if decoder is available
	if desc.Decoder != nil {
		if decoded, ok := desc.Decoder.ToString(edt); ok {
			// Try to parse as integer
			if num, err := strconv.Atoi(decoded); err == nil {
				return PropertyValue{
					EDT:    encodeEDTToBase64(edt),
					Number: &num,
				}
			}
			// If not a number, store as string
			return PropertyValue{
				EDT:    encodeEDTToBase64(edt),
				String: decoded,
			}
		}
	}

	// Fallback to EDT only
	return PropertyValue{
		EDT: encodeEDTToBase64(edt),
	}
}

// encodeEDTToBase64 encodes EDT bytes to base64 string using standard library
func encodeEDTToBase64(edt []byte) string {
	if len(edt) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(edt)
}
