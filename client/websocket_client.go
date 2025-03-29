package client

import (
	"context"
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient implements the ECHONETListClient interface using WebSocket
type WebSocketClient struct {
	ctx             context.Context
	cancel          context.CancelFunc
	conn            *websocket.Conn
	url             string
	debug           bool
	devices         map[string]echonet_lite.DeviceAndProperties
	devicesMutex    sync.RWMutex
	aliases         map[string]echonet_lite.IPAndEOJ
	aliasesMutex    sync.RWMutex
	requestID       int
	requestIDMutex  sync.Mutex
	responseCh      map[string]chan *protocol.Message
	responseChMutex sync.Mutex
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(ctx context.Context, serverURL string, debug bool) (*WebSocketClient, error) {
	// Validate the URL
	_, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	clientCtx, cancel := context.WithCancel(ctx)

	client := &WebSocketClient{
		ctx:        clientCtx,
		cancel:     cancel,
		url:        serverURL,
		debug:      debug,
		devices:    make(map[string]echonet_lite.DeviceAndProperties),
		aliases:    make(map[string]echonet_lite.IPAndEOJ),
		responseCh: make(map[string]chan *protocol.Message),
	}

	return client, nil
}

// Connect connects to the WebSocket server
func (c *WebSocketClient) Connect() error {
	// Connect to the WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("error connecting to WebSocket server: %v", err)
	}
	c.conn = conn

	// Start listening for messages
	go c.listenForMessages()

	return nil
}

// Close closes the WebSocket connection
func (c *WebSocketClient) Close() error {
	c.cancel()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsDebug returns whether debug mode is enabled
func (c *WebSocketClient) IsDebug() bool {
	return c.debug
}

// SetDebug sets the debug mode
func (c *WebSocketClient) SetDebug(debug bool) {
	c.debug = debug
}

// Discover sends a discover_devices message to the server
func (c *WebSocketClient) Discover() error {
	// Create the payload
	payload := protocol.DiscoverDevicesPayload{}

	// Send the message
	_, err := c.sendRequest(protocol.MessageTypeDiscoverDevices, payload)
	if err != nil {
		fmt.Printf("Error discovering devices: %v\n", err)
	}
	return err
}

// UpdateProperties sends an update_properties message to the server
func (c *WebSocketClient) UpdateProperties(criteria FilterCriteria) error {
	// Get devices matching the criteria
	devices := c.GetDevices(criteria.Device)
	if len(devices) == 0 {
		return fmt.Errorf("no devices match the criteria")
	}

	// Create the payload
	targets := make([]string, 0, len(devices))
	for _, device := range devices {
		targets = append(targets, device.Specifier())
	}

	payload := protocol.UpdatePropertiesPayload{
		Targets: targets,
	}

	// Send the message
	_, err := c.sendRequest(protocol.MessageTypeUpdateProperties, payload)
	if err != nil {
		fmt.Printf("Error updating properties: %v\n", err)
	}
	return err
}

// GetDevices returns devices matching the given device specifier
func (c *WebSocketClient) GetDevices(deviceSpec DeviceSpecifier) []IPAndEOJ {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	var result []IPAndEOJ

	for _, device := range c.devices {
		ipAndEOJ := device.Device

		// Filter by IP
		if deviceSpec.IP != nil && !ipAndEOJ.IP.Equal(*deviceSpec.IP) {
			continue
		}

		// Filter by class code
		if deviceSpec.ClassCode != nil && ipAndEOJ.EOJ.ClassCode() != *deviceSpec.ClassCode {
			continue
		}

		// Filter by instance code
		if deviceSpec.InstanceCode != nil && ipAndEOJ.EOJ.InstanceCode() != *deviceSpec.InstanceCode {
			continue
		}

		result = append(result, ipAndEOJ)
	}

	return result
}

// ListDevices returns devices and their properties matching the given criteria
func (c *WebSocketClient) ListDevices(criteria FilterCriteria) []DeviceAndProperties {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	var result []DeviceAndProperties

	// Get devices matching the device specifier
	devices := c.GetDevices(criteria.Device)

	// Filter by property values if specified
	for _, ipAndEOJ := range devices {
		deviceAndProps, ok := c.devices[ipAndEOJ.Specifier()]
		if !ok {
			continue
		}

		// Check if the device has all the specified property values
		match := true
		if len(criteria.PropertyValues) > 0 {
			// For now, we don't check property values since we're using Properties as a slice
			// This would need to be implemented differently
			match = false
		}

		if match {
			result = append(result, deviceAndProps)
		}
	}

	return result
}

// GetProperties gets properties from a device
func (c *WebSocketClient) GetProperties(device IPAndEOJ, EPCs []EPCType, skipValidation bool) (DeviceAndProperties, error) {
	// Check if the device exists
	if !skipValidation {
		c.devicesMutex.RLock()
		_, ok := c.devices[device.Specifier()]
		c.devicesMutex.RUnlock()
		if !ok {
			return DeviceAndProperties{}, fmt.Errorf("device not found: %v", device)
		}
	}

	// Create the payload
	epcs := make([]string, 0, len(EPCs))
	for _, epc := range EPCs {
		epcs = append(epcs, fmt.Sprintf("%02X", byte(epc)))
	}

	payload := protocol.GetPropertiesPayload{
		Targets: []string{device.Specifier()},
		EPCs:    epcs,
	}

	// Send the message
	response, err := c.sendRequest(protocol.MessageTypeGetProperties, payload)
	if err != nil {
		return DeviceAndProperties{}, err
	}

	// Parse the response
	var resultPayload protocol.CommandResultPayload
	if err := protocol.ParsePayload(response, &resultPayload); err != nil {
		return DeviceAndProperties{}, fmt.Errorf("error parsing response: %v", err)
	}

	if !resultPayload.Success {
		if resultPayload.Error != nil {
			return DeviceAndProperties{}, fmt.Errorf("error getting properties: %s: %s", resultPayload.Error.Code, resultPayload.Error.Message)
		}
		return DeviceAndProperties{}, fmt.Errorf("error getting properties: unknown error")
	}

	// Parse the device data
	var deviceData protocol.Device
	if resultPayload.Data != nil {
		if err := json.Unmarshal(resultPayload.Data, &deviceData); err != nil {
			return DeviceAndProperties{}, fmt.Errorf("error parsing device data: %v", err)
		}
	}

	// Convert to DeviceAndProperties
	var props echonet_lite.Properties
	for epcStr, edtStr := range deviceData.Properties {
		epc, err := echonet_lite.ParseEPCString(epcStr)
		if err != nil {
			return DeviceAndProperties{}, fmt.Errorf("error parsing EPC: %v", err)
		}

		edt, err := base64.StdEncoding.DecodeString(edtStr)
		if err != nil {
			return DeviceAndProperties{}, fmt.Errorf("error decoding EDT: %v", err)
		}

		props = append(props, echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		})
	}

	// Parse the device identifier
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(deviceData.IP + " " + deviceData.EOJ)
	if err != nil {
		return DeviceAndProperties{}, fmt.Errorf("error parsing device identifier(GetProperties): %v deviceData: %#v", err, deviceData)
	}

	return DeviceAndProperties{
		Device:     ipAndEOJ,
		Properties: props,
	}, nil
}

// SetProperties sets properties on a device
func (c *WebSocketClient) SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error) {
	// Create the payload
	propsMap := make(map[string]string)
	for _, prop := range properties {
		propsMap[fmt.Sprintf("%02X", byte(prop.EPC))] = base64.StdEncoding.EncodeToString(prop.EDT)
	}

	payload := protocol.SetPropertiesPayload{
		Target:     device.Specifier(),
		Properties: propsMap,
	}

	// Send the message
	response, err := c.sendRequest(protocol.MessageTypeSetProperties, payload)
	if err != nil {
		return DeviceAndProperties{}, err
	}

	// Parse the response
	var resultPayload protocol.CommandResultPayload
	if err := protocol.ParsePayload(response, &resultPayload); err != nil {
		return DeviceAndProperties{}, fmt.Errorf("error parsing response: %v", err)
	}

	if !resultPayload.Success {
		if resultPayload.Error != nil {
			return DeviceAndProperties{}, fmt.Errorf("error setting properties: %s: %s", resultPayload.Error.Code, resultPayload.Error.Message)
		}
		return DeviceAndProperties{}, fmt.Errorf("error setting properties: unknown error")
	}

	// Parse the device data
	var deviceData protocol.Device
	if resultPayload.Data != nil {
		if err := json.Unmarshal(resultPayload.Data, &deviceData); err != nil {
			return DeviceAndProperties{}, fmt.Errorf("error parsing device data: %v", err)
		}
	}

	// Convert to DeviceAndProperties
	var props echonet_lite.Properties
	for epcStr, edtStr := range deviceData.Properties {
		epc, err := echonet_lite.ParseEPCString(epcStr)
		if err != nil {
			return DeviceAndProperties{}, fmt.Errorf("error parsing EPC: %v", err)
		}

		edt, err := base64.StdEncoding.DecodeString(edtStr)
		if err != nil {
			return DeviceAndProperties{}, fmt.Errorf("error decoding EDT: %v", err)
		}

		props = append(props, echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		})
	}

	// Parse the device identifier
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(deviceData.IP + " " + deviceData.EOJ)
	if err != nil {
		return DeviceAndProperties{}, fmt.Errorf("error parsing device identifier: %v", err)
	}

	return DeviceAndProperties{
		Device:     ipAndEOJ,
		Properties: props,
	}, nil
}

// AliasList returns a list of all aliases
func (c *WebSocketClient) AliasList() []AliasDevicePair {
	c.aliasesMutex.RLock()
	defer c.aliasesMutex.RUnlock()

	var result []AliasDevicePair
	for alias, device := range c.aliases {
		// Create a copy of the device to avoid reference issues
		deviceCopy := device
		// Create a new AliasDevicePair with the alias and device pointer
		result = append(result, AliasDevicePair{
			Alias:  alias,
			Device: &deviceCopy,
		})
	}

	return result
}

// AliasSet sets an alias for a device
func (c *WebSocketClient) AliasSet(alias *string, criteria FilterCriteria) error {
	if alias == nil {
		return fmt.Errorf("alias cannot be nil")
	}

	// Get devices matching the criteria
	devices := c.GetDevices(criteria.Device)
	if len(devices) == 0 {
		return fmt.Errorf("no devices match the criteria")
	}
	if len(devices) > 1 {
		return fmt.Errorf("multiple devices match the criteria")
	}

	// Create the payload
	payload := protocol.ManageAliasPayload{
		Action: protocol.AliasActionAdd,
		Alias:  *alias,
		Target: devices[0].String(),
	}

	// Send the message
	_, err := c.sendRequest(protocol.MessageTypeManageAlias, payload)
	return err
}

// AliasDelete deletes an alias
func (c *WebSocketClient) AliasDelete(alias *string) error {
	if alias == nil {
		return fmt.Errorf("alias cannot be nil")
	}

	// Create the payload
	payload := protocol.ManageAliasPayload{
		Action: protocol.AliasActionDelete,
		Alias:  *alias,
	}

	// Send the message
	_, err := c.sendRequest(protocol.MessageTypeManageAlias, payload)
	return err
}

// AliasGet gets the device for an alias
func (c *WebSocketClient) AliasGet(alias *string) (*IPAndEOJ, error) {
	if alias == nil {
		return nil, fmt.Errorf("alias cannot be nil")
	}

	c.aliasesMutex.RLock()
	defer c.aliasesMutex.RUnlock()

	if device, ok := c.aliases[*alias]; ok {
		return &device, nil
	}

	return nil, fmt.Errorf("alias not found: %s", *alias)
}

// GetAliases gets all aliases for a device
func (c *WebSocketClient) GetAliases(device IPAndEOJ) []string {
	c.aliasesMutex.RLock()
	defer c.aliasesMutex.RUnlock()

	var result []string
	for alias, d := range c.aliases {
		if d.IP.Equal(device.IP) && d.EOJ == device.EOJ {
			result = append(result, alias)
		}
	}

	return result
}

// GetDeviceByAlias gets a device by its alias
func (c *WebSocketClient) GetDeviceByAlias(alias string) (IPAndEOJ, bool) {
	c.aliasesMutex.RLock()
	defer c.aliasesMutex.RUnlock()

	device, ok := c.aliases[alias]
	return device, ok
}

// ValidateDeviceAlias validates an alias
func (c *WebSocketClient) ValidateDeviceAlias(alias string) error {
	return echonet_lite.ValidateDeviceAlias(alias)
}

// GetAllPropertyAliases gets all property aliases
func (c *WebSocketClient) GetAllPropertyAliases() []string {
	return echonet_lite.GetAllAliases()
}

// GetPropertyInfo gets information about a property
func (c *WebSocketClient) GetPropertyInfo(classCode EOJClassCode, e EPCType) (*PropertyInfo, bool) {
	return echonet_lite.GetPropertyInfo(classCode, e)
}

// IsPropertyDefaultEPC checks if a property is a default property
func (c *WebSocketClient) IsPropertyDefaultEPC(classCode EOJClassCode, epc EPCType) bool {
	return echonet_lite.IsPropertyDefaultEPC(classCode, epc)
}

// FindPropertyAlias finds a property by its alias
func (c *WebSocketClient) FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool) {
	return echonet_lite.FindPropertyAlias(classCode, alias)
}

// AvailablePropertyAliases gets all available property aliases for a class
func (c *WebSocketClient) AvailablePropertyAliases(classCode EOJClassCode) map[string]string {
	return echonet_lite.AvailablePropertyAliases(classCode)
}

// listenForMessages listens for messages from the WebSocket server
func (c *WebSocketClient) listenForMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// Read a message
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if c.debug {
					fmt.Printf("Error reading message: %v\n", err)
				}
				return
			}

			// Parse the message
			msg, err := protocol.ParseMessage(message)
			if err != nil {
				if c.debug {
					fmt.Printf("Error parsing message: %v\n", err)
				}
				continue
			}

			// Handle the message
			if msg.RequestID != "" {
				// This is a response to a request
				c.responseChMutex.Lock()
				if ch, ok := c.responseCh[msg.RequestID]; ok {
					ch <- msg
					delete(c.responseCh, msg.RequestID)
				}
				c.responseChMutex.Unlock()
			} else {
				// This is a notification
				c.handleNotification(msg)
			}
		}
	}
}

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
	c.devices = make(map[string]echonet_lite.DeviceAndProperties)
	for deviceID, device := range payload.Devices {
		// Parse the device identifier
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(device.IP + " " + device.EOJ)
		if err != nil {
			if c.debug {
				fmt.Printf("Error parsing device identifier: %v\n", err)
			}
			continue
		}

		// Convert properties to Properties slice
		var props echonet_lite.Properties
		for epcStr, edtStr := range device.Properties {
			epc, err := echonet_lite.ParseEPCString(epcStr)
			if err != nil {
				if c.debug {
					fmt.Printf("Error parsing EPC: %v\n", err)
				}
				continue
			}

			edt, err := base64.StdEncoding.DecodeString(edtStr)
			if err != nil {
				if c.debug {
					fmt.Printf("Error decoding EDT: %v\n", err)
				}
				continue
			}

			props = append(props, echonet_lite.Property{
				EPC: epc,
				EDT: edt,
			})
		}

		// Add to devices
		c.devices[deviceID] = echonet_lite.DeviceAndProperties{
			Device:     ipAndEOJ,
			Properties: props,
		}
	}
	c.devicesMutex.Unlock()

	// Update aliases
	c.aliasesMutex.Lock()
	c.aliases = make(map[string]echonet_lite.IPAndEOJ)
	for alias, deviceID := range payload.Aliases {
		// Parse the device identifier
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(deviceID)
		if err != nil {
			if c.debug {
				fmt.Printf("Error parsing device identifier: %v\n", err)
			}
			continue
		}

		// Add to aliases
		c.aliases[alias] = ipAndEOJ
	}
	c.aliasesMutex.Unlock()
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

	// Parse the device identifier
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.Device.IP + " " + payload.Device.EOJ)
	if err != nil {
		if c.debug {
			fmt.Printf("Error parsing device identifier: %v\n", err)
		}
		return
	}

	// Convert properties to Properties slice
	var props echonet_lite.Properties
	for epcStr, edtStr := range payload.Device.Properties {
		epc, err := echonet_lite.ParseEPCString(epcStr)
		if err != nil {
			if c.debug {
				fmt.Printf("Error parsing EPC: %v\n", err)
			}
			continue
		}

		edt, err := base64.StdEncoding.DecodeString(edtStr)
		if err != nil {
			if c.debug {
				fmt.Printf("Error decoding EDT: %v\n", err)
			}
			continue
		}

		props = append(props, echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		})
	}

	// Add to devices
	c.devicesMutex.Lock()
	c.devices[ipAndEOJ.String()] = echonet_lite.DeviceAndProperties{
		Device:     ipAndEOJ,
		Properties: props,
	}
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

	// Parse the device identifier
	ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.Device.IP + " " + payload.Device.EOJ)
	if err != nil {
		if c.debug {
			fmt.Printf("Error parsing device identifier: %v\n", err)
		}
		return
	}

	// Convert properties to Properties slice
	var props echonet_lite.Properties
	for epcStr, edtStr := range payload.Device.Properties {
		epc, err := echonet_lite.ParseEPCString(epcStr)
		if err != nil {
			if c.debug {
				fmt.Printf("Error parsing EPC: %v\n", err)
			}
			continue
		}

		edt, err := base64.StdEncoding.DecodeString(edtStr)
		if err != nil {
			if c.debug {
				fmt.Printf("Error decoding EDT: %v\n", err)
			}
			continue
		}

		props = append(props, echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		})
	}

	// Update devices
	c.devicesMutex.Lock()
	c.devices[ipAndEOJ.String()] = echonet_lite.DeviceAndProperties{
		Device:     ipAndEOJ,
		Properties: props,
	}
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
	delete(c.devices, ipAndEOJ.String())
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
		// Parse the device identifier
		ipAndEOJ, err := echonet_lite.ParseDeviceIdentifier(payload.Target)
		if err != nil {
			if c.debug {
				fmt.Printf("Error parsing device identifier: %v\n", err)
			}
			return
		}

		// Add or update the alias
		c.aliases[payload.Alias] = ipAndEOJ

	case protocol.AliasChangeTypeDeleted:
		// Remove the alias
		delete(c.aliases, payload.Alias)
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
	if deviceProps, ok := c.devices[ipAndEOJ.String()]; ok {
		// Create a new property
		prop := echonet_lite.Property{
			EPC: epc,
			EDT: edt,
		}
		// Register the property
		deviceProps.Properties = append(deviceProps.Properties, prop)
		c.devices[ipAndEOJ.String()] = deviceProps
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

	if c.debug {
		fmt.Printf("Timeout notification: %s %s: %s\n", payload.IP, payload.EOJ, payload.Message)
	}
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

// sendRequest sends a request to the WebSocket server and waits for a response
func (c *WebSocketClient) sendRequest(msgType protocol.MessageType, payload interface{}) (*protocol.Message, error) {
	// Generate a request ID
	c.requestIDMutex.Lock()
	c.requestID++
	requestID := fmt.Sprintf("req-%d", c.requestID)
	c.requestIDMutex.Unlock()

	// Create a channel for the response
	responseCh := make(chan *protocol.Message, 1)
	c.responseChMutex.Lock()
	c.responseCh[requestID] = responseCh
	c.responseChMutex.Unlock()

	// Create the message
	data, err := protocol.CreateMessage(msgType, payload, requestID)
	if err != nil {
		return nil, fmt.Errorf("error creating message: %v", err)
	}

	// Send the message
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return nil, fmt.Errorf("error sending message: %v", err)
	}

	// Wait for the response
	select {
	case response := <-responseCh:
		return response, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	case <-c.ctx.Done():
		return nil, fmt.Errorf("context canceled")
	}
}
