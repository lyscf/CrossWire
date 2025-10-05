package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// Logger 日志记录器
type Logger struct {
	level      LogLevel
	debugLog   *log.Logger
	infoLog    *log.Logger
	warnLog    *log.Logger
	errorLog   *log.Logger
	fatalLog   *log.Logger
	fileWriter io.WriteCloser
}

// NewLogger 创建日志记录器
func NewLogger(level LogLevel, logDir string) (*Logger, error) {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	// 创建日志文件
	logFile := filepath.Join(logDir, fmt.Sprintf("crosswire_%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// 同时输出到文件和控制台
	multiWriter := io.MultiWriter(os.Stdout, file)

	logger := &Logger{
		level:      level,
		fileWriter: file,
		debugLog:   log.New(multiWriter, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile),
		infoLog:    log.New(multiWriter, "[INFO]  ", log.Ldate|log.Ltime),
		warnLog:    log.New(multiWriter, "[WARN]  ", log.Ldate|log.Ltime),
		errorLog:   log.New(multiWriter, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
		fatalLog:   log.New(multiWriter, "[FATAL] ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	return logger, nil
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		l.debugLog.Printf(format, args...)
	}
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		l.infoLog.Printf(format, args...)
	}
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		l.warnLog.Printf(format, args...)
	}
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= LogLevelError {
		l.errorLog.Printf(format, args...)
	}
}

// Fatal 致命错误日志（会终止程序）
func (l *Logger) Fatal(format string, args ...interface{}) {
	if l.level <= LogLevelFatal {
		l.fatalLog.Printf(format, args...)
		os.Exit(1)
	}
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	if l.fileWriter != nil {
		return l.fileWriter.Close()
	}
	return nil
}

// TODO: 实现以下功能
// - 日志轮转（按大小或时间）
// - 日志压缩
// - 结构化日志（JSON格式）
// - 日志过滤
