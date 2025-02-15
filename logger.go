package main

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// Logger is a custom logger that can write to a file and stdout
type Logger struct {
	fileLogger *log.Logger
	debugMode  bool
}

var (
	logFile    *os.File
	logger     *Logger
	logMutex   sync.Mutex
	defaultLog = "echonet-list.log" // デフォルトのログファイル名
)

// NewLogger creates a new logger that writes to the specified file
func NewLogger(filename string, debug bool) (*Logger, error) {
	logMutex.Lock()
	defer logMutex.Unlock()

	// Close existing log file if open
	if logFile != nil {
		logFile.Close()
	}

	// Open log file with append mode
	var err error
	logFile, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("ログファイルを開けませんでした: %w", err)
	}

	// Create file logger
	fileLogger := log.New(logFile, "", log.LstdFlags|log.Lmicroseconds)

	return &Logger{
		fileLogger: fileLogger,
		debugMode:  debug,
	}, nil
}

// Log writes a message to the log file
func (l *Logger) Log(format string, v ...interface{}) {
	if l.fileLogger != nil {
		l.fileLogger.Printf(format, v...)
	}
}

// Debug writes a debug message to stdout if debug mode is enabled
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.debugMode {
		fmt.Printf(format, v...)
	}
}

// SetDebug sets the debug mode
func (l *Logger) SetDebug(debug bool) {
	l.debugMode = debug
}

// Rotate closes and reopens the log file
func (l *Logger) Rotate() error {
	if logFile == nil {
		return nil // No log file to rotate
	}

	currentLogPath := logFile.Name()

	logMutex.Lock()
	defer logMutex.Unlock()

	// Close existing log file
	logFile.Close()

	// Reopen log file
	var err error
	logFile, err = os.OpenFile(currentLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("ログファイルを再オープンできませんでした: %w", err)
	}

	// Update logger
	l.fileLogger = log.New(logFile, "", log.LstdFlags|log.Lmicroseconds)

	return nil
}
