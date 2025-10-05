package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"crosswire/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database 数据库管理器
type Database struct {
	channelDB *gorm.DB // 频道数据库
	userDB    *gorm.DB // 用户数据库
	cacheDB   *gorm.DB // 缓存数据库
	dataDir   string   // 数据目录
}

// Config 数据库配置
type Config struct {
	DataDir   string
	DebugMode bool
}

// NewDatabase 创建数据库实例
func NewDatabase(config *Config) (*Database, error) {
	db := &Database{
		dataDir: config.DataDir,
	}

	// 确保数据目录存在
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	if config.DebugMode {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	// 初始化用户数据库
	userDBPath := filepath.Join(config.DataDir, "user.db")
	userDB, err := gorm.Open(sqlite.Open(userDBPath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open user database: %w", err)
	}
	db.userDB = userDB

	// 配置 SQLite
	if err := db.configureSQLite(userDB); err != nil {
		return nil, err
	}

	// 初始化缓存数据库
	cacheDBPath := filepath.Join(config.DataDir, "cache.db")
	cacheDB, err := gorm.Open(sqlite.Open(cacheDBPath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}
	db.cacheDB = cacheDB

	// 配置 SQLite
	if err := db.configureSQLite(cacheDB); err != nil {
		return nil, err
	}

	// 自动迁移用户数据库
	if err := db.migrateUserDB(); err != nil {
		return nil, err
	}

	// 自动迁移缓存数据库
	if err := db.migrateCacheDB(); err != nil {
		return nil, err
	}

	return db, nil
}

// OpenChannelDB 打开频道数据库
func (db *Database) OpenChannelDB(channelID string) error {
	channelsDir := filepath.Join(db.dataDir, "channels")
	if err := os.MkdirAll(channelsDir, 0755); err != nil {
		return fmt.Errorf("failed to create channels directory: %w", err)
	}

	channelDBPath := filepath.Join(channelsDir, fmt.Sprintf("%s.db", channelID))
	channelDB, err := gorm.Open(sqlite.Open(channelDBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to open channel database: %w", err)
	}

	db.channelDB = channelDB

	// 配置 SQLite
	if err := db.configureSQLite(channelDB); err != nil {
		return err
	}

	// 自动迁移频道数据库
	return db.migrateChannelDB()
}

// configureSQLite 配置 SQLite 优化参数
func (db *Database) configureSQLite(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}

	// 设置连接池
	sqlDB.SetMaxOpenConns(1) // SQLite 建议单连接
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 执行 PRAGMA 配置
	pragmas := []string{
		"PRAGMA journal_mode = WAL",        // 启用 WAL 模式
		"PRAGMA synchronous = NORMAL",      // 平衡性能与安全
		"PRAGMA cache_size = -20000",       // 20MB 缓存
		"PRAGMA foreign_keys = ON",         // 启用外键约束
		"PRAGMA temp_store = MEMORY",       // 临时文件在内存
		"PRAGMA mmap_size = 268435456",     // 256MB mmap
		"PRAGMA page_size = 4096",          // 4KB 页面大小
		"PRAGMA auto_vacuum = INCREMENTAL", // 增量清理
	}

	for _, pragma := range pragmas {
		if err := gdb.Exec(pragma).Error; err != nil {
			return fmt.Errorf("failed to execute pragma: %w", err)
		}
	}

	return nil
}

// migrateUserDB 迁移用户数据库
func (db *Database) migrateUserDB() error {
	return db.userDB.AutoMigrate(
		&models.UserProfile{},
		&models.RecentChannel{},
	)
}

// migrateCacheDB 迁移缓存数据库
func (db *Database) migrateCacheDB() error {
	return db.cacheDB.AutoMigrate(
		&models.CacheEntry{},
	)
}

// migrateChannelDB 迁移频道数据库
func (db *Database) migrateChannelDB() error {
	// TODO: 实现全文搜索表的创建（FTS5）
	// 目前先迁移基础表
	return db.channelDB.AutoMigrate(
		&models.Channel{},
		&models.Member{},
		&models.Message{},
		&models.MessageReaction{},
		&models.TypingStatus{},
		&models.File{},
		&models.FileChunk{},
		&models.AuditLog{},
		&models.MuteRecord{},
		&models.PinnedMessage{},
		&models.Challenge{},
		&models.ChallengeAssignment{},
		&models.ChallengeProgress{},
		&models.ChallengeSubmission{},
		&models.ChallengeHint{},
	)
}

// GetChannelDB 获取频道数据库
func (db *Database) GetChannelDB() *gorm.DB {
	return db.channelDB
}

// GetUserDB 获取用户数据库
func (db *Database) GetUserDB() *gorm.DB {
	return db.userDB
}

// GetCacheDB 获取缓存数据库
func (db *Database) GetCacheDB() *gorm.DB {
	return db.cacheDB
}

// Close 关闭数据库连接
func (db *Database) Close() error {
	var errs []error

	if db.channelDB != nil {
		if sqlDB, err := db.channelDB.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if sqlDB, err := db.userDB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if sqlDB, err := db.cacheDB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}

	return nil
}

// TODO: 实现以下数据库操作方法
// - CreateChannel() 创建频道
// - GetChannel() 获取频道信息
// - UpdateChannel() 更新频道
// - DeleteChannel() 删除频道
//
// - AddMember() 添加成员
// - GetMembers() 获取成员列表
// - UpdateMemberStatus() 更新成员状态
// - RemoveMember() 移除成员
//
// - SaveMessage() 保存消息
// - GetMessages() 获取消息列表
// - SearchMessages() 搜索消息（全文搜索）
// - DeleteMessage() 删除消息
//
// - SaveFile() 保存文件
// - GetFile() 获取文件
// - GetFiles() 获取文件列表
//
// - CreateChallenge() 创建题目
// - GetChallenges() 获取题目列表
// - AssignChallenge() 分配题目
// - SubmitFlag() 提交 Flag
// - UpdateProgress() 更新进度
//
// - SaveAuditLog() 保存审计日志
// - GetAuditLogs() 获取审计日志
//
// - MuteMember() 禁言成员
// - UnmuteMember() 解除禁言
// - IsMuted() 检查是否被禁言
//
// - PinMessage() 置顶消息
// - UnpinMessage() 取消置顶
// - GetPinnedMessages() 获取置顶消息
//
// - SaveCache() 保存缓存
// - GetCache() 获取缓存
// - CleanExpiredCache() 清理过期缓存
