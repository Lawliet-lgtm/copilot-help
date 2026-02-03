package processor

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"linuxFileWatcher/internal/detector/govcheck/extractor"
)

// PdfProcessor PDF文档处理器
type PdfProcessor struct {
	base   *BaseProcessor
	config *PdfProcessorConfig
}

// PdfProcessorConfig PDF处理器配置
type PdfProcessorConfig struct {
	MaxFileSize    int64 // 最大文件大小 (字节)
	MaxPages       int   // 最大处理页数 (0=不限制)
	ExtractStyle   bool  // 是否提取版式特征
	NormalizeSpace bool  // 是否规范化空白字符
}

// DefaultPdfProcessorConfig 返回默认配置
func DefaultPdfProcessorConfig() *PdfProcessorConfig {
	return &PdfProcessorConfig{
		MaxFileSize:    100 * 1024 * 1024, // 100MB
		MaxPages:       0,                  // 不限制
		ExtractStyle:   true,
		NormalizeSpace: true,
	}
}

// NewPdfProcessor 创建PDF处理器
func NewPdfProcessor() *PdfProcessor {
	return NewPdfProcessorWithConfig(nil)
}

// NewPdfProcessorWithConfig 使用指定配置创建PDF处理器
func NewPdfProcessorWithConfig(config *PdfProcessorConfig) *PdfProcessor {
	if config == nil {
		config = DefaultPdfProcessorConfig()
	}

	base := NewBaseProcessor(
		"PdfProcessor",
		"PDF文档处理器",
		[]string{"pdf"},
	)

	return &PdfProcessor{
		base:   base,
		config: config,
	}
}

// Name 返回处理器名称
func (p *PdfProcessor) Name() string {
	return p.base.Name()
}

// Description 返回处理器描述
func (p *PdfProcessor) Description() string {
	return p.base.Description()
}

// SupportedTypes 返回支持的文件类型
func (p *PdfProcessor) SupportedTypes() []string {
	return p.base.SupportedTypes()
}

// Process 处理PDF文件（实现 Processor 接口）
func (p *PdfProcessor) Process(filePath string) (string, error) {
	result, err := p.ProcessWithStyle(filePath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ProcessWithStyle 处理PDF文件并返回版式特征（实现 StyleProcessor 接口）
func (p *PdfProcessor) ProcessWithStyle(filePath string) (*ProcessResultWithStyle, error) {
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

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "读取文件", err)
	}

	// 创建PDF解析器
	parser := NewPdfParser(data)
	if err := parser.Parse(); err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "解析PDF", err)
	}

	// 提取文本
	textExtractor := NewPdfTextExtractor(parser)
	text, err := textExtractor.ExtractText()
	if err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "提取文本", err)
	}

	// 规范化文本
	if p.config.NormalizeSpace {
		text = normalizePdfText(text)
	}

	result.Text = text

	// 提取版式特征
	if p.config.ExtractStyle {
		styleFeatures := p.extractStyleFeatures(parser, data)
		if styleFeatures != nil {
			result.StyleFeatures = styleFeatures
			result.HasStyle = true
		}
	}

	return result, nil
}

// extractStyleFeatures 提取PDF版式特征
func (p *PdfProcessor) extractStyleFeatures(parser *PdfParser, data []byte) *extractor.StyleFeatures {
	sf := &extractor.StyleFeatures{
		StyleReasons: make([]string, 0),
	}

	// 1. 检测页面设置
	p.detectPageSettings(parser, sf)

	// 2. 检测红色内容
	p.detectRedContent(parser, data, sf)

	// 3. 检测字体
	p.detectFonts(parser, sf)

	// 4. 计算版式得分
	p.calculateStyleScore(sf)

	return sf
}

// detectPageSettings 检测页面设置
func (p *PdfProcessor) detectPageSettings(parser *PdfParser, sf *extractor.StyleFeatures) {
	pages := parser.GetPages()
	if len(pages) == 0 {
		return
	}

	// 获取第一页的MediaBox
	firstPage := pages[0]
	mediaBox := firstPage.GetArray("MediaBox")

	if len(mediaBox) >= 4 {
		// MediaBox = [x1, y1, x2, y2]，单位是点(point)，1点 = 1/72英寸
		var x1, y1, x2, y2 float64

		if v, ok := mediaBox[0].(PdfIntObject); ok {
			x1 = float64(v.Value)
		} else if v, ok := mediaBox[0].(PdfRealObject); ok {
			x1 = v.Value
		}

		if v, ok := mediaBox[1].(PdfIntObject); ok {
			y1 = float64(v.Value)
		} else if v, ok := mediaBox[1].(PdfRealObject); ok {
			y1 = v.Value
		}

		if v, ok := mediaBox[2].(PdfIntObject); ok {
			x2 = float64(v.Value)
		} else if v, ok := mediaBox[2].(PdfRealObject); ok {
			x2 = v.Value
		}

		if v, ok := mediaBox[3].(PdfIntObject); ok {
			y2 = float64(v.Value)
		} else if v, ok := mediaBox[3].(PdfRealObject); ok {
			y2 = v.Value
		}

		// 计算页面尺寸（点转毫米：1点 = 25.4/72 mm）
		widthPt := x2 - x1
		heightPt := y2 - y1

		sf.PageWidth = widthPt * 25.4 / 72.0
		sf.PageHeight = heightPt * 25.4 / 72.0

		// 检查是否为A4（210mm x 297mm，允许±3mm误差）
		isA4Width := sf.PageWidth >= 207 && sf.PageWidth <= 213
		isA4Height := sf.PageHeight >= 294 && sf.PageHeight <= 300

		// 也检查横向A4
		isA4Landscape := (sf.PageWidth >= 294 && sf.PageWidth <= 300) &&
			(sf.PageHeight >= 207 && sf.PageHeight <= 213)

		sf.IsA4Paper = (isA4Width && isA4Height) || isA4Landscape

		if sf.IsA4Paper {
			sf.StyleReasons = append(sf.StyleReasons, "纸张为A4规格")
		}
	}
}

// detectRedContent 检测红色内容
func (p *PdfProcessor) detectRedContent(parser *PdfParser, data []byte, sf *extractor.StyleFeatures) {
	pages := parser.GetPages()

	for pageIdx, page := range pages {
		// 获取页面内容流
		contentsObj := page.Get("Contents")
		if contentsObj == nil {
			continue
		}

		contentData := p.getPageContentData(parser, contentsObj)
		if contentData == nil {
			continue
		}

		// 检测红色设置
		hasRed, redInfo := detectRedInContentStream(contentData)

		if hasRed {
			sf.HasRedText = true

			// 前3页的红色可能是红头
			if pageIdx < 1 {
				sf.HasRedHeader = true
				sf.StyleReasons = append(sf.StyleReasons, "检测到红色内容（可能是红头）")
			}

			if redInfo != "" && len(sf.RedSamples) < 3 {
				sf.RedSamples = append(sf.RedSamples, redInfo)
			}
		}
	}

	if sf.HasRedText && !sf.HasRedHeader {
		sf.StyleReasons = append(sf.StyleReasons, "检测到红色内容")
	}

	// 检测嵌入的图片（可能是印章）
	p.detectSealImages(parser, sf)
}

// getPageContentData 获取页面内容数据
func (p *PdfProcessor) getPageContentData(parser *PdfParser, contentsObj PdfObject) []byte {
	contents, err := parser.resolveRef(contentsObj)
	if err != nil {
		return nil
	}

	switch c := contents.(type) {
	case PdfStreamObject:
		data, _ := c.GetDecodedData()
		return data

	case PdfArrayObject:
		var allData []byte
		for _, item := range c.Items {
			streamObj, err := parser.resolveRef(item)
			if err != nil {
				continue
			}
			if stream, ok := streamObj.(PdfStreamObject); ok {
				data, err := stream.GetDecodedData()
				if err == nil {
					allData = append(allData, data...)
					allData = append(allData, '\n')
				}
			}
		}
		return allData
	}

	return nil
}

// detectRedInContentStream 在内容流中检测红色
func detectRedInContentStream(data []byte) (bool, string) {
	content := string(data)

	// 检测RGB红色设置
	// rg 操作符设置非描边颜色 (r g b rg)
	// RG 操作符设置描边颜色 (r g b RG)
	rgPattern := regexp.MustCompile(`([0-9.]+)\s+([0-9.]+)\s+([0-9.]+)\s+(?:rg|RG)`)
	matches := rgPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			r := parseFloat(match[1])
			g := parseFloat(match[2])
			b := parseFloat(match[3])

			// 检查是否为红色（R高，G和B低）
			if r >= 0.7 && g <= 0.3 && b <= 0.3 {
				return true, fmt.Sprintf("RGB(%.2f,%.2f,%.2f)", r, g, b)
			}
		}
	}

	// 检测CMYK红色设置
	// k 操作符设置非描边颜色 (c m y k k)
	// K 操作符设置描边颜色 (c m y k K)
	cmykPattern := regexp.MustCompile(`([0-9.]+)\s+([0-9.]+)\s+([0-9.]+)\s+([0-9.]+)\s+(?:k|K)`)
	matches = cmykPattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 5 {
			c := parseFloat(match[1])
			m := parseFloat(match[2])
			y := parseFloat(match[3])
			// k := parseFloat(match[4])

			// CMYK红色：C低，M和Y高
			if c <= 0.2 && m >= 0.8 && y >= 0.8 {
				return true, fmt.Sprintf("CMYK(%.2f,%.2f,%.2f)", c, m, y)
			}
		}
	}

	// 检测命名颜色���间中的红色
	// /DeviceRGB cs 1 0 0 sc
	if strings.Contains(content, "1 0 0 sc") || strings.Contains(content, "1 0 0 SC") {
		return true, "RGB(1,0,0)"
	}

	return false, ""
}

// detectSealImages 检测印章图片
func (p *PdfProcessor) detectSealImages(parser *PdfParser, sf *extractor.StyleFeatures) {
	pages := parser.GetPages()

	for _, page := range pages {
		// 获取Resources
		resourcesObj := page.Get("Resources")
		if resourcesObj == nil {
			continue
		}

		resources, err := parser.resolveRef(resourcesObj)
		if err != nil {
			continue
		}

		resourcesDict, ok := resources.(PdfDictObject)
		if !ok {
			continue
		}

		// 获取XObject
		xObjectObj := resourcesDict.Get("XObject")
		if xObjectObj == nil {
			continue
		}

		xObjects, err := parser.resolveRef(xObjectObj)
		if err != nil {
			continue
		}

		xObjectsDict, ok := xObjects.(PdfDictObject)
		if !ok {
			continue
		}

		// 遍历XObject
		for name, ref := range xObjectsDict.Dict {
			obj, err := parser.resolveRef(ref)
			if err != nil {
				continue
			}

			// 检查是否是图像
			if stream, ok := obj.(PdfStreamObject); ok {
				subtype := stream.Dict.GetString("Subtype")
				if subtype == "Image" {
					// 获取图像尺寸
					width := stream.Dict.GetInt("Width")
					height := stream.Dict.GetInt("Height")

					// 印章通常是方形或接近方形的图像
					if width > 50 && height > 50 {
						ratio := float64(width) / float64(height)
						if ratio >= 0.7 && ratio <= 1.4 {
							// 可能是印章
							sf.HasSealImage = true
							sf.SealImageHint = fmt.Sprintf("检测到可能的印章图片: %s (%dx%d)", name, width, height)
							sf.StyleReasons = append(sf.StyleReasons, "检测到可能的印章图片")
							return
						}
					}
				}
			}
		}
	}
}

// detectFonts 检测字体
func (p *PdfProcessor) detectFonts(parser *PdfParser, sf *extractor.StyleFeatures) {
	pages := parser.GetPages()
	fontNames := make(map[string]bool)

	for _, page := range pages {
		// 获取Resources
		resourcesObj := page.Get("Resources")
		if resourcesObj == nil {
			continue
		}

		resources, err := parser.resolveRef(resourcesObj)
		if err != nil {
			continue
		}

		resourcesDict, ok := resources.(PdfDictObject)
		if !ok {
			continue
		}

		// 获取Font
		fontObj := resourcesDict.Get("Font")
		if fontObj == nil {
			continue
		}

		fonts, err := parser.resolveRef(fontObj)
		if err != nil {
			continue
		}

		fontsDict, ok := fonts.(PdfDictObject)
		if !ok {
			continue
		}

		// 遍历字体
		for _, ref := range fontsDict.Dict {
			fontDef, err := parser.resolveRef(ref)
			if err != nil {
				continue
			}

			fontDict, ok := fontDef.(PdfDictObject)
			if !ok {
				continue
			}

			// 获取字体名称
			baseFontName := fontDict.GetString("BaseFont")
			if baseFontName != "" {
				fontNames[baseFontName] = true
			}
		}
	}

	// 检查是否包含公文常用字体
	officialFontPatterns := []string{
		"SimSun", "宋体", "Song",
		"SimHei", "黑体", "Hei",
		"FangSong", "仿宋", "Fang",
		"KaiTi", "楷体", "Kai",
		"STSong", "STHei", "STFang", "STKai",
		"XiaoBiaoSong", "小标宋",
	}

	for fontName := range fontNames {
		fontLower := strings.ToLower(fontName)
		for _, pattern := range officialFontPatterns {
			if strings.Contains(fontLower, strings.ToLower(pattern)) {
				sf.HasOfficialFonts = true
				sf.MainFontName = fontName
				sf.StyleReasons = append(sf.StyleReasons, "使用公文常用字体: "+fontName)
				return
			}
		}
	}
}

// calculateStyleScore 计算版式得分
func (p *PdfProcessor) calculateStyleScore(sf *extractor.StyleFeatures) {
	score := 0.0

	// 红色内容 (0.25)
	if sf.HasRedHeader {
		score += 0.18
	} else if sf.HasRedText {
		score += 0.10
	}

	// 印章 (0.15)
	if sf.HasSealImage {
		score += 0.15
	}

	// 页面设置 (0.15)
	if sf.IsA4Paper {
		score += 0.15
	}

	// 字体 (0.10)
	if sf.HasOfficialFonts {
		score += 0.10
	}

	// 综合判断
	sf.StyleScore = score
	sf.IsOfficialStyle = score >= 0.3 || sf.HasRedHeader

	if len(sf.StyleReasons) == 0 {
		sf.StyleReasons = append(sf.StyleReasons, "未检测到明显的公文版式特征")
	}
}

// normalizePdfText 规范化PDF提取的文本
func normalizePdfText(text string) string {
	// 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 移除多余的空白字符
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		// 移除行首尾空白
		line = strings.TrimSpace(line)

		// 合并连续空格
		spaceRegex := regexp.MustCompile(`\s+`)
		line = spaceRegex.ReplaceAllString(line, " ")

		cleanedLines = append(cleanedLines, line)
	}

	text = strings.Join(cleanedLines, "\n")

	// 合并连续空行（超过2个）
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(text)
}