package network

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNetworkMonitorEnabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	config := &NetworkMonitorConfig{Enabled: true}
	multicastIP := net.ParseIP("224.0.23.0")

	conn, err := CreateUDPConnection(ctx, nil, 3610, multicastIP, config)
	if err != nil {
		t.Fatalf("UDPConnection の作成に失敗: %v", err)
	}
	defer conn.Close()

	if !conn.IsNetworkMonitorEnabled() {
		t.Error("ネットワーク監視が有効になっていません")
	}
}

func TestNetworkMonitorDisabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	multicastIP := net.ParseIP("224.0.23.0")

	// Enabled: false の場合
	config := &NetworkMonitorConfig{Enabled: false}
	conn, err := CreateUDPConnection(ctx, nil, 3610, multicastIP, config)
	if err != nil {
		t.Fatalf("UDPConnection の作成に失敗: %v", err)
	}
	defer conn.Close()

	if conn.IsNetworkMonitorEnabled() {
		t.Error("ネットワーク監視が無効になっていません")
	}

	// nil config の場合
	conn2, err := CreateUDPConnection(ctx, nil, 3611, multicastIP, nil)
	if err != nil {
		t.Fatalf("UDPConnection の作成に失敗: %v", err)
	}
	defer conn2.Close()

	if conn2.IsNetworkMonitorEnabled() {
		t.Error("nil config でネットワーク監視が無効になっていません")
	}
}
