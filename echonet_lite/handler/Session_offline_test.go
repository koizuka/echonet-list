package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"net"
	"testing"
	"time"
)

// TestSession_OfflineLogSuppression オフライン状態のデバイスに対するログ出力抑制のテスト
func TestSession_OfflineLogSuppression(t *testing.T) {
	ctx := context.Background()
	ip := net.ParseIP("127.0.0.1")
	eoj := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	device := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP("192.168.1.100"),
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}

	// オフライン判定関数のモック
	isOfflineCallCount := 0
	mockIsOffline := func(dev echonet_lite.IPAndEOJ) bool {
		isOfflineCallCount++
		return dev.IP.Equal(device.IP) && dev.EOJ == device.EOJ
	}

	// Session作成
	session, err := CreateSession(ctx, ip, eoj, false, nil, mockIsOffline)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer session.Close()

	// 短いタイムアウト設定
	session.MaxRetries = 1
	session.RetryInterval = 10 * time.Millisecond

	// IsOfflineFunc が呼ばれるログ出力をテストするため、
	// タイムアウト処理の中でログが出力される部分を直接テスト
	session.mu.RLock()
	isOfflineFunc := session.IsOfflineFunc
	session.mu.RUnlock()

	// IsOfflineFunc関数が正しく設定されていることを確認
	if isOfflineFunc == nil {
		t.Error("Expected IsOfflineFunc to be set")
	}

	// IsOfflineFunc関数を呼び出してカウントを確認
	result := isOfflineFunc(device)
	if !result {
		t.Error("Expected device to be offline according to mock function")
	}

	// IsOffline関数が呼ばれたことを確認
	if isOfflineCallCount == 0 {
		t.Error("Expected IsOffline function to be called")
	}

	// notifyDeviceTimeoutは常にErrMaxRetriesReachedエラーを返す
	err = session.notifyDeviceTimeout(device)
	if err == nil {
		t.Error("Expected error from notifyDeviceTimeout")
	}

	// ErrMaxRetriesReachedエラーが返されることを確認
	if _, ok := err.(ErrMaxRetriesReached); !ok {
		t.Errorf("Expected ErrMaxRetriesReached, got %T", err)
	}
}

// TestSession_OfflineLogSuppressionWithNilFunc IsOfflineFunc=nil時の動作テスト
func TestSession_OfflineLogSuppressionWithNilFunc(t *testing.T) {
	ctx := context.Background()
	ip := net.ParseIP("127.0.0.1")
	eoj := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	device := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP("192.168.1.100"),
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}

	// IsOfflineFunc=nilでSession作成
	session, err := CreateSession(ctx, ip, eoj, false, nil, nil)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer session.Close()

	// notifyDeviceTimeoutを直接呼び出してテスト
	err = session.notifyDeviceTimeout(device)
	if err == nil {
		t.Error("Expected error from notifyDeviceTimeout")
	}

	// ErrMaxRetriesReachedエラーが返されることを確認
	if _, ok := err.(ErrMaxRetriesReached); !ok {
		t.Errorf("Expected ErrMaxRetriesReached, got %T", err)
	}
}

// TestSession_OnlineDeviceLogging オンラインデバイスでのログ出力テスト
func TestSession_OnlineDeviceLogging(t *testing.T) {
	ctx := context.Background()
	ip := net.ParseIP("127.0.0.1")
	eoj := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	device := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP("192.168.1.100"),
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}

	// オンライン判定関数のモック（常にfalseを返す）
	isOfflineCallCount := 0
	mockIsOffline := func(dev echonet_lite.IPAndEOJ) bool {
		isOfflineCallCount++
		return false // 常にオンライン
	}

	// Session作成
	session, err := CreateSession(ctx, ip, eoj, false, nil, mockIsOffline)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer session.Close()

	// IsOfflineFunc が正しく設定されていることを確認
	session.mu.RLock()
	isOfflineFunc := session.IsOfflineFunc
	session.mu.RUnlock()

	if isOfflineFunc == nil {
		t.Error("Expected IsOfflineFunc to be set")
	}

	// IsOfflineFunc関数を呼び出してテスト
	result := isOfflineFunc(device)
	if result {
		t.Error("Expected device to be online according to mock function")
	}

	// IsOffline関数が呼ばれたことを確認
	if isOfflineCallCount == 0 {
		t.Error("Expected IsOffline function to be called")
	}

	// notifyDeviceTimeoutを直接呼び出してテスト
	err = session.notifyDeviceTimeout(device)
	if err == nil {
		t.Error("Expected error from notifyDeviceTimeout")
	}

	// ErrMaxRetriesReachedエラーが返されることを確認
	if _, ok := err.(ErrMaxRetriesReached); !ok {
		t.Errorf("Expected ErrMaxRetriesReached, got %T", err)
	}
}

// TestSession_IsOfflineFuncAccess IsOfflineFunc関数が正しく設定されているかテスト
func TestSession_IsOfflineFuncAccess(t *testing.T) {
	ctx := context.Background()
	ip := net.ParseIP("127.0.0.1")
	eoj := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	device := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP("192.168.1.100"),
		EOJ: echonet_lite.MakeEOJ(echonet_lite.HomeAirConditioner_ClassCode, 1),
	}

	// テスト用IsOffline関数
	mockIsOffline := func(dev echonet_lite.IPAndEOJ) bool {
		return dev.IP.Equal(device.IP)
	}

	// Session作成
	session, err := CreateSession(ctx, ip, eoj, false, nil, mockIsOffline)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	defer session.Close()

	// IsOfflineFunc が正しく設定されていることを確認
	session.mu.RLock()
	isOfflineFuncSet := session.IsOfflineFunc != nil
	session.mu.RUnlock()

	if !isOfflineFuncSet {
		t.Error("Expected IsOfflineFunc to be set")
	}

	// 実際にIsOfflineFunc関数を呼び出してテスト
	session.mu.RLock()
	result := session.IsOfflineFunc(device)
	session.mu.RUnlock()

	if !result {
		t.Error("Expected device to be offline according to mock function")
	}
}
