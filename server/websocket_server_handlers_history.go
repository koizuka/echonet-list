package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
)

const defaultHistoryLimit = 50

func parseHistorySince(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts, nil
	}
	return time.Parse(time.RFC3339, value)
}

// handleGetDeviceHistoryFromClient handles a get_device_history message from a client.
func (ws *WebSocketServer) handleGetDeviceHistoryFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	if ws.historyStore == nil {
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

	if storeLimit := ws.historyStore.PerDeviceLimit(); storeLimit > 0 && limit > storeLimit {
		limit = storeLimit
	}

	since := time.Time{}
	if payload.Since != "" {
		var err error
		since, err = parseHistorySince(payload.Since)
		if err != nil {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid since value: %v", err)
		}
	}

	settableOnly := true
	if payload.SettableOnly != nil {
		settableOnly = *payload.SettableOnly
	}

	query := HistoryQuery{
		Since: since,
		Limit: limit,
	}

	history := ws.historyStore.Query(ipAndEOJ, query)
	resultEntries := make([]protocol.HistoryEntry, 0, len(history))

	for _, entry := range history {
		// For event entries (online/offline), EPC is 0 and should be omitted from the response
		epcStr := ""
		if entry.EPC != 0 {
			epcStr = fmt.Sprintf("%02X", byte(entry.EPC))
		}

		// Calculate settable flag dynamically based on current Set Property Map
		// For event entries (online/offline), settable is always false
		settable := false
		if entry.EPC != 0 && entry.Origin != HistoryOriginOnline && entry.Origin != HistoryOriginOffline {
			settable = ws.isPropertySettable(ipAndEOJ, entry.EPC)
		}

		// Apply settableOnly filter if requested
		if settableOnly && !settable {
			continue
		}

		resultEntries = append(resultEntries, protocol.HistoryEntry{
			Timestamp: entry.Timestamp,
			EPC:       epcStr, // Empty string for events, will be omitted in JSON
			Value:     entry.Value,
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
