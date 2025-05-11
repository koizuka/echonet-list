package server

import (
	"context"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

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

func (m *mockECHONETListClient) UpdateProperties(criteria echonet_lite.FilterCriteria, force bool) error {
	return nil
}

func (m *mockECHONETListClient) GetDevices(deviceSpec echonet_lite.DeviceSpecifier) []echonet_lite.IPAndEOJ {
	return nil
}

func (m *mockECHONETListClient) ListDevices(criteria echonet_lite.FilterCriteria) []handler.DeviceAndProperties {
	return nil
}

func (m *mockECHONETListClient) GetProperties(device echonet_lite.IPAndEOJ, EPCs []echonet_lite.EPCType, skipValidation bool) (handler.DeviceAndProperties, error) {
	return handler.DeviceAndProperties{}, nil
}

func (m *mockECHONETListClient) SetProperties(device echonet_lite.IPAndEOJ, properties echonet_lite.Properties) (handler.DeviceAndProperties, error) {
	return handler.DeviceAndProperties{}, nil
}

func (m *mockECHONETListClient) GetAllPropertyAliases() map[string]PropertyDescription {
	return nil
}

func (m *mockECHONETListClient) GetPropertyDesc(classCode echonet_lite.EOJClassCode, e echonet_lite.EPCType) (*echonet_lite.PropertyDesc, bool) {
	return nil, false
}

func (m *mockECHONETListClient) IsPropertyDefaultEPC(classCode echonet_lite.EOJClassCode, epc echonet_lite.EPCType) bool {
	return false
}

func (m *mockECHONETListClient) FindPropertyAlias(classCode echonet_lite.EOJClassCode, alias string) (echonet_lite.Property, bool) {
	return echonet_lite.Property{}, false
}

type PropertyDescription = echonet_lite.PropertyDescription

func (m *mockECHONETListClient) AvailablePropertyAliases(classCode echonet_lite.EOJClassCode) map[string]echonet_lite.PropertyDescription {
	// テスト用のエイリアスマップを返す
	if classCode == 0 { // 共通プロパティを要求された場合
		return map[string]PropertyDescription{
			"on":  {EPC: 0x80, Name: "Operation status", EDT: []byte{0x30}},
			"off": {EPC: 0x80, Name: "Operation status", EDT: []byte{0x31}},
			// 必要に応じて他の共通プロパティエイリアスを追加
			"living": {EPC: 0x81, Name: "Installation location", EDT: []byte{0x01}},
		}
	} else if classCode == echonet_lite.HomeAirConditioner_ClassCode { // デバイス固有プロパティ
		return map[string]PropertyDescription{
			"auto":    {EPC: 0xB0, Name: "Operation mode setting", EDT: []byte{0x41}},
			"cooling": {EPC: 0xB0, Name: "Operation mode setting", EDT: []byte{0x42}},
			"heating": {EPC: 0xB0, Name: "Operation mode setting", EDT: []byte{0x43}},
			"dry":     {EPC: 0xB0, Name: "Operation mode setting", EDT: []byte{0x44}},
			"fan":     {EPC: 0xB0, Name: "Operation mode setting", EDT: []byte{0x45}},
		}
	}
	// その他のクラスコードの場合は空マップを返す
	return map[string]PropertyDescription{}
}

func (m *mockECHONETListClient) GroupList(groupName *string) []echonet_lite.GroupDevicePair {
	return nil
}

func (m *mockECHONETListClient) GroupAdd(groupName string, devices []echonet_lite.IDString) error {
	return nil
}

func (m *mockECHONETListClient) GroupRemove(groupName string, devices []echonet_lite.IDString) error {
	return nil
}

func (m *mockECHONETListClient) GroupDelete(groupName string) error {
	return nil
}

func (m *mockECHONETListClient) GetDevicesByGroup(groupName string) ([]echonet_lite.IDString, bool) {
	return nil, false
}

func (m *mockECHONETListClient) Close() error {
	return nil
}

func (m *mockECHONETListClient) FindDeviceByIDString(id echonet_lite.IDString) *echonet_lite.IPAndEOJ {
	return nil
}

func (m *mockECHONETListClient) GetIDString(device echonet_lite.IPAndEOJ) echonet_lite.IDString {
	return ""
}

// NotificationChannel は通知チャネルのモック
type NotificationChannel struct {
	ch chan handler.DeviceNotification
}

func NewNotificationChannel() *NotificationChannel {
	return &NotificationChannel{
		ch: make(chan handler.DeviceNotification, 100),
	}
}

// --- Mock Property Decoders/Encoders for Testing ---

// MockDecoderOnly implements PropertyDecoder but not PropertyEncoder
type MockDecoderOnly struct{}

func (d MockDecoderOnly) ToString([]byte) (string, bool) { return "decoded_only", true }

// MockDecoderEncoder implements both PropertyDecoder and PropertyEncoder
type MockDecoderEncoder struct{}

func (d MockDecoderEncoder) ToString([]byte) (string, bool)   { return "decoded_encoded", true }
func (d MockDecoderEncoder) FromString(string) ([]byte, bool) { return []byte("encoded"), true } // Implements PropertyEncoder

// --- Test Function for populateEPCDescriptions ---

func TestPopulateEPCDescriptions(t *testing.T) {
	// 1. Create a mock PropertyTable with various decoder types
	mockTable := echonet_lite.PropertyTable{
		ClassCode:   0x0001, // Example class code
		Description: "Mock Device",
		EPCDesc: map[echonet_lite.EPCType]echonet_lite.PropertyDesc{
			0x80: { // Decoder implements Encoder -> StringSettable: true
				Name:    "EncoderImplemented",
				Decoder: MockDecoderEncoder{},
				Aliases: map[string][]byte{"alias1": {0x01}},
			},
			0x81: { // Decoder does NOT implement Encoder -> StringSettable: false
				Name:    "EncoderNotImplemented",
				Decoder: MockDecoderOnly{},
				Aliases: map[string][]byte{"alias2": {0x02}},
			},
			0x82: { // Decoder is nil -> StringSettable: false
				Name:    "DecoderNil",
				Aliases: map[string][]byte{"alias3": {0x03}},
			},
			0x83: { // NumberDesc implements Encoder -> StringSettable: true
				Name:    "NumberDescProperty",
				Decoder: echonet_lite.NumberDesc{Min: 0, Max: 100, Unit: "U", EDTLen: 1},
			},
			0x84: { // StringDesc implements Encoder -> StringSettable: true
				Name:    "StringDescProperty",
				Decoder: echonet_lite.StringDesc{MaxEDTLen: 5},
			},
		},
	}

	// 2. Create the target map
	targetMap := make(map[string]protocol.EPCDesc)

	// 3. Call the function under test
	populateEPCDescriptions(mockTable, targetMap)

	// 4. Assert the results for each EPC
	// Check EncoderImplemented (0x80)
	desc80, ok80 := targetMap["80"]
	assert.True(t, ok80, "EPC 0x80 should exist")
	assert.Equal(t, "EncoderImplemented", desc80.Description)
	assert.True(t, desc80.StringSettable, "EPC 0x80 (MockDecoderEncoder) should be StringSettable")
	assert.NotNil(t, desc80.Aliases, "EPC 0x80 should have aliases")
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte{0x01}), desc80.Aliases["alias1"])
	assert.Nil(t, desc80.NumberDesc, "EPC 0x80 should not have NumberDesc")
	assert.Nil(t, desc80.StringDesc, "EPC 0x80 should not have StringDesc")

	// Check EncoderNotImplemented (0x81)
	desc81, ok81 := targetMap["81"]
	assert.True(t, ok81, "EPC 0x81 should exist")
	assert.Equal(t, "EncoderNotImplemented", desc81.Description)
	assert.False(t, desc81.StringSettable, "EPC 0x81 (MockDecoderOnly) should NOT be StringSettable")
	assert.NotNil(t, desc81.Aliases, "EPC 0x81 should have aliases")
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte{0x02}), desc81.Aliases["alias2"])
	assert.Nil(t, desc81.NumberDesc, "EPC 0x81 should not have NumberDesc")
	assert.Nil(t, desc81.StringDesc, "EPC 0x81 should not have StringDesc")

	// Check DecoderNil (0x82)
	desc82, ok82 := targetMap["82"]
	assert.True(t, ok82, "EPC 0x82 should exist")
	assert.Equal(t, "DecoderNil", desc82.Description)
	assert.False(t, desc82.StringSettable, "EPC 0x82 (Decoder nil) should NOT be StringSettable")
	assert.NotNil(t, desc82.Aliases, "EPC 0x82 should have aliases")
	assert.Equal(t, base64.StdEncoding.EncodeToString([]byte{0x03}), desc82.Aliases["alias3"])
	assert.Nil(t, desc82.NumberDesc, "EPC 0x82 should not have NumberDesc")
	assert.Nil(t, desc82.StringDesc, "EPC 0x82 should not have StringDesc")

	// Check NumberDescProperty (0x83)
	desc83, ok83 := targetMap["83"]
	assert.True(t, ok83, "EPC 0x83 should exist")
	assert.Equal(t, "NumberDescProperty", desc83.Description)
	assert.True(t, desc83.StringSettable, "EPC 0x83 (NumberDesc) should be StringSettable")
	assert.Nil(t, desc83.Aliases, "EPC 0x83 should not have aliases")
	assert.NotNil(t, desc83.NumberDesc, "EPC 0x83 should have NumberDesc")
	assert.Equal(t, "U", desc83.NumberDesc.Unit)
	assert.Equal(t, 0, desc83.NumberDesc.EdtLen) // EdtLen 1 becomes 0 due to omitempty logic
	assert.Nil(t, desc83.StringDesc, "EPC 0x83 should not have StringDesc")

	// Check StringDescProperty (0x84)
	desc84, ok84 := targetMap["84"]
	assert.True(t, ok84, "EPC 0x84 should exist")
	assert.Equal(t, "StringDescProperty", desc84.Description)
	assert.True(t, desc84.StringSettable, "EPC 0x84 (StringDesc) should be StringSettable")
	assert.Nil(t, desc84.Aliases, "EPC 0x84 should not have aliases")
	assert.Nil(t, desc84.NumberDesc, "EPC 0x84 should not have NumberDesc")
	assert.NotNil(t, desc84.StringDesc, "EPC 0x84 should have StringDesc")
	assert.Equal(t, 5, desc84.StringDesc.MaxEDTLen)
}

// TestHandleGetPropertyDescriptionFromClient は handleGetPropertyDescriptionFromClient メソッドのテスト
func TestHandleGetPropertyDescriptionFromClient(t *testing.T) {
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
			name:       "Empty class code (requests common properties)",
			classCode:  "",
			wantStatus: true, // 修正: 空のclassCodeは有効なリクエスト
			wantError:  nil,  // 修正: エラーは発生しない
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
			mockClient := &mockECHONETListClient{}

			// WebSocketServer の作成
			ws := &WebSocketServer{
				ctx:           context.Background(),
				transport:     nil,
				echonetClient: mockClient,
				handler:       nil, // テストでは使用しないのでnilでOK
			}

			// テスト用のメッセージを作成
			payload := protocol.GetPropertyDescriptionPayload{
				ClassCode: tt.classCode,
			}
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("Failed to marshal payload: %v", err)
			}

			msg := &protocol.Message{
				Type:      protocol.MessageTypeGetPropertyDescription,
				Payload:   payloadBytes,
				RequestID: "test-request-id",
			}

			responsePayload := ws.handleGetPropertyDescriptionFromClient(msg)

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
					// 成功時のDataフィールドは PropertyDescriptionData の JSON 文字列
					var descriptionData protocol.PropertyDescriptionData
					if err := json.Unmarshal(responsePayload.Data, &descriptionData); err != nil {
						t.Fatalf("Failed to unmarshal response data payload: %v", err)
					}

					// クラスコードを確認
					if descriptionData.ClassCode != tt.classCode {
						t.Errorf("Response data.classCode = %v, want %v", descriptionData.ClassCode, tt.classCode)
					}

					// プロパティマップが存在することを確認
					if descriptionData.Properties == nil {
						t.Errorf("Response data.properties is nil, want non-nil")
					}

					switch tt.classCode {
					case "0130": // HomeAirConditionerの場合
						// "B0" (Operation mode setting) - エイリアスのみ
						epcDescB0, okB0 := descriptionData.Properties["B0"]
						if !okB0 {
							t.Errorf("Response data.properties does not contain 'B0' EPC for HomeAirConditioner")
						} else {
							if epcDescB0.Description != "Operation mode setting" {
								t.Errorf("Response data.properties['B0'].description = %v, want %v", epcDescB0.Description, "Operation mode setting")
							}
							if epcDescB0.Aliases == nil {
								t.Errorf("Response data.properties['B0'].aliases is nil, want non-nil")
							} else if _, ok := epcDescB0.Aliases["auto"]; !ok {
								t.Errorf("Response data.properties['B0'].aliases does not contain 'auto' alias")
							}
							if epcDescB0.NumberDesc != nil {
								t.Errorf("Response data.properties['B0'].numberDesc is not nil, want nil")
							}
							if epcDescB0.StringDesc != nil {
								t.Errorf("Response data.properties['B0'].stringDesc is not nil, want nil")
							}
						}
						// "B3" (Set temperature value) - NumberDescのみ
						epcDescB3, okB3 := descriptionData.Properties["B3"]
						if !okB3 {
							t.Errorf("Response data.properties does not contain 'B3' EPC for HomeAirConditioner")
						} else {
							if epcDescB3.Description != "Temperature setting" { // Fix: Correct description
								t.Errorf("Response data.properties['B3'].description = %v, want %v", epcDescB3.Description, "Temperature setting")
							}
							if epcDescB3.Aliases == nil { // Fix: Aliases should exist (ExtraValueAlias)
								t.Errorf("Response data.properties['B3'].aliases is nil, want non-nil")
							} else if _, ok := epcDescB3.Aliases["unknown"]; !ok { // Check one specific alias
								t.Errorf("Response data.properties['B3'].aliases does not contain 'unknown' alias")
							}
							if epcDescB3.NumberDesc == nil {
								t.Errorf("Response data.properties['B3'].numberDesc is nil, want non-nil")
							} else if epcDescB3.NumberDesc.Unit != "℃" { // Fix: Correct unit
								t.Errorf("Response data.properties['B3'].numberDesc.Unit = %v, want %v", epcDescB3.NumberDesc.Unit, "℃")
							}
							if epcDescB3.StringDesc != nil {
								t.Errorf("Response data.properties['B3'].stringDesc is not nil, want nil")
							}
						}
					case "": // 共通プロパティの場合
						// "80" (Operation status) - エイリアスのみ
						epcDesc80, ok80 := descriptionData.Properties["80"]
						if !ok80 {
							t.Errorf("Response data.properties does not contain '80' EPC for common properties")
						} else {
							if epcDesc80.Description != "Operation status" {
								t.Errorf("Response data.properties['80'].description = %v, want %v", epcDesc80.Description, "Operation status")
							}
							if epcDesc80.Aliases == nil {
								t.Errorf("Response data.properties['80'].aliases is nil, want non-nil")
							} else if _, ok := epcDesc80.Aliases["on"]; !ok {
								t.Errorf("Response data.properties['80'].aliases does not contain 'on' alias")
							}
							if epcDesc80.NumberDesc != nil {
								t.Errorf("Response data.properties['80'].numberDesc is not nil, want nil")
							}
							if epcDesc80.StringDesc != nil {
								t.Errorf("Response data.properties['80'].stringDesc is not nil, want nil")
							}
						}
						// "8C" (Product code) - StringDescのみ
						epcDesc8C, ok8C := descriptionData.Properties["8C"]
						if !ok8C {
							t.Errorf("Response data.properties does not contain '8C' EPC for common properties")
						} else {
							if epcDesc8C.Description != "Product code" {
								t.Errorf("Response data.properties['8C'].description = %v, want %v", epcDesc8C.Description, "Product code")
							}
							if epcDesc8C.Aliases != nil {
								t.Errorf("Response data.properties['8C'].aliases is not nil, want nil")
							}
							if epcDesc8C.NumberDesc != nil {
								t.Errorf("Response data.properties['8C'].numberDesc is not nil, want nil")
							}
							if epcDesc8C.StringDesc == nil {
								t.Errorf("Response data.properties['8C'].stringDesc is nil, want non-nil")
							} else if epcDesc8C.StringDesc.MinEDTLen != 12 {
								t.Errorf("Response data.properties['8C'].stringDesc.MinEDTLen = %v, want %v", epcDesc8C.StringDesc.MinEDTLen, 12)
							}
						}
					}
				}
			}
		})
	}
}
