package utils

import (
	"encoding/json"
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
	// 可选：JSON格式与过滤关键词
	jsonMode   bool
	includeKey string
	excludeKey string
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
		if l.jsonMode {
			l.printJSON("debug", format, args...)
			return
		}
		if l.filtered(format) {
			return
		}
		l.debugLog.Printf(format, args...)
	}
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		if l.jsonMode {
			l.printJSON("info", format, args...)
			return
		}
		if l.filtered(format) {
			return
		}
		l.infoLog.Printf(format, args...)
	}
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		if l.jsonMode {
			l.printJSON("warn", format, args...)
			return
		}
		if l.filtered(format) {
			return
		}
		l.warnLog.Printf(format, args...)
	}
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= LogLevelError {
		if l.jsonMode {
			l.printJSON("error", format, args...)
			return
		}
		if l.filtered(format) {
			return
		}
		l.errorLog.Printf(format, args...)
	}
}

// Fatal 致命错误日志（会终止程序）
func (l *Logger) Fatal(format string, args ...interface{}) {
	if l.level <= LogLevelFatal {
		if l.jsonMode {
			l.printJSON("fatal", format, args...)
		} else {
			if !l.filtered(format) {
				l.fatalLog.Printf(format, args...)
			}
		}
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

// EnableJSON 开启JSON日志
func (l *Logger) EnableJSON(enable bool) { l.jsonMode = enable }

// SetFilter 设置包含/排除关键词（简单过滤）
func (l *Logger) SetFilter(include, exclude string) { l.includeKey, l.excludeKey = include, exclude }

func (l *Logger) filtered(format string) bool {
	if l.includeKey != "" && !contains(format, l.includeKey) {
		return true
	}
	if l.excludeKey != "" && contains(format, l.excludeKey) {
		return true
	}
	return false
}

func (l *Logger) printJSON(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if l.filtered(msg) {
		return
	}
	payload := map[string]interface{}{
		"ts":    time.Now().Format(time.RFC3339Nano),
		"level": level,
		"msg":   msg,
	}
	b, _ := json.Marshal(payload)
	// 输出到infoLog以统一
	l.infoLog.Printf("%s", b)
}

// contains 简单包含判断（避免引入strings冲突变量名）
func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && (func() bool { return (len(sub) == 0) || (len(s) > 0 && (stringIndex(s, sub) >= 0)) })()
}

// stringIndex 朴素实现，避免额外依赖
func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
