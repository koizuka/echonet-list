package server

import (
	"context"
	"echonet-list/echonet_lite"
	"fmt"
)

type Server struct {
	ctx     context.Context
	handler *echonet_lite.ECHONETLiteHandler
}

func NewServer(ctx context.Context, debug bool) (*Server, error) {
	// ECHONETLiteHandlerの作成
	handler, err := echonet_lite.NewECHONETLiteHandler(ctx, nil, debug)
	if err != nil {
		return nil, err
	}

	// メインループの開始
	handler.StartMainLoop()

	// 通知を監視するゴルーチン
	go func() {
		for notification := range handler.NotificationCh {
			switch notification.Type {
			case echonet_lite.DeviceAdded:
				fmt.Printf("新しいデバイスが検出されました: %v\n", notification.Device)
			case echonet_lite.DeviceTimeout:
				// fmt.Printf("デバイス %v へのリクエストがタイムアウトしました: %v\n",
				// 	notification.Device, notification.Error)
			}
		}
	}()

	// ノードリストの通知
	_ = handler.NotifyNodeList()

	// 起動時に　discover をするが、時間がかかるので goroutineで実行する
	go func() {
		_ = handler.Discover()
	}()

	return &Server{
		ctx:     ctx,
		handler: handler,
	}, nil
}

func (s *Server) Close() error {
	return s.handler.Close()
}

func (s *Server) GetHandler() *echonet_lite.ECHONETLiteHandler {
	return s.handler
}
