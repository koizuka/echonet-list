package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"echonet-list/protocol"
)

// TestHandleSetPropertiesFromClient covers four scenarios:
// 1. EDT only
// 2. String only
// 3. Both EDT and String consistent
// 4. Both EDT and String conflicting (error)
func TestHandleSetPropertiesFromClient(t *testing.T) {
	tests := []struct {
		name      string
		payload   protocol.SetPropertiesPayload
		wantError bool
		errorCode protocol.ErrorCode
	}{
		{
			name: "EDT only",
			payload: protocol.SetPropertiesPayload{
				Target: "192.168.1.10 0130:1",
				Properties: protocol.PropertyMap{
					"80": {EDT: base64.StdEncoding.EncodeToString([]byte{0x30})},
				},
			},
			wantError: false,
		},
		{
			name: "String only",
			payload: protocol.SetPropertiesPayload{
				Target: "192.168.1.10 0130:1",
				Properties: protocol.PropertyMap{
					"B0": {String: "auto"},
				},
			},
			wantError: false,
		},
		{
			name: "EDT and String consistent",
			payload: protocol.SetPropertiesPayload{
				Target: "192.168.1.10 0130:1",
				Properties: protocol.PropertyMap{
					"80": {
						EDT:    base64.StdEncoding.EncodeToString([]byte{0x30}),
						String: "on",
					},
				},
			},
			wantError: false,
		},
		{
			name: "EDT and String conflicting",
			payload: protocol.SetPropertiesPayload{
				Target: "192.168.1.10 0130:1",
				Properties: protocol.PropertyMap{
					"80": {
						EDT:    base64.StdEncoding.EncodeToString([]byte{0x30}),
						String: "off",
					},
				},
			},
			wantError: true,
			errorCode: protocol.ErrorCodeInvalidParameters,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := newMockWebSocketTransport()
			mockClient := &mockECHONETListClient{}

			ws := &WebSocketServer{
				ctx:           context.Background(),
				transport:     mockTransport,
				echonetClient: mockClient,
				handler:       nil,
			}

			data, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			msg := &protocol.Message{
				Type:      protocol.MessageTypeSetProperties,
				Payload:   data,
				RequestID: "req-id",
			}

			if err := ws.handleSetPropertiesFromClient("conn", msg); err != nil {
				t.Fatalf("handleSetPropertiesFromClient error: %v", err)
			}

			raw, ok := mockTransport.sentMessages["conn"]
			if !ok {
				t.Fatalf("no message sent")
			}
			var resp protocol.Message
			if err := json.Unmarshal(raw, &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if resp.Type != protocol.MessageTypeCommandResult {
				t.Errorf("resp.Type = %v, want %v", resp.Type, protocol.MessageTypeCommandResult)
			}
			var cr protocol.CommandResultPayload
			if err := json.Unmarshal(resp.Payload, &cr); err != nil {
				t.Fatalf("unmarshal CommandResultPayload: %v", err)
			}

			if tt.wantError {
				if cr.Success {
					t.Errorf("expected error but got success")
				}
				if cr.Error == nil {
					t.Errorf("expected error payload but none")
				} else if cr.Error.Code != tt.errorCode {
					t.Errorf("error code = %v, want %v", cr.Error.Code, tt.errorCode)
				}
			} else {
				if !cr.Success {
					t.Errorf("expected success but got error: %+v", cr.Error)
				}
			}
		})
	}
}
