package echonet_lite

import (
	"fmt"
	"sort"
)

// プロパティマップ記述形式
// プロパティマップは、EPC(0x80〜0xff)の有無の集合。
//
// 1. プロパティの個数が16未満の場合 (1+プロパティの個数バイト)
//   1バイト目: プロパティの個数
//   2バイト目以降: EPC がそのまま列挙される
//
// 2. プロパティの個数が16以上の場合 (17バイト)
//   1バイト目: プロパティの個数
//   2〜17バイト目: プロパティコードのビットマップ。8*16=128ビット。EPCは0x80〜0xff
//     ビットの場所は bytes[(EPC & 0x0f)] & (1 << ((EPC >> 4) - 8)) で表す。

type PropertyMap map[EPCType]struct{}

func (m PropertyMap) Has(epc EPCType) bool {
	_, ok := m[epc]
	return ok
}

func (m PropertyMap) Set(epc EPCType) {
	m[epc] = struct{}{}
}

func (m PropertyMap) Delete(epc EPCType) {
	delete(m, epc)
}

func (m PropertyMap) EPCs() []EPCType {
	epcs := make([]EPCType, 0, len(m))
	for epc := range m {
		epcs = append(epcs, epc)
	}
	return epcs
}

type ErrInvalidPropertyMap struct {
	EDT []byte
}

func (e ErrInvalidPropertyMap) Error() string {
	return fmt.Sprintf("invalid property map: %X", e.EDT)
}

type PropertyMapDesc struct{}

func (d PropertyMapDesc) ToString(EDT []byte) (string, bool) {
	p := DecodePropertyMap(EDT)
	if p == nil {
		return "", false
	}
	return p.String(), true
}

func (m PropertyMap) Encode() []byte {
	if len(m) < 16 {
		bytes := make([]byte, 1, 1+len(m))
		bytes[0] = byte(len(m))
		for epc := range m {
			bytes = append(bytes, byte(epc))
		}
		return bytes
	}

	bytes := make([]byte, 17)
	bytes[0] = byte(len(m))
	for epc := range m {
		bytes[epc&0x0f+1] |= 1 << (epc>>4 - 8)
	}
	return bytes
}

func DecodePropertyMap(bytes []byte) PropertyMap {
	m := make(PropertyMap)
	if len(bytes) < 1 {
		return m
	}

	n := int(bytes[0])
	if n < 16 {
		if len(bytes) != n+1 {
			return nil
		}
		for _, epc := range bytes[1:] {
			m[EPCType(epc)] = struct{}{}
		}
	} else {
		if len(bytes) != 17 {
			return nil
		}
		for i, b := range bytes[1:] {
			for j := 0; j < 8; j++ {
				if b&(1<<j) != 0 {
					m[EPCType(i+j<<4+0x80)] = struct{}{}
				}
			}
		}
	}
	return m
}

func (m PropertyMap) String() string {
	var arr []EPCType
	for epc := range m {
		arr = append(arr, epc)
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i] < arr[j]
	})
	return fmt.Sprint(arr)
}
