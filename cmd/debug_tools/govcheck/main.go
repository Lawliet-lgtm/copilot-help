package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"linuxFileWatcher/internal/detector/govcheck"
	"linuxFileWatcher/internal/detector/govcheck/detector"
	"linuxFileWatcher/internal/detector/govcheck/processor"
)

// 版本信息
const (
	ToolName    = "govcheck-debug"
	ToolVersion = "1.0.0"
)

// 命令行参数
type CliConfig struct {
	FilePath    string
	DirPath     string
	OutputJSON  bool
	Verbose     bool
	Threshold   float64
	Workers     int
	ShowVersion bool
	ShowHelp    bool
	ShowStatus  bool
	DisableOCR  bool
	Timeout     int

	// 调试选项
	UseSubDetector bool // 使用 SubDetector 接口（模拟上游调用）
}

func main() {
	cliConfig := parseArgs()

	if cliConfig.ShowVersion {
		printVersion()
		os.Exit(0)
	}

	if cliConfig.ShowHelp {
		printHelp()
		os.Exit(0)
	}

	if cliConfig.ShowStatus {
		printStatus()
		os.Exit(0)
	}

	// 验证参数
	if cliConfig.FilePath == "" && cliConfig.DirPath == "" {
		fmt.Fprintln(os.Stderr, "错误: 请指定待检测的文件 (-file) 或目录 (-dir)")
		fmt.Fprintln(os.Stderr, "使用 -help 查看帮助信息")
		os.Exit(1)
	}

	// 运行检测
	if err := run(cliConfig); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs() *CliConfig {
	cfg := &CliConfig{}

	flag.StringVar(&cfg.FilePath, "file", "", "待检测的单个文件路径")
	flag.StringVar(&cfg.FilePath, "f", "", "待检测的单个文件路径 (简写)")

	flag.StringVar(&cfg.DirPath, "dir", "", "待检测的目录路径")
	flag.StringVar(&cfg.DirPath, "d", "", "待检测的目录路径 (简写)")

	flag.BoolVar(&cfg.OutputJSON, "json", false, "以 JSON 格式输出结果")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "显示详细检测信息")
	flag.BoolVar(&cfg.Verbose, "v", false, "显示详细检测信息 (简写)")

	flag.Float64Var(&cfg.Threshold, "threshold", 0.6, "公文判定阈值 (0-1)")
	flag.Float64Var(&cfg.Threshold, "t", 0.6, "公文判定阈值 (简写)")

	flag.IntVar(&cfg.Workers, "workers", 4, "并行处理的协程数")
	flag.IntVar(&cfg.Workers, "w", 4, "并行处理的协程数 (简写)")

	flag.IntVar(&cfg.Timeout, "timeout", 30, "单文件处理超时（秒）")

	flag.BoolVar(&cfg.ShowVersion, "version", false, "显示版本信息")
	flag.BoolVar(&cfg.ShowHelp, "help", false, "显示帮助信息")
	flag.BoolVar(&cfg.ShowHelp, "h", false, "显示帮助信息 (简写)")
	flag.BoolVar(&cfg.ShowStatus, "status", false, "显示系统状态")
	flag.BoolVar(&cfg.DisableOCR, "no-ocr", false, "禁用 OCR 功能")

	flag.BoolVar(&cfg.UseSubDetector, "sub", false, "使用 SubDetector 接口（模拟上游调用）")

	flag.Parse()

	// 支持位置参数
	if cfg.FilePath == "" && cfg.DirPath == "" && flag.NArg() > 0 {
		cfg.FilePath = flag.Arg(0)
	}

	return cfg
}

func run(cfg *CliConfig) error {
	startTime := time.Now()

	if cfg.Verbose {
		printHeader()
	}

	// 收集文件
	files, err := collectFiles(cfg)
	if err != nil {
		return fmt.Errorf("收集文件失败: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("没有找到待检测的文件")
		return nil
	}

	if cfg.Verbose {
		fmt.Printf("找到 %d 个待检测文件\n\n", len(files))
	}

	// 根据模式选择检测方式
	if cfg.UseSubDetector {
		return runWithSubDetector(cfg, files, startTime)
	}
	return runWithInternalDetector(cfg, files, startTime)
}

// runWithInternalDetector 使用内部检测器（原 CLI 模式）
func runWithInternalDetector(cfg *CliConfig, files []string, startTime time.Time) error {
	// 创建检测器
	detConfig := detector.DefaultConfig()
	detConfig.Threshold = cfg.Threshold
	detConfig.Verbose = cfg.Verbose

	det := detector.New(detConfig)

	// 注册处理器
	registerProcessors(det, cfg)

	if cfg.Verbose {
		fmt.Printf("已注册处理器，支持格式: %v\n", det.SupportedTypes())
		printOCRStatus(cfg)
	}

	// 执行检测
	var results []*detector.DetectionResult

	if len(files) == 1 {
		result := det.Detect(files[0])
		results = []*detector.DetectionResult{result}
	} else {
		results = detectBatchParallel(det, files, cfg.Workers)
	}

	totalTime := time.Since(startTime)

	// 输出结果
	if cfg.OutputJSON {
		return outputJSON(results, totalTime)
	}
	return outputText(results, totalTime, cfg.Verbose)
}

// runWithSubDetector 使用 SubDetector 接口（模拟上游调用）
func runWithSubDetector(cfg *CliConfig, files []string, startTime time.Time) error {
	// 创建 govcheck.Detector（实现 SubDetector 接口）
	govCfg := govcheck.Config{
		Threshold:   cfg.Threshold,
		Timeout:     cfg.Timeout,
		MaxFileSize: 100 * 1024 * 1024,
		EnableOCR:   !cfg.DisableOCR,
		OCRLanguage: "chi_sim+eng",
		Verbose:     cfg.Verbose,
	}

	det := govcheck.NewDetector(govCfg)

	if cfg.Verbose {
		fmt.Println("使用 SubDetector 接口模式（模拟上游调用）")
		fmt.Println()
	}

	// 统计
	var totalCount, hitCount, errorCount int32
	var mu sync.Mutex
	type SubResult struct {
		FilePath string
		IsHit    bool
		RuleDesc string
		Error    string
	}
	var subResults []SubResult

	// 并行处理
	ctx := context.Background()
	sem := make(chan struct{}, cfg.Workers)
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			atomic.AddInt32(&totalCount, 1)

			result, err := det.DetectFile(ctx, filePath)

			sr := SubResult{FilePath: filePath}
			if err != nil {
				sr.Error = err.Error()
				atomic.AddInt32(&errorCount, 1)
			} else if result != nil && result.IsSecret {
				sr.IsHit = true
				sr.RuleDesc = result.RuleDesc
				atomic.AddInt32(&hitCount, 1)
			}

			mu.Lock()
			subResults = append(subResults, sr)
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	// 输出结果
	if cfg.OutputJSON {
		output := map[string]interface{}{
			"mode":        "sub_detector",
			"results":     subResults,
			"total":       totalCount,
			"hit":         hitCount,
			"error":       errorCount,
			"total_time":  totalTime.String(),
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// 文本输出
	fmt.Println("========================================")
	fmt.Println(" SubDetector 模式检测结果")
	fmt.Println("========================================")
	fmt.Println()

	for _, sr := range subResults {
		fileName := filepath.Base(sr.FilePath)
		if sr.Error != "" {
			fmt.Printf("✗ %s - 错误: %s\n", fileName, sr.Error)
		} else if sr.IsHit {
			fmt.Printf("✓ %s - 命中: %s\n", fileName, sr.RuleDesc)
		} else {
			fmt.Printf("○ %s - 未命中\n", fileName)
		}
	}

	fmt.Println()
	fmt.Println("----------------------------------------")
	fmt.Printf("总计: %d 个文件\n", totalCount)
	fmt.Printf("命中: %d 个\n", hitCount)
	fmt.Printf("未命中: %d 个\n", totalCount-hitCount-errorCount)
	fmt.Printf("错误: %d 个\n", errorCount)
	fmt.Printf("耗时: %v\n", totalTime)

	return nil
}

// registerProcessors 注册所有处理器
func registerProcessors(det *detector.Detector, cfg *CliConfig) {
	det.RegisterProcessor(processor.NewTextProcessor())
	det.RegisterProcessor(processor.NewDocxProcessor())
	det.RegisterProcessor(processor.NewDocProcessor())
	det.RegisterProcessor(processor.NewWpsProcessor())
	det.RegisterProcessor(processor.NewPdfProcessor())
	det.RegisterProcessor(processor.NewOfdProcessor())

	if !cfg.DisableOCR {
		imgConfig := processor.DefaultImageProcessorConfig()
		imgProcessor := processor.NewImageProcessorWithConfig(imgConfig)
		if imgProcessor.IsOcrAvailable() {
			det.RegisterProcessor(imgProcessor)
		}
	}
}

// detectBatchParallel 并行批量检测
func detectBatchParallel(det *detector.Detector, files []string, workers int) []*detector.DetectionResult {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	results := make([]*detector.DetectionResult, len(files))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i, file := range files {
		wg.Add(1)
		go func(idx int, filePath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = det.Detect(filePath)
		}(i, file)
	}

	wg.Wait()
	return results
}

// collectFiles 收集待检测文件
func collectFiles(cfg *CliConfig) ([]string, error) {
	var files []string

	if cfg.FilePath != "" {
		absPath, err := filepath.Abs(cfg.FilePath)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("文件不存在: %s", absPath)
		}
		files = append(files, absPath)
	} else if cfg.DirPath != "" {
		err := filepath.Walk(cfg.DirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// 跳过隐藏目录
				if len(info.Name()) > 0 && info.Name()[0] == '.' {
					return filepath.SkipDir
				}
				return nil
			}
			// 跳过隐藏文件
			if len(info.Name()) > 0 && info.Name()[0] == '.' {
				return nil
			}
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			files = append(files, absPath)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// outputJSON JSON 格式输出
func outputJSON(results []*detector.DetectionResult, totalTime time.Duration) error {
	output := struct {
		Results   []*detector.DetectionResult `json:"results"`
		Summary   map[string]interface{}      `json:"summary"`
	}{
		Results: results,
		Summary: buildSummary(results, totalTime),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// outputText 文本格式输出
func outputText(results []*detector.DetectionResult, totalTime time.Duration, verbose bool) error {
	for i, result := range results {
		if i > 0 {
			fmt.Println("----------------------------------------")
		}
		if verbose {
			fmt.Print(result.VerboseSummary())
		} else {
			fmt.Print(result.Summary())
		}
	}

	if len(results) > 1 {
		fmt.Println()
		fmt.Println("========================================")
		fmt.Println(" 批量检测汇总")
		fmt.Println("========================================")
		summary := buildSummary(results, totalTime)
		fmt.Printf("总计: %d 个文件\n", summary["total"])
		fmt.Printf("公文: %d 个\n", summary["official"])
		fmt.Printf("非公文: %d 个\n", summary["non_official"])
		fmt.Printf("失败: %d 个\n", summary["failed"])
		fmt.Printf("总耗时: %v\n", totalTime)
	}

	return nil
}

// buildSummary 构建汇总信息
func buildSummary(results []*detector.DetectionResult, totalTime time.Duration) map[string]interface{} {
	var total, official, nonOfficial, failed int
	for _, r := range results {
		total++
		if !r.Success {
			failed++
		} else if r.IsOfficialDoc {
			official++
		} else {
			nonOfficial++
		}
	}
	return map[string]interface{}{
		"total":        total,
		"official":     official,
		"non_official": nonOfficial,
		"failed":       failed,
		"total_time":   totalTime.String(),
	}
}

// printHeader 打印头部信息
func printHeader() {
	fmt.Println("========================================")
	fmt.Printf(" %s v%s\n", ToolName, ToolVersion)
	fmt.Println(" 公文版式检测调试工具")
	fmt.Println(" 标准: GB/T 9704-2012")
	fmt.Println("========================================")
	fmt.Println()
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("%s version %s\n", ToolName, ToolVersion)
}

// printStatus 打印系统状态
func printStatus() {
	fmt.Println("========================================")
	fmt.Printf(" %s v%s\n", ToolName, ToolVersion)
	fmt.Println(" 系统状态")
	fmt.Println("========================================")
	fmt.Println()

	// OCR 状态
	fmt.Println("[OCR 引擎]")
	ocrManager := processor.GetOcrManager()
	if ocrManager.IsAvailable() {
		engine := ocrManager.GetPrimaryEngine()
		fmt.Printf("  状态: 可用\n")
		fmt.Printf("  引擎: %s\n", engine.GetName())
		fmt.Printf("  版本: %s\n", engine.GetVersion())
	} else {
		fmt.Printf("  状态: 不可用\n")
		fmt.Printf("  提示: 请安装 Tesseract OCR\n")
	}
	fmt.Println()

	// DOC 处理器状态
	fmt.Println("[DOC 处理器]")
	docInfo := processor.GetDocExtractorInfo()
	fmt.Printf("  antiword:    %s\n", formatAvailable(docInfo["antiword"]))
	fmt.Printf("  LibreOffice: %s\n", formatAvailable(docInfo["libreoffice"]))
	fmt.Printf("  基础提取:    可用 (备选)\n")
	fmt.Println()

	// 支持的格式
	fmt.Println("[支持的文件格式]")
	fmt.Println("  文本类: txt, text, html, htm, xml, rtf, mht, mhtml, eml")
	fmt.Println("  文档类: doc, docx, docm, dotx, dotm, wps, wpt")
	fmt.Println("  PDF类:  pdf")
	fmt.Println("  OFD类:  ofd")
	if ocrManager.IsAvailable() {
		fmt.Println("  图片类: jpg, jpeg, png, gif, bmp, tiff, tif, webp")
	} else {
		fmt.Println("  图片类: (OCR 不可用)")
	}
	fmt.Println()
}

// printOCRStatus 打印 OCR 状态
func printOCRStatus(cfg *CliConfig) {
	if cfg.DisableOCR {
		fmt.Println("OCR 状态: 已禁用")
	} else {
		ocrManager := processor.GetOcrManager()
		if ocrManager.IsAvailable() {
			engine := ocrManager.GetPrimaryEngine()
			fmt.Printf("OCR 状态: 可用 - %s %s\n", engine.GetName(), engine.GetVersion())
		} else {
			fmt.Println("OCR 状态: 不可用")
		}
	}
	fmt.Println()
}

// printHelp 打印帮助信息
func printHelp() {
	help := `%s - 公文版式检测调试工具

版本: %s
标准: GB/T 9704-2012

用法:
  %s [选项] [文件路径]
  %s -file <文件路径>
  %s -dir <目录路径>

选项:
  -file, -f <路径>      指定待检测的单个文件
  -dir, -d <路径>       指定待检测的目录
  -threshold, -t <值>   公文判定阈值 (0-1)，默认 0.6
  -workers, -w <数量>   并行处理协程数，默认 4
  -timeout <秒>         单文件处理超时，默认 30
  -json                 JSON 格式输出
  -verbose, -v          详细输出模式
  -no-ocr               禁用 OCR 功能
  -sub                  使用 SubDetector 接口（模拟上游调用）
  -status               显示系统状态
  -version              显示版本信息
  -help, -h             显示帮助信息

调试模式:
  默认模式:   直接调用内部 detector.Detector，输出详细检测结果
  -sub 模式:  调用 govcheck.Detector（SubDetector 接口），模拟上游 Manager 调用

示例:
  # 检测单个文件（详细模式）
  %s -file document.pdf -v

  # 检测目录
  %s -dir ./documents/

  # 使用 SubDetector 接口模式
  %s -file document.pdf -sub

  # 调整阈值，JSON 输出
  %s -file doc.docx -threshold 0.5 -json

  # 查看系统状态
  %s -status
`
	fmt.Printf(help, ToolName, ToolVersion, ToolName, ToolName, ToolName,
		ToolName, ToolName, ToolName, ToolName, ToolName)
}

func formatAvailable(available bool) string {
	if available {
		return "可用"
	}
	return "不可用"
}