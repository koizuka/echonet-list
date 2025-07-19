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
	"strings"
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
	ctx                    context.Context
	cancel                 context.CancelFunc
	transport              WebSocketTransport
	echonetClient          client.ECHONETListClient
	handler                *handler.ECHONETLiteHandler
	activeClients          atomic.Int32 // Number of currently connected clients
	updateTicker           *time.Ticker // Ticker for periodic updates
	tickerDone             chan bool    // Channel to stop the ticker goroutine
	initialStateInProgress atomic.Int32 // Counter for ongoing initial state generations
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
			// Check if any clients are connected and no initial state generation is in progress
			clientCount := ws.activeClients.Load()
			initialStateCount := ws.initialStateInProgress.Load()

			if clientCount > 0 && initialStateCount == 0 {
				if ws.handler.IsDebug() {
					slog.Debug("Ticker triggered: Updating all device properties (clients connected, no initial state in progress)")
				}
				// Run update in a separate goroutine to avoid blocking the ticker
				go func() {
					// Use an empty FilterCriteria to target all devices
					err := ws.handler.UpdateProperties(handler.FilterCriteria{}, false)
					if err != nil {
						// Log the error but don't stop the ticker
						slog.Info("Error during periodic property update", "err", err)
					}
				}()
			} else {
				if ws.handler.IsDebug() {
					if clientCount == 0 {
						slog.Debug("Ticker triggered: Skipping update (no clients connected)")
					} else if initialStateCount > 0 {
						slog.Debug("Ticker triggered: Skipping update (initial state generation in progress)", "count", initialStateCount)
					}
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

	// Send initial state to the client asynchronously
	// Don't wait for completion to avoid blocking the connection handler
	if err := ws.sendInitialStateToClient(connID); err != nil {
		slog.Error("Failed to start initial state sending", "error", err, "connID", connID)
		// Don't return error here as connection should still be established
	}

	return nil
}

// handleClientMessage is called when a message is received from a client
func (ws *WebSocketServer) handleClientMessage(connID string, message []byte) error {
	if ws.handler.IsDebug() {
		slog.Debug("Received WebSocket message", "connID", connID, "message", string(message))
	}

	// Parse the message
	msg, err := protocol.ParseMessage(message)
	if err != nil {
		slog.Error("Error parsing message", "err", err, "connID", connID)
		// エラー応答を送信
		errorPayload := protocol.ErrorNotificationPayload{
			Code:    protocol.ErrorCodeInvalidRequestFormat,
			Message: fmt.Sprintf("Error parsing message: %v", err),
		}
		return ws.sendMessageToClient(connID, protocol.MessageTypeErrorNotification, errorPayload, "")
	}

	if ws.handler.IsDebug() {
		slog.Debug("Parsed message", "connID", connID, "type", msg.Type, "requestID", msg.RequestID)
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
	case protocol.MessageTypeListDevices:
		return handle(ws.handleListDevicesFromClient)
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
		slog.Debug("Sending initial state to client", "connID", connID)
	}

	// Run initial state generation in a separate goroutine to avoid blocking the connection handler
	go func() {
		// Increment initial state generation counter
		ws.initialStateInProgress.Add(1)
		defer ws.initialStateInProgress.Add(-1)

		// Add timeout to prevent indefinite blocking
		ctx, cancel := context.WithTimeout(ws.ctx, 30*time.Second)
		defer cancel()

		// Use channel to signal completion or timeout
		done := make(chan error, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Panic in initial state generation", "error", r, "connID", connID)
					done <- fmt.Errorf("panic during initial state generation: %v", r)
				}
			}()

			if err := ws.generateAndSendInitialState(connID); err != nil {
				done <- err
			} else {
				done <- nil
			}
		}()

		// Wait for completion or timeout
		select {
		case err := <-done:
			if err != nil {
				// Check if the error is due to client disconnect
				if isClientDisconnectedError(err) {
					slog.Debug("Client disconnected during initial state generation", "connID", connID)
					// Don't try to send error notification to disconnected client
					return
				}

				slog.Error("Failed to send initial state", "error", err, "connID", connID)

				// Send error notification to client only if still connected
				errorPayload := protocol.ErrorNotificationPayload{
					Code:    protocol.ErrorCodeInternalServerError,
					Message: "Failed to load initial state",
				}
				if sendErr := ws.sendMessageToClient(connID, protocol.MessageTypeErrorNotification, errorPayload, ""); sendErr != nil {
					if !isClientDisconnectedError(sendErr) {
						slog.Error("Failed to send error notification", "error", sendErr, "connID", connID)
					}
				}
			} else {
				if ws.handler.IsDebug() {
					slog.Debug("Initial state sent successfully", "connID", connID)
				}
			}
		case <-ctx.Done():
			slog.Error("Initial state generation timed out", "connID", connID)
			// Send timeout error to client only if still connected
			errorPayload := protocol.ErrorNotificationPayload{
				Code:    protocol.ErrorCodeInternalServerError,
				Message: "Initial state loading timed out",
			}
			if sendErr := ws.sendMessageToClient(connID, protocol.MessageTypeErrorNotification, errorPayload, ""); sendErr != nil {
				if !isClientDisconnectedError(sendErr) {
					slog.Error("Failed to send timeout notification", "error", sendErr, "connID", connID)
				}
			}
		}
	}()

	return nil
}

// isClientDisconnectedError checks if the error indicates that the client has disconnected
func isClientDisconnectedError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common client disconnection error patterns
	errStr := err.Error()
	return strings.Contains(errStr, "client with ID") && strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "use of closed network connection") ||
		strings.Contains(errStr, "failed to send message to client")
}

// getCachedDeviceList attempts to get a cached device list with minimal blocking
// This is used as a fallback when the main device list fetch times out
func (ws *WebSocketServer) getCachedDeviceList() []handler.DeviceAndProperties {
	if ws.echonetClient == nil {
		return nil
	}

	// Try to get devices with a very short timeout using a goroutine
	devicesCh := make(chan []handler.DeviceAndProperties, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Warn("Panic while getting cached device list", "error", r)
			}
		}()
		// Make another attempt - this might succeed if locks have been released
		devices := ws.echonetClient.ListDevices(handler.FilterCriteria{ExcludeOffline: true})
		select {
		case devicesCh <- devices:
		default:
			// Channel is full, discard
		}
	}()

	// Wait up to 3 seconds for cached data
	select {
	case devices := <-devicesCh:
		return devices
	case <-time.After(3 * time.Second):
		slog.Warn("Cached device list fetch also timed out, returning empty list")
		return []handler.DeviceAndProperties{}
	}
}

// generateAndSendInitialState generates and sends the initial state data
func (ws *WebSocketServer) generateAndSendInitialState(connID string) error {
	if ws.handler.IsDebug() {
		slog.Debug("Starting initial state generation", "connID", connID)
	}

	// Get all devices with timeout-aware fetching
	if ws.handler.IsDebug() {
		slog.Debug("Fetching device list", "connID", connID)
	}
	var devices []handler.DeviceAndProperties
	if ws.echonetClient != nil {
		// Use goroutine with timeout to fetch devices
		devicesCh := make(chan []handler.DeviceAndProperties, 1)
		errorCh := make(chan error, 1)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorCh <- fmt.Errorf("panic in device list fetch: %v", r)
				}
			}()
			deviceList := ws.echonetClient.ListDevices(handler.FilterCriteria{ExcludeOffline: true})
			devicesCh <- deviceList
		}()

		// Use a shorter timeout for device list fetching (20 seconds instead of waiting for full 30s timeout)
		select {
		case devices = <-devicesCh:
			if ws.handler.IsDebug() {
				slog.Debug("Device list fetched successfully", "connID", connID, "deviceCount", len(devices))
			}
		case err := <-errorCh:
			return fmt.Errorf("error fetching device list: %w", err)
		case <-time.After(20 * time.Second):
			slog.Warn("Device list fetch timed out, using cached data if available", "connID", connID)
			// Try to get cached device list with minimal blocking
			devices = ws.getCachedDeviceList()
		}
	} else {
		slog.Warn("echonetClient is nil, returning empty device list", "connID", connID)
	}
	if ws.handler.IsDebug() {
		slog.Debug("Device list processing completed", "connID", connID, "deviceCount", len(devices))
	}

	// Convert devices to protocol format
	protoDevices := make(map[string]protocol.Device)
	for i, device := range devices {
		if ws.handler.IsDebug() && i < 5 { // Log first 5 devices to avoid spam
			slog.Debug("Processing device", "connID", connID, "device", device.Device.Specifier(), "index", i)
		}

		// デバイス構造体のnilチェック
		if device.Device.IP == nil {
			slog.Warn("Skipping device with nil IP", "connID", connID, "device", device.Device.Specifier())
			continue
		}

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

	if ws.handler.IsDebug() {
		slog.Debug("Device conversion completed", "connID", connID, "protoDeviceCount", len(protoDevices))
	}

	// Get all aliases with timeout
	if ws.handler.IsDebug() {
		slog.Debug("Fetching alias list", "connID", connID)
	}
	aliases := make(map[string]client.IDString)
	if ws.echonetClient != nil {
		aliasCh := make(chan []client.AliasIDStringPair, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Warn("Panic while fetching alias list", "error", r, "connID", connID)
				}
			}()
			aliasList := ws.echonetClient.AliasList()
			aliasCh <- aliasList
		}()

		select {
		case aliasList := <-aliasCh:
			for _, alias := range aliasList {
				if alias.Alias != "" && alias.ID != "" {
					aliases[alias.Alias] = alias.ID
				}
			}
		case <-time.After(5 * time.Second):
			slog.Warn("Alias list fetch timed out", "connID", connID)
			// Continue with empty aliases - this is not critical for initial state
		}
	} else {
		slog.Warn("echonetClient is nil for alias list", "connID", connID)
	}
	if ws.handler.IsDebug() {
		slog.Debug("Alias list processing completed", "connID", connID, "aliasCount", len(aliases))
	}

	// Get all groups with timeout
	if ws.handler.IsDebug() {
		slog.Debug("Fetching group list", "connID", connID)
	}
	groups := make(map[string][]client.IDString)
	if ws.echonetClient != nil {
		groupCh := make(chan []client.GroupDevicePair, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Warn("Panic while fetching group list", "error", r, "connID", connID)
				}
			}()
			groupList := ws.echonetClient.GroupList(nil)
			groupCh <- groupList
		}()

		select {
		case groupList := <-groupCh:
			for _, group := range groupList {
				if group.Group != "" {
					groups[group.Group] = group.Devices
				}
			}
		case <-time.After(5 * time.Second):
			slog.Warn("Group list fetch timed out", "connID", connID)
			// Continue with empty groups - this is not critical for initial state
		}
	} else {
		slog.Warn("echonetClient is nil for group list", "connID", connID)
	}
	if ws.handler.IsDebug() {
		slog.Debug("Group list processing completed", "connID", connID, "groupCount", len(groups))
	}

	// Create initial state payload
	payload := protocol.InitialStatePayload{
		Devices: protoDevices,
		Aliases: aliases,
		Groups:  groups,
	}

	if ws.handler.IsDebug() {
		slog.Debug("Sending initial state message", "connID", connID, "totalDevices", len(protoDevices), "totalAliases", len(aliases), "totalGroups", len(groups))
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
				slog.Info("Device offline notification received", "device", notification.Device.Specifier())

				// Create device offline payload
				device := notification.Device
				payload := protocol.DeviceOfflinePayload{
					IP:  device.IP.String(),
					EOJ: device.EOJ.Specifier(),
				}

				// Broadcast the message
				if err := ws.broadcastMessageToClients(protocol.MessageTypeDeviceOffline, payload); err != nil {
					if !isClientDisconnectedError(err) {
						slog.Error("Failed to broadcast device offline message", "error", err, "device", notification.Device.Specifier())
					}
				} else {
					slog.Info("Device offline message broadcasted", "device", notification.Device.Specifier())
				}

			case handler.DeviceOnline:
				slog.Info("Device online notification received", "device", notification.Device.Specifier())

				// Create device online payload
				device := notification.Device
				payload := protocol.DeviceOnlinePayload{
					IP:  device.IP.String(),
					EOJ: device.EOJ.Specifier(),
				}

				// Broadcast the message
				if err := ws.broadcastMessageToClients(protocol.MessageTypeDeviceOnline, payload); err != nil {
					if !isClientDisconnectedError(err) {
						slog.Error("Failed to broadcast device online message", "error", err, "device", notification.Device.Specifier())
					}
				} else {
					slog.Info("Device online message broadcasted", "device", notification.Device.Specifier())
				}
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

			// メッセージを非同期でブロードキャスト
			go func() {
				if err := ws.broadcastMessageToClients(protocol.MessageTypePropertyChanged, payload); err != nil {
					if !isClientDisconnectedError(err) {
						slog.Error("Failed to broadcast property change", "error", err, "device", propertyChange.Device.Specifier())
					}
				}
			}()
		}
	}
}
