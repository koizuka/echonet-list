package server

import (
	"sync"
	"time"
)

// TimeProvider provides an abstraction for time-related operations
type TimeProvider interface {
	// After returns a channel that will send the current time after the duration has elapsed
	After(d time.Duration) <-chan time.Time
}

// RealTimeProvider implements TimeProvider using real time
type RealTimeProvider struct{}

func (r *RealTimeProvider) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// MockTimeProvider implements TimeProvider for testing
type MockTimeProvider struct {
	mu      sync.Mutex
	timers  []*mockTimer
	nowTime time.Time
}

type mockTimer struct {
	deadline time.Time
	ch       chan time.Time
	fired    bool
}

// NewMockTimeProvider creates a new MockTimeProvider
func NewMockTimeProvider() *MockTimeProvider {
	return &MockTimeProvider{
		nowTime: time.Now(),
		timers:  make([]*mockTimer, 0),
	}
}

func (m *MockTimeProvider) After(d time.Duration) <-chan time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan time.Time, 1)
	timer := &mockTimer{
		deadline: m.nowTime.Add(d),
		ch:       ch,
		fired:    false,
	}
	m.timers = append(m.timers, timer)

	// Check if it should fire immediately
	m.checkTimers()

	return ch
}

// Advance advances the mock time by the given duration
func (m *MockTimeProvider) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nowTime = m.nowTime.Add(d)
	m.checkTimers()
}

// checkTimers checks and fires any timers that have passed their deadline
func (m *MockTimeProvider) checkTimers() {
	for _, timer := range m.timers {
		if !timer.fired && !m.nowTime.Before(timer.deadline) {
			timer.fired = true
			// Send in a non-blocking way
			select {
			case timer.ch <- m.nowTime:
			default:
			}
			close(timer.ch)
		}
	}
}
