package server

import (
	"echonet-list/client"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/log"
	"echonet-list/protocol"
	"encoding/json"
	"fmt"
)

// handleManageAliasFromClient handles a manage_alias message from a client
func (ws *WebSocketServer) handleManageAliasFromClient(connID string, msg *protocol.Message) error {
	logger := log.GetLogger()
	// Parse the payload
	var payload protocol.ManageAliasPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if logger != nil {
			logger.Log("Error parsing manage_alias payload: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidRequestFormat,
				Message: fmt.Sprintf("Error parsing manage_alias payload: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Validate the payload
	if payload.Alias == "" {
		if logger != nil {
			logger.Log("Error: no alias specified")
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "No alias specified",
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Handle the action
	switch payload.Action {
	case protocol.AliasActionAdd:
		// Validate the target
		if payload.Target == "" {
			if logger != nil {
				logger.Log("Error: no target specified for add action")
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInvalidParameters,
					Message: "No target specified for add action",
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		ipAndEOJ := ws.handler.FindDeviceByIDString(payload.Target)
		if ipAndEOJ == nil {
			if logger != nil {
				logger.Log("Error: invalid target: %v", payload.Target)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInvalidParameters,
					Message: fmt.Sprintf("Invalid target: %v", payload.Target),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Create filter criteria
		classCode := ipAndEOJ.EOJ.ClassCode()
		instanceCode := ipAndEOJ.EOJ.InstanceCode()
		criteria := echonet_lite.FilterCriteria{
			Device: echonet_lite.DeviceSpecifier{
				IP:           &ipAndEOJ.IP,
				ClassCode:    &classCode,
				InstanceCode: &instanceCode,
			},
		}

		// Set the alias
		if err := ws.echonetClient.AliasSet(&payload.Alias, criteria); err != nil {
			if logger != nil {
				logger.Log("Error setting alias: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeAliasOperationFailed,
					Message: err.Error(),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Broadcast alias changed notification
		aliasChangedPayload := protocol.AliasChangedPayload{
			ChangeType: protocol.AliasChangeTypeAdded,
			Alias:      payload.Alias,
			Target:     payload.Target,
		}
		ws.broadcastMessageToClients(protocol.MessageTypeAliasChanged, aliasChangedPayload)

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}

		// Send the message
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

	case protocol.AliasActionDelete:
		// Delete the alias
		if err := ws.echonetClient.AliasDelete(&payload.Alias); err != nil {
			if logger != nil {
				logger.Log("Error deleting alias: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeAliasOperationFailed,
					Message: err.Error(),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Broadcast alias changed notification
		aliasChangedPayload := protocol.AliasChangedPayload{
			ChangeType: protocol.AliasChangeTypeDeleted,
			Alias:      payload.Alias,
			Target:     "",
		}
		ws.broadcastMessageToClients(protocol.MessageTypeAliasChanged, aliasChangedPayload)

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}

		// Send the message
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

	default:
		if logger != nil {
			logger.Log("Error: unknown alias action: %s", payload.Action)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: fmt.Sprintf("Unknown alias action: %s", payload.Action),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}
}

// handleManageGroupFromClient handles a manage_group message from a client
func (ws *WebSocketServer) handleManageGroupFromClient(connID string, msg *protocol.Message) error {
	logger := log.GetLogger()
	// Parse the payload
	var payload protocol.ManageGroupPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if logger != nil {
			logger.Log("Error parsing manage_group payload: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidRequestFormat,
				Message: fmt.Sprintf("Error parsing manage_group payload: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Validate the payload
	if payload.Group == "" {
		if logger != nil {
			logger.Log("Error: no group specified")
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "No group specified",
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Handle the action
	switch payload.Action {
	case protocol.GroupActionAdd:
		// Validate the devices
		if len(payload.Devices) == 0 {
			if logger != nil {
				logger.Log("Error: no devices specified for add action")
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInvalidParameters,
					Message: "No devices specified for add action",
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Parse the devices
		devices := make([]echonet_lite.IDString, 0, len(payload.Devices))
		for _, ids := range payload.Devices {
			device := ws.handler.FindDeviceByIDString(echonet_lite.IDString(ids))
			if device == nil {
				if logger != nil {
					logger.Log("Error: device not found: %s", ids)
				}
				// エラー応答を送信
				errorPayload := protocol.CommandResultPayload{
					Success: false,
					Error: &protocol.Error{
						Code:    protocol.ErrorCodeInvalidParameters,
						Message: fmt.Sprintf("Device not found: %s", ids),
					},
				}
				return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
			}
			devices = append(devices, echonet_lite.IDString(ids))
		}

		// Add the devices to the group
		if err := ws.echonetClient.GroupAdd(payload.Group, devices); err != nil {
			if logger != nil {
				logger.Log("Error adding devices to group: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInternalServerError,
					Message: err.Error(),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Broadcast group changed notification
		groupChangedPayload := protocol.GroupChangedPayload{
			ChangeType: protocol.GroupChangeTypeAdded,
			Group:      payload.Group,
			Devices:    payload.Devices,
		}
		ws.broadcastMessageToClients(protocol.MessageTypeGroupChanged, groupChangedPayload)

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

	case protocol.GroupActionRemove:
		// Validate the devices
		if len(payload.Devices) == 0 {
			if logger != nil {
				logger.Log("Error: no devices specified for remove action")
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInvalidParameters,
					Message: "No devices specified for remove action",
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Parse the devices
		devices := make([]echonet_lite.IDString, 0, len(payload.Devices))
		for _, ids := range payload.Devices {
			device := ws.handler.FindDeviceByIDString(echonet_lite.IDString(ids))
			if device == nil {
				if logger != nil {
					logger.Log("Error: device not found: %s", ids)
				}
				// エラー応答を送信
				errorPayload := protocol.CommandResultPayload{
					Success: false,
					Error: &protocol.Error{
						Code:    protocol.ErrorCodeInvalidParameters,
						Message: fmt.Sprintf("Device not found: %s", ids),
					},
				}
				return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
			}
			devices = append(devices, echonet_lite.IDString(ids))
		}

		// Remove the devices from the group
		if err := ws.echonetClient.GroupRemove(payload.Group, devices); err != nil {
			if logger != nil {
				logger.Log("Error removing devices from group: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInternalServerError,
					Message: err.Error(),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Get the updated devices in the group
		updatedDevices, exists := ws.echonetClient.GetDevicesByGroup(payload.Group)
		if !exists {
			// Group was deleted (all devices removed)
			groupChangedPayload := protocol.GroupChangedPayload{
				ChangeType: protocol.GroupChangeTypeDeleted,
				Group:      payload.Group,
			}
			ws.broadcastMessageToClients(protocol.MessageTypeGroupChanged, groupChangedPayload)
		} else {
			// Group was updated
			deviceStrs := make([]string, 0, len(updatedDevices))
			for _, ids := range updatedDevices {
				deviceStrs = append(deviceStrs, string(ids))
			}
			groupChangedPayload := protocol.GroupChangedPayload{
				ChangeType: protocol.GroupChangeTypeUpdated,
				Group:      payload.Group,
				Devices:    deviceStrs,
			}
			ws.broadcastMessageToClients(protocol.MessageTypeGroupChanged, groupChangedPayload)
		}

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

	case protocol.GroupActionDelete:
		// Delete the group
		if err := ws.echonetClient.GroupDelete(payload.Group); err != nil {
			if logger != nil {
				logger.Log("Error deleting group: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInternalServerError,
					Message: err.Error(),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Broadcast group deleted notification
		groupChangedPayload := protocol.GroupChangedPayload{
			ChangeType: protocol.GroupChangeTypeDeleted,
			Group:      payload.Group,
		}
		ws.broadcastMessageToClients(protocol.MessageTypeGroupChanged, groupChangedPayload)

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

	case protocol.GroupActionList:
		// Get the group list
		var groupList []client.GroupDevicePair
		if payload.Group != "" {
			// Get a specific group
			groupName := payload.Group
			groupList = ws.echonetClient.GroupList(&groupName)
		} else {
			// Get all groups
			groupList = ws.echonetClient.GroupList(nil)
		}

		// Convert to map for JSON response
		groups := make(map[string][]string)
		for _, group := range groupList {
			deviceStrs := make([]string, 0, len(group.Devices))
			for _, ids := range group.Devices {
				deviceStrs = append(deviceStrs, string(ids))
			}
			groups[group.Group] = deviceStrs
		}

		// Marshal the group data
		groupDataJSON, err := json.Marshal(groups)
		if err != nil {
			if logger != nil {
				logger.Log("Error marshaling group data: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInternalServerError,
					Message: fmt.Sprintf("Error marshaling group data: %v", err),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
			Data:    groupDataJSON,
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

	default:
		if logger != nil {
			logger.Log("Error: unknown group action: %s", payload.Action)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: fmt.Sprintf("Unknown group action: %s", payload.Action),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}
}
