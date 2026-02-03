// Package commander 提供指令处理功能
package commander

import (
	"testing"

	"linuxFileWatcher/internal/model"
)

// TestNewModuleControlHandler 测试创建模块控制处理器
func TestNewModuleControlHandler(t *testing.T) {
	handler := NewModuleControlHandler()
	if handler == nil {
		t.Fatal("Expected handler to be non-nil")
	}

	if handler.moduleManager == nil {
		t.Error("Expected moduleManager to be non-nil")
	}
}

// TestValidateParam 测试参数验证
func TestValidateParam(t *testing.T) {
	handler := NewModuleControlHandler()

	tests := []struct {
		name    string
		param   ModuleControlParam
		wantErr bool
	}{
		{
			name: "valid startm command",
			param: ModuleControlParam{
				Cmd:       "startm",
				Module:    "file_detect",
				Submodule: []string{"secret_level_detect"},
			},
			wantErr: false,
		},
		{
			name: "valid stopm command",
			param: ModuleControlParam{
				Cmd:       "stopm",
				Module:    "file_detect",
				Submodule: []string{"file_hash_detect"},
			},
			wantErr: false,
		},
		{
			name: "valid startm_inner command",
			param: ModuleControlParam{
				Cmd:       "startm_inner",
				Module:    "file_detect",
				Submodule: []string{},
			},
			wantErr: false,
		},
		{
			name: "valid stopm_inner command",
			param: ModuleControlParam{
				Cmd:       "stopm_inner",
				Module:    "file_detect",
				Submodule: []string{},
			},
			wantErr: false,
		},
		{
			name: "invalid module name",
			param: ModuleControlParam{
				Cmd:       "startm",
				Module:    "invalid_module",
				Submodule: []string{"secret_level_detect"},
			},
			wantErr: true,
		},
		{
			name: "invalid command",
			param: ModuleControlParam{
				Cmd:       "invalid_cmd",
				Module:    "file_detect",
				Submodule: []string{"secret_level_detect"},
			},
			wantErr: true,
		},
		{
			name: "empty submodule list",
			param: ModuleControlParam{
				Cmd:       "startm",
				Module:    "file_detect",
				Submodule: []string{},
			},
			wantErr: false,
		},
		{
			name: "invalid submodule name",
			param: ModuleControlParam{
				Cmd:       "startm",
				Module:    "file_detect",
				Submodule: []string{"invalid_submodule"},
			},
			wantErr: false, // 不会返回错误，但会跳过无效子模块
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateParam(tt.param)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateParam() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGetSubmodulesToProcess 测试获取需要处理的子模块列表
func TestGetSubmodulesToProcess(t *testing.T) {
	handler := NewModuleControlHandler()

	tests := []struct {
		name     string
		param    ModuleControlParam
		expected int
	}{
		{
			name: "with specific submodules",
			param: ModuleControlParam{
				Submodule: []string{"secret_level_detect", "file_hash_detect"},
			},
			expected: 2,
		},
		{
			name: "empty submodule list",
			param: ModuleControlParam{
				Submodule: []string{},
			},
			expected: len(ValidSubmodules),
		},
		{
			name: "nil submodule list",
			param: ModuleControlParam{
				Submodule: nil,
			},
			expected: len(ValidSubmodules),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.getSubmodulesToProcess(tt.param)
			if len(result) != tt.expected {
				t.Errorf("getSubmodulesToProcess() = %v, expected %d items", result, tt.expected)
			}
		})
	}
}

// TestBuildResultReport 测试构建结果报告
func TestBuildResultReport(t *testing.T) {
	handler := NewModuleControlHandler()

	tests := []struct {
		name           string
		cmd            string
		successModules []string
		failedModules  map[string]string
		expectSuccess  bool
	}{
		{
			name:           "all success",
			cmd:            "startm",
			successModules: []string{"secret_level_detect", "file_hash_detect"},
			failedModules:  map[string]string{},
			expectSuccess:  true,
		},
		{
			name:           "all failed",
			cmd:            "stopm",
			successModules: []string{},
			failedModules: map[string]string{
				"secret_level_detect": "module not found",
			},
			expectSuccess: false,
		},
		{
			name:           "partial success",
			cmd:            "startm",
			successModules: []string{"secret_level_detect"},
			failedModules: map[string]string{
				"file_hash_detect": "module not found",
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := handler.buildResultReport("cmd-123", tt.cmd, tt.successModules, tt.failedModules)

			if report == nil {
				t.Fatal("Expected report to be non-nil")
			}

			if report.CmdID != "cmd-123" {
				t.Errorf("Expected CmdID cmd-123, got %s", report.CmdID)
			}

			if report.Cmd != tt.cmd {
				t.Errorf("Expected Cmd %s, got %s", tt.cmd, report.Cmd)
			}

			// 检查结果状态
			isSuccess := report.Result == 0
			if isSuccess != tt.expectSuccess {
				t.Errorf("Expected success=%v, got result=%d", tt.expectSuccess, report.Result)
			}

			// 验证详情不为空
			if len(report.Detail) == 0 {
				t.Error("Expected detail to be non-empty")
			}
		})
	}
}

// TestBuildErrorReport 测试构建错误报告
func TestBuildErrorReport(t *testing.T) {
	handler := NewModuleControlHandler()

	report := handler.buildErrorReport("cmd-456", "startm", "validation failed")

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	if report.CmdID != "cmd-456" {
		t.Errorf("Expected CmdID cmd-456, got %s", report.CmdID)
	}

	if report.Cmd != "startm" {
		t.Errorf("Expected Cmd startm, got %s", report.Cmd)
	}

	if report.Result != 1 {
		t.Errorf("Expected Result 1 (failure), got %d", report.Result)
	}

	if report.Message != "validation failed" {
		t.Errorf("Expected Message 'validation failed', got %s", report.Message)
	}

	if len(report.Detail) == 0 {
		t.Error("Expected detail to be non-empty")
	}
}

// TestHandleModuleControl 测试便捷函数
func TestHandleModuleControl(t *testing.T) {
	param := ModuleControlParam{
		Cmd:       "startm_inner",
		Module:    "file_detect",
		Submodule: []string{"secret_level_detect"},
	}

	report := HandleModuleControl(param, "cmd-789")

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	if report.CmdID != "cmd-789" {
		t.Errorf("Expected CmdID cmd-789, got %s", report.CmdID)
	}

	if report.Cmd != "startm_inner" {
		t.Errorf("Expected Cmd startm_inner, got %s", report.Cmd)
	}
}

// TestSubmoduleNameMapping 测试子模块名称映射
func TestSubmoduleNameMapping(t *testing.T) {
	// 测试所有有效的子模块都有映射
	for _, subName := range ValidSubmodules {
		internalName, exists := SubmoduleNameMapping[subName]
		if !exists {
			t.Errorf("Expected mapping for submodule %s", subName)
		}
		if internalName == "" {
			t.Errorf("Expected non-empty internal name for submodule %s", subName)
		}
	}

	// 测试无效的子模块
	_, exists := SubmoduleNameMapping["invalid_submodule"]
	if exists {
		t.Error("Expected no mapping for invalid_submodule")
	}
}

// TestValidSubmodules 测试有效子模块列表
func TestValidSubmodules(t *testing.T) {
	expectedSubmodules := []string{
		"keyword_detect",
		"md5_detect",
		"security_classification_level",
		"secret_level_detect",
		"official_format_detect",
	}

	if len(ValidSubmodules) != len(expectedSubmodules) {
		t.Errorf("Expected %d valid submodules, got %d", len(expectedSubmodules), len(ValidSubmodules))
	}

	for i, sub := range expectedSubmodules {
		if i >= len(ValidSubmodules) || ValidSubmodules[i] != sub {
			t.Errorf("Expected submodule %s at index %d", sub, i)
		}
	}
}

// TestHandleWithInvalidParam 测试处理无效参数
func TestHandleWithInvalidParam(t *testing.T) {
	handler := NewModuleControlHandler()

	// 测试无效的module
	param := ModuleControlParam{
		Cmd:       "startm",
		Module:    "invalid_module",
		Submodule: []string{"secret_level_detect"},
	}

	report := handler.Handle(param, "cmd-001")

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	if report.Result != 1 {
		t.Errorf("Expected failure (result=1) for invalid module, got %d", report.Result)
	}

	if report.Message == "" {
		t.Error("Expected error message for invalid module")
	}
}

// TestHandleWithUnknownCommand 测试处理未知指令
func TestHandleWithUnknownCommand(t *testing.T) {
	handler := NewModuleControlHandler()

	param := ModuleControlParam{
		Cmd:       "unknown_cmd",
		Module:    "file_detect",
		Submodule: []string{"secret_level_detect"},
	}

	report := handler.Handle(param, "cmd-002")

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	if report.Result != 1 {
		t.Errorf("Expected failure (result=1) for unknown command, got %d", report.Result)
	}
}

// TestHandleStartmInnerPlaceholder 测试startm_inner占位实现
func TestHandleStartmInnerPlaceholder(t *testing.T) {
	handler := NewModuleControlHandler()

	param := ModuleControlParam{
		Cmd:       "startm_inner",
		Module:    "file_detect",
		Submodule: []string{"secret_level_detect", "file_hash_detect"},
	}

	report := handler.Handle(param, "cmd-003")

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	// 占位实现应该返回成功
	if report.Result != 0 {
		t.Errorf("Expected success (result=0) for placeholder, got %d", report.Result)
	}

	// 验证详情中包含占位信息
	hasPlaceholderInfo := false
	for _, detail := range report.Detail {
		if detail == "Full implementation pending related interface completion" {
			hasPlaceholderInfo = true
			break
		}
	}

	if !hasPlaceholderInfo {
		t.Error("Expected placeholder info in detail")
	}
}

// TestHandleStopmInnerPlaceholder 测试stopm_inner占位实现
func TestHandleStopmInnerPlaceholder(t *testing.T) {
	handler := NewModuleControlHandler()

	param := ModuleControlParam{
		Cmd:       "stopm_inner",
		Module:    "file_detect",
		Submodule: []string{},
	}

	report := handler.Handle(param, "cmd-004")

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	// 占位实现应该返回成功
	if report.Result != 0 {
		t.Errorf("Expected success (result=0) for placeholder, got %d", report.Result)
	}
}

// TestCommandResultReportCreation 测试结果报告创建
func TestCommandResultReportCreation(t *testing.T) {
	report := model.NewCommandResultReport("test-cmd-id", "test-cmd")

	if report == nil {
		t.Fatal("Expected report to be non-nil")
	}

	if report.CmdID != "test-cmd-id" {
		t.Errorf("Expected CmdID test-cmd-id, got %s", report.CmdID)
	}

	if report.Cmd != "test-cmd" {
		t.Errorf("Expected Cmd test-cmd, got %s", report.Cmd)
	}

	// 测试设置成功
	report.SetSuccess()
	if report.Result != 0 {
		t.Errorf("Expected Result 0 after SetSuccess, got %d", report.Result)
	}

	// 测试设置失败
	report.SetFailure("test failure")
	if report.Result != 1 {
		t.Errorf("Expected Result 1 after SetFailure, got %d", report.Result)
	}

	if report.Message != "test failure" {
		t.Errorf("Expected Message 'test failure', got %s", report.Message)
	}

	// 测试添加详情
	report.AddDetail("detail 1")
	report.AddDetail("detail 2")

	if len(report.Detail) != 2 {
		t.Errorf("Expected 2 details, got %d", len(report.Detail))
	}
}
