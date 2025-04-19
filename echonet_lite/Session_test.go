package echonet_lite

import (
	"context"
	"net"
	"reflect"
	"slices"
	"sync"
	"testing"
)

// Helper function to create a dummy Session for testing
func createTestSession() *Session {
	// Use a dummy EOJ for testing
	// ip := net.ParseIP("192.168.0.1") // Remove unused ip variable
	eoj := MakeEOJ(0x0130, 0x01) // Use MakeEOJ
	ctx, cancel := context.WithCancel(context.Background())

	// Create a minimal Session struct for testing updateFailedEPCs
	// We don't need a real connection for this specific test
	return &Session{
		mu:         sync.RWMutex{},
		eoj:        eoj,
		ctx:        ctx,
		cancel:     cancel,
		failedEPCs: make(map[string][]EPCType),
	}
}

func TestSession_updateFailedEPCs(t *testing.T) {
	// Test device setup
	device1IP := net.ParseIP("192.168.1.10")
	device1EOJ := MakeEOJ(0x0290, 0x01) // Use MakeEOJ (Air conditioner)
	device1 := IPAndEOJ{IP: device1IP, EOJ: device1EOJ}
	device1Key := device1.Key()

	device2IP := net.ParseIP("192.168.1.20")
	device2EOJ := MakeEOJ(0x026B, 0x01) // Use MakeEOJ (Refrigerator)
	device2 := IPAndEOJ{IP: device2IP, EOJ: device2EOJ}
	device2Key := device2.Key()

	// Test EPCs
	epc80 := EPCType(0x80) // Operation status
	epcB0 := EPCType(0xB0) // Set temperature value
	epcB3 := EPCType(0xB3) // Measured room temperature
	epcC0 := EPCType(0xC0) // Measured outdoor air temperature

	// Test cases
	tests := []struct {
		name           string
		device         IPAndEOJ
		initialFailed  map[string][]EPCType // Initial state of session.failedEPCs
		successProps   Properties
		failedEPCsIn   []EPCType
		expectedFailed map[string][]EPCType // Expected state of session.failedEPCs after call
		expectedReturn []EPCType            // Expected return value from updateFailedEPCs
	}{
		{
			name:           "Initial Fail - Single EPC",
			device:         device1,
			initialFailed:  map[string][]EPCType{},
			successProps:   Properties{},
			failedEPCsIn:   []EPCType{epc80},
			expectedFailed: map[string][]EPCType{device1Key: {epc80}},
			expectedReturn: []EPCType{epc80},
		},
		{
			name:           "Consecutive Fail - Same EPC",
			device:         device1,
			initialFailed:  map[string][]EPCType{device1Key: {epc80}},
			successProps:   Properties{},
			failedEPCsIn:   []EPCType{epc80},
			expectedFailed: map[string][]EPCType{device1Key: {epc80}},
			expectedReturn: []EPCType{}, // Should not return already failed EPC
		},
		{
			name:           "Fail then Success - Single EPC",
			device:         device1,
			initialFailed:  map[string][]EPCType{device1Key: {epc80}},
			successProps:   Properties{{EPC: epc80, EDT: []byte{0x30}}}, // Success for epc80
			failedEPCsIn:   []EPCType{},
			expectedFailed: map[string][]EPCType{}, // epc80 should be removed
			expectedReturn: []EPCType{},
		},
		{
			name:           "Success then Fail - Single EPC",
			device:         device1,
			initialFailed:  map[string][]EPCType{},
			successProps:   Properties{},
			failedEPCsIn:   []EPCType{epcB0},
			expectedFailed: map[string][]EPCType{device1Key: {epcB0}},
			expectedReturn: []EPCType{epcB0},
		},
		{
			name:           "Mixed Success/Fail",
			device:         device1,
			initialFailed:  map[string][]EPCType{device1Key: {epc80}},        // epc80 failed previously
			successProps:   Properties{{EPC: epc80, EDT: []byte{0x30}}},      // epc80 succeeds now
			failedEPCsIn:   []EPCType{epcB0, epcB3},                          // epcB0, epcB3 fail now
			expectedFailed: map[string][]EPCType{device1Key: {epcB0, epcB3}}, // epc80 removed, epcB0, epcB3 added
			expectedReturn: []EPCType{epcB0, epcB3},
		},
		{
			name:          "Multiple Devices - Initial Fail Device 2",
			device:        device2,
			initialFailed: map[string][]EPCType{device1Key: {epc80}}, // Device 1 has a failed EPC
			successProps:  Properties{},
			failedEPCsIn:  []EPCType{epcC0},
			expectedFailed: map[string][]EPCType{
				device1Key: {epc80},
				device2Key: {epcC0}, // Device 2 failure added
			},
			expectedReturn: []EPCType{epcC0},
		},
		{
			name:   "Multiple Devices - Consecutive Fail Device 1, Initial Fail Device 2",
			device: device1,
			initialFailed: map[string][]EPCType{
				device1Key: {epc80},
				device2Key: {epcC0},
			},
			successProps: Properties{},
			failedEPCsIn: []EPCType{epc80, epcB0}, // epc80 fails again, epcB0 fails first time
			expectedFailed: map[string][]EPCType{
				device1Key: {epc80, epcB0}, // epcB0 added to device 1
				device2Key: {epcC0},
			},
			expectedReturn: []EPCType{epcB0}, // Only the newly failed epcB0 should be returned
		},
		{
			name:   "Multiple Devices - Fail then Success Device 1",
			device: device1,
			initialFailed: map[string][]EPCType{
				device1Key: {epc80, epcB0},
				device2Key: {epcC0},
			},
			successProps: Properties{{EPC: epc80, EDT: []byte{0x30}}}, // epc80 succeeds
			failedEPCsIn: []EPCType{},
			expectedFailed: map[string][]EPCType{
				device1Key: {epcB0}, // epc80 removed from device 1
				device2Key: {epcC0},
			},
			expectedReturn: []EPCType{},
		},
		{
			name:           "Fail then Success - Multiple EPCs, some remain failed",
			device:         device1,
			initialFailed:  map[string][]EPCType{device1Key: {epc80, epcB0, epcB3}}, // 80, B0, B3 failed previously
			successProps:   Properties{{EPC: epc80, EDT: []byte{0x30}}},             // epc80 succeeds now
			failedEPCsIn:   []EPCType{},                                             // No new failures
			expectedFailed: map[string][]EPCType{device1Key: {epcB0, epcB3}},        // epc80 removed, B0, B3 remain
			expectedReturn: []EPCType{},
		},
		{
			name:           "Fail then Success - All previously failed EPCs succeed",
			device:         device1,
			initialFailed:  map[string][]EPCType{device1Key: {epc80, epcB0}},                             // 80, B0 failed previously
			successProps:   Properties{{EPC: epc80, EDT: []byte{0x30}}, {EPC: epcB0, EDT: []byte{0x19}}}, // Both succeed now
			failedEPCsIn:   []EPCType{},
			expectedFailed: map[string][]EPCType{}, // Device key should be removed entirely
			expectedReturn: []EPCType{},
		},
		{
			name:           "No initial failures, some fail now",
			device:         device1,
			initialFailed:  map[string][]EPCType{},
			successProps:   Properties{{EPC: epc80, EDT: []byte{0x30}}}, // epc80 succeeds
			failedEPCsIn:   []EPCType{epcB0, epcB3},                     // epcB0, epcB3 fail
			expectedFailed: map[string][]EPCType{device1Key: {epcB0, epcB3}},
			expectedReturn: []EPCType{epcB0, epcB3},
		},
		{
			name:           "No changes - no success, no new failures, some already failed",
			device:         device1,
			initialFailed:  map[string][]EPCType{device1Key: {epc80}},
			successProps:   Properties{},
			failedEPCsIn:   []EPCType{},
			expectedFailed: map[string][]EPCType{device1Key: {epc80}},
			expectedReturn: []EPCType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new session for each test case to ensure isolation
			s := createTestSession()
			// Set the initial state for failedEPCs
			s.failedEPCs = make(map[string][]EPCType)
			for k, v := range tt.initialFailed {
				// Create a copy of the slice to avoid modifying the original test data
				epcsCopy := make([]EPCType, len(v))
				copy(epcsCopy, v)
				s.failedEPCs[k] = epcsCopy
			}

			// Call the function under test
			// Make a copy of failedEPCsIn as it might be modified by the function
			failedEPCsInCopy := make([]EPCType, len(tt.failedEPCsIn))
			copy(failedEPCsInCopy, tt.failedEPCsIn)
			actualReturn := s.updateFailedEPCs(tt.device, tt.successProps, failedEPCsInCopy)

			// Sort slices before comparison for consistent results
			sortEPCs := func(epcs []EPCType) {
				slices.SortFunc(epcs, func(a, b EPCType) int { return int(a) - int(b) })
			}
			sortEPCs(actualReturn)
			for _, epcs := range s.failedEPCs {
				sortEPCs(epcs)
			}
			expectedFailedSorted := make(map[string][]EPCType)
			for k, v := range tt.expectedFailed {
				sortedV := make([]EPCType, len(v))
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
