package client

import (
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
func (c *WebSocketClient) UpdateProperties(criteria FilterCriteria, force bool) error {
	// criteriaに基づいてターゲットデバイスのリストを作成
	// criteriaが空の場合、targetsは空のまま送信される
	targets := make([]string, 0)
	// criteria.Device のいずれかのフィールドが設定されているか、PropertyValues が空でないかを確認
	if criteria.Device.IP != nil || criteria.Device.ClassCode != nil || criteria.Device.InstanceCode != nil || len(criteria.PropertyValues) > 0 {
		devices := c.ListDevices(criteria) // FilterCriteriaを直接使うListDevicesに変更
		if len(devices) == 0 {
			// criteriaが指定されているが見つからない場合はエラー
			return fmt.Errorf("no devices match the criteria: %v", criteria)
		}
		for _, dev := range devices {
			targets = append(targets, dev.Device.Specifier())
		}
	}
	// criteriaが空の場合は targets は空のまま

	payload := protocol.UpdatePropertiesPayload{
		Targets: targets, // criteriaが空なら空配列、そうでなければフィルタ結果
		Force:   force,
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
	ipAndEOJ, props, err := protocol.DeviceFromProtocol(deviceData)
	if err != nil {
		return DeviceAndProperties{}, fmt.Errorf("error converting device: %v", err)
	}

	return DeviceAndProperties{
		Device:     ipAndEOJ,
		Properties: props,
	}, nil
}

// SetProperties sets properties on a device
func (c *WebSocketClient) SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error) {
	// Create the payload
	propsMap := make(protocol.PropertyMap)
	for _, prop := range properties {
		propsMap.Set(prop.EPC, protocol.PropertyData{
			EDT:    base64.StdEncoding.EncodeToString(prop.EDT),
			String: "",
		})
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
	ipAndEOJ, props, err := protocol.DeviceFromProtocol(deviceData)
	if err != nil {
		return DeviceAndProperties{}, fmt.Errorf("error converting device: %v", err)
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
	if err := handler.ValidateGroupName(groupName); err != nil {
		return err
	}

	// Create the payload
	payload := protocol.ManageGroupPayload{
		Action:  protocol.GroupActionAdd,
		Group:   groupName,
		Devices: devices,
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
	if err := handler.ValidateGroupName(groupName); err != nil {
		return err
	}

	// Create the payload
	payload := protocol.ManageGroupPayload{
		Action:  protocol.GroupActionRemove,
		Group:   groupName,
		Devices: devices,
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
	if err := handler.ValidateGroupName(groupName); err != nil {
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
