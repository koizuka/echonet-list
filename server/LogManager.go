package server

import (
	"echonet-list/echonet_lite/log"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

type LogManager struct{}

func NewLogManager(logFilename string) (*LogManager, error) {
	// ロガーのセットアップ
	logger, err := log.NewLogger(logFilename)
	if err != nil {
		return nil, err
	}
	log.SetLogger(logger)

	// ログローテーション用のシグナルハンドリング (SIGHUP)
	rotateSignalCh := make(chan os.Signal, 1)
	signal.Notify(rotateSignalCh, syscall.SIGHUP)
	go func() {
		for {
			<-rotateSignalCh
			fmt.Fprintln(os.Stderr, "SIGHUPを受信しました。ログファイルをローテーションします...")
			logger := log.GetLogger()
			logger.Log("SIGHUPを受信しました。ログファイルをローテーションします...")
			if err := logger.Rotate(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "ログローテーションエラー: %v\n", err)
			}
		}
	}()

	return &LogManager{}, nil
}

func (lm *LogManager) Close() error {
	// ログファイルを閉じる
	log.SetLogger(nil)
	return nil
}
