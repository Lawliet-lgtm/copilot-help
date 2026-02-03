// //go:build ignore

// package main

// import (
// 	"fmt"
// 	"os"

// 	"linuxFileWatcher/internal/detector/govcheck/processor"
// )

// func main() {
// 	if len(os.Args) < 2 {
// 		fmt.Println("用法: go run test/pdf_debug.go <pdf文件路径>")
// 		os.Exit(1)
// 	}

// 	filePath := os.Args[1]

// 	fmt.Printf("正在分析PDF文件: %s\n", filePath)
// 	fmt.Println("========================================")

// 	// 创建PDF处理器
// 	pdfProcessor := processor.NewPdfProcessor()

// 	// 处理文件
// 	result, err := pdfProcessor.ProcessWithStyle(filePath)
// 	if err != nil {
// 		fmt.Printf("处理失败: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// 输出提取的文本
// 	fmt.Println("\n【提取的文本内容】")
// 	fmt.Println("----------------------------------------")
// 	text := result.Text
// 	if len(text) > 2000 {
// 		fmt.Println(text[:2000])
// 		fmt.Println("\n... (文本过长，仅显示前2000字符)")
// 	} else if len(text) == 0 {
// 		fmt.Println("(未提取到任何文本)")
// 	} else {
// 		fmt.Println(text)
// 	}
// 	fmt.Println("----------------------------------------")
// 	fmt.Printf("文本长度: %d 字符\n", len(text))

// 	// 输出版式特征
// 	if result.HasStyle && result.StyleFeatures != nil {
// 		sf := result.StyleFeatures
// 		fmt.Println("\n【版式特征】")
// 		fmt.Printf("  红色文本: %v\n", sf.HasRedText)
// 		fmt.Printf("  红头: %v\n", sf.HasRedHeader)
// 		fmt.Printf("  A4纸张: %v\n", sf.IsA4Paper)
// 		fmt.Printf("  页面尺寸: %.1f x %.1f mm\n", sf.PageWidth, sf.PageHeight)
// 		fmt.Printf("  公文字体: %v (%s)\n", sf.HasOfficialFonts, sf.MainFontName)
// 		fmt.Printf("  印章图片: %v\n", sf.HasSealImage)
// 		fmt.Printf("  版式得分: %.2f\n", sf.StyleScore)
// 	}
// }

// //go:build ignore

// package main

// import (
// 	"fmt"
// 	"linuxFileWatcher/internal/detector/govcheck/rules"
// )

// func main() {
// 	// 测试发文字号匹配
// 	fmt.Println("=== 发文字号测试 ===")
// 	testDocNumbers := []string{
// 		"Xxxx发〔20xx〕xxx号",      // 你的测试数据（用x替代）
// 		"国办发〔2024〕1号",          // 真实格式
// 		"京政发〔2023〕15号",         // 真实格式
// 		"〔2024〕123号",            // 简单格式
// 	}

// 	for _, text := range testDocNumbers {
// 		result := rules.ExtractDocNumber(text)
// 		matched := "✗ 未匹配"
// 		if result != "" {
// 			matched = "✓ " + result
// 		}
// 		fmt.Printf("  文本: %-30s 结果: %s\n", text, matched)
// 	}

// 	// 测试日期匹配
// 	fmt.Println("\n=== 成文日期测试 ===")
// 	testDates := []string{
// 		"20xx年xx月xx日",    // 你的测试数据（用x替代）
// 		"2024年1月15日",     // 真实格式
// 		"2023年12月31日",    // 真实格式
// 	}

// 	for _, text := range testDates {
// 		result := rules.ExtractIssueDate(text)
// 		matched := "✗ 未匹配"
// 		if result != "" {
// 			matched = "✓ " + result
// 		}
// 		fmt.Printf("  文本: %-25s 结果: %s\n", text, matched)
// 	}

// 	// 测试标题匹配
// 	fmt.Println("\n=== 公文标题测试 ===")
// 	testTitles := []string{
// 		"关于xxxxxxxx的通知",
// 		"关于印发实施方案的通知",
// 		"关于做好安全生产工作的意见",
// 	}

// 	for _, text := range testTitles {
// 		result := rules.ExtractTitle(text)
// 		matched := "✗ 未匹配"
// 		if result != "" {
// 			matched = "✓ " + result
// 		}
// 		fmt.Printf("  文本: %-35s 结果: %s\n", text, matched)
// 	}
// }


// //go:build ignore


// package main

// import (
// 	"archive/zip"
// 	"fmt"
// 	"io"
// 	"os"
// 	"strings"
// )

// func main() {
// 	if len(os.Args) < 2 {
// 		fmt.Println("用法: go run test/ofd_debug.go <ofd文件路径>")
// 		os.Exit(1)
// 	}

// 	filePath := os.Args[1]

// 	fmt.Printf("正在分析OFD文件: %s\n", filePath)
// 	fmt.Println("========================================")

// 	zipReader, err := zip.OpenReader(filePath)
// 	if err != nil {
// 		fmt.Printf("无法打开ZIP: %v\n", err)
// 		return
// 	}
// 	defer zipReader.Close()

// 	// 1. 列出所有文件
// 	fmt.Println("\n【文件列表】")
// 	fmt.Println("----------------------------------------")
// 	for _, file := range zipReader.File {
// 		fmt.Printf("  %s (%d bytes)\n", file.Name, file.UncompressedSize64)
// 	}

// 	// 2. 读取并显示每个XML文件的内容
// 	fmt.Println("\n【XML文件内容】")
// 	fmt.Println("========================================")

// 	for _, file := range zipReader.File {
// 		if strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
// 			content, err := readZipFile(file)
// 			if err != nil {
// 				fmt.Printf("\n>>> %s: 读取失败 - %v\n", file.Name, err)
// 				continue
// 			}

// 			fmt.Printf("\n>>> %s (%d bytes)\n", file.Name, len(content))
// 			fmt.Println("----------------------------------------")

// 			// 显示内容（限制长度）
// 			contentStr := string(content)
// 			if len(contentStr) > 3000 {
// 				fmt.Println(contentStr[:3000])
// 				fmt.Println("\n... (内容过长，已截断)")
// 			} else {
// 				fmt.Println(contentStr)
// 			}
// 		}
// 	}
// }

// func readZipFile(file *zip.File) ([]byte, error) {
// 	rc, err := file.Open()
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rc.Close()
// 	return io.ReadAll(rc)
// }

//go:build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"linuxFileWatcher/internal/detector/govcheck/processor"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run test/image_debug.go <图片路径>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	fmt.Printf("正在分析图片: %s\n", filePath)
	fmt.Println("========================================")

	// 1. 图像颜色分析
	fmt.Println("\n【图像颜色分析】")
	fmt.Println("----------------------------------------")
	analyzeImageColors(filePath)

	// 2. OCR 状态和识别
	fmt.Println("\n【OCR状态】")
	fmt.Println("----------------------------------------")
	ocrManager := processor.GetOcrManager()
	if !ocrManager.IsAvailable() {
		fmt.Println("OCR 不可用！")
	} else {
		engine := ocrManager.GetPrimaryEngine()
		fmt.Printf("引擎: %s v%s\n", engine.GetName(), engine.GetVersion())

		// 检查语言
		if tesseract, ok := engine.(*processor.TesseractOcr); ok {
			langs := tesseract.GetLanguages()
			fmt.Printf("可用语言: %v\n", langs)

			hasChinese := false
			for _, l := range langs {
				if l == "chi_sim" || l == "chi_tra" {
					hasChinese = true
					break
				}
			}
			if !hasChinese {
				fmt.Println("\n⚠️ 警告: 未安装中文语言包！")
				fmt.Println("请运行以下命令安装:")
				fmt.Println(`  Invoke-WebRequest -Uri "https://github.com/tesseract-ocr/tessdata/raw/main/chi_sim.traineddata" -OutFile "C:\Program Files\Tesseract-OCR\tessdata\chi_sim.traineddata"`)
			}
		}

		fmt.Println("\n【OCR识别结果】")
		fmt.Println("----------------------------------------")
		text, err := ocrManager.Recognize(filePath)
		if err != nil {
			fmt.Printf("OCR识别失败: %v\n", err)
		} else {
			if len(text) > 1000 {
				fmt.Println(text[:1000])
				fmt.Println("... (截断)")
			} else {
				fmt.Println(text)
			}
			fmt.Println("----------------------------------------")
			fmt.Printf("识别文本长度: %d 字符\n", len(text))
		}
	}

	// 3. 使用完整处理器
	fmt.Println("\n【使用ImageProcessor完整处理】")
	fmt.Println("----------------------------------------")
	imgProcessor := processor.NewImageProcessor()
	result, err := imgProcessor.ProcessWithStyle(filePath)
	if err != nil {
		fmt.Printf("处理失败: %v\n", err)
		return
	}

	fmt.Printf("文本长度: %d 字符\n", len(result.Text))

	if result.HasStyle && result.StyleFeatures != nil {
		sf := result.StyleFeatures
		fmt.Println("\n【版式特征检测结果】")
		fmt.Printf("  红色文本: %v\n", sf.HasRedText)
		fmt.Printf("  红头标志: %v\n", sf.HasRedHeader)
		fmt.Printf("  印章图片: %v (%s)\n", sf.HasSealImage, sf.SealImageHint)
		fmt.Printf("  A4纸张: %v\n", sf.IsA4Paper)
		fmt.Printf("  尺寸: %.1f x %.1f mm\n", sf.PageWidth, sf.PageHeight)
		fmt.Printf("  版式得分: %.2f\n", sf.StyleScore)
		fmt.Printf("  公文版式: %v\n", sf.IsOfficialStyle)

		if len(sf.StyleReasons) > 0 {
			fmt.Println("\n  检测详情:")
			for _, reason := range sf.StyleReasons {
				fmt.Printf("    • %s\n", reason)
			}
		}
	}
}

// analyzeImageColors 分析图片颜色
func analyzeImageColors(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("无法打开图片: %v\n", err)
		return
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		fmt.Printf("无法解码图片: %v\n", err)
		return
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	fmt.Printf("图片格式: %s\n", format)
	fmt.Printf("图片尺寸: %d x %d 像素\n", width, height)

	// 采样分析
	step := 2
	if width > 2000 || height > 2000 {
		step = 4
	}

	totalPixels := 0
	redPixels := 0

	topHeight := height / 5
	bottomStart := height * 4 / 5

	topRedPixels := 0
	topTotalPixels := 0
	bottomRedPixels := 0
	bottomTotalPixels := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			totalPixels++
			c := img.At(x, y)

			if isRedColor(c) {
				redPixels++

				relY := y - bounds.Min.Y
				if relY < topHeight {
					topRedPixels++
				} else if relY >= bottomStart {
					bottomRedPixels++
				}
			}

			relY := y - bounds.Min.Y
			if relY < topHeight {
				topTotalPixels++
			} else if relY >= bottomStart {
				bottomTotalPixels++
			}
		}
	}

	fmt.Printf("\n颜色统计:\n")
	fmt.Printf("  采样像素总数: %d\n", totalPixels)
	fmt.Printf("  红色像素数: %d (%.2f%%)\n", redPixels, float64(redPixels)/float64(totalPixels)*100)

	if topTotalPixels > 0 {
		topRatio := float64(topRedPixels) / float64(topTotalPixels) * 100
		fmt.Printf("  顶部区域红色: %d / %d (%.2f%%)\n", topRedPixels, topTotalPixels, topRatio)
		if topRatio > 1 {
			fmt.Printf("  ✓ 顶部检测到红色（可能是红头）\n")
		}
	}

	if bottomTotalPixels > 0 {
		bottomRatio := float64(bottomRedPixels) / float64(bottomTotalPixels) * 100
		fmt.Printf("  底部区域红色: %d / %d (%.2f%%)\n", bottomRedPixels, bottomTotalPixels, bottomRatio)
		if bottomRatio > 0.5 {
			fmt.Printf("  ✓ 底部检测到红色（可能是印章）\n")
		}
	}
}

// isRedColor 判断是否为红色
func isRedColor(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	r8 := r >> 8
	g8 := g >> 8
	b8 := b >> 8

	// 红色判断
	if r8 > 150 && r8 > g8+50 && r8 > b8+50 && g8 < 150 && b8 < 150 {
		return true
	}

	if r8 > 120 && r8 > g8*2 && r8 > b8*2 && g8 < 100 && b8 < 100 {
		return true
	}

	return false
}