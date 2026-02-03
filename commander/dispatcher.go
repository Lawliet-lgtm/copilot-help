// Package commander
package commander

import (
	"fmt"

	"linuxFileWatcher/internal/logger"
	"linuxFileWatcher/internal/model"
)

// SimpleDispatcher 简单的指令分发器实现
type SimpleDispatcher struct{}

// NewSimpleDispatcher 创建新的简单指令分发器
func NewSimpleDispatcher() *SimpleDispatcher {
	return &SimpleDispatcher{}
}

// Dispatch 分发指令
func (d *SimpleDispatcher) Dispatch(cmd model.CommandPayload) error {
	logger.Info("Dispatching command", "type", cmd.Type, "cmd", cmd.Cmd, "module", cmd.Module)

	// 根据type和cmd来分发给不同的具体业务模块
	switch cmd.Type {
	case model.PolicyType:
		// 处理策略类型
		d.handlePolicy(cmd)
	case model.CommandType:
		// 处理命令类型
		d.handleCommand(cmd)
	default:
		logger.Warn("Unknown command type", "type", cmd.Type)
	}

	return nil
}

// handlePolicy 处理策略类型的指令
func (d *SimpleDispatcher) handlePolicy(cmd model.CommandPayload) {
	logger.Info("Handling policy command", "cmd", cmd.Cmd)

	// 根据具体的策略类型进行处理
	switch cmd.Cmd {
	case model.PolicyCmdAdd:
		// 处理增加策略的指令
		logger.Info("Adding policy", "module", cmd.Module)
	case model.PolicyCmdDel:
		// 处理删除策略的指令
		logger.Info("Deleting policy", "module", cmd.Module)
	case model.PolicyCmdReset:
		// 处理重置策略的指令
		logger.Info("Resetting policy", "module", cmd.Module)
	default:
		logger.Warn("Unknown policy command", "cmd", cmd.Cmd)
	}
}

// handleCommand 处理命令类型的指令
func (d *SimpleDispatcher) handleCommand(cmd model.CommandPayload) {
	logger.Info("Handling system command", "cmd", cmd.Cmd)

	// 根据具体的命令类型进行处理
	switch cmd.Cmd {
	case model.CmdFileDetectAuditLog:
		// 处理告警信息检测审计日志上报指令
		logger.Info("Handling file detect audit log command")
	case model.CmdUninstall:
		// 处理组件卸载指令
		logger.Info("Handling uninstall command")
	case model.CmdUpdate:
		// 处理系统软件更新指令
		logger.Info("Handling update command")
	case model.CmdStartm, model.CmdStopm, model.CmdStartmInner, model.CmdStopmInner:
		// 处理模块启停指令
		d.handleModuleControl(cmd)
	case model.CmdInnerPolicyUpdate:
		// 处理系统内置策略更新指令
		logger.Info("Handling inner policy update command")
	default:
		logger.Warn("Unknown system command", "cmd", cmd.Cmd)
	}
}

// handleModuleControl 处理模块启停控制指令
func (d *SimpleDispatcher) handleModuleControl(cmd model.CommandPayload) {
	logger.Info("Handling module control command",
		"cmd", cmd.Cmd,
		"module", cmd.Module,
		"submodule", cmd.Submodule,
	)

	// 构建模块控制参数
	param := ModuleControlParam{
		Cmd:    cmd.Cmd,
		Module: cmd.Module,
	}

	// 解析submodule参数
	if cmd.Submodule != nil {
		switch v := cmd.Submodule.(type) {
		case []string:
			param.Submodule = v
		case []interface{}:
			// 处理JSON解析后的[]interface{}类型
			for _, item := range v {
				if str, ok := item.(string); ok {
					param.Submodule = append(param.Submodule, str)
				}
			}
		case string:
			// 单个字符串，转换为数组
			param.Submodule = []string{v}
		default:
			logger.Warn("Unknown submodule type, using empty array",
				"type", fmt.Sprintf("%T", cmd.Submodule),
			)
		}
	}

	// 调用模块控制处理器
	report := HandleModuleControl(param, cmd.CmdID)

	// 上报执行结果
	ReportCommandResult(report)

	logger.Info("Module control result reported",
		"cmd", cmd.Cmd,
		"result", report.Result,
	)
}
