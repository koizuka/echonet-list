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
func createTestSetup(t *testing.T, ctx context.Context) (*WebSocketServer, *MockECHONETClientWithUpdateTracking, time.Duration) {
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

	ws := &WebSocketServer{
		ctx:           ctx,
		echonetClient: mockClient,
		handler:       handlerInstance,
	}

	return ws, mockClient, desc.UpdateDelay
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
	ws, mockClient, updateDelay := createTestSetup(t, context.Background())
	msg := createOperationModeMessage()

	// Execute and verify response
	executeAndVerifyResponse(t, ws, msg)

	// Wait for the trigger update to be executed (minimized buffer time)
	bufferTime := 100 * time.Millisecond
	time.Sleep(updateDelay + bufferTime)

	// Verify that UpdateProperties was called
	verifyUpdatePropertiesCalls(t, mockClient, 1)
	verifyFilterCriteria(t, mockClient)
}

func TestSetPropertiesWithoutTriggerUpdate(t *testing.T) {
	t.Parallel()
	ws, mockClient, _ := createTestSetup(t, context.Background())
	msg := createTemperatureMessage()

	// Execute and verify response
	executeAndVerifyResponse(t, ws, msg)

	// Wait a bit to ensure no trigger update is executed
	time.Sleep(100 * time.Millisecond)

	// Verify that UpdateProperties was NOT called
	verifyUpdatePropertiesCalls(t, mockClient, 0)
}

func TestSetPropertiesWithTriggerUpdateTiming(t *testing.T) {
	t.Parallel()
	ws, mockClient, updateDelay := createTestSetup(t, context.Background())
	msg := createOperationModeMessage()

	// Record start time
	startTime := time.Now()

	// Execute and verify response
	executeAndVerifyResponse(t, ws, msg)

	// Check that UpdateProperties is NOT called immediately
	verifyUpdatePropertiesCalls(t, mockClient, 0)

	// Wait for half the configured delay
	halfDelay := updateDelay / 2
	time.Sleep(halfDelay)

	// UpdateProperties should still not be called
	verifyUpdatePropertiesCalls(t, mockClient, 0)

	// Wait for remaining time plus buffer
	bufferTime := 100 * time.Millisecond
	remainingTime := updateDelay - halfDelay + bufferTime
	time.Sleep(remainingTime)

	// Now UpdateProperties should have been called
	verifyUpdatePropertiesCalls(t, mockClient, 1)

	// Verify timing: should be approximately the configured delay (with some tolerance)
	elapsedTime := time.Since(startTime)
	tolerance := 200 * time.Millisecond

	if elapsedTime < updateDelay-tolerance || elapsedTime > updateDelay+tolerance*5 {
		t.Errorf("UpdateProperties timing out of expected range. Expected: ~%v, Actual: %v", updateDelay, elapsedTime)
	}
}

func TestSetPropertiesWithTriggerUpdateCancelled(t *testing.T) {
	t.Parallel()
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ws, mockClient, _ := createTestSetup(t, ctx)
	msg := createOperationModeMessage()

	// Execute and verify response
	executeAndVerifyResponse(t, ws, msg)

	// Cancel the context immediately after the request to simulate shutdown
	cancel()

	// Wait enough time to ensure the goroutine would have been triggered
	// but was cancelled instead
	time.Sleep(100 * time.Millisecond)

	// Verify that UpdateProperties was NOT called due to cancellation
	verifyUpdatePropertiesCalls(t, mockClient, 0)
}
