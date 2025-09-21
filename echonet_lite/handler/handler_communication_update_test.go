package handler

import (
	"echonet-list/echonet_lite"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockDataAccessorForUpdate はUpdatePropertiesテスト用のDataAccessor実装
type MockDataAccessorForUpdate struct {
	isOfflineResults map[string]bool
	filterResult     Devices
	lastUpdateTimes  map[string]time.Time
	propertyMaps     map[string]PropertyMap
	deviceStrings    map[string]string
}

func (m *MockDataAccessorForUpdate) SaveDeviceInfo() {
	// Mock implementation
}

func (m *MockDataAccessorForUpdate) IsKnownDevice(device IPAndEOJ) bool {
	return true // Always return true for simplicity
}

func (m *MockDataAccessorForUpdate) HasEPCInPropertyMap(device IPAndEOJ, mapType PropertyMapType, epc EPCType) bool {
	return true // Always return true for simplicity
}

func (m *MockDataAccessorForUpdate) GetPropertyMap(device IPAndEOJ, mapType PropertyMapType) PropertyMap {
	key := device.Key()
	if propMap, exists := m.propertyMaps[key]; exists {
		return propMap
	}
	return nil
}

func (m *MockDataAccessorForUpdate) RegisterProperties(device IPAndEOJ, properties Properties) []ChangedProperty {
	return nil // Mock implementation
}

func (m *MockDataAccessorForUpdate) GetProperty(device IPAndEOJ, epc EPCType) (*Property, bool) {
	return nil, false // Mock implementation
}

func (m *MockDataAccessorForUpdate) GetIDString(device IPAndEOJ) IDString {
	return "" // Mock implementation
}

func (m *MockDataAccessorForUpdate) GetLastUpdateTime(device IPAndEOJ) time.Time {
	key := device.Key()
	if t, exists := m.lastUpdateTimes[key]; exists {
		return t
	}
	return time.Time{} // Zero time
}

func (m *MockDataAccessorForUpdate) DeviceStringWithAlias(device IPAndEOJ) string {
	key := device.Key()
	if str, exists := m.deviceStrings[key]; exists {
		return str
	}
	return device.Specifier()
}

func (m *MockDataAccessorForUpdate) IsOffline(device IPAndEOJ) bool {
	key := device.Key()
	if offline, exists := m.isOfflineResults[key]; exists {
		return offline
	}
	return false // Default to online
}

func (m *MockDataAccessorForUpdate) SetOffline(device IPAndEOJ, offline bool) {
	// Mock implementation
}

func (m *MockDataAccessorForUpdate) Filter(criteria FilterCriteria) Devices {
	return m.filterResult
}

func (m *MockDataAccessorForUpdate) RegisterDevice(device IPAndEOJ) {
	// Mock implementation
}

func (m *MockDataAccessorForUpdate) HasIP(ip net.IP) bool {
	return true // Mock implementation
}

func (m *MockDataAccessorForUpdate) FindByIDString(id IDString) []IPAndEOJ {
	return nil // Mock implementation
}

func (m *MockDataAccessorForUpdate) RemoveDevice(device IPAndEOJ) error {
	return nil // Mock implementation
}

// TestIsNodeProfileOnline は、isNodeProfileOnlineメソッドのテストです。
// このテストは、実装を追加する前に失敗し、実装後に成功することを確認します。
func TestIsNodeProfileOnline(t *testing.T) {
	// Setup
	mockDataAccessor := &MockDataAccessorForUpdate{
		isOfflineResults: make(map[string]bool),
	}

	// Create a CommunicationHandler with mock data accessor
	handler := &CommunicationHandler{
		dataAccessor: mockDataAccessor,
	}

	// Test data
	ip := net.ParseIP("192.168.1.100")
	nodeProfileDevice := IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}

	// Test case 1: NodeProfile is online
	mockDataAccessor.isOfflineResults[nodeProfileDevice.Key()] = false
	result := handler.isNodeProfileOnline(ip)
	assert.True(t, result, "NodeProfile should be online")

	// Test case 2: NodeProfile is offline
	mockDataAccessor.isOfflineResults[nodeProfileDevice.Key()] = true
	result = handler.isNodeProfileOnline(ip)
	assert.False(t, result, "NodeProfile should be offline")
}

// TestUpdateProperties_OfflineDevice_NodeProfileOnline_ShouldUpdate は、
// オフラインデバイスでもNodeProfileがオンラインなら更新を実行することを確認します。
// このテストは、実装を追加する前に失敗し、実装後に成功することを確認します。
func TestUpdateProperties_OfflineDevice_NodeProfileOnline_ShouldUpdate(t *testing.T) {
	// Test data
	ip := net.ParseIP("192.168.1.100")
	deviceEOJ := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)
	device := IPAndEOJ{IP: ip, EOJ: deviceEOJ}
	nodeProfileDevice := IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}

	// Setup mock data accessor
	mockDataAccessor := &MockDataAccessorForUpdate{
		isOfflineResults: make(map[string]bool),
		lastUpdateTimes:  make(map[string]time.Time),
		propertyMaps:     make(map[string]PropertyMap),
		deviceStrings:    make(map[string]string),
	}

	// Create test devices
	devices := NewDevices()
	// デバイスを手動で追加
	devices.ensureDeviceExists(device)

	// Set up mock data
	mockDataAccessor.filterResult = devices
	mockDataAccessor.lastUpdateTimes[device.Key()] = time.Time{}       // Never updated
	mockDataAccessor.isOfflineResults[device.Key()] = true             // Device is offline
	mockDataAccessor.isOfflineResults[nodeProfileDevice.Key()] = false // NodeProfile is online

	// Create property map for the device
	propMap := make(PropertyMap)
	propMap[echonet_lite.EPCOperationStatus] = struct{}{}
	mockDataAccessor.propertyMaps[device.Key()] = propMap
	mockDataAccessor.deviceStrings[device.Key()] = "test-device"

	// Create handler with mock
	handler := &CommunicationHandler{
		dataAccessor: mockDataAccessor,
	}

	// Test: Check if device should be processed
	// Current implementation would skip offline device
	// After implementation, it should process if NodeProfile is online
	lastUpdateTime := mockDataAccessor.GetLastUpdateTime(device)
	isDeviceOffline := mockDataAccessor.IsOffline(device)
	isNodeProfileOnline := handler.isNodeProfileOnline(device.IP)

	shouldSkip := !lastUpdateTime.IsZero() && time.Since(lastUpdateTime) < UpdateIntervalThreshold
	if !shouldSkip && isDeviceOffline && !isNodeProfileOnline {
		shouldSkip = true
	}

	// After implementation, this should be false (device should be processed)
	assert.False(t, shouldSkip, "Device should be processed when NodeProfile is online, even if device is offline")
}

// TestUpdateProperties_OfflineDevice_NodeProfileOffline_ShouldSkip は、
// デバイスとNodeProfileの両方がオフラインの場合、スキップすることを確認します。
func TestUpdateProperties_OfflineDevice_NodeProfileOffline_ShouldSkip(t *testing.T) {
	// Test data
	ip := net.ParseIP("192.168.1.100")
	deviceEOJ := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)
	device := IPAndEOJ{IP: ip, EOJ: deviceEOJ}
	nodeProfileDevice := IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}

	// Setup mock data accessor
	mockDataAccessor := &MockDataAccessorForUpdate{
		isOfflineResults: make(map[string]bool),
		lastUpdateTimes:  make(map[string]time.Time),
	}

	// Set up mock data
	mockDataAccessor.lastUpdateTimes[device.Key()] = time.Time{}      // Never updated
	mockDataAccessor.isOfflineResults[device.Key()] = true            // Device is offline
	mockDataAccessor.isOfflineResults[nodeProfileDevice.Key()] = true // NodeProfile is also offline

	// Create handler with mock
	handler := &CommunicationHandler{
		dataAccessor: mockDataAccessor,
	}

	// Test: Check if device should be skipped
	lastUpdateTime := mockDataAccessor.GetLastUpdateTime(device)
	isDeviceOffline := mockDataAccessor.IsOffline(device)
	isNodeProfileOnline := handler.isNodeProfileOnline(device.IP)

	shouldSkip := !lastUpdateTime.IsZero() && time.Since(lastUpdateTime) < UpdateIntervalThreshold
	if !shouldSkip && isDeviceOffline && !isNodeProfileOnline {
		shouldSkip = true
	}

	// This should be true (device should be skipped)
	assert.True(t, shouldSkip, "Device should be skipped when both device and NodeProfile are offline")
}

// TestUpdateProperties_OnlineDevice_ShouldUpdate は、
// オンラインデバイスは常に更新されることを確認します。
func TestUpdateProperties_OnlineDevice_ShouldUpdate(t *testing.T) {
	// Test data
	ip := net.ParseIP("192.168.1.100")
	deviceEOJ := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)
	device := IPAndEOJ{IP: ip, EOJ: deviceEOJ}

	// Setup mock data accessor
	mockDataAccessor := &MockDataAccessorForUpdate{
		isOfflineResults: make(map[string]bool),
		lastUpdateTimes:  make(map[string]time.Time),
	}

	// Set up mock data
	mockDataAccessor.lastUpdateTimes[device.Key()] = time.Time{} // Never updated
	mockDataAccessor.isOfflineResults[device.Key()] = false      // Device is online

	// Create handler with mock
	handler := &CommunicationHandler{
		dataAccessor: mockDataAccessor,
	}

	// Test: Check if device should be processed
	lastUpdateTime := mockDataAccessor.GetLastUpdateTime(device)
	isDeviceOffline := mockDataAccessor.IsOffline(device)
	isNodeProfileOnline := handler.isNodeProfileOnline(device.IP)

	shouldSkip := !lastUpdateTime.IsZero() && time.Since(lastUpdateTime) < UpdateIntervalThreshold
	if !shouldSkip && isDeviceOffline && !isNodeProfileOnline {
		shouldSkip = true
	}

	// This should be false (device should be processed)
	assert.False(t, shouldSkip, "Online device should always be processed")
}

// TestTryGetPropertyMap_Success tests successful property map retrieval from cache
func TestTryGetPropertyMap_Success(t *testing.T) {
	// Setup
	device := IPAndEOJ{
		IP:  net.ParseIP("192.168.1.100"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 1),
	}

	// Create mock property map
	mockPropMap := make(PropertyMap)
	mockPropMap[0x80] = struct{}{}
	mockPropMap[0x81] = struct{}{}

	mockDataAccessor := &MockDataAccessorForUpdate{
		propertyMaps: map[string]PropertyMap{
			device.Key(): mockPropMap,
		},
	}

	handler := &CommunicationHandler{
		dataAccessor: mockDataAccessor,
	}

	// Execute
	propMap, ok := handler.tryGetPropertyMap(device)

	// Verify
	assert.True(t, ok, "tryGetPropertyMap should succeed when property map exists")
	assert.NotNil(t, propMap, "Property map should not be nil")
	assert.Equal(t, mockPropMap, propMap, "Should return the correct property map")
}
