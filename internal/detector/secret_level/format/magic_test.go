package format

import (
	"encoding/hex"
	"testing"

	"linuxFileWatcher/internal/detector/secret_level/model"
)

func TestIdentifyType(t *testing.T) {
	tests := []struct {
		name   string
		hexStr string // 模拟文件头的 Hex 字符串
		want   model.FileType
	}{
		// 50 4B 03 04 -> Zip/Office
		{"DOCX_Header", "504b030414000600", model.TypeOffice},
		
		// D0 CF 11 E0 -> OLE2/Binary
		{"DOC_Header", "d0cf11e0a1b11ae1", model.TypeBinary},
		
		// 25 50 44 46 -> PDF
		{"PDF_Header", "255044462d312e35", model.TypePDF},
		
		// 89 50 4E 47 -> PNG
		{"PNG_Header", "89504e470d0a1a0a", model.TypeImage},
		
		// 纯文本 (ASCII)
		{"TXT_ASCII", "48656c6c6f20576f726c64", model.TypeText}, // "Hello World"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, _ := hex.DecodeString(tt.hexStr)
			if got := IdentifyType(header); got != tt.want {
				t.Errorf("IdentifyType() = %v, want %v", got, tt.want)
			}
		})
	}
}