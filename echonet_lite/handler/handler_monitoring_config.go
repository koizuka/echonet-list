package handler

// MonitoringConfig は監視システムの設定を定義する
type MonitoringConfig struct {
	// Goroutine数の警告閾値
	GoroutineThreshold int
	// メモリ使用量の警告閾値 (MB)
	MemoryThresholdMB float64
	// チャンネルバッファ使用率の警告閾値 (%)
	ChannelUsageThreshold float64
}

// DefaultMonitoringConfig はデフォルトの監視設定を返す
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		GoroutineThreshold:    1000,
		MemoryThresholdMB:     1000,
		ChannelUsageThreshold: 80.0,
	}
}