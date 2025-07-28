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

func TestCalculateRequestDelay(t *testing.T) {
	baseDelay := 50 * time.Millisecond

	tests := []struct {
		name         string
		requestIndex int
		wantMinDelay time.Duration
		wantMaxDelay time.Duration
	}{
		{
			name:         "最初のリクエストは遅延なし",
			requestIndex: 1,
			wantMinDelay: 0,
			wantMaxDelay: 0,
		},
		{
			name:         "2番目のリクエストは基準遅延×1",
			requestIndex: 2,
			wantMinDelay: time.Duration(float64(baseDelay) * MinIntervalRatio),         // 最小値は基準値の50%
			wantMaxDelay: time.Duration(float64(baseDelay) * (1.0 + JitterPercentage)), // 最大値は+30%
		},
		{
			name:         "3番目のリクエストは基準遅延×2",
			requestIndex: 3,
			wantMinDelay: time.Duration(float64(baseDelay*2) * (1.0 - JitterPercentage)), // -30%
			wantMaxDelay: time.Duration(float64(baseDelay*2) * (1.0 + JitterPercentage)), // +30%
		},
		{
			name:         "最大倍率を超える場合",
			requestIndex: MaxDelayMultiplier + 2, // 7番目
			wantMinDelay: time.Duration(float64(baseDelay*MaxDelayMultiplier) * (1.0 - JitterPercentage)),
			wantMaxDelay: time.Duration(float64(baseDelay*MaxDelayMultiplier) * (1.0 + JitterPercentage)),
		},
		{
			name:         "requestIndex が 0 の場合",
			requestIndex: 0,
			wantMinDelay: 0,
			wantMaxDelay: 0,
		},
		{
			name:         "requestIndex が負の場合",
			requestIndex: -1,
			wantMinDelay: 0,
			wantMaxDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 複数回実行してジッタの範囲を確認
			iterations := 100
			for i := 0; i < iterations; i++ {
				delay := calculateRequestDelay(tt.requestIndex, baseDelay)

				if delay < tt.wantMinDelay {
					t.Errorf("calculateRequestDelay() = %v, want minimum %v", delay, tt.wantMinDelay)
				}
				if delay > tt.wantMaxDelay {
					t.Errorf("calculateRequestDelay() = %v, want maximum %v", delay, tt.wantMaxDelay)
				}

				// 最初のリクエストまたは無効なインデックスの場合は常に0
				if tt.requestIndex <= 1 && delay != 0 {
					t.Errorf("calculateRequestDelay() = %v for requestIndex %d, want 0", delay, tt.requestIndex)
				}
			}
		})
	}
}

func TestCalculateRequestDelayDistribution(t *testing.T) {
	// ジッタの分布が適切かどうかを確認するテスト
	baseDelay := 100 * time.Millisecond
	requestIndex := 2

	iterations := 1000
	var totalDelay time.Duration
	var minDelay = time.Hour // 大きな初期値
	var maxDelay time.Duration

	for i := 0; i < iterations; i++ {
		delay := calculateRequestDelay(requestIndex, baseDelay)
		totalDelay += delay

		if delay < minDelay {
			minDelay = delay
		}
		if delay > maxDelay {
			maxDelay = delay
		}
	}

	// 平均値が基準値に近いことを確認
	avgDelay := totalDelay / time.Duration(iterations)
	expectedAvg := baseDelay * time.Duration(requestIndex-1)
	tolerance := time.Duration(float64(expectedAvg) * 0.05) // 5%の許容範囲

	if avgDelay < expectedAvg-tolerance || avgDelay > expectedAvg+tolerance {
		t.Errorf("Average delay = %v, want approximately %v (±%v)", avgDelay, expectedAvg, tolerance)
	}

	// 最小値と最大値が期待範囲内であることを確認
	expectedMin := time.Duration(float64(baseDelay) * MinIntervalRatio)
	expectedMax := time.Duration(float64(baseDelay) * (1.0 + JitterPercentage))

	if minDelay < expectedMin*9/10 { // 若干の余裕を持たせる
		t.Errorf("Minimum delay too low: %v, expected at least %v", minDelay, expectedMin)
	}
	if maxDelay > expectedMax*11/10 { // 若干の余裕を持たせる
		t.Errorf("Maximum delay too high: %v, expected at most %v", maxDelay, expectedMax)
	}

	t.Logf("Delay distribution: min=%v, avg=%v, max=%v", minDelay, avgDelay, maxDelay)
}
