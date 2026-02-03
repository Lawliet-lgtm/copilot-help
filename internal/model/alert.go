package model

// ==========================================
// 告警记录 - 数据模型
// ==========================================

// AlertRecord 告警记录完整格式
type AlertRecord struct {
	// 告警id，20字节以内，只有数字、字母、下划线，具有唯一性
	ID string `json:"id" gorm:"type:varchar(20);primaryKey;uniqueIndex"`
	// 告警时间，格式为YYYY-MM-DD HH:mm:ss
	Time string `json:"time" gorm:"type:varchar(19);index"`
	// 命中策略id，不超过20位
	RuleID int64 `json:"rule_id" gorm:"type:bigint"`
	// 告警文字摘要，小于1024字节，未提取到置null
	RuleDesc string `json:"rule_desc" gorm:"type:varchar(1024)"`
	// 过滤类型，数值型
	FilterType int `json:"filter_type" gorm:"type:int"`
	// 文件摘要，字符串
	FileSummary string `json:"file_summary" gorm:"type:text"`
	// 告警类型映射
	AlertType AlertType `json:"alert_type" gorm:"type:int"`
	// 告警文件md5，最长64字节
	FileMD5 string `json:"file_md5" gorm:"type:varchar(64)"`
	// 告警文件路径，内容为空或未能提取到置null
	FilePath string `json:"file_path" gorm:"type:text"`
	// 文件名，最长128字节
	FileName string `json:"filename" gorm:"type:varchar(128)"`
	// 文件大小，4字节
	FileSize int `json:"filesize" gorm:"type:int"`
	// 告警关键词，最长512个词
	HighlightText string `json:"highlight_text" gorm:"type:varchar(512)"`
	// 关键词上下文，对上下文内容提取各20个字，全文最长512个字，当字段内容为空则填充null
	FileDesc string `json:"file_desc" gorm:"type:varchar(512)"`
	// 单位名称，最长256字节
	Company string `json:"company" gorm:"type:varchar(256)"`
	// 主机名称，最长256字节
	ComputerName string `json:"computer_name" gorm:"type:varchar(256);index"`
	// 组织机构id，最长256字节
	OrgID string `json:"org_id" gorm:"type:varchar(256);index"`
	// 组织机构全路径，最长512字节
	OrgPath string `json:"org_path" gorm:"type:varchar(512)"`
	// 责任人，最长256字节
	UserName string `json:"user_name" gorm:"type:varchar(256);index"`
	// 责任人ID，最长256字节
	UserID string `json:"user_id" gorm:"type:varchar(256);index"`
	// 文件MJ标志，最长128字节，枚举类型
	FileLevel int `json:"file_xxx_level" gorm:"type:int"`
	// 扩展字段：other
	ExtendFields string `json:"extend_fields" gorm:"type:text"`
}

// TableName 自定义表名
func (AlertRecord) TableName() string {
	return "alert_records"
}

// ==========================================
// 辅助构造函数
// ==========================================

// NewAlertRecord 创建新的告警记录
func NewAlertRecord(id string) *AlertRecord {
	return &AlertRecord{
		ID:            id,
		RuleDesc:      "",
		FilterType:    0,
		FileSummary:   "",
		AlertType:     AlertTypeOther,
		FileMD5:       "",
		FilePath:      "",
		FileName:      "",
		FileSize:      0,
		HighlightText: "",
		FileDesc:      "",
		Company:       "",
		ComputerName:  "",
		OrgID:         "",
		OrgPath:       "",
		UserName:      "",
		UserID:        "",
		FileLevel:     0,
		ExtendFields:  "",
	}
}
