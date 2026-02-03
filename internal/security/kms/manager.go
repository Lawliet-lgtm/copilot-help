package kms

import (
	"fmt"
	"sync"
)

// DeviceKeyManager 负责管理与硬件绑定的密钥
// 它是线程安全的，且密钥仅驻留在内存中
type DeviceKeyManager struct {
	key  []byte    // 派生出的 SM4 密钥
	once sync.Once // 保证初始化只执行一次
	mu   sync.RWMutex
}

// GlobalKeyManager 全局单例
var GlobalKeyManager = &DeviceKeyManager{}

// Initialize 初始化密钥管理器
// 流程: 采集硬件 (hardware.go) -> 派生密钥 (kdf.go) -> 存入内存
func (km *DeviceKeyManager) Initialize() error {
	var err error

	// 使用 Once 确保高并发下只初始化一次
	km.once.Do(func() {
		// 1. 调用 hardware.go 中的逻辑
		fingerprint, e := getHardwareFingerprint()
		if e != nil {
			err = fmt.Errorf("kms init error: %v", e)
			return
		}

		// 2. 调用 kdf.go 中的逻辑
		derivedKey := deriveKey(fingerprint)

		// 3. 写入内存
		km.mu.Lock()
		km.key = derivedKey
		km.mu.Unlock()
	})

	return err
}

// GetKey 获取 SM4 密钥副本
// 对外暴露的唯一获取密钥的接口
func (km *DeviceKeyManager) GetKey() ([]byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if len(km.key) == 0 {
		return nil, fmt.Errorf("kms not initialized")
	}

	// 返回副本，防止外部修改底层数组破坏安全性
	keyCopy := make([]byte, len(km.key))
	copy(keyCopy, km.key)

	return keyCopy, nil
}
