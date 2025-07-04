package echonet_lite

import (
	"bytes"
	"echonet-list/echonet_lite/utils"
	"fmt"
	"strconv"
	"strings"
)

// PropertyDesc はプロパティの情報を表します。
type PropertyDesc struct {
	Name              string                       // 説明（英語のデフォルト値）
	NameMap           map[string]string            // 言語別の説明 (e.g., "ja" -> "照度レベル")
	Aliases           map[string][]byte            // Alias names for EDT values (e.g., "on" -> []byte{0x30})
	AliasTranslations map[string]map[string]string // Alias翻訳テーブル (e.g., "ja" -> {"on" -> "オン"})
	Decoder           PropertyDecoder              // デコーダ。PropertyEncoderも実装すると、文字列から変換できる。
}

// GetName returns the property name for the specified language.
// If the language is not found in NameMap, it returns the default Name.
func (p PropertyDesc) GetName(lang string) string {
	if p.NameMap != nil && lang != "" && lang != "en" {
		if name, ok := p.NameMap[lang]; ok {
			return name
		}
	}
	return p.Name
}

// GetAliasTranslations returns the alias translations for the specified language.
// If the language is not found in AliasTranslations, it returns nil.
func (p PropertyDesc) GetAliasTranslations(lang string) map[string]string {
	if p.AliasTranslations != nil && lang != "" && lang != "en" {
		if translations, ok := p.AliasTranslations[lang]; ok {
			return translations
		}
	}
	return nil
}

func (p PropertyDesc) ToEDT(value string) ([]byte, bool) {
	if p.Aliases != nil {
		if aliases, ok := p.Aliases[value]; ok {
			return aliases, true
		}
	}
	if p.Decoder != nil {
		if encoder, ok := p.Decoder.(PropertyEncoder); ok {
			if result, ok := encoder.FromString(value); ok {
				return result, true
			}
		}
	}
	return nil, false
}

// EDTToString はEDTを文字列に変換します。
// 変換できない場合は空文字列を返します。
func (p PropertyDesc) EDTToString(EDT []byte) string {
	if p.Aliases != nil {
		for alias, value := range p.Aliases {
			if bytes.Equal(EDT, value) {
				return alias
			}
		}
	}
	if p.Decoder != nil {
		if decoded, ok := p.Decoder.ToString(EDT); ok {
			return decoded
		}
	}
	return ""
}

type PropertyDecoder interface {
	ToString(EDT []byte) (string, bool)
}

type PropertyEncoder interface {
	FromString(value string) ([]byte, bool)
}

type PropertyIntConverter interface {
	FromInt(num int) ([]byte, bool)
	ToInt(EDT []byte) (int, string, bool)
}

// NumberDescは、数値のプロパティを表します。
// PropertyDecoderとPropertyEncoderとPropertyIntConverterを実装します。
type NumberDesc struct {
	Min    int
	Max    int
	Offset int    // 値が 0のときにEDTに格納する値
	Unit   string // Unit of the value (e.g., "C", "F", "V")
	EDTLen int    // Length of the EDT in bytes(0のときは1扱い)
}

func (n NumberDesc) GetEDTLen() int {
	if n.EDTLen == 0 {
		return 1
	}
	return n.EDTLen
}

func (n NumberDesc) FromInt(num int) ([]byte, bool) {
	if num >= n.Min && num <= n.Max {
		return utils.Uint32ToBytes(uint32(num+n.Offset), n.GetEDTLen()), true
	}
	return nil, false
}

func (n NumberDesc) ToInt(EDT []byte) (int, string, bool) {
	if len(EDT) == n.GetEDTLen() {
		var num int32
		if n.Min >= 0 {
			num = int32(utils.BytesToUint32(EDT)) - int32(n.Offset)
		} else {
			num = utils.BytesToInt32(EDT) - int32(n.Offset)
		}
		if num >= int32(n.Min) && num <= int32(n.Max) {
			return int(num), n.Unit, true
		}
	}
	return 0, "", false
}

func (n NumberDesc) FromString(s string) ([]byte, bool) {
	v := s
	if strings.HasSuffix(s, n.Unit) {
		v = strings.TrimSuffix(s, n.Unit)
	}
	if num, err := strconv.Atoi(v); err == nil {
		return n.FromInt(num)
	}
	return nil, false
}

func (n NumberDesc) ToString(EDT []byte) (string, bool) {
	if num, unit, ok := n.ToInt(EDT); ok {
		return fmt.Sprintf("%d%s", num, unit), true
	}
	return "", false
}

// StringDescは、文字列(UTF-8)のプロパティを表します。
// PropertyDecoderとPropertyEncoderを実装します。
type StringDesc struct {
	MinEDTLen int // 文字列がこのバイト数に満たないときは、 NUL文字で埋める
	MaxEDTLen int // EDTの最大長
}

func (sd StringDesc) FromString(s string) ([]byte, bool) {
	if len(s) == 0 {
		return nil, false
	}
	edt := []byte(s)
	if len(edt) < sd.MinEDTLen {
		result := make([]byte, sd.MinEDTLen)
		copy(result, edt)
		return result, true
	} else if sd.MaxEDTLen > 0 && len(edt) > sd.MaxEDTLen {
		return nil, false
	}
	return edt, true
}

func (sd StringDesc) ToString(EDT []byte) (string, bool) {
	if sd.MinEDTLen > 0 && len(EDT) <= sd.MinEDTLen {
		// NULバイトまでを切り出す
		if i := bytes.IndexByte(EDT, 0); i != -1 {
			EDT = EDT[:i]
		}
	}
	return string(EDT), true
}
