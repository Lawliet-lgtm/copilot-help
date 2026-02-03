package enforcer

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
)

// IptablesEnforcer 基于 iptables 的实现
type IptablesEnforcer struct {
	mu sync.Mutex // 确保 iptables 命令串行执行，防止并发竞争
}

func NewIptablesEnforcer() *IptablesEnforcer {
	return &IptablesEnforcer{}
}

// BlockIP 立即封禁 IP
func (e *IptablesEnforcer) BlockIP(ip string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. 安全熔断检查 (Safety Check)
	// 绝对不允许封禁本地回环，这是底线
	if ip == "127.0.0.1" || ip == "::1" {
		return fmt.Errorf("safety guard: cannot block loopback address %s", ip)
	}

	// 2. 检查 IP 格式
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid ip address: %s", ip)
	}

	// 3. 执行封禁 (双向封禁)
	// 策略: 插入到 INPUT 和 OUTPUT 链的最前面 (-I)
	// 这样即使后面有 ACCEPT 规则，也会先匹配 DROP

	// Block Inbound
	if err := runIptables("-I", "INPUT", "-s", ip, "-j", "DROP"); err != nil {
		return fmt.Errorf("failed to block inbound %s: %v", ip, err)
	}

	// Block Outbound
	if err := runIptables("-I", "OUTPUT", "-d", ip, "-j", "DROP"); err != nil {
		// 尝试回滚 Inbound，避免状态不一致（可选）
		_ = runIptables("-D", "INPUT", "-s", ip, "-j", "DROP")
		return fmt.Errorf("failed to block outbound %s: %v", ip, err)
	}

	return nil
}

// UnblockIP 解封 IP
func (e *IptablesEnforcer) UnblockIP(ip string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 忽略错误，因为可能规则本来就不存在
	// 使用 -D 删除规则
	_ = runIptables("-D", "INPUT", "-s", ip, "-j", "DROP")
	_ = runIptables("-D", "OUTPUT", "-d", ip, "-j", "DROP")

	return nil
}

// runIptables 封装 exec 调用
func runIptables(args ...string) error {
	// 在生产环境中，应该检查 exec.LookPath("iptables")
	cmd := exec.Command("iptables", args...)

	// 获取输出以便调试
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 清洗 output 中的换行符
		msg := strings.TrimSpace(string(output))
		return fmt.Errorf("iptables exec error: %s (args: %v)", msg, args)
	}
	return nil
}
