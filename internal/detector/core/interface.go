// Package core 定义检测引擎的核心接口和结构
package core

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
)

// Detector 检测插件接口，所有检测子模块都需要实现此接口
type Detector interface {
	// GetName 返回检测器名称
	GetName() string
	
	// Detect 执行检测操作
	// path: 文件路径
	// 返回检测结果和错误
	Detect(path string) (*DetectionResult, error)
	
	// Init 初始化检测器
	// config: 检测器配置
	Init(config interface{}) error
	
	// GetVersion 返回检测器版本
	GetVersion() string
}

// DetectionResult 统一的检测结果结构
type DetectionResult struct {
	// 检测器名称
	DetectorName string `json:"detector_name"`
	
	// 是否检测到敏感信息
	Detected bool `json:"detected"`
	
	// 匹配详情
	Matches []MatchDetail `json:"matches"`
}

// MatchDetail 匹配详情
type MatchDetail struct {
	// 匹配类型
	MatchType string `json:"match_type"`
	
	// 匹配内容
	Content string `json:"content"`
	
	// 匹配位置
	Location string `json:"location"`
	
	// 规则ID
	RuleID int64 `json:"rule_id"`
	
	// 规则描述
	RuleDesc string `json:"rule_desc"`
	
	// 告警类型
	AlertType int `json:"alert_type"`
	
	// 文件摘要
	FileSummary string `json:"file_summary"`
	
	// 关键词上下文
	FileDesc string `json:"file_desc"`
	
	// 文件级别
	FileLevel int `json:"file_level"`
}

// FileInfo 文件信息结构
type FileInfo struct {
	// 文件路径
	Path string `json:"path"`
	
	// 文件名
	Name string `json:"name"`
	
	// 文件大小 (字节)
	Size int64 `json:"size"`
	
	// 文件类型
	Type string `json:"type"`
	
	// 文件扩展名
	Ext string `json:"ext"`
}

// NewFileInfo 创建新的文件信息
func NewFileInfo(path string, size int64) *FileInfo {
	fileType, _ := GetFileType(path)
	return &FileInfo{
		Path: path,
		Name: filepath.Base(path),
		Size: size,
		Ext:  filepath.Ext(path),
		Type: fileType,
	}
}

// isTextFile 检测是否为文本文件
func isTextFile(buf []byte) bool {
	// 检查文件头部是否包含非ASCII字符
	for _, b := range buf {
		if b == 0 {
			// 包含空字节，不是文本文件
			return false
		}
		if b < 32 && b != 9 && b != 10 && b != 13 {
			// 包含控制字符（除了制表符、换行符和回车符），不是文本文件
			return false
		}
	}
	return true
}

// GetFileType 根据文件内容检测文件类型
func GetFileType(path string) (string, error) {
	// 打开文件
	f, err := os.Open(path)
	if err != nil {
		return "other", err
	}
	defer f.Close()
	
	// 读取文件头部
	buf := make([]byte, 261)
	_, err = io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		return "other", err
	}
	
	// 使用filetype库检测文件类型
	if filetype.IsImage(buf) {
		return "image", nil
	} else if filetype.IsArchive(buf) {
		return "archive", nil
	} else if filetype.IsDocument(buf) {
		return "document", nil
	} else if isTextFile(buf) {
		return "text", nil
	}
	
	// 如果无法通过内容检测，回退到后缀名检测
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".log", ".csv", ".json", ".yaml", ".yml":
		return "text", nil
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".tif":
		return "image", nil
	case ".doc", ".docx", ".pdf", ".xls", ".xlsx", ".ppt", ".pptx":
		return "document", nil
	case ".zip", ".rar", ".7z", ".tar", ".gz":
		return "archive", nil
	default:
		return "other", nil
	}
}
