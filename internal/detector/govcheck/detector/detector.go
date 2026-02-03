package detector

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"linuxFileWatcher/internal/detector/govcheck/extractor"
	"linuxFileWatcher/internal/detector/govcheck/fileutil"
	"linuxFileWatcher/internal/detector/govcheck/processor"
	"linuxFileWatcher/internal/detector/govcheck/scorer"
)

// Detector 公文检测器
type Detector struct {
	config     *Config
	processors map[string]Processor
	extractor  *extractor.Extractor
	scorer     *scorer.Scorer
	mu         sync.RWMutex
}

// Config 检测器配置
type Config struct {
	Threshold   float64       // 判定阈值 (0-1)
	Verbose     bool          // 详细模式
	MaxFileSize int64         // 最大文件大小限制 (字节)
	Timeout     time.Duration // 单文件处理超时
}

// Processor 文件处理器接口
type Processor interface {
	Process(filePath string) (string, error)
	SupportedTypes() []string
}

// StyleProcessor 支持版式解析的处理器接口
type StyleProcessor interface {
	Processor
	ProcessWithStyle(filePath string) (*processor.ProcessResultWithStyle, error)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Threshold:   0.6,
		Verbose:     false,
		MaxFileSize: 100 * 1024 * 1024, // 100MB
		Timeout:     30 * time.Second,
	}
}

// New 创建一个新的检测器
func New(config *Config) *Detector {
	if config == nil {
		config = DefaultConfig()
	}

	if config.MaxFileSize <= 0 {
		config.MaxFileSize = 100 * 1024 * 1024
	}

	scorerConfig := scorer.DefaultConfig()
	scorerConfig.Threshold = config.Threshold

	return &Detector{
		config:     config,
		processors: make(map[string]Processor),
		extractor:  extractor.New(nil),
		scorer:     scorer.New(scorerConfig),
	}
}

// RegisterProcessor 注册文件处理器
func (d *Detector) RegisterProcessor(p Processor) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, ext := range p.SupportedTypes() {
		ext = strings.ToLower(strings.TrimPrefix(ext, "."))
		d.processors[ext] = p
	}
}

// GetProcessor 获取指定文件类型的处理器
func (d *Detector) GetProcessor(fileType string) (Processor, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ext := strings.ToLower(strings.TrimPrefix(fileType, "."))
	p, ok := d.processors[ext]
	return p, ok
}

// SupportedTypes 返回所有支持的文件类型
func (d *Detector) SupportedTypes() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	types := make([]string, 0, len(d.processors))
	for ext := range d.processors {
		types = append(types, ext)
	}
	return types
}

// Detect 检测单个文件
func (d *Detector) Detect(filePath string) *DetectionResult {
	startTime := time.Now()

	// 获取文件信息
	fileInfo, err := fileutil.GetFileInfo(filePath)
	if err != nil {
		result := NewDetectionResult(filePath, "", 0)
		result.SetError(fmt.Errorf("无法获取文件信息: %w", err))
		return result
	}

	// 创建结果对象
	result := NewDetectionResult(
		fileInfo.Path,
		fileInfo.Name,
		fileInfo.Size,
	)
	result.Threshold = d.config.Threshold
	result.FileType = fileInfo.Type.Extension

	// 检查文件是否为空
	if fileInfo.Size == 0 {
		result.SetError(fmt.Errorf("文件为空"))
		return result
	}

	// 检查文件大小
	if d.config.MaxFileSize > 0 && fileInfo.Size > d.config.MaxFileSize {
		result.SetError(fmt.Errorf("文件过大: %d 字节 (限制: %d 字节)",
			fileInfo.Size, d.config.MaxFileSize))
		return result
	}

	// 检查文件类型是否支持公文检测
	if !fileutil.IsSupportedForDetection(fileInfo.Type) {
		reason := fileutil.GetUnsupportedReason(fileInfo.Type)
		result.SetError(fmt.Errorf("不支持此文件类型进行公文检测: %s (%s)",
			fileInfo.Type.Description, reason))
		result.ProcessTime = time.Since(startTime)
		return result
	}

	// 获取处理器
	proc, ok := d.GetProcessor(fileInfo.Type.Extension)
	if !ok {
		result.SetError(fmt.Errorf("暂未实现此格式的处理器: %s (%s)",
			fileInfo.Type.Extension, fileInfo.Type.Description))
		result.ProcessTime = time.Since(startTime)
		return result
	}

	// 处理文件
	var textContent string
	var styleFeatures *extractor.StyleFeatures

	// 检查处理器是否支持版式解析
	if styleProc, ok := proc.(StyleProcessor); ok {
		// 使用带版式的处理方法
		styleResult, err := styleProc.ProcessWithStyle(filePath)
		if err != nil {
			result.SetError(fmt.Errorf("处理文件失败: %w", err))
			result.ProcessTime = time.Since(startTime)
			return result
		}
		textContent = styleResult.Text
		if styleResult.HasStyle {
			styleFeatures = styleResult.StyleFeatures
		}
	} else {
		// 使用普通处理方法
		text, err := proc.Process(filePath)
		if err != nil {
			result.SetError(fmt.Errorf("处理文件失败: %w", err))
			result.ProcessTime = time.Since(startTime)
			return result
		}
		textContent = text
	}

	// 提取特征
	var features *extractor.Features
	if styleFeatures != nil {
		features = d.extractor.ExtractWithStyle(textContent, styleFeatures)
	} else {
		features = d.extractor.Extract(textContent)
	}

	// 评分
	scoreResult := d.scorer.Score(features)

	// 填充结果
	result.IsOfficialDoc = scoreResult.IsOfficialDoc
	result.Confidence = scoreResult.TotalScore
	result.TextScore = scoreResult.TextScore
	result.StyleScore = scoreResult.StyleScore

	// 填充特征详情
	d.fillFeatureResult(result, features, scoreResult)

	result.SetSuccess()
	result.ProcessTime = time.Since(startTime)

	return result
}

// fillFeatureResult 填充特征检测结果
func (d *Detector) fillFeatureResult(result *DetectionResult, features *extractor.Features, scoreResult *scorer.ScoreResult) {
	// 文本特征
	result.Features.HasDocNumber = features.HasDocNumber
	result.Features.DocNumber = features.DocNumber
	result.Features.HasSecretLevel = features.HasSecretLevel
	result.Features.SecretLevel = features.SecretLevel
	result.Features.HasUrgencyLevel = features.HasUrgencyLevel
	result.Features.UrgencyLevel = features.UrgencyLevel
	result.Features.HasIssuer = features.HasIssuer
	result.Features.Issuer = features.Issuer
	result.Features.HasTitle = features.HasTitle
	result.Features.Title = features.Title
	result.Features.TitleType = features.TitleType
	result.Features.HasMainSend = features.HasMainSend
	result.Features.MainSend = features.MainSend
	result.Features.HasAttachment = features.HasAttachment
	result.Features.AttachmentInfo = features.Attachment
	result.Features.HasIssueDate = features.HasIssueDate
	result.Features.IssueDate = features.IssueDate
	result.Features.HasCopyTo = features.HasCopyTo
	result.Features.CopyTo = features.CopyTo
	result.Features.HasPrintInfo = features.HasPrintInfo
	result.Features.PrintInfo = features.PrintInfo
	result.Features.HasOrgName = features.HasOrgName
	result.Features.OrgNames = features.OrgNames

	// 版式特征
	if features.StyleFeatures != nil {
		sf := features.StyleFeatures
		result.Features.HasRedHeader = sf.HasRedHeader
		result.Features.HasSeal = sf.HasSealImage

		result.Features.StyleFeatures = &StyleFeatureResult{
			HasRedText:       sf.HasRedText,
			HasRedHeader:     sf.HasRedHeader,
			RedTextCount:     sf.RedTextCount,
			RedSamples:       sf.RedSamples,
			HasOfficialFonts: sf.HasOfficialFonts,
			TitleFontMatch:   sf.TitleFontMatch,
			BodyFontMatch:    sf.BodyFontMatch,
			MainFontName:     sf.MainFontName,
			IsA4Paper:        sf.IsA4Paper,
			MarginMatch:      sf.MarginMatch,
			HasCenteredTitle: sf.HasCenteredTitle,
			LineSpacingMatch: sf.LineSpacingMatch,
			HasSealImage:     sf.HasSealImage,
			SealImageHint:    sf.SealImageHint,
			StyleScore:       sf.StyleScore,
			StyleReasons:     sf.StyleReasons,
		}

		// 页面尺寸描述
		if sf.PageWidth > 0 && sf.PageHeight > 0 {
			result.Features.StyleFeatures.PageSize = fmt.Sprintf("%.0f×%.0fmm", sf.PageWidth, sf.PageHeight)
		}

		// 字号描述
		if sf.MainFontSize > 0 {
			result.Features.StyleFeatures.MainFontSize = fmt.Sprintf("%.1fpt", sf.MainFontSize)
		}
	}

	// 得分明细
	result.Features.ScoreDetails = scoreResult.Details
}

// DetectBatch 批量检测多个文件
func (d *Detector) DetectBatch(filePaths []string) []*DetectionResult {
	results := make([]*DetectionResult, len(filePaths))

	for i, filePath := range filePaths {
		results[i] = d.Detect(filePath)
	}

	return results
}

// DetectBatchParallel 并行批量检测
func (d *Detector) DetectBatchParallel(filePaths []string, workers int) []*DetectionResult {
	if workers <= 0 {
		workers = 1
	}
	if workers > len(filePaths) {
		workers = len(filePaths)
	}

	results := make([]*DetectionResult, len(filePaths))
	var wg sync.WaitGroup

	tasks := make(chan int, len(filePaths))

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range tasks {
				results[i] = d.Detect(filePaths[i])
			}
		}()
	}

	for i := range filePaths {
		tasks <- i
	}
	close(tasks)

	wg.Wait()

	return results
}

// BatchResult 批量检测结果统计
type BatchResult struct {
	Total           int                `json:"total"`
	Success         int                `json:"success"`
	Failed          int                `json:"failed"`
	OfficialDocs    int                `json:"official_docs"`
	NonOfficialDocs int                `json:"non_official_docs"`
	TotalTime       time.Duration      `json:"total_time_ns"`
	Results         []*DetectionResult `json:"results"`
}

// NewBatchResult 从检测结果列表创建批量结果
func NewBatchResult(results []*DetectionResult, totalTime time.Duration) *BatchResult {
	br := &BatchResult{
		Total:     len(results),
		TotalTime: totalTime,
		Results:   results,
	}

	for _, r := range results {
		if r.Success {
			br.Success++
			if r.IsOfficialDoc {
				br.OfficialDocs++
			} else {
				br.NonOfficialDocs++
			}
		} else {
			br.Failed++
		}
	}

	return br
}

// Summary 返回批量结果摘要
func (br *BatchResult) Summary() string {
	var sb strings.Builder

	sb.WriteString("========================================\n")
	sb.WriteString("              检测结果汇总\n")
	sb.WriteString("========================================\n")
	sb.WriteString(fmt.Sprintf("总文件数:     %d\n", br.Total))
	sb.WriteString(fmt.Sprintf("处理成功:     %d\n", br.Success))
	sb.WriteString(fmt.Sprintf("处理失败:     %d\n", br.Failed))
	sb.WriteString(fmt.Sprintf("判定为公文:   %d\n", br.OfficialDocs))
	sb.WriteString(fmt.Sprintf("判定非公文:   %d\n", br.NonOfficialDocs))
	sb.WriteString(fmt.Sprintf("总耗时:       %v\n", br.TotalTime))
	sb.WriteString("========================================\n")

	return sb.String()
}