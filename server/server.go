package server

import (
	"context"
	"echonet-list/config"
	"echonet-list/echonet_lite/handler"
	"echonet-list/echonet_lite/network"
	"fmt"
	"time"
)

type Server struct {
	ctx         context.Context
	liteHandler *handler.ECHONETLiteHandler
}

func NewServer(ctx context.Context, cfg *config.Config) (*Server, error) {
	// ハンドラーオプションを作成
	options := handler.ECHONETLieHandlerOptions{Debug: cfg.Debug}

	// テストモード設定を追加
	if cfg != nil && cfg.TestMode.Enabled {
		options.TestDevicesFile = cfg.TestMode.DevicesFile
		options.TestAliasesFile = cfg.TestMode.AliasesFile
		options.TestGroupsFile = cfg.TestMode.GroupsFile
	}

	// キープアライブ設定を追加
	if cfg != nil && cfg.Multicast.KeepAliveEnabled {
		groupRefreshInterval, err := time.ParseDuration(cfg.Multicast.GroupRefreshInterval)
		if err != nil {
			groupRefreshInterval = 5 * time.Minute // デフォルト値
		}

		options.KeepAliveConfig = &network.KeepAliveConfig{
			Enabled:               cfg.Multicast.KeepAliveEnabled,
			GroupRefreshInterval:  groupRefreshInterval,
			NetworkMonitorEnabled: cfg.Multicast.NetworkMonitorEnabled,
		}
	}

	// ECHONETLiteHandlerの作成
	liteHandler, err := handler.NewECHONETLiteHandler(ctx, options)
	if err != nil {
		return nil, err
	}

	// メインループの開始
	liteHandler.StartMainLoop()

	// 通知を監視するゴルーチン
	go func() {
		for notification := range liteHandler.NotificationCh {
			device := liteHandler.DeviceStringWithAlias(notification.Device)

			switch notification.Type {
			case handler.DeviceAdded:
				fmt.Printf("新しいデバイスが検出されました: %v\n", device)
			case handler.DeviceOffline:
				fmt.Printf("デバイス %v がオフラインになりました\n", device)
			case handler.DeviceTimeout:
				// fmt.Printf("デバイス %v へのリクエストがタイムアウトしました: %v\n",
				// 	device, notification.Error)
			}
		}
	}()

	// ノードリストの通知
	_ = liteHandler.NotifyNodeList()

	// 起動時に　discover をするが、時間がかかるので goroutineで実行する
	go func() {
		_ = liteHandler.Discover()
	}()

	return &Server{
		ctx:         ctx,
		liteHandler: liteHandler,
	}, nil
}

func (s *Server) Close() error {
	return s.liteHandler.Close()
}

func (s *Server) GetHandler() *handler.ECHONETLiteHandler {
	return s.liteHandler
}
