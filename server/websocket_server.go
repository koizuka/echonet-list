package server

import (
	"context"
	"echonet-list/client"
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/json"
	"fmt"
	"log/slog"
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
	// HTTPサーバーの設定
	HTTPEnabled bool
	HTTPWebRoot string
}

// WebSocketServer implements a WebSocket server for ECHONET Lite
type WebSocketServer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	transport     WebSocketTransport
	echonetClient client.ECHONETListClient
	handler       *handler.ECHONETLiteHandler
	activeClients atomic.Int32 // Number of currently connected clients
	updateTicker  *time.Ticker // Ticker for periodic updates
	tickerDone    chan bool    // Channel to stop the ticker goroutine
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(ctx context.Context, addr string, echonetClient client.ECHONETListClient, handler *handler.ECHONETLiteHandler) (*WebSocketServer, error) {
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

// GetTransport returns the WebSocket transport
func (ws *WebSocketServer) GetTransport() WebSocketTransport {
	return ws.transport
}

// periodicUpdater runs in a goroutine, triggering property updates at the configured interval
// if at least one client is connected.
func (ws *WebSocketServer) periodicUpdater() {
	if ws.handler.IsDebug() {
		slog.Debug("Periodic updater started")
	}
	defer func() {
		if ws.handler.IsDebug() {
			slog.Debug("Periodic updater stopped")
		}
	}()

	for {
		select {
		case <-ws.updateTicker.C:
			// Check if any clients are connected
			if ws.activeClients.Load() > 0 {
				if ws.handler.IsDebug() {
					slog.Debug("Ticker triggered: Updating all device properties (clients connected)")
				}
				// Run update in a separate goroutine to avoid blocking the ticker
				go func() {
					// Use an empty FilterCriteria to target all devices
					err := ws.handler.UpdateProperties(handler.FilterCriteria{}, false)
					if err != nil {
						// Log the error but don't stop the ticker
						slog.Error("Error during periodic property update", "err", err)
					}
				}()
			} else {
				if ws.handler.IsDebug() {
					slog.Debug("Ticker triggered: Skipping update (no clients connected)")
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
	if ws.handler.IsDebug() {
		slog.Debug("New WebSocket connection established", "connID", connID)
	}

	// Increment active client count
	ws.activeClients.Add(1)
	if ws.handler.IsDebug() {
		slog.Debug("Active clients", "count", ws.activeClients.Load())
	}

	// Send initial state to the client
	return ws.sendInitialStateToClient(connID)
}

// handleClientMessage is called when a message is received from a client
func (ws *WebSocketServer) handleClientMessage(connID string, message []byte) error {
	// Parse the message
	msg, err := protocol.ParseMessage(message)
	if err != nil {
		slog.Error("Error parsing message", "err", err)
		// エラー応答を送信
		errorPayload := protocol.ErrorNotificationPayload{
			Code:    protocol.ErrorCodeInvalidRequestFormat,
			Message: fmt.Sprintf("Error parsing message: %v", err),
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeErrorNotification, errorPayload, "")
	}

	handle := func(handler func(msg *protocol.Message) protocol.CommandResultPayload) error {
		result := handler(msg)
		if !result.Success {
			slog.Error("Error for RequestID", "requestID", msg.RequestID, "message", result.Error.Message)
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeCommandResult, result, msg.RequestID)
	}

	// Handle the message based on its type
	switch msg.Type {
	case protocol.MessageTypeGetProperties:
		return handle(ws.handleGetPropertiesFromClient)
	case protocol.MessageTypeSetProperties:
		return handle(ws.handleSetPropertiesFromClient)
	case protocol.MessageTypeUpdateProperties:
		return handle(ws.handleUpdatePropertiesFromClient)
	case protocol.MessageTypeManageAlias:
		return handle(ws.handleManageAliasFromClient)
	case protocol.MessageTypeManageGroup:
		return handle(ws.handleManageGroupFromClient)
	case protocol.MessageTypeDiscoverDevices:
		return handle(ws.handleDiscoverDevicesFromClient)
	case protocol.MessageTypeGetPropertyDescription:
		return handle(ws.handleGetPropertyDescriptionFromClient)
	default:
		slog.Error("Unknown message type", "type", msg.Type)
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
	if ws.handler.IsDebug() {
		slog.Debug("WebSocket connection closed", "connID", connID)
	}
	// Decrement active client count
	ws.activeClients.Add(-1)
	if ws.handler.IsDebug() {
		slog.Debug("Active clients", "count", ws.activeClients.Load())
	}
}

// Start starts the WebSocket server and optionally the periodic updater
func (ws *WebSocketServer) Start(options StartOptions) error {
	// HTTPサーバーが有効な場合は静的ファイル配信を設定
	if options.HTTPEnabled {
		if transport, ok := ws.transport.(*DefaultWebSocketTransport); ok {
			if err := transport.SetupStaticFileServer(options.HTTPWebRoot); err != nil {
				return fmt.Errorf("failed to setup static file server: %v", err)
			}
		}
	}

	// Start the periodic updater ticker if interval is positive
	if options.PeriodicUpdateInterval > 0 {
		ws.updateTicker = time.NewTicker(options.PeriodicUpdateInterval)
		go ws.periodicUpdater()
		if ws.handler.IsDebug() {
			slog.Debug("Periodic property updater enabled with interval", "interval", options.PeriodicUpdateInterval)
		}
	} else {
		if ws.handler.IsDebug() {
			slog.Debug("Periodic property updater disabled.")
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
	if ws.handler.IsDebug() {
		slog.Debug("Sending initial state to client")
	}

	// Get all devices
	devices := ws.echonetClient.ListDevices(handler.FilterCriteria{})

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

// SuccessResponse はコマンドの成功応答を作成する
func SuccessResponse(resultJSON json.RawMessage) protocol.CommandResultPayload {
	return protocol.CommandResultPayload{
		Success: true,
		Data:    resultJSON,
	}
}

// ErrorResponse はコマンドのエラー応答を作成する
func ErrorResponse(code protocol.ErrorCode, format string, args ...any) protocol.CommandResultPayload {
	errorPayload := protocol.Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
	return protocol.CommandResultPayload{
		Success: false,
		Error:   &errorPayload,
	}
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
	// Create the message
	data, err := protocol.CreateMessage(msgType, payload, "")
	if err != nil {
		slog.Error("Error creating broadcast message", "err", err)
		return err
	}

	// Send the message to all clients
	return ws.transport.BroadcastMessage(data)
}

// listenForNotifications listens for notifications from the ECHONET Lite handler
func (ws *WebSocketServer) listenForNotifications() {
	for {
		select {
		case <-ws.ctx.Done():
			slog.Debug("Notification listener stopped")
			return
		case notification := <-ws.handler.NotificationCh:
			// Handle the notification
			switch notification.Type {
			case handler.DeviceAdded:
				if ws.handler.IsDebug() {
					slog.Debug("Device added", "device", notification.Device.Specifier())
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
				_ = ws.broadcastMessageToClients(protocol.MessageTypeDeviceAdded, payload)

			case handler.DeviceTimeout:
				slog.Error("Device timeout", "device", notification.Device.Specifier(), "error", notification.Error)

				// Create timeout notification payload
				device := notification.Device
				payload := protocol.TimeoutNotificationPayload{
					IP:      device.IP.String(),
					EOJ:     device.EOJ.Specifier(),
					Code:    protocol.ErrorCodeEchonetTimeout,
					Message: notification.Error.Error(),
				}

				// Broadcast the message
				_ = ws.broadcastMessageToClients(protocol.MessageTypeTimeoutNotification, payload)

			case handler.DeviceOffline:
				if ws.handler.IsDebug() {
					slog.Debug("Device offline", "device", notification.Device.Specifier())
				}

				// Create device offline payload
				device := notification.Device
				payload := protocol.DeviceOfflinePayload{
					IP:  device.IP.String(),
					EOJ: device.EOJ.Specifier(),
				}

				// Broadcast the message
				_ = ws.broadcastMessageToClients(protocol.MessageTypeDeviceOffline, payload)
			}
		case propertyChange := <-ws.handler.PropertyChangeCh:
			// プロパティ変化通知を処理
			if ws.handler.IsDebug() {
				slog.Debug("Property changed", "device", propertyChange.Device.Specifier(), "epc", fmt.Sprintf("%02X", byte(propertyChange.Property.EPC)))
			}

			// プロパティ変化通知ペイロードを作成
			payload := protocol.PropertyChangedPayload{
				IP:    propertyChange.Device.IP.String(),
				EOJ:   propertyChange.Device.EOJ.Specifier(),
				EPC:   fmt.Sprintf("%02X", byte(propertyChange.Property.EPC)),
				Value: protocol.MakePropertyData(propertyChange.Device.EOJ.ClassCode(), propertyChange.Property),
			}

			// メッセージをブロードキャスト
			_ = ws.broadcastMessageToClients(protocol.MessageTypePropertyChanged, payload)
		}
	}
}
