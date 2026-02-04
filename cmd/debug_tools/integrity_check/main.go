// Package main 提供完整性校验模块的独立调试工具
// 用于单独测试和排查 internal/security/integrity 子模块的逻辑问题
package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"linuxFileWatcher/internal/security/integrity"
)

// ==========================================
// 全局变量和配置
// ==========================================

var (
	// 版本信息
	version = "1.0.0"
	appName = "integrity-checker"

	// 命令行参数
	targetFile    string
	checkInterval time.Duration
	verboseMode   bool

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
	Short: "完整性校验模块调试工具",
	Long: `
██╗███╗   ██╗████████╗███████╗ ██████╗ ██████╗ ██╗████████╗██╗   ██╗
██║████╗  ██║╚══██╔══╝██╔════╝██╔════╝ ██╔══██╗██║╚══██╔══╝╚██╗ ██╔╝
██║██╔██╗ ██║   ██║   █████╗  ██║  ███╗██████╔╝██║   ██║    ╚████╔╝ 
██║██║╚██╗██║   ██║   ██╔══╝  ██║   ██║██╔══██╗██║   ██║     ╚██╔╝  
██║██║ ╚████║   ██║   ███████╗╚██████╔╝██║  ██║██║   ██║      ██║   
╚═╝╚═╝  ╚═══╝   ╚═╝   ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝   ╚═╝      ╚═╝   
                                                                     
完整性校验模块 (integrity) 的独立调试工具。

用于单独测试和排查文件完整性校验逻辑，支持：
  - 单次校验：对指定文件执行一次 SM3 哈希计算
  - 持续监控：周期性检查文件是否被篡改或删除
  - 基线生成：生成文件的基线哈希值

示例:
  # 检查指定文件的完整性
  integrity-checker check --file /usr/bin/myapp

  # 启动持续监控模式
  integrity-checker watch --file /usr/bin/myapp --interval 30s

  # 生成基线哈希
  integrity-checker baseline --file /usr/bin/myapp
`,
	Version: version,
}

// ==========================================
// check 命令 - 单次校验
// ==========================================

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "对指定文件执行一次完整性校验",
	Long: `对指定文件执行一次 SM3 哈希计算并显示结果。

如果不指定 --file，默认检查当前程序自身。`,
	RunE: runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	printBanner()

	// 确定目标文件
	target, err := resolveTargetFile()
	if err != nil {
		return err
	}

	colorCyan.Printf("📁 目标文件: %s\n", target)
	printSeparator()

	// 检查文件是否存在
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			colorRed.Println("❌ 文件不存在!")
			return fmt.Errorf("file not found: %s", target)
		}
		return fmt.Errorf("无法访问文件: %v", err)
	}

	// 显示文件信息
	printFileInfo(target, info)
	printSeparator()

	// 计算 SM3 哈希
	colorYellow.Println("🔄 正在计算 SM3 哈希...")
	startTime := time.Now()

	hash, err := integrity.ComputeFileSM3(target)
	if err != nil {
		colorRed.Printf("❌ 哈希计算失败: %v\n", err)
		return err
	}

	elapsed := time.Since(startTime)

	colorGreen.Println("✅ 校验完成!")
	fmt.Println()
	colorWhite.Printf("   SM3 Hash : %s\n", hash)
	colorWhite.Printf("   计算耗时 : %v\n", elapsed)
	colorWhite.Printf("   文件大小 : %s\n", formatFileSize(info.Size()))

	printSeparator()
	colorGreen.Println("📋 校验结果: 文件完整性正常")

	return nil
}

// ==========================================
// baseline 命令 - 生成基线
// ==========================================

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "生成文件的基线哈希值",
	Long: `计算指定文件的 SM3 哈希值，用于建立完整性校验基线。

输出格式适合保存到配置文件或用于后续对比。`,
	RunE: runBaseline,
}

func runBaseline(cmd *cobra.Command, args []string) error {
	printBanner()

	target, err := resolveTargetFile()
	if err != nil {
		return err
	}

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("无法访问文件: %v", err)
	}

	hash, err := integrity.ComputeFileSM3(target)
	if err != nil {
		return fmt.Errorf("哈希计算失败: %v", err)
	}

	colorCyan.Println("📊 基线信息:")
	printSeparator()

	fmt.Printf("文件路径    : %s\n", target)
	fmt.Printf("文件大小    : %s (%d bytes)\n", formatFileSize(info.Size()), info.Size())
	fmt.Printf("修改时间    : %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	fmt.Printf("SM3 哈希    : %s\n", hash)

	printSeparator()

	// 输出可复制的格式
	colorYellow.Println("📋 可复制格式 (YAML):")
	fmt.Printf(`
integrity_baseline:
  path: "%s"
  hash: "%s"
  size: %d
  generated_at: "%s"
`, target, hash, info.Size(), time.Now().Format(time.RFC3339))

	return nil
}

// ==========================================
// watch 命令 - 持续监控
// ==========================================

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "启动持续监控模式",
	Long: `启动后台监控，周期性检查文件完整性。

当检测到文件被篡改或删除时，会输出告警信息。
按 Ctrl+C 停止监控。`,
	RunE: runWatch,
}

func runWatch(cmd *cobra.Command, args []string) error {
	printBanner()

	target, err := resolveTargetFile()
	if err != nil {
		return err
	}

	colorCyan.Printf("📁 监控目标: %s\n", target)
	colorCyan.Printf("⏱️  检查间隔: %v\n", checkInterval)
	printSeparator()

	// 计算初始基线
	colorYellow.Println("🔄 正在建立基线...")

	baselineHash, err := integrity.ComputeFileSM3(target)
	if err != nil {
		return fmt.Errorf("无法建立基线: %v", err)
	}

	colorGreen.Printf("✅ 基线已建立: %s\n", baselineHash)
	printSeparator()

	// 创建自定义 Reporter
	reporter := &DebugReporter{verbose: verboseMode}

	colorMagenta.Println("👀 开始持续监控... (按 Ctrl+C 停止)")
	fmt.Println()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动监控循环
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	checkCount := 0
	startTime := time.Now()

	for {
		select {
		case <-sigChan:
			fmt.Println()
			printSeparator()
			colorYellow.Println("🛑 收到停止信号，正在退出...")
			colorWhite.Printf("   总运行时间: %v\n", time.Since(startTime).Round(time.Second))
			colorWhite.Printf("   检查次数: %d\n", checkCount)
			colorGreen.Println("👋 监控已停止")
			return nil

		case <-ticker.C:
			checkCount++
			performIntegrityCheck(target, baselineHash, reporter, checkCount)
		}
	}
}

// performIntegrityCheck 执行一次完整性检查
func performIntegrityCheck(target, baselineHash string, reporter *DebugReporter, count int) {
	timestamp := time.Now().Format("15:04:05")

	if verboseMode {
		colorWhite.Printf("[%s] 第 %d 次检查...\n", timestamp, count)
	}

	// 1. 检查文件是否存在
	_, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			reporter.Report(integrity.TypeFileDeleted, fmt.Sprintf("文件已删除: %s", target))
		} else {
			reporter.Report(integrity.TypeReadError, fmt.Sprintf("无法访问文件: %v", err))
		}
		return
	}

	// 2. 计算当前哈希
	currentHash, err := integrity.ComputeFileSM3(target)
	if err != nil {
		reporter.Report(integrity.TypeReadError, fmt.Sprintf("哈希计算失败: %v", err))
		return
	}

	// 3. 对比基线
	if currentHash != baselineHash {
		reporter.Report(integrity.TypeFileModified, fmt.Sprintf(
			"文件内容已变更!\n   基线哈希: %s\n   当前哈希: %s",
			baselineHash, currentHash))
		return
	}

	// 正常
	if verboseMode {
		colorGreen.Printf("[%s] ✓ 检查通过 (Hash: %s...)\n", timestamp, currentHash[:16])
	}
}

// ==========================================
// 自定义 Reporter 实现
// ==========================================

// DebugReporter 调试用的告警上报器
type DebugReporter struct {
	verbose bool
}

// Report 实现 integrity.Reporter 接口
func (r *DebugReporter) Report(vType integrity.ViolationType, msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	fmt.Println()
	colorRed.Println("╔══════════════════════════════════════════════════════════════╗")
	colorRed.Println("║                    ⚠️  安全告警 ⚠️                            ║")
	colorRed.Println("╠══════════════════════════════════════════════════════════════╣")
	colorRed.Printf("║  时间: %s                              ║\n", timestamp)
	colorRed.Printf("║  类型: %-54s ║\n", vType)
	colorRed.Println("╠══════════════════════════════════════════════════════════════╣")

	// 分行显示消息
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		// 截断过长的行
		if len(line) > 60 {
			line = line[:57] + "..."
		}
		colorRed.Printf("║  %-62s ║\n", line)
	}

	colorRed.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// ==========================================
// 辅助函数
// ==========================================

// resolveTargetFile 解析目标文件路径
func resolveTargetFile() (string, error) {
	if targetFile != "" {
		// 用户指定了文件
		absPath, err := filepath.Abs(targetFile)
		if err != nil {
			return "", fmt.Errorf("无法解析路径: %v", err)
		}
		return absPath, nil
	}

	// 默认使用当前程序自身
	selfPath, err := integrity.GetSelfExecutablePath()
	if err != nil {
		return "", fmt.Errorf("无法获取自身路径: %v", err)
	}
	return selfPath, nil
}

// printBanner 打印工具标题
func printBanner() {
	fmt.Println()
	colorMagenta.Println("╔════════════════════════════════════════════════════════╗")
	colorMagenta.Println("║         完整性校验模块调试工具 (Integrity Checker)       ║")
	colorMagenta.Printf("║                      Version %s                       ║\n", version)
	colorMagenta.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// printSeparator 打印分隔线
func printSeparator() {
	colorWhite.Println("────────────────────────────────────────────────────────────")
}

// printFileInfo 打印文件详细信息
func printFileInfo(path string, info os.FileInfo) {
	colorCyan.Println("📋 文件信息:")
	fmt.Printf("   名称     : %s\n", info.Name())
	fmt.Printf("   大小     : %s (%d bytes)\n", formatFileSize(info.Size()), info.Size())
	fmt.Printf("   修改时间 : %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	// 检查是否为符号链接
	if info.Mode()&os.ModeSymlink != 0 {
		if realPath, err := filepath.EvalSymlinks(path); err == nil {
			fmt.Printf("   实际路径 : %s (符号链接)\n", realPath)
		}
	}
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// ==========================================
// 初始化
// ==========================================

func init() {
	// 全局参数
	rootCmd.PersistentFlags().StringVarP(&targetFile, "file", "f", "", "要检查的目标文件路径 (默认: 当前程序自身)")
	rootCmd.PersistentFlags().BoolVarP(&verboseMode, "verbose", "v", false, "启用详细输出模式")

	// watch 命令特有参数
	watchCmd.Flags().DurationVarP(&checkInterval, "interval", "i", 30*time.Second, "检查间隔时间 (如: 10s, 1m, 5m)")

	// 注册子命令
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(baselineCmd)
	rootCmd.AddCommand(watchCmd)
}
