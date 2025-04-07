package server

import (
	"echonet-list/client"
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
	"encoding/json"
)

// handleManageAliasFromClient handles a manage_alias message from a client
func (ws *WebSocketServer) handleManageAliasFromClient(connID string, msg *protocol.Message) error {
	// Parse the payload
	var payload protocol.ManageAliasPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidRequestFormat, "Error parsing manage_alias payload: %v", err)
	}

	// Validate the payload
	if payload.Alias == "" {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "No alias specified")
	}

	// Handle the action
	switch payload.Action {
	case protocol.AliasActionAdd:
		// Validate the target
		if payload.Target == "" {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "No target specified for add action")
		}

		ipAndEOJ := ws.handler.FindDeviceByIDString(payload.Target)
		if ipAndEOJ == nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Invalid target: %v", payload.Target)
		}

		// Create filter criteria
		criteria := echonet_lite.FilterCriteriaFromIPAndEOJ(*ipAndEOJ)

		// Set the alias
		if err := ws.echonetClient.AliasSet(&payload.Alias, criteria); err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeAliasOperationFailed, "Error setting alias: %v", err)
		}

		// Broadcast alias changed notification
		aliasChangedPayload := protocol.AliasChangedPayload{
			ChangeType: protocol.AliasChangeTypeAdded,
			Alias:      payload.Alias,
			Target:     payload.Target,
		}
		ws.broadcastMessageToClients(protocol.MessageTypeAliasChanged, aliasChangedPayload)

		// Send the success response
		return ws.sendSuccessResponse(connID, msg.RequestID, nil)

	case protocol.AliasActionDelete:
		// Delete the alias
		if err := ws.echonetClient.AliasDelete(&payload.Alias); err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeAliasOperationFailed, "Error deleting alias: %v", err)
		}

		// Broadcast alias changed notification
		aliasChangedPayload := protocol.AliasChangedPayload{
			ChangeType: protocol.AliasChangeTypeDeleted,
			Alias:      payload.Alias,
			Target:     "",
		}
		ws.broadcastMessageToClients(protocol.MessageTypeAliasChanged, aliasChangedPayload)

		// Send the success response
		return ws.sendSuccessResponse(connID, msg.RequestID, nil)

	default:
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Unknown alias action: %s", payload.Action)
	}
}

// handleManageGroupFromClient handles a manage_group message from a client
func (ws *WebSocketServer) handleManageGroupFromClient(connID string, msg *protocol.Message) error {
	// Parse the payload
	var payload protocol.ManageGroupPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidRequestFormat, "Error parsing manage_group payload: %v", err)
	}

	// Validate the payload
	if payload.Group == "" {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "No group specified")
	}

	// Handle the action
	switch payload.Action {
	case protocol.GroupActionAdd:
		// Validate the devices
		if len(payload.Devices) == 0 {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "No devices specified for add action")
		}

		// Parse the devices
		devices := make([]echonet_lite.IDString, 0, len(payload.Devices))
		for _, ids := range payload.Devices {
			device := ws.handler.FindDeviceByIDString(ids)
			if device == nil {
				return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Device not found: %s", ids)
			}
			devices = append(devices, ids)
		}

		// Add the devices to the group
		if err := ws.echonetClient.GroupAdd(payload.Group, devices); err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInternalServerError, "Error adding devices to group: %v", err)
		}

		updatedDevices, _ := ws.echonetClient.GetDevicesByGroup(payload.Group)
		// Broadcast group changed notification
		groupChangedPayload := protocol.GroupChangedPayload{
			ChangeType: protocol.GroupChangeTypeUpdated,
			Group:      payload.Group,
			Devices:    updatedDevices,
		}
		ws.broadcastMessageToClients(protocol.MessageTypeGroupChanged, groupChangedPayload)

		// Send the success response
		return ws.sendSuccessResponse(connID, msg.RequestID, nil)

	case protocol.GroupActionRemove:
		// Validate the devices
		if len(payload.Devices) == 0 {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "No devices specified for remove action")
		}

		// Parse the devices
		devices := make([]echonet_lite.IDString, 0, len(payload.Devices))
		for _, ids := range payload.Devices {
			device := ws.handler.FindDeviceByIDString(ids)
			if device == nil {
				return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Device not found: %s", ids)
			}
			devices = append(devices, ids)
		}

		// Remove the devices from the group
		if err := ws.echonetClient.GroupRemove(payload.Group, devices); err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInternalServerError, "Error removing devices from group: %v", err)
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
			groupChangedPayload := protocol.GroupChangedPayload{
				ChangeType: protocol.GroupChangeTypeUpdated,
				Group:      payload.Group,
				Devices:    updatedDevices,
			}
			ws.broadcastMessageToClients(protocol.MessageTypeGroupChanged, groupChangedPayload)
		}

		// Send the success response
		return ws.sendSuccessResponse(connID, msg.RequestID, nil)

	case protocol.GroupActionDelete:
		// Delete the group
		if err := ws.echonetClient.GroupDelete(payload.Group); err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInternalServerError, "Error deleting group: %v", err)
		}

		// Broadcast group deleted notification
		groupChangedPayload := protocol.GroupChangedPayload{
			ChangeType: protocol.GroupChangeTypeDeleted,
			Group:      payload.Group,
		}
		ws.broadcastMessageToClients(protocol.MessageTypeGroupChanged, groupChangedPayload)

		// Send the success response
		return ws.sendSuccessResponse(connID, msg.RequestID, nil)

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
		groups := make(map[string][]client.IDString)
		for _, group := range groupList {
			groups[group.Group] = group.Devices
		}

		// Marshal the group data
		groupDataJSON, err := json.Marshal(groups)
		if err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInternalServerError, "Error marshaling group data: %v", err)
		}

		// Send the success response with group data
		return ws.sendSuccessResponse(connID, msg.RequestID, groupDataJSON)

	default:
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Unknown group action: %s", payload.Action)
	}
}
