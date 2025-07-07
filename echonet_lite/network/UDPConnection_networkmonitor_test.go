package network

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNetworkMonitorConfig(t *testing.T) {
	// テスト用のネットワーク監視設定
	config := &NetworkMonitorConfig{
		Enabled: true,
	}

	// テスト用のコンテキスト
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// マルチキャストIPアドレス
	multicastIP := net.ParseIP("224.0.23.0")

	// UDPConnectionをキープアライブ付きで作成
	conn, err := CreateUDPConnection(ctx, nil, 3610, multicastIP, config)
	if err != nil {
		t.Fatalf("UDPConnection の作成に失敗: %v", err)
	}
	defer conn.Close()

	// ネットワーク監視状態をチェック
	enabled := conn.IsNetworkMonitorEnabled()
	if !enabled {
		t.Error("ネットワーク監視が有効になっていません")
	}
}

func TestNetworkMonitorDisabled(t *testing.T) {
	// ネットワーク監視を無効にした設定
	config := &NetworkMonitorConfig{
		Enabled: false,
	}

	// テスト用のコンテキスト
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// マルチキャストIPアドレス
	multicastIP := net.ParseIP("224.0.23.0")

	// UDPConnectionをキープアライブなしで作成
	conn, err := CreateUDPConnection(ctx, nil, 3610, multicastIP, config)
	if err != nil {
		t.Fatalf("UDPConnection の作成に失敗: %v", err)
	}
	defer conn.Close()

	// ネットワーク監視状態をチェック
	enabled := conn.IsNetworkMonitorEnabled()
	if enabled {
		t.Error("ネットワーク監視が無効になっていません")
	}
}

func TestNetworkMonitorNilConfig(t *testing.T) {
	// テスト用のコンテキスト
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// マルチキャストIPアドレス
	multicastIP := net.ParseIP("224.0.23.0")

	// UDPConnectionをnil設定で作成
	conn, err := CreateUDPConnection(ctx, nil, 3610, multicastIP, nil)
	if err != nil {
		t.Fatalf("UDPConnection の作成に失敗: %v", err)
	}
	defer conn.Close()

	// ネットワーク監視状態をチェック
	enabled := conn.IsNetworkMonitorEnabled()
	if enabled {
		t.Error("ネットワーク監視が無効になっていません")
	}
}

func TestCreateUDPConnectionBackwardCompatibility(t *testing.T) {
	// 既存のCreateUDPConnection関数の互換性をテスト
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	multicastIP := net.ParseIP("224.0.23.0")

	// 既存の関数で作成
	conn, err := CreateUDPConnection(ctx, nil, 3610, multicastIP, nil)
	if err != nil {
		t.Fatalf("UDPConnection の作成に失敗: %v", err)
	}
	defer conn.Close()

	// ネットワーク監視が無効であることを確認
	enabled := conn.IsNetworkMonitorEnabled()
	if enabled {
		t.Error("デフォルトでネットワーク監視が有効になっています")
	}
}
