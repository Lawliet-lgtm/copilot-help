package rules

import (
	"strings"
)

// KeywordCategory 关键词分类
type KeywordCategory int

const (
	KwCategoryOrg         KeywordCategory = iota // 机关单位
	KwCategoryDocType                            // 公文文种
	KwCategoryAction                             // 公文动作词
	KwCategoryHeader                             // 版头关键词
	KwCategoryFooter                             // 版记关键词
	KwCategoryFormality                          // 公文正式用语
	KwCategoryProhibited                         // 非公文特征词
)

// KeywordSet 关键词集合
type KeywordSet struct {
	Category    KeywordCategory
	Name        string
	Keywords    []string
	Weight      float64 // 匹配权重
	Description string
}

// Contains 检查是否包含某个关键词
func (ks *KeywordSet) Contains(text string) bool {
	textLower := strings.ToLower(text)
	for _, kw := range ks.Keywords {
		if strings.Contains(textLower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// FindAll 查找所有匹配的关键词
func (ks *KeywordSet) FindAll(text string) []string {
	var found []string
	textLower := strings.ToLower(text)
	for _, kw := range ks.Keywords {
		if strings.Contains(textLower, strings.ToLower(kw)) {
			found = append(found, kw)
		}
	}
	return found
}

// CountMatches 统计匹配的关键词数量
func (ks *KeywordSet) CountMatches(text string) int {
	return len(ks.FindAll(text))
}

// ============================================================
// 机关单位关键词
// ============================================================

// OrgKeywords 机关单位关键词
var OrgKeywords = &KeywordSet{
	Category: KwCategoryOrg,
	Name:     "机关单位",
	Weight:   0.15,
	Keywords: []string{
		// 中央机关
		"中共中央", "国务院", "全国人大", "全国政协",
		"中央办公厅", "国务院办公厅",
		"中央委员会", "中央纪委", "中央组织部", "中央宣传部",

		// 国务院组成部门
		"外交部", "国防部", "发展改革委", "教育部", "科技部",
		"工业和信息化部", "国家民委", "公安部", "国家安全部",
		"民政部", "司法部", "财政部", "人力资源社会保障部",
		"自然资源部", "生态环境部", "住房城乡建设部", "交通运输部",
		"水利部", "农业农村部", "商务部", "文化和旅游部",
		"国家卫生健康委", "退役军人事务部", "应急管理部",
		"中国人民银行", "审计署",

		// 地方政府
		"省人民政府", "省政府", "市人民政府", "市政府",
		"县人民政府", "县政府", "区人民政府", "区政府",
		"自治区人民政府", "直辖市人民政府",
		"省委", "市委", "县委", "区委",
		"省人大", "市人大", "县人大",
		"省政协", "市政协", "县政协",

		// 通用机关后缀
		"办公室", "办公厅", "委员会", "管理局", "管理处",
		"工作委员会", "领导小组", "指挥部",
	},
	Description: "党政机关、政府部门名称关键词",
}

// ============================================================
// 公文文种关键词
// ============================================================

// DocTypeKeywords 公文文种关键词
var DocTypeKeywords = &KeywordSet{
	Category: KwCategoryDocType,
	Name:     "公文文种",
	Weight:   0.20,
	Keywords: []string{
		// 法定公文文种 (GB/T 9704-2012)
		"决议", "决定", "命令", "公报", "公告", "通告",
		"意见", "通知", "通报", "报告", "请示", "批复",
		"议案", "函", "纪要",

		// 常用公文类型
		"办法", "规定", "条例", "细则", "规则",
		"方案", "计划", "总结", "规划", "要点",
		"简报", "会议纪要", "工作要点",

		// 复合文种
		"实施意见", "实施方案", "实施办法", "实施细则",
		"管理办法", "管理规定", "暂行办法", "暂行规定",
		"工作方案", "工作计划", "工作总结", "工作报告",
	},
	Description: "公文文种名称",
}

// ============================================================
// 公文动作词关键词
// ============================================================

// ActionKeywords 公文动作词关键词
var ActionKeywords = &KeywordSet{
	Category: KwCategoryAction,
	Name:     "公文动作词",
	Weight:   0.10,
	Keywords: []string{
		// 发布类
		"印发", "发布", "公布", "颁布", "下发",
		"转发", "批转", "颁发",

		// 请示类
		"请示", "报请", "呈报", "申请", "请求",
		"恳请", "报批", "请批",

		// 批复类
		"批复", "批准", "同意", "核准", "批示",

		// 通知类
		"通知", "告知", "函告", "函复", "知照",

		// 执行类
		"执行", "遵照", "贯彻", "落实", "实施",
		"遵循", "按照", "依照", "根据",
	},
	Description: "公文中常用的动作动词",
}

// ============================================================
// 版头关键词
// ============================================================

// HeaderKeywords 版头关键词
var HeaderKeywords = &KeywordSet{
	Category: KwCategoryHeader,
	Name:     "版头关键词",
	Weight:   0.15,
	Keywords: []string{
		"发文字号", "签发人", "密级", "紧急程度",
		"特急", "加急", "平急", "限时",
		"绝密", "机密", "秘密",
		"文件", "文号",
	},
	Description: "公文版头区域的关键词",
}

// ============================================================
// 版记关键词
// ============================================================

// FooterKeywords 版记关键词
var FooterKeywords = &KeywordSet{
	Category: KwCategoryFooter,
	Name:     "版记关键词",
	Weight:   0.10,
	Keywords: []string{
		"抄送", "主送", "印发", "印制",
		"主题词", "抄报", "发送",
		"存档", "归档",
		"联系人", "联系电话", "联系方式",
		"共印", "份",
	},
	Description: "公文版记区域的关键词",
}

// ============================================================
// 公文正式用语
// ============================================================

// FormalityKeywords 公文正式用语关键词
var FormalityKeywords = &KeywordSet{
	Category: KwCategoryFormality,
	Name:     "公文正式用语",
	Weight:   0.08,
	Keywords: []string{
		// 开头用语
		"为了", "为进一步", "根据", "按照", "依据",
		"遵照", "鉴于", "为贯彻", "为落实",

		// 过渡用语
		"现就", "现将", "特此", "兹", "为此",
		"经研究", "经审核", "经批准",

		// 结尾用语
		"特此通知", "特此通告", "特此公告",
		"请遵照执行", "请认真贯彻执行", "请结合实际",
		"望遵照执行", "请予以支持", "请予以配合",
		"此复", "此函", "特此函复", "专此函达",
		"以上报告", "妥否", "请批示", "请审批",

		// 敬语
		"敬请", "恳请", "请予", "盼复", "函复",
	},
	Description: "公文中的正式书面用语",
}

// ============================================================
// 非公文特征词 (用于反向判断)
// ============================================================

// ProhibitedKeywords 非公文特征词
var ProhibitedKeywords = &KeywordSet{
	Category: KwCategoryProhibited,
	Name:     "非公文特征词",
	Weight:   -0.20, // 负权重，出现时降低公文可能性
	Keywords: []string{
		// 商业文档特征
		"价格", "报价", "合同", "订单", "发票",
		"采购", "销售", "客户", "供应商",
		"折扣", "优惠", "促销",

		// 学术文档特征
		"摘要", "关键词", "参考文献", "引言",
		"abstract", "keywords", "references",
		"致谢", "论文", "学位",

		// 新闻特征
		"记者", "编辑", "来源", "转载",
		"本报讯", "据悉", "消息",

		// 广告特征
		"立即购买", "点击", "咨询热线",
		"免费", "限时", "抢购",
	},
	Description: "通常不出现在公文中的词汇",
}

// ============================================================
// 关键词集合
// ============================================================

// AllKeywordSets 所有关键词集合
var AllKeywordSets = []*KeywordSet{
	OrgKeywords,
	DocTypeKeywords,
	ActionKeywords,
	HeaderKeywords,
	FooterKeywords,
	FormalityKeywords,
	ProhibitedKeywords,
}

// PositiveKeywordSets 正向关键词集合（公文特征）
var PositiveKeywordSets = []*KeywordSet{
	OrgKeywords,
	DocTypeKeywords,
	ActionKeywords,
	HeaderKeywords,
	FooterKeywords,
	FormalityKeywords,
}

// ============================================================
// 辅助函数
// ============================================================

// KeywordMatchResult 关键词匹配结果
type KeywordMatchResult struct {
	Set      *KeywordSet // 关键词集合
	Matched  []string    // 匹配到的关键词
	Count    int         // 匹配数量
	Score    float64     // 得分（数量 * 权重）
}

// AnalyzeKeywords 分析文本中的关键词
func AnalyzeKeywords(text string) []*KeywordMatchResult {
	var results []*KeywordMatchResult

	for _, ks := range AllKeywordSets {
		matched := ks.FindAll(text)
		if len(matched) > 0 {
			results = append(results, &KeywordMatchResult{
				Set:     ks,
				Matched: matched,
				Count:   len(matched),
				Score:   float64(len(matched)) * ks.Weight,
			})
		}
	}

	return results
}

// CalculateKeywordScore 计算关键词总得分
func CalculateKeywordScore(text string) float64 {
	results := AnalyzeKeywords(text)

	totalScore := 0.0
	for _, r := range results {
		totalScore += r.Score
	}

	// 归一化到 0-1 范围
	// 假设最高可能得分为 2.0
	if totalScore > 1.0 {
		totalScore = 1.0
	}
	if totalScore < 0 {
		totalScore = 0
	}

	return totalScore
}

// ContainsOrgName 检查是否包含机关名称
func ContainsOrgName(text string) bool {
	return OrgKeywords.Contains(text)
}

// FindOrgNames 查找所有机关名称
func FindOrgNames(text string) []string {
	return OrgKeywords.FindAll(text)
}

// ContainsDocType 检查是否包含公文文种
func ContainsDocType(text string) bool {
	return DocTypeKeywords.Contains(text)
}

// FindDocTypes 查找所有公文文种
func FindDocTypes(text string) []string {
	return DocTypeKeywords.FindAll(text)
}

// HasProhibitedContent 检查是否包含非公文特征
func HasProhibitedContent(text string) bool {
	return ProhibitedKeywords.Contains(text)
}

// GetProhibitedMatches 获取匹配的非公文特征词
func GetProhibitedMatches(text string) []string {
	return ProhibitedKeywords.FindAll(text)
}