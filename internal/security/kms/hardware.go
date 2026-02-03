package kms

import (
	"fmt"
	"net"
	"strings"

	"github.com/shirou/gopsutil/v3/host"
)

// getHardwareFingerprint 组合 "MachineID + MAC" 生成唯一指纹
// 该函数仅在包内部使用
func getHardwareFingerprint() (string, error) {
	// 1. 获取 MachineID
	// gopsutil 屏蔽了 Linux 发行版差异 (/etc/machine-id 等)
	hostInfo, err := host.Info()
	if err != nil {
		return "", fmt.Errorf("failed to get host info: %v", err)
	}
	machineID := strings.TrimSpace(hostInfo.HostID)
	if machineID == "" {
		return "", fmt.Errorf("machine-id is empty")
	}

	// 2. 获取主网卡 MAC
	macAddr, err := getPrimaryMAC()
	if err != nil {
		// 降级策略：如果获取不到 MAC，仅使用 MachineID，并记录日志
		// 这里的 "00..." 是为了保证指纹格式的一致性
		macAddr = "00:00:00:00:00:00"
	}

	// 3. 组合
	return fmt.Sprintf("%s|%s", machineID, macAddr), nil
}

// getPrimaryMAC 获取第一个非回环、已启用的物理网卡 MAC
func getPrimaryMAC() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// 过滤 Loopback (lo) 和 Down 状态的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		mac := iface.HardwareAddr.String()
		if mac != "" {
			return mac, nil
		}
	}
	return "", fmt.Errorf("no valid mac address found")
}
