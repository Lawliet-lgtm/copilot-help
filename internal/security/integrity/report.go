package integrity

import "fmt"

// ViolationType 定义异常类型
type ViolationType string

const (
	TypeFileModified ViolationType = "FILE_MODIFIED" // 内容被篡改 (Hash不匹配)
	TypeFileDeleted  ViolationType = "FILE_DELETED"  // 文件消失
	TypePermChanged  ViolationType = "PERM_CHANGED"  // 权限/属性变更 (可选，视 Stat 检查深度而定)
	TypeReadError    ViolationType = "READ_ERROR"    // 无法读取 (可能被锁定或无权限)
)

// Reporter 定义上报接口
// 外部模块需要实现此接口，或者使用默认的 LogReporter
type Reporter interface {
	Report(vType ViolationType, msg string)
}

// DefaultConsoleReporter 默认的控制台打印实现 (兜底用)
type DefaultConsoleReporter struct{}

func (r *DefaultConsoleReporter) Report(vType ViolationType, msg string) {
	// 实际开发中，这里会替换为调用 模块四 的 Client 发送告警
	fmt.Printf("[SECURITY ALARM] Type: %s | Msg: %s\n", vType, msg)
}
