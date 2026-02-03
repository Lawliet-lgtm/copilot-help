package format

import (
	"bytes"
	"encoding/hex"
	"strings"

	"linuxFileWatcher/internal/detector/secret_level/model"
)

// IdentifyType 根据文件头识别类型
func IdentifyType(header []byte) model.FileType {
	if len(header) < 5 {
		return model.TypeUnknown
	}

	// 1. Zip / OOXML / OFD
	// PK.. (0x50 0x4B 0x03 0x04)
	if bytes.HasPrefix(header, []byte{0x50, 0x4B, 0x03, 0x04}) {
		return model.TypeOffice
	}

	// 2. PDF
	// %PDF-
	if bytes.HasPrefix(header, []byte{0x25, 0x50, 0x44, 0x46, 0x2D}) {
		return model.TypePDF
	}

	// 3. RTF
	// {\rtf
	if bytes.HasPrefix(header, []byte{0x7B, 0x5C, 0x72, 0x74, 0x66}) {
		// RTF 也是一种文本，但为了走特定的 RTF 逻辑（如果有的话），或者
		// 由于我们把 RTF 逻辑做进了 BinaryScanner，这里返回 TypeBinary 比较稳妥
		// 这样它会直接进入 BinaryScanner 进行转义序列搜索
		return model.TypeBinary
	}

	// 4. OLE2 (旧版 Office)
	if bytes.HasPrefix(header, []byte{0xD0, 0xCF, 0x11, 0xE0}) {
		return model.TypeBinary
	}

	// 5. 常见图片
	h := hex.EncodeToString(header[:4])
	switch {
	case strings.HasPrefix(h, "ffd8ff"): // JPEG
		return model.TypeImage
	case strings.HasPrefix(h, "89504e47"): // PNG
		return model.TypeImage
	case strings.HasPrefix(h, "47494638"): // GIF
		return model.TypeImage
	case strings.HasPrefix(h, "424d"): // BMP
		return model.TypeImage
	case strings.HasPrefix(h, "49492a00") || strings.HasPrefix(h, "4d4d002a"): // TIFF
		return model.TypeImage
	}

	// 6. 文本/代码类
	if isText(header) {
		return model.TypeText
	}

	return model.TypeBinary
}

func isText(data []byte) bool {
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		data = data[3:]
	}
	n := 0
	for _, b := range data {
		if n > 256 {
			break
		}
		if b < 32 && b != 9 && b != 10 && b != 13 {
			return false
		}
		n++
	}
	return true
}