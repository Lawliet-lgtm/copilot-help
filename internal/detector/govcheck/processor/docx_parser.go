package processor

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
)

// DocxStyleParser DOCX样式解析器
type DocxStyleParser struct {
	zipReader *zip.Reader
	features  *DocxStyleFeatures
}

// NewDocxStyleParser 创建样式解析器
func NewDocxStyleParser(zipReader *zip.Reader) *DocxStyleParser {
	return &DocxStyleParser{
		zipReader: zipReader,
		features:  NewDocxStyleFeatures(),
	}
}

// Parse 解析所有样式特征
func (p *DocxStyleParser) Parse() (*DocxStyleFeatures, error) {
	// 1. 解析主文档内容（颜色、字体、段落）
	if err := p.parseDocument(); err != nil {
		// 不中断，继续解析其他内容
	}

	// 2. 解析样式定义
	if err := p.parseStyles(); err != nil {
		// 不中断
	}

	// 3. 解析页面设置
	if err := p.parseSettings(); err != nil {
		// 不中断
	}

	// 4. 解析嵌入图片
	if err := p.parseImages(); err != nil {
		// 不中断
	}

	// 5. 计算综合得分
	p.calculateStyleScore()

	return p.features, nil
}

// ============================================================
// 文档内容解析 (word/document.xml)
// ============================================================

// parseDocument 解析主文档
func (p *DocxStyleParser) parseDocument() error {
	content, err := p.readZipFile("word/document.xml")
	if err != nil {
		return err
	}

	return p.parseDocumentXML(content)
}

// parseDocumentXML 解析文档XML
func (p *DocxStyleParser) parseDocumentXML(content []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(content))

	var currentColor string
	var currentFontName string
	var currentFontSize float64
	var currentIsBold bool
	var currentText strings.Builder
	var paragraphIndex int

	inRun := false      // 是否在 <w:r> 内
	inText := false     // 是否在 <w:t> 内
	inParagraph := false

	colorCounts := make(map[string]int)
	fontInfoMap := make(map[string]*FontInfo)

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p": // 段落开始
				inParagraph = true
				paragraphIndex++
				p.features.ParagraphFeatures.TotalParagraphs++

				// 检查段落对齐方式
				for _, attr := range t.Attr {
					if attr.Name.Local == "jc" && attr.Value == "center" {
						p.features.ParagraphFeatures.CenteredCount++
					}
				}

			case "pPr": // 段落属性
				// 解析段落属性
				p.parseParagraphProperties(decoder)

			case "r": // Run（文本运行）开始
				inRun = true
				currentColor = ""
				currentFontName = ""
				currentFontSize = 0
				currentIsBold = false
				currentText.Reset()

			case "rPr": // Run属性
				if inRun {
					color, font, size, bold := p.parseRunProperties(decoder)
					if color != "" {
						currentColor = color
					}
					if font != "" {
						currentFontName = font
					}
					if size > 0 {
						currentFontSize = size
					}
					currentIsBold = bold
				}

			case "t": // 文本内容
				inText = true

			case "color": // 颜色属性
				for _, attr := range t.Attr {
					if attr.Name.Local == "val" {
						currentColor = attr.Value
					}
				}

			case "sz": // 字号 (半磅为单位)
				for _, attr := range t.Attr {
					if attr.Name.Local == "val" {
						if size, err := strconv.ParseFloat(attr.Value, 64); err == nil {
							currentFontSize = size / 2.0 // 转换为磅
						}
					}
				}

			case "rFonts": // 字体
				for _, attr := range t.Attr {
					if attr.Name.Local == "eastAsia" || attr.Name.Local == "ascii" || attr.Name.Local == "hAnsi" {
						if attr.Value != "" {
							currentFontName = attr.Value
							break
						}
					}
				}

			case "b": // 加粗
				currentIsBold = true
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "p": // 段落结束
				inParagraph = false

			case "r": // Run结束
				if inRun {
					text := strings.TrimSpace(currentText.String())

					// 记录颜色信息
					if currentColor != "" {
						colorCounts[currentColor]++

						if IsRedColor(currentColor) {
							p.features.ColorFeatures.HasRedText = true
							p.features.ColorFeatures.RedTextCount++

							// 记录红色文本示例
							if text != "" && len(p.features.ColorFeatures.RedTextSamples) < 5 {
								p.features.ColorFeatures.RedTextSamples = append(
									p.features.ColorFeatures.RedTextSamples, text)
							}

							// 判断是否为红头（前几个段落的红色文本）
							if paragraphIndex <= 3 {
								p.features.ColorFeatures.HasRedHeader = true
							}
						}
					}

					// 记录字体信息
					if currentFontName != "" || currentFontSize > 0 {
						fontKey := currentFontName
						if fontKey == "" {
							fontKey = "default"
						}

						if existing, ok := fontInfoMap[fontKey]; ok {
							existing.Count++
						} else {
							fontInfoMap[fontKey] = &FontInfo{
								Name:     currentFontName,
								Size:     currentFontSize,
								SizeDesc: GetFontSizeDesc(currentFontSize),
								Count:    1,
								IsBold:   currentIsBold,
								Color:    currentColor,
							}
						}

						// 检查公文字体
						if IsOfficialFont(currentFontName) {
							p.features.FontFeatures.HasOfficialFonts = true
						}

						// 统计字体分布
						p.updateFontDistribution(currentFontName)

						// 检查标题字号
						if IsTitleFontSize(currentFontSize) && currentIsBold {
							p.features.FontFeatures.TitleFontMatch = true
						}

						// 检查正文字号
						if IsBodyFontSize(currentFontSize) {
							p.features.FontFeatures.BodyFontMatch = true
						}
					}

					inRun = false
				}

			case "t":
				inText = false
			}

		case xml.CharData:
			if inText {
				currentText.Write(t)
			}
		}

		_ = inParagraph // 避免未使用警告
	}

	// 整理颜色信息
	for color, count := range colorCounts {
		if count > 0 {
			p.features.ColorFeatures.DominantColors = append(
				p.features.ColorFeatures.DominantColors, color)
		}
	}

	// 整理字体信息
	for _, info := range fontInfoMap {
		p.features.FontFeatures.UsedFonts = append(p.features.FontFeatures.UsedFonts, *info)
	}

	return nil
}

// parseRunProperties 解析Run属性
func (p *DocxStyleParser) parseRunProperties(decoder *xml.Decoder) (color, font string, size float64, bold bool) {
	depth := 1

	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			depth++

			switch t.Name.Local {
			case "color":
				for _, attr := range t.Attr {
					if attr.Name.Local == "val" {
						color = attr.Value
					}
				}

			case "sz", "szCs":
				for _, attr := range t.Attr {
					if attr.Name.Local == "val" {
						if s, err := strconv.ParseFloat(attr.Value, 64); err == nil {
							size = s / 2.0 // 半磅转磅
						}
					}
				}

			case "rFonts":
				for _, attr := range t.Attr {
					if attr.Name.Local == "eastAsia" {
						font = attr.Value
						break
					}
					if attr.Name.Local == "ascii" && font == "" {
						font = attr.Value
					}
				}

			case "b":
				bold = true
			}

		case xml.EndElement:
			depth--
			if t.Name.Local == "rPr" {
				return
			}
		}
	}

	return
}

// parseParagraphProperties 解析段落属性
func (p *DocxStyleParser) parseParagraphProperties(decoder *xml.Decoder) {
	depth := 1

	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			depth++

			switch t.Name.Local {
			case "jc": // 对齐方式
				for _, attr := range t.Attr {
					if attr.Name.Local == "val" && attr.Value == "center" {
						p.features.ParagraphFeatures.CenteredCount++
						p.features.ParagraphFeatures.HasCenteredTitle = true
					}
				}

			case "ind": // 缩进
				for _, attr := range t.Attr {
					if attr.Name.Local == "firstLine" || attr.Name.Local == "firstLineChars" {
						if val, err := strconv.ParseFloat(attr.Value, 64); err == nil {
							// firstLineChars 以百分之一字符为单位
							if attr.Name.Local == "firstLineChars" {
								p.features.ParagraphFeatures.FirstLineIndent = val / 100.0
							} else {
								// firstLine 以 twips 为单位，大约 420 twips = 2字符
								p.features.ParagraphFeatures.FirstLineIndent = val / 210.0
							}
						}
					}
				}

			case "spacing": // 行距
				for _, attr := range t.Attr {
					if attr.Name.Local == "line" {
						if val, err := strconv.ParseFloat(attr.Value, 64); err == nil {
							// 行距以 twips 为单位
							spacingPt := TwipsToPt(val)
							p.features.ParagraphFeatures.LineSpacing = spacingPt
							p.features.ParagraphFeatures.LineSpacingMatch = IsStandardLineSpacing(spacingPt)
						}
					}
				}
			}

		case xml.EndElement:
			depth--
			if t.Name.Local == "pPr" {
				return
			}
		}
	}
}

// updateFontDistribution 更新字体分布统计
func (p *DocxStyleParser) updateFontDistribution(fontName string) {
	fontLower := toLower(fontName)

	if containsString(fontLower, "fangsong") || containsString(fontLower, "仿宋") {
		p.features.FontFeatures.FontDistribution.FangsongCount++
	} else if containsString(fontLower, "heiti") || containsString(fontLower, "simhei") || containsString(fontLower, "黑体") {
		p.features.FontFeatures.FontDistribution.HeiCount++
	} else if containsString(fontLower, "kaiti") || containsString(fontLower, "楷体") {
		p.features.FontFeatures.FontDistribution.KaiCount++
	} else if containsString(fontLower, "song") || containsString(fontLower, "simsun") || containsString(fontLower, "宋体") {
		p.features.FontFeatures.FontDistribution.SongCount++
	} else {
		p.features.FontFeatures.FontDistribution.OtherCount++
	}
}

// ============================================================
// 样式定义解析 (word/styles.xml)
// ============================================================

// parseStyles 解析样式定义文件
func (p *DocxStyleParser) parseStyles() error {
	content, err := p.readZipFile("word/styles.xml")
	if err != nil {
		return err
	}

	return p.parseStylesXML(content)
}

// parseStylesXML 解析样式XML
func (p *DocxStyleParser) parseStylesXML(content []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(content))

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// 检查默认字体
			if t.Name.Local == "rFonts" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "eastAsia" || attr.Name.Local == "ascii" {
						if IsOfficialFont(attr.Value) {
							p.features.FontFeatures.HasOfficialFonts = true
						}
					}
				}
			}

			// 检查默认颜色
			if t.Name.Local == "color" {
				for _, attr := range t.Attr {
					if attr.Name.Local == "val" {
						if IsRedColor(attr.Value) {
							p.features.ColorFeatures.HasRedText = true
						}
					}
				}
			}
		}
	}

	return nil
}

// ============================================================
// 页面设置解析 (word/document.xml 中的 sectPr)
// ============================================================

// parseSettings 解析页面设置
func (p *DocxStyleParser) parseSettings() error {
	// 页面设置通常在 document.xml 的 sectPr 元素中
	content, err := p.readZipFile("word/document.xml")
	if err != nil {
		return err
	}

	return p.parseSectionProperties(content)
}

// parseSectionProperties 解析节属性（页面设置）
func (p *DocxStyleParser) parseSectionProperties(content []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(content))

	inSectPr := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "sectPr" {
				inSectPr = true
			}

			if inSectPr {
				switch t.Name.Local {
				case "pgSz": // 页面大小
					for _, attr := range t.Attr {
						switch attr.Name.Local {
						case "w":
							if w, err := strconv.ParseFloat(attr.Value, 64); err == nil {
								p.features.PageFeatures.PageWidth = TwipsToMM(w)
							}
						case "h":
							if h, err := strconv.ParseFloat(attr.Value, 64); err == nil {
								p.features.PageFeatures.PageHeight = TwipsToMM(h)
							}
						}
					}

					// 检查是否为A4
					p.features.PageFeatures.IsA4 = IsA4Paper(
						p.features.PageFeatures.PageWidth,
						p.features.PageFeatures.PageHeight)

					if p.features.PageFeatures.IsA4 {
						p.features.PageFeatures.PaperSizeDesc = "A4"
					} else {
						p.features.PageFeatures.PaperSizeDesc = "其他"
					}

				case "pgMar": // 页边距
					for _, attr := range t.Attr {
						switch attr.Name.Local {
						case "top":
							if v, err := strconv.ParseFloat(attr.Value, 64); err == nil {
								p.features.PageFeatures.MarginTop = TwipsToMM(v)
							}
						case "bottom":
							if v, err := strconv.ParseFloat(attr.Value, 64); err == nil {
								p.features.PageFeatures.MarginBottom = TwipsToMM(v)
							}
						case "left":
							if v, err := strconv.ParseFloat(attr.Value, 64); err == nil {
								p.features.PageFeatures.MarginLeft = TwipsToMM(v)
							}
						case "right":
							if v, err := strconv.ParseFloat(attr.Value, 64); err == nil {
								p.features.PageFeatures.MarginRight = TwipsToMM(v)
							}
						case "header":
							if v, err := strconv.ParseFloat(attr.Value, 64); err == nil {
								p.features.PageFeatures.HeaderDistance = TwipsToMM(v)
								p.features.PageFeatures.HasHeader = true
							}
						case "footer":
							if v, err := strconv.ParseFloat(attr.Value, 64); err == nil {
								p.features.PageFeatures.FooterDistance = TwipsToMM(v)
								p.features.PageFeatures.HasFooter = true
							}
						}
					}

					// 检查页边距是否符合标准
					p.features.PageFeatures.MarginMatch = CheckMargins(
						p.features.PageFeatures.MarginTop,
						p.features.PageFeatures.MarginBottom,
						p.features.PageFeatures.MarginLeft,
						p.features.PageFeatures.MarginRight)

				case "headerReference":
					p.features.PageFeatures.HasHeader = true

				case "footerReference":
					p.features.PageFeatures.HasFooter = true
				}
			}

		case xml.EndElement:
			if t.Name.Local == "sectPr" {
				inSectPr = false
			}
		}
	}

	return nil
}

// ============================================================
// 嵌入图片解析
// ============================================================

// parseImages 解析嵌入图片
func (p *DocxStyleParser) parseImages() error {
	// 查找 word/media/ 目录下的图片
	for _, file := range p.zipReader.File {
		if strings.HasPrefix(file.Name, "word/media/") {
			p.features.EmbeddedFeatures.HasImages = true
			p.features.EmbeddedFeatures.ImageCount++

			// 获取图片信息
			imgMeta := ImageMeta{
				Name: file.Name,
			}

			// 判断图片类型
			nameLower := toLower(file.Name)
			if strings.HasSuffix(nameLower, ".png") {
				imgMeta.Type = "png"
			} else if strings.HasSuffix(nameLower, ".jpg") || strings.HasSuffix(nameLower, ".jpeg") {
				imgMeta.Type = "jpeg"
			} else if strings.HasSuffix(nameLower, ".gif") {
				imgMeta.Type = "gif"
			} else if strings.HasSuffix(nameLower, ".emf") || strings.HasSuffix(nameLower, ".wmf") {
				imgMeta.Type = "metafile"
			} else {
				imgMeta.Type = "unknown"
			}

			// 读取图片头部检测是否可能为红色图像
			if imgMeta.Type == "png" || imgMeta.Type == "jpeg" {
				if isRed, err := p.checkImageRedish(file); err == nil && isRed {
					imgMeta.IsRedish = true
					p.features.EmbeddedFeatures.HasSealImage = true
					p.features.EmbeddedFeatures.SealImageHint = "检测到可能的红色图章图片"
				}
			}

			p.features.EmbeddedFeatures.Images = append(
				p.features.EmbeddedFeatures.Images, imgMeta)
		}
	}

	return nil
}

// checkImageRedish 检查图片是否为红色主导
func (p *DocxStyleParser) checkImageRedish(file *zip.File) (bool, error) {
	rc, err := file.Open()
	if err != nil {
		return false, err
	}
	defer rc.Close()

	// 读取前几KB进行简单分析
	header := make([]byte, 4096)
	n, err := rc.Read(header)
	if err != nil && err != io.EOF {
		return false, err
	}
	header = header[:n]

	// 简单的红色检测逻辑
	// 对于 JPEG/PNG，检查是否有大量红色像素值
	// 这是一个简化的启发式方法

	redCount := 0
	totalSamples := 0

	// 跳过文件头，每隔几个字节采样
	for i := 100; i < len(header)-3; i += 4 {
		r := int(header[i])
		g := int(header[i+1])
		b := int(header[i+2])

		totalSamples++

		// 检查是否为红色 (R高，G和B低)
		if r > 150 && g < 100 && b < 100 {
			redCount++
		}
	}

	// 如果红色采样点超过一定比例，认为是红色图像
	if totalSamples > 0 && float64(redCount)/float64(totalSamples) > 0.1 {
		return true, nil
	}

	return false, nil
}

// ============================================================
// 综合评分计算
// ============================================================

// calculateStyleScore 计算版式综合得分
func (p *DocxStyleParser) calculateStyleScore() {
	score := 0.0
	maxScore := 0.0

	// 颜色特征评分 (最高 0.25 分)
	maxScore += 0.25
	if p.features.ColorFeatures.HasRedText {
		score += 0.15
		p.features.StyleReasons = append(p.features.StyleReasons, "检测到红色文本")
	}
	if p.features.ColorFeatures.HasRedHeader {
		score += 0.10
		p.features.StyleReasons = append(p.features.StyleReasons, "检测到红头（顶部红色文本）")
	}

	// 字体特征评分 (最高 0.25 分)
	maxScore += 0.25
	if p.features.FontFeatures.HasOfficialFonts {
		score += 0.10
		p.features.StyleReasons = append(p.features.StyleReasons, "使用公文标准字体")
	}
	if p.features.FontFeatures.TitleFontMatch {
		score += 0.08
		p.features.StyleReasons = append(p.features.StyleReasons, "标题字号符合标准（二号）")
	}
	if p.features.FontFeatures.BodyFontMatch {
		score += 0.07
		p.features.StyleReasons = append(p.features.StyleReasons, "正文字号符合标准（三号）")
	}

	// 页面设置评分 (最高 0.25 分)
	maxScore += 0.25
	if p.features.PageFeatures.IsA4 {
		score += 0.10
		p.features.StyleReasons = append(p.features.StyleReasons, "纸张为A4规格")
	}
	if p.features.PageFeatures.MarginMatch {
		score += 0.15
		p.features.StyleReasons = append(p.features.StyleReasons, "页边距符合公文标准")
	}

	// 段落特征评分 (最高 0.15 分)
	maxScore += 0.15
	if p.features.ParagraphFeatures.HasCenteredTitle {
		score += 0.08
		p.features.StyleReasons = append(p.features.StyleReasons, "存在居中标题")
	}
	if p.features.ParagraphFeatures.LineSpacingMatch {
		score += 0.07
		p.features.StyleReasons = append(p.features.StyleReasons, "行距符合标准（28磅）")
	}

	// 嵌入对象评分 (最高 0.10 分)
	maxScore += 0.10
	if p.features.EmbeddedFeatures.HasSealImage {
		score += 0.10
		p.features.StyleReasons = append(p.features.StyleReasons, "检测到可能的印章图片")
	}

	// 归一化得分
	if maxScore > 0 {
		p.features.StyleScore = score / maxScore
	}

	// 判定是否符合公文版式
	// 条件：得分超过 0.4 或者 有红头+有红色文本
	p.features.IsOfficialStyle = p.features.StyleScore >= 0.4 ||
		(p.features.ColorFeatures.HasRedHeader && p.features.ColorFeatures.HasRedText)

	if len(p.features.StyleReasons) == 0 {
		p.features.StyleReasons = append(p.features.StyleReasons, "未检测到明显的公文版式特征")
	}
}

// ============================================================
// 工具方法
// ============================================================

// readZipFile 读取ZIP中的文件
func (p *DocxStyleParser) readZipFile(name string) ([]byte, error) {
	for _, file := range p.zipReader.File {
		if file.Name == name {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			return io.ReadAll(rc)
		}
	}

	return nil, nil // 文件不存在返回nil
}

// GetFeatures 获取解析结果
func (p *DocxStyleParser) GetFeatures() *DocxStyleFeatures {
	return p.features
}