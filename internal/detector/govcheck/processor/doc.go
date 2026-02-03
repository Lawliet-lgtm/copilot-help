package processor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"

	"linuxFileWatcher/internal/detector/govcheck/extractor"
)

// DocProcessor DOC 文档处理器
type DocProcessor struct {
	base   *BaseProcessor
	config *DocProcessorConfig
}

// DocProcessorConfig DOC 处理器配置
type DocProcessorConfig struct {
	MaxFileSize     int64  // 最大文件大小（字节）
	UseAntiword     bool   // 是否使用 antiword
	AntiwordPath    string // antiword 可执行文件路径
	UseLibreOffice  bool   // 是否使用 LibreOffice
	LibreOfficePath string // LibreOffice 可执行文件路径
	FallbackToBasic bool   // 是否回退到基础提取
}

// DefaultDocProcessorConfig 返回默认配置
func DefaultDocProcessorConfig() *DocProcessorConfig {
	return &DocProcessorConfig{
		MaxFileSize:     50 * 1024 * 1024, // 50MB
		UseAntiword:     true,
		AntiwordPath:    "", // 从 PATH 查找
		UseLibreOffice:  true,
		LibreOfficePath: "", // 从 PATH 查找
		FallbackToBasic: true,
	}
}

// NewDocProcessor 创建 DOC 处理器
func NewDocProcessor() *DocProcessor {
	return NewDocProcessorWithConfig(nil)
}

// NewDocProcessorWithConfig 使用指定配置创建 DOC 处理器
func NewDocProcessorWithConfig(config *DocProcessorConfig) *DocProcessor {
	if config == nil {
		config = DefaultDocProcessorConfig()
	}

	base := NewBaseProcessor(
		"DocProcessor",
		"Microsoft Word 97-2003 文档处理器 (.doc)",
		[]string{"doc"},
	)

	return &DocProcessor{
		base:   base,
		config: config,
	}
}

// Name 返回处理器名称
func (p *DocProcessor) Name() string {
	return p.base.Name()
}

// Description 返回处理器描述
func (p *DocProcessor) Description() string {
	return p.base.Description()
}

// SupportedTypes 返回支持的文件类型
func (p *DocProcessor) SupportedTypes() []string {
	return p.base.SupportedTypes()
}

// Process 处理 DOC 文件，返回提取的文本
func (p *DocProcessor) Process(filePath string) (string, error) {
	result, err := p.ProcessWithStyle(filePath)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

// ProcessWithStyle 处理 DOC 文件并返回版式特征
func (p *DocProcessor) ProcessWithStyle(filePath string) (*ProcessResultWithStyle, error) {
	result := &ProcessResultWithStyle{
		HasStyle: false,
	}

	// 验证文件
	if err := p.validateFile(filePath); err != nil {
		return nil, err
	}

	var text string
	var err error
	var extractMethod string

	// 方法 1: 尝试使用 antiword
	if p.config.UseAntiword {
		text, err = p.extractWithAntiword(filePath)
		if err == nil && strings.TrimSpace(text) != "" {
			extractMethod = "antiword"
		}
	}

	// 方法 2: 尝试使用 LibreOffice
	if text == "" && p.config.UseLibreOffice {
		text, err = p.extractWithLibreOffice(filePath)
		if err == nil && strings.TrimSpace(text) != "" {
			extractMethod = "libreoffice"
		}
	}

	// 方法 3: 回退到基础提取
	if text == "" && p.config.FallbackToBasic {
		text, err = p.extractBasic(filePath)
		if err == nil && strings.TrimSpace(text) != "" {
			extractMethod = "basic"
		}
	}

	// 所有方法都失败
	if strings.TrimSpace(text) == "" {
		return nil, NewProcessorError(p.Name(), filePath, "提取文本",
			fmt.Errorf("无法提取 DOC 文件内容，请安装 antiword 或 LibreOffice"))
	}

	// 清理文本
	text = p.cleanText(text)
	result.Text = text

	// 创建基础版式特征
	result.StyleFeatures = &extractor.StyleFeatures{
		StyleReasons: []string{
			fmt.Sprintf("通过 %s 提取文本", extractMethod),
			"DOC 格式无法提取完整版式信息",
		},
	}

	return result, nil
}

// validateFile 验证文件
func (p *DocProcessor) validateFile(filePath string) error {
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

	// 验证文件扩展名
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if ext != "doc" {
		return NewProcessorError(p.Name(), filePath, "检查文件类型",
			fmt.Errorf("不支持的文件格式: %s", ext))
	}

	// 验证文件魔数 (OLE2 复合文档)
	if err := p.validateMagicNumber(filePath); err != nil {
		return NewProcessorError(p.Name(), filePath, "验证文件格式", err)
	}

	return nil
}

// validateMagicNumber 验证 OLE2 魔数
func (p *DocProcessor) validateMagicNumber(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	header := make([]byte, 8)
	n, err := file.Read(header)
	if err != nil || n < 8 {
		return fmt.Errorf("无法读取文件头")
	}

	// OLE2 复合文档魔数: D0 CF 11 E0 A1 B1 1A E1
	ole2Magic := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	if !bytes.Equal(header, ole2Magic) {
		return fmt.Errorf("不是有效的 DOC 文件格式")
	}

	return nil
}

// extractWithAntiword 使用 antiword 提取文本
func (p *DocProcessor) extractWithAntiword(filePath string) (string, error) {
	antiwordPath := p.config.AntiwordPath
	if antiwordPath == "" {
		antiwordPath = "antiword"
	}

	// 检查 antiword 是否可用
	if _, err := exec.LookPath(antiwordPath); err != nil {
		return "", fmt.Errorf("antiword 未安装或不在 PATH 中")
	}

	// 执行 antiword
	cmd := exec.Command(antiwordPath, "-m", "UTF-8", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("antiword 执行失败: %w", err)
	}

	return string(output), nil
}

// extractWithLibreOffice 使用 LibreOffice 提取文本
func (p *DocProcessor) extractWithLibreOffice(filePath string) (string, error) {
	// 查找 LibreOffice
	loPath := p.findLibreOffice()
	if loPath == "" {
		return "", fmt.Errorf("LibreOffice 未安装")
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "doc_convert_")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 获取绝对路径
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 使用 LibreOffice 转换为文本（指定 UTF-8 编码）
	cmd := exec.Command(loPath,
		"--headless",
		"--convert-to", "txt:Text (encoded):UTF8",
		"--outdir", tmpDir,
		absFilePath,
	)

	// 捕获错误输出
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("LibreOffice 转换失败: %w, stderr: %s", err, stderr.String())
	}

	// 读取转换后的文本文件
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	txtPath := filepath.Join(tmpDir, baseName+".txt")

	// 检查文件是否存在
	if _, err := os.Stat(txtPath); os.IsNotExist(err) {
		// 尝试列出目录内容，看看生成了什么文件
		files, _ := os.ReadDir(tmpDir)
		var fileNames []string
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
		return "", fmt.Errorf("转换后的文件不存在: %s, 目录内容: %v", txtPath, fileNames)
	}

	content, err := os.ReadFile(txtPath)
	if err != nil {
		return "", fmt.Errorf("读取转换结果失败: %w", err)
	}

	// 处理可能的 BOM
	text := string(content)
	text = strings.TrimPrefix(text, "\xEF\xBB\xBF") // UTF-8 BOM

	// 如果是空的，尝试其他编码
	if strings.TrimSpace(text) == "" {
		// 可能是 UTF-16 编码
		if len(content) >= 2 {
			if content[0] == 0xFF && content[1] == 0xFE {
				// UTF-16 LE
				text = decodeUTF16LEForDoc(content[2:])
			} else if content[0] == 0xFE && content[1] == 0xFF {
				// UTF-16 BE
				text = decodeUTF16BEForDoc(content[2:])
			}
		}
	}

	return text, nil
}

// decodeUTF16LE 解码 UTF-16 LE
func decodeUTF16LEForDoc(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}

	u16s := make([]uint16, len(data)/2)
	for i := 0; i < len(u16s); i++ {
		u16s[i] = uint16(data[2*i]) | uint16(data[2*i+1])<<8
	}

	return string(utf16.Decode(u16s))
}

// decodeUTF16BE 解码 UTF-16 BE
func decodeUTF16BEForDoc(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}

	u16s := make([]uint16, len(data)/2)
	for i := 0; i < len(u16s); i++ {
		u16s[i] = uint16(data[2*i])<<8 | uint16(data[2*i+1])
	}

	return string(utf16.Decode(u16s))
}

// findLibreOffice 查找 LibreOffice 可执行文件
func (p *DocProcessor) findLibreOffice() string {
	if p.config.LibreOfficePath != "" {
		if _, err := os.Stat(p.config.LibreOfficePath); err == nil {
			return p.config.LibreOfficePath
		}
	}

	// 常见的 LibreOffice 可执行文件名
	names := []string{
		"soffice",
		"libreoffice",
		"loffice",
	}

	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}

	// Windows 常见路径
	windowsPaths := []string{
		`C:\Program Files\LibreOffice\program\soffice.exe`,
		`C:\Program Files (x86)\LibreOffice\program\soffice.exe`,
		`D:\Program Files\LibreOffice\program\soffice.exe`,
		`D:\LibreOffice\program\soffice.exe`,
	}

	for _, path := range windowsPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// macOS 常见路径
	macPaths := []string{
		"/Applications/LibreOffice.app/Contents/MacOS/soffice",
	}

	for _, path := range macPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Linux 常见路径
	linuxPaths := []string{
		"/usr/bin/soffice",
		"/usr/bin/libreoffice",
		"/opt/libreoffice/program/soffice",
	}

	for _, path := range linuxPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// extractBasic 基础文本提取（从 OLE2 复合文档中提取可见文本）
func (p *DocProcessor) extractBasic(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	var allText []string

	// 方法 1: 尝试查找 Word Document 流中的文本
	wordText := p.extractWordDocumentStream(content)
	if wordText != "" {
		allText = append(allText, wordText)
	}

	// 方法 2: 提取 Unicode (UTF-16LE) 文本
	if len(allText) == 0 {
		unicodeText := p.extractUnicodeText(content)
		if unicodeText != "" {
			allText = append(allText, unicodeText)
		}
	}

	// 方法 3: 提取 ASCII 文本（作为最后手段）
	if len(allText) == 0 {
		asciiText := p.extractASCIIText(content)
		if asciiText != "" {
			allText = append(allText, asciiText)
		}
	}

	if len(allText) == 0 {
		return "", fmt.Errorf("无法提取文本内容")
	}

	// 合并结果
	result := strings.Join(allText, "\n")
	return result, nil
}

// extractWordDocumentStream 尝试从 Word Document 流提取文本
func (p *DocProcessor) extractWordDocumentStream(content []byte) string {
	var result strings.Builder
	var currentRun []rune

	// 扫描整个文件，查找连续的可打印字符序列
	i := 0
	for i < len(content)-1 {
		// 尝试读取 UTF-16LE 字符
		lo := content[i]
		hi := content[i+1]
		char := rune(lo) | rune(hi)<<8

		// 检查是否是有效字符
		if p.isValidDocChar(char) {
			currentRun = append(currentRun, char)
			i += 2
		} else {
			// 保存当前片段
			if len(currentRun) >= 3 {
				text := string(currentRun)
				// ���保留包含中文或足够长的文本
				if p.containsChinese(text) || (len(text) >= 10 && p.isReadableText(text)) {
					if result.Len() > 0 {
						result.WriteString(" ")
					}
					result.WriteString(text)
				}
			}
			currentRun = nil
			i++
		}
	}

	// 处理最后一个片段
	if len(currentRun) >= 3 {
		text := string(currentRun)
		if p.containsChinese(text) || (len(text) >= 10 && p.isReadableText(text)) {
			if result.Len() > 0 {
				result.WriteString(" ")
			}
			result.WriteString(text)
		}
	}

	return result.String()
}

// isValidDocChar 检查是否是有效的文档字符
func (p *DocProcessor) isValidDocChar(r rune) bool {
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
	// 中文标点 (0x3000-0x303F)
	if r >= 0x3000 && r <= 0x303F {
		return true
	}
	// 全角字符 (0xFF00-0xFFEF)
	if r >= 0xFF00 && r <= 0xFFEF {
		return true
	}
	// 常见中文标点（使用 Unicode 码点）
	switch r {
	case 0xFF0C, // ，
		0x3002, // 。
		0x3001, // 、
		0xFF1B, // ；
		0xFF1A, // ：
		0xFF1F, // ？
		0xFF01, // ！
		0x201C, // "
		0x201D, // "
		0x2018, // '
		0x2019, // '
		0xFF08, // （
		0xFF09, // ）
		0x3010, // 【
		0x3011, // 】
		0x300A, // 《
		0x300B, // 》
		0x3014, // 〔
		0x3015, // 〕
		0x2014, // —
		0x2026, // …
		0x00B7: // ·
		return true
	}
	return false
}

// isReadableText 检查是否是可读文本
func (p *DocProcessor) isReadableText(text string) bool {
	if len(text) < 3 {
		return false
	}

	// 计算可读字符比例
	readable := 0
	total := 0
	for _, r := range text {
		total++
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == ' ' ||
			(r >= 0x4E00 && r <= 0x9FFF) {
			readable++
		}
	}

	if total == 0 {
		return false
	}

	return float64(readable)/float64(total) > 0.5
}

// extractUnicodeText 提取 UTF-16LE 编码的文本
func (p *DocProcessor) extractUnicodeText(content []byte) string {
	if len(content) < 2 {
		return ""
	}

	var textParts []string
	var currentPart []rune

	// 扫描寻找连续的 UTF-16LE 字符
	for i := 0; i < len(content)-1; i += 2 {
		lo := content[i]
		hi := content[i+1]
		char := rune(lo) | rune(hi)<<8

		// 检查是否是有效的可打印字符
		if p.isValidDocChar(char) {
			currentPart = append(currentPart, char)
		} else {
			// 保存当前部分（如果足够长）
			if len(currentPart) >= 4 {
				text := string(currentPart)
				if p.containsChinese(text) || len(text) >= 10 {
					textParts = append(textParts, text)
				}
			}
			currentPart = nil
		}
	}

	// 处理最后一部分
	if len(currentPart) >= 4 {
		text := string(currentPart)
		if p.containsChinese(text) || len(text) >= 10 {
			textParts = append(textParts, text)
		}
	}

	return strings.Join(textParts, " ")
}

// extractASCIIText 提取 ASCII 文本
func (p *DocProcessor) extractASCIIText(content []byte) string {
	var textParts []string
	var currentPart strings.Builder

	for _, b := range content {
		// 可打印 ASCII 字符或空白字符
		if (b >= 0x20 && b <= 0x7E) || b == 0x0A || b == 0x0D || b == 0x09 {
			currentPart.WriteByte(b)
		} else {
			// 保存当前部分（如果足够长）
			if currentPart.Len() >= 20 {
				text := strings.TrimSpace(currentPart.String())
				if text != "" && !p.isGarbage(text) {
					textParts = append(textParts, text)
				}
			}
			currentPart.Reset()
		}
	}

	// 处理最后一部分
	if currentPart.Len() >= 20 {
		text := strings.TrimSpace(currentPart.String())
		if text != "" && !p.isGarbage(text) {
			textParts = append(textParts, text)
		}
	}

	return strings.Join(textParts, " ")
}

// containsChinese 检查是否包含中文字符
func (p *DocProcessor) containsChinese(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// isGarbage 检查是否是垃圾文本
func (p *DocProcessor) isGarbage(text string) bool {
	// 检查是否主要由重复字符组成
	if len(text) > 10 {
		charCount := make(map[rune]int)
		total := 0
		for _, r := range text {
			charCount[r]++
			total++
		}

		// 如果某个字符占比超过 50%，可能是垃圾
		for _, count := range charCount {
			if float64(count)/float64(total) > 0.5 {
				return true
			}
		}
	}

	// 检查是否包含常见的二进制垃圾模式
	garbagePatterns := []string{
		"AAAAAAAA",
		"........",
		"________",
		"////////",
		"\\\\\\\\",
	}

	for _, pattern := range garbagePatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}

	return false
}

// cleanText 清理提取的文本
func (p *DocProcessor) cleanText(text string) string {
	// 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 移除控制字符
	text = p.removeControlChars(text)

	// 移除多余空行
	lines := strings.Split(text, "\n")
	var cleanedLines []string
	emptyCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			emptyCount++
			if emptyCount <= 2 {
				cleanedLines = append(cleanedLines, "")
			}
		} else {
			emptyCount = 0
			cleanedLines = append(cleanedLines, line)
		}
	}

	text = strings.Join(cleanedLines, "\n")
	text = strings.TrimSpace(text)

	return text
}

// removeControlChars 移除控制字符
func (p *DocProcessor) removeControlChars(text string) string {
	var result strings.Builder

	for _, r := range text {
		// 保留正常字符、空格、换行符、制表符
		if r >= 0x20 || r == '\n' || r == '\t' {
			result.WriteRune(r)
		} else if r == '\r' {
			// 跳过回车（已经处理过）
			continue
		}
	}

	return result.String()
}

// IsAntiwordAvailable 检查 antiword 是否可用
func (p *DocProcessor) IsAntiwordAvailable() bool {
	antiwordPath := p.config.AntiwordPath
	if antiwordPath == "" {
		antiwordPath = "antiword"
	}
	_, err := exec.LookPath(antiwordPath)
	return err == nil
}

// IsLibreOfficeAvailable 检查 LibreOffice 是否可用
func (p *DocProcessor) IsLibreOfficeAvailable() bool {
	return p.findLibreOffice() != ""
}

// GetStatus 获取处理器状态
func (p *DocProcessor) GetStatus() string {
	var status strings.Builder

	status.WriteString("DOC 处理器状态:\n")

	if p.IsAntiwordAvailable() {
		status.WriteString("  antiword: 可用\n")
	} else {
		status.WriteString("  antiword: 不可用\n")
	}

	if p.IsLibreOfficeAvailable() {
		status.WriteString("  LibreOffice: 可用\n")
	} else {
		status.WriteString("  LibreOffice: 不可用\n")
	}

	if p.config.FallbackToBasic {
		status.WriteString("  基础提取: 已启用\n")
	}

	return status.String()
}

// GetDocExtractorInfo 获取可用的 DOC 提取器信息
func GetDocExtractorInfo() map[string]bool {
	proc := NewDocProcessor()
	return map[string]bool{
		"antiword":    proc.IsAntiwordAvailable(),
		"libreoffice": proc.IsLibreOfficeAvailable(),
		"basic":       true,
	}
}

// CleanDocText 清理 DOC 文档中提取的文本（导出函数）
func CleanDocText(text string) string {
	proc := NewDocProcessor()
	return proc.cleanText(text)
}

// ExtractDocText 提取 DOC 文档文本（便捷函数）
func ExtractDocText(filePath string) (string, error) {
	proc := NewDocProcessor()
	return proc.Process(filePath)
}

// ValidateDocFile 验证 DOC 文件格式
func ValidateDocFile(filePath string) error {
	proc := NewDocProcessor()
	return proc.validateMagicNumber(filePath)
}

// DocTextFilter 用于过滤和清理 DOC 文本
type DocTextFilter struct {
	minLineLength    int
	removeEmptyLines bool
	removeDuplicates bool
}

// NewDocTextFilter 创建文本过滤器
func NewDocTextFilter() *DocTextFilter {
	return &DocTextFilter{
		minLineLength:    2,
		removeEmptyLines: true,
		removeDuplicates: false,
	}
}

// Filter 过滤文本
func (f *DocTextFilter) Filter(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检查最小长度
		if len([]rune(line)) < f.minLineLength {
			if !f.removeEmptyLines && line == "" {
				result = append(result, "")
			}
			continue
		}

		// 检查重复
		if f.removeDuplicates {
			if seen[line] {
				continue
			}
			seen[line] = true
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// 正则表达式预编译
var (
	docDatePattern   = regexp.MustCompile(`\d{4}\s*年\s*\d{1,2}\s*月\s*\d{1,2}\s*日`)
	docNumberPattern = regexp.MustCompile(`[\p{Han}A-Za-z]+[发字]〔\d{4}〕\d+号`)
)

// ExtractDocMetadata 从 DOC 文本中提取元数据
func ExtractDocMetadata(text string) map[string]string {
	metadata := make(map[string]string)

	// 提取日期
	if match := docDatePattern.FindString(text); match != "" {
		metadata["date"] = match
	}

	// 提取发文字号
	if match := docNumberPattern.FindString(text); match != "" {
		metadata["doc_number"] = match
	}

	return metadata
}