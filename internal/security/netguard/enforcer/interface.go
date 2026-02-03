package enforcer

// IPEnforcer 定义封禁执行器的接口
type IPEnforcer interface {
	// BlockIP 封禁指定 IP
	// direction: "IN" (入站), "OUT" (出站), "BOTH" (双向)
	BlockIP(ip string) error

	// UnblockIP 解封 IP (用于程序退出清理或误报恢复)
	UnblockIP(ip string) error
}
