package detector

import (
	"os"
	"testing"
)

// ============================================================
// DetectionResult 测试
// ============================================================

func TestNewDetectionResult(t *testing.T) {
	result := NewDetectionResult("/path/to/file.docx", "file.docx", 1024)

	if result == nil {
		t.Fatal("NewDetectionResult 返回 nil")
	}

	if result.FilePath != "/path/to/file.docx" {
		t.Errorf("FilePath = %q, want %q", result.FilePath, "/path/to/file.docx")
	}

	if result.FileName != "file.docx" {
		t.Errorf("FileName = %q, want %q", result.FileName, "file.docx")
	}

	if result.FileSize != 1024 {
		t.Errorf("FileSize = %d, want %d", result.FileSize, 1024)
	}

	if result.Features == nil {
		t.Error("Features 不应为 nil")
	}

	if result.Features.ScoreDetails == nil {
		t.Error("ScoreDetails 不应为 nil")
	}

	if result.Features.StyleFeatures == nil {
		t.Error("StyleFeatures 不应为 nil")
	}
}

func TestDetectionResult_SetError(t *testing.T) {
	result := NewDetectionResult("/path/to/file.docx", "file.docx", 1024)

	err := os.ErrNotExist
	result.SetError(err)

	if result.Success {
		t.Error("SetError 后 Success 应为 false")
	}

	if result.Error == "" {
		t.Error("Error 不应为空")
	}
}

func TestDetectionResult_SetSuccess(t *testing.T) {
	result := NewDetectionResult("/path/to/file.docx", "file.docx", 1024)

	// 先设置错误
	result.SetError(os.ErrNotExist)

	// 再设置成功
	result.SetSuccess()

	if !result.Success {
		t.Error("SetSuccess 后 Success 应为 true")
	}

	if result.Error != "" {
		t.Error("SetSuccess 后 Error 应为空")
	}
}

func TestDetectionResult_ToJSON(t *testing.T) {
	result := NewDetectionResult("/path/to/file.docx", "file.docx", 1024)
	result.IsOfficialDoc = true
	result.Confidence = 0.75
	result.SetSuccess()

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON 失败: %v", err)
	}

	if jsonStr == "" {
		t.Error("JSON 字符串不应为空")
	}

	// 检查包含关键字段
	if !containsString(jsonStr, "file_path") {
		t.Error("JSON 应包含 file_path")
	}

	if !containsString(jsonStr, "is_official_doc") {
		t.Error("JSON 应包含 is_official_doc")
	}

	if !containsString(jsonStr, "confidence") {
		t.Error("JSON 应包含 confidence")
	}
}

func TestDetectionResult_Summary(t *testing.T) {
	result := NewDetectionResult("/path/to/file.docx", "file.docx", 1024)
	result.FileType = "docx"
	result.IsOfficialDoc = true
	result.Confidence = 0.75
	result.Threshold = 0.6
	result.SetSuccess()

	summary := result.Summary()

	if summary == "" {
		t.Error("Summary 不应为空")
	}

	// 检查包含关键信息
	if !containsString(summary, "file.docx") {
		t.Error("Summary 应包含文件名")
	}

	if !containsString(summary, "docx") {
		t.Error("Summary 应包含文件类型")
	}

	if !containsString(summary, "是公文") {
		t.Error("Summary 应包含判定结果")
	}
}

func TestDetectionResult_Summary_Failed(t *testing.T) {
	result := NewDetectionResult("/path/to/file.docx", "file.docx", 1024)
	result.FileType = "docx"
	result.SetError(os.ErrNotExist)

	summary := result.Summary()

	if !containsString(summary, "处理失败") {
		t.Error("失败情况 Summary 应包含'处理失败'")
	}
}

func TestDetectionResult_VerboseSummary(t *testing.T) {
	result := NewDetectionResult("/path/to/file.docx", "file.docx", 1024)
	result.FileType = "docx"
	result.IsOfficialDoc = true
	result.Confidence = 0.75
	result.Threshold = 0.6
	result.TextScore = 0.5
	result.StyleScore = 0.8
	result.SetSuccess()

	// 设置特征
	result.Features.HasDocNumber = true
	result.Features.DocNumber = "X府发〔2024〕1号"
	result.Features.HasTitle = true
	result.Features.Title = "关于加强工作的通知"
	result.Features.TitleType = "通知"
	result.Features.HasCopyNumber = true
	result.Features.CopyNumber = "000001"
	result.Features.ScoreDetails["发文字号"] = 0.18
	result.Features.ScoreDetails["公文标题"] = 0.15

	// 设置版式特征
	result.Features.StyleFeatures.HasRedHeader = true
	result.Features.StyleFeatures.HasSealImage = true
	result.Features.StyleFeatures.IsA4Paper = true
	result.Features.StyleFeatures.StyleScore = 0.8
	result.Features.StyleFeatures.StyleReasons = []string{"检测到红头", "检测到印章"}

	verbose := result.VerboseSummary()

	if verbose == "" {
		t.Error("VerboseSummary 不应为空")
	}

	// 检查包含详细信息
	if !containsString(verbose, "版头特征") {
		t.Error("VerboseSummary 应包含版头特征")
	}

	if !containsString(verbose, "发文字号") {
		t.Error("VerboseSummary 应包含发文字号")
	}

	if !containsString(verbose, "份号") {
		t.Error("VerboseSummary 应包含份号")
	}

	if !containsString(verbose, "版式特征") {
		t.Error("VerboseSummary 应包含版式特征")
	}

	if !containsString(verbose, "得分明细") {
		t.Error("VerboseSummary 应包含得分明细")
	}
}

// ============================================================
// FeatureResult 测试
// ============================================================

func TestFeatureResult_Fields(t *testing.T) {
	result := NewDetectionResult("/path/to/file.docx", "file.docx", 1024)

	// 设置版头特征
	result.Features.HasCopyNumber = true
	result.Features.CopyNumber = "000001"
	result.Features.HasDocNumber = true
	result.Features.DocNumber = "X府发〔2024〕1号"
	result.Features.HasSecretLevel = true
	result.Features.SecretLevel = "绝密"

	// 验证
	if !result.Features.HasCopyNumber {
		t.Error("HasCopyNumber 应为 true")
	}

	if result.Features.CopyNumber != "000001" {
		t.Errorf("CopyNumber = %q, want %q", result.Features.CopyNumber, "000001")
	}

	if !result.Features.HasDocNumber {
		t.Error("HasDocNumber 应为 true")
	}

	if result.Features.DocNumber != "X府发〔2024〕1号" {
		t.Errorf("DocNumber = %q, want %q", result.Features.DocNumber, "X府发〔2024〕1号")
	}
}

// ============================================================
// StyleFeatureResult 测试
// ============================================================

func TestStyleFeatureResult_Fields(t *testing.T) {
	result := NewDetectionResult("/path/to/file.docx", "file.docx", 1024)

	result.Features.StyleFeatures.HasRedText = true
	result.Features.StyleFeatures.HasRedHeader = true
	result.Features.StyleFeatures.HasSealImage = true
	result.Features.StyleFeatures.IsA4Paper = true
	result.Features.StyleFeatures.StyleScore = 0.75
	result.Features.StyleFeatures.StyleReasons = []string{"检测到红头", "检测到印章"}

	sf := result.Features.StyleFeatures

	if !sf.HasRedText {
		t.Error("HasRedText 应为 true")
	}

	if !sf.HasRedHeader {
		t.Error("HasRedHeader 应为 true")
	}

	if !sf.HasSealImage {
		t.Error("HasSealImage 应为 true")
	}

	if !sf.IsA4Paper {
		t.Error("IsA4Paper 应为 true")
	}

	if sf.StyleScore != 0.75 {
		t.Errorf("StyleScore = %v, want 0.75", sf.StyleScore)
	}

	if len(sf.StyleReasons) != 2 {
		t.Errorf("StyleReasons 长度 = %d, want 2", len(sf.StyleReasons))
	}
}

// ============================================================
// 辅助函数测试
// ============================================================

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		got := formatFileSize(tt.size)
		if got != tt.want {
			t.Errorf("formatFileSize(%d) = %q, want %q", tt.size, got, tt.want)
		}
	}
}

func TestFormatBool(t *testing.T) {
	if formatBool(true) != "✓ 是" {
		t.Errorf("formatBool(true) = %q, want %q", formatBool(true), "✓ 是")
	}

	if formatBool(false) != "✗ 否" {
		t.Errorf("formatBool(false) = %q, want %q", formatBool(false), "✗ 否")
	}
}

func TestFormatBoolWithValue(t *testing.T) {
	tests := []struct {
		b     bool
		value string
		want  string
	}{
		{false, "", "✗ 未检测到"},
		{true, "", "✓ 已检���到"},
		{true, "测试值", "✓ 测试值"},
	}

	for _, tt := range tests {
		got := formatBoolWithValue(tt.b, tt.value)
		if got != tt.want {
			t.Errorf("formatBoolWithValue(%v, %q) = %q, want %q", tt.b, tt.value, got, tt.want)
		}
	}
}

func TestValueOrNA(t *testing.T) {
	if valueOrNA("") != "N/A" {
		t.Errorf("valueOrNA(\"\") = %q, want %q", valueOrNA(""), "N/A")
	}

	if valueOrNA("test") != "test" {
		t.Errorf("valueOrNA(\"test\") = %q, want %q", valueOrNA("test"), "test")
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"Hello", 10, "Hello"},
		{"Hello World", 5, "Hello..."},
		{"中文测试内容", 3, "中文测..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		got := truncateString(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

// ============================================================
// 辅助函数
// ============================================================

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}