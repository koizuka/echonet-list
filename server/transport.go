package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
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
	// 先にリスナーをバインド
	listener, err := net.Listen("tcp", t.server.Addr)
	if err != nil {
		return err
	}
	// 待ち受け完了を通知
	if options.Ready != nil {
		close(options.Ready)
	}
	slog.Info("WebSocket server starting", "addr", t.server.Addr)

	// TLS証明書が指定されている場合
	if options.CertFile != "" && options.KeyFile != "" {
		slog.Info("Using TLS with certificate", "certFile", options.CertFile)
		return t.server.ServeTLS(listener, options.CertFile, options.KeyFile)
	}

	// 通常のHTTP (証明書なし)
	return t.server.Serve(listener)
}

// Stop はWebSocketサーバーを停止する
func (t *DefaultWebSocketTransport) Stop() error {
	slog.Info("Stopping WebSocket server", "addr", t.server.Addr)
	t.cancel()
	err := t.server.Shutdown(context.Background())
	if err != nil {
		slog.Info("Error shutting down WebSocket server", "err", err)
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
	t.clientsMutex.RLock()
	defer t.clientsMutex.RUnlock()

	for _, conn := range t.clients {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			slog.Error("Error broadcasting message to client", "err", err)
		}
	}

	return nil
}

// handleWebSocket はWebSocket接続を処理する
func (t *DefaultWebSocketTransport) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading to WebSocket", "err", err)
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
			slog.Error("Error in connect handler", "err", err)
			return
		}
	}

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("Unexpected WebSocket close error", "err", err)
			}
			break
		}

		// Call the message handler if set
		if t.messageHandler != nil {
			if err := t.messageHandler(connID, message); err != nil {
				slog.Error("Error in message handler", "err", err)
			}
		}
	}
}
