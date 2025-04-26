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
			case echonet_lite.DeviceOffline:
				fmt.Printf("デバイス %v がオフラインになりました\n", notification.Device)
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

	// マルチキャスト監視を開始
	go func() {
		// TODO: このマルチキャスト監視では正常に受信できているのに、通知は来なくなっているため、来ない原因は
		// マルチキャストからの離脱ではなさそう
		if err := handler.StartMulticastMonitoring(); err != nil {
			fmt.Printf("マルチキャスト監視の開始に失敗しました: %v\n", err)
		} else {
			fmt.Println("マルチキャスト監視を開始しました")
		}
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
