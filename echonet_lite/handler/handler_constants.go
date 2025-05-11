package handler

import (
	"echonet-list/echonet_lite"
	"time"
)

const (
	DeviceFileName        = "devices.json"
	DeviceAliasesFileName = "aliases.json"
	DeviceGroupsFileName  = "groups.json"

	UpdateIntervalThreshold = 5 * time.Second // プロパティ更新をスキップする閾値
)

// NotificationType は通知の種類を表す型
type NotificationType int

const (
	DeviceAdded NotificationType = iota
	DeviceTimeout
	DeviceOffline
)

// DeviceNotification はデバイスに関する通知を表す構造体
type DeviceNotification struct {
	Device echonet_lite.IPAndEOJ
	Type   NotificationType
	Error  error // タイムアウトの場合はエラー情報
}

// PropertyChangeNotification はプロパティ変化に関する通知を表す構造体
type PropertyChangeNotification struct {
	Device   echonet_lite.IPAndEOJ
	Property echonet_lite.Property
}

// DeviceAndProperties は、デバイスとそのプロパティの組を表す構造体
type DeviceAndProperties struct {
	Device     echonet_lite.IPAndEOJ
	Properties echonet_lite.Properties
}
