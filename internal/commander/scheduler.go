package commander

import (
	"encoding/json"
	"math/rand"
	"time"

	"linuxFileWatcher/internal/api"
	"linuxFileWatcher/internal/logger"
	"linuxFileWatcher/internal/model"
	"linuxFileWatcher/internal/security/transport"
)

// Scheduler 心跳调度器
type Scheduler struct {
	client     *transport.SecureClient
	dispatcher Dispatcher // 依赖接口，而非具体实现

	interval time.Duration
	stopChan chan struct{}
}

// NewScheduler 创建调度器
func NewScheduler(client *transport.SecureClient, dispatcher Dispatcher, interval time.Duration) *Scheduler {
	return &Scheduler{
		client:     client,
		dispatcher: dispatcher,
		interval:   interval,
		stopChan:   make(chan struct{}),
	}
}

// Start 启动心跳循环
func (s *Scheduler) Start() {
	// 1. 启动随机抖动 (Jitter)
	// 防止所有 Agent 在同一秒冲击服务端。随机休眠 0 ~ interval 之间的时间
	jitter := time.Duration(rand.Int63n(int64(s.interval)))
	time.Sleep(jitter)

	logger.Info("Heartbeat scheduler started", "interval", s.interval)

	// 2. 启动 Ticker
	ticker := time.NewTicker(s.interval)

	go func() {
		// 立即执行一次
		s.doHeartbeat()

		for {
			select {
			case <-s.stopChan:
				ticker.Stop()
				return
			case <-ticker.C:
				s.doHeartbeat()
			}
		}
	}()
}

// Stop 停止心跳
func (s *Scheduler) Stop() {
	close(s.stopChan)
}

// doHeartbeat 执行单次心跳逻辑
func (s *Scheduler) doHeartbeat() {
	// 1. 不需要请求体，直接发送空请求

	// 2. 发送加密请求
	// 使用 internal/api 中定义的常量
	url := api.BuildURL(api.RouteBusinessStatus)

	respBytes, err := s.client.PostEncrypted(url, nil)
	if err != nil {
		// 心跳失败是常态（网络波动），记录 Warn 即可，不要 Panic
		logger.Warn("Heartbeat failed", "err", err)
		return
	}

	// 3. 解析响应
	var resp model.HeartbeatResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		logger.Error("Failed to parse heartbeat response", "err", err)
		return
	}

	// 4. 根据响应类型和指令类型分发处理
	logger.Info("Received heartbeat response", "type", resp.Type, "cmd", resp.Cmd)

	// 构建CommandPayload用于分发
	cmdPayload := model.CommandPayload{
		Type:      resp.Type,
		Cmd:       resp.Cmd,
		CmdID:     resp.CmdID,
		Module:    resp.Module,
		Submodule: resp.Submodule,
		Param:     resp.Param,
	}

	// 5. 将指令丢给分发器
	if err := s.dispatcher.Dispatch(cmdPayload); err != nil {
		logger.Error("Failed to dispatch command", "cmd_id", resp.CmdID, "err", err)
	}
}
