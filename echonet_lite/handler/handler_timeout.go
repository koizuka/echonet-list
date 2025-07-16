package handler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// TimeoutManager は、操作のタイムアウト管理を行う
type TimeoutManager struct {
	// 各操作のタイムアウト時間
	DiscoveryTimeout        time.Duration
	PropertyUpdateTimeout   time.Duration
	PropertyGetTimeout      time.Duration
	PropertySetTimeout      time.Duration
}

// DefaultTimeoutManager はデフォルトのタイムアウト設定を返す
func DefaultTimeoutManager() *TimeoutManager {
	return &TimeoutManager{
		DiscoveryTimeout:        30 * time.Second,
		PropertyUpdateTimeout:   60 * time.Second,
		PropertyGetTimeout:      10 * time.Second,
		PropertySetTimeout:      10 * time.Second,
	}
}

// WithTimeout は、指定された操作にタイムアウト制御を追加する
func (tm *TimeoutManager) WithTimeout(ctx context.Context, operation string, timeout time.Duration, fn func() error) error {
	slog.Debug("Starting operation with timeout", "operation", operation, "timeout", timeout)
	
	// タイムアウト付きコンテキストを作成
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// 操作を実行するgoroutineを起動
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		errCh <- fn()
	}()
	
	// 操作完了またはタイムアウトを待つ
	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("Operation failed", "operation", operation, "error", err)
		} else {
			slog.Debug("Operation completed successfully", "operation", operation)
		}
		return err
	case <-timeoutCtx.Done():
		slog.Error("Operation timed out", "operation", operation, "timeout", timeout)
		return fmt.Errorf("operation '%s' timed out after %v", operation, timeout)
	}
}

// WithDiscoveryTimeout は、デバイス検出操作にタイムアウト制御を追加する
func (tm *TimeoutManager) WithDiscoveryTimeout(ctx context.Context, fn func() error) error {
	return tm.WithTimeout(ctx, "discovery", tm.DiscoveryTimeout, fn)
}

// WithPropertyUpdateTimeout は、プロパティ更新操作にタイムアウト制御を追加する
func (tm *TimeoutManager) WithPropertyUpdateTimeout(ctx context.Context, fn func() error) error {
	return tm.WithTimeout(ctx, "property_update", tm.PropertyUpdateTimeout, fn)
}

// WithPropertyGetTimeout は、プロパティ取得操作にタイムアウト制御を追加する
func (tm *TimeoutManager) WithPropertyGetTimeout(ctx context.Context, fn func() error) error {
	return tm.WithTimeout(ctx, "property_get", tm.PropertyGetTimeout, fn)
}

// WithPropertySetTimeout は、プロパティ設定操作にタイムアウト制御を追加する
func (tm *TimeoutManager) WithPropertySetTimeout(ctx context.Context, fn func() error) error {
	return tm.WithTimeout(ctx, "property_set", tm.PropertySetTimeout, fn)
}