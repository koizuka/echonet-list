package client

import (
	"context"
	"net/url"

	"github.com/gorilla/websocket"
)

// WebSocketClientTransport はWebSocketクライアントのネットワーク層を抽象化するインターフェース
type WebSocketClientTransport interface {
	// Connect はWebSocketサーバーに接続する
	Connect() error

	// Close は接続を閉じる
	Close() error

	// ReadMessage はWebSocketサーバーからメッセージを読み込む
	ReadMessage() (messageType int, p []byte, err error)

	// WriteMessage はWebSocketサーバーにメッセージを送信する
	WriteMessage(messageType int, data []byte) error

	// IsConnected は接続が確立されているかどうかを返す
	IsConnected() bool
}

// DefaultWebSocketClientTransport は WebSocketClientTransport インターフェースのデフォルト実装
type DefaultWebSocketClientTransport struct {
	ctx     context.Context
	url     string
	conn    *websocket.Conn
	dialer  *websocket.Dialer
	isDebug bool
}

// NewDefaultWebSocketClientTransport は DefaultWebSocketClientTransport の新しいインスタンスを作成する
func NewDefaultWebSocketClientTransport(ctx context.Context, serverURL string, debug bool) (*DefaultWebSocketClientTransport, error) {
	// URLの検証
	_, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	return &DefaultWebSocketClientTransport{
		ctx:     ctx,
		url:     serverURL,
		dialer:  websocket.DefaultDialer,
		isDebug: debug,
	}, nil
}

// Connect はWebSocketサーバーに接続する
func (t *DefaultWebSocketClientTransport) Connect() error {
	conn, _, err := t.dialer.Dial(t.url, nil)
	if err != nil {
		return err
	}
	t.conn = conn
	return nil
}

// Close は接続を閉じる
func (t *DefaultWebSocketClientTransport) Close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

// ReadMessage はWebSocketサーバーからメッセージを読み込む
func (t *DefaultWebSocketClientTransport) ReadMessage() (messageType int, p []byte, err error) {
	if t.conn == nil {
		return 0, nil, websocket.ErrCloseSent
	}
	return t.conn.ReadMessage()
}

// WriteMessage はWebSocketサーバーにメッセージを送信する
func (t *DefaultWebSocketClientTransport) WriteMessage(messageType int, data []byte) error {
	if t.conn == nil {
		return websocket.ErrCloseSent
	}
	return t.conn.WriteMessage(messageType, data)
}

// IsConnected は接続が確立されているかどうかを返す
func (t *DefaultWebSocketClientTransport) IsConnected() bool {
	return t.conn != nil
}

// IsDebug はデバッグモードが有効かどうかを返す
func (t *DefaultWebSocketClientTransport) IsDebug() bool {
	return t.isDebug
}

// SetDebug はデバッグモードを設定する
func (t *DefaultWebSocketClientTransport) SetDebug(debug bool) {
	t.isDebug = debug
}
