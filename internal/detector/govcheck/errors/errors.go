package errors

import (
	"fmt"
	"strings"
	"time"
)

// ErrorLevel 错误级别
type ErrorLevel int

const (
	LevelInfo    ErrorLevel = iota // 信息
	LevelWarning                   // 警告
	LevelError                     // 错误
	LevelFatal                     // 致命错误
)

// String 返回错误级别的字符串表示
func (l ErrorLevel) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARNING"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ErrorCode 错误代码
type ErrorCode int

const (
	// 通用错误 (1000-1999)
	ErrUnknown       ErrorCode = 1000
	ErrInvalidInput  ErrorCode = 1001
	ErrTimeout       ErrorCode = 1002
	ErrCancelled     ErrorCode = 1003
	ErrNotSupported  ErrorCode = 1004
	ErrInternal      ErrorCode = 1005

	// 文件错误 (2000-2999)
	ErrFileNotFound    ErrorCode = 2000
	ErrFileEmpty       ErrorCode = 2001
	ErrFileTooLarge    ErrorCode = 2002
	ErrFileReadFailed  ErrorCode = 2003
	ErrFileWriteFailed ErrorCode = 2004
	ErrFileFormat      ErrorCode = 2005
	ErrFilePermission  ErrorCode = 2006
	ErrFileLocked      ErrorCode = 2007

	// 处理器错误 (3000-3999)
	ErrProcessorNotFound   ErrorCode = 3000
	ErrProcessorFailed     ErrorCode = 3001
	ErrProcessorTimeout    ErrorCode = 3002
	ErrExtractionFailed    ErrorCode = 3003
	ErrParsingFailed       ErrorCode = 3004
	ErrEncodingFailed      ErrorCode = 3005
	ErrExternalToolMissing ErrorCode = 3006
	ErrExternalToolFailed  ErrorCode = 3007

	// 配置错误 (4000-4999)
	ErrConfigNotFound  ErrorCode = 4000
	ErrConfigInvalid   ErrorCode = 4001
	ErrConfigParsing   ErrorCode = 4002
	ErrConfigValue     ErrorCode = 4003

	// 检测错误 (5000-5999)
	ErrDetectionFailed ErrorCode = 5000
	ErrNoContent       ErrorCode = 5001
	ErrInvalidContent  ErrorCode = 5002
)

// 错误代码描述映射
var errorDescriptions = map[ErrorCode]string{
	ErrUnknown:       "未知错误",
	ErrInvalidInput:  "无效的输入",
	ErrTimeout:       "操作超时",
	ErrCancelled:     "操作已取消",
	ErrNotSupported:  "不支持的操作",
	ErrInternal:      "内部错误",

	ErrFileNotFound:    "文件不存在",
	ErrFileEmpty:       "文件为空",
	ErrFileTooLarge:    "文件过大",
	ErrFileReadFailed:  "文件读取失败",
	ErrFileWriteFailed: "文件写入失败",
	ErrFileFormat:      "文件格式错误",
	ErrFilePermission:  "文件权限不足",
	ErrFileLocked:      "文件被锁定",

	ErrProcessorNotFound:   "处理器未找到",
	ErrProcessorFailed:     "处理器执行失败",
	ErrProcessorTimeout:    "处理器执行超时",
	ErrExtractionFailed:    "内容提取失败",
	ErrParsingFailed:       "解析失败",
	ErrEncodingFailed:      "编码转换失败",
	ErrExternalToolMissing: "外部工具未安装",
	ErrExternalToolFailed:  "外部工具执行失败",

	ErrConfigNotFound:  "配置文件未找到",
	ErrConfigInvalid:   "配置文件无效",
	ErrConfigParsing:   "配置文件解析失败",
	ErrConfigValue:     "配置值无效",

	ErrDetectionFailed: "检测失败",
	ErrNoContent:       "没有可检测的内容",
	ErrInvalidContent:  "内容无效",
}

// Description 返回错误代码的描述
func (c ErrorCode) Description() string {
	if desc, ok := errorDescriptions[c]; ok {
		return desc
	}
	return "未知错误"
}

// DetectorError 检测器错误
type DetectorError struct {
	Code      ErrorCode   // 错误代码
	Level     ErrorLevel  // 错误级别
	Message   string      // 错误消息
	Component string      // 组件名称
	FilePath  string      // 相关文件路径
	Operation string      // 操作名称
	Cause     error       // 原始错误
	Timestamp time.Time   // 发生时间
	Context   ErrorContext // 上下文信息
}

// ErrorContext 错误上下文
type ErrorContext struct {
	FileSize    int64             // 文件大小
	FileType    string            // 文件类型
	ProcessorName string          // 处理器名称
	Duration    time.Duration     // 操作耗时
	Extra       map[string]string // 额外信息
}

// NewDetectorError 创建检测器错误
func NewDetectorError(code ErrorCode, message string) *DetectorError {
	return &DetectorError{
		Code:      code,
		Level:     LevelError,
		Message:   message,
		Timestamp: time.Now(),
		Context:   ErrorContext{Extra: make(map[string]string)},
	}
}

// Error 实现 error 接口
func (e *DetectorError) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] ", e.Level.String()))

	if e.Component != "" {
		sb.WriteString(fmt.Sprintf("[%s] ", e.Component))
	}

	sb.WriteString(e.Message)

	if e.FilePath != "" {
		sb.WriteString(fmt.Sprintf(" (文件: %s)", e.FilePath))
	}

	if e.Cause != nil {
		sb.WriteString(fmt.Sprintf(": %v", e.Cause))
	}

	return sb.String()
}

// Unwrap 返回原始错误
func (e *DetectorError) Unwrap() error {
	return e.Cause
}

// WithLevel 设置错误级别
func (e *DetectorError) WithLevel(level ErrorLevel) *DetectorError {
	e.Level = level
	return e
}

// WithComponent 设置组件名称
func (e *DetectorError) WithComponent(component string) *DetectorError {
	e.Component = component
	return e
}

// WithFile 设置相关文件
func (e *DetectorError) WithFile(filePath string) *DetectorError {
	e.FilePath = filePath
	return e
}

// WithOperation 设置操作名称
func (e *DetectorError) WithOperation(operation string) *DetectorError {
	e.Operation = operation
	return e
}

// WithCause 设置原始错误
func (e *DetectorError) WithCause(cause error) *DetectorError {
	e.Cause = cause
	return e
}

// WithContext 设置上下文
func (e *DetectorError) WithContext(ctx ErrorContext) *DetectorError {
	e.Context = ctx
	return e
}

// AddExtra 添加额外信息
func (e *DetectorError) AddExtra(key, value string) *DetectorError {
	if e.Context.Extra == nil {
		e.Context.Extra = make(map[string]string)
	}
	e.Context.Extra[key] = value
	return e
}

// IsWarning 是否是警告
func (e *DetectorError) IsWarning() bool {
	return e.Level == LevelWarning
}

// IsFatal 是否是致命错误
func (e *DetectorError) IsFatal() bool {
	return e.Level == LevelFatal
}

// UserMessage 返回用户友好的错误消息
func (e *DetectorError) UserMessage() string {
	var sb strings.Builder

	// 基本描述
	sb.WriteString(e.Code.Description())

	// 添加具体信息
	if e.Message != "" && e.Message != e.Code.Description() {
		sb.WriteString("：")
		sb.WriteString(e.Message)
	}

	// 添加解决建议
	suggestion := e.getSuggestion()
	if suggestion != "" {
		sb.WriteString("\n建议：")
		sb.WriteString(suggestion)
	}

	return sb.String()
}

// getSuggestion 获取解决建议
func (e *DetectorError) getSuggestion() string {
	switch e.Code {
	case ErrFileNotFound:
		return "请检查文件路径是否正确"
	case ErrFileEmpty:
		return "请确认文件内容不为空"
	case ErrFileTooLarge:
		return "请尝试处理较小的文件，或调整配置中的文件大小限制"
	case ErrFileFormat:
		return "请确认文件格式正确，未被损坏"
	case ErrFilePermission:
		return "请检查是否有读取该文件的权限"
	case ErrExternalToolMissing:
		return "请安装所需的外部工具（如 antiword、LibreOffice、Tesseract）"
	case ErrExternalToolFailed:
		return "请检查外部工具是否正常工作"
	case ErrProcessorNotFound:
		return "该文件格式暂不支持"
	case ErrNoContent:
		return "请确认文件包含可提取的文本内容"
	case ErrConfigInvalid:
		return "请检查配置文件格式是否正确"
	case ErrTimeout:
		return "请尝试处理较小的文件，或增加超时时间"
	default:
		return ""
	}
}

// ============================================================
// 便捷构造函数
// ============================================================

// FileNotFoundError 文件未找到错误
func FileNotFoundError(filePath string) *DetectorError {
	return NewDetectorError(ErrFileNotFound, fmt.Sprintf("文件不存在: %s", filePath)).
		WithFile(filePath).
		WithLevel(LevelError)
}

// FileEmptyError 文件为空错误
func FileEmptyError(filePath string) *DetectorError {
	return NewDetectorError(ErrFileEmpty, "文件内容为空").
		WithFile(filePath).
		WithLevel(LevelError)
}

// FileTooLargeError 文件过大错误
func FileTooLargeError(filePath string, size, maxSize int64) *DetectorError {
	return NewDetectorError(ErrFileTooLarge,
		fmt.Sprintf("文件大小 %d 字节，超过限制 %d 字节", size, maxSize)).
		WithFile(filePath).
		WithLevel(LevelError).
		AddExtra("size", fmt.Sprintf("%d", size)).
		AddExtra("max_size", fmt.Sprintf("%d", maxSize))
}

// FileFormatError 文件格式错误
func FileFormatError(filePath, expected, actual string) *DetectorError {
	return NewDetectorError(ErrFileFormat,
		fmt.Sprintf("期望格式: %s, 实际格式: %s", expected, actual)).
		WithFile(filePath).
		WithLevel(LevelError)
}

// FileReadError 文件读取错误
func FileReadError(filePath string, cause error) *DetectorError {
	return NewDetectorError(ErrFileReadFailed, "读取文件失败").
		WithFile(filePath).
		WithCause(cause).
		WithLevel(LevelError)
}

// ProcessorError 处理器错误
func ProcessorError(processor, filePath, operation string, cause error) *DetectorError {
	return NewDetectorError(ErrProcessorFailed,
		fmt.Sprintf("%s 执行失败", operation)).
		WithComponent(processor).
		WithFile(filePath).
		WithOperation(operation).
		WithCause(cause).
		WithLevel(LevelError)
}

// ExternalToolMissingError 外部工具缺失错误
func ExternalToolMissingError(toolName string) *DetectorError {
	return NewDetectorError(ErrExternalToolMissing,
		fmt.Sprintf("外部工具未安装: %s", toolName)).
		WithComponent(toolName).
		WithLevel(LevelWarning)
}

// ExternalToolFailedError 外部工具执行失败
func ExternalToolFailedError(toolName string, cause error) *DetectorError {
	return NewDetectorError(ErrExternalToolFailed,
		fmt.Sprintf("外部工具执行失败: %s", toolName)).
		WithComponent(toolName).
		WithCause(cause).
		WithLevel(LevelError)
}

// ExtractionError 内容提取错误
func ExtractionError(filePath, reason string) *DetectorError {
	return NewDetectorError(ErrExtractionFailed, reason).
		WithFile(filePath).
		WithLevel(LevelError)
}

// NoContentError 无内容错误
func NoContentError(filePath string) *DetectorError {
	return NewDetectorError(ErrNoContent, "文件中没有可检测的文本内容").
		WithFile(filePath).
		WithLevel(LevelWarning)
}

// ConfigError 配置错误
func ConfigError(configPath, message string, cause error) *DetectorError {
	return NewDetectorError(ErrConfigInvalid, message).
		WithFile(configPath).
		WithCause(cause).
		WithLevel(LevelError)
}

// ============================================================
// 错误集合（用于批量处理）
// ============================================================

// ErrorCollection 错误集合
type ErrorCollection struct {
	errors   []*DetectorError
	warnings []*DetectorError
}

// NewErrorCollection 创建错误集合
func NewErrorCollection() *ErrorCollection {
	return &ErrorCollection{
		errors:   make([]*DetectorError, 0),
		warnings: make([]*DetectorError, 0),
	}
}

// Add 添加错误
func (c *ErrorCollection) Add(err *DetectorError) {
	if err == nil {
		return
	}

	if err.IsWarning() {
		c.warnings = append(c.warnings, err)
	} else {
		c.errors = append(c.errors, err)
	}
}

// AddError 添加普通错误
func (c *ErrorCollection) AddError(err error) {
	if err == nil {
		return
	}

	if detErr, ok := err.(*DetectorError); ok {
		c.Add(detErr)
	} else {
		c.Add(NewDetectorError(ErrUnknown, err.Error()))
	}
}

// HasErrors 是否有错误
func (c *ErrorCollection) HasErrors() bool {
	return len(c.errors) > 0
}

// HasWarnings 是否有警告
func (c *ErrorCollection) HasWarnings() bool {
	return len(c.warnings) > 0
}

// Errors 返回所有错误
func (c *ErrorCollection) Errors() []*DetectorError {
	return c.errors
}

// Warnings 返回所有警告
func (c *ErrorCollection) Warnings() []*DetectorError {
	return c.warnings
}

// ErrorCount 错误数量
func (c *ErrorCollection) ErrorCount() int {
	return len(c.errors)
}

// WarningCount 警告数量
func (c *ErrorCollection) WarningCount() int {
	return len(c.warnings)
}

// Summary 返回摘要
func (c *ErrorCollection) Summary() string {
	return fmt.Sprintf("%d 个错误, %d 个警告", len(c.errors), len(c.warnings))
}

// FirstError 返回第一个错误
func (c *ErrorCollection) FirstError() *DetectorError {
	if len(c.errors) > 0 {
		return c.errors[0]
	}
	return nil
}

// ============================================================
// 错误判断辅助函数
// ============================================================

// IsDetectorError 检查是否是 DetectorError
func IsDetectorError(err error) bool {
	_, ok := err.(*DetectorError)
	return ok
}

// GetErrorCode 获取错误代码
func GetErrorCode(err error) ErrorCode {
	if detErr, ok := err.(*DetectorError); ok {
		return detErr.Code
	}
	return ErrUnknown
}

// IsFileError 是否是文件相关错误
func IsFileError(err error) bool {
	code := GetErrorCode(err)
	return code >= 2000 && code < 3000
}

// IsProcessorError 是否是处理器相关错误
func IsProcessorError(err error) bool {
	code := GetErrorCode(err)
	return code >= 3000 && code < 4000
}

// IsConfigError 是否是配置相关错误
func IsConfigError(err error) bool {
	code := GetErrorCode(err)
	return code >= 4000 && code < 5000
}

// IsRecoverable 错误是否可恢复
func IsRecoverable(err error) bool {
	if detErr, ok := err.(*DetectorError); ok {
		// 致命错误不可恢复
		if detErr.IsFatal() {
			return false
		}
		// 以下错误代码可恢复
		switch detErr.Code {
		case ErrExternalToolMissing, ErrExternalToolFailed:
			return true // 可以尝试其他工具
		case ErrTimeout:
			return true // 可以重试
		case ErrProcessorFailed:
			return true // 可以尝试其他处理器
		}
	}
	return false
}

// WrapError 包装标准错误为 DetectorError
func WrapError(err error, code ErrorCode, message string) *DetectorError {
	if err == nil {
		return nil
	}

	if detErr, ok := err.(*DetectorError); ok {
		// 已经是 DetectorError，添加上下文
		if message != "" {
			detErr.Message = message + ": " + detErr.Message
		}
		return detErr
	}

	return NewDetectorError(code, message).WithCause(err)
}