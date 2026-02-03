package govcheck

import (
	"context"

	"linuxFileWatcher/internal/model"
)

// Detector 公文版式检测器接口
// 实现 SubDetector 接口，可被上游 Manager 调度
type Detector interface {
	// DetectFile 检测单个文件是否为公文格式
	// 返回通用中间结果，由 Manager 组装成 AlertRecord
	DetectFile(ctx context.Context, filePath string) (*model.SubDetectResult, error)
}

// Config 公文版式检测配置
type Config struct {
	// 检测参数
	Threshold   float64 // 判定阈值 (0-1)，默认 0.6
	Timeout     int     // 超时时间（秒），默认 30
	MaxFileSize int64   // 最大文件大小（字节），默认 100MB

	// OCR 配置
	EnableOCR   bool   // 是否启用 OCR
	OCRLanguage string // OCR 语言，默认 "chi_sim+eng"

	// 评分权重
	TextWeight  float64 // 文本特征权重，默认 0.7
	StyleWeight float64 // 版式特征权重，默认 0.3

	// 调试选项
	Verbose bool // 详细模式
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		Threshold:   0.6,
		Timeout:     30,
		MaxFileSize: 100 * 1024 * 1024,
		EnableOCR:   true,
		OCRLanguage: "chi_sim+eng",
		TextWeight:  0.7,
		StyleWeight: 0.3,
		Verbose:     false,
	}
}

// NewDetector 创建公文版式检测器实例
func NewDetector(cfg Config) Detector {
	return newService(cfg)
}