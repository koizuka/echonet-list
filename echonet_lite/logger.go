package echonet_lite

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// Logger is a custom logger that can write to a file and stdout
type Logger struct {
	logFile    *os.File
	logMutex   sync.Mutex
	fileLogger *log.Logger
	debugMode  bool
}

var (
	logger *Logger
)

func GetLogger() *Logger {
	return logger
}

func SetLogger(l *Logger) {
	if logger != nil {
		logger.Close()
	}
	logger = l
}

// NewLogger creates a new logger that writes to the specified file
func NewLogger(filename string, debug bool) (*Logger, error) {
	// Close existing log file if open

	// Open log file with append mode
	logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("ログファイルを開けませんでした: %w", err)
	}

	// Create file logger
	fileLogger := log.New(logFile, "", log.LstdFlags|log.Lmicroseconds)

	return &Logger{
		logFile:    logFile,
		fileLogger: fileLogger,
		debugMode:  debug,
	}, nil
}

func (l *Logger) Close() {
	l.logMutex.Lock()
	defer l.logMutex.Unlock()

	if l.logFile != nil {
		_ = l.logFile.Close()
		l.logFile = nil
	}
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
	if l.logFile == nil {
		return nil // No log file to rotate
	}

	currentLogPath := l.logFile.Name()

	l.logMutex.Lock()
	defer l.logMutex.Unlock()

	// Close existing log file
	_ = l.logFile.Close()

	// Reopen log file
	var err error
	logFile, err := os.OpenFile(currentLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("ログファイルを再オープンできませんでした: %w", err)
	}

	// Update logger
	l.fileLogger = log.New(logFile, "", log.LstdFlags|log.Lmicroseconds)
	l.logFile = logFile

	return nil
}
