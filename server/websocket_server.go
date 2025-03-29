package server

import (
	"context"
	"echonet-list/client"
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

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
func (ws *WebSocketServer) Start() error {
	fmt.Printf("WebSocket server starting on %s\n", ws.server.Addr)
	return ws.server.ListenAndServe()
}

// Stop stops the WebSocket server
func (ws *WebSocketServer) Stop() error {
	ws.cancel()
	return ws.server.Shutdown(context.Background())
}

// handleWebSocket handles WebSocket connections
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Error upgrading to WebSocket: %v\n", err)
		return
	}
	defer conn.Close()

	// Register the client
	ws.clientsMutex.Lock()
	ws.clients[conn] = true
	ws.clientsMutex.Unlock()

	// Remove the client when the function returns
	defer func() {
		ws.clientsMutex.Lock()
		delete(ws.clients, conn)
		ws.clientsMutex.Unlock()
	}()

	// Send initial state to the client
	if err := ws.sendInitialState(conn); err != nil {
		fmt.Printf("Error sending initial state: %v\n", err)
		return
	}

	// Handle incoming messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket error: %v\n", err)
			}
			break
		}

		// Parse and handle the message
		if err := ws.handleMessage(conn, message); err != nil {
			fmt.Printf("Error handling message: %v\n", err)
			// Send error response
			errorPayload := protocol.ErrorNotificationPayload{
				Code:    protocol.ErrorCodeInternalServerError,
				Message: err.Error(),
			}
			if err := ws.sendMessage(conn, protocol.MessageTypeErrorNotification, errorPayload, ""); err != nil {
				fmt.Printf("Error sending error notification: %v\n", err)
			}
		}
	}
}

// sendInitialState sends the initial state to a client
func (ws *WebSocketServer) sendInitialState(conn *websocket.Conn) error {
	// Get all devices
	devices := ws.echonetClient.ListDevices(echonet_lite.FilterCriteria{})

	// Convert devices to protocol format
	protoDevices := make(map[string]protocol.Device)
	for _, device := range devices {
		// Convert properties to map
		properties := make(map[echonet_lite.EPCType][]byte)
		for _, prop := range device.Properties {
			properties[prop.EPC] = prop.EDT
		}

		// Use DeviceToProtocol to convert to protocol format
		protoDevice := protocol.DeviceToProtocol(
			device.Device.IP.String(),
			device.Device.EOJ,
			properties,
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

	// Create initial state payload
	payload := protocol.InitialStatePayload{
		Devices: protoDevices,
		Aliases: aliases,
	}

	// Send the message
	return ws.sendMessage(conn, protocol.MessageTypeInitialState, payload, "")
}

// handleMessage handles an incoming message from a client
func (ws *WebSocketServer) handleMessage(conn *websocket.Conn, message []byte) error {
	// Parse the message
	msg, err := protocol.ParseMessage(message)
	if err != nil {
		return fmt.Errorf("error parsing message: %v", err)
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
	case protocol.MessageTypeDiscoverDevices:
		return ws.handleDiscoverDevices(conn, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleGetProperties handles a get_properties message
func (ws *WebSocketServer) handleGetProperties(conn *websocket.Conn, msg *protocol.Message) error {
	// Parse the payload
	var payload protocol.GetPropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return fmt.Errorf("error parsing get_properties payload: %v", err)
	}

	// Validate the payload
	if len(payload.Targets) == 0 {
		return fmt.Errorf("no targets specified")
	}

	// Process each target
	results := make([]protocol.Device, 0, len(payload.Targets))
	for _, target := range payload.Targets {
		// Parse the target
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(target)
		if err != nil {
			return fmt.Errorf("invalid target: %v", err)
		}

		// Parse EPCs
		epcs := make([]echonet_lite.EPCType, 0, len(payload.EPCs))
		for _, epcStr := range payload.EPCs {
			epc, err := echonet_lite.ParseEPCString(epcStr)
			if err != nil {
				return fmt.Errorf("invalid EPC: %v", err)
			}
			epcs = append(epcs, epc)
		}

		// Get properties
		deviceAndProps, err := ws.echonetClient.GetProperties(ipAndEOJ, epcs, false)
		if err != nil {
			return fmt.Errorf("error getting properties: %v", err)
		}

		// Convert properties to map
		properties := make(map[echonet_lite.EPCType][]byte)
		for _, prop := range deviceAndProps.Properties {
			properties[prop.EPC] = prop.EDT
		}

		// Use DeviceToProtocol to convert to protocol format
		protoDevice := protocol.DeviceToProtocol(
			deviceAndProps.Device.IP.String(),
			deviceAndProps.Device.EOJ,
			properties,
			time.Now(), // Use current time as last seen
		)

		results = append(results, protoDevice)
	}

	// Send the response
	resultPayload := protocol.CommandResultPayload{
		Success: true,
	}

	// Marshal the results
	resultData, err := protocol.CreateMessage(protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)
	if err != nil {
		return fmt.Errorf("error creating command result message: %v", err)
	}

	// Send the message
	if err := conn.WriteMessage(websocket.TextMessage, resultData); err != nil {
		return fmt.Errorf("error sending command result: %v", err)
	}

	return nil
}

// handleSetProperties handles a set_properties message
func (ws *WebSocketServer) handleSetProperties(conn *websocket.Conn, msg *protocol.Message) error {
	// Parse the payload
	var payload protocol.SetPropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return fmt.Errorf("error parsing set_properties payload: %v", err)
	}

	// Validate the payload
	if payload.Target == "" {
		return fmt.Errorf("no target specified")
	}
	if len(payload.Properties) == 0 {
		return fmt.Errorf("no properties specified")
	}

	// Parse the target
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.Target)
	if err != nil {
		return fmt.Errorf("invalid target: %v", err)
	}

	// Parse properties
	properties := make(echonet_lite.Properties, 0, len(payload.Properties))
	for epcStr, edtStr := range payload.Properties {
		epc, err := echonet_lite.ParseEPCString(epcStr)
		if err != nil {
			return fmt.Errorf("invalid EPC: %v", err)
		}

		edt, err := base64.StdEncoding.DecodeString(edtStr)
		if err != nil {
			return fmt.Errorf("invalid EDT: %v", err)
		}

		properties = append(properties, echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		})
	}

	// Set properties
	deviceAndProps, err := ws.echonetClient.SetProperties(ipAndEOJ, properties)
	if err != nil {
		return fmt.Errorf("error setting properties: %v", err)
	}

	// Convert properties to map
	propsMap := make(map[echonet_lite.EPCType][]byte)
	for _, prop := range deviceAndProps.Properties {
		propsMap[prop.EPC] = prop.EDT
	}

	// Use DeviceToProtocol to convert to protocol format
	deviceData := protocol.DeviceToProtocol(
		deviceAndProps.Device.IP.String(),
		deviceAndProps.Device.EOJ,
		propsMap,
		time.Now(), // Use current time as last seen
	)

	// Marshal the device data
	deviceDataJSON, err := json.Marshal(deviceData)
	if err != nil {
		return fmt.Errorf("error marshaling device data: %v", err)
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
	// Parse the payload
	var payload protocol.UpdatePropertiesPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return fmt.Errorf("error parsing update_properties payload: %v", err)
	}

	// Validate the payload
	if len(payload.Targets) == 0 {
		return fmt.Errorf("no targets specified")
	}

	// Process each target
	for _, target := range payload.Targets {
		// Parse the target
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(target)
		if err != nil {
			return fmt.Errorf("invalid target: %v", err)
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
			return fmt.Errorf("error updating properties: %v", err)
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
	// Parse the payload
	var payload protocol.ManageAliasPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return fmt.Errorf("error parsing manage_alias payload: %v", err)
	}

	// Validate the payload
	if payload.Alias == "" {
		return fmt.Errorf("no alias specified")
	}

	// Handle the action
	switch payload.Action {
	case protocol.AliasActionAdd:
		// Validate the target
		if payload.Target == "" {
			return fmt.Errorf("no target specified for add action")
		}

		// Parse the target
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.Target)
		if err != nil {
			return fmt.Errorf("invalid target: %v", err)
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
			return fmt.Errorf("error setting alias: %v", err)
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
			return fmt.Errorf("error deleting alias: %v", err)
		}

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
		}

		// Send the message
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

	default:
		return fmt.Errorf("unknown alias action: %s", payload.Action)
	}
}

// handleDiscoverDevices handles a discover_devices message
func (ws *WebSocketServer) handleDiscoverDevices(conn *websocket.Conn, msg *protocol.Message) error {
	// Discover devices
	if err := ws.echonetClient.Discover(); err != nil {
		return fmt.Errorf("error discovering devices: %v", err)
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
	// Create the message
	data, err := protocol.CreateMessage(msgType, payload, "")
	if err != nil {
		fmt.Printf("Error creating broadcast message: %v\n", err)
		return
	}

	// Send the message to all clients
	ws.clientsMutex.RLock()
	for conn := range ws.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			fmt.Printf("Error broadcasting message: %v\n", err)
		}
	}
	ws.clientsMutex.RUnlock()
}

// listenForNotifications listens for notifications from the ECHONET Lite handler
func (ws *WebSocketServer) listenForNotifications() {
	for {
		select {
		case <-ws.ctx.Done():
			return
		case notification := <-ws.handler.NotificationCh:
			// Handle the notification
			switch notification.Type {
			case echonet_lite.DeviceAdded:
				// Create device added payload
				device := notification.Device

				// Use DeviceToProtocol to convert to protocol format
				protoDevice := protocol.DeviceToProtocol(
					device.IP.String(),
					device.EOJ,
					make(map[echonet_lite.EPCType][]byte), // Empty properties, will be updated later
					time.Now(),                            // Use current time as last seen
				)

				payload := protocol.DeviceAddedPayload{
					Device: protoDevice,
				}

				// Broadcast the message
				ws.broadcastMessage(protocol.MessageTypeDeviceAdded, payload)

			case echonet_lite.DeviceTimeout:
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
