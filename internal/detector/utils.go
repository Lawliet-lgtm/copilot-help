package detector

import (
	"unicode/utf8"
)

// limitRunes 安全截取字符串的前 n 个字符 (处理 UTF-8 中文)
// 这是一个包内共享的辅助函数
func limitRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n])
}
