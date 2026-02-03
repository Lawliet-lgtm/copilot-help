package processor

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ============================================================
// OFD 解析器
// ============================================================

// OfdParser OFD解析器
type OfdParser struct {
	zipReader    *zip.Reader
	docRoot      string              // 文档根目录，如 "Doc_0"
	pages        []*OfdParsedPage    // 解析后的页面
	fonts        map[string]string   // 字体ID -> 字体名称
	pageWidth    float64             // 页面宽度（mm）
	pageHeight   float64             // 页面高度（mm）
	hasSignature bool                // 是否有签章
	docTitle     string              // 文档标题
	colors       []string            // 检测到的颜色
	hasRedColor  bool                // 是否有红色
}

// OfdParsedPage 解析后的页面
type OfdParsedPage struct {
	PageID      string
	TextContent string
	Colors      []string
	HasRedColor bool
}

// NewOfdParser 创建OFD解析器
func NewOfdParser(zipReader *zip.Reader) *OfdParser {
	return &OfdParser{
		zipReader: zipReader,
		fonts:     make(map[string]string),
		pages:     make([]*OfdParsedPage, 0),
		colors:    make([]string, 0),
	}
}

// Parse 解析OFD文件
func (p *OfdParser) Parse() error {
	// 1. 解析 OFD.xml 获取文档根目录
	if err := p.parseOfdXml(); err != nil {
		return fmt.Errorf("解析OFD.xml失败: %w", err)
	}

	// 2. 解析 Document.xml 获取页面列表和资源
	if err := p.parseDocumentXml(); err != nil {
		return fmt.Errorf("解析Document.xml失败: %w", err)
	}

	// 3. 检查签章
	p.checkSignatures()

	return nil
}

// parseOfdXml 解析 OFD.xml
func (p *OfdParser) parseOfdXml() error {
	content, err := p.readZipFile("OFD.xml")
	if err != nil {
		return err
	}

	// 移除命名空间前缀以简化解析
	content = removeNamespacePrefix(content)

	// 提取 DocRoot
	docRootMatch := regexp.MustCompile(`<DocRoot>([^<]+)</DocRoot>`).FindSubmatch(content)
	if docRootMatch != nil {
		docRoot := string(docRootMatch[1])
		// 提取目录部分，如 "Doc_0/Document.xml" -> "Doc_0"
		p.docRoot = path.Dir(docRoot)
		if p.docRoot == "." {
			p.docRoot = ""
		}
	}

	// 提取文档标题
	titleMatch := regexp.MustCompile(`<Title>([^<]*)</Title>`).FindSubmatch(content)
	if titleMatch != nil {
		p.docTitle = string(titleMatch[1])
	}

	return nil
}

// parseDocumentXml 解析 Document.xml
func (p *OfdParser) parseDocumentXml() error {
	docPath := path.Join(p.docRoot, "Document.xml")
	content, err := p.readZipFile(docPath)
	if err != nil {
		return err
	}

	content = removeNamespacePrefix(content)

	// 提取页面尺寸
	physicalBoxMatch := regexp.MustCompile(`<PhysicalBox>([^<]+)</PhysicalBox>`).FindSubmatch(content)
	if physicalBoxMatch != nil {
		p.parsePhysicalBox(string(physicalBoxMatch[1]))
	}

	// 提取公共资源路径并解析
	publicResMatch := regexp.MustCompile(`<PublicRes>([^<]+)</PublicRes>`).FindSubmatch(content)
	if publicResMatch != nil {
		resPath := path.Join(p.docRoot, string(publicResMatch[1]))
		p.parsePublicRes(resPath)
	}

	// 提取页面列表
	pagePattern := regexp.MustCompile(`<Page\s+ID="([^"]+)"\s+BaseLoc="([^"]+)"`)
	pageMatches := pagePattern.FindAllSubmatch(content, -1)

	for _, match := range pageMatches {
		if len(match) >= 3 {
			pageID := string(match[1])
			baseLoc := string(match[2])

			// 解析页面内容
			pagePath := path.Join(p.docRoot, baseLoc)
			page, err := p.parsePage(pageID, pagePath)
			if err != nil {
				continue
			}
			p.pages = append(p.pages, page)
		}
	}

	// 按页面ID排序
	sort.Slice(p.pages, func(i, j int) bool {
		idI, _ := strconv.Atoi(p.pages[i].PageID)
		idJ, _ := strconv.Atoi(p.pages[j].PageID)
		return idI < idJ
	})

	return nil
}

// parsePhysicalBox 解析页面尺寸
func (p *OfdParser) parsePhysicalBox(box string) {
	parts := strings.Fields(box)
	if len(parts) >= 4 {
		p.pageWidth, _ = strconv.ParseFloat(parts[2], 64)
		p.pageHeight, _ = strconv.ParseFloat(parts[3], 64)
	}
}

// parsePublicRes 解析公共资源
func (p *OfdParser) parsePublicRes(resPath string) {
	content, err := p.readZipFile(resPath)
	if err != nil {
		return
	}

	content = removeNamespacePrefix(content)

	// 提取字体信息
	fontPattern := regexp.MustCompile(`<Font\s+ID="([^"]+)"\s+FontName="([^"]+)"`)
	fontMatches := fontPattern.FindAllSubmatch(content, -1)

	for _, match := range fontMatches {
		if len(match) >= 3 {
			fontID := string(match[1])
			fontName := string(match[2])
			p.fonts[fontID] = fontName
		}
	}
}

// parsePage 解析页面内容
func (p *OfdParser) parsePage(pageID, pagePath string) (*OfdParsedPage, error) {
	content, err := p.readZipFile(pagePath)
	if err != nil {
		return nil, err
	}

	content = removeNamespacePrefix(content)

	page := &OfdParsedPage{
		PageID:  pageID,
		Colors:  make([]string, 0),
	}

	// 提取所有 TextCode 中的文本
	var texts []string
	textCodePattern := regexp.MustCompile(`<TextCode[^>]*>([^<]*)</TextCode>`)
	textMatches := textCodePattern.FindAllSubmatch(content, -1)

	for _, match := range textMatches {
		if len(match) >= 2 {
			text := strings.TrimSpace(string(match[1]))
			if text != "" {
				texts = append(texts, text)
			}
		}
	}

	page.TextContent = strings.Join(texts, "")

	// 提取颜色信息
	colorPattern := regexp.MustCompile(`<FillColor\s+Value="([^"]+)"`)
	colorMatches := colorPattern.FindAllSubmatch(content, -1)

	colorSet := make(map[string]bool)
	for _, match := range colorMatches {
		if len(match) >= 2 {
			color := string(match[1])
			if !colorSet[color] {
				colorSet[color] = true
				page.Colors = append(page.Colors, color)

				if isOfdRedColor(color) {
					page.HasRedColor = true
					p.hasRedColor = true
				}
			}
		}
	}

	return page, nil
}

// checkSignatures 检查签章
func (p *OfdParser) checkSignatures() {
	for _, file := range p.zipReader.File {
		nameLower := strings.ToLower(file.Name)
		if strings.Contains(nameLower, "sign") ||
			strings.Contains(nameLower, "seal") ||
			strings.Contains(nameLower, "stamp") {
			p.hasSignature = true
			return
		}
	}
}

// ============================================================
// 公开方法
// ============================================================

// GetAllText 获取所有文本
func (p *OfdParser) GetAllText() string {
	var allTexts []string

	for _, page := range p.pages {
		if page.TextContent != "" {
			allTexts = append(allTexts, page.TextContent)
		}
	}

	return strings.Join(allTexts, "\n")
}

// GetPageCount 获取页面数量
func (p *OfdParser) GetPageCount() int {
	return len(p.pages)
}

// GetFonts 获取字体列表
func (p *OfdParser) GetFonts() []*OfdFont {
	var fonts []*OfdFont
	for id, name := range p.fonts {
		fonts = append(fonts, &OfdFont{
			ID:       id,
			FontName: name,
		})
	}
	return fonts
}

// HasSignature 是否有签章
func (p *OfdParser) HasSignature() bool {
	return p.hasSignature
}

// GetPageSize 获取页面尺寸（毫米）
func (p *OfdParser) GetPageSize() (width, height float64) {
	return p.pageWidth, p.pageHeight
}

// GetDocType 获取文档类型
func (p *OfdParser) GetDocType() string {
	return "OFD"
}

// GetVersion 获取OFD版本
func (p *OfdParser) GetVersion() string {
	return "1.1"
}

// GetDocTitle 获取文档标题
func (p *OfdParser) GetDocTitle() string {
	return p.docTitle
}

// ============================================================
// 颜色检测
// ============================================================

// OfdColorInfo 颜色信息
type OfdColorInfo struct {
	HasRedColor   bool
	RedColorCount int
	Colors        []string
}

// DetectColors 检测文档中的颜色
func (p *OfdParser) DetectColors() *OfdColorInfo {
	info := &OfdColorInfo{
		Colors: make([]string, 0),
	}

	colorSet := make(map[string]bool)

	for _, page := range p.pages {
		for _, color := range page.Colors {
			if !colorSet[color] {
				colorSet[color] = true
				info.Colors = append(info.Colors, color)

				if isOfdRedColor(color) {
					info.HasRedColor = true
					info.RedColorCount++
				}
			}
		}

		if page.HasRedColor {
			info.HasRedColor = true
		}
	}

	return info
}

// isOfdRedColor 检查是否是红色
func isOfdRedColor(color string) bool {
	color = strings.TrimSpace(color)

	// 处理 "R G B" 格式（0-255）
	parts := strings.Fields(color)
	if len(parts) == 3 {
		r, err1 := strconv.ParseFloat(parts[0], 64)
		g, err2 := strconv.ParseFloat(parts[1], 64)
		b, err3 := strconv.ParseFloat(parts[2], 64)

		if err1 == nil && err2 == nil && err3 == nil {
			// 红色判断：R高，G和B低
			if r >= 200 && g <= 80 && b <= 80 {
				return true
			}
		}
	}

	// 处理 "#RRGGBB" 格式
	if strings.HasPrefix(color, "#") && len(color) == 7 {
		r, err1 := strconv.ParseInt(color[1:3], 16, 64)
		g, err2 := strconv.ParseInt(color[3:5], 16, 64)
		b, err3 := strconv.ParseInt(color[5:7], 16, 64)

		if err1 == nil && err2 == nil && err3 == nil {
			if r >= 200 && g <= 80 && b <= 80 {
				return true
			}
		}
	}

	return false
}

// ============================================================
// 辅助方法
// ============================================================

// readZipFile 读取ZIP中的文件
func (p *OfdParser) readZipFile(name string) ([]byte, error) {
	// 标准化路径
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimPrefix(name, "./")
	name = strings.ReplaceAll(name, "\\", "/")

	for _, file := range p.zipReader.File {
		fileName := strings.TrimPrefix(file.Name, "/")
		fileName = strings.TrimPrefix(fileName, "./")
		fileName = strings.ReplaceAll(fileName, "\\", "/")

		if fileName == name || strings.EqualFold(fileName, name) {
			rc, err := file.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("文件不存在: %s", name)
}

// removeNamespacePrefix 移除XML命名空间前缀
func removeNamespacePrefix(data []byte) []byte {
	// 移除 ofd: 前缀
	data = bytes.ReplaceAll(data, []byte("ofd:"), []byte(""))
	// 移除 xmlns:ofd 声明
	data = bytes.ReplaceAll(data, []byte(`xmlns:ofd="http://www.ofdspec.org/2016"`), []byte(""))
	return data
}

// ListFiles 列出所有文件（调试用）
func (p *OfdParser) ListFiles() []string {
	var files []string
	for _, file := range p.zipReader.File {
		files = append(files, file.Name)
	}
	return files
}

// ============================================================
// OFD 数据结构（简化版，用于兼容现有代码）
// ============================================================

// OfdFont 字体定义
type OfdFont struct {
	ID         string
	FontName   string
	FamilyName string
	Charset    string
	FontFile   string
}