package extractor

import (
	"strings"

	"linuxFileWatcher/internal/detector/govcheck/rules"
)

// Features 公文特征集合
type Features struct {
	// 版头特征
	CopyNumber      string // 份号 (新增)
	HasCopyNumber   bool   // 是否有份号 (新增)
	DocNumber       string // 发文字号
	HasDocNumber    bool   // 是否有发文字号
	SecretLevel     string // 密级
	HasSecretLevel  bool   // 是否有密级
	UrgencyLevel    string // 紧急程度
	HasUrgencyLevel bool   // 是否有紧急程度
	Issuer          string // 签发人
	HasIssuer       bool   // 是否有签发人

	// 主体特征
	Title         string // 公文标题
	HasTitle      bool   // 是否有公文标题
	TitleType     string // 标题文种 (通知/决定/意见等)
	MainSend      string // 主送机关
	HasMainSend   bool   // 是否有主送机关
	Attachment    string // 附件说明
	HasAttachment bool   // 是否有附件

	// 版记特征
	IssueDate    string // 成文日期
	HasIssueDate bool   // 是否有成文日期
	CopyTo       string // 抄送
	HasCopyTo    bool   // 是否有抄送
	PrintInfo    string // 印发信息
	HasPrintInfo bool   // 是否有印发信息

	// 机关特征
	OrgNames   []string // 识别到的机关名称
	HasOrgName bool     // 是否包含机关名称

	// 关键词特征
	DocTypes      []string // 匹配到的公文文种
	ActionWords   []string // 匹配到的公文动作词
	FormalWords   []string // 匹配到的正式用语
	HeaderWords   []string // 匹配到的版头关键词
	FooterWords   []string // 匹配到的版记关键词
	ProhibitWords []string // 匹配到的非公文特征词

	// ============================================================
	// 版式特征
	// ============================================================
	StyleFeatures *StyleFeatures // 版式特征

	// 统计信息
	TextLength       int     // 文本长度
	ChineseCharCount int     // 中文字符数
	PatternMatches   int     // 正则模式匹配数
	KeywordMatches   int     // 关键词匹配数
	TotalScore       float64 // 综合得分 (由Scorer计算)
}

// StyleFeatures 版式特征
type StyleFeatures struct {
	// 颜色特征
	HasRedText   bool     // 是否有红色文本
	HasRedHeader bool     // 是否有红头
	RedTextCount int      // 红色文本数量
	RedSamples   []string // 红色文本示例

	// 字体特征
	HasOfficialFonts bool    // 是否使用公文字体
	TitleFontMatch   bool    // 标题字号是否符合
	BodyFontMatch    bool    // 正文字号是否符合
	MainFontName     string  // 主要使用的字体
	MainFontSize     float64 // 主要使用的字号

	// 页面特征
	IsA4Paper   bool    // 是否A4纸
	MarginMatch bool    // 页边距是否符合
	PageWidth   float64 // 页面宽度(mm)
	PageHeight  float64 // 页面高度(mm)

	// 段落特征
	HasCenteredTitle bool    // 是否有居中标题
	LineSpacingMatch bool    // 行距是否符合
	LineSpacing      float64 // 行距(磅)

	// 印章特征
	HasSealImage  bool   // 是否有印章图片
	SealImageHint string // 印章提示

	// 综合评估
	StyleScore      float64  // 版式得分
	IsOfficialStyle bool     // 是否符合公文版式
	StyleReasons    []string // 判断理由
}

// Extractor 特征提取器
type Extractor struct {
	config *Config
}

// Config 提取器配置
type Config struct {
	EnablePatterns bool // 是否启用正则模���匹配
	EnableKeywords bool // 是否启用关键词匹配
	NormalizeText  bool // 是否预处理文本
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		EnablePatterns: true,
		EnableKeywords: true,
		NormalizeText:  true,
	}
}

// New 创建特征提取器
func New(config *Config) *Extractor {
	if config == nil {
		config = DefaultConfig()
	}
	return &Extractor{
		config: config,
	}
}

// Extract 从文本中提取公文特征
func (e *Extractor) Extract(text string) *Features {
	features := &Features{
		StyleFeatures: &StyleFeatures{
			StyleReasons: make([]string, 0),
		},
	}

	if text == "" {
		return features
	}

	// 文本预处理
	processedText := text
	if e.config.NormalizeText {
		processedText = e.preprocessText(text)
	}

	// 统计基本信息
	features.TextLength = len(text)
	features.ChineseCharCount = countChineseChars(text)

	// 提取正则模式特征
	if e.config.EnablePatterns {
		e.extractPatternFeatures(processedText, features)
	}

	// 提取关键词特征
	if e.config.EnableKeywords {
		e.extractKeywordFeatures(processedText, features)
	}

	return features
}

// ExtractWithStyle 从文本和版式信息中提取公文特征
func (e *Extractor) ExtractWithStyle(text string, styleInfo *StyleFeatures) *Features {
	// 先提取文本特征
	features := e.Extract(text)

	// 合并版式特征
	if styleInfo != nil {
		features.StyleFeatures = styleInfo
	}

	return features
}

// preprocessText 预处理文本
func (e *Extractor) preprocessText(text string) string {
	// 移除多余空白
	text = strings.TrimSpace(text)

	// 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 合并连续空行
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	return text
}

// extractPatternFeatures 提取正则模式特征
func (e *Extractor) extractPatternFeatures(text string, features *Features) {
	matchCount := 0

	// 提取份号 (新增)
	if copyNum := extractCopyNumber(text); copyNum != "" {
		features.CopyNumber = copyNum
		features.HasCopyNumber = true
		matchCount++
	}

	// 提取发文字号
	if docNum := rules.ExtractDocNumber(text); docNum != "" {
		features.DocNumber = docNum
		features.HasDocNumber = true
		matchCount++
	}

	// 提取公文标题
	if title := rules.ExtractTitle(text); title != "" {
		features.Title = title
		features.HasTitle = true
		matchCount++

		// 提取标题文种
		if titleType := rules.ExtractTitleType(title); titleType != "" {
			features.TitleType = titleType
		}
	}

	// 提取密级
	if secret := rules.ExtractSecretLevel(text); secret != "" {
		features.SecretLevel = secret
		features.HasSecretLevel = true
		matchCount++
	}

	// 提取紧急程度
	if urgency := rules.ExtractUrgencyLevel(text); urgency != "" {
		features.UrgencyLevel = urgency
		features.HasUrgencyLevel = true
		matchCount++
	}

	// 提取签发人
	if issuer := rules.ExtractIssuer(text); issuer != "" {
		features.Issuer = issuer
		features.HasIssuer = true
		matchCount++
	}

	// 提取主送机关
	if mainSend := rules.ExtractMainSend(text); mainSend != "" {
		features.MainSend = mainSend
		features.HasMainSend = true
		matchCount++
	}

	// 提取成文日期
	if issueDate := rules.ExtractIssueDate(text); issueDate != "" {
		features.IssueDate = issueDate
		features.HasIssueDate = true
		matchCount++
	}

	// 提取抄送
	if copyTo := rules.ExtractCopyTo(text); copyTo != "" {
		features.CopyTo = copyTo
		features.HasCopyTo = true
		matchCount++
	}

	// 提取附件
	if attachment := rules.ExtractAttachment(text); attachment != "" {
		features.Attachment = attachment
		features.HasAttachment = true
		matchCount++
	}

	// 提取印发信息
	if printInfo := rules.ExtractPrintInfo(text); printInfo != "" {
		features.PrintInfo = printInfo
		features.HasPrintInfo = true
		matchCount++
	}

	features.PatternMatches = matchCount
}

// extractCopyNumber 提取份号
// GB/T 9704-2012: 份号用6位3号阿拉伯数字，顶格编排在版心左上角第一行
func extractCopyNumber(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")

	// 只检查前10行
	maxLines := 10
	if len(lines) < maxLines {
		maxLines = len(lines)
	}

	for i := 0; i < maxLines; i++ {
		line := strings.TrimSpace(lines[i])

		// 跳过空行
		if line == "" {
			continue
		}

		// 模式1: 整行就是6位数字
		if len(line) == 6 && isAllDigits(line) {
			return line
		}

		// 模式2: "第000001号" 格式
		if strings.HasPrefix(line, "第") && strings.HasSuffix(line, "号") {
			inner := strings.TrimPrefix(line, "第")
			inner = strings.TrimSuffix(inner, "号")
			inner = strings.TrimSpace(inner)
			if len(inner) == 6 && isAllDigits(inner) {
				return inner
			}
		}

		// 模式3: "份号：000001" 或 "份号:000001" 格式
		if strings.Contains(line, "份号") {
			parts := strings.SplitN(line, "份号", 2)
			if len(parts) == 2 {
				numPart := strings.TrimSpace(parts[1])
				numPart = strings.TrimLeft(numPart, "：: ")
				if len(numPart) >= 6 {
					candidate := numPart[:6]
					if isAllDigits(candidate) {
						return candidate
					}
				}
			}
		}

		// 模式4: "编号：000001" 格式
		if strings.Contains(line, "编号") {
			parts := strings.SplitN(line, "编号", 2)
			if len(parts) == 2 {
				numPart := strings.TrimSpace(parts[1])
				numPart = strings.TrimLeft(numPart, "：: ")
				if len(numPart) >= 6 {
					candidate := numPart[:6]
					if isAllDigits(candidate) {
						return candidate
					}
				}
			}
		}

		// 模式5: 行首是6位数字（后面可能有其��内容）
		if len(line) >= 6 {
			prefix := line[:6]
			if isAllDigits(prefix) {
				// 确保第7个字符不是数字（确保是6位份号而不是更长的数字）
				if len(line) == 6 || !isDigitChar(line[6]) {
					return prefix
				}
			}
		}
	}

	return ""
}

// isAllDigits 检查字符串是否全为数字
func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isDigitChar 检查字节是否为数字
func isDigitChar(b byte) bool {
	return b >= '0' && b <= '9'
}

// extractKeywordFeatures 提取关键词特征
func (e *Extractor) extractKeywordFeatures(text string, features *Features) {
	totalMatches := 0

	// 提取机关名称
	orgNames := rules.FindOrgNames(text)
	if len(orgNames) > 0 {
		features.OrgNames = uniqueStrings(orgNames)
		features.HasOrgName = true
		totalMatches += len(orgNames)
	}

	// 提取公文文种
	docTypes := rules.FindDocTypes(text)
	if len(docTypes) > 0 {
		features.DocTypes = uniqueStrings(docTypes)
		totalMatches += len(docTypes)
	}

	// 提取动作词
	actionWords := rules.ActionKeywords.FindAll(text)
	if len(actionWords) > 0 {
		features.ActionWords = uniqueStrings(actionWords)
		totalMatches += len(actionWords)
	}

	// 提取正式用语
	formalWords := rules.FormalityKeywords.FindAll(text)
	if len(formalWords) > 0 {
		features.FormalWords = uniqueStrings(formalWords)
		totalMatches += len(formalWords)
	}

	// 提取版头关键词
	headerWords := rules.HeaderKeywords.FindAll(text)
	if len(headerWords) > 0 {
		features.HeaderWords = uniqueStrings(headerWords)
		totalMatches += len(headerWords)
	}

	// 提取版记关键词
	footerWords := rules.FooterKeywords.FindAll(text)
	if len(footerWords) > 0 {
		features.FooterWords = uniqueStrings(footerWords)
		totalMatches += len(footerWords)
	}

	// 提取非公文特征词（反向指标）
	prohibitWords := rules.GetProhibitedMatches(text)
	if len(prohibitWords) > 0 {
		features.ProhibitWords = uniqueStrings(prohibitWords)
	}

	features.KeywordMatches = totalMatches
}

// FeatureSummary 返回特征摘要
func (f *Features) FeatureSummary() map[string]interface{} {
	summary := make(map[string]interface{})

	// 版头特征
	summary["has_copy_number"] = f.HasCopyNumber
	summary["has_doc_number"] = f.HasDocNumber
	summary["has_secret_level"] = f.HasSecretLevel
	summary["has_urgency_level"] = f.HasUrgencyLevel
	summary["has_issuer"] = f.HasIssuer

	// 主体特征
	summary["has_title"] = f.HasTitle
	summary["title_type"] = f.TitleType
	summary["has_main_send"] = f.HasMainSend
	summary["has_attachment"] = f.HasAttachment

	// 版记特征
	summary["has_issue_date"] = f.HasIssueDate
	summary["has_copy_to"] = f.HasCopyTo
	summary["has_print_info"] = f.HasPrintInfo

	// 机关特征
	summary["has_org_name"] = f.HasOrgName
	summary["org_count"] = len(f.OrgNames)

	// 版式特征
	if f.StyleFeatures != nil {
		summary["has_red_text"] = f.StyleFeatures.HasRedText
		summary["has_red_header"] = f.StyleFeatures.HasRedHeader
		summary["is_a4_paper"] = f.StyleFeatures.IsA4Paper
		summary["has_official_fonts"] = f.StyleFeatures.HasOfficialFonts
		summary["style_score"] = f.StyleFeatures.StyleScore
	}

	// 统计
	summary["pattern_matches"] = f.PatternMatches
	summary["keyword_matches"] = f.KeywordMatches
	summary["prohibit_words_count"] = len(f.ProhibitWords)

	return summary
}

// CountPositiveFeatures 统计正向特征数量
func (f *Features) CountPositiveFeatures() int {
	count := 0

	if f.HasCopyNumber {
		count++
	}
	if f.HasDocNumber {
		count++
	}
	if f.HasTitle {
		count++
	}
	if f.HasSecretLevel {
		count++
	}
	if f.HasUrgencyLevel {
		count++
	}
	if f.HasIssuer {
		count++
	}
	if f.HasMainSend {
		count++
	}
	if f.HasIssueDate {
		count++
	}
	if f.HasCopyTo {
		count++
	}
	if f.HasPrintInfo {
		count++
	}
	if f.HasAttachment {
		count++
	}
	if f.HasOrgName {
		count++
	}

	return count
}

// CountStyleFeatures 统计版式特征数量
func (f *Features) CountStyleFeatures() int {
	if f.StyleFeatures == nil {
		return 0
	}

	count := 0
	s := f.StyleFeatures

	if s.HasRedText {
		count++
	}
	if s.HasRedHeader {
		count++
	}
	if s.HasOfficialFonts {
		count++
	}
	if s.TitleFontMatch {
		count++
	}
	if s.BodyFontMatch {
		count++
	}
	if s.IsA4Paper {
		count++
	}
	if s.MarginMatch {
		count++
	}
	if s.HasCenteredTitle {
		count++
	}
	if s.HasSealImage {
		count++
	}

	return count
}

// HasCriticalFeatures 检查是否具有关键公文特征
func (f *Features) HasCriticalFeatures() bool {
	criticalCount := 0

	if f.HasDocNumber {
		criticalCount++
	}
	if f.HasTitle {
		criticalCount++
	}
	if f.HasIssueDate {
		criticalCount++
	}

	return criticalCount >= 2
}

// HasStyleFeatures 检查是否有版式特征
func (f *Features) HasStyleFeatures() bool {
	if f.StyleFeatures == nil {
		return false
	}
	return f.StyleFeatures.HasRedText || f.StyleFeatures.HasRedHeader ||
		f.StyleFeatures.HasOfficialFonts || f.StyleFeatures.IsA4Paper ||
		f.StyleFeatures.HasSealImage
}

// HasProhibitedContent 检查是否包含非公文特征
func (f *Features) HasProhibitedContent() bool {
	return len(f.ProhibitWords) > 0
}

// GetProhibitedRatio 获取非公文特征词占比
func (f *Features) GetProhibitedRatio() float64 {
	if f.KeywordMatches == 0 {
		return 0
	}
	return float64(len(f.ProhibitWords)) / float64(f.KeywordMatches+len(f.ProhibitWords))
}

// ============================================================
// 辅助函数
// ============================================================

// countChineseChars 统计中文字符数量
func countChineseChars(text string) int {
	count := 0
	for _, r := range text {
		if isChinese(r) {
			count++
		}
	}
	return count
}

// isChinese 判断是否为中文字符
func isChinese(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF
}

// uniqueStrings 去重字符串切片
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))

	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}

// ============================================================
// 快捷函数
// ============================================================

// ExtractFeatures 快捷函数：使用默认配置提取特征
func ExtractFeatures(text string) *Features {
	ext := New(nil)
	return ext.Extract(text)
}

// QuickCheck 快速检查文本是否可能是公文
func QuickCheck(text string) bool {
	if len(text) < 100 {
		return false
	}

	hasDocNumber := rules.DocNumberPattern.Match(text)
	hasTitle := rules.TitlePattern.Match(text)
	hasDate := rules.IssueDatePattern.Match(text)
	hasOrg := rules.ContainsOrgName(text)
	hasDocType := rules.ContainsDocType(text)

	matchCount := 0
	if hasDocNumber {
		matchCount++
	}
	if hasTitle {
		matchCount++
	}
	if hasDate {
		matchCount++
	}
	if hasOrg {
		matchCount++
	}
	if hasDocType {
		matchCount++
	}

	return matchCount >= 2
}

// AnalyzeText 完整分析文本并返回结构化结果
func AnalyzeText(text string) *AnalysisResult {
	features := ExtractFeatures(text)

	return &AnalysisResult{
		Features:           features,
		PositiveCount:      features.CountPositiveFeatures(),
		StyleCount:         features.CountStyleFeatures(),
		HasCritical:        features.HasCriticalFeatures(),
		HasStyle:           features.HasStyleFeatures(),
		HasProhibited:      features.HasProhibitedContent(),
		ProhibitedRatio:    features.GetProhibitedRatio(),
		QuickCheckPassed:   QuickCheck(text),
		RecommendedForScan: features.CountPositiveFeatures() >= 3,
	}
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Features           *Features
	PositiveCount      int
	StyleCount         int
	HasCritical        bool
	HasStyle           bool
	HasProhibited      bool
	ProhibitedRatio    float64
	QuickCheckPassed   bool
	RecommendedForScan bool
}