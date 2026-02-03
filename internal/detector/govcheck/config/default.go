package config

import "runtime"

// Default 返回默认配置
func Default() *Config {
	return &Config{
		Version: "1.0",

		Detection: DetectionConfig{
			Threshold:   0.6,
			TextWeight:  0.55,
			StyleWeight: 0.45,
			MaxFileSize: 100 * 1024 * 1024, // 100MB
			Workers:     runtime.NumCPU(),
			Recursive:   true,
			ExcludeExtensions: []string{
				".exe", ".dll", ".so", ".dylib",
				".zip", ".rar", ".7z", ".tar", ".gz",
				".mp3", ".mp4", ".avi", ".mov", ".wmv",
				".db", ".sqlite", ".mdb",
			},
			ExcludeDirectories: []string{
				".git", ".svn", ".hg",
				"node_modules", "vendor",
				"__pycache__", ".cache",
			},
		},

		OCR: OCRConfig{
			Enabled:       true,
			TesseractPath: "", // 从 PATH 查找
			Language:      "chi_sim+eng",
			Timeout:       30,
		},

		Output: OutputConfig{
			Format:   "text",
			Verbose:  false,
			Color:    true,
			LogLevel: "info",
		},

		Weights: DefaultWeights(),
	}
}

// DefaultWeights 返回默认特征权重
func DefaultWeights() WeightsConfig {
	return WeightsConfig{
		Text: TextWeightsConfig{
			// 版头特征
			CopyNumber:   0.04,
			DocNumber:    0.18,
			SecretLevel:  0.06,
			UrgencyLevel: 0.05,
			Issuer:       0.08,

			// 主体特征
			Title:      0.15,
			TitleType:  0.05,
			MainSend:   0.08,
			Attachment: 0.04,

			// 版记特征
			IssueDate: 0.12,
			CopyTo:    0.05,
			PrintInfo: 0.05,

			// 机关特征
			OrgName: 0.10,

			// 关键词
			DocType:    0.05,
			ActionWord: 0.04,
			FormalWord: 0.04,
			HeaderWord: 0.03,
			FooterWord: 0.03,
			Prohibited: 0.20,
		},

		Style: StyleWeightsConfig{
			RedText:       0.12,
			RedHeader:     0.18,
			OfficialFonts: 0.10,
			TitleFont:     0.08,
			BodyFont:      0.07,
			A4Paper:       0.10,
			Margins:       0.10,
			CenteredTitle: 0.07,
			LineSpacing:   0.05,
			SealImage:     0.13,
		},
	}
}

// HighSensitivity 返回高灵敏度配置（更容易判定为公文）
func HighSensitivity() *Config {
	config := Default()
	config.Detection.Threshold = 0.45
	return config
}

// LowSensitivity 返回低灵敏度配置（更严格判定）
func LowSensitivity() *Config {
	config := Default()
	config.Detection.Threshold = 0.75
	return config
}

// ImageOptimized 返回针对图片优化的配置
func ImageOptimized() *Config {
	config := Default()

	// 图片类型更依赖版式特征
	config.Detection.TextWeight = 0.40
	config.Detection.StyleWeight = 0.60

	// 提高版式特征权重
	config.Weights.Style.RedHeader = 0.22
	config.Weights.Style.SealImage = 0.18
	config.Weights.Style.A4Paper = 0.12

	// OCR 超时增加
	config.OCR.Timeout = 60

	return config
}

// StrictMode 返回严格模式配置
func StrictMode() *Config {
	config := Default()

	// 更高阈值
	config.Detection.Threshold = 0.70

	// 提高核心特征权重
	config.Weights.Text.DocNumber = 0.22
	config.Weights.Text.Title = 0.18
	config.Weights.Text.IssueDate = 0.15

	return config
}