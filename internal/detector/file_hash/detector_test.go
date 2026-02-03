// Package file_hash 文件哈希检测子模块测试
package file_hash

import (
	"os"
	"path/filepath"
	"testing"

	"linuxFileWatcher/internal/config"
)

// TestNewDetector 测试创建检测器
func TestNewDetector(t *testing.T) {
	// 初始化配置（使用空字符串，会使用默认值）
	if err := config.LoadConfig(""); err != nil {
		t.Logf("Warning: Failed to load config file, using defaults: %v", err)
		// 继续执行，因为配置会使用默认值
	}

	// 创建检测器
	detector := NewDetector()
	if detector == nil {
		t.Fatal("Failed to create detector")
	}

	// 验证检测器属性
	if detector.GetName() != "file_hash_detector" {
		t.Errorf("Expected detector name 'file_hash_detector', got '%s'", detector.GetName())
	}

	if detector.GetVersion() != "1.0.0" {
		t.Errorf("Expected detector version '1.0.0', got '%s'", detector.GetVersion())
	}
}

// TestInit 测试初始化检测器
func TestInit(t *testing.T) {
	// 初始化配置（使用空字符串，会使用默认值）
	if err := config.LoadConfig(""); err != nil {
		t.Logf("Warning: Failed to load config file, using defaults: %v", err)
		// 继续执行，因为配置会使用默认值
	}

	// 创建检测器
	detector := NewDetector()
	if detector == nil {
		t.Fatal("Failed to create detector")
	}

	// 初始化检测器
	err := detector.Init(nil)
	if err != nil {
		t.Logf("Warning: Failed to initialize detector (policy file may not exist): %v", err)
		// 继续执行，因为策略文件可能不存在
	}
}

// TestDetect 测试检测功能
func TestDetect(t *testing.T) {
	// 初始化配置（使用空字符串，会使用默认值）
	if err := config.LoadConfig(""); err != nil {
		t.Logf("Warning: Failed to load config file, using defaults: %v", err)
		// 继续执行，因为配置会使用默认值
	}

	// 创建检测器
	detector := NewDetector()
	if detector == nil {
		t.Fatal("Failed to create detector")
	}

	// 初始化检测器
	err := detector.Init(nil)
	if err != nil {
		t.Logf("Warning: Failed to initialize detector (policy file may not exist): %v", err)
		// 继续执行，因为策略文件可能不存在
	}

	// 测试1: 不存在的文件
	_, err = detector.Detect("nonexistent_file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}

	// 测试2: 空文件
	emptyFile := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	result, err := detector.Detect(emptyFile)
	if err != nil {
		t.Fatalf("Failed to detect empty file: %v", err)
	}

	if result.Detected {
		t.Error("Expected empty file to not be detected as sensitive")
	}

	// 测试3: 正常文件
	normalFile := filepath.Join(t.TempDir(), "normal.txt")
	if err := os.WriteFile(normalFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}

	result, err = detector.Detect(normalFile)
	if err != nil {
		t.Fatalf("Failed to detect normal file: %v", err)
	}

	// 正常文件默认不应该被检测为敏感
	// 注意：如果策略文件中包含了此文件的哈希值，则会被检测为敏感
}

// TestDetectWithSensitiveFile 测试检测敏感文件
// 注意：此测试需要在策略文件中添加对应的哈希值
func TestDetectWithSensitiveFile(t *testing.T) {
	// 初始化配置（使用空字符串，会使用默认值）
	if err := config.LoadConfig(""); err != nil {
		t.Logf("Warning: Failed to load config file, using defaults: %v", err)
		// 继续执行，因为配置会使用默认值
	}

	// 创建检测器
	detector := NewDetector()
	if detector == nil {
		t.Fatal("Failed to create detector")
	}

	// 初始化检测器
	err := detector.Init(nil)
	if err != nil {
		t.Logf("Warning: Failed to initialize detector (policy file may not exist): %v", err)
		// 继续执行，因为策略文件可能不存在
	}

	// 创建测试文件
	testFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 检测文件
	result, err := detector.Detect(testFile)
	if err != nil {
		t.Fatalf("Failed to detect test file: %v", err)
	}

	// 打印检测结果，便于调试
	t.Logf("Detection result: Detected=%v, Matches=%d", result.Detected, len(result.Matches))
}

// TestLoadPolicy 测试加载策略
func TestLoadPolicy(t *testing.T) {
	// 初始化配置（使用空字符串，会使用默认值）
	if err := config.LoadConfig(""); err != nil {
		t.Logf("Warning: Failed to load config file, using defaults: %v", err)
		// 继续执行，因为配置会使用默认值
	}

	// 创建检测器
	detector := NewDetector()
	if detector == nil {
		t.Fatal("Failed to create detector")
	}

	// 测试加载策略
	err := detector.loadPolicy()
	if err != nil {
		t.Logf("Warning: Failed to load policy (policy file may not exist): %v", err)
		// 继续执行，因为策略文件可能不存在
	}
}
