package model

import (
	"time"
)

// ==========================================
// 告警信息审计日志上报 - 数据模型
// ==========================================

// AlertLogReport 上报主结构
// 对应图片中的正文 body 信息
type AlertLogReport struct {
	// 管理系统下发的指令 ID: 字符串, 64 字节
	CmdID string `json:"cmd_id"`

	// 日志上报时间: 时间类型 (字符串格式)
	Time string `json:"time"`

	// 审计日志内容: 对象数组
	// 一对多关联
	AuditLogs []AlertLogItem `json:"audit_logs"`
}

// AlertLogItem 单条审计日志详情
type AlertLogItem struct {
	// 文件名: 字符串
	FileName string `json:"file_name"`

	// 告警文件路径: 字符串
	FilePath string `json:"file_path"`

	// 告警文件 md5: 字符串 (标准MD5是32字符)
	FileMD5 string `json:"file_md5"`

	// 通常审计日志每条记录也需要时间，这里加上更保险
	Time string `json:"time"`
}

// AlertLogResponse 响应参数结构
type AlertLogResponse struct {
	// 返回信息类型: 0 代表成功, 1 代表失败
	// 图片约束: 最长 128 (通常指字符串长度限制，但业务逻辑是 int)
	// 为了兼容 JSON 解析，如果服务端返回的是数字 0/1，用 int；如果是字符串 "0"/"1"，用 string
	// 这里按惯例设计为 int，兼容性更好
	Type int `json:"type"`

	// 返回消息内容: 最长 512
	Message string `json:"message"`
}

// ==========================================
// 辅助构造工厂
// ==========================================

// NewAlertLogReport 创建新的审计报告
func NewAlertLogReport(cmdID string) *AlertLogReport {
	return &AlertLogReport{
		CmdID:     cmdID,
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		AuditLogs: make([]AlertLogItem, 0),
	}
}

// AddLog 添加一条审计记录
func (r *AlertLogReport) AddLog(name, path, md5, tm string) {
	item := AlertLogItem{
		FileName: name,
		FilePath: path,
		FileMD5:  md5,
		Time:     tm,
	}
	r.AuditLogs = append(r.AuditLogs, item)
}
