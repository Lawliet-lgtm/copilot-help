package transport

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"

	"linuxFileWatcher/internal/config"
	"linuxFileWatcher/internal/security/gmcipher" // 引用子模块 4.2
	"linuxFileWatcher/internal/security/kms"      // 引用子模块 4.1

	"github.com/tjfoc/gmsm/gmtls"
)

// SecureClient 是一个支持国密 SM4 加密 payload 的 GM-TLS 客户端
type SecureClient struct {
	httpClient *http.Client
	sm4Engine  *gmcipher.SM4Engine
}

// NewSecureClient 创建客户端实例
// 参数: 证书路径配置
func NewSecureClient(opts TLSConfigOptions) (*SecureClient, error) {
	// 1. 初始化 KMS (如果尚未初始化)
	// 确保我们有密钥可用
	if err := kms.GlobalKeyManager.Initialize(); err != nil {
		return nil, fmt.Errorf("kms init failed: %v", err)
	}

	// 2. 初始化 SM4 引擎
	// 注入全局的 KeyManager 作为密钥提供者
	engine := gmcipher.NewSM4Engine(kms.GlobalKeyManager)

	// 3. 构建 GM-TLS 配置
	gmTLSConfig, err := buildTLSConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("gm-tls config build failed: %v", err)
	}

	// 4. 初始化 CookieJar 用于会话保持
	// 规范要求：组件后续的所有请求都需携带 Session Cookie
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %v", err)
	}

	// 5. 创建 http.Client，使用 GM-TLS 配置
	// 从全局配置获取HTTP客户端参数
	cfg := config.Get()
	transport := &http.Transport{
		MaxIdleConns:       cfg.Server.MaxIdleConns,
		IdleConnTimeout:    cfg.Server.IdleConnTimeout,
		DisableCompression: true, // 避免压缩破坏加密数据的特征或引起混淆
		DialTLS: func(network, addr string) (net.Conn, error) {
			// 使用国密TLS进行拨号
			return gmtls.Dial(network, addr, gmTLSConfig)
		},
	}

	client := &http.Client{
		Timeout:   cfg.Server.Timeout, // 设置合理的超时
		Jar:       jar,                //自动管理 Cookie
		Transport: transport,
	}

	return &SecureClient{
		httpClient: client,
		sm4Engine:  engine,
	}, nil
}

// PostEncrypted 发送加密 POST 请求
// 流程: 明文 -> SM4加密 -> HTTPS发送 -> 接收响应 -> SM4解密 -> 明文
// PostEncrypted 发送加密 POST 请求
func (c *SecureClient) PostEncrypted(url string, plaintextBody []byte, opts ...RequestOption) ([]byte, error) {
	// ===========================
	// 1. SM4 加密 (业务 Payload)
	// ===========================
	encryptedBody, err := c.sm4Engine.Encrypt(plaintextBody)
	if err != nil {
		return nil, err
	}

	// ===========================
	// 2. 构造基础请求 (暂时使用未压缩的数据)
	// ===========================
	// 我们先假设不需要压缩，直接用加密后的数据创建 Request
	// 这样我们就有了一个 *http.Request 对象，可以用来执行 opts 了
	originalReader := bytes.NewReader(encryptedBody)
	req, err := http.NewRequest("POST", url, originalReader)
	if err != nil {
		return nil, err
	}

	// ===========================
	// 3. 应用默认 Header
	// ===========================
	req.Header.Set("User-Agent", config.GetUserAgent())
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept-Encoding", "gzip")

	// ===========================
	// 4. 应用自定义 Options
	// ===========================
	// 关键点：这里可能会被设置 WithGzipRequest()，从而加上 Content-Encoding: gzip
	for _, opt := range opts {
		opt(req)
	}

	// ===========================
	// 5. 检查并修正 Body (关键修复逻辑)
	// ===========================
	// 检查 Header，如果我们刚才在步骤 4 里设置了 gzip，说明 Body 需要压缩
	if req.Header.Get("Content-Encoding") == "gzip" {
		// 执行压缩
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		if _, err := gw.Write(encryptedBody); err != nil {
			return nil, err
		}
		gw.Close() // 必须 Close 才能写入 Footer

		// 【偷梁换柱】
		// 用压缩后的数据替换掉 Request 里原本的未压缩 Body
		req.Body = io.NopCloser(&buf)
		req.ContentLength = int64(buf.Len()) // 这一点非常重要，必须更新长度
	}

	// ===========================
	// 6. 发送请求
	// ===========================
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// ===========================
	// 7. 处理响应
	// ===========================
	var reader io.ReadCloser = resp.Body
	// 如果响应也是 Gzip 的，解压它
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	respData, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// SM4 解密
	return c.sm4Engine.Decrypt(respData)
}
