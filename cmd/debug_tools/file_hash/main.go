// Package main 文件哈希检测调试工具
// 用于独立测试和调试 file_hash 检测模块
package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"linuxFileWatcher/internal/detector/file_hash"
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
	rulesFile string // 规则文件路径（JSON格式）
	hashValue string // 单条规则的哈希值
	hashType  int    // 哈希类型：0=MD5, 1=SM3
	ruleID    int64  // 单条规则的ID
	ruleDesc  string // 单条规则的描述

	// 检测配置
	maxFileSize int64 // 最大文件大小（MB）
	workers     int   // 并发工作协程数

	// 输出配置
	outputFile   string // 输出文件路径
	outputFormat string // 输出格式：text, json, csv
	verbose      bool   // 详细输出模式
	quiet        bool   // 静默模式（只输出命中结果）
	showProgress bool   // 显示进度
	showHash     bool   // 显示所有文件的哈希值（用于生成规则）

	// 其他
	showHelp    bool // 显示帮助
	showVersion bool // 显示版本
)

const (
	toolName    = "file-hash-debug"
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
	flag.StringVar(&hashValue, "hash", "", "单条规则的哈希值（MD5或SM3）")
	flag.IntVar(&hashType, "type", 0, "哈希类型：0=MD5, 1=SM3")
	flag.Int64Var(&ruleID, "rule-id", 1, "单条规则的ID")
	flag.StringVar(&ruleDesc, "rule-desc", "CLI测试规则", "单条规则的描述")

	// 检测配置
	flag.Int64Var(&maxFileSize, "max-size", 100, "最大文件大小（MB）")
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
	flag.BoolVar(&showHash, "show-hash", false, "显示所有文件的哈希值（用于生成规则）")

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
	MD5Hash     string        `json:"md5_hash,omitempty"`
	SM3Hash     string        `json:"sm3_hash,omitempty"`
	Detected    bool          `json:"detected"`
	RuleID      int64         `json:"rule_id,omitempty"`
	RuleDesc    string        `json:"rule_desc,omitempty"`
	MatchedHash string        `json:"matched_hash,omitempty"`
	HashType    string        `json:"hash_type,omitempty"`
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

	// 如果是显示哈希模式
	if showHash {
		runShowHashMode()
		return
	}

	// 加载规则
	rules, err := loadRules()
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载规则失败: %v\n", err)
		os.Exit(1)
	}

	if len(rules) == 0 {
		fmt.Fprintf(os.Stderr, "错误: 没有有效的检测规则\n")
		fmt.Fprintf(os.Stderr, "提示: 使用 --show-hash 可以查看文件的哈希值，用于生成规则\n")
		os.Exit(1)
	}

	if !quiet {
		fmt.Printf("已加载 %d 条规则\n", len(rules))
		for _, r := range rules {
			hashTypeStr := "MD5"
			if r.RuleType == 1 {
				hashTypeStr = "SM3"
			}
			fmt.Printf("  - [%s] %s: %s\n", hashTypeStr, r.RuleDesc, truncateHash(r.RuleContent))
		}
		fmt.Println()
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
		fmt.Printf("共发现 %d 个文件待扫描\n", len(files))
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

	// 如果不是显示哈希模式，检查规则配置
	if !showHash {
		if rulesFile == "" && hashValue == "" {
			return fmt.Errorf("必须指定规则：使用 -f/--rules 指定规则文件，或使用 --hash 指定单条哈希值\n提示: 使用 --show-hash 可以查看文件的哈希值")
		}
	}

	// 检查哈希类型
	if hashType != 0 && hashType != 1 {
		return fmt.Errorf("不支持的哈希类型: %d（支持: 0=MD5, 1=SM3）", hashType)
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

func loadRules() ([]model.HashDetectRule, error) {
	var rules []model.HashDetectRule

	// 从文件加载规则
	if rulesFile != "" {
		fileRules, err := loadRulesFromFile(rulesFile)
		if err != nil {
			return nil, fmt.Errorf("从文件加载规则失败: %w", err)
		}
		rules = append(rules, fileRules...)
	}

	// 从命令行参数加载单条规则
	if hashValue != "" {
		rules = append(rules, model.HashDetectRule{
			RuleID:      ruleID,
			RuleType:    hashType,
			RuleContent: strings.ToLower(hashValue),
			RuleDesc:    ruleDesc,
		})
	}

	return rules, nil
}

// loadRulesFromFile 从JSON文件加载规则
func loadRulesFromFile(path string) ([]model.HashDetectRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 尝试解析为规则配置
	var config model.HashDetectConfig
	if err := json.Unmarshal(data, &config); err != nil {
		// 尝试直接解析为规则数组
		var rules []model.HashDetectRule
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

func createDetector() file_hash.HashDetectorWithRules {
	cfg := file_hash.Config{
		MaxFileSize: maxFileSize * 1024 * 1024,
		EnableMD5:   true,
		EnableSM3:   true,
	}

	return file_hash.NewHashDetector(cfg)
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
			return nil
		}

		// 跳过目录
		if d.IsDir() {
			if !recursive && path != targetPath {
				return fs.SkipDir
			}
			return nil
		}

		// 处理符号链接
		if d.Type()&fs.ModeSymlink != 0 {
			if !followLinks {
				return nil
			}
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
// 显示哈希模式
// ==========================================

func runShowHashMode() {
	files, err := collectFiles(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "收集文件失败: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("没有找到文件")
		return
	}

	fmt.Printf("计算 %d 个文件的哈希值...\n", len(files))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-32s  %-10s  %s\n", "MD5", "大小", "文件路径")
	fmt.Println(strings.Repeat("-", 80))

	var rules []model.HashDetectRule
	ruleIDCounter := int64(1001)

	for _, filePath := range files {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "警告: 无法获取文件信息 %s: %v\n", filePath, err)
			}
			continue
		}

		// 跳过过大的文件
		if maxFileSize > 0 && fileInfo.Size() > maxFileSize*1024*1024 {
			if verbose {
				fmt.Fprintf(os.Stderr, "跳过: 文件过大 %s (%s)\n", filePath, formatSize(fileInfo.Size()))
			}
			continue
		}

		// 计算MD5
		hash, err := computeFileMD5(filePath)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "警告: 计算哈希失败 %s: %v\n", filePath, err)
			}
			continue
		}

		fmt.Printf("%-32s  %-10s  %s\n", hash, formatSize(fileInfo.Size()), filePath)

		// 收集规则
		rules = append(rules, model.HashDetectRule{
			RuleID:      ruleIDCounter,
			RuleType:    0, // MD5
			RuleContent: hash,
			RuleDesc:    filepath.Base(filePath),
		})
		ruleIDCounter++
	}

	fmt.Println(strings.Repeat("-", 80))

	// 如果指定了输出文件，生成规则文件
	if outputFile != "" {
		config := model.HashDetectConfig{Rules: rules}
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成规则文件失败: %v\n", err)
			return
		}
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "写入规则文件失败: %v\n", err)
			return
		}
		fmt.Printf("\n规则文件已保存到: %s\n", outputFile)
		fmt.Printf("共生成 %d 条规则\n", len(rules))
	}
}

// ==========================================
// 扫描执行
// ==========================================

func runScan(detector file_hash.HashDetectorWithRules, files []string) *ScanSummary {
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
			atomic.AddInt64(&scanned, 1)
			atomic.AddInt64(&summary.TotalSize, result.FileSize)

			if result.Error != "" {
				atomic.AddInt64(&summary.ErrorFiles, 1)
			} else if result.Detected {
				atomic.AddInt64(&detected, 1)
				atomic.AddInt64(&summary.DetectedFiles, 1)
			}

			summary.Results = append(summary.Results, result)

			if result.Detected {
				printDetection(result)
			} else if result.Error != "" && verbose {
				fmt.Fprintf(os.Stderr, "错误: %s - %s\n", result.FilePath, result.Error)
			}

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

	wg.Wait()
	close(resultChan)
	resultWg.Wait()

	summary.EndTime = time.Now()
	summary.Duration = summary.EndTime.Sub(summary.StartTime)
	summary.ScannedFiles = scanned
	summary.SkippedFiles = summary.TotalFiles - summary.ScannedFiles

	if showProgress && !quiet {
		fmt.Println()
	}

	return summary
}

// scanFile 扫描单个文件
func scanFile(detector file_hash.HashDetectorWithRules, filePath string) ScanResult {
	startTime := time.Now()

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

	// 计算哈希值（用于显示）
	if verbose || showHash {
		if hash, err := computeFileMD5(filePath); err == nil {
			result.MD5Hash = hash
		}
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
		result.MatchedHash = detectResult.MatchedText
		// 解析哈希类型
		if strings.Contains(detectResult.ContextText, "MD5") {
			result.HashType = "MD5"
		} else if strings.Contains(detectResult.ContextText, "SM3") {
			result.HashType = "SM3"
		}
	}

	return result
}

// ==========================================
// 输出处理
// ==========================================

func printDetection(result ScanResult) {
	if quiet {
		fmt.Println(result.FilePath)
		return
	}

	fmt.Printf("\n[命中] %s\n", result.FilePath)
	fmt.Printf("  规则ID: %d\n", result.RuleID)
	fmt.Printf("  规则描述: %s\n", result.RuleDesc)
	fmt.Printf("  匹配哈希: %s\n", result.MatchedHash)
	if result.HashType != "" {
		fmt.Printf("  哈希类型: %s\n", result.HashType)
	}
	fmt.Printf("  文件大小: %s\n", formatSize(result.FileSize))
	fmt.Printf("  耗时: %v\n", result.Duration)
}

func outputResults(summary *ScanSummary) {
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
	default:
		output = formatText(summary)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(outputFile, output, 0644)
}

func formatCSV(summary *ScanSummary) []byte {
	var sb strings.Builder
	sb.WriteString("file_path,file_name,file_size,md5_hash,detected,rule_id,rule_desc,hash_type,error,duration_ms\n")

	for _, r := range summary.Results {
		sb.WriteString(fmt.Sprintf("%q,%q,%d,%q,%t,%d,%q,%q,%q,%d\n",
			r.FilePath,
			r.FileName,
			r.FileSize,
			r.MD5Hash,
			r.Detected,
			r.RuleID,
			r.RuleDesc,
			r.HashType,
			r.Error,
			r.Duration.Milliseconds(),
		))
	}

	return []byte(sb.String())
}

func formatText(summary *ScanSummary) []byte {
	var sb strings.Builder

	sb.WriteString("扫描报告 - 文件哈希检测\n")
	sb.WriteString(fmt.Sprintf("生成时间: %s\n", summary.EndTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(strings.Repeat("=", 60) + "\n\n")

	sb.WriteString("扫描统计\n")
	sb.WriteString(strings.Repeat("-", 40) + "\n")
	sb.WriteString(fmt.Sprintf("扫描耗时: %v\n", summary.Duration))
	sb.WriteString(fmt.Sprintf("文件总数: %d\n", summary.TotalFiles))
	sb.WriteString(fmt.Sprintf("已扫描: %d\n", summary.ScannedFiles))
	sb.WriteString(fmt.Sprintf("检测命中: %d\n", summary.DetectedFiles))
	sb.WriteString(fmt.Sprintf("错误数: %d\n", summary.ErrorFiles))
	sb.WriteString(fmt.Sprintf("扫描总大小: %s\n\n", formatSize(summary.TotalSize)))

	if summary.DetectedFiles > 0 {
		sb.WriteString("检测命中详情\n")
		sb.WriteString(strings.Repeat("-", 40) + "\n")
		for _, r := range summary.Results {
			if r.Detected {
				sb.WriteString(fmt.Sprintf("\n文件: %s\n", r.FilePath))
				sb.WriteString(fmt.Sprintf("  大小: %s\n", formatSize(r.FileSize)))
				sb.WriteString(fmt.Sprintf("  规则ID: %d\n", r.RuleID))
				sb.WriteString(fmt.Sprintf("  规则描述: %s\n", r.RuleDesc))
				sb.WriteString(fmt.Sprintf("  匹配哈希: %s\n", r.MatchedHash))
				sb.WriteString(fmt.Sprintf("  哈希类型: %s\n", r.HashType))
			}
		}
	}

	return []byte(sb.String())
}

// ==========================================
// 工具函数
// ==========================================

func computeFileMD5(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
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

func truncateHash(hash string) string {
	if len(hash) > 16 {
		return hash[:16] + "..."
	}
	return hash
}

// ==========================================
// 帮助信息
// ==========================================

func printHelp() {
	fmt.Printf(`%s v%s - 文件哈希检测调试工具

用法:
  %s [选项]

扫描目标:
  -p, --path <路径>          扫描目标路径（文件或目录）[必需]
  -r, --recursive            递归扫描子目录 (默认: true)
      --follow-links         跟随符号链接 (默认: false)

规则配置:
  -f, --rules <文件>         规则文件路径（JSON格式）
      --hash <哈希值>         单条规则的哈希值（MD5或SM3）
      --type <类型>          哈希类型: 0=MD5, 1=SM3 (默认: 0)
      --rule-id <ID>         单条规则的ID (默认: 1)
      --rule-desc <描述>     单条规则的描述 (默认: "CLI测试规则")

检测配置:
      --max-size <MB>        最大文件大小，单位MB (默认: 100)
  -w, --workers <数量>       并发工作协程数 (默认: CPU核心数)

输出配置:
  -o, --output <文件>        输出文件路径
      --format <格式>        输出格式: text, json, csv (默认: text)
  -v, --verbose              详细输出模式
  -q, --quiet                静默模式（只输出命中结果）
      --progress             显示进度 (默认: true)
      --show-hash            显示所有文件的哈希值（用于生成规则）

其他:
  -h, --help                 显示帮助信息
      --version              显示版本信息

规则文件格式 (JSON):
  {
    "rules": [
      {
        "rule_id": 1001,
        "rule_type": 0,
        "rule_content": "d41d8cd98f00b204e9800998ecf8427e",
        "rule_desc": "敏感文件A"
      }
    ]
  }

  rule_type: 0=MD5, 1=SM3

示例:
  # 查看目录中所有文件的哈希值
  %s -p /data/documents --show-hash

  # 查看哈希并生成规则文件
  %s -p /data/sensitive --show-hash -o rules.json

  # 使用规则文件扫描目录
  %s -p /data/documents -f rules.json

  # 使用单条MD5规则扫描
  %s -p /path/to/file.doc --hash "d41d8cd98f00b204e9800998ecf8427e"

  # 使用单条SM3规则扫描
  %s -p /data --hash "e3b0c44298fc1c14..." --type 1

  # 扫描并输出JSON结果
  %s -p /data -f rules.json -o result.json --format json

退出码:
  0    正常完成，未检测到敏感文件
  1    发生错误
  2    检测到敏感文件

`, toolName, toolVersion, toolName, toolName, toolName, toolName, toolName, toolName, toolName)
}
