package network

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestKeepAliveConfig(t *testing.T) {
	// テスト用のキープアライブ設定
	config := &KeepAliveConfig{
		Enabled:               true,
		HeartbeatInterval:     1 * time.Second,
		GroupRefreshInterval:  2 * time.Second,
		NetworkMonitorEnabled: true,
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

	// キープアライブ状態をチェック
	enabled, lastHeartbeat, lastGroupRefresh := conn.GetKeepAliveStatus()
	if !enabled {
		t.Error("キープアライブが有効になっていません")
	}

	// 少し待機してからハートビートとグループ更新をテスト
	time.Sleep(100 * time.Millisecond)

	// 手動でハートビートをトリガー
	conn.TriggerHeartbeat()
	time.Sleep(100 * time.Millisecond)

	// 手動でグループ更新をトリガー
	conn.TriggerGroupRefresh()
	time.Sleep(100 * time.Millisecond)

	// 状態が更新されているかチェック
	_, newLastHeartbeat, newLastGroupRefresh := conn.GetKeepAliveStatus()
	if !newLastHeartbeat.After(lastHeartbeat) {
		t.Error("ハートビートが更新されていません")
	}
	if !newLastGroupRefresh.After(lastGroupRefresh) {
		t.Error("グループ更新が実行されていません")
	}
}

func TestKeepAliveDisabled(t *testing.T) {
	// キープアライブを無効にした設定
	config := &KeepAliveConfig{
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

	// キープアライブ状態をチェック
	enabled, _, _ := conn.GetKeepAliveStatus()
	if enabled {
		t.Error("キープアライブが無効になっていません")
	}
}

func TestKeepAliveNilConfig(t *testing.T) {
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

	// キープアライブ状態をチェック
	enabled, _, _ := conn.GetKeepAliveStatus()
	if enabled {
		t.Error("キープアライブが無効になっていません")
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

	// キープアライブが無効であることを確認
	enabled, _, _ := conn.GetKeepAliveStatus()
	if enabled {
		t.Error("デフォルトでキープアライブが有効になっています")
	}
}
