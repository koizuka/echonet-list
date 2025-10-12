package server

import (
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
)

// DeviceHistoryEntry represents a single history item for a device.
type DeviceHistoryEntry struct {
	Timestamp time.Time
	Device    handler.IPAndEOJ
	EPC       echonet_lite.EPCType
	Value     protocol.PropertyData
	Origin    HistoryOrigin
	Settable  bool
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
}

// HistoryOptions configures the behaviour of the history store.
type HistoryOptions struct {
	PerDeviceLimit int
}

// DefaultHistoryOptions returns the default options used when none are provided.
func DefaultHistoryOptions() HistoryOptions {
	return HistoryOptions{
		PerDeviceLimit: 500,
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
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]

		if !since.IsZero() && entry.Timestamp.Before(since) {
			continue
		}
		if query.SettableOnly && !entry.Settable {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
