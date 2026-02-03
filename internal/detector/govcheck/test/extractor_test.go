package test

import (
	"fmt"
	"testing"

	"linuxFileWatcher/internal/detector/govcheck/extractor"
)

func TestExtractor(t *testing.T) {
	// 模拟公文文本
	officialDocText := `
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

	// 模拟普通文本
	normalText := `
这是一篇普通的文章，讨论一些日常话题。
今天天气很好，适合出去散步。
我们去公园看看花吧。
`

	// 模拟商业文档
	commercialText := `
产品报价单

尊敬的客户：
感谢您对我们产品的关注，以下是报价信息：
产品名称：xxx
产品价格：1000元
折扣：9折
咨询热线：400-xxx-xxxx
`

	fmt.Println("=== 公文文本分析 ===")
	result1 := extractor.AnalyzeText(officialDocText)
	printAnalysisResult(result1)

	fmt.Println("\n=== 普通文本分析 ===")
	result2 := extractor.AnalyzeText(normalText)
	printAnalysisResult(result2)

	fmt.Println("\n=== 商业文档分析 ===")
	result3 := extractor.AnalyzeText(commercialText)
	printAnalysisResult(result3)
}

func printAnalysisResult(result *extractor.AnalysisResult) {
	f := result.Features

	fmt.Printf("文本长度: %d 字符\n", f.TextLength)
	fmt.Printf("中文字符: %d 个\n", f.ChineseCharCount)
	fmt.Println()

	fmt.Println("[版头特征]")
	fmt.Printf("  发文字号: %v (%s)\n", f.HasDocNumber, f.DocNumber)
	fmt.Printf("  密级标志: %v (%s)\n", f.HasSecretLevel, f.SecretLevel)
	fmt.Printf("  紧急程度: %v (%s)\n", f.HasUrgencyLevel, f.UrgencyLevel)
	fmt.Printf("  签发人:   %v (%s)\n", f.HasIssuer, f.Issuer)
	fmt.Println()

	fmt.Println("[主体特征]")
	fmt.Printf("  公文标题: %v (%s)\n", f.HasTitle, truncate(f.Title, 30))
	fmt.Printf("  标题文种: %s\n", f.TitleType)
	fmt.Printf("  主送机关: %v (%s)\n", f.HasMainSend, truncate(f.MainSend, 30))
	fmt.Printf("  附件说明: %v\n", f.HasAttachment)
	fmt.Println()

	fmt.Println("[版记特征]")
	fmt.Printf("  成文日期: %v (%s)\n", f.HasIssueDate, f.IssueDate)
	fmt.Printf("  抄送:     %v\n", f.HasCopyTo)
	fmt.Printf("  印发信息: %v\n", f.HasPrintInfo)
	fmt.Println()

	fmt.Println("[机关特征]")
	fmt.Printf("  机关名称: %v\n", f.HasOrgName)
	if len(f.OrgNames) > 0 {
		fmt.Printf("  识别机关: %v\n", f.OrgNames)
	}
	fmt.Println()

	fmt.Println("[关键词匹配]")
	fmt.Printf("  公文文种: %v\n", f.DocTypes)
	fmt.Printf("  动作词:   %v\n", f.ActionWords)
	fmt.Printf("  正式用语: %v\n", f.FormalWords)
	if len(f.ProhibitWords) > 0 {
		fmt.Printf("  非公文词: %v\n", f.ProhibitWords)
	}
	fmt.Println()

	fmt.Println("[分析结论]")
	fmt.Printf("  正向特征数: %d\n", result.PositiveCount)
	fmt.Printf("  关键特征:   %v\n", result.HasCritical)
	fmt.Printf("  非公文特征: %v\n", result.HasProhibited)
	fmt.Printf("  快速检查:   %v\n", result.QuickCheckPassed)
	fmt.Printf("  建议深度扫描: %v\n", result.RecommendedForScan)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}