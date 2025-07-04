package echonet_lite

import (
	"echonet-list/echonet_lite/utils"
)

const (
	// EPC
	EPC_C_ControllerID     EPCType = 0xc0 // コントローラID
	EPC_C_NumberOfDevices  EPCType = 0xc1 // 管理台数
	EPC_C_Index            EPCType = 0xc2 // インデックス
	EPC_C_DeviceID         EPCType = 0xc3 // 機器ID
	EPC_C_ClassCode        EPCType = 0xc4 // 機種
	EPC_C_Name             EPCType = 0xc5 // 名称
	EPC_C_ConnectionStatus EPCType = 0xc6 // 接続状態
	EPC_C_InstallAddress   EPCType = 0xe0 // 設置住所
)

func (r PropertyRegistry) Controller() PropertyTable {
	return PropertyTable{
		ClassCode:   Controller_ClassCode,
		Description: "Controller",
		EPCDesc: map[EPCType]PropertyDesc{
			EPC_C_ControllerID: {
				Name: "Controller ID",
				NameMap: map[string]string{
					"ja": "コントローラID",
				},
				Aliases: nil,
				Decoder: nil,
			},
			EPC_C_NumberOfDevices: {
				Name: "Number of devices",
				NameMap: map[string]string{
					"ja": "管理台数",
				},
				Aliases: nil,
				Decoder: NumberDesc{EDTLen: 2, Max: 65533},
			},
			EPC_C_Index: {
				Name: "Index",
				NameMap: map[string]string{
					"ja": "インデックス",
				},
				Aliases: nil,
				Decoder: NumberDesc{EDTLen: 2, Max: 65533},
			},
			EPC_C_DeviceID: {
				Name: "Device ID",
				NameMap: map[string]string{
					"ja": "機器ID",
				},
				Aliases: nil,
				Decoder: nil,
			},
			EPC_C_ClassCode: {
				Name: "Class code",
				NameMap: map[string]string{
					"ja": "機種",
				},
				Aliases: nil,
				Decoder: C_ClassCodeDesc{},
			},
			EPC_C_Name: {
				Name: "Name",
				NameMap: map[string]string{
					"ja": "名称",
				},
				Aliases: nil,
				Decoder: StringDesc{MaxEDTLen: 64},
			},
			EPC_C_ConnectionStatus: {
				Name: "Connection status",
				NameMap: map[string]string{
					"ja": "接続状態",
				},
				Aliases: map[string][]byte{
					"connected":    {0x41}, // 接続中
					"disconnected": {0x42}, // 離脱中
					"unregistered": {0x43}, // 未登録
					"deleted":      {0x44}, // 削除
				},
				AliasTranslations: map[string]map[string]string{
					"ja": {
						"connected":    "接続中",
						"disconnected": "離脱中",
						"unregistered": "未登録",
						"deleted":      "削除",
					},
				},
				Decoder: nil,
			},
			EPC_C_InstallAddress: {
				Name: "Install address",
				NameMap: map[string]string{
					"ja": "設置住所",
				},
				Aliases: nil,
				Decoder: StringDesc{MaxEDTLen: 255},
			},
		},
		DefaultEPCs: []EPCType{},
	}
}

type C_ClassCodeDesc struct{}

func (c C_ClassCodeDesc) ToString(EDT []byte) (string, bool) {
	if len(EDT) != 2 {
		return "", false
	}
	classCode := EOJClassCode(utils.BytesToUint32(EDT))
	return classCode.String(), true
}
