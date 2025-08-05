package server

import (
	"context"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// MockECHONETClientWithUpdateTracking extends mockECHONETListClient to track UpdateProperties calls
type MockECHONETClientWithUpdateTracking struct {
	*mockECHONETListClient
	mu                    sync.Mutex
	UpdatePropertiesCalls []handler.FilterCriteria
}

func (m *MockECHONETClientWithUpdateTracking) UpdateProperties(criteria handler.FilterCriteria, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UpdatePropertiesCalls = append(m.UpdatePropertiesCalls, criteria)
	return nil
}

func (m *MockECHONETClientWithUpdateTracking) SetProperties(device echonet_lite.IPAndEOJ, props echonet_lite.Properties) (handler.DeviceAndProperties, error) {
	return handler.DeviceAndProperties{Device: device, Properties: props}, nil
}

// createTestSetup creates a reusable test setup for trigger update tests
func createTestSetup(t *testing.T, ctx context.Context) (*WebSocketServer, *MockECHONETClientWithUpdateTracking, *MockTimeProvider, time.Duration) {
	// Get the UpdateDelay from PropertyTable
	desc, ok := echonet_lite.GetPropertyDesc(echonet_lite.HomeAirConditioner_ClassCode, echonet_lite.EPC_HAC_OperationModeSetting)
	if !ok {
		t.Fatalf("Failed to get property description for operation mode setting")
	}
	if !desc.TriggerUpdate {
		t.Fatalf("Expected TriggerUpdate to be true for operation mode setting")
	}

	// Create mock client
	mockClient := &MockECHONETClientWithUpdateTracking{
		mockECHONETListClient: &mockECHONETListClient{},
		UpdatePropertiesCalls: []handler.FilterCriteria{},
	}

	// Create handler and WebSocket server
	handlerInstance, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Create mock time provider
	mockTimeProvider := NewMockTimeProvider()

	ws := &WebSocketServer{
		ctx:           ctx,
		echonetClient: mockClient,
		handler:       handlerInstance,
		timeProvider:  mockTimeProvider,
	}

	return ws, mockClient, mockTimeProvider, desc.UpdateDelay
}

// createOperationModeMessage creates a standard operation mode change message
func createOperationModeMessage() *protocol.Message {
	return &protocol.Message{
		Type: "set_properties",
		Payload: json.RawMessage(`{
			"target": "192.168.1.100 0130:1",
			"properties": {
				"B0": {"string": "heating"}
			}
		}`),
	}
}

// createTemperatureMessage creates a temperature change message (no trigger)
func createTemperatureMessage() *protocol.Message {
	return &protocol.Message{
		Type: "set_properties",
		Payload: json.RawMessage(`{
			"target": "192.168.1.100 0130:1",
			"properties": {
				"B3": {"number": 25}
			}
		}`),
	}
}

// executeAndVerifyResponse executes a message and verifies the response
func executeAndVerifyResponse(t *testing.T, ws *WebSocketServer, msg *protocol.Message) {
	response := ws.handleSetPropertiesFromClient(msg)
	if !response.Success {
		t.Errorf("Expected success, got error: %s", response.Error.Message)
	}
}

// verifyUpdatePropertiesCalls verifies the number of UpdateProperties calls
func verifyUpdatePropertiesCalls(t *testing.T, mockClient *MockECHONETClientWithUpdateTracking, expectedCount int) {
	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()
	if len(mockClient.UpdatePropertiesCalls) != expectedCount {
		t.Errorf("Expected %d UpdateProperties calls, got %d", expectedCount, len(mockClient.UpdatePropertiesCalls))
	}
}

// verifyFilterCriteria verifies the filter criteria for UpdateProperties calls
func verifyFilterCriteria(t *testing.T, mockClient *MockECHONETClientWithUpdateTracking) {
	mockClient.mu.Lock()
	defer mockClient.mu.Unlock()
	if len(mockClient.UpdatePropertiesCalls) == 0 {
		return
	}

	criteria := mockClient.UpdatePropertiesCalls[0]
	if criteria.Device.IP.String() != "192.168.1.100" {
		t.Errorf("Expected IP address 192.168.1.100, got %s", criteria.Device.IP.String())
	}
	if criteria.Device.ClassCode != nil && *criteria.Device.ClassCode != 0x0130 {
		t.Errorf("Expected class code 0x0130, got %04X", *criteria.Device.ClassCode)
	}
	if criteria.Device.InstanceCode != nil && *criteria.Device.InstanceCode != 0x01 {
		t.Errorf("Expected instance code 0x01, got %02X", *criteria.Device.InstanceCode)
	}
}

func TestSetPropertiesWithTriggerUpdate(t *testing.T) {
	t.Parallel()
	ws, mockClient, mockTime, updateDelay := createTestSetup(t, context.Background())
	msg := createOperationModeMessage()

	// Execute and verify response
	executeAndVerifyResponse(t, ws, msg)

	// Wait for the goroutine to set up the timer
	time.Sleep(10 * time.Millisecond)

	// Advance mock time to trigger the update
	mockTime.Advance(updateDelay)

	// Wait for the goroutine to process the timer event
	time.Sleep(50 * time.Millisecond)

	// Verify that UpdateProperties was called
	verifyUpdatePropertiesCalls(t, mockClient, 1)
	verifyFilterCriteria(t, mockClient)
}

func TestSetPropertiesWithoutTriggerUpdate(t *testing.T) {
	t.Parallel()
	ws, mockClient, mockTime, _ := createTestSetup(t, context.Background())
	msg := createTemperatureMessage()

	// Execute and verify response
	executeAndVerifyResponse(t, ws, msg)

	// Wait for the goroutine to set up the timer (if any)
	time.Sleep(10 * time.Millisecond)

	// Advance time more than enough to trigger an update if there was one
	mockTime.Advance(10 * time.Second)

	// Wait for the goroutine to process (if any timer was set)
	time.Sleep(50 * time.Millisecond)

	// Verify that UpdateProperties was NOT called
	verifyUpdatePropertiesCalls(t, mockClient, 0)
}

func TestSetPropertiesWithTriggerUpdateTiming(t *testing.T) {
	t.Parallel()
	ws, mockClient, mockTime, updateDelay := createTestSetup(t, context.Background())
	msg := createOperationModeMessage()

	// Execute and verify response
	executeAndVerifyResponse(t, ws, msg)

	// Wait for the goroutine to set up the timer
	time.Sleep(10 * time.Millisecond)

	// Check that UpdateProperties is NOT called immediately
	verifyUpdatePropertiesCalls(t, mockClient, 0)

	// Advance time by half the configured delay
	halfDelay := updateDelay / 2
	mockTime.Advance(halfDelay)

	// Wait a bit but no timer should fire yet
	time.Sleep(50 * time.Millisecond)

	// UpdateProperties should still not be called
	verifyUpdatePropertiesCalls(t, mockClient, 0)

	// Advance time to complete the full delay
	mockTime.Advance(halfDelay)

	// Wait for the goroutine to process the timer event
	time.Sleep(50 * time.Millisecond)

	// Now UpdateProperties should have been called
	verifyUpdatePropertiesCalls(t, mockClient, 1)
}

func TestSetPropertiesWithTriggerUpdateCancelled(t *testing.T) {
	t.Parallel()
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ws, mockClient, mockTime, updateDelay := createTestSetup(t, ctx)
	msg := createOperationModeMessage()

	// Execute and verify response
	executeAndVerifyResponse(t, ws, msg)

	// Wait for the goroutine to set up the timer
	time.Sleep(10 * time.Millisecond)

	// Cancel the context immediately after the request to simulate shutdown
	cancel()

	// Advance time past the delay
	mockTime.Advance(updateDelay + time.Second)

	// Wait to ensure no processing occurs
	time.Sleep(50 * time.Millisecond)

	// Verify that UpdateProperties was NOT called due to cancellation
	verifyUpdatePropertiesCalls(t, mockClient, 0)
}
