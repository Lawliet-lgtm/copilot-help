package parser

import (
	"context"
	"io"
	"linuxFileWatcher/internal/detector/secret_level/engine"
	"linuxFileWatcher/internal/detector/secret_level/model"
)

// TextScanner 文本类文件扫描器
// 策略: 读头 + 读尾。避免全量读取大文件。
type TextScanner struct{}

func NewTextScanner() *TextScanner {
	return &TextScanner{}
}

func (s *TextScanner) Detect(ctx context.Context, reader io.ReaderAt, size int64) (*model.ScanResult, error) {
	// 定义读取窗口大小
	const headSize = 4096 // 4KB
	const tailSize = 2048 // 2KB

	// 1. 读取头部
	actualHeadSize := headSize
	if int64(actualHeadSize) > size {
		actualHeadSize = int(size)
	}
	headBuf := make([]byte, actualHeadSize)
	if _, err := reader.ReadAt(headBuf, 0); err != nil && err != io.EOF {
		return nil, err
	}

	// 检测头部
	// 注意: 这里的 string(headBuf) 假设文件是 UTF-8。
	// 如果是 GBK，这可能会导致乱码从而漏报。
	// 考虑到 Go 核心优势和现代环境，这里暂按 UTF-8 处理。
	// 若需严格支持 GBK TXT，需引入 golang.org/x/text 进行探测转换。
	if hit, level, text := engine.MatchContent(string(headBuf)); hit {
		return &model.ScanResult{IsSecret: true, Level: level, MatchedText: text}, nil
	}

	// 如果文件很小，头部已经读完了，就不用读尾部了
	if int64(actualHeadSize) == size {
		return &model.ScanResult{IsSecret: false}, nil
	}

	// 2. 读取尾部
	// 计算尾部起始位置
	tailOffset := size - int64(tailSize)
	if tailOffset < int64(actualHeadSize) {
		tailOffset = int64(actualHeadSize) // 避免重叠
	}
	
	actualTailSize := size - tailOffset
	if actualTailSize <= 0 {
		return &model.ScanResult{IsSecret: false}, nil
	}

	tailBuf := make([]byte, actualTailSize)
	if _, err := reader.ReadAt(tailBuf, tailOffset); err != nil && err != io.EOF {
		return nil, err
	}

	// 检测尾部
	if hit, level, text := engine.MatchContent(string(tailBuf)); hit {
		return &model.ScanResult{IsSecret: true, Level: level, MatchedText: text}, nil
	}

	return &model.ScanResult{IsSecret: false}, nil
}