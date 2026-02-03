package processor

import (
	"archive/zip"
	"fmt"
	"os"
	"regexp"
	"strings"

	"linuxFileWatcher/internal/detector/govcheck/extractor"
)

// OfdProcessor OFD文档处理器
type OfdProcessor struct {
	base   *BaseProcessor
	config *OfdProcessorConfig
}

// OfdProcessorConfig OFD处理器配置
type OfdProcessorConfig struct {
	MaxFileSize    int64 // 最大文件大小（字节）
	ExtractStyle   bool  // 是否提取版式特征
	NormalizeSpace bool  // 是否规范化空白字符
}

// DefaultOfdProcessorConfig 返回默认配置
func DefaultOfdProcessorConfig() *OfdProcessorConfig {
	return &OfdProcessorConfig{
		MaxFileSize:    100 * 1024 * 1024, // 100MB
		ExtractStyle:   true,
		NormalizeSpace: true,
	}
}

// NewOfdProcessor 创建OFD处理器
func NewOfdProcessor() *OfdProcessor {
	return NewOfdProcessorWithConfig(nil)
}

// NewOfdProcessorWithConfig 使用指定配置创建OFD处理器
func NewOfdProcessorWithConfig(config *OfdProcessorConfig) *OfdProcessor {
	if config == nil {
		config = DefaultOfdProcessorConfig()
	}

	base := NewBaseProcessor(
		"OfdProcessor",
		"OFD版式文档处理器",
		[]string{"ofd"},
	)

	return &OfdProcessor{
		base:   base,
		config: config,
	}
}

// Name 返回处理器名称
func (p *OfdProcessor) Name() string {
	return p.base.Name()
}

// Description 返回处理器描述
func (p *OfdProcessor) Description() string {
	return p.base.Description()
}

// SupportedTypes 返回支持的文件类型
func (p *OfdProcessor) SupportedTypes() []string {
	return p.base.SupportedTypes()
}

// Process 处理OFD文件（实现 Processor 接口）
func (p *OfdProcessor) Process(filePath string) (string, error) {
	result, err := p.ProcessWithStyle(filePath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ProcessWithStyle 处理OFD文件并返回版式特征（实现 StyleProcessor 接口）
func (p *OfdProcessor) ProcessWithStyle(filePath string) (*ProcessResultWithStyle, error) {
	result := &ProcessResultWithStyle{}

	// 检查文件
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "获取文件信息", err)
	}

	if info.Size() == 0 {
		return nil, NewProcessorError(p.Name(), filePath, "检查文件", fmt.Errorf("文件为空"))
	}

	if p.config.MaxFileSize > 0 && info.Size() > p.config.MaxFileSize {
		return nil, NewProcessorError(p.Name(), filePath, "检查文件大小",
			fmt.Errorf("文件过大: %d 字节 (限制: %d 字节)", info.Size(), p.config.MaxFileSize))
	}

	// 打开ZIP文件
	zipReader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "打开OFD文件", err)
	}
	defer zipReader.Close()

	// 验证是否为OFD文件
	if !p.isValidOfd(&zipReader.Reader) {
		return nil, NewProcessorError(p.Name(), filePath, "验证文件格式",
			fmt.Errorf("不是有效的OFD文件"))
	}

	// 创建OFD解析器
	parser := NewOfdParser(&zipReader.Reader)
	if err := parser.Parse(); err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "解析OFD文件", err)
	}

	// 提取文本
	text := parser.GetAllText()

	// 规范化文本
	if p.config.NormalizeSpace {
		text = normalizeOfdText(text)
	}

	result.Text = text

	// 提取版式特征
	if p.config.ExtractStyle {
		styleFeatures := p.extractStyleFeatures(parser)
		if styleFeatures != nil {
			result.StyleFeatures = styleFeatures
			result.HasStyle = true
		}
	}

	return result, nil
}

// isValidOfd 验证是否为有效的OFD文件
func (p *OfdProcessor) isValidOfd(zipReader *zip.Reader) bool {
	for _, file := range zipReader.File {
		if file.Name == "OFD.xml" || strings.EqualFold(file.Name, "ofd.xml") {
			return true
		}
	}
	return false
}

// extractStyleFeatures 提取OFD版式特征
func (p *OfdProcessor) extractStyleFeatures(parser *OfdParser) *extractor.StyleFeatures {
	sf := &extractor.StyleFeatures{
		StyleReasons: make([]string, 0),
	}

	// 1. 检测页面设置
	p.detectPageSettings(parser, sf)

	// 2. 检测颜色
	p.detectColors(parser, sf)

	// 3. 检测字体
	p.detectFonts(parser, sf)

	// 4. 检测签章
	p.detectSignatures(parser, sf)

	// 5. 计算版式得分
	p.calculateStyleScore(sf)

	return sf
}

// detectPageSettings 检测页面设置
func (p *OfdProcessor) detectPageSettings(parser *OfdParser, sf *extractor.StyleFeatures) {
	width, height := parser.GetPageSize()

	if width > 0 && height > 0 {
		sf.PageWidth = width
		sf.PageHeight = height

		// 检查是否为A4（210mm x 297mm，允许±3mm误差）
		isA4Width := width >= 207 && width <= 213
		isA4Height := height >= 294 && height <= 300

		// 也检查横向A4
		isA4Landscape := (width >= 294 && width <= 300) &&
			(height >= 207 && height <= 213)

		sf.IsA4Paper = (isA4Width && isA4Height) || isA4Landscape

		if sf.IsA4Paper {
			sf.StyleReasons = append(sf.StyleReasons, "纸张为A4规格")
		}
	}
}

// detectColors 检测颜色
func (p *OfdProcessor) detectColors(parser *OfdParser, sf *extractor.StyleFeatures) {
	colorInfo := parser.DetectColors()

	if colorInfo.HasRedColor {
		sf.HasRedText = true
		sf.RedTextCount = colorInfo.RedColorCount

		// 检查是否可能是红头（第一页有红色）
		// OFD中如果有红色，通常用于红头
		sf.HasRedHeader = true
		sf.StyleReasons = append(sf.StyleReasons, "检测到红色内容（可能是红头）")
	}
}

// detectFonts 检测字体
func (p *OfdProcessor) detectFonts(parser *OfdParser, sf *extractor.StyleFeatures) {
	fonts := parser.GetFonts()

	// 公文常用字体
	officialFontPatterns := []string{
		"simsun", "宋体", "song",
		"simhei", "黑体", "hei",
		"fangsong", "仿宋", "fang",
		"kaiti", "楷体", "kai",
		"stsong", "sthei", "stfang", "stkai",
		"xiaobiaosong", "小标���",
		"fzsxs", "方正小标宋",
		"fzxbs", "fzhtk",
	}

	for _, font := range fonts {
		fontName := strings.ToLower(font.FontName)
		familyName := strings.ToLower(font.FamilyName)

		for _, pattern := range officialFontPatterns {
			if strings.Contains(fontName, pattern) || strings.Contains(familyName, pattern) {
				sf.HasOfficialFonts = true
				if font.FontName != "" {
					sf.MainFontName = font.FontName
				} else {
					sf.MainFontName = font.FamilyName
				}
				sf.StyleReasons = append(sf.StyleReasons, "使用公文常用字体: "+sf.MainFontName)
				return
			}
		}
	}
}

// detectSignatures 检测签章
func (p *OfdProcessor) detectSignatures(parser *OfdParser, sf *extractor.StyleFeatures) {
	if parser.HasSignature() {
		sf.HasSealImage = true
		sf.SealImageHint = "检测到电子签章"
		sf.StyleReasons = append(sf.StyleReasons, "检测到电子签章")
	}
}

// calculateStyleScore 计算版式得分
func (p *OfdProcessor) calculateStyleScore(sf *extractor.StyleFeatures) {
	score := 0.0

	// 红色内容 (0.25)
	if sf.HasRedHeader {
		score += 0.18
	} else if sf.HasRedText {
		score += 0.10
	}

	// 签章 (0.20)
	if sf.HasSealImage {
		score += 0.20
	}

	// 页面设置 (0.15)
	if sf.IsA4Paper {
		score += 0.15
	}

	// 字体 (0.10)
	if sf.HasOfficialFonts {
		score += 0.10
	}

	// OFD格式本身就是版式文档，加分
	score += 0.05

	sf.StyleScore = score
	sf.IsOfficialStyle = score >= 0.3 || sf.HasRedHeader || sf.HasSealImage

	if len(sf.StyleReasons) == 0 {
		sf.StyleReasons = append(sf.StyleReasons, "OFD版式文档格式")
	}
}

// normalizeOfdText 规范化OFD提取的文本
func normalizeOfdText(text string) string {
	// 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 移除多余空白
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		// 移除行首尾空白
		line = strings.TrimSpace(line)

		// 合并连续空格
		spaceRegex := regexp.MustCompile(`\s+`)
		line = spaceRegex.ReplaceAllString(line, " ")

		// 移除CJK字符之间的空格
		line = removeCJKSpacesInLine(line)

		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	text = strings.Join(cleanedLines, "\n")

	// 合并连续空行
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(text)
}

// removeCJKSpacesInLine 移除行内CJK字符之间的空格
func removeCJKSpacesInLine(line string) string {
	runes := []rune(line)
	if len(runes) == 0 {
		return ""
	}

	var result strings.Builder

	for i := 0; i < len(runes); i++ {
		current := runes[i]

		if current == ' ' {
			// 检查前后字符
			var prev, next rune
			if i > 0 {
				prev = runes[i-1]
			}
			if i+1 < len(runes) {
				next = runes[i+1]
			}

			// 如果前后都是CJK字符，跳过空格
			if isCJKRune(prev) && isCJKRune(next) {
				continue
			}

			// 如果前后是CJK字符和数字，跳过空格
			if (isCJKRune(prev) && isDigitRune(next)) ||
				(isDigitRune(prev) && isCJKRune(next)) {
				continue
			}
		}

		result.WriteRune(current)
	}

	return result.String()
}

// isCJKRune 检查是否是CJK字符
func isCJKRune(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x20000 && r <= 0x2A6DF)
}

// isDigitRune 检查是否是数字
func isDigitRune(r rune) bool {
	return r >= '0' && r <= '9'
}