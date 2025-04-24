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
				EPC_RF_DoorOpenStatus: {Desc: "Door open status", Aliases: doorStatusAliases},
				EPC_RF_DoorOpenAlertStatus: {Desc: "Door open alert status", Aliases: map[string][]byte{
					"alert":  {0x41},
					"normal": {0x42},
				}},
				EPC_RF_RefrigeratorDoorOpenStatus: {Desc: "Refrigerator door open status", Aliases: doorStatusAliases},
				EPC_RF_FreezerDoorOpenStatus:      {Desc: "Freezer door open status", Aliases: doorStatusAliases},
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
