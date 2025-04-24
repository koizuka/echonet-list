package echonet_lite

import (
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

// PropertyInfo はプロパティの情報を表します。
type PropertyInfo struct {
	Desc    string              // 説明
	Decoder PropertyDecoderFunc // デコーダ関数
	Aliases map[string][]byte   // Alias names for EDT values (e.g., "on" -> []byte{0x30})
	Number  *NumberValueDesc    // 数値
}

func (p PropertyInfo) ToEDT(value string) ([]byte, bool) {
	if p.Aliases != nil {
		if aliases, ok := p.Aliases[value]; ok {
			return aliases, true
		}
	}
	if p.Number != nil {
		v := value
		if strings.HasSuffix(value, p.Number.Unit) {
			v = strings.TrimSuffix(value, p.Number.Unit)
		}
		if num, err := strconv.Atoi(v); err == nil {
			return p.Number.FromInt(num)
		}
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
		if num, unit, ok := p.Number.ToInt(EDT); ok {
			return fmt.Sprintf("%d%s", num, unit)
		}
	}
	if p.Decoder != nil {
		if decoded, ok := p.Decoder(EDT); ok {
			return decoded.String()
		}
	}
	return ""
}
