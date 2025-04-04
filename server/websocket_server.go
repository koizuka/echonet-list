package server

import (
	"context"
	"echonet-list/client"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/log"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// StartOptions は WebSocketServer の起動オプションを表す
type StartOptions struct {
	// TLS証明書ファイルのパス (TLSを使用する場合)
	CertFile string
	// TLS秘密鍵ファイルのパス (TLSを使用する場合)
	KeyFile string
}

// WebSocketServer implements a WebSocket server for ECHONET Lite
type WebSocketServer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	server        *http.Server
	upgrader      websocket.Upgrader
	clients       map[*websocket.Conn]bool
	clientsMutex  sync.RWMutex
	echonetClient client.ECHONETListClient
	handler       *echonet_lite.ECHONETLiteHandler
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(ctx context.Context, addr string, echonetClient client.ECHONETListClient, handler *echonet_lite.ECHONETLiteHandler) (*WebSocketServer, error) {
	serverCtx, cancel := context.WithCancel(ctx)

	// Create the WebSocket server
	ws := &WebSocketServer{
		ctx:           serverCtx,
		cancel:        cancel,
		upgrader:      websocket.Upgrader{},
		clients:       make(map[*websocket.Conn]bool),
		echonetClient: echonetClient,
		handler:       handler,
	}

	// Create the HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", ws.handleWebSocket)

	ws.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start listening for notifications from the ECHONET Lite handler
	go ws.listenForNotifications()

	return ws, nil
}

// Start starts the WebSocket server
func (ws *WebSocketServer) Start(options StartOptions) error {
	logger := log.GetLogger()
	if logger != nil {
		logger.Log("WebSocket server starting on %s", ws.server.Addr)
	} else {
		fmt.Printf("WebSocket server starting on %s\n", ws.server.Addr)
	}

	// TLS証明書が指定されている場合
	if options.CertFile != "" && options.KeyFile != "" {
		if logger != nil {
			logger.Log("Using TLS with certificate: %s", options.CertFile)
		} else {
			fmt.Printf("Using TLS with certificate: %s\n", options.CertFile)
		}
		return ws.server.ListenAndServeTLS(options.CertFile, options.KeyFile)
	}

	// 通常のHTTP (証明書なし)
	return ws.server.ListenAndServe()
}

// Stop stops the WebSocket server
func (ws *WebSocketServer) Stop() error {
	logger := log.GetLogger()
	if logger != nil {
		logger.Log("Stopping WebSocket server")
	}
	ws.cancel()
	err := ws.server.Shutdown(context.Background())
	if err != nil && logger != nil {
		logger.Log("Error shutting down WebSocket server: %v", err)
	}
	return err
}

// handleWebSocket handles WebSocket connections
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	logger := log.GetLogger()

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if logger != nil {
			logger.Log("Error upgrading to WebSocket: %v", err)
		}
		return
	}
	defer conn.Close()

	if logger != nil && ws.handler.IsDebug() {
		logger.Log("New WebSocket connection established")
	}

	// Register the client
	ws.clientsMutex.Lock()
	ws.clients[conn] = true
	ws.clientsMutex.Unlock()

	// Remove the client when the function returns
	defer func() {
		ws.clientsMutex.Lock()
		delete(ws.clients, conn)
		ws.clientsMutex.Unlock()
		if logger != nil && ws.handler.IsDebug() {
			logger.Log("WebSocket connection closed")
		}
	}()

	// Send initial state to the client
	if err := ws.sendInitialState(conn); err != nil {
		if logger != nil {
			logger.Log("Error sending initial state: %v", err)
		}
		return
	}

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				if logger != nil {
					logger.Log("WebSocket error: %v", err)
				}
			}
			break
		}

		// Parse and handle the message
		if err := ws.handleMessage(conn, message); err != nil {
			if logger != nil {
				logger.Log("Error handling message: %v", err)
			}
			// エラーはハンドラ関数内で処理されるはずなので、ここでは何もしない
			// 未知のメッセージタイプなど、ハンドラ関数に到達する前のエラーのみここで処理される
		}
	}
}

// sendInitialState sends the initial state to a client
func (ws *WebSocketServer) sendInitialState(conn *websocket.Conn) error {
	logger := log.GetLogger()

	if logger != nil && ws.handler.IsDebug() {
		logger.Log("Sending initial state to client")
	}

	// Get all devices
	devices := ws.echonetClient.ListDevices(echonet_lite.FilterCriteria{})

	// Convert devices to protocol format
	protoDevices := make(map[string]protocol.Device)
	for _, device := range devices {
		// Use DeviceToProtocol to convert to protocol format
		protoDevice := protocol.DeviceToProtocol(
			device.Device.IP.String(),
			device.Device.EOJ,
			device.Properties,
			time.Now(), // Use current time as last seen
		)

		// Add to map with device identifier as key
		protoDevices[device.Device.Specifier()] = protoDevice
	}

	// Get all aliases
	aliasList := ws.echonetClient.AliasList()
	aliases := make(map[string]string)
	for _, alias := range aliasList {
		aliases[alias.Alias] = alias.Device.Specifier()
	}

	// Get all groups
	groupList := ws.echonetClient.GroupList(nil)
	groups := make(map[string][]string)
	for _, group := range groupList {
		deviceStrs := make([]string, 0, len(group.Devices))
		for _, device := range group.Devices {
			deviceStrs = append(deviceStrs, device.Specifier())
		}
		groups[group.Group] = deviceStrs
	}

	// Create initial state payload
	payload := protocol.InitialStatePayload{
		Devices: protoDevices,
		Aliases: aliases,
		Groups:  groups,
	}

	// Send the message
	if err := ws.sendMessage(conn, protocol.MessageTypeInitialState, payload, ""); err != nil {
		if logger != nil {
			logger.Log("Error sending initial state: %v", err)
		}
		return err
	}

	if logger != nil && ws.handler.IsDebug() {
		logger.Log("Initial state sent successfully")
	}
	return nil
}

// handleMessage handles an incoming message from a client
func (ws *WebSocketServer) handleMessage(conn *websocket.Conn, message []byte) error {
	logger := log.GetLogger()

	// Parse the message
	msg, err := protocol.ParseMessage(message)
	if err != nil {
		if logger != nil {
			logger.Log("Error parsing message: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.ErrorNotificationPayload{
			Code:    protocol.ErrorCodeInvalidRequestFormat,
			Message: fmt.Sprintf("Error parsing message: %v", err),
		}
		return ws.sendMessage(conn, protocol.MessageTypeErrorNotification, errorPayload, "")
	}

	// Handle the message based on its type
	switch msg.Type {
	case protocol.MessageTypeGetProperties:
		return ws.handleGetProperties(conn, msg)
	case protocol.MessageTypeSetProperties:
		return ws.handleSetProperties(conn, msg)
	case protocol.MessageTypeUpdateProperties:
		return ws.handleUpdateProperties(conn, msg)
	case protocol.MessageTypeManageAlias:
		return ws.handleManageAlias(conn, msg)
	case protocol.MessageTypeManageGroup:
		return ws.handleManageGroup(conn, msg)
	case protocol.MessageTypeDiscoverDevices:
		return ws.handleDiscoverDevices(conn, msg)
	default:
		if logger != nil {
			logger.Log("Unknown message type: %s", msg.Type)
		}
		// エラー応答を送信
		errorPayload := protocol.ErrorNotificationPayload{
			Code:    protocol.ErrorCodeInvalidRequestFormat,
			Message: fmt.Sprintf("Unknown message type: %s", msg.Type),
		}
		return ws.sendMessage(conn, protocol.MessageTypeErrorNotification, errorPayload, msg.RequestID)
	}
}

// handleGetProperties handles a get_properties message
func (ws *WebSocketServer) handleGetProperties(conn *websocket.Conn, msg *protocol.Message) error {
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
				return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}
		resultJSON = deviceJSON
	}

	// Send the response with the device data
	resultPayload := protocol.CommandResultPayload{
		Success: true,
		Data:    resultJSON, // Include the marshaled device (not the array)
	}

	// Send the message using the helper function
	return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)
}

// handleSetProperties handles a set_properties message
func (ws *WebSocketServer) handleSetProperties(conn *websocket.Conn, msg *protocol.Message) error {
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Send the response with device data
	resultPayload := protocol.CommandResultPayload{
		Success: true,
		Data:    deviceDataJSON,
	}

	// Send the message
	return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)
}

// handleUpdateProperties handles an update_properties message
func (ws *WebSocketServer) handleUpdateProperties(conn *websocket.Conn, msg *protocol.Message) error {
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}
	}

	// Send the response
	resultPayload := protocol.CommandResultPayload{
		Success: true,
	}

	// Send the message
	return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)
}

// handleManageAlias handles a manage_alias message
func (ws *WebSocketServer) handleManageAlias(conn *websocket.Conn, msg *protocol.Message) error {
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}

		// Send the message
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}

		// Send the message
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}
}

// handleManageGroup handles a manage_group message
func (ws *WebSocketServer) handleManageGroup(conn *websocket.Conn, msg *protocol.Message) error {
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Parse the devices
		devices := make([]echonet_lite.IPAndEOJ, 0, len(payload.Devices))
		for _, deviceStr := range payload.Devices {
			ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(deviceStr)
			if err != nil {
				if logger != nil {
					logger.Log("Error: invalid device: %v", err)
				}
				// エラー応答を送信
				errorPayload := protocol.CommandResultPayload{
					Success: false,
					Error: &protocol.Error{
						Code:    protocol.ErrorCodeInvalidParameters,
						Message: fmt.Sprintf("Invalid device: %v", err),
					},
				}
				return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
			}
			devices = append(devices, ipAndEOJ)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Broadcast group changed notification
		groupChangedPayload := protocol.GroupChangedPayload{
			ChangeType: protocol.GroupChangeTypeAdded,
			Group:      payload.Group,
			Devices:    payload.Devices,
		}
		ws.broadcastMessage(protocol.MessageTypeGroupChanged, groupChangedPayload)

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Parse the devices
		devices := make([]echonet_lite.IPAndEOJ, 0, len(payload.Devices))
		for _, deviceStr := range payload.Devices {
			ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(deviceStr)
			if err != nil {
				if logger != nil {
					logger.Log("Error: invalid device: %v", err)
				}
				// エラー応答を送信
				errorPayload := protocol.CommandResultPayload{
					Success: false,
					Error: &protocol.Error{
						Code:    protocol.ErrorCodeInvalidParameters,
						Message: fmt.Sprintf("Invalid device: %v", err),
					},
				}
				return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
			}
			devices = append(devices, ipAndEOJ)
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Get the updated devices in the group
		updatedDevices, exists := ws.echonetClient.GetDevicesByGroup(payload.Group)
		if !exists {
			// Group was deleted (all devices removed)
			groupChangedPayload := protocol.GroupChangedPayload{
				ChangeType: protocol.GroupChangeTypeDeleted,
				Group:      payload.Group,
			}
			ws.broadcastMessage(protocol.MessageTypeGroupChanged, groupChangedPayload)
		} else {
			// Group was updated
			deviceStrs := make([]string, 0, len(updatedDevices))
			for _, device := range updatedDevices {
				deviceStrs = append(deviceStrs, device.Specifier())
			}
			groupChangedPayload := protocol.GroupChangedPayload{
				ChangeType: protocol.GroupChangeTypeUpdated,
				Group:      payload.Group,
				Devices:    deviceStrs,
			}
			ws.broadcastMessage(protocol.MessageTypeGroupChanged, groupChangedPayload)
		}

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Broadcast group deleted notification
		groupChangedPayload := protocol.GroupChangedPayload{
			ChangeType: protocol.GroupChangeTypeDeleted,
			Group:      payload.Group,
		}
		ws.broadcastMessage(protocol.MessageTypeGroupChanged, groupChangedPayload)

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

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
			for _, device := range group.Devices {
				deviceStrs = append(deviceStrs, device.Specifier())
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
			return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
		}

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
			Data:    groupDataJSON,
		}
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

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
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}
}

// handleDiscoverDevices handles a discover_devices message
func (ws *WebSocketServer) handleDiscoverDevices(conn *websocket.Conn, msg *protocol.Message) error {
	logger := log.GetLogger()
	// Discover devices
	if err := ws.echonetClient.Discover(); err != nil {
		if logger != nil {
			logger.Log("Error discovering devices: %v", err)
		}
		// エラー応答を送信
		errorPayload := protocol.CommandResultPayload{
			Success: false,
			Error: &protocol.Error{
				Code:    protocol.ErrorCodeEchonetCommunicationError,
				Message: fmt.Sprintf("Error discovering devices: %v", err),
			},
		}
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, errorPayload, msg.RequestID)
	}

	// Send the response
	resultPayload := protocol.CommandResultPayload{
		Success: true,
	}

	// Send the message
	return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)
}

// sendMessage sends a message to a client
func (ws *WebSocketServer) sendMessage(conn *websocket.Conn, msgType protocol.MessageType, payload interface{}, requestID string) error {
	// Create the message
	data, err := protocol.CreateMessage(msgType, payload, requestID)
	if err != nil {
		return fmt.Errorf("error creating message: %v", err)
	}

	// Send the message
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	return nil
}

// broadcastMessage sends a message to all connected clients
func (ws *WebSocketServer) broadcastMessage(msgType protocol.MessageType, payload interface{}) {
	logger := log.GetLogger()

	// Create the message
	data, err := protocol.CreateMessage(msgType, payload, "")
	if err != nil {
		if logger != nil {
			logger.Log("Error creating broadcast message: %v", err)
		}
		return
	}

	// Send the message to all clients
	ws.clientsMutex.RLock()
	for conn := range ws.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			if logger != nil {
				logger.Log("Error broadcasting message to client: %v", err)
			}
		}
	}
	ws.clientsMutex.RUnlock()
}

// listenForNotifications listens for notifications from the ECHONET Lite handler
func (ws *WebSocketServer) listenForNotifications() {
	logger := log.GetLogger()

	for {
		select {
		case <-ws.ctx.Done():
			if logger != nil && ws.handler.IsDebug() {
				logger.Log("Notification listener stopped")
			}
			return
		case notification := <-ws.handler.NotificationCh:
			// Handle the notification
			switch notification.Type {
			case echonet_lite.DeviceAdded:
				if logger != nil && ws.handler.IsDebug() {
					logger.Log("Device added: %s", notification.Device.Specifier())
				}

				// Create device added payload
				device := notification.Device

				// Use DeviceToProtocol to convert to protocol format
				protoDevice := protocol.DeviceToProtocol(
					device.IP.String(),
					device.EOJ,
					echonet_lite.Properties{}, // Empty properties, will be updated later
					time.Now(),                // Use current time as last seen
				)

				payload := protocol.DeviceAddedPayload{
					Device: protoDevice,
				}

				// Broadcast the message
				ws.broadcastMessage(protocol.MessageTypeDeviceAdded, payload)

			case echonet_lite.DeviceTimeout:
				if logger != nil && ws.handler.IsDebug() {
					logger.Log("Device timeout: %s - %v", notification.Device.Specifier(), notification.Error)
				}

				// Create timeout notification payload
				device := notification.Device
				payload := protocol.TimeoutNotificationPayload{
					IP:      device.IP.String(),
					EOJ:     device.EOJ.Specifier(),
					Code:    protocol.ErrorCodeEchonetTimeout,
					Message: notification.Error.Error(),
				}

				// Broadcast the message
				ws.broadcastMessage(protocol.MessageTypeTimeoutNotification, payload)
			}
		}
	}
}
