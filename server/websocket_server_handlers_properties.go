package server

import (
	"bytes"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// handleGetPropertiesFromClient handles a get_properties message from a client
func (ws *WebSocketServer) handleGetPropertiesFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// 操作追跡を開始
	operationID := "get_properties_" + time.Now().Format("20060102_150405.000")

	// Parse the payload
	var payload protocol.GetPropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing get_properties payload: %v", err)
	}

	// Validate the payload
	if len(payload.Targets) == 0 {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No targets specified")
	}

	// ECHONETクライアントからOperationTrackerを取得
	if tracker := ws.getOperationTracker(); tracker != nil {
		tracker.StartOperation(operationID, handler.OperationTypeGetProperties,
			fmt.Sprintf("Get properties for %d targets", len(payload.Targets)),
			map[string]interface{}{
				"source":       "websocket",
				"target_count": len(payload.Targets),
				"epc_count":    len(payload.EPCs),
			})

		defer func() {
			tracker.CompleteOperation(operationID, true, nil)
		}()
	}

	// Process each target
	results := make([]protocol.Device, 0, len(payload.Targets))
	for _, target := range payload.Targets {
		// Parse the target
		ipAndEOJ, err := handler.ParseDeviceIdentifier(target)
		if ws.handler.IsDebug() {
			slog.Debug("Processing target", "target", target, "ipAndEOJ", ipAndEOJ) // DEBUG
		}

		if err != nil {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
		}

		// Parse EPCs
		epcs := make([]echonet_lite.EPCType, 0, len(payload.EPCs))
		for _, epcStr := range payload.EPCs {
			epc, err := handler.ParseEPCString(epcStr)
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
		// Check if device is offline
		var isOffline bool
		if ws.handler != nil {
			isOffline = ws.handler.IsOffline(deviceAndProps.Device)
		}
		protoDevice := protocol.DeviceToProtocol(
			deviceAndProps.Device,
			deviceAndProps.Properties,
			lastSeen,
			isOffline,
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
	// 操作追跡を開始
	operationID := "set_properties_" + time.Now().Format("20060102_150405.000")

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

	// ECHONETクライアントからOperationTrackerを取得
	if tracker := ws.getOperationTracker(); tracker != nil {
		tracker.StartOperation(operationID, handler.OperationTypeSetProperties,
			fmt.Sprintf("Set properties for target %s", payload.Target),
			map[string]interface{}{
				"source":         "websocket",
				"target":         payload.Target,
				"property_count": len(payload.Properties),
			})

		defer func() {
			tracker.CompleteOperation(operationID, true, nil)
		}()
	}

	// Parse the target
	ipAndEOJ, err := handler.ParseDeviceIdentifier(payload.Target)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
	}

	// Parse properties
	properties := make(echonet_lite.Properties, 0, len(payload.Properties))
	for epcStr, propData := range payload.Properties {
		epc, err := handler.ParseEPCString(epcStr)
		if err != nil {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid EPC: %v", err)
		}
		desc, ok := echonet_lite.GetPropertyDesc(ipAndEOJ.EOJ.ClassCode(), epc)
		if !ok {
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Unknown property EPC: %s", epcStr)
		}

		var edtBytes []byte
		var valueBytes []byte

		if propData.EDT != "" {
			decoded, err := base64.StdEncoding.DecodeString(propData.EDT)
			if err != nil {
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid EDT: %v", err)
			}
			edtBytes = decoded
		}
		switch {
		case propData.String != "" && propData.Number != 0:
			// StringとNumberの両方があったらエラー
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Conflicting string and number for EPC: %s", epcStr)
		case propData.Number != 0:
			converter, ok := desc.Decoder.(echonet_lite.PropertyIntConverter)
			if !ok {
				// 数値に対応していないEPCに数値が与えられたエラー
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid number field for EPC %s", epcStr)
			}
			converted, ok := converter.FromInt(propData.Number)
			if !ok {
				// 装置が範囲外
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid number value for EPC %s", epcStr)
			}
			valueBytes = converted

		case propData.String != "":
			converted, ok := desc.ToEDT(propData.String)
			if !ok {
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid string value: %s", propData.String)
			}
			valueBytes = converted
		}

		switch {
		case edtBytes == nil && valueBytes == nil:
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "No EDT or string specified for EPC: %s", epcStr)
		case edtBytes != nil && valueBytes != nil:
			if !bytes.Equal(edtBytes, valueBytes) {
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Conflicting EDT and string for EPC: %s", epcStr)
			}
		case edtBytes == nil && valueBytes != nil:
			edtBytes = valueBytes
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

	// Check if any of the set properties have TriggerUpdate flag
	for _, prop := range properties {
		desc, ok := echonet_lite.GetPropertyDesc(ipAndEOJ.EOJ.ClassCode(), prop.EPC)
		if ok && desc.TriggerUpdate {
			// Launch a goroutine to update properties after the specified delay
			go func(device handler.IPAndEOJ, delay time.Duration, targets []echonet_lite.EPCType) {
				// Wait for the delay or until context is cancelled
				select {
				case <-ws.timeProvider.After(delay):
					// Continue with the update
				case <-ws.ctx.Done():
					// Context was cancelled, abort the update
					if ws.handler.IsDebug() {
						slog.Debug("Property update cancelled due to context cancellation",
							"device", device.Specifier())
					}
					return
				}

				// Create filter criteria for this specific device
				classCode := device.EOJ.ClassCode()
				instanceCode := device.EOJ.InstanceCode()
				criteria := handler.FilterCriteria{
					Device: handler.DeviceSpecifier{
						IP:           &device.IP,
						ClassCode:    &classCode,
						InstanceCode: &instanceCode,
					},
				}

				// Log the update trigger
				if ws.handler.IsDebug() {
					slog.Debug("Triggering property update due to TriggerUpdate flag",
						"device", device.Specifier(),
						"delay", delay,
						"targetCount", len(targets))
				}

				// Force update to bypass the update interval threshold
				if err := ws.echonetClient.UpdateProperties(criteria, true); err != nil {
					slog.Info("Failed to update properties after trigger",
						"device", device.Specifier(),
						"error", err)
				}
			}(ipAndEOJ, desc.UpdateDelay, desc.UpdateTargets)
		}
	}

	// デバイスの最終更新タイムスタンプを取得
	lastSeen := ws.handler.GetLastUpdateTime(deviceAndProps.Device)

	// Use DeviceToProtocol to convert to protocol format
	// Check if device is offline
	var isOffline bool
	if ws.handler != nil {
		isOffline = ws.handler.IsOffline(deviceAndProps.Device)
	}
	deviceData := protocol.DeviceToProtocol(
		deviceAndProps.Device,
		deviceAndProps.Properties,
		lastSeen,
		isOffline,
	)

	// Marshal the device data
	deviceDataJSON, err := json.Marshal(deviceData)
	if err != nil {
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error marshaling device data: %v", err)
	}

	// Send the success response with device data
	return SuccessResponse(deviceDataJSON)
}

// populateEPCDescriptions converts echonet_lite property descriptions to protocol EPC descriptions
func populateEPCDescriptions(propTable echonet_lite.PropertyTable, targetMap map[string]protocol.EPCDesc, lang string) {
	for epc, propDesc := range propTable.EPCDesc {
		epcStr := epc.String()
		epcDesc := protocol.EPCDesc{
			Description: propDesc.GetName(lang),
			Aliases:     make(map[string]string),
		}
		// Add aliases if they exist (always use English aliases)
		if propDesc.Aliases != nil {
			for aliasName, edtBytes := range propDesc.Aliases {
				epcDesc.Aliases[aliasName] = base64.StdEncoding.EncodeToString(edtBytes)
			}
		}
		if len(epcDesc.Aliases) == 0 {
			epcDesc.Aliases = nil // Omit empty map in JSON
		}
		// Add alias translations if they exist for the requested language
		if translations := propDesc.GetAliasTranslations(lang); translations != nil {
			epcDesc.AliasTranslations = translations
		}
		// Check decoder type and populate protocol-specific descriptions
		if propDesc.Decoder != nil {
			switch v := propDesc.Decoder.(type) {
			case echonet_lite.NumberDesc:
				protoNumDesc := &protocol.ProtocolNumberDesc{
					Min:    v.Min,
					Max:    v.Max,
					Offset: v.Offset,
					Unit:   v.Unit,
					EdtLen: v.EDTLen,
				}
				if protoNumDesc.EdtLen == 1 || protoNumDesc.EdtLen == 0 {
					protoNumDesc.EdtLen = 0 // Use omitempty
				}
				epcDesc.NumberDesc = protoNumDesc
			case echonet_lite.StringDesc:
				protoStrDesc := &protocol.ProtocolStringDesc{
					MinEDTLen: v.MinEDTLen,
					MaxEDTLen: v.MaxEDTLen,
				}
				epcDesc.StringDesc = protoStrDesc
			}
			if _, ok := propDesc.Decoder.(echonet_lite.PropertyEncoder); ok {
				// If the decoder is a PropertyEncoder, it means it's settable
				epcDesc.StringSettable = true
			}
		}
		targetMap[epcStr] = epcDesc // Add or overwrite in the target map
	}
}

// handleGetPropertyDescriptionFromClient handles a get_property_description message from a client
func (ws *WebSocketServer) handleGetPropertyDescriptionFromClient(msg *protocol.Message) protocol.CommandResultPayload {
	// Parse the payload
	var payload protocol.GetPropertyDescriptionPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		slog.Error("Error parsing get_property_description payload", "err", err)
		return ErrorResponse(protocol.ErrorCodeInvalidRequestFormat, "Error parsing get_property_description payload: %v", err)
	}

	var classCode echonet_lite.EOJClassCode
	var err error

	// classCodeが空文字列の場合は共通プロパティを要求すると解釈
	if payload.ClassCode == "" {
		classCode = 0 // 共通プロパティを示すゼロ値 (ProfileSuperClass)
		if ws.handler.IsDebug() {
			slog.Debug("Requesting common property descriptions (classCode is empty)")
		}
	} else {
		// Parse the class code if not empty
		classCode, err = handler.ParseEOJClassCodeString(payload.ClassCode)
		if err != nil {
			slog.Error("Error: invalid class code", "err", err)
			return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid class code: %v", err)
		}
		if ws.handler.IsDebug() {
			slog.Debug("Requesting property descriptions for class code", "classCode", payload.ClassCode)
		}
	}

	// Convert to protocol format
	propertiesMap := make(map[string]protocol.EPCDesc)

	// Get language from payload, default to empty string (which will use English)
	lang := payload.Lang

	// Populate common properties first
	populateEPCDescriptions(echonet_lite.ProfileSuperClass_PropertyTable, propertiesMap, lang)

	// Populate specific class properties (overwriting common ones if necessary)
	// Only process if classCode is specified (i.e., not empty request)
	if payload.ClassCode != "" {
		if classTable, ok := echonet_lite.PropertyTables[classCode]; ok {
			populateEPCDescriptions(classTable, propertiesMap, lang)
		} else {
			// Log if the specific class table wasn't found, but still return common properties
			slog.Warn("Property table not found for specific class code", "classCode", payload.ClassCode)
		}
	}

	// Create the data part of the response payload
	data := protocol.PropertyDescriptionData{
		ClassCode:  payload.ClassCode, // Use the requested class code
		Properties: propertiesMap,
	}

	// Marshal the data part to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		slog.Error("Error marshaling property description data", "err", err)
		return ErrorResponse(protocol.ErrorCodeInternalServerError, "Error marshaling property description data: %v", err)
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
	var filterCriteriaList []handler.FilterCriteria

	if len(payload.Targets) == 0 {
		// targetsが空の場合、全デバイスを対象とするフィルター基準を追加
		filterCriteriaList = append(filterCriteriaList, handler.FilterCriteria{})
	} else {
		// targetsが指定されている場合、各ターゲットに対応するフィルター基準を作成
		filterCriteriaList = make([]handler.FilterCriteria, 0, len(payload.Targets)) // スライスを事前に確保
		for _, target := range payload.Targets {
			ipAndEOJ, err := handler.ParseDeviceIdentifier(target)
			if err != nil {
				// エラーが発生した場合、エラー応答を返す
				return ErrorResponse(protocol.ErrorCodeInvalidParameters, "Invalid target: %v", err)
			}
			// 各デバイスに対応するフィルター基準を作成
			criteria := handler.FilterCriteria{
				Device: handler.DeviceSpecifierFromIPAndEOJ(ipAndEOJ),
			}
			filterCriteriaList = append(filterCriteriaList, criteria)
		}
	}

	// 各フィルター基準に基づいてプロパティを更新
	var firstError error
	// 操作追跡を開始
	operationID := "update_properties_" + time.Now().Format("20060102_150405.000")

	// ECHONETクライアントからOperationTrackerを取得
	if tracker := ws.getOperationTracker(); tracker != nil {
		tracker.StartOperation(operationID, handler.OperationTypeUpdateProperties,
			fmt.Sprintf("Property update for %d criteria", len(filterCriteriaList)),
			map[string]interface{}{
				"source":         "websocket",
				"criteria_count": len(filterCriteriaList),
				"force":          payload.Force,
			})

		// 各フィルター基準に基づいてプロパティを更新
		for i, criteria := range filterCriteriaList {
			if err := ws.echonetClient.UpdateProperties(criteria, payload.Force); err != nil {
				// エラーが発生しても処理を継続し、最初のエラーを記録
				if firstError == nil {
					firstError = fmt.Errorf("error updating properties for criteria '%v': %w", criteria, err)
				}
				slog.Debug("Property update failed for criteria", "index", i, "criteria", criteria, "error", err)
			}
		}

		// 操作完了を記録
		tracker.CompleteOperation(operationID, firstError == nil, firstError)
	} else {
		// フォールバック: 従来のログ方式
		start := time.Now()

		for i, criteria := range filterCriteriaList {
			// Update properties based on the criteria
			criteriaStart := time.Now()
			slog.Debug("Updating properties for criteria", "index", i, "criteria", criteria)

			if err := ws.echonetClient.UpdateProperties(criteria, payload.Force); err != nil {
				criteriaDuration := time.Since(criteriaStart)
				slog.Error("Property update failed for criteria", "index", i, "criteria", criteria, "duration", criteriaDuration, "error", err)

				// エラーが発生しても処理を継続し、最初のエラーを記録
				if firstError == nil {
					// エラーメッセージにcriteriaを含める (%v を使用)
					firstError = fmt.Errorf("error updating properties for criteria '%v': %w", criteria, err)
				}
			} else {
				criteriaDuration := time.Since(criteriaStart)
				slog.Debug("Property update completed for criteria", "index", i, "criteria", criteria, "duration", criteriaDuration)
			}
		}

		duration := time.Since(start)
		if firstError != nil {
			slog.Error("Property update completed with errors", "duration", duration, "first_error", firstError)
		} else {
			slog.Info("Property update completed successfully", "duration", duration)
		}
	}

	// 処理中にエラーが発生した場合はエラー応答を返す
	if firstError != nil {
		return ErrorResponse(protocol.ErrorCodeEchonetCommunicationError, firstError.Error())
	}

	// Send the success response
	return SuccessResponse(nil)
}
