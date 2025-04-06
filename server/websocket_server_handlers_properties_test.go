package server

import (
	"context"
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// mockWebSocketTransport は WebSocketTransport インターフェースのモック実装
type mockWebSocketTransport struct {
	sentMessages map[string][]byte // connID -> message
}

func newMockWebSocketTransport() *mockWebSocketTransport {
	return &mockWebSocketTransport{
		sentMessages: make(map[string][]byte),
	}
}

// Start はモックの Start メソッド
func (m *mockWebSocketTransport) Start(options StartOptions) error {
	return nil
}

// Stop はモックの Stop メソッド
func (m *mockWebSocketTransport) Stop() error {
	return nil
}

// SetMessageHandler はモックの SetMessageHandler メソッド
func (m *mockWebSocketTransport) SetMessageHandler(handler func(connID string, message []byte) error) {
}

// SetConnectHandler はモックの SetConnectHandler メソッド
func (m *mockWebSocketTransport) SetConnectHandler(handler func(connID string) error) {
}

// SetDisconnectHandler はモックの SetDisconnectHandler メソッド
func (m *mockWebSocketTransport) SetDisconnectHandler(handler func(connID string)) {
}

// SendMessage はモックの SendMessage メソッド
func (m *mockWebSocketTransport) SendMessage(connID string, message []byte) error {
	m.sentMessages[connID] = message
	return nil
}

// BroadcastMessage はモックの BroadcastMessage メソッド
func (m *mockWebSocketTransport) BroadcastMessage(message []byte) error {
	return nil
}

// mockECHONETListClient は ECHONETListClient インターフェースのモック実装
type mockECHONETListClient struct {
	debug bool
}

func (m *mockECHONETListClient) IsDebug() bool {
	return m.debug
}

func (m *mockECHONETListClient) SetDebug(debug bool) {
	m.debug = debug
}

func (m *mockECHONETListClient) AliasList() []echonet_lite.AliasIDStringPair {
	return nil
}

func (m *mockECHONETListClient) AliasSet(alias *string, criteria echonet_lite.FilterCriteria) error {
	return nil
}

func (m *mockECHONETListClient) AliasDelete(alias *string) error {
	return nil
}

func (m *mockECHONETListClient) AliasGet(alias *string) (*echonet_lite.IPAndEOJ, error) {
	return nil, nil
}

func (m *mockECHONETListClient) GetAliases(device echonet_lite.IPAndEOJ) []string {
	return nil
}

func (m *mockECHONETListClient) GetDeviceByAlias(alias string) (echonet_lite.IPAndEOJ, bool) {
	return echonet_lite.IPAndEOJ{}, false
}

func (m *mockECHONETListClient) Discover() error {
	return nil
}

func (m *mockECHONETListClient) UpdateProperties(criteria echonet_lite.FilterCriteria) error {
	return nil
}

func (m *mockECHONETListClient) GetDevices(deviceSpec echonet_lite.DeviceSpecifier) []echonet_lite.IPAndEOJ {
	return nil
}

func (m *mockECHONETListClient) ListDevices(criteria echonet_lite.FilterCriteria) []echonet_lite.DeviceAndProperties {
	return nil
}

func (m *mockECHONETListClient) GetProperties(device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType, skipValidation bool) (echonet_lite.DeviceAndProperties, error) {
	return echonet_lite.DeviceAndProperties{}, nil
}

func (m *mockECHONETListClient) SetProperties(device echonet_lite.IPAndEOJ, properties echonet_lite.Properties) (echonet_lite.DeviceAndProperties, error) {
	return echonet_lite.DeviceAndProperties{}, nil
}

func (m *mockECHONETListClient) GetAllPropertyAliases() []string {
	return nil
}

func (m *mockECHONETListClient) GetPropertyInfo(classCode echonet_lite.EOJClassCode, e echonet_lite.EPCType) (*echonet_lite.PropertyInfo, bool) {
	return nil, false
}

func (m *mockECHONETListClient) IsPropertyDefaultEPC(classCode echonet_lite.EOJClassCode, epc echonet_lite.EPCType) bool {
	return false
}

func (m *mockECHONETListClient) FindPropertyAlias(classCode echonet_lite.EOJClassCode, alias string) (echonet_lite.Property, bool) {
	return echonet_lite.Property{}, false
}

func (m *mockECHONETListClient) AvailablePropertyAliases(classCode echonet_lite.EOJClassCode) map[string]string {
	// テスト用のエイリアスマップを返す
	if classCode == echonet_lite.HomeAirConditioner_ClassCode {
		return map[string]string{
			"on":      "80(Operation status):30",
			"off":     "80(Operation status):31",
			"auto":    "B0(Operation mode setting):41",
			"cooling": "B0(Operation mode setting):42",
			"heating": "B0(Operation mode setting):43",
			"dry":     "B0(Operation mode setting):44",
			"fan":     "B0(Operation mode setting):45",
		}
	}
	return map[string]string{}
}

func (m *mockECHONETListClient) GroupList(groupName *string) []echonet_lite.GroupDevicePair {
	return nil
}

func (m *mockECHONETListClient) GroupAdd(groupName string, devices []echonet_lite.IPAndEOJ) error {
	return nil
}

func (m *mockECHONETListClient) GroupRemove(groupName string, devices []echonet_lite.IPAndEOJ) error {
	return nil
}

func (m *mockECHONETListClient) GroupDelete(groupName string) error {
	return nil
}

func (m *mockECHONETListClient) GetDevicesByGroup(groupName string) ([]echonet_lite.IPAndEOJ, bool) {
	return nil, false
}

func (m *mockECHONETListClient) Close() error {
	return nil
}

func (m *mockECHONETListClient) FindDeviceByIDString(id echonet_lite.IDString) *echonet_lite.IPAndEOJ {
	return nil
}

// NotificationChannel は通知チャネルのモック
type NotificationChannel struct {
	ch chan echonet_lite.DeviceNotification
}

func NewNotificationChannel() *NotificationChannel {
	return &NotificationChannel{
		ch: make(chan echonet_lite.DeviceNotification, 100),
	}
}

// TestHandleGetPropertyAliasesFromClient は handleGetPropertyAliasesFromClient メソッドのテスト
func TestHandleGetPropertyAliasesFromClient(t *testing.T) {
	// テストケース
	tests := []struct {
		name       string
		classCode  string
		wantStatus bool
		wantError  *protocol.Error
	}{
		{
			name:       "Valid class code (HomeAirConditioner)",
			classCode:  "0130",
			wantStatus: true,
			wantError:  nil,
		},
		{
			name:       "Empty class code",
			classCode:  "",
			wantStatus: false,
			wantError: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "No class code specified",
			},
		},
		{
			name:       "Invalid class code format",
			classCode:  "ZZZZ",
			wantStatus: false,
			wantError: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "Invalid class code: invalid class code: ZZZZ (must be 4 hexadecimal digits)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの作成
			mockTransport := newMockWebSocketTransport()
			mockClient := &mockECHONETListClient{}

			// WebSocketServer の作成
			ws := &WebSocketServer{
				ctx:           context.Background(),
				transport:     mockTransport,
				echonetClient: mockClient,
				handler:       nil, // テストでは使用しないのでnilでOK
			}

			// テスト用のメッセージを作成
			payload := protocol.GetPropertyAliasesPayload{
				ClassCode: tt.classCode,
			}
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("Failed to marshal payload: %v", err)
			}

			msg := &protocol.Message{
				Type:      protocol.MessageTypeGetPropertyAliases,
				Payload:   payloadBytes,
				RequestID: "test-request-id",
			}

			// テスト対象のメソッドを呼び出す
			connID := "test-conn-id"
			err = ws.handleGetPropertyAliasesFromClient(connID, msg)
			if err != nil {
				t.Fatalf("handleGetPropertyAliasesFromClient() error = %v", err)
			}

			// 送信されたメッセージを取得
			sentMessage, ok := mockTransport.sentMessages[connID]
			if !ok {
				t.Fatalf("No message sent to client")
			}

			// メッセージをパース
			var responseMsg protocol.Message
			if err := json.Unmarshal(sentMessage, &responseMsg); err != nil {
				t.Fatalf("Failed to unmarshal response message: %v", err)
			}

			// レスポンスのタイプを確認
			if responseMsg.Type != protocol.MessageTypePropertyAliasesResult {
				t.Errorf("Response message type = %v, want %v", responseMsg.Type, protocol.MessageTypePropertyAliasesResult)
			}

			// リクエストIDを確認
			if responseMsg.RequestID != msg.RequestID {
				t.Errorf("Response requestId = %v, want %v", responseMsg.RequestID, msg.RequestID)
			}

			// ペイロードをパース
			var responsePayload protocol.PropertyAliasesResultPayload
			if err := json.Unmarshal(responseMsg.Payload, &responsePayload); err != nil {
				t.Fatalf("Failed to unmarshal response payload: %v", err)
			}

			// 成功ステータスを確認
			if responsePayload.Success != tt.wantStatus {
				t.Errorf("Response success = %v, want %v", responsePayload.Success, tt.wantStatus)
			}

			// エラーを確認
			if diff := cmp.Diff(tt.wantError, responsePayload.Error); diff != "" {
				t.Errorf("Response error mismatch (-want +got):\n%s", diff)
			}

			// 成功の場合、データの内容を確認
			if tt.wantStatus {
				if responsePayload.Data == nil {
					t.Errorf("Response data is nil, want non-nil")
				} else {
					// クラスコードを確認
					if responsePayload.Data.ClassCode != tt.classCode {
						t.Errorf("Response data.classCode = %v, want %v", responsePayload.Data.ClassCode, tt.classCode)
					}

					// プロパティマップが存在することを確認
					if responsePayload.Data.Properties == nil {
						t.Errorf("Response data.properties is nil, want non-nil")
					}

					// HomeAirConditionerの場合、特定のEPCとエイリアスが含まれていることを確認
					if tt.classCode == "0130" {
						// "80" EPCが含まれていることを確認
						epcInfo, ok := responsePayload.Data.Properties["80"]
						if !ok {
							t.Errorf("Response data.properties does not contain '80' EPC")
						} else {
							// 説明が"Operation status"であることを確認
							if epcInfo.Description != "Operation status" {
								t.Errorf("Response data.properties['80'].description = %v, want %v", epcInfo.Description, "Operation status")
							}

							// "on"エイリアスが含まれていることを確認
							_, ok := epcInfo.Aliases["on"]
							if !ok {
								t.Errorf("Response data.properties['80'].aliases does not contain 'on' alias")
							}
						}
					}
				}
			}
		})
	}
}
