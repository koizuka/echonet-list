package handler

import (
	"echonet-list/echonet_lite"
	"fmt"
	"net"
	"time"
)

// DataAccessor は、データアクセス機能を提供するインターフェース
// CommunicationHandlerがDataManagementHandlerの機能を利用するために使用
type DataAccessor interface {
	// デバイス情報の保存
	SaveDeviceInfo()

	// デバイスの存在確認
	IsKnownDevice(device echonet_lite.IPAndEOJ) bool

	// プロパティマップ関連
	HasEPCInPropertyMap(device echonet_lite.IPAndEOJ, mapType echonet_lite.PropertyMapType, epc echonet_lite.EPCType) bool
	GetPropertyMap(device echonet_lite.IPAndEOJ, mapType echonet_lite.PropertyMapType) echonet_lite.PropertyMap

	// プロパティ関連
	RegisterProperties(device echonet_lite.IPAndEOJ, properties echonet_lite.Properties) []ChangedProperty
	GetProperty(device echonet_lite.IPAndEOJ, epc echonet_lite.EPCType) (*echonet_lite.Property, bool)

	// デバイス情報
	GetIDString(device echonet_lite.IPAndEOJ) echonet_lite.IDString
	GetLastUpdateTime(device echonet_lite.IPAndEOJ) time.Time
	DeviceStringWithAlias(device echonet_lite.IPAndEOJ) string
	IsOffline(device echonet_lite.IPAndEOJ) bool
	SetOffline(device echonet_lite.IPAndEOJ, offline bool)

	// フィルタリング
	Filter(criteria echonet_lite.FilterCriteria) echonet_lite.Devices
	RegisterDevice(device echonet_lite.IPAndEOJ)
	HasIP(ip net.IP) bool
	FindByIDString(id echonet_lite.IDString) []echonet_lite.IPAndEOJ
}

// NotificationRelay は、通知イベントを中継する機能を提供するインターフェース
// 各ハンドラがHandlerCoreに通知を送るために使用
type NotificationRelay interface {
	// デバイスイベントの中継
	RelayDeviceEvent(event echonet_lite.DeviceEvent)

	// セッションタイムアウトイベントの中継
	RelaySessionTimeoutEvent(event SessionTimeoutEvent)

	// プロパティ変更イベントの中継
	RelayPropertyChangeEvent(device echonet_lite.IPAndEOJ, property echonet_lite.Property)
}

type ChangedProperty struct {
	EPC       echonet_lite.EPCType
	beforeEDT []byte
	afterEDT  []byte
}

func (c ChangedProperty) Before() echonet_lite.Property {
	return echonet_lite.Property{
		EPC: c.EPC,
		EDT: c.beforeEDT,
	}
}

func (c ChangedProperty) After() echonet_lite.Property {
	return echonet_lite.Property{
		EPC: c.EPC,
		EDT: c.afterEDT,
	}
}

func (c ChangedProperty) StringForClass(classCode echonet_lite.EOJClassCode) string {
	class := c.EPC.StringForClass(classCode)
	before := c.Before().EDTString(classCode)
	after := c.After().EDTString(classCode)

	if before == "" {
		return fmt.Sprintf("%s:%v", class, after)
	}
	if c.afterEDT == nil {
		return fmt.Sprintf("%s:-%v", class, before)
	}
	return fmt.Sprintf("%s:%v->%v", class, before, after)
}
