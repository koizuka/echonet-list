package client

import (
	"echonet-list/echonet_lite"
	"echonet-list/echonet_lite/handler"
	"echonet-list/protocol"
	"time"
)

type IPAndEOJ = echonet_lite.IPAndEOJ
type EOJClassCode = echonet_lite.EOJClassCode
type EOJInstanceCode = echonet_lite.EOJInstanceCode
type EOJ = echonet_lite.EOJ
type IDString = handler.IDString

func MakeEOJ(class EOJClassCode, instance EOJInstanceCode) EOJ {
	return echonet_lite.MakeEOJ(class, instance)
}

type FilterCriteria = handler.FilterCriteria
type AliasIDStringPair = handler.AliasIDStringPair
type GroupDevicePair = handler.GroupDevicePair
type DeviceSpecifier = handler.DeviceSpecifier
type EPCType = echonet_lite.EPCType
type Property = echonet_lite.Property
type Properties = echonet_lite.Properties
type DeviceAndProperties = handler.DeviceAndProperties

type PropertyDesc = echonet_lite.PropertyDesc
type PropertyDescription = echonet_lite.PropertyDescription

type DeviceHistoryOptions struct {
	Limit        int
	SettableOnly *bool
}

type DeviceHistoryEntry struct {
	Timestamp time.Time
	EPC       EPCType
	Value     protocol.PropertyData
	Origin    protocol.HistoryOrigin
	Settable  bool // Calculated by server based on current Set Property Map
}

type ECHONETListClient interface {
	Debugger
	AliasManager
	DeviceManager
	PropertyDescProvider
	GroupManager
	Close() error
}
