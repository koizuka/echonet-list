package server

import (
	"bytes"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/log"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// handleGetPropertiesFromClient handles a get_properties message from a client
func (ws *WebSocketServer) handleGetPropertiesFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// Parse the payload
	var payload protocol.GetPropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing get_properties payload: %v", err)
	}

	// Validate the payload
	if len(payload.Targets) == 0 {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No targets specified")
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
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
		}

		// Parse EPCs
		epcs := make([]echonet_lite.EPCType, 0, len(payload.EPCs))
		for _, epcStr := range payload.EPCs {
			epc, err := echonet_lite.ParseEPCString(epcStr)
			if err != nil {
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid EPC: %v", err)
			}
			epcs = append(epcs, epc)
		}

		// Get properties
		deviceAndProps, err := ws.echonetClient.GetProperties(ipAndEOJ, epcs, false)
		if err != nil {
			return ErrorResponse(protocol.ErrorCodeEchonetCommunicationError, "Error getting properties: %v", err)
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
			return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error marshaling device: %v", err)
		}
		resultJSON = deviceJSON
	}

	// Send the success response with the device data
	return SuccessResponse(resultJSON)
}

// handleSetPropertiesFromClient handles a set_properties message from a client
func (ws *WebSocketServer) handleSetPropertiesFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// Parse the payload
	var payload protocol.SetPropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing set_properties payload: %v", err)
	}

	// Validate the payload
	if payload.Target == "" {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No target specified")
	}
	if len(payload.Properties) == 0 {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No properties specified")
	}

	// Parse the target
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.Target)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
	}

	// Parse properties
	properties := make(echonet_lite.Properties, 0, len(payload.Properties))
	for epcStr, propData := range payload.Properties {
		epc, err := echonet_lite.ParseEPCString(epcStr)
		if err != nil {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid EPC: %v", err)
		}
		desc, ok := echonet_lite.GetPropertyDesc(ipAndEOJ.EOJ.ClassCode(), epc)
		if !ok {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Unknown property EPC: %s", epcStr)
		}

		var edtBytes []byte
		var stringBytes []byte

		if propData.EDT != "" {
			decoded, err := base64.StdEncoding.DecodeString(propData.EDT)
			if err != nil {
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid EDT: %v", err)
			}
			edtBytes = decoded
		}
		if propData.String != "" {
			converted, ok := desc.ToEDT(propData.String)
			if !ok {
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid string value: %s", propData.String)
			}
			stringBytes = converted
		}
		switch {
		case edtBytes == nil && stringBytes == nil:
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No EDT or string specified for EPC: %s", epcStr)
		case edtBytes != nil && stringBytes != nil:
			if !bytes.Equal(edtBytes, stringBytes) {
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Conflicting EDT and string for EPC: %s", epcStr)
			}
		case edtBytes == nil && stringBytes != nil:
			edtBytes = stringBytes
		}

		properties = append(properties, echonet_lite.Property{
			EPC: epc,
			EDT: edtBytes,
		})
	}

	// Set properties
	deviceAndProps, err := ws.echonetClient.SetProperties(ipAndEOJ, properties)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeEchonetCommunicationError, "Error setting properties: %v", err)
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
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error marshaling device data: %v", err)
	}

	// Send the success response with device data
	return SuccessResponse(deviceDataJSON)
}

// handleGetPropertyAliasesFromClient handles a get_property_aliases message from a client
func (ws *WebSocketServer) handleGetPropertyAliasesFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	logger := log.GetLogger()

	// Parse the payload
	var payload protocol.GetPropertyAliasesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if logger != nil {
			logger.Log("Error parsing get_property_aliases payload: %v", err)
		}
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing get_property_aliases payload: %v", err)
	}

	var classCode echonet_lite.EOJClassCode
	var err error

	// classCodeが空文字列の場合は共通プロパティを要求すると解釈
	if payload.ClassCode == "" {
		classCode = 0 // 共通プロパティを示すゼロ値 (ProfileSuperClass)
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
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid class code: %v", err)
		}
		if logger != nil && ws.handler.IsDebug() {
			logger.Log("Requesting property aliases for class code: %s", payload.ClassCode)
		}
	}

	// Get property aliases from echonetClient using the determined classCode
	aliases := ws.echonetClient.AvailablePropertyAliases(classCode)

	// Convert to protocol format
	propertiesMap := make(map[string]protocol.EPCDesc)

	// Process each alias description
	for aliasName, desc := range aliases {
		epc := desc.EPC.String() // Get EPC as hex string (e.g., "80")

		// Get or create EPCDesc
		epcDesc, exists := propertiesMap[epc]
		if !exists {
			epcDesc = protocol.EPCDesc{
				Description: desc.Name, // Use the description from the alias definition
				Aliases:     make(map[string]string),
			}
		}

		// Add the alias name and its Base64 encoded EDT
		epcDesc.Aliases[aliasName] = base64.StdEncoding.EncodeToString(desc.EDT)
		propertiesMap[epc] = epcDesc // Update the map
	}

	// Create the data part of the response payload
	data := protocol.PropertyAliasesData{
		ClassCode:  payload.ClassCode, // Use the requested class code
		Properties: propertiesMap,
	}

	// Marshal the data part to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		if logger != nil {
			logger.Log("Error marshaling property aliases data: %v", err)
		}
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error marshaling property aliases data: %v", err)
	}

	// Return the success response with the marshaled data
	return SuccessResponse(dataJSON)
}

// handleUpdatePropertiesFromClient handles an update_properties message from a client
func (ws *WebSocketServer) handleUpdatePropertiesFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// Parse the payload
	var payload protocol.UpdatePropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing update_properties payload: %v", err)
	}

	// フィルター基準のリストを準備
	var filterCriteriaList []echonet_lite.FilterCriteria

	if len(payload.Targets) == 0 {
		// targetsが空の場合、全デバイスを対象とするフィルター基準を追加
		filterCriteriaList = append(filterCriteriaList, echonet_lite.FilterCriteria{})
	} else {
		// targetsが指定されている場合、各ターゲットに対応するフィルター基準を作成
		filterCriteriaList = make([]echonet_lite.FilterCriteria, 0, len(payload.Targets)) // スライスを事前に確保
		for _, target := range payload.Targets {
			ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(target)
			if err != nil {
				// エラーが発生した場合、エラー応答を返す
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
			}
			// 各デバイスに対応するフィルター基準を作成
			criteria := echonet_lite.FilterCriteria{
				Device: echonet_lite.DeviceSpecifierFromIPAndEOJ(ipAndEOJ),
			}
			filterCriteriaList = append(filterCriteriaList, criteria)
		}
	}

	// 各フィルター基準に基づいてプロパティを更新
	var firstError error
	for _, criteria := range filterCriteriaList {
		// Update properties based on the criteria
		if err := ws.echonetClient.UpdateProperties(criteria, payload.Force); err != nil {
			// エラーが発生しても処理を継続し、最初のエラーを記録
			if firstError == nil {
				// エラーメッセージにcriteriaを含める (%v を使用)
				firstError = fmt.Errorf("error updating properties for criteria '%v': %w", criteria, err)
			}
			// TODO: Consider logging the error here using log package if needed
			// log.Printf("Error updating properties for criteria %v: %v", criteria, err)
		}
	}

	// 処理中にエラーが発生した場合はエラー応答を返す
	if firstError != nil {
		return ErrorResponse(protocol.ErrorCodeEchonetCommunicationError, firstError.Error())
	}

	// Send the success response
	return SuccessResponse(nil)
}
