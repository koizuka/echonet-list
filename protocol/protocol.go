package protocol

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

type ECHONETLiteHandler = echonet_lite.ECHONETLiteHandler

func GetAllPropertyAliases() []string {
	return echonet_lite.GetAllAliases()
}

type DeviceAliasManager interface {
	GetDeviceByAlias(alias string) (IPAndEOJ, bool)
}

func ValidateDeviceAlias(alias string) error {
	return echonet_lite.ValidateDeviceAlias(alias)
}

type PropertyInfo = echonet_lite.PropertyInfo

func GetPropertyInfo(c EOJClassCode, e EPCType) (*PropertyInfo, bool) {
	return echonet_lite.GetPropertyInfo(c, e)
}

func IsPropertyDefaultEPC(c EOJClassCode, epc EPCType) bool {
	return echonet_lite.IsPropertyDefaultEPC(c, epc)
}

func FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool) {
	return echonet_lite.PropertyTables.FindAlias(classCode, alias)
}

func AvailablePropertyAliases(classCode EOJClassCode) map[string]string {
	return echonet_lite.PropertyTables.AvailableAliases(classCode)
}
