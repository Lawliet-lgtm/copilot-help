// Package gmcipher
package gmcipher

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"github.com/tjfoc/gmsm/sm4"
)

// ==========================================
// 接口定义 (为了解耦)
// ==========================================

// KeyProvider 定义获取密钥的接口
// 这样 SM4 引擎不需要依赖具体的 KMS 实现，方便测试
type KeyProvider interface {
	GetKey() ([]byte, error)
}

// ==========================================
// SM4 引擎结构体
// ==========================================

type SM4Engine struct {
	keyProvider KeyProvider
}

// NewSM4Engine 创建一个 SM4 引擎实例
func NewSM4Engine(kp KeyProvider) *SM4Engine {
	return &SM4Engine{
		keyProvider: kp,
	}
}

// ==========================================
// 核心功能：加密 (Encrypt)
// ==========================================

// Encrypt 对明文进行 SM4-CBC 加密
// 输出格式: [16字节随机IV] + [密文]
func (e *SM4Engine) Encrypt(plaintext []byte) ([]byte, error) {
	// 1. 获取密钥
	key, err := e.keyProvider.GetKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %v", err)
	}

	// 2. 创建 SM4 Cipher Block
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("invalid sm4 key: %v", err)
	}

	// 3. 填充 (PKCS#7 Padding)
	// SM4 分组长度为 16 字节
	paddedText := pkcs7Padding(plaintext, sm4.BlockSize)

	// 4. 生成随机 IV (16字节)
	// 即使每次加密相同的内容，IV 不同也会导致密文完全不同 (安全性要求)
	ciphertext := make([]byte, sm4.BlockSize+len(paddedText))
	iv := ciphertext[:sm4.BlockSize] // 前16字节存放 IV
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate iv: %v", err)
	}

	// 5. 执行 CBC 加密
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[sm4.BlockSize:], paddedText)

	return ciphertext, nil
}

// ==========================================
// 核心功能：解密 (Decrypt)
// ==========================================

// Decrypt 对密文进行 SM4-CBC 解密
// 输入格式必须是: [16字节随机IV] + [密文]
func (e *SM4Engine) Decrypt(cipherBlob []byte) ([]byte, error) {
	// 1. 基础校验
	if len(cipherBlob) < sm4.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	// 2. 获取密钥
	key, err := e.keyProvider.GetKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %v", err)
	}

	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("invalid sm4 key: %v", err)
	}

	// 3. 提取 IV
	iv := cipherBlob[:sm4.BlockSize]
	actualCiphertext := cipherBlob[sm4.BlockSize:]

	// 密文长度必须是分组的整数倍
	if len(actualCiphertext)%sm4.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	// 4. 执行 CBC 解密
	mode := cipher.NewCBCDecrypter(block, iv)
	// 直接在原切片上解密 (节省内存)，也可以 make 新的
	plaintext := make([]byte, len(actualCiphertext))
	mode.CryptBlocks(plaintext, actualCiphertext)

	// 5. 去除填充 (PKCS#7 Unpadding)
	unpaddedText, err := pkcs7Unpadding(plaintext)
	if err != nil {
		return nil, fmt.Errorf("unpadding failed: %v", err)
	}

	return unpaddedText, nil
}

// ==========================================
// 辅助工具：PKCS#7 填充逻辑
// ==========================================

// pkcs7Padding 补码
func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// pkcs7Unpadding 去码
func pkcs7Unpadding(origData []byte) ([]byte, error) {
	length := len(origData)
	if length == 0 {
		return nil, errors.New("input data empty")
	}
	// 获取最后一个字节的值，即为填充的数量
	unpadding := int(origData[length-1])

	if unpadding > length || unpadding == 0 {
		return nil, errors.New("invalid padding")
	}

	// 校验填充字节的正确性 (可选，为了严谨推荐加上)
	// PKCS#7 要求填充的字节值必须等于填充长度
	for i := length - unpadding; i < length; i++ {
		if origData[i] != byte(unpadding) {
			return nil, errors.New("invalid padding bytes")
		}
	}

	return origData[:length-unpadding], nil
}
