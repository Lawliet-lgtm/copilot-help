package test

import (
	"fmt"
	"testing"

	"linuxFileWatcher/internal/detector/govcheck/rules"
)

func TestPatterns(t *testing.T) {
	// 测试发文字号
	testTexts := []string{
		"国发〔2024〕1号",
		"京政发[2023]15号",
		"教办函（2024）123号",
		"这是普通文本",
	}

	fmt.Println("=== 发文字号测试 ===")
	for _, text := range testTexts {
		result := rules.ExtractDocNumber(text)
		fmt.Printf("文本: %s\n结果: %s\n\n", text, result)
	}

	// 测试公文标题
	titleTexts := []string{
		"关于印发《xxx实施方案》的通知",
		"关于做好2024年工作的意见",
		"这是普通标题",
	}

	fmt.Println("=== 公文标题测试 ===")
	for _, text := range titleTexts {
		result := rules.ExtractTitle(text)
		titleType := rules.ExtractTitleType(result)
		fmt.Printf("文本: %s\n标题: %s\n文种: %s\n\n", text, result, titleType)
	}

	// 测试日期
	dateTexts := []string{
		"2024年1月15日",
		"2023年12月31日",
	}

	fmt.Println("=== 成文日期测试 ===")
	for _, text := range dateTexts {
		result := rules.ExtractIssueDate(text)
		fmt.Printf("文本: %s\n日期: %s\n\n", text, result)
	}
}

func TestKeywords(t *testing.T) {
	// 测试关键词
	testText := `
	国务院办公厅关于印发《xxx实施方案》的通知
	
	各省、自治区、直辖市人民政府，国务院各部委、各直属机构：
	
	为贯彻落实xxx精神，现将《xxx实施方案》印发给你们，请认真贯彻执行。
	
	抄送：各省委办公厅。
	国务院办公厅 2024年1月15日印发
	`

	fmt.Println("=== 关键词分析测试 ===")
	results := rules.AnalyzeKeywords(testText)
	for _, r := range results {
		fmt.Printf("类别: %s\n匹配: %v\n得分: %.2f\n\n", r.Set.Name, r.Matched, r.Score)
	}

	totalScore := rules.CalculateKeywordScore(testText)
	fmt.Printf("总得分: %.2f\n", totalScore)
}