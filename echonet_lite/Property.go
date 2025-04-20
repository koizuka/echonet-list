package echonet_lite

import (
	"echonet-list/echonet_lite/utils"
	"fmt"
	"slices"
)

var PropertyTables = BuildPropertyTableMap()

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

	if classCode == 0 {
		// classCodeがゼロ値の場合、共通プロパティのみを返す
		for alias, desc := range ProfileSuperClass_PropertyTable.AvailableAliases() {
			aliases[alias] = desc
		}
	} else {
		// classCodeが指定されている場合、デバイス固有プロパティのみを返す
		if table, ok := pt[classCode]; ok {
			for alias, desc := range table.AvailableAliases() {
				aliases[alias] = desc
			}
		}
		// Note: ここでは共通プロパティは含めない
	}

	return aliases
}

func GetAllAliases() []string {
	exists := map[string]bool{}
	aliases := []string{}
	set := func(available map[string]string) {
		for alias := range available {
			if !exists[alias] {
				aliases = append(aliases, alias)
				exists[alias] = true
			}
		}
	}
	for _, table := range PropertyTables {
		set(table.AvailableAliases())
	}
	set(ProfileSuperClass_PropertyTable.AvailableAliases())
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

type PropertyDecoderFunc func(EDT []byte) (fmt.Stringer, bool)

type PropertyInfo struct {
	EPCs    string
	Decoder PropertyDecoderFunc
	Aliases map[string][]byte // Alias names for EDT values (e.g., "on" -> []byte{0x30})
}

func Decoder[T fmt.Stringer](f func(EDT []byte) T) PropertyDecoderFunc {
	return func(EDT []byte) (fmt.Stringer, bool) {
		if len(EDT) == 0 {
			return nil, false
		}
		result := f(EDT)
		// if T is a pointer, nil check
		if _, ok := any(result).(fmt.Stringer); !ok {
			return nil, false
		}
		return result, true
	}
}

type PropertyTable struct {
	Description string
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
	return utils.FlattenBytes(data)
}

func (ps Properties) GetIdentificationNumber() *IdentificationNumber {
	if p, ok := ps.FindEPC(EPCIdentificationNumber); ok {
		return DecodeIdentificationNumber(p.EDT)
	}
	return nil
}

// EPCType はプロパティコードを表します。
// プロパティコードは、Echonet Lite のプロパティを識別するための 1 バイトの値です。
type EPCType byte

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

func IsPropertyDefaultEPC(c EOJClassCode, epc EPCType) bool {
	if table, ok := PropertyTables[c]; ok {
		if slices.Contains(table.DefaultEPCs, epc) {
			return true
		}
	}
	table := ProfileSuperClass_PropertyTable
	return slices.Contains(table.DefaultEPCs, epc)
}

func (p Property) EPCString(c EOJClassCode) string {
	var EPC string
	if info, ok := GetPropertyInfo(c, p.EPC); ok {
		EPC = info.EPCs
	} else {
		EPC = fmt.Sprintf("? (ClassCode:%v)", c)
	}
	return EPC
}

func (p Property) EDTString(c EOJClassCode) string {
	if p.EDT == nil {
		return "nil"
	}
	var EDT string

	if info, ok := GetPropertyInfo(c, p.EPC); ok {
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
				if decoded, ok := info.Decoder(p.EDT); ok {
					EDT = decoded.String()
				} else {
					EDT = fmt.Sprintf("%X", p.EDT)
				}
			} else {
				EDT = fmt.Sprintf("%X", p.EDT)
			}
		}
	} else {
		EDT = fmt.Sprintf("%X", p.EDT)
	}

	return EDT
}

func (p Property) String(c EOJClassCode) string {
	EPC := p.EPCString(c)
	return fmt.Sprintf("%s(%s): %s", p.EPC, EPC, p.EDTString(c))
}

func (ps Properties) String(ClassCode EOJClassCode) string {
	var results []string
	for _, p := range ps {
		results = append(results, p.String(ClassCode))
	}
	return fmt.Sprintf("[%s]", results)
}

func (ps Properties) FindEPC(epc EPCType) (Property, bool) {
	for _, p := range ps {
		if p.EPC == epc {
			return p, true
		}
	}
	return Property{}, false
}

// UpdateProperty は指定されたEPCのプロパティを更新または追加します。
// 既存のプロパティが見つかった場合は更新し、見つからなかった場合は追加します。
// 更新または追加されたプロパティを含む新しいPropertiesを返します。
func (ps Properties) UpdateProperty(prop Property) Properties {
	// 既存のプロパティを探す
	for i, p := range ps {
		if p.EPC == prop.EPC {
			// 既存のプロパティを更新
			result := make(Properties, len(ps))
			copy(result, ps)
			result[i] = prop
			return result
		}
	}

	// 既存のプロパティが見つからなかった場合は追加
	return append(ps, prop)
}

type IProperty interface {
	Property() *Property
}
