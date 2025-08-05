package handler

import (
	"context"
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

	t.Run("basic active update tracking", func(t *testing.T) {
		// 最初はアクティブではない
		assert.False(t, handler.isUpdateActive(ip, false))

		// アクティブにマーク
		handler.markUpdateActive(ip)

		// アクティブになったことを確認
		assert.True(t, handler.isUpdateActive(ip, false))

		// 非アクティブにマーク
		handler.markUpdateInactive(ip)

		// 非アクティブになったことを確認
		assert.False(t, handler.isUpdateActive(ip, false))
	})

	t.Run("force flag bypasses active check", func(t *testing.T) {
		// アクティブにマーク
		handler.markUpdateActive(ip)

		// force=falseの場合はアクティブ
		assert.True(t, handler.isUpdateActive(ip, false))

		// force=trueの場合は非アクティブとして扱われる
		assert.False(t, handler.isUpdateActive(ip, true))

		// クリーンアップ
		handler.markUpdateInactive(ip)
	})

	t.Run("old entries are cleaned up automatically", func(t *testing.T) {
		// アクティブにマーク
		handler.markUpdateActive(ip)
		assert.True(t, handler.isUpdateActive(ip, false))

		// 古いエントリを手動で設定（テスト用）
		handler.activeUpdatesMu.Lock()
		handler.activeUpdates[ip.String()] = time.Now().Add(-MaxUpdateAge - time.Minute)
		handler.activeUpdatesMu.Unlock()

		// 古いエントリは自動的にクリーンアップされる
		assert.False(t, handler.isUpdateActive(ip, false))
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

				for j := 0; j < iterations; j++ {
					handler.markUpdateActive(testIP)
					handler.isUpdateActive(testIP, false)
					handler.markUpdateInactive(testIP)
				}
			}(i)
		}

		wg.Wait()
		// パニックが発生しないことを確認（暗黙的にテスト）
	})

	t.Run("background cleanup works", func(t *testing.T) {
		// 複数のIPをアクティブにマーク
		ips := []net.IP{
			net.ParseIP("192.168.1.101"),
			net.ParseIP("192.168.1.102"),
			net.ParseIP("192.168.1.103"),
		}

		for _, ip := range ips {
			handler.markUpdateActive(ip)
		}

		// 一部を古いエントリに設定
		handler.activeUpdatesMu.Lock()
		handler.activeUpdates[ips[0].String()] = time.Now().Add(-MaxUpdateAge - time.Minute)
		handler.activeUpdates[ips[1].String()] = time.Now().Add(-MaxUpdateAge - time.Minute)
		handler.activeUpdatesMu.Unlock()

		// バックグラウンドクリーンアップを手動実行
		handler.cleanupStaleActiveUpdates()

		// 古いエントリが削除され、新しいエントリは残っていることを確認
		assert.False(t, handler.isUpdateActive(ips[0], false))
		assert.False(t, handler.isUpdateActive(ips[1], false))
		assert.True(t, handler.isUpdateActive(ips[2], false))

		// クリーンアップ
		handler.markUpdateInactive(ips[2])
	})

	t.Run("double-checked locking works correctly", func(t *testing.T) {
		testIP := net.ParseIP("192.168.1.200")

		// アクティブにマーク
		handler.markUpdateActive(testIP)
		assert.True(t, handler.isUpdateActive(testIP, false))

		// 古いエントリを設定
		handler.activeUpdatesMu.Lock()
		handler.activeUpdates[testIP.String()] = time.Now().Add(-MaxUpdateAge - time.Minute)
		handler.activeUpdatesMu.Unlock()

		// isUpdateActiveを呼び出すと、double-checked lockingによって古いエントリが削除される
		assert.False(t, handler.isUpdateActive(testIP, false))

		// エントリが削除されていることを確認
		handler.activeUpdatesMu.RLock()
		_, exists := handler.activeUpdates[testIP.String()]
		handler.activeUpdatesMu.RUnlock()
		assert.False(t, exists)
	})
}
