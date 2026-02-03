package integrity

import (
	"encoding/hex"
	"io"
	"os"

	"github.com/tjfoc/gmsm/sm3"
)

// ComputeFileSM3 计算指定文件的 SM3 摘要
// 返回十六进制字符串
func ComputeFileSM3(filePath string) (string, error) {
	// 以只读模式打开
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// 创建 SM3 hasher
	h := sm3.New()

	// 流式拷贝，避免大文件占用过多内存
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	// 计算最终 hash
	hashBytes := h.Sum(nil)
	return hex.EncodeToString(hashBytes), nil
}
