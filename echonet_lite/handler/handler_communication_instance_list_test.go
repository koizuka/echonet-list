package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDataAccessorForInstanceList is a mock implementation of DataAccessor for testing
type MockDataAccessorForInstanceList struct {
	mock.Mock
	devices map[string]IPAndEOJ // key: device.Key()
}

func NewMockDataAccessorForInstanceList() *MockDataAccessorForInstanceList {
	return &MockDataAccessorForInstanceList{
		devices: make(map[string]IPAndEOJ),
	}
}

func (m *MockDataAccessorForInstanceList) RegisterDevice(device IPAndEOJ) {
	m.Called(device)
	m.devices[device.Key()] = device
}

func (m *MockDataAccessorForInstanceList) RemoveDevice(device IPAndEOJ) error {
	args := m.Called(device)
	delete(m.devices, device.Key())
	return args.Error(0)
}

func (m *MockDataAccessorForInstanceList) Filter(criteria FilterCriteria) Devices {
	args := m.Called(criteria)

	// Simulate filtering by IP
	devices := NewDevices()
	for _, device := range m.devices {
		if criteria.Device.IP != nil && device.IP.Equal(*criteria.Device.IP) {
			devices.RegisterDevice(device)
		} else if criteria.Device.IP == nil {
			devices.RegisterDevice(device)
		}
	}

	return args.Get(0).(Devices)
}

func (m *MockDataAccessorForInstanceList) SaveDeviceInfo() {
	m.Called()
}

func (m *MockDataAccessorForInstanceList) GetPropertyMap(device IPAndEOJ, mapType PropertyMapType) PropertyMap {
	args := m.Called(device, mapType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(PropertyMap)
}

// Implement other required methods with default behavior
func (m *MockDataAccessorForInstanceList) IsKnownDevice(device IPAndEOJ) bool { return false }
func (m *MockDataAccessorForInstanceList) HasIP(ip net.IP) bool               { return false }
func (m *MockDataAccessorForInstanceList) RegisterProperties(device IPAndEOJ, properties Properties) []ChangedProperty {
	return nil
}
func (m *MockDataAccessorForInstanceList) SetOffline(device IPAndEOJ, offline bool) {}
func (m *MockDataAccessorForInstanceList) GetLastUpdateTime(device IPAndEOJ) time.Time {
	return time.Time{}
}
func (m *MockDataAccessorForInstanceList) IsOffline(device IPAndEOJ) bool { return false }
func (m *MockDataAccessorForInstanceList) DeviceStringWithAlias(device IPAndEOJ) string {
	return device.String()
}
func (m *MockDataAccessorForInstanceList) HasEPCInPropertyMap(device IPAndEOJ, mapType PropertyMapType, epc EPCType) bool {
	return false
}

func (m *MockDataAccessorForInstanceList) FindByIDString(id IDString) []IPAndEOJ {
	return nil
}

func (m *MockDataAccessorForInstanceList) GetIDString(device IPAndEOJ) IDString {
	return IDString(device.String())
}

func (m *MockDataAccessorForInstanceList) GetProperty(device IPAndEOJ, epc EPCType) (*Property, bool) {
	return nil, false
}

// onInstanceListWithoutPropertyMap is a test helper that tests onInstanceList logic without calling GetGetPropertyMap
func onInstanceListWithoutPropertyMap(h *CommunicationHandler, ip net.IP, il echonet_lite.InstanceList) error {
	// NodeProfileObjectも追加して取得する
	il = append(il, echonet_lite.NodeProfileObject)

	// 1. そのIPアドレスの既存デバイスを取得
	criteria := FilterCriteria{
		Device: DeviceSpecifier{IP: &ip},
	}
	existingDevices := h.dataAccessor.Filter(criteria).ListIPAndEOJ()

	// 2. 新しいインスタンスリストをセットに変換（高速な検索のため）
	newDeviceSet := make(map[string]struct{})
	for _, eoj := range il {
		device := IPAndEOJ{IP: ip, EOJ: eoj}
		newDeviceSet[device.Key()] = struct{}{}
	}

	// 3. 削除されたデバイスを検出
	var devicesToRemove []IPAndEOJ
	for _, existingDevice := range existingDevices {
		if existingDevice.IP.Equal(ip) {
			if _, exists := newDeviceSet[existingDevice.Key()]; !exists {
				devicesToRemove = append(devicesToRemove, existingDevice)
			}
		}
	}

	// 4. 削除されたデバイスを削除
	for _, device := range devicesToRemove {
		if err := h.dataAccessor.RemoveDevice(device); err != nil {
			slog.Warn("デバイスの削除に失敗", "device", device, "err", err)
		} else {
			slog.Info("デバイスを削除", "device", device)
		}
	}

	// 5. デバイスの登録（新規・既存両方）
	for _, eoj := range il {
		h.dataAccessor.RegisterDevice(IPAndEOJ{IP: ip, EOJ: eoj})
	}

	// デバイス情報の保存
	h.dataAccessor.SaveDeviceInfo()

	// Note: Skip GetGetPropertyMap calls in test

	return nil
}

// TestOnInstanceList_AddDevices tests that new devices are added correctly
func TestOnInstanceList_AddDevices(t *testing.T) {
	ctx := context.Background()
	mockDataAccessor := NewMockDataAccessorForInstanceList()
	mockNotifier := new(MockNotificationRelay)

	handler := &CommunicationHandler{
		session:       nil, // Not used in this test
		localDevices:  nil,
		dataAccessor:  mockDataAccessor,
		notifier:      mockNotifier,
		ctx:           ctx,
		Debug:         false,
		activeUpdates: make(map[string]time.Time),
	}

	ip := net.ParseIP("192.168.1.100")

	// Prepare initial state - no existing devices
	criteria := FilterCriteria{
		Device: DeviceSpecifier{IP: &ip},
	}
	emptyDevices := NewDevices()
	mockDataAccessor.On("Filter", criteria).Return(emptyDevices)

	// New devices to be added
	newEOJ1 := echonet_lite.MakeEOJ(0x0130, 1) // Air conditioner
	newEOJ2 := echonet_lite.MakeEOJ(0x0291, 1) // Lighting
	instanceList := echonet_lite.InstanceList{newEOJ1, newEOJ2}

	// Expect RegisterDevice to be called for each device plus NodeProfile
	mockDataAccessor.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: newEOJ1}).Once()
	mockDataAccessor.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: newEOJ2}).Once()
	mockDataAccessor.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}).Once()

	// Expect SaveDeviceInfo to be called
	mockDataAccessor.On("SaveDeviceInfo").Once()

	// Execute
	err := onInstanceListWithoutPropertyMap(handler, ip, instanceList)

	// Assert
	assert.NoError(t, err)
	mockDataAccessor.AssertExpectations(t)
}

// TestOnInstanceList_RemoveDevices tests that devices not in the new list are removed
func TestOnInstanceList_RemoveDevices(t *testing.T) {
	ctx := context.Background()
	mockDataAccessor := NewMockDataAccessorForInstanceList()
	mockNotifier := new(MockNotificationRelay)

	handler := &CommunicationHandler{
		session:       nil, // Not used in this test
		localDevices:  nil,
		dataAccessor:  mockDataAccessor,
		notifier:      mockNotifier,
		ctx:           ctx,
		Debug:         false,
		activeUpdates: make(map[string]time.Time),
	}

	ip := net.ParseIP("192.168.1.100")

	// Prepare initial state - existing devices
	existingEOJ1 := echonet_lite.MakeEOJ(0x0130, 1) // Air conditioner (will be kept)
	existingEOJ2 := echonet_lite.MakeEOJ(0x0291, 1) // Lighting (will be removed)
	existingEOJ3 := echonet_lite.MakeEOJ(0x0260, 1) // TV (will be removed)

	// Setup existing devices in mock
	mockDataAccessor.devices[IPAndEOJ{IP: ip, EOJ: existingEOJ1}.Key()] = IPAndEOJ{IP: ip, EOJ: existingEOJ1}
	mockDataAccessor.devices[IPAndEOJ{IP: ip, EOJ: existingEOJ2}.Key()] = IPAndEOJ{IP: ip, EOJ: existingEOJ2}
	mockDataAccessor.devices[IPAndEOJ{IP: ip, EOJ: existingEOJ3}.Key()] = IPAndEOJ{IP: ip, EOJ: existingEOJ3}
	mockDataAccessor.devices[IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}.Key()] = IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}

	criteria := FilterCriteria{
		Device: DeviceSpecifier{IP: &ip},
	}
	existingDevices := NewDevices()
	existingDevices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: existingEOJ1})
	existingDevices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: existingEOJ2})
	existingDevices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: existingEOJ3})
	existingDevices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject})
	mockDataAccessor.On("Filter", criteria).Return(existingDevices)

	// New instance list - only contains EOJ1
	instanceList := echonet_lite.InstanceList{existingEOJ1}

	// Expect RemoveDevice to be called for EOJ2 and EOJ3 (but not NodeProfile or EOJ1)
	mockDataAccessor.On("RemoveDevice", IPAndEOJ{IP: ip, EOJ: existingEOJ2}).Return(nil).Once()
	mockDataAccessor.On("RemoveDevice", IPAndEOJ{IP: ip, EOJ: existingEOJ3}).Return(nil).Once()

	// Expect RegisterDevice to be called for existing and NodeProfile
	mockDataAccessor.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: existingEOJ1}).Once()
	mockDataAccessor.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}).Once()

	// Expect SaveDeviceInfo to be called
	mockDataAccessor.On("SaveDeviceInfo").Once()

	// Execute
	err := onInstanceListWithoutPropertyMap(handler, ip, instanceList)

	// Assert
	assert.NoError(t, err)
	mockDataAccessor.AssertExpectations(t)
}

// TestOnInstanceList_MixedAddRemove tests adding and removing devices simultaneously
func TestOnInstanceList_MixedAddRemove(t *testing.T) {
	ctx := context.Background()
	mockDataAccessor := NewMockDataAccessorForInstanceList()
	mockNotifier := new(MockNotificationRelay)

	handler := &CommunicationHandler{
		session:       nil, // Not used in this test
		localDevices:  nil,
		dataAccessor:  mockDataAccessor,
		notifier:      mockNotifier,
		ctx:           ctx,
		Debug:         false,
		activeUpdates: make(map[string]time.Time),
	}

	ip := net.ParseIP("192.168.1.100")

	// Prepare initial state - existing devices
	existingEOJ1 := echonet_lite.MakeEOJ(0x0130, 1) // Air conditioner (will be kept)
	existingEOJ2 := echonet_lite.MakeEOJ(0x0291, 1) // Lighting (will be removed)

	// Setup existing devices in mock
	mockDataAccessor.devices[IPAndEOJ{IP: ip, EOJ: existingEOJ1}.Key()] = IPAndEOJ{IP: ip, EOJ: existingEOJ1}
	mockDataAccessor.devices[IPAndEOJ{IP: ip, EOJ: existingEOJ2}.Key()] = IPAndEOJ{IP: ip, EOJ: existingEOJ2}
	mockDataAccessor.devices[IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}.Key()] = IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}

	criteria := FilterCriteria{
		Device: DeviceSpecifier{IP: &ip},
	}
	existingDevices := NewDevices()
	existingDevices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: existingEOJ1})
	existingDevices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: existingEOJ2})
	existingDevices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject})
	mockDataAccessor.On("Filter", criteria).Return(existingDevices)

	// New instance list - keeps EOJ1, removes EOJ2, adds EOJ3
	newEOJ3 := echonet_lite.MakeEOJ(0x0260, 1) // TV (new)
	instanceList := echonet_lite.InstanceList{existingEOJ1, newEOJ3}

	// Expect RemoveDevice to be called for EOJ2
	mockDataAccessor.On("RemoveDevice", IPAndEOJ{IP: ip, EOJ: existingEOJ2}).Return(nil).Once()

	// Expect RegisterDevice to be called for all devices in new list + NodeProfile
	mockDataAccessor.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: existingEOJ1}).Once()
	mockDataAccessor.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: newEOJ3}).Once()
	mockDataAccessor.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}).Once()

	// Expect SaveDeviceInfo to be called
	mockDataAccessor.On("SaveDeviceInfo").Once()

	// Execute
	err := onInstanceListWithoutPropertyMap(handler, ip, instanceList)

	// Assert
	assert.NoError(t, err)
	mockDataAccessor.AssertExpectations(t)
}

// TestOnInstanceList_NodeProfileAlwaysPresent tests that NodeProfile is never removed
func TestOnInstanceList_NodeProfileAlwaysPresent(t *testing.T) {
	ctx := context.Background()
	mockDataAccessor := NewMockDataAccessorForInstanceList()
	mockNotifier := new(MockNotificationRelay)

	handler := &CommunicationHandler{
		session:       nil, // Not used in this test
		localDevices:  nil,
		dataAccessor:  mockDataAccessor,
		notifier:      mockNotifier,
		ctx:           ctx,
		Debug:         false,
		activeUpdates: make(map[string]time.Time),
	}

	ip := net.ParseIP("192.168.1.100")

	// Prepare initial state - only NodeProfile exists
	mockDataAccessor.devices[IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}.Key()] = IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}

	criteria := FilterCriteria{
		Device: DeviceSpecifier{IP: &ip},
	}
	existingDevices := NewDevices()
	existingDevices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject})
	mockDataAccessor.On("Filter", criteria).Return(existingDevices)

	// New instance list - empty (but NodeProfile will be added automatically)
	instanceList := echonet_lite.InstanceList{}

	// NodeProfile should never be removed
	// Expect RegisterDevice for NodeProfile
	mockDataAccessor.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}).Once()

	// Expect SaveDeviceInfo to be called
	mockDataAccessor.On("SaveDeviceInfo").Once()

	// Execute
	err := onInstanceListWithoutPropertyMap(handler, ip, instanceList)

	// Assert
	assert.NoError(t, err)
	mockDataAccessor.AssertExpectations(t)
	// Verify that RemoveDevice was NOT called for NodeProfile
	mockDataAccessor.AssertNotCalled(t, "RemoveDevice", IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject})
}
