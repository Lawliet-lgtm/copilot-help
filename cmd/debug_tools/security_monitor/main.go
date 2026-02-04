// Package main 提供集成式安全监控调试工具
// 整合完整性校验和网络连接监控功能，提供统一的监控界面
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"linuxFileWatcher/internal/security/integrity"
	"linuxFileWatcher/internal/security/netguard"
	"linuxFileWatcher/internal/security/netguard/detector"
)

// ==========================================
// 全局变量和配置
// ==========================================

var (
	version = "1.0.0"
	appName = "security-monitor"

	// 模块开关
	enableIntegrity bool
	enableNetguard  bool

	// 完整性校验参数
	integrityFile     string
	integrityInterval time.Duration

	// 网络监控参数
	netguardPIDs      []int
	netguardInterval  time.Duration
	netguardWhitelist []string
	netguardDryRun    bool

	// 通用参数
	verboseMode bool
	quietMode   bool

	// 颜色输出
	colorRed     = color.New(color.FgRed, color.Bold)
	colorGreen   = color.New(color.FgGreen, color.Bold)
	colorYellow  = color.New(color.FgYellow)
	colorCyan    = color.New(color.FgCyan)
	colorMagenta = color.New(color.FgMagenta)
	colorWhite   = color.New(color.FgWhite)
	colorBlue    = color.New(color.FgBlue, color.Bold)
)

// ==========================================
// 统计信息
// ==========================================

type MonitorStats struct {
	StartTime time.Time

	// 完整性校验统计
	IntegrityChecks int64
	IntegrityAlerts int64

	// 网络监控统计
	NetguardScans       int64
	NetguardConnections int64
	NetguardAlerts      int64
	NetguardBlockedIPs  sync.Map // map[string]bool
}

var stats = &MonitorStats{}

// ==========================================
// 统一告警通道
// ==========================================

type AlertType string

const (
	AlertIntegrity AlertType = "INTEGRITY"
	AlertNetwork   AlertType = "NETWORK"
)

type Alert struct {
	Type      AlertType
	Timestamp time.Time
	Module    string
	Level     string // INFO, WARN, CRITICAL
	Title     string
	Details   map[string]string
}

var alertChan = make(chan Alert, 100)

// ==========================================
// 主入口
// ==========================================

func main() {
	if err := rootCmd.Execute(); err != nil {
		colorRed.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// ==========================================
// 根命令
// ==========================================

var rootCmd = &cobra.Command{
	Use:   appName,
	Short: "集成式安全监控调试工具",
	Long: `
███████╗███████╗ ██████╗██╗   ██╗██████╗ ██╗████████╗██╗   ██╗
██╔════╝██╔════╝██╔════╝██║   ██║██╔══██╗██║╚══██╔══╝╚██╗ ██╔╝
███████╗█████╗  ██║     ██║   ██║██████╔╝██║   ██║    ╚████╔╝ 
╚════██║██╔══╝  ██║     ██║   ██║██╔══██╗██║   ██║     ╚██╔╝  
███████║███████╗╚██████╗╚██████╔╝██║  ██║██║   ██║      ██║   
╚══════╝╚══════╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝   ╚═╝      ╚═╝   
                                                               
██╗    ██╗ ██████╗ ███╗   ██╗██╗████████╗ ██████╗ ██████╗      
███╗  ███║██╔═══██╗████╗  ██║██║╚══██╔══╝██╔═══██╗██╔══██╗     
██╔████╔██║██║   ██║██╔██╗ ██║██║   ██║   ██║   ██║██████╔╝     
██║╚██╔╝██║██║   ██║██║╚██╗██║██║   ██║   ██║   ██║██╔══██╗     
██║ ╚═╝ ██║╚██████╔╝██║ ╚████║██║   ██║   ╚██████╔╝██║  ██║     
╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝   ╚═╝    ╚═════╝ ╚═╝  ╚═╝     

集成式安全监控调试工具，整合多个安全子模块：
  • 完整性校验 (Integrity) - 监控文件篡改和删除
  • 网络防护 (NetGuard) - 监控异常网络连接

示例:
  # 启动所有模块
  security-monitor start --all

  # 仅启动完整性校验
  security-monitor start --enable-integrity --integrity-file /usr/bin/myapp

  # 仅启动网络监控
  security-monitor start --enable-netguard --netguard-pid 1234

  # 完整配置示例
  security-monitor start \
    --enable-integrity --integrity-file /opt/app/server --integrity-interval 30s \
    --enable-netguard --netguard-pid 1234 --netguard-interval 5s --dry-run
`,
	Version: version,
}

// ==========================================
// start 命令 - 启动监控
// ==========================================

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动集成安全监控",
	Long: `启动安全监控服务，可选择启用的模块。

至少需要启用一个模块 (--enable-integrity 或 --enable-netguard)，
或使用 --all 启用所有模块。`,
	RunE: runStart,
}

var enableAll bool

func runStart(cmd *cobra.Command, args []string) error {
	printBanner()

	// 处理 --all 参数
	if enableAll {
		enableIntegrity = true
		enableNetguard = true
	}

	// 验证至少启用一个模块
	if !enableIntegrity && !enableNetguard {
		colorRed.Println("❌ 错误: 至少需要启用一个监控模块")
		fmt.Println()
		colorYellow.Println("使用以下参数启用模块:")
		fmt.Println("  --all                启用所有模块")
		fmt.Println("  --enable-integrity   启用完整性校验")
		fmt.Println("  --enable-netguard    启用网络监控")
		return fmt.Errorf("no module enabled")
	}

	// 显示配置摘要
	printConfig()

	// 初始化统计
	stats.StartTime = time.Now()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动告警处理器
	go alertHandler()

	// 启动各模块
	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	if enableIntegrity {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runIntegrityMonitor(stopChan)
		}()
	}

	if enableNetguard {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runNetguardMonitor(stopChan)
		}()
	}

	// 启动状态显示（如果非静默模式）
	if !quietMode {
		go statusPrinter(stopChan)
	}

	printSeparator()
	colorMagenta.Println("🚀 安全监控已启动 (按 Ctrl+C 停止)")
	fmt.Println()

	// 等待停止信号
	<-sigChan
	fmt.Println()
	colorYellow.Println("🛑 收到停止信号，正在关闭...")

	// 通知所有 goroutine 停止
	close(stopChan)

	// 等待所有模块结束
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 超时等待
	select {
	case <-done:
		// 正常结束
	case <-time.After(5 * time.Second):
		colorYellow.Println("⚠️  部分模块未能及时停止")
	}

	// 打印最终统计
	printFinalStats()

	colorGreen.Println("👋 安全监控已停止")
	return nil
}

// ==========================================
// 完整性校验模块
// ==========================================

func runIntegrityMonitor(stopChan <-chan struct{}) {
	moduleName := "Integrity"

	// 解析目标文件
	targetFile := integrityFile
	if targetFile == "" {
		// 默认监控自身
		var err error
		targetFile, err = integrity.GetSelfExecutablePath()
		if err != nil {
			sendAlert(Alert{
				Type:      AlertIntegrity,
				Timestamp: time.Now(),
				Module:    moduleName,
				Level:     "CRITICAL",
				Title:     "模块初始化失败",
				Details:   map[string]string{"error": err.Error()},
			})
			return
		}
	} else {
		var err error
		targetFile, err = filepath.Abs(targetFile)
		if err != nil {
			sendAlert(Alert{
				Type:      AlertIntegrity,
				Timestamp: time.Now(),
				Module:    moduleName,
				Level:     "CRITICAL",
				Title:     "无效的文件路径",
				Details:   map[string]string{"error": err.Error()},
			})
			return
		}
	}

	// 计算基线哈希
	baselineHash, err := integrity.ComputeFileSM3(targetFile)
	if err != nil {
		sendAlert(Alert{
			Type:      AlertIntegrity,
			Timestamp: time.Now(),
			Module:    moduleName,
			Level:     "CRITICAL",
			Title:     "无法建立基线",
			Details:   map[string]string{"file": targetFile, "error": err.Error()},
		})
		return
	}

	if verboseMode {
		colorGreen.Printf("[%s] 基线已建立: %s\n", moduleName, baselineHash[:32]+"...")
	}

	// 启动监控循环
	ticker := time.NewTicker(integrityInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			checkIntegrity(targetFile, baselineHash, moduleName)
		}
	}
}

func checkIntegrity(targetFile, baselineHash, moduleName string) {
	atomic.AddInt64(&stats.IntegrityChecks, 1)

	// 检查文件是否存在
	_, err := os.Stat(targetFile)
	if err != nil {
		atomic.AddInt64(&stats.IntegrityAlerts, 1)

		alertTitle := "文件访问异常"
		if os.IsNotExist(err) {
			alertTitle = "文件已被删除"
		}

		sendAlert(Alert{
			Type:      AlertIntegrity,
			Timestamp: time.Now(),
			Module:    moduleName,
			Level:     "CRITICAL",
			Title:     alertTitle,
			Details: map[string]string{
				"file":  targetFile,
				"error": err.Error(),
			},
		})
		return
	}

	// 计算当前哈希
	currentHash, err := integrity.ComputeFileSM3(targetFile)
	if err != nil {
		atomic.AddInt64(&stats.IntegrityAlerts, 1)
		sendAlert(Alert{
			Type:      AlertIntegrity,
			Timestamp: time.Now(),
			Module:    moduleName,
			Level:     "WARN",
			Title:     "哈希计算失败",
			Details:   map[string]string{"file": targetFile, "error": err.Error()},
		})
		return
	}

	// 对比基线
	if currentHash != baselineHash {
		atomic.AddInt64(&stats.IntegrityAlerts, 1)
		sendAlert(Alert{
			Type:      AlertIntegrity,
			Timestamp: time.Now(),
			Module:    moduleName,
			Level:     "CRITICAL",
			Title:     "文件内容已被篡改",
			Details: map[string]string{
				"file":         targetFile,
				"baselineHash": baselineHash,
				"currentHash":  currentHash,
			},
		})
	}
}

// ==========================================
// 网络监控模块
// ==========================================

func runNetguardMonitor(stopChan <-chan struct{}) {
	moduleName := "NetGuard"

	// 解析目标 PID
	var pids []int32
	if len(netguardPIDs) > 0 {
		for _, p := range netguardPIDs {
			pids = append(pids, int32(p))
		}
	} else {
		// 默认监控自身
		pids = []int32{int32(os.Getpid())}
	}

	// 初始化白名单
	initialWhitelist := []string{"127.0.0.1", "::1"}
	if len(netguardWhitelist) > 0 {
		initialWhitelist = append(initialWhitelist, netguardWhitelist...)
	}

	whitelistMgr := netguard.NewWhitelistManager(initialWhitelist)
	scanner := detector.NewScanner(pids)
	blockedIPs := make(map[string]bool)

	if verboseMode {
		colorGreen.Printf("[%s] 监控 PID: %v, 白名单: %v\n", moduleName, pids, initialWhitelist)
	}

	// 启动监控循环
	ticker := time.NewTicker(netguardInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			scanNetwork(scanner, whitelistMgr, blockedIPs, moduleName)
		}
	}
}

func scanNetwork(scanner *detector.NetworkScanner, whitelist *netguard.WhitelistManager,
	blockedIPs map[string]bool, moduleName string) {

	atomic.AddInt64(&stats.NetguardScans, 1)

	connections, err := scanner.Scan()
	if err != nil {
		if verboseMode {
			colorRed.Printf("[%s] 扫描失败: %v\n", moduleName, err)
		}
		return
	}

	atomic.AddInt64(&stats.NetguardConnections, int64(len(connections)))

	for _, conn := range connections {
		// 跳过空 IP
		if conn.RemoteIP == "" || conn.RemoteIP == "0.0.0.0" || conn.RemoteIP == "::" {
			continue
		}

		// 检查白名单
		if !whitelist.IsAllowed(conn.RemoteIP) {
			// 去重检查
			if blockedIPs[conn.RemoteIP] {
				continue
			}

			blockedIPs[conn.RemoteIP] = true
			stats.NetguardBlockedIPs.Store(conn.RemoteIP, true)
			atomic.AddInt64(&stats.NetguardAlerts, 1)

			// 判断方向
			direction := "OUTBOUND"
			if conn.LocalPort < 1024 {
				direction = "INBOUND"
			}

			actionTaken := "DETECTED"
			if !netguardDryRun {
				actionTaken = "BLOCKED"
			}

			sendAlert(Alert{
				Type:      AlertNetwork,
				Timestamp: time.Now(),
				Module:    moduleName,
				Level:     "CRITICAL",
				Title:     "检测到异常网络连接",
				Details: map[string]string{
					"remoteIP":    conn.RemoteIP,
					"remotePort":  fmt.Sprintf("%d", conn.RemotePort),
					"localPort":   fmt.Sprintf("%d", conn.LocalPort),
					"protocol":    conn.Protocol,
					"direction":   direction,
					"pid":         fmt.Sprintf("%d", conn.PID),
					"status":      conn.Status,
					"actionTaken": actionTaken,
				},
			})
		}
	}
}

// ==========================================
// 告警处理器
// ==========================================

func sendAlert(alert Alert) {
	select {
	case alertChan <- alert:
	default:
		// 通道满了，丢弃告警（避免阻塞）
	}
}

func alertHandler() {
	for alert := range alertChan {
		printAlert(alert)
	}
}

func printAlert(alert Alert) {
	timestamp := alert.Timestamp.Format("2006-01-02 15:04:05")

	// 根据级别选择颜色
	headerColor := colorYellow
	levelIcon := "⚠️"
	if alert.Level == "CRITICAL" {
		headerColor = colorRed
		levelIcon = "🚨"
	}

	// 根据类型选择图标
	typeIcon := "📁"
	if alert.Type == AlertNetwork {
		typeIcon = "🌐"
	}

	fmt.Println()
	headerColor.Println("╔══════════════════════════════════════════════════════════════╗")
	headerColor.Printf("║  %s %s 安全告警 - %s\n", levelIcon, typeIcon, alert.Module)
	headerColor.Println("╠══════════════════════════════════════════════════════════════╣")
	headerColor.Printf("║  时间  : %s\n", timestamp)
	headerColor.Printf("║  级别  : %s\n", alert.Level)
	headerColor.Printf("║  标题  : %s\n", alert.Title)
	headerColor.Println("╠══════════════════════���═══════════════════════════════════════╣")

	// 打印详情
	for key, value := range alert.Details {
		// 截断过长的值
		if len(value) > 50 {
			value = value[:47] + "..."
		}
		headerColor.Printf("║  %-12s: %s\n", key, value)
	}

	headerColor.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// ==========================================
// 状态显示
// ==========================================

func statusPrinter(stopChan <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			if !quietMode && verboseMode {
				printStatus()
			}
		}
	}
}

func printStatus() {
	elapsed := time.Since(stats.StartTime).Round(time.Second)

	// 计算封禁 IP 数
	blockedCount := 0
	stats.NetguardBlockedIPs.Range(func(_, _ interface{}) bool {
		blockedCount++
		return true
	})

	colorBlue.Printf("\n📊 [状态更新] 运行时长: %v\n", elapsed)

	if enableIntegrity {
		colorWhite.Printf("   Integrity: 检查 %d 次, 告警 %d 次\n",
			atomic.LoadInt64(&stats.IntegrityChecks),
			atomic.LoadInt64(&stats.IntegrityAlerts))
	}

	if enableNetguard {
		colorWhite.Printf("   NetGuard:  扫描 %d 次, 连接 %d 个, 告警 %d 次, 封禁 IP %d 个\n",
			atomic.LoadInt64(&stats.NetguardScans),
			atomic.LoadInt64(&stats.NetguardConnections),
			atomic.LoadInt64(&stats.NetguardAlerts),
			blockedCount)
	}
	fmt.Println()
}

func printFinalStats() {
	elapsed := time.Since(stats.StartTime).Round(time.Second)

	blockedCount := 0
	var blockedList []string
	stats.NetguardBlockedIPs.Range(func(key, _ interface{}) bool {
		blockedCount++
		blockedList = append(blockedList, key.(string))
		return true
	})

	printSeparator()
	colorCyan.Println("📊 最终统计报告")
	printSeparator()

	colorWhite.Printf("   运行时长: %v\n", elapsed)
	fmt.Println()

	if enableIntegrity {
		colorWhite.Println("   【完整性校验】")
		colorWhite.Printf("      检查次数: %d\n", atomic.LoadInt64(&stats.IntegrityChecks))
		colorWhite.Printf("      告警次数: %d\n", atomic.LoadInt64(&stats.IntegrityAlerts))
		fmt.Println()
	}

	if enableNetguard {
		colorWhite.Println("   【网络监控】")
		colorWhite.Printf("      扫描次数: %d\n", atomic.LoadInt64(&stats.NetguardScans))
		colorWhite.Printf("      检测连接: %d\n", atomic.LoadInt64(&stats.NetguardConnections))
		colorWhite.Printf("      告警次数: %d\n", atomic.LoadInt64(&stats.NetguardAlerts))
		colorWhite.Printf("      封禁 IP:  %d\n", blockedCount)
		if len(blockedList) > 0 {
			colorWhite.Printf("      封禁列表: %s\n", strings.Join(blockedList, ", "))
		}
	}

	printSeparator()
}

// ==========================================
// config 命令 - 配置管理
// ==========================================

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "显示配置模板和帮助",
	RunE:  runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	printBanner()

	colorCyan.Println("📋 配置参数说明")
	printSeparator()

	fmt.Println(`
【模块开关】
  --all                 启用所有模块
  --enable-integrity    启用完整性校验模块
  --enable-netguard     启用网络监控模块

【完整性校验参数】
  --integrity-file      监控的目标文件路径 (默认: 程序自身)
  --integrity-interval  检查间隔 (默认: 30s)

【网络监控参数】
  --netguard-pid        监控的目标进程 PID，可多次指定 (默认: 自身)
  --netguard-interval   扫描间隔 (默认: 5s)
  --netguard-whitelist  白名单 IP/CIDR，可多次指定
  --dry-run             仅检测，不执行 iptables 封禁

【通用参数】
  --verbose, -v         详细输出模式
  --quiet, -q           静默模式，仅输出告警
`)

	printSeparator()
	colorCyan.Println("📝 使用示例")
	printSeparator()

	fmt.Println(`
# 1. 启动所有模块（默认配置）
security-monitor start --all

# 2. 仅监控指定文件的完整性
security-monitor start --enable-integrity \
  --integrity-file /opt/myapp/server \
  --integrity-interval 1m

# 3. 仅监控指定进程的网络连接
security-monitor start --enable-netguard \
  --netguard-pid 1234 \
  --netguard-interval 3s \
  --netguard-whitelist 192.168.1.0/24 \
  --netguard-whitelist 10.0.0.1 \
  --dry-run

# 4. 完整配置
security-monitor start \
  --enable-integrity \
  --integrity-file /opt/myapp/server \
  --integrity-interval 30s \
  --enable-netguard \
  --netguard-pid 1234 \
  --netguard-interval 5s \
  --netguard-whitelist 192.168.0.0/16 \
  --dry-run \
  --verbose
`)

	return nil
}

// ==========================================
// 辅助函数
// ==========================================

func printBanner() {
	fmt.Println()
	colorMagenta.Println("╔════════════════════════════════════════════════════════════╗")
	colorMagenta.Println("║            集成式安全监控调试工具 (Security Monitor)         ║")
	colorMagenta.Printf("║                        Version %s                         ║\n", version)
	colorMagenta.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func printSeparator() {
	colorWhite.Println("────────────────────────────────────────────────────────────────")
}

func printConfig() {
	colorCyan.Println("📋 当前配置")
	printSeparator()

	colorWhite.Printf("   本进程 PID: %d\n", os.Getpid())
	fmt.Println()

	// 完整性校验配置
	integrityStatus := "❌ 禁用"
	if enableIntegrity {
		integrityStatus = "✅ 启用"
	}
	colorWhite.Printf("   【完整性校验】 %s\n", integrityStatus)
	if enableIntegrity {
		targetDisplay := integrityFile
		if targetDisplay == "" {
			targetDisplay = "(程序自身)"
		}
		colorWhite.Printf("      目标文件: %s\n", targetDisplay)
		colorWhite.Printf("      检查间隔: %v\n", integrityInterval)
	}
	fmt.Println()

	// 网络监控配置
	netguardStatus := "❌ 禁用"
	if enableNetguard {
		netguardStatus = "✅ 启用"
	}
	colorWhite.Printf("   【网络监控】 %s\n", netguardStatus)
	if enableNetguard {
		pidDisplay := fmt.Sprintf("%v", netguardPIDs)
		if len(netguardPIDs) == 0 {
			pidDisplay = fmt.Sprintf("[%d] (自身)", os.Getpid())
		}
		colorWhite.Printf("      目标 PID: %s\n", pidDisplay)
		colorWhite.Printf("      扫描间隔: %v\n", netguardInterval)

		whitelist := append([]string{"127.0.0.1", "::1"}, netguardWhitelist...)
		colorWhite.Printf("      白名单:   %v\n", whitelist)

		if netguardDryRun {
			colorYellow.Println("      模式:     仅检测 (dry-run)")
		} else {
			colorRed.Println("      模式:     检测并封禁")
		}
	}
	fmt.Println()

	// 输出模式
	outputMode := "标准模式"
	if quietMode {
		outputMode = "静默模式"
	} else if verboseMode {
		outputMode = "详细模式"
	}
	colorWhite.Printf("   输出模式: %s\n", outputMode)
}

// ==========================================
// 初始化
// ==========================================

func init() {
	// start 命令参数
	startCmd.Flags().BoolVar(&enableAll, "all", false, "启用所有监控模块")
	startCmd.Flags().BoolVar(&enableIntegrity, "enable-integrity", false, "启用完整性校验模块")
	startCmd.Flags().BoolVar(&enableNetguard, "enable-netguard", false, "启用网络监控模块")

	// 完整性校验参数
	startCmd.Flags().StringVar(&integrityFile, "integrity-file", "", "完整性校验目标文件 (默认: 程序自身)")
	startCmd.Flags().DurationVar(&integrityInterval, "integrity-interval", 30*time.Second, "完整性检查间隔")

	// 网络监控参数
	startCmd.Flags().IntSliceVar(&netguardPIDs, "netguard-pid", nil, "网络监控目标 PID (可多次指定)")
	startCmd.Flags().DurationVar(&netguardInterval, "netguard-interval", 5*time.Second, "网络扫描间隔")
	startCmd.Flags().StringSliceVar(&netguardWhitelist, "netguard-whitelist", nil, "网络白名单 IP/CIDR (可多次指定)")
	startCmd.Flags().BoolVar(&netguardDryRun, "dry-run", false, "仅检测，不执行封禁")

	// 通用参数
	startCmd.Flags().BoolVarP(&verboseMode, "verbose", "v", false, "详细输出模式")
	startCmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "静默模式")

	// 注册命令
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(configCmd)
}
