package echonet_lite

import (
	"bytes"
	"testing"
)

func TestPropertyTable_FindAlias(t *testing.T) {
	// Test data setup
	testTable := PropertyTable{
		EPCDesc: map[EPCType]PropertyDesc{
			0x80: { // Operation status
				Name: "OperationStatus",
				Aliases: map[string][]byte{
					"on":  {0x30},
					"off": {0x31},
				},
			},
			0xB0: { // Illuminance level
				Name: "IlluminanceLevel",
				Aliases: map[string][]byte{
					"dark":  {0x41},
					"light": {0x42},
				},
			},
		},
	}

	tests := []struct {
		name      string
		alias     string
		wantProp  Property
		wantFound bool
	}{
		{
			name:      "Find existing alias 'on'",
			alias:     "on",
			wantProp:  Property{EPC: 0x80, EDT: []byte{0x30}},
			wantFound: true,
		},
		{
			name:      "Find existing alias 'off'",
			alias:     "off",
			wantProp:  Property{EPC: 0x80, EDT: []byte{0x31}},
			wantFound: true,
		},
		{
			name:      "Find existing alias 'dark'",
			alias:     "dark",
			wantProp:  Property{EPC: 0xB0, EDT: []byte{0x41}},
			wantFound: true,
		},
		{
			name:      "Find non-existing alias 'unknown'",
			alias:     "unknown",
			wantProp:  Property{}, // Expect zero value
			wantFound: false,
		},
		{
			name:      "Find empty alias",
			alias:     "",
			wantProp:  Property{}, // Expect zero value
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProp, gotFound := testTable.FindAlias(tt.alias)

			if gotFound != tt.wantFound {
				t.Errorf("FindAlias() found = %v, want %v", gotFound, tt.wantFound)
			}

			// Only compare Property if found is expected to be true
			if tt.wantFound {
				if gotProp.EPC != tt.wantProp.EPC {
					t.Errorf("FindAlias() gotProp.EPC = %X, want %X", gotProp.EPC, tt.wantProp.EPC)
				}
				if !bytes.Equal(gotProp.EDT, tt.wantProp.EDT) {
					t.Errorf("FindAlias() gotProp.EDT = %X, want %X", gotProp.EDT, tt.wantProp.EDT)
				}
			} else {
				// If not found, ensure the returned property is the zero value
				if gotProp.EPC != 0 || gotProp.EDT != nil {
					t.Errorf("FindAlias() expected zero Property when not found, but got EPC=%X, EDT=%X", gotProp.EPC, gotProp.EDT)
				}
			}
		})
	}
}
