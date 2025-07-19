package handler

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// OperationType は操作の種類を表す
type OperationType int

const (
	OperationTypeDiscover OperationType = iota
	OperationTypeUpdateProperties
	OperationTypeGetProperties
	OperationTypeSetProperties
)

func (ot OperationType) String() string {
	switch ot {
	case OperationTypeDiscover:
		return "discover"
	case OperationTypeUpdateProperties:
		return "update_properties"
	case OperationTypeGetProperties:
		return "get_properties"
	case OperationTypeSetProperties:
		return "set_properties"
	default:
		return "unknown"
	}
}

// OperationInfo は実行中の操作の情報を保持する
type OperationInfo struct {
	ID          string
	Type        OperationType
	StartTime   time.Time
	Description string
	Context     map[string]interface{}
}

// OperationTracker は操作の追跡と監視を行う
type OperationTracker struct {
	mu            sync.RWMutex
	operations    map[string]*OperationInfo
	ctx           context.Context
	cancel        context.CancelFunc
	checkInterval time.Duration
	timeouts      map[OperationType]time.Duration
}

// NewOperationTracker は新しい操作追跡システムを作成する
func NewOperationTracker(ctx context.Context, checkInterval time.Duration) *OperationTracker {
	trackingCtx, cancel := context.WithCancel(ctx)

	return &OperationTracker{
		operations:    make(map[string]*OperationInfo),
		ctx:           trackingCtx,
		cancel:        cancel,
		checkInterval: checkInterval,
		timeouts: map[OperationType]time.Duration{
			OperationTypeDiscover:         30 * time.Second,
			OperationTypeUpdateProperties: 60 * time.Second,
			OperationTypeGetProperties:    10 * time.Second,
			OperationTypeSetProperties:    10 * time.Second,
		},
	}
}

// Start は監視goroutineを開始する
func (ot *OperationTracker) Start() {
	go func() {
		ticker := time.NewTicker(ot.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ot.checkTimeouts()
			case <-ot.ctx.Done():
				return
			}
		}
	}()
}

// Stop は監視を停止する
func (ot *OperationTracker) Stop() {
	if ot.cancel != nil {
		ot.cancel()
	}
}

// StartOperation は新しい操作の追跡を開始する
func (ot *OperationTracker) StartOperation(id string, opType OperationType, description string, context map[string]interface{}) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.operations[id] = &OperationInfo{
		ID:          id,
		Type:        opType,
		StartTime:   time.Now(),
		Description: description,
		Context:     context,
	}

	slog.Debug("Operation started", "id", id, "type", opType.String(), "description", description)
}

// CompleteOperation は操作の完了を記録し、追跡を終了する
func (ot *OperationTracker) CompleteOperation(id string, success bool, err error) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	if info, exists := ot.operations[id]; exists {
		duration := time.Since(info.StartTime)

		if success {
			// 成功時は通常のログレベルで記録
			slog.Debug("Operation completed successfully",
				"id", id,
				"type", info.Type.String(),
				"duration", duration,
				"description", info.Description)
		} else {
			// 失敗時はエラーログで記録
			slog.Error("Operation failed",
				"id", id,
				"type", info.Type.String(),
				"duration", duration,
				"description", info.Description,
				"error", err)
		}

		delete(ot.operations, id)
	}
}

// checkTimeouts は実行中の操作のタイムアウトをチェックする
func (ot *OperationTracker) checkTimeouts() {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	now := time.Now()
	var timedOutOps []string

	for id, info := range ot.operations {
		timeout, exists := ot.timeouts[info.Type]
		if !exists {
			continue
		}

		if now.Sub(info.StartTime) > timeout {
			duration := now.Sub(info.StartTime)

			slog.Warn("Operation timeout detected",
				"id", id,
				"type", info.Type.String(),
				"duration", duration,
				"timeout", timeout,
				"description", info.Description,
				"context", info.Context)

			timedOutOps = append(timedOutOps, id)
		}
	}

	// タイムアウトした操作を削除
	for _, id := range timedOutOps {
		delete(ot.operations, id)
	}
}

// GetRunningOperations は現在実行中の操作一覧を返す
func (ot *OperationTracker) GetRunningOperations() []OperationInfo {
	ot.mu.RLock()
	defer ot.mu.RUnlock()

	var operations []OperationInfo
	for _, info := range ot.operations {
		operations = append(operations, *info)
	}

	return operations
}

// GetOperationCount は現在実行中の操作数を返す
func (ot *OperationTracker) GetOperationCount() int {
	ot.mu.RLock()
	defer ot.mu.RUnlock()

	return len(ot.operations)
}

// GetOperationsByType は指定された種類の操作一覧を返す
func (ot *OperationTracker) GetOperationsByType(opType OperationType) []OperationInfo {
	ot.mu.RLock()
	defer ot.mu.RUnlock()

	var operations []OperationInfo
	for _, info := range ot.operations {
		if info.Type == opType {
			operations = append(operations, *info)
		}
	}

	return operations
}

// SetTimeout は操作種別のタイムアウト時間を設定する
func (ot *OperationTracker) SetTimeout(opType OperationType, timeout time.Duration) {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	ot.timeouts[opType] = timeout
}
