package client

import (
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
	"encoding/base64"
	"fmt"
	"time"
)

// handleNotification handles a notification from the WebSocket server
func (c *WebSocketClient) handleNotification(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MessageTypeInitialState:
		c.handleInitialState(msg)
	case protocol.MessageTypeDeviceAdded:
		c.handleDeviceAdded(msg)
	case protocol.MessageTypeDeviceUpdated:
		c.handleDeviceUpdated(msg)
	case protocol.MessageTypeDeviceRemoved:
		c.handleDeviceRemoved(msg)
	case protocol.MessageTypeAliasChanged:
		c.handleAliasChanged(msg)
	case protocol.MessageTypeGroupChanged:
		c.handleGroupChanged(msg)
	case protocol.MessageTypePropertyChanged:
		c.handlePropertyChanged(msg)
	case protocol.MessageTypeTimeoutNotification:
		c.handleTimeoutNotification(msg)
	case protocol.MessageTypeErrorNotification:
		c.handleErrorNotification(msg)
	}
}

// handleInitialState handles an initial_state message
func (c *WebSocketClient) handleInitialState(msg *protocol.Message) {
	var payload protocol.InitialStatePayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if c.debug {
			fmt.Printf("Error parsing initial_state payload: %v\n", err)
		}
		return
	}

	// Update devices
	c.devicesMutex.Lock()
	c.lastSeenMutex.Lock()

	c.devices = make(map[string]echonet_lite.DeviceAndProperties)
	c.lastSeenTimes = make(map[string]time.Time)

	for deviceID, device := range payload.Devices {
		// Convert protocol.Device to echonet_lite types using DeviceFromProtocol
		ipAndEOJ, properties, err := protocol.DeviceFromProtocol(device)
		if err != nil {
			if c.debug {
				fmt.Printf("Error converting device: %v\n", err)
			}
			continue
		}

		// Properties are already in the correct format
		props := properties

		// Add to devices
		c.devices[deviceID] = echonet_lite.DeviceAndProperties{
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
		if c.debug {
			fmt.Printf("Error parsing device_added payload: %v\n", err)
		}
		return
	}

	// Convert protocol.Device to echonet_lite types using DeviceFromProtocol
	ipAndEOJ, props, err := protocol.DeviceFromProtocol(payload.Device)
	if err != nil {
		if c.debug {
			fmt.Printf("Error converting device: %v\n", err)
		}
		return
	}

	// Add to devices
	c.devicesMutex.Lock()
	c.lastSeenMutex.Lock()

	// ipAndEOJ.Specifier() をキーとして使用
	c.devices[ipAndEOJ.Specifier()] = echonet_lite.DeviceAndProperties{
		Device:     ipAndEOJ,
		Properties: props,
	}

	// Update lastSeenTimes
	c.lastSeenTimes[ipAndEOJ.Specifier()] = payload.Device.LastSeen

	c.lastSeenMutex.Unlock()
	c.devicesMutex.Unlock()
}

// handleDeviceUpdated handles a device_updated message
func (c *WebSocketClient) handleDeviceUpdated(msg *protocol.Message) {
	var payload protocol.DeviceUpdatedPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if c.debug {
			fmt.Printf("Error parsing device_updated payload: %v\n", err)
		}
		return
	}

	// Convert protocol.Device to echonet_lite types using DeviceFromProtocol
	ipAndEOJ, props, err := protocol.DeviceFromProtocol(payload.Device)
	if err != nil {
		if c.debug {
			fmt.Printf("Error converting device: %v\n", err)
		}
		return
	}

	// Update devices
	c.devicesMutex.Lock()
	c.lastSeenMutex.Lock()

	// ipAndEOJ.Specifier() をキーとして使用
	c.devices[ipAndEOJ.Specifier()] = echonet_lite.DeviceAndProperties{
		Device:     ipAndEOJ,
		Properties: props,
	}

	// Update lastSeenTimes
	c.lastSeenTimes[ipAndEOJ.Specifier()] = payload.Device.LastSeen

	c.lastSeenMutex.Unlock()
	c.devicesMutex.Unlock()
}

// handleDeviceRemoved handles a device_removed message
func (c *WebSocketClient) handleDeviceRemoved(msg *protocol.Message) {
	var payload protocol.DeviceRemovedPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if c.debug {
			fmt.Printf("Error parsing device_removed payload: %v\n", err)
		}
		return
	}

	// Parse the device identifier
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.IP + " " + payload.EOJ)
	if err != nil {
		if c.debug {
			fmt.Printf("Error parsing device identifier: %v\n", err)
		}
		return
	}

	// Remove from devices
	c.devicesMutex.Lock()
	c.lastSeenMutex.Lock()

	// ipAndEOJ.Specifier() をキーとして使用
	delete(c.devices, ipAndEOJ.Specifier())

	// Remove from lastSeenTimes
	delete(c.lastSeenTimes, ipAndEOJ.Specifier())

	c.lastSeenMutex.Unlock()
	c.devicesMutex.Unlock()
}

// handleAliasChanged handles an alias_changed message
func (c *WebSocketClient) handleAliasChanged(msg *protocol.Message) {
	var payload protocol.AliasChangedPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if c.debug {
			fmt.Printf("Error parsing alias_changed payload: %v\n", err)
		}
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
		if c.debug {
			fmt.Printf("Error parsing group_changed payload: %v\n", err)
		}
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
		if c.debug {
			fmt.Printf("Error parsing property_changed payload: %v\n", err)
		}
		return
	}

	// Parse the device identifier
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.IP + " " + payload.EOJ)
	if err != nil {
		if c.debug {
			fmt.Printf("Error parsing device identifier: %v\n", err)
		}
		return
	}

	// Parse the EPC
	epc, err := echonet_lite.ParseEPCString(payload.EPC)
	if err != nil {
		if c.debug {
			fmt.Printf("Error parsing EPC: %v\n", err)
		}
		return
	}

	// Parse the EDT
	edt, err := base64.StdEncoding.DecodeString(payload.Value)
	if err != nil {
		if c.debug {
			fmt.Printf("Error decoding EDT: %v\n", err)
		}
		return
	}

	// Update the property
	c.devicesMutex.Lock()
	// ipAndEOJ.Specifier() をキーとして使用
	if deviceProps, ok := c.devices[ipAndEOJ.Specifier()]; ok {
		// Create a new property
		newProp := echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		}

		// UpdatePropertyメソッドを使用してプロパティを更新
		deviceProps.Properties = deviceProps.Properties.UpdateProperty(newProp)
		c.devices[ipAndEOJ.Specifier()] = deviceProps
		if c.debug {
			fmt.Printf("プロパティ更新: %s EPC:%02X EDT:%X\n", ipAndEOJ.String(), byte(epc), edt)
		}
	}
	c.devicesMutex.Unlock()
}

// handleTimeoutNotification handles a timeout_notification message
func (c *WebSocketClient) handleTimeoutNotification(msg *protocol.Message) {
	var payload protocol.TimeoutNotificationPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if c.debug {
			fmt.Printf("Error parsing timeout_notification payload: %v\n", err)
		}
		return
	}

	// Always print the timeout notification, regardless of debug flag
	fmt.Printf("[TIMEOUT] Device %s %s: %s\n", payload.IP, payload.EOJ, payload.Message)
}

// handleErrorNotification handles an error_notification message
func (c *WebSocketClient) handleErrorNotification(msg *protocol.Message) {
	var payload protocol.ErrorNotificationPayload
	if err := protocol.ParsePayload(msg, &payload); err != nil {
		if c.debug {
			fmt.Printf("Error parsing error_notification payload: %v\n", err)
		}
		return
	}

	if c.debug {
		fmt.Printf("Error notification: %s: %s\n", payload.Code, payload.Message)
	}
}
