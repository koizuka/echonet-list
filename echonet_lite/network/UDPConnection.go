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

// CreateUDPConnection は IPv4/IPv6 両対応で UDP ソケットを作成します。
// broadcastIP がマルチキャストアドレスの場合は ListenMulticastUDP を使います。
func CreateUDPConnection(ctx context.Context, ip net.IP, port int, broadcastIP net.IP, opt UDPConnectionOptions) (*UDPConnection, error) {
	// ネットワーク種別を判定 (broadcastIP優先)
	network := "udp4"
	if broadcastIP != nil {
		if broadcastIP.To4() == nil {
			network = "udp6"
		}
	} else if ip.To4() == nil {
		network = "udp6"
	}

	var conn *net.UDPConn
	var err error
	var iface *net.Interface

	if broadcastIP != nil && broadcastIP.IsMulticast() {
		if network == "udp6" {
			// IPv6マルチキャスト受信: 利用可能なインターフェースを選択し、グループに参加
			ifaces, _ := net.Interfaces()
			// iface は外側で宣言済み
			for _, i := range ifaces {
				if i.Flags&net.FlagUp != 0 && i.Flags&net.FlagMulticast != 0 {
					iface = &i
					break
				}
			}
			if iface == nil {
				return nil, fmt.Errorf("no suitable interface for IPv6 multicast")
			}
			group := &net.UDPAddr{IP: broadcastIP, Port: port, Zone: iface.Name}
			conn, err = net.ListenMulticastUDP(network, iface, group)
		} else {
			group := &net.UDPAddr{IP: broadcastIP, Port: port}
			conn, err = net.ListenMulticastUDP(network, nil, group)
		}
	} else {
		laddr := &net.UDPAddr{IP: ip, Port: port}
		conn, err = net.ListenUDP(network, laddr)
	}
	if err != nil {
		return nil, err
	}

	if opt.DefaultTimeout == 0 {
		opt.DefaultTimeout = 30 * time.Second
	}

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetReadDeadline(deadline); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to set read deadline: %w", err)
		}
	} else {
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to clear read deadline: %w", err)
		}
	}

	// LocalAddr を設定 (IPv6 マルチキャスト時はゾーン付き)
	var localAddr *net.UDPAddr
	if broadcastIP != nil && broadcastIP.IsMulticast() && network == "udp6" && iface != nil {
		localAddr = &net.UDPAddr{IP: net.IPv6unspecified, Port: port, Zone: iface.Name}
	} else {
		localAddr = conn.LocalAddr().(*net.UDPAddr)
	}
	return &UDPConnection{UdpConn: conn, LocalAddr: localAddr, Port: port}, nil
}

// Close はソケットを閉じます
func (c *UDPConnection) Close() error {
	return c.UdpConn.Close()
}

// SendTo は指定先にデータを送信します
func (c *UDPConnection) SendTo(ip net.IP, data []byte) (int, error) {
	dst := &net.UDPAddr{IP: ip, Port: c.Port}
	return c.UdpConn.WriteTo(data, dst)
}

// bufferPool は受信バッファのプールです
var bufferPool = sync.Pool{
	New: func() interface{} { return make([]byte, 1500) },
}

// Receive は UDP パケットを受信し、送信元アドレスとデータを返します。
// コンテキストキャンセルで ReadDeadline をリセットして停止します。
func (c *UDPConnection) Receive(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if err := c.UdpConn.SetReadDeadline(deadline); err != nil {
			return nil, nil, fmt.Errorf("failed to set read deadline: %w", err)
		}
	} else {
		if err := c.UdpConn.SetReadDeadline(time.Time{}); err != nil {
			return nil, nil, fmt.Errorf("failed to clear read deadline: %w", err)
		}
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
		udpAddr := addr.(*net.UDPAddr)
		if udpAddr.IP.Equal(c.LocalAddr.IP) && udpAddr.Port == c.LocalAddr.Port {
			ch <- result{nil, nil, nil}
			return
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		ch <- result{data, udpAddr, nil}
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
