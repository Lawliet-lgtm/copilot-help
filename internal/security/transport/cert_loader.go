package transport

import (
	"fmt"
	"os"

	"github.com/tjfoc/gmsm/gmtls"
	"github.com/tjfoc/gmsm/x509"
)

// TLSConfigOptions 定义加载证书所需的路径参数
type TLSConfigOptions struct {
	CAPath     string // 根证书路径 (用于验证服务端)
	CertPath   string // 客户端证书路径 (用于证明自己)
	KeyPath    string // 客户端私钥路径
	ServerName string // 服务端证书的 Common Name (可选，用于覆盖校验)
}

// buildTLSConfig 读取国密证书文件并构建 GM-TLS 配置
func buildTLSConfig(opts TLSConfigOptions) (*gmtls.Config, error) {
	// 1. 加载 CA 根证书
	caCert, err := os.ReadFile(opts.CAPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %v", err)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	// 2. 加载客户端国密证书对 (Cert + Key)
	clientCert, err := gmtls.LoadX509KeyPair(opts.CertPath, opts.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client keypair: %v", err)
	}

	// 3. 构建 GM-TLS 配置
	config := &gmtls.Config{
		RootCAs:      caCertPool,                    // 信任的 CA
		Certificates: []gmtls.Certificate{clientCert}, // 发送给服务端的客户端证书
		MinVersion:   gmtls.VersionTLS12,            // 强制 TLS 1.2+ (安全基线)
		ServerName:   opts.ServerName,               // 如果 IP 直连，可能需要手动指定域名以通过校验
		// 国密包会自动选择合适的算法套件
	}

	return config, nil
}
