// Package model
package model

// ==========================================
// 异常状态上报相关枚举 (对应图片定义)
// ==========================================

// SecurityEventType 异常类型 (数值型)
type SecurityEventType int

const (
	// TypeSecurityAbnormal 安全异常 (对应表中的大类)
	// 假设表中"安全异常"对应的数值是 1，如果文档有具体数值请替换
	TypeSecurityAbnormal SecurityEventType = 1
)

// SecurityEventSubType 异常子类 (字符型)
// 必须严格匹配表中文字
const (
	SubTypeSignature = "签名异常"
	SubTypeNetworkIP = "通信 IP 异常"
	SubTypeOther     = "其他"
)

// SecurityRiskLevel 告警级别 (数值型)
type SecurityRiskLevel int

const (
	RiskLevelNone     SecurityRiskLevel = 0 // 无风险
	RiskLevelGeneral  SecurityRiskLevel = 1 // 一般级
	RiskLevelNotice   SecurityRiskLevel = 2 // 关注级
	RiskLevelSevere   SecurityRiskLevel = 3 // 严重级
	RiskLevelCritical SecurityRiskLevel = 4 // 紧急级
)

// ==========================================
// 检测策略下发相关枚举
// ==========================================

// PolicyType 检测策略指令类型
const (
	PolicyType = "policy" // 固定值
)

// PolicyCommand 策略下发类型
const (
	PolicyCmdAdd   = "add"   // 增量式添加
	PolicyCmdDel   = "del"   // 增量式删除
	PolicyCmdReset = "reset" // 全量式
)

// DetectionModule 检测策略对应的模块名
const (
	ModuleKeywordDetect          = "keyword_detect"           // 关键词检测策略
	ModuleMD5Detect              = "md5_detect"               // 文件哈希检测策略
	ModuleSecretLevelDetect      = "secret_level_detect"      // 密级标志检测策略
	ModuleElectronicSecretDetect = "electronic_secret_detect" // 电子密级标志检测策略
	ModuleOfficialFormatDetect   = "official_format_detect"   // 公文版式检测策略
)

// FileType 文件类型枚举
const (
	FileTypeDocument = 1 // 文档
	FileTypeImage    = 2 // 图片
	FileTypeText     = 3 // 文本
	FileTypeArchive  = 4 // 压缩包
	FileTypeEmail    = 5 // 邮件
)

// HashType 哈希类型枚举
const (
	HashTypeMD5 = iota // MD5
	HashTypeSM3        // SM3
)

// ==========================================
// 系统操作指令相关枚举
// ==========================================

// CommandType 系统操作指令类型
const (
	CommandType = "command" // 固定值
)

// CommandParam 指令参数类型
const (
	CmdFileDetectAuditLog = "file_detect_audit_log" // 告警信息检测审计日志上报指令
	CmdUninstall          = "uninstall"             // 组件卸载指令
	CmdUpdate             = "update"                // 系统软件更新指令
	CmdStartm             = "startm"                // 启动系统模块指令
	CmdStartmInner        = "startm_inner"          // 启动内部系统模块指令
	CmdStopm              = "stopm"                 // 停止系统模块指令
	CmdStopmInner         = "stopm_inner"           // 停止内部系统模块指令
	CmdInnerPolicyUpdate  = "inner_policy_update"   // 系统内置策略更新指令
)

// ModuleName 系统模块名
const (
	ModuleFileDetect = "file_detect" // SM信息检测
)

// SubModuleName 系统子模块名
const (
	SubModuleKeywordDetect          = "keyword_detect"          // 关键词检测
	SubModuleMD5Detect              = "md5_detect"              // 文件完整性校验检测
	SubModuleSecurityClassification = "security_classification" // 电子文件m级标志检测
	SubModuleSecretLevelDetected    = "secret_level_detected"   // m级标志检测
	SubModuleOfficialFormatDetect   = "official_format_detect"  // 公文版式文件检测
)

// ==========================================
// 告警类型枚举
// ==========================================

// AlertType 告警类型映射
type AlertType int

const (
	AlertTypeRename             AlertType = 1  // 重命名
	AlertTypeCutPaste           AlertType = 2  // 剪切粘贴
	AlertTypeClearRecycleBin    AlertType = 3  // 清空回收站
	AlertTypeRecoverFromRecycle AlertType = 4  // 从回收站恢复
	AlertTypeMoveToRecycle      AlertType = 5  // 移入回收站
	AlertTypeDelete             AlertType = 6  // 彻底删除
	AlertTypeCopyPaste          AlertType = 7  // 复制粘贴
	AlertTypeBurn               AlertType = 8  // 刻录
	AlertTypeSaveOpen           AlertType = 9  // 保存/关闭
	AlertTypeOpen               AlertType = 10 // 打开
	AlertTypeLocalToUSB         AlertType = 11 // 本地拷贝到USB外设
	AlertTypeUSBToLocal         AlertType = 12 // USB外设拷贝到本地
	AlertTypeUSBToUSB           AlertType = 13 // USB外设拷贝到USB外设
	AlertTypeLocalCutToUSB      AlertType = 14 // 本地剪切到USB外设
	AlertTypeUSBCutToLocal      AlertType = 15 // USB外设剪切到本地
	AlertTypeUSBCutToUSB        AlertType = 16 // USB外设剪切到USB外设
	AlertTypePrint              AlertType = 17 // 打印
	AlertTypeOther              AlertType = 99 // 其他
)

// ==========================================
// 密级相关枚举
// ==========================================

// SecretLevelStr 定义密级类型
type SecretLevelStr string

const (
	SecretLevelNone         SecretLevelStr = "公开"
	SecretLevelInternal     SecretLevelStr = "内部"
	SecretLevelSecret       SecretLevelStr = "秘密"
	SecretLevelConfidential SecretLevelStr = "机密"
	SecretLevelTopSecret    SecretLevelStr = "绝密"
	SecretLevelUnknown      SecretLevelStr = "未知/检测失败"
)

// SecretLevelPriority 密级权重映射（用于多个标志共存时取最高等级）
var SecretLevelPriority = map[SecretLevelStr]int{
	SecretLevelNone:         0,
	SecretLevelInternal:     1,
	SecretLevelSecret:       2,
	SecretLevelConfidential: 3,
	SecretLevelTopSecret:    4,
	SecretLevelUnknown:      -1,
}

// SystemAuditLogType 系统审计日志类型
type SystemAuditLogType string

// SystemAuditOpType 系统审计操作类型
type SystemAuditOpType string

// 审计日志类型常量
const (
	// 安装卸载相关
	LogTypeInstallUninstall SystemAuditLogType = "安装卸载"
	// 策略变更相关
	LogTypePolicyChange SystemAuditLogType = "策略变更"
	// 本地操作相关
	LogTypeLocalOperation SystemAuditLogType = "本地操作"
	// 其他日志类型
	LogTypeOther SystemAuditLogType = "其他日志类型"
)

// 审计操作类型常量
const (
	// 安装操作
	OpTypeInstall SystemAuditOpType = "安装"
	// 卸载操作
	OpTypeUninstall SystemAuditOpType = "卸载"
	// 升级操作
	OpTypeUpgrade SystemAuditOpType = "升级"
	// 添加策略操作
	OpTypeAddPolicy SystemAuditOpType = "添加策略"
	// 删除策略操作
	OpTypeDeletePolicy SystemAuditOpType = "删除策略"
	// 重置策略操作
	OpTypeResetPolicy SystemAuditOpType = "重置策略"
	// 开机操作
	OpTypeStartup SystemAuditOpType = "开机"
	// 关机操作
	OpTypeShutdown SystemAuditOpType = "关机"
	// 登录操作
	OpTypeLogin SystemAuditOpType = "登录"
	// 注销操作
	OpTypeLogout SystemAuditOpType = "注销"
	// 文件变更操作
	OpTypeFileChange SystemAuditOpType = "文件变更"
	// 上线操作
	OpTypeOnline SystemAuditOpType = "上线"
	// 离线操作
	OpTypeOffline SystemAuditOpType = "离线"
	// 进程启动操作
	OpTypeProcessStart SystemAuditOpType = "进程启动"
	// 进程退出操作
	OpTypeProcessExit SystemAuditOpType = "进程退出"
)
