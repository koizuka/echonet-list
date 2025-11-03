package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
)

const defaultHistoryLimit = 50

// handleGetDeviceHistoryFromClient handles a get_device_history message from a client.
func (ws *WebSocketServer) handleGetDeviceHistoryFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	if ws.GetHistoryStore() == nil {
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "History store is not available")
	}

	var payload protocol.GetDeviceHistoryPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing get_device_history payload: %v", err)
	}

	target := strings.TrimSpace(payload.Target)
	if target == "" {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No target specified")
	}

	ipAndEOJ, err := handler.ParseDeviceIdentifier(target)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
	}

	if ws.deviceResolver == nil {
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "Handler is not available")
	}

	if !ws.deviceResolver(ipAndEOJ) {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Unknown device: %s", target)
	}

	limit := defaultHistoryLimit
	if payload.Limit != nil {
		if *payload.Limit <= 0 {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Limit must be greater than zero")
		}
		limit = *payload.Limit
	}

	if storeLimit := ws.GetHistoryStore().PerDeviceTotalLimit(); storeLimit > 0 && limit > storeLimit {
		limit = storeLimit
	}

	settableOnly := true
	if payload.SettableOnly != nil {
		settableOnly = *payload.SettableOnly
	}

	query := handler.HistoryQuery{
		Limit:        limit,
		SettableOnly: settableOnly,
	}

	history := ws.GetHistoryStore().Query(ipAndEOJ, query)
	resultEntries := make([]protocol.HistoryEntry, 0, len(history))

	for _, entry := range history {
		// For event entries (online/offline), EPC is 0 and should be omitted from the response
		epcStr := ""
		if entry.EPC != 0 {
			epcStr = fmt.Sprintf("%02X", byte(entry.EPC))
		}

		// Use the settable flag from the stored entry
		// This preserves the settable state at the time of recording
		settable := entry.Settable

		resultEntries = append(resultEntries, protocol.HistoryEntry{
			Timestamp: entry.Timestamp,
			EPC:       epcStr, // Empty string for events, will be omitted in JSON
			Value:     protocol.PropertyDataFromHandlerValue(entry.Value),
			Origin:    protocol.HistoryOrigin(entry.Origin),
			Settable:  settable,
		})
	}

	response := protocol.DeviceHistoryResponse{
		Entries: resultEntries,
	}

	data, err := json.Marshal(response)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error marshaling history data: %v", err)
	}

	return SuccessResponse(data)
}
