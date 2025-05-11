package echonet_lite

func (r PropertyRegistry) NodeProfileObject() PropertyTable {
	return PropertyTable{
		ClassCode:   NodeProfile_ClassCode,
		Description: "Node Profile",
		EPCDesc: map[EPCType]PropertyDesc{
			EPC_NPO_OperationStatus: {"Operation status", map[string][]byte{"on": {0x30}, "off": {0x31}}, nil},
			EPC_NPO_VersionInfo:     {"Version information", nil, NPO_VersionInfoDesc{}},
			EPC_NPO_IDNumber:        {"Identification number", nil, IdentificationNumberDesc{}},
			EPCFaultStatus: {"Fault occurrence status", map[string][]byte{
				"fault":    {0x41},
				"no_fault": {0x42},
			}, nil},
			EPC_NPO_FaultStatus:              {"Fault status", nil, nil},
			EPCManufacturerCode:              {"Manufacturer code", ManufacturerCodeEDTs, nil},
			EPCBusinessFacilityCode:          {"Business facility code", nil, nil},
			EPCProductCode:                   {"Product code", nil, StringDesc{MinEDTLen: 12, MaxEDTLen: 12}},
			EPCStatusAnnouncementPropertyMap: {"Status announcement property map", nil, PropertyMapDesc{}},
			EPCSetPropertyMap:                {"Set property map", nil, PropertyMapDesc{}},
			EPCGetPropertyMap:                {"Get property map", nil, PropertyMapDesc{}},
			EPC_NPO_IndividualID:             {"Individual identification information", nil, nil},
			EPC_NPO_SelfNodeInstances:        {"Self-node instances number", nil, NumberDesc{EDTLen: 3, Max: 16777215}},
			EPC_NPO_SelfNodeClasses:          {"Self-node classes number", nil, NumberDesc{EDTLen: 2, Max: 65535}},
			EPC_NPO_InstanceListNotification: {"instance list notification", nil, InstanceListNotificationDesc{}},
			EPC_NPO_SelfNodeInstanceListS:    {"Self-node instance list S", nil, SelfNodeInstanceListDesc{}},
			EPC_NPO_SelfNodeClassListS:       {"Self-node class list S", nil, SelfNodeClassListDesc{}},
		},
		DefaultEPCs: []EPCType{}, // これが空だと通常の devices で表示されない
	}
}
