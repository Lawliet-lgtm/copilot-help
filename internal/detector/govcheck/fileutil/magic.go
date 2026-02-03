package fileutil

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Category 文件分类
type Category string

// 文件分类常量
const (
	CategoryText     Category = "text"
	CategoryDocument Category = "document"
	CategoryPDF      Category = "pdf"
	CategoryOFD      Category = "ofd"
	CategoryImage    Category = "image"
	CategoryArchive  Category = "archive"
	CategoryOther    Category = "other"
)

// DetectionMethod 检测方法
type DetectionMethod string

const (
	MethodMagic     DetectionMethod = "magic"     // 通过魔数检测
	MethodContent   DetectionMethod = "content"   // 通过内容特征检测
	MethodExtension DetectionMethod = "extension" // 通过扩展名检测（不可靠）
	MethodUnknown   DetectionMethod = "unknown"   // 未知
)

// FileType 表示文件类型
type FileType struct {
	Extension   string          // 文件扩展名 (不含点)
	MimeType    string          // MIME类型
	Description string          // 描述
	Category    Category        // 分类: text, document, pdf, ofd, image, archive, other
	Method      DetectionMethod // 检测方法
	Reliable    bool            // 检测结果是否可靠
}

// 常见文件类型定义
var (
	// 文本类
	TypeTXT   = FileType{"txt", "text/plain", "纯文本文件", CategoryText, MethodUnknown, false}
	TypeTEXT  = FileType{"text", "text/plain", "纯文本文件", CategoryText, MethodUnknown, false}
	TypeHTML  = FileType{"html", "text/html", "HTML文档", CategoryText, MethodUnknown, false}
	TypeHTM   = FileType{"htm", "text/html", "HTML文档", CategoryText, MethodUnknown, false}
	TypeXML   = FileType{"xml", "application/xml", "XML文档", CategoryText, MethodUnknown, false}
	TypeRTF   = FileType{"rtf", "application/rtf", "富文本格式", CategoryText, MethodUnknown, false}
	TypeMHT   = FileType{"mht", "message/rfc822", "MHTML网页存档", CategoryText, MethodUnknown, false}
	TypeMHTML = FileType{"mhtml", "message/rfc822", "MHTML网页存档", CategoryText, MethodUnknown, false}
	TypeEML   = FileType{"eml", "message/rfc822", "电子邮件", CategoryText, MethodUnknown, false}

	// 文档类
	TypeDOC  = FileType{"doc", "application/msword", "Microsoft Word 文档 (旧版)", CategoryDocument, MethodUnknown, false}
	TypeDOCX = FileType{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "Microsoft Word 文档", CategoryDocument, MethodUnknown, false}
	TypeDOCM = FileType{"docm", "application/vnd.ms-word.document.macroEnabled.12", "Microsoft Word 宏文档", CategoryDocument, MethodUnknown, false}
	TypeDOTX = FileType{"dotx", "application/vnd.openxmlformats-officedocument.wordprocessingml.template", "Microsoft Word 模板", CategoryDocument, MethodUnknown, false}
	TypeDOTM = FileType{"dotm", "application/vnd.ms-word.template.macroEnabled.12", "Microsoft Word 宏模板", CategoryDocument, MethodUnknown, false}
	TypeWPS  = FileType{"wps", "application/vnd.ms-works", "WPS文字文档", CategoryDocument, MethodUnknown, false}
	TypeWPT  = FileType{"wpt", "application/vnd.ms-works", "WPS文字模板", CategoryDocument, MethodUnknown, false}

	// PDF类
	TypePDF = FileType{"pdf", "application/pdf", "PDF文档", CategoryPDF, MethodUnknown, false}

	// OFD类
	TypeOFD = FileType{"ofd", "application/ofd", "OFD版式文档", CategoryOFD, MethodUnknown, false}

	// 图片类
	TypeJPG  = FileType{"jpg", "image/jpeg", "JPEG图片", CategoryImage, MethodUnknown, false}
	TypeJPEG = FileType{"jpeg", "image/jpeg", "JPEG图片", CategoryImage, MethodUnknown, false}
	TypePNG  = FileType{"png", "image/png", "PNG图片", CategoryImage, MethodUnknown, false}
	TypeGIF  = FileType{"gif", "image/gif", "GIF图片", CategoryImage, MethodUnknown, false}
	TypeBMP  = FileType{"bmp", "image/bmp", "BMP图片", CategoryImage, MethodUnknown, false}
	TypeTIFF = FileType{"tiff", "image/tiff", "TIFF图片", CategoryImage, MethodUnknown, false}
	TypeTIF  = FileType{"tif", "image/tiff", "TIFF图片", CategoryImage, MethodUnknown, false}
	TypeWEBP = FileType{"webp", "image/webp", "WebP图片", CategoryImage, MethodUnknown, false}

	// 压缩文件类
	TypeZIP = FileType{"zip", "application/zip", "ZIP压缩文件", CategoryArchive, MethodUnknown, false}
	TypeRAR = FileType{"rar", "application/x-rar-compressed", "RAR压缩文件", CategoryArchive, MethodUnknown, false}
	Type7Z  = FileType{"7z", "application/x-7z-compressed", "7-Zip压缩文件", CategoryArchive, MethodUnknown, false}
	TypeGZ  = FileType{"gz", "application/gzip", "Gzip压缩文件", CategoryArchive, MethodUnknown, false}
	TypeTAR = FileType{"tar", "application/x-tar", "Tar归档文件", CategoryArchive, MethodUnknown, false}

	// 未知类型
	TypeUnknown = FileType{"", "application/octet-stream", "未知文件类型", CategoryOther, MethodUnknown, false}
)

// 文件魔数签名
var magicSignatures = []struct {
	Magic    []byte
	Offset   int
	FileType FileType
}{
	// PDF
	{[]byte("%PDF"), 0, FileType{"pdf", "application/pdf", "PDF文档", CategoryPDF, MethodMagic, true}},

	// ZIP格式 (包括DOCX, OFD等)
	{[]byte{0x50, 0x4B, 0x03, 0x04}, 0, FileType{"zip", "application/zip", "ZIP压缩文件", CategoryArchive, MethodMagic, true}},
	{[]byte{0x50, 0x4B, 0x05, 0x06}, 0, FileType{"zip", "application/zip", "ZIP压缩文件(空)", CategoryArchive, MethodMagic, true}},
	{[]byte{0x50, 0x4B, 0x07, 0x08}, 0, FileType{"zip", "application/zip", "ZIP压缩文件(分卷)", CategoryArchive, MethodMagic, true}},

	// RAR
	{[]byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00}, 0, FileType{"rar", "application/x-rar-compressed", "RAR压缩文件", CategoryArchive, MethodMagic, true}},
	{[]byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01, 0x00}, 0, FileType{"rar", "application/x-rar-compressed", "RAR5压缩文件", CategoryArchive, MethodMagic, true}},

	// 7Z
	{[]byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}, 0, FileType{"7z", "application/x-7z-compressed", "7-Zip压缩文件", CategoryArchive, MethodMagic, true}},

	// GZIP
	{[]byte{0x1F, 0x8B}, 0, FileType{"gz", "application/gzip", "Gzip压缩文件", CategoryArchive, MethodMagic, true}},

	// 图片
	{[]byte{0xFF, 0xD8, 0xFF}, 0, FileType{"jpg", "image/jpeg", "JPEG图片", CategoryImage, MethodMagic, true}},
	{[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0, FileType{"png", "image/png", "PNG图片", CategoryImage, MethodMagic, true}},
	{[]byte("GIF87a"), 0, FileType{"gif", "image/gif", "GIF图片", CategoryImage, MethodMagic, true}},
	{[]byte("GIF89a"), 0, FileType{"gif", "image/gif", "GIF图片", CategoryImage, MethodMagic, true}},
	{[]byte{0x42, 0x4D}, 0, FileType{"bmp", "image/bmp", "BMP图片", CategoryImage, MethodMagic, true}},
	{[]byte{0x49, 0x49, 0x2A, 0x00}, 0, FileType{"tiff", "image/tiff", "TIFF图片(LE)", CategoryImage, MethodMagic, true}},
	{[]byte{0x4D, 0x4D, 0x00, 0x2A}, 0, FileType{"tiff", "image/tiff", "TIFF图片(BE)", CategoryImage, MethodMagic, true}},
	{[]byte("RIFF"), 0, FileType{"webp", "image/webp", "WebP图片", CategoryImage, MethodMagic, true}}, // 需要进一步检查 WEBP

	// RTF
	{[]byte("{\\rtf"), 0, FileType{"rtf", "application/rtf", "富文本格式", CategoryText, MethodMagic, true}},

	// 旧版DOC/WPS (OLE2 复合文档格式)
	{[]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, 0, FileType{"doc", "application/msword", "OLE2复合文档", CategoryDocument, MethodMagic, true}},
}

// 扩展名到文件类型的映射（仅作为最后备选）
var extensionMap = map[string]FileType{
	// 文本类
	"txt":   TypeTXT,
	"text":  TypeTEXT,
	"html":  TypeHTML,
	"htm":   TypeHTM,
	"xml":   TypeXML,
	"rtf":   TypeRTF,
	"mht":   TypeMHT,
	"mhtml": TypeMHTML,
	"eml":   TypeEML,

	// 文档类
	"doc":  TypeDOC,
	"docx": TypeDOCX,
	"docm": TypeDOCM,
	"dotx": TypeDOTX,
	"dotm": TypeDOTM,
	"wps":  TypeWPS,
	"wpt":  TypeWPT,

	// PDF类
	"pdf": TypePDF,

	// OFD类
	"ofd": TypeOFD,

	// 图片类
	"jpg":  TypeJPG,
	"jpeg": TypeJPEG,
	"png":  TypePNG,
	"gif":  TypeGIF,
	"bmp":  TypeBMP,
	"tiff": TypeTIFF,
	"tif":  TypeTIF,
	"webp": TypeWEBP,

	// 压缩类
	"zip": TypeZIP,
	"rar": TypeRAR,
	"7z":  Type7Z,
	"gz":  TypeGZ,
	"tar": TypeTAR,
}

// DetectFileType 检测文件类型（增强版，优先使用内容检测）
func DetectFileType(filePath string) (FileType, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return TypeUnknown, err
	}
	defer file.Close()

	// 读取文件头（用于魔数和内容检测）
	header := make([]byte, 4096)
	n, err := file.Read(header)
	if err != nil && n == 0 {
		return TypeUnknown, err
	}
	header = header[:n]

	// 步骤 1: 魔数检测（最可靠）
	detectedType := detectByMagic(header)

	// 如果检测到 ZIP 格式，进一步判断是 DOCX、OFD 还是普通 ZIP
	if detectedType.Extension == "zip" {
		specificType := detectZipSubtype(filePath)
		if specificType.Extension != "" {
			specificType.Method = MethodMagic
			specificType.Reliable = true
			return specificType, nil
		}
		// 保持为 ZIP
		return detectedType, nil
	}

	// 如果检测到 OLE2 格式，进一步判断是 DOC 还是 WPS
	if detectedType.Extension == "doc" && detectedType.Method == MethodMagic {
		specificType := detectOLE2Subtype(filePath, header)
		return specificType, nil
	}

	// 如果检测到 WEBP，需要验证
	if detectedType.Extension == "webp" {
		if !isValidWebP(header) {
			detectedType = TypeUnknown
		}
	}

	// 魔数检测成功，直接返回
	if detectedType.Extension != "" && detectedType.Reliable {
		return detectedType, nil
	}

	// 步骤 2: 内容特征检测（针对文本类文件）
	contentType := detectByContent(header)
	if contentType.Extension != "" {
		return contentType, nil
	}

	// 步骤 3: 扩展名检测（最后备选，标记为不可靠）
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if fileType, ok := extensionMap[ext]; ok {
		fileType.Method = MethodExtension
		fileType.Reliable = false
		return fileType, nil
	}

	return TypeUnknown, nil
}

// detectByMagic 通过魔数检测文件类型
func detectByMagic(header []byte) FileType {
	for _, sig := range magicSignatures {
		if len(header) >= sig.Offset+len(sig.Magic) {
			if bytes.Equal(header[sig.Offset:sig.Offset+len(sig.Magic)], sig.Magic) {
				return sig.FileType
			}
		}
	}
	return TypeUnknown
}

// detectByContent 通过内容特征检测文件类型
func detectByContent(content []byte) FileType {
	if len(content) == 0 {
		return TypeUnknown
	}

	// 检查是否是有效的文本内容
	if !isLikelyText(content) {
		return TypeUnknown
	}

	text := string(content)
	textLower := strings.ToLower(text)

	// 检测 XML（包括 XHTML）
	if isXML(text) {
		// 进一步检查是否是 XHTML
		if strings.Contains(textLower, "<!doctype html") ||
			strings.Contains(textLower, "<html") ||
			strings.Contains(textLower, "xmlns=\"http://www.w3.org/1999/xhtml\"") {
			return FileType{"html", "text/html", "XHTML文档", CategoryText, MethodContent, true}
		}
		return FileType{"xml", "application/xml", "XML文档", CategoryText, MethodContent, true}
	}

	// 检测 HTML
	if isHTML(textLower) {
		return FileType{"html", "text/html", "HTML文档", CategoryText, MethodContent, true}
	}

	// 检测 MHT/MHTML
	if isMHTML(text) {
		return FileType{"mht", "message/rfc822", "MHTML网页存档", CategoryText, MethodContent, true}
	}

	// 检测 EML
	if isEML(text) {
		return FileType{"eml", "message/rfc822", "电子邮件", CategoryText, MethodContent, true}
	}

	// 如果是有效的 UTF-8 文本，归类为纯文本
	if utf8.Valid(content) && isPrintableText(content) {
		return FileType{"txt", "text/plain", "纯文本文件", CategoryText, MethodContent, true}
	}

	return TypeUnknown
}

// isLikelyText 检查内容是否像文本
func isLikelyText(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// 检查 BOM
	if hasBOM(content) {
		return true
	}

	// 统计可打印字符和控制字符
	printable := 0
	control := 0
	total := len(content)

	// 只检查前 1024 字节
	checkLen := total
	if checkLen > 1024 {
		checkLen = 1024
	}

	for i := 0; i < checkLen; i++ {
		b := content[i]
		if b == 0 {
			// NULL 字节通常表示二进制文件
			control++
		} else if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			control++
		} else {
			printable++
		}
	}

	// 如果控制字符超过 10%，可能不是文本
	if float64(control)/float64(checkLen) > 0.1 {
		return false
	}

	return true
}

// hasBOM 检查是否有 BOM 标记
func hasBOM(content []byte) bool {
	if len(content) >= 3 {
		// UTF-8 BOM
		if content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
			return true
		}
	}
	if len(content) >= 2 {
		// UTF-16 LE BOM
		if content[0] == 0xFF && content[1] == 0xFE {
			return true
		}
		// UTF-16 BE BOM
		if content[0] == 0xFE && content[1] == 0xFF {
			return true
		}
	}
	return false
}

// isPrintableText 检查是否是可打印文本
func isPrintableText(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// 检查前 512 字节
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}

	for i := 0; i < checkLen; i++ {
		b := content[i]
		// 允许常见的空白字符和可打印 ASCII
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}

	return true
}

// isXML 检查是否是 XML 文档
func isXML(text string) bool {
	trimmed := strings.TrimSpace(text)

	// XML 声明
	if strings.HasPrefix(trimmed, "<?xml") {
		return true
	}

	// 检查是否以 < 开头且包含 XML 结构特征
	if strings.HasPrefix(trimmed, "<") {
		// 查找第一个标签
		if idx := strings.Index(trimmed, ">"); idx > 0 {
			tag := trimmed[1:idx]
			// 检查是否像 XML 标签（排除 HTML DOCTYPE）
			if !strings.HasPrefix(strings.ToLower(tag), "!doctype html") &&
				!strings.HasPrefix(strings.ToLower(tag), "html") {
				// 检查是否有命名空间或属性
				if strings.Contains(tag, "xmlns") || strings.Contains(tag, ":") {
					return true
				}
			}
		}
	}

	return false
}

// isHTML 检查是否是 HTML 文档
func isHTML(textLower string) bool {
	trimmed := strings.TrimSpace(textLower)

	// DOCTYPE 声明
	if strings.HasPrefix(trimmed, "<!doctype html") {
		return true
	}

	// HTML 标签
	htmlIndicators := []string{
		"<html",
		"<head",
		"<body",
		"<title",
		"<meta",
		"<link",
		"<script",
		"<style",
		"<div",
		"<span",
		"<table",
		"<form",
	}

	matchCount := 0
	for _, indicator := range htmlIndicators {
		if strings.Contains(trimmed, indicator) {
			matchCount++
			if matchCount >= 2 {
				return true
			}
		}
	}

	// 单独的 <html> 标签也算
	if strings.Contains(trimmed, "<html") && strings.Contains(trimmed, "</html>") {
		return true
	}

	return false
}

// isMHTML 检查是否是 MHTML 文档
func isMHTML(text string) bool {
	textLower := strings.ToLower(text)

	// MHTML 特征
	if strings.Contains(textLower, "mime-version:") &&
		strings.Contains(textLower, "content-type:") &&
		(strings.Contains(textLower, "multipart/related") ||
			strings.Contains(textLower, "text/html")) {
		return true
	}

	// 检查 MHTML 边界标记
	if strings.Contains(text, "------=_") || strings.Contains(text, "Content-Location:") {
		return true
	}

	return false
}

// isEML 检查是否是电子邮件
func isEML(text string) bool {
	textLower := strings.ToLower(text)

	// 邮件头特征
	emailHeaders := []string{
		"from:",
		"to:",
		"subject:",
		"date:",
		"message-id:",
	}

	matchCount := 0
	for _, header := range emailHeaders {
		if strings.Contains(textLower, header) {
			matchCount++
		}
	}

	// 至少匹配 3 个邮件头
	return matchCount >= 3
}

// isValidWebP 验证是否是有效的 WebP 文件
func isValidWebP(header []byte) bool {
	if len(header) < 12 {
		return false
	}

	// RIFF....WEBP
	if bytes.Equal(header[0:4], []byte("RIFF")) &&
		bytes.Equal(header[8:12], []byte("WEBP")) {
		return true
	}

	return false
}

// detectZipSubtype 检测 ZIP 子类型（DOCX, OFD 等）
func detectZipSubtype(filePath string) FileType {
	file, err := os.Open(filePath)
	if err != nil {
		return TypeUnknown
	}
	defer file.Close()

	// 读取更多内容来判断 ZIP 内部结构
	content := make([]byte, 8192)
	n, _ := file.Read(content)
	content = content[:n]

	contentStr := string(content)

	// 检测 OFD 特征
	if strings.Contains(contentStr, "OFD.xml") ||
		strings.Contains(contentStr, "ofd.xml") ||
		(strings.Contains(contentStr, "Document.xml") && strings.Contains(contentStr, "Doc_")) {
		return FileType{"ofd", "application/ofd", "OFD版式文档", CategoryOFD, MethodMagic, true}
	}

	// 检测 DOCX 特征
	if strings.Contains(contentStr, "[Content_Types].xml") ||
		strings.Contains(contentStr, "word/document.xml") ||
		strings.Contains(contentStr, "word/") {
		// 进一步区分 docx/docm/dotx/dotm
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
		switch ext {
		case "docm":
			return FileType{"docm", "application/vnd.ms-word.document.macroEnabled.12", "Microsoft Word 宏文档", CategoryDocument, MethodMagic, true}
		case "dotx":
			return FileType{"dotx", "application/vnd.openxmlformats-officedocument.wordprocessingml.template", "Microsoft Word 模板", CategoryDocument, MethodMagic, true}
		case "dotm":
			return FileType{"dotm", "application/vnd.ms-word.template.macroEnabled.12", "Microsoft Word 宏模板", CategoryDocument, MethodMagic, true}
		default:
			return FileType{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "Microsoft Word 文档", CategoryDocument, MethodMagic, true}
		}
	}

	// 检测 WPS (新版基于 ZIP)
	if strings.Contains(contentStr, "customXml") && strings.Contains(contentStr, "wps") {
		return FileType{"wps", "application/vnd.ms-works", "WPS文字文档", CategoryDocument, MethodMagic, true}
	}

	return TypeUnknown
}

// detectOLE2Subtype 检测 OLE2 子类型（DOC, WPS 等）
func detectOLE2Subtype(filePath string, header []byte) FileType {
	// 读取更多内容来分析
	file, err := os.Open(filePath)
	if err != nil {
		return FileType{"doc", "application/msword", "OLE2文档", CategoryDocument, MethodMagic, true}
	}
	defer file.Close()

	content := make([]byte, 8192)
	n, _ := file.Read(content)
	content = content[:n]

	// 检查 WPS 特征
	// WPS 文档中通常包含 "WPS" 或 "Kingsoft" 字符串
	contentStr := string(content)
	if strings.Contains(contentStr, "Kingsoft") ||
		strings.Contains(contentStr, "WPS Office") ||
		strings.Contains(contentStr, "\x00W\x00P\x00S") { // UTF-16 LE "WPS"
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
		if ext == "wpt" {
			return FileType{"wpt", "application/vnd.ms-works", "WPS文字模板", CategoryDocument, MethodMagic, true}
		}
		return FileType{"wps", "application/vnd.ms-works", "WPS文字文档", CategoryDocument, MethodMagic, true}
	}

	// 检查 Word 文档特征
	if strings.Contains(contentStr, "Microsoft Word") ||
		strings.Contains(contentStr, "MSWordDoc") ||
		strings.Contains(contentStr, "Word.Document") {
		return FileType{"doc", "application/msword", "Microsoft Word 文档", CategoryDocument, MethodMagic, true}
	}

	// 默认为 DOC
	return FileType{"doc", "application/msword", "OLE2文档", CategoryDocument, MethodMagic, true}
}

// IsSupportedForDetection 检查文件类型是否支持公文检测
func IsSupportedForDetection(fileType FileType) bool {
	supportedCategories := map[Category]bool{
		CategoryText:     true,
		CategoryDocument: true,
		CategoryPDF:      true,
		CategoryOFD:      true,
		CategoryImage:    true, // 支持图片（需要OCR）
	}
	return supportedCategories[fileType.Category]
}

// GetUnsupportedReason 获取不支持的原因
func GetUnsupportedReason(fileType FileType) string {
	switch fileType.Category {
	case CategoryImage:
		return "图片文件需要OCR识别，请确保已安装Tesseract"
	case CategoryArchive:
		return "压缩文件，请先解压后检测"
	case CategoryOther:
		return "未知或不支持的文件格式"
	default:
		return "不支持的文件类型"
	}
}

// GetFileTypeByExtension 根据扩展名获取文件类型（标记为不可靠）
func GetFileTypeByExtension(ext string) FileType {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if fileType, ok := extensionMap[ext]; ok {
		fileType.Method = MethodExtension
		fileType.Reliable = false
		return fileType
	}
	return TypeUnknown
}

// IsTextFile 检查是否是文本文件
func IsTextFile(fileType FileType) bool {
	return fileType.Category == CategoryText
}

// IsDocumentFile 检查是否是文档文件
func IsDocumentFile(fileType FileType) bool {
	return fileType.Category == CategoryDocument
}

// IsPdfFile 检查是否是PDF文件
func IsPdfFile(fileType FileType) bool {
	return fileType.Category == CategoryPDF
}

// IsOfdFile 检查是否是OFD文件
func IsOfdFile(fileType FileType) bool {
	return fileType.Category == CategoryOFD
}

// IsImageFile 检查是否是图片文件
func IsImageFile(fileType FileType) bool {
	return fileType.Category == CategoryImage
}

// IsArchiveFile 检查是否是压缩文件
func IsArchiveFile(fileType FileType) bool {
	return fileType.Category == CategoryArchive
}

// IsReliableDetection 检查检测结果是否可靠
func IsReliableDetection(fileType FileType) bool {
	return fileType.Reliable
}

// GetDetectionMethod 获取检测方法
func GetDetectionMethod(fileType FileType) DetectionMethod {
	return fileType.Method
}

// GetSupportedCategories 获取支持公文检测的分类列表
func GetSupportedCategories() []Category {
	return []Category{
		CategoryText,
		CategoryDocument,
		CategoryPDF,
		CategoryOFD,
	}
}

// GetAllCategories 获取所有分类列表
func GetAllCategories() []Category {
	return []Category{
		CategoryText,
		CategoryDocument,
		CategoryPDF,
		CategoryOFD,
		CategoryImage,
		CategoryArchive,
		CategoryOther,
	}
}

// DetectFileTypeStrict 严格模式检测文件类型（只接受可靠的检测结果）
func DetectFileTypeStrict(filePath string) (FileType, error) {
	fileType, err := DetectFileType(filePath)
	if err != nil {
		return TypeUnknown, err
	}

	if !fileType.Reliable {
		return TypeUnknown, nil
	}

	return fileType, nil
}

// ValidateFileType 验证文件类型是否与扩展名匹配
func ValidateFileType(filePath string) (matched bool, detected FileType, expected FileType) {
	// 检测实际类型
	detected, _ = DetectFileType(filePath)

	// 获取扩展名期望的类型
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	expected = GetFileTypeByExtension(ext)

	// 比较
	if detected.Extension == "" {
		// 无法检测，不确定是否匹配
		return true, detected, expected
	}

	if expected.Extension == "" {
		// 未知扩展名
		return true, detected, expected
	}

	// 检查是否匹配
	matched = detected.Extension == expected.Extension ||
		detected.Category == expected.Category

	return matched, detected, expected
}