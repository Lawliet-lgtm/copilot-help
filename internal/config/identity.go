package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/shirou/gopsutil/v3/host"
)

// =========================================================================
// 1. 编译时注入变量 (Build-Time Variables)
// 通过 -ldflags -X 修改
// =========================================================================

var (
	// Version 软件版本
	// 格式: YYYYMMDD_厂商自定义
	Version string = "00000000_DevBuild"

	// Vendor 厂商名称
	Vendor string = "OpenSource"

	// CommitID Git 提交哈希
	CommitID string = "HEAD"

	// BuildTime 编译时间
	BuildTime string = "Unknown"
)

// =========================================================================
// 2. 运行时变量与配置 (Runtime Variables)
// =========================================================================

var (
	// DeviceID 平台下发的唯一标识
	// 默认为空，启动时尝试从本地文件加载
	DeviceID string

	// HardwareFingerprint 本机硬件指纹
	// 基于 machine-id 计算，用于向平台申请 DeviceID
	HardwareFingerprint string

	// DefaultDataDir 数据存储目录
	// 默认指向 Linux FHS 标准路径，但允许编译时注入修改适配特殊系统
	DefaultDataDir = "/var/lib/linuxFileWatcher"

	// idFilename ID 文件名
	idFilename = "agent.id"

	// idFilePath 最终决定的 ID 文件绝对路径 (在 InitIdentity 中计算)
	idFilePath string

	// mu 读写锁，保护 DeviceID 的并发读写
	mu sync.RWMutex
)

// =========================================================================
// 3. 核心生命周期方法
// =========================================================================

// InitIdentity 初始化身份信息
// 必须在 main.go 启动时最先调用
func InitIdentity() error {
	// 1. 计算硬件指纹 (永远需要，用于注册或校验)
	fp, err := generateHardwareFingerprint()
	if err != nil {
		return fmt.Errorf("hardware fingerprint init failed: %v", err)
	}
	HardwareFingerprint = fp

	// 2. 智能确定存储路径 (适配不同环境)
	resolveIDFilePath()

	// 3. 尝试加载已保存的 DeviceID
	if err := loadDeviceID(); err != nil {
		// 加载失败不报错，仅打印信息，说明是首次启动或未注册
		fmt.Printf("[Identity] DeviceID not found at %s. Waiting for registration.\n", idFilePath)
	} else {
		fmt.Printf("[Identity] Loaded DeviceID: %s\n", DeviceID)
	}

	return nil
}

// IsRegistered 判断是否已完成注册
func IsRegistered() bool {
	mu.RLock()
	defer mu.RUnlock()
	return DeviceID != ""
}

// GetUserAgent 根据注册状态生成符合通信规范的 UA
// 逻辑:
// - 已注册: device-id / soft_version (vendor-name)
// - 未注册: soft_version (vendor-name)
func GetUserAgent() string {
	mu.RLock()
	defer mu.RUnlock()

	// 截断保护
	v := limitString(Version, 32)
	ven := limitString(Vendor, 32)

	// 情况 A: 未注册
	if DeviceID == "" {
		return fmt.Sprintf("%s (%s)", v, ven)
	}

	// 情况 B: 已注册
	did := limitString(DeviceID, 64)
	return fmt.Sprintf("%s / %s (%s)", did, v, ven)
}

// GetFullVersionInfo 获取详细调试信息
func GetFullVersionInfo() string {
	mu.RLock()
	defer mu.RUnlock()
	status := "Registered"
	if DeviceID == "" {
		status = "Unregistered"
	}
	return fmt.Sprintf(
		"Version:     %s\nVendor:      %s\nStatus:      %s\nDeviceID:    %s\nStoragePath: %s\nHW-FP:       %s\nBuilt:       %s",
		Version, Vendor, status, DeviceID, idFilePath, HardwareFingerprint, BuildTime,
	)
}

// =========================================================================
// 4. 持久化与存储逻辑 (Expert Level)
// =========================================================================

// UpdateAndPersistDeviceID 更新并持久化 DeviceID
// 包含目录创建、文件解锁、写入、文件锁定流程
func UpdateAndPersistDeviceID(newID string) error {
	if newID == "" {
		return fmt.Errorf("device id cannot be empty")
	}

	mu.Lock()
	defer mu.Unlock()

	// 1. 确保目录存在
	// 0755: rwxr-xr-x, 保证 Agent 能进入，其他用户可读但不可写(除非是Root)
	dir := filepath.Dir(idFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir %s: %v", dir, err)
	}

	// 2. 尝试解锁文件 (chattr -i)
	// 如果是 Root 运行且文件存在，这一步会移除不可变属性，允许写入
	_ = toggleImmutable(idFilePath, false)

	// 3. 写入文件
	// O_TRUNC: 清空重写
	// 0600: 仅所有者(Root)可读写，极其重要！
	file, err := os.OpenFile(idFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open id file: %v", err)
	}

	_, writeErr := file.WriteString(newID)
	syncErr := file.Sync() // 确保落盘
	closeErr := file.Close()

	if writeErr != nil || syncErr != nil || closeErr != nil {
		return fmt.Errorf("failed to write id file: %v | %v | %v", writeErr, syncErr, closeErr)
	}

	// 4. 锁定文件 (chattr +i)
	// 防止被误删或被恶意脚本篡改
	if err := toggleImmutable(idFilePath, true); err != nil {
		// 锁定失败只打印警告，不阻断流程 (可能是文件系统不支持或非Root)
		fmt.Printf("[Identity] Warning: Failed to lock file attribute: %v\n", err)
	}

	// 5. 更新内存
	DeviceID = newID
	fmt.Printf("[Identity] DeviceID persisted securely at %s\n", idFilePath)
	return nil
}

// resolveIDFilePath 智能路径判定策略
func resolveIDFilePath() {
	// 策略 A: 开发环境 (Windows/Mac) -> 当前目录
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		wd, _ := os.Getwd()
		idFilePath = filepath.Join(wd, idFilename)
		return
	}

	// 策略 B: 生产环境 (Linux Root) -> 标准 /var/lib
	if os.Geteuid() == 0 {
		idFilePath = filepath.Join(DefaultDataDir, idFilename)
		return
	}

	// 策略 C: 降级模式 (Linux 非 Root) -> 当前目录
	// 这种情况通常发生在 CI 测试或本地调试中
	fmt.Println("[Identity] Warning: Not running as root, falling back to local directory for data storage.")
	wd, _ := os.Getwd()
	idFilePath = filepath.Join(wd, idFilename)
}

func loadDeviceID() error {
	content, err := os.ReadFile(idFilePath)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(string(content))
	if id == "" {
		return fmt.Errorf("empty device id file")
	}

	mu.Lock()
	DeviceID = id
	mu.Unlock()
	return nil
}

// =========================================================================
// 5. 内部工具函数
// =========================================================================

// toggleImmutable 切换文件的不可变属性 (chattr +/-i)
func toggleImmutable(path string, enable bool) error {
	// 只有 Linux Root 用户才支持此操作
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		return nil
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	op := "+i"
	if !enable {
		op = "-i"
	}

	// 调用系统 chattr 命令
	cmd := exec.Command("chattr", op, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("exec chattr %s failed: %v, out: %s", op, err, string(output))
	}
	return nil
}

func generateHardwareFingerprint() (string, error) {
	info, err := host.Info()
	if err != nil {
		return "", err
	}

	rawID := strings.TrimSpace(info.HostID)
	// 兜底逻辑：容器环境可能没有 machine-id
	if rawID == "" {
		if info.Hostname != "" {
			rawID = info.Hostname
		} else {
			return "", fmt.Errorf("machine-id and hostname are empty")
		}
	}

	// 使用 SHA256 规范化指纹长度，且不暴露原始信息
	hash := sha256.Sum256([]byte(rawID))
	return hex.EncodeToString(hash[:]), nil
}

func limitString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
