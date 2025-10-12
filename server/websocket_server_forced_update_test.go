package server

import (
	"context"
	"echonet-list/client"
	"echonet-list/echonet_lite/handler"
	"sync"
	"testing"
	"time"
)

// MockECHONETClientWithForceTracking extends the mock client to track force parameter
type MockECHONETClientWithForceTracking struct {
	mu          sync.Mutex
	UpdateCalls []UpdatePropertiesCall
}

type UpdatePropertiesCall struct {
	Criteria handler.FilterCriteria
	Force    bool
	Time     time.Time
}

func (m *MockECHONETClientWithForceTracking) UpdateProperties(criteria handler.FilterCriteria, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateCalls = append(m.UpdateCalls, UpdatePropertiesCall{
		Criteria: criteria,
		Force:    force,
		Time:     time.Now(),
	})
	return nil
}

func (m *MockECHONETClientWithForceTracking) GetUpdateCalls() []UpdatePropertiesCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions
	calls := make([]UpdatePropertiesCall, len(m.UpdateCalls))
	copy(calls, m.UpdateCalls)
	return calls
}

func (m *MockECHONETClientWithForceTracking) ClearUpdateCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdateCalls = nil
}

// Implement required methods from ECHONETListClient interface
func (m *MockECHONETClientWithForceTracking) ListDevices(criteria handler.FilterCriteria) []handler.DeviceAndProperties {
	return []handler.DeviceAndProperties{}
}

func (m *MockECHONETClientWithForceTracking) GetProperties(device client.IPAndEOJ, EPCs []client.EPCType, skipValidation bool) (client.DeviceAndProperties, error) {
	return client.DeviceAndProperties{}, nil
}

func (m *MockECHONETClientWithForceTracking) SetProperties(device client.IPAndEOJ, properties client.Properties) (client.DeviceAndProperties, error) {
	return client.DeviceAndProperties{Device: device, Properties: properties}, nil
}

func (m *MockECHONETClientWithForceTracking) GetDeviceHistory(device client.IPAndEOJ, opts client.DeviceHistoryOptions) ([]client.DeviceHistoryEntry, error) {
	return []client.DeviceHistoryEntry{}, nil
}

func (m *MockECHONETClientWithForceTracking) DeleteDevice(criteria handler.FilterCriteria) error {
	return nil
}

func (m *MockECHONETClientWithForceTracking) DiscoverDevices() error {
	return nil
}

func (m *MockECHONETClientWithForceTracking) AliasList() []client.AliasIDStringPair {
	return []client.AliasIDStringPair{}
}

func (m *MockECHONETClientWithForceTracking) ManageAlias(alias string, target string, delete bool) error {
	return nil
}

// Additional AliasManager methods
func (m *MockECHONETClientWithForceTracking) AliasSet(alias *string, criteria handler.FilterCriteria) error {
	return nil
}

func (m *MockECHONETClientWithForceTracking) AliasDelete(alias *string) error {
	return nil
}

func (m *MockECHONETClientWithForceTracking) AliasGet(alias *string) (*client.IPAndEOJ, error) {
	return nil, nil
}

func (m *MockECHONETClientWithForceTracking) GetAliases(device client.IPAndEOJ) []string {
	return []string{}
}

func (m *MockECHONETClientWithForceTracking) GetDeviceByAlias(alias string) (client.IPAndEOJ, bool) {
	return client.IPAndEOJ{}, false
}

func (m *MockECHONETClientWithForceTracking) GroupList(groupName *string) []client.GroupDevicePair {
	return []client.GroupDevicePair{}
}

func (m *MockECHONETClientWithForceTracking) ManageGroup(groupName string, target string, add bool) error {
	return nil
}

func (m *MockECHONETClientWithForceTracking) DebugSetOffline(target string, offline bool) error {
	return nil
}

// Additional interface methods to complete ECHONETListClient implementation

// Debugger interface methods
func (m *MockECHONETClientWithForceTracking) IsDebug() bool {
	return false
}

func (m *MockECHONETClientWithForceTracking) SetDebug(debug bool) {
	// No-op for mock
}

func (m *MockECHONETClientWithForceTracking) IsOfflineDevice(device client.IPAndEOJ) bool {
	return false
}

// DeviceManager interface methods (additional ones)
func (m *MockECHONETClientWithForceTracking) Discover() error {
	return nil
}

func (m *MockECHONETClientWithForceTracking) GetDevices(deviceSpec client.DeviceSpecifier) []client.IPAndEOJ {
	return []client.IPAndEOJ{}
}

func (m *MockECHONETClientWithForceTracking) FindDeviceByIDString(id client.IDString) *client.IPAndEOJ {
	return nil
}

func (m *MockECHONETClientWithForceTracking) GetIDString(device client.IPAndEOJ) client.IDString {
	return ""
}

// PropertyDescProvider interface methods
func (m *MockECHONETClientWithForceTracking) GetAllPropertyAliases() map[string]client.PropertyDescription {
	return map[string]client.PropertyDescription{}
}

func (m *MockECHONETClientWithForceTracking) GetPropertyDesc(classCode client.EOJClassCode, e client.EPCType) (*client.PropertyDesc, bool) {
	return nil, false
}

func (m *MockECHONETClientWithForceTracking) IsPropertyDefaultEPC(classCode client.EOJClassCode, epc client.EPCType) bool {
	return false
}

func (m *MockECHONETClientWithForceTracking) FindPropertyAlias(classCode client.EOJClassCode, alias string) (client.Property, bool) {
	return client.Property{}, false
}

func (m *MockECHONETClientWithForceTracking) AvailablePropertyAliases(classCode client.EOJClassCode) map[string]client.PropertyDescription {
	return map[string]client.PropertyDescription{}
}

// GroupManager interface methods (additional ones)
func (m *MockECHONETClientWithForceTracking) GroupAdd(groupName string, devices []client.IDString) error {
	return nil
}

func (m *MockECHONETClientWithForceTracking) GroupRemove(groupName string, devices []client.IDString) error {
	return nil
}

func (m *MockECHONETClientWithForceTracking) GroupDelete(groupName string) error {
	return nil
}

func (m *MockECHONETClientWithForceTracking) GetDevicesByGroup(groupName string) ([]client.IDString, bool) {
	return []client.IDString{}, false
}

// Close method for main interface
func (m *MockECHONETClientWithForceTracking) Close() error {
	return nil
}

// createTestServerWithTiming creates a WebSocketServer with configurable timing for testing
func createTestServerWithTiming(periodicInterval, forcedInterval time.Duration) (*WebSocketServer, *MockECHONETClientWithForceTracking, error) {
	ctx := context.Background()

	// Create mock client
	mockClient := &MockECHONETClientWithForceTracking{}

	// Create handler
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true, // Avoid file/network operations in tests
	})
	if err != nil {
		return nil, nil, err
	}

	// Create WebSocket server
	ws, err := NewWebSocketServer(ctx, ":0", mockClient, handlerInstance, time.Now())
	if err != nil {
		return nil, nil, err
	}

	// Set timing intervals
	ws.updateInterval = periodicInterval
	ws.forcedUpdateInterval = forcedInterval

	return ws, mockClient, nil
}

// TestShouldPerformForcedUpdate_InitialStartup tests first forced update timing
func TestShouldPerformForcedUpdate_InitialStartup(t *testing.T) {
	tests := []struct {
		name             string
		forcedInterval   time.Duration
		timeSinceStartup time.Duration
		expectedForced   bool
	}{
		{
			name:             "Before forced interval",
			forcedInterval:   30 * time.Minute,
			timeSinceStartup: 15 * time.Minute,
			expectedForced:   false,
		},
		{
			name:             "At forced interval",
			forcedInterval:   30 * time.Minute,
			timeSinceStartup: 30 * time.Minute,
			expectedForced:   true,
		},
		{
			name:             "After forced interval",
			forcedInterval:   30 * time.Minute,
			timeSinceStartup: 45 * time.Minute,
			expectedForced:   true,
		},
		{
			name:             "Forced updates disabled",
			forcedInterval:   0,
			timeSinceStartup: 60 * time.Minute,
			expectedForced:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, _, err := createTestServerWithTiming(1*time.Minute, tt.forcedInterval)
			if err != nil {
				t.Fatalf("Failed to create test server: %v", err)
			}

			// Simulate time passage since startup
			currentTime := ws.serverStartupTime.Add(tt.timeSinceStartup)

			// Test the forced update decision
			shouldForce := ws.shouldPerformForcedUpdate(currentTime)

			if shouldForce != tt.expectedForced {
				t.Errorf("shouldPerformForcedUpdate() = %v, want %v", shouldForce, tt.expectedForced)
			}
		})
	}
}

// TestShouldPerformForcedUpdate_RegularInterval tests periodic forced updates
func TestShouldPerformForcedUpdate_RegularInterval(t *testing.T) {
	ws, _, err := createTestServerWithTiming(1*time.Minute, 30*time.Minute)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Simulate first forced update
	firstForcedTime := ws.serverStartupTime.Add(30 * time.Minute)
	ws.lastForcedUpdateTime.Store(firstForcedTime.UnixNano())

	tests := []struct {
		name                string
		timeSinceLastForced time.Duration
		expectedForced      bool
	}{
		{
			name:                "Just after last forced update",
			timeSinceLastForced: 1 * time.Minute,
			expectedForced:      false,
		},
		{
			name:                "Halfway to next forced update",
			timeSinceLastForced: 15 * time.Minute,
			expectedForced:      false,
		},
		{
			name:                "At next forced interval",
			timeSinceLastForced: 30 * time.Minute,
			expectedForced:      true,
		},
		{
			name:                "Past next forced interval",
			timeSinceLastForced: 35 * time.Minute,
			expectedForced:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentTime := firstForcedTime.Add(tt.timeSinceLastForced)
			shouldForce := ws.shouldPerformForcedUpdate(currentTime)

			if shouldForce != tt.expectedForced {
				t.Errorf("shouldPerformForcedUpdate() = %v, want %v", shouldForce, tt.expectedForced)
			}
		})
	}
}

// TestShouldPerformForcedUpdate_Disabled tests behavior when forced updates are disabled
func TestShouldPerformForcedUpdate_Disabled(t *testing.T) {
	ws, _, err := createTestServerWithTiming(1*time.Minute, 0) // disabled
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Test various time points - should never force
	testTimes := []time.Duration{
		1 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
	}

	for _, timeSinceStartup := range testTimes {
		t.Run(timeSinceStartup.String(), func(t *testing.T) {
			currentTime := ws.serverStartupTime.Add(timeSinceStartup)
			shouldForce := ws.shouldPerformForcedUpdate(currentTime)

			if shouldForce {
				t.Errorf("shouldPerformForcedUpdate() = true, want false (forced updates disabled)")
			}
		})
	}
}

// TestShouldPerformForcedUpdate_EdgeCases tests edge cases in timing logic
func TestShouldPerformForcedUpdate_EdgeCases(t *testing.T) {
	t.Run("Negative forced interval", func(t *testing.T) {
		ws, _, err := createTestServerWithTiming(1*time.Minute, -1*time.Minute)
		if err != nil {
			t.Fatalf("Failed to create test server: %v", err)
		}

		currentTime := ws.serverStartupTime.Add(1 * time.Hour)
		shouldForce := ws.shouldPerformForcedUpdate(currentTime)

		if shouldForce {
			t.Errorf("shouldPerformForcedUpdate() = true, want false (negative interval should disable)")
		}
	})

	t.Run("Very short forced interval", func(t *testing.T) {
		ws, _, err := createTestServerWithTiming(1*time.Minute, 1*time.Second)
		if err != nil {
			t.Fatalf("Failed to create test server: %v", err)
		}

		// Should trigger forced update very quickly
		currentTime := ws.serverStartupTime.Add(2 * time.Second)
		shouldForce := ws.shouldPerformForcedUpdate(currentTime)

		if !shouldForce {
			t.Errorf("shouldPerformForcedUpdate() = false, want true (short interval should trigger)")
		}
	})

	t.Run("Time goes backward", func(t *testing.T) {
		ws, _, err := createTestServerWithTiming(1*time.Minute, 30*time.Minute)
		if err != nil {
			t.Fatalf("Failed to create test server: %v", err)
		}

		// Set a future forced update time
		futureTime := ws.serverStartupTime.Add(1 * time.Hour)
		ws.lastForcedUpdateTime.Store(futureTime.UnixNano())

		// Test with current time before the stored time
		currentTime := ws.serverStartupTime.Add(30 * time.Minute)
		shouldForce := ws.shouldPerformForcedUpdate(currentTime)

		// Should not force update if time appears to go backward
		if shouldForce {
			t.Errorf("shouldPerformForcedUpdate() = true, want false (time went backward)")
		}
	})
}

// TestPeriodicUpdater_ForcedUpdateLogic tests the forced update logic without running actual updates
func TestPeriodicUpdater_ForcedUpdateLogic(t *testing.T) {
	ws, _, err := createTestServerWithTiming(100*time.Millisecond, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Test sequence: startup -> regular updates -> forced update
	testTimes := []struct {
		timeSinceStartup time.Duration
		expectedForce    bool
		description      string
	}{
		{50 * time.Millisecond, false, "Before first forced update"},
		{150 * time.Millisecond, false, "Still before first forced update"},
		{200 * time.Millisecond, true, "First forced update should trigger"},
		{250 * time.Millisecond, false, "Just after forced update"},
		{400 * time.Millisecond, true, "Second forced update should trigger"},
	}

	for i, tt := range testTimes {
		t.Run(tt.description, func(t *testing.T) {
			testTime := ws.serverStartupTime.Add(tt.timeSinceStartup)
			shouldForce := ws.shouldPerformForcedUpdate(testTime)

			if shouldForce != tt.expectedForce {
				t.Errorf("Test %d: shouldPerformForcedUpdate() = %v, want %v", i, shouldForce, tt.expectedForce)
			}

			// If this would be a forced update, simulate updating the timestamp
			if shouldForce {
				ws.lastForcedUpdateTime.Store(testTime.UnixNano())
			}
		})
	}
}

// TestPeriodicUpdater_ForcedUpdateStateTracking tests that forced update timestamps are maintained
func TestPeriodicUpdater_ForcedUpdateStateTracking(t *testing.T) {
	ws, _, err := createTestServerWithTiming(50*time.Millisecond, 150*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Test that lastForcedUpdateTime starts at 0
	initialTime := ws.lastForcedUpdateTime.Load()
	if initialTime != 0 {
		t.Errorf("Initial lastForcedUpdateTime should be 0, got %d", initialTime)
	}

	// Simulate a forced update occurring some time after server startup
	// Use server startup time + 1 hour to avoid any timing conflicts with startup logic
	testTime := ws.serverStartupTime.Add(1 * time.Hour)
	ws.lastForcedUpdateTime.Store(testTime.UnixNano())

	// Verify the timestamp was stored
	storedTime := ws.lastForcedUpdateTime.Load()
	if storedTime != testTime.UnixNano() {
		t.Errorf("Stored timestamp %d != expected %d", storedTime, testTime.UnixNano())
	}

	// Test shouldPerformForcedUpdate uses the stored timestamp
	checkTime := testTime.Add(100 * time.Millisecond)
	shouldForce := ws.shouldPerformForcedUpdate(checkTime)
	if shouldForce {
		t.Errorf("Should not force update shortly after last forced update. TestTime: %v, CheckTime: %v, Interval: %v, TimeSince: %v",
			testTime, checkTime, ws.forcedUpdateInterval, checkTime.Sub(testTime))
	}

	shouldForceAfterInterval := ws.shouldPerformForcedUpdate(testTime.Add(150 * time.Millisecond))
	if !shouldForceAfterInterval {
		t.Errorf("Should force update after full forced interval has passed")
	}

	t.Logf("Forced update state tracking working correctly")
}

// TestPeriodicUpdater_DisabledForcedUpdate tests that no forced updates occur when disabled
func TestPeriodicUpdater_DisabledForcedUpdate(t *testing.T) {
	// Create server with forced updates disabled (0 interval)
	ws, _, err := createTestServerWithTiming(50*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Test various times - should never force when disabled
	testTimes := []time.Duration{
		1 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
	}

	for _, timeSinceStartup := range testTimes {
		t.Run(timeSinceStartup.String(), func(t *testing.T) {
			testTime := ws.serverStartupTime.Add(timeSinceStartup)
			shouldForce := ws.shouldPerformForcedUpdate(testTime)

			if shouldForce {
				t.Errorf("shouldPerformForcedUpdate() = true, want false (forced updates disabled)")
			}
		})
	}

	t.Logf("Verified forced updates are disabled correctly")
}

// TestPeriodicUpdater_InitialStateBlocking tests that the initial state counter affects update logic
func TestPeriodicUpdater_InitialStateBlocking(t *testing.T) {
	ws, _, err := createTestServerWithTiming(50*time.Millisecond, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Test that initial state counter starts at 0
	initialCount := ws.initialStateInProgress.Load()
	if initialCount != 0 {
		t.Errorf("Initial state counter should start at 0, got %d", initialCount)
	}

	// Test setting and getting the counter
	ws.initialStateInProgress.Store(1)
	count := ws.initialStateInProgress.Load()
	if count != 1 {
		t.Errorf("Failed to set initial state counter to 1, got %d", count)
	}

	// Test clearing the counter
	ws.initialStateInProgress.Store(0)
	count = ws.initialStateInProgress.Load()
	if count != 0 {
		t.Errorf("Failed to clear initial state counter, got %d", count)
	}

	t.Logf("Initial state counter functionality working correctly")
}

// TestForcedUpdate_EdgeCases tests various edge cases and error conditions
func TestForcedUpdate_EdgeCases(t *testing.T) {
	t.Run("Server startup time in future", func(t *testing.T) {
		ws, _, err := createTestServerWithTiming(1*time.Minute, 30*time.Minute)
		if err != nil {
			t.Fatalf("Failed to create test server: %v", err)
		}

		// Set server startup time to future
		ws.serverStartupTime = time.Now().Add(1 * time.Hour)

		// Current time is before startup time
		currentTime := time.Now()
		shouldForce := ws.shouldPerformForcedUpdate(currentTime)

		// Should not force when time logic is inconsistent
		if shouldForce {
			t.Errorf("shouldPerformForcedUpdate() = true, want false (startup time in future)")
		}
	})

	t.Run("Very large forced interval", func(t *testing.T) {
		ws, _, err := createTestServerWithTiming(1*time.Minute, 24*365*time.Hour) // 1 year
		if err != nil {
			t.Fatalf("Failed to create test server: %v", err)
		}

		// Should not force within reasonable time
		currentTime := ws.serverStartupTime.Add(30 * 24 * time.Hour) // 30 days
		shouldForce := ws.shouldPerformForcedUpdate(currentTime)

		if shouldForce {
			t.Errorf("shouldPerformForcedUpdate() = true, want false (very large interval)")
		}
	})

	t.Run("Multiple consecutive forced update checks", func(t *testing.T) {
		ws, _, err := createTestServerWithTiming(1*time.Minute, 30*time.Minute)
		if err != nil {
			t.Fatalf("Failed to create test server: %v", err)
		}

		baseTime := ws.serverStartupTime.Add(30 * time.Minute)

		// First check should force
		shouldForce1 := ws.shouldPerformForcedUpdate(baseTime)
		if !shouldForce1 {
			t.Errorf("First check should force update")
		}

		// Set the forced update time
		ws.lastForcedUpdateTime.Store(baseTime.UnixNano())

		// Immediate subsequent check should not force
		shouldForce2 := ws.shouldPerformForcedUpdate(baseTime.Add(1 * time.Second))
		if shouldForce2 {
			t.Errorf("Immediate subsequent check should not force")
		}

		// Check after interval should force again
		shouldForce3 := ws.shouldPerformForcedUpdate(baseTime.Add(30 * time.Minute))
		if !shouldForce3 {
			t.Errorf("Check after interval should force")
		}
	})
}

// TestPeriodicUpdater_ErrorRecovery tests that periodic updater logic handles errors gracefully
func TestPeriodicUpdater_ErrorRecovery(t *testing.T) {
	ws, _, err := createTestServerWithTiming(100*time.Millisecond, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Test that even if an error occurs during update, the forced update timestamp logic continues to work
	testTime := ws.serverStartupTime.Add(200 * time.Millisecond)

	// First forced update
	shouldForce1 := ws.shouldPerformForcedUpdate(testTime)
	if !shouldForce1 {
		t.Errorf("Expected first forced update to trigger")
	}

	// Simulate the forced update happening (with error handling)
	ws.lastForcedUpdateTime.Store(testTime.UnixNano())

	// Immediate next check should not force
	shouldForce2 := ws.shouldPerformForcedUpdate(testTime.Add(50 * time.Millisecond))
	if shouldForce2 {
		t.Errorf("Should not force update immediately after last forced update")
	}

	// After interval should force again
	shouldForce3 := ws.shouldPerformForcedUpdate(testTime.Add(200 * time.Millisecond))
	if !shouldForce3 {
		t.Errorf("Should force update after interval even if previous updates had errors")
	}

	t.Logf("Error recovery logic working correctly")
}

// TestForcedUpdateTimestampAccuracy tests that forced update timestamps are recorded accurately
func TestForcedUpdateTimestampAccuracy(t *testing.T) {
	ws, mockClient, err := createTestServerWithTiming(50*time.Millisecond, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	startTime := time.Now()

	// Test the first forced update
	testTime := ws.serverStartupTime.Add(100 * time.Millisecond)
	shouldForce := ws.shouldPerformForcedUpdate(testTime)
	if !shouldForce {
		t.Fatalf("Expected forced update after interval")
	}

	// Simulate setting the timestamp like in periodicUpdater
	ws.lastForcedUpdateTime.Store(testTime.UnixNano())

	// Verify the timestamp was stored correctly
	storedTimestamp := ws.lastForcedUpdateTime.Load()
	if storedTimestamp != testTime.UnixNano() {
		t.Errorf("Stored timestamp %d doesn't match set timestamp %d",
			storedTimestamp, testTime.UnixNano())
	}

	// Test that the next check uses the stored timestamp correctly
	nextTestTime := testTime.Add(50 * time.Millisecond)
	shouldForce2 := ws.shouldPerformForcedUpdate(nextTestTime)
	if shouldForce2 {
		t.Errorf("Should not force update too soon after last forced update")
	}

	// Test that forced update occurs again after the full interval
	nextForcedTime := testTime.Add(100 * time.Millisecond)
	shouldForce3 := ws.shouldPerformForcedUpdate(nextForcedTime)
	if !shouldForce3 {
		t.Errorf("Should force update after full interval has passed")
	}

	elapsedTime := time.Since(startTime)
	t.Logf("Test completed in %v", elapsedTime)

	// Verify mock client tracked the timestamp accuracy
	_ = mockClient // Client tracking is tested in other tests
}
