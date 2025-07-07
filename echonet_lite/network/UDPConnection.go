package network

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

// UDPConnection は UDP ソケットを管理します
type UDPConnection struct {
	UdpConn        *net.UDPConn
	LocalAddr      *net.UDPAddr
	localIPs       []net.IP // ローカルインターフェースのIPリスト
	Port           int
	multicastIP    net.IP // マルチキャストIPアドレス
	mu             sync.RWMutex
	networkMonitor *NetworkMonitor
}

// NetworkMonitor はネットワークインターフェースの監視を行います
type NetworkMonitor struct {
	ctx          context.Context
	cancel       context.CancelFunc
	interfaces   []net.Interface
	interfacesMu sync.RWMutex
	done         chan struct{} // goroutine終了通知用
}

// NetworkMonitorConfig はネットワーク監視の設定を表します
type NetworkMonitorConfig struct {
	Enabled bool
}

// CreateUDPConnection は IPv4 の unicast と multicast (マルチキャスト) を受信対応します。
// ip が nil の場合はワイルドカード listen、multicastIP がブロードキャストかつIPv4の場合は broadcast として受信。
// multicastIP が真のマルチキャストかつIPv4の場合はグループ参加。
// ip または multicastIP がIPv6の場合はエラーになります。
func CreateUDPConnection(ctx context.Context, ip net.IP, port int, multicastIP net.IP, networkMonitorConfig *NetworkMonitorConfig) (*UDPConnection, error) {
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

	// ReadDeadline 設定
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetReadDeadline(deadline)
	} else {
		conn.SetReadDeadline(time.Time{})
	}

	// ローカルのIPv4アドレスを取得
	localIPs, err := GetLocalIPv4s()
	if err != nil {
		fmt.Printf("Warning: could not reliably determine local IPs for self-message filtering: %v\n", err)
		localIPs = []net.IP{} // エラー時も空スライスで続行
	}
	// Listen したアドレスが Unspecified でない場合、それもリストに追加する（フォールバック）
	listenAddrIP := conn.LocalAddr().(*net.UDPAddr).IP
	if listenAddrIP.To4() != nil && !listenAddrIP.IsUnspecified() {
		isAlreadyAdded := false
		for _, lip := range localIPs {
			if lip.Equal(listenAddrIP) {
				isAlreadyAdded = true
				break
			}
		}
		if !isAlreadyAdded {
			localIPs = append(localIPs, listenAddrIP)
		}
	}

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	udpConn := &UDPConnection{
		UdpConn:     conn,
		LocalAddr:   localAddr,
		localIPs:    localIPs,
		Port:        port,
		multicastIP: multicastIP,
	}

	// ネットワーク監視機能を初期化
	if networkMonitorConfig != nil && networkMonitorConfig.Enabled {
		err := udpConn.initNetworkMonitor(ctx)
		if err != nil {
			slog.Warn("ネットワーク監視の初期化に失敗", "err", err)
		}
	}

	return udpConn, nil
}

// isSelfPacket は指定されたアドレスが自身のいずれかのローカルIPとポートから送信されたものかを確認します
func (c *UDPConnection) isSelfPacket(src *net.UDPAddr) bool {
	if src == nil {
		return false
	}
	// まずポート番号を確認
	if src.Port != c.Port {
		return false
	}
	// 次にIPアドレスがローカルIPリストに含まれるか確認
	for _, localIP := range c.localIPs {
		if src.IP.Equal(localIP) {
			return true
		}
	}
	return false
}

// IsLocalIP は指定されたIPアドレスが自身のローカルIPのいずれかと一致するかを確認します
func (c *UDPConnection) IsLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, localIP := range c.localIPs {
		if ip.Equal(localIP) {
			return true
		}
	}
	return false
}

// Close はソケットを閉じます
func (c *UDPConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// ネットワーク監視を停止
	if c.networkMonitor != nil {
		c.stopNetworkMonitor()
	}

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
		if c.isSelfPacket(src) {
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

// initNetworkMonitor はネットワーク監視機能を初期化します
func (c *UDPConnection) initNetworkMonitor(ctx context.Context) error {
	monitorCtx, cancel := context.WithCancel(ctx)

	c.networkMonitor = &NetworkMonitor{
		ctx:        monitorCtx,
		cancel:     cancel,
		interfaces: []net.Interface{},
		done:       make(chan struct{}),
	}

	// 初期のネットワークインターフェース情報を取得
	if err := c.networkMonitor.updateNetworkInterfaces(); err != nil {
		slog.Warn("ネットワークインターフェース情報の取得に失敗", "err", err)
		// エラーでも継続（空の interface リストで開始）
	}

	// ネットワーク監視ループを開始
	go c.networkMonitorLoop()

	slog.Info("ネットワーク監視が開始されました")

	return nil
}

// stopNetworkMonitor はネットワーク監視機能を停止します
func (c *UDPConnection) stopNetworkMonitor() {
	if c.networkMonitor != nil && c.networkMonitor.cancel != nil {
		c.networkMonitor.cancel()
		// goroutineの終了を待機
		<-c.networkMonitor.done
		c.networkMonitor = nil
		slog.Info("ネットワーク監視が停止されました")
	}
}

// networkMonitorLoop はネットワーク監視のメインループです
func (c *UDPConnection) networkMonitorLoop() {
	c.mu.RLock()
	networkMonitor := c.networkMonitor
	c.mu.RUnlock()

	if networkMonitor == nil {
		return
	}

	// 終了時にdoneチャンネルを必ずcloseする
	defer close(networkMonitor.done)

	networkMonitorTicker := time.NewTicker(10 * time.Second) // ネットワーク監視は10秒間隔
	defer networkMonitorTicker.Stop()

	for {
		select {
		case <-networkMonitor.ctx.Done():
			slog.Debug("ネットワーク監視ループを終了します")
			return

		case <-networkMonitorTicker.C:
			c.monitorNetworkChanges()
		}
	}
}

// monitorNetworkChanges はネットワークインターフェースの変更を監視します
func (c *UDPConnection) monitorNetworkChanges() {
	c.mu.RLock()
	networkMonitor := c.networkMonitor
	c.mu.RUnlock()

	if networkMonitor == nil {
		return
	}

	// 現在のネットワークインターフェース情報を取得
	currentInterfaces, err := net.Interfaces()
	if err != nil {
		slog.Warn("ネットワークインターフェース情報の取得に失敗", "err", err)
		// 次回の監視に期待し、エラーでも継続
		return
	}

	networkMonitor.interfacesMu.Lock()
	previousInterfaces := networkMonitor.interfaces
	networkMonitor.interfacesMu.Unlock()

	// インターフェースの変更をチェック
	if c.hasNetworkChanged(previousInterfaces, currentInterfaces) {
		slog.Info("ネットワークインターフェースの変更を検出しました")

		// ネットワークインターフェース情報を更新
		networkMonitor.interfacesMu.Lock()
		networkMonitor.interfaces = currentInterfaces
		networkMonitor.interfacesMu.Unlock()

		// ローカルIPアドレスを再取得
		newLocalIPs, err := GetLocalIPv4s()
		if err != nil {
			slog.Warn("ローカルIPアドレスの再取得に失敗", "err", err)
			// エラーでも既存のIPリストを保持して継続
		} else {
			c.mu.Lock()
			c.localIPs = newLocalIPs
			c.mu.Unlock()
			slog.Debug("ローカルIPアドレスを更新しました", "count", len(newLocalIPs))
		}
	}
}

// hasNetworkChanged はネットワークインターフェースが変更されたかをチェックします
func (c *UDPConnection) hasNetworkChanged(previous, current []net.Interface) bool {
	if len(previous) != len(current) {
		return true
	}

	// インターフェース名とフラグの変更をチェック
	prevMap := make(map[string]net.Flags)
	for _, iface := range previous {
		prevMap[iface.Name] = iface.Flags
	}

	for _, iface := range current {
		if prevFlags, exists := prevMap[iface.Name]; !exists || prevFlags != iface.Flags {
			return true
		}
	}

	return false
}

// updateNetworkInterfaces はネットワークインターフェース情報を更新します
func (nm *NetworkMonitor) updateNetworkInterfaces() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	nm.interfacesMu.Lock()
	nm.interfaces = interfaces
	nm.interfacesMu.Unlock()

	return nil
}

// IsNetworkMonitorEnabled はネットワーク監視が有効かどうかを返します
func (c *UDPConnection) IsNetworkMonitorEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.networkMonitor != nil
}
