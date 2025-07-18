package server

import (
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"log/slog"
	"time"
)

// handleDiscoverDevicesFromClient handles a discover_devices message from a client
func (ws *WebSocketServer) handleDiscoverDevicesFromClient(_ *protocol.Message) protocol.CommandResultPayload {
	// 操作追跡を開始
	operationID := "discover_" + time.Now().Format("20060102_150405.000")

	// ECHONETクライアントからOperationTrackerを取得
	if tracker := ws.getOperationTracker(); tracker != nil {
		tracker.StartOperation(operationID, handler.OperationTypeDiscover, "Device discovery from WebSocket", map[string]interface{}{
			"source": "websocket",
		})

		// Discover devices
		if err := ws.echonetClient.Discover(); err != nil {
			tracker.CompleteOperation(operationID, false, err)
			return ErrorResponse(protocol.ErrorCodeEchonetCommunicationError, "Error discovering devices: %v", err)
		}

		tracker.CompleteOperation(operationID, true, nil)
	} else {
		// フォールバック: 従来のログ方式
		slog.Info("Starting device discovery")
		start := time.Now()

		if err := ws.echonetClient.Discover(); err != nil {
			duration := time.Since(start)
			slog.Error("Device discovery failed", "duration", duration, "error", err)
			return ErrorResponse(protocol.ErrorCodeEchonetCommunicationError, "Error discovering devices: %v", err)
		}

		duration := time.Since(start)
		slog.Info("Device discovery completed", "duration", duration)
	}

	// Send the success response
	return SuccessResponse(nil)
}
