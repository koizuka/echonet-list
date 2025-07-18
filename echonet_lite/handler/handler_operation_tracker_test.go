package handler

import (
	"context"
	"testing"
	"time"
)

func TestOperationTracker_StartCompleteOperation(t *testing.T) {
	ctx := context.Background()
	tracker := NewOperationTracker(ctx, 100*time.Millisecond)

	// 操作を開始
	tracker.StartOperation("test_1", OperationTypeDiscover, "Test discovery", map[string]interface{}{
		"target": "broadcast",
	})

	// 実行中の操作を確認
	running := tracker.GetRunningOperations()
	if len(running) != 1 {
		t.Errorf("Expected 1 running operation, got %d", len(running))
	}

	if running[0].ID != "test_1" {
		t.Errorf("Expected operation ID 'test_1', got %s", running[0].ID)
	}

	// 操作を完了
	tracker.CompleteOperation("test_1", true, nil)

	// 実行中の操作が削除されていることを確認
	running = tracker.GetRunningOperations()
	if len(running) != 0 {
		t.Errorf("Expected 0 running operations after completion, got %d", len(running))
	}
}

func TestOperationTracker_Timeout(t *testing.T) {
	ctx := context.Background()
	tracker := NewOperationTracker(ctx, 50*time.Millisecond)

	// 短いタイムアウトを設定
	tracker.SetTimeout(OperationTypeDiscover, 100*time.Millisecond)

	// 監視を開始
	tracker.Start()
	defer tracker.Stop()

	// 操作を開始
	tracker.StartOperation("test_timeout", OperationTypeDiscover, "Test timeout", nil)

	// タイムアウトが発生するまで待つ
	time.Sleep(200 * time.Millisecond)

	// タイムアウトにより操作が削除されていることを確認
	running := tracker.GetRunningOperations()
	if len(running) != 0 {
		t.Errorf("Expected 0 running operations after timeout, got %d", len(running))
	}
}

func TestOperationTracker_GetOperationsByType(t *testing.T) {
	ctx := context.Background()
	tracker := NewOperationTracker(ctx, 1*time.Second)

	// 異なる種類の操作を開始
	tracker.StartOperation("discover_1", OperationTypeDiscover, "Discovery 1", nil)
	tracker.StartOperation("update_1", OperationTypeUpdateProperties, "Update 1", nil)
	tracker.StartOperation("discover_2", OperationTypeDiscover, "Discovery 2", nil)

	// 発見操作のみを取得
	discoveries := tracker.GetOperationsByType(OperationTypeDiscover)
	if len(discoveries) != 2 {
		t.Errorf("Expected 2 discovery operations, got %d", len(discoveries))
	}

	// 更新操作のみを取得
	updates := tracker.GetOperationsByType(OperationTypeUpdateProperties)
	if len(updates) != 1 {
		t.Errorf("Expected 1 update operation, got %d", len(updates))
	}
}

func TestOperationTracker_GetOperationCount(t *testing.T) {
	ctx := context.Background()
	tracker := NewOperationTracker(ctx, 1*time.Second)

	// 初期状態では0個
	count := tracker.GetOperationCount()
	if count != 0 {
		t.Errorf("Expected 0 operations initially, got %d", count)
	}

	// 操作を追加
	tracker.StartOperation("test_1", OperationTypeDiscover, "Test 1", nil)
	tracker.StartOperation("test_2", OperationTypeUpdateProperties, "Test 2", nil)

	count = tracker.GetOperationCount()
	if count != 2 {
		t.Errorf("Expected 2 operations, got %d", count)
	}

	// 操作を完了
	tracker.CompleteOperation("test_1", true, nil)

	count = tracker.GetOperationCount()
	if count != 1 {
		t.Errorf("Expected 1 operation after completion, got %d", count)
	}
}

func TestOperationType_String(t *testing.T) {
	tests := []struct {
		opType   OperationType
		expected string
	}{
		{OperationTypeDiscover, "discover"},
		{OperationTypeUpdateProperties, "update_properties"},
		{OperationTypeGetProperties, "get_properties"},
		{OperationTypeSetProperties, "set_properties"},
	}

	for _, test := range tests {
		result := test.opType.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}
