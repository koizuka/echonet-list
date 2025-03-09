package echonet_lite

import (
	"fmt"
	"strings"
)

// ECHONET Lite 資料
// https://echonet.jp/spec_g/
//  https://echonet.jp/spec_v114_lite/ (ECHONET Lite)
//  https://echonet.jp/spec_object_rr2/ (ECHONET Liteオブジェクト)

// ECHONETLiteMessage はECHONET Liteのメッセージを表します。
type ECHONETLiteMessage struct {
	EHD              EHDType    // ヘッダ
	TID              TIDType    // トランザクションID
	SEOJ             EOJ        // 送信元ECHONETオブジェクト
	DEOJ             EOJ        // 宛先ECHONETオブジェクト
	ESV              ESVType    // サービスコード
	Properties       Properties // プロパティリスト
	SetGetProperties Properties // SetGetのときのGetプロパティ
}

const (
	EHD_ECHONETLite EHDType = 0x1081 // ECHONET Liteのヘッダ

	ECHONETLitePort = 3610 // ECHONET Liteのポート番号
)

type EHDType uint16

func DecodeEHD(data []byte) EHDType {
	if len(data) < 2 {
		return 0
	}
	return EHDType(data[0])<<8 + EHDType(data[1])
}
func (e EHDType) Encode() []byte {
	return []byte{byte(e >> 8), byte(e & 0xff)}
}

func (e EHDType) String() string {
	switch e {
	case EHD_ECHONETLite:
		return "ECHONET Lite"
	default:
		return fmt.Sprintf("(%X)", uint16(e))
	}
}

type TIDType uint16

func DecodeTID(data []byte) TIDType {
	if len(data) < 2 {
		return 0
	}
	return TIDType(data[0])<<8 + TIDType(data[1])
}
func (t TIDType) Encode() []byte {
	return []byte{byte(t >> 8), byte(t & 0xff)}
}

func (m *ECHONETLiteMessage) EOJ() EOJ {
	switch m.ESV {
	case ESVSet_Res, ESVGet_Res, ESVINF, ESVINFC, ESVINFC_Res, ESVSetGet_Res,
		ESVSetI_SNA, ESVSetC_SNA, ESVGet_SNA, ESVINF_REQ_SNA, ESVSetGet_SNA:
		return m.SEOJ
	}
	return m.DEOJ
}

func (m *ECHONETLiteMessage) String() string {
	EOJ := m.EOJ()
	parts := []string{
		fmt.Sprintf("EHD:%v", m.EHD),
		fmt.Sprintf("TID:%v", m.TID),
		fmt.Sprintf("SEOJ:%v", m.SEOJ),
		fmt.Sprintf("DEOJ:%v", m.DEOJ),
		fmt.Sprintf("ESV:%v", m.ESV),
	}
	if m.ESV.ISSetGet() {
		parts = append(parts,
			fmt.Sprintf("Properties(Set):%v", m.Properties.String(EOJ.ClassCode())),
			fmt.Sprintf("Properties(Get):%v", m.SetGetProperties.String(EOJ.ClassCode())),
		)
	} else {
		parts = append(parts,
			fmt.Sprintf("Properties:%v", m.Properties.String(EOJ.ClassCode())),
		)
	}
	return strings.Join(parts, ", ")
}

type ESVType byte

func (e ESVType) Encode() []byte {
	return []byte{byte(e)}
}

const (
	ESVSetI    ESVType = 0x60 // SetI プロパティ値書き込み要求（応答不要）
	ESVSetC    ESVType = 0x61 // SetC プロパティ値書き込み要求（応答要）
	ESVGet     ESVType = 0x62 // Get プロパティ値読み出し要求
	ESVINF_REQ ESVType = 0x63 // INF_REQ プロパティ値通知要求
	ESVSetGet  ESVType = 0x6e // SetGet プロパティ値書き込み・読み出し要求

	ESVSet_Res    ESVType = 0x71 // Set_Res プロパティ値書き込み応答
	ESVGet_Res    ESVType = 0x72 // Get_Res プロパティ値読み出し応答
	ESVINF        ESVType = 0x73 // INF プロパティ値通知
	ESVINFC       ESVType = 0x74 // INFC プロパティ値通知（応答要）
	ESVINFC_Res   ESVType = 0x7a // INFC_Res プロパティ値通知応答
	ESVSetGet_Res ESVType = 0x7e // SetGet_Res プロパティ値書き込み・読み出し応答

	ESVSetI_SNA    ESVType = 0x50 // SetI_SNA プロパティ値書き込み要求不可応答
	ESVSetC_SNA    ESVType = 0x51 // SetC_SNA プロパティ値書き込み要求不可応答
	ESVGet_SNA     ESVType = 0x52 // Get_SNA プロパティ値読み出し要求不可応答
	ESVINF_REQ_SNA ESVType = 0x53 // INF_REQ_SNA プロパティ値通知要求不可応答
	ESVSetGet_SNA  ESVType = 0x5e // SetGet_SNA プロパティ値書き込み・読み出し要求不可応答
)

func (e ESVType) String() string {
	switch e {
	case ESVSetI:
		return "SetI"
	case ESVSetC:
		return "SetC"
	case ESVGet:
		return "Get"
	case ESVINF_REQ:
		return "INF_REQ"
	case ESVSetGet:
		return "SetGet"
	case ESVINF:
		return "INF"
	case ESVINFC:
		return "INFC"
	case ESVINFC_Res:
		return "INFC_Res"
	case ESVSet_Res:
		return "Set_Res"
	case ESVGet_Res:
		return "Get_Res"
	case ESVSetGet_Res:
		return "SetGet_Res"
	case ESVSetI_SNA:
		return "SetI_SNA"
	case ESVSetC_SNA:
		return "SetC_SNA"
	case ESVGet_SNA:
		return "Get_SNA"
	case ESVINF_REQ_SNA:
		return "INF_REQ_SNA"
	case ESVSetGet_SNA:
		return "SetGet_SNA"

	default:
		return fmt.Sprintf("(%X)", byte(e))
	}
}

// ESVSetI -> 成功:応答無し, 失敗: ESVSetI_SNA
// ESVSetC -> 成功:ESVSet_Res, 失敗: ESVSetC_SNA
// ESVGet -> 成功:ESVGet_Res, 失敗: ESVGet_SNA
// ESVINF_REQ -> 成功:ESVINF, 失敗: ESVINF_REQ_SNA
// ESVSetGet -> 成功:ESVSetGet_Res, 失敗: ESVSetGet_SNA
// ESVINFC -> 成功:ESVINFC_Res
func (e ESVType) ResponseESVs() []ESVType {
	switch e {
	case ESVSetI:
		return []ESVType{ESVSetI_SNA}
	case ESVSetC:
		return []ESVType{ESVSet_Res, ESVSetC_SNA}
	case ESVGet:
		return []ESVType{ESVGet_Res, ESVGet_SNA}
	case ESVINF_REQ:
		return []ESVType{ESVINF, ESVINF_REQ_SNA}
	case ESVSetGet:
		return []ESVType{ESVSetGet_Res, ESVSetGet_SNA}
	case ESVINFC:
		return []ESVType{ESVINFC_Res}
	default:
		return nil
	}
}

func (e ESVType) ISSetGet() bool {
	return e == ESVSetGet || e == ESVSetGet_Res || e == ESVSetGet_SNA
}

func parseProperties(data []byte, pos int) (int, []Property, error) {
	OPC := data[pos]
	pos++
	properties := make([]Property, 0, OPC)
	for i := 0; i < int(OPC); i++ {
		if pos+2 > len(data) {
			return pos, nil, fmt.Errorf("プロパティの長さが不正です")
		}
		prop := Property{
			EPC: EPCType(data[pos]),
		}
		PDC := int(data[pos+1])
		pos += 2
		if PDC > 0 {
			if pos+PDC > len(data) {
				return pos, nil, fmt.Errorf("EDTの長さが不正です")
			}
			prop.EDT = data[pos : pos+PDC]
			pos += PDC
		}
		properties = append(properties, prop)
	}
	return pos, properties, nil
}

// ParseECHONETLiteMessage は受信したバイト列からECHONET Liteメッセージをパースします。
func ParseECHONETLiteMessage(data []byte) (*ECHONETLiteMessage, error) {
	// 最低限、EHD(2)+TID(2)+SEOJ(3)+DEOJ(3)+ESV(1)+OPC(1)=12バイトは必要
	if len(data) < 12 {
		return nil, fmt.Errorf("パケットが短すぎます: %d バイト", len(data))
	}

	msg := &ECHONETLiteMessage{
		EHD:  DecodeEHD(data[0:2]),
		TID:  DecodeTID(data[2:4]),
		SEOJ: DecodeEOJ(data[4:7]),
		DEOJ: DecodeEOJ(data[7:10]),
		ESV:  ESVType(data[10]),
	}
	pos, properties, err := parseProperties(data, 11)
	if err != nil {
		return nil, err
	}
	msg.Properties = properties

	if msg.ESV.ISSetGet() {
		_, properties, err = parseProperties(data, pos)
		if err != nil {
			return nil, err
		}
		msg.SetGetProperties = properties
	}
	return msg, nil
}

func flattenBytes(chunks [][]byte) []byte {
	// 合計サイズを計算
	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk)
	}

	// 必要なサイズを確保
	result := make([]byte, 0, totalSize)

	// バイト列を結合
	for _, chunk := range chunks {
		result = append(result, chunk...)
	}

	return result
}

type IEncodable interface {
	Encode() []byte
}

func encode(encodables ...IEncodable) []byte {
	data := make([][]byte, len(encodables))
	for i, encodable := range encodables {
		data[i] = encodable.Encode()
	}
	return flattenBytes(data)
}

func (m *ECHONETLiteMessage) Encode() []byte {
	EHD := m.EHD
	if EHD == 0 {
		EHD = EHD_ECHONETLite
	}
	return encode(EHD, m.TID, m.SEOJ, m.DEOJ, m.ESV, m.Properties)
}
