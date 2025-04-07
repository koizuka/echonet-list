package server

import (
	"echonet-list/protocol"
)

// handleDiscoverDevicesFromClient handles a discover_devices message from a client
func (ws *WebSocketServer) handleDiscoverDevicesFromClient(connID string, msg *protocol.Message) error {
	// Discover devices
	if err := ws.echonetClient.Discover(); err != nil {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeEchonetCommunicationError, "Error discovering devices: %v", err)
	}

	// Send the success response
	return ws.sendSuccessResponse(connID, msg.RequestID, nil)
}
