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
	fmt.Printf("WebSocket server starting on %s\n", ws.server.Addr)

	// TLS証明書が指定されている場合
	if options.CertFile != "" && options.KeyFile != "" {
		fmt.Printf("Using TLS with certificate: %s\n", options.CertFile)
		return ws.server.ListenAndServeTLS(options.CertFile, options.KeyFile)
	}

	// 通常のHTTP (証明書なし)
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
	case protocol.MessageTypeManageGroup:
		return ws.handleManageGroup(conn, msg)
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
		fmt.Printf("target: %v, ipAndEOJ: %v\n", target, ipAndEOJ) // DEBUG

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
			return fmt.Errorf("error marshaling device: %v", err)
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

// handleManageGroup handles a manage_group message
func (ws *WebSocketServer) handleManageGroup(conn *websocket.Conn, msg *protocol.Message) error {
	// Parse the payload
	var payload protocol.ManageGroupPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		return fmt.Errorf("error parsing manage_group payload: %v", err)
	}

	// Validate the payload
	if payload.Group == "" {
		return fmt.Errorf("no group specified")
	}

	// Handle the action
	switch payload.Action {
	case protocol.GroupActionAdd:
		// Validate the devices
		if len(payload.Devices) == 0 {
			return fmt.Errorf("no devices specified for add action")
		}

		// Parse the devices
		devices := make([]echonet_lite.IPAndEOJ, 0, len(payload.Devices))
		for _, deviceStr := range payload.Devices {
			ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(deviceStr)
			if err != nil {
				return fmt.Errorf("invalid device: %v", err)
			}
			devices = append(devices, ipAndEOJ)
		}

		// Add the devices to the group
		if err := ws.echonetClient.GroupAdd(payload.Group, devices); err != nil {
			return fmt.Errorf("error adding devices to group: %v", err)
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
			return fmt.Errorf("no devices specified for remove action")
		}

		// Parse the devices
		devices := make([]echonet_lite.IPAndEOJ, 0, len(payload.Devices))
		for _, deviceStr := range payload.Devices {
			ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(deviceStr)
			if err != nil {
				return fmt.Errorf("invalid device: %v", err)
			}
			devices = append(devices, ipAndEOJ)
		}

		// Remove the devices from the group
		if err := ws.echonetClient.GroupRemove(payload.Group, devices); err != nil {
			return fmt.Errorf("error removing devices from group: %v", err)
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
			return fmt.Errorf("error deleting group: %v", err)
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
			return fmt.Errorf("error marshaling group data: %v", err)
		}

		// Send the response
		resultPayload := protocol.CommandResultPayload{
			Success: true,
			Data:    groupDataJSON,
		}
		return ws.sendMessage(conn, protocol.MessageTypeCommandResult, resultPayload, msg.RequestID)

	default:
		return fmt.Errorf("unknown group action: %s", payload.Action)
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
					echonet_lite.Properties{}, // Empty properties, will be updated later
					time.Now(),                // Use current time as last seen
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
