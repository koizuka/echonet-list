package echonet_lite

import (
	"echonet-list/echonet_lite/utils"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Property は各プロパティ（EPC, PDC, EDT）を表します。
type Property struct {
	EPC EPCType // プロパティコード
	EDT []byte  // プロパティデータ
}
type Properties []Property

func (p Property) Encode() []byte {
	PDC := len(p.EDT)
	data := make([]byte, 2+PDC) // Create with full length to include EDT
	data[0] = byte(p.EPC)
	data[1] = byte(PDC)
	copy(data[2:], p.EDT)
	return data
}

type PropertyTable struct {
	ClassCode               EOJClassCode
	Description             string            // 英語のデフォルト説明
	DescriptionTranslations map[string]string // 言語別の説明 (e.g., "ja" -> "単機能照明")
	EPCDesc                 map[EPCType]PropertyDesc
	DefaultEPCs             []EPCType
}

// GetDescription returns the description for the specified language.
// If the language is not found in DescriptionMap, it returns the default Description.
func (pt PropertyTable) GetDescription(lang string) string {
	if pt.DescriptionTranslations != nil && lang != "" && lang != "en" {
		if desc, ok := pt.DescriptionTranslations[lang]; ok {
			return desc
		}
	}
	return pt.Description
}

func (pt PropertyTable) FindAlias(alias string) (Property, bool) {
	for epc, desc := range pt.EPCDesc {
		if aliases, ok := desc.Aliases[alias]; ok {
			return Property{EPC: epc, EDT: aliases}, true
		}
	}
	return Property{}, false
}

type PropertyDescription struct {
	ClassCode EOJClassCode
	EPC       EPCType // プロパティコード
	Name      string
	EDT       []byte // プロパティデータ
}

func (p PropertyDescription) String() string {
	return fmt.Sprintf("%s(%s):%X", p.EPC, p.Name, p.EDT)
}

func (pt PropertyTable) AvailableAliases() map[string]PropertyDescription {
	aliases := make(map[string]PropertyDescription)
	for epc, desc := range pt.EPCDesc {
		for alias := range desc.Aliases {
			aliases[alias] = PropertyDescription{
				ClassCode: pt.ClassCode,
				EPC:       epc,
				Name:      desc.Name,
				EDT:       desc.Aliases[alias],
			}
		}
	}
	return aliases
}

func (ps Properties) Encode() []byte {
	data := make([][]byte, len(ps)+1)
	data[0] = []byte{byte(len(ps))}
	for i, p := range ps {
		data[i+1] = p.Encode()
	}
	return utils.FlattenBytes(data)
}

func (ps Properties) GetIdentificationNumber() *IdentificationNumber {
	if p, ok := ps.FindEPC(EPCIdentificationNumber); ok {
		return DecodeIdentificationNumber(p.EDT)
	}
	return nil
}

// EPCType はプロパティコードを表します。
// プロパティコードは、Echonet Lite のプロパティを識別するための 1 バイトの値です。
type EPCType byte

// MarshalJSON は EPCType を "0xXX" 形式のJSON文字列にエンコードします。
func (e EPCType) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("0x%02x", byte(e)))
}

// UnmarshalJSON は "0xXX" 形式または10進数形式のJSON文字列から EPCType をデコードします。
func (e *EPCType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("EPCType should be a string, got %s: %w", data, err)
	}

	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		// 16進数形式 ("0xXX")
		val, err := strconv.ParseUint(s[2:], 16, 8)
		if err != nil {
			return fmt.Errorf("invalid hex EPCType string %q: %w", s, err)
		}
		*e = EPCType(val)
	} else {
		// 10進数形式 (旧フォーマット互換)
		val, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			// 16進数でも10進数でもない場合はエラー
			return fmt.Errorf("invalid decimal or hex EPCType string %q: %w", s, err)
		}
		*e = EPCType(val)
	}
	return nil
}

func (e EPCType) String() string {
	return fmt.Sprintf("%02X", byte(e))
}

func (e EPCType) StringForClass(c EOJClassCode) string {
	if info, ok := GetPropertyDesc(c, e); ok {
		return fmt.Sprintf("%s(%s)", e.String(), info.Name)
	}
	return e.String()
}

func (p Property) EPCString(c EOJClassCode) string {
	EPC := p.EPC.String()
	if info, ok := GetPropertyDesc(c, p.EPC); ok {
		EPC = fmt.Sprintf("%s(%s)", EPC, info.Name)
	}
	return EPC
}

func (p Property) EDTString(c EOJClassCode) string {
	if p.EDT == nil {
		return "nil"
	}
	var result string
	if info, ok := GetPropertyDesc(c, p.EPC); ok {
		result = info.EDTToString(p.EDT)
	}
	if result == "" {
		result = fmt.Sprintf("%X", p.EDT)
	}

	return result
}

func (p Property) String(c EOJClassCode) string {
	return fmt.Sprintf("%s:%s", p.EPCString(c), p.EDTString(c))
}

func (ps Properties) String(ClassCode EOJClassCode) string {
	var results []string
	for _, p := range ps {
		results = append(results, p.String(ClassCode))
	}
	return fmt.Sprintf("[%s]", results)
}

func (ps Properties) FindEPC(epc EPCType) (Property, bool) {
	for _, p := range ps {
		if p.EPC == epc {
			return p, true
		}
	}
	return Property{}, false
}

// UpdateProperty は指定されたEPCのプロパティを更新または追加します。
// 既存のプロパティが見つかった場合は更新し、見つからなかった場合は追加します。
// 更新または追加されたプロパティを含む新しいPropertiesを返します。
func (ps Properties) UpdateProperty(prop Property) Properties {
	// 既存のプロパティを探す
	for i, p := range ps {
		if p.EPC == prop.EPC {
			// 既存のプロパティを更新
			result := make(Properties, len(ps))
			copy(result, ps)
			result[i] = prop
			return result
		}
	}

	// 既存のプロパティが見つからなかった場合は追加
	return append(ps, prop)
}
