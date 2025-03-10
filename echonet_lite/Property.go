package echonet_lite

import "fmt"

// PropertyTable: 新しい機器クラスを定義するときは、ここに追加すること
type PropertyTableMap map[EOJClassCode]PropertyTable

var PropertyTables = PropertyTableMap{
	NodeProfile_ClassCode:            NPO_PropertyTable,
	SingleFunctionLighting_ClassCode: SF_PropertyTable,
	HomeAirConditioner_ClassCode:     HAC_PropertyTable,
	FloorHeating_ClassCode:           FH_PropertyTable,
}

func (pt PropertyTableMap) FindAlias(classCode EOJClassCode, alias string) (Property, bool) {
	if prop, ok := ProfileSuperClass_PropertyTable.FindAlias(alias); ok {
		return prop, true
	}
	if table, ok := pt[classCode]; ok {
		if prop, ok := table.FindAlias(alias); ok {
			return prop, true
		}
	}
	return Property{}, false
}

func (pt PropertyTableMap) AvailableAliases(classCode EOJClassCode) map[string]string {
	aliases := map[string]string{}
	if table, ok := pt[classCode]; ok {
		for alias, desc := range table.AvailableAliases() {
			aliases[alias] = desc
		}
	}
	for alias, desc := range ProfileSuperClass_PropertyTable.AvailableAliases() {
		aliases[alias] = desc
	}
	return aliases
}

func GetAllAliases() []string {
	aliases := []string{}
	for _, table := range PropertyTables {
		for alias := range table.AvailableAliases() {
			aliases = append(aliases, alias)
		}
	}
	for alias := range ProfileSuperClass_PropertyTable.AvailableAliases() {
		aliases = append(aliases, alias)
	}
	return aliases
}

// Property は各プロパティ（EPC, PDC, EDT）を表します。
type Property struct {
	EPC EPCType // プロパティコード
	EDT []byte  // プロパティデータ
}
type Properties []Property

func (p Property) Property() *Property {
	return &p
}

func (p Property) Encode() []byte {
	PDC := len(p.EDT)
	data := make([]byte, 2+PDC) // Create with full length to include EDT
	data[0] = byte(p.EPC)
	data[1] = byte(PDC)
	copy(data[2:], p.EDT)
	return data
}

type PropertyDecoderFunc func(EDT []byte) fmt.Stringer

type PropertyInfo struct {
	EPCs    string
	Decoder PropertyDecoderFunc
	Aliases map[string][]byte // Alias names for EDT values (e.g., "on" -> []byte{0x30})
}

func Decoder[T fmt.Stringer](f func(EDT []byte) T) PropertyDecoderFunc {
	return func(EDT []byte) fmt.Stringer {
		return f(EDT)
	}
}

type PropertyTable struct {
	EPCInfo     map[EPCType]PropertyInfo
	DefaultEPCs []EPCType
}

func (pt PropertyTable) FindAlias(alias string) (Property, bool) {
	for epc, info := range pt.EPCInfo {
		if aliases, ok := info.Aliases[alias]; ok {
			return Property{EPC: epc, EDT: aliases}, true
		}
	}
	return Property{}, false
}

func (pt PropertyTable) AvailableAliases() map[string]string {
	aliases := map[string]string{}
	for epc, info := range pt.EPCInfo {
		for alias := range info.Aliases {
			aliases[alias] = fmt.Sprintf("%s(%s):%X", epc, info.EPCs, info.Aliases[alias])
		}
	}
	return aliases
}

func (ps Properties) Encode() []byte {
	data := make([][]byte, len(ps)+1)
	data[0] = []byte{byte(len(ps))}
	for i, p := range ps {
		data[i+1] = p.Encode()
	}
	return flattenBytes(data)
}

type EPCType byte

func (e EPCType) PropertyForGet() *Property {
	return &Property{EPC: e}
}

func (e EPCType) String() string {
	return fmt.Sprintf("%02X", byte(e))
}

func (e EPCType) StringForClass(c EOJClassCode) string {
	if info, ok := GetPropertyInfo(c, e); ok {
		return fmt.Sprintf("%s(%s)", e.String(), info.EPCs)
	}
	return e.String()
}

func GetPropertyInfo(c EOJClassCode, e EPCType) (*PropertyInfo, bool) {
	if table, ok := PropertyTables[c]; ok {
		if ps, ok := table.EPCInfo[e]; ok {
			return &ps, true
		}
	}
	if ps, ok := ProfileSuperClass_PropertyTable.EPCInfo[e]; ok {
		return &ps, true
	}
	return nil, false
}

func GetEDTFromAlias(c EOJClassCode, e EPCType, alias string) ([]byte, bool) {
	if info, ok := GetPropertyInfo(c, e); ok && info.Aliases != nil {
		if aliases, ok := info.Aliases[alias]; ok {
			return aliases, true
		}
	}
	return nil, false
}

func IsPropertyDefaultEPC(c EOJClassCode, epc EPCType) bool {
	isDefaultEPC := func(table PropertyTable) bool {
		for _, e := range table.DefaultEPCs {
			if e == epc {
				return true
			}
		}
		return false
	}

	if table, ok := PropertyTables[c]; ok {
		if isDefaultEPC(table) {
			return true
		}
	}
	table := ProfileSuperClass_PropertyTable
	return isDefaultEPC(table)
}

func (p Property) String(c EOJClassCode) string {
	var EPC, EDT string

	if info, ok := GetPropertyInfo(c, p.EPC); ok {
		EPC = info.EPCs
		if info.Aliases != nil {
			for alias, value := range info.Aliases {
				if string(p.EDT) == string(value) {
					EDT = alias
					break
				}
			}
		}
		if EDT == "" {
			if info.Decoder != nil {
				EDT = info.Decoder(p.EDT).String()
			} else {
				EDT = fmt.Sprintf("%X", p.EDT)
			}
		}
	} else {
		EPC = fmt.Sprintf("? (ClassCode:%v)", c)
		EDT = fmt.Sprintf("%X", p.EDT)
	}

	return fmt.Sprintf("%s(%s): %s", p.EPC, EPC, EDT)
}

func (ps Properties) String(ClassCode EOJClassCode) string {
	var results []string
	for _, p := range ps {
		results = append(results, p.String(ClassCode))
	}
	return fmt.Sprintf("[%s]", results)
}

type IProperty interface {
	Property() *Property
}
type IPropertyForGet interface {
	PropertyForGet() *Property
}

func PropertiesForESVGet(p ...IPropertyForGet) []Property {
	props := make([]Property, 0, len(p))
	for _, prop := range p {
		props = append(props, *prop.PropertyForGet())
	}
	return props
}

func PropertiesForESVSet(p ...IProperty) []Property {
	props := make([]Property, 0, len(p))
	for _, prop := range p {
		props = append(props, *prop.Property())
	}
	return props
}
