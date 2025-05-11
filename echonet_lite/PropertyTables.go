package echonet_lite

import (
	"fmt"
	"slices"
)

var PropertyTables = BuildPropertyTableMap()

func (pt PropertyTableMap) FindAlias(classCode EOJClassCode, alias string) (Property, bool) {
	if classCode != NodeProfile_ClassCode {
		if prop, ok := ProfileSuperClass_PropertyTable.FindAlias(alias); ok {
			return prop, true
		}
	}
	if table, ok := pt[classCode]; ok {
		if prop, ok := table.FindAlias(alias); ok {
			return prop, true
		}
	}
	return Property{}, false
}

func (pt PropertyTableMap) AvailableAliases(classCode EOJClassCode) map[string]PropertyDescription {
	if classCode == 0 {
		// classCodeがゼロ値の場合、共通プロパティのみを返す
		return ProfileSuperClass_PropertyTable.AvailableAliases()
	} else {
		// classCodeが指定されている場合、デバイス固有プロパティのみを返す
		if table, ok := pt[classCode]; ok {
			return table.AvailableAliases()
		}
		// Note: ここでは共通プロパティは含めない
	}

	return map[string]PropertyDescription{}
}

func GetAllAliases() map[string]PropertyDescription {
	aliases := map[string]PropertyDescription{}
	set := func(available map[string]PropertyDescription) {
		for alias, desc := range available {
			aliases[alias] = desc
		}
	}
	for _, table := range PropertyTables {
		set(table.AvailableAliases())
	}
	set(ProfileSuperClass_PropertyTable.AvailableAliases())
	return aliases
}

func GetPropertyDesc(c EOJClassCode, e EPCType) (*PropertyDesc, bool) {
	if table, ok := PropertyTables[c]; ok {
		if ps, ok := table.EPCDesc[e]; ok {
			return &ps, true
		}
	}
	if c != NodeProfile_ClassCode {
		if ps, ok := ProfileSuperClass_PropertyTable.EPCDesc[e]; ok {
			return &ps, true
		}
	}
	return nil, false
}

func PropertyFromInt(c EOJClassCode, epc EPCType, value int) (*Property, error) {
	info, ok := PropertyTables[c].EPCDesc[epc]
	if !ok || info.Decoder == nil {
		return nil, fmt.Errorf("not found Decoder for EPC %s", epc)
	}
	numberConverter, ok := info.Decoder.(PropertyIntConverter)
	if !ok {
		return nil, fmt.Errorf("not found PropertyIntConverter for EPC %s", epc)
	}
	edt, ok := numberConverter.FromInt(value)
	if !ok {
		return nil, fmt.Errorf("failed to convert %d to EDT for EPC %s", value, epc)
	}
	return &Property{
		EPC: epc,
		EDT: edt,
	}, nil
}

func IsPropertyDefaultEPC(c EOJClassCode, epc EPCType) bool {
	if table, ok := PropertyTables[c]; ok {
		if slices.Contains(table.DefaultEPCs, epc) {
			return true
		}
	}
	if c != NodeProfile_ClassCode {
		table := ProfileSuperClass_PropertyTable
		return slices.Contains(table.DefaultEPCs, epc)
	}
	return false
}
