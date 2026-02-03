package kms

import (
	"github.com/tjfoc/gmsm/sm3"
	"golang.org/x/crypto/pbkdf2"
)

// 配置常量
const (
	// ApplicationSalt 是应用的“基因盐”
	// 生产环境建议通过 -ldflags 在编译时注入，防止硬编码泄露
	ApplicationSalt = "MyMonitor_S3cr3t_S@lt_2024_GM"

	// SM4KeyLen SM4 密钥长度固定为 128位 (16字节)
	SM4KeyLen = 16

	// Iterations 迭代次数，越高越安全，但启动越慢
	Iterations = 4096
)

// deriveKey 执行 PBKDF2-HMAC-SM3 算法
// 输入: 原始指纹字符串
// 输出: 16字节 SM4 密钥
func deriveKey(source string) []byte {
	return pbkdf2.Key(
		[]byte(source),          // 密码
		[]byte(ApplicationSalt), // 盐
		Iterations,              // 迭代次数
		SM4KeyLen,               // 输出长度
		sm3.New,                 // 哈希算法: 国密 SM3
	)
}
