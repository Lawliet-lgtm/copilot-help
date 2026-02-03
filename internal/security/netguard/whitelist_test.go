package netguard

import (
	"testing"
)

func TestWhitelistManager_Basic(t *testing.T) {
	// 初始化白名单，包含一个具体 IP 和一个网段
	initial := []string{
		"192.168.1.100", // 管理平台
		"10.0.0.0/8",    // 内网段
	}
	wm := NewWhitelistManager(initial)

	// 测试用例
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"Allowed Exact IP", "192.168.1.100", true},
		{"Allowed CIDR IP", "10.1.2.3", true},
		{"Allowed Loopback v4", "127.0.0.1", true}, // 强制默认白名单
		{"Allowed Loopback v6", "::1", true},       // 强制默认白名单
		{"Blocked IP", "8.8.8.8", false},           // 未在白名单
		{"Blocked IP Neighbor", "192.168.1.101", false},
		{"Invalid IP", "not-an-ip", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wm.IsAllowed(tt.ip); got != tt.want {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestWhitelistManager_Concurrency(t *testing.T) {
	// 压力测试：验证并发读写的安全性 (go test -race)
	wm := NewWhitelistManager([]string{})

	// 模拟并发读写
	go func() {
		for i := 0; i < 1000; i++ {
			wm.Add("1.1.1.1")
		}
	}()

	for i := 0; i < 1000; i++ {
		wm.IsAllowed("1.1.1.1")
	}
}
