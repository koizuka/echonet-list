package server

import (
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/log"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// handleGetPropertiesFromClient handles a get_properties message from a client
func (ws *WebSocketServer) handleGetPropertiesFromClient(connID string, msg *protocol.Message) error {
	// Parse the payload
	var payload protocol.GetPropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidRequestFormat, "Error parsing get_properties payload: %v", err)
	}

	// Validate the payload
	if len(payload.Targets) == 0 {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "No targets specified")
	}

	// Process each target
	results := make([]protocol.Device, 0, len(payload.Targets))
	for _, target := range payload.Targets {
		// Parse the target
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(target)
		logger := log.GetLogger()
		if logger != nil && ws.handler.IsDebug() {
			logger.Log("target: %v, ipAndEOJ: %v", target, ipAndEOJ) // DEBUG
		}

		if err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
		}

		// Parse EPCs
		epcs := make([]echonet_lite.EPCType, 0, len(payload.EPCs))
		for _, epcStr := range payload.EPCs {
			epc, err := echonet_lite.ParseEPCString(epcStr)
			if err != nil {
				return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Invalid EPC: %v", err)
			}
			epcs = append(epcs, epc)
		}

		// Get properties
		deviceAndProps, err := ws.echonetClient.GetProperties(ipAndEOJ, epcs, false)
		if err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeEchonetCommunicationError, "Error getting properties: %v", err)
		}

		// デバイスの最終更新タイムスタンプを取得
		lastSeen := ws.handler.GetLastUpdateTime(deviceAndProps.Device)

		// Use DeviceToProtocol to convert to protocol format
		protoDevice := protocol.DeviceToProtocol(
			deviceAndProps.Device,
			deviceAndProps.Properties,
			lastSeen,
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
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInternalServerError, "Error marshaling device: %v", err)
		}
		resultJSON = deviceJSON
	}

	// Send the success response with the device data
	return ws.sendSuccessResponse(connID, msg.RequestID, resultJSON)
}

// handleSetPropertiesFromClient handles a set_properties message from a client
func (ws *WebSocketServer) handleSetPropertiesFromClient(connID string, msg *protocol.Message) error {

	// Parse the payload
	var payload protocol.SetPropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidRequestFormat, "Error parsing set_properties payload: %v", err)
	}

	// Validate the payload
	if payload.Target == "" {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "No target specified")
	}
	if len(payload.Properties) == 0 {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "No properties specified")
	}

	// Parse the target
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.Target)
	if err != nil {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
	}

	// Parse properties
	properties := make(echonet_lite.Properties, 0, len(payload.Properties))
	for epcStr, edtStr := range payload.Properties {
		epc, err := echonet_lite.ParseEPCString(epcStr)
		if err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Invalid EPC: %v", err)
		}

		edt, err := base64.StdEncoding.DecodeString(edtStr)
		if err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Invalid EDT: %v", err)
		}

		properties = append(properties, echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		})
	}

	// Set properties
	deviceAndProps, err := ws.echonetClient.SetProperties(ipAndEOJ, properties)
	if err != nil {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeEchonetCommunicationError, "Error setting properties: %v", err)
	}

	// デバイスの最終更新タイムスタンプを取得
	lastSeen := ws.handler.GetLastUpdateTime(deviceAndProps.Device)

	// Use DeviceToProtocol to convert to protocol format
	deviceData := protocol.DeviceToProtocol(
		deviceAndProps.Device,
		deviceAndProps.Properties,
		lastSeen,
	)

	// Marshal the device data
	deviceDataJSON, err := json.Marshal(deviceData)
	if err != nil {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInternalServerError, "Error marshaling device data: %v", err)
	}

	// Send the success response with device data
	return ws.sendSuccessResponse(connID, msg.RequestID, deviceDataJSON)
}

// handleGetPropertyAliasesFromClient handles a get_property_aliases message from a client
func (ws *WebSocketServer) handleGetPropertyAliasesFromClient(connID string, msg *protocol.Message) error {
	logger := log.GetLogger()

	// Parse the payload
	var payload protocol.GetPropertyAliasesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		// PropertyAliasesResultPayloadを使用する必要があるため、sendErrorResponseは使用できない
		if logger != nil {
			logger.Log("Error parsing get_property_aliases payload: %v", err)
		}
		errorPayload := protocol.PropertyAliasesResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeInvalidRequestFormat,
				Message: fmt.Sprintf("Error parsing get_property_aliases payload: %v", err),
			},
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypePropertyAliasesResult, errorPayload, msg.RequestID)
	}

	var classCode echonet_lite.EOJClassCode
	var err error

	// classCodeが空文字列の場合は共通プロパティを要求すると解釈
	if payload.ClassCode == "" {
		classCode = 0 // 共通プロパティを示すゼロ値
		if logger != nil && ws.handler.IsDebug() {
			logger.Log("Requesting common property aliases (classCode is empty)")
		}
	} else {
		// Parse the class code if not empty
		classCode, err = echonet_lite.ParseEOJClassCodeString(payload.ClassCode)
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
		if logger != nil && ws.handler.IsDebug() {
			logger.Log("Requesting property aliases for class code: %s", payload.ClassCode)
		}
	}

	// Get property aliases from client using the determined classCode
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
	// Parse the payload
	var payload protocol.UpdatePropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidRequestFormat, "Error parsing update_properties payload: %v", err)
	}

	// Validate the payload
	if len(payload.Targets) == 0 {
		return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "No targets specified")
	}

	// Process each target
	for _, target := range payload.Targets {
		// Parse the target
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(target)
		if err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
		}

		// Create filter criteria
		criteria := echonet_lite.FilterCriteria{
			Device: echonet_lite.DeviceSpecifierFromIPAndEOJ(ipAndEOJ),
		}

		// Update properties
		if err := ws.echonetClient.UpdateProperties(criteria); err != nil {
			return ws.sendErrorResponse(connID, msg.RequestID, protocol.ErrorCodeEchonetCommunicationError, "Error updating properties: %v", err)
		}
	}

	// Send the success response
	return ws.sendSuccessResponse(connID, msg.RequestID, nil)
}
