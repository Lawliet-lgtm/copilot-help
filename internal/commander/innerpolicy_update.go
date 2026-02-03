package commander

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var path = "/C2/policy/update"

// 负责：拼接URL -> 下载文件 -> 保存临时文件 -> 触发解压
func DownloadAndUnzip(serverAddr, filename, destDir string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	url := fmt.Sprintf("%s%s?filename=%s", serverAddr, path, filename)

	fmt.Printf("[Update] Fetching: %s\n", url)
	// 发GET请求
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error: status %s", resp.Status)
	}

	// 1. 创建临时文件
	tmpFile, err := os.CreateTemp("", "policy-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	defer tmpFile.Close()

	// 2. 将响应体保存到文件
	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save zip: %w", err)
	}
	tmpFile.Close()

	// 3. 执行解压逻辑
	return unzip(tmpPath, destDir)
}

// 负责打开 zip 包并遍历里面的文件
func unzip(src, dest string) error {

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open zip failed: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(absDest, 0755); err != nil {
		return err
	}

	// 遍历 zip 包里的每一个文件或目录
	for _, f := range r.File {
		// --- 新增：处理文件名中的中文乱码 ---
		fileName := decodeName(f)

		// 使用转换后的 fileName 构建路径
		fpath := filepath.Join(absDest, fileName)

		// 路径穿越防御
		if !strings.HasPrefix(fpath, filepath.Clean(absDest)+string(os.PathSeparator)) && fpath != absDest {
			continue // 忽略非法路径
		}

		// 如果该项是一个目录，则创建它
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			continue
		}

		// 如果是文件，调用 extractFile 进行写入
		if err := extractFile(f, fpath); err != nil {
			return err
		}
	}
	fmt.Printf("[Update] Successfully extracted to: %s\n", absDest)
	return nil
}

// decodeName 是一个工具函数，专门用来修复 ZIP 文件名的编码问题
func decodeName(f *zip.File) string {
	// ZIP 规范中，Flags 的第 11 位如果为 1，说明文件名已经是 UTF-8 编码
	if f.Flags&0x800 != 0 {
		return f.Name
	}

	// 如果没有 UTF-8 标志位，假设它是 Windows 下常用的 GBK 编码进行转换
	// 这样在 Linux (UTF-8) 下就能正确显示中文名了
	i := simplifiedchinese.GBK.NewDecoder()
	decoded, _, err := transform.String(i, f.Name)
	if err != nil {
		// 如果转码失败，则返回原名作为保底策略
		return f.Name
	}
	return decoded
}

// 真正执行将解压后的数据写入
func extractFile(f *zip.File, destPath string) error {
	os.MkdirAll(filepath.Dir(destPath), 0755)
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
