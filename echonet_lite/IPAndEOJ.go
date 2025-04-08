package echonet_lite

import (
	"bytes"
	"fmt"
	"net"
)

// IPAndEOJ は、デバイスの情報を表す構造体
type IPAndEOJ struct {
	IP  net.IP
	EOJ EOJ
}

func (d IPAndEOJ) String() string {
	return fmt.Sprintf("%v %v", d.IP, d.EOJ)
}

func (d IPAndEOJ) Specifier() string {
	return fmt.Sprintf("%v %v", d.IP, d.EOJ.Specifier())
}

// Key は IPAndEOJ をマップのキーとして使用するための文字列表現を返します
// "IP EOJ" 形式の文字列を返します（例: "192.168.0.1 0130:1"）
func (d IPAndEOJ) Key() string {
	return d.Specifier()
}

func (d IPAndEOJ) Compare(other IPAndEOJ) int {
	if d.IP.Equal(other.IP) {
		if d.EOJ > other.EOJ {
			return 1
		} else if d.EOJ < other.EOJ {
			return -1
		}
		return 0
	}
	return bytes.Compare(d.IP, other.IP)
}
