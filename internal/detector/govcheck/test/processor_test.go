package test

import (
	"fmt"
	"testing"

	"linuxFileWatcher/internal/detector/govcheck/processor"
)

// MockTextProcessor 模拟文本处理器 (用于测试)
type MockTextProcessor struct {
	*processor.BaseProcessor
}

func NewMockTextProcessor() *MockTextProcessor {
	return &MockTextProcessor{
		BaseProcessor: processor.NewBaseProcessor(
			"MockTextProcessor",
			"模拟文本处理器(测试用)",
			[]string{"txt", "text"},
		),
	}
}

func (p *MockTextProcessor) Process(filePath string) (string, error) {
	return "这是模拟提取的文本内容", nil
}

// MockDocProcessor 模拟文档处理器 (用于测试)
type MockDocProcessor struct {
	*processor.BaseProcessor
}

func NewMockDocProcessor() *MockDocProcessor {
	return &MockDocProcessor{
		BaseProcessor: processor.NewBaseProcessor(
			"MockDocProcessor",
			"模拟文档处理器(测试用)",
			[]string{"docx", "doc"},
		),
	}
}

func (p *MockDocProcessor) Process(filePath string) (string, error) {
	return "模拟DOCX文档内容：关于印发xxx的通知", nil
}

func TestProcessorRegistry(t *testing.T) {
	fmt.Println("========================================")
	fmt.Println("          处理器注册表测试")
	fmt.Println("========================================")

	// 创建注册表
	registry := processor.NewRegistry()

	// 注册处理器
	registry.Register(NewMockTextProcessor())
	registry.Register(NewMockDocProcessor())

	// 列出所有支持的类型
	fmt.Println("\n支持的文件类型:")
	for _, ext := range registry.SupportedTypes() {
		fmt.Printf("  - %s\n", ext)
	}

	// 列出所有处理器
	fmt.Println("\n已注册的处理器:")
	for _, p := range registry.List() {
		fmt.Printf("  - %s: %s\n", p.Name(), p.Description())
		fmt.Printf("    支持: %v\n", p.SupportedTypes())
	}

	// 测试获取处理器
	fmt.Println("\n处理器查找测试:")

	testCases := []string{"txt", "TXT", ".txt", "docx", "pdf", "unknown"}
	for _, ext := range testCases {
		p, ok := registry.Get(ext)
		if ok {
			fmt.Printf("  [✓] %s -> %s\n", ext, p.Name())
		} else {
			fmt.Printf("  [✗] %s -> 未找到\n", ext)
		}
	}
}

func TestProcessorInterface(t *testing.T) {
	fmt.Println("\n========================================")
	fmt.Println("          处理器接口测试")
	fmt.Println("========================================")

	// 创建处理器
	textProc := NewMockTextProcessor()

	// 测试接口方法
	fmt.Printf("名称: %s\n", textProc.Name())
	fmt.Printf("描述: %s\n", textProc.Description())
	fmt.Printf("支持类型: %v\n", textProc.SupportedTypes())

	// 测试处理
	text, err := textProc.Process("test.txt")
	if err != nil {
		t.Errorf("处理失败: %v", err)
	}
	fmt.Printf("处理结果: %s\n", text)
}

func TestProcessorError(t *testing.T) {
	fmt.Println("\n========================================")
	fmt.Println("          处理器错误测试")
	fmt.Println("========================================")

	// 创建错误
	err := processor.NewProcessorError(
		"TestProcessor",
		"/path/to/file.txt",
		"读取文件",
		fmt.Errorf("文件不存在"),
	)

	fmt.Printf("错误信息: %s\n", err.Error())
	fmt.Printf("原始错误: %v\n", err.Unwrap())
}

func TestBaseProcessor(t *testing.T) {
	fmt.Println("\n========================================")
	fmt.Println("          基础处理器测试")
	fmt.Println("========================================")

	// 测试扩展名规范化
	base := processor.NewBaseProcessor(
		"TestProcessor",
		"测试处理器",
		[]string{".TXT", "Doc", ".DOCX"},
	)

	fmt.Printf("名称: %s\n", base.Name())
	fmt.Printf("描述: %s\n", base.Description())
	fmt.Printf("规范化后的扩展名: %v\n", base.SupportedTypes())
}

func TestDefaultRegistry(t *testing.T) {
	fmt.Println("\n========================================")
	fmt.Println("          默认注册表测试")
	fmt.Println("========================================")

	// 注册到默认注册表
	processor.RegisterDefault(NewMockTextProcessor())

	// 从默认注册表获取
	p, ok := processor.GetDefault("txt")
	if ok {
		fmt.Printf("从默认注册表获取: %s\n", p.Name())
	}

	// 获取支持的类型
	types := processor.SupportedTypesDefault()
	fmt.Printf("默认注册表支持: %v\n", types)
}