package detector

import (
	"os"

	"github.com/shirou/gopsutil/v3/process"
)

// ConnectionInfo 简化后的连接信息
type ConnectionInfo struct {
	RemoteIP   string
	RemotePort uint32
	LocalPort  uint32
	Status     string // ESTABLISHED, SYN_SENT, etc.
	PID        int32
	Protocol   string // TCP / UDP
}

// NetworkScanner 网络扫描器
type NetworkScanner struct {
	TargetPIDs []int32
}

// NewScanner 创建扫描器
// targetPIDs: 需要监控的进程ID列表。若为空，则默认监控当前进程。
func NewScanner(targetPIDs []int32) *NetworkScanner {
	// 如果未指定，默认监控自身
	if len(targetPIDs) == 0 {
		targetPIDs = []int32{int32(os.Getpid())}
	}
	return &NetworkScanner{
		TargetPIDs: targetPIDs,
	}
}

// Scan 执行一次扫描，返回所有活跃的外部连接
func (s *NetworkScanner) Scan() ([]ConnectionInfo, error) {
	var results []ConnectionInfo

	// 遍历监控的所有 PID
	for _, pid := range s.TargetPIDs {
		proc, err := process.NewProcess(pid)
		if err != nil {
			// 进程可能已退出，跳过
			continue
		}

		// 获取该进程的所有网络连接 (TCP & UDP)
		conns, err := proc.Connections()
		if err != nil {
			// 权限不足或进程瞬时消失
			continue
		}

		for _, c := range conns {
			// 过滤掉无关状态
			// LISTEN: 只是在监听，还没有发生实际通信，暂不视为违规（除非被动扫描，那是 SYN_RECV）
			// TIME_WAIT/CLOSE_WAIT: 连接已结束，封禁也晚了，忽略以减少噪音
			if c.Status == "LISTEN" || c.Status == "TIME_WAIT" || c.Status == "CLOSE_WAIT" || c.Status == "NONE" {
				continue
			}

			// 提取协议类型
			proto := "TCP"
			if c.Type == 2 { // gopsutil 中 2 通常代表 UDP
				proto = "UDP"
			}

			results = append(results, ConnectionInfo{
				RemoteIP:   c.Raddr.IP,
				RemotePort: c.Raddr.Port,
				LocalPort:  c.Laddr.Port,
				Status:     c.Status,
				PID:        pid,
				Protocol:   proto,
			})
		}
	}

	return results, nil
}
