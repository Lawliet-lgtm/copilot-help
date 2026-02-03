package model

import (
	"path/filepath"
	"strings"
)

// ==========================================
// 1. 常量定义 (避免魔术字符串)
// ==========================================

// DetectType 定义检测子模块的类型
type DetectType string

const (
	TypeHash            DetectType = "FILE_HASH"        // 文件哈希检测
	TypeElectronicLabel DetectType = "ELECTRONIC_LABEL" // 电子密级标志 (元数据)
	TypeSecretMark      DetectType = "SECRET_MARK"      // 可见密级标志 (水印/文字)
	TypeOfficialLayout  DetectType = "OFFICIAL_LAYOUT"  // 公文版式
	TypeKeyword         DetectType = "KEYWORD"          // 敏感关键词 (兼容你原有的)
)

// ==========================================
// 2. 输入协议：检测上下文
// ==========================================

// DetectContext 包含检测所需的所有信息
// 这是一个"被动"的数据容器，由外部调用者填充
type DetectContext struct {
	// 基础文件信息
	FilePath string // 文件绝对路径 (用于读取文件流、计算哈希)
	FileExt  string // 文件后缀 (小写，如 .docx)，用于快速过滤策略

	// 内容信息 (由 extractous-go 提取后填入)
	// 如果尚未提取，此字段为空字符串
	ExtractedText string

	// 预留扩展：如果未来需要图片 OCR 的坐标信息，可在此添加
	// OCRResult []OCRItem
}

// NewDetectContext 辅助构造函数，自动处理后缀大小写
func NewDetectContext(path string, text string) *DetectContext {
	return &DetectContext{
		FilePath:      path,
		FileExt:       strings.ToLower(filepath.Ext(path)),
		ExtractedText: text,
	}
}

