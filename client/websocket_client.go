package client

import (
	"bytes"
	"context"
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient implements the ECHONETListClient interface using WebSocket
type WebSocketClient struct {
	ctx             context.Context
	cancel          context.CancelFunc
	transport       WebSocketClientTransport
	debug           bool
	devices         map[string]echonet_lite.DeviceAndProperties
	devicesMutex    sync.RWMutex
	aliases         map[string]IDString
	aliasesMutex    sync.RWMutex
	groups          []GroupDevicePair
	groupsMutex     sync.RWMutex
	requestID       int
	requestIDMutex  sync.Mutex
	responseCh      map[string]chan *protocol.Message
	responseChMutex sync.Mutex
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(ctx context.Context, serverURL string, debug bool) (*WebSocketClient, error) {
	clientCtx, cancel := context.WithCancel(ctx)

	// Create the transport
	transport, err := NewDefaultWebSocketClientTransport(clientCtx, serverURL, debug)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	client := &WebSocketClient{
		ctx:        clientCtx,
		cancel:     cancel,
		transport:  transport,
		debug:      debug,
		devices:    make(map[string]echonet_lite.DeviceAndProperties),
		aliases:    make(map[string]IDString),
		groups:     make([]GroupDevicePair, 0),
		responseCh: make(map[string]chan *protocol.Message),
	}

	return client, nil
}

// Connect connects to the WebSocket server
func (c *WebSocketClient) Connect() error {
	// Connect to the WebSocket server using the transport
	if err := c.transport.Connect(); err != nil {
		return fmt.Errorf("error connecting to WebSocket server: %v", err)
	}

	// Start listening for messages
	go c.listenForMessages()

	return nil
}

// Close closes the WebSocket connection
func (c *WebSocketClient) Close() error {
	c.cancel()
	return c.transport.Close()
}

// IsDebug returns whether debug mode is enabled
func (c *WebSocketClient) IsDebug() bool {
	return c.debug
}

// SetDebug sets the debug mode
func (c *WebSocketClient) SetDebug(debug bool) {
	c.debug = debug

	// トランスポートがDefaultWebSocketClientTransportの場合、そのデバッグモードも設定
	if t, ok := c.transport.(*DefaultWebSocketClientTransport); ok {
		t.SetDebug(debug)
	}
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

func (c *WebSocketClient) FindDeviceByIDString(id IDString) *IPAndEOJ {
	c.devicesMutex.RLock()
	defer c.devicesMutex.RUnlock()

	// device の EOJ と properties の IdentificationNumber をもとに IDStringを組み立て、一致する物を探す
	for _, device := range c.devices {
		eoj := device.Device.EOJ
		for _, prop := range device.Properties {
			if prop.EPC == echonet_lite.EPCIdentificationNumber {
				decoded := echonet_lite.DecodeIdentificationNumber(prop.EDT)
				if decoded == nil {
					continue
				}
				// IDString を組み立てる
				idString := echonet_lite.MakeIDString(eoj, *decoded)
				// IDString が一致するか確認
				if idString == id {
					return &device.Device
				}
			}
		}
	}
	return nil
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
			for _, prop := range criteria.PropertyValues {
				found, ok := deviceAndProps.Properties.FindEPC(prop.EPC)
				// Check if the property exists
				if !ok {
					match = false
					break
				}

				// Check if the property value matches
				if !bytes.Equal(found.EDT, prop.EDT) {
					match = false
					break
				}
			}
		}

		if match {
			result = append(result, deviceAndProps)
		}
	}

	// IPアドレスとEOJでソート
	sort.Slice(result, func(i, j int) bool {
		// IPアドレスでソート
		c := bytes.Compare(result[i].Device.IP, result[j].Device.IP)
		if c != 0 {
			// IPアドレスをバイト値として比較 (IPv4/IPv6両対応)
			return c < 0
		}
		// IPアドレスが同じ場合はEOJでソート
		return result[i].Device.EOJ < result[j].Device.EOJ
	})

	return result
}

// AliasList returns a list of all aliases
func (c *WebSocketClient) AliasList() []AliasIDStringPair {
	c.aliasesMutex.RLock()
	defer c.aliasesMutex.RUnlock()

	var result []AliasIDStringPair
	for alias, id := range c.aliases {
		// Create a copy of the device to avoid reference issues
		// Create a new AliasIDStringPair with the alias and device pointer
		result = append(result, AliasIDStringPair{
			Alias: alias,
			ID:    id,
		})
	}

	return result
}

// AliasGet gets the device for an alias
func (c *WebSocketClient) AliasGet(alias *string) (*IPAndEOJ, error) {
	if alias == nil {
		return nil, fmt.Errorf("alias cannot be nil")
	}

	c.aliasesMutex.RLock()
	defer c.aliasesMutex.RUnlock()

	if id, ok := c.aliases[*alias]; ok {
		device := c.FindDeviceByIDString(id)
		if device == nil {
			return nil, fmt.Errorf("device not found for alias: %s", *alias)
		}
		return device, nil
	}

	return nil, fmt.Errorf("alias not found: %s", *alias)
}

// GetAliases gets all aliases for a device
func (c *WebSocketClient) GetAliases(device IPAndEOJ) []string {
	c.aliasesMutex.RLock()
	defer c.aliasesMutex.RUnlock()

	var result []string
	for alias, id := range c.aliases {
		if d := c.FindDeviceByIDString(id); d != nil {
			if d.IP.Equal(device.IP) && d.EOJ == device.EOJ {
				result = append(result, alias)
			}
		}
	}

	return result
}

// GetDeviceByAlias gets a device by its alias
func (c *WebSocketClient) GetDeviceByAlias(alias string) (IPAndEOJ, bool) {
	c.aliasesMutex.RLock()
	defer c.aliasesMutex.RUnlock()

	id, ok := c.aliases[alias]
	if !ok {
		return IPAndEOJ{}, false
	}
	device := c.FindDeviceByIDString(id)
	if device == nil {
		return IPAndEOJ{}, false
	}
	return *device, ok
}

// GroupManager インターフェースの実装

// GroupList returns a list of all groups
func (c *WebSocketClient) GroupList(groupName *string) []GroupDevicePair {
	// キャッシュされたグループ情報を使用
	c.groupsMutex.RLock()
	defer c.groupsMutex.RUnlock()

	// グループ名が指定されている場合は、そのグループのみを返す
	if groupName != nil {
		for _, group := range c.groups {
			if group.Group == *groupName {
				// コピーを作成して返す
				result := make([]GroupDevicePair, 1)
				result[0] = GroupDevicePair{
					Group:   group.Group,
					Devices: make([]IPAndEOJ, len(group.Devices)),
				}
				copy(result[0].Devices, group.Devices)
				return result
			}
		}
		// 指定されたグループが見つからない場合は空のスライスを返す
		return []GroupDevicePair{}
	}

	// グループ名が指定されていない場合は、全てのグループを返す
	result := make([]GroupDevicePair, len(c.groups))
	for i, group := range c.groups {
		result[i] = GroupDevicePair{
			Group:   group.Group,
			Devices: make([]IPAndEOJ, len(group.Devices)),
		}
		copy(result[i].Devices, group.Devices)
	}
	return result
}

// GetDevicesByGroup gets devices in a group
func (c *WebSocketClient) GetDevicesByGroup(groupName string) ([]IPAndEOJ, bool) {
	// Validate the group name
	if err := echonet_lite.ValidateGroupName(groupName); err != nil {
		return nil, false
	}

	// Get the group list
	groups := c.GroupList(&groupName)
	if len(groups) == 0 {
		return nil, false
	}

	// Return the devices
	return groups[0].Devices, true
}

// listenForMessages listens for messages from the WebSocket server
func (c *WebSocketClient) listenForMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// Read a message using the transport
			_, message, err := c.transport.ReadMessage()
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

	// Send the message using the transport
	if err := c.transport.WriteMessage(websocket.TextMessage, data); err != nil {
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
