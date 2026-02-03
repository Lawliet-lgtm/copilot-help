package model

import (
	"time"
)

// ==========================================
// 异常状态上报
// ==========================================

// SecurityStatusReport 顶层上报结构 (对应数据库表: security_reports)
// 包含多条 SuspectedEvent
type SecurityStatusReport struct {
	// 数据库主键，JSON 序列化时忽略
	ID uint `gorm:"primaryKey;autoIncrement" json:"-"`

	// 软件版本: 字符串, 最长 32
	// gorm: 限制数据库字段长度
	SoftVersion string `gorm:"type:varchar(32);not null" json:"soft_version"`

	// 业务状态采集时间: 时间类型, 最长 128
	// 这里为了完全匹配 JSON 协议保持 string，数据库存为 varchar
	// 建立索引方便按时间查询
	Time string `gorm:"type:varchar(128);index" json:"time"`

	// 一对多关联: ReportID 是 SuspectedEvent 表的外键
	// gorm: 指定外键关系
	Suspected []SuspectedEvent `gorm:"foreignKey:ReportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"suspected"`

	// 记录入库时间 (本地管理用，不在协议中)
	CreatedAt time.Time `json:"-"`
}

// SuspectedEvent 单个异常事件 (对应数据库表: suspected_events)
type SuspectedEvent struct {
	// 数据库主键，JSON 序列化时忽略
	ID uint `gorm:"primaryKey;autoIncrement" json:"-"`

	// 外键: 关联归属的 Report
	ReportID uint `gorm:"index" json:"-"`

	// 异常类型: 数值型
	EventType SecurityEventType `gorm:"type:int;not null" json:"event_type"`

	// 异常子类: 字符型
	EventSubType string `gorm:"type:varchar(64)" json:"event_sub_type"`

	// 异常产生时间: 时间类型
	Time string `gorm:"type:varchar(64)" json:"time"`

	// 告警级别: 数值型 (0-4)
	Risk SecurityRiskLevel `gorm:"type:smallint" json:"risk"`

	// 异常事件描述: 字符串, 最长 128
	Msg string `gorm:"type:varchar(128)" json:"msg"`
}

// TableName 自定义表名 (可选，符合 SQLite 命名习惯)
func (SecurityStatusReport) TableName() string {
	return "security_reports"
}

func (SuspectedEvent) TableName() string {
	return "suspected_events"
}

// ==========================================
// 辅助构造函数
// ==========================================

func NewSecurityStatusReport(version string) *SecurityStatusReport {
	return &SecurityStatusReport{
		SoftVersion: version,
		Time:        time.Now().Format("2006-01-02 15:04:05"),
		// 初始化切片，避免 json 输出 null
		Suspected: make([]SuspectedEvent, 0),
	}
}

// AddSignatureAlert 添加一条“签名异常”
// 注意：这里我们不再需要手动维护 ReportID，GORM 在保存主表时会自动处理子表的外键
func (r *SecurityStatusReport) AddSignatureAlert(filePath string, msg string) {
	event := SuspectedEvent{
		EventType:    TypeSecurityAbnormal,
		EventSubType: SubTypeSignature,
		Time:         time.Now().Format("2006-01-02 15:04:05"),
		Risk:         RiskLevelCritical,
		Msg:          limitString(msg, 128),
	}
	r.Suspected = append(r.Suspected, event)
}

// AddNetworkAlert 添加一条“通信 IP 异常”
func (r *SecurityStatusReport) AddNetworkAlert(remoteIP string, port uint16, msg string) {
	fullMsg := msg
	if fullMsg == "" {
		fullMsg = "Detected unauthorized communication with " + remoteIP
	}

	event := SuspectedEvent{
		EventType:    TypeSecurityAbnormal,
		EventSubType: SubTypeNetworkIP,
		Time:         time.Now().Format("2006-01-02 15:04:05"),
		Risk:         RiskLevelSevere,
		Msg:          limitString(fullMsg, 128),
	}
	r.Suspected = append(r.Suspected, event)
}

func limitString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen])
	}
	return s
}
