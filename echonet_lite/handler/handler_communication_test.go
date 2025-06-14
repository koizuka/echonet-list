package handler

import (
	"echonet-list/echonet_lite"
	"net"
	"sync"
	"testing"
	"time"
)

// MockSession はテスト用のSession実装
type MockSession struct {
	broadcastCalls []BroadcastCall
	mu             sync.Mutex
}

type BroadcastCall struct {
	SEOJ       EOJ
	ESV        echonet_lite.ESVType
	Properties Properties
}

func (m *MockSession) Broadcast(seoj EOJ, esv echonet_lite.ESVType, properties Properties) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcastCalls = append(m.broadcastCalls, BroadcastCall{
		SEOJ:       seoj,
		ESV:        esv,
		Properties: properties,
	})
	return nil
}

func (m *MockSession) GetBroadcastCalls() []BroadcastCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]BroadcastCall, len(m.broadcastCalls))
	copy(calls, m.broadcastCalls)
	return calls
}

func (m *MockSession) SendResponse(ip net.IP, originalMsg *echonet_lite.ECHONETLiteMessage, esv echonet_lite.ESVType, setProperties Properties, getProperties Properties) error {
	return nil
}

func (m *MockSession) IsLocalIP(ip net.IP) bool {
	return false
}

func (m *MockSession) MainLoop() {}

func (m *MockSession) SetTimeoutChannel(ch chan SessionTimeoutEvent) {}

func (m *MockSession) OnInf(handler func(net.IP, *echonet_lite.ECHONETLiteMessage) error) {}

func (m *MockSession) OnReceive(handler func(net.IP, *echonet_lite.ECHONETLiteMessage) error) {}

// MockDataAccessor はテスト用のDataAccessor実装
type MockDataAccessor struct{}

func (m *MockDataAccessor) SaveDeviceInfo()                    {}
func (m *MockDataAccessor) IsKnownDevice(device IPAndEOJ) bool { return false }
func (m *MockDataAccessor) HasEPCInPropertyMap(device IPAndEOJ, mapType PropertyMapType, epc EPCType) bool {
	return false
}
func (m *MockDataAccessor) GetPropertyMap(device IPAndEOJ, mapType PropertyMapType) PropertyMap {
	return nil
}
func (m *MockDataAccessor) RegisterProperties(device IPAndEOJ, properties Properties) []ChangedProperty {
	return nil
}
func (m *MockDataAccessor) GetProperty(device IPAndEOJ, epc EPCType) (*Property, bool) {
	return nil, false
}
func (m *MockDataAccessor) GetIDString(device IPAndEOJ) IDString         { return "" }
func (m *MockDataAccessor) GetLastUpdateTime(device IPAndEOJ) time.Time  { return time.Time{} }
func (m *MockDataAccessor) DeviceStringWithAlias(device IPAndEOJ) string { return "" }
func (m *MockDataAccessor) IsOffline(device IPAndEOJ) bool               { return false }
func (m *MockDataAccessor) SetOffline(device IPAndEOJ, offline bool)     {}
func (m *MockDataAccessor) Filter(criteria FilterCriteria) Devices       { return NewDevices() }
func (m *MockDataAccessor) RegisterDevice(device IPAndEOJ)               {}
func (m *MockDataAccessor) HasIP(ip net.IP) bool                         { return false }
func (m *MockDataAccessor) FindByIDString(id IDString) []IPAndEOJ        { return nil }

// MockNotificationRelay はテスト用のNotificationRelay実装
type MockNotificationRelay struct{}

func (m *MockNotificationRelay) RelayDeviceEvent(event DeviceEvent)                          {}
func (m *MockNotificationRelay) RelaySessionTimeoutEvent(event SessionTimeoutEvent)          {}
func (m *MockNotificationRelay) RelayPropertyChangeEvent(device IPAndEOJ, property Property) {}

func (m *MockNotificationRelay) SendPropertyChangeNotification(PropertyChangeNotification) {}

// TestDeviceProperties_SetPropertiesWithAnnouncement はアナウンス対象プロパティの変更通知テストの準備です
func TestDeviceProperties_SetPropertiesWithAnnouncement(t *testing.T) {
	// テスト用のlocalDevicesを作成
	localDevices := make(DeviceProperties)
	controllerEOJ := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	// Status Announcement Property Mapを設定（設置場所を含む）
	announcementMap := make(PropertyMap)
	announcementMap.Set(echonet_lite.EPCInstallationLocation) // 0x81

	err := localDevices.Set(controllerEOJ,
		Property{EPC: echonet_lite.EPCOperationStatus, EDT: []byte{0x30}},
		Property{EPC: echonet_lite.EPCInstallationLocation, EDT: []byte{0x00}}, // 初期値：未設定
		Property{EPC: echonet_lite.EPCStatusAnnouncementPropertyMap, EDT: announcementMap.Encode()},
	)
	if err != nil {
		t.Fatalf("Failed to set initial local device properties: %v", err)
	}

	// 設置場所を「台所」(0x08)に変更
	properties := Properties{
		{EPC: echonet_lite.EPCInstallationLocation, EDT: []byte{0x08}},
	}

	// SetPropertiesを実行
	resultProps, success := localDevices.SetProperties(controllerEOJ, properties)
	if !success {
		t.Fatalf("SetProperties should succeed for installation location, but it failed")
	}

	// 結果プロパティのEDTが空であることを確認（成功を示す）
	if len(resultProps) != 1 || len(resultProps[0].EDT) != 0 {
		t.Errorf("Expected result property to have empty EDT on success, got %v", resultProps[0].EDT)
	}

	// 変更された値を確認
	updatedProp, ok := localDevices.Get(controllerEOJ, echonet_lite.EPCInstallationLocation)
	if !ok {
		t.Fatalf("Expected installation location property to exist after update, but it doesn't")
	}
	if len(updatedProp.EDT) != 1 || updatedProp.EDT[0] != 0x08 {
		t.Errorf("Expected installation location updated value to be [0x08], got %v", updatedProp.EDT)
	}

	// アナウンス対象かどうかを確認
	if !localDevices.IsAnnouncementTarget(controllerEOJ, echonet_lite.EPCInstallationLocation) {
		t.Errorf("Installation location should be announcement target, but IsAnnouncementTarget returned false")
	}

	// 通知機能は実装時にCommunicationHandlerに追加予定
	t.Log("プロパティ変更通知機能のテスト準備完了 - 実装時にbroadcast処理を追加")
}
