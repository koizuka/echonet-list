package client

type Debugger interface {
	IsDebug() bool
	SetDebug(debug bool)
}

type AliasManager interface {
	AliasList() []AliasDevicePair
	AliasSet(alias *string, criteria FilterCriteria) error
	AliasDelete(alias *string) error
	AliasGet(alias *string) (*IPAndEOJ, error)
	GetAliases(device IPAndEOJ) []string
	GetDeviceByAlias(alias string) (IPAndEOJ, bool)
}

type DeviceManager interface {
	Discover() error
	UpdateProperties(criteria FilterCriteria) error
	GetDevices(deviceSpec DeviceSpecifier) []IPAndEOJ
	ListDevices(criteria FilterCriteria) []DeviceAndProperties
	GetProperties(device IPAndEOJ, EPCs []EPCType, skipValidation bool) (DeviceAndProperties, error)
	SetProperties(device IPAndEOJ, properties Properties) (DeviceAndProperties, error)
}

type PropertyInfoProvider interface {
	GetAllPropertyAliases() []string
	GetPropertyInfo(classCode EOJClassCode, e EPCType) (*PropertyInfo, bool)
	IsPropertyDefaultEPC(classCode EOJClassCode, epc EPCType) bool
	FindPropertyAlias(classCode EOJClassCode, alias string) (Property, bool)
	AvailablePropertyAliases(classCode EOJClassCode) map[string]string
}

type GroupManager interface {
	GroupList(groupName *string) []GroupDevicePair
	GroupAdd(groupName string, devices []IPAndEOJ) error
	GroupRemove(groupName string, devices []IPAndEOJ) error
	GroupDelete(groupName string) error
	GetDevicesByGroup(groupName string) ([]IPAndEOJ, bool)
}
