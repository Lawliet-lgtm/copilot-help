package processor

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"linuxFileWatcher/internal/detector/govcheck/extractor"
)

// DocxProcessor DOCX文档处理器
type DocxProcessor struct {
	base   *BaseProcessor
	config *DocxProcessorConfig
}

// DocxProcessorConfig DOCX处理器配置
type DocxProcessorConfig struct {
	MaxFileSize      int64 // 最大文件大小 (字节)
	ExtractHeaders   bool  // 是否提取页眉
	ExtractFooters   bool  // 是否提取页脚
	NormalizeSpace   bool  // 是否规范化空白字符
	PreserveNewlines bool  // 是否保留换行
	ParseStyle       bool  // 是否解析版式特征
}

// DefaultDocxProcessorConfig 返回默认配置
func DefaultDocxProcessorConfig() *DocxProcessorConfig {
	return &DocxProcessorConfig{
		MaxFileSize:      100 * 1024 * 1024, // 100MB
		ExtractHeaders:   true,
		ExtractFooters:   true,
		NormalizeSpace:   true,
		PreserveNewlines: true,
		ParseStyle:       true,
	}
}

// NewDocxProcessor 创建DOCX处理器
func NewDocxProcessor() *DocxProcessor {
	return NewDocxProcessorWithConfig(nil)
}

// NewDocxProcessorWithConfig 使用指定配置创建DOCX处理器
func NewDocxProcessorWithConfig(config *DocxProcessorConfig) *DocxProcessor {
	if config == nil {
		config = DefaultDocxProcessorConfig()
	}

	base := NewBaseProcessor(
		"DocxProcessor",
		"Microsoft Word文档处理器 (DOCX/DOCM/DOT)",
		[]string{"docx", "docm", "dotx", "dotm"},
	)

	return &DocxProcessor{
		base:   base,
		config: config,
	}
}

// Name 返回处理器名称
func (p *DocxProcessor) Name() string {
	return p.base.Name()
}

// Description 返回处理器描述
func (p *DocxProcessor) Description() string {
	return p.base.Description()
}

// SupportedTypes 返回支持的文件类型
func (p *DocxProcessor) SupportedTypes() []string {
	return p.base.SupportedTypes()
}

// Process 处理DOCX文件（实现 Processor 接口）
func (p *DocxProcessor) Process(filePath string) (string, error) {
	result, err := p.ProcessWithStyle(filePath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ProcessWithStyle 处理DOCX文件并返回版式特征（实现 StyleProcessor 接口）
func (p *DocxProcessor) ProcessWithStyle(filePath string) (*ProcessResultWithStyle, error) {
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
		return nil, NewProcessorError(p.Name(), filePath, "打开DOCX文件", err)
	}
	defer zipReader.Close()

	// 1. 提取文本内容
	text, err := p.extractText(&zipReader.Reader)
	if err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "提取文本内容", err)
	}
	result.Text = text

	// 2. 解析版式特征（如果启用）
	if p.config.ParseStyle {
		styleParser := NewDocxStyleParser(&zipReader.Reader)
		docxStyleFeatures, err := styleParser.Parse()
		if err == nil && docxStyleFeatures != nil {
			// 转换为 extractor.StyleFeatures
			result.StyleFeatures = p.convertStyleFeatures(docxStyleFeatures)
			result.HasStyle = true
		}
	}

	return result, nil
}

// convertStyleFeatures 将 DocxStyleFeatures 转换为 extractor.StyleFeatures
func (p *DocxProcessor) convertStyleFeatures(dsf *DocxStyleFeatures) *extractor.StyleFeatures {
	sf := &extractor.StyleFeatures{
		// 颜色特征
		HasRedText:   dsf.ColorFeatures.HasRedText,
		HasRedHeader: dsf.ColorFeatures.HasRedHeader,
		RedTextCount: dsf.ColorFeatures.RedTextCount,
		RedSamples:   dsf.ColorFeatures.RedTextSamples,

		// 字体特征
		HasOfficialFonts: dsf.FontFeatures.HasOfficialFonts,
		TitleFontMatch:   dsf.FontFeatures.TitleFontMatch,
		BodyFontMatch:    dsf.FontFeatures.BodyFontMatch,

		// 页面特征
		IsA4Paper:   dsf.PageFeatures.IsA4,
		MarginMatch: dsf.PageFeatures.MarginMatch,
		PageWidth:   dsf.PageFeatures.PageWidth,
		PageHeight:  dsf.PageFeatures.PageHeight,

		// 段落特征
		HasCenteredTitle: dsf.ParagraphFeatures.HasCenteredTitle,
		LineSpacingMatch: dsf.ParagraphFeatures.LineSpacingMatch,
		LineSpacing:      dsf.ParagraphFeatures.LineSpacing,

		// 印章特征
		HasSealImage:  dsf.EmbeddedFeatures.HasSealImage,
		SealImageHint: dsf.EmbeddedFeatures.SealImageHint,

		// 综合评估
		StyleScore:      dsf.StyleScore,
		IsOfficialStyle: dsf.IsOfficialStyle,
		StyleReasons:    dsf.StyleReasons,
	}

	// 提取主要字体信息
	if len(dsf.FontFeatures.UsedFonts) > 0 {
		// 找到使用次数最多的字体
		var maxCount int
		for _, font := range dsf.FontFeatures.UsedFonts {
			if font.Count > maxCount {
				maxCount = font.Count
				sf.MainFontName = font.Name
				sf.MainFontSize = font.Size
			}
		}
	}

	return sf
}

// extractText 提取所有文本内容
func (p *DocxProcessor) extractText(zipReader *zip.Reader) (string, error) {
	var textBuilder strings.Builder

	// 提取页眉
	if p.config.ExtractHeaders {
		headers := p.extractHeaders(zipReader)
		if headers != "" {
			textBuilder.WriteString(headers)
			textBuilder.WriteString("\n")
		}
	}

	// 提取主文档内容
	mainContent, err := p.extractDocument(zipReader)
	if err != nil {
		return "", err
	}
	textBuilder.WriteString(mainContent)

	// 提取页脚
	if p.config.ExtractFooters {
		footers := p.extractFooters(zipReader)
		if footers != "" {
			textBuilder.WriteString("\n")
			textBuilder.WriteString(footers)
		}
	}

	text := textBuilder.String()

	// 规范化空白字符
	if p.config.NormalizeSpace {
		text = p.normalizeText(text)
	}

	return text, nil
}

// extractDocument 提取主文档内容
func (p *DocxProcessor) extractDocument(zipReader *zip.Reader) (string, error) {
	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			return p.extractXMLText(file)
		}
	}

	return "", fmt.Errorf("未找到文档内容 (word/document.xml)")
}

// extractHeaders 提取所有页眉
func (p *DocxProcessor) extractHeaders(zipReader *zip.Reader) string {
	var headers []string

	for _, file := range zipReader.File {
		if strings.HasPrefix(file.Name, "word/header") && strings.HasSuffix(file.Name, ".xml") {
			if text, err := p.extractXMLText(file); err == nil && text != "" {
				headers = append(headers, text)
			}
		}
	}

	return strings.Join(headers, "\n")
}

// extractFooters 提取所有页脚
func (p *DocxProcessor) extractFooters(zipReader *zip.Reader) string {
	var footers []string

	for _, file := range zipReader.File {
		if strings.HasPrefix(file.Name, "word/footer") && strings.HasSuffix(file.Name, ".xml") {
			if text, err := p.extractXMLText(file); err == nil && text != "" {
				footers = append(footers, text)
			}
		}
	}

	return strings.Join(footers, "\n")
}

// extractXMLText 从XML文件中提取文本
func (p *DocxProcessor) extractXMLText(file *zip.File) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	return p.parseWordML(content)
}

// parseWordML 解析Word ML XML格式
func (p *DocxProcessor) parseWordML(content []byte) (string, error) {
	var textBuilder strings.Builder

	decoder := xml.NewDecoder(bytes.NewReader(content))

	inParagraph := false
	inText := false
	inTab := false
	inBreak := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inParagraph = true
			case "t":
				inText = true
			case "tab":
				inTab = true
			case "br", "cr":
				inBreak = true
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if inParagraph {
					if p.config.PreserveNewlines {
						textBuilder.WriteString("\n")
					} else {
						textBuilder.WriteString(" ")
					}
					inParagraph = false
				}
			case "t":
				inText = false
			case "tab":
				if inTab {
					textBuilder.WriteString("\t")
					inTab = false
				}
			case "br", "cr":
				if inBreak {
					textBuilder.WriteString("\n")
					inBreak = false
				}
			}

		case xml.CharData:
			if inText {
				textBuilder.Write(t)
			}
		}
	}

	return textBuilder.String(), nil
}

// normalizeText 规范化文本
func (p *DocxProcessor) normalizeText(text string) string {
	// 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 移除行首尾空白
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	// 合并连续空行
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	// 合并连续空格
	re := regexp.MustCompile(`[ \t]+`)
	text = re.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// ============================================================
// ProcessResultWithStyle 定义
// ============================================================

// ProcessResultWithStyle 带版式信息的处理结果
type ProcessResultWithStyle struct {
	Text          string                   // 提取的文本内容
	StyleFeatures *extractor.StyleFeatures // 版式特征
	HasStyle      bool                     // 是否包含版式信息
}

// ============================================================
// WPS 文档处理器（复用 DOCX 处理逻辑）
// ============================================================

// WpsProcessor WPS文档处理器
type WpsProcessor struct {
	base       *BaseProcessor
	docxParser *DocxProcessor // 复用 DOCX 处理器
}

// NewWpsProcessor 创建WPS处理器
func NewWpsProcessor() *WpsProcessor {
	base := NewBaseProcessor(
		"WpsProcessor",
		"WPS文字文档处理器 (WPS/WPT)",
		[]string{"wps", "wpt"},
	)

	return &WpsProcessor{
		base:       base,
		docxParser: NewDocxProcessor(), // 复用 DOCX 处理器
	}
}

// Name 返回处理器名称
func (p *WpsProcessor) Name() string {
	return p.base.Name()
}

// Description 返回处理器描述
func (p *WpsProcessor) Description() string {
	return p.base.Description()
}

// SupportedTypes 返回支持的文件类型
func (p *WpsProcessor) SupportedTypes() []string {
	return p.base.SupportedTypes()
}

// Process 处理WPS文件
func (p *WpsProcessor) Process(filePath string) (string, error) {
	result, err := p.ProcessWithStyle(filePath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ProcessWithStyle 处理WPS文件并返回版式特征
func (p *WpsProcessor) ProcessWithStyle(filePath string) (*ProcessResultWithStyle, error) {
	// 检查文件
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "获取文件信息", err)
	}

	if info.Size() == 0 {
		return nil, NewProcessorError(p.Name(), filePath, "检查文件", fmt.Errorf("文件为空"))
	}

	// 方法1: 尝试作为 ZIP 格式处理（新版 WPS，兼容 DOCX）
	zipReader, err := zip.OpenReader(filePath)
	if err == nil {
		defer zipReader.Close()
		return p.processAsZip(filePath, &zipReader.Reader)
	}

	// 方法2: 尝试作为 OLE2 格式处理（旧版 WPS）
	return p.processAsOLE2(filePath)
}

// processAsZip 作为 ZIP 格式处理
func (p *WpsProcessor) processAsZip(filePath string, zipReader *zip.Reader) (*ProcessResultWithStyle, error) {
	result := &ProcessResultWithStyle{}

	// 尝试多种可能的文档路径
	documentPaths := []string{
		"word/document.xml",
		"wps/document.xml",
		"content.xml",
		"document.xml",
	}

	var mainContent string
	var foundPath string

	for _, docPath := range documentPaths {
		for _, file := range zipReader.File {
			if file.Name == docPath {
				content, err := p.extractXMLText(file)
				if err == nil && content != "" {
					mainContent = content
					foundPath = docPath
					break
				}
			}
		}
		if mainContent != "" {
			break
		}
	}

	// 如果标准��径找不到，遍历所有 XML 文件
	if mainContent == "" {
		var allText strings.Builder
		for _, file := range zipReader.File {
			if strings.HasSuffix(strings.ToLower(file.Name), ".xml") {
				content, err := p.extractXMLText(file)
				if err == nil && content != "" {
					allText.WriteString(content)
					allText.WriteString("\n")
				}
			}
		}
		mainContent = allText.String()
	}

	if strings.TrimSpace(mainContent) == "" {
		return nil, NewProcessorError(p.Name(), filePath, "提取内容", fmt.Errorf("未能从WPS文件提取文本"))
	}

	result.Text = normalizeWhitespacefordocx(mainContent)

	// 如果是 word/document.xml，尝试解析版式特征
	if foundPath == "word/document.xml" {
		styleParser := NewDocxStyleParser(zipReader)
		if features, err := styleParser.Parse(); err == nil && features != nil {
			result.StyleFeatures = p.docxParser.convertStyleFeatures(features)
			result.HasStyle = true
		}
	}

	return result, nil
}

// processAsOLE2 作为 OLE2 格式处理（旧版 WPS）
func (p *WpsProcessor) processAsOLE2(filePath string) (*ProcessResultWithStyle, error) {
	result := &ProcessResultWithStyle{}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, NewProcessorError(p.Name(), filePath, "读取文件", err)
	}

	// 检查是否是 OLE2 格式
	if len(content) < 8 {
		return nil, NewProcessorError(p.Name(), filePath, "检查格式", fmt.Errorf("文件太小"))
	}

	ole2Magic := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	if !bytes.Equal(content[:8], ole2Magic) {
		return nil, NewProcessorError(p.Name(), filePath, "检查格式", fmt.Errorf("不是有效的WPS文件格式"))
	}

	// 从 OLE2 中提取文本
	text := p.extractOLE2Text(content)
	if strings.TrimSpace(text) == "" {
		return nil, NewProcessorError(p.Name(), filePath, "提取内容", fmt.Errorf("未能从旧版WPS文件提取文本"))
	}

	result.Text = normalizeWhitespacefordocx(text)
	return result, nil
}

// extractXMLText 从 XML 文件中提取文本
func (p *WpsProcessor) extractXMLText(file *zip.File) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	return extractAllTextFromXML(content), nil
}

// extractOLE2Text 从 OLE2 内容中提取文本
func (p *WpsProcessor) extractOLE2Text(content []byte) string {
	var result strings.Builder

	// 方法1: 提取 UTF-16LE 编码的中文文本
	utf16Text := extractUTF16LEText(content)
	if utf16Text != "" {
		result.WriteString(utf16Text)
	}

	// 方法2: 提取可见 ASCII 文本
	asciiText := extractVisibleASCII(content)
	if asciiText != "" {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(asciiText)
	}

	return result.String()
}

// ============================================================
// 辅助函数
// ============================================================

// extractAllTextFromXML 从 XML 中提取所有文本（增强版）
func extractAllTextFromXML(content []byte) string {
	var textBuilder strings.Builder

	decoder := xml.NewDecoder(bytes.NewReader(content))

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			// 不需要特殊处理
		case xml.EndElement:
			if t.Name.Local == "p" || t.Name.Local == "paragraph" {
				textBuilder.WriteString("\n")
			}
		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" {
				textBuilder.WriteString(text)
			}
		}
	}

	return textBuilder.String()
}

// extractUTF16LEText 提取 UTF-16LE 编码的文本
func extractUTF16LEText(content []byte) string {
	var result strings.Builder
	var currentRun []rune

	for i := 0; i < len(content)-1; i += 2 {
		// 读取 UTF-16LE 字符
		lo := content[i]
		hi := content[i+1]
		char := rune(lo) | rune(hi)<<8

		// 检查是否是有效的中文或可打印字符
		if isValidTextChar(char) {
			currentRun = append(currentRun, char)
		} else {
			// 保存当前片段（如果足够长且包含中文）
			if len(currentRun) >= 2 && containsChineseRune(currentRun) {
				result.WriteString(string(currentRun))
				result.WriteString(" ")
			}
			currentRun = nil
		}
	}

	// 处理最后一个片段
	if len(currentRun) >= 2 && containsChineseRune(currentRun) {
		result.WriteString(string(currentRun))
	}

	return result.String()
}

// extractVisibleASCII 提取可见 ASCII 文本
func extractVisibleASCII(content []byte) string {
	var result strings.Builder
	var currentRun []byte

	for _, b := range content {
		if (b >= 0x20 && b <= 0x7E) || b == '\t' || b == '\n' {
			currentRun = append(currentRun, b)
		} else {
			if len(currentRun) >= 10 {
				text := strings.TrimSpace(string(currentRun))
				if text != "" && !isGarbageText(text) {
					result.WriteString(text)
					result.WriteString(" ")
				}
			}
			currentRun = nil
		}
	}

	if len(currentRun) >= 10 {
		text := strings.TrimSpace(string(currentRun))
		if text != "" && !isGarbageText(text) {
			result.WriteString(text)
		}
	}

	return result.String()
}

// isValidTextChar 检查是否是有效的文本字符
func isValidTextChar(r rune) bool {
	// ASCII 可打印字符
	if r >= 0x20 && r <= 0x7E {
		return true
	}
	// 换行、制表符
	if r == '\n' || r == '\r' || r == '\t' {
		return true
	}
	// 中文字符
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	// CJK 扩展
	if r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	// 中文标点
	if r >= 0x3000 && r <= 0x303F {
		return true
	}
	// 全角字符
	if r >= 0xFF00 && r <= 0xFFEF {
		return true
	}
	return false
}

// containsChineseRune 检查是否包含中文字符
func containsChineseRune(runes []rune) bool {
	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// isGarbageText 检查是否是垃圾文本
func isGarbageText(text string) bool {
	if len(text) < 3 {
		return true
	}

	// 检查重复字符
	charCount := make(map[rune]int)
	total := 0
	for _, r := range text {
		charCount[r]++
		total++
	}

	for _, count := range charCount {
		if float64(count)/float64(total) > 0.6 {
			return true
		}
	}

	return false
}

// normalizeWhitespace 规范化空白字符
func normalizeWhitespacefordocx(text string) string {
	// 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 移除行首尾空白
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	text = strings.Join(cleanLines, "\n")

	// 合并连续空行
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(text)
}