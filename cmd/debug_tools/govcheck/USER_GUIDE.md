# 公文版式检测工具 - 用户手册

## 目录

1. [简介](#简介)
2. [安装配置](#安装配置)
3. [基本使用](#基本使用)
4. [高级功能](#高级功能)
5. [配置详解](#配置详解)
6. [输出说明](#输出说明)
7. [故障排除](#故障排除)

---

## 简介

公文版式检测工具是一款基于 GB/T 9704-2012《党政机关公文格式》国家标准的自动化检测工具。它能够分析各种格式的文档，判断其是否符合公文规范。

### 适用场景

- 公文起草审核
- 档案数字化分类
- 文件批量筛选
- 公文格式培训

### 检测能力

| 能力 | 说明 |
|------|------|
| 文本特征识别 | 发文字号、标题、主送机关、成文日期等 |
| 版式特征识别 | 字体、字号、页边距、纸张大小等 |
| 机关识别 | 自动识别发文机关名称 |
| 文种识别 | 通知、决定、公告、意见等 15 种文种 |

---

## 安装配置

### 系统要求

- 操作系统：Windows 10+、Linux、macOS
- Go 语言：1.21 或更高版本（仅编译需要）
- 磁盘空间：约 50MB

### 编译安装

```bash
# 获取源码
git clone https://github.com/yourname/official-doc-detector.git
cd official-doc-detector

# 编译
go build -o detector ./cmd/detector

# 验证安装
./detector -version
```

### 安装可选依赖

#### Tesseract OCR（图片识别）

**Windows:**
1. 下载安装包：https://github.com/UB-Mannheim/tesseract/wiki
2. 安装时勾选"Chinese Simplified"语言包
3. 添加安装路径到系统 PATH

**Linux:**
```bash
sudo apt-get update
sudo apt-get install tesseract-ocr tesseract-ocr-chi-sim
```

**macOS:**
```bash
brew install tesseract tesseract-lang
```

#### LibreOffice（DOC 增强）

**Windows:**
1. 下载：https://www.libreoffice.org/download/
2. 安装后程序会自动检测

**Linux:**
```bash
sudo apt-get install libreoffice
```

### 验证依赖

```bash
./detector -status
```

输出示例：
```
[OCR引擎]
  状态: 可用
  引擎: Tesseract OCR
  版本: 5.5.1
  语言: [chi_sim, eng]

[DOC处理器]
  antiword:    不可用
  LibreOffice: 可用
  基础提取:    可用 (备选)
```

---

## 基本使用

### 检测单个文件

```bash
# 基本检测
./detector -file document.pdf

# 详细输出
./detector -file document.pdf -verbose

# 简写形式
./detector -f document.pdf -v
```

### 检测目录

```bash
# 检测目录下所有文件
./detector -dir ./documents/

# 使用多协程加速
./detector -dir ./documents/ -workers 8
```

### 输出格式

```bash
# 文本格式（默认）
./detector -file document.pdf

# JSON 格式
./detector -file document.pdf -json

# JSON 格式 + 保存到文件
./detector -file document.pdf -json > result.json
```

### 调整阈值

```bash
# 降低阈值（更容易判定为公文）
./detector -file document.pdf -threshold 0.5

# 提高阈值（更严格）
./detector -file document.pdf -threshold 0.8
```

---

## 高级功能

### 使用配置文件

```bash
# 生成默认配置
./detector -gen-config

# 使用指定配置文件
./detector -config myconfig.json -file document.pdf

# 查看当前配置
./detector -show-config

# 保存当前配置（含命令行覆盖）
./detector -threshold 0.7 -save-config custom.json
```

### 禁用 OCR

```bash
# 跳过图片文件处理
./detector -dir ./documents/ -no-ocr
```

### 批量处理脚本

**Windows (PowerShell):**
```powershell
# 检测目录并保存结果
.\detector.exe -dir .\documents\ -json | Out-File result.json -Encoding UTF8

# 只显示公文
.\detector.exe -dir .\documents\ | Select-String "是公文"
```

**Linux/macOS (Bash):**
```bash
# 检测目录并保存结果
./detector -dir ./documents/ -json > result.json

# 只显示公文
./detector -dir ./documents/ | grep "是公文"

# 统计公文数量
./detector -dir ./documents/ | grep -c "是公文"
```

---

## 配置详解

### 配置文件位置

程序按以下顺序查找配置文件：
1. `-config` 参数指定的路径
2. 当前目录的 `config.json`
3. 当前目录的 `detector.json`
4. 可执行文件目录的 `config.json`

### 完整配置示例

```json
{
  "detection": {
    "threshold": 0.6,
    "workers": 4,
    "timeout": 30,
    "max_file_size": 104857600,
    "exclude_extensions": [".exe", ".dll", ".zip", ".rar"],
    "exclude_directories": [".git", "node_modules", "__pycache__"]
  },
  "ocr": {
    "enabled": true,
    "language": "chi_sim+eng",
    "dpi": 300,
    "timeout": 60
  },
  "output": {
    "format": "text",
    "verbose": false,
    "color": true,
    "show_score_details": true
  },
  "scoring": {
    "text_weight": 0.7,
    "style_weight": 0.3
  },
  "processors": {
    "doc": {
      "use_libreoffice": true,
      "use_antiword": true,
      "fallback_to_basic": true
    }
  }
}
```

### 配置项说明

#### detection（检测配置）

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| threshold | float | 0.6 | 公文判定阈值，0-1 之间 |
| workers | int | 4 | 并行处理协程数 |
| timeout | int | 30 | 单文件处理超时（秒） |
| max_file_size | int | 104857600 | 最大文件大小（字节，默认 100MB） |
| exclude_extensions | []string | [] | 排除的文件扩展名 |
| exclude_directories | []string | [] | 排除的目录名 |

#### ocr（OCR 配置）

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| enabled | bool | true | 是否启用 OCR |
| language | string | "chi_sim+eng" | OCR 语言 |
| dpi | int | 300 | 图片处理 DPI |
| timeout | int | 60 | OCR 超时（秒） |

#### scoring（评分配置）

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| text_weight | float | 0.7 | 文本特征权重 |
| style_weight | float | 0.3 | 版式特征权重 |

---

## 输出说明

### 文本输出格式

```
文件: example.docx
类型: docx
大小: 25.30 KB
状态: 处理成功
耗时: 45.2ms
置信度: 92.33%        ← 综合得分
阈值: 60.00%          ← 判定阈值
判定: ✓ 是公文        ← 最终判定

分项得分:
  文本特征: 87.67%    ← 基于文本内容的得分
  版式特征: 45.00%    ← 基于版式格式的得分
```

### JSON 输出格式

```json
{
  "results": [
    {
      "file_path": "/path/to/document.pdf",
      "file_type": "pdf",
      "file_size": 25900,
      "is_official": true,
      "confidence": 0.9233,
      "threshold": 0.6,
      "text_score": 0.8767,
      "style_score": 0.45,
      "process_time": "45.2ms",
      "features": {
        "doc_number": "国办发〔2024〕1号",
        "title": "关于开展工作的通知",
        "title_type": "通知",
        "recipient": "各省、自治区、直辖市人民政府：",
        "date": "2024年1月15日",
        "has_seal": true,
        "has_cc": true
      }
    }
  ],
  "summary": {
    "total": 1,
    "official": 1,
    "non_official": 0,
    "failed": 0,
    "total_time": "45.2ms"
  }
}
```

### 得分明细说明

| 特征 | 分值 | 说明 |
|------|------|------|
| 发文字号 | +0.18 | 如"国发〔2024〕1号" |
| 公文标题 | +0.15 | 包含文种的规范标题 |
| 成文日期 | +0.12 | 如"2024年1月15日" |
| 机关名称 | +0.10 | 识别到的政府机关 |
| 主送机关 | +0.08 | 如"各省、自治区..." |
| 印章 | +0.08 | 检测到印章图片 |
| 抄送 | +0.05 | 包含抄送信息 |
| 印发信息 | +0.05 | 包含印发机关和日期 |
| 标题文种 | +0.05 | 标题包含规范文种 |

---

## 故障排除

### 常见问题

#### 1. "OCR 不可用"

**原因**：未安装 Tesseract OCR

**解决**：
```bash
# Linux
sudo apt-get install tesseract-ocr tesseract-ocr-chi-sim

# macOS
brew install tesseract tesseract-lang

# Windows
# 下载安装：https://github.com/UB-Mannheim/tesseract/wiki
```

#### 2. DOC 文件提取失败

**原因**：未安装 LibreOffice 或 Antiword

**解决**：
```bash
# 安装 LibreOffice
sudo apt-get install libreoffice

# 或安装 Antiword（轻量级）
sudo apt-get install antiword
```

#### 3. 中文乱码

**原因**：Tesseract 未安装中文语言包

**解决**：
```bash
# Linux
sudo apt-get install tesseract-ocr-chi-sim

# macOS
brew install tesseract-lang
```

#### 4. 文件处理超时

**原因**：文件过大或处理时间过长

**解决**：
```json
// config.json
{
  "detection": {
    "timeout": 120,
    "max_file_size": 209715200
  }
}
```

#### 5. 内存不足

**原因**：同时处理文件过多

**解决**：减少并行协程数
```bash
./detector -dir ./documents/ -workers 2
```

### 错误代码

| 代码 | 说明 | 解决方案 |
|------|------|----------|
| 2000 | 文件不存在 | 检查文件路径 |
| 2001 | 文件为空 | 确认文件内容 |
| 2002 | 文件过大 | 调整 max_file_size |
| 2005 | 文件格式错误 | 确认文件未损坏 |
| 3006 | 外部工具未安装 | 安装相关依赖 |
| 5001 | 无可检测内容 | 文件可能不包含文本 |

### 获取帮助

```bash
# 查看帮助
./detector -help

# 查看系统状态
./detector -status

# 查看版本
./detector -version
```