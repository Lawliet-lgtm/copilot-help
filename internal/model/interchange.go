package model

// SecretLevel 定义密级枚举 (全系统通用)
type SecretLevel int

const (
	LevelUnknown SecretLevel = iota
	LevelTopSecret           // 绝密
	LevelSecret              // 机密
	LevelConfidential        // 秘密
	LevelInternal            // 内部公开/工作秘密
)

// SubDetectResult 是子模块返回给 Manager 的精简结果
// 子模块只负责填它能填的，通用的由 Manager 填
type SubDetectResult struct {
	IsSecret      bool
	SecretLevel   SecretLevel
	RuleID        int64  // 命中的规则ID (如果有)
	RuleDesc      string // 规则描述
	MatchedText   string // 命中的关键词 (对应 HighlightText)
	ContextText   string // 上下文 (对应 FileDesc)
	AlertType     int    // 告警类型映射
}