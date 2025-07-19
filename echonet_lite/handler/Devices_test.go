package handler

import (
	"bytes"
	"echonet-list/echonet_lite"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
	defer func(name string) {
		_ = os.Remove(name)
	}(tempFile) // Clean up after test

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
	ip1eoj := IPAndEOJ{IP: ip1, EOJ: eoj}
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
	defer func(name string) {
		_ = os.Remove(name)
	}(tempFile) // Clean up after test

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

	ip1eoj := IPAndEOJ{IP: ip1, EOJ: eoj}

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
	defer func(name string) {
		_ = os.Remove(name)
	}(tempFile) // Clean up after test

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

	ip1eoj1 := IPAndEOJ{IP: ip1, EOJ: eoj1}
	ip2eoj2 := IPAndEOJ{IP: ip2, EOJ: eoj2}

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
	// Verify the loaded data matches the original data using GetProperty and cmp.Diff
	// Use string representation for map key as IPAndEOJ is not comparable
	expectedProperties := map[string][]Property{
		ip1eoj1.String(): {property1},
		ip2eoj2.String(): {property2},
	}

	// Iterate through known devices in loadedDevices and compare
	// Note: This requires a way to iterate through loaded devices.
	// Assuming a hypothetical GetAllKnownDevices() method for demonstration.
	// If not available, we stick to checking only the expected devices.

	// Check expected devices explicitly
	devicesToCheck := []IPAndEOJ{ip1eoj1, ip2eoj2}
	for _, device := range devicesToCheck {
		deviceKey := device.String()
		expectedProps, keyExists := expectedProperties[deviceKey]
		if !keyExists {
			// This case should ideally not happen if devicesToCheck is derived from expectedProperties keys
			continue
		}

		if !loadedDevices.IsKnownDevice(device) {
			t.Errorf("Expected loaded device %v to exist, but it doesn't", device)
			continue
		}

		for _, expectedProp := range expectedProps {
			actualProp, ok := loadedDevices.GetProperty(device, expectedProp.EPC)
			if !ok {
				t.Errorf("Expected property EPC %X for device %v to exist, but it doesn't", expectedProp.EPC, device)
				continue
			}
			// Compare only EDT using cmp.Diff
			if diff := cmp.Diff(expectedProp.EDT, actualProp.EDT); diff != "" {
				t.Errorf("Property EDT mismatch for device %v, EPC %X (-want +got):\n%s", device, expectedProp.EPC, diff)
			}
		}
	}
	// Check if there are any unexpected devices or properties (optional, depends on strictness)
	// This requires iterating through loadedDevices, which might need a new public method like GetAllDevices()
	// For now, we only check if the expected data exists and is correct.
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
	defer func(name string) {
		_ = os.Remove(name)
	}(tempFile)

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
	eoj1 := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)
	eoj2 := echonet_lite.MakeEOJ(echonet_lite.LightingSystem_ClassCode, 2)
	eoj3 := echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 0) // インスタンスコード 0 のケース

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

	// 期待される DeviceProperties を作成
	expectedProps := make(DeviceProperties)
	eoj1 := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)
	eoj2 := echonet_lite.MakeEOJ(echonet_lite.LightingSystem_ClassCode, 2)
	eoj3 := echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 0)

	expectedProps[eoj1] = EPCPropertyMap{
		EPCType(0x80): {EPC: EPCType(0x80), EDT: []byte{0x30}},
	}
	expectedProps[eoj2] = EPCPropertyMap{
		EPCType(0x81): {EPC: EPCType(0x81), EDT: []byte{0x41, 0x42}},
	}
	expectedProps[eoj3] = EPCPropertyMap{
		EPCType(0x82): {EPC: EPCType(0x82), EDT: []byte{0x50}},
	}

	// cmp.Diff で比較
	if diff := cmp.Diff(expectedProps, props); diff != "" {
		t.Errorf("UnmarshalJSON mismatch (-want +got):\n%s", diff)
	}
}

// TestDevices_SaveLoadToFile_EOJFormat は EOJ キーが文字列形式で保存され、正しく読み込まれることをテストします
func TestDevices_SaveLoadToFile_EOJFormat(t *testing.T) {
	// 一時ファイルを作成
	tempFile := "test_eoj_format.json"
	defer func(name string) {
		_ = os.Remove(name)
	}(tempFile)

	// テスト用のデータを作成
	devices := NewDevices()

	// IPアドレスを定義
	ip := net.ParseIP("192.168.1.1")

	// 異なるタイプの EOJ を作成
	eoj1 := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)
	eoj2 := echonet_lite.MakeEOJ(echonet_lite.LightingSystem_ClassCode, 2)
	eoj3 := echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 0) // インスタンスコード 0 のケース

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
	// 読み込んだデータと元のデータを比較
	dev1 := IPAndEOJ{IP: ip, EOJ: eoj1}
	dev2 := IPAndEOJ{IP: ip, EOJ: eoj2}
	dev3 := IPAndEOJ{IP: ip, EOJ: eoj3}
	expectedProperties := map[string][]Property{
		dev1.String(): {{EPC: EPCType(0x80), EDT: []byte{0x30}}},
		dev2.String(): {{EPC: EPCType(0x81), EDT: []byte{0x41, 0x42}}},
		dev3.String(): {{EPC: EPCType(0x82), EDT: []byte{0x50}}},
	}

	devicesToCheck := []IPAndEOJ{dev1, dev2, dev3}
	for _, device := range devicesToCheck {
		deviceKey := device.String()
		expectedProps, keyExists := expectedProperties[deviceKey]
		if !keyExists {
			continue
		}

		if !loadedDevices.IsKnownDevice(device) {
			t.Errorf("Expected loaded device %v to exist, but it doesn't", device)
			continue
		}
		for _, expectedProp := range expectedProps {
			actualProp, ok := loadedDevices.GetProperty(device, expectedProp.EPC)
			if !ok {
				t.Errorf("Expected property EPC %X for device %v to exist, but it doesn't", expectedProp.EPC, device)
				continue
			}
			if diff := cmp.Diff(expectedProp.EDT, actualProp.EDT); diff != "" {
				t.Errorf("Property EDT mismatch for device %v, EPC %X (-want +got):\n%s", device, expectedProp.EPC, diff)
			}
		}
	}
}

func TestDevices_TimestampUpdate(t *testing.T) {
	devices := NewDevices()
	device := IPAndEOJ{IP: net.ParseIP("192.168.1.10"), EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)}
	prop1 := Property{EPC: echonet_lite.EPCOperationStatus, EDT: []byte{0x30}}
	prop2 := Property{EPC: echonet_lite.EPC_HAC_OperationModeSetting, EDT: []byte{0x41}}

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
	nonExistentDevice := IPAndEOJ{IP: net.ParseIP("192.168.1.11"), EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 2)}
	lastUpdate3 := devices.GetLastUpdateTime(nonExistentDevice)
	if !lastUpdate3.IsZero() {
		t.Errorf("Expected zero timestamp for non-existent device, got %v", lastUpdate3)
	}
}

// TestDevices_SaveLoadNewFormat は新しいJSONフォーマット(v2)での保存と読み込みをテストします
func TestDevices_SaveLoadNewFormat(t *testing.T) {
	tempFile := "test_save_load_v2.json"
	defer func(name string) {
		_ = os.Remove(name)
	}(tempFile)

	// 1. テストデータ準備
	originalDevices := NewDevices()
	ip := net.ParseIP("192.168.1.100")
	eoj1 := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1) // 0130:1
	eoj2 := echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 0)        // 0EF0
	epc1 := EPCType(0x80)                                                      // Operation Status
	epc2 := EPCType(0xB0)                                                      // Operation Mode Setting
	epc3 := EPCType(echonet_lite.EPC_NPO_VersionInfo)                          // 0x82 (Corrected)

	prop1 := Property{EPC: epc1, EDT: []byte{0x30}}                   // ON
	prop2 := Property{EPC: epc2, EDT: []byte{0x41}}                   // Auto
	prop3 := Property{EPC: epc3, EDT: []byte{0x01, 0x01, 0x61, 0x00}} // Ver. 1.1, Type a

	now := time.Now()
	originalDevices.RegisterProperty(IPAndEOJ{IP: ip, EOJ: eoj1}, prop1, now)
	originalDevices.RegisterProperty(IPAndEOJ{IP: ip, EOJ: eoj1}, prop2, now)
	originalDevices.RegisterProperty(IPAndEOJ{IP: ip, EOJ: eoj2}, prop3, now)

	// 2. ファイルに保存
	err := originalDevices.SaveToFile(tempFile)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// 3. ファイル内容の直接検証
	fileData, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// 3a. バージョン確認
	var rawMap map[string]interface{}
	err = json.Unmarshal(fileData, &rawMap)
	if err != nil {
		t.Fatalf("Unmarshal raw map failed: %v", err)
	}
	if version, ok := rawMap["version"].(float64); !ok || int(version) != currentDevicesFileVersion {
		t.Errorf("Expected version %d, got %v", currentDevicesFileVersion, rawMap["version"])
	}

	// 3b. データ構造とフォーマット確認 (部分的に確認)
	fileStr := string(fileData)
	// EOJキーの確認
	if !bytes.Contains(fileData, []byte(`"0130:1"`)) {
		t.Errorf("Expected file to contain EOJ key \"0130:1\", got: %s", fileStr)
	}
	if !bytes.Contains(fileData, []byte(`"0EF0"`)) {
		t.Errorf("Expected file to contain EOJ key \"0EF0\", got: %s", fileStr)
	}
	// EPCキー (0x80) と Base64値 (MA==) の確認
	if !bytes.Contains(fileData, []byte(`"0x80":"MA=="`)) {
		t.Errorf("Expected file to contain EPC key/value \"0x80\":\"MA==\", got: %s", fileStr)
	}
	// EPCキー (0xb0) と Base64値 (QQ==) の確認
	if !bytes.Contains(fileData, []byte(`"0xb0":"QQ=="`)) {
		t.Errorf("Expected file to contain EPC key/value \"0xb0\":\"QQ==\", got: %s", fileStr)
	}
	// EPCキー (0x82) と Base64値 (AQFhAA==) の確認
	if !bytes.Contains(fileData, []byte(`"0x82":"AQFhAA=="`)) {
		t.Errorf("Expected file to contain EPC key/value \"0x82\":\"AQFhAA==\", got: %s", fileStr)
	}

	// 4. ファイルから読み込み
	loadedDevices := NewDevices()
	err = loadedDevices.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	// 5. 読み込んだデータの検証
	dev1 := IPAndEOJ{IP: ip, EOJ: eoj1}
	dev2 := IPAndEOJ{IP: ip, EOJ: eoj2}
	expectedProperties := map[string][]Property{
		dev1.String(): {prop1, prop2},
		dev2.String(): {prop3},
	}

	devicesToCheck := []IPAndEOJ{dev1, dev2}
	for _, device := range devicesToCheck {
		deviceKey := device.String()
		expectedProps, keyExists := expectedProperties[deviceKey]
		if !keyExists {
			continue
		}

		if !loadedDevices.IsKnownDevice(device) {
			t.Errorf("Expected loaded device %v to exist, but it doesn't", device)
			continue
		}
		for _, expectedProp := range expectedProps {
			actualProp, ok := loadedDevices.GetProperty(device, expectedProp.EPC)
			if !ok {
				t.Errorf("Expected property EPC %X for device %v to exist, but it doesn't", expectedProp.EPC, device)
				continue
			}
			if diff := cmp.Diff(expectedProp.EDT, actualProp.EDT); diff != "" {
				t.Errorf("Property EDT mismatch for device %v, EPC %X (-want +got):\n%s", device, expectedProp.EPC, diff)
			}
		}
	}
}

// TestDevices_LoadOldFormat は古いJSONフォーマット(v1)の読み込みをテストします
func TestDevices_LoadOldFormat(t *testing.T) {
	tempFile := "test_load_v1.json"
	defer func(name string) {
		_ = os.Remove(name)
	}(tempFile)

	// 古いフォーマットのJSONデータを作成
	oldJsonData := []byte(`{
		"192.168.1.200": {
			"0130:1": {
				"128": {
					"EPC": 128,
					"EDT": "MQ=="
				},
				"176": {
					"EPC": 176,
					"EDT": "Qg=="
				}
			},
			"0EF0": {
				"130": {
					"EPC": 130,
					"EDT": "AQFiAA=="
				}
			}
		}
	}`)

	err := os.WriteFile(tempFile, oldJsonData, 0644)
	if err != nil {
		t.Fatalf("Failed to write old format JSON: %v", err)
	}

	// 読み込み
	loadedDevices := NewDevices()
	err = loadedDevices.LoadFromFile(tempFile)
	if err != nil {
		t.Fatalf("LoadFromFile failed for old format: %v", err)
	}
	// 期待されるプロパティデータを作成
	ip := net.ParseIP("192.168.1.200")
	eoj1 := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)
	eoj2 := echonet_lite.MakeEOJ(echonet_lite.NodeProfile_ClassCode, 0)
	dev1 := IPAndEOJ{IP: ip, EOJ: eoj1}
	dev2 := IPAndEOJ{IP: ip, EOJ: eoj2}
	expectedProperties := map[string][]Property{
		dev1.String(): {
			{EPC: EPCType(0x80), EDT: []byte{0x31}}, // "MQ=="
			{EPC: EPCType(0xB0), EDT: []byte{0x42}}, // "Qg=="
		},
		dev2.String(): {
			{EPC: EPCType(0x82), EDT: []byte{0x01, 0x01, 0x62, 0x00}}, // "AQFiAA=="
		},
	}

	// 読み込んだデータと比較
	devicesToCheck := []IPAndEOJ{dev1, dev2}
	for _, device := range devicesToCheck {
		deviceKey := device.String()
		expectedProps, keyExists := expectedProperties[deviceKey]
		if !keyExists {
			continue
		}

		if !loadedDevices.IsKnownDevice(device) {
			t.Errorf("Expected loaded device %v from old format to exist, but it doesn't", device)
			continue
		}
		for _, expectedProp := range expectedProps {
			actualProp, ok := loadedDevices.GetProperty(device, expectedProp.EPC)
			if !ok {
				t.Errorf("Expected property EPC %X for device %v (old fmt) to exist, but it doesn't", expectedProp.EPC, device)
				continue
			}
			if diff := cmp.Diff(expectedProp.EDT, actualProp.EDT); diff != "" {
				t.Errorf("Property EDT mismatch for device %v (old fmt), EPC %X (-want +got):\n%s", device, expectedProp.EPC, diff)
			}
		}
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

// TestDevices_SetOfflineEvents はオフライン/オンラインイベントの送信をテストします
func TestDevices_SetOfflineEvents(t *testing.T) {
	// イベントチャンネルを作成
	eventCh := make(chan DeviceEvent, 10)
	devices := NewDevices()
	devices.SetEventChannel(eventCh)

	device := IPAndEOJ{
		IP:  net.ParseIP("192.168.1.10"),
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}

	// 最初はオンライン状態
	if devices.IsOffline(device) {
		t.Errorf("Expected device to be online initially, but IsOffline returned true")
	}

	// オフラインに設定 - イベントが送信されるはず
	devices.SetOffline(device, true)
	if !devices.IsOffline(device) {
		t.Errorf("Expected device to be offline after SetOffline(true), but IsOffline returned false")
	}

	// オフラインイベントを確認
	select {
	case event := <-eventCh:
		if event.Type != DeviceEventOffline {
			t.Errorf("Expected DeviceEventOffline, got %v", event.Type)
		}
		if !event.Device.IP.Equal(device.IP) || event.Device.EOJ != device.EOJ {
			t.Errorf("Expected event for device %v, got %v", device, event.Device)
		}
	default:
		t.Errorf("Expected DeviceEventOffline event to be sent")
	}

	// オンラインに復旧 - イベントが送信されるはず
	devices.SetOffline(device, false)
	if devices.IsOffline(device) {
		t.Errorf("Expected device to be online after SetOffline(false), but IsOffline returned true")
	}

	// オンラインイベントを確認
	select {
	case event := <-eventCh:
		if event.Type != DeviceEventOnline {
			t.Errorf("Expected DeviceEventOnline, got %v", event.Type)
		}
		if !event.Device.IP.Equal(device.IP) || event.Device.EOJ != device.EOJ {
			t.Errorf("Expected event for device %v, got %v", device, event.Device)
		}
	default:
		t.Errorf("Expected DeviceEventOnline event to be sent")
	}

	// 既にオンラインのデバイスをオンラインに設定 - イベントは送信されないはず
	devices.SetOffline(device, false)
	select {
	case event := <-eventCh:
		t.Errorf("Expected no event for already online device, but got %v", event.Type)
	default:
		// 期待される動作: イベントなし
	}
}

// TestDeviceProperties_SetPropertyMap は SetPropertyMap の動的生成をテストします
func TestDeviceProperties_SetPropertyMap(t *testing.T) {
	props := make(DeviceProperties)
	eoj := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)

	// テスト用プロパティを設定
	err := props.Set(eoj,
		Property{EPC: echonet_lite.EPCOperationStatus, EDT: []byte{0x30}},              // 動作状態（読み取り専用）
		Property{EPC: echonet_lite.EPCInstallationLocation, EDT: []byte{0x08}},         // 設置場所（設定可能）
		Property{EPC: echonet_lite.EPCGetPropertyMap, EDT: []byte{0x80, 0x81}},         // GetPropertyMap（システム生成）
		Property{EPC: echonet_lite.EPCManufacturerCode, EDT: []byte{0x00, 0x00, 0x05}}, // メーカコード（読み取り専用）
		Property{EPC: echonet_lite.EPC_HAC_OperationModeSetting, EDT: []byte{0x41}},    // 運転モード設定（設定可能）
		Property{EPC: echonet_lite.EPC_HAC_TemperatureSetting, EDT: []byte{0x16}},      // 温度設定（設定可能）
	)
	if err != nil {
		t.Fatalf("Failed to set properties: %v", err)
	}

	// SetPropertyMapを取得
	setPropMapProperty, ok := props.Get(eoj, echonet_lite.EPCSetPropertyMap)
	if !ok {
		t.Fatalf("Expected SetPropertyMap to exist, but it doesn't")
	}

	// PropertyMapをデコード
	propMap := echonet_lite.DecodePropertyMap(setPropMapProperty.EDT)
	if propMap == nil {
		t.Fatalf("Failed to decode SetPropertyMap")
	}

	// 読み取り専用プロパティが含まれていないことを確認
	readOnlyEPCs := []EPCType{
		echonet_lite.EPCOperationStatus,                       // 動作状態
		echonet_lite.EPCGetPropertyMap,                        // GetPropertyMap
		echonet_lite.EPCSetPropertyMap,                        // SetPropertyMap
		echonet_lite.EPCStatusAnnouncementPropertyMap,         // 状態通知プロパティマップ
		echonet_lite.EPCStandardVersion,                       // 規格Version情報
		echonet_lite.EPCIdentificationNumber,                  // 識別番号
		echonet_lite.EPCMeasuredInstantaneousPowerConsumption, // 瞬時消費電力計測値
		echonet_lite.EPCMeasuredCumulativePowerConsumption,    // 積算消費電力量計測値
		echonet_lite.EPCManufacturerCode,                      // メーカコード
		echonet_lite.EPCBusinessFacilityCode,                  // 事業場コード
		echonet_lite.EPCProductCode,                           // 商品コード
		echonet_lite.EPCProductionNumber,                      // 製造番号
		echonet_lite.EPCProductionDate,                        // 製造年月日
	}

	for _, epc := range readOnlyEPCs {
		if propMap.Has(epc) {
			t.Errorf("Expected read-only EPC 0x%02X to be excluded from SetPropertyMap, but it's included", epc)
		}
	}

	// 設定可能プロパティが含まれていることを確認
	settableEPCs := []EPCType{
		echonet_lite.EPCInstallationLocation,      // 設置場所
		echonet_lite.EPC_HAC_OperationModeSetting, // 運転モード設定
		echonet_lite.EPC_HAC_TemperatureSetting,   // 温度設定
	}

	for _, epc := range settableEPCs {
		if !propMap.Has(epc) {
			t.Errorf("Expected settable EPC 0x%02X to be included in SetPropertyMap, but it's excluded", epc)
		}
	}
}

// TestLocalDevices_InstallationLocation はlocalDevicesでの設置場所プロパティをテストします
func TestLocalDevices_InstallationLocation(t *testing.T) {
	props := make(DeviceProperties)
	controllerEOJ := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	// コントローラーのプロパティを設定（ECHONETLiteHandlerと同じ構成）
	err := props.Set(controllerEOJ,
		Property{EPC: echonet_lite.EPCOperationStatus, EDT: []byte{0x30}},              // 動作状態：ON
		Property{EPC: echonet_lite.EPCInstallationLocation, EDT: []byte{0x00}},         // 設置場所：未設定
		Property{EPC: echonet_lite.EPCManufacturerCode, EDT: []byte{0x00, 0x00, 0x05}}, // メーカコード
	)
	if err != nil {
		t.Fatalf("Failed to set controller properties: %v", err)
	}

	// 設置場所プロパティが初期値0で設定されていることを確認
	installationProp, ok := props.Get(controllerEOJ, echonet_lite.EPCInstallationLocation)
	if !ok {
		t.Fatalf("Expected InstallationLocation property to exist, but it doesn't")
	}
	if len(installationProp.EDT) != 1 || installationProp.EDT[0] != 0x00 {
		t.Errorf("Expected InstallationLocation initial value to be [0x00], got %v", installationProp.EDT)
	}

	// SetPropertyMapに設置場所が含まれていることを確認
	setPropMapProperty, ok := props.Get(controllerEOJ, echonet_lite.EPCSetPropertyMap)
	if !ok {
		t.Fatalf("Expected SetPropertyMap to exist, but it doesn't")
	}

	propMap := echonet_lite.DecodePropertyMap(setPropMapProperty.EDT)
	if propMap == nil {
		t.Fatalf("Failed to decode SetPropertyMap")
	}

	if !propMap.Has(echonet_lite.EPCInstallationLocation) {
		t.Errorf("Expected InstallationLocation (0x%02X) to be included in SetPropertyMap, but it's excluded", echonet_lite.EPCInstallationLocation)
	}

	// 設置場所の値を変更できることを確認
	newInstallationProps := []Property{
		{EPC: echonet_lite.EPCInstallationLocation, EDT: []byte{0x08}}, // 台所
	}
	resultProps, success := props.SetProperties(controllerEOJ, newInstallationProps)
	if !success {
		t.Errorf("Expected SetProperties to succeed for InstallationLocation, but it failed")
	}
	if len(resultProps) != 1 || len(resultProps[0].EDT) != 0 {
		t.Errorf("Expected successful SetProperties to return empty EDT, got %v", resultProps[0].EDT)
	}

	// 変更された値を確認
	updatedProp, ok := props.Get(controllerEOJ, echonet_lite.EPCInstallationLocation)
	if !ok {
		t.Fatalf("Expected InstallationLocation property to exist after update, but it doesn't")
	}
	if len(updatedProp.EDT) != 1 || updatedProp.EDT[0] != 0x08 {
		t.Errorf("Expected InstallationLocation updated value to be [0x08], got %v", updatedProp.EDT)
	}
}

// TestDeviceProperties_GetPropertyMap は GetPropertyMap の動的生成をテストします
func TestDeviceProperties_GetPropertyMap(t *testing.T) {
	props := make(DeviceProperties)
	eoj := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)

	// テスト用プロパティを設定
	err := props.Set(eoj,
		Property{EPC: echonet_lite.EPCOperationStatus, EDT: []byte{0x30}},
		Property{EPC: echonet_lite.EPCInstallationLocation, EDT: []byte{0x08}},
		Property{EPC: echonet_lite.EPCManufacturerCode, EDT: []byte{0x00, 0x00, 0x05}},
		Property{EPC: echonet_lite.EPC_HAC_OperationModeSetting, EDT: []byte{0x41}},
	)
	if err != nil {
		t.Fatalf("Failed to set properties: %v", err)
	}

	// GetPropertyMapを取得
	getPropMapProperty, ok := props.Get(eoj, echonet_lite.EPCGetPropertyMap)
	if !ok {
		t.Fatalf("Expected GetPropertyMap to exist, but it doesn't")
	}

	// PropertyMapをデコード
	propMap := echonet_lite.DecodePropertyMap(getPropMapProperty.EDT)
	if propMap == nil {
		t.Fatalf("Failed to decode GetPropertyMap")
	}

	// すべてのプロパティが含まれていることを確認
	expectedEPCs := []EPCType{
		echonet_lite.EPCOperationStatus,
		echonet_lite.EPCInstallationLocation,
		echonet_lite.EPCManufacturerCode,
		echonet_lite.EPC_HAC_OperationModeSetting,
		echonet_lite.EPCGetPropertyMap, // GetPropertyMap自体は必ず含まれる
	}

	for _, epc := range expectedEPCs {
		if !propMap.Has(epc) {
			t.Errorf("Expected EPC 0x%02X to be included in GetPropertyMap, but it's excluded", epc)
		}
	}
}

// TestDeviceProperties_IsAnnouncementTarget はアナウンス対象判定をテストします
func TestDeviceProperties_IsAnnouncementTarget(t *testing.T) {
	props := make(DeviceProperties)
	controllerEOJ := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	// Status Announcement Property Mapを設定（設置場所を含む）
	announcementMap := make(PropertyMap)
	announcementMap.Set(echonet_lite.EPCInstallationLocation) // 0x81

	err := props.Set(controllerEOJ,
		Property{EPC: echonet_lite.EPCOperationStatus, EDT: []byte{0x30}},
		Property{EPC: echonet_lite.EPCInstallationLocation, EDT: []byte{0x00}},
		Property{EPC: echonet_lite.EPCStatusAnnouncementPropertyMap, EDT: announcementMap.Encode()},
	)
	if err != nil {
		t.Fatalf("Failed to set controller properties: %v", err)
	}

	// テストケース
	testCases := []struct {
		name     string
		eoj      EOJ
		epc      EPCType
		expected bool
	}{
		{
			name:     "設置場所はアナウンス対象",
			eoj:      controllerEOJ,
			epc:      echonet_lite.EPCInstallationLocation,
			expected: true,
		},
		{
			name:     "動作状態はアナウンス対象ではない",
			eoj:      controllerEOJ,
			epc:      echonet_lite.EPCOperationStatus,
			expected: false,
		},
		{
			name:     "存在しないEOJは常にfalse",
			eoj:      echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
			epc:      echonet_lite.EPCInstallationLocation,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := props.IsAnnouncementTarget(tc.eoj, tc.epc)
			if result != tc.expected {
				t.Errorf("IsAnnouncementTarget(%v, 0x%02X) = %v, expected %v",
					tc.eoj, tc.epc, result, tc.expected)
			}
		})
	}
}

// TestDeviceProperties_IsAnnouncementTarget_NoMap はStatus Announcement Property Mapが無い場合のテストです
func TestDeviceProperties_IsAnnouncementTarget_NoMap(t *testing.T) {
	props := make(DeviceProperties)
	controllerEOJ := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	// Status Announcement Property Mapを設定しない
	err := props.Set(controllerEOJ,
		Property{EPC: echonet_lite.EPCOperationStatus, EDT: []byte{0x30}},
		Property{EPC: echonet_lite.EPCInstallationLocation, EDT: []byte{0x00}},
	)
	if err != nil {
		t.Fatalf("Failed to set controller properties: %v", err)
	}

	// Status Announcement Property Mapが存在しない場合は常にfalse
	result := props.IsAnnouncementTarget(controllerEOJ, echonet_lite.EPCInstallationLocation)
	if result != false {
		t.Errorf("IsAnnouncementTarget with no Status Announcement Property Map should return false, got %v", result)
	}
}

// TestDevices_Filter_OfflineDevices はオフラインデバイスのフィルタリングをテストします
func TestDevices_Filter_OfflineDevices(t *testing.T) {
	devices := NewDevices()

	// テスト用のデバイスを作成
	ip1 := net.ParseIP("192.168.1.1")
	ip2 := net.ParseIP("192.168.1.2")
	eoj := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)

	device1 := IPAndEOJ{IP: ip1, EOJ: eoj}
	device2 := IPAndEOJ{IP: ip2, EOJ: eoj}

	// プロパティを登録
	property := Property{
		EPC: echonet_lite.EPCOperationStatus,
		EDT: []byte{0x30},
	}
	devices.RegisterProperty(device1, property, time.Now())
	devices.RegisterProperty(device2, property, time.Now())

	// device1をオフラインに設定
	devices.SetOffline(device1, true)

	tests := []struct {
		name             string
		criteria         FilterCriteria
		expectedCount    int
		shouldContain    []IPAndEOJ
		shouldNotContain []IPAndEOJ
	}{
		{
			name:             "空のFilterCriteria（デフォルト）- オフラインデバイスを含む",
			criteria:         FilterCriteria{},
			expectedCount:    2,
			shouldContain:    []IPAndEOJ{device1, device2},
			shouldNotContain: []IPAndEOJ{},
		},
		{
			name:             "ExcludeOffline=true - オフラインデバイスを除外",
			criteria:         FilterCriteria{ExcludeOffline: true},
			expectedCount:    1,
			shouldContain:    []IPAndEOJ{device2},
			shouldNotContain: []IPAndEOJ{device1},
		},
		{
			name:             "ExcludeOffline=false - オフラインデバイスを含む",
			criteria:         FilterCriteria{ExcludeOffline: false},
			expectedCount:    2,
			shouldContain:    []IPAndEOJ{device1, device2},
			shouldNotContain: []IPAndEOJ{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := devices.Filter(tt.criteria)

			// 期待される数をチェック
			if filtered.Len() != tt.expectedCount {
				t.Errorf("Expected %d devices, got %d", tt.expectedCount, filtered.Len())
			}

			// 含まれるべきデバイスをチェック
			for _, device := range tt.shouldContain {
				if !filtered.IsKnownDevice(device) {
					t.Errorf("Expected device %s to be included", device.String())
				}
			}

			// 含まれないべきデバイスをチェック
			for _, device := range tt.shouldNotContain {
				if filtered.IsKnownDevice(device) {
					t.Errorf("Expected device %s to be excluded", device.String())
				}
			}
		})
	}
}

// TestDevices_Filter_OfflineDevices_WithInitialState はinitial_state用のフィルタリングをテストします
func TestDevices_Filter_OfflineDevices_WithInitialState(t *testing.T) {
	devices := NewDevices()

	// テスト用のデバイスを作成
	ip1 := net.ParseIP("192.168.1.1")
	ip2 := net.ParseIP("192.168.1.2")
	eoj := echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1)

	device1 := IPAndEOJ{IP: ip1, EOJ: eoj}
	device2 := IPAndEOJ{IP: ip2, EOJ: eoj}

	// プロパティを登録
	property := Property{
		EPC: echonet_lite.EPCOperationStatus,
		EDT: []byte{0x30},
	}
	devices.RegisterProperty(device1, property, time.Now())
	devices.RegisterProperty(device2, property, time.Now())

	// device1をオフラインに設定
	devices.SetOffline(device1, true)

	// initial_state用のフィルタリング（空のFilterCriteriaでExcludeOfflineを自動設定）
	filtered := devices.Filter(FilterCriteria{ExcludeOffline: true})

	// オフラインデバイスが除外されることを確認
	if filtered.Len() != 1 {
		t.Errorf("Expected 1 device for initial state, got %d", filtered.Len())
	}

	if filtered.IsKnownDevice(device1) {
		t.Error("Offline device should not be included in initial state")
	}

	if !filtered.IsKnownDevice(device2) {
		t.Error("Online device should be included in initial state")
	}
}
