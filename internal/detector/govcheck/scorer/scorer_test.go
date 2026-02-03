package scorer

import (
	"testing"

	"linuxFileWatcher/internal/detector/govcheck/extractor"
)

// ============================================================
// Scorer 创建测试
// ============================================================

func TestNew_DefaultConfig(t *testing.T) {
	scorer := New(nil)

	if scorer == nil {
		t.Fatal("New(nil) 返回 nil")
	}

	if scorer.config == nil {
		t.Fatal("scorer.config 为 nil")
	}

	if scorer.config.Threshold != 0.6 {
		t.Errorf("Threshold = %v, want 0.6", scorer.config.Threshold)
	}

	if scorer.config.TextWeight != 0.55 {
		t.Errorf("TextWeight = %v, want 0.55", scorer.config.TextWeight)
	}

	if scorer.config.StyleWeight != 0.45 {
		t.Errorf("StyleWeight = %v, want 0.45", scorer.config.StyleWeight)
	}
}

func TestNew_CustomConfig(t *testing.T) {
	config := &Config{
		Threshold:   0.7,
		TextWeight:  0.6,
		StyleWeight: 0.4,
		Weights:     DefaultWeights(),
	}

	scorer := New(config)

	if scorer.config.Threshold != 0.7 {
		t.Errorf("Threshold = %v, want 0.7", scorer.config.Threshold)
	}

	if scorer.config.TextWeight != 0.6 {
		t.Errorf("TextWeight = %v, want 0.6", scorer.config.TextWeight)
	}
}

// ============================================================
// DefaultConfig 测试
// ============================================================

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Threshold != 0.6 {
		t.Errorf("Threshold = %v, want 0.6", config.Threshold)
	}

	// 检查权重总和
	totalTextWeight := config.TextWeight
	totalStyleWeight := config.StyleWeight

	if totalTextWeight+totalStyleWeight != 1.0 {
		t.Errorf("TextWeight + StyleWeight = %v, want 1.0", totalTextWeight+totalStyleWeight)
	}
}

// ============================================================
// DefaultWeights 测试
// ============================================================

func TestDefaultWeights(t *testing.T) {
	weights := DefaultWeights()

	// 检查关键权重是否设置
	if weights.DocNumber <= 0 {
		t.Error("DocNumber 权重应大于 0")
	}

	if weights.Title <= 0 {
		t.Error("Title 权重应大于 0")
	}

	if weights.IssueDate <= 0 {
		t.Error("IssueDate 权重应大于 0")
	}

	if weights.RedHeader <= 0 {
		t.Error("RedHeader 权重应大于 0")
	}

	if weights.SealImage <= 0 {
		t.Error("SealImage 权重应大于 0")
	}

	if weights.CopyNumber <= 0 {
		t.Error("CopyNumber 权重应大于 0")
	}
}

// ============================================================
// Score 基础测试
// ============================================================

func TestScore_NilFeatures(t *testing.T) {
	scorer := New(nil)
	result := scorer.Score(nil)

	if result == nil {
		t.Fatal("Score(nil) 返回 nil")
	}

	if result.TotalScore != 0 {
		t.Errorf("TotalScore = %v, want 0", result.TotalScore)
	}

	if result.IsOfficialDoc {
		t.Error("nil features 不应判定为公文")
	}

	if len(result.Reasons) == 0 {
		t.Error("应该有判定理由")
	}
}

func TestScore_EmptyFeatures(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{}

	result := scorer.Score(features)

	if result.TotalScore < 0 || result.TotalScore > 1 {
		t.Errorf("TotalScore = %v, 应在 0-1 范围内", result.TotalScore)
	}

	if result.IsOfficialDoc {
		t.Error("空 features 不应判定为公文")
	}
}

// ============================================================
// 文本特征评分测试
// ============================================================

func TestScore_TextFeatures_CopyNumber(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasCopyNumber: true,
		CopyNumber:    "000001",
		TextLength:    500,
	}

	result := scorer.Score(features)

	if _, ok := result.Details["份号"]; !ok {
		t.Error("应该有份号得分")
	}

	if result.Details["份号"] != scorer.config.Weights.CopyNumber {
		t.Errorf("份号得分 = %v, want %v", result.Details["份号"], scorer.config.Weights.CopyNumber)
	}
}

func TestScore_TextFeatures_DocNumber(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasDocNumber: true,
		DocNumber:    "X府发〔2024〕1号",
		TextLength:   500,
	}

	result := scorer.Score(features)

	if _, ok := result.Details["发文字号"]; !ok {
		t.Error("应该有发文字号得分")
	}

	if result.Details["发文字号"] != scorer.config.Weights.DocNumber {
		t.Errorf("发文字号得分 = %v, want %v", result.Details["发文字号"], scorer.config.Weights.DocNumber)
	}
}

func TestScore_TextFeatures_Title(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasTitle:   true,
		Title:      "关于加强工作的通知",
		TitleType:  "通知",
		TextLength: 500,
	}

	result := scorer.Score(features)

	if _, ok := result.Details["公文标题"]; !ok {
		t.Error("应该有公文标题得分")
	}

	if _, ok := result.Details["标题文种"]; !ok {
		t.Error("应该有标题文种得分")
	}
}

func TestScore_TextFeatures_SecretLevel(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasSecretLevel: true,
		SecretLevel:    "绝密★启用前",
		TextLength:     500,
	}

	result := scorer.Score(features)

	if _, ok := result.Details["密级标志"]; !ok {
		t.Error("应该有密级标志得分")
	}
}

func TestScore_TextFeatures_UrgencyLevel(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasUrgencyLevel: true,
		UrgencyLevel:    "特急",
		TextLength:      500,
	}

	result := scorer.Score(features)

	if _, ok := result.Details["紧急程度"]; !ok {
		t.Error("应该有紧急程度得分")
	}
}

func TestScore_TextFeatures_IssueDate(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasIssueDate: true,
		IssueDate:    "2024年1月15日",
		TextLength:   500,
	}

	result := scorer.Score(features)

	if _, ok := result.Details["成文日期"]; !ok {
		t.Error("应该有成文日期得分")
	}
}

func TestScore_TextFeatures_AllCore(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasDocNumber: true,
		DocNumber:    "X府发〔2024〕1号",
		HasTitle:     true,
		Title:        "关于加强工作的通知",
		TitleType:    "通知",
		HasIssueDate: true,
		IssueDate:    "2024年1月15日",
		TextLength:   500,
	}

	result := scorer.Score(features)

	// 应该有核心特征组合加分
	if _, ok := result.Details["文本核心特征组合"]; !ok {
		t.Error("应该有文本核心特征组合加分")
	}

	// 分数应该较高
	if result.TotalScore < 0.3 {
		t.Errorf("具备核心特征的分数应较高，实际: %v", result.TotalScore)
	}
}

// ============================================================
// 版式特征评分测试
// ============================================================

func TestScore_StyleFeatures_RedHeader(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength: 500,
		StyleFeatures: &extractor.StyleFeatures{
			HasRedHeader: true,
		},
	}

	result := scorer.Score(features)

	if _, ok := result.Details["红头标志"]; !ok {
		t.Error("应该有红头标志得分")
	}
}

func TestScore_StyleFeatures_RedText(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength: 500,
		StyleFeatures: &extractor.StyleFeatures{
			HasRedText: true,
		},
	}

	result := scorer.Score(features)

	if _, ok := result.Details["红色文本"]; !ok {
		t.Error("应该有红色文本得分")
	}
}

func TestScore_StyleFeatures_SealImage(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength: 500,
		StyleFeatures: &extractor.StyleFeatures{
			HasSealImage: true,
		},
	}

	result := scorer.Score(features)

	if _, ok := result.Details["印章图片"]; !ok {
		t.Error("应该有印章图片得分")
	}
}

func TestScore_StyleFeatures_A4Paper(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength: 500,
		StyleFeatures: &extractor.StyleFeatures{
			IsA4Paper: true,
		},
	}

	result := scorer.Score(features)

	if _, ok := result.Details["A4纸张"]; !ok {
		t.Error("应该有A4纸张得分")
	}
}

func TestScore_StyleFeatures_CoreCombo(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength: 500,
		StyleFeatures: &extractor.StyleFeatures{
			HasRedHeader: true,
			HasSealImage: true,
		},
	}

	result := scorer.Score(features)

	// 应该有版式核心特征组合加分
	if _, ok := result.Details["版式核心特征组合"]; !ok {
		t.Error("应该有版式核心特征组合加分")
	}
}

func TestScore_StyleFeatures_FullCombo(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength: 500,
		StyleFeatures: &extractor.StyleFeatures{
			HasRedHeader: true,
			HasRedText:   true,
			IsA4Paper:    true,
		},
	}

	result := scorer.Score(features)

	// 应该有版式特征组合加分
	if _, ok := result.Details["版式特征组合加分"]; !ok {
		t.Error("应该有版式特征组合加分")
	}
}

// ============================================================
// 图片类型加分测试
// ============================================================

func TestScore_ImageType_Bonus(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength: 150,
		StyleFeatures: &extractor.StyleFeatures{
			HasRedHeader: true,
			HasSealImage: true,
			IsA4Paper:    true,
			StyleReasons: []string{"通过OCR提取文本内容"},
		},
	}

	result := scorer.Score(features)

	// 应该有图片版式三要素加分
	if _, ok := result.Details["图片版式三要素加分"]; !ok {
		t.Error("应该有图片版式三要素加分")
	}

	// 应该有OCR补偿
	if _, ok := result.Details["图片OCR补偿"]; !ok {
		t.Error("应该有图片OCR补偿")
	}
}

func TestScore_ImageType_TwoElements(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength: 150,
		StyleFeatures: &extractor.StyleFeatures{
			HasRedHeader: true,
			HasSealImage: true,
			IsA4Paper:    false,
			StyleReasons: []string{"通过OCR提取文本内容"},
		},
	}

	result := scorer.Score(features)

	// 应该有图片版式双要素加分
	if _, ok := result.Details["图片版式双要素加分"]; !ok {
		t.Error("应该有图片版式双要素加分")
	}
}

// ============================================================
// 惩罚测试
// ============================================================

func TestScore_Penalty_ShortText(t *testing.T) {
	scorer := New(nil)

	// 文本过短 (< 100)
	features := &extractor.Features{
		TextLength:   50,
		HasDocNumber: true,
	}

	result := scorer.Score(features)

	if _, ok := result.Details["文本过短惩罚"]; !ok {
		t.Error("应该有文本过短惩罚")
	}

	// 文本较短 (100-200)
	features2 := &extractor.Features{
		TextLength:   150,
		HasDocNumber: true,
	}

	result2 := scorer.Score(features2)

	if _, ok := result2.Details["文本较短惩罚"]; !ok {
		t.Error("应该有文本较短惩罚")
	}
}

func TestScore_Penalty_ProhibitWords(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength:    500,
		ProhibitWords: []string{"广告", "促销", "优惠"},
	}

	result := scorer.Score(features)

	if _, ok := result.Details["非公文特征惩罚"]; !ok {
		t.Error("应该有非公文特征惩罚")
	}

	if result.Details["非公文特征惩罚"] >= 0 {
		t.Error("非公文特征惩罚应为负值")
	}
}

// ============================================================
// 综合评分测试
// ============================================================

func TestScore_FullDocument_Pass(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasCopyNumber:   true,
		CopyNumber:      "000001",
		HasDocNumber:    true,
		DocNumber:       "X府发〔2024〕1号",
		HasSecretLevel:  true,
		SecretLevel:     "绝密",
		HasTitle:        true,
		Title:           "关于加强工作的通知",
		TitleType:       "通知",
		HasMainSend:     true,
		MainSend:        "各市人民政府",
		HasIssueDate:    true,
		IssueDate:       "2024年1月15日",
		HasCopyTo:       true,
		HasPrintInfo:    true,
		HasOrgName:      true,
		OrgNames:        []string{"省人民政府"},
		TextLength:      1000,
		KeywordMatches:  10,
		PatternMatches:  8,
		StyleFeatures: &extractor.StyleFeatures{
			HasRedHeader:     true,
			HasRedText:       true,
			HasSealImage:     true,
			IsA4Paper:        true,
			MarginMatch:      true,
			HasOfficialFonts: true,
		},
	}

	result := scorer.Score(features)

	if !result.IsOfficialDoc {
		t.Errorf("完整公文应判定为公文，得分: %v", result.TotalScore)
	}

	if result.TotalScore < 0.6 {
		t.Errorf("完整公文得分应 >= 0.6，实际: %v", result.TotalScore)
	}

	if result.Confidence == "" {
		t.Error("应该有置信度描述")
	}
}

func TestScore_NonDocument_Fail(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		TextLength:    500,
		ProhibitWords: []string{"广告", "促销", "优惠", "打折", "限时"},
	}

	result := scorer.Score(features)

	if result.IsOfficialDoc {
		t.Error("非公文内容不应判定为公文")
	}

	if result.TotalScore >= 0.6 {
		t.Errorf("非公文得分应 < 0.6，实际: %v", result.TotalScore)
	}
}

// ============================================================
// 置信度等级测试
// ============================================================

func TestGetConfidenceLevel(t *testing.T) {
	scorer := New(nil)

	tests := []struct {
		score float64
		want  string
	}{
		{0.90, "很高"},
		{0.85, "很高"},
		{0.75, "高"},
		{0.70, "高"},
		{0.60, "中"},
		{0.55, "中"},
		{0.45, "低"},
		{0.40, "低"},
		{0.30, "很低"},
		{0.10, "很低"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := scorer.getConfidenceLevel(tt.score)
			if got != tt.want {
				t.Errorf("getConfidenceLevel(%v) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

// ============================================================
// 分数边界测试
// ============================================================

func TestScore_BoundaryCheck(t *testing.T) {
	scorer := New(nil)

	// 极端高分情况
	features := &extractor.Features{
		HasCopyNumber:   true,
		HasDocNumber:    true,
		HasSecretLevel:  true,
		HasUrgencyLevel: true,
		HasIssuer:       true,
		HasTitle:        true,
		TitleType:       "通知",
		HasMainSend:     true,
		HasAttachment:   true,
		HasIssueDate:    true,
		HasCopyTo:       true,
		HasPrintInfo:    true,
		HasOrgName:      true,
		OrgNames:        []string{"a", "b", "c", "d"},
		DocTypes:        []string{"通知", "决定", "意见"},
		ActionWords:     []string{"加强", "推进", "落实"},
		FormalWords:     []string{"特此", "为此"},
		HeaderWords:     []string{"发文字号"},
		FooterWords:     []string{"抄送"},
		TextLength:      2000,
		StyleFeatures: &extractor.StyleFeatures{
			HasRedHeader:     true,
			HasRedText:       true,
			HasSealImage:     true,
			IsA4Paper:        true,
			MarginMatch:      true,
			HasOfficialFonts: true,
			TitleFontMatch:   true,
			BodyFontMatch:    true,
			HasCenteredTitle: true,
			LineSpacingMatch: true,
		},
	}

	result := scorer.Score(features)

	if result.TotalScore > 1.0 {
		t.Errorf("TotalScore = %v, 应 <= 1.0", result.TotalScore)
	}

	if result.TotalScore < 0 {
		t.Errorf("TotalScore = %v, 应 >= 0", result.TotalScore)
	}
}

func TestScore_NegativeScoreBoundary(t *testing.T) {
	scorer := New(nil)

	// 大量禁止词，极短文本
	features := &extractor.Features{
		TextLength:    30,
		ProhibitWords: []string{"a", "b", "c", "d", "e", "f"},
	}

	result := scorer.Score(features)

	if result.TotalScore < 0 {
		t.Errorf("TotalScore = %v, 应 >= 0", result.TotalScore)
	}
}

// ============================================================
// 快捷函数测试
// ============================================================

func TestScoreFeatures(t *testing.T) {
	features := &extractor.Features{
		HasDocNumber: true,
		DocNumber:    "X府发〔2024〕1号",
		TextLength:   500,
	}

	result := ScoreFeatures(features)

	if result == nil {
		t.Fatal("ScoreFeatures 返回 nil")
	}

	if result.Threshold != 0.6 {
		t.Errorf("Threshold = %v, want 0.6", result.Threshold)
	}
}

func TestScoreText(t *testing.T) {
	text := `XX省人民政府文件

X府发〔2024〕1号

关于加强工作的通知

各市人民政府：

　　请做好相关工作。

                                        XX省人民政府
                                        2024年1月15日`

	result := ScoreText(text)

	if result == nil {
		t.Fatal("ScoreText 返回 nil")
	}

	// 应该检测到一些特征
	if result.TotalScore == 0 {
		t.Error("公文文本得分不应为 0")
	}
}

func TestScoreTextWithThreshold(t *testing.T) {
	text := `XX省人民政府文件

X府发〔2024〕1号

关于加强工作的通知`

	result := ScoreTextWithThreshold(text, 0.8)

	if result == nil {
		t.Fatal("ScoreTextWithThreshold 返回 nil")
	}

	if result.Threshold != 0.8 {
		t.Errorf("Threshold = %v, want 0.8", result.Threshold)
	}
}

func TestQuickScore(t *testing.T) {
	text := `XX省人民政府文件

X府发〔2024〕1号

关于加强工作的通知

各市人民政府：

　　请做好相关工作。

                                        XX省人民政府
                                        2024年1月15日`

	score, isOfficial := QuickScore(text, 0.3)

	if score == 0 {
		t.Error("公文文本得分不应为 0")
	}

	// 使用较低阈值应该判定为公文
	if !isOfficial {
		t.Errorf("使用阈值 0.3 应判定为公文，得分: %v", score)
	}
}

// ============================================================
// 辅助函数测试
// ============================================================

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{0, 0, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"Hello", 10, "Hello"},
		{"Hello World", 5, "Hello..."},
		{"中文测试文本", 3, "中文测..."},
		{"", 5, ""},
		{"Hi", 2, "Hi"},
	}

	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

// ============================================================
// ScoreResult 测试
// ============================================================

func TestScoreResult_Details(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasDocNumber: true,
		DocNumber:    "X府发〔2024〕1号",
		HasTitle:     true,
		Title:        "关于工作的通知",
		TextLength:   500,
	}

	result := scorer.Score(features)

	if result.Details == nil {
		t.Fatal("Details 不应为 nil")
	}

	if len(result.Details) == 0 {
		t.Error("Details 不应为空")
	}
}

func TestScoreResult_Factors(t *testing.T) {
	scorer := New(nil)
	features := &extractor.Features{
		HasDocNumber:  true,
		TextLength:    50, // 触发惩罚
		ProhibitWords: []string{"广告"},
	}

	result := scorer.Score(features)

	if len(result.PositiveFactors) == 0 {
		t.Error("应该有正向因素")
	}

	if len(result.NegativeFactors) == 0 {
		t.Error("应该有负向因素")
	}
}

func TestScoreResult_Reasons(t *testing.T) {
	scorer := New(nil)

	// 判定为公文的情况
	features1 := &extractor.Features{
		HasDocNumber: true,
		HasTitle:     true,
		TitleType:    "通知",
		HasIssueDate: true,
		HasOrgName:   true,
		TextLength:   500,
		StyleFeatures: &extractor.StyleFeatures{
			HasRedHeader: true,
			HasSealImage: true,
		},
	}

	result1 := scorer.Score(features1)
	if len(result1.Reasons) == 0 {
		t.Error("应该有判定理由")
	}

	// 判定为非公文的情况
	features2 := &extractor.Features{
		TextLength: 500,
	}

	result2 := scorer.Score(features2)
	if len(result2.Reasons) == 0 {
		t.Error("应该有判定理由")
	}
}