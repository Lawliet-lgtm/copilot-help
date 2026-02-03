package model

// ==========================================
// 检测策略下发 - 数据模型
// ==========================================

// ==========================================
// 主结构体定义
// ==========================================

// PolicyRequest 检测策略下发请求结构体
type PolicyRequest struct {
	// 声明为检测策略指令，固定值"policy"
	Type string `json:"type" binding:"required,eq=policy"`

	// 检测策略对应的模块名
	Module string `json:"module" binding:"required,oneof=keyword_detect md5_detect"`

	// 策略对应版本号
	Version string `json:"version" binding:"required,max=64"`

	// 策略下发类型
	Cmd string `json:"cmd" binding:"required,oneof=add del reset"`

	// 策略个数
	Num int `json:"num" binding:"required,min=0"`

	// 新的策略内容，根据模块不同内容不同
	Config interface{} `json:"config"`
}

// ==========================================
// 策略内容结构体定义
// ==========================================

// FilterFileSize 文件大小过滤配置
type FilterFileSize struct {
	// 最小文件大小（单位：KB）
	MinSize int `json:"min_size"`
	// 最大文件大小（单位：KB）
	MaxSize int `json:"max_size"`
}

// KeywordDetectRule 关键词检测策略规则
type KeywordDetectRule struct {
	// 策略ID，必填，数值，不超过20位数字的整数
	RuleID int64 `json:"rule_id" binding:"required"`
	// 策略内容，必填，字符串，关键词规则为关键词表达式（括号为键词式表达）
	RuleContent string `json:"rule_content" binding:"required"`
	// 策略描述，可选，字符串，最长128
	RuleDesc string `json:"rule_desc,omitempty" binding:"max=128"`
	// 最少命中参数，可选，数值，不填默认1
	MinMatchCount int `json:"min_match_count,omitempty"`
	// 过滤文件类型，可选，数值数组，不选默认空，表示对文件类型不做要求
	FilterFileType []int `json:"filter_file_type,omitempty"`
	// 过滤文件大小，可选，对象类型，不选默认null，表示对文件大小不做要求
	FilterFileSize *FilterFileSize `json:"filter_file_size,omitempty"`
	// 扩展字段集合，可选，json格式，由厂商根据市场需求增加的内容
	ExtendedFields map[string]interface{} `json:"extended_fields,omitempty"`
}

// KeywordDetectConfig 关键词检测策略配置
type KeywordDetectConfig struct {
	// 关键词检测策略规则列表
	Rules []KeywordDetectRule `json:"rules"`
}

// HashDetectRule 文件哈希检测策略规则
type HashDetectRule struct {
	// 策略ID，必填，数值，不超过20位数字的整数
	RuleID int64 `json:"rule_id" binding:"required"`
	// 策略内容类型，必填，数值型：0.md5，1.sm3
	RuleType int `json:"rule_type" binding:"required,oneof=0 1"`
	// 策略内容，必填，字符串，最长128
	RuleContent string `json:"rule_content" binding:"required,max=128"`
	// 策略描述，可选，字符串，最长128
	RuleDesc string `json:"rule_desc,omitempty" binding:"max=128"`
	// 扩展字段集合，可选，json格式，由厂商根据市场需求增加的内容
	ExtendedFields map[string]interface{} `json:"extended_fields,omitempty"`
}

// HashDetectConfig 文件哈希检测策略配置
type HashDetectConfig struct {
	// 文件哈希检测策略规则列表
	Rules []HashDetectRule `json:"rules"`
}

// SecretLevelDetectRule 密级标志检测策略规则
type SecretLevelDetectRule struct {
	// 策略ID，必填，数值，不超过20位数字的整数
	RuleID int64 `json:"rule_id" binding:"required"`
	// 策略内容，必填，字符串，密级标志关键词
	RuleContent string `json:"rule_content" binding:"required"`
	// 策略描述，可选，字符串，最长128
	RuleDesc string `json:"rule_desc,omitempty" binding:"max=128"`
	// 敏感级别，必填，数值，1-5
	SensitivityLevel int `json:"sensitivity_level" binding:"required,min=1,max=5"`
	// 过滤文件类型，可选，数值数组，不选默认空，表示对文件类型不做要求
	FilterFileType []int `json:"filter_file_type,omitempty"`
	// 过滤文件大小，可选，对象类型，不选默认null，表示对文件大小不做要求
	FilterFileSize *FilterFileSize `json:"filter_file_size,omitempty"`
	// 扩展字段集合，可选，json格式，由厂商根据市场需求增加的内容
	ExtendedFields map[string]interface{} `json:"extended_fields,omitempty"`
}

// SecretLevelDetectConfig 密级标志检测策略配置
type SecretLevelDetectConfig struct {
	// 密级标志检测策略规则列表
	Rules []SecretLevelDetectRule `json:"rules"`
	// OCR配置，可选
	OCRConfig *OCRConfig `json:"ocr_config,omitempty"`
}

// OCRConfig OCR配置
type OCRConfig struct {
	// 是否启用OCR
	Enabled bool `json:"enabled"`
	// OCR引擎类型
	EngineType string `json:"engine_type,omitempty"`
	// 置信度阈值
	ConfidenceThreshold int `json:"confidence_threshold,omitempty"`
}

// ElectronicSecretDetectRule 电子密级标志检测策略规则
type ElectronicSecretDetectRule struct {
	// 策略ID，必填，数值，不超过20位数字的整数
	RuleID int64 `json:"rule_id" binding:"required"`
	// 策略内容，必填，字符串，电子密级标志特征
	RuleContent string `json:"rule_content" binding:"required"`
	// 策略描述，可选，字符串，最长128
	RuleDesc string `json:"rule_desc,omitempty" binding:"max=128"`
	// 敏感级别，必填，数值，1-5
	SensitivityLevel int `json:"sensitivity_level" binding:"required,min=1,max=5"`
	// 检测算法参数，可选
	AlgorithmParams map[string]interface{} `json:"algorithm_params,omitempty"`
	// 扩展字段集合，可选，json格式，由厂商根据市场需求增加的内容
	ExtendedFields map[string]interface{} `json:"extended_fields,omitempty"`
}

// ElectronicSecretDetectConfig 电子密级标志检测策略配置
type ElectronicSecretDetectConfig struct {
	// 电子密级标志检测策略规则列表
	Rules []ElectronicSecretDetectRule `json:"rules"`
	// 图像处理配置
	ImageConfig *ImageConfig `json:"image_config,omitempty"`
}

// ImageConfig 图像处理配置
type ImageConfig struct {
	// 最大处理图像尺寸
	MaxImageSize int `json:"max_image_size,omitempty"`
	// 检测精度
	DetectionPrecision string `json:"detection_precision,omitempty"`
}

// OfficialFormatDetectRule 公文版式检测策略规则
type OfficialFormatDetectRule struct {
	// 策略ID，必填，数值，不超过20位数字的整数
	RuleID int64 `json:"rule_id" binding:"required"`
	// 策略内容，必填，字符串，公文版式特征
	RuleContent string `json:"rule_content" binding:"required"`
	// 策略描述，可选，字符串，最长128
	RuleDesc string `json:"rule_desc,omitempty" binding:"max=128"`
	// 敏感级别，必填，数值，1-5
	SensitivityLevel int `json:"sensitivity_level" binding:"required,min=1,max=5"`
	// 版式元素要求
	FormatElements []string `json:"format_elements,omitempty"`
	// 扩展字段集合，可选，json格式，由厂商根据市场需求增加的内容
	ExtendedFields map[string]interface{} `json:"extended_fields,omitempty"`
}

// OfficialFormatDetectConfig 公文版式检测策略配置
type OfficialFormatDetectConfig struct {
	// 公文版式检测策略规则列表
	Rules []OfficialFormatDetectRule `json:"rules"`
	// 版式检测配置
	FormatConfig *FormatConfig `json:"format_config,omitempty"`
}

// FormatConfig 版式检测配置
type FormatConfig struct {
	// 文档类型
	DocumentType string `json:"document_type,omitempty"`
	// 格式偏差容忍度
	ToleranceLevel int `json:"tolerance_level,omitempty"`
}

// ==========================================
// 响应结构体定义
// ==========================================

// PolicyResponse 检测策略下发响应结构体
type PolicyResponse struct {
	// 返回信息类型: 0 代表成功, 1 代表失败
	Type int `json:"type"`

	// 返回消息内容
	Message string `json:"message"`
}

// ==========================================
// 辅助构造函数
// ==========================================

// NewFilterFileSize 创建新的文件大小过滤配置
func NewFilterFileSize(minSize, maxSize int) *FilterFileSize {
	return &FilterFileSize{
		MinSize: minSize,
		MaxSize: maxSize,
	}
}

// NewKeywordDetectRule 创建新的关键词检测策略规则
func NewKeywordDetectRule(ruleID int64, ruleContent string) *KeywordDetectRule {
	return &KeywordDetectRule{
		RuleID:         ruleID,
		RuleContent:    ruleContent,
		MinMatchCount:  1, // 默认值1
		FilterFileType: make([]int, 0),
		ExtendedFields: make(map[string]interface{}),
	}
}

// NewKeywordDetectConfig 创建新的关键词检测策略配置
func NewKeywordDetectConfig() *KeywordDetectConfig {
	return &KeywordDetectConfig{
		Rules: make([]KeywordDetectRule, 0),
	}
}

// NewHashDetectRule 创建新的文件哈希检测策略规则
func NewHashDetectRule(ruleID int64, ruleType int, ruleContent string) *HashDetectRule {
	return &HashDetectRule{
		RuleID:         ruleID,
		RuleType:       ruleType,
		RuleContent:    ruleContent,
		ExtendedFields: make(map[string]interface{}),
	}
}

// NewHashDetectConfig 创建新的文件哈希检测策略配置
func NewHashDetectConfig() *HashDetectConfig {
	return &HashDetectConfig{
		Rules: make([]HashDetectRule, 0),
	}
}

// ==========================================
// 密级标志检测策略辅助构造函数
// ==========================================

// NewSecretLevelDetectRule 创建新的密级标志检测策略规则
func NewSecretLevelDetectRule(ruleID int64, ruleContent string, sensitivityLevel int) *SecretLevelDetectRule {
	return &SecretLevelDetectRule{
		RuleID:           ruleID,
		RuleContent:      ruleContent,
		SensitivityLevel: sensitivityLevel,
		FilterFileType:   make([]int, 0),
		ExtendedFields:   make(map[string]interface{}),
	}
}

// NewSecretLevelDetectConfig 创建新的密级标志检测策略配置
func NewSecretLevelDetectConfig() *SecretLevelDetectConfig {
	return &SecretLevelDetectConfig{
		Rules: make([]SecretLevelDetectRule, 0),
	}
}

// NewOCRConfig 创建新的OCR配置
func NewOCRConfig(enabled bool) *OCRConfig {
	return &OCRConfig{
		Enabled:             enabled,
		EngineType:          "default",
		ConfidenceThreshold: 80,
	}
}

// ==========================================
// 电子密级标志检测策略辅助构造函数
// ==========================================

// NewElectronicSecretDetectRule 创建新的电子密级标志检测策略规则
func NewElectronicSecretDetectRule(ruleID int64, ruleContent string, sensitivityLevel int) *ElectronicSecretDetectRule {
	return &ElectronicSecretDetectRule{
		RuleID:           ruleID,
		RuleContent:      ruleContent,
		SensitivityLevel: sensitivityLevel,
		AlgorithmParams:  make(map[string]interface{}),
		ExtendedFields:   make(map[string]interface{}),
	}
}

// NewElectronicSecretDetectConfig 创建新的电子密级标志检测策略配置
func NewElectronicSecretDetectConfig() *ElectronicSecretDetectConfig {
	return &ElectronicSecretDetectConfig{
		Rules: make([]ElectronicSecretDetectRule, 0),
	}
}

// NewImageConfig 创建新的图像处理配置
func NewImageConfig() *ImageConfig {
	return &ImageConfig{
		MaxImageSize:       2048,
		DetectionPrecision: "medium",
	}
}

// ==========================================
// 公文版式检测策略辅助构造函数
// ==========================================

// NewOfficialFormatDetectRule 创建新的公文版式检测策略规则
func NewOfficialFormatDetectRule(ruleID int64, ruleContent string, sensitivityLevel int) *OfficialFormatDetectRule {
	return &OfficialFormatDetectRule{
		RuleID:           ruleID,
		RuleContent:      ruleContent,
		SensitivityLevel: sensitivityLevel,
		FormatElements:   make([]string, 0),
		ExtendedFields:   make(map[string]interface{}),
	}
}

// NewOfficialFormatDetectConfig 创建新的公文版式检测策略配置
func NewOfficialFormatDetectConfig() *OfficialFormatDetectConfig {
	return &OfficialFormatDetectConfig{
		Rules: make([]OfficialFormatDetectRule, 0),
	}
}

// NewFormatConfig 创建新的版式检测配置
func NewFormatConfig() *FormatConfig {
	return &FormatConfig{
		DocumentType:   "official",
		ToleranceLevel: 2,
	}
}

// NewPolicyRequest 创建新的检测策略请求
func NewPolicyRequest(module, version, cmd string, num int, config interface{}) *PolicyRequest {
	return &PolicyRequest{
		Type:    PolicyType,
		Module:  module,
		Version: version,
		Cmd:     cmd,
		Num:     num,
		Config:  config,
	}
}

// NewPolicyResponse 创建新的检测策略响应
func NewPolicyResponse(success bool, message string) *PolicyResponse {
	responseType := 0
	if !success {
		responseType = 1
	}
	return &PolicyResponse{
		Type:    responseType,
		Message: message,
	}
}
