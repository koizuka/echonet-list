package echonet_lite

import (
	"bytes"
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
	Illuminance := NumberValueDesc{Min: 0, Max: 100, Unit: "%"}

	return PropertyRegistryEntry{
		ClassCode: LightingSystem_ClassCode,
		PropertyTable: PropertyTable{
			Description: "Lighting System",
			EPCInfo: map[EPCType]PropertyInfo{
				EPC_LS_Illuminance:     {Desc: "Illuminance level", Number: &Illuminance},
				EPC_LS_SceneControl:    {Desc: "Scene control", Number: &NumberValueDesc{EDTLen: 1, Min: 0, Max: 253}},     // 0:未設定, 1-253:シーン番号
				EPC_LS_MaxSceneControl: {Desc: "Max scene control", Number: &NumberValueDesc{EDTLen: 1, Min: 1, Max: 253}}, // 1-253:最大シーン番号
				EPC_LS_PanasonicF1:     {Desc: "Panasonic F1", Decoder: Decoder(LS_DecodePanasonicFx)},
				EPC_LS_PanasonicF2:     {Desc: "Panasonic F2", Decoder: Decoder(LS_DecodePanasonicFx)},
				EPC_LS_PanasonicF3:     {Desc: "Panasonic F3", Decoder: Decoder(LS_DecodePanasonicFx)},
				EPC_LS_PanasonicF4:     {Desc: "Panasonic F4", Decoder: Decoder(LS_DecodePanasonicFx)},
			},
			DefaultEPCs: []EPCType{
				EPC_LS_Illuminance,
				EPC_LS_SceneControl,
				EPC_LS_MaxSceneControl,
			},
		},
	}
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
