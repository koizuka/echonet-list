package echonet_lite

import (
	"bytes"
	"echonet-list/echonet_lite/utils"
	"fmt"
	"strconv"
	"strings"
)

type PropertyDecoderFunc func(EDT []byte) (fmt.Stringer, bool)

func Decoder[T fmt.Stringer](f func(EDT []byte) T) PropertyDecoderFunc {
	return func(EDT []byte) (fmt.Stringer, bool) {
		if len(EDT) == 0 {
			return nil, false
		}
		result := f(EDT)
		// if T is a pointer, nil check
		if _, ok := any(result).(fmt.Stringer); !ok {
			return nil, false
		}
		return result, true
	}
}

type NumberValueDesc struct {
	Min    int
	Max    int
	Offset int    // 値が 0のときにEDTに格納する値
	Unit   string // Unit of the value (e.g., "C", "F", "V")
	EDTLen int    // Length of the EDT in bytes(0のときは1扱い)
}

func (n NumberValueDesc) GetEDTLen() int {
	if n.EDTLen == 0 {
		return 1
	}
	return n.EDTLen
}

func (n NumberValueDesc) FromInt(num int) ([]byte, bool) {
	if num >= n.Min && num <= n.Max {
		return utils.Uint32ToBytes(uint32(num+n.Offset), n.GetEDTLen()), true
	}
	return nil, false
}

func (n NumberValueDesc) ToInt(EDT []byte) (int, string, bool) {
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

func (n NumberValueDesc) FromString(s string) ([]byte, bool) {
	v := s
	if strings.HasSuffix(s, n.Unit) {
		v = strings.TrimSuffix(s, n.Unit)
	}
	if num, err := strconv.Atoi(v); err == nil {
		return n.FromInt(num)
	}
	return nil, false
}

func (n NumberValueDesc) ToString(EDT []byte) (string, bool) {
	if num, unit, ok := n.ToInt(EDT); ok {
		return fmt.Sprintf("%d%s", num, unit), true
	}
	return "", false
}

// StringValueDescは、文字列(UTF-8)のプロパティを表します。
type StringValueDesc struct {
	MinEDTLen int // 文字列がこのバイト数に満たないときは、 NUL文字で埋める
	MaxEDTLen int // EDTの最大長
}

func (sd StringValueDesc) FromString(s string) ([]byte, bool) {
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

func (sd StringValueDesc) ToString(EDT []byte) (string, bool) {
	if sd.MinEDTLen > 0 && len(EDT) <= sd.MinEDTLen {
		// NULバイトまでを切り出す
		if i := bytes.IndexByte(EDT, 0); i != -1 {
			EDT = EDT[:i]
		}
	}
	return string(EDT), true
}

// PropertyInfo はプロパティの情報を表します。
type PropertyInfo struct {
	Desc    string              // 説明
	Decoder PropertyDecoderFunc // デコーダ関数
	Aliases map[string][]byte   // Alias names for EDT values (e.g., "on" -> []byte{0x30})
	Number  *NumberValueDesc    // 数値
	String  *StringValueDesc    // 文字列
}

func (p PropertyInfo) ToEDT(value string) ([]byte, bool) {
	if p.Aliases != nil {
		if aliases, ok := p.Aliases[value]; ok {
			return aliases, true
		}
	}
	if p.Number != nil {
		return p.Number.FromString(value)
	}
	if p.String != nil {
		return p.String.FromString(value)
	}
	return nil, false
}

// EDTToString はEDTを文字列に変換します。
// 変換できない場合は空文字列を返します。
func (p PropertyInfo) EDTToString(EDT []byte) string {
	if p.Aliases != nil {
		for alias, value := range p.Aliases {
			if string(EDT) == string(value) {
				return alias
			}
		}
	}
	if p.Number != nil {
		if decoded, ok := p.Number.ToString(EDT); ok {
			return decoded
		}
	}
	if p.String != nil {
		if decoded, ok := p.String.ToString(EDT); ok {
			return decoded
		}
	}
	if p.Decoder != nil {
		if decoded, ok := p.Decoder(EDT); ok {
			return decoded.String()
		}
	}
	return ""
}
