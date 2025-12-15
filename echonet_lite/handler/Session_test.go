package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"net"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

// Helper function to create a dummy Session for testing
func createTestSession() *Session {
	// Use a dummy EOJ for testing
	// ip := net.ParseIP("192.168.0.1") // Remove unused ip variable
	eoj := echonet_lite.MakeEOJ(0x0130, 0x01) // Use MakeEOJ
	ctx, cancel := context.WithCancel(context.Background())

	// Create a minimal Session struct for testing updateFailedEPCs
	// We don't need a real connection for this specific test
	return &Session{
		mu:            sync.RWMutex{},
		eoj:           eoj,
		ctx:           ctx,
		cancel:        cancel,
		failedEPCs:    make(map[string][]echonet_lite.EPCType),
		lastAliveTime: make(map[string]time.Time),
	}
}

func TestSession_updateFailedEPCs(t *testing.T) {
	// Test device setup
	device1IP := net.ParseIP("192.168.1.10")
	device1EOJ := echonet_lite.MakeEOJ(0x0290, 0x01) // Use MakeEOJ (Air conditioner)
	device1 := echonet_lite.IPAndEOJ{IP: device1IP, EOJ: device1EOJ}
	device1Key := device1.Key()

	device2IP := net.ParseIP("192.168.1.20")
	device2EOJ := echonet_lite.MakeEOJ(0x026B, 0x01) // Use MakeEOJ (Refrigerator)
	device2 := echonet_lite.IPAndEOJ{IP: device2IP, EOJ: device2EOJ}
	device2Key := device2.Key()

	// Test EPCs
	epc80 := echonet_lite.EPCType(0x80) // Operation status
	epcB0 := echonet_lite.EPCType(0xB0) // Set temperature value
	epcB3 := echonet_lite.EPCType(0xB3) // Measured room temperature
	epcC0 := echonet_lite.EPCType(0xC0) // Measured outdoor air temperature

	// Test cases
	tests := []struct {
		name           string
		device         echonet_lite.IPAndEOJ
		initialFailed  map[string][]echonet_lite.EPCType // Initial state of session.failedEPCs
		successProps   echonet_lite.Properties
		failedEPCsIn   []echonet_lite.EPCType
		expectedFailed map[string][]echonet_lite.EPCType // Expected state of session.failedEPCs after call
		expectedReturn []echonet_lite.EPCType            // Expected return value from updateFailedEPCs
	}{
		{
			name:           "Initial Fail - Single EPC",
			device:         device1,
			initialFailed:  map[string][]echonet_lite.EPCType{},
			successProps:   echonet_lite.Properties{},
			failedEPCsIn:   []echonet_lite.EPCType{epc80},
			expectedFailed: map[string][]echonet_lite.EPCType{device1Key: {epc80}},
			expectedReturn: []echonet_lite.EPCType{epc80},
		},
		{
			name:           "Consecutive Fail - Same EPC",
			device:         device1,
			initialFailed:  map[string][]echonet_lite.EPCType{device1Key: {epc80}},
			successProps:   echonet_lite.Properties{},
			failedEPCsIn:   []echonet_lite.EPCType{epc80},
			expectedFailed: map[string][]echonet_lite.EPCType{device1Key: {epc80}},
			expectedReturn: []echonet_lite.EPCType{}, // Should not return already failed EPC
		},
		{
			name:           "Fail then Success - Single EPC",
			device:         device1,
			initialFailed:  map[string][]echonet_lite.EPCType{device1Key: {epc80}},
			successProps:   echonet_lite.Properties{{EPC: epc80, EDT: []byte{0x30}}}, // Success for epc80
			failedEPCsIn:   []echonet_lite.EPCType{},
			expectedFailed: map[string][]echonet_lite.EPCType{}, // epc80 should be removed
			expectedReturn: []echonet_lite.EPCType{},
		},
		{
			name:           "Success then Fail - Single EPC",
			device:         device1,
			initialFailed:  map[string][]echonet_lite.EPCType{},
			successProps:   echonet_lite.Properties{},
			failedEPCsIn:   []echonet_lite.EPCType{epcB0},
			expectedFailed: map[string][]echonet_lite.EPCType{device1Key: {epcB0}},
			expectedReturn: []echonet_lite.EPCType{epcB0},
		},
		{
			name:           "Mixed Success/Fail",
			device:         device1,
			initialFailed:  map[string][]echonet_lite.EPCType{device1Key: {epc80}},        // epc80 failed previously
			successProps:   echonet_lite.Properties{{EPC: epc80, EDT: []byte{0x30}}},      // epc80 succeeds now
			failedEPCsIn:   []echonet_lite.EPCType{epcB0, epcB3},                          // epcB0, epcB3 fail now
			expectedFailed: map[string][]echonet_lite.EPCType{device1Key: {epcB0, epcB3}}, // epc80 removed, epcB0, epcB3 added
			expectedReturn: []echonet_lite.EPCType{epcB0, epcB3},
		},
		{
			name:          "Multiple Devices - Initial Fail Device 2",
			device:        device2,
			initialFailed: map[string][]echonet_lite.EPCType{device1Key: {epc80}}, // Device 1 has a failed EPC
			successProps:  echonet_lite.Properties{},
			failedEPCsIn:  []echonet_lite.EPCType{epcC0},
			expectedFailed: map[string][]echonet_lite.EPCType{
				device1Key: {epc80},
				device2Key: {epcC0}, // Device 2 failure added
			},
			expectedReturn: []echonet_lite.EPCType{epcC0},
		},
		{
			name:   "Multiple Devices - Consecutive Fail Device 1, Initial Fail Device 2",
			device: device1,
			initialFailed: map[string][]echonet_lite.EPCType{
				device1Key: {epc80},
				device2Key: {epcC0},
			},
			successProps: echonet_lite.Properties{},
			failedEPCsIn: []echonet_lite.EPCType{epc80, epcB0}, // epc80 fails again, epcB0 fails first time
			expectedFailed: map[string][]echonet_lite.EPCType{
				device1Key: {epc80, epcB0}, // epcB0 added to device 1
				device2Key: {epcC0},
			},
			expectedReturn: []echonet_lite.EPCType{epcB0}, // Only the newly failed epcB0 should be returned
		},
		{
			name:   "Multiple Devices - Fail then Success Device 1",
			device: device1,
			initialFailed: map[string][]echonet_lite.EPCType{
				device1Key: {epc80, epcB0},
				device2Key: {epcC0},
			},
			successProps: echonet_lite.Properties{{EPC: epc80, EDT: []byte{0x30}}}, // epc80 succeeds
			failedEPCsIn: []echonet_lite.EPCType{},
			expectedFailed: map[string][]echonet_lite.EPCType{
				device1Key: {epcB0}, // epc80 removed from device 1
				device2Key: {epcC0},
			},
			expectedReturn: []echonet_lite.EPCType{},
		},
		{
			name:           "Fail then Success - Multiple EPCs, some remain failed",
			device:         device1,
			initialFailed:  map[string][]echonet_lite.EPCType{device1Key: {epc80, epcB0, epcB3}}, // 80, B0, B3 failed previously
			successProps:   echonet_lite.Properties{{EPC: epc80, EDT: []byte{0x30}}},             // epc80 succeeds now
			failedEPCsIn:   []echonet_lite.EPCType{},                                             // No new failures
			expectedFailed: map[string][]echonet_lite.EPCType{device1Key: {epcB0, epcB3}},        // epc80 removed, B0, B3 remain
			expectedReturn: []echonet_lite.EPCType{},
		},
		{
			name:           "Fail then Success - All previously failed EPCs succeed",
			device:         device1,
			initialFailed:  map[string][]echonet_lite.EPCType{device1Key: {epc80, epcB0}},                             // 80, B0 failed previously
			successProps:   echonet_lite.Properties{{EPC: epc80, EDT: []byte{0x30}}, {EPC: epcB0, EDT: []byte{0x19}}}, // Both succeed now
			failedEPCsIn:   []echonet_lite.EPCType{},
			expectedFailed: map[string][]echonet_lite.EPCType{}, // Device key should be removed entirely
			expectedReturn: []echonet_lite.EPCType{},
		},
		{
			name:           "No initial failures, some fail now",
			device:         device1,
			initialFailed:  map[string][]echonet_lite.EPCType{},
			successProps:   echonet_lite.Properties{{EPC: epc80, EDT: []byte{0x30}}}, // epc80 succeeds
			failedEPCsIn:   []echonet_lite.EPCType{epcB0, epcB3},                     // epcB0, epcB3 fail
			expectedFailed: map[string][]echonet_lite.EPCType{device1Key: {epcB0, epcB3}},
			expectedReturn: []echonet_lite.EPCType{epcB0, epcB3},
		},
		{
			name:           "No changes - no success, no new failures, some already failed",
			device:         device1,
			initialFailed:  map[string][]echonet_lite.EPCType{device1Key: {epc80}},
			successProps:   echonet_lite.Properties{},
			failedEPCsIn:   []echonet_lite.EPCType{},
			expectedFailed: map[string][]echonet_lite.EPCType{device1Key: {epc80}},
			expectedReturn: []echonet_lite.EPCType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new session for each test case to ensure isolation
			s := createTestSession()
			// Set the initial state for failedEPCs
			s.failedEPCs = make(map[string][]echonet_lite.EPCType)
			for k, v := range tt.initialFailed {
				// Create a copy of the slice to avoid modifying the original test data
				epcsCopy := make([]echonet_lite.EPCType, len(v))
				copy(epcsCopy, v)
				s.failedEPCs[k] = epcsCopy
			}

			// Call the function under test
			// Make a copy of failedEPCsIn as it might be modified by the function
			failedEPCsInCopy := make([]echonet_lite.EPCType, len(tt.failedEPCsIn))
			copy(failedEPCsInCopy, tt.failedEPCsIn)
			actualReturn := s.updateFailedEPCs(tt.device, tt.successProps, failedEPCsInCopy)

			// Sort slices before comparison for consistent results
			sortEPCs := func(epcs []echonet_lite.EPCType) {
				slices.SortFunc(epcs, func(a, b echonet_lite.EPCType) int { return int(a) - int(b) })
			}
			sortEPCs(actualReturn)
			for _, epcs := range s.failedEPCs {
				sortEPCs(epcs)
			}
			expectedFailedSorted := make(map[string][]echonet_lite.EPCType)
			for k, v := range tt.expectedFailed {
				sortedV := make([]echonet_lite.EPCType, len(v))
				copy(sortedV, v)
				sortEPCs(sortedV)
				expectedFailedSorted[k] = sortedV
			}
			sortEPCs(tt.expectedReturn)

			// Assert the state of session.failedEPCs
			if !reflect.DeepEqual(s.failedEPCs, expectedFailedSorted) {
				t.Errorf("failedEPCs state mismatch: got %v, want %v", s.failedEPCs, expectedFailedSorted)
			}

			// Assert the return value
			if !reflect.DeepEqual(actualReturn, tt.expectedReturn) {
				t.Errorf("return value mismatch: got %v, want %v", actualReturn, tt.expectedReturn)
			}

			// Clean up context
			s.cancel()
		})
	}
}

// TestSession_SignalDeviceAlive tests the device alive signaling mechanism
func TestSession_SignalDeviceAlive(t *testing.T) {
	s := createTestSession()
	defer s.cancel()

	// Test device
	deviceIP := net.ParseIP("192.168.1.100")
	deviceEOJ := echonet_lite.MakeEOJ(0x03B7, 0x01) // Refrigerator
	device := echonet_lite.IPAndEOJ{IP: deviceIP, EOJ: deviceEOJ}

	// Initially, lastAliveTime should be zero
	aliveTime := s.getLastAliveTime(device)
	if !aliveTime.IsZero() {
		t.Errorf("Expected zero time for untracked device, got %v", aliveTime)
	}

	// Signal the device is alive
	beforeSignal := time.Now()
	s.SignalDeviceAlive(device)
	afterSignal := time.Now()

	// Check the recorded time
	aliveTime = s.getLastAliveTime(device)
	if aliveTime.IsZero() {
		t.Error("Expected non-zero time after SignalDeviceAlive")
	}
	if aliveTime.Before(beforeSignal) || aliveTime.After(afterSignal) {
		t.Errorf("Alive time %v should be between %v and %v", aliveTime, beforeSignal, afterSignal)
	}

	// Signal again and check it updates
	time.Sleep(10 * time.Millisecond)
	s.SignalDeviceAlive(device)
	newAliveTime := s.getLastAliveTime(device)
	if !newAliveTime.After(aliveTime) {
		t.Errorf("Expected new alive time %v to be after previous %v", newAliveTime, aliveTime)
	}
}

// TestSession_makeAliveKey tests the key generation for device alive tracking
func TestSession_makeAliveKey(t *testing.T) {
	tests := []struct {
		name     string
		device   echonet_lite.IPAndEOJ
		expected string
	}{
		{
			name: "Refrigerator",
			device: echonet_lite.IPAndEOJ{
				IP:  net.ParseIP("192.168.0.83"),
				EOJ: echonet_lite.MakeEOJ(0x03B7, 0x01),
			},
			expected: "192.168.0.83:03B7:01",
		},
		{
			name: "Air Conditioner instance 2",
			device: echonet_lite.IPAndEOJ{
				IP:  net.ParseIP("10.0.0.1"),
				EOJ: echonet_lite.MakeEOJ(0x0130, 0x02),
			},
			expected: "10.0.0.1:0130:02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := makeAliveKey(tt.device)
			if key != tt.expected {
				t.Errorf("makeAliveKey() = %v, want %v", key, tt.expected)
			}
		})
	}
}

// TestSession_AliveSignalDifferentDevices tests that alive signals are tracked separately per device
func TestSession_AliveSignalDifferentDevices(t *testing.T) {
	s := createTestSession()
	defer s.cancel()

	// Two different devices
	device1 := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP("192.168.0.83"),
		EOJ: echonet_lite.MakeEOJ(0x03B7, 0x01),
	}
	device2 := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP("192.168.0.84"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 0x01),
	}

	// Signal device1
	s.SignalDeviceAlive(device1)
	time1 := s.getLastAliveTime(device1)
	time2 := s.getLastAliveTime(device2)

	if time1.IsZero() {
		t.Error("device1 should have non-zero alive time")
	}
	if !time2.IsZero() {
		t.Error("device2 should have zero alive time (not signaled)")
	}

	// Signal device2
	time.Sleep(10 * time.Millisecond)
	s.SignalDeviceAlive(device2)
	newTime1 := s.getLastAliveTime(device1)
	newTime2 := s.getLastAliveTime(device2)

	// device1's time should be unchanged
	if newTime1 != time1 {
		t.Errorf("device1 time should be unchanged, got %v, want %v", newTime1, time1)
	}
	// device2's time should be set now
	if newTime2.IsZero() {
		t.Error("device2 should have non-zero alive time after signaling")
	}
}
