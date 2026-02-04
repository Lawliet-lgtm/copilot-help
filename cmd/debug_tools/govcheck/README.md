# 公文版式检测工具 (Official Document Detector)

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/Version-0.7.0-orange.svg)](CHANGELOG.md)

基于 **GB/T 9704-2012《党政机关公文格式》** 国家标准的公文版式自动检测工具。

## 功能特性

- 🔍 **智能检测**：自动识别文件是否为规范公文格式
- 📄 **多格式支持**：TXT、DOC、DOCX、WPS、PDF、OFD、图片等
- 🎯 **特征提取**：发文字号、标题、主送机关、成文日期等
- 📊 **版式分析**：字体、字号、页边距、纸张大小等
- 🖼️ **OCR 支持**：支持扫描件和图片公文识别
- ⚡ **高性能**：支持批量并行处理
- 🔧 **可配置**：灵活的配置文件支持

## 快速开始

### 安装

```bash
# 克隆项目
git clone https://github.com/yourname/official-doc-detector.git
cd official-doc-detector

# 编译
go build -o detector ./cmd/detector

# Windows
go build -o detector.exe ./cmd/detector
```

### 基本使用

```bash
# 检测单个文件
./detector -file document.pdf

# 检测目录下所有文件
./detector -dir ./documents/

# 显示详细信息
./detector -file document.docx -verbose

# 输出 JSON 格式
./detector -file document.doc -json

# 查看系统状态
./detector -status
```

### 示例输出

```
文件: 关于开展工作的通知.docx
类型: docx
大小: 25.30 KB
状态: 处理成功
耗时: 45.2ms
置信度: 92.33%
阈值: 60.00%
判定: ✓ 是公文

特征检测详情:
─────────────────────────────
[版头特征]
  发文字号: ✓ 国办发〔2024〕1号
  密级标志: ✗ 未检测到
  紧急程度: ✗ 未检测到
[主体特征]
  公文标题: ✓ 关于开展工作的通知
  标题类型: 通知
  主送机关: ✓ 各省、自治区、直辖市人民政府：
[版记特征]
  成文日期: ✓ 2024年1月15日
  印章:     ✓ 是
  抄送:     ✓ 是
```

## 支持的文件格式

| 类型 | 扩展名 | 说明 |
|------|--------|------|
| 文本 | txt, text, html, htm, xml, rtf, mht, mhtml, eml | 纯文本和标记语言 |
| 文档 | doc, docx, docm, dotx, dotm, wps, wpt | Office 和 WPS 文档 |
| PDF | pdf | 便携式文档格式 |
| OFD | ofd | 中国版式文档格式 |
| 图片 | jpg, jpeg, png, gif, bmp, tiff, tif, webp | 需要 OCR 支持 |

## 检测标准

基于 **GB/T 9704-2012《党政机关公文格式》**，检测以下要素：

### 版头要素
- 份号（六位数字）
- 密级和保密期限
- 紧急程度（特急、加急）
- 发文机关标志
- 发文字号

### 主体要素
- 标题（事由 + 文种）
- 主送机关
- 正文
- 附件说明
- 发文机关署名
- 成文日期
- 印章

### 版记要素
- 抄送机关
- 印发机关和印发日期

### 版式要素
- 纸张规格（A4）
- 页边距
- 字体字号
- 行距

## 配置文件

### 生成默认配置

```bash
./detector -gen-config
```

### 配置文件示例 (config.json)

```json
{
  "detection": {
    "threshold": 0.6,
    "workers": 4,
    "timeout": 30,
    "max_file_size": 104857600,
    "exclude_extensions": [".exe", ".dll", ".zip"],
    "exclude_directories": [".git", "node_modules"]
  },
  "ocr": {
    "enabled": true,
    "language": "chi_sim+eng",
    "dpi": 300
  },
  "output": {
    "format": "text",
    "verbose": false,
    "color": true
  },
  "scoring": {
    "text_weight": 0.7,
    "style_weight": 0.3
  }
}
```

### 配置说明

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `detection.threshold` | 0.6 | 公文判定阈值 (0-1) |
| `detection.workers` | 4 | 并行处理协程数 |
| `detection.timeout` | 30 | 单文件处理超时(秒) |
| `ocr.enabled` | true | 是否启用 OCR |
| `ocr.language` | chi_sim+eng | OCR 语言 |
| `output.format` | text | 输出格式 (text/json) |

## 命令行参数

```
用��:
  detector [选项] [文件路径]
  detector -file <文件路径>
  detector -dir <目录路径>

选项:
  -file, -f <路径>      指定待检测的单个文件
  -dir, -d <路径>       指定待检测的目录
  -threshold, -t <值>   公文判定阈值 (0-1)
  -workers, -w <数量>   并行处理协程数
  -json                 JSON 格式输出
  -verbose, -v          详细输出模式
  -no-ocr               禁用 OCR 功能
  -status               显示系统状态
  -version              显示版本信息
  -help, -h             显示帮助信息

配置文件:
  -config, -c <路径>    指定配置文件
  -gen-config           生成默认配置文件
  -save-config <路径>   保存当前配置
  -show-config          显示当前配置
```

## 依赖项

### 必需
- Go 1.21 或更高版本

### 可选（增强功能）
- **Tesseract OCR**：图片文字识别
  - Windows: https://github.com/UB-Mannheim/tesseract/wiki
  - Linux: `sudo apt-get install tesseract-ocr tesseract-ocr-chi-sim`
  - macOS: `brew install tesseract tesseract-lang`

- **LibreOffice**：DOC 格式支持（增强）
  - https://www.libreoffice.org/download/

- **Antiword**：DOC 格式支持（轻量）
  - Linux: `sudo apt-get install antiword`

## 项目结构

```
official-doc-detector/
├── cmd/
│   └── detector/
│       └── main.go              # 命令行入口
├── internal/
│   └── detector/
│       └── govcheck/
│           ├── detector/        # 检测器核心
│           ├── extractor/       # 特征提取
│           ├── scorer/          # 评分逻辑
│           └── processor/       # 文件处理器
├── pkg/
│   ├── config/                  # 配置管理
│   ├── errors/                  # 错误处理
│   └── fileutil/                # 文件工具
├── config.json                  # 配置文件
├── README.md                    # 项目说明
├── CHANGELOG.md                 # 更新日志
└── go.mod                       # Go 模块定义
```

## 开发指南

### 添加新的文件处理器

1. 在 `processor/` 目录创建新处理器文件
2. 实现 `Processor` 接口
3. 在 `main.go` 中注册处理器

```go
// 实现 Processor 接口
type MyProcessor struct {
    base *BaseProcessor
}

func NewMyProcessor() *MyProcessor {
    return &MyProcessor{
        base: NewBaseProcessor(
            "MyProcessor",
            "我的处理器描述",
            []string{"myext"},
        ),
    }
}

func (p *MyProcessor) Name() string { return p.base.Name() }
func (p *MyProcessor) Description() string { return p.base.Description() }
func (p *MyProcessor) SupportedTypes() []string { return p.base.SupportedTypes() }
func (p *MyProcessor) Process(filePath string) (string, error) {
    // 实现文本提取逻辑
    return "", nil
}
```

### 运行测试

```bash
go test ./...
go test -v ./internal/detector/govcheck/...
```

## 常见问题

### Q: OCR 不可用怎么办？

A: 安装 Tesseract OCR：
```bash
# Ubuntu/Debian
sudo apt-get install tesseract-ocr tesseract-ocr-chi-sim

# macOS
brew install tesseract tesseract-lang

# Windows
# 下载安装: https://github.com/UB-Mannheim/tesseract/wiki
```

### Q: DOC 文件处理失败？

A: 安装 LibreOffice 或 Antiword：
```bash
# LibreOffice
sudo apt-get install libreoffice

# Antiword (轻量级)
sudo apt-get install antiword
```

### Q: 如何调整判定阈值？

A: 使用 `-threshold` 参数或修改配置文件：
```bash
./detector -file doc.pdf -threshold 0.5
```

### Q: 如何处理大量文件？

A: 使用目录模式和多协程：
```bash
./detector -dir ./documents/ -workers 8
```

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 贡献

欢迎提交 Issue 和 Pull Request！

## 更新日志

详见 [CHANGELOG.md](CHANGELOG.md)