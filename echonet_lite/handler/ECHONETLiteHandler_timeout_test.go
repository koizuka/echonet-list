package handler

import (
	"echonet-list/echonet_lite"
	"net"
	"testing"
)

// MockOfflineManager is a mock implementation of OfflineManager for testing
type MockOfflineManager struct {
	offlineDevices  map[string]bool // key: device.Key()
	setOfflineCalls []struct {
		device  IPAndEOJ
		offline bool
	}
	setOfflineByIPCalls []struct {
		ip      net.IP
		offline bool
	}
}

func NewMockOfflineManager() *MockOfflineManager {
	return &MockOfflineManager{
		offlineDevices: make(map[string]bool),
	}
}

func (m *MockOfflineManager) IsOffline(device IPAndEOJ) bool {
	return m.offlineDevices[device.Key()]
}

func (m *MockOfflineManager) SetOffline(device IPAndEOJ, offline bool) {
	m.offlineDevices[device.Key()] = offline
	m.setOfflineCalls = append(m.setOfflineCalls, struct {
		device  IPAndEOJ
		offline bool
	}{device, offline})
}

func (m *MockOfflineManager) SetOfflineByIP(ip net.IP, offline bool) {
	// Mark all devices with this IP as offline/online
	// In the mock, we just record the call
	m.setOfflineByIPCalls = append(m.setOfflineByIPCalls, struct {
		ip      net.IP
		offline bool
	}{ip, offline})
}

func TestHandleDeviceTimeout_NodeProfile(t *testing.T) {
	// Setup
	mock := NewMockOfflineManager()
	ip := net.ParseIP("192.168.1.100")

	// Create a NodeProfile device
	nodeProfileDevice := IPAndEOJ{
		IP:  ip,
		EOJ: echonet_lite.NodeProfileObject,
	}

	// Execute
	handleDeviceTimeout(nodeProfileDevice, mock)

	// Verify: SetOfflineByIP should be called with the IP
	if len(mock.setOfflineByIPCalls) != 1 {
		t.Errorf("Expected 1 SetOfflineByIP call, got %d", len(mock.setOfflineByIPCalls))
	}
	if len(mock.setOfflineByIPCalls) > 0 {
		if !mock.setOfflineByIPCalls[0].ip.Equal(ip) {
			t.Errorf("Expected IP %s, got %s", ip, mock.setOfflineByIPCalls[0].ip)
		}
		if !mock.setOfflineByIPCalls[0].offline {
			t.Error("Expected offline to be true")
		}
	}

	// Verify: SetOffline should not be called
	if len(mock.setOfflineCalls) != 0 {
		t.Errorf("Expected 0 SetOffline calls, got %d", len(mock.setOfflineCalls))
	}
}

func TestHandleDeviceTimeout_NonNodeProfile_NodeProfileOffline(t *testing.T) {
	// Setup
	mock := NewMockOfflineManager()
	ip := net.ParseIP("192.168.1.100")

	// Set NodeProfile as offline
	nodeProfile := IPAndEOJ{
		IP:  ip,
		EOJ: echonet_lite.NodeProfileObject,
	}
	mock.SetOffline(nodeProfile, true)

	// Create a non-NodeProfile device
	device := IPAndEOJ{
		IP:  ip,
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}

	// Reset call tracking after setup
	mock.setOfflineCalls = nil

	// Execute
	handleDeviceTimeout(device, mock)

	// Verify: SetOffline should be called for the device
	if len(mock.setOfflineCalls) != 1 {
		t.Errorf("Expected 1 SetOffline call, got %d", len(mock.setOfflineCalls))
	}
	if len(mock.setOfflineCalls) > 0 {
		if !mock.setOfflineCalls[0].device.IP.Equal(device.IP) || mock.setOfflineCalls[0].device.EOJ != device.EOJ {
			t.Errorf("Expected device %s, got %s", device.Specifier(), mock.setOfflineCalls[0].device.Specifier())
		}
		if !mock.setOfflineCalls[0].offline {
			t.Error("Expected offline to be true")
		}
	}

	// Verify: SetOfflineByIP should not be called
	if len(mock.setOfflineByIPCalls) != 0 {
		t.Errorf("Expected 0 SetOfflineByIP calls, got %d", len(mock.setOfflineByIPCalls))
	}
}

func TestHandleDeviceTimeout_NonNodeProfile_NodeProfileOnline(t *testing.T) {
	// Setup
	mock := NewMockOfflineManager()
	ip := net.ParseIP("192.168.1.100")

	// NodeProfile is online (not set as offline)
	// No need to set anything since the default state is online

	// Create a non-NodeProfile device
	device := IPAndEOJ{
		IP:  ip,
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}

	// Execute
	handleDeviceTimeout(device, mock)

	// Verify: SetOffline should NOT be called
	if len(mock.setOfflineCalls) != 0 {
		t.Errorf("Expected 0 SetOffline calls, got %d", len(mock.setOfflineCalls))
	}

	// Verify: SetOfflineByIP should NOT be called
	if len(mock.setOfflineByIPCalls) != 0 {
		t.Errorf("Expected 0 SetOfflineByIP calls, got %d", len(mock.setOfflineByIPCalls))
	}
}

func TestHandleDeviceTimeout_DifferentIPs(t *testing.T) {
	// Setup
	mock := NewMockOfflineManager()
	ip1 := net.ParseIP("192.168.1.100")
	ip2 := net.ParseIP("192.168.1.101")

	// Set NodeProfile of IP1 as offline
	nodeProfile1 := IPAndEOJ{
		IP:  ip1,
		EOJ: echonet_lite.NodeProfileObject,
	}
	mock.SetOffline(nodeProfile1, true)

	// Create a device on IP2 (different IP)
	device2 := IPAndEOJ{
		IP:  ip2,
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}

	// Reset call tracking after setup
	mock.setOfflineCalls = nil

	// Execute: Device timeout on IP2 should not be affected by IP1's NodeProfile
	handleDeviceTimeout(device2, mock)

	// Verify: SetOffline should NOT be called since IP2's NodeProfile is online
	if len(mock.setOfflineCalls) != 0 {
		t.Errorf("Expected 0 SetOffline calls, got %d", len(mock.setOfflineCalls))
	}

	// Verify: SetOfflineByIP should NOT be called
	if len(mock.setOfflineByIPCalls) != 0 {
		t.Errorf("Expected 0 SetOfflineByIP calls, got %d", len(mock.setOfflineByIPCalls))
	}
}

func TestHandleDeviceTimeout_MultipleDevicesSameIP(t *testing.T) {
	// Setup
	mock := NewMockOfflineManager()
	ip := net.ParseIP("192.168.1.100")

	// Create NodeProfile device
	nodeProfile := IPAndEOJ{
		IP:  ip,
		EOJ: echonet_lite.NodeProfileObject,
	}

	// Execute: NodeProfile timeout should mark all devices with same IP as offline
	handleDeviceTimeout(nodeProfile, mock)

	// Verify: SetOfflineByIP should be called once
	if len(mock.setOfflineByIPCalls) != 1 {
		t.Errorf("Expected 1 SetOfflineByIP call, got %d", len(mock.setOfflineByIPCalls))
	}
	if len(mock.setOfflineByIPCalls) > 0 {
		if !mock.setOfflineByIPCalls[0].ip.Equal(ip) {
			t.Errorf("Expected IP %s, got %s", ip, mock.setOfflineByIPCalls[0].ip)
		}
		if !mock.setOfflineByIPCalls[0].offline {
			t.Error("Expected offline to be true")
		}
	}

	// Verify: SetOffline should not be called for individual devices
	if len(mock.setOfflineCalls) != 0 {
		t.Errorf("Expected 0 SetOffline calls, got %d", len(mock.setOfflineCalls))
	}
}
