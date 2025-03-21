package client

import (
	"echonet-list/echonet_lite"
)

type IPAndEOJ = echonet_lite.IPAndEOJ
type EOJClassCode = echonet_lite.EOJClassCode
type EOJInstanceCode = echonet_lite.EOJInstanceCode
type EOJ = echonet_lite.EOJ

func MakeEOJ(class EOJClassCode, instance EOJInstanceCode) EOJ {
	return echonet_lite.MakeEOJ(class, instance)
}

type FilterCriteria = echonet_lite.FilterCriteria
type AliasDevicePair = echonet_lite.AliasDevicePair
type DeviceSpecifier = echonet_lite.DeviceSpecifier
type EPCType = echonet_lite.EPCType
type Property = echonet_lite.Property
type Properties = echonet_lite.Properties
type DeviceAndProperties = echonet_lite.DeviceAndProperties

type PropertyInfo = echonet_lite.PropertyInfo

type ECHONETListClient interface {
	Close() error
	IsDebug() bool
	SetDebug(debug bool)

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

	GetDeviceByAlias(alias string) (IPAndEOJ, bool)
	ValidateDeviceAlias(alias string) error

	GetAllPropertyAliases() []string
	GetPropertyInfo(classCode EOJClassCode, e EPCType) (*PropertyInfo, bool)
	IsPropertyDefaultEPC(classCode EOJClassCode, epc EPCType) bool
	FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool)
	AvailablePropertyAliases(classCode EOJClassCode) map[string]string
}
