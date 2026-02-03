package processor

import (
	"fmt"
	"strings"
	"sync"

	"linuxFileWatcher/internal/detector/govcheck/extractor"
	"linuxFileWatcher/internal/detector/govcheck/scorer"
	"linuxFileWatcher/internal/detector/govcheck/errors"
)

// ============================================================
// 基础处理器
// ============================================================

// BaseProcessor 基础处理器
type BaseProcessor struct {
	name        string
	description string
	types       []string
}

// NewBaseProcessor 创建基础处理器
func NewBaseProcessor(name, description string, types []string) *BaseProcessor {
	// 规范化扩展名
	normalized := make([]string, len(types))
	for i, ext := range types {
		normalized[i] = normalizeExtension(ext)
	}

	return &BaseProcessor{
		name:        name,
		description: description,
		types:       normalized,
	}
}

// Name 返回处理器名称
func (p *BaseProcessor) Name() string {
	return p.name
}

// Description 返回处理器描述
func (p *BaseProcessor) Description() string {
	return p.description
}

// SupportedTypes 返回支持的文件类型
func (p *BaseProcessor) SupportedTypes() []string {
	return p.types
}

// normalizeExtension 规范化扩展名
func normalizeExtension(ext string) string {
	ext = strings.ToLower(ext)
	ext = strings.TrimPrefix(ext, ".")
	return ext
}

// ============================================================
// 处理器接口定义
// ============================================================

// Processor 处理器接口
type Processor interface {
	Name() string
	Description() string
	SupportedTypes() []string
	Process(filePath string) (string, error)
}

// StyleProcessor 支持版式特征的处理器接口
type StyleProcessor interface {
	Processor
	ProcessWithStyle(filePath string) (*ProcessResultWithStyle, error)
}

// AdvancedProcessor 高级处理器接口 (可选实现)
type AdvancedProcessor interface {
	Processor
	ProcessWithMetadata(filePath string) (*ProcessResult, error)
	CanProcess(filePath string) bool
	Priority() int
}

// ============================================================
// 处理结果
// ============================================================

// ProcessResult 处理结果
type ProcessResult struct {
	Text     string            // 提取的文本内容
	Metadata map[string]string // 元数据
	Pages    int               // 页数
	Images   []ImageInfo       // 图片信息
	Error    error             // 非致命错误
}

// ImageInfo 图片信息
type ImageInfo struct {
	Index  int
	Width  int
	Height int
	Format string
	Data   []byte
}

// NewProcessResult 创建处理结果
func NewProcessResult(text string) *ProcessResult {
	return &ProcessResult{
		Text:     text,
		Metadata: make(map[string]string),
	}
}

// ============================================================
// 处理器错误定义
// ============================================================

// ProcessorError 处理器错误
type ProcessorError struct {
	Processor string
	FilePath  string
	Operation string
	Err       error
}

// Error 实现 error 接口
func (e *ProcessorError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s - %v", e.Processor, e.FilePath, e.Operation, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Processor, e.FilePath, e.Operation)
}

// Unwrap 返回原始错误
func (e *ProcessorError) Unwrap() error {
	return e.Err
}

// NewProcessorError 创建处理器错误
func NewProcessorError(processor, filePath, operation string, err error) *ProcessorError {
	return &ProcessorError{
		Processor: processor,
		FilePath:  filePath,
		Operation: operation,
		Err:       err,
	}
}

// ToDetectorError 转换为 DetectorError
func (e *ProcessorError) ToDetectorError() *errors.DetectorError {
	return errors.ProcessorError(e.Processor, e.FilePath, e.Operation, e.Err)
}

// ============================================================
// 错误处理辅助函数
// ============================================================

// WrapProcessorError 包装错误为处理器错误
func WrapProcessorError(processor, filePath, operation string, err error) error {
	if err == nil {
		return nil
	}
	return NewProcessorError(processor, filePath, operation, err)
}

// FileValidationError 文件验证错误
func FileValidationError(processor, filePath, reason string) *ProcessorError {
	return NewProcessorError(processor, filePath, "文件验证", fmt.Errorf(reason))
}

// ContentExtractionError 内容提取错误
func ContentExtractionError(processor, filePath, reason string) *ProcessorError {
	return NewProcessorError(processor, filePath, "内容提取", fmt.Errorf(reason))
}

// ParsingError 解析错误
func ParsingError(processor, filePath, reason string) *ProcessorError {
	return NewProcessorError(processor, filePath, "解析", fmt.Errorf(reason))
}

// FormatError 格式错误
func FormatError(processor, filePath, expected, actual string) *ProcessorError {
	return NewProcessorError(processor, filePath, "格式检查",
		fmt.Errorf("期望格式: %s, 实际格式: %s", expected, actual))
}

// FileSizeError 文件大小错误
func FileSizeError(processor, filePath string, size, maxSize int64) *ProcessorError {
	return NewProcessorError(processor, filePath, "大小检查",
		fmt.Errorf("文件大小 %d 字节, 超过限制 %d 字节", size, maxSize))
}

// EmptyFileError 空文件错误
func EmptyFileError(processor, filePath string) *ProcessorError {
	return NewProcessorError(processor, filePath, "文件检查", fmt.Errorf("文件为空"))
}

// ExternalToolError 外部工具错误
func ExternalToolError(processor, filePath, toolName string, err error) *ProcessorError {
	return NewProcessorError(processor, filePath, fmt.Sprintf("调用%s", toolName), err)
}

// ============================================================
// 处理器注册表
// ============================================================

// Registry 处理器注册表
type Registry struct {
	processors map[string]Processor
	typeMap    map[string]Processor
	mu         sync.RWMutex
}

// NewRegistry 创建处理器注册表
func NewRegistry() *Registry {
	return &Registry{
		processors: make(map[string]Processor),
		typeMap:    make(map[string]Processor),
	}
}

// Register 注册处理器
func (r *Registry) Register(p Processor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.processors[p.Name()] = p
	for _, ext := range p.SupportedTypes() {
		ext = normalizeExtension(ext)
		r.typeMap[ext] = p
	}
}

// Get 获取处理器（按名称）
func (r *Registry) Get(name string) (Processor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.processors[name]
	return p, ok
}

// GetByType 根据文件类型获取处理器
func (r *Registry) GetByType(fileType string) (Processor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ext := normalizeExtension(fileType)
	p, ok := r.typeMap[ext]
	return p, ok
}

// Has 检查是否有指定扩展名的处理器
func (r *Registry) Has(ext string) bool {
	_, ok := r.GetByType(ext)
	return ok
}

// SupportedTypes 获取所有支持的文件类型
func (r *Registry) SupportedTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.typeMap))
	for t := range r.typeMap {
		types = append(types, t)
	}
	return types
}

// All 获取所有处理器
func (r *Registry) All() []Processor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 去重
	seen := make(map[string]bool)
	var processors []Processor

	for _, p := range r.processors {
		name := p.Name()
		if !seen[name] {
			seen[name] = true
			processors = append(processors, p)
		}
	}

	return processors
}

// Count 处理器数量
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.processors)
}

// List 返回所有已注册的处理器（All 的别名）
func (r *Registry) List() []Processor {
	return r.All()
}

// ============================================================
// 全局默认注册表
// ============================================================

var defaultRegistry = NewRegistry()

// GetDefaultRegistry 获取默认注册表
func GetDefaultRegistry() *Registry {
	return defaultRegistry
}

// RegisterProcessor 注册处理器到默认注册表
func RegisterProcessor(p Processor) {
	defaultRegistry.Register(p)
}

// GetProcessor 从默认注册表获取处理器
func GetProcessor(name string) (Processor, bool) {
	return defaultRegistry.Get(name)
}

// GetProcessorByType 从默认注册表根据类型获取处理器
func GetProcessorByType(fileType string) (Processor, bool) {
	return defaultRegistry.GetByType(fileType)
}

// RegisterDefault 注册到默认注册表（别名）
func RegisterDefault(p Processor) {
	defaultRegistry.Register(p)
}

// GetDefault 从默认注册表获取处理器（别名）
func GetDefault(ext string) (Processor, bool) {
	return defaultRegistry.GetByType(ext)
}

// SupportedTypesDefault 获取默认注册表支持的类型
func SupportedTypesDefault() []string {
	return defaultRegistry.SupportedTypes()
}

// ============================================================
// 处理器链
// ============================================================

// ProcessorChain 处理器链 (用于复杂处理流程)
type ProcessorChain struct {
	processors []Processor
}

// NewProcessorChain 创建处理器链
func NewProcessorChain(processors ...Processor) *ProcessorChain {
	return &ProcessorChain{
		processors: processors,
	}
}

// Process 依次尝试各处理器直到成功
func (c *ProcessorChain) Process(filePath string) (string, error) {
	var lastErr error

	for _, p := range c.processors {
		text, err := p.Process(filePath)
		if err == nil {
			return text, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return "", fmt.Errorf("所有处理器都失败: %w", lastErr)
	}
	return "", fmt.Errorf("没有可用的处理器")
}

// ============================================================
// 完整处理器（组合提取和评分功能）
// ============================================================

// FullProcessor 完整处理器
type FullProcessor struct {
	registry  *Registry
	extractor *extractor.Extractor
	scorer    *scorer.Scorer
}

// NewFullProcessor 创建完整处理器
func NewFullProcessor(registry *Registry) *FullProcessor {
	return &FullProcessor{
		registry:  registry,
		extractor: extractor.New(nil),
		scorer:    scorer.New(nil),
	}
}

// ProcessAndScore 处理文件并评分
func (fp *FullProcessor) ProcessAndScore(filePath string, fileType string) (*FullResult, error) {
	proc, ok := fp.registry.GetByType(fileType)
	if !ok {
		return nil, fmt.Errorf("不支持的文件类型: %s", fileType)
	}

	text, err := proc.Process(filePath)
	if err != nil {
		return nil, fmt.Errorf("处理文件失败: %w", err)
	}

	features := fp.extractor.Extract(text)
	scoreResult := fp.scorer.Score(features)

	return &FullResult{
		Text:        text,
		Features:    features,
		ScoreResult: scoreResult,
	}, nil
}

// FullResult 完整处理结果
type FullResult struct {
	Text        string
	Features    *extractor.Features
	ScoreResult *scorer.ScoreResult
}

// ============================================================
// 处理器类别和信息
// ============================================================

// Category 处理器类别
type Category int

const (
	CategoryText     Category = iota // 文本类
	CategoryDocument                 // 文档类
	CategoryImage                    // 图片类
)

// String 返回类别名称
func (c Category) String() string {
	switch c {
	case CategoryText:
		return "文本"
	case CategoryDocument:
		return "文档"
	case CategoryImage:
		return "图片"
	default:
		return "未知"
	}
}

// ProcessorInfo 处理器信息
type ProcessorInfo struct {
	Name        string
	Description string
	Category    Category
	Extensions  []string
	Priority    int
}

// GetProcessorInfo 获取处理器信息
func GetProcessorInfo(p Processor) ProcessorInfo {
	info := ProcessorInfo{
		Name:        p.Name(),
		Description: p.Description(),
		Extensions:  p.SupportedTypes(),
	}

	if ap, ok := p.(AdvancedProcessor); ok {
		info.Priority = ap.Priority()
	}

	return info
}