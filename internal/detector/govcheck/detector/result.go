package detector

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DetectionResult 表示单个文件的检测结果
type DetectionResult struct {
	// 基本信息
	FilePath string `json:"file_path"` // 文件绝对路径
	FileName string `json:"file_name"` // 文件名
	FileSize int64  `json:"file_size"` // 文件大小(字节)
	FileType string `json:"file_type"` // 识别出的文件类型 (如 docx, pdf)

	// 检测结果
	IsOfficialDoc bool    `json:"is_official_doc"` // 是否判定为公文
	Confidence    float64 `json:"confidence"`      // 置信度 (0-1)
	Threshold     float64 `json:"threshold"`       // 使用的判定阈值

	// 分项得分
	TextScore  float64 `json:"text_score"`  // 文本特征得分
	StyleScore float64 `json:"style_score"` // 版式特征得分

	// 特征匹配详情
	Features *FeatureResult `json:"features,omitempty"` // 匹配到的公文特征

	// 处理信息
	ProcessTime time.Duration `json:"process_time_ns"` // 处理耗时
	Error       string        `json:"error,omitempty"` // 错误信息(如有)
	Success     bool          `json:"success"`         // 是否处理成功
}

// FeatureResult 表示公文特征检测结果
type FeatureResult struct {
	// 版头特征
	HasCopyNumber   bool   `json:"has_copy_number"`             // 是否有份号 (新增)
	CopyNumber      string `json:"copy_number,omitempty"`       // 份号内容 (新增)
	HasDocNumber    bool   `json:"has_doc_number"`              // 是否有发文字号
	DocNumber       string `json:"doc_number,omitempty"`        // 发文字号内容
	HasSecretLevel  bool   `json:"has_secret_level"`            // 是否有密级标志
	SecretLevel     string `json:"secret_level,omitempty"`      // 密级内容
	HasUrgencyLevel bool   `json:"has_urgency_level"`           // 是否有紧急程度
	UrgencyLevel    string `json:"urgency_level,omitempty"`     // 紧急程度内容
	HasIssuer       bool   `json:"has_issuer"`                  // 是否有签发人
	Issuer          string `json:"issuer,omitempty"`            // 签发人内容

	// 主体特征
	HasTitle       bool   `json:"has_title"`                 // 是否有公文标题
	Title          string `json:"title,omitempty"`           // 标题内容
	TitleType      string `json:"title_type,omitempty"`      // 标题类型(通知/决定/意见等)
	HasMainSend    bool   `json:"has_main_send"`             // 是否有主送机关
	MainSend       string `json:"main_send,omitempty"`       // 主送机关内容
	HasAttachment  bool   `json:"has_attachment"`            // 是否有附件说明
	AttachmentInfo string `json:"attachment_info,omitempty"` // 附件说明内容

	// 版记特征
	HasIssueDate bool   `json:"has_issue_date"`           // 是否有成文日期
	IssueDate    string `json:"issue_date,omitempty"`     // 成文日期内容
	HasSeal      bool   `json:"has_seal"`                 // 是否有印章(图片检测)
	HasCopyTo    bool   `json:"has_copy_to"`              // 是否有抄送
	CopyTo       string `json:"copy_to,omitempty"`        // 抄送内容
	HasPrintInfo bool   `json:"has_print_info"`           // 是否有印发信息
	PrintInfo    string `json:"print_info,omitempty"`     // 印发信息内容

	// 机关特征
	HasOrgName   bool     `json:"has_org_name"`             // 是否包含机关名称
	OrgNames     []string `json:"org_names,omitempty"`      // 识别到的机关名称
	HasRedHeader bool     `json:"has_red_header"`           // 是否有红头标志(图片检测)

	// 版式特征
	StyleFeatures *StyleFeatureResult `json:"style_features,omitempty"` // 版式特征详情

	// 综合得分明细
	ScoreDetails map[string]float64 `json:"score_details,omitempty"` // 各项得分明细
}

// StyleFeatureResult 版式特征检测结果
type StyleFeatureResult struct {
	// 颜色特征
	HasRedText   bool     `json:"has_red_text"`             // 是否有红色文本
	HasRedHeader bool     `json:"has_red_header"`           // 是否有红头
	RedTextCount int      `json:"red_text_count,omitempty"` // 红色文本数量
	RedSamples   []string `json:"red_samples,omitempty"`    // 红色文本示例

	// 字体特征
	HasOfficialFonts bool   `json:"has_official_fonts"`          // 是否使用公文字体
	TitleFontMatch   bool   `json:"title_font_match"`            // 标题字号是否符合
	BodyFontMatch    bool   `json:"body_font_match"`             // 正文字号是否符合
	MainFontName     string `json:"main_font_name,omitempty"`    // 主要字体名称
	MainFontSize     string `json:"main_font_size,omitempty"`    // 主要字号

	// 页面特征
	IsA4Paper   bool   `json:"is_a4_paper"`              // 是否A4纸
	MarginMatch bool   `json:"margin_match"`             // 页边距是否符合
	PageSize    string `json:"page_size,omitempty"`      // 页面尺寸描述

	// 段落特征
	HasCenteredTitle bool `json:"has_centered_title"` // 是否有居中标题
	LineSpacingMatch bool `json:"line_spacing_match"` // 行距是否符合

	// 印章特征
	HasSealImage  bool   `json:"has_seal_image"`            // 是否有印章图片
	SealImageHint string `json:"seal_image_hint,omitempty"` // 印章提示

	// 综合
	StyleScore   float64  `json:"style_score"`             // 版式得分
	StyleReasons []string `json:"style_reasons,omitempty"` // 判断理由
}

// NewDetectionResult 创建一个新的检测结果
func NewDetectionResult(filePath, fileName string, fileSize int64) *DetectionResult {
	return &DetectionResult{
		FilePath: filePath,
		FileName: fileName,
		FileSize: fileSize,
		Features: &FeatureResult{
			ScoreDetails:  make(map[string]float64),
			StyleFeatures: &StyleFeatureResult{},
		},
	}
}

// SetError 设置错误信息
func (r *DetectionResult) SetError(err error) {
	r.Success = false
	if err != nil {
		r.Error = err.Error()
	}
}

// SetSuccess 设置处理成功
func (r *DetectionResult) SetSuccess() {
	r.Success = true
	r.Error = ""
}

// ToJSON 将结果转换为JSON字符串
func (r *DetectionResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Summary 返回结果摘要字符串
func (r *DetectionResult) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("文件: %s\n", r.FileName))
	sb.WriteString(fmt.Sprintf("类型: %s\n", r.FileType))
	sb.WriteString(fmt.Sprintf("大小: %s\n", formatFileSize(r.FileSize)))

	if !r.Success {
		sb.WriteString(fmt.Sprintf("状态: 处理失败\n"))
		sb.WriteString(fmt.Sprintf("错误: %s\n", r.Error))
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("状态: 处理成功\n"))
	sb.WriteString(fmt.Sprintf("耗时: %v\n", r.ProcessTime))
	sb.WriteString(fmt.Sprintf("置信度: %.2f%%\n", r.Confidence*100))
	sb.WriteString(fmt.Sprintf("阈值: %.2f%%\n", r.Threshold*100))

	if r.IsOfficialDoc {
		sb.WriteString("判定: ✓ 是公文\n")
	} else {
		sb.WriteString("判定: ✗ 不是公文\n")
	}

	return sb.String()
}

// VerboseSummary 返回详细结果摘要
func (r *DetectionResult) VerboseSummary() string {
	var sb strings.Builder

	sb.WriteString(r.Summary())

	if r.Features == nil || !r.Success {
		return sb.String()
	}

	// 分项得分
	sb.WriteString(fmt.Sprintf("\n分项得分:\n"))
	sb.WriteString(fmt.Sprintf("  文本特征: %.2f%%\n", r.TextScore*100))
	sb.WriteString(fmt.Sprintf("  版式特征: %.2f%%\n", r.StyleScore*100))

	sb.WriteString("\n特征检测详情:\n")
	sb.WriteString("─────────────────────────────\n")

	// 版头特征
	sb.WriteString("[版头特征]\n")
	sb.WriteString(fmt.Sprintf("  份号:     %s\n", formatBoolWithValue(r.Features.HasCopyNumber, r.Features.CopyNumber)))
	sb.WriteString(fmt.Sprintf("  发文字号: %s\n", formatBoolWithValue(r.Features.HasDocNumber, r.Features.DocNumber)))
	sb.WriteString(fmt.Sprintf("  密级标志: %s\n", formatBoolWithValue(r.Features.HasSecretLevel, r.Features.SecretLevel)))
	sb.WriteString(fmt.Sprintf("  紧急程度: %s\n", formatBoolWithValue(r.Features.HasUrgencyLevel, r.Features.UrgencyLevel)))
	sb.WriteString(fmt.Sprintf("  签发人:   %s\n", formatBoolWithValue(r.Features.HasIssuer, r.Features.Issuer)))

	// 主体特征
	sb.WriteString("[主体特征]\n")
	sb.WriteString(fmt.Sprintf("  公文标题: %s\n", formatBoolWithValue(r.Features.HasTitle, truncateString(r.Features.Title, 30))))
	sb.WriteString(fmt.Sprintf("  标题类型: %s\n", valueOrNA(r.Features.TitleType)))
	sb.WriteString(fmt.Sprintf("  主送机关: %s\n", formatBoolWithValue(r.Features.HasMainSend, truncateString(r.Features.MainSend, 30))))
	sb.WriteString(fmt.Sprintf("  附件说明: %s\n", formatBool(r.Features.HasAttachment)))

	// 版记特征
	sb.WriteString("[版记特征]\n")
	sb.WriteString(fmt.Sprintf("  成文日期: %s\n", formatBoolWithValue(r.Features.HasIssueDate, r.Features.IssueDate)))
	sb.WriteString(fmt.Sprintf("  印章:     %s\n", formatBool(r.Features.HasSeal)))
	sb.WriteString(fmt.Sprintf("  抄送:     %s\n", formatBool(r.Features.HasCopyTo)))
	sb.WriteString(fmt.Sprintf("  印发信息: %s\n", formatBool(r.Features.HasPrintInfo)))

	// 机关特征
	sb.WriteString("[机关特征]\n")
	sb.WriteString(fmt.Sprintf("  机关名称: %s\n", formatBool(r.Features.HasOrgName)))
	if len(r.Features.OrgNames) > 0 {
		sb.WriteString(fmt.Sprintf("  识别机关: %v\n", r.Features.OrgNames))
	}
	sb.WriteString(fmt.Sprintf("  红头标志: %s\n", formatBool(r.Features.HasRedHeader)))

	// 版式特征
	if r.Features.StyleFeatures != nil {
		sf := r.Features.StyleFeatures
		sb.WriteString("[版式特征]\n")
		sb.WriteString(fmt.Sprintf("  红色文本: %s\n", formatBool(sf.HasRedText)))
		sb.WriteString(fmt.Sprintf("  红头版式: %s\n", formatBool(sf.HasRedHeader)))
		sb.WriteString(fmt.Sprintf("  公文字体: %s\n", formatBool(sf.HasOfficialFonts)))
		sb.WriteString(fmt.Sprintf("  标题字号: %s\n", formatBool(sf.TitleFontMatch)))
		sb.WriteString(fmt.Sprintf("  正文字号: %s\n", formatBool(sf.BodyFontMatch)))
		sb.WriteString(fmt.Sprintf("  A4纸张:   %s\n", formatBool(sf.IsA4Paper)))
		sb.WriteString(fmt.Sprintf("  页边距:   %s\n", formatBool(sf.MarginMatch)))
		sb.WriteString(fmt.Sprintf("  居中标题: %s\n", formatBool(sf.HasCenteredTitle)))
		sb.WriteString(fmt.Sprintf("  印章图片: %s\n", formatBool(sf.HasSealImage)))
		sb.WriteString(fmt.Sprintf("  版式得分: %.2f%%\n", sf.StyleScore*100))

		if len(sf.StyleReasons) > 0 {
			sb.WriteString("  版式判断:\n")
			for _, reason := range sf.StyleReasons {
				sb.WriteString(fmt.Sprintf("    • %s\n", reason))
			}
		}
	}

	// 得分明细
	if len(r.Features.ScoreDetails) > 0 {
		sb.WriteString("\n[得分明细]\n")
		for name, score := range r.Features.ScoreDetails {
			if score > 0 {
				sb.WriteString(fmt.Sprintf("  [+] %s: +%.2f\n", name, score))
			} else {
				sb.WriteString(fmt.Sprintf("  [-] %s: %.2f\n", name, score))
			}
		}
	}

	return sb.String()
}

// ============================================================
// 辅助函数
// ============================================================

func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func formatBool(b bool) string {
	if b {
		return "✓ 是"
	}
	return "✗ 否"
}

func formatBoolWithValue(b bool, value string) string {
	if !b {
		return "✗ 未检测到"
	}
	if value == "" {
		return "✓ 已检测到"
	}
	return fmt.Sprintf("✓ %s", value)
}

func valueOrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}