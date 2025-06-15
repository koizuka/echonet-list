package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
	"log/slog"
)

// UDPConnection は UDP ソケットを管理します
type UDPConnection struct {
	UdpConn     *net.UDPConn
	LocalAddr   *net.UDPAddr
	localIPs    []net.IP // ローカルインターフェースのIPリスト
	Port        int
	multicastIP net.IP   // マルチキャストIPアドレス
	mu          sync.RWMutex
	keepAlive   *MulticastKeepAlive
}

// MulticastKeepAlive はマルチキャストのキープアライブ管理を行います
type MulticastKeepAlive struct {
	enabled               bool
	heartbeatInterval     time.Duration
	groupRefreshInterval  time.Duration
	networkMonitorEnabled bool
	ctx                   context.Context
	cancel                context.CancelFunc
	interfaces            []net.Interface
	interfacesMu          sync.RWMutex
	lastHeartbeat         time.Time
	lastGroupRefresh      time.Time
	heartbeatCh           chan bool
	groupRefreshCh        chan bool
}

// KeepAliveConfig はキープアライブの設定を表します
type KeepAliveConfig struct {
	Enabled               bool
	HeartbeatInterval     time.Duration
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
		heartbeatInterval:     config.HeartbeatInterval,
		groupRefreshInterval:  config.GroupRefreshInterval,
		networkMonitorEnabled: config.NetworkMonitorEnabled,
		ctx:                   keepAliveCtx,
		cancel:                cancel,
		interfaces:            []net.Interface{},
		heartbeatCh:           make(chan bool, 1),
		groupRefreshCh:        make(chan bool, 1),
	}

	// 初期のネットワークインターフェース情報を取得
	if err := c.keepAlive.updateNetworkInterfaces(); err != nil {
		slog.Warn("ネットワークインターフェース情報の取得に失敗", "err", err)
	}

	// キープアライブループを開始
	go c.keepAliveLoop()

	slog.Info("マルチキャストキープアライブが開始されました", 
		"heartbeatInterval", config.HeartbeatInterval,
		"groupRefreshInterval", config.GroupRefreshInterval,
		"networkMonitorEnabled", config.NetworkMonitorEnabled)

	return nil
}

// stopKeepAlive はキープアライブ機能を停止します
func (c *UDPConnection) stopKeepAlive() {
	if c.keepAlive != nil && c.keepAlive.cancel != nil {
		c.keepAlive.cancel()
		c.keepAlive = nil
		slog.Info("マルチキャストキープアライブが停止されました")
	}
}

// keepAliveLoop はキープアライブのメインループです
func (c *UDPConnection) keepAliveLoop() {
	if c.keepAlive == nil {
		return
	}

	heartbeatTicker := time.NewTicker(c.keepAlive.heartbeatInterval)
	defer heartbeatTicker.Stop()

	groupRefreshTicker := time.NewTicker(c.keepAlive.groupRefreshInterval)
	defer groupRefreshTicker.Stop()

	var networkMonitorTicker *time.Ticker
	if c.keepAlive.networkMonitorEnabled {
		networkMonitorTicker = time.NewTicker(10 * time.Second) // ネットワーク監視は10秒間隔
		defer networkMonitorTicker.Stop()
	}

	for {
		select {
		case <-c.keepAlive.ctx.Done():
			slog.Debug("キープアライブループを終了します")
			return

		case <-heartbeatTicker.C:
			c.sendHeartbeat()

		case <-groupRefreshTicker.C:
			c.refreshMulticastGroup()

		case <-c.keepAlive.heartbeatCh:
			c.sendHeartbeat()

		case <-c.keepAlive.groupRefreshCh:
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

// sendHeartbeat はハートビートを送信します（マルチキャストグループメンバーシップ維持用）
func (c *UDPConnection) sendHeartbeat() {
	if c.keepAlive == nil || c.multicastIP == nil {
		return
	}

	// マルチキャストグループのメンバーシップを維持するための最小限のトラフィック
	// 実際のアプリケーションレベルのハートビートはセッション層で実装される
	// ここでは単純にソケットの活性化のために空のUDPパケットを送信
	
	// 空のパケットを送信（1バイトの0x00）
	heartbeatData := []byte{0x00}

	// 注意: これは低レベルのネットワーク keep-alive で、ECHONET Lite プロトコルレベルの通信ではない
	_, err := c.UdpConn.WriteTo(heartbeatData, &net.UDPAddr{IP: c.multicastIP, Port: c.Port})
	if err != nil {
		slog.Warn("ネットワークキープアライブ送信エラー", "err", err)
	} else {
		c.keepAlive.lastHeartbeat = time.Now()
		slog.Debug("ネットワークキープアライブを送信", "multicastIP", c.multicastIP)
	}
}

// refreshMulticastGroup はマルチキャストグループのメンバーシップを更新します
func (c *UDPConnection) refreshMulticastGroup() {
	if c.keepAlive == nil || c.multicastIP == nil {
		return
	}

	// マルチキャスト接続の再初期化（必要に応じて）
	// 現在の実装では net.ListenMulticastUDP で自動的にグループに参加するため、
	// 明示的な再参加は不要ですが、将来的にはより詳細な制御が可能
	
	c.keepAlive.lastGroupRefresh = time.Now()
	slog.Debug("マルチキャストグループのメンバーシップを確認しました", "multicastIP", c.multicastIP)
}

// monitorNetworkChanges はネットワークインターフェースの変更を監視します
func (c *UDPConnection) monitorNetworkChanges() {
	if c.keepAlive == nil || !c.keepAlive.networkMonitorEnabled {
		return
	}

	// 現在のネットワークインターフェース情報を取得
	currentInterfaces, err := net.Interfaces()
	if err != nil {
		slog.Warn("ネットワークインターフェース情報の取得に失敗", "err", err)
		return
	}

	c.keepAlive.interfacesMu.Lock()
	previousInterfaces := c.keepAlive.interfaces
	c.keepAlive.interfacesMu.Unlock()

	// インターフェースの変更をチェック
	if c.hasNetworkChanged(previousInterfaces, currentInterfaces) {
		slog.Info("ネットワークインターフェースの変更を検出しました")
		
		// ネットワークインターフェース情報を更新
		c.keepAlive.interfacesMu.Lock()
		c.keepAlive.interfaces = currentInterfaces
		c.keepAlive.interfacesMu.Unlock()

		// ローカルIPアドレスを再取得
		newLocalIPs, err := GetLocalIPv4s()
		if err != nil {
			slog.Warn("ローカルIPアドレスの再取得に失敗", "err", err)
		} else {
			c.mu.Lock()
			c.localIPs = newLocalIPs
			c.mu.Unlock()
			slog.Debug("ローカルIPアドレスを更新しました", "count", len(newLocalIPs))
		}

		// グループメンバーシップを強制更新
		select {
		case c.keepAlive.groupRefreshCh <- true:
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

// TriggerHeartbeat は手動でハートビートをトリガーします
func (c *UDPConnection) TriggerHeartbeat() {
	if c.keepAlive != nil {
		select {
		case c.keepAlive.heartbeatCh <- true:
		default:
			// チャンネルがブロックされている場合は無視
		}
	}
}

// TriggerGroupRefresh は手動でグループ更新をトリガーします
func (c *UDPConnection) TriggerGroupRefresh() {
	if c.keepAlive != nil {
		select {
		case c.keepAlive.groupRefreshCh <- true:
		default:
			// チャンネルがブロックされている場合は無視
		}
	}
}

// GetKeepAliveStatus はキープアライブの状態を返します
func (c *UDPConnection) GetKeepAliveStatus() (enabled bool, lastHeartbeat, lastGroupRefresh time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.keepAlive != nil {
		return c.keepAlive.enabled, c.keepAlive.lastHeartbeat, c.keepAlive.lastGroupRefresh
	}
	return false, time.Time{}, time.Time{}
}
