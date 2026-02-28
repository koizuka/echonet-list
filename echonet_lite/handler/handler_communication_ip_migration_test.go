package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Devices layer tests ---

func TestFindIPsWithSameNodeProfileID_NoMatch(t *testing.T) {
	devices := NewDevices()
	ip1 := net.ParseIP("192.168.0.1")
	npo1 := IPAndEOJ{IP: ip1, EOJ: echonet_lite.NodeProfileObject}
	devices.RegisterProperty(npo1, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: []byte{0x01, 0x02, 0x03}}, time.Now())

	result := devices.FindIPsWithSameNodeProfileID([]byte{0xAA, 0xBB}, ip1.String())
	assert.Empty(t, result)
}

func TestFindIPsWithSameNodeProfileID_SingleMatch(t *testing.T) {
	devices := NewDevices()
	idEDT := []byte{0x01, 0x02, 0x03, 0x04}

	ip1 := net.ParseIP("192.168.0.1")
	ip2 := net.ParseIP("192.168.0.2")
	npo1 := IPAndEOJ{IP: ip1, EOJ: echonet_lite.NodeProfileObject}
	npo2 := IPAndEOJ{IP: ip2, EOJ: echonet_lite.NodeProfileObject}
	devices.RegisterProperty(npo1, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT}, time.Now())
	devices.RegisterProperty(npo2, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT}, time.Now())

	// Exclude ip2, should find ip1
	result := devices.FindIPsWithSameNodeProfileID(idEDT, ip2.String())
	assert.Equal(t, []string{ip1.String()}, result)
}

func TestFindIPsWithSameNodeProfileID_MultipleMatches(t *testing.T) {
	devices := NewDevices()
	idEDT := []byte{0x01, 0x02, 0x03, 0x04}

	ip1 := net.ParseIP("192.168.0.1")
	ip2 := net.ParseIP("192.168.0.2")
	ip3 := net.ParseIP("192.168.0.3")
	npo1 := IPAndEOJ{IP: ip1, EOJ: echonet_lite.NodeProfileObject}
	npo2 := IPAndEOJ{IP: ip2, EOJ: echonet_lite.NodeProfileObject}
	npo3 := IPAndEOJ{IP: ip3, EOJ: echonet_lite.NodeProfileObject}
	devices.RegisterProperty(npo1, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT}, time.Now())
	devices.RegisterProperty(npo2, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT}, time.Now())
	devices.RegisterProperty(npo3, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT}, time.Now())

	result := devices.FindIPsWithSameNodeProfileID(idEDT, ip3.String())
	assert.Len(t, result, 2)
	assert.Contains(t, result, ip1.String())
	assert.Contains(t, result, ip2.String())
}

func TestFindIPsWithSameNodeProfileID_ExcludesSelf(t *testing.T) {
	devices := NewDevices()
	idEDT := []byte{0x01, 0x02, 0x03, 0x04}

	ip1 := net.ParseIP("192.168.0.1")
	npo1 := IPAndEOJ{IP: ip1, EOJ: echonet_lite.NodeProfileObject}
	devices.RegisterProperty(npo1, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT}, time.Now())

	// Exclude self - should return nothing
	result := devices.FindIPsWithSameNodeProfileID(idEDT, ip1.String())
	assert.Empty(t, result)
}

func TestFindIPsWithSameNodeProfileID_EmptyEDT(t *testing.T) {
	devices := NewDevices()
	result := devices.FindIPsWithSameNodeProfileID([]byte{}, "192.168.0.1")
	assert.Nil(t, result)

	result = devices.FindIPsWithSameNodeProfileID(nil, "192.168.0.1")
	assert.Nil(t, result)
}

func TestFindIPsWithSameNodeProfileID_NoNodeProfile(t *testing.T) {
	devices := NewDevices()
	idEDT := []byte{0x01, 0x02, 0x03, 0x04}

	// Register a non-NodeProfile device
	ip1 := net.ParseIP("192.168.0.1")
	device := IPAndEOJ{IP: ip1, EOJ: echonet_lite.MakeEOJ(0x0130, 1)}
	devices.RegisterProperty(device, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT}, time.Now())

	result := devices.FindIPsWithSameNodeProfileID(idEDT, "192.168.0.2")
	assert.Empty(t, result)
}

func TestRemoveAllDevicesByIP_AllRemoved(t *testing.T) {
	devices := NewDevices()
	ch := make(chan DeviceEvent, 10)
	devices.SetEventChannel(ch)

	ip := net.ParseIP("192.168.0.1")
	eoj1 := echonet_lite.MakeEOJ(0x0130, 1)
	eoj2 := echonet_lite.MakeEOJ(0x0291, 1)
	npo := echonet_lite.NodeProfileObject

	devices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: eoj1})
	devices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: eoj2})
	devices.RegisterDevice(IPAndEOJ{IP: ip, EOJ: npo})
	// Drain add events
	for len(ch) > 0 {
		<-ch
	}

	removed := devices.RemoveAllDevicesByIP(ip)
	assert.Len(t, removed, 3)

	// Verify IP no longer exists
	assert.False(t, devices.HasIP(ip))

	// Verify removal events were sent
	eventCount := len(ch)
	assert.Equal(t, 3, eventCount)
	for i := 0; i < eventCount; i++ {
		event := <-ch
		assert.Equal(t, DeviceEventRemoved, event.Type)
	}
}

func TestRemoveAllDevicesByIP_NonExistentIP(t *testing.T) {
	devices := NewDevices()
	ip := net.ParseIP("192.168.0.99")
	removed := devices.RemoveAllDevicesByIP(ip)
	assert.Nil(t, removed)
}

func TestRemoveAllDevicesByIP_CleansUpTimestampsAndOffline(t *testing.T) {
	devices := NewDevices()
	ip := net.ParseIP("192.168.0.1")
	eoj := echonet_lite.MakeEOJ(0x0130, 1)
	device := IPAndEOJ{IP: ip, EOJ: eoj}

	devices.RegisterProperty(device, Property{EPC: 0x80, EDT: []byte{0x30}}, time.Now())
	devices.SetOffline(device, true)

	assert.True(t, devices.IsOffline(device))
	assert.False(t, devices.GetLastUpdateTime(device).IsZero())

	removed := devices.RemoveAllDevicesByIP(ip)
	assert.Len(t, removed, 1)

	// After removal, these should return defaults
	assert.False(t, devices.IsOffline(device))
	assert.True(t, devices.GetLastUpdateTime(device).IsZero())
}

// --- Migration logic tests ---

func newMigrationTestHandler() (*CommunicationHandler, *MockDataAccessorForInstanceList) {
	ctx := context.Background()
	mockDataAccessor := NewMockDataAccessorForInstanceList()
	mockNotifier := new(MockNotificationRelay)

	handler := &CommunicationHandler{
		session:       nil,
		localDevices:  nil,
		dataAccessor:  mockDataAccessor,
		notifier:      mockNotifier,
		ctx:           ctx,
		Debug:         false,
		activeUpdates: make(map[string]*activeUpdateEntry),
	}
	return handler, mockDataAccessor
}

func TestMigrateDevicesFromOldIP_BasicMigration(t *testing.T) {
	handler, mockDA := newMigrationTestHandler()

	oldIP := net.ParseIP("192.168.0.91")
	newIP := net.ParseIP("192.168.0.140")
	idEDT := []byte{0xFE, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}

	oldNodeProfile := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.NodeProfileObject}
	oldDevice1 := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.MakeEOJ(0x027B, 1)}

	// Setup: FindIPsWithSameNodeProfileID returns old IP
	mockDA.On("FindIPsWithSameNodeProfileID", idEDT, newIP.String()).Return([]string{oldIP.String()})

	// Old NodeProfile is offline -> should migrate
	mockDA.On("IsOffline", oldNodeProfile).Return(true)

	// RemoveAllDevicesByIP should be called
	mockDA.On("RemoveAllDevicesByIP", mock.MatchedBy(func(ip net.IP) bool {
		return ip.Equal(oldIP)
	})).Return([]IPAndEOJ{oldNodeProfile, oldDevice1})

	// SaveDeviceInfo should be called
	mockDA.On("SaveDeviceInfo").Once()

	// Execute
	newDevice := IPAndEOJ{IP: newIP, EOJ: echonet_lite.NodeProfileObject}
	handler.migrateDevicesFromOldIP(newDevice, idEDT)

	// Assert
	mockDA.AssertExpectations(t)
}

func TestMigrateDevicesFromOldIP_OldIPOnline_Skip(t *testing.T) {
	handler, mockDA := newMigrationTestHandler()

	oldIP := net.ParseIP("192.168.0.91")
	newIP := net.ParseIP("192.168.0.140")
	idEDT := []byte{0xFE, 0x01, 0x02, 0x03}

	oldNodeProfile := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.NodeProfileObject}

	// Setup: FindIPsWithSameNodeProfileID returns old IP
	mockDA.On("FindIPsWithSameNodeProfileID", idEDT, newIP.String()).Return([]string{oldIP.String()})

	// Old NodeProfile is online -> should NOT migrate
	mockDA.On("IsOffline", oldNodeProfile).Return(false)

	// SaveDeviceInfo should still be called
	mockDA.On("SaveDeviceInfo").Once()

	// Execute
	newDevice := IPAndEOJ{IP: newIP, EOJ: echonet_lite.NodeProfileObject}
	handler.migrateDevicesFromOldIP(newDevice, idEDT)

	// Assert - RemoveAllDevicesByIP should NOT have been called
	mockDA.AssertNotCalled(t, "RemoveAllDevicesByIP", mock.Anything)
	mockDA.AssertExpectations(t)
}

func TestMigrateDevicesFromOldIP_NoOldEntries(t *testing.T) {
	handler, mockDA := newMigrationTestHandler()

	newIP := net.ParseIP("192.168.0.140")
	idEDT := []byte{0xFE, 0x01, 0x02, 0x03}

	// Setup: FindIPsWithSameNodeProfileID returns nothing
	mockDA.On("FindIPsWithSameNodeProfileID", idEDT, newIP.String()).Return([]string(nil))

	// Execute
	newDevice := IPAndEOJ{IP: newIP, EOJ: echonet_lite.NodeProfileObject}
	handler.migrateDevicesFromOldIP(newDevice, idEDT)

	// Assert - no further calls should be made
	mockDA.AssertNotCalled(t, "IsOffline", mock.Anything)
	mockDA.AssertNotCalled(t, "RemoveAllDevicesByIP", mock.Anything)
	mockDA.AssertNotCalled(t, "SaveDeviceInfo")
	mockDA.AssertExpectations(t)
}

func TestMigrateDevicesFromOldIP_EmptyEDT(t *testing.T) {
	handler, mockDA := newMigrationTestHandler()

	newDevice := IPAndEOJ{IP: net.ParseIP("192.168.0.140"), EOJ: echonet_lite.NodeProfileObject}

	// Execute with empty EDT
	handler.migrateDevicesFromOldIP(newDevice, []byte{})

	// Assert - no calls should be made
	mockDA.AssertNotCalled(t, "FindIPsWithSameNodeProfileID", mock.Anything, mock.Anything)
	mockDA.AssertExpectations(t)
}

func TestMigrateDevicesFromOldIP_MultipleOldIPs(t *testing.T) {
	handler, mockDA := newMigrationTestHandler()

	oldIP1 := net.ParseIP("192.168.0.91")
	oldIP2 := net.ParseIP("192.168.0.100")
	newIP := net.ParseIP("192.168.0.140")
	idEDT := []byte{0xFE, 0x01, 0x02, 0x03}

	oldNPO1 := IPAndEOJ{IP: oldIP1, EOJ: echonet_lite.NodeProfileObject}
	oldNPO2 := IPAndEOJ{IP: oldIP2, EOJ: echonet_lite.NodeProfileObject}

	// Both old IPs found
	mockDA.On("FindIPsWithSameNodeProfileID", idEDT, newIP.String()).Return([]string{oldIP1.String(), oldIP2.String()})

	// Both are offline
	mockDA.On("IsOffline", oldNPO1).Return(true)
	mockDA.On("IsOffline", oldNPO2).Return(true)

	// Both should be removed
	mockDA.On("RemoveAllDevicesByIP", mock.MatchedBy(func(ip net.IP) bool {
		return ip.Equal(oldIP1)
	})).Return([]IPAndEOJ{oldNPO1})
	mockDA.On("RemoveAllDevicesByIP", mock.MatchedBy(func(ip net.IP) bool {
		return ip.Equal(oldIP2)
	})).Return([]IPAndEOJ{oldNPO2})

	mockDA.On("SaveDeviceInfo").Once()

	// Execute
	newDevice := IPAndEOJ{IP: newIP, EOJ: echonet_lite.NodeProfileObject}
	handler.migrateDevicesFromOldIP(newDevice, idEDT)

	mockDA.AssertExpectations(t)
}

func TestMigrateDevicesFromOldIP_MultipleDeviceInstances(t *testing.T) {
	handler, mockDA := newMigrationTestHandler()

	oldIP := net.ParseIP("192.168.0.91")
	newIP := net.ParseIP("192.168.0.140")
	idEDT := []byte{0xFE, 0x01, 0x02, 0x03}

	// 4 floor heating instances + NodeProfile
	oldNPO := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.NodeProfileObject}
	oldDev1 := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.MakeEOJ(0x027B, 1)}
	oldDev2 := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.MakeEOJ(0x027B, 2)}
	oldDev3 := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.MakeEOJ(0x027B, 3)}
	oldDev4 := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.MakeEOJ(0x027B, 4)}

	mockDA.On("FindIPsWithSameNodeProfileID", idEDT, newIP.String()).Return([]string{oldIP.String()})
	mockDA.On("IsOffline", oldNPO).Return(true)
	mockDA.On("RemoveAllDevicesByIP", mock.MatchedBy(func(ip net.IP) bool {
		return ip.Equal(oldIP)
	})).Return([]IPAndEOJ{oldNPO, oldDev1, oldDev2, oldDev3, oldDev4})
	mockDA.On("SaveDeviceInfo").Once()

	newDevice := IPAndEOJ{IP: newIP, EOJ: echonet_lite.NodeProfileObject}
	handler.migrateDevicesFromOldIP(newDevice, idEDT)

	mockDA.AssertExpectations(t)
}

func TestMigrateDevicesFromOldIP_Idempotent(t *testing.T) {
	handler, mockDA := newMigrationTestHandler()

	newIP := net.ParseIP("192.168.0.140")
	idEDT := []byte{0xFE, 0x01, 0x02, 0x03}

	// First call: old IP exists
	oldIP := net.ParseIP("192.168.0.91")
	oldNPO := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.NodeProfileObject}

	mockDA.On("FindIPsWithSameNodeProfileID", idEDT, newIP.String()).Return([]string{oldIP.String()}).Once()
	mockDA.On("IsOffline", oldNPO).Return(true).Once()
	mockDA.On("RemoveAllDevicesByIP", mock.MatchedBy(func(ip net.IP) bool {
		return ip.Equal(oldIP)
	})).Return([]IPAndEOJ{oldNPO}).Once()
	mockDA.On("SaveDeviceInfo").Once()

	newDevice := IPAndEOJ{IP: newIP, EOJ: echonet_lite.NodeProfileObject}
	handler.migrateDevicesFromOldIP(newDevice, idEDT)
	mockDA.AssertExpectations(t)

	// Second call: no old IP found anymore (already removed)
	mockDA.ExpectedCalls = nil
	mockDA.Calls = nil
	mockDA.On("FindIPsWithSameNodeProfileID", idEDT, newIP.String()).Return([]string(nil)).Once()

	handler.migrateDevicesFromOldIP(newDevice, idEDT)
	mockDA.AssertExpectations(t)
	mockDA.AssertNotCalled(t, "RemoveAllDevicesByIP", mock.Anything)
}

func TestProcessPropertyUpdateHooks_IDNumber_TriggersMigration(t *testing.T) {
	handler, mockDA := newMigrationTestHandler()

	newIP := net.ParseIP("192.168.0.140")
	idEDT := []byte{0xFE, 0x01, 0x02, 0x03}
	nodeProfile := IPAndEOJ{IP: newIP, EOJ: echonet_lite.NodeProfileObject}

	// No old IPs
	mockDA.On("FindIPsWithSameNodeProfileID", idEDT, newIP.String()).Return([]string(nil))

	properties := Properties{
		{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT},
	}

	err := handler.ProcessPropertyUpdateHooks(nodeProfile, properties)
	assert.NoError(t, err)
	mockDA.AssertExpectations(t)
}

func TestProcessPropertyUpdateHooks_InstanceListAndIDNumber_Independent(t *testing.T) {
	handler, mockDA := newMigrationTestHandler()

	ip := net.ParseIP("192.168.0.140")
	nodeProfile := IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}

	// Test: InstanceList property should trigger instance list processing, not migration
	eoj1 := echonet_lite.MakeEOJ(0x0130, 1)
	instanceList := echonet_lite.SelfNodeInstanceListS([]echonet_lite.EOJ{eoj1})
	instanceListProperty := *instanceList.Property()

	// Expect instance list processing calls
	mockDA.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: eoj1}).Once()
	mockDA.On("RegisterDevice", IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}).Once()
	mockDA.On("IsOffline", IPAndEOJ{IP: ip, EOJ: eoj1}).Return(false).Once()
	mockDA.On("IsOffline", IPAndEOJ{IP: ip, EOJ: echonet_lite.NodeProfileObject}).Return(false).Once()
	mockDA.On("SaveDeviceInfo").Once()

	properties := Properties{instanceListProperty}
	err := handler.ProcessPropertyUpdateHooks(nodeProfile, properties)
	assert.NoError(t, err)

	// Migration should NOT have been triggered
	mockDA.AssertNotCalled(t, "FindIPsWithSameNodeProfileID", mock.Anything, mock.Anything)
	mockDA.AssertExpectations(t)
}

// --- Devices layer integration test ---

func TestFindIPsAndRemoveAllByIP_Integration(t *testing.T) {
	devices := NewDevices()
	ch := make(chan DeviceEvent, 20)
	devices.SetEventChannel(ch)

	idEDT := []byte{0xFE, 0x01, 0x02, 0x03, 0x04}
	oldIP := net.ParseIP("192.168.0.91")
	newIP := net.ParseIP("192.168.0.140")

	// Register devices on old IP with NodeProfile ID
	oldNPO := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.NodeProfileObject}
	oldDev := IPAndEOJ{IP: oldIP, EOJ: echonet_lite.MakeEOJ(0x027B, 1)}
	devices.RegisterProperty(oldNPO, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT}, time.Now())
	devices.RegisterDevice(oldDev)

	// Register devices on new IP with same NodeProfile ID
	newNPO := IPAndEOJ{IP: newIP, EOJ: echonet_lite.NodeProfileObject}
	devices.RegisterProperty(newNPO, Property{EPC: echonet_lite.EPC_NPO_IDNumber, EDT: idEDT}, time.Now())

	// Drain add events
	for len(ch) > 0 {
		<-ch
	}

	// Find old IP
	oldIPs := devices.FindIPsWithSameNodeProfileID(idEDT, newIP.String())
	assert.Equal(t, []string{oldIP.String()}, oldIPs)

	// Remove old IP's devices
	removed := devices.RemoveAllDevicesByIP(oldIP)
	assert.Len(t, removed, 2) // NodeProfile + device

	// Old IP should be gone
	assert.False(t, devices.HasIP(oldIP))

	// New IP should still be there
	assert.True(t, devices.HasIP(newIP))

	// No more matches for old IP
	oldIPs = devices.FindIPsWithSameNodeProfileID(idEDT, newIP.String())
	assert.Empty(t, oldIPs)
}
