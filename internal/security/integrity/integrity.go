package integrity

import (
	"fmt"
	"time"

	"linuxFileWatcher/internal/config"
	"linuxFileWatcher/internal/logger"
)

// Service 自身完整性校验服务
type Service struct {
	monitor *FileIntegrityMonitor
}

// NewService 初始化服务
// reporter: 传入 nil 则使用默认控制台输出，实际集成时应传入适配了 HTTP Client 的 struct
func NewService(reporter Reporter) (*Service, error) {
	mon, err := NewMonitor(reporter)
	if err != nil {
		return nil, fmt.Errorf("integrity service init failed: %v", err)
	}
	return &Service{monitor: mon}, nil
}

// StartService 启动服务
// 推荐检查间隔: 5分钟 (300秒)，太频繁会消耗 IO
func (s *Service) StartService(interval time.Duration) {
	// 安全防御：防止外部传入 0 或负数导致 CPU 空转
	if interval < 1*time.Second {
		// 使用配置中的默认值，如果配置未初始化则使用硬编码默认值作为备选
		var defaultInterval time.Duration = 1 * time.Minute
		if config.GlobalConfig != nil {
			defaultInterval = config.GlobalConfig.Security.Integrity.DefaultInterval
		}
		interval = defaultInterval
		logger.Warn("Integrity service interval too short, resetting to default", "interval", interval)
	}

	logger.Info("Integrity service starting self-check loop", "interval", interval)
	s.monitor.Start(interval)
}

// StopService 停止服务
func (s *Service) StopService() {
	s.monitor.Stop()
	logger.Info("Integrity service stopped")
}
