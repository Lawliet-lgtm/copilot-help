package parser

import (
	"bytes"
	"context"
	"io"
	"linuxFileWatcher/internal/detector/secret_level/model"
)

// 关键词定义
var (
	// UTF-8: 绝密, 机密, 秘密
	kwUtf8TopSecret    = []byte("绝密")
	kwUtf8Secret       = []byte("机密")
	kwUtf8Confidential = []byte("秘密")

	// GBK (Windows常用)
	kwGbkTopSecret    = []byte{0xBE, 0xF8, 0xC3, 0xDC}
	kwGbkSecret       = []byte{0xBB, 0xFA, 0xC3, 0xDC}
	kwGbkConfidential = []byte{0xC3, 0xDC, 0xC3, 0xDC}

	// UTF-16LE (Windows内存/OLE2)
	kwUtf16TopSecret    = []byte{0x5D, 0x7E, 0xC6, 0x5B}
	kwUtf16Secret       = []byte{0x3A, 0x67, 0xC6, 0x5B}
	kwUtf16Confidential = []byte{0xC6, 0x5B, 0xC6, 0x5B}

	// RTF 转义序列 (GBK based)
	// RTF 中文通常表示为 \'hh\'hh
	// 绝密 (GBK: BE F8 C3 DC) -> \'be\'f8\'c3\'dc
	// 注意 RTF 是大小写不敏感的，但通常生成的是小写。这里我们只匹配小写形式。
	// 为防万一，可以增加大写变体，但这里先只做最常见的。
	kwRtfTopSecret    = []byte(`\'be\'f8\'c3\'dc`)
	kwRtfSecret       = []byte(`\'bb\'fa\'c3\'dc`)
	kwRtfConfidential = []byte(`\'c3\'dc\'c3\'dc`)
)

// BinaryScanner 兜底扫描器
type BinaryScanner struct{}

func NewBinaryScanner() *BinaryScanner {
	return &BinaryScanner{}
}

func (s *BinaryScanner) Detect(ctx context.Context, reader io.ReaderAt, size int64) (*model.ScanResult, error) {
	// 策略: 只扫描前 1MB
	const scanLimit = 1024 * 1024
	readSize := size
	if readSize > scanLimit {
		readSize = scanLimit
	}

	buf := make([]byte, readSize)
	if _, err := reader.ReadAt(buf, 0); err != nil && err != io.EOF {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// 为了支持不区分大小写的 RTF 匹配，可以将 buffer 转小写后再匹配 RTF 关键字
	// 但这会消耗一次内存拷贝。考虑到 RTF 关键字本身很长，直接匹配字节序列性能更好。
	// 这里简单起见，直接字节匹配。

	// 1. RTF Check (最优先，因为特征最明显)
	// 简单转小写 check 可能太慢，我们只针对 buffer 做 Contains
	// 真实的 RTF 生成器大多用小写 hex
	if bytes.Contains(buf, kwRtfTopSecret) {
		return found(model.LevelTopSecret, "绝密(RTF)")
	}
	if bytes.Contains(buf, kwRtfSecret) {
		return found(model.LevelSecret, "机密(RTF)")
	}
	if bytes.Contains(buf, kwRtfConfidential) {
		return found(model.LevelConfidential, "秘密(RTF)")
	}

	// 2. UTF-8 Check
	if bytes.Contains(buf, kwUtf8TopSecret) {
		return found(model.LevelTopSecret, "绝密(UTF8)")
	}
	if bytes.Contains(buf, kwUtf8Secret) {
		return found(model.LevelSecret, "机密(UTF8)")
	}
	if bytes.Contains(buf, kwUtf8Confidential) {
		return found(model.LevelConfidential, "秘密(UTF8)")
	}

	// 3. GBK Check
	if bytes.Contains(buf, kwGbkTopSecret) {
		return found(model.LevelTopSecret, "绝密(GBK)")
	}
	if bytes.Contains(buf, kwGbkSecret) {
		return found(model.LevelSecret, "机密(GBK)")
	}
	if bytes.Contains(buf, kwGbkConfidential) {
		return found(model.LevelConfidential, "秘密(GBK)")
	}

	// 4. UTF-16LE Check
	if bytes.Contains(buf, kwUtf16TopSecret) {
		return found(model.LevelTopSecret, "绝密(UTF16)")
	}
	if bytes.Contains(buf, kwUtf16Secret) {
		return found(model.LevelSecret, "机密(UTF16)")
	}
	if bytes.Contains(buf, kwUtf16Confidential) {
		return found(model.LevelConfidential, "秘密(UTF16)")
	}

	return &model.ScanResult{IsSecret: false}, nil
}

func found(level model.SecretLevel, note string) (*model.ScanResult, error) {
	return &model.ScanResult{
		IsSecret:    true,
		Level:       level,
		MatchedText: note,
	}, nil
}