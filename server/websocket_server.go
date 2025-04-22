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
	"sync/atomic"
	"time"
)

// StartOptions は WebSocketServer の起動オプションを表す
type StartOptions struct {
	// TLS証明書ファイルのパス (TLSを使用する場合)
	CertFile string
	// TLS秘密鍵ファイルのパス (TLSを使用する場合)
	KeyFile string
	// 定期的なプロパティ更新の間隔 (0以下で無効)
	PeriodicUpdateInterval time.Duration
	// サーバーの待ち受け完了を通知するチャネル
	Ready chan struct{}
}

// WebSocketServer implements a WebSocket server for ECHONET Lite
type WebSocketServer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	transport     WebSocketTransport
	echonetClient client.ECHONETListClient
	handler       *echonet_lite.ECHONETLiteHandler
	activeClients atomic.Int32 // Number of currently connected clients
	updateTicker  *time.Ticker // Ticker for periodic updates
	tickerDone    chan bool    // Channel to stop the ticker goroutine
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
		tickerDone:    make(chan bool), // Initialize the done channel
	}

	// Set up the transport handlers
	transport.SetConnectHandler(ws.handleClientConnect)
	transport.SetMessageHandler(ws.handleClientMessage)
	transport.SetDisconnectHandler(ws.handleClientDisconnect)

	// Start listening for notifications from the ECHONET Lite handler
	go ws.listenForNotifications()

	return ws, nil
}

// periodicUpdater runs in a goroutine, triggering property updates every minute
// if at least one client is connected.
func (ws *WebSocketServer) periodicUpdater() {
	logger := log.GetLogger()
	if logger != nil && ws.handler.IsDebug() {
		logger.Log("Periodic updater started")
	}
	defer func() {
		if logger != nil && ws.handler.IsDebug() {
			logger.Log("Periodic updater stopped")
		}
	}()

	for {
		select {
		case <-ws.updateTicker.C:
			// Check if any clients are connected
			if ws.activeClients.Load() > 0 {
				if logger != nil && ws.handler.IsDebug() {
					logger.Log("Ticker triggered: Updating all device properties (clients connected)")
				}
				// Run update in a separate goroutine to avoid blocking the ticker
				go func() {
					// Use an empty FilterCriteria to target all devices
					err := ws.handler.UpdateProperties(echonet_lite.FilterCriteria{}, false)
					if err != nil && logger != nil {
						// Log the error but don't stop the ticker
						logger.Log("Error during periodic property update: %v", err)
					}
				}()
			} else {
				if logger != nil && ws.handler.IsDebug() {
					logger.Log("Ticker triggered: Skipping update (no clients connected)")
				}
			}
		case <-ws.tickerDone:
			ws.updateTicker.Stop()
			return
		case <-ws.ctx.Done(): // Ensure goroutine exits if server context is cancelled
			ws.updateTicker.Stop()
			return
		}
	}
}

// handleClientConnect is called when a new client connects
func (ws *WebSocketServer) handleClientConnect(connID string) error {
	logger := log.GetLogger()
	if logger != nil && ws.handler.IsDebug() {
		logger.Log("New WebSocket connection established: %s", connID)
	}

	// Increment active client count
	ws.activeClients.Add(1)
	if logger != nil && ws.handler.IsDebug() {
		logger.Log("Active clients: %d", ws.activeClients.Load())
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
	// Decrement active client count
	ws.activeClients.Add(-1)
	if logger != nil && ws.handler.IsDebug() {
		logger.Log("Active clients: %d", ws.activeClients.Load())
	}
}

// Start starts the WebSocket server and optionally the periodic updater
func (ws *WebSocketServer) Start(options StartOptions) error {
	// Start the periodic updater ticker if interval is positive
	if options.PeriodicUpdateInterval > 0 {
		ws.updateTicker = time.NewTicker(options.PeriodicUpdateInterval)
		go ws.periodicUpdater()
		logger := log.GetLogger()
		if logger != nil && ws.handler.IsDebug() {
			logger.Log("Periodic property updater enabled with interval: %v", options.PeriodicUpdateInterval)
		}
	} else {
		logger := log.GetLogger()
		if logger != nil && ws.handler.IsDebug() {
			logger.Log("Periodic property updater disabled.")
		}
	}

	return ws.transport.Start(options)
}

// Stop stops the WebSocket server and the periodic updater
func (ws *WebSocketServer) Stop() error {
	// Signal the periodic updater to stop if it was started
	if ws.updateTicker != nil {
		close(ws.tickerDone)
	}

	ws.cancel() // Cancel the server context
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
		// デバイスの最終更新タイムスタンプを取得
		lastSeen := ws.handler.GetLastUpdateTime(device.Device)

		// Use DeviceToProtocol to convert to protocol format
		protoDevice := protocol.DeviceToProtocol(
			device.Device,
			device.Properties,
			lastSeen,
		)

		// Add to map with device identifier as key
		protoDevices[device.Device.Specifier()] = protoDevice
	}

	// Get all aliases
	aliasList := ws.echonetClient.AliasList()
	aliases := make(map[string]client.IDString)
	for _, alias := range aliasList {
		aliases[alias.Alias] = alias.ID
	}

	// Get all groups
	groupList := ws.echonetClient.GroupList(nil)
	groups := make(map[string][]client.IDString)
	for _, group := range groupList {
		groups[group.Group] = group.Devices
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

// sendSuccessResponse logs success and sends a success response to the client
func (ws *WebSocketServer) sendSuccessResponse(connID string, requestID string, data json.RawMessage) error {
	resultPayload := protocol.CommandResultPayload{
		Success: true,
		Data:    data,
	}

	return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, resultPayload, requestID)
}

// sendErrorResponse logs an error and sends an error response to the client
func (ws *WebSocketServer) sendErrorResponse(connID string, requestID string, errorCode protocol.ErrorCode, format string, args ...interface{}) error {
	errorMessage := fmt.Sprintf(format, args...)
	logger := log.GetLogger()
	if logger != nil {
		logger.Log("Error for RequestID %s: %s", requestID, errorMessage)
	}

	errorPayload := protocol.CommandResultPayload{
		Success: false,
		Error: &protocol.Error{
			Code:    errorCode,
			Message: errorMessage,
		},
	}

	return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, errorPayload, requestID)
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

				// デバイスの最終更新タイムスタンプを取得
				lastSeen := ws.handler.GetLastUpdateTime(device)

				// Use DeviceToProtocol to convert to protocol format
				protoDevice := protocol.DeviceToProtocol(
					device,
					echonet_lite.Properties{}, // Empty properties, will be updated later
					lastSeen,
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

			case echonet_lite.DeviceOffline:
				if logger != nil && ws.handler.IsDebug() {
					logger.Log("Device offline: %s", notification.Device.Specifier())
				}

				// Create device offline payload
				device := notification.Device
				payload := protocol.DeviceOfflinePayload{
					IP:  device.IP.String(),
					EOJ: device.EOJ.Specifier(),
				}

				// Broadcast the message
				ws.broadcastMessageToClients(protocol.MessageTypeDeviceOffline, payload)
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
