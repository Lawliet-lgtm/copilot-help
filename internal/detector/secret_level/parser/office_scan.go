package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"strings"
	"linuxFileWatcher/internal/detector/secret_level/engine"
	"linuxFileWatcher/internal/detector/secret_level/model"
)

// OfficeScanner 针对 OOXML (docx, xlsx) 和 OFD
type OfficeScanner struct{}

func NewOfficeScanner() *OfficeScanner {
	return &OfficeScanner{}
}

func (s *OfficeScanner) Detect(ctx context.Context, reader io.ReaderAt, size int64) (*model.ScanResult, error) {
	// 1. 作为 Zip 打开 (OFD 也是 Zip)
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, err 
	}
	
	for _, f := range zipReader.File {
		// Context check
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 2. 目标文件筛选 (增加了 OFD 支持)
		isTarget := false
		name := f.Name

		// OOXML 规则
		if name == "word/document.xml" || 
		   (strings.HasPrefix(name, "word/header")) ||
		   name == "xl/sharedStrings.xml" ||
		   (strings.HasPrefix(name, "ppt/slides/slide")) {
			isTarget = true
		}

		// OFD 规则 (Doc_0/Pages/Page_N/Content.xml)
		// 也可以扫描 Doc_0/Document.xml (元数据)
		if strings.HasSuffix(name, "Content.xml") && strings.Contains(name, "Pages") {
			isTarget = true
		}

		if !isTarget {
			continue
		}

		// 3. 提取并检测
		rc, err := f.Open()
		if err != nil {
			continue
		}
		
		// 稍微放大一点限制，OFD 的 Content.xml 可能比较啰嗦
		content := extractTextFromXML(rc, 20480) // 20KB limit
		rc.Close()

		if hit, level, text := engine.MatchContent(content); hit {
			return &model.ScanResult{
				IsSecret:    true,
				Level:       level,
				MatchedText: text + " (in " + name + ")",
			}, nil
		}
	}

	return &model.ScanResult{IsSecret: false}, nil
}

// extractTextFromXML 保持不变
func extractTextFromXML(r io.Reader, limit int) string {
	decoder := xml.NewDecoder(io.LimitReader(r, int64(limit)))
	var buf bytes.Buffer
	for {
		token, err := decoder.Token()
		if err != nil { break }
		switch t := token.(type) {
		case xml.CharData:
			buf.Write(t)
			buf.WriteByte(' ')
		}
		if buf.Len() >= limit { break }
	}
	return buf.String()
}