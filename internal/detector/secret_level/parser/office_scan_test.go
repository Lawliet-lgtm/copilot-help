package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"linuxFileWatcher/internal/detector/secret_level/model"
)

// createMockDocx 创建一个内存中的伪造 docx zip 流
func createMockDocx(content string) *bytes.Buffer {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// 写入 word/document.xml
	f, _ := w.Create("word/document.xml")
	// 简单的 Word XML 结构
	xmlBody := `<w:document><w:body><w:p><w:r><w:t>` + content + `</w:t></w:r></w:p></w:body></w:document>`
	f.Write([]byte(xmlBody))

	w.Close()
	return buf
}

func TestOfficeScanner_Detect(t *testing.T) {
	tests := []struct {
		name      string
		docContent string
		wantHit    bool
		wantLevel  model.SecretLevel
	}{
		{"Hit_Secret", "这是一个绝密★文件的正文", true, model.LevelTopSecret},
		{"No_Hit", "这是一个普通文件的正文", false, model.LevelNone},
	}

	scanner := NewOfficeScanner()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. 构造数据
			fileBuf := createMockDocx(tt.docContent)
			reader := bytes.NewReader(fileBuf.Bytes())
			size := int64(fileBuf.Len())

			// 2. 执行检测
			res, err := scanner.Detect(context.Background(), reader, size)
			if err != nil {
				t.Fatalf("Detect failed: %v", err)
			}

			if res.IsSecret != tt.wantHit {
				t.Errorf("IsSecret = %v, want %v", res.IsSecret, tt.wantHit)
			}
			if res.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", res.Level, tt.wantLevel)
			}
		})
	}
}