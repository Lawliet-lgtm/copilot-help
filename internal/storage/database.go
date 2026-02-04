package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"linuxFileWatcher/internal/logger"
)

var (
	db   *gorm.DB
	once sync.Once
)

// Options 数据库初始化选项
type Options struct {
	DataDir         string
	FileName        string
	LogLevel        string        // silent, error, warn, info
	MaxOpenConns    int           // 推荐: 1
	MaxIdleConns    int           // 推荐: 1
	ConnMaxLifetime time.Duration // 推荐: 1h
	JournalMode     string        // WAL
	Synchronous     string        // NORMAL
	TempStore       string        // MEMORY
	ForeignKeys     bool          // true
}

// Setup 初始化数据库
// 修改点：增加 error 返回值，让调用者感知失败
func Setup(opts Options) error {
	var err error

	once.Do(func() {
		// 1. config模块已设置参数默认值兜底，此处无需重复兜底

		// 2. 创建目录
		if mkErr := os.MkdirAll(opts.DataDir, 0755); mkErr != nil {
			err = fmt.Errorf("failed to create db dir %s: %w", opts.DataDir, mkErr)
			logger.Error("DB Setup Error", "details", err)
			return
		}

		dbPath := filepath.Join(opts.DataDir, opts.FileName)

		// 3. 配置 GORM 日志
		var gormLogLevel gormlogger.LogLevel
		switch strings.ToLower(opts.LogLevel) {
		case "silent":
			gormLogLevel = gormlogger.Silent
		case "error":
			gormLogLevel = gormlogger.Error
		case "info":
			gormLogLevel = gormlogger.Info
		default:
			gormLogLevel = gormlogger.Warn
		}

		gormConfig := &gorm.Config{
			Logger:                 gormlogger.Default.LogMode(gormLogLevel),
			PrepareStmt:            true,
			SkipDefaultTransaction: true, // 禁用默认事务，避免事务冲突
		}

		// 4. 打开连接
		dbConn, openErr := gorm.Open(sqlite.Open(dbPath), gormConfig)
		if openErr != nil {
			err = fmt.Errorf("failed to open sqlite %s: %w", dbPath, openErr)
			logger.Error("DB Setup Error", "details", err)
			return
		}

		// 5. 配置连接池
		sqlDB, sqlErr := dbConn.DB()
		if sqlErr != nil {
			err = fmt.Errorf("failed to get sql.DB: %w", sqlErr)
			return
		}

		sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
		sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)

		// 6. 执行 PRAGMA 优化
		// 注意：Foreign Keys 在 SQLite 中是连接级生效的，
		// 但因为我们锁定了 MaxOpenConns=1，所以在这里执行一次是安全的。
		pragmas := []string{
			fmt.Sprintf("PRAGMA journal_mode = %s;", opts.JournalMode),
			fmt.Sprintf("PRAGMA synchronous = %s;", opts.Synchronous),
			fmt.Sprintf("PRAGMA temp_store = %s;", opts.TempStore),
		}

		if opts.ForeignKeys {
			pragmas = append(pragmas, "PRAGMA foreign_keys = ON;")
		}

		for _, p := range pragmas {
			if execErr := dbConn.Exec(p).Error; execErr != nil {
				err = fmt.Errorf("failed to exec pragma %s: %w", p, execErr)
				logger.Error("DB Setup Error", "details", err)
				return
			}
		}

		// 赋值给全局变量
		db = dbConn

		logger.Info("Database initialized",
			"path", dbPath,
			"journal_mode", opts.JournalMode,
			"foreign_keys", opts.ForeignKeys,
		)
	})

	return err
}

// GetDB 获取数据库实例
func GetDB() (*gorm.DB, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized! call Setup() first")
	}
	return db, nil
}

// CloseDB 关闭数据库连接
// 用于测试结束时释放资源
func CloseDB() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return fmt.Errorf("failed to get sql.DB: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}
