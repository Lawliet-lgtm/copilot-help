// Package commander 提供指令处理功能
package commander

import (
	"fmt"

	"linuxFileWatcher/internal/detector/module"
	"linuxFileWatcher/internal/logger"
	"linuxFileWatcher/internal/model"
)

// ModuleControlParam 模块控制参数
type ModuleControlParam struct {
	Cmd       string   `json:"cmd"`       // 指令: startm/stopm/startm_inner/stopm_inner
	Module    string   `json:"module"`    // 模块名称: "file_detect"
	Submodule []string `json:"submodule"` // 子模块列表
}

// SubmoduleNameMapping 子模块名称映射表
// 将输入的子模块名称映射到内部模块管理器使用的名称
var SubmoduleNameMapping = map[string]string{
	"keyword_detect":                "keyword_detect",
	"md5_detect":                    "file_hash_detect",
	"security_classification_level": "electronic_secret_detect",
	"secret_level_detect":           "secret_level_detect",
	"official_format_detect":        "official_format_detect",
}

// ValidSubmodules 有效的子模块列表
var ValidSubmodules = []string{
	"keyword_detect",
	"md5_detect",
	"security_classification_level",
	"secret_level_detect",
	"official_format_detect",
}

// ModuleControlHandler 模块控制处理器
type ModuleControlHandler struct {
	moduleManager *module.Manager
}

// NewModuleControlHandler 创建模块控制处理器
// 自动获取已初始化的module.Manager实例
func NewModuleControlHandler() *ModuleControlHandler {
	return &ModuleControlHandler{
		moduleManager: getModuleManager(),
	}
}

// getModuleManager 获取模块管理器实例
// 这里使用单例模式获取全局的module.Manager实例
func getModuleManager() *module.Manager {
	// 使用默认配置路径创建管理器
	mgr := module.NewManager(module.DefaultConfigPath)

	// 如果管理器未初始化，进行初始化
	if !mgr.IsInitialized() {
		if err := mgr.Init(); err != nil {
			logger.Error("Failed to initialize module manager", "error", err)
			// 即使初始化失败也返回管理器，后续操作会报错
		}
	}

	return mgr
}

// Handle 处理模块控制指令
// param: 模块控制参数
// cmdID: 指令ID，用于结果上报
// 返回执行结果报告
func (h *ModuleControlHandler) Handle(param ModuleControlParam, cmdID string) *model.CommandResultReport {
	logger.Info("Handling module control command",
		"cmd", param.Cmd,
		"module", param.Module,
		"submodules", param.Submodule,
		"cmd_id", cmdID,
	)

	// 参数验证
	if err := h.validateParam(param); err != nil {
		logger.Error("Module control parameter validation failed",
			"error", err,
			"cmd", param.Cmd,
		)
		return h.buildErrorReport(cmdID, param.Cmd, err.Error())
	}

	// 根据指令类型分发处理
	switch param.Cmd {
	case "startm":
		return h.handleStartm(param, cmdID)
	case "stopm":
		return h.handleStopm(param, cmdID)
	case "startm_inner":
		return h.handleStartmInner(param, cmdID)
	case "stopm_inner":
		return h.handleStopmInner(param, cmdID)
	default:
		logger.Error("Unknown module control command", "cmd", param.Cmd)
		return h.buildErrorReport(cmdID, param.Cmd, "unknown command: "+param.Cmd)
	}
}

// validateParam 验证参数
func (h *ModuleControlHandler) validateParam(param ModuleControlParam) error {
	// 验证module参数
	if param.Module != "file_detect" {
		return fmt.Errorf("invalid module name: %s, expected: file_detect", param.Module)
	}

	// 验证cmd参数
	validCmds := []string{"startm", "stopm", "startm_inner", "stopm_inner"}
	isValidCmd := false
	for _, cmd := range validCmds {
		if param.Cmd == cmd {
			isValidCmd = true
			break
		}
	}
	if !isValidCmd {
		return fmt.Errorf("invalid command: %s", param.Cmd)
	}

	// 验证submodule参数（如果非空）
	if len(param.Submodule) > 0 {
		for _, sub := range param.Submodule {
			if _, exists := SubmoduleNameMapping[sub]; !exists {
				logger.Warn("Invalid submodule name, will be skipped",
					"submodule", sub,
				)
			}
		}
	}

	return nil
}

// handleStartm 处理启动模块指令
func (h *ModuleControlHandler) handleStartm(param ModuleControlParam, cmdID string) *model.CommandResultReport {
	logger.Info("Processing startm command", "cmd_id", cmdID)

	// 确定要处理的子模块列表
	submodules := h.getSubmodulesToProcess(param)

	// 执行启用操作
	successModules := []string{}
	failedModules := make(map[string]string) // 子模块名 -> 错误信息

	for _, subName := range submodules {
		// 映射到内部模块名称
		internalName, exists := SubmoduleNameMapping[subName]
		if !exists {
			logger.Warn("Skipping unknown submodule", "submodule", subName)
			continue
		}

		// 调用module.Manager启用模块
		if err := h.moduleManager.EnableModule(internalName); err != nil {
			logger.Error("Failed to enable module",
				"submodule", subName,
				"internal_name", internalName,
				"error", err,
			)
			failedModules[subName] = err.Error()
		} else {
			logger.Info("Module enabled successfully",
				"submodule", subName,
				"internal_name", internalName,
			)
			successModules = append(successModules, subName)
		}
	}

	// 构建并返回结果报告
	return h.buildResultReport(cmdID, "startm", successModules, failedModules)
}

// handleStopm 处理停止模块指令
func (h *ModuleControlHandler) handleStopm(param ModuleControlParam, cmdID string) *model.CommandResultReport {
	logger.Info("Processing stopm command", "cmd_id", cmdID)

	// 确定要处理的子模块列表
	submodules := h.getSubmodulesToProcess(param)

	// 执行禁用操作
	successModules := []string{}
	failedModules := make(map[string]string)

	for _, subName := range submodules {
		// 映射到内部模块名称
		internalName, exists := SubmoduleNameMapping[subName]
		if !exists {
			logger.Warn("Skipping unknown submodule", "submodule", subName)
			continue
		}

		// 调用module.Manager禁用模块
		if err := h.moduleManager.DisableModule(internalName); err != nil {
			logger.Error("Failed to disable module",
				"submodule", subName,
				"internal_name", internalName,
				"error", err,
			)
			failedModules[subName] = err.Error()
		} else {
			logger.Info("Module disabled successfully",
				"submodule", subName,
				"internal_name", internalName,
			)
			successModules = append(successModules, subName)
		}
	}

	// 构建并返回结果报告
	return h.buildResultReport(cmdID, "stopm", successModules, failedModules)
}

// handleStartmInner 处理启动模块内置策略检测功能（占位实现）
func (h *ModuleControlHandler) handleStartmInner(param ModuleControlParam, cmdID string) *model.CommandResultReport {
	logger.Info("Processing startm_inner command (placeholder)", "cmd_id", cmdID)

	// 确定要处理的子模块列表
	submodules := h.getSubmodulesToProcess(param)

	// 占位实现：记录日志，返回成功
	logger.Warn("startm_inner command is not fully implemented yet, placeholder only",
		"submodules", submodules,
		"cmd_id", cmdID,
	)

	// 构建结果报告
	report := model.NewCommandResultReport(cmdID, "startm_inner")
	report.SetSuccess()
	report.AddDetail(fmt.Sprintf("Placeholder: startm_inner for submodules: %v", submodules))
	report.AddDetail("Full implementation pending related interface completion")

	return report
}

// handleStopmInner 处理关闭模块内置策略检测功能（占位实现）
func (h *ModuleControlHandler) handleStopmInner(param ModuleControlParam, cmdID string) *model.CommandResultReport {
	logger.Info("Processing stopm_inner command (placeholder)", "cmd_id", cmdID)

	// 确定要处理的子模块列表
	submodules := h.getSubmodulesToProcess(param)

	// 占位实现：记录日志，返回成功
	logger.Warn("stopm_inner command is not fully implemented yet, placeholder only",
		"submodules", submodules,
		"cmd_id", cmdID,
	)

	// 构建结果报告
	report := model.NewCommandResultReport(cmdID, "stopm_inner")
	report.SetSuccess()
	report.AddDetail(fmt.Sprintf("Placeholder: stopm_inner for submodules: %v", submodules))
	report.AddDetail("Full implementation pending related interface completion")

	return report
}

// getSubmodulesToProcess 获取需要处理的子模块列表
// 如果param.Submodule为空，则返回所有有效的子模块
func (h *ModuleControlHandler) getSubmodulesToProcess(param ModuleControlParam) []string {
	if len(param.Submodule) > 0 {
		return param.Submodule
	}

	// 返回所有有效的子模块
	return ValidSubmodules
}

// buildResultReport 构建结果报告
func (h *ModuleControlHandler) buildResultReport(
	cmdID string,
	cmd string,
	successModules []string,
	failedModules map[string]string,
) *model.CommandResultReport {
	report := model.NewCommandResultReport(cmdID, cmd)

	// 判断整体执行结果
	if len(failedModules) == 0 {
		// 全部成功
		report.SetSuccess()
		report.AddDetail(fmt.Sprintf("All modules %s successfully: %v", cmd, successModules))
	} else if len(successModules) == 0 {
		// 全部失败
		report.SetFailure("All modules failed")
		for subName, errMsg := range failedModules {
			report.AddDetail(fmt.Sprintf("Failed to %s %s: %s", cmd, subName, errMsg))
		}
	} else {
		// 部分成功
		report.SetFailure("Partial success")
		report.AddDetail(fmt.Sprintf("Successfully %s modules: %v", cmd, successModules))
		for subName, errMsg := range failedModules {
			report.AddDetail(fmt.Sprintf("Failed to %s %s: %s", cmd, subName, errMsg))
		}
	}

	return report
}

// buildErrorReport 构建错误报告
func (h *ModuleControlHandler) buildErrorReport(cmdID string, cmd string, errMsg string) *model.CommandResultReport {
	report := model.NewCommandResultReport(cmdID, cmd)
	report.SetFailure(errMsg)
	report.AddDetail(fmt.Sprintf("Command execution failed: %s", errMsg))
	return report
}

// HandleModuleControl 便捷函数，直接处理模块控制指令
// 用于在dispatcher中快速调用
func HandleModuleControl(param ModuleControlParam, cmdID string) *model.CommandResultReport {
	handler := NewModuleControlHandler()
	return handler.Handle(param, cmdID)
}
