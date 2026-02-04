package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	// 引入公共模型
	"linuxFileWatcher/internal/model"
	// 引入我们封装好的密级标志检测子模块
	"linuxFileWatcher/internal/detector/secret_level"
)

var (
	targetDir string
	workers   int
	verbose   bool
	enableOCR bool
)

func main() {
	// 1. 参数解析 (复刻之前的 CLI 体验)
	flag.StringVar(&targetDir, "d", ".", "要扫描的目录路径")
	flag.IntVar(&workers, "w", runtime.NumCPU(), "并发工作线程数")
	flag.BoolVar(&verbose, "v", false, "显示详细调试日志")
	flag.BoolVar(&enableOCR, "ocr", true, "开启 OCR 检测 (默认开启)")
	flag.Parse()

	// 2. 初始化配置
	if verbose {
		absPath, _ := os.Getwd()
		fmt.Printf("[DEBUG] 启动独立调试模式\n")
		fmt.Printf("[DEBUG] 当前工作目录: %s\n", absPath)
		fmt.Printf("[DEBUG] 目标目录: %s\n", targetDir)
		fmt.Printf("[DEBUG] OCR 状态: %v\n", enableOCR)
		if os.Getenv("TESSDATA_PREFIX") == "" && enableOCR {
			fmt.Println("[WARN] 未设置 TESSDATA_PREFIX，OCR 可能会失败")
		}
	}

	// 校验目录
	stat, err := os.Stat(targetDir)
	if err != nil {
		fmt.Printf("Fatal: 无法访问目标目录: %v\n", err)
		os.Exit(1)
	}
	if !stat.IsDir() {
		fmt.Printf("Fatal: 目标路径不是一个目录: %s\n", targetDir)
		os.Exit(1)
	}

	// 3. 初始化核心检测服务 (使用最新的 internal 模块)
	cfg := secret_level.Config{
		EnableOCR:      enableOCR,
		OCRMaxFileSize: 20 * 1024 * 1024, // 20MB
	}
	detector := secret_level.NewDetector(cfg)

	// 4. 实现并发调度 (因为 service 层现在只管单文件，所以这里要手写调度)
	fileChan := make(chan string, 100)
	var wg sync.WaitGroup
	startTime := time.Now()
	countFound := 0
	var mu sync.Mutex // 保护 countFound 和输出

	fmt.Printf("[INFO] 开始扫描... 并发数: %d\n", workers)
	fmt.Println("------------------------------------------------")

	// 启动 Worker 池
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for path := range fileChan {
				processFile(detector, path, verbose, &mu, &countFound)
			}
		}(i)
	}

	// 遍历目录并分发任务
	go func() {
		err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				if verbose {
					fmt.Printf("[DEBUG] 访问错误: %s\n", err)
				}
				return nil
			}
			if info.IsDir() {
				// 跳过隐藏目录
				if strings.HasPrefix(info.Name(), ".") && len(info.Name()) > 1 {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(info.Name(), ".") {
				return nil
			}

			// 发送任务
			fileChan <- path
			return nil
		})
		if err != nil {
			fmt.Printf("遍历目录出错: %v\n", err)
		}
		close(fileChan) // 遍历完关闭通道
	}()

	// 等待所有任务完成
	wg.Wait()

	fmt.Println("------------------------------------------------")
	fmt.Printf("扫描结束。耗时: %v\n", time.Since(startTime))
	fmt.Printf("发现涉密文件数: %d\n", countFound)
}

// processFile 单个文件处理逻辑
func processFile(det secret_level.Detector, path string, verbose bool, mu *sync.Mutex, count *int) {
	// 创建带超时的 Context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 调用核心库
	res, err := det.DetectFile(ctx, path)

	if err != nil {
		if verbose && !strings.Contains(err.Error(), "deadline exceeded") {
			mu.Lock()
			fmt.Printf("[DEBUG] 检测出错 [%s]: %v\n", path, err)
			mu.Unlock()
		}
		return
	}

	if res != nil && res.IsSecret {
		mu.Lock()
		*count++
		
		// 将枚举转换为字符串以便打印
		levelStr := "未知"
		switch res.SecretLevel {
		case model.LevelTopSecret:
			levelStr = "绝密"
		case model.LevelSecret:
			levelStr = "机密"
		case model.LevelConfidential:
			levelStr = "秘密"
		}

		fmt.Printf("[发现涉密] [%s] %s (匹配: %s)\n", levelStr, path, res.MatchedText)
		mu.Unlock()
	} else if verbose {
		// fmt.Printf("[DEBUG] 安全: %s\n", path)
	}
}