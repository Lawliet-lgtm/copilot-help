package processor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ============================================================
// OCR 引擎接口
// ============================================================

// OcrEngine OCR引擎接口
type OcrEngine interface {
	// IsAvailable 检查OCR引擎是否可用
	IsAvailable() bool

	// Recognize 识别图片中的文字
	Recognize(imagePath string) (string, error)

	// RecognizeWithLang 使用指定语言识别
	RecognizeWithLang(imagePath string, lang string) (string, error)

	// GetName 获取引擎名称
	GetName() string

	// GetVersion 获取引擎版本
	GetVersion() string
}

// ============================================================
// Tesseract OCR 实现
// ============================================================

// TesseractOcr Tesseract OCR引擎
type TesseractOcr struct {
	execPath    string   // tesseract 可执行文件路径
	dataPath    string   // tessdata 数据目录路径
	defaultLang string   // 默认语言
	available   bool     // 是否可用
	version     string   // 版本号
	languages   []string // 可用语言列表
}

// TesseractConfig Tesseract配置
type TesseractConfig struct {
	ExecPath    string   // tesseract 可执行文件路径（空则自动查找）
	DataPath    string   // tessdata 数据目录路径（空则使用默认）
	DefaultLang string   // 默认语言（空则使用 chi_sim+eng）
	Languages   []string // 额外语言列表
}

// DefaultTesseractConfig 返回默认配置
func DefaultTesseractConfig() *TesseractConfig {
	return &TesseractConfig{
		DefaultLang: "chi_sim+eng", // 简体中文 + 英文
	}
}

// NewTesseractOcr 创建 Tesseract OCR 引擎
func NewTesseractOcr() *TesseractOcr {
	return NewTesseractOcrWithConfig(nil)
}

// NewTesseractOcrWithConfig 使用指定配置创建 Tesseract OCR 引擎
func NewTesseractOcrWithConfig(config *TesseractConfig) *TesseractOcr {
	if config == nil {
		config = DefaultTesseractConfig()
	}

	ocr := &TesseractOcr{
		execPath:    config.ExecPath,
		dataPath:    config.DataPath,
		defaultLang: config.DefaultLang,
	}

	// 初始化
	ocr.init()

	return ocr
}

// init 初始化 OCR 引擎
func (t *TesseractOcr) init() {
	// 查找 tesseract 可执行文件
	if t.execPath == "" {
		t.execPath = t.findTesseract()
	}

	if t.execPath == "" {
		t.available = false
		return
	}

	// 检查是否可执行
	if _, err := os.Stat(t.execPath); err != nil {
		t.available = false
		return
	}

	// 获取版本
	t.version = t.getVersionInfo()
	if t.version == "" {
		t.available = false
		return
	}

	// 获取可用语言
	t.languages = t.getAvailableLanguages()

	t.available = true
}

// findTesseract 查找 tesseract 可执行文件
func (t *TesseractOcr) findTesseract() string {
	var candidates []string

	if runtime.GOOS == "windows" {
		candidates = []string{
			"tesseract.exe",
			"tesseract",
			filepath.Join(os.Getenv("ProgramFiles"), "Tesseract-OCR", "tesseract.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Tesseract-OCR", "tesseract.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Tesseract-OCR", "tesseract.exe"),
			`C:\Program Files\Tesseract-OCR\tesseract.exe`,
			`C:\Program Files (x86)\Tesseract-OCR\tesseract.exe`,
		}
	} else {
		candidates = []string{
			"tesseract",
			"/usr/bin/tesseract",
			"/usr/local/bin/tesseract",
			"/opt/homebrew/bin/tesseract", // macOS Homebrew (Apple Silicon)
			"/usr/local/opt/tesseract/bin/tesseract", // macOS Homebrew (Intel)
		}
	}

	// 首先尝试 PATH 中查找
	if path, err := exec.LookPath("tesseract"); err == nil {
		return path
	}

	// 尝试候选路径
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// getVersionInfo 获取版本信息
func (t *TesseractOcr) getVersionInfo() string {
	cmd := exec.Command(t.execPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	// 解析版本号，格式如: "tesseract 5.3.0"
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	return strings.TrimSpace(string(output))
}

// getAvailableLanguages 获取可用语言列表
func (t *TesseractOcr) getAvailableLanguages() []string {
	cmd := exec.Command(t.execPath, "--list-langs")
	if t.dataPath != "" {
		cmd.Env = append(os.Environ(), "TESSDATA_PREFIX="+t.dataPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	var languages []string
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		// 跳过第一行（通常是路径信息）
		if i == 0 {
			continue
		}
		lang := strings.TrimSpace(line)
		if lang != "" && !strings.HasPrefix(lang, "List of") {
			languages = append(languages, lang)
		}
	}

	return languages
}

// ============================================================
// OcrEngine 接口实现
// ============================================================

// IsAvailable 检查OCR引擎是否可用
func (t *TesseractOcr) IsAvailable() bool {
	return t.available
}

// GetName 获取引擎名称
func (t *TesseractOcr) GetName() string {
	return "Tesseract OCR"
}

// GetVersion 获取引擎版本
func (t *TesseractOcr) GetVersion() string {
	return t.version
}

// GetLanguages 获取可用语言列表
func (t *TesseractOcr) GetLanguages() []string {
	return t.languages
}

// HasLanguage 检查是否支持指定语言
func (t *TesseractOcr) HasLanguage(lang string) bool {
	for _, l := range t.languages {
		if l == lang {
			return true
		}
	}
	return false
}

// Recognize 识别图片中的文字（使用默认语言）
func (t *TesseractOcr) Recognize(imagePath string) (string, error) {
	return t.RecognizeWithLang(imagePath, t.defaultLang)
}

// RecognizeWithLang 使用指定语言识别
func (t *TesseractOcr) RecognizeWithLang(imagePath string, lang string) (string, error) {
	if !t.available {
		return "", fmt.Errorf("Tesseract OCR 不可用")
	}

	// 检查图片文件
	if _, err := os.Stat(imagePath); err != nil {
		return "", fmt.Errorf("图片文件不存在: %s", imagePath)
	}

	// 如果未指定语言，使用默认语言
	if lang == "" {
		lang = t.defaultLang
	}

	// 验证语言是否可用
	if !t.validateLanguage(lang) {
		// 尝试退回到基本语言
		lang = t.fallbackLanguage(lang)
	}

	// 构建命令
	// tesseract imagePath stdout -l lang
	args := []string{imagePath, "stdout", "-l", lang}

	cmd := exec.Command(t.execPath, args...)

	if t.dataPath != "" {
		cmd.Env = append(os.Environ(), "TESSDATA_PREFIX="+t.dataPath)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("OCR识别失败: %s", errMsg)
	}

	// 返回识别结果
	result := stdout.String()
	result = strings.TrimSpace(result)

	return result, nil
}

// validateLanguage 验证语言是否可用
func (t *TesseractOcr) validateLanguage(lang string) bool {
	// 处理组合语言，如 "chi_sim+eng"
	langs := strings.Split(lang, "+")
	for _, l := range langs {
		l = strings.TrimSpace(l)
		if !t.HasLanguage(l) {
			return false
		}
	}
	return true
}

// fallbackLanguage 退回到可用的语言
func (t *TesseractOcr) fallbackLanguage(lang string) string {
	// 尝试使用英语
	if t.HasLanguage("eng") {
		return "eng"
	}

	// 使用第一个可用语言
	if len(t.languages) > 0 {
		return t.languages[0]
	}

	return lang
}

// ============================================================
// OCR 管理器
// ============================================================

// OcrManager OCR管理器
type OcrManager struct {
	engines   []OcrEngine
	primary   OcrEngine
	available bool
}

// NewOcrManager 创建 OCR 管理器
func NewOcrManager() *OcrManager {
	manager := &OcrManager{
		engines: make([]OcrEngine, 0),
	}

	// 注册 Tesseract
	tesseract := NewTesseractOcr()
	manager.RegisterEngine(tesseract)

	// 设置主引擎
	if tesseract.IsAvailable() {
		manager.primary = tesseract
		manager.available = true
	}

	return manager
}

// RegisterEngine 注册 OCR 引擎
func (m *OcrManager) RegisterEngine(engine OcrEngine) {
	m.engines = append(m.engines, engine)

	// 如果当前没有可用的主引擎，设置这个
	if m.primary == nil && engine.IsAvailable() {
		m.primary = engine
		m.available = true
	}
}

// IsAvailable 检查是否有可用的 OCR 引擎
func (m *OcrManager) IsAvailable() bool {
	return m.available
}

// GetPrimaryEngine 获取主 OCR 引擎
func (m *OcrManager) GetPrimaryEngine() OcrEngine {
	return m.primary
}

// Recognize 使用主引擎识别图片
func (m *OcrManager) Recognize(imagePath string) (string, error) {
	if m.primary == nil {
		return "", fmt.Errorf("没有可用的 OCR 引擎")
	}
	return m.primary.Recognize(imagePath)
}

// GetStatus 获取 OCR 状态信息
func (m *OcrManager) GetStatus() string {
	if !m.available {
		return "OCR 不可用 - 未检测到 Tesseract OCR。请安装 Tesseract: https://github.com/tesseract-ocr/tesseract"
	}

	if tesseract, ok := m.primary.(*TesseractOcr); ok {
		langs := tesseract.GetLanguages()
		return fmt.Sprintf("OCR 可用 - %s v%s (语言: %s)",
			m.primary.GetName(),
			m.primary.GetVersion(),
			strings.Join(langs, ", "))
	}

	return fmt.Sprintf("OCR 可用 - %s v%s", m.primary.GetName(), m.primary.GetVersion())
}

// ============================================================
// 全局 OCR 实例
// ============================================================

var globalOcrManager *OcrManager

// GetOcrManager 获取全局 OCR 管理器
func GetOcrManager() *OcrManager {
	if globalOcrManager == nil {
		globalOcrManager = NewOcrManager()
	}
	return globalOcrManager
}

// IsOcrAvailable 检查 OCR 是否可用
func IsOcrAvailable() bool {
	return GetOcrManager().IsAvailable()
}

// OcrRecognize 使用 OCR 识别图片
func OcrRecognize(imagePath string) (string, error) {
	return GetOcrManager().Recognize(imagePath)
}