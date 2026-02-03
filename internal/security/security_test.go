package security

import (
	"bytes"
	"testing"
)

// TestSecurityFacade 验证整个安全模块的门面接口
func TestSecurityFacade(t *testing.T) {
	// 1. 测试初始化
	err := Setup()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 2. 测试本地加解密 (模拟 SQLite 存储场景)
	sensitiveData := []byte("User Password or Secret Log")

	// 加密
	encrypted, err := EncryptLocal(sensitiveData)
	if err != nil {
		t.Fatalf("EncryptLocal failed: %v", err)
	}
	t.Logf("Encrypted Data Length: %d", len(encrypted))

	// 解密
	decrypted, err := DecryptLocal(encrypted)
	if err != nil {
		t.Fatalf("DecryptLocal failed: %v", err)
	}

	// 验证一致性
	if !bytes.Equal(sensitiveData, decrypted) {
		t.Errorf("Data mismatch after decrypt.\nOriginal: %s\nDecrypted: %s", sensitiveData, decrypted)
	} else {
		t.Log("Local Encryption/Decryption Test Passed")
	}

	// 3. 测试多次调用 Setup (验证幂等性)
	err = Setup()
	if err != nil {
		t.Errorf("Subsequent Setup call shouldn't fail: %v", err)
	}
}
