package netguard

import (
	"testing"
	"time"
)

// TestWhitelistManager 测试白名单管理器功能
func TestWhitelistManager(t *testing.T) {
	// 1. 创建白名单管理器
	initialList := []string{
		"192.168.1.0/24",
		"10.0.0.1",
		"2001:db8::/32",
	}
	wm := NewWhitelistManager(initialList)

	// 2. 测试初始白名单规则
	tests := []struct {
		ip          string
		shouldAllow bool
		desc        string
	}{
		{"127.0.0.1", true, "localhost should be allowed (auto added)"},
		{"::1", true, "localhost IPv6 should be allowed (auto added)"},
		{"192.168.1.100", true, "IP in CIDR range should be allowed"},
		{"10.0.0.1", true, "exact IP match should be allowed"},
		{"2001:db8::1", true, "IPv6 in CIDR range should be allowed"},
		{"8.8.8.8", false, "external IP should be blocked"},
		{"invalid-ip", false, "invalid IP should be blocked"},
		{"192.168.2.1", false, "IP outside CIDR should be blocked"},
	}

	for _, tt := range tests {
		result := wm.IsAllowed(tt.ip)
		if result != tt.shouldAllow {
			t.Errorf("TestWhitelistManager.IsAllowed(%s): expected %v, got %v - %s", tt.ip, tt.shouldAllow, result, tt.desc)
		}
	}

	// 3. 测试动态添加白名单
	wm.Add("8.8.8.8")
	if !wm.IsAllowed("8.8.8.8") {
		t.Errorf("TestWhitelistManager.Add: 8.8.8.8 should be allowed after adding to whitelist")
	}

	// 4. 测试添加CIDR
	wm.Add("172.16.0.0/12")
	if !wm.IsAllowed("172.16.10.5") {
		t.Errorf("TestWhitelistManager.Add: 172.16.10.5 should be allowed after adding 172.16.0.0/12 to whitelist")
	}

	// 5. 测试IPv6精确匹配
	wm.Add("2001:0db8:85a3:0000:0000:8a2e:0370:7334")
	if !wm.IsAllowed("2001:0db8:85a3:0000:0000:8a2e:0370:7334") {
		t.Errorf("TestWhitelistManager.Add: IPv6 exact match should be allowed")
	}

	// 6. 测试无效输入处理
	wm.Add("")             // 空字符串应该被忽略
	wm.Add("invalid-cidr") // 无效CIDR应该被忽略
	// 这些操作不应该导致panic
}

// TestManagerBasic 测试Manager基本功能
func TestManagerBasic(t *testing.T) {
	// 1. 创建默认配置
	cfg := DefaultConfig()

	// 2. 创建管理器实例
	manager := NewManager(cfg, nil)

	// 3. 测试启动和停止功能
	manager.Start()
	if !manager.running {
		t.Errorf("TestManagerBasic.Start: manager should be running after Start()")
	}

	// 4. 测试重复启动
	manager.Start() // 第二次启动应该无副作用

	// 5. 测试停止
	manager.Stop()
	if manager.running {
		t.Errorf("TestManagerBasic.Stop: manager should not be running after Stop()")
	}

	// 6. 测试重复停止
	manager.Stop() // 第二次停止应该无副作用

	// 7. 测试动态添加白名单
	manager.AddWhitelist("1.1.1.1")
	// 这里只是测试调用不会panic，具体白名单效果在WhitelistManager测试中验证
}

// TestManagerWithDisabledConfig 测试禁用配置
func TestManagerWithDisabledConfig(t *testing.T) {
	// 1. 创建禁用配置
	cfg := DefaultConfig()
	cfg.Enable = false

	// 2. 创建管理器实例
	manager := NewManager(cfg, nil)

	// 3. 测试启动（应该不运行）
	manager.Start()
	if manager.running {
		t.Errorf("TestManagerWithDisabledConfig.Start: manager should not be running when Enable=false")
	}

	// 4. 测试停止（应该无副作用）
	manager.Stop()
}

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// 验证默认配置值
	if !cfg.Enable {
		t.Errorf("TestDefaultConfig: Enable should be true by default")
	}

	if cfg.CheckInterval != 1*time.Second {
		t.Errorf("TestDefaultConfig: CheckInterval should be 1 second by default, got %v", cfg.CheckInterval)
	}

	if !cfg.MonitorSelf {
		t.Errorf("TestDefaultConfig: MonitorSelf should be true by default")
	}

	// 验证初始白名单包含本地回环
	foundV4Loopback := false
	foundV6Loopback := false
	for _, ip := range cfg.InitialWhitelist {
		if ip == "127.0.0.1" {
			foundV4Loopback = true
		}
		if ip == "::1" {
			foundV6Loopback = true
		}
	}

	if !foundV4Loopback {
		t.Errorf("TestDefaultConfig: InitialWhitelist should contain 127.0.0.1")
	}

	if !foundV6Loopback {
		t.Errorf("TestDefaultConfig: InitialWhitelist should contain ::1")
	}
}

// TestWhitelistAutoAddLoopback 测试白名单自动添加本地回环
func TestWhitelistAutoAddLoopback(t *testing.T) {
	// 创建一个空的初始白名单
	wm := NewWhitelistManager([]string{})

	// 验证本地回环被自动添加
	if !wm.IsAllowed("127.0.0.1") {
		t.Errorf("TestWhitelistAutoAddLoopback: 127.0.0.1 should be auto-added to whitelist")
	}

	if !wm.IsAllowed("::1") {
		t.Errorf("TestWhitelistAutoAddLoopback: ::1 should be auto-added to whitelist")
	}
}

// TestWhitelistIPv6ZoneHandling 测试IPv6 zone ID处理
func TestWhitelistIPv6ZoneHandling(t *testing.T) {
	wm := NewWhitelistManager([]string{"fe80::1"})

	// 测试带有zone ID的IPv6地址
	if !wm.IsAllowed("fe80::1%eth0") {
		t.Errorf("TestWhitelistIPv6ZoneHandling: IPv6 with zone ID should be allowed")
	}

	if !wm.IsAllowed("fe80::1%wlan0") {
		t.Errorf("TestWhitelistIPv6ZoneHandling: IPv6 with different zone ID should be allowed")
	}
}
