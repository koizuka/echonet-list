package protocol

import (
	"echonet-list/echonet_lite"
)

type IPAndEOJ = echonet_lite.IPAndEOJ
type EOJClassCode = echonet_lite.EOJClassCode
type EOJ = echonet_lite.EOJ

type FilterCriteria = echonet_lite.FilterCriteria
type AliasDevicePair = echonet_lite.AliasDevicePair
type DeviceSpecifier = echonet_lite.DeviceSpecifier
type DevicePropertyData = echonet_lite.DevicePropertyData
type EPCType = echonet_lite.EPCType
type Properties = echonet_lite.Properties
type DeviceAndProperties = echonet_lite.DeviceAndProperties

type ECHONETLiteHandler = echonet_lite.ECHONETLiteHandler

func GetAllPropertyAliases() []string {
	return echonet_lite.GetAllAliases()
}

type DeviceAliasManager interface {
	GetDeviceByAlias(alias string) (IPAndEOJ, bool)
}
