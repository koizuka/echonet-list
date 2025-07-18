package handler

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// MonitoringManager は、システムリソースの監視を行う
type MonitoringManager struct {
	ctx      context.Context
	cancel   context.CancelFunc
	interval time.Duration
	config   MonitoringConfig
}

// NewMonitoringManager は、新しい監視マネージャーを作成する
func NewMonitoringManager(ctx context.Context, interval time.Duration) *MonitoringManager {
	monitorCtx, cancel := context.WithCancel(ctx)
	return &MonitoringManager{
		ctx:      monitorCtx,
		cancel:   cancel,
		interval: interval,
		config:   DefaultMonitoringConfig(),
	}
}

// NewMonitoringManagerWithConfig は、設定付きの監視マネージャーを作成する
func NewMonitoringManagerWithConfig(ctx context.Context, interval time.Duration, config MonitoringConfig) *MonitoringManager {
	monitorCtx, cancel := context.WithCancel(ctx)
	return &MonitoringManager{
		ctx:      monitorCtx,
		cancel:   cancel,
		interval: interval,
		config:   config,
	}
}

// Start は、監視を開始する
func (m *MonitoringManager) Start() {
	slog.Info("Starting monitoring", "interval", m.interval)

	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.collectMetrics()
			case <-m.ctx.Done():
				slog.Info("Monitoring stopped")
				return
			}
		}
	}()
}

// Stop は、監視を停止する
func (m *MonitoringManager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// collectMetrics は、システムメトリクスを収集・記録する
func (m *MonitoringManager) collectMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Goroutine数をチェック
	goroutineCount := runtime.NumGoroutine()

	// メモリ使用量（MB単位）
	allocMB := float64(memStats.Alloc) / 1024 / 1024
	sysMB := float64(memStats.Sys) / 1024 / 1024

	slog.Info("System metrics",
		"goroutines", goroutineCount,
		"memory_alloc_mb", allocMB,
		"memory_sys_mb", sysMB,
		"gc_cycles", memStats.NumGC,
	)

	// 異常な値の検出
	if goroutineCount > m.config.GoroutineThreshold {
		slog.Warn("High goroutine count detected", "count", goroutineCount, "threshold", m.config.GoroutineThreshold)
	}

	if allocMB > m.config.MemoryThresholdMB {
		slog.Warn("High memory allocation detected", "alloc_mb", allocMB, "threshold_mb", m.config.MemoryThresholdMB)
	}
}

// ChannelMonitor は、チャンネルのバッファ使用率を監視する
type ChannelMonitor struct {
	name      string
	capacity  int
	lenFunc   func() int
	threshold float64
}

// NewChannelMonitor は、新しいチャンネル監視を作成する
func NewChannelMonitor(name string, capacity int, lenFunc func() int) *ChannelMonitor {
	return &ChannelMonitor{
		name:      name,
		capacity:  capacity,
		lenFunc:   lenFunc,
		threshold: 80.0, // デフォルト閾値
	}
}

// NewChannelMonitorWithThreshold は、閾値指定でチャンネル監視を作成する
func NewChannelMonitorWithThreshold(name string, capacity int, lenFunc func() int, threshold float64) *ChannelMonitor {
	return &ChannelMonitor{
		name:      name,
		capacity:  capacity,
		lenFunc:   lenFunc,
		threshold: threshold,
	}
}

// CheckUsage は、チャンネルの使用率をチェックする
func (cm *ChannelMonitor) CheckUsage() {
	if cm.lenFunc == nil {
		return
	}

	currentLen := cm.lenFunc()
	usagePercent := float64(currentLen) / float64(cm.capacity) * 100

	slog.Debug("Channel usage",
		"name", cm.name,
		"current", currentLen,
		"capacity", cm.capacity,
		"usage_percent", usagePercent,
	)

	// 高い使用率を警告
	if usagePercent > cm.threshold {
		slog.Warn("High channel buffer usage",
			"name", cm.name,
			"current", currentLen,
			"capacity", cm.capacity,
			"usage_percent", usagePercent,
			"threshold", cm.threshold,
		)
	}
}

// ChannelMonitorManager は、複数のチャンネル監視を管理する
type ChannelMonitorManager struct {
	monitors []ChannelMonitor
}

// NewChannelMonitorManager は、新しいチャンネル監視マネージャーを作成する
func NewChannelMonitorManager() *ChannelMonitorManager {
	return &ChannelMonitorManager{
		monitors: make([]ChannelMonitor, 0),
	}
}

// AddMonitor は、チャンネル監視を追加する
func (cmm *ChannelMonitorManager) AddMonitor(monitor ChannelMonitor) {
	cmm.monitors = append(cmm.monitors, monitor)
}

// CheckAll は、すべてのチャンネル監視をチェックする
func (cmm *ChannelMonitorManager) CheckAll() {
	for _, monitor := range cmm.monitors {
		monitor.CheckUsage()
	}
}
