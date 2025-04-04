package server

import (
	"context"
	"echonet-list/echonet_lite/log"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// StartOptions は websocket_server.go で定義されています

// WebSocketTransport はWebSocketサーバーのネットワーク層を抽象化するインターフェース
type WebSocketTransport interface {
	// Start はWebSocketサーバーを起動する
	Start(options StartOptions) error

	// Stop はWebSocketサーバーを停止する
	Stop() error

	// SetMessageHandler はクライアントからメッセージを受信した時に呼び出されるハンドラを設定する
	// connID はクライアント接続を識別するための一意なID
	SetMessageHandler(handler func(connID string, message []byte) error)

	// SetConnectHandler は新しいクライアントが接続した時に呼び出されるハンドラを設定する
	// connID はクライアント接続を識別するための一意なID
	SetConnectHandler(handler func(connID string) error)

	// SetDisconnectHandler はクライアントが切断した時に呼び出されるハンドラを設定する
	// connID はクライアント接続を識別するための一意なID
	SetDisconnectHandler(handler func(connID string))

	// SendMessage は特定のクライアントにメッセージを送信する
	// connID はクライアント接続を識別するための一意なID
	SendMessage(connID string, message []byte) error

	// BroadcastMessage は接続中の全クライアントにメッセージを送信する
	BroadcastMessage(message []byte) error
}

// DefaultWebSocketTransport は WebSocketTransport インターフェースのデフォルト実装
type DefaultWebSocketTransport struct {
	ctx               context.Context
	cancel            context.CancelFunc
	server            *http.Server
	upgrader          websocket.Upgrader
	clients           map[string]*websocket.Conn
	clientsReverse    map[*websocket.Conn]string
	clientsMutex      sync.RWMutex
	messageHandler    func(connID string, message []byte) error
	connectHandler    func(connID string) error
	disconnectHandler func(connID string)
}

// NewDefaultWebSocketTransport は DefaultWebSocketTransport の新しいインスタンスを作成する
func NewDefaultWebSocketTransport(ctx context.Context, addr string) *DefaultWebSocketTransport {
	transportCtx, cancel := context.WithCancel(ctx)

	transport := &DefaultWebSocketTransport{
		ctx:            transportCtx,
		cancel:         cancel,
		upgrader:       websocket.Upgrader{},
		clients:        make(map[string]*websocket.Conn),
		clientsReverse: make(map[*websocket.Conn]string),
		clientsMutex:   sync.RWMutex{},
	}

	// Create the HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", transport.handleWebSocket)

	transport.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return transport
}

// Start はWebSocketサーバーを起動する
func (t *DefaultWebSocketTransport) Start(options StartOptions) error {
	logger := log.GetLogger()
	if logger != nil {
		logger.Log("WebSocket server starting on %s", t.server.Addr)
	} else {
		fmt.Printf("WebSocket server starting on %s\n", t.server.Addr)
	}

	// TLS証明書が指定されている場合
	if options.CertFile != "" && options.KeyFile != "" {
		if logger != nil {
			logger.Log("Using TLS with certificate: %s", options.CertFile)
		} else {
			fmt.Printf("Using TLS with certificate: %s\n", options.CertFile)
		}
		return t.server.ListenAndServeTLS(options.CertFile, options.KeyFile)
	}

	// 通常のHTTP (証明書なし)
	return t.server.ListenAndServe()
}

// Stop はWebSocketサーバーを停止する
func (t *DefaultWebSocketTransport) Stop() error {
	logger := log.GetLogger()
	if logger != nil {
		logger.Log("Stopping WebSocket server")
	}
	t.cancel()
	err := t.server.Shutdown(context.Background())
	if err != nil && logger != nil {
		logger.Log("Error shutting down WebSocket server: %v", err)
	}
	return err
}

// SetMessageHandler はクライアントからメッセージを受信した時に呼び出されるハンドラを設定する
func (t *DefaultWebSocketTransport) SetMessageHandler(handler func(connID string, message []byte) error) {
	t.messageHandler = handler
}

// SetConnectHandler は新しいクライアントが接続した時に呼び出されるハンドラを設定する
func (t *DefaultWebSocketTransport) SetConnectHandler(handler func(connID string) error) {
	t.connectHandler = handler
}

// SetDisconnectHandler はクライアントが切断した時に呼び出されるハンドラを設定する
func (t *DefaultWebSocketTransport) SetDisconnectHandler(handler func(connID string)) {
	t.disconnectHandler = handler
}

// SendMessage は特定のクライアントにメッセージを送信する
func (t *DefaultWebSocketTransport) SendMessage(connID string, message []byte) error {
	t.clientsMutex.RLock()
	conn, exists := t.clients[connID]
	t.clientsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("client with ID %s not found", connID)
	}

	return conn.WriteMessage(websocket.TextMessage, message)
}

// BroadcastMessage は接続中の全クライアントにメッセージを送信する
func (t *DefaultWebSocketTransport) BroadcastMessage(message []byte) error {
	logger := log.GetLogger()

	t.clientsMutex.RLock()
	defer t.clientsMutex.RUnlock()

	for _, conn := range t.clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			if logger != nil {
				logger.Log("Error broadcasting message to client: %v", err)
			}
		}
	}

	return nil
}

// handleWebSocket はWebSocket接続を処理する
func (t *DefaultWebSocketTransport) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger()

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if logger != nil {
			logger.Log("Error upgrading to WebSocket: %v", err)
		}
		return
	}
	defer conn.Close()

	// Generate a unique connection ID
	connID := fmt.Sprintf("%p", conn)

	// Register the client
	t.clientsMutex.Lock()
	t.clients[connID] = conn
	t.clientsReverse[conn] = connID
	t.clientsMutex.Unlock()

	// Remove the client when the function returns
	defer func() {
		t.clientsMutex.Lock()
		delete(t.clients, connID)
		delete(t.clientsReverse, conn)
		t.clientsMutex.Unlock()

		// Call the disconnect handler if set
		if t.disconnectHandler != nil {
			t.disconnectHandler(connID)
		}
	}()

	// Call the connect handler if set
	if t.connectHandler != nil {
		if err := t.connectHandler(connID); err != nil {
			if logger != nil {
				logger.Log("Error in connect handler: %v", err)
			}
			return
		}
	}

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				if logger != nil {
					logger.Log("WebSocket error: %v", err)
				}
			}
			break
		}

		// Call the message handler if set
		if t.messageHandler != nil {
			if err := t.messageHandler(connID, message); err != nil {
				if logger != nil {
					logger.Log("Error in message handler: %v", err)
				}
			}
		}
	}
}
