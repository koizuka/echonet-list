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

func (r PropertyRegistry) Controller() PropertyRegistryEntry {
	return PropertyRegistryEntry{
		ClassCode: Controller_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Controller",
			EPCDesc: map[EPCType]PropertyDesc{
				EPC_C_ControllerID:    {"コントローラID", nil, nil},
				EPC_C_NumberOfDevices: {"管理台数", nil, NumberDesc{EDTLen: 2, Max: 65533}},
				EPC_C_Index:           {"インデックス", nil, NumberDesc{EDTLen: 2, Max: 65533}},
				EPC_C_DeviceID:        {"機器ID", nil, nil},
				EPC_C_ClassCode:       {"機種", nil, C_ClassCodeDesc{}},
				EPC_C_Name:            {"名称", nil, StringDesc{MaxEDTLen: 64}},
				EPC_C_ConnectionStatus: {"接続状態", map[string][]byte{
					"connected":    {0x41}, // 接続中
					"disconnected": {0x42}, // 離脱中
					"unregistered": {0x43}, // 未登録
					"deleted":      {0x44}, // 削除
				}, nil},
				EPC_C_InstallAddress: {"設置住所", nil, StringDesc{MaxEDTLen: 255}},
			},
			DefaultEPCs: []EPCType{},
		},
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
