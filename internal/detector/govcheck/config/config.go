package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 应用配置
type Config struct {
	// 版本信息
	Version string `json:"version"`

	// 检测配置
	Detection DetectionConfig `json:"detection"`

	// OCR 配置
	OCR OCRConfig `json:"ocr"`

	// 输出配置
	Output OutputConfig `json:"output"`

	// 特征权重配置
	Weights WeightsConfig `json:"weights"`
}

// DetectionConfig 检测配置
type DetectionConfig struct {
	// 判定阈值 (0-1)，默认 0.6
	Threshold float64 `json:"threshold"`

	// 文本特征权重占比 (0-1)，默认 0.55
	TextWeight float64 `json:"text_weight"`

	// 版式特征权重占比 (0-1)，默认 0.45
	StyleWeight float64 `json:"style_weight"`

	// 最大文件大小限制 (字节)，默认 100MB
	MaxFileSize int64 `json:"max_file_size"`

	// 并行处理的工作线程数，默认 CPU 核心数
	Workers int `json:"workers"`

	// 是否递归处理子目录，默认 true
	Recursive bool `json:"recursive"`

	// 要排除的文件扩展名
	ExcludeExtensions []string `json:"exclude_extensions,omitempty"`

	// 要排除的目录名
	ExcludeDirectories []string `json:"exclude_directories,omitempty"`
}

// OCRConfig OCR 配置
type OCRConfig struct {
	// 是否启用 OCR，默认 true
	Enabled bool `json:"enabled"`

	// Tesseract 可执行文件路径，默认从 PATH 查找
	TesseractPath string `json:"tesseract_path,omitempty"`

	// OCR 语言，默认 "chi_sim+eng"
	Language string `json:"language"`

	// OCR 超时时间 (秒)，默认 30
	Timeout int `json:"timeout"`
}

// OutputConfig 输出配置
type OutputConfig struct {
	// 输出格式: "text" 或 "json"，默认 "text"
	Format string `json:"format"`

	// 是否显示详细信息，默认 false
	Verbose bool `json:"verbose"`

	// 是否显示颜色，默认 true
	Color bool `json:"color"`

	// 日志级别: "debug", "info", "warn", "error"，默认 "info"
	LogLevel string `json:"log_level"`
}

// WeightsConfig 特征权重配置
type WeightsConfig struct {
	// 文本特征权重
	Text TextWeightsConfig `json:"text"`

	// 版式特征权重
	Style StyleWeightsConfig `json:"style"`
}

// TextWeightsConfig 文本特征权重
type TextWeightsConfig struct {
	// 版头特征
	CopyNumber   float64 `json:"copy_number"`   // 份号
	DocNumber    float64 `json:"doc_number"`    // 发文字号
	SecretLevel  float64 `json:"secret_level"`  // 密级
	UrgencyLevel float64 `json:"urgency_level"` // 紧急程度
	Issuer       float64 `json:"issuer"`        // 签发人

	// 主体特征
	Title      float64 `json:"title"`      // 公文标题
	TitleType  float64 `json:"title_type"` // 标题文种
	MainSend   float64 `json:"main_send"`  // 主送机关
	Attachment float64 `json:"attachment"` // 附件

	// 版记特征
	IssueDate float64 `json:"issue_date"` // 成文日期
	CopyTo    float64 `json:"copy_to"`    // 抄送
	PrintInfo float64 `json:"print_info"` // 印发信息

	// 机关特征
	OrgName float64 `json:"org_name"` // 机关名称

	// 关键词
	DocType    float64 `json:"doc_type"`    // 公文文种关键词
	ActionWord float64 `json:"action_word"` // 公文动作词
	FormalWord float64 `json:"formal_word"` // 正式用语
	HeaderWord float64 `json:"header_word"` // 版头关键词
	FooterWord float64 `json:"footer_word"` // 版记关键词
	Prohibited float64 `json:"prohibited"`  // 非公文特征惩罚
}

// StyleWeightsConfig 版式特征权重
type StyleWeightsConfig struct {
	RedText       float64 `json:"red_text"`       // 红色文本
	RedHeader     float64 `json:"red_header"`     // 红头
	OfficialFonts float64 `json:"official_fonts"` // 公文字体
	TitleFont     float64 `json:"title_font"`     // 标题字号
	BodyFont      float64 `json:"body_font"`      // 正文字号
	A4Paper       float64 `json:"a4_paper"`       // A4纸张
	Margins       float64 `json:"margins"`        // 页边距
	CenteredTitle float64 `json:"centered_title"` // 居中标题
	LineSpacing   float64 `json:"line_spacing"`   // 行距
	SealImage     float64 `json:"seal_image"`     // 印章图片
}

// Load 从文件加载配置
func Load(filePath string) (*Config, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", filePath)
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 先加载默认配置
	config := Default()

	// 解析 JSON
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return config, nil
}

// LoadOrDefault 加载配置文件，如果不存在则返回默认配置
func LoadOrDefault(filePath string) *Config {
	if filePath == "" {
		return Default()
	}

	config, err := Load(filePath)
	if err != nil {
		// 加载失败，返回默认配置
		return Default()
	}

	return config
}

// Save 保存配置到文件
func (c *Config) Save(filePath string) error {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}
	}

	// 序列化为 JSON（格式化输出）
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	// 验证阈值
	if c.Detection.Threshold < 0 || c.Detection.Threshold > 1 {
		return fmt.Errorf("detection.threshold 必须在 0-1 之间，当前值: %v", c.Detection.Threshold)
	}

	// 验证权重
	if c.Detection.TextWeight < 0 || c.Detection.TextWeight > 1 {
		return fmt.Errorf("detection.text_weight 必须在 0-1 之间，当前值: %v", c.Detection.TextWeight)
	}

	if c.Detection.StyleWeight < 0 || c.Detection.StyleWeight > 1 {
		return fmt.Errorf("detection.style_weight 必须在 0-1 之间，当前值: %v", c.Detection.StyleWeight)
	}

	// 验证权重总和（允许小误差）
	totalWeight := c.Detection.TextWeight + c.Detection.StyleWeight
	if totalWeight < 0.99 || totalWeight > 1.01 {
		return fmt.Errorf("detection.text_weight + detection.style_weight 应等于 1.0，当前值: %v", totalWeight)
	}

	// 验证文件大小限制
	if c.Detection.MaxFileSize <= 0 {
		return fmt.Errorf("detection.max_file_size 必须大于 0，当前值: %v", c.Detection.MaxFileSize)
	}

	// 验证工作线程数
	if c.Detection.Workers < 0 {
		return fmt.Errorf("detection.workers 不能为负数，当前值: %v", c.Detection.Workers)
	}

	// 验证 OCR 超时
	if c.OCR.Timeout <= 0 {
		return fmt.Errorf("ocr.timeout 必须大于 0，当前值: %v", c.OCR.Timeout)
	}

	// 验证输出格式
	if c.Output.Format != "text" && c.Output.Format != "json" {
		return fmt.Errorf("output.format 必须是 'text' 或 'json'，当前值: %v", c.Output.Format)
	}

	// 验证日志级别
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Output.LogLevel] {
		return fmt.Errorf("output.log_level 无效，当前值: %v", c.Output.LogLevel)
	}

	return nil
}

// Clone 创建配置的深拷贝
func (c *Config) Clone() *Config {
	clone := *c

	// 复制切片
	if c.Detection.ExcludeExtensions != nil {
		clone.Detection.ExcludeExtensions = make([]string, len(c.Detection.ExcludeExtensions))
		copy(clone.Detection.ExcludeExtensions, c.Detection.ExcludeExtensions)
	}

	if c.Detection.ExcludeDirectories != nil {
		clone.Detection.ExcludeDirectories = make([]string, len(c.Detection.ExcludeDirectories))
		copy(clone.Detection.ExcludeDirectories, c.Detection.ExcludeDirectories)
	}

	return &clone
}

// String 返回配置的字符串表示
func (c *Config) String() string {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Sprintf("Config{error: %v}", err)
	}
	return string(data)
}