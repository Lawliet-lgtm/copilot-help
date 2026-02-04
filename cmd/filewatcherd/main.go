package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"linuxFileWatcher/internal/config"
	"linuxFileWatcher/internal/detector"
	"linuxFileWatcher/internal/identity"
	"linuxFileWatcher/internal/logger"
	"linuxFileWatcher/internal/postmanager"
	"linuxFileWatcher/internal/security"
	detectorservice "linuxFileWatcher/internal/service/detector"
	securityservice "linuxFileWatcher/internal/service/security"
	"linuxFileWatcher/internal/storage"
)

// ==========================================
// 全局服务实例
// ==========================================

var (
	// 涉密检测服务实例
	scannerSvc *detectorservice.ScannerService

	// 安全监控服务实例
	securityMonitorSvc *securityservice.SecurityMonitorService
)

// ==========================================
// 参数解析
// ==========================================

// parseArgs 解析命令行参数
func parseArgs() string {
	configPath := flag.String("c", "configs/config.yml", "配置文件路径")
	flag.Parse()
	return *configPath
}

// ==========================================
// 配置加载
// ==========================================

// loadConfig 加载配置文件
func loadConfig(configPath string) error {
	fmt.Printf("正在加载配置文件: %s\n", configPath)
	err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %v", err)
	}
	fmt.Printf("配置文件加载成功: %s\n", configPath)
	return nil
}

// ==========================================
// 基础设施初始化
// ==========================================

// initLogger 初始化日志系统
func initLogger() error {
	cfg := config.Get()
	fmt.Println("正在初始化日志系统...")
	if err := logger.Setup(logger.Options{
		Level:      cfg.Agent.LogLevel,
		FilePath:   cfg.Agent.LogFile,
		MaxSize:    cfg.Agent.LogMaxSize,
		MaxBackups: cfg.Agent.LogMaxBackups,
		MaxAge:     cfg.Agent.LogMaxAge,
		Compress:   cfg.Agent.LogCompress,
		Stdout:     cfg.Agent.LogStdout,
	}); err != nil {
		return fmt.Errorf("日志系统初始化失败: %w", err)
	}
	logger.Info("Agent initialized", "version", config.Version)
	return nil
}

// initSecurity 初始化安全模块（KMS、加密引擎等）
func initSecurity() error {
	fmt.Println("正在初始化安全模块...")
	if err := security.Setup(); err != nil {
		return fmt.Errorf("安全模块初始化失败: %w", err)
	}
	logger.Info("安全模块初始化成功")
	return nil
}

// initDatabase 初始化数据库
func initDatabase() error {
	fmt.Println("正在初始化数据库...")
	cfg := config.Get()
	dbCfg := cfg.Database

	if err := storage.Setup(storage.Options{
		DataDir:         cfg.Agent.DataDir,
		FileName:        dbCfg.FileName,
		LogLevel:        dbCfg.LogLevel,
		MaxOpenConns:    dbCfg.MaxOpenConns,
		MaxIdleConns:    dbCfg.MaxIdleConns,
		ConnMaxLifetime: dbCfg.ConnMaxLifetime,
		JournalMode:     dbCfg.JournalMode,
		Synchronous:     dbCfg.Synchronous,
		TempStore:       dbCfg.TempStore,
		ForeignKeys:     dbCfg.ForeignKeys,
	}); err != nil {
		return fmt.Errorf("database setup failed: %w", err)
	}
	logger.Info("数据库初始化成功")
	return nil
}

// initStores 初始化存储实例
func initStores() error {
	fmt.Println("正在初始化存储实例...")
	cfg := config.Get()
	storeCfg := cfg.Storage

	db, err := storage.GetDB()
	if err != nil {
		return fmt.Errorf("failed to get DB instance: %w", err)
	}

	if err := storage.SetupStores(db, storage.StoresOptions{
		AlertsMemoryLimit:    storeCfg.AlertsMemoryLimit,
		AuditLogsMemoryLimit: storeCfg.AuditLogsMemoryLimit,
		SecurityReportsLimit: storeCfg.SecurityReportsLimit,
		AlertLogsMemoryLimit: storeCfg.AlertLogsMemoryLimit,
	}); err != nil {
		return fmt.Errorf("failed to setup stores: %w", err)
	}
	logger.Info("存储实例初始化成功")
	return nil
}

// ==========================================
// 业务模块初始化
// ==========================================

// initIdentity 初始化身份信息
func initIdentity() error {
	fmt.Println("正在初始化身份信息...")
	cfg := config.Get()

	if err := identity.Init(cfg.Agent.DataDir); err != nil {
		return fmt.Errorf("身份信息初始化失败: %w", err)
	}

	id := identity.Get()
	logger.Info("身份信息加载完成",
		"computer", id.ComputerName,
		"user", id.UserName,
		"company", id.Company,
		"org_id", id.OrgID,
	)
	return nil
}

// initDetectorManager 初始化全局检测器管理器
func initDetectorManager() error {
	fmt.Println("正在初始化检测器管理器...")
	cfg := config.Get()
	id := identity.Get()

	detectorCfg := detector.GlobalConfig{
		// 检测模块开关
		EnableElectronicLabel: true,
		EnableSecretMarker:    true,
		EnableLayout:          true,
		EnableHash:            true,
		EnableKeywords:        true,

		// 检测配置
		SecretMarkerOCR: true,
		LayoutThreshold: 0.8,
		LayoutEnableOCR: true,

		// 基础环境信息（从 identity 读取）
		CurrentCompany:      id.Company,
		CurrentComputerName: id.ComputerName,
		CurrentOrgID:        id.OrgID,
		CurrentOrgPath:      id.OrgPath,
		CurrentUserName:     id.UserName,
		CurrentUserID:       id.UserID,

		// 配置文件路径
		ConfigPath: filepath.Join(cfg.Agent.DataDir, "detector_config.json"),
	}

	mgr := detector.InitGlobalManager(detectorCfg)

	if err := mgr.LoadConfig(detectorCfg.ConfigPath); err != nil {
		logger.Warn("加载检测器配置失败，使用默认配置", "error", err)
	}

	logger.Info("检测器管理器初始化成功")
	return nil
}

// initScannerService 初始化涉密检测服务
func initScannerService() error {
	fmt.Println("正在初始化涉密检测服务...")

	// 创建存储处理器
	storageHandler := detectorservice.NewStorageHandler()

	// 创建检测服务
	scannerSvc = detectorservice.NewScannerService(storageHandler)

	logger.Info("涉密检测服务初始化成功")
	return nil
}

// initSecurityMonitor 初始化安全监控服务
func initSecurityMonitor() error {
	fmt.Println("正在初始化安全监控服务...")

	// 从全局配置加载安全监控配置
	cfg := loadSecurityMonitorConfig()

	// 创建安全事件处理器（写入存储）
	handler := securityservice.NewSecurityHandler()

	// 创建安全监控服务
	securityMonitorSvc = securityservice.NewSecurityMonitorService(cfg, handler)

	logger.Info("安全监控服务初始化成功",
		"integrity_enabled", cfg.EnableIntegrity,
		"netguard_enabled", cfg.EnableNetguard,
	)

	return nil
}

// loadSecurityMonitorConfig 加载安全监控配置
func loadSecurityMonitorConfig() securityservice.SecurityMonitorConfig {
	cfg := securityservice.DefaultSecurityMonitorConfig()

	// 从全局配置读取（如果有配置项的话）
	globalCfg := config.Get()
	if globalCfg != nil {
		// 完整性校验配置
		if globalCfg.Security.Integrity.DefaultInterval > 0 {
			cfg.IntegrityInterval = globalCfg.Security.Integrity.DefaultInterval
		}

		// 可以根据需要扩展更多配置项
		// 例如从配置文件读取：
		// cfg.EnableIntegrity = globalCfg.Security.Integrity.Enable
		// cfg.EnableNetguard = globalCfg.Security.NetGuard.Enable
		// cfg.NetguardInterval = globalCfg.Security.NetGuard.CheckInterval
		// cfg.NetguardWhitelist = globalCfg.Security.NetGuard.Whitelist
		// cfg.NetguardDryRun = globalCfg.Security.NetGuard.DryRun
	}

	// 默认配置：启用完整性校验，网络监控使用 dry-run 模式
	cfg.EnableIntegrity = true
	cfg.EnableNetguard = true
	cfg.NetguardDryRun = true // 生产环境建议先用 dry-run 模式测试

	return cfg
}

// ==========================================
// 服务启动
// ==========================================

// startScannerService 启动涉密检测服务
func startScannerService() {
	if scannerSvc == nil {
		logger.Warn("涉密检测服务未初始化，跳过启动")
		return
	}

	fmt.Println("正在启动涉密检测服务...")
	scannerSvc.Start()
	logger.Info("涉密检测服务启动成功")
}

// startSecurityMonitor 启动安全监控服务 (非阻塞)
func startSecurityMonitor() {
	if securityMonitorSvc == nil {
		logger.Warn("安全监控服务未初始化，跳过启动")
		return
	}

	fmt.Println("正在启动安全监控服务...")
	if err := securityMonitorSvc.Start(); err != nil {
		logger.Error("安全监控服务启动失败", "error", err)
		return
	}

	logger.Info("安全监控服务启动成功")
}

// startPostManager 启动上报服务 (非阻塞)
func startPostManager() {
	fmt.Println("正在启动 PostManager 上报服务...")
	if err := postmanager.Init(); err != nil {
		logger.Error("postmanager模块初始化失败", "error", err)
		return
	}
	postmanager.StartAllReporting()
	logger.Info("所有上报服务启动成功")
}

// startFileWatcherSimulation 模拟文件监控 (仅用于测试数据生产)
func startFileWatcherSimulation() {
	if scannerSvc == nil {
		logger.Warn("涉密检测服务未初始化，跳过文件监控模拟")
		return
	}

	go func() {
		testDir := "./test_data"
		logger.Info("启动模拟文件监控", "watch_dir", testDir)

		for i := 0; i < 1; i++ {
			if _, err := os.Stat(testDir); os.IsNotExist(err) {
				logger.Warn("测试目录不存在，跳过模拟", "path", testDir)
				return
			}

			err := filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				scannerSvc.SubmitTask(path)
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			if err != nil {
				logger.Error("模拟遍历出错", "error", err)
			}
		}
	}()
}

// ==========================================
// 服务停止
// ==========================================

// stopSecurityMonitor 停止安全监控服务
func stopSecurityMonitor() {
	if securityMonitorSvc != nil && securityMonitorSvc.IsRunning() {
		fmt.Println("正在停止安全监控服务...")
		securityMonitorSvc.Stop()
	}
}

// stopScannerService 停止涉密检测服务
func stopScannerService() {
	if scannerSvc != nil {
		fmt.Println("正在停止涉密检测服务...")
		scannerSvc.Stop()
	}
}

// flushStorage 刷新存储
func flushStorage() {
	fmt.Println("正在刷新存储...")
	if err := storage.FlushAll(); err != nil {
		logger.Error("Failed to flush stores", "error", err)
	} else {
		logger.Info("存储刷新完成")
	}
}

// ==========================================
// 主入口
// ==========================================

func main() {
	fmt.Println("1")
	// ==========================================
	// 阶段 1: 参数解析与配置加载
	// ==========================================
	configPath := parseArgs()

	if err := loadConfig(configPath); err != nil {
		panic(fmt.Sprintf("配置加载失败: %v", err))
	}

	// ==========================================
	// 阶段 2: 基础设施初始化
	// ==========================================
	if err := initLogger(); err != nil {
		panic(fmt.Sprintf("日志系统初始化失败: %v", err))
	}

	// 安全模块必须在数据库之前初始化（存储需要加密功能）
	if err := initSecurity(); err != nil {
		panic(fmt.Sprintf("安全模块初始化失败: %v", err))
	}

	if err := initDatabase(); err != nil {
		panic(fmt.Sprintf("数据库初始化失败: %v", err))
	}

	if err := initStores(); err != nil {
		panic(fmt.Sprintf("存储实例初始化失败: %v", err))
	}

	// ==========================================
	// 阶段 3: 业务模块初始化
	// ==========================================
	if err := initIdentity(); err != nil {
		panic(fmt.Sprintf("身份信息初始化失败: %v", err))
	}

	if err := initDetectorManager(); err != nil {
		panic(fmt.Sprintf("检测器管理器初始化失败: %v", err))
	}

	if err := initScannerService(); err != nil {
		panic(fmt.Sprintf("涉密检测服务初始化失败: %v", err))
	}

	// 安全监控初始化失败不中断程序
	if err := initSecurityMonitor(); err != nil {
		logger.Error("安全监控服务初始化失败", "error", err)
	}

	// ==========================================
	// 阶段 4: 服务启动
	// ==========================================
	startScannerService()
	startPostManager()
	startSecurityMonitor()
	startFileWatcherSimulation()

	// ==========================================
	// 阶段 5: 运行中
	// ==========================================
	fmt.Println("=== 应用已完全启动 (按 Ctrl+C 停止) ===")
	logger.Info("应用启动完成")

	// ==========================================
	// 阶段 6: 优雅退出
	// ==========================================
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	fmt.Printf("\n[Main] 收到信号: %v，正在关闭服务...\n", sig)

	// 按依赖顺序停止服务（后启动的先停止）
	stopSecurityMonitor()
	stopScannerService()
	flushStorage()

	fmt.Println("[Main] 程序已安全退出")
}
