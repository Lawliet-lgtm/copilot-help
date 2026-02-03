package processor

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
)

// PdfTextExtractor PDF文本提取器
type PdfTextExtractor struct {
	parser        *PdfParser
	cMaps         map[string]map[uint16]rune
	fontEncodings map[string]string
}

// NewPdfTextExtractor 创建文本提取器
func NewPdfTextExtractor(parser *PdfParser) *PdfTextExtractor {
	return &PdfTextExtractor{
		parser:        parser,
		cMaps:         make(map[string]map[uint16]rune),
		fontEncodings: make(map[string]string),
	}
}

// ExtractText 提取所有页面的文本
func (e *PdfTextExtractor) ExtractText() (string, error) {
	pages := e.parser.GetPages()
	if len(pages) == 0 {
		return "", nil
	}

	var allText strings.Builder

	for i, page := range pages {
		pageText, err := e.extractPageText(&page)
		if err != nil {
			continue
		}

		if pageText != "" {
			if i > 0 {
				allText.WriteString("\n")
			}
			allText.WriteString(pageText)
		}
	}

	// 后处理
	result := postProcessPdfText(allText.String())
	return result, nil
}

// extractPageText 提取单个页面的文本
func (e *PdfTextExtractor) extractPageText(page *PdfDictObject) (string, error) {
	e.loadPageFonts(page)

	contentsObj := page.Get("Contents")
	if contentsObj == nil {
		return "", nil
	}

	contentData, err := e.getContentData(contentsObj)
	if err != nil {
		return "", err
	}

	return e.parseContentStream(contentData), nil
}

// loadPageFonts 加载页面使用的字体
func (e *PdfTextExtractor) loadPageFonts(page *PdfDictObject) {
	resourcesObj := page.Get("Resources")
	if resourcesObj == nil {
		return
	}

	resources, err := e.parser.resolveRef(resourcesObj)
	if err != nil {
		return
	}

	resourcesDict, ok := resources.(PdfDictObject)
	if !ok {
		return
	}

	fontObj := resourcesDict.Get("Font")
	if fontObj == nil {
		return
	}

	fonts, err := e.parser.resolveRef(fontObj)
	if err != nil {
		return
	}

	fontsDict, ok := fonts.(PdfDictObject)
	if !ok {
		return
	}

	for fontName, fontRef := range fontsDict.Dict {
		fontDef, err := e.parser.resolveRef(fontRef)
		if err != nil {
			continue
		}

		fontDict, ok := fontDef.(PdfDictObject)
		if !ok {
			continue
		}

		encoding := fontDict.GetString("Encoding")
		if encoding != "" {
			e.fontEncodings[fontName] = encoding
		}

		toUnicodeObj := fontDict.Get("ToUnicode")
		if toUnicodeObj != nil {
			toUnicode, err := e.parser.resolveRef(toUnicodeObj)
			if err != nil {
				continue
			}

			if stream, ok := toUnicode.(PdfStreamObject); ok {
				data, err := stream.GetDecodedData()
				if err == nil {
					cmap := parseCMap(data)
					if len(cmap) > 0 {
						e.cMaps[fontName] = cmap
					}
				}
			}
		}
	}
}

// getContentData 获取内容流数据
func (e *PdfTextExtractor) getContentData(contentsObj PdfObject) ([]byte, error) {
	contents, err := e.parser.resolveRef(contentsObj)
	if err != nil {
		return nil, err
	}

	switch c := contents.(type) {
	case PdfStreamObject:
		return c.GetDecodedData()

	case PdfArrayObject:
		var allData bytes.Buffer
		for _, item := range c.Items {
			streamObj, err := e.parser.resolveRef(item)
			if err != nil {
				continue
			}
			if stream, ok := streamObj.(PdfStreamObject); ok {
				data, err := stream.GetDecodedData()
				if err == nil {
					allData.Write(data)
					allData.WriteByte('\n')
				}
			}
		}
		return allData.Bytes(), nil

	default:
		return nil, nil
	}
}

// parseContentStream 解析内容流提取文本（简化版，不添加换行）
func (e *PdfTextExtractor) parseContentStream(data []byte) string {
	var result strings.Builder
	var currentFont string

	tokens := tokenizeContentStream(data)
	operandStack := make([]string, 0)

	for _, token := range tokens {
		if isOperator(token) {
			switch token {
			case "Tf":
				if len(operandStack) >= 2 {
					fontName := operandStack[len(operandStack)-2]
					fontName = strings.TrimPrefix(fontName, "/")
					currentFont = fontName
				}

			case "Tj":
				if len(operandStack) >= 1 {
					text := e.decodeTextString(operandStack[len(operandStack)-1], currentFont)
					result.WriteString(text)
				}

			case "TJ":
				if len(operandStack) >= 1 {
					text := e.decodeTJArray(operandStack[len(operandStack)-1], currentFont)
					result.WriteString(text)
				}

			case "'", "\"":
				// 这些操作符包含换行语义
				if len(operandStack) >= 1 {
					result.WriteString("\n")
					text := e.decodeTextString(operandStack[len(operandStack)-1], currentFont)
					result.WriteString(text)
				}

			case "T*":
				// 明确的换行
				result.WriteString("\n")
			}

			operandStack = operandStack[:0]
		} else {
			operandStack = append(operandStack, token)
		}
	}

	return result.String()
}

// decodeTextString 解码文本字符串
func (e *PdfTextExtractor) decodeTextString(s string, fontName string) string {
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	s = strings.TrimPrefix(s, "<")
	s = strings.TrimSuffix(s, ">")

	if isHexString(s) {
		return e.decodeHexString(s, fontName)
	}

	s = unescapePdfString(s)

	if cmap, ok := e.cMaps[fontName]; ok && len(cmap) > 0 {
		return e.applyCMap(s, cmap)
	}

	return convertToUTF8(s)
}

// decodeHexString 解码十六进制字符串
func (e *PdfTextExtractor) decodeHexString(hex string, fontName string) string {
	hex = strings.ReplaceAll(hex, " ", "")
	hex = strings.ReplaceAll(hex, "\n", "")
	hex = strings.ReplaceAll(hex, "\r", "")

	if len(hex)%2 == 1 {
		hex += "0"
	}

	var data []byte
	for i := 0; i < len(hex); i += 2 {
		if val, err := strconv.ParseInt(hex[i:i+2], 16, 16); err == nil {
			data = append(data, byte(val))
		}
	}

	if cmap, ok := e.cMaps[fontName]; ok && len(cmap) > 0 {
		return e.applyCMapBytes(data, cmap)
	}

	if len(data) >= 2 && len(data)%2 == 0 {
		utf16Chars := make([]uint16, len(data)/2)
		for i := 0; i < len(data); i += 2 {
			utf16Chars[i/2] = uint16(data[i])<<8 | uint16(data[i+1])
		}

		hasValidChars := false
		for _, c := range utf16Chars {
			if c >= 0x4E00 && c <= 0x9FFF {
				hasValidChars = true
				break
			}
			if c >= 0x20 && c <= 0x7E {
				hasValidChars = true
				break
			}
		}

		if hasValidChars {
			return string(utf16.Decode(utf16Chars))
		}
	}

	return convertToUTF8(string(data))
}

// applyCMap 应用CMap映射
func (e *PdfTextExtractor) applyCMap(s string, cmap map[uint16]rune) string {
	var result strings.Builder

	data := []byte(s)
	for i := 0; i < len(data); {
		if i+1 < len(data) {
			code := uint16(data[i])<<8 | uint16(data[i+1])
			if r, ok := cmap[code]; ok {
				result.WriteRune(r)
				i += 2
				continue
			}
		}

		code := uint16(data[i])
		if r, ok := cmap[code]; ok {
			result.WriteRune(r)
		} else {
			result.WriteByte(data[i])
		}
		i++
	}

	return result.String()
}

// applyCMapBytes 应用CMap映射（字节数组版本）
func (e *PdfTextExtractor) applyCMapBytes(data []byte, cmap map[uint16]rune) string {
	var result strings.Builder

	for i := 0; i < len(data); {
		if i+1 < len(data) {
			code := uint16(data[i])<<8 | uint16(data[i+1])
			if r, ok := cmap[code]; ok {
				result.WriteRune(r)
				i += 2
				continue
			}
		}

		code := uint16(data[i])
		if r, ok := cmap[code]; ok {
			result.WriteRune(r)
		} else if data[i] >= 0x20 && data[i] <= 0x7E {
			result.WriteByte(data[i])
		}
		i++
	}

	return result.String()
}

// decodeTJArray 解码TJ操作符的数组参数
func (e *PdfTextExtractor) decodeTJArray(arrayStr string, fontName string) string {
	var result strings.Builder

	arrayStr = strings.TrimPrefix(arrayStr, "[")
	arrayStr = strings.TrimSuffix(arrayStr, "]")

	elements := parseTJArrayElements(arrayStr)

	for _, elem := range elements {
		elem = strings.TrimSpace(elem)
		if elem == "" {
			continue
		}

		if strings.HasPrefix(elem, "(") || strings.HasPrefix(elem, "<") {
			text := e.decodeTextString(elem, fontName)
			result.WriteString(text)
		}
	}

	return result.String()
}

// ============================================================
// 文本后处理
// ============================================================

// postProcessPdfText 后处理提取的PDF文本
func postProcessPdfText(text string) string {
	// 1. 移除控制字符
	text = removeControlChars(text)

	// 2. 移除所有多余空格（CJK字符之间的空格）
	text = removeAllExtraSpaces(text)

	// 3. 智能分行
	text = smartSplitLines(text)

	// 4. 清理
	text = cleanupText(text)

	return strings.TrimSpace(text)
}

// removeAllExtraSpaces 移除所有多余空格
func removeAllExtraSpaces(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return ""
	}

	var result strings.Builder

	for i := 0; i < len(runes); i++ {
		current := runes[i]

		if current == ' ' || current == '\t' {
			// 检查是否需要保留这个空格
			if shouldKeepThisSpace(runes, i) {
				result.WriteRune(' ')
			}
		} else if current == '\n' || current == '\r' {
			// 换行符转为特殊标记，后面处理
			result.WriteString("<<NEWLINE>>")
		} else {
			result.WriteRune(current)
		}
	}

	return result.String()
}

// shouldKeepThisSpace 判断是否保留空格
func shouldKeepThisSpace(runes []rune, pos int) bool {
	var prev, next rune

	// 向前找非空白字符
	for j := pos - 1; j >= 0; j-- {
		if runes[j] != ' ' && runes[j] != '\t' {
			prev = runes[j]
			break
		}
	}

	// 向后找非空白字符
	for j := pos + 1; j < len(runes); j++ {
		if runes[j] != ' ' && runes[j] != '\t' {
			next = runes[j]
			break
		}
	}

	if prev == 0 || next == 0 {
		return false
	}

	// 两个英文单词之间保留空格
	if isASCIILetter(prev) && isASCIILetter(next) {
		return true
	}

	// 其他情况都不保留
	return false
}

// smartSplitLines 智能分行
func smartSplitLines(text string) string {
	// 恢复换行标记
	text = strings.ReplaceAll(text, "<<NEWLINE>>", "\n")

	// 在特定位置添加换行
	text = addLineBreaks(text)

	return text
}

// addLineBreaks 在合适的位置添加换行
func addLineBreaks(text string) string {
	var result strings.Builder
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		result.WriteRune(r)

		// 在句号、问号、感叹号后添加换行（如果后面不是引号）
		if r == 0x3002 || r == 0xFF01 || r == 0xFF1F { // 。！？
			if i+1 < len(runes) {
				next := runes[i+1]
				// 如果后面不是引号，添加换行
				if !isQuote(next) && next != '\n' {
					result.WriteRune('\n')
				}
			}
		}

		// 在冒号后面如果紧跟换行相关的内容，保持换行
		if r == 0xFF1A || r == ':' { // ：或:
			// 这里不自动换行，保持原样
		}
	}

	return result.String()
}

// isQuote 检查是否是引号
func isQuote(r rune) bool {
	return r == 0x201D || r == 0x201C || r == 0x2019 || r == 0x2018 ||
		r == '"' || r == '\'' || r == 0x300B || r == 0x3009
}

// cleanupText 清理文本
func cleanupText(text string) string {
	// 规范化换行
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 处理每行
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// 跳过页码
		if isPageMarker(line) {
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	// 合并连续空行
	text = strings.Join(cleanedLines, "\n")
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return text
}

// isPageMarker 检查是否是页码标记
func isPageMarker(line string) bool {
	line = strings.TrimSpace(line)

	if len(line) == 0 {
		return false
	}

	// 纯数字且长度短
	allDigits := true
	for _, r := range line {
		if !unicode.IsDigit(r) {
			allDigits = false
			break
		}
	}
	if allDigits && len(line) <= 4 {
		return true
	}

	// 检测带破折号的页码: ––1–– 或 --1-- 或 — 1 —
	// 包含的字符：数字、各种破折号、空格
	validChars := true
	hasDigit := false
	dashCount := 0
	
	for _, r := range line {
		if unicode.IsDigit(r) {
			hasDigit = true
		} else if r == '-' || r == 0x2013 || r == 0x2014 || r == 0x2212 || r == 0x2010 || r == 0x2011 {
			// - (hyphen), – (en dash), — (em dash), − (minus), ‐ (hyphen), ‑ (non-breaking hyphen)
			dashCount++
		} else if r == ' ' {
			// 空格允许
		} else {
			validChars = false
			break
		}
	}
	
	// 如果只包含数字、破折号、空格，且有数字，且破折号数量合理
	if validChars && hasDigit && dashCount >= 2 && len(line) < 20 {
		return true
	}

	return false
}

// removeControlChars 移除控制字符
func removeControlChars(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			result.WriteRune(r)
		} else if r >= 0x20 && r != 0x7F {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// isCJKChar 检查是否是CJK字符
func isCJKChar(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x20000 && r <= 0x2A6DF) ||
		(r >= 0x2A700 && r <= 0x2B73F) ||
		(r >= 0x2B740 && r <= 0x2B81F)
}

// isCJKPunct 检查是否是CJK标点
func isCJKPunct(r rune) bool {
	if r >= 0x3000 && r <= 0x303F {
		return true
	}
	if r >= 0xFF00 && r <= 0xFFEF {
		return true
	}
	punctMarks := []rune{
		0x3014, 0x3015,
		0xFF08, 0xFF09,
		0xFF1A, 0xFF1B,
		0xFF0C, 0x3002,
		0x3001,
		0x300A, 0x300B,
		0x201C, 0x201D,
		0x2018, 0x2019,
	}
	for _, p := range punctMarks {
		if r == p {
			return true
		}
	}
	return false
}

// isASCIILetter 检查是否是ASCII字母
func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// ============================================================
// 内容流词法分析
// ============================================================

// tokenizeContentStream 对内容流进行词法分析
func tokenizeContentStream(data []byte) []string {
	var tokens []string
	var current strings.Builder

	i := 0
	for i < len(data) {
		ch := data[i]

		switch {
		case ch == '(':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			str := readLiteralStringToken(data, &i)
			tokens = append(tokens, str)

		case ch == '<' && i+1 < len(data) && data[i+1] != '<':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			str := readHexStringToken(data, &i)
			tokens = append(tokens, str)

		case ch == '[':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			arr := readArrayToken(data, &i)
			tokens = append(tokens, arr)

		case ch == '%':
			for i < len(data) && data[i] != '\n' && data[i] != '\r' {
				i++
			}

		case isWhitespace(ch):
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			i++

		default:
			current.WriteByte(ch)
			i++
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// readLiteralStringToken 读取文字字符串token
func readLiteralStringToken(data []byte, pos *int) string {
	var result strings.Builder
	result.WriteByte('(')
	*pos++

	depth := 1
	for *pos < len(data) && depth > 0 {
		ch := data[*pos]
		result.WriteByte(ch)

		if ch == '\\' && *pos+1 < len(data) {
			*pos++
			result.WriteByte(data[*pos])
		} else if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
		}
		*pos++
	}

	return result.String()
}

// readHexStringToken 读取十六进制字符串token
func readHexStringToken(data []byte, pos *int) string {
	var result strings.Builder
	result.WriteByte('<')
	*pos++

	for *pos < len(data) {
		ch := data[*pos]
		result.WriteByte(ch)
		*pos++
		if ch == '>' {
			break
		}
	}

	return result.String()
}

// readArrayToken 读取数组token
func readArrayToken(data []byte, pos *int) string {
	var result strings.Builder
	result.WriteByte('[')
	*pos++

	depth := 1
	for *pos < len(data) && depth > 0 {
		ch := data[*pos]

		if ch == '(' {
			str := readLiteralStringToken(data, pos)
			result.WriteString(str)
			continue
		}

		if ch == '<' && *pos+1 < len(data) && data[*pos+1] != '<' {
			str := readHexStringToken(data, pos)
			result.WriteString(str)
			continue
		}

		result.WriteByte(ch)
		if ch == '[' {
			depth++
		} else if ch == ']' {
			depth--
		}
		*pos++
	}

	return result.String()
}

// ============================================================
// 辅助函数
// ============================================================

// isOperator 检查是否是操作符
func isOperator(token string) bool {
	operators := map[string]bool{
		"Tc": true, "Tw": true, "Tz": true, "TL": true,
		"Tf": true, "Tr": true, "Ts": true,
		"Td": true, "TD": true, "Tm": true, "T*": true,
		"Tj": true, "TJ": true, "'": true, "\"": true,
		"BT": true, "ET": true,
		"q": true, "Q": true, "cm": true,
		"w": true, "J": true, "j": true, "M": true, "d": true,
		"ri": true, "i": true, "gs": true,
		"m": true, "l": true, "c": true, "v": true, "y": true, "h": true, "re": true,
		"S": true, "s": true, "f": true, "F": true, "f*": true,
		"B": true, "B*": true, "b": true, "b*": true, "n": true,
		"W": true, "W*": true,
		"CS": true, "cs": true, "SC": true, "SCN": true, "sc": true, "scn": true,
		"G": true, "g": true, "RG": true, "rg": true, "K": true, "k": true,
		"Do": true,
		"BMC": true, "BDC": true, "EMC": true,
		"MP": true, "DP": true,
		"BX": true, "EX": true,
	}
	return operators[token]
}

// isHexString 检查是否是十六进制字符串
func isHexString(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}

	for _, ch := range s {
		if !((ch >= '0' && ch <= '9') ||
			(ch >= 'a' && ch <= 'f') ||
			(ch >= 'A' && ch <= 'F') ||
			ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t') {
			return false
		}
	}

	return true
}

// unescapePdfString 处理PDF字符串转义
func unescapePdfString(s string) string {
	var result strings.Builder
	i := 0

	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				result.WriteByte('\n')
			case 'r':
				result.WriteByte('\r')
			case 't':
				result.WriteByte('\t')
			case 'b':
				result.WriteByte('\b')
			case 'f':
				result.WriteByte('\f')
			case '(', ')', '\\':
				result.WriteByte(s[i])
			default:
				if s[i] >= '0' && s[i] <= '7' {
					octal := string(s[i])
					for j := 0; j < 2 && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '7'; j++ {
						i++
						octal += string(s[i])
					}
					if val, err := strconv.ParseInt(octal, 8, 8); err == nil {
						result.WriteByte(byte(val))
					}
				} else {
					result.WriteByte(s[i])
				}
			}
		} else {
			result.WriteByte(s[i])
		}
		i++
	}

	return result.String()
}

// parseTJArrayElements 解析TJ数组元素
func parseTJArrayElements(s string) []string {
	var elements []string
	var current strings.Builder
	depth := 0
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if inString {
			current.WriteByte(ch)
			if ch == '\\' && i+1 < len(s) {
				i++
				current.WriteByte(s[i])
			} else if ch == stringChar {
				if stringChar == ')' {
					depth--
					if depth == 0 {
						inString = false
						elements = append(elements, current.String())
						current.Reset()
					}
				} else {
					inString = false
					elements = append(elements, current.String())
					current.Reset()
				}
			} else if ch == '(' && stringChar == ')' {
				depth++
			}
		} else {
			if ch == '(' {
				if current.Len() > 0 {
					elements = append(elements, current.String())
					current.Reset()
				}
				inString = true
				stringChar = ')'
				depth = 1
				current.WriteByte(ch)
			} else if ch == '<' {
				if current.Len() > 0 {
					elements = append(elements, current.String())
					current.Reset()
				}
				inString = true
				stringChar = '>'
				current.WriteByte(ch)
			} else if isWhitespace(ch) {
				if current.Len() > 0 {
					elements = append(elements, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(ch)
			}
		}
	}

	if current.Len() > 0 {
		elements = append(elements, current.String())
	}

	return elements
}

// parseFloat 解析浮点数
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

// convertToUTF8 尝试将字符串转换为UTF-8
func convertToUTF8(s string) string {
	if isValidUTF8String(s) {
		return s
	}

	if decoded, err := decodeGBK([]byte(s)); err == nil {
		return decoded
	}

	var result strings.Builder
	for _, r := range s {
		if r >= 0x20 && r != 0x7F {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// isValidUTF8String 检查字符串是否是有效的UTF-8
func isValidUTF8String(s string) bool {
	return isValidUTF8([]byte(s))
}

// ============================================================
// CMap解析
// ============================================================

// ParseToUnicodeCMap 解析ToUnicode CMap流
func ParseToUnicodeCMap(data []byte) map[uint16]rune {
	result := make(map[uint16]rune)

	bfcharPattern := regexp.MustCompile(`beginbfchar\s*(.*?)\s*endbfchar`)
	matches := bfcharPattern.FindAllSubmatch(data, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			parseBfcharBlock(match[1], result)
		}
	}

	bfrangePattern := regexp.MustCompile(`beginbfrange\s*(.*?)\s*endbfrange`)
	matches = bfrangePattern.FindAllSubmatch(data, -1)

	for _, match := range matches {
		if len(match) >= 2 {
			parseBfrangeBlock(match[1], result)
		}
	}

	return result
}

// parseBfcharBlock 解析bfchar块
func parseBfcharBlock(data []byte, result map[uint16]rune) {
	pattern := regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
	matches := pattern.FindAllSubmatch(data, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			srcHex := string(match[1])
			dstHex := string(match[2])

			srcVal, err1 := strconv.ParseUint(srcHex, 16, 16)
			dstVal, err2 := strconv.ParseUint(dstHex, 16, 32)

			if err1 == nil && err2 == nil {
				result[uint16(srcVal)] = rune(dstVal)
			}
		}
	}
}

// parseBfrangeBlock 解析bfrange块
func parseBfrangeBlock(data []byte, result map[uint16]rune) {
	pattern := regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
	matches := pattern.FindAllSubmatch(data, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			startHex := string(match[1])
			endHex := string(match[2])
			dstHex := string(match[3])

			startVal, err1 := strconv.ParseUint(startHex, 16, 16)
			endVal, err2 := strconv.ParseUint(endHex, 16, 16)
			dstVal, err3 := strconv.ParseUint(dstHex, 16, 32)

			if err1 == nil && err2 == nil && err3 == nil {
				for code := startVal; code <= endVal; code++ {
					result[uint16(code)] = rune(dstVal + (code - startVal))
				}
			}
		}
	}
}