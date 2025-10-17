package client

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"echonet-list/protocol"
)

// GetDeviceHistory retrieves device history entries from the server using the WebSocket API.
func (c *WebSocketClient) GetDeviceHistory(device IPAndEOJ, opts DeviceHistoryOptions) ([]DeviceHistoryEntry, error) {
	payload := protocol.GetDeviceHistoryPayload{
		Target: device.Specifier(),
	}

	if opts.Limit > 0 {
		limit := opts.Limit
		payload.Limit = &limit
	}

	if opts.Since != nil {
		payload.Since = opts.Since.UTC().Format(time.RFC3339Nano)
	}

	if opts.SettableOnly != nil {
		payload.SettableOnly = opts.SettableOnly
	}

	response, err := c.sendRequest(protocol.MessageTypeGetDeviceHistory, payload)
	if err != nil {
		return nil, err
	}

	var resultPayload protocol.CommandResultPayload
	if err := protocol.ParsePayload(response, &resultPayload); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !resultPayload.Success {
		if resultPayload.Error != nil {
			return nil, fmt.Errorf("error getting device history: %s: %s", resultPayload.Error.Code, resultPayload.Error.Message)
		}
		return nil, fmt.Errorf("error getting device history: unknown error")
	}

	if resultPayload.Data == nil {
		return []DeviceHistoryEntry{}, nil
	}

	var history protocol.DeviceHistoryResponse
	if err := json.Unmarshal(resultPayload.Data, &history); err != nil {
		return nil, fmt.Errorf("error parsing history data: %v", err)
	}

	entries := make([]DeviceHistoryEntry, 0, len(history.Entries))
	for _, entry := range history.Entries {
		// For event entries (online/offline), EPC is empty/omitted
		var epcValue uint64
		if entry.EPC != "" {
			var err error
			epcValue, err = strconv.ParseUint(entry.EPC, 16, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid EPC value in response: %v", err)
			}

			if len(entry.EPC) != 2 {
				return nil, fmt.Errorf("invalid EPC format in response: %s", entry.EPC)
			}
		}
		// If EPC is empty, epcValue remains 0 (for event entries)

		entries = append(entries, DeviceHistoryEntry{
			Timestamp: entry.Timestamp,
			EPC:       EPCType(epcValue),
			Value:     entry.Value,
			Origin:    entry.Origin,
			Settable:  entry.Settable,
		})
	}

	return entries, nil
}
