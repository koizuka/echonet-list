package handler

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"echonet-list/echonet_lite"
)

func TestFanoutNotifications_DisconnectsFullBufferSubscribers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core := NewHandlerCore(ctx, cancel, false)
	defer core.Close()

	// バッファサイズ1の購読者を2つ作成
	subscriber1 := core.SubscribeNotifications(1)
	subscriber2 := core.SubscribeNotifications(1)

	// subscriber1のバッファを埋める（読み取らない）
	testDevice := IPAndEOJ{
		IP:  net.ParseIP("192.168.0.1"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 1),
	}
	notification1 := DeviceNotification{
		Device: testDevice,
		Type:   DeviceAdded,
	}

	// 最初の通知を送信（両方のsubscriberに届く）
	core.notify(notification1)

	// subscriber1から読み取らずにバッファをフルにしておく
	// subscriber2からは読み取る
	select {
	case <-subscriber2:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber2 should have received notification1")
	}

	// 2番目の通知を送信
	// subscriber1はバッファフルなので切断されるはず
	notification2 := DeviceNotification{
		Device: testDevice,
		Type:   DeviceTimeout,
	}
	core.notify(notification2)

	// subscriber1は切断されているので、チャネルが閉じられているはず
	// 少し待ってからチェック
	time.Sleep(50 * time.Millisecond)

	// subscriber2は2番目の通知を受信できるはず
	select {
	case n := <-subscriber2:
		if n.Type != DeviceTimeout {
			t.Errorf("Expected DeviceTimeout, got %v", n.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber2 should have received notification2")
	}

	// subscriber1は閉じられているはず（最初の通知は読み取り可能だが、その後閉じられる）
	// まず最初の通知を読み取る
	select {
	case <-subscriber1:
		// OK - 最初の通知
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber1 should have the first notification in buffer")
	}

	// 次の読み取りでチャネルが閉じられていることを確認
	select {
	case _, ok := <-subscriber1:
		if ok {
			t.Error("subscriber1 should be closed after buffer full")
		}
		// チャネルが閉じられている（ok == false）- 期待通り
	case <-time.After(100 * time.Millisecond):
		t.Fatal("subscriber1 channel should be closed")
	}
}

func TestFanoutNotifications_ActiveSubscribersReceiveNotifications(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core := NewHandlerCore(ctx, cancel, false)
	defer core.Close()

	// バッファサイズ10の購読者を作成
	subscriber := core.SubscribeNotifications(10)

	testDevice := IPAndEOJ{
		IP:  net.ParseIP("192.168.0.1"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 1),
	}

	// 複数の通知を送信
	notificationTypes := []NotificationType{DeviceAdded, DeviceTimeout, DeviceOffline, DeviceOnline}
	for _, notifType := range notificationTypes {
		core.notify(DeviceNotification{
			Device: testDevice,
			Type:   notifType,
		})
	}

	// すべての通知を受信できることを確認
	for i, expectedType := range notificationTypes {
		select {
		case n := <-subscriber:
			if n.Type != expectedType {
				t.Errorf("Notification %d: Expected %v, got %v", i, expectedType, n.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Failed to receive notification %d", i)
		}
	}
}

func TestSubscribeNotifications_MultipleSubscribers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core := NewHandlerCore(ctx, cancel, false)
	defer core.Close()

	// 3つの購読者を作成
	subscriber1 := core.SubscribeNotifications(10)
	subscriber2 := core.SubscribeNotifications(10)
	subscriber3 := core.SubscribeNotifications(10)

	testDevice := IPAndEOJ{
		IP:  net.ParseIP("192.168.0.1"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 1),
	}

	// 通知を送信
	core.notify(DeviceNotification{
		Device: testDevice,
		Type:   DeviceAdded,
	})

	// すべての購読者が通知を受信できることを確認
	subscribers := []<-chan DeviceNotification{subscriber1, subscriber2, subscriber3}
	for i, sub := range subscribers {
		select {
		case n := <-sub:
			if n.Type != DeviceAdded {
				t.Errorf("Subscriber %d: Expected DeviceAdded, got %v", i+1, n.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Subscriber %d failed to receive notification", i+1)
		}
	}
}

// mockOfflineChecker はテスト用のOfflineChecker実装（スレッドセーフ）
type mockOfflineChecker struct {
	mu             sync.RWMutex
	offlineDevices map[string]bool
}

func newMockOfflineChecker() *mockOfflineChecker {
	return &mockOfflineChecker{
		offlineDevices: make(map[string]bool),
	}
}

func (m *mockOfflineChecker) IsOffline(device IPAndEOJ) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.offlineDevices[device.Key()]
}

func (m *mockOfflineChecker) setOffline(device IPAndEOJ, offline bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.offlineDevices[device.Key()] = offline
}

func TestRelaySessionTimeoutEvent_SkipsOfflineDevices(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core := NewHandlerCore(ctx, cancel, false)
	defer core.Close()

	// モックオフラインチェッカーを設定
	checker := newMockOfflineChecker()
	core.SetOfflineChecker(checker)

	// 購読者を作成
	subscriber := core.SubscribeNotifications(10)

	testDevice := IPAndEOJ{
		IP:  net.ParseIP("192.168.0.1"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 1),
	}

	// デバイスがオンラインの状態でタイムアウトイベントを送信 - 通知されるはず
	core.RelaySessionTimeoutEvent(SessionTimeoutEvent{
		Device: testDevice,
		Type:   SessionTimeoutMaxRetries,
	})

	select {
	case n := <-subscriber:
		if n.Type != DeviceTimeout {
			t.Errorf("Expected DeviceTimeout, got %v", n.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected DeviceTimeout notification for online device")
	}

	// デバイスをオフラインに設定
	checker.setOffline(testDevice, true)

	// オフラインのデバイスにタイムアウトイベントを送信 - 通知されないはず
	core.RelaySessionTimeoutEvent(SessionTimeoutEvent{
		Device: testDevice,
		Type:   SessionTimeoutMaxRetries,
	})

	select {
	case n := <-subscriber:
		t.Errorf("Expected no notification for offline device, but got %v", n.Type)
	case <-time.After(100 * time.Millisecond):
		// 期待される動作: 通知なし
	}
}

func TestRelaySessionTimeoutEvent_WorksWithoutOfflineChecker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core := NewHandlerCore(ctx, cancel, false)
	defer core.Close()

	// オフラインチェッカーを設定しない（nil）

	// 購読者を作成
	subscriber := core.SubscribeNotifications(10)

	testDevice := IPAndEOJ{
		IP:  net.ParseIP("192.168.0.1"),
		EOJ: echonet_lite.MakeEOJ(0x0130, 1),
	}

	// タイムアウトイベントを送信 - チェッカーがなくても通知されるはず
	core.RelaySessionTimeoutEvent(SessionTimeoutEvent{
		Device: testDevice,
		Type:   SessionTimeoutMaxRetries,
	})

	select {
	case n := <-subscriber:
		if n.Type != DeviceTimeout {
			t.Errorf("Expected DeviceTimeout, got %v", n.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected DeviceTimeout notification even without offline checker")
	}
}

func TestRelaySessionTimeoutEvent_ConcurrentAccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core := NewHandlerCore(ctx, cancel, false)
	defer core.Close()

	checker := newMockOfflineChecker()
	core.SetOfflineChecker(checker)

	// 購読者を作成（大きなバッファで通知を受け取る）
	subscriber := core.SubscribeNotifications(1000)

	// 複数のデバイスを作成
	devices := make([]IPAndEOJ, 10)
	for i := 0; i < 10; i++ {
		devices[i] = IPAndEOJ{
			IP:  net.ParseIP("192.168.0.1"),
			EOJ: echonet_lite.MakeEOJ(0x0130, echonet_lite.EOJInstanceCode(i+1)),
		}
	}

	// 複数のゴルーチンから同時にタイムアウトイベントを送信
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				core.RelaySessionTimeoutEvent(SessionTimeoutEvent{
					Device: devices[idx],
					Type:   SessionTimeoutMaxRetries,
				})
				// オフライン状態を切り替え
				if j%2 == 0 {
					checker.setOffline(devices[idx], true)
				} else {
					checker.setOffline(devices[idx], false)
				}
			}
			done <- struct{}{}
		}(i)
	}

	// 全ゴルーチンの完了を待つ
	for i := 0; i < 10; i++ {
		<-done
	}

	// 通知を消費（パニックが発生しないことを確認）
	consumeCount := 0
	for {
		select {
		case <-subscriber:
			consumeCount++
		case <-time.After(100 * time.Millisecond):
			// タイムアウト - これ以上通知がない
			t.Logf("Received %d notifications without panic (concurrent access test passed)", consumeCount)
			return
		}
	}
}
