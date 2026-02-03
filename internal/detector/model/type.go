package model

// SecretLevel 定义密级枚举 (全系统通用)
type SecretLevel string

const (
	LevelTopSecret    SecretLevel = "绝密"
	LevelSecret       SecretLevel = "机密"
	LevelConfidential SecretLevel = "秘密"
)

// DetectResult 是所有子检测模块 (SubDetector) 返回给上游调度器 (Manager) 的统一结果
type DetectResult struct {
	IsSecret    bool        `json:"is_secret"`
	Level       SecretLevel `json:"level"`
	MatchedText string      `json:"matched_text"` // 命中的关键词或证据
	Source      string      `json:"source"`       // 来源模块 (例如: "SecretMarker", "Keywords")
}