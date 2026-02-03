package kms

import (
	"encoding/hex"
	"testing"
)

func TestRefactoredKMS(t *testing.T) {
	km := GlobalKeyManager

	// 1. 测试初始化
	err := km.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 2. 测试获取密钥
	key, err := km.GetKey()
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}

	// 3. 验证属性
	if len(key) != 16 {
		t.Errorf("Key length mismatch. Want 16, got %d", len(key))
	}

	t.Logf("Generated Key (Hex): %s", hex.EncodeToString(key))
}
