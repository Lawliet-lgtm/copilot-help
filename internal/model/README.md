# Model 模块开发文档

## 1. 模块概述

Model模块是Linux文件监控系统的核心数据模型层，负责定义系统中所有数据结构和枚举类型，为上层业务逻辑和下层存储层提供数据交换标准。该模块采用GORM框架实现与SQLite数据库的映射，确保数据的一致性和完整性。

### 1.1 设计目标

- **统一数据格式**：为系统各模块提供一致的数据结构定义
- **易于扩展**：支持新功能的数据模型快速添加
- **数据库友好**：与GORM框架无缝集成，简化数据库操作
- **接口标准化**：规范系统内部和外部接口的数据格式

### 1.2 核心功能

- 定义系统所有枚举类型和常量
- 实现告警信息审计日志上报数据模型
- 实现检测策略下发数据模型
- 实现系统操作指令数据模型
- 实现指令执行结果上报数据模型
- 实现安全状态报告数据模型
- 实现策略执行结果响应上报数据模型
- 实现注册/注销/认证数据模型
- 实现终端日志审计数据模型

## 2. 目录结构

```
model/
├── alert.go              // 告警记录数据模型
├── audit_log.go          // 告警信息审计日志上报数据模型
├── command.go            // 系统操作指令数据模型
├── command_result.go     // 指令执行结果上报数据模型
├── enums.go              // 枚举类型和常量定义
├── policy.go             // 检测策略下发数据模型
├── register.go           // 注册/注销/认证数据模型
├── security_report.go    // 安全状态报告数据模型
├── strategy_report.go    // 策略执行结果响应上报数据模型
└── system_audit.go       // 终端日志审计数据模型
```

## 3. 核心数据结构

### 3.1 枚举类型定义

#### 3.1.1 安全事件相关枚举

```go
// SecurityEventType 异常类型
type SecurityEventType int

const (
    TypeSecurityAbnormal SecurityEventType = 1 // 安全异常
)

// SecurityEventSubType 异常子类
const (
    SubTypeSignature = "签名异常"
    SubTypeNetworkIP = "通信 IP 异常"
    SubTypeOther     = "其他"
)

// SecurityRiskLevel 告警级别
type SecurityRiskLevel int

const (
    RiskLevelNone     SecurityRiskLevel = 0 // 无风险
    RiskLevelGeneral  SecurityRiskLevel = 1 // 一般级
    RiskLevelNotice   SecurityRiskLevel = 2 // 关注级
    RiskLevelSevere   SecurityRiskLevel = 3 // 严重级
    RiskLevelCritical SecurityRiskLevel = 4 // 紧急级
)
```

#### 3.1.2 检测策略相关枚举

```go
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
```

#### 3.1.3 系统操作指令相关枚举

```go
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
```

#### 3.1.4 告警类型枚举

```go
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
```

### 3.2 核心数据结构

#### 3.2.1 告警信息审计日志上报

```go
// AlertLogReport 上报主结构
type AlertLogReport struct {
    ID        uint           `gorm:"primaryKey;autoIncrement" json:"-"`
    CmdID     string         `gorm:"type:varchar(64);index" json:"cmd_id"`
    Time      string         `gorm:"type:varchar(64);index" json:"time"`
    AuditLogs []AlertLogItem `gorm:"foreignKey:ReportID" json:"audit_logs"`
    CreatedAt time.Time      `json:"-"`
}

// AlertLogItem 单条审计日志详情
type AlertLogItem struct {
    ID       uint   `gorm:"primaryKey;autoIncrement" json:"-"`
    ReportID uint   `gorm:"index" json:"-"`
    FileName string `gorm:"type:varchar(255)" json:"file_name"`
    FilePath string `gorm:"type:text" json:"file_path"`
    FileMD5  string `gorm:"type:varchar(64)" json:"file_md5"`
    Time     string `gorm:"type:varchar(64)" json:"time"`
}
```

#### 3.2.2 检测策略下发

```go
// PolicyRequest 检测策略下发请求结构体
type PolicyRequest struct {
    Type    string      `json:"type" binding:"required,eq=policy"`
    Module  string      `json:"module" binding:"required,oneof=keyword_detect md5_detect"`
    Version string      `json:"version" binding:"required,max=64"`
    Cmd     string      `json:"cmd" binding:"required,oneof=add del reset"`
    Num     int         `json:"num" binding:"required,min=0"`
    Config  interface{} `json:"config"`
}

// KeywordDetectRule 关键词检测策略规则
type KeywordDetectRule struct {
    RuleID         int64                 `json:"rule_id" binding:"required"`
    RuleContent    string                `json:"rule_content" binding:"required"`
    RuleDesc       string                `json:"rule_desc,omitempty" binding:"max=128"`
    MinMatchCount  int                   `json:"min_match_count,omitempty"`
    FilterFileType []int                 `json:"filter_file_type,omitempty"`
    FilterFileSize *FilterFileSize       `json:"filter_file_size,omitempty"`
    ExtendedFields map[string]interface{} `json:"extended_fields,omitempty"`
}

// HashDetectRule 文件哈希检测策略规则
type HashDetectRule struct {
    RuleID         int64                 `json:"rule_id" binding:"required"`
    RuleType       int                   `json:"rule_type" binding:"required,oneof=0 1"`
    RuleContent    string                `json:"rule_content" binding:"required,max=128"`
    RuleDesc       string                `json:"rule_desc,omitempty" binding:"max=128"`
    ExtendedFields map[string]interface{} `json:"extended_fields,omitempty"`
}
```

#### 3.2.3 系统操作指令

```go
// AuditLogCommandRequest 告警信息检测审计日志上报指令请求
type AuditLogCommandRequest struct {
    Type  string                    `json:"type" binding:"required,eq=command"`
    Cmd   string                    `json:"cmd" binding:"required,eq=file_detect_audit_log"`
    Param FileDetectAuditLogParam   `json:"param"`
    CmdID string                    `json:"cmd_id" binding:"required,max=128"`
}

// UninstallCommandRequest 组件卸载指令
type UninstallCommandRequest struct {
    Type  string `json:"type" binding:"required,eq=command"`
    Cmd   string `json:"cmd" binding:"required,eq=uninstall"`
    CmdID string `json:"cmd_id" binding:"required,max=128"`
}

// StartStopCommandRequest 启停系统模块指令请求
type StartStopCommandRequest struct {
    Type      string   `json:"type" binding:"required,eq=command"`
    Cmd       string   `json:"cmd" binding:"required,oneof=startm startm_inner stopm stopm_inner"`
    Module    string   `json:"module" binding:"required,max=128"`
    Submodule []string `json:"submodule,omitempty"`
    CmdID     string   `json:"cmd_id" binding:"required,max=128"`
}
```

#### 3.2.4 指令执行结果上报

```go
// CommandResultReport 指令执行结果上报主结构
type CommandResultReport struct {
    ID        uint      `gorm:"primaryKey;autoIncrement" json:"-"`
    Time      string    `gorm:"type:varchar(128);index" json:"time"`
    Type      string    `gorm:"type:varchar(128);not null" json:"type"`
    Cmd       string    `gorm:"type:varchar(128);not null" json:"cmd"`
    CmdID     string    `gorm:"type:varchar(128);index" json:"cmd_id"`
    Result    int       `gorm:"type:int;not null" json:"result"`
    Message   string    `gorm:"type:varchar(128)" json:"message"`
    Detail    []string  `gorm:"type:json" json:"detail,omitempty"`
    CreatedAt time.Time `json:"-"`
}
```

#### 3.2.5 安全状态报告

```go
// SecurityStatusReport 顶层上报结构
type SecurityStatusReport struct {
    ID          uint             `gorm:"primaryKey;autoIncrement" json:"-"`
    SoftVersion string           `gorm:"type:varchar(32);not null" json:"soft_version"`
    Time        string           `gorm:"type:varchar(128);index" json:"time"`
    Suspected   []SuspectedEvent `gorm:"foreignKey:ReportID" json:"suspected"`
    CreatedAt   time.Time        `json:"-"`
}

// SuspectedEvent 单个异常事件
type SuspectedEvent struct {
    ID           uint                `gorm:"primaryKey;autoIncrement" json:"-"`
    ReportID     uint                `gorm:"index" json:"-"`
    EventType    SecurityEventType   `gorm:"type:int;not null" json:"event_type"`
    EventSubType string              `gorm:"type:varchar(64)" json:"event_sub_type"`
    Time         string              `gorm:"type:varchar(64)" json:"time"`
    Risk         SecurityRiskLevel   `gorm:"type:smallint" json:"risk"`
    Msg          string              `gorm:"type:varchar(128)" json:"msg"`
}
```

#### 3.2.6 告警记录

```go
// AlertRecord 告警记录完整格式
type AlertRecord struct {
    ID             string    `json:"id" gorm:"type:varchar(20);primaryKey;uniqueIndex"`
    Time           string    `json:"time" gorm:"type:varchar(19);index"`
    RuleID         int64     `json:"rule_id" gorm:"type:bigint"`
    RuleDesc       string    `json:"rule_desc" gorm:"type:varchar(1024)"`
    FilterType     int       `json:"filter_type" gorm:"type:int"`
    FileSummary    string    `json:"file_summary" gorm:"type:text"`
    AlertType      AlertType `json:"alert_type" gorm:"type:int"`
    FileMD5        string    `json:"file_md5" gorm:"type:varchar(64)"`
    FilePath       string    `json:"file_path" gorm:"type:text"`
    FileName       string    `json:"filename" gorm:"type:varchar(128)"`
    FileSize       int       `json:"filesize" gorm:"type:int"`
    HighlightText  string    `json:"highlight_text" gorm:"type:varchar(512)"`
    FileDesc       string    `json:"file_desc" gorm:"type:varchar(512)"`
    Company        string    `json:"company" gorm:"type:varchar(256)"`
    ComputerName   string    `json:"computer_name" gorm:"type:varchar(256);index"`
    OrgID          string    `json:"org_id" gorm:"type:varchar(256);index"`
    OrgPath        string    `json:"org_path" gorm:"type:varchar(512)"`
    UserName       string    `json:"user_name" gorm:"type:varchar(256);index"`
    UserID         string    `json:"user_id" gorm:"type:varchar(256);index"`
    FileLevel      int       `json:"file_xxx_level" gorm:"type:int"`
    ExtendFields   string    `json:"extend_fields" gorm:"type:text"`
}
```

#### 3.2.7 注册/注销/认证

```go
// RegisterRequest 注册请求
type RegisterRequest struct {
    SoftVersion    string                `json:"soft_version" binding:"required,max=32"`
    Interface      []InterfaceConfig     `json:"interface"`
    MemTotal       int                   `json:"mem_total" binding:"max=128"`
    CPUInfo        []CPUInfo             `json:"cpu_info"`
    DiskInfo       []DiskInfo            `json:"disk_info"`
    OrgID          string                `json:"org_id" binding:"max=128"`
    OrgCode        string                `json:"org_code" binding:"max=128"`
    UserID         string                `json:"user_id" binding:"max=128"`
    UserCode       string                `json:"user_code" binding:"max=128"`
    UserName       string                `json:"user_name" binding:"max=128"`
    HostName       string                `json:"host_name" binding:"max=128"`
    OS             string                `json:"os" binding:"max=128"`
    Arch           string                `json:"arch" binding:"max=128"`
    Memo           string                `json:"memo" binding:"max=128"`
    ExtendedFields map[string]interface{} `json:"extended_fields,omitempty"`
}

// AuthLoginResponse 认证响应
type AuthLoginResponse struct {
    Type    int    `json:"type"`
    Message string `json:"message" binding:"max=128"`
}

// RegCancelRequest 注销请求
type RegCancelRequest struct {}

// RegCancelResponse 注销响应
type RegCancelResponse struct {
    Type    int    `json:"type"`
    Message string `json:"message" binding:"max=128"`
}
```

## 3. 接口规范

### 3.1 数据模型创建接口

每个数据模型都提供了对应的构造函数，用于创建新的模型实例。例如：

```go
// 创建新的告警信息检测审计日志参数
func NewFileDetectAuditLogParam(fileMd5 []string) *FileDetectAuditLogParam

// 创建新的告警信息检测审计日志上报指令请求
func NewAuditLogCommandRequest(cmdID string, fileMd5 []string) *AuditLogCommandRequest

// 创建新的检测策略请求
func NewPolicyRequest(module, version, cmd string, num int, config interface{}) *PolicyRequest

// 创建新的关键词检测策略规则
func NewKeywordDetectRule(ruleID int64, ruleContent string) *KeywordDetectRule
```

### 3.2 数据模型方法

部分数据模型提供了辅助方法，用于简化数据操作。例如：

```go
// SecurityStatusReport 方法
func (r *SecurityStatusReport) AddSignatureAlert(filePath string, msg string) // 添加签名异常告警
func (r *SecurityStatusReport) AddNetworkAlert(remoteIP string, port uint16, msg string) // 添加网络异常告警

// CommandResultReport 方法
func (r *CommandResultReport) SetSuccess() // 设置成功结果
func (r *CommandResultReport) SetFailure(reason string) // 设置失败结果
func (r *CommandResultReport) AddDetail(detail string) // 添加详情

// StrategyExecReport 方法
func (r *StrategyExecReport) AddSuccess(ruleID int64) // 添加成功ID
func (r *StrategyExecReport) AddFail(ruleID int64, msg string) // 添加失败项
```

## 4. 使用示例

### 4.1 创建告警信息检测审计日志上报请求

```go
// 创建文件MD5数组
fileMd5 := []string{"d41d8cd98f00b204e9800998ecf8427e", "e10adc3949ba59abbe56e057f20f883e"}

// 创建告警信息检测审计日志上报指令请求
request := model.NewAuditLogCommandRequest("cmd-123456", fileMd5)

// 输出请求JSON
jsonData, err := json.Marshal(request)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(jsonData))
```

### 4.2 创建检测策略请求

```go
// 创建关键词检测策略规则
rule1 := model.NewKeywordDetectRule(1001, "机密 AND 文档")
rule2 := model.NewKeywordDetectRule(1002, "秘密 OR 绝密")

// 创建关键词检测策略配置
config := model.NewKeywordDetectConfig()
config.Rules = append(config.Rules, *rule1, *rule2)

// 创建检测策略请求
request := model.NewPolicyRequest("keyword_detect", "v1.0.0", "add", 2, config)

// 输出请求JSON
jsonData, err := json.Marshal(request)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(jsonData))
```

### 4.3 创建安全状态报告

```go
// 创建安全状态报告
report := model.NewSecurityStatusReport("v1.0.0")

// 添加签名异常告警
report.AddSignatureAlert("/etc/passwd", "文件签名验证失败")

// 添加网络异常告警
report.AddNetworkAlert("192.168.1.100", 8080, "未授权IP访问")

// 输出报告JSON
jsonData, err := json.Marshal(report)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(jsonData))
```

## 5. 数据库映射

Model模块使用GORM框架实现与SQLite数据库的映射，每个数据模型都定义了对应的数据库表名和字段类型。主要映射规则如下：

### 5.1 表名映射

每个数据模型通过`TableName()`方法指定对应的数据库表名，例如：

```go
func (AlertLogReport) TableName() string {
    return "audit_log_reports"
}

func (AlertLogItem) TableName() string {
    return "audit_log_items"
}
```

### 5.2 字段类型映射

| Go类型 | SQLite类型 | GORM标签 |
|--------|------------|----------|
| string | varchar(n) | `gorm:"type:varchar(n)"` |
| string | text | `gorm:"type:text"` |
| int    | int | `gorm:"type:int"` |
| int64  | bigint | `gorm:"type:bigint"` |
| bool   | bool | `gorm:"type:bool"` |
| []int64 | json | `gorm:"serializer:json"` |
| time.Time | datetime | 自动映射 |

### 5.3 关系映射

Model模块使用GORM的外键功能实现数据模型之间的关系映射，例如：

```go
// AlertLogReport 与 AlertLogItem 一对多关系
type AlertLogReport struct {
    // ...
    AuditLogs []AlertLogItem `gorm:"foreignKey:ReportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"audit_logs"`
    // ...
}

// SecurityStatusReport 与 SuspectedEvent 一对多关系
type SecurityStatusReport struct {
    // ...
    Suspected []SuspectedEvent `gorm:"foreignKey:ReportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"suspected"`
    // ...
}
```

## 6. 注意事项

### 6.1 数据验证

所有请求数据模型都使用了Gin框架的binding标签进行数据验证，确保数据的合法性。在使用这些模型时，建议结合Gin框架的验证机制进行数据校验。

### 6.2 数据库索引

关键字段都添加了数据库索引，以提高查询性能。在进行大量数据操作时，应注意索引的维护和优化。

### 6.3 数据大小限制

所有字符串字段都设置了合理的长度限制，避免数据库存储空间浪费和性能问题。在使用这些模型时，应确保输入数据不超过指定的长度限制。

### 6.4 时间格式

所有时间字段都使用统一的格式："2006-01-02 15:04:05"，确保时间数据的一致性和可读性。

### 6.5 扩展字段

部分数据模型提供了扩展字段（ExtendedFields），用于存储额外的自定义数据。在使用扩展字段时，应注意数据格式的一致性和兼容性。

## 7. 版本历史

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| v1.0.0 | 2026-01-20 | 初始版本，包含所有核心数据模型 |
| v1.1.0 | 2026-01-21 | 添加系统审计日志数据模型 |
| v1.2.0 | 2026-01-22 | 优化数据模型结构，添加索引 |

## 8. 总结

Model模块是Linux文件监控系统的核心数据模型层，定义了系统中所有数据结构和枚举类型。该模块采用GORM框架实现与SQLite数据库的映射，确保数据的一致性和完整性。通过使用统一的数据模型，系统各模块之间的数据交换更加规范和高效，同时也便于系统的扩展和维护。

该模块的设计遵循了以下原则：

- **单一职责**：每个数据模型只负责定义一种数据结构
- **易于扩展**：支持新功能的数据模型快速添加
- **数据库友好**：与GORM框架无缝集成
- **接口标准化**：规范系统内部和外部接口的数据格式

通过合理使用Model模块提供的数据模型和接口，可以简化系统开发，提高代码质量和可维护性。