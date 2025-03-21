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
	Debugger
	AliasManager
	DeviceManager
	PropertyInfoProvider
	Close() error
}
