package client

import (
	"encoding/json"
	"net"
	"testing"

	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
)

func TestHandleDeviceOffline(t *testing.T) {
	// WebSocketクライアントを作成（最小限の初期化）
	client := &WebSocketClient{
		devices: make(map[string]WebSocketDeviceAndProperties),
	}

	// テスト用デバイスを作成
	deviceIP := "192.168.1.100"
	deviceEOJ := "0EF0:1"
	deviceID := deviceIP + " " + deviceEOJ

	testDevice := WebSocketDeviceAndProperties{
		DeviceAndProperties: handler.DeviceAndProperties{
			Device: handler.IPAndEOJ{
				IP:  net.ParseIP(deviceIP),
				EOJ: MakeEOJ(0x0EF0, 0x01),
			},
		},
		IsOffline: false,
	}

	// デバイスをクライアントに追加
	client.devicesMutex.Lock()
	client.devices[deviceID] = testDevice
	client.devicesMutex.Unlock()

	// device_offline メッセージを作成
	payload := protocol.DeviceOfflinePayload{
		IP:  deviceIP,
		EOJ: deviceEOJ,
	}

	payloadBytes, _ := json.Marshal(payload)
	msg := &protocol.Message{
		Type:    "device_offline",
		Payload: json.RawMessage(payloadBytes),
	}

	// handleDeviceOffline を実行
	client.handleDeviceOffline(msg)

	// デバイスがまだ存在し、IsOfflineがtrueになっていることを確認
	client.devicesMutex.Lock()
	device, exists := client.devices[deviceID]
	client.devicesMutex.Unlock()

	if !exists {
		t.Fatal("デバイスが削除されてしまいました。IsOfflineフラグを設定するべきです")
	}

	if !device.IsOffline {
		t.Error("デバイスのIsOfflineフラグがtrueに設定されていません")
	}

	// 元のデバイスと同じ基本情報を持っていることを確認
	if !device.Device.IP.Equal(testDevice.Device.IP) {
		t.Errorf("デバイスのIPが変更されました: expected %s, got %s", testDevice.Device.IP, device.Device.IP)
	}

	if device.Device.EOJ != testDevice.Device.EOJ {
		t.Errorf("デバイスのEOJが変更されました: expected %v, got %v", testDevice.Device.EOJ, device.Device.EOJ)
	}
}

func TestHandleDeviceOffline_DeviceNotExists(t *testing.T) {
	// WebSocketクライアントを作成
	client := &WebSocketClient{
		devices: make(map[string]WebSocketDeviceAndProperties),
	}

	// 存在しないデバイスのdevice_offline メッセージを作成
	payload := protocol.DeviceOfflinePayload{
		IP:  "192.168.1.999",
		EOJ: "0EF0:1",
	}

	payloadBytes, _ := json.Marshal(payload)
	msg := &protocol.Message{
		Type:    "device_offline",
		Payload: json.RawMessage(payloadBytes),
	}

	// handleDeviceOffline を実行（パニックしないことを確認）
	client.handleDeviceOffline(msg)

	// 何もエラーが発生しないことを確認（この時点でテストが完了していればOK）
}

func TestHandleDeviceOffline_MultipleDevices(t *testing.T) {
	// WebSocketクライアントを作成
	client := &WebSocketClient{
		devices: make(map[string]WebSocketDeviceAndProperties),
	}

	// 複数のテスト用デバイスを作成
	devices := []WebSocketDeviceAndProperties{
		{
			DeviceAndProperties: handler.DeviceAndProperties{
				Device: handler.IPAndEOJ{
					IP:  net.ParseIP("192.168.1.100"),
					EOJ: MakeEOJ(0x0EF0, 0x01),
				},
			},
			IsOffline: false,
		},
		{
			DeviceAndProperties: handler.DeviceAndProperties{
				Device: handler.IPAndEOJ{
					IP:  net.ParseIP("192.168.1.101"),
					EOJ: MakeEOJ(0x0291, 0x01),
				},
			},
			IsOffline: false,
		},
	}

	// デバイスをクライアントに追加
	client.devicesMutex.Lock()
	client.devices["192.168.1.100 0EF0:1"] = devices[0]
	client.devices["192.168.1.101 0291:1"] = devices[1]
	client.devicesMutex.Unlock()

	// 最初のデバイスをオフラインに設定
	payload := protocol.DeviceOfflinePayload{
		IP:  "192.168.1.100",
		EOJ: "0EF0:1",
	}

	payloadBytes, _ := json.Marshal(payload)
	msg := &protocol.Message{
		Type:    "device_offline",
		Payload: json.RawMessage(payloadBytes),
	}

	client.handleDeviceOffline(msg)

	// 最初のデバイスがオフラインになり、2番目のデバイスはオンラインのままであることを確認
	client.devicesMutex.Lock()

	device1, exists1 := client.devices["192.168.1.100 0EF0:1"]
	device2, exists2 := client.devices["192.168.1.101 0291:1"]

	client.devicesMutex.Unlock()

	if !exists1 {
		t.Fatal("最初のデバイスが削除されました")
	}
	if !exists2 {
		t.Fatal("2番目のデバイスが削除されました")
	}

	if !device1.IsOffline {
		t.Error("最初のデバイスがオフライン状態になっていません")
	}
	if device2.IsOffline {
		t.Error("2番目のデバイスがオフライン状態になってしまいました")
	}
}

func TestGetDevices_IncludesOfflineDevices(t *testing.T) {
	// WebSocketクライアントを作成
	client := &WebSocketClient{
		devices: make(map[string]WebSocketDeviceAndProperties),
	}

	// テスト用デバイスを作成（オンラインとオフライン）
	onlineDevice := WebSocketDeviceAndProperties{
		DeviceAndProperties: handler.DeviceAndProperties{
			Device: handler.IPAndEOJ{
				IP:  net.ParseIP("192.168.1.100"),
				EOJ: MakeEOJ(0x0EF0, 0x01),
			},
		},
		IsOffline: false,
	}

	offlineDevice := WebSocketDeviceAndProperties{
		DeviceAndProperties: handler.DeviceAndProperties{
			Device: handler.IPAndEOJ{
				IP:  net.ParseIP("192.168.1.101"),
				EOJ: MakeEOJ(0x0EF0, 0x01),
			},
		},
		IsOffline: true,
	}

	// デバイスをクライアントに追加
	client.devicesMutex.Lock()
	client.devices["192.168.1.100 0EF0:1"] = onlineDevice
	client.devices["192.168.1.101 0EF0:1"] = offlineDevice
	client.devicesMutex.Unlock()

	// クラスコード0x0EF0のデバイスを検索
	classCode := EOJClassCode(0x0EF0)
	deviceSpec := DeviceSpecifier{
		ClassCode: &classCode,
	}

	devices := client.GetDevices(deviceSpec)

	// オンラインとオフライン両方のデバイスが取得されることを確認
	if len(devices) != 2 {
		t.Errorf("期待されるデバイス数: 2, 実際: %d", len(devices))
	}

	// IPアドレスで確認
	foundIPs := make(map[string]bool)
	for _, device := range devices {
		foundIPs[device.IP.String()] = true
	}

	if !foundIPs["192.168.1.100"] {
		t.Error("オンラインデバイス(192.168.1.100)が見つかりませんでした")
	}
	if !foundIPs["192.168.1.101"] {
		t.Error("オフラインデバイス(192.168.1.101)が見つかりませんでした")
	}
}

func TestGetDevices_ByIPIncludesOfflineDevice(t *testing.T) {
	// WebSocketクライアントを作成
	client := &WebSocketClient{
		devices: make(map[string]WebSocketDeviceAndProperties),
	}

	// オフライン状態のデバイスを作成
	offlineDevice := WebSocketDeviceAndProperties{
		DeviceAndProperties: handler.DeviceAndProperties{
			Device: handler.IPAndEOJ{
				IP:  net.ParseIP("192.168.1.100"),
				EOJ: MakeEOJ(0x0EF0, 0x01),
			},
		},
		IsOffline: true,
	}

	// デバイスをクライアントに追加
	client.devicesMutex.Lock()
	client.devices["192.168.1.100 0EF0:1"] = offlineDevice
	client.devicesMutex.Unlock()

	// IPアドレスでデバイスを検索
	ip := net.ParseIP("192.168.1.100")
	deviceSpec := DeviceSpecifier{
		IP: &ip,
	}

	devices := client.GetDevices(deviceSpec)

	// オフラインデバイスも取得されることを確認
	if len(devices) != 1 {
		t.Errorf("期待されるデバイス数: 1, 実際: %d", len(devices))
	}

	if devices[0].IP.String() != "192.168.1.100" {
		t.Errorf("期待されるIP: 192.168.1.100, 実際: %s", devices[0].IP.String())
	}
}
