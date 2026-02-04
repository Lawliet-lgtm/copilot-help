// Package main 电子流式标识检测调试工具
// 用于独立测试和调试 electronic_secret 检测模块
package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"linuxFileWatcher/internal/detector/electronic_secret"
	"linuxFileWatcher/internal/model"
)

// ==========================================
// 命令行参数
// ==========================================

var (
	// 扫描目标
	targetPath  string // 扫描目标路径（文件或目录）
	recursive   bool   // 是否递归扫描子目录
	followLinks bool   // 是否跟随符号链接

	// 规则配置
	rulesFile  string // 规则文件路径（JSON格式）
	ruleHex    string // 单条规则（十六进制格式）
	ruleBase64 string // 单条规则（Base64格式）
	ruleID     int64  // 单条规则的ID
	ruleDesc   string // 单条规则的描述

	// 检测配置
	maxFileSize int64 // 最大文件大小（MB）
	timeout     int   // 单文件超时时间（秒）
	scanArchive bool  // 是否扫描压缩包内容
	workers     int   // 并发工作协程数

	// 输出配置
	outputFile   string // 输出文件路径
	outputFormat string // 输出格式：text, json, csv
	verbose      bool   // 详细输出模式
	quiet        bool   // 静默模式（只输出命中结果）
	showProgress bool   // 显示进度

	// 其他
	showHelp    bool // 显示帮助
	showVersion bool // 显示版本
)

const (
	toolName    = "stream-marker-debug"
	toolVersion = "1.0.0"
)

func init() {
	// 扫描目标
	flag.StringVar(&targetPath, "path", "", "扫描目标路径（文件或目录）")
	flag.StringVar(&targetPath, "p", "", "扫描目标路径（简写）")
	flag.BoolVar(&recursive, "recursive", true, "递归扫描子目录")
	flag.BoolVar(&recursive, "r", true, "递归扫描子目录（简写）")
	flag.BoolVar(&followLinks, "follow-links", false, "跟随符号链接")

	// 规则配置
	flag.StringVar(&rulesFile, "rules", "", "规则文件路径（JSON格式）")
	flag.StringVar(&rulesFile, "f", "", "规则文件路径（简写）")
	flag.StringVar(&ruleHex, "hex", "", "单条规则的十六进制内容")
	flag.StringVar(&ruleBase64, "base64", "", "单条规则的Base64内容")
	flag.Int64Var(&ruleID, "rule-id", 1, "单条规则的ID")
	flag.StringVar(&ruleDesc, "rule-desc", "CLI测试规则", "单条规则的描述")

	// 检测配置
	flag.Int64Var(&maxFileSize, "max-size", 500, "最大文件大小（MB）")
	flag.IntVar(&timeout, "timeout", 30, "单文件超时时间（秒）")
	flag.BoolVar(&scanArchive, "scan-archive", true, "扫描压缩包内容")
	flag.IntVar(&workers, "workers", 0, "并发工作协程数（0=CPU核心数）")
	flag.IntVar(&workers, "w", 0, "并发工作协程数（简写）")

	// 输出配置
	flag.StringVar(&outputFile, "output", "", "输出文件路径")
	flag.StringVar(&outputFile, "o", "", "输出文件路径（简写）")
	flag.StringVar(&outputFormat, "format", "text", "输出格式：text, json, csv")
	flag.BoolVar(&verbose, "verbose", false, "详细输出模式")
	flag.BoolVar(&verbose, "v", false, "详细输出模式（简写）")
	flag.BoolVar(&quiet, "quiet", false, "静默模式（只输出命中结果）")
	flag.BoolVar(&quiet, "q", false, "静默模式（简写）")
	flag.BoolVar(&showProgress, "progress", true, "显示进度")

	// 其他
	flag.BoolVar(&showHelp, "help", false, "显示帮助信息")
	flag.BoolVar(&showHelp, "h", false, "显示帮助信息（简写）")
	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
}

// ==========================================
// 数据结构
// ==========================================

// ScanResult 扫描结果
type ScanResult struct {
	FilePath    string        `json:"file_path"`
	FileName    string        `json:"file_name"`
	FileSize    int64         `json:"file_size"`
	Detected    bool          `json:"detected"`
	RuleID      int64         `json:"rule_id,omitempty"`
	RuleDesc    string        `json:"rule_desc,omitempty"`
	MatchedText string        `json:"matched_text,omitempty"`
	Location    string        `json:"location,omitempty"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration_ns"`
}

// ScanSummary 扫描摘要
type ScanSummary struct {
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration_ns"`
	TotalFiles    int64         `json:"total_files"`
	ScannedFiles  int64         `json:"scanned_files"`
	SkippedFiles  int64         `json:"skipped_files"`
	ErrorFiles    int64         `json:"error_files"`
	DetectedFiles int64         `json:"detected_files"`
	TotalSize     int64         `json:"total_size_bytes"`
	RulesCount    int           `json:"rules_count"`
	Results       []ScanResult  `json:"results,omitempty"`
}

// ==========================================
// 主函数
// ==========================================

func main() {
	flag.Parse()

	// 显示帮助
	if showHelp {
		printHelp()
		return
	}

	// 显示版本
	if showVersion {
		fmt.Printf("%s version %s\n", toolName, toolVersion)
		return
	}

	// 验证参数
	if err := validateArgs(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		fmt.Fprintf(os.Stderr, "使用 -h 或 --help 查看帮助信息\n")
		os.Exit(1)
	}

	// 加载规则
	rules, err := loadRules()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载规则失败: %v\n", err)
		os.Exit(1)
	}

	if len(rules) == 0 {
		fmt.Fprintf(os.Stderr, "错误: 没有有效的检测规则\n")
		os.Exit(1)
	}

	if !quiet {
		fmt.Printf("已加载 %d 条规则\n", len(rules))
	}

	// 创建检测器
	detector := createDetector()

	// 设置规则
	if err := detector.SetRules(rules); err != nil {
		fmt.Fprintf(os.Stderr, "设置规则失败: %v\n", err)
		os.Exit(1)
	}

	// 收集文件列表
	files, err := collectFiles(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "收集文件失败: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("没有找到需要扫描的文件")
		return
	}

	if !quiet {
		fmt.Printf("共发现 %d 个���件待扫描\n", len(files))
	}

	// 执行扫描
	summary := runScan(detector, files)

	// 输出结果
	outputResults(summary)

	// 设置退出码
	if summary.DetectedFiles > 0 {
		os.Exit(2) // 检测到敏感文件
	}
	if summary.ErrorFiles > 0 {
		os.Exit(1) // 有错误发生
	}
}

// ==========================================
// 参数验证
// ==========================================

func validateArgs() error {
	if targetPath == "" {
		return fmt.Errorf("必须指定扫描目标路径 (-p 或 --path)")
	}

	// 检查路径是否存在
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("路径不存在: %s", targetPath)
	}

	// 检查规则配置
	if rulesFile == "" && ruleHex == "" && ruleBase64 == "" {
		return fmt.Errorf("必须指定规则：使用 -f/--rules 指定规则文件，或使用 --hex/--base64 指定单条规则")
	}

	// 检查输出格式
	switch outputFormat {
	case "text", "json", "csv":
		// 有效格式
	default:
		return fmt.Errorf("不支持的输出格式: %s（支持: text, json, csv）", outputFormat)
	}

	return nil
}

// ==========================================
// 规则加载
// ==========================================

func loadRules() ([]model.StreamMarkerDetectRule, error) {
	var rules []model.StreamMarkerDetectRule

	// 从文件加载规则
	if rulesFile != "" {
		fileRules, err := loadRulesFromFile(rulesFile)
		if err != nil {
			return nil, fmt.Errorf("从文件加载规则失败: %w", err)
		}
		rules = append(rules, fileRules...)
	}

	// 从命令行参数加载单条规则
	if ruleHex != "" {
		content, err := hex.DecodeString(strings.ReplaceAll(ruleHex, " ", ""))
		if err != nil {
			return nil, fmt.Errorf("解析十六进制规则失败: %w", err)
		}
		rules = append(rules, model.StreamMarkerDetectRule{
			RuleID:      ruleID,
			RuleContent: content,
			RuleDesc:    ruleDesc,
		})
	}

	if ruleBase64 != "" {
		content, err := base64.StdEncoding.DecodeString(ruleBase64)
		if err != nil {
			return nil, fmt.Errorf("解析Base64规则失败: %w", err)
		}
		rules = append(rules, model.StreamMarkerDetectRule{
			RuleID:      ruleID,
			RuleContent: content,
			RuleDesc:    ruleDesc,
		})
	}

	return rules, nil
}

// loadRulesFromFile 从JSON文件加载规则
func loadRulesFromFile(path string) ([]model.StreamMarkerDetectRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 尝试解析为规则配置
	var config model.StreamMarkerDetectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		// 尝试直接解析为规则数组
		var rules []model.StreamMarkerDetectRule
		if err2 := json.Unmarshal(data, &rules); err2 != nil {
			return nil, fmt.Errorf("JSON解析失败: %w", err)
		}
		return rules, nil
	}

	return config.Rules, nil
}

// ==========================================
// 检测器创建
// ==========================================

func createDetector() electronic_secret.DetectorWithRules {
	cfg := electronic_secret.Config{
		Enabled:             true,
		MaxFileSize:         maxFileSize * 1024 * 1024,
		Timeout:             time.Duration(timeout) * time.Second,
		ScanArchiveContent:  scanArchive,
		MaxArchiveEntrySize: 50 * 1024 * 1024,
		MmapThreshold:       10 * 1024 * 1024,
		ChunkSize:           4 * 1024 * 1024,
		Verbose:             verbose,
	}

	return electronic_secret.NewDetector(cfg)
}

// ==========================================
// 文件收集
// ==========================================

func collectFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// 如果是单个文件
	if !info.IsDir() {
		return []string{path}, nil
	}

	// 遍历目录
	var files []string
	var walkFunc fs.WalkDirFunc

	walkFunc = func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "警告: 访问路径失败 %s: %v\n", path, err)
			}
			return nil // 跳过错误，继续遍历
		}

		// 跳过目录
		if d.IsDir() {
			// 如果不递归，跳过子目录
			if !recursive && path != targetPath {
				return fs.SkipDir
			}
			return nil
		}

		// 处理符号链接
		if d.Type()&fs.ModeSymlink != 0 {
			if !followLinks {
				return nil // 跳过符号链接
			}
			// 解析符号链接
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil
			}
			path = realPath
		}

		files = append(files, path)
		return nil
	}

	if err := filepath.WalkDir(path, walkFunc); err != nil {
		return nil, err
	}

	return files, nil
}

// ==========================================
// 扫描执行
// ==========================================

func runScan(detector electronic_secret.DetectorWithRules, files []string) *ScanSummary {
	summary := &ScanSummary{
		StartTime:  time.Now(),
		TotalFiles: int64(len(files)),
		Results:    make([]ScanResult, 0),
	}

	// 确定工作协程数
	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers > len(files) {
		numWorkers = len(files)
	}

	if !quiet {
		fmt.Printf("使用 %d 个工作协程\n", numWorkers)
		fmt.Println("开始扫描...")
		fmt.Println(strings.Repeat("-", 60))
	}

	// 创建任务通道和结果通道
	taskChan := make(chan string, numWorkers*2)
	resultChan := make(chan ScanResult, numWorkers*2)

	// 进度统计
	var scanned int64
	var detected int64

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range taskChan {
				result := scanFile(detector, filePath)
				resultChan <- result
			}
		}()
	}

	// 启动结果收集协程
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		for result := range resultChan {
			// 更新统计
			atomic.AddInt64(&scanned, 1)
			atomic.AddInt64(&summary.TotalSize, result.FileSize)

			if result.Error != "" {
				atomic.AddInt64(&summary.ErrorFiles, 1)
			} else if result.Detected {
				atomic.AddInt64(&detected, 1)
				atomic.AddInt64(&summary.DetectedFiles, 1)
			}

			// 保存结果
			summary.Results = append(summary.Results, result)

			// 输出进度和结果
			if result.Detected {
				printDetection(result)
			} else if result.Error != "" && verbose {
				fmt.Fprintf(os.Stderr, "错误: %s - %s\n", result.FilePath, result.Error)
			}

			// 显示进度
			if showProgress && !quiet {
				current := atomic.LoadInt64(&scanned)
				if current%100 == 0 || current == int64(len(files)) {
					fmt.Printf("\r进度: %d/%d (检测到: %d)", current, len(files), atomic.LoadInt64(&detected))
				}
			}
		}
	}()

	// 分发任务
	for _, file := range files {
		taskChan <- file
	}
	close(taskChan)

	// 等待工作协程完成
	wg.Wait()
	close(resultChan)

	// 等待结果收集完成
	resultWg.Wait()

	// 完成统计
	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)
	summary.ScannedFiles = scanned
	summary.SkippedFiles = summary.TotalFiles - summary.ScannedFiles

	if showProgress && !quiet {
		fmt.Println() // 换行
	}

	return summary
}

// scanFile 扫描单个文件
func scanFile(detector electronic_secret.DetectorWithRules, filePath string) ScanResult {
	startTime := time.Now()

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return ScanResult{
			FilePath: filePath,
			Error:    err.Error(),
			Duration: time.Since(startTime),
		}
	}

	result := ScanResult{
		FilePath: filePath,
		FileName: fileInfo.Name(),
		FileSize: fileInfo.Size(),
		Detected: false,
	}

	// 执行检测
	ctx := context.Background()
	detectResult, err := detector.DetectFile(ctx, filePath)

	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = err.Error()
		return result
	}

	if detectResult != nil && detectResult.IsSecret {
		result.Detected = true
		result.RuleID = detectResult.RuleID
		result.RuleDesc = detectResult.RuleDesc
		result.MatchedText = detectResult.MatchedText
		result.Location = detectResult.ContextText
	}

	return result
}

// ==========================================
// 输出处理
// ==========================================

func printDetection(result ScanResult) {
	if quiet {
		// 静默模式只输出文件路径
		fmt.Println(result.FilePath)
		return
	}

	fmt.Printf("\n[命中] %s\n", result.FilePath)
	fmt.Printf("  规则ID: %d\n", result.RuleID)
	fmt.Printf("  规则描述: %s\n", result.RuleDesc)
	fmt.Printf("  匹配信息: %s\n", result.MatchedText)
	fmt.Printf("  位置: %s\n", result.Location)
	fmt.Printf("  耗时: %v\n", result.Duration)
}

func outputResults(summary *ScanSummary) {
	// 输出摘要
	if !quiet {
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println("扫描完成")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("扫描耗时: %v\n", summary.Duration)
		fmt.Printf("文件总数: %d\n", summary.TotalFiles)
		fmt.Printf("已扫描: %d\n", summary.ScannedFiles)
		fmt.Printf("已跳过: %d\n", summary.SkippedFiles)
		fmt.Printf("错误数: %d\n", summary.ErrorFiles)
		fmt.Printf("检测命中: %d\n", summary.DetectedFiles)
		fmt.Printf("扫描总大小: %s\n", formatSize(summary.TotalSize))
		if summary.Duration.Seconds() > 0 {
			speed := float64(summary.TotalSize) / summary.Duration.Seconds() / 1024 / 1024
			fmt.Printf("扫描速度: %.2f MB/s\n", speed)
		}
	}

	// 输出到文件
	if outputFile != "" {
		if err := writeOutput(summary); err != nil {
			fmt.Fprintf(os.Stderr, "写入输出文件失败: %v\n", err)
		} else if !quiet {
			fmt.Printf("结果已保存到: %s\n", outputFile)
		}
	}
}

func writeOutput(summary *ScanSummary) error {
	var output []byte
	var err error

	switch outputFormat {
	case "json":
		output, err = json.MarshalIndent(summary, "", "  ")
	case "csv":
		output = formatCSV(summary)
	default: // text
		output = formatText(summary)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(outputFile, output, 0644)
}

func formatCSV(summary *ScanSummary) []byte {
	var sb strings.Builder
	sb.WriteString("file_path,file_name,file_size,detected,rule_id,rule_desc,matched_text,location,error,duration_ms\n")

	for _, r := range summary.Results {
		sb.WriteString(fmt.Sprintf("%q,%q,%d,%t,%d,%q,%q,%q,%q,%d\n",
			r.FilePath,
			r.FileName,
			r.FileSize,
			r.Detected,
			r.RuleID,
			r.RuleDesc,
			r.MatchedText,
			r.Location,
			r.Error,
			r.Duration.Milliseconds(),
		))
	}

	return []byte(sb.String())
}

func formatText(summary *ScanSummary) []byte {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("扫描报告\n"))
	sb.WriteString(fmt.Sprintf("生成时间: %s\n", summary.EndTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	sb.WriteString(fmt.Sprintf("扫描统计\n"))
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("扫描耗时: %v\n", summary.Duration))
	sb.WriteString(fmt.Sprintf("文件总数: %d\n", summary.TotalFiles))
	sb.WriteString(fmt.Sprintf("已扫描: %d\n", summary.ScannedFiles))
	sb.WriteString(fmt.Sprintf("检测命中: %d\n", summary.DetectedFiles))
	sb.WriteString(fmt.Sprintf("错误数: %d\n", summary.ErrorFiles))
	sb.WriteString(fmt.Sprintf("扫描总大小: %s\n\n", formatSize(summary.TotalSize)))

	if summary.DetectedFiles > 0 {
		sb.WriteString(fmt.Sprintf("检测命中详情\n"))
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, r := range summary.Results {
			if r.Detected {
				sb.WriteString(fmt.Sprintf("\n文件: %s\n", r.FilePath))
				sb.WriteString(fmt.Sprintf("  大小: %s\n", formatSize(r.FileSize)))
				sb.WriteString(fmt.Sprintf("  规则ID: %d\n", r.RuleID))
				sb.WriteString(fmt.Sprintf("  规则描述: %s\n", r.RuleDesc))
				sb.WriteString(fmt.Sprintf("  匹配信息: %s\n", r.MatchedText))
				sb.WriteString(fmt.Sprintf("  位置: %s\n", r.Location))
			}
		}
	}

	if summary.ErrorFiles > 0 && verbose {
		sb.WriteString(fmt.Sprintf("\n错误详情\n"))
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, r := range summary.Results {
			if r.Error != "" {
				sb.WriteString(fmt.Sprintf("%s: %s\n", r.FilePath, r.Error))
			}
		}
	}

	return []byte(sb.String())
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

// ==========================================
// 帮助信息
// ==========================================

func printHelp() {
	fmt.Printf(`%s v%s - 电子流式标识检测调试工具

用法:
  %s [选项]

扫描目标:
  -p, --path <路径>          扫描目标路径（文件或目录）[必需]
  -r, --recursive            递归扫描子目录 (默认: true)
      --follow-links         跟随符号链接 (默认: false)

规则配置:
  -f, --rules <文件>         规则文件路径（JSON格式）
      --hex <十六进制>        单条规则的十六进制内容
      --base64 <Base64>      单条规则的Base64内容
      --rule-id <ID>         单条规则的ID (默认: 1)
      --rule-desc <描述>     单条规则的描述 (默认: "CLI测试规则")

检测配置:
      --max-size <MB>        最大文件大小，单位MB (默认: 500)
      --timeout <秒>         单文件超时时间 (默认: 30)
      --scan-archive         扫描压缩包内容 (默认: true)
  -w, --workers <数量>       并发工作协程数 (默认: CPU核心数)

输出配置:
  -o, --output <文件>        输出文件路径
      --format <格式>        输出格式: text, json, csv (默认: text)
  -v, --verbose              详细输出模式
  -q, --quiet                静默模式（只输出命中结果）
      --progress             显示进度 (默认: true)

其他:
  -h, --help                 显示帮助信息
      --version              显示版本信息

规则文件格式 (JSON):
  {
    "rules": [
      {
        "rule_id": 1001,
        "rule_content": "Base64编码的256字节内容",
        "rule_desc": "规则描述"
      }
    ]
  }

  或者直接使用规则数组:
  [
    {
      "rule_id": 1001,
      "rule_content": "Base64编码的256字节内容",
      "rule_desc": "规则描述"
    }
  ]

示例:
  # 使用规则文件扫描目录
  %s -p /data/documents -f rules.json

  # 使用十六进制规则扫描单个文件
  %s -p /path/to/file.docx --hex "AABBCCDD..." --rule-id 1001

  # 扫描目录并输出JSON结果
  %s -p /data -f rules.json -o result.json --format json

  # 静默模式，只输出命中的文件路径
  %s -p /data -f rules.json -q

  # 使用4个工作协程扫描
  %s -p /data -f rules.json -w 4

退出码:
  0    正常完成，未检测到敏感文件
  1    发生错误
  2    检测到敏感文件

`, toolName, toolVersion, toolName, toolName, toolName, toolName, toolName, toolName)
}
