package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// StartOptions は websocket_server.go で定義されていますが、ここにHTTPサーバー用の設定も追加します

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

// clientConnection wraps a WebSocket connection with a mutex for safe concurrent writes
type clientConnection struct {
	conn  *websocket.Conn
	mutex sync.Mutex
}

// DefaultWebSocketTransport は WebSocketTransport インターフェースのデフォルト実装
type DefaultWebSocketTransport struct {
	ctx               context.Context
	cancel            context.CancelFunc
	server            *http.Server
	upgrader          websocket.Upgrader
	clients           map[string]*clientConnection
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
		ctx:    transportCtx,
		cancel: cancel,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development
				return true
			},
		},
		clients:        make(map[string]*clientConnection),
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

// SetupStaticFileServer は静的ファイル配信を設定する
func (t *DefaultWebSocketTransport) SetupStaticFileServer(webRoot string) error {
	if webRoot == "" {
		return nil
	}

	// Webルートディレクトリの存在チェック
	if _, err := os.Stat(webRoot); os.IsNotExist(err) {
		return fmt.Errorf("webroot directory '%s' not found: %v", webRoot, err)
	}

	// 既存のmuxを取得
	if mux, ok := t.server.Handler.(*http.ServeMux); ok {
		// ファイルサーバーのハンドラを作成
		fs := http.FileServer(http.Dir(webRoot))
		// ルートパスに静的ファイル配信を追加（WebSocketより後に追加することで優先度を調整）
		mux.Handle("/", fs)
		slog.Info("Static file server configured", "webroot", webRoot)
	}

	return nil
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

// isConnectionClosedError checks if the error indicates a closed connection
func isConnectionClosedError(err error) bool {
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) ||
		strings.Contains(err.Error(), "close sent") ||
		strings.Contains(err.Error(), "use of closed network connection") ||
		strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "connection reset by peer")
}

// removeClient safely removes a client from the transport and calls the disconnect handler.
// Returns true if the client was actually removed, false if it was already removed.
func (t *DefaultWebSocketTransport) removeClient(connID string) bool {
	t.clientsMutex.Lock()
	defer t.clientsMutex.Unlock()

	client, exists := t.clients[connID]
	if !exists {
		return false
	}

	delete(t.clients, connID)
	if client.conn != nil {
		delete(t.clientsReverse, client.conn)
	}

	// Call disconnect handler outside of the mutex lock
	go func() {
		select {
		case <-t.ctx.Done():
			return
		default:
			if t.disconnectHandler != nil {
				t.disconnectHandler(connID)
			}
		}
	}()

	return true
}

// SendMessage は特定のクライアントにメッセージを送信する
func (t *DefaultWebSocketTransport) SendMessage(connID string, message []byte) error {
	t.clientsMutex.RLock()
	client, exists := t.clients[connID]
	t.clientsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("client with ID %s not found", connID)
	}

	client.mutex.Lock()
	defer client.mutex.Unlock()

	err := client.conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		// Check if this is a connection close error
		if isConnectionClosedError(err) {
			// Remove the client using the common method
			t.removeClient(connID)
		}
		return fmt.Errorf("failed to send message to client %s: %w", connID, err)
	}

	return nil
}

// BroadcastMessage は接続中の全クライアントにメッセージを送信する
func (t *DefaultWebSocketTransport) BroadcastMessage(message []byte) error {
	t.clientsMutex.RLock()
	clients := make(map[string]*clientConnection)
	clientsReverse := make(map[*websocket.Conn]string)
	for connID, client := range t.clients {
		clients[connID] = client
		clientsReverse[client.conn] = connID
	}
	t.clientsMutex.RUnlock()

	var disconnectedClients []string

	for connID, client := range clients {
		client.mutex.Lock()
		if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			// Check if this is a client disconnection error
			if isConnectionClosedError(err) {
				disconnectedClients = append(disconnectedClients, connID)
			} else {
				slog.Error("Error broadcasting message to client", "err", err, "connID", connID)
			}
		}
		client.mutex.Unlock()
	}

	// Clean up disconnected clients using the common method
	for _, connID := range disconnectedClients {
		t.removeClient(connID)
	}

	return nil
}

// handleWebSocket はWebSocket接続を処理する
func (t *DefaultWebSocketTransport) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	slog.Debug("WebSocket upgrade request received",
		"origin", r.Header.Get("Origin"),
		"host", r.Header.Get("Host"),
		"upgrade", r.Header.Get("Upgrade"),
		"connection", r.Header.Get("Connection"),
		"sec-websocket-key", r.Header.Get("Sec-WebSocket-Key"),
		"sec-websocket-version", r.Header.Get("Sec-WebSocket-Version"))

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading to WebSocket", "err", err,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.Header.Get("User-Agent"))
		return
	}
	defer conn.Close()

	// Generate a unique connection ID
	connID := fmt.Sprintf("%p", conn)

	// Register the client
	client := &clientConnection{
		conn:  conn,
		mutex: sync.Mutex{},
	}
	t.clientsMutex.Lock()
	t.clients[connID] = client
	t.clientsReverse[conn] = connID
	t.clientsMutex.Unlock()

	// Remove the client when the function returns
	defer func() {
		t.removeClient(connID)
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
			// Check for unexpected close errors. Expected close codes:
			// - 1000 (Normal): Intentional client disconnect (e.g., HMR, manual close)
			// - 1001 (Going Away): Browser navigation or server shutdown
			// - 1005 (No Status): No close code provided
			// - 1006 (Abnormal): Connection lost without close frame
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				slog.Error("Unexpected WebSocket close error", "err", err)
			}
			break
		}

		// Call the message handler if set
		if t.messageHandler != nil {
			if err := t.messageHandler(connID, message); err != nil {
				// Check if this is a client disconnection error
				errStr := err.Error()
				if !isConnectionClosedError(err) &&
					!(strings.Contains(errStr, "client with ID") && strings.Contains(errStr, "not found")) {
					// Only log non-disconnection errors
					slog.Error("Error in message handler", "err", err)
				}
			}
		}
	}
}
