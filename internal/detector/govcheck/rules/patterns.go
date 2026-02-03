package rules

import (
	"regexp"
	"strings"
)

// Pattern 表示一个正则模式
type Pattern struct {
	Name        string         // 模式名称
	Regex       *regexp.Regexp // 编译后的正则表达式
	Description string         // 描述
	Weight      float64        // 权重 (用于评分)
	Examples    []string       // 示例
}

// Match 检查文本是否匹配该模式
func (p *Pattern) Match(text string) bool {
	return p.Regex.MatchString(text)
}

// FindString 查找第一个匹配的字符串
func (p *Pattern) FindString(text string) string {
	return p.Regex.FindString(text)
}

// FindAllString 查找所有匹配的字符串
func (p *Pattern) FindAllString(text string, n int) []string {
	return p.Regex.FindAllString(text, n)
}

// ============================================================
// 发文字号模式
// 格式: XX〔2024〕1号 或 XX[2024]1号 或 XX（2024）1号
// ============================================================

// DocNumberPattern 发文字号正则模式
var DocNumberPattern = &Pattern{
	Name: "发文字号",
	Regex: regexp.MustCompile(
		`[〔\[\(（]` +
			`\s*` +
			`(19|20)\d{2}` +
			`\s*` +
			`[〕\]\)）]` +
			`\s*` +
			`\d{1,4}` +
			`\s*号`),
	Description: "发文字号，如：国发〔2024〕1号",
	Weight:      0.20,
	Examples: []string{
		"国发〔2024〕1号",
		"京政发[2023]15号",
		"教办函（2024）123号",
	},
}

// DocNumberFullPattern 完整发文字号模式（包含发文机关代字）
var DocNumberFullPattern = &Pattern{
	Name: "完整发文字号",
	Regex: regexp.MustCompile(
		`[\p{Han}]{1,10}` +
			`[发办函令〔\[\(（]` +
			`.*?` +
			`[〔\[\(（]` +
			`\s*(19|20)\d{2}\s*` +
			`[〕\]\)）]` +
			`\s*\d{1,4}\s*号`),
	Description: "完整发文字号，包含机关代字",
	Weight:      0.25,
	Examples: []string{
		"国务院令〔2024〕1号",
		"京政办发[2023]15号",
		"教育部办公厅函（2024）123号",
	},
}

// ============================================================
// 公文标题模式
// 格式: 关于XXX的通知/决定/意见/报告/请示/批复/函/公告/通报
// ============================================================

// TitlePattern 公文标题正则模式
var TitlePattern = &Pattern{
	Name: "公文标题",
	Regex: regexp.MustCompile(
		`关于.{2,100}的(通知|决定|意见|报告|请示|批复|函|公告|通报|命令|议案|纪要|办法|规定|条例|细则|方案|计划|总结|规划)`),
	Description: "公文标题格式",
	Weight:      0.20,
	Examples: []string{
		"关于印发xxx的通知",
		"关于做好xxx工作的意见",
		"关于xxx问题的请示",
	},
}

// TitleTypePattern 标题文种提取模式
var TitleTypePattern = &Pattern{
	Name: "标题文种",
	Regex: regexp.MustCompile(
		`(通知|决定|意见|报告|请示|批复|函|公告|通报|命令|议案|纪要|办法|规定|条例|细则|方案|计划|总结|规划)$`),
	Description: "提取标题末尾的文种",
	Weight:      0.10,
}

// ============================================================
// 密级标志模式
// ============================================================

// SecretLevelPattern 密级标志正则模式
var SecretLevelPattern = &Pattern{
	Name:        "密级标志",
	Regex:       regexp.MustCompile(`(绝密|机密|秘密)[★☆*]?\s*(\d{1,3}年)?`),
	Description: "公文密级标志",
	Weight:      0.15,
	Examples: []string{
		"绝密★",
		"机密★30年",
		"秘密",
	},
}

// ============================================================
// 紧急程度模式
// ============================================================

// UrgencyLevelPattern 紧急程度正则模式
var UrgencyLevelPattern = &Pattern{
	Name:        "紧急程度",
	Regex:       regexp.MustCompile(`(特急|加急|平急|特提|限时)`),
	Description: "公文紧急程度标志",
	Weight:      0.10,
	Examples: []string{
		"特急",
		"加急",
	},
}

// ============================================================
// 签发人模式
// ============================================================

// IssuerPattern 签发人正则模式
var IssuerPattern = &Pattern{
	Name:        "签发人",
	Regex:       regexp.MustCompile(`签\s*发\s*人\s*[:：]?\s*[\p{Han}]{2,4}`),
	Description: "签发人标志",
	Weight:      0.15,
	Examples: []string{
		"签发人：张三",
		"签发人:李四",
	},
}

// MultiIssuerPattern 多签发人模式
var MultiIssuerPattern = &Pattern{
	Name:        "多签发人",
	Regex:       regexp.MustCompile(`签\s*发\s*人\s*[:：]?\s*([\p{Han}]{2,4}\s*)+`),
	Description: "多个签发人",
	Weight:      0.15,
}

// ============================================================
// 主送机��模式
// ============================================================

// MainSendPattern 主送机关正则模式
var MainSendPattern = &Pattern{
	Name:        "主送机关",
	Regex:       regexp.MustCompile(`(各省|各市|各县|各区|各部门|各单位|各有关)[\p{Han}、,，\s]*[:：]?`),
	Description: "主送机关标志",
	Weight:      0.10,
	Examples: []string{
		"各省、自治区、直辖市人民政府：",
		"各有关部门：",
	},
}

// ============================================================
// 成文日期模式
// ============================================================

// IssueDatePattern 成文日期正则模式（中文格式）
var IssueDatePattern = &Pattern{
	Name:        "成文日期",
	Regex:       regexp.MustCompile(`(19|20)\d{2}\s*年\s*(0?[1-9]|1[0-2])\s*月\s*(0?[1-9]|[12]\d|3[01])\s*日`),
	Description: "成文日期（中文格式）",
	Weight:      0.15,
	Examples: []string{
		"2024年1月1日",
		"2023年12月31日",
	},
}

// IssueDatePattern2 成文日期正则模式（数字格式）
var IssueDatePattern2 = &Pattern{
	Name:        "成文日期(数字)",
	Regex:       regexp.MustCompile(`(19|20)\d{2}[-/.年](0?[1-9]|1[0-2])[-/.月](0?[1-9]|[12]\d|3[01])日?`),
	Description: "成文日期（数字格式）",
	Weight:      0.10,
}

// ============================================================
// 抄送模式
// ============================================================

// CopyToPattern 抄送正则模式
var CopyToPattern = &Pattern{
	Name:        "抄送",
	Regex:       regexp.MustCompile(`抄\s*送\s*[:：]\s*.{2,200}[。.]?`),
	Description: "抄送机关",
	Weight:      0.10,
	Examples: []string{
		"抄送：省委办公厅、省政府办公厅。",
	},
}

// ============================================================
// 印发信息模式
// ============================================================

// PrintInfoPattern 印发信息正则模式
var PrintInfoPattern = &Pattern{
	Name:        "印发信息",
	Regex:       regexp.MustCompile(`[\p{Han}]{2,30}\s*(19|20)\d{2}年\d{1,2}月\d{1,2}日\s*(印发|印制|发布)`),
	Description: "印发机关和日期",
	Weight:      0.10,
	Examples: []string{
		"国务院办公厅 2024年1月1日印发",
	},
}

// ============================================================
// 附件模式
// ============================================================

// AttachmentPattern 附件说明正则模式
var AttachmentPattern = &Pattern{
	Name:        "附件",
	Regex:       regexp.MustCompile(`附\s*件\s*[:：]?\s*(\d+\s*[.、]\s*)?.{2,100}`),
	Description: "附件说明",
	Weight:      0.08,
	Examples: []string{
		"附件：1. xxxx",
		"附件：xxxx实施方案",
	},
}

// ============================================================
// 联系人/联系方式模式
// ============================================================

// ContactPattern 联系人正则模式
var ContactPattern = &Pattern{
	Name:        "联系人",
	Regex:       regexp.MustCompile(`联\s*系\s*人\s*[:：]\s*[\p{Han}]{2,4}`),
	Description: "联系人信息",
	Weight:      0.05,
}

// PhonePattern 联系电话正则模式
var PhonePattern = &Pattern{
	Name:        "联系电话",
	Regex:       regexp.MustCompile(`(联系电话|电\s*话|联系方式)\s*[:：]?\s*[\d\-\(\)\s]{7,20}`),
	Description: "联系电话",
	Weight:      0.05,
}

// ============================================================
// 落款/署名模式
// ============================================================

// SignaturePattern 落款署名模式
var SignaturePattern = &Pattern{
	Name:        "落款署名",
	Regex:       regexp.MustCompile(`[\p{Han}]{4,30}\s*[\(（]?\s*(盖\s*章|印)?\s*[\)）]?`),
	Description: "落款机关署名",
	Weight:      0.08,
}

// ============================================================
// 模式集合
// ============================================================

// AllPatterns 所有模式的集合
var AllPatterns = []*Pattern{
	DocNumberPattern,
	DocNumberFullPattern,
	TitlePattern,
	TitleTypePattern,
	SecretLevelPattern,
	UrgencyLevelPattern,
	IssuerPattern,
	MultiIssuerPattern,
	MainSendPattern,
	IssueDatePattern,
	IssueDatePattern2,
	CopyToPattern,
	PrintInfoPattern,
	AttachmentPattern,
	ContactPattern,
	PhonePattern,
	SignaturePattern,
}

// ============================================================
// 辅助函数
// ============================================================

// MatchPatterns 检查文本匹配哪些模式
func MatchPatterns(text string, patterns []*Pattern) []*PatternMatch {
	var matches []*PatternMatch

	for _, p := range patterns {
		if found := p.FindString(text); found != "" {
			matches = append(matches, &PatternMatch{
				Pattern: p,
				Matched: found,
			})
		}
	}

	return matches
}

// PatternMatch 模式匹配结果
type PatternMatch struct {
	Pattern *Pattern // 匹配的模式
	Matched string   // 匹配到的内容
}

// NormalizeText 规范化文本（用于匹配前预处理）
func NormalizeText(text string) string {
	// 统一换行符
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// 统一中文冒号为英文冒号
	text = strings.ReplaceAll(text, "：", ":")

	// 统一中文逗号为英文逗号
	text = strings.ReplaceAll(text, "，", ",")

	// 统一中文句号为英文句号
	text = strings.ReplaceAll(text, "。", ".")

	// 统一中文分号为英文分号
	text = strings.ReplaceAll(text, "；", ";")

	// 统一中文括号为英文括号
	text = strings.ReplaceAll(text, "（", "(")
	text = strings.ReplaceAll(text, "）", ")")

	// 统一中文方括号为英文方括号
	text = strings.ReplaceAll(text, "【", "[")
	text = strings.ReplaceAll(text, "】", "]")

	return text
}

// ExtractDocNumber 提取发文字号
func ExtractDocNumber(text string) string {
	// 先尝试完整模式
	if match := DocNumberFullPattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	// 再尝试简单模式
	if match := DocNumberPattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

// ExtractTitle 提取公文标题
func ExtractTitle(text string) string {
	if match := TitlePattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

// ExtractTitleType 提取标题文种
func ExtractTitleType(title string) string {
	if match := TitleTypePattern.FindString(title); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

// ExtractIssueDate 提取成文日期
func ExtractIssueDate(text string) string {
	if match := IssueDatePattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	if match := IssueDatePattern2.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

// ExtractSecretLevel 提取密级
func ExtractSecretLevel(text string) string {
	if match := SecretLevelPattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

// ExtractUrgencyLevel 提取紧急程度
func ExtractUrgencyLevel(text string) string {
	if match := UrgencyLevelPattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

// ExtractIssuer 提取签发人
func ExtractIssuer(text string) string {
	match := IssuerPattern.FindString(text)
	if match == "" {
		return ""
	}

	// 去掉"签发人："前缀
	match = strings.TrimSpace(match)

	// 尝试用英文冒号分割
	if idx := strings.Index(match, ":"); idx != -1 {
		return strings.TrimSpace(match[idx+1:])
	}

	// 尝试用中文冒号分割
	if idx := strings.Index(match, "："); idx != -1 {
		return strings.TrimSpace(match[idx+3:]) // 中文冒号占3字节
	}

	return ""
}

// ExtractCopyTo 提取抄送信息
func ExtractCopyTo(text string) string {
	if match := CopyToPattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

// ExtractMainSend 提取主送机关
func ExtractMainSend(text string) string {
	if match := MainSendPattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

// ExtractAttachment 提取附件信息
func ExtractAttachment(text string) string {
	if match := AttachmentPattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

// ExtractPrintInfo 提取印发信息
func ExtractPrintInfo(text string) string {
	if match := PrintInfoPattern.FindString(text); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}