# 公文版式检测工具 - API 文档

## 目录

1. [核心包](#核心包)
2. [检测器 (detector)](#检测器-detector)
3. [特征提取 (extractor)](#特征提取-extractor)
4. [评分器 (scorer)](#评分器-scorer)
5. [处理器 (processor)](#处理器-processor)
6. [配置 (config)](#配置-config)
7. [错误处理 (errors)](#错误处理-errors)

---

## 核心包

```
linuxFileWatcher/
├── internal/detector/govcheck/
│   ├── detector/     # 检测器核心
│   ├── extractor/    # 特征提取
│   ├── scorer/       # 评分逻辑
│   └── processor/    # 文件处理器
└── pkg/
    ├── config/       # 配置管理
    ├── errors/       # 错误处理
    └── fileutil/     # 文件工具
```

---

## 检测器 (detector)

### Detector

主检测器，组合特征提取和评分功能。

```go
import "linuxFileWatcher/internal/detector/govcheck/detector"

// 创建检测器
det := detector.New(nil)                    // 使用默认配置
det := detector.NewWithConfig(config)       // 使用自定义配置

// 注册处理器
det.RegisterProcessor(processor)

// 检测单个文件
result := det.Detect(filePath)

// 批量检测
results := det.DetectBatch(files)

// 并行批量检测
results := det.DetectBatchParallel(files, workers)
```

### DetectorConfig

```go
type DetectorConfig struct {
    Threshold float64  // 判定阈值 (0-1)
    Verbose   bool     // 详细模式
}

// 默认配置
config := detector.DefaultConfig()
```

### DetectionResult

```go
type DetectionResult struct {
    FilePath    string   // 文件路径
    FileType    string   // 文件类型
    FileSize    int64    // 文件大小
    IsOfficial  bool     // 是否为公文
    Confidence  float64  // 置信度
    TextScore   float64  // 文本特征得分
    StyleScore  float64  // 版式特征得分
    Features    *Features // 提取的特征
    ProcessTime Duration // 处理耗时
    Error       string   // 错误信息
}

// 获取摘要
summary := result.Summary()
verboseSummary := result.VerboseSummary()
```

---

## 特征提取 (extractor)

### Extractor

从文本中提取公文特征。

```go
import "linuxFileWatcher/internal/detector/govcheck/extractor"

// 创建提取器
ext := extractor.New(nil)                   // 默认配置
ext := extractor.NewWithConfig(config)      // 自定义配置

// 提取特征
features := ext.Extract(text)

// 带版式特征提取
features := ext.ExtractWithStyle(text, styleFeatures)
```

### Features

```go
type Features struct {
    // 版头特征
    SerialNumber    string   // 份号
    DocNumber       string   // 发文字号
    SecretLevel     string   // 密级
    UrgencyLevel    string   // 紧急程度
    Signer          string   // 签发人

    // 主体特征
    Title           string   // 标题
    TitleType       string   // 标题文种
    Recipient       string   // 主送机关
    HasAttachment   bool     // 是否有附件

    // 版记特征
    Date            string   // 成文日期
    HasSeal         bool     // 是否有印章
    HasCC           bool     // 是否有抄送
    HasPrintInfo    bool     // 是否有印发信息

    // 机关特征
    HasOrganization bool     // 是否有机关名称
    Organizations   []string // 识别的机关

    // 版式特征
    StyleFeatures   *StyleFeatures
}
```

### StyleFeatures

```go
type StyleFeatures struct {
    HasRedText       bool     // 有红色文本
    HasRedHeader     bool     // 有红头
    RedTextCount     int      // 红色文本数量
    HasOfficialFonts bool     // 使用公文字体
    TitleFontMatch   bool     // 标题字号匹配
    BodyFontMatch    bool     // 正文字号匹配
    IsA4Paper        bool     // A4 纸张
    MarginMatch      bool     // 页边距匹配
    HasCenteredTitle bool     // 居中标题
    HasSealImage     bool     // 有印章图片
    StyleScore       float64  // 版式得分
}
```

---

## 评分器 (scorer)

### Scorer

根据特征计算公文得分。

```go
import "linuxFileWatcher/internal/detector/govcheck/scorer"

// 创建评分器
s := scorer.New(nil)                        // 默认配置
s := scorer.NewWithConfig(config)           // 自定义配置

// 评分
result := s.Score(features)
```

### ScorerConfig

```go
type ScorerConfig struct {
    TextWeight  float64  // 文本特征权重 (默认 0.7)
    StyleWeight float64  // 版式特征权重 (默认 0.3)
}
```

### ScoreResult

```go
type ScoreResult struct {
    TotalScore  float64           // 总分
    TextScore   float64           // 文本特征得分
    StyleScore  float64           // 版式特征得分
    Details     []ScoreDetail     // 得分明细
    IsOfficial  bool              // 是否判定为公文
}

type ScoreDetail struct {
    Name   string   // 特征名称
    Score  float64  // 得分
    Reason string   // 得分原因
}
```

---

## 处理器 (processor)

### Processor 接口

所有文件处理器必须实现此接口。

```go
type Processor interface {
    Name() string                           // 处理器名称
    Description() string                    // 处理器描述
    SupportedTypes() []string               // 支持的文件类型
    Process(filePath string) (string, error) // 处理文件
}

// 支持版式特征的处理器
type StyleProcessor interface {
    Processor
    ProcessWithStyle(filePath string) (*ProcessResultWithStyle, error)
}
```

### 内置处理器

```go
import "linuxFileWatcher/internal/detector/govcheck/processor"

// 文本处理器
processor.NewTextProcessor()

// DOCX 处理器
processor.NewDocxProcessor()

// DOC 处理器
processor.NewDocProcessor()

// WPS 处理器
processor.NewWpsProcessor()

// PDF 处理器
processor.NewPdfProcessor()

// OFD 处理器
processor.NewOfdProcessor()

// 图片处理器
processor.NewImageProcessor()
```

### 处理器注册表

```go
// 创建注册表
registry := processor.NewRegistry()

// 注册处理器
registry.Register(processor.NewTextProcessor())

// 获取处理器
p, ok := registry.GetByType("docx")

// 获取所有支持的类型
types := registry.SupportedTypes()
```

---

## 配置 (config)

### Config

```go
import "linuxFileWatcher/pkg/config"

// 默认配置
cfg := config.Default()

// 从文件加载
cfg, err := config.Load("config.json")

// 保存到文件
err := cfg.Save("config.json")

// 验证配置
err := cfg.Validate()
```

### 配置结构

```go
type Config struct {
    Detection DetectionConfig
    OCR       OCRConfig
    Output    OutputConfig
    Scoring   ScoringConfig
}

type DetectionConfig struct {
    Threshold          float64
    Workers            int
    Timeout            int
    MaxFileSize        int64
    ExcludeExtensions  []string
    ExcludeDirectories []string
}

type OCRConfig struct {
    Enabled  bool
    Language string
    DPI      int
}

type OutputConfig struct {
    Format  string  // "text" 或 "json"
    Verbose bool
    Color   bool
}

type ScoringConfig struct {
    TextWeight  float64
    StyleWeight float64
}
```

---

## 错误处理 (errors)

### DetectorError

```go
import "linuxFileWatcher/pkg/errors"

// 创建错误
err := errors.NewDetectorError(errors.ErrFileNotFound, "文件不存在")

// 链式设置
err = err.
    WithFile("/path/to/file").
    WithComponent("DocProcessor").
    WithCause(originalError)

// 获取用户友好消息
message := err.UserMessage()

// 检查错误类型
if errors.IsFileError(err) {
    // 处理文件错误
}
```

### 错误代码

```go
// 通用错误 (1000-1999)
errors.ErrUnknown
errors.ErrInvalidInput
errors.ErrTimeout

// 文件错误 (2000-2999)
errors.ErrFileNotFound
errors.ErrFileEmpty
errors.ErrFileTooLarge
errors.ErrFileFormat

// 处理器错误 (3000-3999)
errors.ErrProcessorNotFound
errors.ErrProcessorFailed
errors.ErrExternalToolMissing

// 配置错误 (4000-4999)
errors.ErrConfigInvalid
errors.ErrConfigValue

// 检测错误 (5000-5999)
errors.ErrDetectionFailed
errors.ErrNoContent
```

### 错误收集器

```go
// 创建收集器
coll := errors.NewErrorCollection()

// 添加错误
coll.AddError(err)

// 检查状态
if coll.HasErrors() {
    fmt.Println(coll.Summary())
}

// 获取所有错误
for _, e := range coll.Errors() {
    fmt.Println(e.UserMessage())
}
```

### 安全执行

```go
// 带 panic 恢复的执行
err := errors.SafeExecute(func() error {
    // 可能 panic 的代码
    return nil
})

// 带返回值的安全执行
result, err := errors.SafeExecuteWithResult(func() (string, error) {
    return "result", nil
})

// 重试执行
err := errors.Retry(func() error {
    return someOperation()
}, errors.DefaultRetryConfig())
```