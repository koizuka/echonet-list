package client

import (
	"echonet-list/protocol"
)

type ECHONETListClient interface {
	Close() error
	IsDebug() bool
	SetDebug(debug bool)

	GetDeviceAliasManager() protocol.DeviceAliasManager
	AliasList() []protocol.AliasDevicePair
	AliasSet(alias *string, criteria protocol.FilterCriteria) error
	AliasDelete(alias *string) error
	AliasGet(alias *string) (*protocol.IPAndEOJ, error)
	GetAliases(device protocol.IPAndEOJ) []string

	Discover() error
	UpdateProperties(criteria protocol.FilterCriteria) error
	GetDevices(deviceSpec protocol.DeviceSpecifier) []protocol.IPAndEOJ
	ListDevices(criteria protocol.FilterCriteria) []protocol.DevicePropertyData
	GetProperties(device protocol.IPAndEOJ, EPCs []protocol.EPCType, skipValidation bool) (protocol.DeviceAndProperties, error)
	SetProperties(device protocol.IPAndEOJ, properties protocol.Properties) (protocol.DeviceAndProperties, error)
	GetAllPropertyAliases() []string
}

// ECHONETListClientProxy は、ECHONETListClientのlocal proxy
type ECHONETListClientProxy struct {
	handler *protocol.ECHONETLiteHandler // handlerを持つのは移行のため
}

func NewECHONETListClientProxy(handler *protocol.ECHONETLiteHandler) ECHONETListClient {
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

func (c *ECHONETListClientProxy) GetDeviceAliasManager() protocol.DeviceAliasManager {
	return c.handler.DeviceAliases
}

func (c *ECHONETListClientProxy) IsDebug() bool {
	return c.handler.IsDebug()
}

func (c *ECHONETListClientProxy) SetDebug(debug bool) {
	c.handler.SetDebug(debug)
}

func (c *ECHONETListClientProxy) UpdateProperties(criteria protocol.FilterCriteria) error {
	return c.handler.UpdateProperties(criteria)
}

func (c *ECHONETListClientProxy) AliasList() []protocol.AliasDevicePair {
	return c.handler.AliasList()
}

func (c *ECHONETListClientProxy) AliasSet(alias *string, criteria protocol.FilterCriteria) error {
	return c.handler.AliasSet(alias, criteria)
}

func (c *ECHONETListClientProxy) AliasDelete(alias *string) error {
	return c.handler.AliasDelete(alias)
}

func (c *ECHONETListClientProxy) AliasGet(alias *string) (*protocol.IPAndEOJ, error) {
	return c.handler.AliasGet(alias)
}

func (c *ECHONETListClientProxy) GetAliases(device protocol.IPAndEOJ) []string {
	return c.handler.DeviceAliases.GetAliases(device)
}

func (c *ECHONETListClientProxy) GetDevices(deviceSpec protocol.DeviceSpecifier) []protocol.IPAndEOJ {
	return c.handler.GetDevices(deviceSpec)
}

func (c *ECHONETListClientProxy) ListDevices(criteria protocol.FilterCriteria) []protocol.DevicePropertyData {
	return c.handler.ListDevices(criteria)
}

func (c *ECHONETListClientProxy) GetProperties(device protocol.IPAndEOJ, EPCs []protocol.EPCType, skipValidation bool) (protocol.DeviceAndProperties, error) {
	return c.handler.GetProperties(device, EPCs, skipValidation)
}

func (c *ECHONETListClientProxy) SetProperties(device protocol.IPAndEOJ, properties protocol.Properties) (protocol.DeviceAndProperties, error) {
	return c.handler.SetProperties(device, properties)
}

func (c *ECHONETListClientProxy) GetAllPropertyAliases() []string {
	return protocol.GetAllPropertyAliases()
}
