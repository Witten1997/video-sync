package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"bili-download/internal/config"
)

// Logger 日志记录器
type Logger struct {
	*log.Logger
	level    LogLevel
	hooks    []LogHook
	fileSize int64
	maxSize  int64
}

// LogLevel 日志级别
type LogLevel int

const (
	// DEBUG 调试级别
	DEBUG LogLevel = iota
	// INFO 信息级别
	INFO
	// WARN 警告级别
	WARN
	// ERROR 错误级别
	ERROR
)

// LogEntry 日志条目
type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	File      string    `json:"file,omitempty"`
	Line      int       `json:"line,omitempty"`
}

// LogHook 日志钩子接口
type LogHook interface {
	OnLog(entry LogEntry)
}

var (
	globalLogger *Logger
)

// InitLogger 初始化日志系统
func InitLogger(cfg *config.Config) error {
	var writers []io.Writer

	// 标准输出
	writers = append(writers, os.Stdout)

	// 文件输出
	if cfg.Logging.File != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(cfg.Logging.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}

		// 打开日志文件
		file, err := os.OpenFile(cfg.Logging.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %w", err)
		}
		writers = append(writers, file)
	}

	// 创建多写入器
	multiWriter := io.MultiWriter(writers...)

	// 创建日志记录器
	logger := log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)

	// 解析日志级别
	level := parseLogLevel(cfg.Logging.Level)

	globalLogger = &Logger{
		Logger:   logger,
		level:    level,
		hooks:    make([]LogHook, 0),
		maxSize:  int64(cfg.Logging.MaxSizeMB) * 1024 * 1024,
		fileSize: 0,
	}

	return nil
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// GetLogger 获取全局日志记录器
func GetLogger() *Logger {
	if globalLogger == nil {
		// 默认日志记录器
		globalLogger = &Logger{
			Logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
			level:  INFO,
		}
	}
	return globalLogger
}

// AddHook 添加日志钩子
func (l *Logger) AddHook(hook LogHook) {
	l.hooks = append(l.hooks, hook)
}

// RemoveHook 移除日志钩子
func (l *Logger) RemoveHook(hook LogHook) {
	for i, h := range l.hooks {
		if h == hook {
			l.hooks = append(l.hooks[:i], l.hooks[i+1:]...)
			break
		}
	}
}

// triggerHooks 触发钩子
func (l *Logger) triggerHooks(entry LogEntry) {
	for _, hook := range l.hooks {
		go hook.OnLog(entry)
	}
}

// Debug 调试日志
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG {
		msg := fmt.Sprintf(format, v...)
		l.Output(2, fmt.Sprintf("[DEBUG] "+msg))
		l.triggerHooks(LogEntry{
			Level:     "debug",
			Message:   msg,
			Timestamp: time.Now(),
		})
	}
}

// Info 信息日志
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= INFO {
		msg := fmt.Sprintf(format, v...)
		l.Output(2, fmt.Sprintf("[INFO] "+msg))
		l.triggerHooks(LogEntry{
			Level:     "info",
			Message:   msg,
			Timestamp: time.Now(),
		})
	}
}

// Warn 警告日志
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= WARN {
		msg := fmt.Sprintf(format, v...)
		l.Output(2, fmt.Sprintf("[WARN] "+msg))
		l.triggerHooks(LogEntry{
			Level:     "warn",
			Message:   msg,
			Timestamp: time.Now(),
		})
	}
}

// Error 错误日志
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= ERROR {
		msg := fmt.Sprintf(format, v...)
		l.Output(2, fmt.Sprintf("[ERROR] "+msg))
		l.triggerHooks(LogEntry{
			Level:     "error",
			Message:   msg,
			Timestamp: time.Now(),
		})
	}
}

// 全局便捷方法

// Debug 调试日志
func Debug(format string, v ...interface{}) {
	GetLogger().Debug(format, v...)
}

// Info 信息日志
func Info(format string, v ...interface{}) {
	GetLogger().Info(format, v...)
}

// Warn 警告日志
func Warn(format string, v ...interface{}) {
	GetLogger().Warn(format, v...)
}

// Error 错误日志
func Error(format string, v ...interface{}) {
	GetLogger().Error(format, v...)
}
