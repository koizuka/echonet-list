package server

import (
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// handleListDevicesFromClient handles a list_devices message from a client
func (ws *WebSocketServer) handleListDevicesFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// 操作追跡を開始
	operationID := "list_devices_" + time.Now().Format("20060102_150405.000")

	// Parse the payload
	var payload protocol.ListDevicesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing list_devices payload: %v", err)
	}

	// ECHONETクライアントからOperationTrackerを取得
	if tracker := ws.getOperationTracker(); tracker != nil {
		tracker.StartOperation(operationID, handler.OperationTypeGetProperties,
			fmt.Sprintf("List devices for %d targets", len(payload.Targets)),
			map[string]interface{}{
				"source":       "websocket",
				"target_count": len(payload.Targets),
			})

		defer func() {
			tracker.CompleteOperation(operationID, true, nil)
		}()
	}

	var devices []handler.DeviceAndProperties

	if len(payload.Targets) == 0 {
		// No targets specified, return all online devices (like initial_state)
		devices = ws.echonetClient.ListDevices(handler.FilterCriteria{ExcludeOffline: true})
	} else {
		// Process specific targets
		for _, target := range payload.Targets {
			// Parse the target
			ipAndEOJ, err := handler.ParseDeviceIdentifier(target)
			if ws.handler.IsDebug() {
				slog.Debug("Processing target for list_devices", "target", target, "ipAndEOJ", ipAndEOJ)
			}

			if err != nil {
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
			}

			// Get device from cache using ListDevices with specific filter
			targetDevices := ws.echonetClient.ListDevices(handler.FilterCriteria{
				Device: handler.DeviceSpecifierFromIPAndEOJ(ipAndEOJ),
			})

			// Add found devices to result
			devices = append(devices, targetDevices...)
		}
	}

	if ws.handler.IsDebug() {
		slog.Debug("List devices completed", "deviceCount", len(devices))
	}

	// Convert devices to protocol format
	results := make([]protocol.Device, 0, len(devices))
	for _, device := range devices {
		// デバイスの最終更新タイムスタンプを取得
		lastSeen := ws.handler.GetLastUpdateTime(device.Device)

		// Use DeviceToProtocol to convert to protocol format
		protoDevice := protocol.DeviceToProtocol(
			device.Device,
			device.Properties,
			lastSeen,
		)
		results = append(results, protoDevice)
	}

	// Marshal the results
	var resultJSON json.RawMessage
	if len(results) == 1 {
		// Single device - return the device directly (same format as get_properties)
		deviceJSON, err := json.Marshal(results[0])
		if err != nil {
			return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error marshaling device: %v", err)
		}
		resultJSON = deviceJSON
	} else {
		// Multiple devices - return as array
		devicesJSON, err := json.Marshal(results)
		if err != nil {
			return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error marshaling devices: %v", err)
		}
		resultJSON = devicesJSON
	}

	// Send the success response
	return SuccessResponse(resultJSON)
}

// handleDeleteDeviceFromClient handles a delete_device message from a client
func (ws *WebSocketServer) handleDeleteDeviceFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// Parse the payload
	var payload protocol.DeleteDevicePayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing delete_device payload: %v", err)
	}

	// Parse the target device identifier
	ipAndEOJ, err := handler.ParseDeviceIdentifier(payload.Target)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target device identifier: %v", err)
	}

	if ws.handler.IsDebug() {
		slog.Debug("Deleting device", "target", payload.Target, "ipAndEOJ", ipAndEOJ)
	}

	// Remove the device from the handler
	if err := ws.handler.RemoveDevice(ipAndEOJ); err != nil {
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "Failed to remove device: %v", err)
	}

	// Broadcast device_deleted notification to all connected clients
	deletePayload := protocol.DeviceDeletedPayload{
		IP:  ipAndEOJ.IP.String(),
		EOJ: ipAndEOJ.EOJ.Specifier(),
	}

	if err := ws.broadcastMessageToClients(protocol.MessageTypeDeviceDeleted, deletePayload); err != nil {
		slog.Error("Failed to broadcast device_deleted notification", "error", err)
		// Don't return error here since the device was successfully deleted
	}

	if ws.handler.IsDebug() {
		slog.Debug("Device deleted successfully", "target", payload.Target)
	}

	// Return success response with empty data
	return SuccessResponse(nil)
}
