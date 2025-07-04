package echonet_lite

const (
	// EPC
	EPC_RF_DoorOpenStatus             EPCType = 0xB0 // ドア開閉状態
	EPC_RF_DoorOpenAlertStatus        EPCType = 0xB1 // ドア開閉警告状態
	EPC_RF_RefrigeratorDoorOpenStatus EPCType = 0xB2 // 冷蔵室ドア開閉状態
	EPC_RF_FreezerDoorOpenStatus      EPCType = 0xB3 // 冷凍室ドア開閉状態
)

func (r PropertyRegistry) Refrigerator() PropertyTable {
	var doorStatusAliases = map[string][]byte{
		"open":   {0x41},
		"closed": {0x42},
	}
	var doorStatusAliasTranslations = map[string]map[string]string{
		"ja": {
			"open":   "開",
			"closed": "閉",
		},
	}

	return PropertyTable{
		ClassCode:   Refrigerator_ClassCode,
		Description: "Refrigerator",
		DescriptionMap: map[string]string{
			"ja": "冷蔵庫",
		},
		EPCDesc: map[EPCType]PropertyDesc{
			EPC_RF_DoorOpenStatus: {
				Name: "Door open status",
				NameMap: map[string]string{
					"ja": "ドア開閉状態",
				},
				Aliases:           doorStatusAliases,
				AliasTranslations: doorStatusAliasTranslations,
				Decoder:           nil,
			},
			EPC_RF_DoorOpenAlertStatus: {
				Name: "Door open alert status",
				NameMap: map[string]string{
					"ja": "ドア開閉警告状態",
				},
				Aliases: map[string][]byte{
					"alert":  {0x41},
					"normal": {0x42},
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"alert":  "警告あり",
						"normal": "通常",
					},
				},
				Decoder: nil,
			},
			EPC_RF_RefrigeratorDoorOpenStatus: {
				Name: "Refrigerator door open status",
				NameMap: map[string]string{
					"ja": "冷蔵室ドア開閉状態",
				},
				Aliases:           doorStatusAliases,
				AliasTranslations: doorStatusAliasTranslations,
				Decoder:           nil,
			},
			EPC_RF_FreezerDoorOpenStatus: {
				Name: "Freezer door open status",
				NameMap: map[string]string{
					"ja": "冷凍室ドア開閉状態",
				},
				Aliases:           doorStatusAliases,
				AliasTranslations: doorStatusAliasTranslations,
				Decoder:           nil,
			},
		},
		DefaultEPCs: []EPCType{
			EPC_RF_DoorOpenStatus,
			EPC_RF_DoorOpenAlertStatus,
			EPC_RF_RefrigeratorDoorOpenStatus,
			EPC_RF_FreezerDoorOpenStatus,
		},
	}
}
