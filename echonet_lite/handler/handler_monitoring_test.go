package handler

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestMonitoringManager_Start_Stop(t *testing.T) {
	ctx := context.Background()
	monitor := NewMonitoringManager(ctx, 10*time.Millisecond)
	
	// 監視を開始
	monitor.Start()
	
	// 少し待つ
	time.Sleep(50 * time.Millisecond)
	
	// 監視を停止
	monitor.Stop()
	
	// 停止後は新しいメトリクスが収集されないことを確認
	// （実際のテストでは、ログの確認が必要）
}

func TestChannelMonitor_CheckUsage(t *testing.T) {
	// モックチャンネルを作成
	ch := make(chan int, 10)
	
	// 5つの要素を追加
	for i := 0; i < 5; i++ {
		ch <- i
	}
	
	monitor := NewChannelMonitor("test_channel", 10, func() int {
		return len(ch)
	})
	
	// 使用率チェック（50%使用中）
	monitor.CheckUsage()
	
	// 高い使用率をテスト
	for i := 0; i < 4; i++ {
		ch <- i
	}
	
	// 使用率チェック（90%使用中）
	monitor.CheckUsage()
}

func TestChannelMonitorManager_AddMonitor(t *testing.T) {
	manager := NewChannelMonitorManager()
	
	ch1 := make(chan int, 10)
	ch2 := make(chan string, 20)
	
	monitor1 := NewChannelMonitor("channel1", 10, func() int { return len(ch1) })
	monitor2 := NewChannelMonitor("channel2", 20, func() int { return len(ch2) })
	
	manager.AddMonitor(*monitor1)
	manager.AddMonitor(*monitor2)
	
	if len(manager.monitors) != 2 {
		t.Errorf("Expected 2 monitors, got %d", len(manager.monitors))
	}
}

func TestChannelMonitorManager_CheckAll(t *testing.T) {
	manager := NewChannelMonitorManager()
	
	ch1 := make(chan int, 10)
	ch2 := make(chan string, 20)
	
	// チャンネルにデータを追加
	ch1 <- 1
	ch2 <- "test"
	
	monitor1 := NewChannelMonitor("channel1", 10, func() int { return len(ch1) })
	monitor2 := NewChannelMonitor("channel2", 20, func() int { return len(ch2) })
	
	manager.AddMonitor(*monitor1)
	manager.AddMonitor(*monitor2)
	
	// すべてのチャンネルをチェック
	manager.CheckAll()
}

func TestMonitoringManager_collectMetrics(t *testing.T) {
	ctx := context.Background()
	monitor := NewMonitoringManager(ctx, 1*time.Second)
	
	// 現在のGoroutine数を記録
	initialGoroutines := runtime.NumGoroutine()
	
	// メトリクス収集を実行
	monitor.collectMetrics()
	
	// Goroutine数が増えていないことを確認
	currentGoroutines := runtime.NumGoroutine()
	if currentGoroutines > initialGoroutines+1 {
		t.Errorf("Goroutine count increased unexpectedly: %d -> %d", initialGoroutines, currentGoroutines)
	}
}