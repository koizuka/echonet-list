package server

import (
	"context"
	"echonet-list/client"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/log"
	"echonet-list/protocol"
	"encoding/base64"
	"fmt"
	"time"
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
	transport     WebSocketTransport
	echonetClient client.ECHONETListClient
	handler       *echonet_lite.ECHONETLiteHandler
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(ctx context.Context, addr string, echonetClient client.ECHONETListClient, handler *echonet_lite.ECHONETLiteHandler) (*WebSocketServer, error) {
	serverCtx, cancel := context.WithCancel(ctx)

	// Create the transport
	transport := NewDefaultWebSocketTransport(serverCtx, addr)

	// Create the WebSocket server
	ws := &WebSocketServer{
		ctx:           serverCtx,
		cancel:        cancel,
		transport:     transport,
		echonetClient: echonetClient,
		handler:       handler,
	}

	// Set up the transport handlers
	transport.SetConnectHandler(ws.handleClientConnect)
	transport.SetMessageHandler(ws.handleClientMessage)
	transport.SetDisconnectHandler(ws.handleClientDisconnect)

	// Start listening for notifications from the ECHONET Lite handler
	go ws.listenForNotifications()

	return ws, nil
}

// handleClientConnect is called when a new client connects
func (ws *WebSocketServer) handleClientConnect(connID string) error {
	logger := log.GetLogger()
	if logger != nil && ws.handler.IsDebug() {
		logger.Log("New WebSocket connection established: %s", connID)
	}

	// Send initial state to the client
	return ws.sendInitialStateToClient(connID)
}

// handleClientMessage is called when a message is received from a client
func (ws *WebSocketServer) handleClientMessage(connID string, message []byte) error {
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
		return ws.sendMessageToClient(connID, protocol.MessageTypeErrorNotification, errorPayload, "")
	}

	// Handle the message based on its type
	switch msg.Type {
	case protocol.MessageTypeGetProperties:
		return ws.handleGetPropertiesFromClient(connID, msg)
	case protocol.MessageTypeSetProperties:
		return ws.handleSetPropertiesFromClient(connID, msg)
	case protocol.MessageTypeUpdateProperties:
		return ws.handleUpdatePropertiesFromClient(connID, msg)
	case protocol.MessageTypeManageAlias:
		return ws.handleManageAliasFromClient(connID, msg)
	case protocol.MessageTypeManageGroup:
		return ws.handleManageGroupFromClient(connID, msg)
	case protocol.MessageTypeDiscoverDevices:
		return ws.handleDiscoverDevicesFromClient(connID, msg)
	case protocol.MessageTypeGetPropertyAliases:
		return ws.handleGetPropertyAliasesFromClient(connID, msg)
	default:
		if logger != nil {
			logger.Log("Unknown message type: %s", msg.Type)
		}
		// エラー応答を送信
		errorPayload := protocol.ErrorNotificationPayload{
			Code:    protocol.ErrorCodeInvalidRequestFormat,
			Message: fmt.Sprintf("Unknown message type: %s", msg.Type),
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeErrorNotification, errorPayload, msg.RequestID)
	}
}

// handleClientDisconnect is called when a client disconnects
func (ws *WebSocketServer) handleClientDisconnect(connID string) {
	logger := log.GetLogger()
	if logger != nil && ws.handler.IsDebug() {
		logger.Log("WebSocket connection closed: %s", connID)
	}
}

// Start starts the WebSocket server
func (ws *WebSocketServer) Start(options StartOptions) error {
	return ws.transport.Start(options)
}

// Stop stops the WebSocket server
func (ws *WebSocketServer) Stop() error {
	ws.cancel()
	return ws.transport.Stop()
}

// sendInitialStateToClient sends the initial state to a client
func (ws *WebSocketServer) sendInitialStateToClient(connID string) error {
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
		if alias.Device != nil {
			aliases[alias.Alias] = alias.Device.Specifier()
		} else {
			aliases[alias.Alias] = "" // 登録されたデバイスが見つからない
		}
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
	return ws.sendMessageToClient(connID, protocol.MessageTypeInitialState, payload, "")
}

// sendMessageToClient sends a message to a client
func (ws *WebSocketServer) sendMessageToClient(connID string, msgType protocol.MessageType, payload interface{}, requestID string) error {
	// Create the message
	data, err := protocol.CreateMessage(msgType, payload, requestID)
	if err != nil {
		return fmt.Errorf("error creating message: %v", err)
	}

	// Send the message
	return ws.transport.SendMessage(connID, data)
}

// broadcastMessageToClients sends a message to all connected clients
func (ws *WebSocketServer) broadcastMessageToClients(msgType protocol.MessageType, payload interface{}) error {
	logger := log.GetLogger()

	// Create the message
	data, err := protocol.CreateMessage(msgType, payload, "")
	if err != nil {
		if logger != nil {
			logger.Log("Error creating broadcast message: %v", err)
		}
		return err
	}

	// Send the message to all clients
	return ws.transport.BroadcastMessage(data)
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
				ws.broadcastMessageToClients(protocol.MessageTypeDeviceAdded, payload)

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
				ws.broadcastMessageToClients(protocol.MessageTypeTimeoutNotification, payload)
			}
		case propertyChange := <-ws.handler.PropertyChangeCh:
			// プロパティ変化通知を処理
			if logger != nil && ws.handler.IsDebug() {
				logger.Log("Property changed: %s - EPC: %02X", propertyChange.Device.Specifier(), byte(propertyChange.Property.EPC))
			}

			// プロパティ変化通知ペイロードを作成
			payload := protocol.PropertyChangedPayload{
				IP:    propertyChange.Device.IP.String(),
				EOJ:   propertyChange.Device.EOJ.Specifier(),
				EPC:   fmt.Sprintf("%02X", byte(propertyChange.Property.EPC)),
				Value: base64.StdEncoding.EncodeToString(propertyChange.Property.EDT),
			}

			// メッセージをブロードキャスト
			ws.broadcastMessageToClients(protocol.MessageTypePropertyChanged, payload)
		}
	}
}
