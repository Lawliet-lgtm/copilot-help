// Package main Detector 模块集成调试工具
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"linuxFileWatcher/internal/detector"
	"linuxFileWatcher/internal/model"
)

// ==========================================
// 命令行参数
// ==========================================

var (
	targetPath string
	recursive  bool

	// 模块开关 - 注意这些变量会被 flag 和 resolveModuleFlags 修改
	enableSecretMarker bool
	enableLayout       bool
	enableHash         bool
	enableElectronic   bool
	enableKeywords     bool
	enableAll          bool
	disableAll         bool

	hashRulesFile   string
	streamRulesFile string

	workers     int
	timeout     int
	maxFileSize int64

	outputFile   string
	outputFormat string
	verbose      bool
	quiet        bool
	showProgress bool
	showConfig   bool

	showHelp    bool
	showVersion bool
)

const (
	toolName    = "detector-debug"
	toolVersion = "1.0.0"
)

func init() {
	flag.StringVar(&targetPath, "path", "", "扫描目标路径")
	flag.StringVar(&targetPath, "p", "", "扫描目标路径（简写）")
	flag.BoolVar(&recursive, "recursive", true, "递归扫描")
	flag.BoolVar(&recursive, "r", true, "递归扫描（简写）")

	// 模块开关 - 默认全部 false
	flag.BoolVar(&enableSecretMarker, "secret-marker", false, "启用密级标志检测")
	flag.BoolVar(&enableLayout, "layout", false, "启用公文版式检测")
	flag.BoolVar(&enableHash, "hash", false, "启用文件哈希检测")
	flag.BoolVar(&enableElectronic, "electronic", false, "启用电子密级检测")
	flag.BoolVar(&enableKeywords, "keywords", false, "启用关键词检测")
	flag.BoolVar(&enableAll, "all", false, "启用所有检测模块")
	flag.BoolVar(&disableAll, "none", false, "不自动启用所有模块")

	flag.StringVar(&hashRulesFile, "hash-rules", "", "哈希检测规则文件")
	flag.StringVar(&streamRulesFile, "stream-rules", "", "流式标识规则文件")

	flag.IntVar(&workers, "workers", 0, "并发工作数")
	flag.IntVar(&workers, "w", 0, "并发工作数（简写）")
	flag.IntVar(&timeout, "timeout", 30, "单文件超时（秒）")
	flag.Int64Var(&maxFileSize, "max-size", 100, "最大文件大小（MB）")

	flag.StringVar(&outputFile, "output", "", "输出文件路径")
	flag.StringVar(&outputFile, "o", "", "输出文件路径（简写）")
	flag.StringVar(&outputFormat, "format", "text", "输出格式")
	flag.BoolVar(&verbose, "verbose", false, "详细输出")
	flag.BoolVar(&verbose, "v", false, "详细输出（简写）")
	flag.BoolVar(&quiet, "quiet", false, "静默模式")
	flag.BoolVar(&quiet, "q", false, "静默模式（简写）")
	flag.BoolVar(&showProgress, "progress", true, "显示进度")
	flag.BoolVar(&showConfig, "show-config", false, "显示配置")

	flag.BoolVar(&showHelp, "help", false, "帮助")
	flag.BoolVar(&showHelp, "h", false, "帮助（简写）")
	flag.BoolVar(&showVersion, "version", false, "版本")
}

// ==========================================
// 数据结构
// ==========================================

type ScanResult struct {
	FilePath    string        `json:"file_path"`
	FileName    string        `json:"file_name"`
	FileSize    int64         `json:"file_size"`
	Detected    bool          `json:"detected"`
	SecretLevel int           `json:"secret_level,omitempty"`
	AlertType   int           `json:"alert_type,omitempty"`
	RuleID      int64         `json:"rule_id,omitempty"`
	RuleDesc    string        `json:"rule_desc,omitempty"`
	MatchedText string        `json:"matched_text,omitempty"`
	Duration    time.Duration `json:"duration_ns"`
	Error       string        `json:"error,omitempty"`
}

type ScanSummary struct {
	StartTime     time.Time       `json:"start_time"`
	EndTime       time.Time       `json:"end_time"`
	Duration      time.Duration   `json:"duration_ns"`
	TotalFiles    int64           `json:"total_files"`
	ScannedFiles  int64           `json:"scanned_files"`
	DetectedFiles int64           `json:"detected_files"`
	ErrorFiles    int64           `json:"error_files"`
	TotalSize     int64           `json:"total_size_bytes"`
	ModulesConfig map[string]bool `json:"modules_config"`
	Results       []ScanResult    `json:"results,omitempty"`
}

// ==========================================
// 主函数
// ==========================================

func main() {
	flag.Parse()

	if showHelp {
		printHelp()
		return
	}

	if showVersion {
		fmt.Printf("%s version %s\n", toolName, toolVersion)
		return
	}

	// 处理模块开关（关键修复点）
	resolveModuleFlags()

	if showConfig {
		printCurrentConfig()
		return
	}

	if targetPath == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须指定 -p 参数")
		fmt.Fprintln(os.Stderr, "使用 -h 查看帮助")
		os.Exit(1)
	}

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "错误: 路径不存在: %s\n", targetPath)
		os.Exit(1)
	}

	if !quiet {
		printBanner()
	}

	// 先加载规则（这可能会自动启用模块）
	rulesLoaded := preloadRules()

	// 初始化检测器
	mgr := initDetectorManager()

	// 设置规则到检测器
	applyRules(mgr, rulesLoaded)

	// 收集文件
	files := collectFiles(targetPath)
	if len(files) == 0 {
		fmt.Println("没有找到待扫描的文件")
		return
	}

	if !quiet {
		fmt.Printf("共发现 %d 个文件待扫描\n", len(files))
	}

	// 执行扫描
	summary := runScan(mgr, files)

	// 输出结果
	outputResults(summary)
}

// ==========================================
// 模块开关处理（修复版）
// ==========================================

func resolveModuleFlags() {
	// 检查是否有任何模块被显式启用
	anyModuleExplicit := enableSecretMarker || enableLayout || enableHash || enableElectronic || enableKeywords || enableAll

	// 如果指定了 --all，启用所有模块
	if enableAll {
		enableSecretMarker = true
		enableLayout = true
		enableHash = true
		enableElectronic = true
		enableKeywords = true
		return
	}

	// 如果指定了 --none，只保留显式启用的模块
	if disableAll {
		// 不做任何操作，保持用户显式指定的状态
		return
	}

	// 如果没有任何显式指定，默认启用所有
	if !anyModuleExplicit {
		enableSecretMarker = true
		enableLayout = true
		enableHash = true
		enableElectronic = true
		enableKeywords = true
	}
}

func printCurrentConfig() {
	fmt.Println("当前检测模块配置:")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("  密级标志检测:   %v\n", enableSecretMarker)
	fmt.Printf("  公文版式检测:   %v\n", enableLayout)
	fmt.Printf("  文件哈希检测:   %v\n", enableHash)
	fmt.Printf("  电子密级检测:   %v\n", enableElectronic)
	fmt.Printf("  关键词检测:     %v\n", enableKeywords)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("  哈希规则文件:   %s\n", hashRulesFile)
	fmt.Printf("  流式规则文件:   %s\n", streamRulesFile)
}

// ==========================================
// 规则预加载
// ==========================================

type LoadedRules struct {
	HashRules   []model.HashDetectRule
	StreamRules []model.StreamMarkerDetectRule
}

func preloadRules() *LoadedRules {
	if !quiet {
		fmt.Println("\n[1] 加载检测规则...")
	}

	loaded := &LoadedRules{}

	// 加载哈希规则
	if hashRulesFile != "" {
		rules, err := loadHashRules(hashRulesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ 加载哈希规则失败: %v\n", err)
		} else {
			loaded.HashRules = rules
			if !quiet {
				fmt.Printf("  ✓ 已加载 %d 条哈希规则\n", len(rules))
			}
			// 自动启用哈希检测
			enableHash = true
		}
	}

	// 加载流式标识规则
	if streamRulesFile != "" {
		rules, err := loadStreamMarkerRules(streamRulesFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ 加载流式标识规则失败: %v\n", err)
		} else {
			loaded.StreamRules = rules
			if !quiet {
				fmt.Printf("  ✓ 已加载 %d 条流式标识规则\n", len(rules))
			}
			// 自动启用电子密级检测
			enableElectronic = true
		}
	}

	if len(loaded.HashRules) == 0 && len(loaded.StreamRules) == 0 {
		if !quiet {
			fmt.Println("  ⚠ 未加载任何规则文件")
		}
	}

	return loaded
}

func applyRules(mgr *detector.Manager, rules *LoadedRules) {
	if len(rules.HashRules) > 0 {
		if err := mgr.SetHashRules(rules.HashRules); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ 设置哈希规则失败: %v\n", err)
		}
	}

	if len(rules.StreamRules) > 0 {
		if err := mgr.SetStreamMarkerRules(rules.StreamRules); err != nil {
			fmt.Fprintf(os.Stderr, "  ⚠ 设置流式标识规则失败: %v\n", err)
		}
	}
}

func loadHashRules(path string) ([]model.HashDetectRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config model.HashDetectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		var rules []model.HashDetectRule
		if err2 := json.Unmarshal(data, &rules); err2 != nil {
			return nil, err
		}
		return rules, nil
	}
	return config.Rules, nil
}

func loadStreamMarkerRules(path string) ([]model.StreamMarkerDetectRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config model.StreamMarkerDetectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config.Rules, nil
}

// ==========================================
// 初始化
// ==========================================

func printBanner() {
	fmt.Println(`
+======================================================================+
|                   Detector 模块集成调试工具                           |
|                       detector-debug v1.0.0                          |
+======================================================================+`)
}

func initDetectorManager() *detector.Manager {
	if !quiet {
		fmt.Println("\n[2] 初始化检测器管理器...")
	}

	cfg := detector.GlobalConfig{
		EnableSecretMarker:    enableSecretMarker,
		EnableLayout:          enableLayout,
		EnableHash:            enableHash,
		EnableElectronicLabel: enableElectronic,
		EnableKeywords:        enableKeywords,

		SecretMarkerOCR:         true,
		LayoutThreshold:         0.8,
		StreamMarkerMaxFileSize: maxFileSize * 1024 * 1024,
		HashMaxFileSize:         maxFileSize * 1024 * 1024,

		CurrentCompany:      "调试模式",
		CurrentComputerName: getHostname(),
		CurrentUserName:     getUsername(),
	}

	mgr := detector.NewManager(cfg)
	detector.SetGlobalManager(mgr)

	if !quiet {
		fmt.Println("  ✓ 检测器管理器初始化完成")
		fmt.Println("  ✓ 已启用模块:")
		status := mgr.GetAllSubModuleStatus()
		count := 0
		for name, enabled := range status {
			if enabled {
				fmt.Printf("    - %s\n", name)
				count++
			}
		}
		if count == 0 {
			fmt.Println("    (无)")
		}
	}

	return mgr
}

// ==========================================
// 文件收集
// ==========================================

func collectFiles(path string) []string {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	if !info.IsDir() {
		return []string{path}
	}

	var files []string
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && info.IsDir() && !recursive && p != path {
				return filepath.SkipDir
			}
			return nil
		}

		if maxFileSize > 0 && info.Size() > maxFileSize*1024*1024 {
			if verbose {
				fmt.Printf("  跳过大文件: %s\n", p)
			}
			return nil
		}

		files = append(files, p)
		return nil
	})

	return files
}

// ==========================================
// 扫描执行
// ==========================================

func runScan(mgr *detector.Manager, files []string) *ScanSummary {
	summary := &ScanSummary{
		StartTime:     time.Now(),
		TotalFiles:    int64(len(files)),
		Results:       make([]ScanResult, 0),
		ModulesConfig: mgr.GetAllSubModuleStatus(),
	}

	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers > len(files) {
		numWorkers = len(files)
	}

	if !quiet {
		fmt.Printf("\n[3] 开始扫描 (并发数: %d)\n", numWorkers)
		fmt.Println(strings.Repeat("=", 70))
	}

	taskChan := make(chan string, numWorkers*2)
	resultChan := make(chan ScanResult, numWorkers*2)

	var scanned, detected int64

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range taskChan {
				result := scanFile(mgr, filePath)
				resultChan <- result
			}
		}()
	}

	var resultWg sync.WaitGroup
	var mu sync.Mutex
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		for result := range resultChan {
			atomic.AddInt64(&scanned, 1)
			atomic.AddInt64(&summary.TotalSize, result.FileSize)

			if result.Error != "" {
				atomic.AddInt64(&summary.ErrorFiles, 1)
			} else if result.Detected {
				atomic.AddInt64(&detected, 1)
			}

			mu.Lock()
			summary.Results = append(summary.Results, result)
			mu.Unlock()

			if result.Detected {
				printDetection(result)
			} else if result.Error != "" && verbose {
				fmt.Printf("  [错误] %s: %s\n", result.FileName, result.Error)
			} else if verbose {
				fmt.Printf("  [安全] %s (耗时: %v)\n", result.FileName, result.Duration)
			}

			if showProgress && !quiet {
				cur := atomic.LoadInt64(&scanned)
				if cur%20 == 0 || cur == int64(len(files)) {
					fmt.Printf("\r进度: %d/%d (命中: %d)", cur, len(files), atomic.LoadInt64(&detected))
				}
			}
		}
	}()

	for _, f := range files {
		taskChan <- f
	}
	close(taskChan)

	wg.Wait()
	close(resultChan)
	resultWg.Wait()

	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)
	summary.ScannedFiles = scanned
	summary.DetectedFiles = detected

	if showProgress && !quiet {
		fmt.Println()
	}

	return summary
}

func scanFile(mgr *detector.Manager, filePath string) ScanResult {
	start := time.Now()

	info, err := os.Stat(filePath)
	if err != nil {
		return ScanResult{
			FilePath: filePath,
			FileName: filepath.Base(filePath),
			Error:    err.Error(),
			Duration: time.Since(start),
		}
	}

	result := ScanResult{
		FilePath: filePath,
		FileName: info.Name(),
		FileSize: info.Size(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	detected, alert, _, err := mgr.Detect(ctx, filePath)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Detected = detected
	if detected && alert != nil {
		result.SecretLevel = alert.FileLevel
		result.AlertType = int(alert.AlertType)
		result.RuleID = alert.RuleID
		result.RuleDesc = alert.RuleDesc
		result.MatchedText = alert.HighlightText
	}

	return result
}

// ==========================================
// 输出
// ==========================================

func printDetection(r ScanResult) {
	if quiet {
		fmt.Println(r.FilePath)
		return
	}

	fmt.Printf("\n  [命中] %s\n", r.FilePath)
	fmt.Printf("         类型: %s\n", getAlertTypeStr(r.AlertType))
	fmt.Printf("         密级: %s (Level %d)\n", getSecretLevelStr(r.SecretLevel), r.SecretLevel)
	fmt.Printf("         规则: [%d] %s\n", r.RuleID, r.RuleDesc)
	if r.MatchedText != "" {
		fmt.Printf("         匹配: %s\n", truncate(r.MatchedText, 50))
	}
	fmt.Printf("         大小: %s | 耗时: %v\n", formatSize(r.FileSize), r.Duration)
}

func getSecretLevelStr(level int) string {
	switch level {
	case 4:
		return "绝密"
	case 3:
		return "机密"
	case 2:
		return "秘密"
	case 1:
		return "内部"
	default:
		return "未知"
	}
}

func getAlertTypeStr(t int) string {
	switch t {
	case 1:
		return "电子密级检测"
	case 2:
		return "密级标志检测"
	case 3:
		return "公文版式检测"
	case 4:
		return "关键词检测"
	case 5:
		return "文件哈希检测"
	case 6:
		return "流式标识检测"
	default:
		return fmt.Sprintf("未知(%d)", t)
	}
}

func outputResults(summary *ScanSummary) {
	if !quiet {
		fmt.Println()
		fmt.Println(strings.Repeat("=", 70))
		fmt.Println("[4] 扫描统计报告")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("  扫描耗时:       %v\n", summary.Duration)
		fmt.Printf("  文件总数:       %d\n", summary.TotalFiles)
		fmt.Printf("  已扫描:         %d\n", summary.ScannedFiles)
		fmt.Printf("  检测命中:       %d\n", summary.DetectedFiles)
		fmt.Printf("  错误数:         %d\n", summary.ErrorFiles)
		fmt.Printf("  扫描总大小:     %s\n", formatSize(summary.TotalSize))

		if summary.Duration.Seconds() > 0 {
			speed := float64(summary.TotalSize) / summary.Duration.Seconds() / 1024 / 1024
			fmt.Printf("  扫描速度:       %.2f MB/s\n", speed)
		}

		fmt.Println()
		fmt.Println("  启用的检测模块:")
		for name, enabled := range summary.ModulesConfig {
			status := "禁用"
			if enabled {
				status = "启用"
			}
			fmt.Printf("    - %-25s [%s]\n", name, status)
		}

		fmt.Println()
		fmt.Println(strings.Repeat("=", 70))
	}

	if outputFile != "" {
		saveOutput(summary)
	}
}

func saveOutput(summary *ScanSummary) {
	var data []byte
	var err error

	if outputFormat == "json" {
		data, err = json.MarshalIndent(summary, "", "  ")
	} else {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("扫描报告 - %s\n\n", summary.EndTime.Format("2006-01-02 15:04:05")))
		sb.WriteString(fmt.Sprintf("总文件: %d, 命中: %d\n\n", summary.TotalFiles, summary.DetectedFiles))
		for _, r := range summary.Results {
			if r.Detected {
				sb.WriteString(fmt.Sprintf("[%s] %s\n  规则: %s\n  匹配: %s\n\n",
					getAlertTypeStr(r.AlertType), r.FilePath, r.RuleDesc, r.MatchedText))
			}
		}
		data = []byte(sb.String())
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "生成输出失败: %v\n", err)
		return
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "保存失败: %v\n", err)
	} else if !quiet {
		fmt.Printf("结果已保存: %s\n", outputFile)
	}
}

// ==========================================
// 工具函数
// ==========================================

func getHostname() string {
	h, _ := os.Hostname()
	if h == "" {
		return "unknown"
	}
	return h
}

func getUsername() string {
	u := os.Getenv("USER")
	if u == "" {
		u = os.Getenv("USERNAME")
	}
	if u == "" {
		return "unknown"
	}
	return u
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func printHelp() {
	fmt.Printf(`%s v%s - Detector 模块集成调试工具

用法:
  %s -p <路径> [选项]

扫描目标:
  -p, --path <路径>      扫描目标路径 [必需]
  -r, --recursive        递归扫描 (默认: true)

模块开关:
      --all              启用所有模块
      --none             不自动启用（配合单独指定模块）
      --hash             启用文件哈希检测
      --stream-marker    启用流式标识检测
      --secret-marker    启用密级标志检测
      --layout           启用公文版式检测
      --electronic       启用电子密级检测
      --keywords         启用关键词检测

规则文件:
      --hash-rules       哈希规则文件（自动启用哈希检测）
      --stream-rules     流式标识规则文件（自动启用流式检测）

运行配置:
  -w, --workers          并发数 (默认: CPU核心数)
      --timeout          单文件超时秒数 (默认: 30)
      --max-size         最大文件MB (默认: 100)

输出:
  -o, --output           输出文件
      --format           格式: text, json (默认: text)
  -v, --verbose          详细输出
  -q, --quiet            静默模式
      --show-config      显示配置

示例:
  # 使用所有模块扫描
  %s -p ./test_files -v

  # 只用哈希检测 + 规则文件
  %s -p ./test_files --none --hash-rules rules.json

  # 查看配置
  %s --none --hash --show-config

`, toolName, toolVersion, toolName, toolName, toolName, toolName)
}
