package test

import (
	"fmt"
	"testing"

	"linuxFileWatcher/internal/detector/govcheck/scorer"
)

func TestScorer(t *testing.T) {
	// 测试用例1：标准公文
	officialDoc := `
国务院办公厅关于印发《xxx实施方案》的通知
国办发〔2024〕1号

各省、自治区、直辖市人民政府，国务院各部委、各直属机构：

	为贯彻落实党中央、国务院关于xxx的决策部署，现将《xxx实施方案》印发给你们，请认真贯彻执行。

	附件：xxx实施方案

国务院办公厅
2024年1月15日

抄送：各省委办公厅、省政府办公厅。
国务院办公厅 2024年1月15日印发
`

	// 测试用例2：普通文本
	normalText := `
这是一篇普通的文章，讨论一些日常话题。
今天天气很好，适合出去散步。
我们去公园看看花吧。文章写于2024年1月15日。
`

	// 测试用例3：商业文档
	commercialDoc := `
产品报价单

尊敬的客户：
感谢您对我们产品的关注，以下是报价信息：
产品名称：xxx
产品价格：1000元
折扣：9折
咨询热线：400-xxx-xxxx
欢迎来电咨询！
`

	// 测试用例4：部分公文特征
	partialDoc := `
关于做好2024年安全生产工作的通知

各部门：
为进一步加强安全生产工作，现将有关事项通知如下：
一、提高认识
二、落实责任
三、加强检查

2024年1月10日
`

	fmt.Println("========================================")
	fmt.Println("          评分引擎测试")
	fmt.Println("========================================")

	testCases := []struct {
		name string
		text string
	}{
		{"标准公文", officialDoc},
		{"普通文本", normalText},
		{"商业文档", commercialDoc},
		{"部分公文特征", partialDoc},
	}

	for _, tc := range testCases {
		fmt.Printf("\n--- %s ---\n", tc.name)
		result := scorer.ScoreText(tc.text)
		printScoreResult(result)
	}
}

func printScoreResult(result *scorer.ScoreResult) {
	fmt.Printf("总分: %.2f (%.0f%%)\n", result.TotalScore, result.TotalScore*100)
	fmt.Printf("阈值: %.2f\n", result.Threshold)
	fmt.Printf("判定: %v\n", formatJudgment(result.IsOfficialDoc))
	fmt.Printf("置信度: %s\n", result.Confidence)

	fmt.Println("\n得分明细:")
	for name, score := range result.Details {
		if score > 0 {
			fmt.Printf("  [+] %s: +%.2f\n", name, score)
		} else {
			fmt.Printf("  [-] %s: %.2f\n", name, score)
		}
	}

	if len(result.PositiveFactors) > 0 {
		fmt.Println("\n正向因素:")
		for _, f := range result.PositiveFactors {
			fmt.Printf("  ✓ %s\n", f)
		}
	}

	if len(result.NegativeFactors) > 0 {
		fmt.Println("\n负向因素:")
		for _, f := range result.NegativeFactors {
			fmt.Printf("  ✗ %s\n", f)
		}
	}

	fmt.Println("\n判定理由:")
	for _, r := range result.Reasons {
		fmt.Printf("  • %s\n", r)
	}
}

func formatJudgment(isOfficial bool) string {
	if isOfficial {
		return "✓ 是公文"
	}
	return "✗ 不是公文"
}

func TestQuickScore(t *testing.T) {
	fmt.Println("\n========================================")
	fmt.Println("          快速评分测试")
	fmt.Println("========================================")

	text := `国务院关于印发xxx的通知 国发〔2024〕1号 2024年1月1日`

	score, isOfficial := scorer.QuickScore(text, 0.6)
	fmt.Printf("文本: %s\n", text)
	fmt.Printf("得分: %.2f\n", score)
	fmt.Printf("是否公文: %v\n", isOfficial)
}