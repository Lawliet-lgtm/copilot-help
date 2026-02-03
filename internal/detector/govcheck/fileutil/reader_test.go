package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

// ============================================================
// ReadFileHeader 测试
// ============================================================

func TestReadFileHeader(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("Hello, World! This is a test file.")
	err := os.WriteFile(tmpFile, content, 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	tests := []struct {
		name     string
		size     int
		wantLen  int
		wantData string
	}{
		{"读取5字节", 5, 5, "Hello"},
		{"读取10字节", 10, 10, "Hello, Wor"},
		{"读取超过文件长度", 100, len(content), string(content)},
		{"读取0字节", 0, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := ReadFileHeader(tmpFile, tt.size)
			if err != nil {
				t.Fatalf("ReadFileHeader 失败: %v", err)
			}

			if len(header) != tt.wantLen {
				t.Errorf("header 长度 = %d, want %d", len(header), tt.wantLen)
			}

			if string(header) != tt.wantData {
				t.Errorf("header = %q, want %q", string(header), tt.wantData)
			}
		})
	}
}

func TestReadFileHeader_NonExistent(t *testing.T) {
	_, err := ReadFileHeader("/non/existent/file.txt", 10)
	if err == nil {
		t.Error("期望返回错误")
	}
}

// ============================================================
// ReadFileContent 测试
// ============================================================

func TestReadFileContent(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("Test file content for reading.")
	err := os.WriteFile(tmpFile, content, 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	readContent, err := ReadFileContent(tmpFile)
	if err != nil {
		t.Fatalf("ReadFileContent 失��: %v", err)
	}

	if string(readContent) != string(content) {
		t.Errorf("读取内容 = %q, want %q", string(readContent), string(content))
	}
}

func TestReadFileContent_NonExistent(t *testing.T) {
	_, err := ReadFileContent("/non/existent/file.txt")
	if err == nil {
		t.Error("期望返回错误")
	}
}

// ============================================================
// ReadFileSafe 测试
// ============================================================

func TestReadFileSafe(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	content := []byte("Small file content.")
	err := os.WriteFile(tmpFile, content, 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 正常读取
	readContent, err := ReadFileSafe(tmpFile, 1024)
	if err != nil {
		t.Fatalf("ReadFileSafe 失败: %v", err)
	}
	if string(readContent) != string(content) {
		t.Errorf("读取内容 = %q, want %q", string(readContent), string(content))
	}

	// 超过大小限制
	_, err = ReadFileSafe(tmpFile, 5)
	if err == nil {
		t.Error("期望返回文件过大错误")
	}
}

// ============================================================
// GetFileInfo 测试
// ============================================================

func TestGetFileInfo(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.docx")

	content := []byte("test content")
	err := os.WriteFile(tmpFile, content, 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	info, err := GetFileInfo(tmpFile)
	if err != nil {
		t.Fatalf("GetFileInfo 失败: %v", err)
	}

	if info.Name != "test.docx" {
		t.Errorf("Name = %q, want %q", info.Name, "test.docx")
	}

	if info.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", info.Size, len(content))
	}

	if info.Extension != ".docx" {
		t.Errorf("Extension = %q, want %q", info.Extension, ".docx")
	}
}

func TestGetFileInfo_NonExistent(t *testing.T) {
	_, err := GetFileInfo("/non/existent/file.txt")
	if err == nil {
		t.Error("期望返回错误")
	}
}

func TestGetFileInfo_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := GetFileInfo(tmpDir)
	if err == nil {
		t.Error("期望返回目录错误")
	}
}

// ============================================================
// ValidateFile 测试
// ============================================================

func TestValidateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 正常文件
	normalFile := filepath.Join(tmpDir, "normal.txt")
	err := os.WriteFile(normalFile, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	err = ValidateFile(normalFile, 1024)
	if err != nil {
		t.Errorf("正常文件验证失败: %v", err)
	}

	// 空文件
	emptyFile := filepath.Join(tmpDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("创建空文件失败: %v", err)
	}

	err = ValidateFile(emptyFile, 1024)
	if err == nil {
		t.Error("空文件应该验证失败")
	}

	// 不存在的文件
	err = ValidateFile("/non/existent/file.txt", 1024)
	if err == nil {
		t.Error("不存在的文件应该验证失败")
	}

	// 目录
	err = ValidateFile(tmpDir, 1024)
	if err == nil {
		t.Error("目录应该验证失败")
	}

	// 超过大小限制
	err = ValidateFile(normalFile, 1)
	if err == nil {
		t.Error("超过大小限制应该验证失败")
	}
}

// ============================================================
// FileExists 测试
// ============================================================

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "exists.txt")

	err := os.WriteFile(tmpFile, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	if !FileExists(tmpFile) {
		t.Error("文件应该存在")
	}

	if FileExists("/non/existent/file.txt") {
		t.Error("文件不应该存在")
	}

	// 目录不应该被认为是文件
	if FileExists(tmpDir) {
		t.Error("目录不应该被认为是文件存在")
	}
}

// ============================================================
// IsDirectory 测试
// ============================================================

func TestIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")

	err := os.WriteFile(tmpFile, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	if !IsDirectory(tmpDir) {
		t.Error("应该识别为目录")
	}

	if IsDirectory(tmpFile) {
		t.Error("文件不应该识别为目录")
	}

	if IsDirectory("/non/existent/path") {
		t.Error("不存在的路径不应该识别为目录")
	}
}

// ============================================================
// GetFileSize 测试
// ============================================================

func TestGetFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "size.txt")

	content := []byte("12345678901234567890") // 20字节
	err := os.WriteFile(tmpFile, content, 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	size, err := GetFileSize(tmpFile)
	if err != nil {
		t.Fatalf("GetFileSize 失败: %v", err)
	}

	if size != 20 {
		t.Errorf("Size = %d, want 20", size)
	}
}

func TestGetFileSize_NonExistent(t *testing.T) {
	_, err := GetFileSize("/non/existent/file.txt")
	if err == nil {
		t.Error("期望返回错误")
	}
}

// ============================================================
// CollectFiles 测试
// ============================================================

func TestCollectFiles_NonRecursive(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	files := []string{"file1.txt", "file2.docx", "file3.pdf"}
	for _, f := range files {
		err := os.WriteFile(filepath.Join(tmpDir, f), []byte("content"), 0644)
		if err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	// 创建子目录和文件
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}
	err = os.WriteFile(filepath.Join(subDir, "subfile.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("创建子目录文件失败: %v", err)
	}

	// 非递归收集
	collected, err := CollectFiles(tmpDir, false)
	if err != nil {
		t.Fatalf("CollectFiles 失败: %v", err)
	}

	if len(collected) != 3 {
		t.Errorf("收集到 %d 个文件, want 3", len(collected))
	}
}

func TestCollectFiles_Recursive(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建子目录和文件
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("创建子目录失败: %v", err)
	}
	err = os.WriteFile(filepath.Join(subDir, "subfile.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("创建子目录文件失败: %v", err)
	}

	// 递归收集
	collected, err := CollectFiles(tmpDir, true)
	if err != nil {
		t.Fatalf("CollectFiles 失败: %v", err)
	}

	if len(collected) != 2 {
		t.Errorf("收集到 %d 个文件, want 2", len(collected))
	}
}

func TestCollectFiles_SkipHidden(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建普通文件
	err := os.WriteFile(filepath.Join(tmpDir, "normal.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建隐藏文件
	err = os.WriteFile(filepath.Join(tmpDir, ".hidden.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("创建隐藏文件失败: %v", err)
	}

	collected, err := CollectFiles(tmpDir, false)
	if err != nil {
		t.Fatalf("CollectFiles 失败: %v", err)
	}

	// 应该跳过隐藏文件
	if len(collected) != 1 {
		t.Errorf("收集到 %d 个文件, want 1 (跳过隐藏文件)", len(collected))
	}
}

// ============================================================
// FilterFilesByType 测试
// ============================================================

func TestFilterFilesByType(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建不同类型的文件
	testFiles := map[string][]byte{
		"doc.txt":  []byte("text content"),
		"doc.docx": []byte("docx content"),
		"doc.pdf":  []byte("%PDF-1.4 content"),
		"img.jpg":  {0xFF, 0xD8, 0xFF},
	}

	var filePaths []string
	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, content, 0644)
		if err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
		filePaths = append(filePaths, path)
	}

	// 只过滤文本类型
	filtered, err := FilterFilesByType(filePaths, CategoryText)
	if err != nil {
		t.Fatalf("FilterFilesByType 失败: %v", err)
	}

	if len(filtered) != 1 {
		t.Errorf("过滤后 %d 个文件, want 1", len(filtered))
	}

	// 过滤多个类型
	filtered, err = FilterFilesByType(filePaths, CategoryText, CategoryPDF)
	if err != nil {
		t.Fatalf("FilterFilesByType 失败: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("过滤后 %d 个文件, want 2", len(filtered))
	}

	// 不传入类型，返回全部
	filtered, err = FilterFilesByType(filePaths)
	if err != nil {
		t.Fatalf("FilterFilesByType 失败: %v", err)
	}

	if len(filtered) != len(filePaths) {
		t.Errorf("过滤后 %d 个文件, want %d", len(filtered), len(filePaths))
	}
}