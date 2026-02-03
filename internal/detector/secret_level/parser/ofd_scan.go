package parser

import (
	"archive/zip"
	"context"
	"io"
	"strings"

	"linuxFileWatcher/internal/detector/secret_level/engine"
	"linuxFileWatcher/internal/detector/secret_level/model"
)

// OFDScanner 针对国产 OFD 格式
// 结构：ZIP 容器 -> Doc_*/Pages/Page_*/Content.xml
type OFDScanner struct{}

func NewOFDScanner() *OFDScanner {
	return &OFDScanner{}
}

func (s *OFDScanner) Detect(ctx context.Context, reader io.ReaderAt, size int64) (*model.ScanResult, error) {
	// 1. 打开 Zip
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, err
	}

	// 2. 遍历 Zip 寻找页面内容
	// 典型路径: Doc_0/Pages/Page_0/Content.xml
	// 我们只关心 xml 文件，且路径包含 "Pages" 和 "Content.xml"
	
	// 同样限制扫描的文件数量，防止 OFD 炸弹
	scannedFiles := 0
	const maxFiles = 10

	for _, f := range zipReader.File {
		// 快速过滤
		if !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		
		// 路径特征匹配：OFD 的正文内容都在 Content.xml 里
		// 另外 OFD 也有元数据文件，但也可能包含密级
		// 策略：只要是 xml 且不是 manifest 等无关文件，尽量都扫一下，重点关注 Content.xml
		isContent := strings.Contains(f.Name, "Content.xml")
		isDocInfo := strings.Contains(f.Name, "Document.xml") // 有时密级在元数据里

		if !isContent && !isDocInfo {
			continue
		}
		
		if scannedFiles >= maxFiles {
			break
		}

		// 检查 context
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 3. 提取并检测
		rc, err := f.Open()
		if err != nil {
			continue
		}
		
		// 复用 office_scan.go 中的 extractTextFromXML
		// OFD 的 XML 结构也是标签包围文本，可以直接提取纯文本
		content := extractTextFromXML(rc, 10240) // 10KB 限制
		rc.Close()
		scannedFiles++

		if hit, level, text := engine.MatchContent(content); hit {
			return &model.ScanResult{
				IsSecret:    true,
				Level:       level,
				MatchedText: text + " (in OFD " + f.Name + ")",
			}, nil
		}
	}

	return &model.ScanResult{IsSecret: false}, nil
}