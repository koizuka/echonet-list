package echonet_lite

import (
	"bytes"
	"echonet-list/echonet_lite/props"
	"fmt"
	"strings"
)

const (
	// EPC
	EPC_LS_Illuminance     EPCType = 0xb0
	EPC_LS_SceneControl    EPCType = 0xc0
	EPC_LS_MaxSceneControl EPCType = 0xc1

	EPC_LS_PanasonicF1 EPCType = 0xf1
	EPC_LS_PanasonicF2 EPCType = 0xf2
	EPC_LS_PanasonicF3 EPCType = 0xf3
	EPC_LS_PanasonicF4 EPCType = 0xf4
)

func (r PropertyRegistry) LightingSystem() PropertyRegistryEntry {
	return PropertyRegistryEntry{
		ClassCode: LightingSystem_ClassCode,
		PropertyTable: PropertyTable{
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_LS_Illuminance:     {"Illuminance level", Decoder(props.DecodeIlluminance), nil},
				EPC_LS_SceneControl:    {"Scene control", Decoder(LS_DecodeSceneControl), nil},
				EPC_LS_MaxSceneControl: {"Max scene control", Decoder(LS_DecodeMaxSceneControl), nil},
				EPC_LS_PanasonicF1:     {"Panasonic F1", Decoder(LS_DecodePanasonicFx), nil},
				EPC_LS_PanasonicF2:     {"Panasonic F2", Decoder(LS_DecodePanasonicFx), nil},
				EPC_LS_PanasonicF3:     {"Panasonic F3", Decoder(LS_DecodePanasonicFx), nil},
				EPC_LS_PanasonicF4:     {"Panasonic F4", Decoder(LS_DecodePanasonicFx), nil},
			},
			DefaultEPCs: []EPCType{
				EPC_LS_Illuminance,
				EPC_LS_SceneControl,
				EPC_LS_MaxSceneControl,
			},
		},
	}
}

type LS_SceneControl uint8 // 0:未設定, 1-253:シーン番号

func LS_DecodeSceneControl(EDT []byte) *LS_SceneControl {
	if len(EDT) < 1 {
		return nil
	}
	sceneControl := LS_SceneControl(EDT[0])
	return &sceneControl
}

func (s *LS_SceneControl) String() string {
	if s == nil {
		return "nil"
	}
	return fmt.Sprintf("Scene Control: %d", *s)
}

func (s *LS_SceneControl) EDT() []byte {
	return []byte{byte(*s)}
}

type LS_MaxSceneControl uint8 // 1-253:最大シーン番号

func LS_DecodeMaxSceneControl(EDT []byte) *LS_MaxSceneControl {
	if len(EDT) < 1 {
		return nil
	}
	maxSceneControl := LS_MaxSceneControl(EDT[0])
	return &maxSceneControl
}

func (s *LS_MaxSceneControl) String() string {
	if s == nil {
		return "nil"
	}
	return fmt.Sprintf("Max Scene Control: %d", *s)
}

func (s *LS_MaxSceneControl) EDT() []byte {
	return []byte{byte(*s)}
}

type LS_PanasonicFx struct {
	EPC    EPCType
	Number uint8    // 01
	Labels []string // 0〜10個のラベル
}

func decodeNulTerminatedString(b []byte) string {
	// NULバイトまでを切り出す
	if i := bytes.IndexByte(b, 0); i != -1 {
		b = b[:i]
	}
	return string(b)
}

// TODO Manufacturer codeがPanasonicの場合にのみ使うようにする

func LS_DecodePanasonicFx(EDT []byte) *LS_PanasonicFx {
	// 1バイト目に 01 が入り、そのあと24バイトずつ10個のラベルが入る
	if len(EDT) < 1+24*10 {
		return nil
	}

	panasonicFx := &LS_PanasonicFx{}
	panasonicFx.Number = EDT[0]

	panasonicFx.Labels = make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		// 24バイトだが、00h が入っているところまでがラベル
		label := EDT[1+i*24 : 1+i*24+24]
		panasonicFx.Labels = append(panasonicFx.Labels, decodeNulTerminatedString(label))
	}

	numLabels := 0
	for i, label := range panasonicFx.Labels {
		if label != "" {
			numLabels = i + 1
		}
	}
	panasonicFx.Labels = panasonicFx.Labels[:numLabels]

	return panasonicFx
}

func (p *LS_PanasonicFx) String() string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprintf("%d: %s", p.Number, strings.Join(p.Labels, ", "))
}

func (p *LS_PanasonicFx) EDT() []byte {
	edt := make([]byte, 1+24*10)
	edt[0] = 0x01
	nLabels := len(p.Labels)
	if nLabels > 10 {
		nLabels = 10
	}
	for i := 0; i < nLabels; i++ {
		label := []byte(p.Labels[i])
		if len(label) > 24 {
			label = label[:24]
		}
		copy(edt[1+i*24:], label)
	}
	return edt
}
