package handler

import (
	"context"
	"echonet-list/echonet_lite"
	"net"
	"testing"
	"time"
)

// TestNewECHONETLiteHandler_ControllerInitialization はコントローラーの初期化をテストします
func TestNewECHONETLiteHandler_ControllerInitialization(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// テスト用のIPアドレスを設定（ローカルループバック）
	testIP := net.ParseIP("127.0.0.1")

	options := ECHONETLieHandlerOptions{
		IP:               testIP,
		Debug:            false,
		ManufacturerCode: "Experimental",
		UniqueIdentifier: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d},
	}

	handler, err := NewECHONETLiteHandler(ctx, options)
	if err != nil {
		t.Fatalf("Failed to create ECHONETLiteHandler: %v", err)
	}
	defer func() {
		if err := handler.Close(); err != nil {
			t.Errorf("Failed to close handler: %v", err)
		}
	}()

	// コントローラーのEOJを取得
	controllerEOJ := echonet_lite.MakeEOJ(echonet_lite.Controller_ClassCode, 1)

	// Status Announcement Property Map (EPC=0x9d) が設定されていることを確認
	announcementProp, ok := handler.comm.localDevices.Get(controllerEOJ, echonet_lite.EPCStatusAnnouncementPropertyMap)
	if !ok {
		t.Fatalf("Expected Status Announcement Property Map (0x9d) to be set for controller, but it's not")
	}

	// PropertyMapをデコード
	propMap := echonet_lite.DecodePropertyMap(announcementProp.EDT)
	if propMap == nil {
		t.Fatalf("Failed to decode Status Announcement Property Map")
	}

	// 設置場所 (EPC=0x81) がアナウンス対象に含まれていることを確認
	if !propMap.Has(echonet_lite.EPCInstallationLocation) {
		t.Errorf("Expected Installation Location (0x81) to be included in Status Announcement Property Map, but it's not")
	}

	// Get Property Map (EPC=0x9f) に Status Announcement Property Map (EPC=0x9d) が含まれていることを確認
	getPropMapProperty, ok := handler.comm.localDevices.Get(controllerEOJ, echonet_lite.EPCGetPropertyMap)
	if !ok {
		t.Fatalf("Expected Get Property Map (0x9f) to be set for controller, but it's not")
	}

	getPropertyMap := echonet_lite.DecodePropertyMap(getPropMapProperty.EDT)
	if getPropertyMap == nil {
		t.Fatalf("Failed to decode Get Property Map")
	}

	if !getPropertyMap.Has(echonet_lite.EPCStatusAnnouncementPropertyMap) {
		t.Errorf("Expected Status Announcement Property Map (0x9d) to be included in Get Property Map (0x9f), but it's not")
	}

	// IsAnnouncementTargetメソッドで設置場所がアナウンス対象として判定されることを確認
	if !handler.comm.localDevices.IsAnnouncementTarget(controllerEOJ, echonet_lite.EPCInstallationLocation) {
		t.Errorf("Expected Installation Location (0x81) to be announcement target, but IsAnnouncementTarget returned false")
	}
}
