package server

import (
	"echonet-list/echonet_lite/handler"
)

// getOperationTracker は、WebSocketServerからOperationTrackerを取得する
func (ws *WebSocketServer) getOperationTracker() *handler.OperationTracker {
	if ws.handler == nil {
		return nil
	}

	// ECHONETLiteHandlerからHandlerCoreを取得
	if core := ws.handler.GetCore(); core != nil {
		return core.OperationTracker
	}

	return nil
}

// GetOperationStats は、現在の操作統計を取得する
func (ws *WebSocketServer) GetOperationStats() map[string]interface{} {
	tracker := ws.getOperationTracker()
	if tracker == nil {
		return map[string]interface{}{
			"available": false,
			"message":   "Operation tracker not available",
		}
	}

	runningOps := tracker.GetRunningOperations()
	stats := map[string]interface{}{
		"available":          true,
		"running_operations": len(runningOps),
		"operations_by_type": make(map[string]int),
	}

	// 操作種別ごとの統計を作成
	for _, op := range runningOps {
		opType := op.Type.String()
		if count, exists := stats["operations_by_type"].(map[string]int)[opType]; exists {
			stats["operations_by_type"].(map[string]int)[opType] = count + 1
		} else {
			stats["operations_by_type"].(map[string]int)[opType] = 1
		}
	}

	return stats
}

// GetRunningOperationsInfo は、実行中の操作の詳細情報を取得する
func (ws *WebSocketServer) GetRunningOperationsInfo() []map[string]interface{} {
	tracker := ws.getOperationTracker()
	if tracker == nil {
		return []map[string]interface{}{}
	}

	runningOps := tracker.GetRunningOperations()
	var result []map[string]interface{}

	for _, op := range runningOps {
		opInfo := map[string]interface{}{
			"id":          op.ID,
			"type":        op.Type.String(),
			"description": op.Description,
			"start_time":  op.StartTime,
			"duration":    op.StartTime.Sub(op.StartTime),
			"context":     op.Context,
		}
		result = append(result, opInfo)
	}

	return result
}
