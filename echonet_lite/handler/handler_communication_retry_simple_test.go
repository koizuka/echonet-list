package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestActiveUpdatesTracking(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// テスト用のCommunicationHandlerを作成（依存関係はnilで問題ない）
	handler := &CommunicationHandler{
		ctx:           ctx,
		activeUpdates: make(map[string]time.Time),
	}

	// バックグラウンドクリーンアップを開始
	go handler.startActiveUpdatesCleanup()

	// 少し待ってバックグラウンドクリーンアップが開始されることを確認
	time.Sleep(10 * time.Millisecond)

	ip := net.ParseIP("192.168.1.100")
	device := IPAndEOJ{IP: ip, EOJ: echonet_lite.MakeEOJ(0x0291, 0x01)}

	t.Run("basic active update tracking", func(t *testing.T) {
		// 最初はアクティブではない
		assert.False(t, handler.isUpdateActive(device, false))

		// アクティブにマーク
		handler.markUpdateActive(device)

		// アクティブになったことを確認
		assert.True(t, handler.isUpdateActive(device, false))

		// 非アクティブにマーク
		handler.markUpdateInactive(device)

		// 非アクティブになったことを確認
		assert.False(t, handler.isUpdateActive(device, false))
	})

	t.Run("force flag bypasses active check", func(t *testing.T) {
		// アクティブにマーク
		handler.markUpdateActive(device)

		// force=falseの場合はアクティブ
		assert.True(t, handler.isUpdateActive(device, false))

		// force=trueの場合は非アクティブとして扱われる
		assert.False(t, handler.isUpdateActive(device, true))

		// クリーンアップ
		handler.markUpdateInactive(device)
	})

	t.Run("old entries are cleaned up automatically", func(t *testing.T) {
		// アクティブにマーク
		handler.markUpdateActive(device)
		assert.True(t, handler.isUpdateActive(device, false))

		// 古いエントリを手動で設定（テスト用）
		handler.activeUpdatesMu.Lock()
		deviceKey := fmt.Sprintf("%s:%04X:%02X", device.IP.String(), uint16(device.EOJ.ClassCode()), device.EOJ.InstanceCode())
		handler.activeUpdates[deviceKey] = time.Now().Add(-MaxUpdateAge - time.Minute)
		handler.activeUpdatesMu.Unlock()

		// 古いエントリは自動的にクリーンアップされる
		assert.False(t, handler.isUpdateActive(device, false))
	})

	t.Run("concurrent access safety", func(t *testing.T) {
		const numGoroutines = 10
		const iterations = 100

		var wg sync.WaitGroup

		// 複数のgoroutineで同時にアクセス
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				testIP := net.ParseIP(fmt.Sprintf("192.168.1.%d", 100+id))
				testDevice := IPAndEOJ{IP: testIP, EOJ: echonet_lite.MakeEOJ(0x0291, 0x01)}

				for j := 0; j < iterations; j++ {
					handler.markUpdateActive(testDevice)
					handler.isUpdateActive(testDevice, false)
					handler.markUpdateInactive(testDevice)
				}
			}(i)
		}

		wg.Wait()
		// パニックが発生しないことを確認（暗黙的にテスト）
	})

	t.Run("background cleanup works", func(t *testing.T) {
		// 複数のデバイスをアクティブにマーク
		devices := []IPAndEOJ{
			{IP: net.ParseIP("192.168.1.101"), EOJ: echonet_lite.MakeEOJ(0x0291, 0x01)},
			{IP: net.ParseIP("192.168.1.102"), EOJ: echonet_lite.MakeEOJ(0x0291, 0x01)},
			{IP: net.ParseIP("192.168.1.103"), EOJ: echonet_lite.MakeEOJ(0x0291, 0x01)},
		}

		for _, device := range devices {
			handler.markUpdateActive(device)
		}

		// 一部を古いエントリに設定
		handler.activeUpdatesMu.Lock()
		deviceKey0 := fmt.Sprintf("%s:%04X:%02X", devices[0].IP.String(), uint16(devices[0].EOJ.ClassCode()), devices[0].EOJ.InstanceCode())
		deviceKey1 := fmt.Sprintf("%s:%04X:%02X", devices[1].IP.String(), uint16(devices[1].EOJ.ClassCode()), devices[1].EOJ.InstanceCode())
		handler.activeUpdates[deviceKey0] = time.Now().Add(-MaxUpdateAge - time.Minute)
		handler.activeUpdates[deviceKey1] = time.Now().Add(-MaxUpdateAge - time.Minute)
		handler.activeUpdatesMu.Unlock()

		// バックグラウンドクリーンアップを手動実行
		handler.cleanupStaleActiveUpdates()

		// 古いエントリが削除され、新しいエントリは残っていることを確認
		assert.False(t, handler.isUpdateActive(devices[0], false))
		assert.False(t, handler.isUpdateActive(devices[1], false))
		assert.True(t, handler.isUpdateActive(devices[2], false))

		// クリーンアップ
		handler.markUpdateInactive(devices[2])
	})

	t.Run("double-checked locking works correctly", func(t *testing.T) {
		testDevice := IPAndEOJ{IP: net.ParseIP("192.168.1.200"), EOJ: echonet_lite.MakeEOJ(0x0291, 0x01)}

		// アクティブにマーク
		handler.markUpdateActive(testDevice)
		assert.True(t, handler.isUpdateActive(testDevice, false))

		// 古いエントリを設定
		handler.activeUpdatesMu.Lock()
		deviceKey := fmt.Sprintf("%s:%04X:%02X", testDevice.IP.String(), uint16(testDevice.EOJ.ClassCode()), testDevice.EOJ.InstanceCode())
		handler.activeUpdates[deviceKey] = time.Now().Add(-MaxUpdateAge - time.Minute)
		handler.activeUpdatesMu.Unlock()

		// isUpdateActiveを呼び出すと、double-checked lockingによって古いエントリが削除される
		assert.False(t, handler.isUpdateActive(testDevice, false))

		// エントリが削除されていることを確認
		handler.activeUpdatesMu.RLock()
		_, exists := handler.activeUpdates[deviceKey]
		handler.activeUpdatesMu.RUnlock()
		assert.False(t, exists)
	})
}
