package secret_level

import (
	"context"
	"linuxFileWatcher/internal/model" // 引用根目录的 model
)

// Detector 定义接口
type Detector interface {
	// DetectFile 检测单个文件
	// 返回通用中间结果，由 Manager 组装成 AlertRecord
	DetectFile(ctx context.Context, filePath string) (*model.SubDetectResult, error)
}

// Config 组件配置
type Config struct {
	EnableOCR      bool 
	OCRMaxFileSize int64 
}

// NewDetector 创建实例
func NewDetector(cfg Config) Detector {
	return newService(cfg)
}