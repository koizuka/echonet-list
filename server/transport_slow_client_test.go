package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// startTestTransport は実際の handleWebSocket を httptest サーバーで公開する
func startTestTransport(t *testing.T, sendBufferSize int) (*DefaultWebSocketTransport, string, chan string, chan string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	transport := NewDefaultWebSocketTransport(ctx, ":0")
	transport.sendBufferSize = sendBufferSize

	connected := make(chan string, 16)
	disconnected := make(chan string, 16)
	transport.SetConnectHandler(func(connID string) error {
		connected <- connID
		return nil
	})
	transport.SetDisconnectHandler(func(connID string) {
		disconnected <- connID
	})

	srv := httptest.NewServer(http.HandlerFunc(transport.handleWebSocket))
	t.Cleanup(srv.Close)
	return transport, "ws" + strings.TrimPrefix(srv.URL, "http"), connected, disconnected
}

func waitConnID(t *testing.T, ch chan string) string {
	t.Helper()
	select {
	case id := <-ch:
		return id
	case <-time.After(2 * time.Second):
		t.Fatal("connect handler was not called")
		return ""
	}
}

// setBroadcastLogger は issue #971 と同じく WARN 以上のログを WebSocket に流す設定にする
func setBroadcastLogger(t *testing.T, transport WebSocketTransport) {
	t.Helper()
	prev := slog.Default()
	inner := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(NewBroadcastHandler(inner, transport, slog.LevelWarn)))
	t.Cleanup(func() { slog.SetDefault(prev) })
}

// TestBroadcastMessageDoesNotBlockOnStalledClient は issue #971 の回帰テスト。
// 受信を止めたクライアント（スリープしたスマホ等）がいても BroadcastMessage は
// ブロックせず、詰まったクライアントは切断され、他のクライアントには配信が続く。
func TestBroadcastMessageDoesNotBlockOnStalledClient(t *testing.T) {
	transport, url, connected, disconnected := startTestTransport(t, 16)
	setBroadcastLogger(t, transport)

	// 受信しないクライアント
	stalled, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial stalled: %v", err)
	}
	defer stalled.Close()
	stalledID := waitConnID(t, connected)

	// 正常に受信するクライアント
	healthy, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial healthy: %v", err)
	}
	defer healthy.Close()
	waitConnID(t, connected)

	var received atomic.Int64
	lastMsg := make(chan struct{})
	go func() {
		for {
			_, msg, err := healthy.ReadMessage()
			if err != nil {
				return
			}
			if strings.HasPrefix(string(msg), "last") {
				close(lastMsg)
				return
			}
			received.Add(1)
		}
	}()

	payload := []byte(strings.Repeat("x", 64*1024))
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 400; i++ {
			_ = transport.BroadcastMessage(payload)
			// 健全なクライアントが追いつける程度のペースで送る
			time.Sleep(time.Millisecond)
			// ロック保持中ログによる再入が起きないことも確認するため WARN を混ぜる
			if i%50 == 0 {
				slog.Warn("test warning during broadcast", "i", i)
			}
		}
		_ = transport.BroadcastMessage([]byte("last"))
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("BroadcastMessage blocked on stalled client (deadlock?)")
	}

	select {
	case id := <-disconnected:
		if id != stalledID {
			t.Errorf("disconnected %s, want stalled client %s", id, stalledID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stalled client was not disconnected")
	}

	select {
	case <-lastMsg:
	case <-time.After(5 * time.Second):
		t.Fatalf("healthy client did not receive last message (received %d)", received.Load())
	}
}

// TestSendMessageDoesNotBlockOnStalledClient は SendMessage も詰まったクライアントで
// ブロックせず、エラーを返して切断することを確認する。
func TestSendMessageDoesNotBlockOnStalledClient(t *testing.T) {
	transport, url, connected, disconnected := startTestTransport(t, 4)
	setBroadcastLogger(t, transport)

	stalled, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer stalled.Close()
	id := waitConnID(t, connected)

	payload := []byte(strings.Repeat("x", 64*1024))
	var sendErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 400; i++ {
			if err := transport.SendMessage(id, payload); err != nil {
				sendErr = err
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SendMessage blocked on stalled client")
	}
	if sendErr == nil {
		t.Fatal("expected SendMessage to fail for stalled client")
	}
	if !errors.Is(sendErr, ErrClientDisconnected) {
		t.Errorf("err = %v, want ErrClientDisconnected", sendErr)
	}
	if !isConnectionClosedError(sendErr) {
		t.Errorf("isConnectionClosedError(%v) = false, want true", sendErr)
	}

	select {
	case <-disconnected:
	case <-time.After(5 * time.Second):
		t.Fatal("stalled client was not disconnected")
	}
}

// TestSendAndBroadcastPreserveOrder は SendMessage と BroadcastMessage が
// 同一クライアントへ呼び出し順に届くことを確認する（初期状態→差分の順序保証）。
func TestSendAndBroadcastPreserveOrder(t *testing.T) {
	transport, url, connected, _ := startTestTransport(t, 64)

	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	id := waitConnID(t, connected)

	want := []string{"a", "b", "c", "d"}
	_ = transport.SendMessage(id, []byte("a"))
	_ = transport.BroadcastMessage([]byte("b"))
	_ = transport.SendMessage(id, []byte("c"))
	_ = transport.BroadcastMessage([]byte("d"))

	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	for _, w := range want {
		_, msg, err := c.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if string(msg) != w {
			t.Errorf("got %q, want %q", msg, w)
		}
	}
}
