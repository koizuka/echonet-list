package echonet_lite

func (r PropertyRegistry) NodeProfileObject() PropertyTable {
	return PropertyTable{
		ClassCode:   NodeProfile_ClassCode,
		Description: "Node Profile",
		DescriptionMap: map[string]string{
			"ja": "ノードプロファイル",
		},
		EPCDesc: map[EPCType]PropertyDesc{
			EPC_NPO_OperationStatus: {
				Name: "Operation status",
				NameMap: map[string]string{
					"ja": "動作状態",
				},
				Aliases: map[string][]byte{"on": {0x30}, "off": {0x31}},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"on":  "動作中",
						"off": "停止中",
					},
				},
				Decoder: nil,
			},
			EPC_NPO_VersionInfo: {
				Name: "Version information",
				NameMap: map[string]string{
					"ja": "バージョン情報",
				},
				Aliases: nil,
				Decoder: NPO_VersionInfoDesc{},
			},
			EPC_NPO_IDNumber: {
				Name: "Identification number",
				NameMap: map[string]string{
					"ja": "識別番号",
				},
				Aliases: nil,
				Decoder: IdentificationNumberDesc{},
			},
			EPCFaultStatus: {
				Name: "Fault occurrence status",
				NameMap: map[string]string{
					"ja": "異常発生状態",
				},
				Aliases: map[string][]byte{
					"fault":    {0x41},
					"no_fault": {0x42},
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"fault":    "異常あり",
						"no_fault": "異常なし",
					},
				},
				Decoder: nil,
			},
			EPC_NPO_FaultStatus: {
				Name: "Fault status",
				NameMap: map[string]string{
					"ja": "異常状態",
				},
				Aliases: nil,
				Decoder: nil,
			},
			EPCManufacturerCode: {
				Name: "Manufacturer code",
				NameMap: map[string]string{
					"ja": "メーカコード",
				},
				Aliases: ManufacturerCodeEDTs,
				Decoder: nil,
			},
			EPCBusinessFacilityCode: {
				Name: "Business facility code",
				NameMap: map[string]string{
					"ja": "事業場コード",
				},
				Aliases: nil,
				Decoder: nil,
			},
			EPCProductCode: {
				Name: "Product code",
				NameMap: map[string]string{
					"ja": "商品コード",
				},
				Aliases: nil,
				Decoder: StringDesc{MinEDTLen: 12, MaxEDTLen: 12},
			},
			EPCStatusAnnouncementPropertyMap: {
				Name: "Status announcement property map",
				NameMap: map[string]string{
					"ja": "状変アナウンスプロパティマップ",
				},
				Aliases: nil,
				Decoder: PropertyMapDesc{},
			},
			EPCSetPropertyMap: {
				Name: "Set property map",
				NameMap: map[string]string{
					"ja": "Setプロパティマップ",
				},
				Aliases: nil,
				Decoder: PropertyMapDesc{},
			},
			EPCGetPropertyMap: {
				Name: "Get property map",
				NameMap: map[string]string{
					"ja": "Getプロパティマップ",
				},
				Aliases: nil,
				Decoder: PropertyMapDesc{},
			},
			EPC_NPO_IndividualID: {
				Name: "Individual identification information",
				NameMap: map[string]string{
					"ja": "個体識別情報",
				},
				Aliases: nil,
				Decoder: nil,
			},
			EPC_NPO_SelfNodeInstances: {
				Name: "Self-node instances number",
				NameMap: map[string]string{
					"ja": "自ノードインスタンス数",
				},
				Aliases: nil,
				Decoder: NumberDesc{EDTLen: 3, Max: 16777215},
			},
			EPC_NPO_SelfNodeClasses: {
				Name: "Self-node classes number",
				NameMap: map[string]string{
					"ja": "自ノードクラス数",
				},
				Aliases: nil,
				Decoder: NumberDesc{EDTLen: 2, Max: 65535},
			},
			EPC_NPO_InstanceListNotification: {
				Name: "instance list notification",
				NameMap: map[string]string{
					"ja": "インスタンスリスト通知",
				},
				Aliases: nil,
				Decoder: InstanceListNotificationDesc{},
			},
			EPC_NPO_SelfNodeInstanceListS: {
				Name: "Self-node instance list S",
				NameMap: map[string]string{
					"ja": "自ノードインスタンスリストS",
				},
				Aliases: nil,
				Decoder: SelfNodeInstanceListDesc{},
			},
			EPC_NPO_SelfNodeClassListS: {
				Name: "Self-node class list S",
				NameMap: map[string]string{
					"ja": "自ノードクラスリストS",
				},
				Aliases: nil,
				Decoder: SelfNodeClassListDesc{},
			},
		},
		DefaultEPCs: []EPCType{}, // これが空だと通常の devices で表示されない
	}
}
