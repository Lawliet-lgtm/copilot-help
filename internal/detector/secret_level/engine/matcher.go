package engine

import (
	"regexp"
	"linuxFileWatcher/internal/detector/secret_level/model"
)

var (
	// [严格标准] 给 Office/PDF/Text 使用
	// 必须包含星号，符合 GB/T 9704
	// 例如: "绝密★", "机密 ★", "秘密*"
	strictPattern = regexp.MustCompile(`(绝密|机密|秘密)\s*[★\*]\s*(\d{1,2}年|长期)?`)

	// [宽松标准] 仅给 OCR 使用
	// 允许没有星号，或者星号被识别成了其他怪字符
	// 匹配逻辑：只要出现了密级关键词，就视为命中。
	// 这能有效解决 OCR 将 "★" 识别为空格、"太"、"大" 等导致漏报的问题。
	ocrPattern = regexp.MustCompile(`(绝密|机密|秘密)`)
)

// MatchContent 严格匹配 (用于 Office, Text, PDF)
// 只有匹配到 "密级 + 星号" 才算涉密，防止误报。
func MatchContent(content string) (bool, model.SecretLevel, string) {
	return matchWithPattern(content, strictPattern)
}

// MatchOCRContent 宽松匹配 (仅用于 OCR)
// 只要匹配到 "密级" 关键词即算涉密，防止漏报。
func MatchOCRContent(content string) (bool, model.SecretLevel, string) {
	return matchWithPattern(content, ocrPattern)
}

// 内部通用匹配逻辑
func matchWithPattern(content string, pattern *regexp.Regexp) (bool, model.SecretLevel, string) {
	matches := pattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		// matches[0] 是全匹配字符串
		// matches[1] 是第一个括号捕获的内容 (即密级关键词)
		levelStr := matches[1]
		
		var level model.SecretLevel
		switch levelStr {
		case "绝密":
			level = model.LevelTopSecret
		case "机密":
			level = model.LevelSecret
		case "秘密":
			level = model.LevelConfidential
		default:
			return false, "", ""
		}
		return true, level, matches[0]
	}
	return false, "", ""
}