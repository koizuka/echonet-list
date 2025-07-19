package network

import (
	"context"
	"net"
	"os"
	"testing"
	"time"
)

func TestNetworkMonitorEnabled(t *testing.T) {
	// CI環境では net.Interfaces() が hang する可能性があるためスキップ
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("スキップ: ネットワーク監視テストはCI環境では実行しません")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := &NetworkMonitorConfig{Enabled: true}
	multicastIP := net.ParseIP("224.0.23.0")

	conn, err := CreateUDPConnection(ctx, nil, 3610, multicastIP, config)
	if err != nil {
		t.Fatalf("UDPConnection の作成に失敗: %v", err)
	}

	// 監視機能が有効になっていることを確認
	if !conn.IsNetworkMonitorEnabled() {
		t.Error("ネットワーク監視が有効になっていません")
	}

	// 必ず接続を閉じて goroutine を適切に終了させる
	err = conn.Close()
	if err != nil {
		t.Errorf("接続のクローズに失敗: %v", err)
	}

	// goroutine が終了するまで少し待機
	time.Sleep(100 * time.Millisecond)
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
