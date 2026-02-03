package processor

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"linuxFileWatcher/internal/detector/govcheck/extractor"
)

// ImageProcessor 图片处理器
type ImageProcessor struct {
	base      *BaseProcessor
	config    *ImageProcessorConfig
	ocrEngine OcrEngine
}

// ImageProcessorConfig 图片处理器配置
type ImageProcessorConfig struct {
	MaxFileSize    int64  // 最大文件大小（字节）
	OcrLang        string // OCR 语言
	EnableOcr      bool   // 是否启用 OCR
	NormalizeSpace bool   // 是否规范化空白字符
}

// DefaultImageProcessorConfig 返回默认配置
func DefaultImageProcessorConfig() *ImageProcessorConfig {
	return &ImageProcessorConfig{
		MaxFileSize:    50 * 1024 * 1024, // 50MB
		OcrLang:        "chi_sim+eng",    // 简体中文 + 英文
		EnableOcr:      true,
		NormalizeSpace: true,
	}
}

// NewImageProcessor 创建图片处理器
func NewImageProcessor() *ImageProcessor {
	return NewImageProcessorWithConfig(nil)
}

// NewImageProcessorWithConfig 使用指定配置创建图片处理器
func NewImageProcessorWithConfig(config *ImageProcessorConfig) *ImageProcessor {
	if config == nil {
		config = DefaultImageProcessorConfig()
	}

	base := NewBaseProcessor(
		"ImageProcessor",
		"图片文档处理器 (OCR+图像分析)",
		[]string{"jpg", "jpeg", "png", "gif", "bmp", "tiff", "tif", "webp"},
	)

	processor := &ImageProcessor{
		base:   base,
		config: config,
	}

	if config.EnableOcr {
		processor.ocrEngine = GetOcrManager().GetPrimaryEngine()
	}

	return processor
}

// Name 返回处理器名称
func (p *ImageProcessor) Name() string {
	return p.base.Name()
}

// Description 返回处理器描述
func (p *ImageProcessor) Description() string {
	return p.base.Description()
}

// SupportedTypes 返回支持的文件类型
func (p *ImageProcessor) SupportedTypes() []string {
	return p.base.SupportedTypes()
}

// IsOcrAvailable 检查 OCR 是否可用
func (p *ImageProcessor) IsOcrAvailable() bool {
	return p.ocrEngine != nil && p.ocrEngine.IsAvailable()
}

// Process 处理图片文件
func (p *ImageProcessor) Process(filePath string) (string, error) {
	result, err := p.ProcessWithStyle(filePath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ProcessWithStyle 处理图片文件并返回版式特征
func (p *ImageProcessor) ProcessWithStyle(filePath string) (*ProcessResultWithStyle, error) {
	result := &ProcessResultWithStyle{}

	if err := p.validateFile(filePath); err != nil {
		return nil, err
	}

	// 图像分析
	picInfo, colorAnalysis, err := p.analyzeImage(filePath)
	if err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "分析图片", err)
	}

	// OCR 识别
	if p.config.EnableOcr && p.IsOcrAvailable() {
		text, err := p.performOcr(filePath)
		if err != nil {
			result.Text = ""
		} else {
			result.Text = text
		}
	} else {
		return nil, NewProcessorError(p.Name(), filePath, "OCR识别",
			fmt.Errorf("OCR 不可用，请安装 Tesseract OCR"))
	}

	if p.config.NormalizeSpace && result.Text != "" {
		result.Text = normalizeImageText(result.Text)
	}

	styleFeatures := p.extractStyleFeatures(picInfo, colorAnalysis, result.Text)
	if styleFeatures != nil {
		result.StyleFeatures = styleFeatures
		result.HasStyle = true
	}

	return result, nil
}

// validateFile 验证文件
func (p *ImageProcessor) validateFile(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return NewProcessorError(p.Name(), filePath, "获取文件信息", err)
	}

	if info.Size() == 0 {
		return NewProcessorError(p.Name(), filePath, "检查文件", fmt.Errorf("文件为空"))
	}

	if p.config.MaxFileSize > 0 && info.Size() > p.config.MaxFileSize {
		return NewProcessorError(p.Name(), filePath, "检查文件大小",
			fmt.Errorf("文件过大: %d 字节 (限制: %d 字节)", info.Size(), p.config.MaxFileSize))
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	supported := false
	for _, t := range p.SupportedTypes() {
		if t == ext {
			supported = true
			break
		}
	}
	if !supported {
		return NewProcessorError(p.Name(), filePath, "检查文件类型",
			fmt.Errorf("不支持的图片格式: %s", ext))
	}

	return nil
}

// PictureFileInfo 图片文件信息
type PictureFileInfo struct {
	PixelWidth  int
	PixelHeight int
	Format      string
	WidthMM     float64
	HeightMM    float64
	IsA4        bool
	IsPortrait  bool
}

// ColorAnalysis 颜色分析结果
type ColorAnalysis struct {
	HasRedInTop      bool
	HasRedInBottom   bool
	RedPixelRatio    float64
	TopRedRatio      float64
	BottomRedRatio   float64
	HasPotentialSeal bool
	SealRegionDesc   string
	HasRedLine       bool
}

// analyzeImage 分析图片
func (p *ImageProcessor) analyzeImage(filePath string) (*PictureFileInfo, *ColorAnalysis, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	info := &PictureFileInfo{
		PixelWidth:  width,
		PixelHeight: height,
		Format:      format,
		IsPortrait:  height > width,
	}

	// 使用 150 DPI 作为默认（扫描件常用）
	dpi := 150.0
	info.WidthMM = float64(width) / dpi * 25.4
	info.HeightMM = float64(height) / dpi * 25.4

	// 智能 A4 检测
	info.IsA4 = p.checkA4Size(width, height)

	colorAnalysis := p.analyzeColors(img)

	return info, colorAnalysis, nil
}

// checkA4Size 检查是否为 A4 尺寸
func (p *ImageProcessor) checkA4Size(width, height int) bool {
	// 方法1: 尝试多个常见 DPI 值
	dpiValues := []float64{72, 96, 150, 200, 300}

	for _, dpi := range dpiValues {
		widthMM := float64(width) / dpi * 25.4
		heightMM := float64(height) / dpi * 25.4

		// 纵向 A4（210 x 297 mm，允许 ±5% 误差）
		if widthMM >= 199.5 && widthMM <= 220.5 && heightMM >= 282 && heightMM <= 312 {
			return true
		}

		// 横向 A4
		if heightMM >= 199.5 && heightMM <= 220.5 && widthMM >= 282 && widthMM <= 312 {
			return true
		}
	}

	// 方法2: 检查宽高比是否接近 A4（1:1.414）
	var ratio float64
	if height > width {
		ratio = float64(height) / float64(width)
	} else {
		ratio = float64(width) / float64(height)
	}

	// A4 比例约 1.414，允许 ±8% 误差
	if ratio >= 1.30 && ratio <= 1.53 {
		return true
	}

	return false
}

// analyzeColors 分析图片颜色
func (p *ImageProcessor) analyzeColors(img image.Image) *ColorAnalysis {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	analysis := &ColorAnalysis{}

	step := 2
	if width > 2000 || height > 2000 {
		step = 4
	}

	totalPixels := 0
	redPixels := 0

	topHeight := height / 5
	bottomStart := height * 3 / 4  // 底部 25%
	rightStart := width * 2 / 3    // 右侧 33%

	topRedPixels := 0
	topTotalPixels := 0
	bottomRedPixels := 0
	bottomTotalPixels := 0
	bottomRightRedPixels := 0
	bottomRightTotalPixels := 0

	redLineRows := make(map[int]int)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		rowRedCount := 0

		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			totalPixels++
			c := img.At(x, y)

			relY := y - bounds.Min.Y
			relX := x - bounds.Min.X

			isRed := isRedPixel(c)

			if isRed {
				redPixels++
				rowRedCount++

				if relY < topHeight {
					topRedPixels++
				}

				if relY >= bottomStart {
					bottomRedPixels++
					if relX >= rightStart {
						bottomRightRedPixels++
					}
				}
			}

			if relY < topHeight {
				topTotalPixels++
			}
			if relY >= bottomStart {
				bottomTotalPixels++
				if relX >= rightStart {
					bottomRightTotalPixels++
				}
			}
		}

		redLineRows[y] = rowRedCount
	}

	if totalPixels > 0 {
		analysis.RedPixelRatio = float64(redPixels) / float64(totalPixels)
	}
	if topTotalPixels > 0 {
		analysis.TopRedRatio = float64(topRedPixels) / float64(topTotalPixels)
	}
	if bottomTotalPixels > 0 {
		analysis.BottomRedRatio = float64(bottomRedPixels) / float64(bottomTotalPixels)
	}

	analysis.HasRedInTop = analysis.TopRedRatio > 0.01
	analysis.HasRedInBottom = analysis.BottomRedRatio > 0.003

	// 检测红色横线
	samplesPerRow := width / step
	for _, count := range redLineRows {
		if float64(count)/float64(samplesPerRow) > 0.3 {
			analysis.HasRedLine = true
			break
		}
	}

	// 检测印章（右下角区域）
	if bottomRightTotalPixels > 0 {
		bottomRightRatio := float64(bottomRightRedPixels) / float64(bottomRightTotalPixels)
		if bottomRightRatio > 0.008 {
			analysis.HasPotentialSeal = true
			analysis.SealRegionDesc = fmt.Sprintf("右下角检测到红色区域 (占比: %.2f%%)", bottomRightRatio*100)
		}
	}

	if !analysis.HasPotentialSeal && analysis.BottomRedRatio > 0.005 {
		analysis.HasPotentialSeal = true
		analysis.SealRegionDesc = "底部检测到红色聚集区域（可能是印章）"
	}

	return analysis
}

// isRedPixel 判断像素是否为红色
func isRedPixel(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	r8 := r >> 8
	g8 := g >> 8
	b8 := b >> 8

	if r8 > 150 && r8 > g8+50 && r8 > b8+50 && g8 < 150 && b8 < 150 {
		return true
	}

	if r8 > 120 && r8 > g8*2 && r8 > b8*2 && g8 < 100 && b8 < 100 {
		return true
	}

	return false
}

// performOcr 执行 OCR 识别
func (p *ImageProcessor) performOcr(filePath string) (string, error) {
	if p.ocrEngine == nil {
		return "", fmt.Errorf("OCR 引擎未初始化")
	}

	return p.ocrEngine.RecognizeWithLang(filePath, p.config.OcrLang)
}

// extractStyleFeatures 提取版式特征
func (p *ImageProcessor) extractStyleFeatures(picInfo *PictureFileInfo, colorAnalysis *ColorAnalysis, text string) *extractor.StyleFeatures {
	sf := &extractor.StyleFeatures{
		StyleReasons: make([]string, 0),
	}

	if picInfo != nil {
		sf.PageWidth = picInfo.WidthMM
		sf.PageHeight = picInfo.HeightMM

		if picInfo.IsA4 {
			sf.IsA4Paper = true
			sf.StyleReasons = append(sf.StyleReasons, "图片尺寸/比例符合A4规格")
		}

		sf.StyleReasons = append(sf.StyleReasons,
			fmt.Sprintf("图片尺寸: %dx%d 像素", picInfo.PixelWidth, picInfo.PixelHeight))
	}

	if colorAnalysis != nil {
		if colorAnalysis.HasRedInTop {
			sf.HasRedText = true
			sf.HasRedHeader = true
			sf.StyleReasons = append(sf.StyleReasons,
				fmt.Sprintf("顶部检测到红色内容 (占比: %.2f%%)", colorAnalysis.TopRedRatio*100))
		}

		if colorAnalysis.HasRedLine {
			sf.StyleReasons = append(sf.StyleReasons, "检测到红色横线（红头分隔线）")
			if !sf.HasRedHeader {
				sf.HasRedHeader = true
			}
		}

		if colorAnalysis.HasPotentialSeal {
			sf.HasSealImage = true
			sf.SealImageHint = colorAnalysis.SealRegionDesc
			sf.StyleReasons = append(sf.StyleReasons, colorAnalysis.SealRegionDesc)
		}

		if colorAnalysis.RedPixelRatio > 0.005 {
			sf.HasRedText = true
			sf.StyleReasons = append(sf.StyleReasons,
				fmt.Sprintf("图片整体红色像素占比: %.2f%%", colorAnalysis.RedPixelRatio*100))
		}
	}

	if text != "" {
		if !sf.HasRedHeader && containsRedHeaderKeywords(text) {
			sf.StyleReasons = append(sf.StyleReasons, "文本包含公文红头特征词")
		}

		if containsSealKeywords(text) {
			if !sf.HasSealImage {
				sf.HasSealImage = true
				sf.SealImageHint = "文本包含印章相关关键词"
			}
			sf.StyleReasons = append(sf.StyleReasons, "文本包含印章相关词汇")
		}
	}

	sf.StyleReasons = append(sf.StyleReasons, "通过OCR提取文本内容")

	p.calculateStyleScore(sf)

	return sf
}

// calculateStyleScore 计算版式得分
func (p *ImageProcessor) calculateStyleScore(sf *extractor.StyleFeatures) {
	score := 0.0

	// A4尺寸 (+0.12)
	if sf.IsA4Paper {
		score += 0.12
	}

	// 红头检测 (+0.20)
	if sf.HasRedHeader {
		score += 0.20
	} else if sf.HasRedText {
		score += 0.10
	}

	// 印章检测 (+0.18)
	if sf.HasSealImage {
		score += 0.18
	}

	// 图片基础分 (+0.05)
	score += 0.05

	// 红头+印章组合加分 (+0.10)
	if sf.HasRedHeader && sf.HasSealImage {
		score += 0.10
	}

	// 红头+A4组合加分 (+0.05)
	if sf.HasRedHeader && sf.IsA4Paper {
		score += 0.05
	}

	// 三要素齐全额外加分 (+0.05)
	if sf.HasRedHeader && sf.HasSealImage && sf.IsA4Paper {
		score += 0.05
	}

	sf.StyleScore = score
	sf.IsOfficialStyle = score >= 0.35 || (sf.HasRedHeader && sf.HasSealImage)
}

// containsRedHeaderKeywords 检查是否包含红头关键词
func containsRedHeaderKeywords(text string) bool {
	keywords := []string{
		"文件", "通知", "通报", "决定", "命令", "公告",
		"意见", "报告", "请示", "批复", "函",
		"发〔", "发[", "发（", "字〔", "字[",
	}

	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}

	return false
}

// containsSealKeywords 检查是否包含印章关键词
func containsSealKeywords(text string) bool {
	keywords := []string{
		"印章", "盖章", "公章", "专用章", "合同章",
		"人民政府", "委员会", "办公室", "局", "厅",
		"有限公司", "股份公司",
	}

	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}

	return false
}

// normalizeImageText 规范化 OCR 识别的文本
func normalizeImageText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 移除 CJK 字符之间的空格（OCR 常见问题）
		line = removeCjkSpaces(line)

		line = removeOcrNoise(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	text = strings.Join(cleanedLines, "\n")

	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(text)
}

// removeCjkSpaces 移除 CJK 字符之间的空格
func removeCjkSpaces(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}

	var result strings.Builder

	for i := 0; i < len(runes); i++ {
		current := runes[i]

		if current == ' ' {
			var prev, next rune
			if i > 0 {
				prev = runes[i-1]
			}
			if i+1 < len(runes) {
				next = runes[i+1]
			}

			// CJK 字符之间的空格
			if isCjk(prev) && isCjk(next) {
				continue
			}

			// CJK 和标点之间的空格
			if isCjk(prev) && isCjkPunctuation(next) {
				continue
			}
			if isCjkPunctuation(prev) && isCjk(next) {
				continue
			}

			// CJK 和数字之间的空格
			if isCjk(prev) && isDigit(next) {
				continue
			}
			if isDigit(prev) && isCjk(next) {
				continue
			}
		}

		result.WriteRune(current)
	}

	return result.String()
}

// isCjk 检查是否是 CJK 字符
func isCjk(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x20000 && r <= 0x2A6DF)
}

// isCjkPunctuation 检查是否是 CJK 标点
// isCjkPunctuation 检查是否是 CJK 标点
func isCjkPunctuation(r rune) bool {
	punctuations := `，。、；：？！""''（）【】《》〔〕·—`
	return strings.ContainsRune(punctuations, r)
}

// isDigit 检查是否是数字
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// removeOcrNoise 移除 OCR 识别噪点
func removeOcrNoise(line string) string {
	if len(line) <= 2 {
		allNoise := true
		for _, r := range line {
			if isCjk(r) {
				allNoise = false
				break
			}
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
				allNoise = false
				break
			}
			if r >= '0' && r <= '9' {
				allNoise = false
				break
			}
		}
		if allNoise {
			return ""
		}
	}
	return line
}

// GetOcrStatus 获取 OCR 状态
func (p *ImageProcessor) GetOcrStatus() string {
	if !p.config.EnableOcr {
		return "OCR 已禁用"
	}

	if p.ocrEngine == nil {
		return "OCR 引擎未初始化"
	}

	if !p.ocrEngine.IsAvailable() {
		return "OCR 不可用 - 请安装 Tesseract OCR"
	}

	return fmt.Sprintf("OCR 可用 - %s v%s",
		p.ocrEngine.GetName(),
		p.ocrEngine.GetVersion())
}