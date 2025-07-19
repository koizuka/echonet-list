package server

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type LogManager struct {
	logFilename string
	file        *os.File
	mu          sync.Mutex
	transport   WebSocketTransport
}

func NewLogManager(logFilename string) (*LogManager, error) {
	lm := &LogManager{logFilename: logFilename}
	if err := lm.openAndSetLogger(); err != nil {
		return nil, err
	}
	return lm, nil
}

func (lm *LogManager) openAndSetLogger() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.file != nil {
		lm.file.Close()
	}
	file, err := os.OpenFile(lm.logFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("ログファイルを開けませんでした: %w", err)
	}

	// Create text handler for file logging
	textHandler := slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})

	// Wrap with broadcast handler if transport is available
	var handler slog.Handler = textHandler
	if lm.transport != nil {
		handler = NewBroadcastHandler(textHandler, lm.transport, slog.LevelWarn)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	lm.file = file
	return nil
}

// AutoRotateは、SIGHUPシグナルを受信したときにログファイルをローテーションします。
func (lm *LogManager) AutoRotate() {
	rotateSignalCh := make(chan os.Signal, 1)
	signal.Notify(rotateSignalCh, syscall.SIGHUP)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_, _ = fmt.Fprintf(os.Stderr, "ログローテーションgoroutineでpanicが発生しました: %v\n", r)
			}
		}()

		for range rotateSignalCh {
			fmt.Fprintln(os.Stderr, "SIGHUPを受信しました。ログファイルをローテーションします...")
			err := lm.openAndSetLogger()
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "ログローテーションエラー: %v\n", err)
			} else {
				slog.Info("SIGHUPを受信しました。ログファイルをローテーションしました")
			}
		}
	}()
}

// SetTransport sets the WebSocket transport for broadcasting logs
func (lm *LogManager) SetTransport(transport WebSocketTransport) error {
	lm.mu.Lock()
	lm.transport = transport
	lm.mu.Unlock()

	// Reopen logger to apply the transport
	return lm.openAndSetLogger()
}

func (lm *LogManager) Close() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if lm.file != nil {
		err := lm.file.Close()
		lm.file = nil
		return err
	}
	return nil
}
