package echonet_lite

import (
	"net"
	"time"
)

// DataAccessor は、データアクセス機能を提供するインターフェース
// CommunicationHandlerがDataManagementHandlerの機能を利用するために使用
type DataAccessor interface {
	// デバイス情報の保存
	SaveDeviceInfo()

	// デバイスの存在確認
	IsKnownDevice(device IPAndEOJ) bool

	// プロパティマップ関連
	HasEPCInPropertyMap(device IPAndEOJ, mapType PropertyMapType, epc EPCType) bool
	GetPropertyMap(device IPAndEOJ, mapType PropertyMapType) PropertyMap

	// プロパティ関連
	RegisterProperties(device IPAndEOJ, properties Properties) []EPCType
	GetProperty(device IPAndEOJ, epc EPCType) (*Property, bool)

	// デバイス情報
	GetIDString(device IPAndEOJ) IDString
	GetLastUpdateTime(device IPAndEOJ) time.Time
	DeviceStringWithAlias(device IPAndEOJ) string

	// フィルタリング
	Filter(criteria FilterCriteria) Devices
	RegisterDevice(device IPAndEOJ)
	HasIP(ip net.IP) bool
	FindByIDString(id IDString) []IPAndEOJ
}

// NotificationRelay は、通知イベントを中継する機能を提供するインターフェース
// 各ハンドラがHandlerCoreに通知を送るために使用
type NotificationRelay interface {
	// デバイスイベントの中継
	RelayDeviceEvent(event DeviceEvent)

	// セッションタイムアウトイベントの中継
	RelaySessionTimeoutEvent(event SessionTimeoutEvent)

	// プロパティ変更イベントの中継
	RelayPropertyChangeEvent(device IPAndEOJ, property Property)
}
