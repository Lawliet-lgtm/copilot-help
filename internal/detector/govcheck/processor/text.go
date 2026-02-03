package processor

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// TextProcessor 文本文件处理器
type TextProcessor struct {
	base   *BaseProcessor
	config *TextProcessorConfig
}

// TextProcessorConfig 文本处理器配置
type TextProcessorConfig struct {
	MaxFileSize    int64 // 最大文件大小 (字节)
	AutoDetectGBK  bool  // 自动检测GBK编码
	StripHTMLTags  bool  // 是否去除HTML标签
	NormalizeSpace bool  // 是否规范化空白字符
}

// DefaultTextProcessorConfig 返回默认配置
func DefaultTextProcessorConfig() *TextProcessorConfig {
	return &TextProcessorConfig{
		MaxFileSize:    50 * 1024 * 1024, // 50MB
		AutoDetectGBK:  true,
		StripHTMLTags:  true,
		NormalizeSpace: true,
	}
}

// NewTextProcessor 创建文本处理器
func NewTextProcessor() *TextProcessor {
	return NewTextProcessorWithConfig(nil)
}

// NewTextProcessorWithConfig 使用指定配置创建文本处理器
func NewTextProcessorWithConfig(config *TextProcessorConfig) *TextProcessor {
	if config == nil {
		config = DefaultTextProcessorConfig()
	}

	base := NewBaseProcessor(
		"TextProcessor",
		"文本文件处理器 (TXT/HTML/XML/RTF/MHT/EML)",
		[]string{"txt", "text", "html", "htm", "xml", "rtf", "mht", "mhtml", "eml"},
	)

	return &TextProcessor{
		base:   base,
		config: config,
	}
}

// Name 返回处理器名称
func (p *TextProcessor) Name() string {
	return p.base.Name()
}

// Description 返回处理器描述
func (p *TextProcessor) Description() string {
	return p.base.Description()
}

// SupportedTypes 返回支持的文件类型
func (p *TextProcessor) SupportedTypes() []string {
	return p.base.SupportedTypes()
}

// Process 处理文本文件
func (p *TextProcessor) Process(filePath string) (string, error) {
	// 检查文件大小
	info, err := os.Stat(filePath)
	if err != nil {
		return "", NewProcessorError(p.Name(), filePath, "获取文件信息", err)
	}

	if info.Size() == 0 {
		return "", NewProcessorError(p.Name(), filePath, "检查文件", fmt.Errorf("文件为空"))
	}

	if p.config.MaxFileSize > 0 && info.Size() > p.config.MaxFileSize {
		return "", NewProcessorError(p.Name(), filePath, "检查文件大小",
			fmt.Errorf("文件过大: %d 字节 (限制: %d 字节)", info.Size(), p.config.MaxFileSize))
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", NewProcessorError(p.Name(), filePath, "读取文件", err)
	}

	// 检测并转换编码
	text := p.decodeContent(content)

	// 根据文件类型进行处理
	ext := strings.ToLower(getFileExtension(filePath))
	switch ext {
	case "html", "htm", "mht", "mhtml":
		text = p.processHTML(text)
	case "xml":
		text = p.processXML(text)
	case "rtf":
		text = p.processRTF(text)
	case "eml":
		text = p.processEML(text)
	default:
		// TXT 等纯文本直接处理
		text = p.processPlainText(text)
	}

	// 规范化空白字符
	if p.config.NormalizeSpace {
		text = normalizeWhitespace(text)
	}

	return text, nil
}

// decodeContent 检测并解码文件内容
func (p *TextProcessor) decodeContent(content []byte) string {
	// 检查BOM
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		// UTF-8 BOM
		return string(content[3:])
	}
	if len(content) >= 2 {
		if content[0] == 0xFF && content[1] == 0xFE {
			// UTF-16 LE BOM
			return decodeUTF16LE(content[2:])
		}
		if content[0] == 0xFE && content[1] == 0xFF {
			// UTF-16 BE BOM
			return decodeUTF16BE(content[2:])
		}
	}

	// 尝试作为UTF-8解析
	text := string(content)
	if isValidUTF8(content) {
		return text
	}

	// 自动检测GBK编码
	if p.config.AutoDetectGBK && mightBeGBK(content) {
		if decoded, err := decodeGBK(content); err == nil {
			return decoded
		}
	}

	return text
}

// processHTML 处理HTML内容
func (p *TextProcessor) processHTML(content string) string {
	if !p.config.StripHTMLTags {
		return content
	}

	// 解析HTML并提取文本
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		// 解析失败，使用简单的标签移除
		return stripHTMLTagsSimple(content)
	}

	var textBuilder strings.Builder
	extractTextFromNode(doc, &textBuilder)
	return textBuilder.String()
}

// processXML 处理XML内容
func (p *TextProcessor) processXML(content string) string {
	// 移除XML标签，保留文本内容
	return stripXMLTags(content)
}

// processRTF 处理RTF内容
func (p *TextProcessor) processRTF(content string) string {
	// 简单的RTF文本提取
	return extractRTFText(content)
}

// processEML 处理EML邮件内容
func (p *TextProcessor) processEML(content string) string {
	// 提取邮件正文
	return extractEMLBody(content)
}

// processPlainText 处理纯文本
func (p *TextProcessor) processPlainText(content string) string {
	return content
}

// ============================================================
// 辅助函数
// ============================================================

// getFileExtension 获取文件扩展名
func getFileExtension(filePath string) string {
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '.' {
			return filePath[i+1:]
		}
		if filePath[i] == '/' || filePath[i] == '\\' {
			break
		}
	}
	return ""
}

// isValidUTF8 检查是否为有效的UTF-8编码
func isValidUTF8(data []byte) bool {
	i := 0
	for i < len(data) {
		if data[i] < 0x80 {
			i++
			continue
		}

		var size int
		if data[i]&0xE0 == 0xC0 {
			size = 2
		} else if data[i]&0xF0 == 0xE0 {
			size = 3
		} else if data[i]&0xF8 == 0xF0 {
			size = 4
		} else {
			return false
		}

		if i+size > len(data) {
			return false
		}

		for j := 1; j < size; j++ {
			if data[i+j]&0xC0 != 0x80 {
				return false
			}
		}
		i += size
	}
	return true
}

// mightBeGBK 检测是否可能是GBK编码
func mightBeGBK(data []byte) bool {
	hasHighByte := false
	for _, b := range data {
		if b >= 0x80 {
			hasHighByte = true
			break
		}
	}
	return hasHighByte && !isValidUTF8(data)
}

// decodeGBK 解码GBK编码
func decodeGBK(data []byte) (string, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// decodeUTF16LE 解码UTF-16 LE
func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}

	var result strings.Builder
	for i := 0; i < len(data); i += 2 {
		r := rune(data[i]) | rune(data[i+1])<<8
		result.WriteRune(r)
	}
	return result.String()
}

// decodeUTF16BE 解码UTF-16 BE
func decodeUTF16BE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}

	var result strings.Builder
	for i := 0; i < len(data); i += 2 {
		r := rune(data[i])<<8 | rune(data[i+1])
		result.WriteRune(r)
	}
	return result.String()
}

// extractTextFromNode 从HTML节点提取文本
func extractTextFromNode(n *html.Node, sb *strings.Builder) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			sb.WriteString(text)
			sb.WriteString(" ")
		}
	}

	// 跳过 script 和 style 标签
	if n.Type == html.ElementNode {
		if n.Data == "script" || n.Data == "style" {
			return
		}
		// 在块级元素后添加换行
		if isBlockElement(n.Data) {
			sb.WriteString("\n")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractTextFromNode(c, sb)
	}
}

// isBlockElement 检查是否为块级元素
func isBlockElement(tag string) bool {
	blockElements := map[string]bool{
		"div": true, "p": true, "br": true, "hr": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"ul": true, "ol": true, "li": true,
		"table": true, "tr": true, "td": true, "th": true,
		"header": true, "footer": true, "section": true, "article": true,
	}
	return blockElements[tag]
}

// stripHTMLTagsSimple 简单的HTML标签移除
func stripHTMLTagsSimple(content string) string {
	// 移除HTML标签
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(content, " ")

	// 解码HTML实体
	text = decodeHTMLEntities(text)

	return text
}

// stripXMLTags 移除XML标签
func stripXMLTags(content string) string {
	// 移除XML声明
	re := regexp.MustCompile(`<\?xml[^?]*\?>`)
	content = re.ReplaceAllString(content, "")

	// 移除CDATA
	re = regexp.MustCompile(`<!\[CDATA\[(.*?)\]\]>`)
	content = re.ReplaceAllString(content, "$1")

	// 移除标签
	re = regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(content, " ")

	return text
}

// decodeHTMLEntities 解码HTML实体
func decodeHTMLEntities(text string) string {
	// 使用 strings.NewReplacer 进行批量替���
	replacer := strings.NewReplacer(
		"&nbsp;", " ",
		"&lt;", "<",
		"&gt;", ">",
		"&amp;", "&",
		"&quot;", `"`,
		"&apos;", "'",
	)
	text = replacer.Replace(text)

	// 处理更多实体 - 使用 Unicode 码点
	text = strings.ReplaceAll(text, "&copy;", string(rune(0x00A9)))
	text = strings.ReplaceAll(text, "&reg;", string(rune(0x00AE)))
	text = strings.ReplaceAll(text, "&mdash;", string(rune(0x2014)))
	text = strings.ReplaceAll(text, "&ndash;", string(rune(0x2013)))
	text = strings.ReplaceAll(text, "&ldquo;", string(rune(0x201C)))
	text = strings.ReplaceAll(text, "&rdquo;", string(rune(0x201D)))
	text = strings.ReplaceAll(text, "&lsquo;", string(rune(0x2018)))
	text = strings.ReplaceAll(text, "&rsquo;", string(rune(0x2019)))
	text = strings.ReplaceAll(text, "&hellip;", string(rune(0x2026)))
	text = strings.ReplaceAll(text, "&bull;", string(rune(0x2022)))
	text = strings.ReplaceAll(text, "&trade;", string(rune(0x2122)))

	// 处理数字实体 &#xxx;
	re := regexp.MustCompile(`&#(\d+);`)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		var num int
		if _, err := fmt.Sscanf(match, "&#%d;", &num); err == nil {
			if num > 0 && num < 0x10FFFF {
				return string(rune(num))
			}
		}
		return match
	})

	// 处理十六进制数字实体 &#xXXXX;
	re = regexp.MustCompile(`&#[xX]([0-9a-fA-F]+);`)
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		var num int
		if _, err := fmt.Sscanf(match, "&#x%x;", &num); err == nil {
			if num > 0 && num < 0x10FFFF {
				return string(rune(num))
			}
		}
		if _, err := fmt.Sscanf(match, "&#X%x;", &num); err == nil {
			if num > 0 && num < 0x10FFFF {
				return string(rune(num))
			}
		}
		return match
	})

	return text
}

// extractRTFText 从RTF中提取文本
func extractRTFText(content string) string {
	var result strings.Builder
	inGroup := 0
	skipGroup := false
	i := 0

	for i < len(content) {
		ch := content[i]

		switch ch {
		case '{':
			inGroup++
			// 检查是否需要跳过的组
			if i+10 < len(content) {
				ahead := content[i : i+10]
				if strings.Contains(ahead, "\\fonttbl") ||
					strings.Contains(ahead, "\\colortbl") ||
					strings.Contains(ahead, "\\stylesheet") ||
					strings.Contains(ahead, "\\pict") {
					skipGroup = true
				}
			}
			i++

		case '}':
			inGroup--
			if inGroup <= 1 {
				skipGroup = false
			}
			i++

		case '\\':
			if skipGroup {
				i++
				continue
			}

			i++
			if i < len(content) {
				if content[i] == '\'' {
					// 十六进制字符 \'xx
					i += 3
				} else if content[i] == '\n' || content[i] == '\r' {
					i++
				} else {
					// 普通控制字
					for i < len(content) && ((content[i] >= 'a' && content[i] <= 'z') ||
						(content[i] >= 'A' && content[i] <= 'Z') ||
						(content[i] >= '0' && content[i] <= '9') ||
						content[i] == '-') {
						i++
					}
					if i < len(content) && content[i] == ' ' {
						i++
					}
				}
			}

		case '\n', '\r':
			i++

		default:
			if !skipGroup && inGroup >= 1 {
				result.WriteByte(ch)
			}
			i++
		}
	}

	return result.String()
}

// extractEMLBody 从EML邮件中提取正文
func extractEMLBody(content string) string {
	// 查找空行后的内容作为正文
	parts := strings.SplitN(content, "\r\n\r\n", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(content, "\n\n", 2)
	}

	if len(parts) == 2 {
		body := parts[1]

		// 检查是否为HTML格式
		headerLower := strings.ToLower(parts[0])
		if strings.Contains(headerLower, "content-type: text/html") ||
			strings.Contains(body, "<html") {
			return stripHTMLTagsSimple(body)
		}

		// 处理quoted-printable编码
		if strings.Contains(headerLower, "quoted-printable") {
			body = decodeQuotedPrintable(body)
		}

		return body
	}

	return content
}

// decodeQuotedPrintable 解码Quoted-Printable
func decodeQuotedPrintable(text string) string {
	var result strings.Builder

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		// 移除软换行
		line = strings.TrimSuffix(line, "=\r")
		line = strings.TrimSuffix(line, "=")

		i := 0
		for i < len(line) {
			if line[i] == '=' && i+2 < len(line) {
				hex := line[i+1 : i+3]
				var val int
				if _, err := fmt.Sscanf(hex, "%X", &val); err == nil {
					result.WriteByte(byte(val))
					i += 3
					continue
				}
			}
			result.WriteByte(line[i])
			i++
		}
		result.WriteString("\n")
	}

	return result.String()
}

// normalizeWhitespace 规范化空白字符
func normalizeWhitespace(text string) string {
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