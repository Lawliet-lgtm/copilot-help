package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// GlobalConfig 全局配置单例
// 在调用 LoadConfig 成功后，该变量会被填充，后续模块直接读取即可
var (
	GlobalConfig *AppConfig
	loadOnce     sync.Once
)

// LoadConfig 加载配置
// configPath: 配置文件路径 (e.g., "/etc/linuxFileWatcher/config.yaml")
// 如果传入空字符串，Viper 会尝试在默认路径搜索
func LoadConfig(configPath string) error {
	var err error

	loadOnce.Do(func() {
		v := viper.New()

		// 1. 设置默认值 (兜底策略)
		setDefaults(v)

		// 2. 配置读取规则
		if configPath != "" {
			// 如果指定了具体文件，直接读取
			v.SetConfigFile(configPath)
		} else {
			// 否则在常见目录搜索名为 "config" 的文件
			v.SetConfigName("config")
			v.SetConfigType("yaml")
			v.AddConfigPath("/etc/linuxFileWatcher/") // 生产环境标准路径
			v.AddConfigPath(".")                      // 当前目录 (开发调试用)
		}

		// 3. 配置环境变量覆盖 (高级特性)
		// 允许通过环境变量 LFW_SERVER_URL 来覆盖 server.url
		v.SetEnvPrefix("LFW")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()

		// 4. 读取配置文件
		if err = v.ReadInConfig(); err != nil {
			// 如果是“未找到配置文件”错误，且我们要用默认值跑，可以忽略
			// 但对于安全软件，建议强制要求配置文件存在
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				err = fmt.Errorf("config file not found: %v", err)
				return
			}
			err = fmt.Errorf("failed to read config file: %v", err)
			return
		}

		// 5. 反序列化到结构体
		var config AppConfig
		if err = v.Unmarshal(&config); err != nil {
			err = fmt.Errorf("failed to unmarshal config: %v", err)
			return
		}

		// 6. 赋值给全局单例
		GlobalConfig = &config
		fmt.Printf("[Config] Loaded successfully from: %s\n", v.ConfigFileUsed())
	})

	return err
}

// setDefaults 定义配置文件的“默认行为”
func setDefaults(v *viper.Viper) {
	// Agent 基础
	v.SetDefault("agent.log_level", "info")
	v.SetDefault("agent.log_file", "/var/log/linuxFileWatcher/agent.log")
	v.SetDefault("agent.data_dir", "/var/lib/linuxFileWatcher") // 数据存储目录默认值
	// 【新增】日志轮转默认值 (参考业界标准)
	v.SetDefault("agent.log_max_size", 100)  // 100MB 切割
	v.SetDefault("agent.log_max_backups", 5) // 保留最近 5 个
	v.SetDefault("agent.log_max_age", 30)    // 保留 30 天
	v.SetDefault("agent.log_compress", true) // 默认压缩旧日志
	v.SetDefault("agent.log_stdout", false)  // 生产环境默认不打控制台(静默模式)

	// Server 通信
	v.SetDefault("server.timeout", "30s")
	v.SetDefault("server.max_idle_conns", 10)
	v.SetDefault("server.idle_conn_timeout", "30s")

	// Scanner 扫描策略 (保守默认值)
	v.SetDefault("scanner.rate_limit", 500)
	v.SetDefault("scanner.workers", 1)
	v.SetDefault("scanner.watch_dirs", []string{"/home"}) // 默认只扫 home
	v.SetDefault("scanner.policies_path", "./policies")   // 默认策略文件目录

	// Security 安全策略
	v.SetDefault("security.integrity.check_interval", "5m")
	v.SetDefault("security.integrity.default_interval", "1m")

	v.SetDefault("security.netguard.enable", true)
	v.SetDefault("security.netguard.check_interval", "1s")
	v.SetDefault("security.netguard.deduplication_time", "1h")
	v.SetDefault("security.netguard.monitor_self", true)
	// 默认白名单至少包含回环，虽然代码里强制加了，这里配置上也体现一下更好
	v.SetDefault("security.netguard.whitelist", []string{"127.0.0.1", "::1"})

	// Database 数据库配置
	v.SetDefault("database.file_name", "agent.db")
	v.SetDefault("database.log_level", "warn")
	v.SetDefault("database.max_open_conns", 1)
	v.SetDefault("database.max_idle_conns", 1)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("database.journal_mode", "WAL")
	v.SetDefault("database.synchronous", "NORMAL")
	v.SetDefault("database.temp_store", "MEMORY")
	v.SetDefault("database.foreign_keys", true)

	// Storage 存储引擎配置
	v.SetDefault("storage.alerts_memory_limit", 100)     // 告警记录内存限制：100条
	v.SetDefault("storage.audit_logs_memory_limit", 200) // 审计日志内存限制：200条
	v.SetDefault("storage.security_reports_limit", 50)   // 安全状态上报内存限制：50条
}

// Get 获取配置的安全访问器 (可选)
func Get() *AppConfig {
	if GlobalConfig == nil {
		// 防御性编程：如果没有初始化就调用，返回一个空结构或 panic
		// 这里为了安全起见，建议 panic 提示开发者必须先 Init
		panic("Config not initialized! Call LoadConfig() first.")
	}
	return GlobalConfig
}
