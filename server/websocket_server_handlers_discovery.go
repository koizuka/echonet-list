package server

import (
	"echonet-list/protocol"
)

// handleDiscoverDevicesFromClient handles a discover_devices message from a client
func (ws *WebSocketServer) handleDiscoverDevicesFromClient(_ *protocol.Message) protocol.CommandResultPayload {
	// Discover devices
	if err := ws.echonetClient.Discover(); err != nil {
		return ErrorResponse(protocol.ErrorCodeEchonetCommunicationError, "Error discovering devices: %v", err)
	}

	// Send the success response
	return SuccessResponse(nil)
}
