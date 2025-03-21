package client

import (
	"echonet-list/protocol"
)

type IPAndEOJ = protocol.IPAndEOJ
type EOJClassCode = protocol.EOJClassCode
type EOJInstanceCode = protocol.EOJInstanceCode
type EOJ = protocol.EOJ

func MakeEOJ(class EOJClassCode, instance EOJInstanceCode) EOJ {
	return protocol.MakeEOJ(class, instance)
}

type DeviceAliasManager = protocol.DeviceAliasManager

type FilterCriteria = protocol.FilterCriteria
type AliasDevicePair = protocol.AliasDevicePair
type DeviceSpecifier = protocol.DeviceSpecifier
type EPCType = protocol.EPCType
type Property = protocol.Property
type Properties = protocol.Properties
type DeviceAndProperties = protocol.DeviceAndProperties

type ECHONETListClient interface {
	Close() error
	IsDebug() bool
	SetDebug(debug bool)

	GetDeviceAliasManager() protocol.DeviceAliasManager
	AliasList() []AliasDevicePair
	AliasSet(alias *string, criteria FilterCriteria) error
	AliasDelete(alias *string) error
	AliasGet(alias *string) (*IPAndEOJ, error)
	GetAliases(device IPAndEOJ) []string

	Discover() error
	UpdateProperties(criteria FilterCriteria) error
	GetDevices(deviceSpec DeviceSpecifier) []IPAndEOJ
	ListDevices(criteria FilterCriteria) []DeviceAndProperties
	GetProperties(device IPAndEOJ, EPCs []EPCType, skipValidation bool) (DeviceAndProperties, error)
	SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error)
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

func (c *ECHONETListClientProxy) UpdateProperties(criteria FilterCriteria) error {
	return c.handler.UpdateProperties(criteria)
}

func (c *ECHONETListClientProxy) AliasList() []AliasDevicePair {
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
	return c.handler.DeviceAliases.GetAliases(device)
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
	return protocol.GetAllPropertyAliases()
}

func ValidateDeviceAlias(alias string) error {
	return protocol.ValidateDeviceAlias(alias)
}

type PropertyInfo = protocol.PropertyInfo

func GetPropertyInfo(c EOJClassCode, e EPCType) (*PropertyInfo, bool) {
	return protocol.GetPropertyInfo(c, e)
}

func IsPropertyDefaultEPC(c EOJClassCode, epc EPCType) bool {
	return protocol.IsPropertyDefaultEPC(c, epc)
}

func FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool) {
	return protocol.FindPropertyAlias(classCode, alias)
}

func AvailablePropertyAliases(classCode EOJClassCode) map[string]string {
	return protocol.AvailablePropertyAliases(classCode)
}
