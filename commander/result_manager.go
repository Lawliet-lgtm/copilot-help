package commander

import (
	"encoding/json"
	"fmt"
	"time"

	"linuxFileWatcher/internal/api"
	"linuxFileWatcher/internal/logger"
	"linuxFileWatcher/internal/model"
	"linuxFileWatcher/internal/security/transport"
	"linuxFileWatcher/internal/storage"
)

// ResultManager 执行结果上报模块
// 职责：封装指令执行结果和策略接收结果的上报接口，支持断网缓存重传
type ResultManager struct {
	client *transport.SecureClient
}

var resultMgr = &ResultManager{}

// InitResultManager 初始化结果上报管理器
// 需要在程序启动时调用
func InitResultManager(client *transport.SecureClient) {
	resultMgr.client = client
	
	// 启动后台补发调度器
	go resultMgr.startScheduler()
}

// startScheduler 后台任务：定期检查缓存并重试
func (m *ResultManager) startScheduler() {
	// 每 30 秒检查一次是否有积压的报告
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	logger.Info("Result report scheduler started")

	for range ticker.C {
		m.retryCommandResults()
		m.retryPolicyResults()
	}
}

// ==========================================
// 1. 指令执行结果上报
// ==========================================

// ReportCommandResult 上报指令执行结果 (供业务模块调用)
// 对应接口: /C2/sys_manager/command_result
func ReportCommandResult(report *model.CommandResultReport) {
	// 1. 尝试立即发送
	err := resultMgr.sendCommandResult(report)
	if err == nil {
		logger.Info("指令结果上报成功", "cmd_id", report.CmdID)
		return
	}

	// 2. 发送失败（如网络中断），写入 HybridStore 缓存
	logger.Warn("指令结果上报失败，转入本地缓存", "cmd_id", report.CmdID, "err", err)
	
	store := storage.GetStores().CommandResults
	if pushErr := store.Push(*report); pushErr != nil {
		logger.Error("写入指令结果缓存失败(数据丢失风险)", "cmd_id", report.CmdID, "err", pushErr)
	}
}

// sendCommandResult 执行实际的网络请求
func (m *ResultManager) sendCommandResult(report *model.CommandResultReport) error {
	// A. 序列化
	jsonData, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	// B. 构建 URL
	url := api.BuildURL(api.RouteSysManagerCommandResult)

	// C. 发送加密请求
	// 重点：使用 transport.WithGzipRequest() 满足接口文档的 Content-Encoding: gzip 要求
	respBytes, err := m.client.PostEncrypted(url, jsonData, transport.WithGzipRequest())
	if err != nil {
		return fmt.Errorf("网络请求失败: %v", err)
	}

	// D. 解析响应 (检查业务状态码)
	var resp model.CommandResultResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("响应解析失败: %v", err)
	}

	if resp.Type != 0 {
		return fmt.Errorf("服务端返回业务错误: %s", resp.Message)
	}

	return nil
}

// retryCommandResults 重试缓存中的指令报告
func (m *ResultManager) retryCommandResults() {
	store := storage.GetStores().CommandResults
	
	// PopAll 会从数据库/内存中取出并移除数据
	reports, err := store.PopAll()
	if err != nil {
		logger.Error("读取指令结果缓存失败", "err", err)
		return
	}

	if len(reports) == 0 {
		return
	}

	logger.Info("开始重试缓存的指令结果", "count", len(reports))

	for _, report := range reports {
		// 必须使用局部变量副本
		r := report 
		if err := m.sendCommandResult(&r); err != nil {
			// 如果依然失败，放回缓存
			if pushErr := store.Push(r); pushErr != nil {
				logger.Error("重试失败且放回缓存失败", "cmd_id", r.CmdID, "send_err", err, "push_err", pushErr)
			} else {
				logger.Debug("重试失败，放回缓存", "cmd_id", r.CmdID)
			}
		} else {
			logger.Info("指令结果重试上报成功", "cmd_id", r.CmdID)
		}
	}
}

// ==========================================
// 2. 策略执行结果上报
// ==========================================

// ReportPolicyResult 上报策略执行/接收结果 (供业务模块调用)
// 对应接口: /C2/sys_manager/policy_result
func ReportPolicyResult(report *model.StrategyExecReport) {
	err := resultMgr.sendPolicyResult(report)
	if err == nil {
		logger.Info("策略结果上报成功", "module", report.Module)
		return
	}

	logger.Warn("策略结果上报失败，转入本地缓存", "module", report.Module, "err", err)
	
	store := storage.GetStores().PolicyResults
	if pushErr := store.Push(*report); pushErr != nil {
		logger.Error("写入策略结果缓存失败", "module", report.Module, "err", pushErr)
	}
}

// sendPolicyResult 执行实际的网络请求
func (m *ResultManager) sendPolicyResult(report *model.StrategyExecReport) error {
	// A. 序列化
	jsonData, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("JSON序列化失败: %v", err)
	}

	// B. 构建 URL
	url := api.BuildURL(api.RouteSysManagerPolicyResult)

	// C. 发送加密请求 (带 Gzip)
	respBytes, err := m.client.PostEncrypted(url, jsonData, transport.WithGzipRequest())
	if err != nil {
		return fmt.Errorf("网络请求失败: %v", err)
	}

	// D. 解析响应
	var resp model.StrategyResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return fmt.Errorf("响应解析失败: %v", err)
	}

	if resp.Type != 0 {
		return fmt.Errorf("服务端返回业务错误: %s", resp.Message)
	}

	return nil
}

// retryPolicyResults 重试缓存中的策略报告
func (m *ResultManager) retryPolicyResults() {
	store := storage.GetStores().PolicyResults
	
	reports, err := store.PopAll()
	if err != nil {
		logger.Error("读取策略结果缓存失败", "err", err)
		return
	}

	if len(reports) == 0 {
		return
	}

	logger.Info("开始重试缓存的策略结果", "count", len(reports))

	for _, report := range reports {
		r := report
		if err := m.sendPolicyResult(&r); err != nil {
			if pushErr := store.Push(r); pushErr != nil {
				logger.Error("策略结果重试失败且放回缓存失败", "module", r.Module, "send_err", err, "push_err", pushErr)
			} else {
				logger.Debug("策略结果重试失败，放回缓存", "module", r.Module)
			}
		} else {
			logger.Info("策略结果重试上报成功", "module", r.Module)
		}
	}
}
