package processor

// ============================================================
// DOCX 版式特征数据结构
// 基于 GB/T 9704-2012 公文格式标准
// ============================================================

// DocxStyleFeatures DOCX版式特征
type DocxStyleFeatures struct {
	// 颜色特征
	ColorFeatures ColorFeatures `json:"color_features"`

	// 字体特征
	FontFeatures FontFeatures `json:"font_features"`

	// 页面设置特征
	PageFeatures PageFeatures `json:"page_features"`

	// 段落特征
	ParagraphFeatures ParagraphFeatures `json:"paragraph_features"`

	// 嵌入对象特征
	EmbeddedFeatures EmbeddedFeatures `json:"embedded_features"`

	// 综合评估
	StyleScore      float64  `json:"style_score"`       // 版式得分 (0-1)
	IsOfficialStyle bool     `json:"is_official_style"` // 是否符合公文版式
	StyleReasons    []string `json:"style_reasons"`     // 判断理由
}

// ============================================================
// 颜色特征
// ============================================================

// ColorFeatures 颜色特征
type ColorFeatures struct {
	HasRedText       bool     `json:"has_red_text"`        // 是否有红色文本
	RedTextCount     int      `json:"red_text_count"`      // 红色文本数量
	RedTextPositions []string `json:"red_text_positions"`  // 红色文本位置描述
	RedTextSamples   []string `json:"red_text_samples"`    // 红色文本示例
	HasRedHeader     bool     `json:"has_red_header"`      // 是否有红头（顶部红色文本）
	HasRedLine       bool     `json:"has_red_line"`        // 是否有红线（分隔线）
	DominantColors   []string `json:"dominant_colors"`     // 主要使用的颜色
}

// 红色相关的颜色值（十六进制，不区分大小写）
var RedColorValues = []string{
	"FF0000", // 纯红
	"ff0000",
	"C00000", // 深红
	"c00000",
	"CC0000",
	"cc0000",
	"990000",
	"800000",
	"E60000",
	"e60000",
	"D40000",
	"d40000",
	"B20000",
	"b20000",
}

// IsRedColor 检查是否为红色
func IsRedColor(color string) bool {
	// 移除可能的 # 前缀
	color = trimColorPrefix(color)

	// 检查是否在预定义红色列表中
	for _, red := range RedColorValues {
		if equalsIgnoreCase(color, red) {
			return true
		}
	}

	// 检查RGB分量（R值高，G和B值低）
	if len(color) == 6 {
		r := hexToInt(color[0:2])
		g := hexToInt(color[2:4])
		b := hexToInt(color[4:6])

		// 红色判断条件：R分量高，G和B分量低
		if r >= 180 && g <= 80 && b <= 80 {
			return true
		}
	}

	return false
}

// ============================================================
// 字体特征
// ============================================================

// FontFeatures 字体特征
type FontFeatures struct {
	UsedFonts        []FontInfo `json:"used_fonts"`          // 使用的字体列表
	HasOfficialFonts bool       `json:"has_official_fonts"`  // 是否使用公文字体
	FontDistribution FontDist   `json:"font_distribution"`   // 字体分布
	TitleFontMatch   bool       `json:"title_font_match"`    // 标题字体是否符合
	BodyFontMatch    bool       `json:"body_font_match"`     // 正文字体是否符合
}

// FontInfo 字体信息
type FontInfo struct {
	Name     string  `json:"name"`      // 字体名称
	Size     float64 `json:"size"`      // 字号（磅）
	SizeDesc string  `json:"size_desc"` // 字号描述（如"二号"、"三号"）
	Count    int     `json:"count"`     // 使用次数
	IsBold   bool    `json:"is_bold"`   // 是否加粗
	Color    string  `json:"color"`     // 颜色
}

// FontDist 字体分布
type FontDist struct {
	SongCount    int `json:"song_count"`     // 宋体使用次数
	FangsongCount int `json:"fangsong_count"` // 仿宋使用次数
	HeiCount     int `json:"hei_count"`      // 黑体使用次数
	KaiCount     int `json:"kai_count"`      // 楷体使用次数
	OtherCount   int `json:"other_count"`    // 其他字体使用次数
}

// 公文标准字体（GB/T 9704-2012）
var OfficialFonts = []string{
	// 标题字体
	"方正小标宋简体", "方正小标宋", "小标宋", "FZXiaoBiaoSong",
	// 正文字体
	"仿宋", "仿宋_GB2312", "FangSong", "FangSong_GB2312",
	// 其他公文字体
	"黑体", "SimHei", "Heiti",
	"宋体", "SimSun", "Songti",
	"楷体", "KaiTi", "Kaiti",
}

// 公文标准字号（GB/T 9704-2012）
// 以磅(pt)为单位
var OfficialFontSizes = map[string]float64{
	"一号":  26.0,
	"小一":  24.0,
	"二号":  22.0,  // 标题
	"小二":  18.0,
	"三号":  16.0,  // 正文
	"小三":  15.0,
	"四号":  14.0,
	"小四":  12.0,
	"五号":  10.5,
	"小五":  9.0,
}

// IsOfficialFont 检查是否为公文字体
func IsOfficialFont(fontName string) bool {
	for _, official := range OfficialFonts {
		if containsIgnoreCase(fontName, official) {
			return true
		}
	}
	return false
}

// GetFontSizeDesc 获取字号描述
func GetFontSizeDesc(sizeInPt float64) string {
	// 允许±0.5pt的误差
	for desc, size := range OfficialFontSizes {
		if sizeInPt >= size-0.5 && sizeInPt <= size+0.5 {
			return desc
		}
	}
	return ""
}

// IsTitleFontSize 检查是否为标题字号（二号，22pt）
func IsTitleFontSize(sizeInPt float64) bool {
	return sizeInPt >= 21.5 && sizeInPt <= 22.5
}

// IsBodyFontSize 检查是否为正文字号（三号，16pt）
func IsBodyFontSize(sizeInPt float64) bool {
	return sizeInPt >= 15.5 && sizeInPt <= 16.5
}

// ============================================================
// 页面设置特征
// ============================================================

// PageFeatures 页面设置特征
type PageFeatures struct {
	// 纸张大小
	PageWidth    float64 `json:"page_width"`    // 页面宽度 (mm)
	PageHeight   float64 `json:"page_height"`   // 页面高度 (mm)
	IsA4         bool    `json:"is_a4"`         // 是否为A4纸
	PaperSizeDesc string `json:"paper_size_desc"` // 纸张大小描述

	// 页边距
	MarginTop    float64 `json:"margin_top"`    // 上边距 (mm)
	MarginBottom float64 `json:"margin_bottom"` // 下边距 (mm)
	MarginLeft   float64 `json:"margin_left"`   // 左边距 (mm)
	MarginRight  float64 `json:"margin_right"`  // 右边距 (mm)
	MarginMatch  bool    `json:"margin_match"`  // 页边距是否符合标准

	// 页眉页脚
	HasHeader     bool    `json:"has_header"`      // 是否有页眉
	HasFooter     bool    `json:"has_footer"`      // 是否有页脚
	HeaderDistance float64 `json:"header_distance"` // 页眉距离 (mm)
	FooterDistance float64 `json:"footer_distance"` // 页脚距离 (mm)
}

// GB/T 9704-2012 页面标准
const (
	// A4纸张尺寸 (mm)
	A4Width  = 210.0
	A4Height = 297.0

	// 页边距标准 (mm)
	StandardMarginTop    = 37.0
	StandardMarginBottom = 35.0
	StandardMarginLeft   = 28.0
	StandardMarginRight  = 26.0

	// 页边距允许误差 (mm)
	MarginTolerance = 2.0

	// Twips 转换常量 (1 twip = 1/20 point = 1/1440 inch)
	TwipsPerMM = 56.692913 // 1440 / 25.4
	TwipsPerPt = 20.0
)

// IsA4Paper 检查是否为A4纸张
func IsA4Paper(widthMM, heightMM float64) bool {
	// 允许±2mm的误差
	widthOK := widthMM >= A4Width-2 && widthMM <= A4Width+2
	heightOK := heightMM >= A4Height-2 && heightMM <= A4Height+2
	return widthOK && heightOK
}

// CheckMargins 检查页边距是否符合标准
func CheckMargins(top, bottom, left, right float64) bool {
	topOK := top >= StandardMarginTop-MarginTolerance && top <= StandardMarginTop+MarginTolerance
	bottomOK := bottom >= StandardMarginBottom-MarginTolerance && bottom <= StandardMarginBottom+MarginTolerance
	leftOK := left >= StandardMarginLeft-MarginTolerance && left <= StandardMarginLeft+MarginTolerance
	rightOK := right >= StandardMarginRight-MarginTolerance && right <= StandardMarginRight+MarginTolerance

	return topOK && bottomOK && leftOK && rightOK
}

// TwipsToMM 将Twips转换为毫米
func TwipsToMM(twips float64) float64 {
	return twips / TwipsPerMM
}

// TwipsToPt 将Twips转换为磅
func TwipsToPt(twips float64) float64 {
	return twips / TwipsPerPt
}

// ============================================================
// 段落特征
// ============================================================

// ParagraphFeatures 段落特征
type ParagraphFeatures struct {
	TotalParagraphs  int     `json:"total_paragraphs"`   // 总段落数
	CenteredCount    int     `json:"centered_count"`     // 居中段落数
	HasCenteredTitle bool    `json:"has_centered_title"` // 是否有居中标题
	FirstLineIndent  float64 `json:"first_line_indent"`  // 首行缩进 (字符)
	LineSpacing      float64 `json:"line_spacing"`       // 行距 (磅)
	LineSpacingMatch bool    `json:"line_spacing_match"` // 行距是否符合标准
}

// 标准行距 (磅)
const StandardLineSpacing = 28.0

// IsStandardLineSpacing 检查行距是否符合标准
func IsStandardLineSpacing(spacingPt float64) bool {
	// 允许±2pt的误差
	return spacingPt >= StandardLineSpacing-2 && spacingPt <= StandardLineSpacing+2
}

// ============================================================
// 嵌入对象特征
// ============================================================

// EmbeddedFeatures 嵌入对象特征
type EmbeddedFeatures struct {
	HasImages     bool        `json:"has_images"`      // 是否有嵌入图片
	ImageCount    int         `json:"image_count"`     // 图片数量
	Images        []ImageMeta `json:"images"`          // 图片信息
	HasSealImage  bool        `json:"has_seal_image"`  // 是否可能有印章图片
	SealImageHint string      `json:"seal_image_hint"` // 印章图片提示
}

// ImageMeta 图片元数据
type ImageMeta struct {
	Name     string  `json:"name"`      // 文件名
	Type     string  `json:"type"`      // 类型 (png/jpeg/etc)
	Width    float64 `json:"width"`     // 宽度 (像素)
	Height   float64 `json:"height"`    // 高度 (像素)
	Position string  `json:"position"`  // 位置描述
	IsRedish bool    `json:"is_redish"` // 是否可能为红色图像
}

// ============================================================
// 辅助函数
// ============================================================

// trimColorPrefix 移除颜色值的前缀
func trimColorPrefix(color string) string {
	if len(color) > 0 && color[0] == '#' {
		return color[1:]
	}
	return color
}

// equalsIgnoreCase 不区分大小写比较
func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// containsIgnoreCase 不区分大小写包含检查
func containsIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)

	return containsString(sLower, substrLower)
}

// toLower 转小写（简易版）
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

// containsString 检查字符串包含
func containsString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// hexToInt 十六进制转整数
func hexToInt(hex string) int {
	result := 0
	for i := 0; i < len(hex); i++ {
		c := hex[i]
		var val int
		if c >= '0' && c <= '9' {
			val = int(c - '0')
		} else if c >= 'a' && c <= 'f' {
			val = int(c - 'a' + 10)
		} else if c >= 'A' && c <= 'F' {
			val = int(c - 'A' + 10)
		}
		result = result*16 + val
	}
	return result
}

// NewDocxStyleFeatures 创建新的版式特征对象
func NewDocxStyleFeatures() *DocxStyleFeatures {
	return &DocxStyleFeatures{
		ColorFeatures:     ColorFeatures{},
		FontFeatures:      FontFeatures{UsedFonts: make([]FontInfo, 0)},
		PageFeatures:      PageFeatures{},
		ParagraphFeatures: ParagraphFeatures{},
		EmbeddedFeatures:  EmbeddedFeatures{Images: make([]ImageMeta, 0)},
		StyleReasons:      make([]string, 0),
	}
}