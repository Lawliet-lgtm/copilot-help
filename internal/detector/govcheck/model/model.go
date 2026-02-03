// Package model
package model

import (
	"time"
)

// DetectResult 检测结果（与上游 model.DetectResult 兼容）
// 如果上游已有此定义，可直接引用上游的 model 包
type DetectResult struct {
	// 基础判定
	IsSecret bool   `json:"is_secret"` // 是否涉密（对于公文检测：是否为公文）
	Module   string `json:"module"`    // 检测模块名称

	// 置信度和阈值
	Confidence float64 `json:"confidence"` // 置信度 (0-1)
	Threshold  float64 `json:"threshold"`  // 判定阈值

	// 文件信息
	FilePath string `json:"file_path"` // 文件路径
	FileType string `json:"file_type"` // 文件类型
	FileSize int64  `json:"file_size"` // 文件大小

	// 检测详情
	Details *DetectDetails `json:"details,omitempty"` // 详细信息

	// 处理信息
	ProcessTime time.Duration `json:"process_time"` // 处理耗时
	Error       string        `json:"error,omitempty"` // 错误信息

	// 时间戳
	DetectedAt time.Time `json:"detected_at"` // 检测时间
}

// DetectDetails 检测详情
type DetectDetails struct {
	// 公文特征
	DocNumber      string   `json:"doc_number,omitempty"`      // 发文字号
	Title          string   `json:"title,omitempty"`           // 公文标题
	TitleType      string   `json:"title_type,omitempty"`      // 标题文种
	Recipient      string   `json:"recipient,omitempty"`       // 主送机关
	Date           string   `json:"date,omitempty"`            // 成文日期
	Organizations  []string `json:"organizations,omitempty"`   // 识别的机关
	SerialNumber   string   `json:"serial_number,omitempty"`   // 份号
	SecretLevel    string   `json:"secret_level,omitempty"`    // 密级标志
	UrgencyLevel   string   `json:"urgency_level,omitempty"`   // 紧急程度

	// 版式特征
	HasSeal      bool `json:"has_seal"`       // 有印章
	HasRedHeader bool `json:"has_red_header"` // 有红头
	HasCC        bool `json:"has_cc"`         // 有抄送
	HasPrintInfo bool `json:"has_print_info"` // 有印发信息

	// 评分详情
	TextScore   float64       `json:"text_score"`   // 文本特征得分
	StyleScore  float64       `json:"style_score"`  // 版式特征得分
	ScoreItems  []ScoreItem   `json:"score_items,omitempty"` // 得分明细

	// 原始文本（可选，用于调试）
	ExtractedText string `json:"extracted_text,omitempty"`
}

// ScoreItem 得分项
type ScoreItem struct {
	Name   string  `json:"name"`   // 特征名称
	Score  float64 `json:"score"`  // 得分
	Reason string  `json:"reason"` // 原因
}

// NewDetectResult 创建检测结果
func NewDetectResult(filePath string) *DetectResult {
	return &DetectResult{
		FilePath:   filePath,
		Module:     "govcheck",
		DetectedAt: time.Now(),
		Details:    &DetectDetails{},
	}
}

// MarkAsOfficial 标记为公文
func (r *DetectResult) MarkAsOfficial(confidence float64) {
	r.IsSecret = true
	r.Confidence = confidence
}

// MarkAsNonOfficial 标记为非公文
func (r *DetectResult) MarkAsNonOfficial(confidence float64) {
	r.IsSecret = false
	r.Confidence = confidence
}

// SetError 设置错误
func (r *DetectResult) SetError(err error) {
	if err != nil {
		r.Error = err.Error()
	}
}

// Summary 返回摘要信息
func (r *DetectResult) Summary() string {
	if r.IsSecret {
		return "检测结果: 公文文件"
	}
	return "检测结果: 非公文文件"
}