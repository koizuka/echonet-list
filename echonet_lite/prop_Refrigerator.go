package echonet_lite

const (
	// EPC
	EPC_RF_DoorOpenStatus             EPCType = 0xB0 // ドア開閉状態
	EPC_RF_DoorOpenAlertStatus        EPCType = 0xB1 // ドア開閉警告状態
	EPC_RF_RefrigeratorDoorOpenStatus EPCType = 0xB2 // 冷蔵室ドア開閉状態
	EPC_RF_FreezerDoorOpenStatus      EPCType = 0xB3 // 冷凍室ドア開閉状態
)

func (r PropertyRegistry) Refrigerator() PropertyRegistryEntry {
	var doorStatusAliases = map[string][]byte{
		"open":   {0x41},
		"closed": {0x42},
	}

	return PropertyRegistryEntry{
		ClassCode: Refrigerator_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Refrigerator",
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_RF_DoorOpenStatus: {"Door open status", nil, doorStatusAliases, nil},
				EPC_RF_DoorOpenAlertStatus: {"Door open alert status", nil, map[string][]byte{
					"alert":  {0x41},
					"normal": {0x42},
				}, nil},
				EPC_RF_RefrigeratorDoorOpenStatus: {"Refrigerator door open status", nil, doorStatusAliases, nil},
				EPC_RF_FreezerDoorOpenStatus:      {"Freezer door open status", nil, doorStatusAliases, nil},
			},
			DefaultEPCs: []EPCType{
				EPC_RF_DoorOpenStatus,
				EPC_RF_DoorOpenAlertStatus,
				EPC_RF_RefrigeratorDoorOpenStatus,
				EPC_RF_FreezerDoorOpenStatus,
			},
		},
	}
}
