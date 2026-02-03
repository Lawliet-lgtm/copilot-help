package integrity

import (
	"fmt"
	"os"
	"sync"
	"time"

	"linuxFileWatcher/internal/logger"
)

// FileIntegrityMonitor 自身完整性监控器
type FileIntegrityMonitor struct {
	targetPath   string // 自身二进制路径
	baselineHash string // 启动时计算的“可信基线”

	ticker   *time.Ticker  // 周期定时器
	stopChan chan struct{} // 停止信号
	running  bool          // 运行状态标记
	mu       sync.Mutex    // 锁

	reporter Reporter // 上报组件
}

// NewMonitor 创建监控器实例
func NewMonitor(reportImpl Reporter) (*FileIntegrityMonitor, error) {
	// 1. 定位自身
	path, err := GetSelfExecutablePath()
	if err != nil {
		return nil, err
	}

	// 2. 计算基线 (Baseline)
	// 这一步非常关键：我们假设程序启动这一刻是安全的
	hash, err := ComputeFileSM3(path)
	if err != nil {
		return nil, fmt.Errorf("failed to compute baseline hash: %v", err)
	}

	if reportImpl == nil {
		reportImpl = &DefaultConsoleReporter{}
	}

	logger.Info("Integrity baseline established", "path", path, "hash", hash)

	return &FileIntegrityMonitor{
		targetPath:   path,
		baselineHash: hash,
		stopChan:     make(chan struct{}),
		reporter:     reportImpl,
	}, nil
}

// Start 启动后台巡检
// interval: 巡检周期 (建议 1-5 分钟)
func (m *FileIntegrityMonitor) Start(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}

	m.ticker = time.NewTicker(interval)
	m.running = true

	// 启动协程
	go m.loop()
}

// Stop 停止监控
func (m *FileIntegrityMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	if m.ticker != nil {
		m.ticker.Stop()
	}
	close(m.stopChan)
	m.running = false
}

// loop 后台循环逻辑
func (m *FileIntegrityMonitor) loop() {
	for {
		select {
		case <-m.stopChan:
			return
		case <-m.ticker.C:
			m.checkIntegrity()
		}
	}
}

// checkIntegrity 执行一次完整性检查
func (m *FileIntegrityMonitor) checkIntegrity() {
	// 1. 检查文件是否存在/可读 (Stat)
	info, err := os.Stat(m.targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			m.reporter.Report(TypeFileDeleted, fmt.Sprintf("Executable file vanished: %s", m.targetPath))
		} else {
			m.reporter.Report(TypeReadError, fmt.Sprintf("Cannot stat executable: %v", err))
		}
		return
	}

	// 可选：检查权限是否被恶意修改 (例如变成全局可写)
	if info.Mode().Perm()&0002 != 0 {
		// 这是一个简单的示例，检查 other 用户是否有写权限
		m.reporter.Report(TypePermChanged, "Executable became world-writable!")
	}

	// 2. 计算当前 Hash
	currentHash, err := ComputeFileSM3(m.targetPath)
	if err != nil {
		m.reporter.Report(TypeReadError, fmt.Sprintf("Failed to compute hash during runtime: %v", err))
		return
	}

	// 3. 比对基线
	if currentHash != m.baselineHash {
		msg := fmt.Sprintf("CRITICAL: Integrity Mismatch! Baseline=%s, Current=%s", m.baselineHash, currentHash)
		m.reporter.Report(TypeFileModified, msg)
	}
}
