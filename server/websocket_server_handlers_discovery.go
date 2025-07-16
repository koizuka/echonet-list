package server

import (
	"echonet-list/protocol"
	"log/slog"
	"time"
)

// handleDiscoverDevicesFromClient handles a discover_devices message from a client
func (ws *WebSocketServer) handleDiscoverDevicesFromClient(_ *protocol.Message) protocol.CommandResultPayload {
	// Discover devices
	slog.Info("Starting device discovery")
	start := time.Now()
	
	if err := ws.echonetClient.Discover(); err != nil {
		duration := time.Since(start)
		slog.Error("Device discovery failed", "duration", duration, "error", err)
		return ErrorResponse(protocol.ErrorCodeEchonetCommunicationError, "Error discovering devices: %v", err)
	}

	duration := time.Since(start)
	slog.Info("Device discovery completed", "duration", duration)
	
	// Send the success response
	return SuccessResponse(nil)
}
