package security

import (
	"fmt"
	"sync"

	"linuxFileWatcher/internal/security/gmcipher"
	"linuxFileWatcher/internal/security/kms"
	"linuxFileWatcher/internal/security/transport"
)

// localEngine 是一个单例 SM4 引擎，专门用于本地数据（如 SQLite 字段）的加解密
// 它直接复用全局 KMS 的密钥
var (
	localEngine *gmcipher.SM4Engine
	initOnce    sync.Once
)

// Setup 初始化整个安全模块
// 必须在程序启动时最先调用（在 main.go 中）
func Setup() error {
	var err error
	initOnce.Do(func() {
		// 1. 初始化密钥管理服务 (KMS)
		// 这会触发硬件指纹采集和密钥派生
		if e := kms.GlobalKeyManager.Initialize(); e != nil {
			err = fmt.Errorf("security setup failed: %v", e)
			return
		}

		// 2. 初始化本地加解密引擎
		// 用于本地落盘数据的保护
		localEngine = gmcipher.NewSM4Engine(kms.GlobalKeyManager)
	})
	return err
}

// ==========================================
// API 1: 本地数据加解密 (供 SQLite 模块使用)
// ==========================================

// EncryptLocal 使用本机硬件密钥加密数据
// 场景: 敏感日志落盘、数据库字段加密
func EncryptLocal(plaintext []byte) ([]byte, error) {
	if localEngine == nil {
		return nil, fmt.Errorf("security module not setup. call security.Setup() first")
	}
	return localEngine.Encrypt(plaintext)
}

// DecryptLocal 解密本地数据
func DecryptLocal(ciphertext []byte) ([]byte, error) {
	if localEngine == nil {
		return nil, fmt.Errorf("security module not setup")
	}
	return localEngine.Decrypt(ciphertext)
}

// ==========================================
// API 2: 安全网络传输 (供上报模块使用)
// ==========================================

// NewSecureClient 创建一个支持 mTLS 和 SM4 的 HTTP 客户端
// 透传 transport 包的配置对象
func NewSecureClient(opts transport.TLSConfigOptions) (*transport.SecureClient, error) {
	// 确保 KMS 已就绪
	if err := Setup(); err != nil {
		return nil, err
	}
	return transport.NewSecureClient(opts)
}
