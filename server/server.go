package server

import (
	"context"
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketのアップグレーダーを定義
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 開発用に全てのオリジンを許可
	},
}

// ECHONETLiteServer はWebSocket経由でECHONETLiteHandlerを操作するサーバー
type ECHONETLiteServer struct {
	handler      *echonet_lite.ECHONETLiteHandler
	clients      map[*websocket.Conn]bool
	clientsMutex sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewECHONETLiteServer は新しいECHONETLiteServerを作成する
func NewECHONETLiteServer(ctx context.Context, handler *echonet_lite.ECHONETLiteHandler) *ECHONETLiteServer {
	serverCtx, cancel := context.WithCancel(ctx)
	
	return &ECHONETLiteServer{
		handler:      handler,
		clients:      make(map[*websocket.Conn]bool),
		clientsMutex: sync.Mutex{},
		ctx:          serverCtx,
		cancel:       cancel,
	}
}

// Start はサーバーを指定されたアドレスで起動する
func (s *ECHONETLiteServer) Start(addr string) error {
	// WebSocketハンドラーを設定
	http.HandleFunc("/ws", s.handleWebSocket)

	// 通知を監視するゴルーチンを起動
	go s.handleNotifications()

	// HTTPサーバーを作成
	server := &http.Server{
		Addr:    addr,
		Handler: nil, // DefaultServeMux使用
	}

	// サーバーをゴルーチンで起動
	go func() {
		log.Printf("WebSocketサーバーを起動しています: %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("サーバー起動エラー: %v", err)
		}
	}()

	// コンテキストの完了を待機
	<-s.ctx.Done()

	// サーバーをグレースフルに停止
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("サーバーのシャットダウンエラー: %w", err)
	}

	return nil
}

// Stop はサーバーを停止する
func (s *ECHONETLiteServer) Stop() {
	s.cancel()
}

// handleNotifications はECHONETLiteHandlerからの通知を処理する
func (s *ECHONETLiteServer) handleNotifications() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case notification, ok := <-s.handler.NotificationCh:
			if !ok {
				// チャネルが閉じられた場合は終了
				return
			}

			// 通知の種類に応じて処理
			switch notification.Type {
			case echonet_lite.DeviceAdded:
				s.broadcastNotification("deviceAdded", notification.Device, nil)
			case echonet_lite.DeviceTimeout:
				s.broadcastNotification("deviceTimeout", notification.Device, notification.Error)
			}
		}
	}
}

// broadcastNotification は全クライアントに通知を送信する
func (s *ECHONETLiteServer) broadcastNotification(eventType string, device echonet_lite.IPAndEOJ, err error) {
	aliases := s.handler.GetAliases(device)
	deviceInfo := protocol.ConvertIPAndEOJToDeviceInfo(device, aliases)

	notification := protocol.NotificationMessage{
		Message: protocol.Message{
			Type: "notification",
			ID:   uuid.New().String(),
		},
		Event:      eventType,
		DeviceInfo: deviceInfo,
	}

	if err != nil {
		notification.Data = map[string]string{
			"error": err.Error(),
		}
	}

	message, jsonErr := json.Marshal(notification)
	if jsonErr != nil {
		log.Printf("通知のJSONエンコードに失敗: %v", jsonErr)
		return
	}

	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	for client := range s.clients {
		if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("WebSocket書き込みエラー: %v", err)
			client.Close()
			delete(s.clients, client)
		}
	}
}

// handleWebSocket はWebSocket接続リクエストを処理する
func (s *ECHONETLiteServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// WebSocketにアップグレード
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket接続のアップグレードに失敗: %v", err)
		return
	}
	defer conn.Close()

	// クライアント登録
	s.clientsMutex.Lock()
	s.clients[conn] = true
	s.clientsMutex.Unlock()

	// クライアント登録解除（接続終了時）
	defer func() {
		s.clientsMutex.Lock()
		delete(s.clients, conn)
		s.clientsMutex.Unlock()
	}()

	// メッセージの処理ループ
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket読み取りエラー: %v", err)
			}
			break
		}

		// メッセージをデコード
		var cmd protocol.CommandMessage
		if err := json.Unmarshal(message, &cmd); err != nil {
			s.sendErrorResponse(conn, "", "無効なJSONメッセージ", err)
			continue
		}

		// コマンド処理（非同期）
		go s.handleCommand(conn, cmd)
	}
}

// handleCommand はクライアントからのコマンドを処理する
func (s *ECHONETLiteServer) handleCommand(conn *websocket.Conn, cmd protocol.CommandMessage) {
	var success bool
	var data interface{}
	var err error

	// コマンドの種類に応じて処理
	switch cmd.Command {
	case "discover":
		success, data, err = s.handleDiscoverCommand(cmd)
	case "devices":
		success, data, err = s.handleDevicesCommand(cmd)
	case "get":
		success, data, err = s.handleGetCommand(cmd)
	case "set":
		success, data, err = s.handleSetCommand(cmd)
	case "update":
		success, data, err = s.handleUpdateCommand(cmd)
	case "debug":
		success, data, err = s.handleDebugCommand(cmd)
	case "alias":
		success, data, err = s.handleAliasCommand(cmd)
	default:
		err = fmt.Errorf("不明なコマンド: %s", cmd.Command)
		success = false
	}

	// エラーがあれば送信
	if err != nil {
		s.sendErrorResponse(conn, cmd.ID, fmt.Sprintf("コマンド '%s' の実行中にエラーが発生", cmd.Command), err)
		return
	}

	// 成功レスポンスを送信
	s.sendResponse(conn, cmd.ID, success, data)
}

// sendResponse はクライアントに成功レスポンスを送信する
func (s *ECHONETLiteServer) sendResponse(conn *websocket.Conn, id string, success bool, data interface{}) {
	response := protocol.ResponseMessage{
		Message: protocol.Message{
			Type: "response",
			ID:   id,
		},
		Success: success,
		Data:    data,
	}

	message, err := json.Marshal(response)
	if err != nil {
		log.Printf("レスポンスのJSONエンコードに失敗: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
		log.Printf("WebSocket書き込みエラー: %v", err)
	}
}

// sendErrorResponse はクライアントにエラーレスポンスを送信する
func (s *ECHONETLiteServer) sendErrorResponse(conn *websocket.Conn, id string, message string, err error) {
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	response := protocol.ResponseMessage{
		Message: protocol.Message{
			Type: "response",
			ID:   id,
		},
		Success: false,
		Error:   errorMsg,
	}

	jsonData, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		log.Printf("エラーレスポンスのJSONエンコードに失敗: %v", jsonErr)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Printf("WebSocket書き込みエラー: %v", err)
	}
}

// ここから各コマンドのハンドラー実装
func (s *ECHONETLiteServer) handleDiscoverCommand(cmd protocol.CommandMessage) (bool, interface{}, error) {
	err := s.handler.Discover()
	return err == nil, nil, err
}

func (s *ECHONETLiteServer) handleDevicesCommand(cmd protocol.CommandMessage) (bool, interface{}, error) {
	// オプションからプロパティモードを取得
	var propMode echonet_lite.PropertyDisplayMode
	var deviceSpec echonet_lite.DeviceSpecifier
	var properties []echonet_lite.Property

	// DeviceSpecの処理
	if cmd.DeviceSpec != nil {
		specData, err := json.Marshal(cmd.DeviceSpec)
		if err != nil {
			return false, nil, fmt.Errorf("DeviceSpecのエンコードに失敗: %w", err)
		}

		var protocolSpec protocol.DeviceSpecifier
		if err := json.Unmarshal(specData, &protocolSpec); err != nil {
			return false, nil, fmt.Errorf("DeviceSpecのデコードに失敗: %w", err)
		}

		deviceSpec = protocol.ConvertDeviceSpecifierToEchonetDeviceSpecifier(protocolSpec)
	}

	// オプションの処理
	if cmd.Options != nil {
		optionsData, err := json.Marshal(cmd.Options)
		if err != nil {
			return false, nil, fmt.Errorf("オプションのエンコードに失敗: %w", err)
		}

		var options protocol.CommandOptions
		if err := json.Unmarshal(optionsData, &options); err != nil {
			return false, nil, fmt.Errorf("オプションのデコードに失敗: %w", err)
		}

		if options.PropMode != nil {
			propMode = echonet_lite.PropertyDisplayMode(*options.PropMode)
		}
	}

	// プロパティの処理（フィルタリング条件）
	if cmd.Properties != nil {
		propsData, err := json.Marshal(cmd.Properties)
		if err != nil {
			return false, nil, fmt.Errorf("プロパティのエンコードに失敗: %w", err)
		}

		var protocolProps []protocol.Property
		if err := json.Unmarshal(propsData, &protocolProps); err != nil {
			return false, nil, fmt.Errorf("プロパティのデコードに失敗: %w", err)
		}

		for _, prop := range protocolProps {
			properties = append(properties, protocol.ConvertPropertyToEchonetProperty(prop))
		}
	}

	// フィルタリング条件の作成
	criteria := echonet_lite.FilterCriteria{
		Device:         deviceSpec,
		PropertyValues: properties,
	}

	// デバイスリストの取得
	result := s.handler.ListDevices(criteria)

	// 結果をprotocolパッケージの型に変換
	deviceResults := make([]protocol.DevicePropertyResult, len(result))
	for i, deviceResult := range result {
		aliases := s.handler.GetAliases(deviceResult.Device)
		deviceInfo := protocol.ConvertIPAndEOJToDeviceInfo(deviceResult.Device, aliases)

		propInfos := make([]protocol.PropertyInfo, len(deviceResult.Properties))
		for j, prop := range deviceResult.Properties {
			propInfos[j] = protocol.ConvertPropertyToPropertyInfo(prop, deviceResult.Device.EOJ.ClassCode())
		}

		deviceResults[i] = protocol.DevicePropertyResult{
			Device:     deviceInfo,
			Properties: propInfos,
			Success:    true,
		}
	}

	return true, deviceResults, nil
}

func (s *ECHONETLiteServer) handleGetCommand(cmd protocol.CommandMessage) (bool, interface{}, error) {
	// DeviceSpecのパース
	var deviceSpec echonet_lite.DeviceSpecifier
	if cmd.DeviceSpec != nil {
		specData, err := json.Marshal(cmd.DeviceSpec)
		if err != nil {
			return false, nil, fmt.Errorf("DeviceSpecのエンコードに失敗: %w", err)
		}

		var protocolSpec protocol.DeviceSpecifier
		if err := json.Unmarshal(specData, &protocolSpec); err != nil {
			return false, nil, fmt.Errorf("DeviceSpecのデコードに失敗: %w", err)
		}

		deviceSpec = protocol.ConvertDeviceSpecifierToEchonetDeviceSpecifier(protocolSpec)
	}

	// エイリアスの処理
	if deviceSpec.IP == nil && deviceSpec.ClassCode == nil && deviceSpec.InstanceCode == nil {
		if cmd.DeviceSpec != nil {
			specData, _ := json.Marshal(cmd.DeviceSpec)
			var protocolSpec protocol.DeviceSpecifier
			_ = json.Unmarshal(specData, &protocolSpec)

			if protocolSpec.Alias != nil {
				device, err := s.handler.AliasGet(protocolSpec.Alias)
				if err != nil {
					return false, nil, err
				}
				
				classCode := device.EOJ.ClassCode()
				instanceCode := device.EOJ.InstanceCode()
				deviceSpec.IP = &device.IP
				deviceSpec.ClassCode = &classCode
				deviceSpec.InstanceCode = &instanceCode
			}
		}
	}

	// 実装するには、DeviceSpecifierから単一のデバイスを取得する処理が必要
	devices := s.handler.GetDevices(deviceSpec)
	if len(devices) == 0 {
		return false, nil, fmt.Errorf("デバイスが見つかりません")
	}
	if len(devices) > 1 {
		return false, nil, fmt.Errorf("複数のデバイスが見つかりました。より具体的な条件が必要です")
	}

	// EPCsの取得
	epcs := make([]echonet_lite.EPCType, len(cmd.EPCs))
	for i, epc := range cmd.EPCs {
		epcs[i] = echonet_lite.EPCType(epc)
	}

	// オプションの処理
	skipValidation := false
	if cmd.Options != nil {
		optionsData, err := json.Marshal(cmd.Options)
		if err != nil {
			return false, nil, fmt.Errorf("オプションのエンコードに失敗: %w", err)
		}

		var options protocol.CommandOptions
		if err := json.Unmarshal(optionsData, &options); err != nil {
			return false, nil, fmt.Errorf("オプションのデコードに失敗: %w", err)
		}

		if options.SkipValidation != nil {
			skipValidation = *options.SkipValidation
		}
	}

	// プロパティの取得
	result, err := s.handler.GetProperties(devices[0], epcs, skipValidation)
	if err != nil {
		return false, nil, err
	}

	// 結果をprotocolパッケージの型に変換
	aliases := s.handler.GetAliases(result.Device)
	deviceInfo := protocol.ConvertIPAndEOJToDeviceInfo(result.Device, aliases)

	propInfos := make([]protocol.PropertyInfo, len(result.Properties))
	for i, prop := range result.Properties {
		propInfos[i] = protocol.ConvertPropertyToPropertyInfo(prop, result.Device.EOJ.ClassCode())
	}

	deviceResult := protocol.DevicePropertyResult{
		Device:     deviceInfo,
		Properties: propInfos,
		Success:    true,
	}

	return true, deviceResult, nil
}

func (s *ECHONETLiteServer) handleSetCommand(cmd protocol.CommandMessage) (bool, interface{}, error) {
	// DeviceSpecのパース
	var deviceSpec echonet_lite.DeviceSpecifier
	if cmd.DeviceSpec != nil {
		specData, err := json.Marshal(cmd.DeviceSpec)
		if err != nil {
			return false, nil, fmt.Errorf("DeviceSpecのエンコードに失敗: %w", err)
		}

		var protocolSpec protocol.DeviceSpecifier
		if err := json.Unmarshal(specData, &protocolSpec); err != nil {
			return false, nil, fmt.Errorf("DeviceSpecのデコードに失敗: %w", err)
		}

		deviceSpec = protocol.ConvertDeviceSpecifierToEchonetDeviceSpecifier(protocolSpec)
	}

	// エイリアスの処理
	if deviceSpec.IP == nil && deviceSpec.ClassCode == nil && deviceSpec.InstanceCode == nil {
		if cmd.DeviceSpec != nil {
			specData, _ := json.Marshal(cmd.DeviceSpec)
			var protocolSpec protocol.DeviceSpecifier
			_ = json.Unmarshal(specData, &protocolSpec)

			if protocolSpec.Alias != nil {
				device, err := s.handler.AliasGet(protocolSpec.Alias)
				if err != nil {
					return false, nil, err
				}
				
				classCode := device.EOJ.ClassCode()
				instanceCode := device.EOJ.InstanceCode()
				deviceSpec.IP = &device.IP
				deviceSpec.ClassCode = &classCode
				deviceSpec.InstanceCode = &instanceCode
			}
		}
	}

	// 単一デバイスの取得
	devices := s.handler.GetDevices(deviceSpec)
	if len(devices) == 0 {
		return false, nil, fmt.Errorf("デバイスが見つかりません")
	}
	if len(devices) > 1 {
		return false, nil, fmt.Errorf("複数のデバイスが見つかりました。より具体的な条件が必要です")
	}

	// プロパティの処理
	var properties []echonet_lite.Property
	if cmd.Properties != nil {
		propsData, err := json.Marshal(cmd.Properties)
		if err != nil {
			return false, nil, fmt.Errorf("プロパティのエンコードに失敗: %w", err)
		}

		var protocolProps []protocol.Property
		if err := json.Unmarshal(propsData, &protocolProps); err != nil {
			return false, nil, fmt.Errorf("プロパティのデコードに失敗: %w", err)
		}

		for _, prop := range protocolProps {
			properties = append(properties, protocol.ConvertPropertyToEchonetProperty(prop))
		}
	}

	// プロパティの設定
	result, err := s.handler.SetProperties(devices[0], properties)
	if err != nil {
		return false, nil, err
	}

	// 結果をprotocolパッケージの型に変換
	aliases := s.handler.GetAliases(result.Device)
	deviceInfo := protocol.ConvertIPAndEOJToDeviceInfo(result.Device, aliases)

	propInfos := make([]protocol.PropertyInfo, len(result.Properties))
	for i, prop := range result.Properties {
		propInfos[i] = protocol.ConvertPropertyToPropertyInfo(prop, result.Device.EOJ.ClassCode())
	}

	deviceResult := protocol.DevicePropertyResult{
		Device:     deviceInfo,
		Properties: propInfos,
		Success:    true,
	}

	return true, deviceResult, nil
}

func (s *ECHONETLiteServer) handleUpdateCommand(cmd protocol.CommandMessage) (bool, interface{}, error) {
	// DeviceSpecのパース
	var deviceSpec echonet_lite.DeviceSpecifier
	if cmd.DeviceSpec != nil {
		specData, err := json.Marshal(cmd.DeviceSpec)
		if err != nil {
			return false, nil, fmt.Errorf("DeviceSpecのエンコードに失敗: %w", err)
		}

		var protocolSpec protocol.DeviceSpecifier
		if err := json.Unmarshal(specData, &protocolSpec); err != nil {
			return false, nil, fmt.Errorf("DeviceSpecのデコードに失敗: %w", err)
		}

		deviceSpec = protocol.ConvertDeviceSpecifierToEchonetDeviceSpecifier(protocolSpec)
	}

	// フィルタリング条件の作成
	criteria := echonet_lite.FilterCriteria{
		Device: deviceSpec,
	}

	// プロパティの更新
	err := s.handler.UpdateProperties(criteria)
	if err != nil {
		return false, nil, err
	}

	return true, nil, nil
}

func (s *ECHONETLiteServer) handleDebugCommand(cmd protocol.CommandMessage) (bool, interface{}, error) {
	// オプションからデバッグモードを取得
	if cmd.Options != nil {
		optionsData, err := json.Marshal(cmd.Options)
		if err != nil {
			return false, nil, fmt.Errorf("オプションのエンコードに失敗: %w", err)
		}

		var options protocol.CommandOptions
		if err := json.Unmarshal(optionsData, &options); err != nil {
			return false, nil, fmt.Errorf("オプションのデコードに失敗: %w", err)
		}

		if options.DebugMode != nil {
			debugMode := *options.DebugMode == "on"
			s.handler.SetDebug(debugMode)
			return true, map[string]bool{"debug": debugMode}, nil
		}
	}

	// デバッグモードの取得
	return true, map[string]bool{"debug": s.handler.IsDebug()}, nil
}

func (s *ECHONETLiteServer) handleAliasCommand(cmd protocol.CommandMessage) (bool, interface{}, error) {
	// TODO: エイリアスコマンドの実装
	// list, get, set, deleteの各サブコマンドを処理
	return true, nil, fmt.Errorf("エイリアスコマンドは未実装です")
}