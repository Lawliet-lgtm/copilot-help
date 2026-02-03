package netguard

import (
	"net"
	"strings"
	"sync"
)

// WhitelistManager 管理内存中的 IP 白名单
// 它是并发安全的，支持运行时动态更新
type WhitelistManager struct {
	mu           sync.RWMutex
	exactIPs     map[string]struct{} // 精确 IP 匹配 (O(1) 查找)
	cidrNetworks []*net.IPNet        // 网段匹配 (O(N) 遍历)
}

// NewWhitelistManager 初始化白名单
func NewWhitelistManager(initialList []string) *WhitelistManager {
	wm := &WhitelistManager{
		exactIPs: make(map[string]struct{}),
	}

	// 强制添加本地回环，防止误封禁导致系统瘫痪或自身崩溃
	// 这是资深开发者的防御性编程习惯
	wm.addWithoutLock("127.0.0.1")
	wm.addWithoutLock("::1")

	for _, ipOrCidr := range initialList {
		wm.addWithoutLock(ipOrCidr)
	}

	return wm
}

// Add 动态添加规则
func (wm *WhitelistManager) Add(ipOrCidr string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.addWithoutLock(ipOrCidr)
}

// IsAllowed 检查 IP 是否在白名单中
func (wm *WhitelistManager) IsAllowed(remoteIP string) bool {
	// 1. 数据清洗
	// 移除可能存在的 IPv6 zone ID (如 fe80::1%eth0 -> fe80::1)
	if idx := strings.Index(remoteIP, "%"); idx != -1 {
		remoteIP = remoteIP[:idx]
	}

	parsedIP := net.ParseIP(remoteIP)
	if parsedIP == nil {
		// 如果不是有效 IP，默认放行还是拦截？
		// 安全原则：未知即威胁。但在网络层，解析失败通常意味着脏数据。
		// 这里返回 false 表示不允许
		return false
	}

	wm.mu.RLock()
	defer wm.mu.RUnlock()

	// 2. 精确匹配 (最快)
	if _, ok := wm.exactIPs[parsedIP.String()]; ok {
		return true
	}

	// 3. CIDR 网段匹配
	for _, network := range wm.cidrNetworks {
		if network.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// addWithoutLock 内部添加逻辑
func (wm *WhitelistManager) addWithoutLock(ipOrCidr string) {
	ipOrCidr = strings.TrimSpace(ipOrCidr)
	if ipOrCidr == "" {
		return
	}

	// 尝试解析为 CIDR (IP段)
	if _, network, err := net.ParseCIDR(ipOrCidr); err == nil {
		wm.cidrNetworks = append(wm.cidrNetworks, network)
		return
	}

	// 尝试解析为 单个 IP
	if ip := net.ParseIP(ipOrCidr); ip != nil {
		wm.exactIPs[ip.String()] = struct{}{}
		return
	}

	// 既不是 IP 也不是 CIDR，记录日志或忽略
	// 实际开发中建议接入日志模块记录 Warning
}
