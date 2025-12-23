package server

import (
	"echonet-list/protocol"
	"encoding/json"
)

// handleGetLocationSettingsFromClient handles a get_location_settings message from a client
func (ws *WebSocketServer) handleGetLocationSettingsFromClient(_ *protocol.Message) protocol.CommandResultPayload {
	aliases, order := ws.handler.GetLocationSettings()

	data := protocol.LocationSettingsData{
		Aliases: aliases,
		Order:   order,
	}

	// Ensure maps are not nil
	if data.Aliases == nil {
		data.Aliases = make(map[string]string)
	}
	if data.Order == nil {
		data.Order = []string{}
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error marshaling location settings: %v", err)
	}

	return SuccessResponse(dataJSON)
}

// handleManageLocationAliasFromClient handles a manage_location_alias message from a client
func (ws *WebSocketServer) handleManageLocationAliasFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// Parse the payload
	var payload protocol.ManageLocationAliasPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing manage_location_alias payload: %v", err)
	}

	// Validate the alias
	if payload.Alias == "" {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No alias specified")
	}

	// Handle the action
	switch payload.Action {
	case protocol.LocationAliasActionAdd:
		// Validate the value
		if payload.Value == "" {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No value specified for add action")
		}

		// Add the alias
		if err := ws.handler.LocationAliasAdd(payload.Alias, payload.Value); err != nil {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Error adding location alias: %v", err)
		}

		// Broadcast location settings changed notification
		changedPayload := protocol.LocationSettingsChangedPayload{
			ChangeType: protocol.LocationSettingsChangeTypeAliasAdded,
			Alias:      payload.Alias,
			Value:      payload.Value,
		}
		_ = ws.broadcastMessageToClients(protocol.MessageTypeLocationSettingsChanged, changedPayload)

		return SuccessResponse(nil)

	case protocol.LocationAliasActionUpdate:
		// Validate the value
		if payload.Value == "" {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No value specified for update action")
		}

		// Update the alias
		if err := ws.handler.LocationAliasUpdate(payload.Alias, payload.Value); err != nil {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Error updating location alias: %v", err)
		}

		// Broadcast location settings changed notification
		changedPayload := protocol.LocationSettingsChangedPayload{
			ChangeType: protocol.LocationSettingsChangeTypeAliasUpdated,
			Alias:      payload.Alias,
			Value:      payload.Value,
		}
		_ = ws.broadcastMessageToClients(protocol.MessageTypeLocationSettingsChanged, changedPayload)

		return SuccessResponse(nil)

	case protocol.LocationAliasActionDelete:
		// Delete the alias
		if err := ws.handler.LocationAliasDelete(payload.Alias); err != nil {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Error deleting location alias: %v", err)
		}

		// Broadcast location settings changed notification
		changedPayload := protocol.LocationSettingsChangedPayload{
			ChangeType: protocol.LocationSettingsChangeTypeAliasDeleted,
			Alias:      payload.Alias,
		}
		_ = ws.broadcastMessageToClients(protocol.MessageTypeLocationSettingsChanged, changedPayload)

		return SuccessResponse(nil)

	default:
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Unknown location alias action: %s", payload.Action)
	}
}

// handleSetLocationOrderFromClient handles a set_location_order message from a client
func (ws *WebSocketServer) handleSetLocationOrderFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// Parse the payload
	var payload protocol.SetLocationOrderPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing set_location_order payload: %v", err)
	}

	// Set the location order (empty array means reset)
	if err := ws.handler.SetLocationOrder(payload.Order); err != nil {
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error setting location order: %v", err)
	}

	// Broadcast location settings changed notification
	changedPayload := protocol.LocationSettingsChangedPayload{
		ChangeType: protocol.LocationSettingsChangeTypeOrderChanged,
		Order:      payload.Order,
	}
	_ = ws.broadcastMessageToClients(protocol.MessageTypeLocationSettingsChanged, changedPayload)

	return SuccessResponse(nil)
}
