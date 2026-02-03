package model

import (
	"time"
)

// ==========================================
// 策略执行结果响应上报 - 数据模型
// ==========================================

// StrategyExecReport 上报主结构 (对应数据库表: strategy_exec_reports)
type StrategyExecReport struct {
	// 上报时间: 字符串, 最长 128
	Time string `json:"time"`

	// 检测策略类型: 固定值 "policy", 最长 128
	Type string `json:"type"`

	// 指令名称: 最长 128
	Cmd string `json:"cmd"`

	// 策略类型: 最长 64
	Module string `json:"module"`

	// 任务ID (Version): 最长 128
	Version string `json:"version"`

	// 成功列表: 整形数组
	Success []int64 `json:"success"`

	// 失败列表: 对象数组
	Fail []StrategyFailItem `json:"fail"`

	// 本地记录时间
	CreatedAt time.Time `json:"-"`
}

// StrategyFailItem 失败详情 (对应数据库表: strategy_fail_items)
type StrategyFailItem struct {
	// 策略 ID (rule_id)
	RuleID int64 `json:"rule_id"`

	// 失败原因 (msg)
	Msg string `json:"msg"`
}

// StrategyResponse 响应结构
type StrategyResponse struct {
	// 0 代表成功, 1 代表失败
	Type int `json:"type"`

	// 返回消息内容, 最长 512
	Message string `json:"message"`
}

// ==========================================
// 工厂方法
// ==========================================

// NewStrategyExecReport 创建报告
// cmd: 指令名称, version: 任务ID, module: 策略类型
func NewStrategyExecReport(cmd, version, module string) *StrategyExecReport {
	return &StrategyExecReport{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Type:    "policy", // 接口约束: 取值 policy
		Cmd:     cmd,
		Version: version,
		Module:  module,
		Success: make([]int64, 0),
		Fail:    make([]StrategyFailItem, 0),
	}
}

// AddSuccess 添加成功 ID
func (r *StrategyExecReport) AddSuccess(ruleID int64) {
	r.Success = append(r.Success, ruleID)
}

// AddFail 添加失败项
func (r *StrategyExecReport) AddFail(ruleID int64, msg string) {
	r.Fail = append(r.Fail, StrategyFailItem{
		RuleID: ruleID,
		Msg:    msg,
	})
}
