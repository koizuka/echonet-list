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
