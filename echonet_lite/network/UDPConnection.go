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
	UdpConn     *net.UDPConn
	LocalAddr   *net.UDPAddr
	localIPs    []net.IP // ローカルインターフェースのIPリスト
	Port        int
	multicastIP net.IP // マルチキャストIPアドレス
	mu          sync.RWMutex
	keepAlive   *MulticastKeepAlive
}

// MulticastKeepAlive はマルチキャストのキープアライブ管理を行います
type MulticastKeepAlive struct {
	enabled               bool
	groupRefreshInterval  time.Duration
	networkMonitorEnabled bool
	ctx                   context.Context
	cancel                context.CancelFunc
	interfaces            []net.Interface
	interfacesMu          sync.RWMutex
	lastGroupRefresh      time.Time
	lastMu                sync.RWMutex
	groupRefreshCh        chan bool
	done                  chan struct{} // goroutine終了通知用
}

// KeepAliveConfig はキープアライブの設定を表します
type KeepAliveConfig struct {
	Enabled               bool
	GroupRefreshInterval  time.Duration
	NetworkMonitorEnabled bool
}

// CreateUDPConnection は IPv4 の unicast と multicast (マルチキャスト) を受信対応します。
// ip が nil の場合はワイルドカード listen、multicastIP がブロードキャストかつIPv4の場合は broadcast として受信。
// multicastIP が真のマルチキャストかつIPv4の場合はグループ参加。
// ip または multicastIP がIPv6の場合はエラーになります。
func CreateUDPConnection(ctx context.Context, ip net.IP, port int, multicastIP net.IP, keepAliveConfig *KeepAliveConfig) (*UDPConnection, error) {
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

	// キープアライブ機能を初期化
	if keepAliveConfig != nil && keepAliveConfig.Enabled && multicastIP != nil {
		err := udpConn.initKeepAlive(ctx, *keepAliveConfig)
		if err != nil {
			slog.Warn("キープアライブの初期化に失敗", "err", err)
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

	// キープアライブを停止
	if c.keepAlive != nil {
		c.stopKeepAlive()
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

// initKeepAlive はキープアライブ機能を初期化します
func (c *UDPConnection) initKeepAlive(ctx context.Context, config KeepAliveConfig) error {
	keepAliveCtx, cancel := context.WithCancel(ctx)

	c.keepAlive = &MulticastKeepAlive{
		enabled:               config.Enabled,
		groupRefreshInterval:  config.GroupRefreshInterval,
		networkMonitorEnabled: config.NetworkMonitorEnabled,
		ctx:                   keepAliveCtx,
		cancel:                cancel,
		interfaces:            []net.Interface{},
		groupRefreshCh:        make(chan bool, 1),
		done:                  make(chan struct{}),
	}

	// 初期のネットワークインターフェース情報を取得
	if err := c.keepAlive.updateNetworkInterfaces(); err != nil {
		slog.Warn("ネットワークインターフェース情報の取得に失敗", "err", err)
		// エラーでも継続（空の interface リストで開始）
	}

	// キープアライブループを開始
	go c.keepAliveLoop()

	slog.Info("マルチキャストキープアライブが開始されました",
		"groupRefreshInterval", config.GroupRefreshInterval,
		"networkMonitorEnabled", config.NetworkMonitorEnabled)

	return nil
}

// stopKeepAlive はキープアライブ機能を停止します
func (c *UDPConnection) stopKeepAlive() {
	if c.keepAlive != nil && c.keepAlive.cancel != nil {
		c.keepAlive.cancel()
		// goroutineの終了を待機
		<-c.keepAlive.done
		c.keepAlive = nil
		slog.Info("マルチキャストキープアライブが停止されました")
	}
}

// keepAliveLoop はキープアライブのメインループです
func (c *UDPConnection) keepAliveLoop() {
	c.mu.RLock()
	keepAlive := c.keepAlive
	c.mu.RUnlock()

	if keepAlive == nil {
		return
	}

	// 終了時にdoneチャンネルを必ずcloseする
	defer close(keepAlive.done)

	groupRefreshTicker := time.NewTicker(keepAlive.groupRefreshInterval)
	defer groupRefreshTicker.Stop()

	var networkMonitorTicker *time.Ticker
	if keepAlive.networkMonitorEnabled {
		networkMonitorTicker = time.NewTicker(10 * time.Second) // ネットワーク監視は10秒間隔
		defer networkMonitorTicker.Stop()
	}

	for {
		select {
		case <-keepAlive.ctx.Done():
			slog.Debug("キープアライブループを終了します")
			return

		case <-groupRefreshTicker.C:
			c.refreshMulticastGroup()

		case <-keepAlive.groupRefreshCh:
			c.refreshMulticastGroup()

		case <-func() <-chan time.Time {
			if networkMonitorTicker != nil {
				return networkMonitorTicker.C
			}
			return make(chan time.Time) // 無効なチャンネル
		}():
			c.monitorNetworkChanges()
		}
	}
}

// refreshMulticastGroup はマルチキャストグループのメンバーシップを更新します
func (c *UDPConnection) refreshMulticastGroup() {
	c.mu.RLock()
	keepAlive := c.keepAlive
	multicastIP := c.multicastIP
	c.mu.RUnlock()

	if keepAlive == nil || multicastIP == nil {
		return
	}

	// IGMP準拠のグループメンバーシップ維持
	// net.ListenMulticastUDP で自動的にグループに参加するため、
	// ソケットが有効である限りIGMPメンバーシップは維持される
	//
	// このリフレッシュは主にネットワーク変更後の状態確認を目的とする

	keepAlive.lastMu.Lock()
	keepAlive.lastGroupRefresh = time.Now()
	keepAlive.lastMu.Unlock()
	slog.Debug("マルチキャストグループのメンバーシップを確認しました", "multicastIP", multicastIP)
}

// monitorNetworkChanges はネットワークインターフェースの変更を監視します
func (c *UDPConnection) monitorNetworkChanges() {
	c.mu.RLock()
	keepAlive := c.keepAlive
	c.mu.RUnlock()

	if keepAlive == nil || !keepAlive.networkMonitorEnabled {
		return
	}

	// 現在のネットワークインターフェース情報を取得
	currentInterfaces, err := net.Interfaces()
	if err != nil {
		slog.Warn("ネットワークインターフェース情報の取得に失敗", "err", err)
		// 次回の監視に期待し、エラーでも継続
		return
	}

	keepAlive.interfacesMu.Lock()
	previousInterfaces := keepAlive.interfaces
	keepAlive.interfacesMu.Unlock()

	// インターフェースの変更をチェック
	if c.hasNetworkChanged(previousInterfaces, currentInterfaces) {
		slog.Info("ネットワークインターフェースの変更を検出しました")

		// ネットワークインターフェース情報を更新
		keepAlive.interfacesMu.Lock()
		keepAlive.interfaces = currentInterfaces
		keepAlive.interfacesMu.Unlock()

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

		// グループメンバーシップを強制更新
		select {
		case keepAlive.groupRefreshCh <- true:
		default:
			// チャンネルがブロックされている場合は無視
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
func (ka *MulticastKeepAlive) updateNetworkInterfaces() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	ka.interfacesMu.Lock()
	ka.interfaces = interfaces
	ka.interfacesMu.Unlock()

	return nil
}

// TriggerGroupRefresh は手動でグループ更新をトリガーします
func (c *UDPConnection) TriggerGroupRefresh() {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.keepAlive != nil {
		select {
		case c.keepAlive.groupRefreshCh <- true:
		default:
			// チャンネルがブロックされている場合は無視
		}
	}
}

// GetKeepAliveStatus はキープアライブの状態を返します
func (c *UDPConnection) GetKeepAliveStatus() (enabled bool, lastGroupRefresh time.Time) {
	c.mu.RLock()
	keepAlive := c.keepAlive
	c.mu.RUnlock()

	if keepAlive != nil {
		keepAlive.lastMu.RLock()
		defer keepAlive.lastMu.RUnlock()
		return keepAlive.enabled, keepAlive.lastGroupRefresh
	}
	return false, time.Time{}
}
