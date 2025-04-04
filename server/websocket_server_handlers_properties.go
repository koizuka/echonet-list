package server

import (
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/log"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// handleGetPropertiesFromClient handles a get_properties message from a client
func (ws *WebSocketServer) handleGetPropertiesFromClient(connID string, msg *protocol.Message) error {
	logger := log.GetLogger()
	// Parse the payload
	var payload protocol.GetPropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if logger != nil {
			logger.Log("Error parsing get_properties payload: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidRequestFormat,
				Message: fmt.Sprintf("Error parsing get_properties payload: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Validate the payload
	if len(payload.Targets) == 0 {
		if logger != nil {
			logger.Log("Error: no targets specified")
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "No targets specified",
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Process each target
	results := make([]protocol.Device, 0, len(payload.Targets))
	for _, target := range payload.Targets {
		// Parse the target
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(target)
		if logger != nil && ws.handler.IsDebug() {
			logger.Log("target: %v, ipAndEOJ: %v", target, ipAndEOJ) // DEBUG
		}

		if err != nil {
			if logger != nil {
				logger.Log("Error: invalid target: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInvalidParameters,
					Message: fmt.Sprintf("Invalid target: %v", err),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Parse EPCs
		epcs := make([]echonet_lite.EPCType, 0, len(payload.EPCs))
		for _, epcStr := range payload.EPCs {
			epc, err := echonet_lite.ParseEPCString(epcStr)
			if err != nil {
				if logger != nil {
					logger.Log("Error: invalid EPC: %v", err)
				}
				// エラー応答を送信
				errorPayload := protocol.CommandResultPayload{
					Success: false,
					Error: &protocol.Error{
						Code:    protocol.ErrorCodeInvalidParameters,
						Message: fmt.Sprintf("Invalid EPC: %v", err),
					},
				}
				return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
			}
			epcs = append(epcs, epc)
		}

		// Get properties
		deviceAndProps, err := ws.echonetClient.GetProperties(ipAndEOJ, epcs, false)
		if err != nil {
			if logger != nil {
				logger.Log("Error getting properties: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeEchonetCommunicationError,
					Message: err.Error(),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Use DeviceToProtocol to convert to protocol format
		protoDevice := protocol.DeviceToProtocol(
			deviceAndProps.Device.IP.String(),
			deviceAndProps.Device.EOJ,
			deviceAndProps.Properties,
			time.Now(), // Use current time as last seen
		)
		results = append(results, protoDevice)
	}

	// The client expects a single device, not an array
	// Since we're processing a single target at a time in the client's GetProperties method,
	// we should return just the first device if available
	var resultJSON json.RawMessage
	if len(results) > 0 {
		// Marshal just the first device
		deviceJSON, err := json.Marshal(results[0])
		if err != nil {
			if logger != nil {
				logger.Log("Error marshaling device: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInternalServerError,
					Message: fmt.Sprintf("Error marshaling device: %v", err),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}
		resultJSON = deviceJSON
	}

	// Send the response with the device data
	resultPayload := protocol.CommandResultPayload{
		Success: true,
		Data:    resultJSON, // Include the marshaled device (not the array)
	}

	// Send the message using the helper function
	return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)
}

// handleSetPropertiesFromClient handles a set_properties message from a client
func (ws *WebSocketServer) handleSetPropertiesFromClient(connID string, msg *protocol.Message) error {
	logger := log.GetLogger()

	// Parse the payload
	var payload protocol.SetPropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if logger != nil {
			logger.Log("Error parsing set_properties payload: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidRequestFormat,
				Message: fmt.Sprintf("Error parsing set_properties payload: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Validate the payload
	if payload.Target == "" {
		if logger != nil {
			logger.Log("Error: no target specified")
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "No target specified",
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}
	if len(payload.Properties) == 0 {
		if logger != nil {
			logger.Log("Error: no properties specified")
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "No properties specified",
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Parse the target
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.Target)
	if err != nil {
		if logger != nil {
			logger.Log("Error: invalid target: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: fmt.Sprintf("Invalid target: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Parse properties
	properties := make(echonet_lite.Properties, 0, len(payload.Properties))
	for epcStr, edtStr := range payload.Properties {
		epc, err := echonet_lite.ParseEPCString(epcStr)
		if err != nil {
			if logger != nil {
				logger.Log("Error: invalid EPC: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInvalidParameters,
					Message: fmt.Sprintf("Invalid EPC: %v", err),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		edt, err := base64.StdEncoding.DecodeString(edtStr)
		if err != nil {
			if logger != nil {
				logger.Log("Error: invalid EDT: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInvalidParameters,
					Message: fmt.Sprintf("Invalid EDT: %v", err),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		properties = append(properties, echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		})
	}

	// Set properties
	deviceAndProps, err := ws.echonetClient.SetProperties(ipAndEOJ, properties)
	if err != nil {
		if logger != nil {
			logger.Log("Error setting properties: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeEchonetCommunicationError,
				Message: err.Error(),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Use DeviceToProtocol to convert to protocol format
	deviceData := protocol.DeviceToProtocol(
		deviceAndProps.Device.IP.String(),
		deviceAndProps.Device.EOJ,
		deviceAndProps.Properties,
		time.Now(), // Use current time as last seen
	)

	// Marshal the device data
	deviceDataJSON, err := json.Marshal(deviceData)
	if err != nil {
		if logger != nil {
			logger.Log("Error marshaling device data: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInternalServerError,
				Message: fmt.Sprintf("Error marshaling device data: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Send the response with device data
	resultPayload := protocol.CommandResultPayload{
		Success: true,
		Data:    deviceDataJSON,
	}

	// Send the message
	return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)
}

// handleUpdatePropertiesFromClient handles an update_properties message from a client
func (ws *WebSocketServer) handleUpdatePropertiesFromClient(connID string, msg *protocol.Message) error {
	logger := log.GetLogger()
	// Parse the payload
	var payload protocol.UpdatePropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if logger != nil {
			logger.Log("Error parsing update_properties payload: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidRequestFormat,
				Message: fmt.Sprintf("Error parsing update_properties payload: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Validate the payload
	if len(payload.Targets) == 0 {
		if logger != nil {
			logger.Log("Error: no targets specified")
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "No targets specified",
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Process each target
	for _, target := range payload.Targets {
		// Parse the target
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(target)
		if err != nil {
			if logger != nil {
				logger.Log("Error: invalid target: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeInvalidParameters,
					Message: fmt.Sprintf("Invalid target: %v", err),
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

		// Update properties
		if err := ws.echonetClient.UpdateProperties(criteria); err != nil {
			if logger != nil {
				logger.Log("Error updating properties: %v", err)
			}
			// エラー応答を送信
			errorPayload := protocol.CommandResultPayload{
				Success: false,
				Error: &protocol.Error{
					Code:    protocol.ErrorCodeEchonetCommunicationError,
					Message: err.Error(),
				},
			}
			return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}
	}

	// Send the response
	resultPayload := protocol.CommandResultPayload{
		Success: true,
	}

	// Send the message
	return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)
}
