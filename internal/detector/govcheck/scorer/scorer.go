package scorer

import (
	"linuxFileWatcher/internal/detector/govcheck/extractor"
)

// ScoreResult 评分结果
type ScoreResult struct {
	TotalScore      float64            // 总分 (0-1)
	TextScore       float64            // 文本特征得分
	StyleScore      float64            // 版式特征得分
	IsOfficialDoc   bool               // 是否判定为公文
	Confidence      string             // 置信度描述 (高/中/低)
	Threshold       float64            // 使用的阈值
	Details         map[string]float64 // 各项得分明细
	Reasons         []string           // 判定理由
	PositiveFactors []string           // 正向因素
	NegativeFactors []string           // 负向因素
}

// Scorer 评分器
type Scorer struct {
	config *Config
}

// Config 评分器配置
type Config struct {
	Threshold float64 // 判定阈值 (默认0.6)

	// 文本/版式权重分配
	TextWeight  float64 // 文本特征权重 (默认0.6)
	StyleWeight float64 // 版式特征权重 (默认0.4)

	// 各特征权重
	Weights WeightConfig
}

// WeightConfig 权重配置
type WeightConfig struct {
	// ============================================================
	// 文本特征权重
	// ============================================================

	// 版头特征权重
	CopyNumber   float64 // 份号 (新增)
	DocNumber    float64 // 发文字号
	SecretLevel  float64 // 密级
	UrgencyLevel float64 // 紧急程度
	Issuer       float64 // 签发人

	// 主体特征权重
	Title      float64 // 公文标题
	TitleType  float64 // 标题文种
	MainSend   float64 // 主送机关
	Attachment float64 // 附件

	// 版记特征权重
	IssueDate float64 // 成文日期
	CopyTo    float64 // 抄送
	PrintInfo float64 // 印发信息

	// 机关特征权重
	OrgName float64 // 机关名称

	// 关键词权重
	DocType    float64 // 公文文种关键词
	ActionWord float64 // 公文动作词
	FormalWord float64 // 正式用语
	HeaderWord float64 // 版头关键词
	FooterWord float64 // 版记关键词
	Prohibited float64 // 非公文特征 (负权重)

	// ============================================================
	// 版式特征权重
	// ============================================================

	RedText       float64 // 红色文本
	RedHeader     float64 // 红头
	OfficialFonts float64 // 公文字体
	TitleFont     float64 // 标题字号
	BodyFont      float64 // 正文字号
	A4Paper       float64 // A4纸张
	Margins       float64 // 页边距
	CenteredTitle float64 // 居中标题
	LineSpacing   float64 // 行距
	SealImage     float64 // 印章图片
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Threshold:   0.6,
		TextWeight:  0.55, // 文本特征占55%
		StyleWeight: 0.45, // 版式特征占45%
		Weights:     DefaultWeights(),
	}
}

// DefaultWeights 返回默认权重
func DefaultWeights() WeightConfig {
	return WeightConfig{
		// ============================================================
		// 文本特征权重 (总和约1.0，用于文本得分计算)
		// ============================================================

		// 版头特征 (核心特征，权重较高)
		CopyNumber:   0.04, // 份号（可选要素，权重较低）
		DocNumber:    0.18,
		SecretLevel:  0.06,
		UrgencyLevel: 0.05,
		Issuer:       0.08,

		// 主体特征 (核心特征)
		Title:      0.15,
		TitleType:  0.05,
		MainSend:   0.08,
		Attachment: 0.04,

		// 版记特征
		IssueDate: 0.12,
		CopyTo:    0.05,
		PrintInfo: 0.05,

		// 机关特征
		OrgName: 0.10,

		// 关键词 (辅助特征)
		DocType:    0.05,
		ActionWord: 0.04,
		FormalWord: 0.04,
		HeaderWord: 0.03,
		FooterWord: 0.03,
		Prohibited: 0.20, // 惩罚权重

		// ============================================================
		// 版式特征权重 (总和约1.0，用于版式得分计算)
		// ============================================================

		RedText:       0.12,
		RedHeader:     0.18, // 红头是重要特征
		OfficialFonts: 0.10,
		TitleFont:     0.08,
		BodyFont:      0.07,
		A4Paper:       0.10,
		Margins:       0.10,
		CenteredTitle: 0.07,
		LineSpacing:   0.05,
		SealImage:     0.13, // 印章是重要特征
	}
}

// New 创建评分器
func New(config *Config) *Scorer {
	if config == nil {
		config = DefaultConfig()
	}
	return &Scorer{
		config: config,
	}
}

// Score 对特征进行评分
func (s *Scorer) Score(features *extractor.Features) *ScoreResult {
	result := &ScoreResult{
		Details:   make(map[string]float64),
		Threshold: s.config.Threshold,
	}

	if features == nil {
		result.Reasons = append(result.Reasons, "无特征数据")
		return result
	}

	// 1. 计算��本特征得分
	textScore := s.scoreTextFeatures(features, result)

	// 2. 计算版式特征得分
	styleScore := s.scoreStyleFeatures(features, result)

	// 3. 计算综合得分
	result.TextScore = textScore
	result.StyleScore = styleScore

	// 加权综合
	hasStyleFeatures := features.StyleFeatures != nil && features.HasStyleFeatures()

	if hasStyleFeatures {
		// 有版式特征时，按权重计算
		result.TotalScore = textScore*s.config.TextWeight + styleScore*s.config.StyleWeight
	} else {
		// 无版式特征时，只使用文本得分
		result.TotalScore = textScore
	}

	// 4. 应用惩罚和加成
	s.applyAdjustments(features, result)

	// 5. 确保分数在 0-1 范围内
	if result.TotalScore < 0 {
		result.TotalScore = 0
	}
	if result.TotalScore > 1 {
		result.TotalScore = 1
	}

	// 6. 判定结果
	result.IsOfficialDoc = result.TotalScore >= s.config.Threshold
	result.Confidence = s.getConfidenceLevel(result.TotalScore)
	result.Reasons = s.generateReasons(result, features)

	return result
}

// scoreTextFeatures 计算文本特征得分
func (s *Scorer) scoreTextFeatures(features *extractor.Features, result *ScoreResult) float64 {
	w := s.config.Weights
	score := 0.0

	// ============================================================
	// 版头特征评分
	// ============================================================

	// 份号 (新增)
	if features.HasCopyNumber {
		score += w.CopyNumber
		result.Details["份号"] = w.CopyNumber
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到份号: "+features.CopyNumber)
	}

	if features.HasDocNumber {
		score += w.DocNumber
		result.Details["发文字号"] = w.DocNumber
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到发文字号: "+truncate(features.DocNumber, 20))
	}

	if features.HasSecretLevel {
		score += w.SecretLevel
		result.Details["密级标志"] = w.SecretLevel
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到密级标志: "+features.SecretLevel)
	}

	if features.HasUrgencyLevel {
		score += w.UrgencyLevel
		result.Details["紧急程度"] = w.UrgencyLevel
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到紧急程度: "+features.UrgencyLevel)
	}

	if features.HasIssuer {
		score += w.Issuer
		result.Details["签发人"] = w.Issuer
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到签发人: "+features.Issuer)
	}

	// ============================================================
	// 主体特征评分
	// ============================================================

	if features.HasTitle {
		score += w.Title
		result.Details["公文标题"] = w.Title
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到公文标题格式")

		if features.TitleType != "" {
			score += w.TitleType
			result.Details["标题文种"] = w.TitleType
			result.PositiveFactors = append(result.PositiveFactors,
				"标题文种: "+features.TitleType)
		}
	}

	if features.HasMainSend {
		score += w.MainSend
		result.Details["主送机关"] = w.MainSend
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到主送机关")
	}

	if features.HasAttachment {
		score += w.Attachment
		result.Details["附件说明"] = w.Attachment
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到附件说明")
	}

	// ============================================================
	// 版记特征评分
	// ============================================================

	if features.HasIssueDate {
		score += w.IssueDate
		result.Details["成文日期"] = w.IssueDate
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到成文日期: "+features.IssueDate)
	}

	if features.HasCopyTo {
		score += w.CopyTo
		result.Details["抄送"] = w.CopyTo
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到抄送信息")
	}

	if features.HasPrintInfo {
		score += w.PrintInfo
		result.Details["印发信息"] = w.PrintInfo
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到印发信息")
	}

	// ============================================================
	// 机关特征评分
	// ============================================================

	if features.HasOrgName {
		orgScore := w.OrgName
		if len(features.OrgNames) > 3 {
			orgScore = w.OrgName * 1.2
		}
		score += orgScore
		result.Details["机关名称"] = orgScore
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到机关名称")
	}

	// ============================================================
	// 关键词评分
	// ============================================================

	if len(features.DocTypes) > 0 {
		docTypeScore := w.DocType * float64(min(len(features.DocTypes), 3)) / 3.0
		score += docTypeScore
		result.Details["公文文种词"] = docTypeScore
	}

	if len(features.ActionWords) > 0 {
		actionScore := w.ActionWord * float64(min(len(features.ActionWords), 3)) / 3.0
		score += actionScore
		result.Details["公文动作词"] = actionScore
	}

	if len(features.FormalWords) > 0 {
		formalScore := w.FormalWord * float64(min(len(features.FormalWords), 3)) / 3.0
		score += formalScore
		result.Details["正式用语"] = formalScore
	}

	if len(features.HeaderWords) > 0 {
		headerScore := w.HeaderWord * float64(min(len(features.HeaderWords), 3)) / 3.0
		score += headerScore
		result.Details["版头关键词"] = headerScore
	}

	if len(features.FooterWords) > 0 {
		footerScore := w.FooterWord * float64(min(len(features.FooterWords), 3)) / 3.0
		score += footerScore
		result.Details["版记关键词"] = footerScore
	}

	// 非公文特征惩罚
	if len(features.ProhibitWords) > 0 {
		penaltyRatio := float64(len(features.ProhibitWords)) / 5.0
		if penaltyRatio > 1.0 {
			penaltyRatio = 1.0
		}
		penalty := w.Prohibited * penaltyRatio
		score -= penalty
		result.Details["非公文特征惩罚"] = -penalty
		result.NegativeFactors = append(result.NegativeFactors,
			"检测到非公文特征词")
	}

	// 归一化到 0-1
	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0
	}

	return score
}

// scoreStyleFeatures 计算版式特征得分
func (s *Scorer) scoreStyleFeatures(features *extractor.Features, result *ScoreResult) float64 {
	if features.StyleFeatures == nil {
		return 0
	}

	w := s.config.Weights
	sf := features.StyleFeatures
	score := 0.0

	// ============================================================
	// 颜色特征
	// ============================================================

	if sf.HasRedText {
		score += w.RedText
		result.Details["红色文本"] = w.RedText
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到红色文本")
	}

	if sf.HasRedHeader {
		score += w.RedHeader
		result.Details["红头标志"] = w.RedHeader
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到红头（顶部红色文本）")
	}

	// ============================================================
	// 字体特征
	// ============================================================

	if sf.HasOfficialFonts {
		score += w.OfficialFonts
		result.Details["公文字体"] = w.OfficialFonts
		result.PositiveFactors = append(result.PositiveFactors,
			"使用公文标准字体")
	}

	if sf.TitleFontMatch {
		score += w.TitleFont
		result.Details["标题字号"] = w.TitleFont
		result.PositiveFactors = append(result.PositiveFactors,
			"标题字号符合标准（二号）")
	}

	if sf.BodyFontMatch {
		score += w.BodyFont
		result.Details["正文字号"] = w.BodyFont
		result.PositiveFactors = append(result.PositiveFactors,
			"正文字号符合标准（三号）")
	}

	// ============================================================
	// 页面特征
	// ============================================================

	if sf.IsA4Paper {
		score += w.A4Paper
		result.Details["A4纸张"] = w.A4Paper
		result.PositiveFactors = append(result.PositiveFactors,
			"纸张为A4规格")
	}

	if sf.MarginMatch {
		score += w.Margins
		result.Details["页边距"] = w.Margins
		result.PositiveFactors = append(result.PositiveFactors,
			"页边距符合公文标准")
	}

	// ============================================================
	// 段落特征
	// ============================================================

	if sf.HasCenteredTitle {
		score += w.CenteredTitle
		result.Details["居中标题"] = w.CenteredTitle
		result.PositiveFactors = append(result.PositiveFactors,
			"存在居中标题")
	}

	if sf.LineSpacingMatch {
		score += w.LineSpacing
		result.Details["行距"] = w.LineSpacing
		result.PositiveFactors = append(result.PositiveFactors,
			"行距符合标准（28磅）")
	}

	// ============================================================
	// 印章特征
	// ============================================================

	if sf.HasSealImage {
		score += w.SealImage
		result.Details["印章图片"] = w.SealImage
		result.PositiveFactors = append(result.PositiveFactors,
			"检测到可能的印章图片")
	}

	// 归一化到 0-1
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// applyAdjustments 应用调整（加成和惩罚）
func (s *Scorer) applyAdjustments(features *extractor.Features, result *ScoreResult) {
	// ============================================================
	// 文本长度调整
	// ============================================================

	if features.TextLength < 100 {
		penalty := 0.15
		result.TotalScore -= penalty
		result.Details["文本过短惩罚"] = -penalty
		result.NegativeFactors = append(result.NegativeFactors, "文本长度过短")
	} else if features.TextLength < 200 {
		penalty := 0.08
		result.TotalScore -= penalty
		result.Details["文本较短惩罚"] = -penalty
		result.NegativeFactors = append(result.NegativeFactors, "文本长度较短")
	}

	// ============================================================
	// 核心特征组合加分
	// ============================================================

	// 文本核心三要素
	if features.HasDocNumber && features.HasTitle && features.HasIssueDate {
		bonus := 0.06
		result.TotalScore += bonus
		result.Details["文本核心特征组合"] = bonus
		result.PositiveFactors = append(result.PositiveFactors,
			"具备公文核心三要素(发文字号+标题+日期)")
	}

	// 份号+发文字号组合加分 (新增)
	if features.HasCopyNumber && features.HasDocNumber {
		bonus := 0.03
		result.TotalScore += bonus
		result.Details["份号发文字号组合"] = bonus
		result.PositiveFactors = append(result.PositiveFactors,
			"具备份号和发文字号")
	}

	// 版式核心特征
	if features.StyleFeatures != nil {
		sf := features.StyleFeatures
		if sf.HasRedHeader && sf.HasSealImage {
			bonus := 0.08
			result.TotalScore += bonus
			result.Details["版式核心特征组合"] = bonus
			result.PositiveFactors = append(result.PositiveFactors,
				"具备公文版式核心特征(红头+印章)")
		}

		// 红头+红色文本+A4纸
		if sf.HasRedHeader && sf.HasRedText && sf.IsA4Paper {
			bonus := 0.05
			result.TotalScore += bonus
			result.Details["版式特征组合加分"] = bonus
			result.PositiveFactors = append(result.PositiveFactors,
				"版式特征组合(红头+红色文本+A4)")
		}

		// ============================================================
		// 图片类型版式特征加分（红头+印章+A4 三要素）
		// ============================================================

		// 判断是否为图片类型：通过 StyleReasons 中是否包含 OCR 标记
		isImageType := false
		for _, reason := range sf.StyleReasons {
			if reason == "通过OCR提取文本内容" {
				isImageType = true
				break
			}
		}

		if isImageType {
			// 图片类型，版式特征更重要
			// 红头+印章+A4 三要素齐全，大幅加分
			if sf.HasRedHeader && sf.HasSealImage && sf.IsA4Paper {
				bonus := 0.15
				result.TotalScore += bonus
				result.Details["图片版式三要素加分"] = bonus
				result.PositiveFactors = append(result.PositiveFactors,
					"图片公文版式三要素齐全(红头+印章+A4)")
			} else if sf.HasRedHeader && sf.HasSealImage {
				// 红头+印章
				bonus := 0.10
				result.TotalScore += bonus
				result.Details["图片版式双要素加分"] = bonus
				result.PositiveFactors = append(result.PositiveFactors,
					"图片公文版式双要素(红头+印章)")
			} else if sf.HasRedHeader && sf.IsA4Paper {
				// 红头+A4
				bonus := 0.06
				result.TotalScore += bonus
				result.Details["图片红头A4加分"] = bonus
				result.PositiveFactors = append(result.PositiveFactors,
					"图片公文版式(红头+A4)")
			}

			// 图片类型减少文本过短惩罚（OCR 识别本身有局限）
			if features.TextLength < 200 && features.TextLength >= 50 {
				// 已经扣了分，这里补回一部分
				compensation := 0.05
				result.TotalScore += compensation
				result.Details["图片OCR补偿"] = compensation
				result.PositiveFactors = append(result.PositiveFactors,
					"图片OCR识别文本量有限，给予补偿")
			}
		}
	}

	// ============================================================
	// 文本+版式综合加分
	// ============================================================

	if features.HasDocNumber && features.StyleFeatures != nil && features.StyleFeatures.HasRedHeader {
		bonus := 0.05
		result.TotalScore += bonus
		result.Details["文本版式综合加分"] = bonus
		result.PositiveFactors = append(result.PositiveFactors,
			"文本和版式特征相互印证(发文字号+红头)")
	}
}

// getConfidenceLevel 获取置信度等级
func (s *Scorer) getConfidenceLevel(score float64) string {
	switch {
	case score >= 0.85:
		return "很高"
	case score >= 0.7:
		return "高"
	case score >= 0.55:
		return "中"
	case score >= 0.4:
		return "低"
	default:
		return "很低"
	}
}

// generateReasons 生成判定理由
func (s *Scorer) generateReasons(result *ScoreResult, features *extractor.Features) []string {
	var reasons []string

	if result.IsOfficialDoc {
		reasons = append(reasons, "综合得分超过阈值，判定为公文")

		// 文本特征理由
		if features.HasCopyNumber {
			reasons = append(reasons, "包含份号: "+features.CopyNumber)
		}
		if features.HasDocNumber {
			reasons = append(reasons, "包含标准发文字号格式")
		}
		if features.HasTitle && features.TitleType != "" {
			reasons = append(reasons, "包含公文标题格式("+features.TitleType+")")
		}
		if features.HasOrgName {
			reasons = append(reasons, "包含党政机关名称")
		}

		// 版式特征理由
		if features.StyleFeatures != nil {
			sf := features.StyleFeatures
			if sf.HasRedHeader {
				reasons = append(reasons, "检测到红头版式")
			}
			if sf.HasSealImage {
				reasons = append(reasons, "检测到印章图片")
			}
			if sf.IsA4Paper && sf.MarginMatch {
				reasons = append(reasons, "页面设置符合公文规范")
			}
		}
	} else {
		reasons = append(reasons, "综合得分未达到阈值，判定为非公文")

		// 缺失的关键特征
		if !features.HasDocNumber {
			reasons = append(reasons, "缺少发文字号")
		}
		if !features.HasTitle {
			reasons = append(reasons, "缺少公文标题格式")
		}
		if !features.HasIssueDate {
			reasons = append(reasons, "缺少成文日期")
		}

		// 版式问题
		if features.StyleFeatures != nil {
			sf := features.StyleFeatures
			if !sf.HasRedHeader && !sf.HasRedText {
				reasons = append(reasons, "未检测到红色版式元素")
			}
		}

		if len(features.ProhibitWords) > 0 {
			reasons = append(reasons, "包含非公文特征内容")
		}
	}

	return reasons
}

// ============================================================
// 辅助函数
// ============================================================

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ============================================================
// 快捷函数
// ============================================================

// ScoreFeatures 快捷函数：使用默认配置评分
func ScoreFeatures(features *extractor.Features) *ScoreResult {
	scorer := New(nil)
	return scorer.Score(features)
}

// ScoreText 快捷函数：直接对文本评分
func ScoreText(text string) *ScoreResult {
	features := extractor.ExtractFeatures(text)
	return ScoreFeatures(features)
}

// ScoreTextWithThreshold 快捷函数：使用指定阈值对文本评分
func ScoreTextWithThreshold(text string, threshold float64) *ScoreResult {
	features := extractor.ExtractFeatures(text)
	config := DefaultConfig()
	config.Threshold = threshold
	scorer := New(config)
	return scorer.Score(features)
}

// QuickScore 快速评分：只返回分数和判定结果
func QuickScore(text string, threshold float64) (score float64, isOfficialDoc bool) {
	result := ScoreTextWithThreshold(text, threshold)
	return result.TotalScore, result.IsOfficialDoc
}