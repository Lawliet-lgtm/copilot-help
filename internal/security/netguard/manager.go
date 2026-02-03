package netguard

import (
	"fmt"
	"os"
	"sync"
	"time"

	"linuxFileWatcher/internal/logger"
	"linuxFileWatcher/internal/security/netguard/detector"
	"linuxFileWatcher/internal/security/netguard/enforcer"
	"linuxFileWatcher/internal/security/netguard/event"
	"linuxFileWatcher/internal/security/netguard/reporter"
)

// Manager 网络防护模块主控制器
type Manager struct {
	config    Config
	whitelist *WhitelistManager
	scanner   *detector.NetworkScanner
	enforcer  enforcer.IPEnforcer
	report    reporter.Reporter

	// 并发控制
	ticker   *time.Ticker
	stopChan chan struct{}
	running  bool
	mu       sync.Mutex

	// 去重缓存 (Deduplication Cache)
	// 用于防止同一个异常连接在被封禁后，由于 TCP 超时未断开而导致重复频繁告警
	// Key: RemoteIP, Value: 上次处理时间
	handledIPs map[string]time.Time
}

// NewManager 创建管理器实例
// reporter: 上报接口实现，如果为 nil 则使用默认控制台输出
func NewManager(cfg Config, rep reporter.Reporter) *Manager {
	if rep == nil {
		rep = &reporter.MockConsoleReporter{}
	}

	// 1. 初始化白名单
	wm := NewWhitelistManager(cfg.InitialWhitelist)

	// 2. 确定监控目标 PID
	targetPIDs := cfg.TargetPIDs
	if len(targetPIDs) == 0 && cfg.MonitorSelf {
		targetPIDs = []int32{int32(os.Getpid())}
	}

	// 3. 初始化扫描器
	scanner := detector.NewScanner(targetPIDs)

	// 4. 初始化执行器 (默认使用 iptables)
	enf := enforcer.NewIptablesEnforcer()

	return &Manager{
		config:     cfg,
		whitelist:  wm,
		scanner:    scanner,
		enforcer:   enf,
		report:     rep,
		stopChan:   make(chan struct{}),
		handledIPs: make(map[string]time.Time),
	}
}

// Start 启动防护
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}

	// --- 【新增】权限预检 ---
	if os.Geteuid() != 0 {
		logger.Error("NetGuard not running as Root! IPTables enforcement will fail.")
		// 这里可以选择直接 return，或者仅打印警告让其继续运行（只检测不阻断）
		// 建议仅打印警告，因为 Detector 依然有效
	}

	if !m.config.Enable {
		logger.Info("NetGuard module disabled by config")
		return
	}

	logger.Info("NetGuard starting protection loop", "interval", m.config.CheckInterval, "pids", m.scanner.TargetPIDs)

	m.ticker = time.NewTicker(m.config.CheckInterval)
	m.running = true

	// 启动后台巡检协程
	go m.loop()
}

// Stop 停止防护
func (m *Manager) Stop() {
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
	logger.Info("NetGuard protection stopped")
}

// AddWhitelist 动态添加白名单 (供外部调用)
func (m *Manager) AddWhitelist(ipOrCidr string) {
	m.whitelist.Add(ipOrCidr)
	logger.Info("NetGuard added to whitelist", "ip", ipOrCidr)
}

// loop 主循环
func (m *Manager) loop() {
	for {
		select {
		case <-m.stopChan:
			return
		case <-m.ticker.C:
			m.processNetworkState()
		}
	}
}

// processNetworkState 执行一次完整的检测流程
func (m *Manager) processNetworkState() {
	// 1. 扫描当前连接
	conns, err := m.scanner.Scan()
	if err != nil {
		logger.Error("NetGuard scan error", "error", err)
		return
	}

	for _, conn := range conns {
		// 2. 白名单检查
		if m.whitelist.IsAllowed(conn.RemoteIP) {
			continue
		}

		// 3. 去重检查 (防止刷屏)
		// 如果这个 IP 最近已经被处理过(封禁过)，则跳过
		// 即使封禁了，连接可能处于 TIME_WAIT 或重试中，不需要反复调 iptables
		if lastTime, exists := m.handledIPs[conn.RemoteIP]; exists {
			// 如果配置的去重时间内处理过，就不再处理
			if time.Since(lastTime) < m.config.DeduplicationTime {
				continue
			}
		}

		// ============== 发现异常 ==============

		// 4. 执行封禁
		blockErr := m.enforcer.BlockIP(conn.RemoteIP)
		action := "BLOCKED"
		if blockErr != nil {
			action = fmt.Sprintf("BLOCK_FAILED (%v)", blockErr)
			// 如果封禁失败（比如权限不足），我们不更新 handledIPs，以便下次继续尝试报警
		} else {
			// 封禁成功，记录缓存
			m.handledIPs[conn.RemoteIP] = time.Now()
		}

		// 5. 判断方向 (简单推断)
		direction := event.DirectionOutbound
		if conn.Status == "SYN_RECV" {
			direction = event.DirectionInbound
		}
		// ESTABLISHED 状态较难判断发起方，默认归为 Outbound 或由人工研判

		// 6. 构造告警
		alert := event.NetworkAlert{
			Timestamp:   time.Now(),
			AlertTime:   time.Now().Unix(),
			Direction:   direction,
			RemoteIP:    conn.RemoteIP,
			RemotePort:  uint16(conn.RemotePort),
			LocalPort:   uint16(conn.LocalPort),
			Protocol:    conn.Protocol,
			ProcessName: "self", // 暂定，如果有多 PID 需反查名称
			PID:         conn.PID,
			ActionTaken: action,
		}

		// 7. 上报
		_ = m.report.Report(alert)
	}
}
