package server

import (
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/log"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
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

// handleGetPropertyAliasesFromClient handles a get_property_aliases message from a client
func (ws *WebSocketServer) handleGetPropertyAliasesFromClient(connID string, msg *protocol.Message) error {
	logger := log.GetLogger()

	// Parse the payload
	var payload protocol.GetPropertyAliasesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if logger != nil {
			logger.Log("Error parsing get_property_aliases payload: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.PropertyAliasesResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidRequestFormat,
				Message: fmt.Sprintf("Error parsing get_property_aliases payload: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypePropertyAliasesResult, errorPayload, msg.RequestID)
	}

	// Validate the payload
	if payload.ClassCode == "" {
		if logger != nil {
			logger.Log("Error: no class code specified")
		}
		// エラー応答を送信
		errorPayload := protocol.PropertyAliasesResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: "No class code specified",
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypePropertyAliasesResult, errorPayload, msg.RequestID)
	}

	// Parse the class code
	classCode, err := echonet_lite.ParseEOJClassCodeString(payload.ClassCode)
	if err != nil {
		if logger != nil {
			logger.Log("Error: invalid class code: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.PropertyAliasesResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidParameters,
				Message: fmt.Sprintf("Invalid class code: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypePropertyAliasesResult, errorPayload, msg.RequestID)
	}

	// Get property aliases from client
	aliases := ws.echonetClient.AvailablePropertyAliases(classCode)

	// Convert to protocol format
	propertiesMap := make(map[string]protocol.EPCInfo)

	// Process each alias
	for alias, desc := range aliases {
		// Parse EPC and EDT from description (format: "EPC(説明):EDT")
		parts := strings.Split(desc, ":")
		if len(parts) != 2 {
			continue
		}

		// EPCとその説明部分を取得
		epcPart := parts[0]
		// 括弧の位置を見つける
		openParenIndex := strings.Index(epcPart, "(")
		closeParenIndex := strings.Index(epcPart, ")")
		if openParenIndex == -1 || closeParenIndex == -1 || closeParenIndex <= openParenIndex {
			continue
		}

		// EPCは括弧の前の部分
		epc := epcPart[:openParenIndex]
		// 説明は括弧の中の部分
		description := epcPart[openParenIndex+1 : closeParenIndex]

		// EDTを解析
		edt, err := echonet_lite.ParseHexString(parts[1])
		if err != nil {
			continue
		}

		// EPCInfoを取得または作成
		epcInfo, exists := propertiesMap[epc]
		if !exists {
			epcInfo = protocol.EPCInfo{
				Description: description,
				Aliases:     make(map[string]string),
			}
		}

		// エイリアスを追加
		epcInfo.Aliases[alias] = base64.StdEncoding.EncodeToString(edt)
		propertiesMap[epc] = epcInfo
	}

	// Create response payload
	resultPayload := protocol.PropertyAliasesResultPayload{
		Success: true,
		Data: &protocol.PropertyAliasesData{
			ClassCode:  payload.ClassCode,
			Properties: propertiesMap,
		},
	}

	// Send the response
	return ws.sendMessageToClient(connID, protocol.MessageTypePropertyAliasesResult, resultPayload, msg.RequestID)
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
