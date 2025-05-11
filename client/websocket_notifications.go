package client

import (
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"
)

// handleNotification handles a notification from the WebSocket server
func (c *WebSocketClient) handleNotification(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MessageTypeInitialState:
		c.handleInitialState(msg)
	case protocol.MessageTypeDeviceAdded:
		c.handleDeviceAdded(msg)
	case protocol.MessageTypeAliasChanged:
		c.handleAliasChanged(msg)
	case protocol.MessageTypeGroupChanged:
		c.handleGroupChanged(msg)
	case protocol.MessageTypePropertyChanged:
		c.handlePropertyChanged(msg)
	case protocol.MessageTypeTimeoutNotification:
		c.handleTimeoutNotification(msg)
	case protocol.MessageTypeDeviceOffline:
		c.handleDeviceOffline(msg)
	case protocol.MessageTypeErrorNotification:
		c.handleErrorNotification(msg)
	}
}

// handleInitialState handles an initial_state message
func (c *WebSocketClient) handleInitialState(msg *protocol.Message) {
	var payload protocol.InitialStatePayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		slog.Error("WebSocketClient.handleInitialState: Error parsing initial_state payload", "err", err)
		return
	}

	// Update devices
	c.devicesMutex.Lock()
	c.lastSeenMutex.Lock()

	c.devices = make(map[string]handler.DeviceAndProperties)
	c.lastSeenTimes = make(map[string]time.Time)

	for deviceID, device := range payload.Devices {
		// Convert protocol.Device to echonet_lite types using DeviceFromProtocol
		ipAndEOJ, properties, err := protocol.DeviceFromProtocol(device)
		if err != nil {
			slog.Error("WebSocketClient.handleInitialState: Error converting device", "err", err)
			continue
		}

		// Properties are already in the correct format
		props := properties

		// Add to devices
		c.devices[deviceID] = handler.DeviceAndProperties{
			Device:     ipAndEOJ,
			Properties: props,
		}

		// Update lastSeenTimes
		c.lastSeenTimes[ipAndEOJ.Specifier()] = device.LastSeen
	}

	c.lastSeenMutex.Unlock()
	c.devicesMutex.Unlock()

	// Update aliases
	c.aliasesMutex.Lock()
	c.aliases = make(map[string]IDString)
	for alias, id := range payload.Aliases {
		c.aliases[alias] = id
	}
	c.aliasesMutex.Unlock()

	// Update groups
	c.groupsMutex.Lock()
	c.groups = make([]GroupDevicePair, 0, len(payload.Groups))
	for groupName, devices := range payload.Groups {
		c.groups = append(c.groups, GroupDevicePair{
			Group:   groupName,
			Devices: devices,
		})
	}
	c.groupsMutex.Unlock()
}

// handleDeviceAdded handles a device_added message
func (c *WebSocketClient) handleDeviceAdded(msg *protocol.Message) {
	var payload protocol.DeviceAddedPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		slog.Error("WebSocketClient.handleDeviceAdded: Error parsing device_added payload", "err", err)
		return
	}

	// Convert protocol.Device to echonet_lite types using DeviceFromProtocol
	ipAndEOJ, props, err := protocol.DeviceFromProtocol(payload.Device)
	if err != nil {
		slog.Error("WebSocketClient.handleDeviceAdded: Error converting device", "err", err)
		return
	}

	// Add to devices
	c.devicesMutex.Lock()
	c.lastSeenMutex.Lock()

	// ipAndEOJ.Specifier() をキーとして使用
	c.devices[ipAndEOJ.Specifier()] = handler.DeviceAndProperties{
		Device:     ipAndEOJ,
		Properties: props,
	}

	// Update lastSeenTimes
	c.lastSeenTimes[ipAndEOJ.Specifier()] = payload.Device.LastSeen

	c.lastSeenMutex.Unlock()
	c.devicesMutex.Unlock()
}

// handleAliasChanged handles an alias_changed message
func (c *WebSocketClient) handleAliasChanged(msg *protocol.Message) {
	var payload protocol.AliasChangedPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		slog.Error("WebSocketClient.handleAliasChanged: Error parsing alias_changed payload", "err", err)
		return
	}

	c.aliasesMutex.Lock()
	defer c.aliasesMutex.Unlock()

	switch payload.ChangeType {
	case protocol.AliasChangeTypeAdded, protocol.AliasChangeTypeUpdated:
		// Add or update the alias
		c.aliases[payload.Alias] = payload.Target

	case protocol.AliasChangeTypeDeleted:
		// Remove the alias
		delete(c.aliases, payload.Alias)
	}
}

// handleGroupChanged handles a group_changed message
func (c *WebSocketClient) handleGroupChanged(msg *protocol.Message) {
	var payload protocol.GroupChangedPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		slog.Error("WebSocketClient.handleGroupChanged: Error parsing group_changed payload", "err", err)
		return
	}

	c.groupsMutex.Lock()
	defer c.groupsMutex.Unlock()

	switch payload.ChangeType {
	case protocol.GroupChangeTypeAdded:
		// グループが追加された場合
		c.groups = append(c.groups, GroupDevicePair{
			Group:   payload.Group,
			Devices: payload.Devices,
		})

	case protocol.GroupChangeTypeUpdated:
		// グループが更新された場合
		found := false
		for i, group := range c.groups {
			if group.Group == payload.Group {
				// 既存のグループを更新
				c.groups[i].Devices = payload.Devices
				found = true
				break
			}
		}
		if !found && len(payload.Devices) > 0 {
			// グループが見つからない場合は追加
			c.groups = append(c.groups, GroupDevicePair{
				Group:   payload.Group,
				Devices: payload.Devices,
			})
		}

	case protocol.GroupChangeTypeDeleted:
		// グループが削除された場合
		for i, group := range c.groups {
			if group.Group == payload.Group {
				// グループを削除
				c.groups = append(c.groups[:i], c.groups[i+1:]...)
				break
			}
		}
	}
}

// handlePropertyChanged handles a property_changed message
func (c *WebSocketClient) handlePropertyChanged(msg *protocol.Message) {
	var payload protocol.PropertyChangedPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		slog.Error("WebSocketClient.handlePropertyChanged: Error parsing property_changed payload", "err", err)
		return
	}

	// Parse the device identifier
	ipAndEOJ, err := handler.ParseDeviceIdentifier(payload.IP + " " + payload.EOJ)
	if err != nil {
		slog.Error("WebSocketClient.handlePropertyChanged: Error parsing device identifier", "err", err)
		return
	}

	// Parse the EPC
	epc, err := handler.ParseEPCString(payload.EPC)
	if err != nil {
		slog.Error("WebSocketClient.handlePropertyChanged: Error parsing EPC", "err", err)
		return
	}

	edt, err := base64.StdEncoding.DecodeString(payload.Value.EDT)
	if err != nil {
		slog.Error("WebSocketClient.handlePropertyChanged: Error decoding EDT", "err", err)
		return
	}

	// Update the property
	c.devicesMutex.Lock()
	key := ipAndEOJ.Specifier()
	if deviceProps, ok := c.devices[key]; ok {
		// UpdatePropertyメソッドを使用してプロパティを更新
		newProp := echonet_lite.Property{EPC: epc, EDT: edt}
		deviceProps.Properties = deviceProps.Properties.UpdateProperty(newProp)
		c.devices[key] = deviceProps
		if c.debug {
			slog.Info("WebSocketClient.handlePropertyChanged: プロパティ更新",
				"device", ipAndEOJ.String(),
				"epc", fmt.Sprintf("%02X", byte(epc)),
				"edt", fmt.Sprintf("%X", edt),
			)
		}
	}
	c.devicesMutex.Unlock()
}

// handleTimeoutNotification handles a timeout_notification message
func (c *WebSocketClient) handleTimeoutNotification(msg *protocol.Message) {
	var payload protocol.TimeoutNotificationPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		slog.Error("WebSocketClient.handleTimeoutNotification: Error parsing timeout_notification payload", "err", err)
		return
	}

	// Always print the timeout notification, regardless of debug flag
	fmt.Printf("[TIMEOUT] Device %s %s: %s\n", payload.IP, payload.EOJ, payload.Message)
}

// handleDeviceOffline handles a device_offline message
func (c *WebSocketClient) handleDeviceOffline(msg *protocol.Message) {
	var payload protocol.DeviceOfflinePayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		slog.Error("WebSocketClient.handleDeviceOffline: Error parsing device_offline payload", "err", err)
		return
	}

	// Parse the device identifier
	ipAndEOJ, err := handler.ParseDeviceIdentifier(payload.IP + " " + payload.EOJ)
	if err != nil {
		slog.Error("WebSocketClient.handleDeviceOffline: Error parsing device identifier for offline notification", "err", err)
		return
	}

	deviceID := ipAndEOJ.Specifier()

	// Remove from devices and lastSeenTimes
	c.devicesMutex.Lock()
	c.lastSeenMutex.Lock()

	delete(c.devices, deviceID)
	delete(c.lastSeenTimes, deviceID)

	c.lastSeenMutex.Unlock()
	c.devicesMutex.Unlock()

	if c.debug {
		slog.Info("WebSocketClient.handleDeviceOffline: [OFFLINE] Device removed due to offline status",
			"deviceID", deviceID,
		)
	}
}

// handleErrorNotification handles an error_notification message
func (c *WebSocketClient) handleErrorNotification(msg *protocol.Message) {
	var payload protocol.ErrorNotificationPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		slog.Error("WebSocketClient.handleErrorNotification: Error parsing error_notification payload", "err", err)
		return
	}

	if c.debug {
		slog.Info("WebSocketClient.handleErrorNotification: Error notification",
			"code", payload.Code,
			"message", payload.Message,
		)
	}
}
