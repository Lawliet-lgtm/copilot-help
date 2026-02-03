// Package model
package model

// SecretLevel 定义密级枚举
type SecretLevel string

const (
	LevelTopSecret    SecretLevel = "绝密"
	LevelSecret       SecretLevel = "机密"
	LevelConfidential SecretLevel = "秘密"
	LevelNone         SecretLevel = ""
)

// FileType 定义文件类型枚举
type FileType int

const (
	TypeUnknown FileType = iota
	TypeOffice           // docx, xlsx, pptx
	TypePDF
	TypeOFD
	TypeRTF
	TypeText
	TypeImage
	TypeBinary
)

// ScanResult 是底层解析器 (Parser) 返回的原始结果
// [关键] 保留这个结构体，确保 parser 包下的代码不飘红
type ScanResult struct {
	IsSecret    bool        `json:"is_secret"`
	Level       SecretLevel `json:"level"`
	MatchedText string      `json:"matched_text"`
	FilePath    string      `json:"file_path"` // 解析器有时会填充这个字段
}
