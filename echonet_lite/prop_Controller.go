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
)

func (r PropertyRegistry) Controller() PropertyRegistryEntry {
	return PropertyRegistryEntry{
		ClassCode: Controller_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Controller",
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_C_ControllerID:    {"コントローラID", nil, nil, nil},
				EPC_C_NumberOfDevices: {"管理台数", Decoder(C_DecodeNumberOfDevices), nil, nil},
				EPC_C_Index:           {"インデックス", Decoder(C_DecodeIndex), nil, nil},
				EPC_C_DeviceID:        {"機器ID", nil, nil, nil},
				EPC_C_ClassCode:       {"機種", Decoder(C_DecodeClassCode), nil, nil},
				EPC_C_Name:            {"名称", Decoder(C_DecodeName), nil, nil},
				EPC_C_ConnectionStatus: {"接続状態", nil, map[string][]byte{
					"connected":    {0x41}, // 接続中
					"disconnected": {0x42}, // 離脱中
					"unregistered": {0x43}, // 未登録
					"deleted":      {0x44}, // 削除
				}, nil},
			},
			DefaultEPCs: []EPCType{},
		},
	}
}

type C_NumberOfDevices uint16

func C_DecodeNumberOfDevices(data []byte) C_NumberOfDevices {
	if len(data) != 2 {
		return 0
	}
	return C_NumberOfDevices(utils.BytesToUint32(data))
}

func (n C_NumberOfDevices) String() string {
	return fmt.Sprintf("%d", n)
}

type C_Index uint16

func C_DecodeIndex(data []byte) C_Index {
	if len(data) != 2 {
		return 0
	}
	return C_Index(utils.BytesToUint32(data))
}
func (i C_Index) String() string {
	return fmt.Sprintf("%d", i)
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

type C_Name string

func C_DecodeName(data []byte) C_Name {
	if len(data) == 0 {
		return ""
	}
	return C_Name(string(data))
}
func (n C_Name) String() string {
	return string(n)
}
