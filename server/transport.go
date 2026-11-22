package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait は書き込み操作のタイムアウト時間
	writeWait = 10 * time.Second
	// pongWait はクライアントからのPongを待つ時間
	pongWait = 60 * time.Second
	// pingPeriod はPingを送信する間隔（pongWaitより短くする必要がある）
	pingPeriod = 54 * time.Second
	// defaultSendBufferSize はクライアントごとの送信キューの長さ。
	// これを超えて溜まったクライアントは「受信できていない」とみなして切断する。
	// ブラウザはスリープ等でいつでも通信が止まりうるが、再接続時に初期状態を
	// 丸ごと受け取って最新状態に回復できるため、待たずに切ってよい。
	defaultSendBufferSize = 512
	// defaultSendBufferBytes はクライアントごとの送信キューに溜められる合計バイト数の上限。
	// メモリの少ない環境（Raspberry Pi 等）で詰まったクライアントがメモリを食い潰さないようにする。
	// キューが空のときは上限を超える1メッセージ（大きな初期状態など）も受け付ける。
	defaultSendBufferBytes = 4 * 1024 * 1024
)

// ErrClientDisconnected は送信先クライアントが切断済み、または送信キューが溢れて
// 切断されたことを示す
var ErrClientDisconnected = errors.New("websocket client disconnected")

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

// clientConnection は1つのWebSocket接続を表す。
// conn への書き込みは writePump ゴルーチンだけが行い、他のゴルーチンは send キューに
// 積むだけにする。これにより送信側（ブロードキャストやログ出力）がネットワーク書き込みで
// ブロックしたり、ロック保持中のログで再入デッドロックしたりしない (issue #971)。
type clientConnection struct {
	conn      *websocket.Conn // immutable after creation
	send      chan []byte     // 送信キュー（close しない。終了は done で通知）
	done      chan struct{}   // 接続終了時に close される
	closeOnce sync.Once
	mu        sync.Mutex // closed・queuedBytes とキュー投入を排他する（保持中にブロックする処理はしない）
	closed    bool
	maxBytes  int // 送信キューの合計バイト数上限
	queued    int // 送信キューに積まれている合計バイト数
}

func newClientConnection(conn *websocket.Conn, bufferSize, maxBytes int) *clientConnection {
	return &clientConnection{
		conn:     conn,
		send:     make(chan []byte, bufferSize),
		done:     make(chan struct{}),
		maxBytes: maxBytes,
	}
}

// enqueue はメッセージを送信キューに積む。ブロックしない。
// 切断済み、またはキューが満杯の場合は false を返す。
func (c *clientConnection) enqueue(message []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	if c.queued > 0 && c.queued+len(message) > c.maxBytes {
		return false
	}
	select {
	case c.send <- message:
		c.queued += len(message)
		return true
	default:
		return false
	}
}

// dequeued は writePump がキューから取り出したメッセージ分のバイト数を戻す
func (c *clientConnection) dequeued(message []byte) {
	c.mu.Lock()
	c.queued -= len(message)
	c.mu.Unlock()
}

// close は接続を閉じる。何度呼んでもよい。
// conn を閉じると読み取りループがエラーで抜け、クライアントの登録解除が行われる。
func (c *clientConnection) close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		close(c.done)
		if c.conn != nil {
			c.conn.Close()
		}
	})
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
	sendBufferSize    int
	sendBufferBytes   int
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
		clients:         make(map[string]*clientConnection),
		clientsReverse:  make(map[*websocket.Conn]string),
		clientsMutex:    sync.RWMutex{},
		sendBufferSize:  defaultSendBufferSize,
		sendBufferBytes: defaultSendBufferBytes,
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
	return errors.Is(err, ErrClientDisconnected) ||
		websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) ||
		strings.Contains(err.Error(), "close sent") ||
		strings.Contains(err.Error(), "use of closed network connection") ||
		strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "connection reset by peer")
}

// removeClient safely removes a client from the transport and calls the disconnect handler.
// target が登録中の接続と同一の場合のみ削除する（同じ connID の別接続を誤って消さないため）。
// Returns true if the client was actually removed, false if it was already removed.
func (t *DefaultWebSocketTransport) removeClient(connID string, target *clientConnection) bool {
	t.clientsMutex.Lock()
	defer t.clientsMutex.Unlock()

	client, exists := t.clients[connID]
	if !exists || client != target {
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

// SendMessage は特定のクライアントにメッセージを送信する。
// 実際の書き込みは非同期で行われ、この関数はブロックしない。
// message はコピーされるので、呼び出し後に再利用してよい。
func (t *DefaultWebSocketTransport) SendMessage(connID string, message []byte) error {
	t.clientsMutex.RLock()
	client, exists := t.clients[connID]
	t.clientsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("client with ID %s not found", connID)
	}

	if !t.enqueueOrDrop(connID, client, bytes.Clone(message)) {
		return fmt.Errorf("failed to send message to client %s: %w", connID, ErrClientDisconnected)
	}
	return nil
}

// BroadcastMessage は接続中の全クライアントにメッセージを送信する。
// 実際の書き込みは非同期で行われ、この関数はブロックしない。
// 受信が追いつかないクライアントは切断する。
// message はコピーされるので、呼び出し後に再利用してよい。
func (t *DefaultWebSocketTransport) BroadcastMessage(message []byte) error {
	t.clientsMutex.RLock()
	clients := make(map[string]*clientConnection, len(t.clients))
	for connID, client := range t.clients {
		clients[connID] = client
	}
	t.clientsMutex.RUnlock()

	if len(clients) == 0 {
		return nil
	}
	// 全クライアントで同じコピーを共有する（writePump は読み取りのみ）
	message = bytes.Clone(message)
	for connID, client := range clients {
		t.enqueueOrDrop(connID, client, message)
	}
	return nil
}

// enqueueOrDrop はメッセージを送信キューに積み、積めなければクライアントを切断する。
// ロックを保持せずに呼ぶこと（ここでのログ出力は BroadcastHandler 経由で再入しうる）。
func (t *DefaultWebSocketTransport) enqueueOrDrop(connID string, client *clientConnection, message []byte) bool {
	if client.enqueue(message) {
		return true
	}
	select {
	case <-client.done:
		// すでに切断処理中
	default:
		slog.Info("WebSocket client is not receiving, disconnecting", "connID", connID, "queued", len(client.send))
		client.close()
	}
	return false
}

// writePump は送信キューと ping を conn に書き込む。conn への書き込みはここだけで行う。
// 書き込みに失敗（タイムアウト含む）したら接続を閉じ、登録も解除して終了する。
// 読み取り側が messageHandler 内で待っていても、閉じた接続が登録に残らないようにするため。
func (t *DefaultWebSocketTransport) writePump(connID string, client *clientConnection) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer func() {
		client.close()
		t.removeClient(connID, client)
	}()

	write := func(messageType int, data []byte) bool {
		client.conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := client.conn.WriteMessage(messageType, data); err != nil {
			if !isConnectionClosedError(err) {
				slog.Info("WebSocket write failed, disconnecting", "connID", connID, "err", err)
			}
			return false
		}
		return true
	}

	for {
		select {
		case message := <-client.send:
			client.dequeued(message)
			if !write(websocket.TextMessage, message) {
				return
			}
		case <-ticker.C:
			if !write(websocket.PingMessage, nil) {
				return
			}
		case <-client.done:
			return
		case <-t.ctx.Done():
			return
		}
	}
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

	// Generate a unique connection ID
	connID := fmt.Sprintf("%p", conn)

	// Register the client
	client := newClientConnection(conn, t.sendBufferSize, t.sendBufferBytes)
	t.clientsMutex.Lock()
	t.clients[connID] = client
	t.clientsReverse[conn] = connID
	t.clientsMutex.Unlock()

	// Remove the client when the function returns
	defer func() {
		client.close()
		t.removeClient(connID, client)
	}()

	// Start writer goroutine (messages and periodic pings).
	// connectHandler が初期状態を送るので、それより先に起動しておく。
	go t.writePump(connID, client)

	// Call the connect handler if set
	if t.connectHandler != nil {
		if err := t.connectHandler(connID); err != nil {
			slog.Error("Error in connect handler", "err", err)
			return
		}
	}

	// Configure Pong handler and read deadline for heartbeat
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

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
