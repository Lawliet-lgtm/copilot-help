package model

// ==========================================
// 终端日志审计接口 - 数据模型
// ==========================================

// ==========================================
// 请求结构体
// ==========================================

// SystemAuditRequest 系统审计日志请求
type SystemAuditRequest struct {
	// 日志id，字符串，只能用除空白、数字、下划线“_”以外字符，并不保证id的唯一性和重复性，≤20字节
	ID string `json:"id" binding:"required,max=20"`
	// 操作用户，本地登录用户名
	User string `json:"user" binding:"required"`
	// 事件时间，时间类型，≤24字节，精确到毫秒
	Time string `json:"time" binding:"required,max=24"`
	// 日志类型，字符串，≤64字节
	EventType SystemAuditLogType `json:"event_type" binding:"required,max=64"`
	// 操作类型，字符串，≤64字节
	OpType SystemAuditOpType `json:"opt_type" binding:"required,max=64"`
	// 日志详情，字符串
	Message string `json:"message" binding:"required"`
}

// ==========================================
// 响应结构体
// ==========================================

// SystemAuditResponse 系统审计日志响应
type SystemAuditResponse struct {
	// 响应类型，0：成功，1：失败
	Type int `json:"type"`
	// 响应消息，≤512字节，成功：“上报成功”，失败：“失败原因”
	Message string `json:"message" binding:"max=512"`
}

// ==========================================
// 辅助构造函数
// ==========================================

// NewSystemAuditRequest 创建新的系统审计日志请求
func NewSystemAuditRequest(id, user, time string, eventType SystemAuditLogType, opType SystemAuditOpType, message string) *SystemAuditRequest {
	return &SystemAuditRequest{
		ID:        id,
		User:      user,
		Time:      time,
		EventType: eventType,
		OpType:    opType,
		Message:   message,
	}
}

// NewSystemAuditResponse 创建新的系统审计日志响应
func NewSystemAuditResponse(success bool, message string) *SystemAuditResponse {
	responseType := 0
	if !success {
		responseType = 1
	}
	return &SystemAuditResponse{
		Type:    responseType,
		Message: message,
	}
}
