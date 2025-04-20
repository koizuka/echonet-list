package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// UDPConnection は UDP ソケットを管理します
type UDPConnection struct {
	UdpConn   *net.UDPConn
	LocalAddr *net.UDPAddr
	Port      int
}

// UDPConnectionOptions は接続オプションを指定します
type UDPConnectionOptions struct {
	DefaultTimeout time.Duration
}

// CreateUDPConnection は IPv4 の unicast と multicast (マルチキャスト) を受信対応します。
// ip が nil の場合はワイルドカード listen、multicastIP がブロードキャストかつIPv4の場合は broadcast として受信。
// multicastIP が真のマルチキャストかつIPv4の場合はグループ参加。
// ip または multicastIP がIPv6の場合はエラーになります。
func CreateUDPConnection(ctx context.Context, ip net.IP, port int, multicastIP net.IP, opt UDPConnectionOptions) (*UDPConnection, error) {
	// IPv6 非対応チェック
	if ip != nil && ip.To4() == nil {
		return nil, fmt.Errorf("IPv6 not supported for unicast ip")
	}
	if multicastIP != nil && multicastIP.To4() == nil {
		return nil, fmt.Errorf("IPv6 not supported for multicastIP")
	}

	// IPv4 broadcast 指定時は multicastIP を無視して listen
	if multicastIP != nil && multicastIP.Equal(net.IPv4bcast) {
		multicastIP = nil
	}

	var conn *net.UDPConn
	var err error

	if multicastIP != nil {
		// IPv4 マルチキャスト
		if !multicastIP.IsMulticast() {
			return nil, fmt.Errorf("multicastIP is not a multicast address")
		}
		conn, err = net.ListenMulticastUDP("udp4", nil, &net.UDPAddr{IP: multicastIP, Port: port})
		if err != nil {
			return nil, fmt.Errorf("failed to ListenMulticastUDP: %w", err)
		}
	} else {
		// IPv4 unicast or wildcard listen (broadcast received via WriteToUDP)
		bindIP := ip
		if bindIP == nil || bindIP.IsUnspecified() {
			bindIP = net.IPv4zero
		}
		conn, err = net.ListenUDP("udp4", &net.UDPAddr{IP: bindIP, Port: port})
		if err != nil {
			return nil, err
		}
	}

	// デフォルトタイムアウト
	if opt.DefaultTimeout == 0 {
		opt.DefaultTimeout = 30 * time.Second
	}
	// ReadDeadline 設定
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetReadDeadline(deadline)
	} else {
		conn.SetReadDeadline(time.Time{})
	}

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return &UDPConnection{UdpConn: conn, LocalAddr: localAddr, Port: port}, nil
}

// Close はソケットを閉じます
func (c *UDPConnection) Close() error {
	return c.UdpConn.Close()
}

// SendTo は指定先にデータを送信します
func (c *UDPConnection) SendTo(dstIP net.IP, data []byte) (int, error) {
	return c.UdpConn.WriteTo(data, &net.UDPAddr{IP: dstIP, Port: c.Port})
}

// bufferPool は受信バッファのプールです
var bufferPool = sync.Pool{
	New: func() interface{} { return make([]byte, 1500) },
}

// Receive は UDP パケットを受信し、送信元アドレスとデータを返します。
// 自送信パケットを除外し、コンテキストキャンセルに対応します。
func (c *UDPConnection) Receive(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	if deadline, ok := ctx.Deadline(); ok {
		c.UdpConn.SetReadDeadline(deadline)
	} else {
		c.UdpConn.SetReadDeadline(time.Time{})
	}

	type result struct {
		data []byte
		addr *net.UDPAddr
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		buf := bufferPool.Get().([]byte)
		defer bufferPool.Put(buf)
		n, addr, err := c.UdpConn.ReadFrom(buf)
		if err != nil {
			ch <- result{nil, nil, err}
			return
		}
		src := addr.(*net.UDPAddr)
		if src.IP.Equal(c.LocalAddr.IP) && src.Port == c.LocalAddr.Port {
			ch <- result{nil, nil, nil}
			return
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		ch <- result{data, src, nil}
	}()

	select {
	case <-ctx.Done():
		c.UdpConn.SetReadDeadline(time.Now())
		<-ch
		return nil, nil, ctx.Err()
	case res := <-ch:
		return res.data, res.addr, res.err
	}
}
