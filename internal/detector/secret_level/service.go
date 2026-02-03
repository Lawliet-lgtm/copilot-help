package secret_level

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	globalModel "linuxFileWatcher/internal/model" // 引用全局 model

	"linuxFileWatcher/internal/detector/secret_level/format"
	"linuxFileWatcher/internal/detector/secret_level/model"
	"linuxFileWatcher/internal/detector/secret_level/parser"
)

type service struct {
	config        Config
	officeScanner *parser.OfficeScanner
	ofdScanner    *parser.OFDScanner
	pdfScanner    *parser.PDFScanner
	textScanner   *parser.TextScanner
	binaryScanner *parser.BinaryScanner
	imageScanner  *parser.ImageScanner
}

func newService(cfg Config) *service {
	if cfg.OCRMaxFileSize <= 0 {
		cfg.OCRMaxFileSize = 20 * 1024 * 1024
	}
	return &service{
		config:        cfg,
		officeScanner: parser.NewOfficeScanner(),
		ofdScanner:    parser.NewOFDScanner(),
		pdfScanner:    parser.NewPDFScanner(),
		textScanner:   parser.NewTextScanner(),
		binaryScanner: parser.NewBinaryScanner(),
		imageScanner:  parser.NewImageScanner(),
	}
}

// DetectFile 实现
func (s *service) DetectFile(ctx context.Context, path string) (*globalModel.SubDetectResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()

	// 1. 识别格式
	header := make([]byte, 261)
	if _, err := f.ReadAt(header, 0); err != nil && err != io.EOF {
		return nil, err
	}
	fileType := format.IdentifyType(header)
	ext := strings.ToLower(filepath.Ext(path))

	// 2. 超时控制
	var scanCtx context.Context
	var cancel context.CancelFunc

	timeout := 2 * time.Second
	switch fileType {
	case model.TypeOffice, model.TypePDF:
		timeout = 5 * time.Second
	case model.TypeImage:
		timeout = 10 * time.Second
	}

	if _, ok := ctx.Deadline(); ok {
		scanCtx = ctx
		cancel = func() {}
	} else {
		scanCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	// 3. 执行检测
	var rawResult *model.ScanResult
	var scanErr error

	defer func() {
		if r := recover(); r != nil {
			scanErr = fmt.Errorf("panic within parser: %v", r)
		}
	}()

	switch fileType {
	case model.TypeOffice:
		if ext == ".ofd" {
			rawResult, scanErr = s.ofdScanner.Detect(scanCtx, f, size)
		} else {
			rawResult, scanErr = s.officeScanner.Detect(scanCtx, f, size)
		}
		if scanErr != nil {
			rawResult, scanErr = s.binaryScanner.Detect(scanCtx, f, size)
		}
	case model.TypePDF:
		rawResult, scanErr = s.pdfScanner.Detect(scanCtx, f, size)
		if scanErr != nil {
			rawResult, scanErr = s.binaryScanner.Detect(scanCtx, f, size)
		}
	case model.TypeText:
		rawResult, scanErr = s.textScanner.Detect(scanCtx, f, size)
	case model.TypeImage:
		if s.config.EnableOCR {
			if size > s.config.OCRMaxFileSize {
				return nil, nil // 图片太大跳过
			}
			rawResult, scanErr = s.imageScanner.Detect(scanCtx, f, size)
		} else {
			return nil, nil
		}
	default:
		rawResult, scanErr = s.binaryScanner.Detect(scanCtx, f, size)
	}

	if scanErr != nil {
		if strings.Contains(scanErr.Error(), "context deadline exceeded") {
			return nil, nil
		}
		return nil, scanErr
	}

	// 4. 结果转换：Internal ScanResult -> Global SubDetectResult
	if rawResult != nil && rawResult.IsSecret {
		// 转换密级枚举
		var globalLevel globalModel.SecretLevel
		switch rawResult.Level {
		case model.LevelTopSecret:
			globalLevel = globalModel.LevelTopSecret
		case model.LevelSecret:
			globalLevel = globalModel.LevelSecret
		case model.LevelConfidential:
			globalLevel = globalModel.LevelConfidential
		default:
			globalLevel = globalModel.LevelInternal
		}

		return &globalModel.SubDetectResult{
			IsSecret:    true,
			SecretLevel: globalLevel,
			RuleDesc:    "密级标志检测命中: " + rawResult.MatchedText,
			MatchedText: rawResult.MatchedText,
			AlertType:   2, // 假设 2 代表密级标志告警
		}, nil
	}

	return nil, nil
}