package storage

// DiskRecord 是真正存入 SQLite 的物理表结构
// 无论业务数据长什么样，在磁盘上都只是加密后的二进制块
type DiskRecord struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Data      []byte `gorm:"type:blob"`      // 核心数据：SM4(JSON(BusinessData))
	CreatedAt int64  `gorm:"autoCreateTime"` // 可选：用于调试查看写入时间
}

// 我们需要为不同的业务类型（告警、审计）指定不同的表名
// 否则它们会混在一个表里
type TableNamer interface {
	TableName() string
}
