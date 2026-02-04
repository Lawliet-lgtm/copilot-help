package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestInitIdentity 测试 InitIdentity 函数
func TestInitIdentity(t *testing.T) {
	// 保存原始值
	originalDeviceID := DeviceID
	originalHardwareFingerprint := HardwareFingerprint
	originalIDFilePath := idFilePath

	// 测试后重置值
	defer func() {
		DeviceID = originalDeviceID
		HardwareFingerprint = originalHardwareFingerprint
		idFilePath = originalIDFilePath
	}()

	// 测试 InitIdentity
	err := InitIdentity()
	if err != nil {
		t.Fatalf("InitIdentity 失败: %v", err)
	}

	// 检查 HardwareFingerprint 是否生成
	if HardwareFingerprint == "" {
		t.Error("InitIdentity 后 HardwareFingerprint 不应为空")
	}

	// 检查 idFilePath 是否设置
	if idFilePath == "" {
		t.Error("InitIdentity 后 idFilePath 不应为空")
	}
}

// TestIsRegistered 测试 IsRegistered 函数
func TestIsRegistered(t *testing.T) {
	// 保存原始值
	originalDeviceID := DeviceID

	// 测试后重置值
	defer func() {
		DeviceID = originalDeviceID
	}()

	// 测试未注册情况
	DeviceID = ""
	if IsRegistered() {
		t.Error("当 DeviceID 为空时，IsRegistered 应返回 false")
	}

	// 测试已注册情况
	DeviceID = "test-device-id"
	if !IsRegistered() {
		t.Error("当 DeviceID 不为空时，IsRegistered 应返回 true")
	}
}

// TestGetUserAgent 测试 GetUserAgent 函数
func TestGetUserAgent(t *testing.T) {
	// 保存原始值
	originalDeviceID := DeviceID
	originalVersion := Version
	originalVendor := Vendor

	// 测试后重置值
	defer func() {
		DeviceID = originalDeviceID
		Version = originalVersion
		Vendor = originalVendor
	}()

	// 测试未注册情况
	DeviceID = ""
	Version = "20230101_TestVersion"
	Vendor = "TestVendor"

	expectedUA := "20230101_TestVersion (TestVendor)"
	actualUA := GetUserAgent()
	if actualUA != expectedUA {
		t.Errorf("GetUserAgent() = %s, 期望 %s", actualUA, expectedUA)
	}

	// 测试已注册情况
	DeviceID = "test-device-id"
	expectedUA = "test-device-id / 20230101_TestVersion (TestVendor)"
	actualUA = GetUserAgent()
	if actualUA != expectedUA {
		t.Errorf("GetUserAgent() = %s, 期望 %s", actualUA, expectedUA)
	}

	// 测试长字符串（应被截断）
	DeviceID = strings.Repeat("a", 100)
	Version = strings.Repeat("b", 100)
	Vendor = strings.Repeat("c", 100)

	actualUA = GetUserAgent()
	if len(actualUA) > 64+32+32+10 { // device-id(64) + version(32) + vendor(32) + 分隔符
		t.Error("GetUserAgent 应截断长字符串")
	}
}

// TestGetFullVersionInfo 测试 GetFullVersionInfo 函数
func TestGetFullVersionInfo(t *testing.T) {
	// 保存原始值
	originalDeviceID := DeviceID
	originalVersion := Version
	originalVendor := Vendor
	originalHardwareFingerprint := HardwareFingerprint
	originalIDFilePath := idFilePath
	originalBuildTime := BuildTime

	// 测试后重置值
	defer func() {
		DeviceID = originalDeviceID
		Version = originalVersion
		Vendor = originalVendor
		HardwareFingerprint = originalHardwareFingerprint
		idFilePath = originalIDFilePath
		BuildTime = originalBuildTime
	}()

	// 设置测试值
	DeviceID = "test-device-id"
	Version = "20230101_TestVersion"
	Vendor = "TestVendor"
	HardwareFingerprint = "test-fingerprint"
	idFilePath = "/test/path/agent.id"
	BuildTime = "2023-01-01 12:00:00"

	// 获取版本信息
	versionInfo := GetFullVersionInfo()

	// 检查是否包含所有预期字段
	expectedFields := []string{
		"Version:",
		"Vendor:",
		"Status:",
		"DeviceID:",
		"StoragePath:",
		"HW-FP:",
		"Built:",
	}

	for _, field := range expectedFields {
		if !strings.Contains(versionInfo, field) {
			t.Errorf("GetFullVersionInfo() 应包含字段 %q", field)
		}
	}

	// 检查已注册状态
	if !strings.Contains(versionInfo, "Registered") {
		t.Error("当 DeviceID 设置时，GetFullVersionInfo 应显示 'Registered'")
	}

	// 检查未注册状态
	DeviceID = ""
	versionInfo = GetFullVersionInfo()
	if !strings.Contains(versionInfo, "Unregistered") {
		t.Error("当 DeviceID 未设置时，GetFullVersionInfo 应显示 'Unregistered'")
	}
}

// TestUpdateAndPersistDeviceID 测试 UpdateAndPersistDeviceID 函数
func TestUpdateAndPersistDeviceID(t *testing.T) {
	// 创建测试用临时目录
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir) // 测试后清理

	// 保存原始值
	originalDeviceID := DeviceID
	originalIDFilePath := idFilePath
	originalDefaultDataDir := DefaultDataDir

	// 测试后重置值
	defer func() {
		DeviceID = originalDeviceID
		idFilePath = originalIDFilePath
		DefaultDataDir = originalDefaultDataDir
	}()

	// 设置测试自定义路径
	idFilePath = filepath.Join(tempDir, "agent.id")

	// 测试空设备ID
	err = UpdateAndPersistDeviceID("")
	if err == nil {
		t.Error("当设备ID为空时，UpdateAndPersistDeviceID应返回错误")
	}

	// 测试有效设备ID
	testDeviceID := "test-device-id-123"
	err = UpdateAndPersistDeviceID(testDeviceID)
	if err != nil {
		t.Fatalf("UpdateAndPersistDeviceID失败: %v", err)
	}

	// 检查内存中的DeviceID是否更新
	if DeviceID != testDeviceID {
		t.Errorf("内存中的DeviceID应为 %q, 实际为 %q", testDeviceID, DeviceID)
	}

	// 检查是否创建了文件
	if _, err := os.Stat(idFilePath); os.IsNotExist(err) {
		t.Errorf("应在 %q 创建设备ID文件", idFilePath)
	}

	// 读取文件并检查内容
	content, err := os.ReadFile(idFilePath)
	if err != nil {
		t.Fatalf("读取设备ID文件失败: %v", err)
	}

	fileContent := strings.TrimSpace(string(content))
	if fileContent != testDeviceID {
		t.Errorf("设备ID文件内容应为 %q, 实际为 %q", testDeviceID, fileContent)
	}

	// 测试从文件加载
	// 重置DeviceID
	DeviceID = ""
	// 从文件加载
	err = loadDeviceID()
	if err != nil {
		t.Fatalf("loadDeviceID失败: %v", err)
	}

	// 检查DeviceID是否正确加载
	if DeviceID != testDeviceID {
		t.Errorf("从文件加载的DeviceID应为 %q, 实际为 %q", testDeviceID, DeviceID)
	}
}

// TestLimitString 测试 limitString 函数
func TestLimitString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		max      int
		expected string
	}{{
		"空字符串", "", 5, ""},
		{
			"字符串长度小于最大值", "test", 5, "test"},
		{
			"字符串长度等于最大值", "test1", 5, "test1"},
		{
			"字符串长度大于最大值", "test123", 5, "test1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := limitString(tt.input, tt.max)
			if result != tt.expected {
				t.Errorf("limitString(%q, %d) = %q, 期望 %q", tt.input, tt.max, result, tt.expected)
			}
		})
	}
}

// TestResolveIDFilePath 测试 resolveIDFilePath 函数
func TestResolveIDFilePath(t *testing.T) {
	// 保存原始值
	originalDefaultDataDir := DefaultDataDir

	// 测试后重置值
	defer func() {
		DefaultDataDir = originalDefaultDataDir
		idFilePath = ""
	}()

	// 测试不同OS和用户场景
	tests := []struct {
		name         string
		goos         string
		euid         int
		expectedPath string
	}{}

	// 运行每个场景的测试
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 runtime.GOOS 和 os.Geteuid
			// 注意：我们无法轻松模拟这些，因此我们将通过检查函数行为来测试逻辑
			resolveIDFilePath()
			if idFilePath == "" {
				t.Error("resolveIDFilePath 后 idFilePath 不应为空")
			}
		})
	}
}

// TestIsRegisteredConcurrent 测试 IsRegistered 的并发访问
func TestIsRegisteredConcurrent(t *testing.T) {
	// 保存原始值
	originalDeviceID := DeviceID

	// 测试后重置值
	defer func() {
		DeviceID = originalDeviceID
	}()

	// 测试并发访问
	var wg sync.WaitGroup
	iterations := 100

	// 启动检查 IsRegistered 的 goroutine
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			IsRegistered()
		}()
	}

	// 启动更新 DeviceID 的 goroutine
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				DeviceID = "test-device-id"
			} else {
				DeviceID = ""
			}
			time.Sleep(100 * time.Microsecond)
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
}
