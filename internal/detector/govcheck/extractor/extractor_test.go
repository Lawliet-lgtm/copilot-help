package extractor

import (
	"strings"
	"testing"
)

// ============================================================
// 份号提取测试
// ============================================================

func TestExtractCopyNumber(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		want     string
		hasMatch bool
	}{
		{
			name:     "标准6位份号在行首",
			text:     "000001\n\n绝密★启用前\n\nXX省人民政府文件",
			want:     "000001",
			hasMatch: true,
		},
		{
			name:     "份号带前缀第X号",
			text:     "第000123号\n\nXX省人民政府文件",
			want:     "000123",
			hasMatch: true,
		},
		{
			name:     "份号：格式",
			text:     "份号：000456\n\nXX省人民政府文件",
			want:     "000456",
			hasMatch: true,
		},
		{
			name:     "份号:格式（英文冒号）",
			text:     "份号:000789\n\nXX省人民政府文件",
			want:     "000789",
			hasMatch: true,
		},
		{
			name:     "编号：格式",
			text:     "编号：000111\n\nXX省人民政府文件",
			want:     "000111",
			hasMatch: true,
		},
		{
			name:     "无份号",
			text:     "XX省人民政府文件\n\n关于加强工作的通知",
			want:     "",
			hasMatch: false,
		},
		{
			name:     "非6位数字不匹配",
			text:     "12345\n\nXX省人民政府文件",
			want:     "",
			hasMatch: false,
		},
		{
			name:     "7位数字不匹配",
			text:     "1234567\n\nXX省人民政府文件",
			want:     "",
			hasMatch: false,
		},
		{
			name:     "份号在前几行",
			text:     "\n\n000999\n\n绝密★启用前",
			want:     "000999",
			hasMatch: true,
		},
		{
			name:     "空文本",
			text:     "",
			want:     "",
			hasMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCopyNumber(tt.text)
			if tt.hasMatch {
				if got != tt.want {
					t.Errorf("extractCopyNumber() = %q, want %q", got, tt.want)
				}
			} else {
				if got != "" {
					t.Errorf("extractCopyNumber() = %q, want empty", got)
				}
			}
		})
	}
}

// ============================================================
// isAllDigits 测试
// ============================================================

func TestIsAllDigits(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"123456", true},
		{"000000", true},
		{"999999", true},
		{"12345a", false},
		{"a12345", false},
		{"12 345", false},
		{"", false},
		{"一二三", false},
		{"123.45", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isAllDigits(tt.input)
			if got != tt.want {
				t.Errorf("isAllDigits(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ============================================================
// Features 提取综合测试
// ============================================================

func TestExtractFeatures_FullDocument(t *testing.T) {
	text := `000001

绝密★启用前

XX省数据安全厅文件

X数安〔2024〕1号

关于加强数据安全管理工作的通知

各市数据安全局：

　　根据上级要求，现就加强数据安全管理工作通知如下：

　　一、提高认识，加强领导。

　　二、完善制度，规范管理。

　　附件：1. 数据安全管理办法
　　      2. 实施细则

　　特此通知。

                                        XX省数据安全厅
                                        2024年1月15日

抄送：省委办公厅，省政府办公厅。
XX省数据安全厅办公室                    2024年1月15日印发
`

	features := ExtractFeatures(text)

	// 份号
	if !features.HasCopyNumber {
		t.Error("期望检测到份号")
	}
	if features.CopyNumber != "000001" {
		t.Errorf("份号 = %q, want %q", features.CopyNumber, "000001")
	}

	// 密级
	if !features.HasSecretLevel {
		t.Error("期望检测到密级")
	}

	// 发文字号
	if !features.HasDocNumber {
		t.Error("期望检测到发文字号")
	}

	// 标题
	if !features.HasTitle {
		t.Error("期望检测到标题")
	}

	// 成文日期
	if !features.HasIssueDate {
		t.Error("期望检测到成文日期")
	}

	// 抄送
	if !features.HasCopyTo {
		t.Error("期望检测到抄送")
	}

	// 印发信息
	if !features.HasPrintInfo {
		t.Error("期望检测到印发信息")
	}

	// 附件
	if !features.HasAttachment {
		t.Error("期望检测到附件说明")
	}
}

func TestExtractFeatures_MinimalDocument(t *testing.T) {
	text := `XX市人民政府文件

X府发〔2024〕5号

关于做好年度工作的通知

各县区人民政府：

　　请做好相关工作。

                                        XX市人民政府
                                        2024年3月1日
`

	features := ExtractFeatures(text)

	// 无份号
	if features.HasCopyNumber {
		t.Error("不期望检测到份号")
	}

	// 无密级
	if features.HasSecretLevel {
		t.Error("不期望检测到密级")
	}

	// 发文字号
	if !features.HasDocNumber {
		t.Error("期望检测到发文字号")
	}

	// 标题
	if !features.HasTitle {
		t.Error("期望检测到标题")
	}

	// 成文日期
	if !features.HasIssueDate {
		t.Error("期望检测到成文日期")
	}
}

func TestExtractFeatures_EmptyText(t *testing.T) {
	features := ExtractFeatures("")

	if features.HasCopyNumber {
		t.Error("空文本不应检测到份号")
	}
	if features.HasDocNumber {
		t.Error("空文本不应检测到发文字号")
	}
	if features.HasTitle {
		t.Error("空文本不应检测到标题")
	}
	if features.TextLength != 0 {
		t.Errorf("TextLength = %d, want 0", features.TextLength)
	}
}

// ============================================================
// 密级提取测试
// ============================================================

func TestExtractFeatures_SecretLevel(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		hasSecret  bool
		wantLevel  string
	}{
		{
			name:       "绝密级别",
			text:       "绝密★启用前\n\nXX省人民政府文件",
			hasSecret:  true,
			wantLevel:  "绝密",
		},
		{
			name:       "机密级别",
			text:       "机密★一年\n\nXX省人民政府文件",
			hasSecret:  true,
			wantLevel:  "机密",
		},
		{
			name:       "秘密级别",
			text:       "秘密★半年\n\nXX省人民政府文件",
			hasSecret:  true,
			wantLevel:  "秘密",
		},
		{
			name:       "无密级",
			text:       "XX省人民政府文件\n\n关于工作的通知",
			hasSecret:  false,
			wantLevel:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := ExtractFeatures(tt.text)

			if features.HasSecretLevel != tt.hasSecret {
				t.Errorf("HasSecretLevel = %v, want %v", features.HasSecretLevel, tt.hasSecret)
			}

			if tt.hasSecret && !strings.Contains(features.SecretLevel, tt.wantLevel) {
				t.Errorf("SecretLevel = %q, want contains %q", features.SecretLevel, tt.wantLevel)
			}
		})
	}
}

// ============================================================
// 紧急程度提取测试
// ============================================================

func TestExtractFeatures_UrgencyLevel(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		hasUrgency  bool
		wantLevel   string
	}{
		{
			name:        "特急",
			text:        "特急\n\nXX省人民政府文件",
			hasUrgency:  true,
			wantLevel:   "特急",
		},
		{
			name:        "加急",
			text:        "加急\n\nXX省人民政府文件",
			hasUrgency:  true,
			wantLevel:   "加急",
		},
		{
			name:        "平急",
			text:        "平急\n\nXX省人民政府文件",
			hasUrgency:  true,
			wantLevel:   "平急",
		},
		{
			name:        "无紧急程度",
			text:        "XX省人民政府文件\n\n关于工作的通知",
			hasUrgency:  false,
			wantLevel:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := ExtractFeatures(tt.text)

			if features.HasUrgencyLevel != tt.hasUrgency {
				t.Errorf("HasUrgencyLevel = %v, want %v", features.HasUrgencyLevel, tt.hasUrgency)
			}

			if tt.hasUrgency && features.UrgencyLevel != tt.wantLevel {
				t.Errorf("UrgencyLevel = %q, want %q", features.UrgencyLevel, tt.wantLevel)
			}
		})
	}
}

// ============================================================
// 标题类型提取测试
// ============================================================

func TestExtractFeatures_TitleType(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		hasTitle  bool
		wantType  string
	}{
		{
			name:      "通知类型",
			text:       "关于加强安全管理工作的通知\n\n各单位：",
			hasTitle:  true,
			wantType:  "通知",
		},
		{
			name:      "决定类型",
			text:       "关于表彰先进的决定\n\n各单位：",
			hasTitle:  true,
			wantType:  "决定",
		},
		{
			name:      "意见类型",
			text:       "关于深化改革的意见\n\n各单位：",
			hasTitle:  true,
			wantType:  "意见",
		},
		{
			name:      "报告类型",
			text:       "关于工作情况的报告\n\n市人民政府：",
			hasTitle:  true,
			wantType:  "报告",
		},
		{
			name:      "请示类型",
			text:       "关于申请资金的请示\n\n省人民政府：",
			hasTitle:  true,
			wantType:  "请示",
		},
		{
			name:      "批复类型",
			text:       "关于同意设立机构的批复\n\n市人民政府：",
			hasTitle:  true,
			wantType:  "批复",
		},
		{
			name:      "函类型",
			text:       "关于商请支持的函\n\n省发改委：",
			hasTitle:  true,
			wantType:  "函",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := ExtractFeatures(tt.text)

			if features.HasTitle != tt.hasTitle {
				t.Errorf("HasTitle = %v, want %v", features.HasTitle, tt.hasTitle)
			}

			if tt.hasTitle && features.TitleType != tt.wantType {
				t.Errorf("TitleType = %q, want %q", features.TitleType, tt.wantType)
			}
		})
	}
}

// ============================================================
// 成文日期提取测试
// ============================================================

func TestExtractFeatures_IssueDate(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		hasDate  bool
		wantDate string
	}{
		{
			name:     "标准格式年月日",
			text:     "XX省人民政府\n2024年1月15日",
			hasDate:  true,
			wantDate: "2024年1月15日",
		},
		{
			name:     "两位月日",
			text:     "XX省人民政府\n2024年01月05日",
			hasDate:  true,
			wantDate: "2024年01月05日",
		},
		{
			name:     "二〇格式",
			text:     "XX省人民政府\n二〇二四年一月十五日",
			hasDate:  true,
			wantDate: "二〇二四年一月十五日",
		},
		{
			name:     "无日期",
			text:     "XX省人民政府\n关于工作的通知",
			hasDate:  false,
			wantDate: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := ExtractFeatures(tt.text)

			if features.HasIssueDate != tt.hasDate {
				t.Errorf("HasIssueDate = %v, want %v", features.HasIssueDate, tt.hasDate)
			}

			if tt.hasDate && features.IssueDate != tt.wantDate {
				t.Errorf("IssueDate = %q, want %q", features.IssueDate, tt.wantDate)
			}
		})
	}
}

// ============================================================
// Features 方法测试
// ============================================================

func TestFeatures_CountPositiveFeatures(t *testing.T) {
	features := &Features{
		HasCopyNumber:  true,
		HasDocNumber:   true,
		HasTitle:       true,
		HasIssueDate:   true,
		HasSecretLevel: false,
	}

	count := features.CountPositiveFeatures()
	if count != 4 {
		t.Errorf("CountPositiveFeatures() = %d, want 4", count)
	}
}

func TestFeatures_HasCriticalFeatures(t *testing.T) {
	tests := []struct {
		name     string
		features *Features
		want     bool
	}{
		{
			name: "三要素齐全",
			features: &Features{
				HasDocNumber: true,
				HasTitle:     true,
				HasIssueDate: true,
			},
			want: true,
		},
		{
			name: "两个要素",
			features: &Features{
				HasDocNumber: true,
				HasTitle:     true,
				HasIssueDate: false,
			},
			want: true,
		},
		{
			name: "只有一个要素",
			features: &Features{
				HasDocNumber: true,
				HasTitle:     false,
				HasIssueDate: false,
			},
			want: false,
		},
		{
			name: "无要素",
			features: &Features{
				HasDocNumber: false,
				HasTitle:     false,
				HasIssueDate: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.features.HasCriticalFeatures()
			if got != tt.want {
				t.Errorf("HasCriticalFeatures() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeatures_HasStyleFeatures(t *testing.T) {
	tests := []struct {
		name     string
		features *Features
		want     bool
	}{
		{
			name: "有红头",
			features: &Features{
				StyleFeatures: &StyleFeatures{
					HasRedHeader: true,
				},
			},
			want: true,
		},
		{
			name: "有红色文本",
			features: &Features{
				StyleFeatures: &StyleFeatures{
					HasRedText: true,
				},
			},
			want: true,
		},
		{
			name: "有印章",
			features: &Features{
				StyleFeatures: &StyleFeatures{
					HasSealImage: true,
				},
			},
			want: true,
		},
		{
			name: "无版式特征",
			features: &Features{
				StyleFeatures: &StyleFeatures{},
			},
			want: false,
		},
		{
			name: "StyleFeatures为nil",
			features: &Features{
				StyleFeatures: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.features.HasStyleFeatures()
			if got != tt.want {
				t.Errorf("HasStyleFeatures() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeatures_CountStyleFeatures(t *testing.T) {
	features := &Features{
		StyleFeatures: &StyleFeatures{
			HasRedText:       true,
			HasRedHeader:     true,
			HasOfficialFonts: true,
			IsA4Paper:        true,
			HasSealImage:     false,
		},
	}

	count := features.CountStyleFeatures()
	if count != 4 {
		t.Errorf("CountStyleFeatures() = %d, want 4", count)
	}
}

func TestFeatures_HasProhibitedContent(t *testing.T) {
	tests := []struct {
		name     string
		features *Features
		want     bool
	}{
		{
			name: "有禁止词",
			features: &Features{
				ProhibitWords: []string{"广告", "促销"},
			},
			want: true,
		},
		{
			name: "无禁止词",
			features: &Features{
				ProhibitWords: []string{},
			},
			want: false,
		},
		{
			name: "禁止词为nil",
			features: &Features{
				ProhibitWords: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.features.HasProhibitedContent()
			if got != tt.want {
				t.Errorf("HasProhibitedContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeatures_GetProhibitedRatio(t *testing.T) {
	tests := []struct {
		name     string
		features *Features
		want     float64
	}{
		{
			name: "有禁止词和关键词匹配",
			features: &Features{
				KeywordMatches: 8,
				ProhibitWords:  []string{"广告", "促销"},
			},
			want: 0.2, // 2 / (8 + 2)
		},
		{
			name: "无关键词匹配",
			features: &Features{
				KeywordMatches: 0,
				ProhibitWords:  []string{"广告"},
			},
			want: 0,
		},
		{
			name: "无禁止词",
			features: &Features{
				KeywordMatches: 10,
				ProhibitWords:  []string{},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.features.GetProhibitedRatio()
			if got != tt.want {
				t.Errorf("GetProhibitedRatio() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================
// FeatureSummary 测试
// ============================================================

func TestFeatures_FeatureSummary(t *testing.T) {
	features := &Features{
		HasCopyNumber:  true,
		HasDocNumber:   true,
		HasTitle:       true,
		TitleType:      "通知",
		HasIssueDate:   true,
		PatternMatches: 5,
		KeywordMatches: 10,
		StyleFeatures: &StyleFeatures{
			HasRedText:   true,
			HasRedHeader: true,
			IsA4Paper:    true,
			StyleScore:   0.75,
		},
	}

	summary := features.FeatureSummary()

	// 检查关键字段
	if summary["has_copy_number"] != true {
		t.Error("summary[has_copy_number] should be true")
	}
	if summary["has_doc_number"] != true {
		t.Error("summary[has_doc_number] should be true")
	}
	if summary["has_title"] != true {
		t.Error("summary[has_title] should be true")
	}
	if summary["title_type"] != "通知" {
		t.Errorf("summary[title_type] = %v, want 通知", summary["title_type"])
	}
	if summary["has_red_header"] != true {
		t.Error("summary[has_red_header] should be true")
	}
	if summary["style_score"] != 0.75 {
		t.Errorf("summary[style_score] = %v, want 0.75", summary["style_score"])
	}
}

// ============================================================
// QuickCheck 测试
// ============================================================

func TestQuickCheck(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "完整公文",
			text: `XX省人民政府文件

X府发〔2024〕1号

关于加强工作的通知

各市人民政府：

　　请做好工作。

                                        XX省人民政府
                                        2024年1月15日`,
			want: true,
		},
		{
			name: "文本过短",
			text: "这是一段很短的文本",
			want: false,
		},
		{
			name: "普通文本无公文特征",
			text: strings.Repeat("这是一段普通的文本内容，没有公文特征。", 20),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := QuickCheck(tt.text)
			if got != tt.want {
				t.Errorf("QuickCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ============================================================
// Extractor 配置测试
// ============================================================

func TestExtractor_DefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.EnablePatterns {
		t.Error("默认配置应启用正则模式")
	}
	if !config.EnableKeywords {
		t.Error("默认配置应启用关键词匹配")
	}
	if !config.NormalizeText {
		t.Error("默认配置应启用文本规范化")
	}
}

func TestExtractor_DisablePatterns(t *testing.T) {
	config := &Config{
		EnablePatterns: false,
		EnableKeywords: true,
		NormalizeText:  true,
	}

	ext := New(config)
	text := "X府发〔2024〕1号\n关于工作的通知"
	features := ext.Extract(text)

	// 禁用正则后，发文字号不应被提取
	if features.HasDocNumber {
		t.Error("禁用正则后不应检测到发文字号")
	}
}

func TestExtractor_DisableKeywords(t *testing.T) {
	config := &Config{
		EnablePatterns: true,
		EnableKeywords: false,
		NormalizeText:  true,
	}

	ext := New(config)
	text := "XX省人民政府文件\n关于加强工作的通知"
	features := ext.Extract(text)

	// 禁用关键词后，机关名称不应被提取
	if features.HasOrgName {
		t.Error("禁用关键词后不应检测到机关名称")
	}
}

// ============================================================
// 辅助函数测试
// ============================================================

func TestCountChineseChars(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"中文测试", 4},
		{"Hello世界", 2},
		{"12345", 0},
		{"", 0},
		{"中English混合文Text本", 5},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := countChineseChars(tt.text)
			if got != tt.want {
				t.Errorf("countChineseChars(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  int
	}{
		{
			name:  "有重复",
			input: []string{"a", "b", "a", "c", "b"},
			want:  3,
		},
		{
			name:  "无重复",
			input: []string{"a", "b", "c"},
			want:  3,
		},
		{
			name:  "全重复",
			input: []string{"a", "a", "a"},
			want:  1,
		},
		{
			name:  "空切片",
			input: []string{},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueStrings(tt.input)
			if len(got) != tt.want {
				t.Errorf("uniqueStrings() len = %d, want %d", len(got), tt.want)
			}
		})
	}
}