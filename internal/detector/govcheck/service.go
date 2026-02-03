package govcheck

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	globalModel "linuxFileWatcher/internal/model"

	"linuxFileWatcher/internal/detector/govcheck/detector"
	"linuxFileWatcher/internal/detector/govcheck/processor"
)

// 公文版式检测规则 ID
const (
	RuleIDGovCheck int64 = 3001 // 公文版式检测规则 ID
)

// service 公文版式检测服务实现
type service struct {
	config   Config
	detector *detector.Detector
}

// newService 创建服务实例
func newService(cfg Config) *service {
	// 验证配置
	if cfg.Threshold <= 0 || cfg.Threshold > 1 {
		cfg.Threshold = 0.6
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}
	if cfg.MaxFileSize <= 0 {
		cfg.MaxFileSize = 100 * 1024 * 1024
	}

	// 创建内部检测器
	detConfig := detector.DefaultConfig()
	detConfig.Threshold = cfg.Threshold
	detConfig.Verbose = cfg.Verbose

	det := detector.New(detConfig)

	// 注册处理器
	registerProcessors(det, cfg)

	return &service{
		config:   cfg,
		detector: det,
	}
}

// registerProcessors 注册所有文件处理器
func registerProcessors(det *detector.Detector, cfg Config) {
	// 文本处理器
	det.RegisterProcessor(processor.NewTextProcessor())

	// DOCX 处理器
	det.RegisterProcessor(processor.NewDocxProcessor())

	// DOC 处理器
	det.RegisterProcessor(processor.NewDocProcessor())

	// WPS 处理器
	det.RegisterProcessor(processor.NewWpsProcessor())

	// PDF 处理器
	det.RegisterProcessor(processor.NewPdfProcessor())

	// OFD 处理器
	det.RegisterProcessor(processor.NewOfdProcessor())

	// 图片处理器（如果启用 OCR）
	if cfg.EnableOCR {
		imgConfig := processor.DefaultImageProcessorConfig()
		if cfg.OCRLanguage != "" {
			imgConfig.OcrLang = cfg.OCRLanguage
		}
		imgProcessor := processor.NewImageProcessorWithConfig(imgConfig)
		if imgProcessor.IsOcrAvailable() {
			det.RegisterProcessor(imgProcessor)
		}
	}
}

// DetectFile 实现 Detector 接口
func (s *service) DetectFile(ctx context.Context, filePath string) (*globalModel.SubDetectResult, error) {
	// 1. 验证文件
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("文件不存在: %s", filePath)
		}
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("路径是目录而非文件: %s", filePath)
	}

	if fileInfo.Size() == 0 {
		return nil, nil // 空文件跳过
	}

	if s.config.MaxFileSize > 0 && fileInfo.Size() > s.config.MaxFileSize {
		return nil, nil // 文件过大跳过
	}

	// 2. 获取文件类型
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if ext == "" {
		return nil, nil // 无扩展名跳过
	}

	// 检查是否支持该文件类型
	supportedTypes := s.detector.SupportedTypes()
	if !isTypeSupported(ext, supportedTypes) {
		return nil, nil // 不支持的类型跳过
	}

	// 3. 设置超时控制
	var detectCtx context.Context
	var cancel context.CancelFunc

	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		detectCtx = ctx
		cancel = func() {}
	} else {
		detectCtx, cancel = context.WithTimeout(ctx, time.Duration(s.config.Timeout)*time.Second)
	}
	defer cancel()

	// 4. 执行检测（带 panic 恢复）
	var result *detector.DetectionResult
	var detectErr error

	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				detectErr = fmt.Errorf("检测过程发生异常: %v", r)
			}
			close(done)
		}()
		result = s.detector.Detect(filePath)
	}()

	// 等待检测完成或超时
	select {
	case <-done:
		// 检测完成
	case <-detectCtx.Done():
		// 超时
		return nil, nil
	}

	if detectErr != nil {
		return nil, detectErr
	}

	// 5. 结果转换
	if result == nil {
		return nil, nil
	}

	// 检测失败
	if !result.Success || result.Error != "" {
		return nil, nil
	}

	// 未达到阈值，不是公文
	if !result.IsOfficialDoc {
		return nil, nil
	}

	// 6. 构造返回结果
	subResult := &globalModel.SubDetectResult{
		IsSecret:    true,
		SecretLevel: globalModel.LevelInternal, // 公文默认内部级别
		RuleID:      RuleIDGovCheck,            // int64 类型
		RuleDesc:    s.buildRuleDesc(result),
		MatchedText: s.buildMatchedText(result),
		ContextText: s.buildContextText(result),
		AlertType:   3, // 公文版式告警类型
	}

	return subResult, nil
}

// buildRuleDesc 构建规则描述
func (s *service) buildRuleDesc(result *detector.DetectionResult) string {
	if result.Features == nil {
		return fmt.Sprintf("公文版式检测命中 (置信度: %.1f%%)", result.Confidence*100)
	}

	var parts []string

	// 添加文种信息
	if result.Features.TitleType != "" {
		parts = append(parts, fmt.Sprintf("文种: %s", result.Features.TitleType))
	}

	// 添加发文机关
	if result.Features.HasOrgName && len(result.Features.OrgNames) > 0 {
		parts = append(parts, fmt.Sprintf("发文机关: %s", strings.Join(result.Features.OrgNames, "、")))
	}

	// 添加置信度
	parts = append(parts, fmt.Sprintf("置信度: %.1f%%", result.Confidence*100))

	if len(parts) > 0 {
		return fmt.Sprintf("公文版式检测命中 (%s)", strings.Join(parts, "; "))
	}

	return "公文版式检测命中"
}

// buildMatchedText 构建匹配文本（用于高亮显示）
func (s *service) buildMatchedText(result *detector.DetectionResult) string {
	if result.Features == nil {
		return ""
	}

	var highlights []string

	// 发文字号
	if result.Features.HasDocNumber && result.Features.DocNumber != "" {
		highlights = append(highlights, result.Features.DocNumber)
	}

	// 标题
	if result.Features.HasTitle && result.Features.Title != "" {
		highlights = append(highlights, result.Features.Title)
	}

	// 成文日期
	if result.Features.HasIssueDate && result.Features.IssueDate != "" {
		highlights = append(highlights, result.Features.IssueDate)
	}

	return strings.Join(highlights, " | ")
}

// buildContextText 构建上下文文本
func (s *service) buildContextText(result *detector.DetectionResult) string {
	if result.Features == nil {
		return ""
	}

	var context []string

	// 份号
	if result.Features.HasCopyNumber && result.Features.CopyNumber != "" {
		context = append(context, fmt.Sprintf("份号: %s", result.Features.CopyNumber))
	}

	// 发文字号
	if result.Features.HasDocNumber && result.Features.DocNumber != "" {
		context = append(context, fmt.Sprintf("发文字号: %s", result.Features.DocNumber))
	}

	// 密级
	if result.Features.HasSecretLevel && result.Features.SecretLevel != "" {
		context = append(context, fmt.Sprintf("密级: %s", result.Features.SecretLevel))
	}

	// 标题
	if result.Features.HasTitle && result.Features.Title != "" {
		context = append(context, fmt.Sprintf("标题: %s", result.Features.Title))
	}

	// 主送机关
	if result.Features.HasMainSend && result.Features.MainSend != "" {
		context = append(context, fmt.Sprintf("主送机关: %s", result.Features.MainSend))
	}

	// 成文日期
	if result.Features.HasIssueDate && result.Features.IssueDate != "" {
		context = append(context, fmt.Sprintf("成文日期: %s", result.Features.IssueDate))
	}

	// 识别机关
	if result.Features.HasOrgName && len(result.Features.OrgNames) > 0 {
		context = append(context, fmt.Sprintf("识别机关: %s", strings.Join(result.Features.OrgNames, "、")))
	}

	// 抄送
	if result.Features.HasCopyTo && result.Features.CopyTo != "" {
		context = append(context, fmt.Sprintf("抄送: %s", result.Features.CopyTo))
	}

	// 印发信息
	if result.Features.HasPrintInfo && result.Features.PrintInfo != "" {
		context = append(context, fmt.Sprintf("印发: %s", result.Features.PrintInfo))
	}

	return strings.Join(context, "\n")
}

// isTypeSupported 检查文件类型是否支持
func isTypeSupported(fileType string, supportedTypes []string) bool {
	for _, t := range supportedTypes {
		if t == fileType {
			return true
		}
	}
	return false
}