package parser

import (
	"context"
	"io"

	"github.com/ledongthuc/pdf"
	"linuxFileWatcher/internal/detector/secret_level/engine"
	"linuxFileWatcher/internal/detector/secret_level/model"
)

// PDFScanner 使用 ledongthuc/pdf 库进行纯文本提取
type PDFScanner struct{}

func NewPDFScanner() *PDFScanner {
	return &PDFScanner{}
}

func (s *PDFScanner) Detect(ctx context.Context, reader io.ReaderAt, size int64) (*model.ScanResult, error) {
	// ledongthuc/pdf 需要一个 ReaderAt 和 size
	// 注意：PDF 解析非常消耗 CPU 和内存，必须要限制页数
	
	// 1. 创建 PDF Reader
	pdfReader, err := pdf.NewReader(reader, size)
	if err != nil {
		return nil, err
	}

	totalPage := pdfReader.NumPage()
	
	// 2. 策略：只检查前 5 页。
	// 密级标志如果不在封面或前几页，而在第 100 页，这不符合公文规范。
	// 限制页数能极大提升性能。
	scanPages := 5
	if totalPage < scanPages {
		scanPages = totalPage
	}

	for i := 1; i <= scanPages; i++ {
		// 检查 context 取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 3. 提取单页文本
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// GetPlainText 可能会比较慢
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		// 4. 匹配检测
		// PDF 提取出来的文本可能包含乱码或多余空格，engine.MatchContent 里的正则必须足够鲁棒
		// 我们的正则 \s* 已经能处理多余空格
		if hit, level, text := engine.MatchContent(content); hit {
			return &model.ScanResult{
				IsSecret:    true,
				Level:       level,
				MatchedText: text + " (Page " + string(rune(i+'0')) + ")", // 简单的页码标记
			}, nil
		}
	}

	return &model.ScanResult{IsSecret: false}, nil
}