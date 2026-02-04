// Package config
package config

import "time"

// ==========================================
// 顶层配置结构
// ==========================================

type AppConfig struct {
	Agent    AgentConfig    `mapstructure:"agent" yaml:"agent"`
	Server   ServerConfig   `mapstructure:"server" yaml:"server"`
	Scanner  ScannerConfig  `mapstructure:"scanner" yaml:"scanner"`
	Security SecurityConfig `mapstructure:"security" yaml:"security"`
	Database DatabaseConfig `mapstructure:"database" yaml:"database"`
	Storage  StorageConfig  `mapstructure:"storage" yaml:"storage"`
}

// ==========================================
// 1. 基础配置
// ==========================================

type AgentConfig struct {
	// 日志级别: debug, info, warn, error
	LogLevel string `mapstructure:"log_level" yaml:"log_level"`
	// 日志文件路径
	LogFile string `mapstructure:"log_file" yaml:"log_file"`
	// 数据存储目录 (覆盖默认的 /var/lib/...)
	DataDir string `mapstructure:"data_dir" yaml:"data_dir"`
	// 【新增】日志轮转高级配置
	LogMaxSize    int  `mapstructure:"log_max_size" yaml:"log_max_size"`       // MB
	LogMaxBackups int  `mapstructure:"log_max_backups" yaml:"log_max_backups"` // 个数
	LogMaxAge     int  `mapstructure:"log_max_age" yaml:"log_max_age"`         // 天数
	LogCompress   bool `mapstructure:"log_compress" yaml:"log_compress"`       // 是否压缩
	LogStdout     bool `mapstructure:"log_stdout" yaml:"log_stdout"`           // 是否打印到控制台
}

// ==========================================
// 5. 数据库配置
// ==========================================

type DatabaseConfig struct {
	// 数据库文件名
	FileName string `mapstructure:"file_name" yaml:"file_name"`
	// GORM 日志级别: silent, error, warn, info
	LogLevel string `mapstructure:"log_level" yaml:"log_level"`
	// 最大打开连接数 (SQLite 建议 1)
	MaxOpenConns int `mapstructure:"max_open_conns" yaml:"max_open_conns"`
	// 最大空闲连接数 (SQLite 建议 1)
	MaxIdleConns int `mapstructure:"max_idle_conns" yaml:"max_idle_conns"`
	// 连接最大生命周期
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" yaml:"conn_max_lifetime"`
	// SQLite Journal 模式: WAL, DELETE, TRUNCATE, PERSIST, MEMORY
	JournalMode string `mapstructure:"journal_mode" yaml:"journal_mode"`
	// SQLite 同步模式: FULL, NORMAL, OFF
	Synchronous string `mapstructure:"synchronous" yaml:"synchronous"`
	// SQLite 临时存储: MEMORY, FILE
	TempStore string `mapstructure:"temp_store" yaml:"temp_store"`
	// 是否启用外键约束
	ForeignKeys bool `mapstructure:"foreign_keys" yaml:"foreign_keys"`
}

// ==========================================
// 6. 存储引擎配置
// ==========================================

type StorageConfig struct {
	// 告警记录内存存储上限
	AlertsMemoryLimit int `mapstructure:"alerts_memory_limit" yaml:"alerts_memory_limit"`
	// 审计日志内存存储上限
	AuditLogsMemoryLimit int `mapstructure:"audit_logs_memory_limit" yaml:"audit_logs_memory_limit"`
	// 安全状态上报内存存储上限
	SecurityReportsLimit int `mapstructure:"security_reports_limit" yaml:"security_reports_limit"`
	AlertLogsMemoryLimit int `mapstructure:"alert_logs_memory_limit" yaml:"alert_logs_memory_limit"`
}

// ==========================================
// 2. 通信配置 (对应模块四)
// ==========================================

type ServerConfig struct {
	// 管理平台地址 (e.g., https://10.0.0.1:8443)
	URL string `mapstructure:"url" yaml:"url"`
	// CA 根证书路径
	CACert string `mapstructure:"ca_cert" yaml:"ca_cert"`
	// 客户端证书路径
	ClientCert string `mapstructure:"client_cert" yaml:"client_cert"`
	// 客户端私钥路径
	ClientKey string `mapstructure:"client_key" yaml:"client_key"`
	// HTTP 请求超时
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout"`
	// 最大空闲连接数
	MaxIdleConns int `mapstructure:"max_idle_conns" yaml:"max_idle_conns"`
	// 空闲连接超时
	IdleConnTimeout time.Duration `mapstructure:"idle_conn_timeout" yaml:"idle_conn_timeout"`
}

// ==========================================
// 3. 扫描策略 (对应模块一)
// ==========================================

type ScannerConfig struct {
	// 监控目录列表
	WatchDirs []string `mapstructure:"watch_dirs" yaml:"watch_dirs"`
	// 排除目录列表
	ExcludeDirs []string `mapstructure:"exclude_dirs" yaml:"exclude_dirs"`
	// 扫描限流 (每秒文件数)
	RateLimit int `mapstructure:"rate_limit" yaml:"rate_limit"`
	// 并发 Worker 数
	Workers int `mapstructure:"workers" yaml:"workers"`
	// 策略文件目录路径
	PoliciesPath string `mapstructure:"policies_path" yaml:"policies_path"`
}

// ==========================================
// 4. 安全策略 (对应模块五 & 六)
// ==========================================

type SecurityConfig struct {
	// 自身完整性检测
	Integrity IntegrityConfig `mapstructure:"integrity" yaml:"integrity"`
	// 网络异常检测
	NetGuard NetGuardConfig `mapstructure:"netguard" yaml:"netguard"`
}

type IntegrityConfig struct {
	// 检测周期 (e.g., "5m")
	CheckInterval time.Duration `mapstructure:"check_interval" yaml:"check_interval"`
	// 默认检测周期 (当传入无效值时使用)
	DefaultInterval time.Duration `mapstructure:"default_interval" yaml:"default_interval"`
}

type NetGuardConfig struct {
	// 是否开启
	Enable bool `mapstructure:"enable" yaml:"enable"`
	// 检测周期 (e.g., "1s")
	CheckInterval time.Duration `mapstructure:"check_interval" yaml:"check_interval"`
	// 额外白名单 (除回环和服务端IP外)
	Whitelist []string `mapstructure:"whitelist" yaml:"whitelist"`
	// 去重时间 (e.g., "1h")
	DeduplicationTime time.Duration `mapstructure:"deduplication_time" yaml:"deduplication_time"`
	// 监控自身
	MonitorSelf bool `mapstructure:"monitor_self" yaml:"monitor_self"`
}
