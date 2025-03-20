package client

import (
	"context"
	"echonet-list/protocol"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ECHONETLiteClient はWebSocket経由でECHONETLiteServerと通信するクライアント
type ECHONETLiteClient struct {
	conn                *websocket.Conn
	url                 string
	responseChannels    map[string]chan protocol.ResponseMessage
	responseChannelsMux sync.Mutex
	notificationCh      chan protocol.NotificationMessage
	done                chan struct{}
	ctx                 context.Context
	cancel              context.CancelFunc
}

// NewECHONETLiteClient は新しいECHONETLiteClientを作成する
func NewECHONETLiteClient(ctx context.Context, url string) (*ECHONETLiteClient, error) {
	clientCtx, cancel := context.WithCancel(ctx)

	client := &ECHONETLiteClient{
		url:              url,
		responseChannels: make(map[string]chan protocol.ResponseMessage),
		notificationCh:   make(chan protocol.NotificationMessage, 100),
		done:             make(chan struct{}),
		ctx:              clientCtx,
		cancel:           cancel,
	}

	// WebSocketに接続
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("WebSocket接続エラー: %w", err)
	}
	client.conn = conn

	// メッセージ受信ループを開始
	go client.receiveLoop()

	return client, nil
}

// Close はクライアントを閉じる
func (c *ECHONETLiteClient) Close() error {
	c.cancel()
	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	<-c.done // 受信ループが終了するまで待機
	return err
}

// GetNotificationChannel は通知チャネルを返す
func (c *ECHONETLiteClient) GetNotificationChannel() <-chan protocol.NotificationMessage {
	return c.notificationCh
}

// receiveLoop はWebSocketからのメッセージを受信するループ
func (c *ECHONETLiteClient) receiveLoop() {
	defer close(c.done)
	defer c.conn.Close()

	for {
		select {
		case <-c.ctx.Done():
			// コンテキストがキャンセルされた場合は終了
			return
		default:
			// 継続
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket読み取りエラー: %v", err)
			}
			break
		}

		// メッセージタイプを判別
		var baseMsg struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal(message, &baseMsg); err != nil {
			log.Printf("メッセージの解析エラー: %v", err)
			continue
		}

		switch baseMsg.Type {
		case "response":
			var resp protocol.ResponseMessage
			if err := json.Unmarshal(message, &resp); err != nil {
				log.Printf("応答の解析エラー: %v", err)
				continue
			}

			// 対応するレスポンスチャネルにメッセージを送信
			c.responseChannelsMux.Lock()
			if ch, exists := c.responseChannels[resp.ID]; exists {
				ch <- resp
				delete(c.responseChannels, resp.ID)
			}
			c.responseChannelsMux.Unlock()

		case "notification":
			var notification protocol.NotificationMessage
			if err := json.Unmarshal(message, &notification); err != nil {
				log.Printf("通知の解析エラー: %v", err)
				continue
			}

			// 通知チャネルにメッセージを送信
			select {
			case c.notificationCh <- notification:
			default:
				log.Printf("通知チャネルがブロックされています")
			}

		default:
			log.Printf("不明なメッセージタイプ: %s", baseMsg.Type)
		}
	}
}

// sendCommand はコマンドをサーバーに送信する
func (c *ECHONETLiteClient) SendCommand(cmd string, deviceSpec interface{}, epcs []protocol.EPCType, properties interface{}, options interface{}) (*protocol.ResponseMessage, error) {
	// コマンドIDを生成
	id := uuid.New().String()

	// コマンドメッセージを作成
	cmdMsg := protocol.CommandMessage{
		Message: protocol.Message{
			Type: "command",
			ID:   id,
		},
		Command:    cmd,
		DeviceSpec: deviceSpec,
		EPCs:       epcs,
		Properties: properties,
		Options:    options,
	}

	// レスポンスチャネルを作成
	respCh := make(chan protocol.ResponseMessage, 1)
	c.responseChannelsMux.Lock()
	c.responseChannels[id] = respCh
	c.responseChannelsMux.Unlock()

	// コマンドを送信
	message, err := json.Marshal(cmdMsg)
	if err != nil {
		c.responseChannelsMux.Lock()
		delete(c.responseChannels, id)
		c.responseChannelsMux.Unlock()
		return nil, fmt.Errorf("コマンドのJSONエンコードに失敗: %w", err)
	}

	if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
		c.responseChannelsMux.Lock()
		delete(c.responseChannels, id)
		c.responseChannelsMux.Unlock()
		return nil, fmt.Errorf("WebSocket書き込みエラー: %w", err)
	}

	// レスポンスを待機
	select {
	case resp := <-respCh:
		return &resp, nil
	case <-time.After(10 * time.Second):
		c.responseChannelsMux.Lock()
		delete(c.responseChannels, id)
		c.responseChannelsMux.Unlock()
		return nil, fmt.Errorf("コマンド応答のタイムアウト")
	case <-c.ctx.Done():
		c.responseChannelsMux.Lock()
		delete(c.responseChannels, id)
		c.responseChannelsMux.Unlock()
		return nil, c.ctx.Err()
	}
}

// Discover はECHONETLiteデバイスを発見する
func (c *ECHONETLiteClient) Discover() error {
	resp, err := c.SendCommand("discover", nil, nil, nil, nil)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(resp.Error)
	}
	return nil
}

// ListDevices は条件に一致するデバイスの一覧を取得する
func (c *ECHONETLiteClient) ListDevices(deviceSpec protocol.DeviceSpecifier, propMode protocol.PropertyMode, epcs []protocol.EPCType, properties []protocol.Property) ([]protocol.DevicePropertyResult, error) {
	// オプションの設定
	options := protocol.CommandOptions{
		PropMode: &propMode,
	}

	resp, err := c.SendCommand("devices", deviceSpec, epcs, properties, options)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(resp.Error)
	}

	// データをDevicePropertyResult配列に変換
	var results []protocol.DevicePropertyResult
	if resp.Data != nil {
		data, err := json.Marshal(resp.Data)
		if err != nil {
			return nil, fmt.Errorf("結果のエンコードに失敗: %w", err)
		}

		if err := json.Unmarshal(data, &results); err != nil {
			return nil, fmt.Errorf("結果のデコードに失敗: %w", err)
		}
	}

	return results, nil
}

// GetProperties はデバイスのプロパティを取得する
func (c *ECHONETLiteClient) GetProperties(deviceSpec protocol.DeviceSpecifier, epcs []protocol.EPCType, skipValidation bool) (*protocol.DevicePropertyResult, error) {
	// オプションの設定
	options := protocol.CommandOptions{
		SkipValidation: &skipValidation,
	}

	resp, err := c.SendCommand("get", deviceSpec, epcs, nil, options)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(resp.Error)
	}

	// データをDevicePropertyResultに変換
	var result protocol.DevicePropertyResult
	if resp.Data != nil {
		data, err := json.Marshal(resp.Data)
		if err != nil {
			return nil, fmt.Errorf("結果のエンコードに失敗: %w", err)
		}

		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("結果のデコードに失敗: %w", err)
		}
	}

	return &result, nil
}

// SetProperties はデバイスのプロパティを設定する
func (c *ECHONETLiteClient) SetProperties(deviceSpec protocol.DeviceSpecifier, properties []protocol.Property) (*protocol.DevicePropertyResult, error) {
	resp, err := c.SendCommand("set", deviceSpec, nil, properties, nil)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf(resp.Error)
	}

	// データをDevicePropertyResultに変換
	var result protocol.DevicePropertyResult
	if resp.Data != nil {
		data, err := json.Marshal(resp.Data)
		if err != nil {
			return nil, fmt.Errorf("結果のエンコードに失敗: %w", err)
		}

		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("結果のデコードに失敗: %w", err)
		}
	}

	return &result, nil
}

// UpdateProperties はデバイスのプロパティキャッシュを更新する
func (c *ECHONETLiteClient) UpdateProperties(deviceSpec protocol.DeviceSpecifier) error {
	resp, err := c.SendCommand("update", deviceSpec, nil, nil, nil)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf(resp.Error)
	}
	return nil
}

// DebugMode はデバッグモードを設定または取得する
func (c *ECHONETLiteClient) DebugMode(mode *string) (bool, error) {
	var options protocol.CommandOptions
	if mode != nil {
		options.DebugMode = mode
	}

	resp, err := c.SendCommand("debug", nil, nil, nil, options)
	if err != nil {
		return false, err
	}
	if !resp.Success {
		return false, fmt.Errorf(resp.Error)
	}

	// データからデバッグモードを取得
	if resp.Data != nil {
		data := resp.Data.(map[string]interface{})
		if debug, ok := data["debug"].(bool); ok {
			return debug, nil
		}
	}

	return false, fmt.Errorf("デバッグモードの取得に失敗")
}