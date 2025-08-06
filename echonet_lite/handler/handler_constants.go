package handler

import (
	"time"
)

const (
	DeviceFileName        = "devices.json"
	DeviceAliasesFileName = "aliases.json"
	DeviceGroupsFileName  = "groups.json"

	UpdateIntervalThreshold = 5 * time.Second  // プロパティ更新をスキップする閾値
	MaxUpdateAge            = 10 * time.Minute // IP更新の最大有効期間
)

// NotificationType は通知の種類を表す型
type NotificationType int

const (
	DeviceAdded NotificationType = iota
	DeviceRemoved
	DeviceTimeout
	DeviceOffline
	DeviceOnline
)

// DeviceNotification はデバイスに関する通知を表す構造体
type DeviceNotification struct {
	Device IPAndEOJ
	Type   NotificationType
	Error  error // タイムアウトの場合はエラー情報
}

// PropertyChangeNotification はプロパティ変化に関する通知を表す構造体
type PropertyChangeNotification struct {
	Device   IPAndEOJ
	Property Property
}

// DeviceAndProperties は、デバイスとそのプロパティの組を表す構造体
type DeviceAndProperties struct {
	Device     IPAndEOJ
	Properties Properties
}
