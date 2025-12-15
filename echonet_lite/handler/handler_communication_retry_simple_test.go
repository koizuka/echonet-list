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
		activeUpdates: make(map[string]*activeUpdateEntry),
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

		// アクティブにマーク（テストではキャンセル関数はnilで良い）
		handler.markUpdateActive(device, nil)

		// アクティブになったことを確認
		assert.True(t, handler.isUpdateActive(device, false))

		// 非アクティブにマーク
		handler.markUpdateInactive(device)

		// 非アクティブになったことを確認
		assert.False(t, handler.isUpdateActive(device, false))
	})

	t.Run("force flag cancels existing update and bypasses active check", func(t *testing.T) {
		// アクティブにマーク
		handler.markUpdateActive(device, nil)

		// force=falseの場合はアクティブ
		assert.True(t, handler.isUpdateActive(device, false))

		// force=trueの場合は既存の更新をキャンセルして非アクティブとして扱われる
		assert.False(t, handler.isUpdateActive(device, true))

		// force=trueでキャンセル後はエントリが削除されている
		assert.False(t, handler.isUpdateActive(device, false))
	})

	t.Run("old entries are cleaned up automatically", func(t *testing.T) {
		// アクティブにマーク
		handler.markUpdateActive(device, nil)
		assert.True(t, handler.isUpdateActive(device, false))

		// 古いエントリを手動で設定（テスト用）
		handler.activeUpdatesMu.Lock()
		deviceKey := makeDeviceKey(device)
		handler.activeUpdates[deviceKey] = &activeUpdateEntry{
			startTime: time.Now().Add(-MaxUpdateAge - time.Minute),
			cancel:    nil,
		}
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
					handler.markUpdateActive(testDevice, nil)
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
			handler.markUpdateActive(device, nil)
		}

		// 一部を古いエントリに設定
		handler.activeUpdatesMu.Lock()
		deviceKey0 := makeDeviceKey(devices[0])
		deviceKey1 := makeDeviceKey(devices[1])
		handler.activeUpdates[deviceKey0] = &activeUpdateEntry{
			startTime: time.Now().Add(-MaxUpdateAge - time.Minute),
			cancel:    nil,
		}
		handler.activeUpdates[deviceKey1] = &activeUpdateEntry{
			startTime: time.Now().Add(-MaxUpdateAge - time.Minute),
			cancel:    nil,
		}
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
		handler.markUpdateActive(testDevice, nil)
		assert.True(t, handler.isUpdateActive(testDevice, false))

		// 古いエントリを設定
		handler.activeUpdatesMu.Lock()
		deviceKey := makeDeviceKey(testDevice)
		handler.activeUpdates[deviceKey] = &activeUpdateEntry{
			startTime: time.Now().Add(-MaxUpdateAge - time.Minute),
			cancel:    nil,
		}
		handler.activeUpdatesMu.Unlock()

		// isUpdateActiveを呼び出すと、double-checked lockingによって古いエントリが削除される
		assert.False(t, handler.isUpdateActive(testDevice, false))

		// エントリが削除されていることを確認
		handler.activeUpdatesMu.RLock()
		_, exists := handler.activeUpdates[deviceKey]
		handler.activeUpdatesMu.RUnlock()
		assert.False(t, exists)
	})

	t.Run("force flag calls cancel function", func(t *testing.T) {
		testDevice := IPAndEOJ{IP: net.ParseIP("192.168.1.201"), EOJ: echonet_lite.MakeEOJ(0x0291, 0x01)}

		// キャンセルが呼ばれたかどうかを追跡
		cancelCalled := false
		testCancel := func() {
			cancelCalled = true
		}

		// アクティブにマーク（キャンセル関数付き）
		handler.markUpdateActive(testDevice, testCancel)
		assert.True(t, handler.isUpdateActive(testDevice, false))

		// force=trueでキャンセルが呼ばれることを確認
		assert.False(t, handler.isUpdateActive(testDevice, true))
		assert.True(t, cancelCalled)

		// エントリが削除されていることを確認
		assert.False(t, handler.isUpdateActive(testDevice, false))
	})
}
