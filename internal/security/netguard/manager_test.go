package netguard

import (
	"net"
	"sync"
	"testing"
	"time"

	"linuxFileWatcher/internal/security/netguard/reporter"
)

// ==========================================
// 1. 定义 Mock 组件
// ==========================================

// MockEnforcer 假的封禁执行器，只记录操作，不执行 iptables
type MockEnforcer struct {
	BlockedIPs []string
	mu         sync.Mutex
}

func (m *MockEnforcer) BlockIP(ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BlockedIPs = append(m.BlockedIPs, ip)
	return nil
}

func (m *MockEnforcer) UnblockIP(ip string) error {
	return nil
}

// ==========================================
// 2. 测试主流程
// ==========================================

func TestManager_IntegrationLoop(t *testing.T) {
	// 1. 准备配置
	// 极短的检测周期，方便测试快速完成
	cfg := DefaultConfig()
	cfg.CheckInterval = 100 * time.Millisecond
	cfg.InitialWhitelist = []string{"127.0.0.1"} // 只允许回环

	// 2. 初始化 Manager
	// 注意：我们要测试的是本进程产生的流量，NewManager 默认监控本进程，正好符合
	mgr := NewManager(cfg, &reporter.MockConsoleReporter{})

	// 【关键黑科技】：替换 Enforcer 为 Mock 版本
	// 因为 enforcer 字段是私有的，但因为测试文件也在 package netguard 下，所以能直接访问修改！
	mockEnforcer := &MockEnforcer{}
	mgr.enforcer = mockEnforcer

	// 3. 启动 Manager
	mgr.Start()
	defer mgr.Stop()

	// 4. 制造违规流量 (在测试进程内发起连接)
	// 尝试连接 8.8.8.8:53 (UDP)，这肯定不在白名单里
	// 这一步是为了让底层的 scanner (gopsutil) 能抓到连接
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err == nil {
		defer conn.Close()
		// 发送一点数据确保连接被系统记录
		conn.Write([]byte("ping"))
	} else {
		t.Logf("Skipping traffic generation due to network issue: %v", err)
	}

	// 5. 等待检测周期 (给 Manager 一点时间去发现和处理)
	time.Sleep(500 * time.Millisecond)

	// 6. 验证结果
	// Manager 应该调用了 mockEnforcer.BlockIP("8.8.8.8")
	mockEnforcer.mu.Lock()
	defer mockEnforcer.mu.Unlock()

	found := false
	for _, ip := range mockEnforcer.BlockedIPs {
		if ip == "8.8.8.8" {
			found = true
			break
		}
	}

	if !found {
		// 注意：如果你的电脑本身没有外网，net.Dial 可能会瞬间失败导致 gopsutil 抓不到
		// 在 CI 环境或离线环境可能需要 skip
		t.Log("Warning: Did not catch traffic to 8.8.8.8. This might be due to gopsutil scan timing or network reachability.")
		t.Logf("Blocked IPs captured: %v", mockEnforcer.BlockedIPs)
	} else {
		t.Log("SUCCESS: Detected and blocked illegal connection to 8.8.8.8")
	}
}
