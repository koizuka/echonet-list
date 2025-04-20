package echonet_lite

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"
)

// HasPropertyWithValue is a test helper function that checks if a property with the expected EPC and EDT exists for the given device
func HasPropertyWithValue(d Devices, device IPAndEOJ, epc EPCType, expectedEDT []byte) bool {
	criteria := FilterCriteria{
		Device:         DeviceSpecifierFromIPAndEOJ(device),
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
	devices.RegisterProperty(ip1eoj, property, time.Now())

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
	tempDevices.RegisterProperty(ip1eoj, property, time.Now())

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
	originalDevices.RegisterProperty(ip1eoj1, property1, time.Now())
	originalDevices.RegisterProperty(ip2eoj2, property2, time.Now())

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

	devices.RegisterProperty(device2, property, time.Now())

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

// TestDeviceProperties_MarshalJSON は DeviceProperties の MarshalJSON メソッドをテストします
func TestDeviceProperties_MarshalJSON(t *testing.T) {
	// テスト用のデータを作成
	props := make(DeviceProperties)

	// EOJ を作成
	eoj1 := MakeEOJ(HomeAirConditioner_ClassCode, 1)
	eoj2 := MakeEOJ(LightingSystem_ClassCode, 2)
	eoj3 := MakeEOJ(NodeProfile_ClassCode, 0) // インスタンスコード 0 のケース

	// プロパティを作成
	props[eoj1] = make(EPCPropertyMap)
	props[eoj1][EPCType(0x80)] = Property{EPC: EPCType(0x80), EDT: []byte{0x30}}

	props[eoj2] = make(EPCPropertyMap)
	props[eoj2][EPCType(0x81)] = Property{EPC: EPCType(0x81), EDT: []byte{0x41, 0x42}}

	props[eoj3] = make(EPCPropertyMap)
	props[eoj3][EPCType(0x82)] = Property{EPC: EPCType(0x82), EDT: []byte{0x50}}

	// JSON にエンコード
	data, err := json.Marshal(props)
	if err != nil {
		t.Fatalf("Failed to marshal DeviceProperties: %v", err)
	}

	// 期待される文字列キーが含まれていることを確認
	jsonStr := string(data)

	// eoj1 のキーは "0130:1" 形式であることを確認
	if !bytes.Contains(data, []byte(`"0130:1"`)) {
		t.Errorf("Expected JSON to contain key \"0130:1\", but got: %s", jsonStr)
	}

	// eoj2 のキーは "02A3:2" 形式であることを確認
	if !bytes.Contains(data, []byte(`"02A3:2"`)) {
		t.Errorf("Expected JSON to contain key \"02A3:2\", but got: %s", jsonStr)
	}

	// eoj3 のキーは "0EF0" 形式であることを確認 (インスタンスコード 0 の場合)
	if !bytes.Contains(data, []byte(`"0EF0"`)) {
		t.Errorf("Expected JSON to contain key \"0EF0\", but got: %s", jsonStr)
	}
}

// TestDeviceProperties_UnmarshalJSON は DeviceProperties の UnmarshalJSON メソッドをテストします
func TestDeviceProperties_UnmarshalJSON(t *testing.T) {
	// テスト用の JSON データを作成
	jsonData := []byte(`{
		"0130:1": {
			"128": {
				"EPC": 128,
				"EDT": "MA=="
			}
		},
		"02A3:2": {
			"129": {
				"EPC": 129,
				"EDT": "QUI="
			}
		},
		"0EF0": {
			"130": {
				"EPC": 130,
				"EDT": "UA=="
			}
		}
	}`)

	// JSON からデコード
	var props DeviceProperties
	err := json.Unmarshal(jsonData, &props)
	if err != nil {
		t.Fatalf("Failed to unmarshal DeviceProperties: %v", err)
	}

	// 期待される EOJ キーが存在することを確認
	eoj1 := MakeEOJ(HomeAirConditioner_ClassCode, 1)
	eoj2 := MakeEOJ(LightingSystem_ClassCode, 2)
	eoj3 := MakeEOJ(NodeProfile_ClassCode, 0)

	// eoj1 のプロパティを確認
	if prop, ok := props[eoj1][EPCType(0x80)]; !ok {
		t.Errorf("Expected property with EPC 0x80 for EOJ %v to exist, but it doesn't", eoj1)
	} else if !bytes.Equal(prop.EDT, []byte{0x30}) {
		t.Errorf("Expected EDT [0x30] for EOJ %v and EPC 0x80, but got %v", eoj1, prop.EDT)
	}

	// eoj2 のプロパティを確認
	if prop, ok := props[eoj2][EPCType(0x81)]; !ok {
		t.Errorf("Expected property with EPC 0x81 for EOJ %v to exist, but it doesn't", eoj2)
	} else if !bytes.Equal(prop.EDT, []byte{0x41, 0x42}) {
		t.Errorf("Expected EDT [0x41, 0x42] for EOJ %v and EPC 0x81, but got %v", eoj2, prop.EDT)
	}

	// eoj3 のプロパティを確認
	if prop, ok := props[eoj3][EPCType(0x82)]; !ok {
		t.Errorf("Expected property with EPC 0x82 for EOJ %v to exist, but it doesn't", eoj3)
	} else if !bytes.Equal(prop.EDT, []byte{0x50}) {
		t.Errorf("Expected EDT [0x50] for EOJ %v and EPC 0x82, but got %v", eoj3, prop.EDT)
	}
}

// TestDevices_SaveLoadToFile_EOJFormat は EOJ キーが文字列形式で保存され、正しく読み込まれることをテストします
func TestDevices_SaveLoadToFile_EOJFormat(t *testing.T) {
	// 一時ファイルを作成
	tempFile := "test_eoj_format.json"
	defer os.Remove(tempFile)

	// テスト用のデータを作成
	devices := NewDevices()

	// IPアドレスを定義
	ip := net.ParseIP("192.168.1.1")

	// 異なるタイプの EOJ を作成
	eoj1 := MakeEOJ(HomeAirConditioner_ClassCode, 1)
	eoj2 := MakeEOJ(LightingSystem_ClassCode, 2)
	eoj3 := MakeEOJ(NodeProfile_ClassCode, 0) // インスタンスコード 0 のケース

	// プロパティを登録
	now := time.Now()
	devices.RegisterProperty(IPAndEOJ{IP: ip, EOJ: eoj1}, Property{EPC: EPCType(0x80), EDT: []byte{0x30}}, now)
	devices.RegisterProperty(IPAndEOJ{IP: ip, EOJ: eoj2}, Property{EPC: EPCType(0x81), EDT: []byte{0x41, 0x42}}, now)
	devices.RegisterProperty(IPAndEOJ{IP: ip, EOJ: eoj3}, Property{EPC: EPCType(0x82), EDT: []byte{0x50}}, now)

	// ファイルに保存
	err := devices.SaveToFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to save devices to file: %v", err)
	}

	// ファイルの内容を確認
	fileData, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	fileStr := string(fileData)

	// EOJ キーが文字列形式で保存されていることを確認
	if !bytes.Contains(fileData, []byte(`"0130:1"`)) {
		t.Errorf("Expected file to contain key \"0130:1\", but got: %s", fileStr)
	}

	if !bytes.Contains(fileData, []byte(`"02A3:2"`)) {
		t.Errorf("Expected file to contain key \"02A3:2\", but got: %s", fileStr)
	}

	if !bytes.Contains(fileData, []byte(`"0EF0"`)) {
		t.Errorf("Expected file to contain key \"0EF0\", but got: %s", fileStr)
	}

	// 新しい Devices インスタンスを作成して読み込み
	loadedDevices := NewDevices()
	err = loadedDevices.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to load devices from file: %v", err)
	}

	// 読み込んだデータを確認
	device1 := IPAndEOJ{IP: ip, EOJ: eoj1}
	device2 := IPAndEOJ{IP: ip, EOJ: eoj2}
	device3 := IPAndEOJ{IP: ip, EOJ: eoj3}

	// デバイスが存在することを確認
	if !loadedDevices.IsKnownDevice(device1) {
		t.Errorf("Expected device %v to exist, but it doesn't", device1)
	}

	if !loadedDevices.IsKnownDevice(device2) {
		t.Errorf("Expected device %v to exist, but it doesn't", device2)
	}

	if !loadedDevices.IsKnownDevice(device3) {
		t.Errorf("Expected device %v to exist, but it doesn't", device3)
	}

	// プロパティが正しく読み込まれていることを確認
	if !HasPropertyWithValue(loadedDevices, device1, EPCType(0x80), []byte{0x30}) {
		t.Errorf("Property value for device %v was not correctly loaded", device1)
	}

	if !HasPropertyWithValue(loadedDevices, device2, EPCType(0x81), []byte{0x41, 0x42}) {
		t.Errorf("Property value for device %v was not correctly loaded", device2)
	}

	if !HasPropertyWithValue(loadedDevices, device3, EPCType(0x82), []byte{0x50}) {
		t.Errorf("Property value for device %v was not correctly loaded", device3)
	}
}

func TestDevices_TimestampUpdate(t *testing.T) {
	devices := NewDevices()
	device := IPAndEOJ{IP: net.ParseIP("192.168.1.10"), EOJ: MakeEOJ(HomeAirConditioner_ClassCode, 1)}
	prop1 := Property{EPC: EPCOperationStatus, EDT: []byte{0x30}}
	prop2 := Property{EPC: EPC_HAC_OperationModeSetting, EDT: []byte{0x41}}

	// 1. RegisterProperty でタイムスタンプが設定されるか確認
	testTime1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	devices.RegisterProperty(device, prop1, testTime1)

	lastUpdate1 := devices.GetLastUpdateTime(device)
	if !lastUpdate1.Equal(testTime1) {
		t.Errorf("Expected timestamp %v after RegisterProperty, got %v", testTime1, lastUpdate1)
	}

	// 2. RegisterProperties でタイムスタンプが更新されるか確認
	testTime2 := testTime1.Add(time.Hour)
	devices.RegisterProperties(device, []Property{prop2}, testTime2) // 別のプロパティで更新

	lastUpdate2 := devices.GetLastUpdateTime(device)
	if !lastUpdate2.Equal(testTime2) {
		t.Errorf("Expected timestamp %v after RegisterProperties, got %v", testTime2, lastUpdate2)
	}

	// 3. 存在しないデバイスのタイムスタンプはゼロ値か確認
	nonExistentDevice := IPAndEOJ{IP: net.ParseIP("192.168.1.11"), EOJ: MakeEOJ(HomeAirConditioner_ClassCode, 2)}
	lastUpdate3 := devices.GetLastUpdateTime(nonExistentDevice)
	if !lastUpdate3.IsZero() {
		t.Errorf("Expected zero timestamp for non-existent device, got %v", lastUpdate3)
	}
}

func TestDevices_OfflineStatus(t *testing.T) {
	devices := NewDevices()
	device := IPAndEOJ{IP: net.ParseIP("192.168.0.10"), EOJ: EOJ(0x013001)}

	// 初期状態はオンライン (false)
	if devices.IsOffline(device) {
		t.Errorf("Expected device to be online initially, but IsOffline returned true")
	}

	// オフラインに設定
	devices.SetOffline(device, true)
	if !devices.IsOffline(device) {
		t.Errorf("Expected device to be offline after SetOffline(true), but IsOffline returned false")
	}

	// オンラインに戻す
	devices.SetOffline(device, false)
	if devices.IsOffline(device) {
		t.Errorf("Expected device to be online after SetOffline(false), but IsOffline returned true")
	}
}
