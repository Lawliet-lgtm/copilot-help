package integrity

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetSelfExecutablePath 获取当前进程二进制文件的绝对路径
// 兼容主流 Linux 发行版，并自动解析软链接
func GetSelfExecutablePath() (string, error) {
	// 1. 核心方法: os.Executable
	// 在 Linux 上底层通常读取 /proc/self/exe
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %v", err)
	}

	// 2. 解析软链接 (关键步骤)
	// /proc/self/exe 本身就是一个软链接，或者用户可能通过软链接启动程序
	// 我们需要监控的是“实体文件”，而不是链接本身
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink: %v", err)
	}

	// 3. 确保是绝对路径
	absPath, err := filepath.Abs(realPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %v", err)
	}

	return absPath, nil
}
