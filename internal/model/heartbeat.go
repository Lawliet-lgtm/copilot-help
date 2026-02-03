package model

// ==========================================
// 心跳接口 - 数据模型
// ==========================================

// ==========================================
// 请求结构体
// ==========================================

// HeartbeatRequest 心跳请求结构体
type HeartbeatRequest struct {
	// 设备ID，字符串，唯一标识设备
	AgentID string `json:"agent_id" binding:"required"`
	// 软件版本号，字符串
	Version string `json:"version" binding:"required"`
	// 时间戳，数值类型
	Timestamp int64 `json:"timestamp" binding:"required"`
	// 当前状态，字符串，如"running"
	Status string `json:"status" binding:"required"`
}

// ==========================================
// 响应结构体
// ==========================================

// HeartbeatResponse 心跳响应结构体
type HeartbeatResponse struct {
	// 响应类型，字符串类型
	Type string `json:"type" binding:"required"`
	// 具体指令或策略类型，字符串类型
	Cmd string `json:"cmd" binding:"required"`
	// 子模块，对象类型
	Submodule interface{} `json:"submodule,omitempty"`
	// 指令ID，字符串，最长128字节
	CmdID string `json:"cmd_id,omitempty" gorm:"type:varchar(128)"`
	// 参数，对象类型
	Param interface{} `json:"param,omitempty"`
	// 模块名，字符串，最长128字节
	Module string `json:"module,omitempty" gorm:"type:varchar(128)"`
	// 版本号，字符串，最长64字节
	Version string `json:"version,omitempty" gorm:"type:varchar(64)"`
	// 数值，数值类型
	Num string `json:"num,omitempty" gorm:"type:varchar(128)"`
	// 配置，字符串，最长128字节
	Config string `json:"config,omitempty" gorm:"type:varchar(128)"`
}

// CommandOrPolicy 指令或策略结构体
type CommandOrPolicy struct {
	// 响应类型，字符串类型，如"command"（指令）或"policy"（策略）
	Type string `json:"type" binding:"required"`
	// 具体指令或策略类型，如"add"（增加）、"del"（删除）、"reset"（重置）
	Cmd string `json:"cmd" binding:"required"`
	// 指令或策略的具体内容，根据不同类型可能有不同结构
	Config interface{} `json:"config,omitempty"`
}