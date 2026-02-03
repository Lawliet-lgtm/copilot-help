// Package model
package model

// CommandPayload 指令负载结构体
type CommandPayload struct {
	// 指令类型，字符串类型，如"command"、"policy"
	Type string `json:"type" binding:"required"`
	// 具体指令或策略类型，字符串类型
	Cmd string `json:"cmd" binding:"required"`
	// 指令ID，字符串
	CmdID string `json:"cmd_id,omitempty"`
	// 模块名，字符串
	Module string `json:"module,omitempty"`
	// 子模块，对象类型
	Submodule interface{} `json:"submodule,omitempty"`
	// 参数，对象类型
	Param interface{} `json:"param,omitempty"`
}
