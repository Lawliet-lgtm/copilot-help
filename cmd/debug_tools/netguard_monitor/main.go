// Package main 提供网络连接监控模块的独立调试工具
// 用于单独测试和排查 internal/security/netguard 子模块的逻辑问题
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"linuxFileWatcher/internal/security/netguard"
	"linuxFileWatcher/internal/security/netguard/detector"
	"linuxFileWatcher/internal/security/netguard/event"
)

// ==========================================
// 全局变量和配置
// ==========================================

var (
	// 版本信息
	version = "1.0.0"
	appName = "netguard-monitor"

	// 命令行参数
	targetPIDs   []int
	scanInterval time.Duration
	verboseMode  bool
	quietMode    bool
	dryRunMode   bool
	whitelistIPs []string

	// 颜色输出
	colorRed     = color.New(color.FgRed, color.Bold)
	colorGreen   = color.New(color.FgGreen, color.Bold)
	colorYellow  = color.New(color.FgYellow)
	colorCyan    = color.New(color.FgCyan)
	colorMagenta = color.New(color.FgMagenta)
	colorWhite   = color.New(color.FgWhite)
)

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
	Short: "网络连接监控模块调试工具",
	Long: `
███╗   ██╗███████╗████████╗ ██████╗ ██╗   ██╗ █████╗ ██████╗ ██████╗ 
████╗  ██║██╔════╝╚══██╔══╝██╔════╝ ██║   ██║██╔══██╗██╔══██╗██╔══██╗
██╔██╗ ██║█████╗     ██║   ██║  ███╗██║   ██║███████║██████╔╝██║  ██║
██║╚██╗██║██╔══╝     ██║   ██║   ██║██║   ██║██╔══██║██╔══██╗██║  ██║
██║ ╚████║███████╗   ██║   ╚██████╔╝╚██████╔╝██║  ██║██║  ██║██████╔╝
╚═╝  ╚═══╝╚══════╝   ╚═╝    ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ 
                                                                      
网络连接监控模块 (netguard) 的独立调试工具。

用于单独测试和排查网络连接监控逻辑，支持：
  - 单次扫描：执行一次网络连接扫描并显示结果
  - 持续监控：周期性扫描进程网络连接，检测异常
  - 白名单管理：查看和测试白名单规则

示例:
  # 扫描当前进程的网络连接
  netguard-monitor scan

  # 扫描指定 PID 的网络连接
  netguard-monitor scan --pid 1234

  # 启动持续监控（仅检测不封禁）
  netguard-monitor watch --interval 5s --dry-run

  # 添加白名单并监控
  netguard-monitor watch --whitelist 192.168.1.0/24,10.0.0.1
`,
	Version: version,
}

// ==========================================
// scan 命令 - 单次扫描
// ==========================================

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "执行一次网络连接扫描",
	Long: `扫描指定进程的所有网络连接并以表格形式展示。

如果不指定 --pid，默认扫描当前程序自身。
可以同时指定多个 PID: --pid 1234 --pid 5678`,
	RunE: runScan,
}

func runScan(cmd *cobra.Command, args []string) error {
	printBanner()

	// 确定目标 PID
	pids := resolveTargetPIDs()

	colorCyan.Printf("🔍 扫描目标 PID: %v\n", pids)
	printSeparator()

	// 创建扫描器
	scanner := detector.NewScanner(pids)

	// 执行扫描
	colorYellow.Println("🔄 正在扫描网络连接...")
	startTime := time.Now()

	connections, err := scanner.Scan()
	if err != nil {
		colorRed.Printf("❌ 扫描失败: %v\n", err)
		return err
	}

	elapsed := time.Since(startTime)

	colorGreen.Printf("✅ 扫描完成! (耗时: %v)\n", elapsed)
	printSeparator()

	if len(connections) == 0 {
		colorYellow.Println("📭 未发现活跃的网络连接")
		return nil
	}

	// 显示连接列表
	colorCyan.Printf("📊 发现 %d 个活跃连接:\n", len(connections))
	fmt.Println()

	printConnectionTable(connections)

	// 统计信息
	printSeparator()
	printConnectionStats(connections)

	return nil
}

// ==========================================
// watch 命令 - 持续监控
// ==========================================

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "启动持续监控模式",
	Long: `启动后台监控，周期性扫描进程网络连接。

当检测到白名单外的连接时，会输出告警信息。
按 Ctrl+C 停止监控。

模式说明:
  --dry-run: 仅检测，不执行 iptables 封禁（推荐调试时使用）
  默认模式: 检测到异常会尝试封禁（需要 root 权限）`,
	RunE: runWatch,
}

func runWatch(cmd *cobra.Command, args []string) error {
	printBanner()

	// 确定目标 PID
	pids := resolveTargetPIDs()

	colorCyan.Printf("🔍 监控目标 PID: %v\n", pids)
	colorCyan.Printf("⏱️  扫描间隔: %v\n", scanInterval)

	if dryRunMode {
		colorYellow.Println("🔒 运行模式: 仅检测 (dry-run)")
	} else {
		colorRed.Println("🔒 运行模式: 检测并封禁 (需要 root 权限)")
	}

	if quietMode {
		colorCyan.Println("🔇 输出模式: 静默模式（仅显示异常）")
	} else if verboseMode {
		colorCyan.Println("📢 输出模式: 详细模式")
	} else {
		colorCyan.Println("📢 输出模式: 标准模式")
	}

	printSeparator()

	// 初始化白名单
	initialWhitelist := []string{"127.0.0.1", "::1"}
	if len(whitelistIPs) > 0 {
		initialWhitelist = append(initialWhitelist, whitelistIPs...)
	}

	colorCyan.Println("📋 白名单规则:")
	for _, ip := range initialWhitelist {
		fmt.Printf("   • %s\n", ip)
	}
	printSeparator()

	// 创建白名单管理器
	whitelistMgr := netguard.NewWhitelistManager(initialWhitelist)

	// 创建扫描器
	scanner := detector.NewScanner(pids)

	// 创建 Reporter
	reporter := &DebugReporter{dryRun: dryRunMode}

	colorMagenta.Println("👀 开始持续监控... (按 Ctrl+C 停止)")
	fmt.Println()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动监控循环
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	scanCount := 0
	alertCount := 0
	totalConnections := 0
	blockedIPs := make(map[string]bool) // 去重缓存
	startTime := time.Now()

	for {
		select {
		case <-sigChan:
			fmt.Println()
			printSeparator()
			colorYellow.Println("🛑 收到停止信号，正在退出...")
			colorWhite.Printf("   总运行时间   : %v\n", time.Since(startTime).Round(time.Second))
			colorWhite.Printf("   扫描次数     : %d\n", scanCount)
			colorWhite.Printf("   检测连接总数 : %d\n", totalConnections)
			colorWhite.Printf("   告警次数     : %d\n", alertCount)
			colorWhite.Printf("   封禁 IP 数   : %d\n", len(blockedIPs))
			colorGreen.Println("👋 监控已停止")
			return nil

		case <-ticker.C:
			scanCount++
			alerts, connCount := performNetworkScan(scanner, whitelistMgr, reporter, scanCount, blockedIPs)
			alertCount += alerts
			totalConnections += connCount
		}
	}
}

// performNetworkScan 执行一次网络扫描
// 返回值: (告警数, 连接数)
func performNetworkScan(scanner *detector.NetworkScanner, whitelist *netguard.WhitelistManager,
	reporter *DebugReporter, count int, blockedIPs map[string]bool) (int, int) {

	timestamp := time.Now().Format("15:04:05")

	// 1. 扫描连接
	connections, err := scanner.Scan()
	if err != nil {
		if !quietMode {
			colorRed.Printf("[%s] ❌ 扫描失败: %v\n", timestamp, err)
		}
		return 0, 0
	}

	connCount := len(connections)
	alertCount := 0
	violationCount := 0

	// 2. 检查每个连接
	for _, conn := range connections {
		// 跳过空 IP（可能是 LISTEN 状态的残留）
		if conn.RemoteIP == "" || conn.RemoteIP == "0.0.0.0" || conn.RemoteIP == "::" {
			continue
		}

		// 检查白名单
		if !whitelist.IsAllowed(conn.RemoteIP) {
			violationCount++

			// 去重检查
			if blockedIPs[conn.RemoteIP] {
				continue
			}

			// 记录并上报
			blockedIPs[conn.RemoteIP] = true
			alertCount++

			// 构建告警
			alert := event.NetworkAlert{
				Timestamp:   time.Now(),
				AlertTime:   time.Now().Unix(),
				Direction:   determineDirection(conn),
				RemoteIP:    conn.RemoteIP,
				RemotePort:  uint16(conn.RemotePort),
				LocalPort:   uint16(conn.LocalPort),
				Protocol:    conn.Protocol,
				PID:         conn.PID,
				ActionTaken: "DETECTED",
			}

			if !dryRunMode {
				alert.ActionTaken = "BLOCKED"
			}

			reporter.Report(alert)
		}
	}

	// 3. 输出状态
	if !quietMode {
		if violationCount > 0 {
			colorYellow.Printf("[%s] 扫描 #%d | 连接数: %d | 违规: %d | 新告警: %d\n",
				timestamp, count, connCount, violationCount, alertCount)
		} else if verboseMode {
			colorGreen.Printf("[%s] ✓ 扫描 #%d 通过 | 连接数: %d | 全部在白名单内\n",
				timestamp, count, connCount)
		} else {
			colorGreen.Printf("[%s] ✓ 扫描 #%d 通过 | 连接数: %d\n",
				timestamp, count, connCount)
		}
	}

	return alertCount, connCount
}

// determineDirection 判断连接方向
func determineDirection(conn detector.ConnectionInfo) event.TrafficDirection {
	// 简单判断：如果本地端口小于 1024，通常是服务端（被动接收）
	if conn.LocalPort < 1024 {
		return event.DirectionInbound
	}
	return event.DirectionOutbound
}

// ==========================================
// whitelist 命令 - 白名单管理
// ==========================================

var whitelistCmd = &cobra.Command{
	Use:   "whitelist",
	Short: "白名单管理和测试",
	Long: `查看默认白名单规则，或测试 IP 是否匹配白名单。

示例:
  # 查看默认白名单
  netguard-monitor whitelist list

  # 测试 IP 是否在白名单中
  netguard-monitor whitelist test 192.168.1.100

  # 测试带自定义白名单
  netguard-monitor whitelist test 10.0.0.5 --whitelist 10.0.0.0/8`,
}

var whitelistListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出默认白名单规则",
	RunE:  runWhitelistList,
}

var whitelistTestCmd = &cobra.Command{
	Use:   "test [IP]",
	Short: "测试 IP 是否匹配白名单",
	Args:  cobra.ExactArgs(1),
	RunE:  runWhitelistTest,
}

func runWhitelistList(cmd *cobra.Command, args []string) error {
	printBanner()

	colorCyan.Println("📋 默认白名单规则:")
	printSeparator()

	defaultRules := []struct {
		rule string
		desc string
	}{
		{"127.0.0.1", "IPv4 本地回环"},
		{"::1", "IPv6 本地回环"},
	}

	// 添加用户自定义规则
	for _, ip := range whitelistIPs {
		defaultRules = append(defaultRules, struct {
			rule string
			desc string
		}{ip, "用户自定义"})
	}

	// 使用纯文本表格，避免依赖 tablewriter
	fmt.Println()
	fmt.Printf("  %-25s %-12s %s\n", "规则", "类型", "说明")
	fmt.Println("  " + strings.Repeat("-", 55))

	for _, r := range defaultRules {
		ruleType := "精确IP"
		if strings.Contains(r.rule, "/") {
			ruleType = "CIDR网段"
		}
		fmt.Printf("  %-25s %-12s %s\n", r.rule, ruleType, r.desc)
	}
	fmt.Println()

	return nil
}

func runWhitelistTest(cmd *cobra.Command, args []string) error {
	printBanner()

	testIP := args[0]

	// 验证 IP 格式
	if net.ParseIP(testIP) == nil && !strings.Contains(testIP, "/") {
		colorRed.Printf("❌ 无效的 IP 地址: %s\n", testIP)
		return fmt.Errorf("invalid IP address")
	}

	colorCyan.Printf("🧪 测试 IP: %s\n", testIP)
	printSeparator()

	// 初始化白名单
	initialWhitelist := []string{"127.0.0.1", "::1"}
	if len(whitelistIPs) > 0 {
		initialWhitelist = append(initialWhitelist, whitelistIPs...)
	}

	colorCyan.Println("📋 当前白名单规则:")
	for _, ip := range initialWhitelist {
		fmt.Printf("   • %s\n", ip)
	}
	printSeparator()

	// 创建白名单管理器并测试
	whitelistMgr := netguard.NewWhitelistManager(initialWhitelist)
	allowed := whitelistMgr.IsAllowed(testIP)

	if allowed {
		colorGreen.Printf("✅ 结果: IP %s 在白名单中 (允许通过)\n", testIP)
	} else {
		colorRed.Printf("❌ 结果: IP %s 不在白名单中 (将被拦截)\n", testIP)
	}

	return nil
}

// ==========================================
// connections 命令 - 显示当前连接
// ==========================================

var connectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "显示系统当前所有网络连接",
	Long: `扫描并显示指定进程的所有网络连接详情。

支持按状态、协议、IP 过滤和排序。`,
	RunE: runConnections,
}

func runConnections(cmd *cobra.Command, args []string) error {
	printBanner()

	pids := resolveTargetPIDs()

	colorCyan.Printf("🔍 目标 PID: %v\n", pids)
	printSeparator()

	scanner := detector.NewScanner(pids)
	connections, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("扫描失败: %v", err)
	}

	if len(connections) == 0 {
		colorYellow.Println("📭 未发现活跃的网络连接")
		return nil
	}

	// 初始化白名单用于标记
	initialWhitelist := []string{"127.0.0.1", "::1"}
	if len(whitelistIPs) > 0 {
		initialWhitelist = append(initialWhitelist, whitelistIPs...)
	}
	whitelistMgr := netguard.NewWhitelistManager(initialWhitelist)

	colorCyan.Printf("📊 发现 %d 个连接:\n", len(connections))
	fmt.Println()

	printConnectionTableWithStatus(connections, whitelistMgr)

	printSeparator()
	printConnectionStats(connections)

	return nil
}

// ==========================================
// 自定义 Reporter 实现
// ==========================================

// DebugReporter 调试用的告警上报器
type DebugReporter struct {
	dryRun bool
}

// Report 上报网络告警
func (r *DebugReporter) Report(alert event.NetworkAlert) error {
	timestamp := alert.Timestamp.Format("2006-01-02 15:04:05")

	fmt.Println()

	headerColor := colorRed
	actionText := "已封禁"
	if r.dryRun {
		headerColor = colorYellow
		actionText = "仅检测(dry-run)"
	}

	headerColor.Println("╔══════════════════════════════════════════════════════════════╗")
	headerColor.Println("║                    ⚠️  网络安全告警 ⚠️                        ║")
	headerColor.Println("╠══════════════════════════════════════════════════════════════╣")
	headerColor.Printf("║  时间     : %-50s ║\n", timestamp)
	headerColor.Printf("║  动作     : %-50s ║\n", actionText)
	headerColor.Println("╠══════════════════════════════════════════════════════════════╣")
	headerColor.Printf("║  远程地址 : %-50s ║\n", fmt.Sprintf("%s:%d", alert.RemoteIP, alert.RemotePort))
	headerColor.Printf("║  本地端口 : %-50d ║\n", alert.LocalPort)
	headerColor.Printf("║  协议     : %-50s ║\n", alert.Protocol)
	headerColor.Printf("║  方向     : %-50s ║\n", alert.Direction)
	headerColor.Printf("║  进程 PID : %-50d ║\n", alert.PID)
	headerColor.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	return nil
}

// ==========================================
// 辅助函数
// ==========================================

// resolveTargetPIDs 解析目标 PID 列表
func resolveTargetPIDs() []int32 {
	if len(targetPIDs) > 0 {
		pids := make([]int32, len(targetPIDs))
		for i, p := range targetPIDs {
			pids[i] = int32(p)
		}
		return pids
	}
	// 默认监控自身
	return []int32{int32(os.Getpid())}
}

// printConnectionTable 打印连接表格（纯文本实现，无外部依赖）
func printConnectionTable(connections []detector.ConnectionInfo) {
	// 表头
	fmt.Printf("  %-4s %-6s %-10s %-20s %-10s %-12s %-8s\n",
		"#", "协议", "本地端口", "远程地址", "远程端口", "状态", "PID")
	fmt.Println("  " + strings.Repeat("-", 75))

	// 数据行
	for i, conn := range connections {
		remoteIP := conn.RemoteIP
		if remoteIP == "" {
			remoteIP = "-"
		}

		// 截断过长的 IP
		if len(remoteIP) > 18 {
			remoteIP = remoteIP[:15] + "..."
		}

		fmt.Printf("  %-4d %-6s %-10d %-20s %-10d %-12s %-8d\n",
			i+1,
			conn.Protocol,
			conn.LocalPort,
			remoteIP,
			conn.RemotePort,
			conn.Status,
			conn.PID,
		)
	}
	fmt.Println()
}

// printConnectionTableWithStatus 打印带白名单状态的连接表格
func printConnectionTableWithStatus(connections []detector.ConnectionInfo, whitelist *netguard.WhitelistManager) {
	// 表头
	fmt.Printf("  %-4s %-6s %-10s %-20s %-10s %-12s %-8s %-8s\n",
		"#", "协议", "本地端口", "远程地址", "远程端口", "状态", "PID", "白名单")
	fmt.Println("  " + strings.Repeat("-", 85))

	// 数据行
	for i, conn := range connections {
		remoteIP := conn.RemoteIP
		if remoteIP == "" {
			remoteIP = "-"
		}

		// 检查白名单状态
		whitelistStatus := "❌ 否"
		if remoteIP == "-" || remoteIP == "0.0.0.0" || remoteIP == "::" {
			whitelistStatus = "➖ N/A"
		} else if whitelist.IsAllowed(remoteIP) {
			whitelistStatus = "✅ 是"
		}

		// 截断过长的 IP
		displayIP := remoteIP
		if len(displayIP) > 18 {
			displayIP = displayIP[:15] + "..."
		}

		fmt.Printf("  %-4d %-6s %-10d %-20s %-10d %-12s %-8d %-8s\n",
			i+1,
			conn.Protocol,
			conn.LocalPort,
			displayIP,
			conn.RemotePort,
			conn.Status,
			conn.PID,
			whitelistStatus,
		)
	}
	fmt.Println()
}

// printConnectionStats 打印连接统计信息
func printConnectionStats(connections []detector.ConnectionInfo) {
	// 统计协议分布
	protoStats := make(map[string]int)
	statusStats := make(map[string]int)
	uniqueIPs := make(map[string]bool)

	for _, conn := range connections {
		protoStats[conn.Protocol]++
		statusStats[conn.Status]++
		if conn.RemoteIP != "" && conn.RemoteIP != "0.0.0.0" && conn.RemoteIP != "::" {
			uniqueIPs[conn.RemoteIP] = true
		}
	}

	colorCyan.Println("📈 统计信息:")
	fmt.Printf("   总连接数    : %d\n", len(connections))
	fmt.Printf("   唯一远程 IP : %d\n", len(uniqueIPs))

	// 协议统计
	fmt.Print("   协议分布    : ")
	var protoList []string
	for proto, count := range protoStats {
		protoList = append(protoList, fmt.Sprintf("%s(%d)", proto, count))
	}
	sort.Strings(protoList)
	fmt.Println(strings.Join(protoList, ", "))

	// 状态统计
	fmt.Print("   状态分布    : ")
	var statusList []string
	for status, count := range statusStats {
		statusList = append(statusList, fmt.Sprintf("%s(%d)", status, count))
	}
	sort.Strings(statusList)
	fmt.Println(strings.Join(statusList, ", "))
}

// printBanner 打印工具标题
func printBanner() {
	fmt.Println()
	colorMagenta.Println("╔════════════════════════════════════════════════════════════╗")
	colorMagenta.Println("║          网络连接监控调试工具 (NetGuard Monitor)             ║")
	colorMagenta.Printf("║                       Version %s                          ║\n", version)
	colorMagenta.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// printSeparator 打印分隔线
func printSeparator() {
	colorWhite.Println("────────────────────────────────────────────────────────────────")
}

// ==========================================
// 初始化
// ==========================================

func init() {
	// 全局参数
	rootCmd.PersistentFlags().IntSliceVarP(&targetPIDs, "pid", "p", nil, "目标进程 PID (可多次指定，默认: 当前进程)")
	rootCmd.PersistentFlags().BoolVarP(&verboseMode, "verbose", "v", false, "启用详细输出模式")
	rootCmd.PersistentFlags().StringSliceVarP(&whitelistIPs, "whitelist", "w", nil, "白名单 IP 或 CIDR (可多次指定)")

	// watch 命令参数
	watchCmd.Flags().DurationVarP(&scanInterval, "interval", "i", 5*time.Second, "扫描间隔时间 (如: 5s, 1m)")
	watchCmd.Flags().BoolVarP(&quietMode, "quiet", "q", false, "静默模式，仅在异常时输出")
	watchCmd.Flags().BoolVarP(&dryRunMode, "dry-run", "d", false, "仅检测，不执行封禁")

	// 注册子命令
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(connectionsCmd)

	// whitelist 子命令
	whitelistCmd.AddCommand(whitelistListCmd)
	whitelistCmd.AddCommand(whitelistTestCmd)
	rootCmd.AddCommand(whitelistCmd)
}
