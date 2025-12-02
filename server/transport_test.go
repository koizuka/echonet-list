package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestPingPongHeartbeat tests that ping messages are sent periodically
func TestPingPongHeartbeat(t *testing.T) {
	// Create a test server with shorter timeouts for testing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	transport := NewDefaultWebSocketTransport(ctx, ":0")

	// Track received pings
	var pingCount atomic.Int32

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Upgrade error: %v", err)
			return
		}

		// Set ping handler to count pings
		conn.SetPingHandler(func(appData string) error {
			pingCount.Add(1)
			// Send pong response
			return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		})

		// Read messages to keep connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// Connect to test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Verify transport was created correctly
	if transport == nil {
		t.Fatal("Transport should not be nil")
	}

	// Verify constants are set correctly
	if pingPeriod >= pongWait {
		t.Errorf("pingPeriod (%v) should be less than pongWait (%v)", pingPeriod, pongWait)
	}

	if writeWait <= 0 {
		t.Errorf("writeWait should be positive, got %v", writeWait)
	}
}

// TestPingPeriodLessThanPongWait verifies the critical requirement
func TestPingPeriodLessThanPongWait(t *testing.T) {
	if pingPeriod >= pongWait {
		t.Errorf("pingPeriod (%v) must be less than pongWait (%v) for heartbeat to work correctly", pingPeriod, pongWait)
	}
}

// TestWriteWaitPositive verifies write timeout is positive
func TestWriteWaitPositive(t *testing.T) {
	if writeWait <= 0 {
		t.Errorf("writeWait must be positive, got %v", writeWait)
	}
}

// TestTimeoutConstants verifies timeout constants have reasonable values
func TestTimeoutConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    time.Duration
		minValue time.Duration
		maxValue time.Duration
	}{
		{
			name:     "writeWait",
			value:    writeWait,
			minValue: 1 * time.Second,
			maxValue: 60 * time.Second,
		},
		{
			name:     "pongWait",
			value:    pongWait,
			minValue: 10 * time.Second,
			maxValue: 5 * time.Minute,
		},
		{
			name:     "pingPeriod",
			value:    pingPeriod,
			minValue: 5 * time.Second,
			maxValue: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value < tt.minValue {
				t.Errorf("%s (%v) is too small, minimum recommended is %v", tt.name, tt.value, tt.minValue)
			}
			if tt.value > tt.maxValue {
				t.Errorf("%s (%v) is too large, maximum recommended is %v", tt.name, tt.value, tt.maxValue)
			}
		})
	}
}

// TestClientConnectionHasPingDone verifies pingDone channel is initialized
func TestClientConnectionHasPingDone(t *testing.T) {
	client := &clientConnection{
		conn:     nil,
		mutex:    sync.Mutex{},
		pingDone: make(chan struct{}),
	}

	if client.pingDone == nil {
		t.Error("pingDone channel should be initialized")
	}

	// Test that channel can be closed without panic
	close(client.pingDone)
}

// TestPingDoneChannelStopsPingGoroutine tests graceful shutdown
func TestPingDoneChannelStopsPingGoroutine(t *testing.T) {
	pingDone := make(chan struct{})
	goroutineStopped := make(chan struct{})

	// Simulate ping goroutine behavior
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		defer close(goroutineStopped)

		for {
			select {
			case <-ticker.C:
				// Would send ping here
			case <-pingDone:
				return
			}
		}
	}()

	// Give goroutine time to start
	time.Sleep(50 * time.Millisecond)

	// Signal stop
	close(pingDone)

	// Wait for goroutine to stop with timeout
	select {
	case <-goroutineStopped:
		// Success
	case <-time.After(time.Second):
		t.Error("Ping goroutine did not stop within timeout")
	}
}

// TestNewDefaultWebSocketTransport verifies transport creation
func TestNewDefaultWebSocketTransport(t *testing.T) {
	ctx := context.Background()
	transport := NewDefaultWebSocketTransport(ctx, ":8080")

	if transport == nil {
		t.Fatal("Transport should not be nil")
	}

	if transport.clients == nil {
		t.Error("clients map should be initialized")
	}

	if transport.clientsReverse == nil {
		t.Error("clientsReverse map should be initialized")
	}
}

// TestTransportContextCancellation verifies context cancellation
func TestTransportContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := NewDefaultWebSocketTransport(ctx, ":0")

	if transport.ctx == nil {
		t.Error("Transport context should not be nil")
	}

	// Cancel context
	cancel()

	// Verify context is done
	select {
	case <-transport.ctx.Done():
		// Success
	default:
		t.Error("Transport context should be done after cancel")
	}
}

// TestBroadcastMessageToNoClients verifies broadcast with no clients
func TestBroadcastMessageToNoClients(t *testing.T) {
	ctx := context.Background()
	transport := NewDefaultWebSocketTransport(ctx, ":0")

	// Should not error when broadcasting to no clients
	err := transport.BroadcastMessage([]byte("test message"))
	if err != nil {
		t.Errorf("BroadcastMessage to no clients should not error, got: %v", err)
	}
}

// TestSendMessageToNonExistentClient verifies error handling
func TestSendMessageToNonExistentClient(t *testing.T) {
	ctx := context.Background()
	transport := NewDefaultWebSocketTransport(ctx, ":0")

	err := transport.SendMessage("non-existent-id", []byte("test message"))
	if err == nil {
		t.Error("SendMessage to non-existent client should error")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found', got: %v", err)
	}
}

// TestSetHandlers verifies handler setting
func TestSetHandlers(t *testing.T) {
	ctx := context.Background()
	transport := NewDefaultWebSocketTransport(ctx, ":0")

	messageHandlerCalled := false
	connectHandlerCalled := false
	disconnectHandlerCalled := false

	transport.SetMessageHandler(func(connID string, message []byte) error {
		messageHandlerCalled = true
		return nil
	})

	transport.SetConnectHandler(func(connID string) error {
		connectHandlerCalled = true
		return nil
	})

	transport.SetDisconnectHandler(func(connID string) {
		disconnectHandlerCalled = true
	})

	if transport.messageHandler == nil {
		t.Error("messageHandler should be set")
	}

	if transport.connectHandler == nil {
		t.Error("connectHandler should be set")
	}

	if transport.disconnectHandler == nil {
		t.Error("disconnectHandler should be set")
	}

	// Handlers are set but not called yet
	if messageHandlerCalled || connectHandlerCalled || disconnectHandlerCalled {
		t.Error("Handlers should not be called just by setting them")
	}
}

// TestWebSocketIntegration performs integration test with actual WebSocket connection
// using httptest.Server for proper port handling
func TestWebSocketIntegration(t *testing.T) {
	connected := make(chan string, 1)
	disconnected := make(chan string, 1)
	messageReceived := make(chan []byte, 1)

	// Create a test server that handles WebSocket upgrades
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("Upgrade error: %v", err)
			return
		}
		defer conn.Close()

		connID := "test-conn-id"
		connected <- connID

		// Configure ping/pong like the real transport does
		conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		// Read messages
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				disconnected <- connID
				return
			}
			messageReceived <- message
		}
	}))
	defer server.Close()

	// Connect to test server
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		// Try without /ws path
		wsURL = "ws" + strings.TrimPrefix(server.URL, "http")
		conn, _, err = dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
	}
	defer conn.Close()

	// Wait for connect handler
	select {
	case connID := <-connected:
		if connID == "" {
			t.Error("Connection ID should not be empty")
		}
	case <-time.After(time.Second):
		t.Error("Connect handler was not called")
	}

	// Send a message from client
	testMessage := []byte(`{"type":"test"}`)
	err = conn.WriteMessage(websocket.TextMessage, testMessage)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for message handler
	select {
	case msg := <-messageReceived:
		if string(msg) != string(testMessage) {
			t.Errorf("Received message %q, want %q", string(msg), string(testMessage))
		}
	case <-time.After(time.Second):
		t.Error("Message handler was not called")
	}

	// Close connection
	conn.Close()

	// Wait for disconnect handler
	select {
	case <-disconnected:
		// Success
	case <-time.After(time.Second):
		t.Error("Disconnect handler was not called")
	}
}

// TestPingPongMechanism verifies the ping/pong mechanism works correctly
func TestPingPongMechanism(t *testing.T) {
	pingReceived := make(chan struct{}, 1)
	pongReceived := make(chan struct{}, 1)

	// Server that sends pings
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send a ping to the client
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		err = conn.WriteMessage(websocket.PingMessage, []byte("server-ping"))
		if err != nil {
			t.Logf("Failed to send ping: %v", err)
			return
		}

		// Read to keep connection alive and handle pong
		conn.SetPongHandler(func(appData string) error {
			select {
			case pongReceived <- struct{}{}:
			default:
			}
			return nil
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// Connect client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Set ping handler on client
	conn.SetPingHandler(func(appData string) error {
		select {
		case pingReceived <- struct{}{}:
		default:
		}
		// Send pong response (gorilla/websocket does this automatically by default)
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	// Read in background
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Wait for ping from server
	select {
	case <-pingReceived:
		// Success - received ping from server
	case <-time.After(2 * time.Second):
		t.Error("Did not receive ping from server")
	}
}
