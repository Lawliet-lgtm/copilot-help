package storage

import (
	"encoding/json"
	"fmt"
	"sync"

	"gorm.io/gorm"

	"linuxFileWatcher/internal/logger"
	"linuxFileWatcher/internal/security" // 需要调用加密
)

// HybridStore 混合存储引擎
type HybridStore[T any] struct {
	db        *gorm.DB
	tableName string // 对应的数据库表名 (e.g., "storage_alerts")

	memStore []T
	memLimit int
	mu       sync.RWMutex
}

// NewHybridStore 初始化
// tableName: 必须指定，用于区分是告警表还是审计表
func NewHybridStore[T any](db *gorm.DB, limit int, tableName string) (*HybridStore[T], error) {
	// 检查并创建表（如果不存在）
	if !db.Migrator().HasTable(tableName) {
		if err := db.Table(tableName).AutoMigrate(&DiskRecord{}); err != nil {
			logger.Error("Failed to create table", "table", tableName, "error", err)
			return nil, err
		}
		logger.Info("Created table successfully", "table", tableName)
	} else {
		logger.Debug("Table already exists", "table", tableName)
	}

	return &HybridStore[T]{
		db:        db,
		tableName: tableName,
		memStore:  make([]T, 0, limit),
		memLimit:  limit,
	}, nil
}

// Push 写入数据
func (s *HybridStore[T]) Push(item T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 内存未满，直接存
	if len(s.memStore) < s.memLimit {
		s.memStore = append(s.memStore, item)
		return nil
	}

	// 2. 内存已满，触发溢出落盘 (Spillover)
	// 执行：结构体 -> JSON -> SM4 -> DiskRecord -> DB
	return s.persistToDisk([]T{item})
}

// PopAll 取出并清空 (消费端调用)
func (s *HybridStore[T]) PopAll() ([]T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []T

	// 1. 取出内存数据
	if len(s.memStore) > 0 {
		result = append(result, s.memStore...)
		// 清空内存
		s.memStore = make([]T, 0, s.memLimit)
	}

	// 2. 取出磁盘数据
	// 动态指定表名查询
	var diskRecords []DiskRecord
	err := s.db.Table(s.tableName).Find(&diskRecords).Error
	if err != nil {
		return nil, fmt.Errorf("read disk failed: %v", err)
	}

	if len(diskRecords) > 0 {
		// 解密并反序列化
		for _, rec := range diskRecords {
			item, err := decodeAndDecrypt[T](rec.Data)
			if err != nil {
				// 遇到解密失败的数据（可能被篡改），记录日志并跳过，防止阻塞整个上报
				logger.Error("Storage decrypt error", "id", rec.ID, "error", err)
				continue
			}
			result = append(result, *item)
		}

		// 3. 物理删除磁盘数据
		// 使用 Unscoped 硬删除
		if err := s.db.Table(s.tableName).Unscoped().Where("1 = 1").Delete(&DiskRecord{}).Error; err != nil {
			return nil, fmt.Errorf("clean disk failed: %v", err)
		}
	}

	return result, nil
}

// FlushMemoryToDisk 强制刷盘 (程序退出时用)
func (s *HybridStore[T]) FlushMemoryToDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.memStore) == 0 {
		return nil
	}

	// 批量写入
	if err := s.persistToDisk(s.memStore); err != nil {
		return err
	}

	flushedCount := len(s.memStore)
	s.memStore = make([]T, 0, s.memLimit)
	logger.Info("Storage flushed items to disk", "count", flushedCount, "table", s.tableName)
	return nil
}

// ==========================================
// 内部私有辅助函数
// ==========================================

// persistToDisk 将一组业务对象加密写入磁盘
func (s *HybridStore[T]) persistToDisk(items []T) error {
	diskRecords := make([]DiskRecord, 0, len(items))

	for _, item := range items {
		// A. 序列化
		jsonBytes, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("json marshal failed: %v", err)
		}

		// B. 全量加密
		cipherBytes, err := security.EncryptLocal(jsonBytes)
		if err != nil {
			return fmt.Errorf("encrypt failed: %v", err)
		}

		// C. 包装
		diskRecords = append(diskRecords, DiskRecord{
			Data: cipherBytes,
		})
	}

	// D. 批量插入 (动态表名)
	// 检查并创建表（如果不存在）
	if !s.db.Migrator().HasTable(s.tableName) {
		if err := s.db.Table(s.tableName).AutoMigrate(&DiskRecord{}); err != nil {
			return fmt.Errorf("create table failed: %v", err)
		}
	}

	// 执行批量插入，不使用事务，避免事务冲突
	return s.db.Table(s.tableName).CreateInBatches(diskRecords, 100).Error
}

// decodeAndDecrypt 解密并反序列化
func decodeAndDecrypt[T any](cipherData []byte) (*T, error) {
	// A. 解密
	jsonBytes, err := security.DecryptLocal(cipherData)
	if err != nil {
		return nil, err
	}

	// B. 反序列化
	var item T
	if err := json.Unmarshal(jsonBytes, &item); err != nil {
		return nil, err
	}

	return &item, nil
}
