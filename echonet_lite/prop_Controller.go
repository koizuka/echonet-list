package echonet_lite

import (
	"echonet-list/echonet_lite/utils"
	"fmt"
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
	// TODO
	EPC_C_InstallAddress EPCType = 0xe0 // 接地住所
)

func (r PropertyRegistry) Controller() PropertyRegistryEntry {
	return PropertyRegistryEntry{
		ClassCode: Controller_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Controller",
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_C_ControllerID:    {Desc: "コントローラID"},
				EPC_C_NumberOfDevices: {Desc: "管理台数", Number: &NumberValueDesc{EDTLen: 2, Max: 65533}},
				EPC_C_Index:           {Desc: "インデックス", Number: &NumberValueDesc{EDTLen: 2, Max: 65533}},
				EPC_C_DeviceID:        {Desc: "機器ID"},
				EPC_C_ClassCode:       {Desc: "機種", Decoder: Decoder(C_DecodeClassCode)},
				EPC_C_Name:            {Desc: "名称", String: &StringValueDesc{MaxEDTLen: 64}},
				EPC_C_ConnectionStatus: {Desc: "接続状態", Aliases: map[string][]byte{
					"connected":    {0x41}, // 接続中
					"disconnected": {0x42}, // 離脱中
					"unregistered": {0x43}, // 未登録
					"deleted":      {0x44}, // 削除
				}},
				EPC_C_InstallAddress: {Desc: "接地住所", String: &StringValueDesc{MaxEDTLen: 255}},
			},
			DefaultEPCs: []EPCType{},
		},
	}
}

type C_ClassCode EOJClassCode

func C_DecodeClassCode(data []byte) C_ClassCode {
	if len(data) != 2 {
		return 0
	}
	return C_ClassCode(EOJClassCode(utils.BytesToUint32(data)))
}
func (c C_ClassCode) String() string {
	return fmt.Sprintf("%v", EOJClassCode(c))
}
