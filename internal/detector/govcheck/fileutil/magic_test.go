package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

// ============================================================
// DetectFileType 魔数检测测试
// ============================================================

func TestDetectFileType_ByMagicNumber(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		content      []byte
		wantCategory Category
		wantExt      string
		wantReliable bool
	}{
		{
			name:         "PDF魔数",
			filename:     "test.bin",
			content:      []byte("%PDF-1.4 test content"),
			wantCategory: CategoryPDF,
			wantExt:      "pdf",
			wantReliable: true,
		},
		{
			name:         "JPEG魔数",
			filename:     "test.bin",
			content:      []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46},
			wantCategory: CategoryImage,
			wantExt:      "jpg",
			wantReliable: true,
		},
		{
			name:         "PNG魔数",
			filename:     "test.bin",
			content:      []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00},
			wantCategory: CategoryImage,
			wantExt:      "png",
			wantReliable: true,
		},
		{
			name:         "GIF87a魔数",
			filename:     "test.bin",
			content:      []byte("GIF87a" + "testdata"),
			wantCategory: CategoryImage,
			wantExt:      "gif",
			wantReliable: true,
		},
		{
			name:         "GIF89a魔数",
			filename:     "test.bin",
			content:      []byte("GIF89a" + "testdata"),
			wantCategory: CategoryImage,
			wantExt:      "gif",
			wantReliable: true,
		},
		{
			name:         "BMP魔数",
			filename:     "test.bin",
			content:      []byte{0x42, 0x4D, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantCategory: CategoryImage,
			wantExt:      "bmp",
			wantReliable: true,
		},
		{
			name:         "TIFF_LE魔数",
			filename:     "test.bin",
			content:      []byte{0x49, 0x49, 0x2A, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantCategory: CategoryImage,
			wantExt:      "tiff",
			wantReliable: true,
		},
		{
			name:         "TIFF_BE魔数",
			filename:     "test.bin",
			content:      []byte{0x4D, 0x4D, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x00},
			wantCategory: CategoryImage,
			wantExt:      "tiff",
			wantReliable: true,
		},
		{
			name:         "ZIP魔数",
			filename:     "test.bin",
			content:      []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00},
			wantCategory: CategoryArchive,
			wantExt:      "zip",
			wantReliable: true,
		},
		{
			name:         "RAR魔数",
			filename:     "test.bin",
			content:      []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00, 0x00},
			wantCategory: CategoryArchive,
			wantExt:      "rar",
			wantReliable: true,
		},
		{
			name:         "7Z魔数",
			filename:     "test.bin",
			content:      []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C, 0x00, 0x00},
			wantCategory: CategoryArchive,
			wantExt:      "7z",
			wantReliable: true,
		},
		{
			name:         "GZIP魔数",
			filename:     "test.bin",
			content:      []byte{0x1F, 0x8B, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00},
			wantCategory: CategoryArchive,
			wantExt:      "gz",
			wantReliable: true,
		},
		{
			name:         "RTF魔数",
			filename:     "test.bin",
			content:      []byte("{\\rtf1\\ansi test content"),
			wantCategory: CategoryText,
			wantExt:      "rtf",
			wantReliable: true,
		},
		{
			name:         "OLE2魔数(DOC)",
			filename:     "test.bin",
			content:      append([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, make([]byte, 100)...),
			wantCategory: CategoryDocument,
			wantExt:      "doc",
			wantReliable: true,
		},
		{
			name:         "WebP魔数",
			filename:     "test.bin",
			content:      []byte("RIFF\x00\x00\x00\x00WEBP"),
			wantCategory: CategoryImage,
			wantExt:      "webp",
			wantReliable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.filename)

			err := os.WriteFile(tmpFile, tt.content, 0644)
			if err != nil {
				t.Fatalf("创建临时文件失败: %v", err)
			}

			fileType, err := DetectFileType(tmpFile)
			if err != nil {
				t.Fatalf("DetectFileType 失败: %v", err)
			}

			if fileType.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", fileType.Category, tt.wantCategory)
			}

			if fileType.Extension != tt.wantExt {
				t.Errorf("Extension = %v, want %v", fileType.Extension, tt.wantExt)
			}

			if fileType.Reliable != tt.wantReliable {
				t.Errorf("Reliable = %v, want %v", fileType.Reliable, tt.wantReliable)
			}

			if tt.wantReliable && fileType.Method != MethodMagic {
				t.Errorf("Method = %v, want %v", fileType.Method, MethodMagic)
			}
		})
	}
}

// ============================================================
// 内容特征检测测试
// ============================================================

func TestDetectFileType_ByContent(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		content      string
		wantCategory Category
		wantExt      string
		wantReliable bool
		wantMethod   DetectionMethod
	}{
		{
			name:         "HTML文档_DOCTYPE",
			filename:     "test.bin",
			content:      "<!DOCTYPE html>\n<html>\n<head><title>Test</title></head>\n<body>Hello</body>\n</html>",
			wantCategory: CategoryText,
			wantExt:      "html",
			wantReliable: true,
			wantMethod:   MethodContent,
		},
		{
			name:         "HTML文档_标签",
			filename:     "test.bin",
			content:      "<html>\n<head><title>Test</title></head>\n<body><div>Hello</div></body>\n</html>",
			wantCategory: CategoryText,
			wantExt:      "html",
			wantReliable: true,
			wantMethod:   MethodContent,
		},
		{
			name:         "XML文档_声明",
			filename:     "test.bin",
			content:      "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<root>\n<item>test</item>\n</root>",
			wantCategory: CategoryText,
			wantExt:      "xml",
			wantReliable: true,
			wantMethod:   MethodContent,
		},
		{
			name:         "XML文档_命名空间",
			filename:     "test.bin",
			content:      "<root xmlns:ns=\"http://example.com\">\n<ns:item>test</ns:item>\n</root>",
			wantCategory: CategoryText,
			wantExt:      "xml",
			wantReliable: true,
			wantMethod:   MethodContent,
		},
		{
			name:         "XHTML文档",
			filename:     "test.bin",
			content:      "<?xml version=\"1.0\"?>\n<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0//EN\">\n<html xmlns=\"http://www.w3.org/1999/xhtml\">\n<head><title>Test</title></head>\n<body>Hello</body>\n</html>",
			wantCategory: CategoryText,
			wantExt:      "html",
			wantReliable: true,
			wantMethod:   MethodContent,
		},
		{
			name:         "EML邮件",
			filename:     "test.bin",
			content:      "From: sender@example.com\nTo: receiver@example.com\nSubject: Test Email\nDate: Mon, 1 Jan 2024 00:00:00 +0000\nMessage-ID: <123@example.com>\n\nThis is a test email.",
			wantCategory: CategoryText,
			wantExt:      "eml",
			wantReliable: true,
			wantMethod:   MethodContent,
		},
		{
			name:         "MHT文档",
			filename:     "test.bin",
			content:      "MIME-Version: 1.0\nContent-Type: multipart/related; boundary=\"----=_NextPart\"\n\n------=_NextPart\nContent-Type: text/html\n\n<html><body>Test</body></html>",
			wantCategory: CategoryText,
			wantExt:      "mht",
			wantReliable: true,
			wantMethod:   MethodContent,
		},
		{
			name:         "纯文本_UTF8",
			filename:     "test.bin",
			content:      "这是一段中文测试文本。\nThis is English text.\n1234567890",
			wantCategory: CategoryText,
			wantExt:      "txt",
			wantReliable: true,
			wantMethod:   MethodContent,
		},
		{
			name:         "纯文本_UTF8_BOM",
			filename:     "test.bin",
			content:      "\xEF\xBB\xBF这是带BOM的UTF-8文本。",
			wantCategory: CategoryText,
			wantExt:      "txt",
			wantReliable: true,
			wantMethod:   MethodContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.filename)

			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("创建临时文件失败: %v", err)
			}

			fileType, err := DetectFileType(tmpFile)
			if err != nil {
				t.Fatalf("DetectFileType 失败: %v", err)
			}

			if fileType.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", fileType.Category, tt.wantCategory)
			}

			if fileType.Extension != tt.wantExt {
				t.Errorf("Extension = %v, want %v", fileType.Extension, tt.wantExt)
			}

			if fileType.Reliable != tt.wantReliable {
				t.Errorf("Reliable = %v, want %v", fileType.Reliable, tt.wantReliable)
			}

			if fileType.Method != tt.wantMethod {
				t.Errorf("Method = %v, want %v", fileType.Method, tt.wantMethod)
			}
		})
	}
}

// ============================================================
// 扩展名检测测试（备选方案）
// ============================================================

func TestDetectFileType_ByExtension_Fallback(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		content      []byte
		wantCategory Category
		wantExt      string
		wantReliable bool
		wantMethod   DetectionMethod
	}{
		{
			name:         "未知内容_TXT扩��名",
			filename:     "test.txt",
			content:      []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}, // 二进制内容
			wantCategory: CategoryText,
			wantExt:      "txt",
			wantReliable: false,
			wantMethod:   MethodExtension,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.filename)

			err := os.WriteFile(tmpFile, tt.content, 0644)
			if err != nil {
				t.Fatalf("创建临时文件失败: %v", err)
			}

			fileType, err := DetectFileType(tmpFile)
			if err != nil {
				t.Fatalf("DetectFileType 失败: %v", err)
			}

			if fileType.Category != tt.wantCategory {
				t.Errorf("Category = %v, want %v", fileType.Category, tt.wantCategory)
			}

			if fileType.Reliable != tt.wantReliable {
				t.Errorf("Reliable = %v, want %v", fileType.Reliable, tt.wantReliable)
			}

			if fileType.Method != tt.wantMethod {
				t.Errorf("Method = %v, want %v", fileType.Method, tt.wantMethod)
			}
		})
	}
}

// ============================================================
// 扩展名伪造检测测试
// ============================================================

func TestDetectFileType_FakeExtension(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		content          []byte
		wantRealCategory Category
		wantRealExt      string
		description      string
	}{
		{
			name:             "PDF伪装成TXT",
			filename:         "fake.txt",
			content:          []byte("%PDF-1.4 This is actually a PDF file"),
			wantRealCategory: CategoryPDF,
			wantRealExt:      "pdf",
			description:      "PDF文件被重命名为.txt",
		},
		{
			name:             "JPEG伪装成PNG",
			filename:         "fake.png",
			content:          []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46},
			wantRealCategory: CategoryImage,
			wantRealExt:      "jpg",
			description:      "JPEG文件被重命名为.png",
		},
		{
			name:             "ZIP伪装成DOC",
			filename:         "fake.doc",
			content:          []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00},
			wantRealCategory: CategoryArchive,
			wantRealExt:      "zip",
			description:      "ZIP文件被重命名为.doc",
		},
		{
			name:             "EXE伪装成PDF",
			filename:         "fake.pdf",
			content:          []byte{0x4D, 0x5A, 0x90, 0x00, 0x03, 0x00, 0x00, 0x00}, // MZ header
			wantRealCategory: CategoryOther,
			wantRealExt:      "",
			description:      "EXE文件被重命名为.pdf",
		},
		{
			name:             "RAR伪装成DOCX",
			filename:         "fake.docx",
			content:          []byte{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00, 0x00},
			wantRealCategory: CategoryArchive,
			wantRealExt:      "rar",
			description:      "RAR文件被重命名为.docx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.filename)

			err := os.WriteFile(tmpFile, tt.content, 0644)
			if err != nil {
				t.Fatalf("创建临时文件失败: %v", err)
			}

			fileType, err := DetectFileType(tmpFile)
			if err != nil {
				t.Fatalf("DetectFileType 失败: %v", err)
			}

			// 应该检测到真实类型，而不是扩展名类型
			if fileType.Category != tt.wantRealCategory {
				t.Errorf("%s: Category = %v, want %v", tt.description, fileType.Category, tt.wantRealCategory)
			}

			if fileType.Extension != tt.wantRealExt {
				t.Errorf("%s: Extension = %v, want %v", tt.description, fileType.Extension, tt.wantRealExt)
			}

			// 通过魔数检测的应该是可靠的
			if tt.wantRealExt != "" && !fileType.Reliable {
				t.Errorf("%s: 应该是可靠的检测结果", tt.description)
			}
		})
	}
}

// ============================================================
// ValidateFileType 测试
// ============================================================

func TestValidateFileType(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     []byte
		wantMatched bool
	}{
		{
			name:        "PDF扩展名匹配",
			filename:    "test.pdf",
			content:     []byte("%PDF-1.4 test"),
			wantMatched: true,
		},
		{
			name:        "PDF扩展名不匹配",
			filename:    "test.txt",
			content:     []byte("%PDF-1.4 test"),
			wantMatched: false,
		},
		{
			name:        "JPEG扩展名匹配",
			filename:    "test.jpg",
			content:     []byte{0xFF, 0xD8, 0xFF, 0xE0},
			wantMatched: true,
		},
		{
			name:        "JPEG扩展名不匹配",
			filename:    "test.png",
			content:     []byte{0xFF, 0xD8, 0xFF, 0xE0},
			wantMatched: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, tt.filename)

			err := os.WriteFile(tmpFile, tt.content, 0644)
			if err != nil {
				t.Fatalf("创建临时文件失败: %v", err)
			}

			matched, detected, expected := ValidateFileType(tmpFile)

			if matched != tt.wantMatched {
				t.Errorf("matched = %v, want %v (detected: %s, expected: %s)",
					matched, tt.wantMatched, detected.Extension, expected.Extension)
			}
		})
	}
}

// ============================================================
// DetectFileTypeStrict 测试
// ============================================================

func TestDetectFileTypeStrict(t *testing.T) {
	tmpDir := t.TempDir()

	// 可靠的检测（魔数）
	pdfFile := filepath.Join(tmpDir, "test.pdf")
	err := os.WriteFile(pdfFile, []byte("%PDF-1.4 test"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	fileType, err := DetectFileTypeStrict(pdfFile)
	if err != nil {
		t.Fatalf("DetectFileTypeStrict 失败: %v", err)
	}

	if fileType.Extension != "pdf" {
		t.Errorf("Extension = %v, want pdf", fileType.Extension)
	}

	// 不可靠的检测（仅扩展名）
	unknownFile := filepath.Join(tmpDir, "test.xyz")
	err = os.WriteFile(unknownFile, []byte{0x00, 0x01, 0x02, 0x03}, 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	fileType, err = DetectFileTypeStrict(unknownFile)
	if err != nil {
		t.Fatalf("DetectFileTypeStrict 失败: %v", err)
	}

	// 严格模式下，不可靠的检测应该返回 Unknown
	if fileType.Extension != "" {
		t.Errorf("严格模式下不可靠的检测应返回空扩展名, got: %v", fileType.Extension)
	}
}

// ============================================================
// IsSupportedForDetection 测试
// ============================================================

func TestIsSupportedForDetection(t *testing.T) {
	tests := []struct {
		name     string
		fileType FileType
		want     bool
	}{
		{"文本类型支持", TypeTXT, true},
		{"HTML类型支持", TypeHTML, true},
		{"文档类型支持", TypeDOCX, true},
		{"PDF类型支持", TypePDF, true},
		{"OFD类型支持", TypeOFD, true},
		{"图片类型支持", TypeJPG, true},
		{"PNG图片支持", TypePNG, true},
		{"压缩类型不支持", TypeZIP, false},
		{"RAR类型不支持", TypeRAR, false},
		{"未知类型不支持", TypeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSupportedForDetection(tt.fileType)
			if got != tt.want {
				t.Errorf("IsSupportedForDetection(%v) = %v, want %v", tt.fileType.Extension, got, tt.want)
			}
		})
	}
}

// ============================================================
// GetUnsupportedReason 测试
// ============================================================

func TestGetUnsupportedReason(t *testing.T) {
	tests := []struct {
		name        string
		fileType    FileType
		wantContain string
	}{
		{"图片文件", TypeJPG, "OCR"},
		{"压缩文件", TypeZIP, "解压"},
		{"未知文件", TypeUnknown, "不支持"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetUnsupportedReason(tt.fileType)
			if got == "" {
				t.Error("期望返回非空原因")
			}
		})
	}
}

// ============================================================
// GetFileTypeByExtension 测试
// ============================================================

func TestGetFileTypeByExtension(t *testing.T) {
	tests := []struct {
		ext      string
		wantExt  string
		wantReli bool
	}{
		{"txt", "txt", false},
		{"TXT", "txt", false},
		{".txt", "txt", false},
		{".TXT", "txt", false},
		{"docx", "docx", false},
		{"DOCX", "docx", false},
		{"pdf", "pdf", false},
		{"ofd", "ofd", false},
		{"jpg", "jpg", false},
		{"jpeg", "jpeg", false},
		{"png", "png", false},
		{"xyz", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := GetFileTypeByExtension(tt.ext)

			if got.Extension != tt.wantExt {
				t.Errorf("GetFileTypeByExtension(%q).Extension = %v, want %v", tt.ext, got.Extension, tt.wantExt)
			}

			// 通过扩展名获取的类型应该标记为不可靠
			if got.Extension != "" && got.Reliable != tt.wantReli {
				t.Errorf("GetFileTypeByExtension(%q).Reliable = %v, want %v", tt.ext, got.Reliable, tt.wantReli)
			}

			if got.Extension != "" && got.Method != MethodExtension {
				t.Errorf("GetFileTypeByExtension(%q).Method = %v, want %v", tt.ext, got.Method, MethodExtension)
			}
		})
	}
}

// ============================================================
// 文件类型检查函数测试
// ============================================================

func TestIsTextFile(t *testing.T) {
	if !IsTextFile(TypeTXT) {
		t.Error("TypeTXT 应该是文本文件")
	}
	if !IsTextFile(TypeHTML) {
		t.Error("TypeHTML 应该是文本文件")
	}
	if IsTextFile(TypeDOCX) {
		t.Error("TypeDOCX 不应该是文本文件")
	}
	if IsTextFile(TypePDF) {
		t.Error("TypePDF 不应该是文本文件")
	}
}

func TestIsDocumentFile(t *testing.T) {
	if !IsDocumentFile(TypeDOCX) {
		t.Error("TypeDOCX 应该是文档文件")
	}
	if !IsDocumentFile(TypeDOC) {
		t.Error("TypeDOC 应该是文档文件")
	}
	if !IsDocumentFile(TypeWPS) {
		t.Error("TypeWPS 应该是文档文件")
	}
	if IsDocumentFile(TypeTXT) {
		t.Error("TypeTXT 不应该是文档文件")
	}
}

func TestIsPdfFile(t *testing.T) {
	if !IsPdfFile(TypePDF) {
		t.Error("TypePDF 应该是PDF文件")
	}
	if IsPdfFile(TypeDOCX) {
		t.Error("TypeDOCX 不应该是PDF文件")
	}
}

func TestIsOfdFile(t *testing.T) {
	if !IsOfdFile(TypeOFD) {
		t.Error("TypeOFD 应该是OFD文件")
	}
	if IsOfdFile(TypePDF) {
		t.Error("TypePDF 不应该是OFD文件")
	}
}

func TestIsImageFile(t *testing.T) {
	imageTypes := []FileType{TypeJPG, TypeJPEG, TypePNG, TypeGIF, TypeBMP, TypeTIFF, TypeTIF, TypeWEBP}
	for _, ft := range imageTypes {
		if !IsImageFile(ft) {
			t.Errorf("%s 应该是图片文件", ft.Extension)
		}
	}

	if IsImageFile(TypeTXT) {
		t.Error("TypeTXT 不应该是图片文件")
	}
}

func TestIsArchiveFile(t *testing.T) {
	archiveTypes := []FileType{TypeZIP, TypeRAR, Type7Z, TypeGZ, TypeTAR}
	for _, ft := range archiveTypes {
		if !IsArchiveFile(ft) {
			t.Errorf("%s 应该是压缩文件", ft.Extension)
		}
	}

	if IsArchiveFile(TypeTXT) {
		t.Error("TypeTXT 不应该是压缩文件")
	}
}

// ============================================================
// IsReliableDetection 测试
// ============================================================

func TestIsReliableDetection(t *testing.T) {
	reliableType := FileType{Extension: "pdf", Reliable: true, Method: MethodMagic}
	unreliableType := FileType{Extension: "txt", Reliable: false, Method: MethodExtension}

	if !IsReliableDetection(reliableType) {
		t.Error("应该返回 true")
	}

	if IsReliableDetection(unreliableType) {
		t.Error("应该返回 false")
	}
}

// ============================================================
// GetDetectionMethod 测试
// ============================================================

func TestGetDetectionMethod(t *testing.T) {
	tests := []struct {
		fileType FileType
		want     DetectionMethod
	}{
		{FileType{Method: MethodMagic}, MethodMagic},
		{FileType{Method: MethodContent}, MethodContent},
		{FileType{Method: MethodExtension}, MethodExtension},
		{FileType{Method: MethodUnknown}, MethodUnknown},
	}

	for _, tt := range tests {
		got := GetDetectionMethod(tt.fileType)
		if got != tt.want {
			t.Errorf("GetDetectionMethod() = %v, want %v", got, tt.want)
		}
	}
}

// ============================================================
// GetSupportedCategories 和 GetAllCategories 测试
// ============================================================

func TestGetSupportedCategories(t *testing.T) {
	categories := GetSupportedCategories()

	if len(categories) == 0 {
		t.Error("支持的分类不应为空")
	}

	hasText := false
	hasDocument := false
	hasPDF := false
	hasOFD := false

	for _, c := range categories {
		switch c {
		case CategoryText:
			hasText = true
		case CategoryDocument:
			hasDocument = true
		case CategoryPDF:
			hasPDF = true
		case CategoryOFD:
			hasOFD = true
		}
	}

	if !hasText {
		t.Error("支持的分类应包含 CategoryText")
	}
	if !hasDocument {
		t.Error("支持的分类应包含 CategoryDocument")
	}
	if !hasPDF {
		t.Error("支持的分类应包含 CategoryPDF")
	}
	if !hasOFD {
		t.Error("支持的分类应包含 CategoryOFD")
	}
}

func TestGetAllCategories(t *testing.T) {
	categories := GetAllCategories()

	if len(categories) != 7 {
		t.Errorf("所有分类应有7个，实际有 %d 个", len(categories))
	}

	expectedCategories := map[Category]bool{
		CategoryText:     false,
		CategoryDocument: false,
		CategoryPDF:      false,
		CategoryOFD:      false,
		CategoryImage:    false,
		CategoryArchive:  false,
		CategoryOther:    false,
	}

	for _, c := range categories {
		expectedCategories[c] = true
	}

	for cat, found := range expectedCategories {
		if !found {
			t.Errorf("缺少分类: %s", cat)
		}
	}
}

// ============================================================
// FileType 结构体测试
// ============================================================

func TestFileType_Fields(t *testing.T) {
	if TypeDOCX.Extension != "docx" {
		t.Errorf("TypeDOCX.Extension = %q, want %q", TypeDOCX.Extension, "docx")
	}
	if TypeDOCX.Category != CategoryDocument {
		t.Errorf("TypeDOCX.Category = %v, want %v", TypeDOCX.Category, CategoryDocument)
	}
	if TypeDOCX.MimeType == "" {
		t.Error("TypeDOCX.MimeType 不应为空")
	}
	if TypeDOCX.Description == "" {
		t.Error("TypeDOCX.Description 不应为空")
	}

	if TypeUnknown.Extension != "" {
		t.Errorf("TypeUnknown.Extension = %q, want empty", TypeUnknown.Extension)
	}
	if TypeUnknown.Category != CategoryOther {
		t.Errorf("TypeUnknown.Category = %v, want %v", TypeUnknown.Category, CategoryOther)
	}
}

// ============================================================
// 边界情况测试
// ============================================================

func TestDetectFileType_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(tmpFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("创建空文件失败: %v", err)
	}

	fileType, err := DetectFileType(tmpFile)
	// 空文件应该返回错误或未知类型
	if err == nil && fileType.Extension != "" && fileType.Reliable {
		t.Error("空文件不应返回可靠的文件类型")
	}
}

func TestDetectFileType_NonExistentFile(t *testing.T) {
	_, err := DetectFileType("/non/existent/file.txt")
	if err == nil {
		t.Error("期望返回错误")
	}
}

func TestDetectFileType_TinyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tiny.bin")

	// 只有 2 字节
	err := os.WriteFile(tmpFile, []byte{0x00, 0x01}, 0644)
	if err != nil {
		t.Fatalf("创建小文件失败: %v", err)
	}

	// 不应崩溃
	_, err = DetectFileType(tmpFile)
	if err != nil {
		t.Logf("小文件检测结果: %v", err)
	}
}