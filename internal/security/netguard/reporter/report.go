package reporter

import (
	"fmt"
	"linuxFileWatcher/internal/security/netguard/event"
)

// Reporter 定义告警上报接口
type Reporter interface {
	Report(alert event.NetworkAlert) error
}

// MockConsoleReporter 默认的控制台输出实现 (调试用)
type MockConsoleReporter struct{}

func (m *MockConsoleReporter) Report(alert event.NetworkAlert) error {
	// 模拟：实际场景中这里会调用模块四的 SecureClient 发送 JSON
	fmt.Printf("\n[NETGUARD ALARM] >>>>>\n%s\n<<<<<\n", alert.String())
	return nil
}
