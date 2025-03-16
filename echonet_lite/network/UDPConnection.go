package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type UDPConnection struct {
	UdpConn   *net.UDPConn
	LocalAddr *net.UDPAddr
	Port      int
}

type UDPConnectionOptions struct {
	DefaultTimeout time.Duration
}

func CreateUDPConnection(ctx context.Context, ip net.IP, port int, broadcastIP net.IP, opt UDPConnectionOptions) (*UDPConnection, error) {
	// UDPソケットの作成
	addr := &net.UDPAddr{
		IP:   ip,
		Port: port,
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	// デフォルトのタイムアウト設定
	if opt.DefaultTimeout == 0 {
		opt.DefaultTimeout = 30 * time.Second
	}

	// contextからタイムアウトを設定（タイムアウトが設定されている場合のみ）
	deadline, ok := ctx.Deadline()
	if ok {
		// タイムアウトが設定されている場合は、それを使用
		if err := conn.SetReadDeadline(deadline); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("failed to set read deadline: %w", err)
		}
	} else {
		// タイムアウトが設定されていない場合は、タイムアウトを解除
		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("failed to clear read deadline: %w", err)
		}
	}

	localAddr, err := GetLocalUDPAddressFor(broadcastIP, port)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to get local address: %w", err)
	}

	return &UDPConnection{UdpConn: conn, LocalAddr: localAddr, Port: port}, nil
}

func (c *UDPConnection) Close() error {
	return c.UdpConn.Close()
}

func (c *UDPConnection) SendTo(ip net.IP, data []byte) (int, error) {
	dst := &net.UDPAddr{
		IP:   ip,
		Port: c.Port,
	}
	return c.UdpConn.WriteTo(data, dst)
}

// バッファのプール（メモリ割り当てを削減）
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 1500)
	},
}

func (c *UDPConnection) Receive(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	// コマンドライン対話用の場合はタイムアウトを設定しない
	// contextからタイムアウトを設定（タイムアウトが設定されている場合のみ）
	deadline, ok := ctx.Deadline()
	if ok {
		if err := c.UdpConn.SetReadDeadline(deadline); err != nil {
			return nil, nil, fmt.Errorf("failed to set read deadline: %w", err)
		}
	} else {
		// タイムアウトが設定されていない場合は、タイムアウトを解除
		if err := c.UdpConn.SetReadDeadline(time.Time{}); err != nil {
			return nil, nil, fmt.Errorf("failed to clear read deadline: %w", err)
		}
	}

	// contextのキャンセルを監視するgoroutineを起動
	readDone := make(chan struct{})
	var result []byte
	var udpAddr *net.UDPAddr
	var readErr error

	go func() {
		defer close(readDone)
		// プールからバッファを取得
		rawBuffer := bufferPool.Get()
		buffer := rawBuffer.([]byte)
		defer bufferPool.Put(rawBuffer) // 必ず実行されるようにdeferで登録

		n, addr, err := c.UdpConn.ReadFrom(buffer)
		if err != nil {
			readErr = err
			return
		}

		if c.LocalAddr.IP.Equal(addr.(*net.UDPAddr).IP) {
			// ローカルアドレスからのパケットは無視
			return
		}

		// 結果をコピー（受信データのスライスだけを返す）
		result = make([]byte, n)
		copy(result, buffer[:n])
		udpAddr = addr.(*net.UDPAddr)
	}()

	// contextのキャンセルとReadFromの完了を待つ
	select {
	case <-ctx.Done():
		// contextがキャンセルされた場合
		// ReadFromをキャンセルするためにSetReadDeadlineを呼び出す
		_ = c.UdpConn.SetReadDeadline(time.Now())
		<-readDone // ReadFromの終了を待つ
		return nil, nil, ctx.Err()
	case <-readDone:
		// ReadFromが完了した場合
		return result, udpAddr, readErr
	}
}
