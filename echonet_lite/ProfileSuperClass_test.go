package echonet_lite

import (
	"reflect"
	"testing"
)

func TestOperationStatus_Encode(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name        string
		status      OperationStatus
		expectedEDT byte
	}{
		{
			name:        "OperationStatus(true)",
			status:      OperationStatus(true),
			expectedEDT: 0x30,
		},
		{
			name:        "OperationStatus(false)",
			status:      OperationStatus(false),
			expectedEDT: 0x31,
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the Property object
			prop := tc.status.Property()

			// Check that the Property has the expected EPC and EDT
			if prop.EPC != EPCOperationStatus {
				t.Errorf("Expected EPC to be 0x%X, got 0x%X", EPCOperationStatus, prop.EPC)
			}

			expectedEDT := []byte{tc.expectedEDT}
			if !reflect.DeepEqual(prop.EDT, expectedEDT) {
				t.Errorf("Expected EDT to be %v, got %v", expectedEDT, prop.EDT)
			}

			// Test the Encode method
			encoded := prop.Encode()

			// Print the encoded data for debugging
			t.Logf("Encoded data: %v (length: %d)", encoded, len(encoded))

			// Check that the encoded data has the correct format
			// The first byte should be the EPC (0x80)
			if encoded[0] != byte(EPCOperationStatus) {
				t.Errorf("Expected encoded[0] to be 0x%X, got 0x%X", EPCOperationStatus, encoded[0])
			}

			// The second byte should be the PDC (length of EDT, which is 1)
			if encoded[1] != 0x01 {
				t.Errorf("Expected encoded[1] to be 0x01, got 0x%X", encoded[1])
			}

			// Check if the encoded data has the correct length and EDT value
			expectedLength := 3 // EPC (1 byte) + PDC (1 byte) + EDT (1 byte)
			if len(encoded) != expectedLength {
				t.Errorf("Expected encoded length to be %d, got %d", expectedLength, len(encoded))
			} else if encoded[2] != tc.expectedEDT {
				t.Errorf("Expected encoded[2] to be 0x%X, got 0x%X", tc.expectedEDT, encoded[2])
			}
		})
	}
}
