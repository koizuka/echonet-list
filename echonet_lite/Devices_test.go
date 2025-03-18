package echonet_lite

import (
	"net"
	"os"
	"testing"
	"time"
)

// HasPropertyWithValue is a test helper function that checks if a property with the expected EPC and EDT exists for the given device
func HasPropertyWithValue(d Devices, device IPAndEOJ, epc EPCType, expectedEDT []byte) bool {
	classCode := device.EOJ.ClassCode()
	instanceCode := device.EOJ.InstanceCode()
	criteria := FilterCriteria{
		Device:         &DeviceSpecifier{IP: &device.IP, ClassCode: &classCode, InstanceCode: &instanceCode},
		PropertyValues: []Property{{EPC: epc, EDT: expectedEDT}},
	}

	return d.Filter(criteria).Len() > 0
}

func TestDevices_SaveToFile(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "test_save.json"
	defer os.Remove(tempFile) // Clean up after test

	// Define IP addresses
	ip1 := net.ParseIP("192.168.1.1")

	// Create a Devices instance with test data
	devices := NewDevices()

	// Create test EOJ and Property
	eoj := EOJ(0x013001) // Example EOJ
	epc := EPCType(0x80) // Example EPC
	property := Property{
		EPC: epc,
		EDT: []byte{0x30},
	}

	// Register the test property
	ip1eoj := IPAndEOJ{ip1, eoj}
	devices.RegisterProperty(ip1eoj, property)

	// Save to file
	err := devices.SaveToFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to save devices to file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatalf("File was not created: %v", err)
	}

	// Create a new Devices instance and load the saved file
	loadedDevices := NewDevices()
	err = loadedDevices.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load devices from file: %v", err)
	}

	// Verify the loaded data using public methods
	if !loadedDevices.HasIP(ip1) {
		t.Errorf("Expected device with IP 192.168.1.1 to exist, but it doesn't")
	}

	if !loadedDevices.IsKnownDevice(ip1eoj) {
		t.Errorf("Expected device with IP 192.168.1.1 and EOJ %v to exist, but it doesn't", eoj)
	}

	// Verify the property value (EPC and EDT) is correctly saved and loaded
	if !HasPropertyWithValue(loadedDevices, ip1eoj, epc, []byte{0x30}) {
		t.Errorf("Property value was not correctly saved and loaded")
	}
}

func TestDevices_LoadFromFile(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "test_load.json"
	defer os.Remove(tempFile) // Clean up after test

	// Define IP addresses
	ip1 := net.ParseIP("192.168.1.1")

	// Create a temporary Devices instance with test data
	tempDevices := NewDevices()

	// Create test EOJ and Property
	eoj := EOJ(0x013001) // Example EOJ
	epc := EPCType(0x80) // Example EPC
	property := Property{
		EPC: epc,
		EDT: []byte{0x30},
	}

	ip1eoj := IPAndEOJ{ip1, eoj}

	// Register the test property
	tempDevices.RegisterProperty(ip1eoj, property)

	// Save to the temporary file
	err := tempDevices.SaveToFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to save test data to file: %v", err)
	}

	// Create a new Devices instance
	devices := NewDevices()

	// Load from file
	err = devices.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load devices from file: %v", err)
	}

	// Verify the loaded data using public methods
	if !devices.HasIP(ip1) {
		t.Errorf("Expected device with IP 192.168.1.1 to exist, but it doesn't")
	}

	if !devices.IsKnownDevice(ip1eoj) {
		t.Errorf("Expected device with IP 192.168.1.1 and EOJ %v to exist, but it doesn't", eoj)
	}

	// Verify the property value (EPC and EDT) is correctly loaded
	if !HasPropertyWithValue(devices, ip1eoj, epc, []byte{0x30}) {
		t.Errorf("Property value was not correctly loaded")
	}
}

func TestDevices_SaveAndLoadFromFile(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "test_save_load.json"
	defer os.Remove(tempFile) // Clean up after test

	// Define IP addresses
	ip1 := net.ParseIP("192.168.1.1")
	ip2 := net.ParseIP("192.168.1.2")

	// Create a Devices instance with test data
	originalDevices := NewDevices()

	// Create test EOJs and Properties
	eoj1 := EOJ(0x013001) // Example EOJ 1
	eoj2 := EOJ(0x028801) // Example EOJ 2

	epc1 := EPCType(0x80) // Example EPC 1
	epc2 := EPCType(0x81) // Example EPC 2

	property1 := Property{
		EPC: epc1,
		EDT: []byte{0x30},
	}

	property2 := Property{
		EPC: epc2,
		EDT: []byte{0x41, 0x42},
	}

	ip1eoj1 := IPAndEOJ{ip1, eoj1}
	ip2eoj2 := IPAndEOJ{ip2, eoj2}

	// Register the test properties
	originalDevices.RegisterProperty(ip1eoj1, property1)
	originalDevices.RegisterProperty(ip2eoj2, property2)

	// Save to file
	err := originalDevices.SaveToFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to save devices to file: %v", err)
	}

	// Create a new Devices instance
	loadedDevices := NewDevices()

	// Load from file
	err = loadedDevices.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load devices from file: %v", err)
	}

	// Verify the loaded data matches the original data using public methods
	// Check IPs
	if !loadedDevices.HasIP(ip1) {
		t.Errorf("Expected loaded device with IP 192.168.1.1 to exist, but it doesn't")
	}
	if !loadedDevices.HasIP(ip2) {
		t.Errorf("Expected loaded device with IP 192.168.1.2 to exist, but it doesn't")
	}

	// Check EOJs
	if !loadedDevices.IsKnownDevice(ip1eoj1) {
		t.Errorf("Expected loaded device with IP 192.168.1.1 and EOJ %v to exist, but it doesn't", eoj1)
	}
	if !loadedDevices.IsKnownDevice(ip2eoj2) {
		t.Errorf("Expected loaded device with IP 192.168.1.2 and EOJ %v to exist, but it doesn't", eoj2)
	}

	// Verify the property values (EPC and EDT) are correctly saved and loaded
	if !HasPropertyWithValue(loadedDevices, ip1eoj1, epc1, []byte{0x30}) {
		t.Errorf("Property 1 value was not correctly saved and loaded")
	}
	if !HasPropertyWithValue(loadedDevices, ip2eoj2, epc2, []byte{0x41, 0x42}) {
		t.Errorf("Property 2 value was not correctly saved and loaded")
	}
}

func TestDevices_SaveToFile_Error(t *testing.T) {
	// Create a Devices instance
	devices := NewDevices()

	// Try to save to an invalid path
	err := devices.SaveToFile("/invalid/path/test.json")
	if err == nil {
		t.Error("Expected an error when saving to an invalid path, but got nil")
	}
}

func TestDevices_LoadFromFile_Error(t *testing.T) {
	// Create a Devices instance
	devices := NewDevices()

	// Try to load from a non-existent file
	// After the modification, LoadFromFile should return nil for non-existent files
	err := devices.LoadFromFile("non_existent_file.json")
	if err != nil {
		t.Errorf("Expected nil when loading from a non-existent file, but got error: %v", err)
	}

	// Create a temporary file with invalid JSON
	tempFile := "invalid_json.json"
	defer os.Remove(tempFile)

	err = os.WriteFile(tempFile, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON to file: %v", err)
	}

	// Try to load from a file with invalid JSON
	err = devices.LoadFromFile(tempFile)
	if err == nil {
		t.Error("Expected an error when loading from a file with invalid JSON, but got nil")
	}
}

func TestDevices_DeviceEvents(t *testing.T) {
	// Devicesインスタンスの作成
	devices := NewDevices()

	// イベント受信用チャンネルの作成（バッファ付き）
	eventCh := make(chan DeviceEvent, 10)

	// イベントチャンネルの設定
	devices.SetEventChannel(eventCh)

	// テスト用のデバイス情報
	ip1 := net.ParseIP("192.168.1.1")
	eoj1 := EOJ(0x013001)
	device1 := IPAndEOJ{IP: ip1, EOJ: eoj1}

	// 1. 新規デバイス登録時にイベントが送信されることを確認
	devices.RegisterDevice(device1)

	// イベントチャンネルからイベントを受信
	select {
	case event := <-eventCh:
		// イベントの内容を検証
		if !event.Device.IP.Equal(ip1) || event.Device.EOJ != eoj1 || event.Type != DeviceEventAdded {
			t.Errorf("Expected event for device %v with type %v, got %v with type %v",
				device1, DeviceEventAdded, event.Device, event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timed out waiting for device event")
	}

	// 2. 同じデバイスを再登録してもイベントが送信されないことを確認
	devices.RegisterDevice(device1)

	// イベントが送信されないことを確認
	select {
	case event := <-eventCh:
		t.Errorf("Unexpected event received: %v", event)
	case <-time.After(100 * time.Millisecond):
		// タイムアウトは期待通りの動作
	}

	// 3. プロパティ登録時に新しいデバイスが登録された場合のイベント送信を確認
	ip2 := net.ParseIP("192.168.1.2")
	eoj2 := EOJ(0x013002)
	device2 := IPAndEOJ{IP: ip2, EOJ: eoj2}
	property := Property{EPC: EPCType(0x80), EDT: []byte{0x30}}

	devices.RegisterProperty(device2, property)

	// イベントチャンネルからイベントを受信
	select {
	case event := <-eventCh:
		// イベントの内容を検証
		if !event.Device.IP.Equal(ip2) || event.Device.EOJ != eoj2 || event.Type != DeviceEventAdded {
			t.Errorf("Expected event for device %v with type %v, got %v with type %v",
				device2, DeviceEventAdded, event.Device, event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timed out waiting for device event")
	}

	// 4. バッファがいっぱいの場合のテスト
	// バッファサイズ0のチャンネルを作成
	blockingCh := make(chan DeviceEvent)
	devices.SetEventChannel(blockingCh)

	// 新しいデバイスを登録（チャンネルがブロックされているため送信されない）
	ip3 := net.ParseIP("192.168.1.3")
	eoj3 := EOJ(0x013003)
	device3 := IPAndEOJ{IP: ip3, EOJ: eoj3}

	// ブロックされたチャンネルでもデバイス登録が成功することを確認
	devices.RegisterDevice(device3)

	// デバイスが正しく登録されていることを確認
	if !devices.IsKnownDevice(device3) {
		t.Errorf("Device %v was not registered", device3)
	}
}
