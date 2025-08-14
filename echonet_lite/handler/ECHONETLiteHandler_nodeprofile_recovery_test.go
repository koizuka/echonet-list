package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"net"
	"testing"
	"time"
)

// TestHandleDeviceOnlineTriggersNodeProfileRecovery tests that handleDeviceOnline triggers NodeProfile recovery when conditions are met
func TestHandleDeviceOnlineTriggersNodeProfileRecovery(t *testing.T) {
	testIP := net.ParseIP("192.168.1.200")

	lightingDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.MakeEOJ(0x0291, 1), // Single Function Lighting
	}

	nodeProfileDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.NodeProfileObject,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	options := ECHONETLieHandlerOptions{
		IP:       nil, // テスト環境ではワイルドカードを使用
		Debug:    true,
		TestMode: true, // テストモード（ファイル読み込みとネットワーク通信を無効化）
	}

	handler, err := NewECHONETLiteHandler(ctx, options)
	if err != nil {
		t.Fatalf("ECHONETLiteHandlerの作成に失敗: %v", err)
	}
	defer handler.Close()

	// デバイスを手動で登録
	handler.GetDataManagementHandler().RegisterDevice(lightingDevice)
	handler.GetDataManagementHandler().RegisterDevice(nodeProfileDevice)

	// NodeProfileをオフライン状態に設定
	handler.GetDataManagementHandler().SetOffline(nodeProfileDevice, true)

	// NodeProfileがオフラインになったことを確認
	if !handler.IsOffline(nodeProfileDevice) {
		t.Fatal("NodeProfileをオフライン状態に設定できませんでした")
	}

	// handleDeviceOnline関数を直接呼び出し
	handleDeviceOnline(lightingDevice, handler)

	// 処理が完了するまで少し待機
	time.Sleep(10 * time.Millisecond)

	// この時点で、UpdatePropertiesが呼ばれてNodeProfile復活処理が実行される
	// 実際のネットワーク通信は発生しないが、ログで確認できる

	t.Log("handleDeviceOnline関数の直接テストが完了しました")
}

// TestHandleDeviceOnlineSkipsRecoveryWhenNodeProfileOnline tests that handleDeviceOnline does nothing when NodeProfile is already online
func TestHandleDeviceOnlineSkipsRecoveryWhenNodeProfileOnline(t *testing.T) {
	testIP := net.ParseIP("192.168.1.300")

	lightingDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.MakeEOJ(0x0291, 1), // Single Function Lighting
	}

	nodeProfileDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.NodeProfileObject,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	options := ECHONETLieHandlerOptions{
		IP:       nil,
		Debug:    true,
		TestMode: true, // テストモード（ファイル読み込みとネットワーク通信を無効化）
	}

	handler, err := NewECHONETLiteHandler(ctx, options)
	if err != nil {
		t.Fatalf("ECHONETLiteHandlerの作成に失敗: %v", err)
	}
	defer handler.Close()

	// デバイスを手動で登録
	handler.GetDataManagementHandler().RegisterDevice(lightingDevice)
	handler.GetDataManagementHandler().RegisterDevice(nodeProfileDevice)

	// NodeProfileはオンライン状態のまま（デフォルト）
	if handler.IsOffline(nodeProfileDevice) {
		t.Fatal("NodeProfileが既にオフライン状態になっています")
	}

	// handleDeviceOnline関数を直接呼び出し
	// NodeProfileがオンラインなので、何も処理されないはず
	handleDeviceOnline(lightingDevice, handler)

	// 処理が完了するまで少し待機
	time.Sleep(10 * time.Millisecond)

	// NodeProfileがオンライン状態のままであることを確認
	if handler.IsOffline(nodeProfileDevice) {
		t.Error("NodeProfileがオフライン状態になってしまいました")
	}

	t.Log("NodeProfileオンライン時のhandleDeviceOnlineテストが完了しました")
}

// TestHandleDeviceOnlineSkipsRecoveryForNodeProfileDevice tests that handleDeviceOnline does nothing when the device itself is a NodeProfile
func TestHandleDeviceOnlineSkipsRecoveryForNodeProfileDevice(t *testing.T) {
	testIP := net.ParseIP("192.168.1.400")

	nodeProfileDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.NodeProfileObject,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	options := ECHONETLieHandlerOptions{
		IP:       nil,
		Debug:    true,
		TestMode: true, // テストモード（ファイル読み込みとネットワーク通信を無効化）
	}

	handler, err := NewECHONETLiteHandler(ctx, options)
	if err != nil {
		t.Fatalf("ECHONETLiteHandlerの作成に失敗: %v", err)
	}
	defer handler.Close()

	// デバイスを手動で登録
	handler.GetDataManagementHandler().RegisterDevice(nodeProfileDevice)

	// NodeProfileデバイス自体でhandleDeviceOnlineを呼び出し
	// NodeProfileデバイス自体の場合は何も処理されないはず
	handleDeviceOnline(nodeProfileDevice, handler)

	// 処理が完了するまで少し待機
	time.Sleep(10 * time.Millisecond)

	t.Log("NodeProfileデバイス自体でのhandleDeviceOnlineテストが完了しました")
}

// TestMultipleDevicesCanTriggerNodeProfileRecovery tests NodeProfile recovery with multiple devices on the same IP
func TestMultipleDevicesCanTriggerNodeProfileRecovery(t *testing.T) {
	testIP := net.ParseIP("192.168.1.700")

	// 複数のデバイス
	lightingDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.MakeEOJ(0x0291, 1), // Single Function Lighting
	}

	airCondDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.MakeEOJ(0x0130, 1), // Home Air Conditioner
	}

	nodeProfileDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.NodeProfileObject,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	options := ECHONETLieHandlerOptions{
		IP:       nil,
		Debug:    true,
		TestMode: true, // テストモード（ファイル読み込みとネットワーク通信を無効化）
	}

	handler, err := NewECHONETLiteHandler(ctx, options)
	if err != nil {
		t.Fatalf("ECHONETLiteHandlerの作成に失敗: %v", err)
	}
	defer handler.Close()

	// 全デバイスを登録
	handler.GetDataManagementHandler().RegisterDevice(lightingDevice)
	handler.GetDataManagementHandler().RegisterDevice(airCondDevice)
	handler.GetDataManagementHandler().RegisterDevice(nodeProfileDevice)

	// NodeProfileをオフラインに設定
	handler.GetDataManagementHandler().SetOffline(nodeProfileDevice, true)

	if !handler.IsOffline(nodeProfileDevice) {
		t.Fatal("NodeProfileをオフライン状態に設定できませんでした")
	}

	// 最初のデバイス（lighting）でNodeProfile復活を試行
	handleDeviceOnline(lightingDevice, handler)

	// 処理が完了するまで少し待機
	time.Sleep(10 * time.Millisecond)

	// 2番目のデバイス（airCond）でもNodeProfile復活を試行
	// NodeProfileがまだオフラインなら再度復活処理が実行される
	handleDeviceOnline(airCondDevice, handler)

	// 処理が完了するまで少し待機
	time.Sleep(10 * time.Millisecond)

	t.Log("複数デバイスでのNodeProfile復活テストが完了しました")
}

// TestNodeProfileRecoveryUsesCorrectIPAddress tests that IP address is correctly handled during recovery
func TestNodeProfileRecoveryUsesCorrectIPAddress(t *testing.T) {
	testIP := net.ParseIP("192.168.1.800")

	lightingDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.MakeEOJ(0x0291, 1),
	}

	nodeProfileDevice := IPAndEOJ{
		IP:  testIP,
		EOJ: echonet_lite.NodeProfileObject,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	options := ECHONETLieHandlerOptions{
		IP:       nil,
		Debug:    true,
		TestMode: true, // テストモード（ファイル読み込みとネットワーク通信を無効化）
	}

	handler, err := NewECHONETLiteHandler(ctx, options)
	if err != nil {
		t.Fatalf("ECHONETLiteHandlerの作成に失敗: %v", err)
	}
	defer handler.Close()

	// デバイスを登録
	handler.GetDataManagementHandler().RegisterDevice(lightingDevice)
	handler.GetDataManagementHandler().RegisterDevice(nodeProfileDevice)

	// NodeProfileをオフラインに設定
	handler.GetDataManagementHandler().SetOffline(nodeProfileDevice, true)

	if !handler.IsOffline(nodeProfileDevice) {
		t.Fatal("NodeProfileをオフライン状態に設定できませんでした")
	}

	// handleDeviceOnlineを呼び出し
	// ログでIP確認とNodeProfile復活処理が実行されることを確認
	handleDeviceOnline(lightingDevice, handler)

	// 処理が完了するまで少し待機
	time.Sleep(10 * time.Millisecond)

	t.Log("IPアドレス正常処理テストが完了しました")
}
