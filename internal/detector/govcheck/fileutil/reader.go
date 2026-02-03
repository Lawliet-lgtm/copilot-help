package fileutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileInfo 文件信息
type FileInfo struct {
	Path      string   // 绝对路径
	Name      string   // 文件名
	Size      int64    // 文件大小
	Type      FileType // 文件类型
	Extension string   // 原始扩展名
}

// ReadFileHeader 读取文件头部指定字节数
func ReadFileHeader(filePath string, size int) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	header := make([]byte, size)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("读取文件头失败: %w", err)
	}

	return header[:n], nil
}

// ReadFileContent 读取整个文件内容
func ReadFileContent(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	return content, nil
}

// ReadFileSafe 安全读取文件内容（带大小限制）
func ReadFileSafe(filePath string, maxSize int64) ([]byte, error) {
	// 检查文件大小
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法获取文件信息: %w", err)
	}

	if info.Size() > maxSize {
		return nil, fmt.Errorf("文件过大: %d 字节 (限制: %d 字节)", info.Size(), maxSize)
	}

	return ReadFileContent(filePath)
}

// GetFileInfo 获取文件信息
func GetFileInfo(filePath string) (*FileInfo, error) {
	// 获取绝对路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 获取文件状态
	stat, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("路径是目录而非文件: %s", absPath)
	}

	// 检测文件类型
	fileType, err := DetectFileType(absPath)
	if err != nil {
		fileType = TypeUnknown
	}

	return &FileInfo{
		Path:      absPath,
		Name:      stat.Name(),
		Size:      stat.Size(),
		Type:      fileType,
		Extension: filepath.Ext(stat.Name()),
	}, nil
}

// ValidateFile 验证文件是否可用于检测
func ValidateFile(filePath string, maxSize int64) error {
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("文件不存在: %s", filePath)
		}
		return fmt.Errorf("无法访问文件: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("路径是目录而非文件: %s", filePath)
	}

	if info.Size() == 0 {
		return fmt.Errorf("文件为空: %s", filePath)
	}

	if info.Size() > maxSize {
		return fmt.Errorf("文件过大: %d 字节 (限制: %d 字节)", info.Size(), maxSize)
	}

	return nil
}

// FileExists 检查文件是否存在
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// IsDirectory 检查路径是否为目录
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetFileSize 获取文件大小
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CollectFiles 从目录收集所有文件
func CollectFiles(dirPath string, recursive bool) ([]string, error) {
	var files []string

	if recursive {
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && !isHiddenFile(info.Name()) {
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				files = append(files, absPath)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("遍历目录失败: %w", err)
		}
	} else {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, fmt.Errorf("读取目录失败: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && !isHiddenFile(entry.Name()) {
				absPath, err := filepath.Abs(filepath.Join(dirPath, entry.Name()))
				if err != nil {
					return nil, err
				}
				files = append(files, absPath)
			}
		}
	}

	return files, nil
}

// isHiddenFile 检查是否为隐藏文件
func isHiddenFile(name string) bool {
	return len(name) > 0 && name[0] == '.'
}

// FilterFilesByType 按文件类型过滤文件列表
func FilterFilesByType(files []string, categories ...Category) ([]string, error) {
	if len(categories) == 0 {
		return files, nil
	}

	categorySet := make(map[Category]bool)
	for _, c := range categories {
		categorySet[c] = true
	}

	var filtered []string
	for _, file := range files {
		ft, err := DetectFileType(file)
		if err != nil {
			continue
		}
		if categorySet[ft.Category] {
			filtered = append(filtered, file)
		}
	}

	return filtered, nil
}