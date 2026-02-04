package storage

import (
	"sync"

	"gorm.io/gorm"

	"linuxFileWatcher/internal/model"
)

// Stores 存储实例管理器
// 集中管理所有混合存储引擎实例
var (
	stores     *Stores
	storesOnce sync.Once
)

// Stores 存储实例集合
// 为每种业务数据类型提供对应的 HybridStore
// 这种设计方便统一管理和访问
// 后续新增业务类型时，只需在此结构体中添加对应字段即可
// 避免了在代码中到处创建和传递存储实例
// 支持并发访问，因为 HybridStore 内部已经实现了线程安全
// 注意：此结构体需要在应用启动时初始化
// 使用方式：storage.GetStores().Alerts.Push(alert)
type Stores struct {
	// 告警记录存储
	Alerts *HybridStore[model.AlertRecord]

	// 审计日志上报存储
	AuditLogs *HybridStore[model.SystemAuditRequest]

	// 安全状态上报存储
	SecurityReports *HybridStore[model.SecurityStatusReport]

	// 告警日志上报存储
	AlertLogs *HybridStore[model.AlertLogItem]
	// --- 新增：结果上报缓存 (用于断网重传) ---
	// CommandResults 缓存发送失败的指令执行结果
	CommandResults *HybridStore[model.CommandResultReport]
	// PolicyResults 缓存发送失败的策略执行结果
	PolicyResults *HybridStore[model.StrategyExecReport]
}

// StoresOptions 存储实例配置选项
// 为每种存储类型提供独立的内存限制配置
// 这样可以根据不同业务数据的特性进行调优
// 例如：告警记录可能产生频繁，可以设置较小的内存限制
// 而审计日志可能批量产生，可以设置较大的内存限制
// 安全状态上报和指令/策略结果上报可能批量产生，也可以设置较大的内存限制
type StoresOptions struct {
	AlertsMemoryLimit    int // 告警记录内存存储上限
	AuditLogsMemoryLimit int // 审计日志内存存储上限
	SecurityReportsLimit int // 安全状态上报内存存储上限
	AlertLogsMemoryLimit int // 告警日志内存存储上限
	// // 新增模块的内存限制通常较小，可以直接内置或扩展配置，这里为了简洁使用内置默认值
	//CommandResultsLimit int // 指令执行结果缓存内存存储上限
	//PolicyResultsLimit  int // 策略执行结果缓存内存存储上限
}

// SetupStores 初始化所有存储实例
// db: 数据库连接实例，必须提前初始化
// opts: 存储实例配置选项
// 返回值：错误信息，如果初始化失败
// 注意：此函数使用 sync.Once 确保只初始化一次
// 调用方需要确保 db 连接已经成功初始化
// 使用方式：storage.SetupStores(db, opts)
func SetupStores(db *gorm.DB, opts StoresOptions) error {
	var err error

	storesOnce.Do(func() {
		// 1. 初始化告警记录存储
		alertsStore, alertsErr := NewHybridStore[model.AlertRecord](
			db,
			opts.AlertsMemoryLimit,
			"storage_alerts", // 表名：存储告警记录
		)
		if alertsErr != nil {
			err = alertsErr
			return
		}

		// 2. 初始化审计日志存储
		auditLogsStore, auditErr := NewHybridStore[model.SystemAuditRequest](
			db,
			opts.AuditLogsMemoryLimit,
			"storage_audit_logs", // 表名：存储审计日志
		)
		if auditErr != nil {
			err = auditErr
			return
		}

		// 3. 初始化安全状态上报存储
		securityReportsStore, securityErr := NewHybridStore[model.SecurityStatusReport](
			db,
			opts.SecurityReportsLimit,
			"storage_security_reports", // 表名：存储安全状态上报
		)
		if securityErr != nil {
			err = securityErr
			return
		}
		// 新加的4. 初始化指令结果缓存
		// 内存保留 50 条，多余的落盘，表名 storage_command_results
		cmdResultStore, cmdErr := NewHybridStore[model.CommandResultReport](db, 50, "storage_command_results")
		if cmdErr != nil {
			err = cmdErr
			return
		}

		// 新加的5. 初始化策略结果缓存
		// 内存保留 50 条，多余的落盘，表名 storage_policy_results
		policyResultStore, policyErr := NewHybridStore[model.StrategyExecReport](db, 50, "storage_policy_results")
		if policyErr != nil {
			err = policyErr
			return
		}

		// 4. 初始化告警日志存储
		alertLogsStore, alertLogsErr := NewHybridStore[model.AlertLogItem](
			db,
			opts.AlertLogsMemoryLimit,
			"storage_alert_logs", // 表名：存储告警日志
		)
		if alertLogsErr != nil {
			err = alertLogsErr
			return
		}
		// 5. 创建存储实例管理器
		stores = &Stores{
			Alerts:          alertsStore,
			AuditLogs:       auditLogsStore,
			SecurityReports: securityReportsStore,
			AlertLogs:       alertLogsStore,
			CommandResults:  cmdResultStore,
			PolicyResults:   policyResultStore,
		}
	})

	return err
}

// GetStores 获取存储实例管理器
// 返回值：存储实例管理器指针
// 注意：必须先调用 SetupStores 初始化，否则返回 nil
// 使用方式：storage.GetStores().Alerts.Push(alert)
func GetStores() *Stores {
	return stores
}

// FlushAll 刷新所有存储实例的内存数据到磁盘
// 返回值：错误信息，如果刷新失败
// 注意：在应用程序退出前应该调用此方法，确保数据不丢失
// 使用方式：defer storage.FlushAll()
func FlushAll() error {
	if stores == nil {
		return nil // 未初始化，直接返回
	}

	// 依次刷新所有存储实例
	if err := stores.Alerts.FlushMemoryToDisk(); err != nil {
		return err
	}

	if err := stores.AuditLogs.FlushMemoryToDisk(); err != nil {
		return err
	}

	if err := stores.SecurityReports.FlushMemoryToDisk(); err != nil {
		return err
	}

	if err := stores.CommandResults.FlushMemoryToDisk(); err != nil {
		return err
	}

	if err := stores.PolicyResults.FlushMemoryToDisk(); err != nil {
		return err
	}

	if err := stores.AlertLogs.FlushMemoryToDisk(); err != nil {

		return err
	}
	return nil
}
