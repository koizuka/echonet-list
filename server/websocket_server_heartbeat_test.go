package server

import (
	"context"
	"echonet-list/protocol"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockHeartbeatTransport captures broadcast messages for heartbeat tests.
type mockHeartbeatTransport struct {
	mock.Mock
	broadcastMessages [][]byte
}

func (m *mockHeartbeatTransport) Start(options StartOptions) error { return nil }
func (m *mockHeartbeatTransport) Stop() error                      { return nil }
func (m *mockHeartbeatTransport) SetMessageHandler(handler func(connID string, message []byte) error) {
}
func (m *mockHeartbeatTransport) SetConnectHandler(handler func(connID string) error) {}
func (m *mockHeartbeatTransport) SetDisconnectHandler(handler func(connID string))    {}
func (m *mockHeartbeatTransport) SendMessage(connID string, message []byte) error     { return nil }
func (m *mockHeartbeatTransport) BroadcastMessage(message []byte) error {
	args := m.Called(message)
	m.broadcastMessages = append(m.broadcastMessages, message)
	return args.Error(0)
}

func newHeartbeatTestServer(t *testing.T) (*WebSocketServer, *mockHeartbeatTransport) {
	t.Helper()
	mockTransport := new(mockHeartbeatTransport)
	mockTransport.On("BroadcastMessage", mock.Anything).Return(nil)
	ws := &WebSocketServer{
		ctx:       context.Background(),
		transport: mockTransport,
	}
	return ws, mockTransport
}

func TestSendHeartbeat_NoClients_SkipsBroadcast(t *testing.T) {
	ws, mockTransport := newHeartbeatTestServer(t)
	// activeClients defaults to 0

	sent := ws.sendHeartbeat()

	assert.False(t, sent, "should not send a heartbeat when there are no clients")
	assert.Empty(t, mockTransport.broadcastMessages)
}

func TestSendHeartbeat_WithClients_BroadcastsServerHeartbeat(t *testing.T) {
	ws, mockTransport := newHeartbeatTestServer(t)
	ws.activeClients.Store(1)

	sent := ws.sendHeartbeat()

	assert.True(t, sent)
	require.Len(t, mockTransport.broadcastMessages, 1)

	var msg protocol.Message
	require.NoError(t, json.Unmarshal(mockTransport.broadcastMessages[0], &msg))
	assert.Equal(t, protocol.MessageTypeServerHeartbeat, msg.Type)

	var payload protocol.ServerHeartbeatPayload
	require.NoError(t, json.Unmarshal(msg.Payload, &payload))
	assert.NotEmpty(t, payload.Time, "heartbeat payload should carry a timestamp")
}
