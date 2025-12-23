package server

import (
	"context"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockLocationTransport はテスト用のモックトランスポート
type mockLocationTransport struct {
	mock.Mock
	broadcastMessages [][]byte
}

func (m *mockLocationTransport) Start(options StartOptions) error { return nil }
func (m *mockLocationTransport) Stop() error                      { return nil }
func (m *mockLocationTransport) SetMessageHandler(handler func(connID string, message []byte) error) {
}
func (m *mockLocationTransport) SetConnectHandler(handler func(connID string) error) {}
func (m *mockLocationTransport) SetDisconnectHandler(handler func(connID string))    {}
func (m *mockLocationTransport) SendMessage(connID string, message []byte) error     { return nil }
func (m *mockLocationTransport) BroadcastMessage(message []byte) error {
	args := m.Called(message)
	m.broadcastMessages = append(m.broadcastMessages, message)
	return args.Error(0)
}

func TestHandleManageLocationAliasFromClient_Add(t *testing.T) {
	ctx := context.Background()

	// Create handler with test mode
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true,
	})
	require.NoError(t, err)

	mockTransport := new(mockLocationTransport)
	mockTransport.On("BroadcastMessage", mock.Anything).Return(nil)

	ws := &WebSocketServer{
		ctx:       ctx,
		handler:   handlerInstance,
		transport: mockTransport,
	}

	// Create a manage_location_alias message
	msg := &protocol.Message{
		Type: protocol.MessageTypeManageLocationAlias,
		Payload: json.RawMessage(`{
			"action": "add",
			"alias": "#2F寝室",
			"value": "room2"
		}`),
		RequestID: "test-req-1",
	}

	// Execute handler
	result := ws.handleManageLocationAliasFromClient(msg)

	// Verify result
	assert.True(t, result.Success)
	assert.Nil(t, result.Error)

	// Verify the alias was added
	aliases, _ := handlerInstance.GetLocationSettings()
	assert.Equal(t, "room2", aliases["#2F寝室"])
}

func TestHandleManageLocationAliasFromClient_Delete(t *testing.T) {
	ctx := context.Background()

	// Create handler with test mode
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true,
	})
	require.NoError(t, err)

	// Add an alias first
	err = handlerInstance.LocationAliasAdd("#リビング", "living")
	require.NoError(t, err)

	mockTransport := new(mockLocationTransport)
	mockTransport.On("BroadcastMessage", mock.Anything).Return(nil)

	ws := &WebSocketServer{
		ctx:       ctx,
		handler:   handlerInstance,
		transport: mockTransport,
	}

	// Create a manage_location_alias message to delete
	msg := &protocol.Message{
		Type: protocol.MessageTypeManageLocationAlias,
		Payload: json.RawMessage(`{
			"action": "delete",
			"alias": "#リビング"
		}`),
		RequestID: "test-req-2",
	}

	// Execute handler
	result := ws.handleManageLocationAliasFromClient(msg)

	// Verify result
	assert.True(t, result.Success)

	// Verify the alias was deleted
	aliases, _ := handlerInstance.GetLocationSettings()
	_, exists := aliases["#リビング"]
	assert.False(t, exists)
}

func TestHandleManageLocationAliasFromClient_InvalidAlias(t *testing.T) {
	ctx := context.Background()

	// Create handler with test mode
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true,
	})
	require.NoError(t, err)

	mockTransport := new(mockLocationTransport)
	// No broadcast expected for error case

	ws := &WebSocketServer{
		ctx:       ctx,
		handler:   handlerInstance,
		transport: mockTransport,
	}

	// Create a manage_location_alias message with invalid alias (no # prefix)
	msg := &protocol.Message{
		Type: protocol.MessageTypeManageLocationAlias,
		Payload: json.RawMessage(`{
			"action": "add",
			"alias": "invalid_alias",
			"value": "room2"
		}`),
		RequestID: "test-req-3",
	}

	// Execute handler
	result := ws.handleManageLocationAliasFromClient(msg)

	// Verify result - should fail
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}

func TestHandleSetLocationOrderFromClient(t *testing.T) {
	ctx := context.Background()

	// Create handler with test mode
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true,
	})
	require.NoError(t, err)

	mockTransport := new(mockLocationTransport)
	mockTransport.On("BroadcastMessage", mock.Anything).Return(nil)

	ws := &WebSocketServer{
		ctx:       ctx,
		handler:   handlerInstance,
		transport: mockTransport,
	}

	// Create a set_location_order message
	msg := &protocol.Message{
		Type: protocol.MessageTypeSetLocationOrder,
		Payload: json.RawMessage(`{
			"order": ["living", "room2", "kitchen"]
		}`),
		RequestID: "test-req-4",
	}

	// Execute handler
	result := ws.handleSetLocationOrderFromClient(msg)

	// Verify result
	assert.True(t, result.Success)

	// Verify the order was set
	_, order := handlerInstance.GetLocationSettings()
	assert.Equal(t, []string{"living", "room2", "kitchen"}, order)
}

func TestHandleSetLocationOrderFromClient_Reset(t *testing.T) {
	ctx := context.Background()

	// Create handler with test mode
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true,
	})
	require.NoError(t, err)

	// Set an initial order
	err = handlerInstance.SetLocationOrder([]string{"a", "b", "c"})
	require.NoError(t, err)

	mockTransport := new(mockLocationTransport)
	mockTransport.On("BroadcastMessage", mock.Anything).Return(nil)

	ws := &WebSocketServer{
		ctx:       ctx,
		handler:   handlerInstance,
		transport: mockTransport,
	}

	// Create a set_location_order message with empty order to reset
	msg := &protocol.Message{
		Type: protocol.MessageTypeSetLocationOrder,
		Payload: json.RawMessage(`{
			"order": []
		}`),
		RequestID: "test-req-5",
	}

	// Execute handler
	result := ws.handleSetLocationOrderFromClient(msg)

	// Verify result
	assert.True(t, result.Success)

	// Verify the order was reset
	_, order := handlerInstance.GetLocationSettings()
	assert.Empty(t, order)
}

func TestHandleGetLocationSettingsFromClient(t *testing.T) {
	ctx := context.Background()

	// Create handler with test mode
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true,
	})
	require.NoError(t, err)

	// Add some location settings
	err = handlerInstance.LocationAliasAdd("#テスト", "test")
	require.NoError(t, err)
	err = handlerInstance.SetLocationOrder([]string{"test", "other"})
	require.NoError(t, err)

	mockTransport := new(mockLocationTransport)
	// No broadcast expected for get operation

	ws := &WebSocketServer{
		ctx:       ctx,
		handler:   handlerInstance,
		transport: mockTransport,
	}

	// Create a get_location_settings message
	msg := &protocol.Message{
		Type:      protocol.MessageTypeGetLocationSettings,
		Payload:   json.RawMessage(`{}`),
		RequestID: "test-req-6",
	}

	// Execute handler
	result := ws.handleGetLocationSettingsFromClient(msg)

	// Verify result
	assert.True(t, result.Success)

	// Verify the data contains location settings
	require.NotNil(t, result.Data)

	var settingsData protocol.LocationSettingsData
	err = json.Unmarshal(result.Data, &settingsData)
	require.NoError(t, err)

	assert.Equal(t, "test", settingsData.Aliases["#テスト"])
	assert.Equal(t, []string{"test", "other"}, settingsData.Order)
}

func TestHandleManageLocationAliasFromClient_Update(t *testing.T) {
	ctx := context.Background()

	// Create handler with test mode
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true,
	})
	require.NoError(t, err)

	// Add an alias first
	err = handlerInstance.LocationAliasAdd("#テスト", "old_value")
	require.NoError(t, err)

	mockTransport := new(mockLocationTransport)
	mockTransport.On("BroadcastMessage", mock.Anything).Return(nil)

	ws := &WebSocketServer{
		ctx:       ctx,
		handler:   handlerInstance,
		transport: mockTransport,
	}

	// Create a manage_location_alias message to update
	msg := &protocol.Message{
		Type: protocol.MessageTypeManageLocationAlias,
		Payload: json.RawMessage(`{
			"action": "update",
			"alias": "#テスト",
			"value": "new_value"
		}`),
		RequestID: "test-req-7",
	}

	// Execute handler
	result := ws.handleManageLocationAliasFromClient(msg)

	// Verify result
	assert.True(t, result.Success)

	// Verify the alias was updated
	aliases, _ := handlerInstance.GetLocationSettings()
	assert.Equal(t, "new_value", aliases["#テスト"])
}

func TestHandleManageLocationAliasFromClient_EmptyAlias(t *testing.T) {
	ctx := context.Background()

	// Create handler with test mode
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true,
	})
	require.NoError(t, err)

	mockTransport := new(mockLocationTransport)
	// No broadcast expected for error case

	ws := &WebSocketServer{
		ctx:       ctx,
		handler:   handlerInstance,
		transport: mockTransport,
	}

	// Create a manage_location_alias message with empty alias
	msg := &protocol.Message{
		Type: protocol.MessageTypeManageLocationAlias,
		Payload: json.RawMessage(`{
			"action": "add",
			"alias": "",
			"value": "room2"
		}`),
		RequestID: "test-req-8",
	}

	// Execute handler
	result := ws.handleManageLocationAliasFromClient(msg)

	// Verify result - should fail
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}

func TestHandleManageLocationAliasFromClient_DeleteNonExistent(t *testing.T) {
	ctx := context.Background()

	// Create handler with test mode
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{
		TestMode: true,
	})
	require.NoError(t, err)

	mockTransport := new(mockLocationTransport)
	// No broadcast expected for error case

	ws := &WebSocketServer{
		ctx:       ctx,
		handler:   handlerInstance,
		transport: mockTransport,
	}

	// Create a manage_location_alias message to delete a non-existent alias
	msg := &protocol.Message{
		Type: protocol.MessageTypeManageLocationAlias,
		Payload: json.RawMessage(`{
			"action": "delete",
			"alias": "#存在しないエイリアス"
		}`),
		RequestID: "test-req-9",
	}

	// Execute handler
	result := ws.handleManageLocationAliasFromClient(msg)

	// Verify result - should fail (alias not found)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
}
