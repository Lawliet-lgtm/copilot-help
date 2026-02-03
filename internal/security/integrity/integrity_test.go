package integrity

import (
	"os"
	"testing"
	"time"
)

// ==========================================
// 1. Mock 工具：用于捕获上报的异常
// ==========================================

// MockReporter 用于测试中接收报警信号
type MockReporter struct {
	AlertChan chan ViolationType // 使用通道来同步测试状态
	LastMsg   string
}

func NewMockReporter() *MockReporter {
	return &MockReporter{
		// 缓冲设为 10，防止测试阻塞
		AlertChan: make(chan ViolationType, 10),
	}
}

func (m *MockReporter) Report(vType ViolationType, msg string) {
	m.LastMsg = msg
	// 将接收到的错误类型发送到通道
	select {
	case m.AlertChan <- vType:
	default:
	}
}

// ==========================================
// 2. 基础功能测试
// ==========================================

func TestComputeFileSM3(t *testing.T) {
	// 创建一个临时文件
	tmpFile, err := os.CreateTemp("", "sm3_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) // 清理

	// 写入已知内容
	content := []byte("hello world")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// 计算 Hash
	hash, err := ComputeFileSM3(tmpFile.Name())
	if err != nil {
		t.Fatalf("ComputeFileSM3 failed: %v", err)
	}

	// "hello world" 的 SM3 标准值 (Hex)
	expected := "44f0061e69fa6fdfc290c494654a05dc0c053da7e5c52b8469356066190543e7"

	if hash != expected {
		t.Errorf("SM3 hash mismatch.\nGot:  %s\nWant: %s", hash, expected)
	}
}

func TestGetSelfExecutablePath(t *testing.T) {
	// 这是一个冒烟测试，确保在当前环境下调用不报错
	path, err := GetSelfExecutablePath()
	if err != nil {
		t.Fatalf("Failed to get self path: %v", err)
	}
	t.Logf("Self Path detected: %s", path)

	if len(path) == 0 {
		t.Error("Returned path is empty")
	}
}

// ==========================================
// 3. 核心逻辑测试：模拟篡改与删除
// ==========================================

func TestIntegrityMonitor_Tamper(t *testing.T) {
	// --- 准备工作 ---

	// 1. 创建一个“伪造的二进制文件”
	// 我们不能直接修改正在运行的 test 程序，所以造一个替身
	fakeBin, err := os.CreateTemp("", "fake_agent_bin")
	if err != nil {
		t.Fatal(err)
	}
	fakeBinPath := fakeBin.Name()
	defer os.Remove(fakeBinPath) // 测试结束后清理

	// 写入初始版本 "v1.0"
	fakeBin.WriteString("version 1.0 (secure)")
	fakeBin.Close()

	// 2. 初始化 Mock 上报器
	mockReporter := NewMockReporter()

	// 3. 初始化监控器
	// 注意：NewMonitor 默认会读取真实的自身路径。
	// 我们需要先创建它，然后利用 Go 包内测试权限，强行修改它的 targetPath 指向伪造文件
	monitor, err := NewMonitor(mockReporter)
	if err != nil {
		t.Fatalf("NewMonitor failed: %v", err)
	}

	// 【黑科技】强行修改私有字段，指向我们的临时文件
	monitor.targetPath = fakeBinPath

	// 重新计算基线 Hash (因为 targetPath 变了)
	monitor.baselineHash, _ = ComputeFileSM3(fakeBinPath)
	t.Logf("Test Baseline Hash: %s", monitor.baselineHash)

	// --- 场景 A: 正常运行 ---

	// 手动触发一次检查，不应报错
	monitor.checkIntegrity()
	select {
	case v := <-mockReporter.AlertChan:
		t.Fatalf("Unexpected alert during normal state: %v", v)
	default:
		// OK
	}

	// --- 场景 B: 模拟篡改 (File Modified) ---

	t.Log("Simulating file tampering...")
	// 修改文件内容 (模拟黑客注入)
	// 必须用 os.OpenFile 以覆盖模式写入
	f, _ := os.OpenFile(fakeBinPath, os.O_WRONLY|os.O_TRUNC, 0755)
	f.WriteString("version 6.6.6 (hacked)")
	f.Close()

	// 手动触发检查
	monitor.checkIntegrity()

	// 验证是否收到告警
	select {
	case v := <-mockReporter.AlertChan:
		if v != TypeFileModified {
			t.Errorf("Expected TypeFileModified, got %v", v)
		} else {
			t.Log("SUCCESS: Detected file tampering!")
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout: Monitor failed to detect tampering")
	}

	// --- 场景 C: 模拟删除 (File Deleted) ---

	t.Log("Simulating file deletion...")
	os.Remove(fakeBinPath)

	// 手动触发检查
	monitor.checkIntegrity()

	select {
	case v := <-mockReporter.AlertChan:
		// 注意：根据之前的 monitor.go 实现，文件消失可能报 TypeFileDeleted 或者 TypeReadError
		// 只要报了其中之一就算通过
		if v != TypeFileDeleted && v != TypeReadError {
			t.Errorf("Expected TypeFileDeleted or ReadError, got %v", v)
		} else {
			t.Logf("SUCCESS: Detected file deletion (Type: %v)", v)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout: Monitor failed to detect deletion")
	}
}

// TestService_Lifecycle 测试 Service 的启动停止逻辑
func TestService_Lifecycle(t *testing.T) {
	mockReporter := NewMockReporter()
	srv, err := NewService(mockReporter)
	if err != nil {
		t.Fatal(err)
	}

	// 变更点：测试时可以传入一个很短的时间，或者传入标准时间
	go srv.StartService(100 * time.Millisecond) // 传入参数

	time.Sleep(200 * time.Millisecond)

	srv.StopService()
}
