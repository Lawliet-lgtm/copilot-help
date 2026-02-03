// Package policy 通用策略管理模块
package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Manager 策略管理器
type Manager struct {
	// 策略存储根路径
	rootPath string
}

// NewManager 创建新的策略管理器
func NewManager(rootPath string) *Manager {
	// 确保策略存储根目录存在
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		fmt.Printf("Warning: Failed to create policy root directory: %v\n", err)
	}

	return &Manager{
		rootPath: rootPath,
	}
}

// LoadPolicy 从本地文件加载策略
// moduleName: 模块名称，如 "file_hash", "secret_level" 等
// config: 用于接收策略配置的指针
func (m *Manager) LoadPolicy(moduleName string, config interface{}) error {
	// 构建策略文件路径
	policyFile := filepath.Join(m.rootPath, moduleName, "policy.json")

	// 检查文件是否存在
	if _, err := os.Stat(policyFile); os.IsNotExist(err) {
		// 文件不存在，返回 nil 表示加载成功但没有策略
		return nil
	}

	// 读取文件内容
	policyData, err := os.ReadFile(policyFile)
	if err != nil {
		return fmt.Errorf("failed to read policy file: %w", err)
	}

	// 反序列化策略配置
	if err := json.Unmarshal(policyData, config); err != nil {
		return fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	return nil
}

// GetPolicyPath 获取指定模块的策略文件路径
func (m *Manager) GetPolicyPath(moduleName string) string {
	return filepath.Join(m.rootPath, moduleName, "policy.json")
}
