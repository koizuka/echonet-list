package server

import (
	"context"
	"echonet-list/echonet_lite/handler"
	"fmt"
)

type Server struct {
	ctx         context.Context
	liteHandler *handler.ECHONETLiteHandler
}

func NewServer(ctx context.Context, debug bool) (*Server, error) {
	// ECHONETLiteHandlerの作成
	liteHandler, err := handler.NewECHONETLiteHandler(ctx, handler.ECHONETLieHandlerOptions{Debug: debug})
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
