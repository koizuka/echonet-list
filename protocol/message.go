package protocol

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// メッセージの共通部分
type Message struct {
	Type string `json:"type"` // "command", "response", "notification"のいずれか
	ID   string `json:"id"`   // メッセージ識別子（コマンドとレスポンスの関連付け用）
}

// コマンドメッセージ
type CommandMessage struct {
	Message
	Command    string      `json:"command"`    // コマンド名: "discover", "get", "set", "devices", "debug", etc.
	DeviceSpec interface{} `json:"deviceSpec"` // デバイス指定子
	EPCs       []EPCType   `json:"epcs"`       // EPCリスト
	Properties interface{} `json:"properties"` // プロパティリスト
	Options    interface{} `json:"options"`    // オプション指定
}

// レスポンスメッセージ
type ResponseMessage struct {
	Message
	Success bool        `json:"success"` // 成功/失敗のフラグ
	Data    interface{} `json:"data"`    // レスポンスデータ
	Error   string      `json:"error"`   // エラーメッセージ（失敗時）
}

// 通知メッセージ
type NotificationMessage struct {
	Message
	Event      string      `json:"event"`      // イベント種別: "deviceAdded", "deviceTimeout", etc.
	DeviceInfo interface{} `json:"deviceInfo"` // デバイス情報
	Data       interface{} `json:"data"`       // 追加データ
}

// EPCType はEPCのカスタム型
type EPCType uint8

// MarshalJSON は EPCType を16進数文字列としてJSONにマーシャルする
func (e EPCType) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%02X", uint8(e)))
}

// UnmarshalJSON は16進数文字列からEPCTypeにアンマーシャルする
func (e *EPCType) UnmarshalJSON(data []byte) error {
	var hexStr string
	if err := json.Unmarshal(data, &hexStr); err != nil {
		return err
	}
	
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return err
	}
	
	if len(decoded) != 1 {
		return fmt.Errorf("invalid EPC length: expected 1 byte, got %d bytes", len(decoded))
	}
	
	*e = EPCType(decoded[0])
	return nil
}

// ByteArray はカスタムJSONマーシャリングを行うためのバイト配列型
type ByteArray []byte

// MarshalJSON はByteArrayを16進数文字列としてJSONにマーシャルする
func (b ByteArray) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte("null"), nil
	}
	return json.Marshal(hex.EncodeToString(b))
}

// UnmarshalJSON は16進数文字列からByteArrayにアンマーシャルする
func (b *ByteArray) UnmarshalJSON(data []byte) error {
	var hexStr string
	if err := json.Unmarshal(data, &hexStr); err != nil {
		// nullや別の型の場合
		if string(data) == "null" {
			*b = nil
			return nil
		}
		return err
	}
	
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return err
	}
	
	*b = ByteArray(decoded)
	return nil
}

// ClassCode はクラスコードのカスタム型
type ClassCode uint16

// MarshalJSON は ClassCode を16進数文字列としてJSONにマーシャルする
func (c ClassCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%04X", uint16(c)))
}

// UnmarshalJSON は16進数文字列からClassCodeにアンマーシャルする
func (c *ClassCode) UnmarshalJSON(data []byte) error {
	var hexStr string
	if err := json.Unmarshal(data, &hexStr); err != nil {
		return err
	}
	
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return err
	}
	
	if len(decoded) != 2 {
		return fmt.Errorf("invalid ClassCode length: expected 2 bytes, got %d bytes", len(decoded))
	}
	
	*c = ClassCode(uint16(decoded[0])<<8 | uint16(decoded[1]))
	return nil
}

// デバイス情報の変換用
type DeviceInfo struct {
	IP      string     `json:"ip"`      // IPアドレス
	EOJ     EOJInfo    `json:"eoj"`     // EOJ情報
	Aliases []string   `json:"aliases"` // エイリアスリスト
}

// EOJ情報
type EOJInfo struct {
	ClassCode    ClassCode `json:"classCode"`    // クラスコード
	InstanceCode uint8     `json:"instanceCode"` // インスタンスコード
}

// プロパティ情報
type PropertyInfo struct {
	EPC         EPCType    `json:"epc"`                  // EPC
	EDT         ByteArray  `json:"edt"`                  // EDT
	Name        string     `json:"name,omitempty"`       // プロパティ名（存在する場合）
	Description string     `json:"description,omitempty"`// プロパティの説明（存在する場合）
	Value       interface{}`json:"value,omitempty"`      // 変換された値（存在する場合）
}

// DeviceSpecifier情報（クライアント→サーバー通信用）
type DeviceSpecifier struct {
	IP           *string    `json:"ip,omitempty"`           // IPアドレス（省略可）
	ClassCode    *ClassCode `json:"classCode,omitempty"`    // クラスコード（省略可）
	InstanceCode *uint8     `json:"instanceCode,omitempty"` // インスタンスコード（省略可）
	Alias        *string    `json:"alias,omitempty"`        // エイリアス（省略可）
}

// Property情報（クライアント→サーバー通信用）
type Property struct {
	EPC EPCType   `json:"epc"` // EPC
	EDT ByteArray `json:"edt"` // EDT
}

// プロパティ表示モード（devicesコマンド用）
type PropertyMode int

const (
	PropDefault PropertyMode = iota // デフォルトのプロパティを表示
	PropKnown                       // 既知のプロパティのみ表示
	PropAll                         // 全てのプロパティを表示
	PropEPC                         // 特定のEPCのみ表示
)

// DevicePropertyResult はデバイスのプロパティ取得結果
type DevicePropertyResult struct {
	Device     DeviceInfo     `json:"device"`     // デバイス情報
	Properties []PropertyInfo `json:"properties"` // プロパティリスト
	Success    bool           `json:"success"`    // 成功/失敗のフラグ
	Error      string         `json:"error"`      // エラーメッセージ（失敗時）
}

// コマンドオプション（コマンド固有のオプション）
type CommandOptions struct {
	PropMode       *PropertyMode `json:"propMode,omitempty"`       // プロパティ表示モード
	DebugMode      *string       `json:"debugMode,omitempty"`      // デバッグモード ("on"/"off")
	SkipValidation *bool         `json:"skipValidation,omitempty"` // 検証をスキップするフラグ
}