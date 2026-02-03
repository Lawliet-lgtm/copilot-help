// Package netguard
package netguard

import (
	"time"
)

// Config 网络防护模块配置
type Config struct {
	// Enable 开启开关
	Enable bool

	// CheckInterval 检测周期
	// 建议: 100ms - 1s，越短越灵敏但消耗 CPU 越高
	CheckInterval time.Duration

	// InitialWhitelist 初始白名单列表
	// 支持单IP ("192.168.1.100") 和 CIDR ("10.0.0.0/8")
	InitialWhitelist []string

	// MonitorSelf 是否监控本程序自身的网络连接 (强烈建议为 true)
	MonitorSelf bool

	// TargetPIDs 如果需要监控其他特定业务进程，在此指定 PID
	// 如果为空，且 MonitorSelf=true，则只监控自己
	TargetPIDs []int32

	// DeduplicationTime 异常IP处理去重时间
	// 防止同一个异常连接在被封禁后，由于 TCP 超时未断开而导致重复频繁告警
	DeduplicationTime time.Duration
}

// DefaultConfig 返回一个安全的默认配置
func DefaultConfig() Config {
	// 使用全局配置中的默认值，如果全局配置未初始化则使用硬编码默认值
	var cfg Config
	cfg.Enable = true
	cfg.CheckInterval = 1 * time.Second
	cfg.MonitorSelf = true
	cfg.DeduplicationTime = 1 * time.Hour
	cfg.InitialWhitelist = []string{
		"127.0.0.1", // 本地回环必须白名单，防止组件间通信中断
		"::1",
	}
	return cfg
}
