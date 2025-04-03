package echonet_lite

import (
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

// Equals は2つのIPAndEOJが等しいかどうかを判定する
func (d IPAndEOJ) Equals(other IPAndEOJ) bool {
	// IPアドレスの比較
	if !d.IP.Equal(other.IP) {
		return false
	}
	// EOJの比較
	return d.EOJ == other.EOJ
}
