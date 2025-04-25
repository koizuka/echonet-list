package network

import (
	"fmt"
	"net"
)

// GetIPv4BroadcastIP は、ローカルネットワークのIPv4ブロードキャストアドレスを自動的に検出します
func GetIPv4BroadcastIP() net.IP {
	// デフォルトのブロードキャストアドレス（見つからない場合に使用）
	defaultBroadcast := net.ParseIP("255.255.255.255")

	// すべてのネットワークインターフェースを取得
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("ネットワークインターフェースの取得に失敗しました: %v\n", err)
		return defaultBroadcast
	}

	// 各インターフェースを処理
	for _, iface := range interfaces {
		// ループバックインターフェースをスキップ
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// インターフェースが起動していない場合はスキップ
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// インターフェースのアドレスを取得
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			// IPネットワークアドレスの場合のみ処理
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// IPv4アドレスのみを処理
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}

			// ブロードキャストアドレスを計算
			// ブロードキャストアドレス = IPアドレス | (^サブネットマスク)
			broadcast := net.IP(make([]byte, 4))
			for i := range ip4 {
				broadcast[i] = ip4[i] | ^ipnet.Mask[i]
			}

			fmt.Printf("インターフェース %s のIPv4ブロードキャストアドレス: %v\n", iface.Name, broadcast)
			return broadcast
		}
	}

	// 適切なインターフェースが見つからない場合はデフォルトを返す
	return defaultBroadcast
}

// GetLocalUDPAddressFor は、指定された宛先IPアドレスとポートに対するローカルアドレスを取得します
func GetLocalUDPAddressFor(ip net.IP, port int) (*net.UDPAddr, error) {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: ip, Port: port})
	if err != nil {
		return nil, err
	}
	defer func(conn *net.UDPConn) {
		_ = conn.Close()
	}(conn)
	return conn.LocalAddr().(*net.UDPAddr), nil
}

// GetLocalIPv4s はローカルマシンの非ループバックIPv4アドレスのリストを取得します
func GetLocalIPv4s() ([]net.IP, error) {
	localIPs := []net.IP{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}
	for _, i := range ifaces {
		// インターフェースがダウンしている、またはループバックの場合はスキップ
		if (i.Flags&net.FlagUp == 0) || (i.Flags&net.FlagLoopback != 0) {
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			// エラーが発生しても他のインターフェースの処理を続ける
			fmt.Printf("Warning: failed to get addresses for interface %s: %v\n", i.Name, err)
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// IPv4 アドレスのみを対象とする
			if ip != nil && ip.To4() != nil {
				localIPs = append(localIPs, ip)
			}
		}
	}
	if len(localIPs) == 0 {
		// 適切なIPが見つからなかった場合は警告を出す
		fmt.Println("Warning: no suitable local IPv4 addresses found.")
	}
	return localIPs, nil
}
