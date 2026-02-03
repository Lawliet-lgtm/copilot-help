// Package electronic_secret 电子密级标志检测子模块
package electronic_secret

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"linuxFileWatcher/internal/detector/core"
	"linuxFileWatcher/internal/extractous"
	"linuxFileWatcher/internal/model"
)

// Detector 电子密级标志检测器
type Detector struct {
	// 检测器名称
	name string

	// 检测器版本
	version string

	// 检测规则
	rules []model.ElectronicSecretDetectRule

	// 特征模板映射
	featureTemplates map[string]int64

	// 敏感标签列表
	secretTags []string
}

// NewDetector 创建新的电子密级标志检测器
func NewDetector() *Detector {
	return &Detector{
		name:             "electronic_secret_detector",
		version:          "1.0.0",
		rules:            make([]model.ElectronicSecretDetectRule, 0),
		featureTemplates: make(map[string]int64),
		secretTags:       []string{"机密", "绝密", "SecretLevel", "秘密"},
	}
}

// GetName 返回检测器名称
func (d *Detector) GetName() string {
	return d.name
}

// GetVersion 返回检测器版本
func (d *Detector) GetVersion() string {
	return d.version
}

// Init 初始化检测器
func (d *Detector) Init(config interface{}) error {
	if config == nil {
		return nil
	}

	// 解析配置
	electronicSecretConfig, ok := config.(*model.ElectronicSecretDetectConfig)
	if !ok {
		return fmt.Errorf("invalid config type for electronic secret detector")
	}

	// 设置规则
	d.rules = electronicSecretConfig.Rules

	// 编译特征模板映射
	d.compileFeatureTemplates()

	return nil
}

// compileFeatureTemplates 编译特征模板映射
func (d *Detector) compileFeatureTemplates() {
	d.featureTemplates = make(map[string]int64)

	for _, rule := range d.rules {
		features := strings.Split(rule.RuleContent, ",")
		for _, feature := range features {
			feature = strings.TrimSpace(feature)
			if feature != "" {
				d.featureTemplates[feature] = rule.RuleID
			}
		}
	}
}

// Detect 执行检测操作
func (d *Detector) Detect(path string) (*core.DetectionResult, error) {

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", path)
	}

	// 获取文件信息
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 检查文件大小
	if fileInfo.Size() == 0 {
		return &core.DetectionResult{
			DetectorName: d.name,
			Detected:     false,
			Matches:      []core.MatchDetail{},
		}, nil
	}

	// 获取文件类型
	fileType, err := core.GetFileType(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file type: %w", err)
	}

	// 只处理图片和PDF文件
	if fileType != "image" && fileType != "document" {
		return &core.DetectionResult{
			DetectorName: d.name,
			Detected:     false,
			Matches:      []core.MatchDetail{},
		}, nil
	}

	// 执行电子密级标志检测
	matches := []core.MatchDetail{}

	switch fileType {
	case "image":
		// 图片文件检测
		matches = d.detectInImage(path)
	case "document":
		// 文档文件检测
		matches = d.detectInDocument(path)
	}

	// 构建检测结果
	result := &core.DetectionResult{
		DetectorName: d.name,
		Detected:     len(matches) > 0,
		Matches:      matches,
	}

	return result, nil
}

// detectInImage 在图片中检测电子密级标志
func (d *Detector) detectInImage(path string) []core.MatchDetail {
	matches := []core.MatchDetail{}

	// 使用 tesseract OCR 库提取图片中的文字
	// 构建 tesseract 命令
	cmd := exec.Command("tesseract", path, "stdout", "--oem", "3", "--psm", "6")

	// 捕获标准输出
	var out bytes.Buffer
	cmd.Stdout = &out

	// 执行命令
	err := cmd.Run()
	if err != nil {
		// 如果 OCR 失败，返回空匹配
		return matches
	}

	// 获取提取的文本
	content := out.String()

	// 检查文本中的电子密级标志特征
	for feature, ruleID := range d.featureTemplates {
		if strings.Contains(strings.ToLower(content), strings.ToLower(feature)) {
			// 查找对应的规则
			var rule model.ElectronicSecretDetectRule
			for _, r := range d.rules {
				if r.RuleID == ruleID {
					rule = r
					break
				}
			}

			// 添加匹配详情
			matches = append(matches, core.MatchDetail{
				MatchType:   "electronic_secret",
				Content:     feature,
				Location:    "image",
				RuleID:      ruleID,
				RuleDesc:    rule.RuleDesc,
				AlertType:   int(model.AlertTypeOther),
				FileSummary: "检测到电子密级标志",
				FileDesc:    "在图片中检测到电子密级标志",
				FileLevel:   5,
			})
		}
	}

	return matches
}

// detectInDocument 在文档中检测电子密级标志
func (d *Detector) detectInDocument(path string) []core.MatchDetail {
	matches := []core.MatchDetail{}

	// 使用 extractous-go 库提取文档内容
	extractor := extractous.New()
	content, err := extractor.Extract(path)
	if err != nil {
		return matches
	}

	// 检查文档内容中的电子密级标志特征
	for feature, ruleID := range d.featureTemplates {
		if strings.Contains(strings.ToLower(content), strings.ToLower(feature)) {
			// 查找对应的规则
			var rule model.ElectronicSecretDetectRule
			for _, r := range d.rules {
				if r.RuleID == ruleID {
					rule = r
					break
				}
			}

			// 添加匹配详情
			matches = append(matches, core.MatchDetail{
				MatchType:   "electronic_secret",
				Content:     feature,
				Location:    "document",
				RuleID:      ruleID,
				RuleDesc:    rule.RuleDesc,
				AlertType:   int(model.AlertTypeOther),
				FileSummary: "检测到电子密级标志",
				FileDesc:    "在文档中检测到电子密级标志",
				FileLevel:   5,
			})
		}
	}

	// 检查文档元数据中的电子密级标志
	metaMatches := d.detectInMetadata(path)
	matches = append(matches, metaMatches...)

	return matches
}

// detectInMetadata 在文档元数据中检测电子密级标志
func (d *Detector) detectInMetadata(path string) []core.MatchDetail {
	matches := []core.MatchDetail{}

	// 获取文件扩展名
	ext := strings.ToLower(path[strings.LastIndex(path, "."):])

	// 根据后缀分发不同的解析逻辑
	switch ext {
	case ".docx", ".xlsx", ".pptx", ".ofd":
		// 处理基于 Zip 结构的文档 (Office OpenXML / OFD)
		metaMatches := d.checkZipBasedDocs(path)
		matches = append(matches, metaMatches...)
	// TODO: 后续可以扩展 PDF 的 XMP 解析逻辑
	// case ".pdf":
	//     return c.checkPDF(ctx.FilePath)
	default:
		// 不支持的格式，跳过
		return matches
	}

	return matches
}

// checkZipBasedDocs 处理基于 Zip 结构的文档 (Office OpenXML / OFD)
func (d *Detector) checkZipBasedDocs(path string) []core.MatchDetail {
	matches := []core.MatchDetail{}

	// 1. 尝试作为 Zip 打开
	r, err := zip.OpenReader(path)
	if err != nil {
		// 如果打不开（可能加密了，或者损坏了），视为未命中
		return matches
	}
	defer r.Close()

	// 2. 遍历 Zip 内的文件列表
	for _, f := range r.File {
		// 我们只关心包含元数据的特定文件
		// docProps/custom.xml: Office 自定义属性 (最常存放密级)
		// docProps/core.xml:   Office 核心属性 (标题、备注等)
		// OFD.xml:             OFD 主入口
		if strings.Contains(f.Name, "docProps/custom.xml") ||
			strings.Contains(f.Name, "docProps/core.xml") ||
			strings.HasSuffix(f.Name, "OFD.xml") {

			// 3. 读取元数据文件内容并检测
			metaMatches := d.scanZipEntry(f)
			matches = append(matches, metaMatches...)
		}
	}

	return matches
}

// scanZipEntry 读取 Zip 中的单个文件并匹配关键词
func (d *Detector) scanZipEntry(f *zip.File) []core.MatchDetail {
	matches := []core.MatchDetail{}

	rc, err := f.Open()
	if err != nil {
		return matches
	}
	defer rc.Close()

	// 读取内容 (元数据文件通常很小，几KB，可以直接读入内存)
	content, err := io.ReadAll(rc)
	if err != nil {
		return matches
	}

	xmlContent := string(content)

	// 4. 暴力字符串匹配 (比解析 XML 更快且兼容性更好)
	for _, tag := range d.secretTags {
		if strings.Contains(xmlContent, tag) {
			// 添加匹配详情
			matches = append(matches, core.MatchDetail{
				MatchType:   "electronic_secret",
				Content:     tag,
				Location:    "metadata",
				RuleID:      0, // 元数据检测无特定规则ID
				RuleDesc:    "电子密级标志元数据检测",
				AlertType:   int(model.AlertTypeOther),
				FileSummary: "检测到电子密级标志",
				FileDesc:    fmt.Sprintf("在元数据文件 '%s' 中检测到电子密级标志", f.Name),
				FileLevel:   5,
			})
		}
	}

	return matches
}
