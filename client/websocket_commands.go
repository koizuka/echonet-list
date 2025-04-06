package client

import (
	"echonet-list/echonet_lite"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
)

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

	// Convert protocol.Device to echonet_lite types using DeviceFromProtocol
	ip, eoj, props, err := protocol.DeviceFromProtocol(deviceData)
	if err != nil {
		return DeviceAndProperties{}, fmt.Errorf("error converting device: %v", err)
	}

	// Create IPAndEOJ
	ipAndEOJ := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP(ip),
		EOJ: eoj,
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

	// Convert protocol.Device to echonet_lite types using DeviceFromProtocol
	ip, eoj, props, err := protocol.DeviceFromProtocol(deviceData)
	if err != nil {
		return DeviceAndProperties{}, fmt.Errorf("error converting device: %v", err)
	}

	// Create IPAndEOJ
	ipAndEOJ := echonet_lite.IPAndEOJ{
		IP:  net.ParseIP(ip),
		EOJ: eoj,
	}

	return DeviceAndProperties{
		Device:     ipAndEOJ,
		Properties: props,
	}, nil
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

	ids := c.GetIDString(devices[0])
	if ids == "" {
		return fmt.Errorf("device ID is empty")
	}

	// Create the payload
	payload := protocol.ManageAliasPayload{
		Action: protocol.AliasActionAdd,
		Alias:  *alias,
		Target: ids,
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

// GroupAdd adds devices to a group
func (c *WebSocketClient) GroupAdd(groupName string, devices []IDString) error {
	// Validate the group name
	if err := echonet_lite.ValidateGroupName(groupName); err != nil {
		return err
	}

	// Convert devices to strings
	deviceStrs := make([]string, 0, len(devices))
	for _, ids := range devices {
		deviceStrs = append(deviceStrs, string(ids))
	}

	// Create the payload
	payload := protocol.ManageGroupPayload{
		Action:  protocol.GroupActionAdd,
		Group:   groupName,
		Devices: deviceStrs,
	}

	// Send the message
	response, err := c.sendRequest(protocol.MessageTypeManageGroup, payload)
	if err != nil {
		return fmt.Errorf("error adding devices to group: %v", err)
	}

	// Parse the response
	var resultPayload protocol.CommandResultPayload
	if err := protocol.ParsePayload(response, &resultPayload); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if !resultPayload.Success {
		if resultPayload.Error != nil {
			return fmt.Errorf("error adding devices to group: %s: %s", resultPayload.Error.Code, resultPayload.Error.Message)
		}
		return fmt.Errorf("error adding devices to group: unknown error")
	}

	return nil
}

// GroupRemove removes devices from a group
func (c *WebSocketClient) GroupRemove(groupName string, devices []IDString) error {
	// Validate the group name
	if err := echonet_lite.ValidateGroupName(groupName); err != nil {
		return err
	}

	// Convert devices to strings
	deviceStrs := make([]string, 0, len(devices))
	for _, ids := range devices {
		deviceStrs = append(deviceStrs, string(ids))
	}

	// Create the payload
	payload := protocol.ManageGroupPayload{
		Action:  protocol.GroupActionRemove,
		Group:   groupName,
		Devices: deviceStrs,
	}

	// Send the message
	response, err := c.sendRequest(protocol.MessageTypeManageGroup, payload)
	if err != nil {
		return fmt.Errorf("error removing devices from group: %v", err)
	}

	// Parse the response
	var resultPayload protocol.CommandResultPayload
	if err := protocol.ParsePayload(response, &resultPayload); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if !resultPayload.Success {
		if resultPayload.Error != nil {
			return fmt.Errorf("error removing devices from group: %s: %s", resultPayload.Error.Code, resultPayload.Error.Message)
		}
		return fmt.Errorf("error removing devices from group: unknown error")
	}

	return nil
}

// GroupDelete deletes a group
func (c *WebSocketClient) GroupDelete(groupName string) error {
	// Validate the group name
	if err := echonet_lite.ValidateGroupName(groupName); err != nil {
		return err
	}

	// Create the payload
	payload := protocol.ManageGroupPayload{
		Action: protocol.GroupActionDelete,
		Group:  groupName,
	}

	// Send the message
	response, err := c.sendRequest(protocol.MessageTypeManageGroup, payload)
	if err != nil {
		return fmt.Errorf("error deleting group: %v", err)
	}

	// Parse the response
	var resultPayload protocol.CommandResultPayload
	if err := protocol.ParsePayload(response, &resultPayload); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if !resultPayload.Success {
		if resultPayload.Error != nil {
			return fmt.Errorf("error deleting group: %s: %s", resultPayload.Error.Code, resultPayload.Error.Message)
		}
		return fmt.Errorf("error deleting group: unknown error")
	}

	return nil
}
