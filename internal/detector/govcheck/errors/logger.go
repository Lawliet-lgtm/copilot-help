package errors

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Logger 错误日志记录器
type Logger struct {
	mu       sync.Mutex
	output   io.Writer
	minLevel ErrorLevel
	prefix   string
}

// NewLogger 创建日志记录器
func NewLogger(output io.Writer) *Logger {
	if output == nil {
		output = os.Stderr
	}
	return &Logger{
		output:   output,
		minLevel: LevelInfo,
	}
}

// SetMinLevel 设置最小日志级别
func (l *Logger) SetMinLevel(level ErrorLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

// SetPrefix 设置日志前缀
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// Log 记录错误
func (l *Logger) Log(err *DetectorError) {
	if err == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err.Level < l.minLevel {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	var msg string
	if l.prefix != "" {
		msg = fmt.Sprintf("[%s] %s %s\n", timestamp, l.prefix, err.Error())
	} else {
		msg = fmt.Sprintf("[%s] %s\n", timestamp, err.Error())
	}

	fmt.Fprint(l.output, msg)
}

// LogError 记录普通错误
func (l *Logger) LogError(err error) {
	if err == nil {
		return
	}

	if detErr, ok := err.(*DetectorError); ok {
		l.Log(detErr)
	} else {
		l.Log(NewDetectorError(ErrUnknown, err.Error()))
	}
}

// Info 记录信息
func (l *Logger) Info(message string) {
	l.Log(NewDetectorError(ErrUnknown, message).WithLevel(LevelInfo))
}

// Warning 记录警告
func (l *Logger) Warning(message string) {
	l.Log(NewDetectorError(ErrUnknown, message).WithLevel(LevelWarning))
}

// Error 记录错误
func (l *Logger) Error(message string) {
	l.Log(NewDetectorError(ErrUnknown, message).WithLevel(LevelError))
}

// Fatal 记录致命错误
func (l *Logger) Fatal(message string) {
	l.Log(NewDetectorError(ErrUnknown, message).WithLevel(LevelFatal))
}

// 全局默认日志记录器
var defaultLogger = NewLogger(os.Stderr)

// SetDefaultLogger 设置默认日志记录器
func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

// GetDefaultLogger 获取默认日志记录器
func GetDefaultLogger() *Logger {
	return defaultLogger
}

// LogInfo 使用默认日志记录器记录信息
func LogInfo(message string) {
	defaultLogger.Info(message)
}

// LogWarning 使用默认日志记录器记录警告
func LogWarning(message string) {
	defaultLogger.Warning(message)
}

// LogError 使用默认日志记录器记录错误
func LogErrorMsg(message string) {
	defaultLogger.Error(message)
}

// LogFatal 使用默认日志记录器记录致命错误
func LogFatal(message string) {
	defaultLogger.Fatal(message)
}