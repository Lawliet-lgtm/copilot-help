package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	config := Default()

	if config == nil {
		t.Fatal("Default() 返回 nil")
	}

	if config.Detection.Threshold != 0.6 {
		t.Errorf("Threshold = %v, want 0.6", config.Detection.Threshold)
	}

	if config.Detection.TextWeight != 0.55 {
		t.Errorf("TextWeight = %v, want 0.55", config.Detection.TextWeight)
	}

	if config.Detection.StyleWeight != 0.45 {
		t.Errorf("StyleWeight = %v, want 0.45", config.Detection.StyleWeight)
	}

	if config.OCR.Language != "chi_sim+eng" {
		t.Errorf("OCR.Language = %v, want chi_sim+eng", config.OCR.Language)
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	config := Default()

	err := config.Validate()
	if err != nil {
		t.Errorf("默认配置验证失败: %v", err)
	}
}

func TestConfig_Validate_InvalidThreshold(t *testing.T) {
	config := Default()

	// 阈值过高
	config.Detection.Threshold = 1.5
	err := config.Validate()
	if err == nil {
		t.Error("阈值 1.5 应该验证失败")
	}

	// 阈值为负
	config.Detection.Threshold = -0.1
	err = config.Validate()
	if err == nil {
		t.Error("阈值 -0.1 应该验证失败")
	}
}

func TestConfig_Validate_InvalidWeights(t *testing.T) {
	config := Default()

	// 权重总和不为 1
	config.Detection.TextWeight = 0.8
	config.Detection.StyleWeight = 0.8
	err := config.Validate()
	if err == nil {
		t.Error("权重总和 1.6 应该验证失败")
	}
}

func TestConfig_Validate_InvalidFormat(t *testing.T) {
	config := Default()

	config.Output.Format = "xml"
	err := config.Validate()
	if err == nil {
		t.Error("输出格式 xml 应该验证失败")
	}
}

func TestConfig_Validate_InvalidLogLevel(t *testing.T) {
	config := Default()

	config.Output.LogLevel = "trace"
	err := config.Validate()
	if err == nil {
		t.Error("日志级别 trace 应该验证失败")
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	config := Default()
	config.Detection.Threshold = 0.75

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")

	// 保存
	err := config.Save(tmpFile)
	if err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("配置文件未创建")
	}

	// 加载
	loaded, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load 失败: %v", err)
	}

	// 验证值
	if loaded.Detection.Threshold != 0.75 {
		t.Errorf("加载的 Threshold = %v, want 0.75", loaded.Detection.Threshold)
	}
}

func TestLoad_NonExistent(t *testing.T) {
	_, err := Load("/non/existent/config.json")
	if err == nil {
		t.Error("加载不存在的文件应该失败")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.json")

	// 写入无效 JSON
	err := os.WriteFile(tmpFile, []byte("{ invalid json }"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	_, err = Load(tmpFile)
	if err == nil {
		t.Error("加载无效 JSON 应该失败")
	}
}

func TestLoadOrDefault_NonExistent(t *testing.T) {
	config := LoadOrDefault("/non/existent/config.json")

	if config == nil {
		t.Fatal("LoadOrDefault 返回 nil")
	}

	// 应该返回默认配置
	if config.Detection.Threshold != 0.6 {
		t.Errorf("Threshold = %v, want 0.6", config.Detection.Threshold)
	}
}

func TestLoadOrDefault_Empty(t *testing.T) {
	config := LoadOrDefault("")

	if config == nil {
		t.Fatal("LoadOrDefault(\"\") 返回 nil")
	}

	// 应该返回默认配置
	if config.Detection.Threshold != 0.6 {
		t.Errorf("Threshold = %v, want 0.6", config.Detection.Threshold)
	}
}

func TestConfig_Clone(t *testing.T) {
	config := Default()
	config.Detection.Threshold = 0.8

	clone := config.Clone()

	// 修改原配置
	config.Detection.Threshold = 0.5

	// clone 不应受影响
	if clone.Detection.Threshold != 0.8 {
		t.Errorf("Clone 的 Threshold = %v, want 0.8", clone.Detection.Threshold)
	}
}

func TestConfig_Clone_Slices(t *testing.T) {
	config := Default()
	config.Detection.ExcludeExtensions = []string{".exe", ".dll"}

	clone := config.Clone()

	// 修改原配置的切片
	config.Detection.ExcludeExtensions[0] = ".bat"

	// clone 不应受影响
	if clone.Detection.ExcludeExtensions[0] != ".exe" {
		t.Errorf("Clone 的切片被修改了")
	}
}

func TestConfig_String(t *testing.T) {
	config := Default()

	str := config.String()

	if str == "" {
		t.Error("String() 返回空字符串")
	}

	// 应该包含关键字段
	if !containsStr(str, "threshold") {
		t.Error("String() 应包含 threshold")
	}

	if !containsStr(str, "detection") {
		t.Error("String() 应包含 detection")
	}
}

func TestHighSensitivity(t *testing.T) {
	config := HighSensitivity()

	if config.Detection.Threshold != 0.45 {
		t.Errorf("HighSensitivity Threshold = %v, want 0.45", config.Detection.Threshold)
	}
}

func TestLowSensitivity(t *testing.T) {
	config := LowSensitivity()

	if config.Detection.Threshold != 0.75 {
		t.Errorf("LowSensitivity Threshold = %v, want 0.75", config.Detection.Threshold)
	}
}

func TestImageOptimized(t *testing.T) {
	config := ImageOptimized()

	if config.Detection.TextWeight != 0.40 {
		t.Errorf("ImageOptimized TextWeight = %v, want 0.40", config.Detection.TextWeight)
	}

	if config.Detection.StyleWeight != 0.60 {
		t.Errorf("ImageOptimized StyleWeight = %v, want 0.60", config.Detection.StyleWeight)
	}

	if config.OCR.Timeout != 60 {
		t.Errorf("ImageOptimized OCR.Timeout = %v, want 60", config.OCR.Timeout)
	}
}

func TestStrictMode(t *testing.T) {
	config := StrictMode()

	if config.Detection.Threshold != 0.70 {
		t.Errorf("StrictMode Threshold = %v, want 0.70", config.Detection.Threshold)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}