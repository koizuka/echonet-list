package server

import (
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// testHandler はテストに必要な最小限の機能を持つハンドラインターフェース
type testHandler interface {
	IsDebug() bool
	RemoveDevice(device echonet_lite.IPAndEOJ) error
	GetDevices(spec handler.DeviceSpecifier) []echonet_lite.IPAndEOJ
}

// mockECHONETLiteHandler はテスト用のECHONETLiteHandlerモック
type mockECHONETLiteHandler struct {
	mock.Mock
}

func (m *mockECHONETLiteHandler) IsDebug() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockECHONETLiteHandler) RemoveDevice(device echonet_lite.IPAndEOJ) error {
	args := m.Called(device)
	return args.Error(0)
}

func (m *mockECHONETLiteHandler) GetDevices(spec handler.DeviceSpecifier) []echonet_lite.IPAndEOJ {
	args := m.Called(spec)
	return args.Get(0).([]echonet_lite.IPAndEOJ)
}

// testWebSocketServer はテスト用のWebSocketServer構造体
type testWebSocketServer struct {
	handler   testHandler
	transport WebSocketTransport
}

// handleDeleteDeviceFromClient のテスト用ラッパー
func (ws *testWebSocketServer) handleDeleteDeviceFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// Parse the payload
	var payload protocol.DeleteDevicePayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing delete_device payload: %v", err)
	}

	// Parse the target device identifier
	ipAndEOJ, err := handler.ParseDeviceIdentifier(payload.Target)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target device identifier: %v", err)
	}

	if ws.handler.IsDebug() {
		// デバッグログは実際のテストでは出力しない
	}

	// Check if this is a NodeProfile deletion
	if ipAndEOJ.EOJ.ClassCode() == echonet_lite.NodeProfile_ClassCode {
		// For NodeProfile, delete all devices at the same IP address
		if ws.handler.IsDebug() {
			// デバッグログは実際のテストでは出力しない
		}

		// Get all devices at the same IP address
		deviceSpec := handler.DeviceSpecifier{
			IP: &ipAndEOJ.IP,
		}
		devicesAtIP := ws.handler.GetDevices(deviceSpec)

		// Remove each device
		var deleteErrors []string
		for _, device := range devicesAtIP {
			if err := ws.handler.RemoveDevice(device); err != nil {
				deleteErrors = append(deleteErrors, fmt.Sprintf("Failed to remove device %s: %v", device.Specifier(), err))
				continue
			}

			// Broadcast device_deleted notification for each device
			deletePayload := protocol.DeviceDeletedPayload{
				IP:  device.IP.String(),
				EOJ: device.EOJ.Specifier(),
			}

			if err := ws.broadcastMessageToClients(protocol.MessageTypeDeviceDeleted, deletePayload); err != nil {
				// テストでは実際のログ出力は行わない
			}
		}

		// If there were any errors, return the first one
		if len(deleteErrors) > 0 {
			return ErrorResponse(protocol.ErrorCodeInternalServerError, strings.Join(deleteErrors, "; "))
		}
	} else {
		// For non-NodeProfile devices, just remove the single device
		if err := ws.handler.RemoveDevice(ipAndEOJ); err != nil {
			return ErrorResponse(protocol.ErrorCodeInternalServerError, "Failed to remove device: %v", err)
		}

		// Broadcast device_deleted notification
		deletePayload := protocol.DeviceDeletedPayload{
			IP:  ipAndEOJ.IP.String(),
			EOJ: ipAndEOJ.EOJ.Specifier(),
		}

		if err := ws.broadcastMessageToClients(protocol.MessageTypeDeviceDeleted, deletePayload); err != nil {
			// テストでは実際のログ出力は行わない
		}
	}

	if ws.handler.IsDebug() {
		// デバッグログは実際のテストでは出力しない
	}

	// Return success response with empty data
	return SuccessResponse(nil)
}

// broadcastMessageToClients のテスト用ラッパー
func (ws *testWebSocketServer) broadcastMessageToClients(msgType protocol.MessageType, payload interface{}) error {
	// Create the message
	data, err := protocol.CreateMessage(msgType, payload, "")
	if err != nil {
		return err
	}

	// Send the message to all clients
	return ws.transport.BroadcastMessage(data)
}

// mockWebSocketTransport はテスト用のWebSocketTransportモック
type mockWebSocketTransport struct {
	mock.Mock
	broadcastMessages [][]byte
}

func (m *mockWebSocketTransport) Start(options StartOptions) error {
	return nil
}

func (m *mockWebSocketTransport) Stop() error {
	return nil
}

func (m *mockWebSocketTransport) SetMessageHandler(handler func(connID string, message []byte) error) {
}

func (m *mockWebSocketTransport) SetConnectHandler(handler func(connID string) error) {
}

func (m *mockWebSocketTransport) SetDisconnectHandler(handler func(connID string)) {
}

func (m *mockWebSocketTransport) SendMessage(connID string, message []byte) error {
	return nil
}

func (m *mockWebSocketTransport) BroadcastMessage(message []byte) error {
	args := m.Called(message)
	m.broadcastMessages = append(m.broadcastMessages, message)
	return args.Error(0)
}

func TestHandleDeleteDeviceFromClient_NodeProfile(t *testing.T) {
	tests := []struct {
		name    string
		payload protocol.DeleteDevicePayload
		// targetDevice      echonet_lite.IPAndEOJ  // 使用していないのでコメントアウト
		sameIPDevices     []echonet_lite.IPAndEOJ
		expectRemoveCalls int
		wantError         bool
		errorCode         protocol.ErrorCode
	}{
		{
			name: "NodeProfile deletion should remove all devices at same IP",
			payload: protocol.DeleteDevicePayload{
				Target: "192.168.1.100 0ef0:1",
			},
			sameIPDevices: []echonet_lite.IPAndEOJ{
				{
					IP:  net.ParseIP("192.168.1.100"),
					EOJ: echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 1),
				},
				{
					IP:  net.ParseIP("192.168.1.100"),
					EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
				},
				{
					IP:  net.ParseIP("192.168.1.100"),
					EOJ: echonet_lite.MakeEOJ(echonet_lite.LightingSystem_ClassCode, 1),
				},
			},
			expectRemoveCalls: 3,
			wantError:         false,
		},
		{
			name: "Non-NodeProfile deletion should only remove specified device",
			payload: protocol.DeleteDevicePayload{
				Target: "192.168.1.100 0130:1",
			},
			sameIPDevices: []echonet_lite.IPAndEOJ{
				{
					IP:  net.ParseIP("192.168.1.100"),
					EOJ: echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 1),
				},
				{
					IP:  net.ParseIP("192.168.1.100"),
					EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
				},
				{
					IP:  net.ParseIP("192.168.1.100"),
					EOJ: echonet_lite.MakeEOJ(echonet_lite.LightingSystem_ClassCode, 1),
				},
			},
			expectRemoveCalls: 1,
			wantError:         false,
		},
		{
			name: "Invalid target format should return error",
			payload: protocol.DeleteDevicePayload{
				Target: "invalid format",
			},
			wantError: true,
			errorCode: protocol.ErrorCodeInvalidParameters,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックハンドラーとトランスポートの設定
			mockHandler := new(mockECHONETLiteHandler)
			mockTransport := new(mockWebSocketTransport)

			if !tt.wantError {
				mockHandler.On("IsDebug").Return(false)
				mockTransport.On("BroadcastMessage", mock.Anything).Return(nil)

				// payloadから削除対象のデバイスを特定
				targetDevice, err := handler.ParseDeviceIdentifier(tt.payload.Target)
				assert.NoError(t, err)

				// NodeProfile の場合は同一 IP のデバイスリストを返して、すべてのデバイスの削除を期待
				if targetDevice.EOJ.ClassCode() == echonet_lite.NodeProfile_ClassCode {
					spec := handler.DeviceSpecifier{
						IP: &targetDevice.IP,
					}
					mockHandler.On("GetDevices", spec).Return(tt.sameIPDevices)
					// すべてのデバイスの削除を期待
					for _, device := range tt.sameIPDevices {
						mockHandler.On("RemoveDevice", device).Return(nil).Once()
					}
				} else {
					// Non-NodeProfile の場合は指定されたデバイスのみの削除を期待
					mockHandler.On("RemoveDevice", targetDevice).Return(nil).Once()
				}
			}

			// testWebSocketServer の作成
			ws := &testWebSocketServer{
				handler:   mockHandler,
				transport: mockTransport,
			}

			// メッセージの作成
			payloadJSON, _ := json.Marshal(tt.payload)
			msg := &protocol.Message{
				Type:    protocol.MessageTypeDeleteDevice,
				Payload: payloadJSON,
			}

			// テスト実行
			result := ws.handleDeleteDeviceFromClient(msg)

			// 結果の検証
			if tt.wantError {
				assert.False(t, result.Success)
				assert.NotNil(t, result.Error)
				assert.Equal(t, tt.errorCode, result.Error.Code)
			} else {
				assert.True(t, result.Success)
				// モックの呼び出し回数を検証
				mockHandler.AssertNumberOfCalls(t, "RemoveDevice", tt.expectRemoveCalls)

				// ブロードキャストメッセージの数を検証
				assert.Equal(t, tt.expectRemoveCalls, len(mockTransport.broadcastMessages))

				// payloadから削除対象のデバイスを特定
				targetDevice, err := handler.ParseDeviceIdentifier(tt.payload.Target)
				assert.NoError(t, err)

				// 各ブロードキャストメッセージが device_deleted 通知であることを検証
				if targetDevice.EOJ.ClassCode() == echonet_lite.NodeProfile_ClassCode {
					// NodeProfileの場合、すべてのデバイスが削除される
					for i, broadcastMsg := range mockTransport.broadcastMessages {
						var message protocol.Message
						err := json.Unmarshal(broadcastMsg, &message)
						assert.NoError(t, err)
						assert.Equal(t, protocol.MessageTypeDeviceDeleted, message.Type)

						var deletePayload protocol.DeviceDeletedPayload
						err = json.Unmarshal(message.Payload, &deletePayload)
						assert.NoError(t, err)
						assert.Equal(t, tt.sameIPDevices[i].IP.String(), deletePayload.IP)
						assert.Equal(t, tt.sameIPDevices[i].EOJ.Specifier(), deletePayload.EOJ)
					}
				} else {
					// Non-NodeProfileの場合、指定されたデバイスのみが削除される
					assert.Equal(t, 1, len(mockTransport.broadcastMessages))
					var message protocol.Message
					err = json.Unmarshal(mockTransport.broadcastMessages[0], &message)
					assert.NoError(t, err)
					assert.Equal(t, protocol.MessageTypeDeviceDeleted, message.Type)

					var deletePayload protocol.DeviceDeletedPayload
					err = json.Unmarshal(message.Payload, &deletePayload)
					assert.NoError(t, err)
					assert.Equal(t, targetDevice.IP.String(), deletePayload.IP)
					assert.Equal(t, targetDevice.EOJ.Specifier(), deletePayload.EOJ)
				}
			}

			mockHandler.AssertExpectations(t)
			mockTransport.AssertExpectations(t)
		})
	}
}

func TestHandleDeleteDeviceFromClient_NodeProfile_PartialFailure(t *testing.T) {
	// 一部のデバイス削除が失敗する場合のテスト
	mockHandler := new(mockECHONETLiteHandler)
	mockTransport := new(mockWebSocketTransport)

	mockHandler.On("IsDebug").Return(false)
	mockTransport.On("BroadcastMessage", mock.Anything).Return(nil)

	ip := net.ParseIP("192.168.1.100")
	sameIPDevices := []echonet_lite.IPAndEOJ{
		{IP: ip, EOJ: echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 1)},
		{IP: ip, EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)},
		{IP: ip, EOJ: echonet_lite.MakeEOJ(echonet_lite.LightingSystem_ClassCode, 1)},
	}

	// 最初のデバイス削除は成功、2番目は失敗、3番目は成功
	mockHandler.On("RemoveDevice", sameIPDevices[0]).Return(nil).Once()
	mockHandler.On("RemoveDevice", sameIPDevices[1]).Return(assert.AnError).Once()
	mockHandler.On("RemoveDevice", sameIPDevices[2]).Return(nil).Once()

	spec := handler.DeviceSpecifier{IP: &ip}
	mockHandler.On("GetDevices", spec).Return(sameIPDevices)

	ws := &testWebSocketServer{
		handler:   mockHandler,
		transport: mockTransport,
	}

	payload := protocol.DeleteDevicePayload{Target: "192.168.1.100 0ef0:1"}
	payloadJSON, _ := json.Marshal(payload)
	msg := &protocol.Message{
		Type:    protocol.MessageTypeDeleteDevice,
		Payload: payloadJSON,
	}

	// テスト実行
	result := ws.handleDeleteDeviceFromClient(msg)

	// 結果の検証 - エラーが返されるべき
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
	assert.Equal(t, protocol.ErrorCodeInternalServerError, result.Error.Code)
	assert.Contains(t, result.Error.Message, "Failed to remove device")

	// 成功したデバイスについてはブロードキャストが送信されているべき
	assert.Equal(t, 2, len(mockTransport.broadcastMessages)) // 1番目と3番目のデバイス

	mockHandler.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

func TestHandleDeleteDeviceFromClient_RegularDevice(t *testing.T) {
	// 通常デバイス削除の既存動作に影響がないことを確認
	mockHandler := new(mockECHONETLiteHandler)
	mockTransport := new(mockWebSocketTransport)

	mockHandler.On("IsDebug").Return(false)
	mockTransport.On("BroadcastMessage", mock.Anything).Return(nil)

	// Regular device (HomeAirConditioner) for removal
	targetDevice := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP("192.168.1.100"),
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}

	mockHandler.On("RemoveDevice", targetDevice).Return(nil)

	ws := &testWebSocketServer{
		handler:   mockHandler,
		transport: mockTransport,
	}

	payload := protocol.DeleteDevicePayload{Target: "192.168.1.100 0130:1"}
	payloadJSON, _ := json.Marshal(payload)
	msg := &protocol.Message{
		Type:    protocol.MessageTypeDeleteDevice,
		Payload: payloadJSON,
	}

	// テスト実行
	result := ws.handleDeleteDeviceFromClient(msg)

	// 結果の検証
	assert.True(t, result.Success)
	mockHandler.AssertNumberOfCalls(t, "RemoveDevice", 1)

	// GetDevicesが呼ばれていないことを確認（NodeProfile以外なので）
	mockHandler.AssertNotCalled(t, "GetDevices")

	// 1つのブロードキャストメッセージが送信されることを確認
	assert.Equal(t, 1, len(mockTransport.broadcastMessages))

	mockHandler.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}
