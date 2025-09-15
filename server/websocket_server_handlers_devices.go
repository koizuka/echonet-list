package server

import (
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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
		// Check if device is offline
		var isOffline bool
		if ws.handler != nil {
			isOffline = ws.handler.IsOffline(device.Device)
		}
		protoDevice := protocol.DeviceToProtocol(
			device.Device,
			device.Properties,
			lastSeen,
			isOffline,
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

	// Check if this is a NodeProfile deletion
	if ipAndEOJ.EOJ.ClassCode() == echonet_lite.NodeProfile_ClassCode {
		// For NodeProfile, delete all devices at the same IP address
		if ws.handler.IsDebug() {
			slog.Debug("NodeProfile deletion detected, removing all devices at IP", "ip", ipAndEOJ.IP.String())
		}

		// Get all devices at the same IP address
		deviceSpec := handler.DeviceSpecifier{
			IP: &ipAndEOJ.IP,
		}
		devicesAtIP := ws.handler.GetDevices(deviceSpec)

		// Remove each device
		var deleteErrors []string
		for _, device := range devicesAtIP {
			if err := ws.handler.RemoveDevice(device); err != nil {
				deleteErrors = append(deleteErrors, fmt.Sprintf("Failed to remove device %s: %v", device.Specifier(), err))
				continue
			}

			// Broadcast device_deleted notification for each device
			deletePayload := protocol.DeviceDeletedPayload{
				IP:  device.IP.String(),
				EOJ: device.EOJ.Specifier(),
			}

			if err := ws.broadcastMessageToClients(protocol.MessageTypeDeviceDeleted, deletePayload); err != nil {
				slog.Error("Failed to broadcast device_deleted notification", "error", err, "device", device.Specifier())
			}
		}

		// If there were any errors, return the first one
		if len(deleteErrors) > 0 {
			return ErrorResponse(protocol.ErrorCodeInternalServerError, "%s", strings.Join(deleteErrors, "; "))
		}
	} else {
		// For non-NodeProfile devices, just remove the single device
		if err := ws.handler.RemoveDevice(ipAndEOJ); err != nil {
			return ErrorResponse(protocol.ErrorCodeInternalServerError, "Failed to remove device: %v", err)
		}

		// Broadcast device_deleted notification
		deletePayload := protocol.DeviceDeletedPayload{
			IP:  ipAndEOJ.IP.String(),
			EOJ: ipAndEOJ.EOJ.Specifier(),
		}

		if err := ws.broadcastMessageToClients(protocol.MessageTypeDeviceDeleted, deletePayload); err != nil {
			slog.Error("Failed to broadcast device_deleted notification", "error", err)
			// Don't return error here since the device was successfully deleted
		}
	}

	if ws.handler.IsDebug() {
		slog.Debug("Device deleted successfully", "target", payload.Target)
	}

	// Return success response with empty data
	return SuccessResponse(nil)
}

// handleDebugSetOfflineFromClient handles a debug_set_offline message from a client
func (ws *WebSocketServer) handleDebugSetOfflineFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// Parse the payload
	var payload protocol.DebugSetOfflinePayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing debug_set_offline payload: %v", err)
	}

	// Parse the target device identifier
	ipAndEOJ, err := handler.ParseDeviceIdentifier(payload.Target)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target device identifier: %v", err)
	}

	if ws.handler.IsDebug() {
		slog.Debug("Debug set offline", "target", payload.Target, "offline", payload.Offline, "ipAndEOJ", ipAndEOJ)
	}

	// Set the device offline/online state directly using DataManagementHandler
	ws.handler.GetDataManagementHandler().SetOffline(ipAndEOJ, payload.Offline)

	slog.Info("Debug set device offline state", "target", payload.Target, "offline", payload.Offline)

	// Return success response with empty data
	return SuccessResponse(nil)
}
