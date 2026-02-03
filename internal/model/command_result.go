package model

import (
	"time"
)

// ==========================================
// 指令执行结果上报 - 数据模型
// ==========================================

// CommandResultReport 指令执行结果上报主结构 (对应数据库表: command_results)
// 对应表格中的正文 body 信息

type CommandResultReport struct {
	// 上报时间: 时间类型，精确到秒，最长 128
	Time string `json:"time"`

	// 指令类型: 字符串，最长 128，取值: command
	Type string `json:"type"`

	// 指令名称: 字符串，最长 128
	Cmd string `json:"cmd"`

	// 管理系统下发的指令 id 信息: 字符串，最长 128
	CmdID string `json:"cmd_id"`

	// 执行结果: 数值类型，0 代表成功，1 代表失败，最长 128
	Result int `json:"result"`

	// 执行结果描述: 字符串类型，最长 128，"成功"或是"失败原因"
	Message string `json:"message"`

	// 根据不同的指令类型，定制不同的详情内容: 可选，数组类型，最长 128
	Detail []string `json:"detail,omitempty"`

	// 本地记录创建时间
	CreatedAt time.Time `json:"-"`
}

// CommandResultResponse 指令执行结果上报响应参数结构
type CommandResultResponse struct {
	// 返回信息类型: 0 代表成功, 1 代表失败，最长 128
	Type int `json:"type"`

	// 返回消息内容: 最长 512，"上报成功"或是"失败原因"
	Message string `json:"message"`
}

// ==========================================
// 辅助构造工厂
// ==========================================

// NewCommandResultReport 创建新的指令执行结果报告
func NewCommandResultReport(cmdID, cmd string) *CommandResultReport {
	return &CommandResultReport{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Type:    "command", // 默认类型为 command
		Cmd:     cmd,
		CmdID:   cmdID,
		Result:  0, // 默认成功
		Message: "成功",
		Detail:  make([]string, 0),
	}
}

// SetSuccess 设置成功结果
func (r *CommandResultReport) SetSuccess() {
	r.Result = 0
	r.Message = "成功"
}

// SetFailure 设置失败结果
func (r *CommandResultReport) SetFailure(reason string) {
	r.Result = 1
	r.Message = limitString(reason, 128)
}

// AddDetail 添加详情
func (r *CommandResultReport) AddDetail(detail string) {
	if len(r.Detail) < 128 { // 限制最长 128 个元素
		r.Detail = append(r.Detail, limitString(detail, 128))
	}
}
