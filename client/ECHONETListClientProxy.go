package client

import (
	"echonet-list/echonet_lite"
)

// ECHONETListClientProxy は、ECHONETListClientのlocal proxy
type ECHONETListClientProxy struct {
	handler *echonet_lite.ECHONETLiteHandler // handlerを持つのは移行のため
}

func NewECHONETListClientProxy(handler *echonet_lite.ECHONETLiteHandler) ECHONETListClient {
	return &ECHONETListClientProxy{
		handler: handler,
	}
}

func (c *ECHONETListClientProxy) Close() error {
	return nil
}

func (c *ECHONETListClientProxy) Discover() error {
	return c.handler.Discover()
}

func (c *ECHONETListClientProxy) GetDeviceByAlias(alias string) (IPAndEOJ, bool) {
	device, err := c.handler.AliasGet(&alias)
	if err != nil {
		return IPAndEOJ{}, false
	}
	return *device, true
}

func (c *ECHONETListClientProxy) IsDebug() bool {
	return c.handler.IsDebug()
}

func (c *ECHONETListClientProxy) SetDebug(debug bool) {
	c.handler.SetDebug(debug)
}

func (c *ECHONETListClientProxy) UpdateProperties(criteria FilterCriteria) error {
	return c.handler.UpdateProperties(criteria)
}

func (c *ECHONETListClientProxy) AliasList() []AliasIDStringPair {
	return c.handler.AliasList()
}

func (c *ECHONETListClientProxy) AliasSet(alias *string, criteria FilterCriteria) error {
	return c.handler.AliasSet(alias, criteria)
}

func (c *ECHONETListClientProxy) AliasDelete(alias *string) error {
	return c.handler.AliasDelete(alias)
}

func (c *ECHONETListClientProxy) AliasGet(alias *string) (*IPAndEOJ, error) {
	return c.handler.AliasGet(alias)
}

func (c *ECHONETListClientProxy) GetAliases(device IPAndEOJ) []string {
	return c.handler.GetAliases(device)
}

func (c *ECHONETListClientProxy) GetDevices(deviceSpec DeviceSpecifier) []IPAndEOJ {
	return c.handler.GetDevices(deviceSpec)
}

func (c *ECHONETListClientProxy) ListDevices(criteria FilterCriteria) []DeviceAndProperties {
	return c.handler.ListDevices(criteria)
}

func (c *ECHONETListClientProxy) GetProperties(device IPAndEOJ, EPCs []EPCType, skipValidation bool) (DeviceAndProperties, error) {
	return c.handler.GetProperties(device, EPCs, skipValidation)
}

func (c *ECHONETListClientProxy) SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error) {
	return c.handler.SetProperties(device, properties)
}

func (c *ECHONETListClientProxy) GetAllPropertyAliases() []string {
	return echonet_lite.GetAllAliases()
}

func (c *ECHONETListClientProxy) GetPropertyInfo(classCode EOJClassCode, e EPCType) (*PropertyInfo, bool) {
	return echonet_lite.GetPropertyInfo(classCode, e)
}

func (c *ECHONETListClientProxy) IsPropertyDefaultEPC(classCode EOJClassCode, epc EPCType) bool {
	return echonet_lite.IsPropertyDefaultEPC(classCode, epc)
}

func (c *ECHONETListClientProxy) FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool) {
	return echonet_lite.PropertyTables.FindAlias(classCode, alias)
}

func (c *ECHONETListClientProxy) AvailablePropertyAliases(classCode EOJClassCode) map[string]string {
	return echonet_lite.PropertyTables.AvailableAliases(classCode)
}

// GroupManager インターフェースの実装

func (c *ECHONETListClientProxy) GroupList(groupName *string) []GroupDevicePair {
	return c.handler.DeviceGroups.GroupList(groupName)
}

func (c *ECHONETListClientProxy) GroupAdd(groupName string, devices []IPAndEOJ) error {
	err := c.handler.DeviceGroups.GroupAdd(groupName, devices)
	if err != nil {
		return err
	}
	return c.handler.SaveGroupFile()
}

func (c *ECHONETListClientProxy) GroupRemove(groupName string, devices []IPAndEOJ) error {
	err := c.handler.DeviceGroups.GroupRemove(groupName, devices)
	if err != nil {
		return err
	}
	return c.handler.SaveGroupFile()
}

func (c *ECHONETListClientProxy) GroupDelete(groupName string) error {
	err := c.handler.DeviceGroups.GroupDelete(groupName)
	if err != nil {
		return err
	}
	return c.handler.SaveGroupFile()
}

func (c *ECHONETListClientProxy) GetDevicesByGroup(groupName string) ([]IPAndEOJ, bool) {
	return c.handler.DeviceGroups.GetDevicesByGroup(groupName)
}

func (c *ECHONETListClientProxy) FindDeviceByIDString(id IDString) *IPAndEOJ {
	return c.handler.FindDeviceByIDString(id)
}
