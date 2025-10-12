package client

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"echonet-list/echonet_lite"
	"echonet-list/protocol"
)

type mockHistoryTransport struct {
	t         *testing.T
	client    *WebSocketClient
	handler   func(*protocol.Message) (*protocol.Message, error)
	connected bool
}

func (m *mockHistoryTransport) Connect() error { m.connected = true; return nil }
func (m *mockHistoryTransport) Close() error   { m.connected = false; return nil }
func (m *mockHistoryTransport) ReadMessage() (int, []byte, error) {
	return 0, nil, nil
}
func (m *mockHistoryTransport) WriteMessage(messageType int, data []byte) error {
	msg, err := protocol.ParseMessage(data)
	if err != nil {
		return err
	}

	if m.handler == nil {
		m.t.Fatalf("mock handler is nil")
	}

	response, err := m.handler(msg)
	if err != nil {
		return err
	}

	m.client.responseChMutex.Lock()
	ch, ok := m.client.responseCh[msg.RequestID]
	m.client.responseChMutex.Unlock()
	if !ok {
		m.t.Fatalf("response channel for %s not found", msg.RequestID)
	}
	if response != nil {
		ch <- response
	}
	return nil
}
func (m *mockHistoryTransport) IsConnected() bool { return m.connected }

func TestGetDeviceHistorySuccess(t *testing.T) {
	ctx := context.Background()
	client := &WebSocketClient{
		ctx:        ctx,
		responseCh: make(map[string]chan *protocol.Message),
	}

	mock := &mockHistoryTransport{t: t, client: client}
	client.transport = mock

	device := IPAndEOJ{
		IP:  netParseIP(t, "192.168.1.10"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 0x01),
	}

	since := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	limit := 5
	settableOnly := false

	var capturedPayload protocol.GetDeviceHistoryPayload
	mock.handler = func(msg *protocol.Message) (*protocol.Message, error) {
		if msg.Type != protocol.MessageTypeGetDeviceHistory {
			t.Fatalf("unexpected message type: %s", msg.Type)
		}
		if err := protocol.ParsePayload(msg, &capturedPayload); err != nil {
			t.Fatalf("failed to parse payload: %v", err)
		}

		entry := protocol.HistoryEntry{
			Timestamp: since.Add(10 * time.Minute),
			EPC:       "80",
			Value:     protocol.PropertyData{String: "on"},
			Origin:    protocol.HistoryOriginSet,
			Settable:  true,
		}

		history := protocol.DeviceHistoryResponse{
			Entries: []protocol.HistoryEntry{entry},
		}
		historyData, _ := json.Marshal(history)
		resultPayload := protocol.CommandResultPayload{
			Success: true,
			Data:    historyData,
		}
		resultData, _ := json.Marshal(resultPayload)
		return &protocol.Message{
			Type:      protocol.MessageTypeCommandResult,
			RequestID: msg.RequestID,
			Payload:   resultData,
		}, nil
	}

	opts := DeviceHistoryOptions{
		Limit:        limit,
		Since:        &since,
		SettableOnly: &settableOnly,
	}

	entries, err := client.GetDeviceHistory(device, opts)
	if err != nil {
		t.Fatalf("GetDeviceHistory returned error: %v", err)
	}

	if capturedPayload.Target != device.Specifier() {
		t.Fatalf("expected target %s, got %s", device.Specifier(), capturedPayload.Target)
	}
	if capturedPayload.Limit == nil || *capturedPayload.Limit != limit {
		t.Fatalf("expected limit %d, got %v", limit, capturedPayload.Limit)
	}
	if capturedPayload.Since != since.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected since value: %s", capturedPayload.Since)
	}
	if capturedPayload.SettableOnly == nil || *capturedPayload.SettableOnly != settableOnly {
		t.Fatalf("expected settableOnly false, got %v", capturedPayload.SettableOnly)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].EPC != 0x80 {
		t.Fatalf("expected EPC 0x80, got 0x%02X", entries[0].EPC)
	}
	if entries[0].Value.String != "on" {
		t.Fatalf("unexpected value: %s", entries[0].Value.String)
	}
	if entries[0].Origin != protocol.HistoryOriginSet {
		t.Fatalf("unexpected origin: %s", entries[0].Origin)
	}
}

func TestGetDeviceHistoryError(t *testing.T) {
	ctx := context.Background()
	client := &WebSocketClient{
		ctx:        ctx,
		responseCh: make(map[string]chan *protocol.Message),
	}

	mock := &mockHistoryTransport{t: t, client: client}
	client.transport = mock

	device := IPAndEOJ{
		IP:  netParseIP(t, "192.168.1.10"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 0x01),
	}

	mock.handler = func(msg *protocol.Message) (*protocol.Message, error) {
		resultPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "invalid target",
			},
		}
		resultData, _ := json.Marshal(resultPayload)
		return &protocol.Message{
			Type:      protocol.MessageTypeCommandResult,
			RequestID: msg.RequestID,
			Payload:   resultData,
		}, nil
	}

	_, err := client.GetDeviceHistory(device, DeviceHistoryOptions{})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func netParseIP(t *testing.T, addr string) net.IP {
	t.Helper()
	ip := net.ParseIP(addr)
	if ip == nil {
		t.Fatalf("failed to parse IP %s", addr)
	}
	return ip
}
