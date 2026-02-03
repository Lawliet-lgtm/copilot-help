package errors

import (
	"fmt"
	"runtime/debug"
)

// RecoveryHandler panic 恢复处理器
type RecoveryHandler func(recovered interface{}, stack []byte) error

// DefaultRecoveryHandler 默认恢复处理器
func DefaultRecoveryHandler(recovered interface{}, stack []byte) error {
	return NewDetectorError(ErrInternal,
		fmt.Sprintf("程序发生异常: %v", recovered)).
		WithLevel(LevelFatal).
		AddExtra("stack", string(stack))
}

// SafeExecute 安全执行函数（带 panic 恢复）
func SafeExecute(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			err = DefaultRecoveryHandler(r, stack)
		}
	}()

	return fn()
}

// SafeExecuteWithHandler 安全执行函数（自定义恢复处理器）
func SafeExecuteWithHandler(fn func() error, handler RecoveryHandler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			if handler != nil {
				err = handler(r, stack)
			} else {
				err = DefaultRecoveryHandler(r, stack)
			}
		}
	}()

	return fn()
}

// SafeExecuteWithResult 安全执行带返回值的函数
func SafeExecuteWithResult[T any](fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			err = DefaultRecoveryHandler(r, stack)
		}
	}()

	return fn()
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxAttempts int           // 最大尝试次数
	ShouldRetry func(error) bool // 判断是否应该重试
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 3,
		ShouldRetry: IsRecoverable,
	}
}

// Retry 重试执行
func Retry(fn func() error, config *RetryConfig) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := SafeExecute(fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否应该重试
		if !config.ShouldRetry(err) {
			return err
		}

		// 最后一次尝试不再重试
		if attempt == config.MaxAttempts {
			break
		}
	}

	return WrapError(lastErr, ErrUnknown,
		fmt.Sprintf("重试 %d 次后仍然失败", config.MaxAttempts))
}

// RetryWithResult 重试执行带返回值的函数
func RetryWithResult[T any](fn func() (T, error), config *RetryConfig) (T, error) {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	var zero T

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		result, err := SafeExecuteWithResult(fn)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !config.ShouldRetry(err) {
			return zero, err
		}

		if attempt == config.MaxAttempts {
			break
		}
	}

	return zero, WrapError(lastErr, ErrUnknown,
		fmt.Sprintf("重试 %d 次后仍然失败", config.MaxAttempts))
}