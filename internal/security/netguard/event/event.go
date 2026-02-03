package event

import (
	"fmt"
	"time"
)

// TrafficDirection 流量方向
type TrafficDirection string

const (
	DirectionInbound  TrafficDirection = "INBOUND"  // 被动接收 (Remote -> Local)
	DirectionOutbound TrafficDirection = "OUTBOUND" // 主动发起 (Local -> Remote)
)

// NetworkAlert 网络安全异常告警结构
type NetworkAlert struct {
	Timestamp   time.Time        `json:"timestamp"`
	AlertTime   int64            `json:"alert_time_unix"`
	Direction   TrafficDirection `json:"direction"`
	RemoteIP    string           `json:"remote_ip"`
	RemotePort  uint16           `json:"remote_port"`
	LocalPort   uint16           `json:"local_port"`
	Protocol    string           `json:"protocol"` // TCP/UDP
	ProcessName string           `json:"process_name"`
	PID         int32            `json:"pid"`
	ActionTaken string           `json:"action_taken"` // 执行了什么动作，如 "BLOCKED"
}

func (a NetworkAlert) String() string {
	return fmt.Sprintf("[%s] %s Traffic Violation: Remote=%s:%d (PID=%d)",
		a.ActionTaken, a.Direction, a.RemoteIP, a.RemotePort, a.PID)
}
