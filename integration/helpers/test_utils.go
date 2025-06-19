//go:build integration

package helpers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConnection はWebSocket接続のテスト用ラッパー
type WebSocketConnection struct {
	conn   *websocket.Conn
	url    string
	closed bool
}

// NewWebSocketConnection は新しいWebSocket接続を作成する
func NewWebSocketConnection(serverURL string) (*WebSocketConnection, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("URLの解析に失敗: %v", err)
	}

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 5 * time.Second

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("WebSocket接続に失敗: %v", err)
	}

	return &WebSocketConnection{
		conn: conn,
		url:  serverURL,
	}, nil
}

// SendMessage はWebSocketメッセージを送信する
func (wsc *WebSocketConnection) SendMessage(message interface{}) error {
	if wsc.closed {
		return fmt.Errorf("接続が既に閉じられています")
	}

	return wsc.conn.WriteJSON(message)
}

// ReceiveMessage はWebSocketメッセージを受信する
func (wsc *WebSocketConnection) ReceiveMessage(timeout time.Duration) (map[string]interface{}, error) {
	if wsc.closed {
		return nil, fmt.Errorf("接続が既に閉じられています")
	}

	// タイムアウトを設定
	if timeout > 0 {
		wsc.conn.SetReadDeadline(time.Now().Add(timeout))
	}

	var message map[string]interface{}
	err := wsc.conn.ReadJSON(&message)
	if err != nil {
		return nil, fmt.Errorf("メッセージの受信に失敗: %v", err)
	}

	return message, nil
}

// WaitForMessage は特定の条件にマッチするメッセージを待機する
func (wsc *WebSocketConnection) WaitForMessage(predicate func(map[string]interface{}) bool, timeout time.Duration) (map[string]interface{}, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		message, err := wsc.ReceiveMessage(time.Until(deadline))
		if err != nil {
			return nil, err
		}

		if predicate(message) {
			return message, nil
		}
	}

	return nil, fmt.Errorf("タイムアウト: 条件にマッチするメッセージが受信されませんでした")
}

// Close はWebSocket接続を閉じる
func (wsc *WebSocketConnection) Close() error {
	if wsc.closed {
		return nil
	}

	wsc.closed = true
	return wsc.conn.Close()
}

// CreateTempFile は一時ファイルを作成する
func CreateTempFile(t *testing.T, content string, suffix string) string {
	t.Helper()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "temp"+suffix)

	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗: %v", err)
	}

	return tempFile
}

// LoadJSONFile はJSONファイルを読み込む
func LoadJSONFile(t *testing.T, filename string) map[string]interface{} {
	t.Helper()

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ファイルの読み込みに失敗: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("JSONの解析に失敗: %v", err)
	}

	return result
}

// AssertEqual は値が等しいことを確認する
func AssertEqual(t *testing.T, expected, actual interface{}, message string) {
	t.Helper()

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%s: 期待値: %v, 実際の値: %v", message, expected, actual)
	}
}

// AssertNotEqual は値が等しくないことを確認する
func AssertNotEqual(t *testing.T, expected, actual interface{}, message string) {
	t.Helper()

	if reflect.DeepEqual(expected, actual) {
		t.Errorf("%s: 値が等しくありません: %v", message, actual)
	}
}

// AssertTrue は値がtrueであることを確認する
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()

	if !condition {
		t.Errorf("%s: 条件がfalseです", message)
	}
}

// AssertFalse は値がfalseであることを確認する
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()

	if condition {
		t.Errorf("%s: 条件がtrueです", message)
	}
}

// AssertNoError はエラーがないことを確認する
func AssertNoError(t *testing.T, err error, message string) {
	t.Helper()

	if err != nil {
		t.Errorf("%s: 予期しないエラー: %v", message, err)
	}
}

// AssertError はエラーがあることを確認する
func AssertError(t *testing.T, err error, message string) {
	t.Helper()

	if err == nil {
		t.Errorf("%s: エラーが期待されましたが、発生しませんでした", message)
	}
}

// WaitForCondition は条件が満たされるまで待機する
func WaitForCondition(condition func() bool, timeout time.Duration, interval time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}

	return false
}

// GetAbsolutePath は相対パスから絶対パスを取得する
func GetAbsolutePath(relativePath string) (string, error) {
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		return "", fmt.Errorf("絶対パスの取得に失敗: %v", err)
	}

	return absPath, nil
}

// FileExists はファイルが存在するかどうかを確認する
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// RemoveFileIfExists はファイルが存在する場合は削除する
func RemoveFileIfExists(filename string) error {
	if FileExists(filename) {
		return os.Remove(filename)
	}
	return nil
}
