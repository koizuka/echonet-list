package server

import (
	"echonet-list/echonet_lite/log"
	"echonet-list/protocol"
	"fmt"
)

// handleDiscoverDevicesFromClient handles a discover_devices message from a client
func (ws *WebSocketServer) handleDiscoverDevicesFromClient(connID string, msg *protocol.Message) error {
	logger := log.GetLogger()
	// Discover devices
	if err := ws.echonetClient.Discover(); err != nil {
		if logger != nil {
			logger.Log("Error discovering devices: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeEchonetCommunicationError,
				Message: fmt.Sprintf("Error discovering devices: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Send the response
	resultPayload := protocol.CommandResultPayload{
		Success: true,
	}

	// Send the message
	return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)
}
