package client

type Debugger interface {
	IsDebug() bool
	SetDebug(debug bool)
	DebugSetOffline(target string, offline bool) error
	IsOfflineDevice(device IPAndEOJ) bool
}

type AliasManager interface {
	AliasList() []AliasIDStringPair
	AliasSet(alias *string, criteria FilterCriteria) error
	AliasDelete(alias *string) error
	AliasGet(alias *string) (*IPAndEOJ, error)
	GetAliases(device IPAndEOJ) []string
	GetDeviceByAlias(alias string) (IPAndEOJ, bool)
}

type DeviceManager interface {
	Discover() error
	UpdateProperties(criteria FilterCriteria, force bool) error
	GetDevices(deviceSpec DeviceSpecifier) []IPAndEOJ
	ListDevices(criteria FilterCriteria) []DeviceAndProperties
	GetProperties(device IPAndEOJ, EPCs []EPCType, skipValidation bool) (DeviceAndProperties, error)
	SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error)
	GetDeviceHistory(device IPAndEOJ, opts DeviceHistoryOptions) ([]DeviceHistoryEntry, error)
	FindDeviceByIDString(id IDString) *IPAndEOJ
	GetIDString(device IPAndEOJ) IDString
}

type PropertyDescProvider interface {
	GetAllPropertyAliases() map[string]PropertyDescription
	GetPropertyDesc(classCode EOJClassCode, e EPCType) (*PropertyDesc, bool)
	IsPropertyDefaultEPC(classCode EOJClassCode, epc EPCType) bool
	FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool)
	AvailablePropertyAliases(classCode EOJClassCode) map[string]PropertyDescription
}

type GroupManager interface {
	GroupList(groupName *string) []GroupDevicePair
	GroupAdd(groupName string, devices []IDString) error
	GroupRemove(groupName string, devices []IDString) error
	GroupDelete(groupName string) error
	GetDevicesByGroup(groupName string) ([]IDString, bool)
}
