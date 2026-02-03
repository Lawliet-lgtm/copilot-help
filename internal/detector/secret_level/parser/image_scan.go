package parser

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"strings"

	"github.com/otiai10/gosseract/v2"
	"linuxFileWatcher/internal/detector/secret_level/engine"
	"linuxFileWatcher/internal/detector/secret_level/model"

	// 注册图片格式解码器
	// 必须匿名导入以注册 init() 中的解码器
	_ "image/jpeg" 
	_ "image/gif"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ImageScanner 使用 Tesseract OCR 进行识别
type ImageScanner struct{}

func NewImageScanner() *ImageScanner {
	return &ImageScanner{}
}

func (s *ImageScanner) Detect(ctx context.Context, reader io.ReaderAt, size int64) (*model.ScanResult, error) {
	// 1. 读取图片数据
	// OCR 需要完整图片数据解码，不能像文本那样只读头。
	// 但为了防爆内存，限制最大图片大小，例如 20MB。
	const maxImgSize = 20 * 1024 * 1024
	if size > maxImgSize {
		// 图片太大，OCR 会极慢，跳过或降级
		return nil, nil // 返回 nil 表示不处理，worker 会决定是否 fallback
	}

	buf := make([]byte, size)
	if _, err := reader.ReadAt(buf, 0); err != nil && err != io.EOF {
		return nil, err
	}

	// 2. 解码图片对象
	// 这里会自动调用已注册的解码器（jpeg, png, bmp 等）
	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	// 3. 智能裁剪 (ROI - Region of Interest)
	// 密级标志 99% 在右上角、左上角或页眉居中。
	// 策略：只识别图片 Top 20% 的区域。
	// 这能让 OCR 速度提升 5 倍以上。
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	
	// 如果图片太小（比如图标），直接不扫
	if width < 100 || height < 100 {
		return &model.ScanResult{IsSecret: false}, nil
	}

	cropHeight := height / 5 // Top 20%
	if cropHeight < 200 {
		cropHeight = 200 // 至少保证有 200px 高
	}
	if cropHeight > height {
		cropHeight = height
	}

	// 裁剪出顶部区域
	// 这是一个关键优化，如果你的密级标志不在顶部，可能需要调整策略
	subImg := cropImage(img, image.Rect(0, 0, width, cropHeight))
	
	var cropBuf bytes.Buffer
	// 转为 PNG (无损) 喂给 OCR，虽然慢点但准
	if err := png.Encode(&cropBuf, subImg); err != nil {
		return nil, err
	}

	// 4. 调用 OCR
	client := gosseract.NewClient()
	defer client.Close()

	// 设置语言：中文简体 + 英文
	client.SetLanguage("chi_sim", "eng")
	
	// 传入图片数据
	if err := client.SetImageFromBytes(cropBuf.Bytes()); err != nil {
		return nil, err
	}

	// 检查 Context (支持超时控制)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 获取文本
	text, err := client.Text()
	if err != nil {
		return nil, err
	}

	// [DEBUG] 打印 OCR 识别出的原始内容
	// 去除换行符，方便单行显示，仅保留前100个字符
	cleanText := strings.ReplaceAll(text, "\n", " ")
	if len(cleanText) > 100 {
		cleanText = cleanText[:100] + "..."
	}
	
	if len(strings.TrimSpace(cleanText)) > 0 {
		fmt.Printf("DEBUG: OCR 原始内容: [%s]\n", cleanText)
	} else {
		fmt.Printf("DEBUG: OCR 原始内容为空 (可能图片太模糊或无文字)\n")
	}

	// 5. 匹配检测
	// 重点修改：调用专门针对 OCR 的宽松匹配逻辑 MatchOCRContent
	if hit, level, matchText := engine.MatchOCRContent(text); hit {
		return &model.ScanResult{
			IsSecret:    true,
			Level:       level,
			MatchedText: matchText + " (OCR)",
		}, nil
	}

	return &model.ScanResult{IsSecret: false}, nil
}

// cropImage 辅助函数：裁剪图片
func cropImage(img image.Image, cropRect image.Rectangle) image.Image {
	// 如果原图支持 SubImage (如 image.RGBA, image.YCbCr)
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	if si, ok := img.(subImager); ok {
		return si.SubImage(cropRect)
	}

	// 否则 fallback: 返回原图
	// 实际上 image.Decode 出来的通常都支持 SubImage
	return img 
}