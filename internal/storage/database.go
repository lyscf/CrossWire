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
	// 迁移基础表
	if err := db.channelDB.AutoMigrate(
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
	); err != nil {
		return err
	}

	// 注意：不使用 FTS5，搜索功能使用 LIKE 查询实现
	return nil
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

// ==================== Repository方法 ====================

// MessageRepo 获取消息仓库
func (db *Database) MessageRepo() *MessageRepository {
	return NewMessageRepository(db)
}

// FileRepo 获取文件仓库
func (db *Database) FileRepo() *FileRepository {
	return NewFileRepository(db)
}

// MemberRepo 获取成员仓库
func (db *Database) MemberRepo() *MemberRepository {
	return NewMemberRepository(db)
}

// ChallengeRepo 获取题目仓库
func (db *Database) ChallengeRepo() *ChallengeRepository {
	return NewChallengeRepository(db)
}

// ChallengeSubmissionRepo 获取题目提交仓库（使用ChallengeRepo代替）
func (db *Database) ChallengeSubmissionRepo() *ChallengeRepository {
	return NewChallengeRepository(db)
}

// ChannelRepo 获取频道仓库
func (db *Database) ChannelRepo() *ChannelRepository {
	return NewChannelRepository(db)
}

// AuditRepo 获取审计仓库
func (db *Database) AuditRepo() *AuditRepository {
	return NewAuditRepository(db)
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

// ==================== 频道（Channel） ====================

// CreateChannel 创建频道
func (db *Database) CreateChannel(channel *models.Channel) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.ChannelRepo().Create(channel)
}

// GetChannel 获取频道信息
func (db *Database) GetChannel(channelID string) (*models.Channel, error) {
	if db.channelDB == nil {
		return nil, fmt.Errorf("channel database is not opened")
	}
	return db.ChannelRepo().GetByID(channelID)
}

// UpdateChannel 更新频道
func (db *Database) UpdateChannel(channel *models.Channel) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.ChannelRepo().Update(channel)
}

// DeleteChannel 删除频道
func (db *Database) DeleteChannel(channelID string) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.ChannelRepo().Delete(channelID)
}

// ==================== 成员（Member） ====================

// AddMember 添加成员
func (db *Database) AddMember(member *models.Member) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.MemberRepo().Create(member)
}

// GetMembers 获取成员列表
func (db *Database) GetMembers(channelID string) ([]*models.Member, error) {
	if db.channelDB == nil {
		return nil, fmt.Errorf("channel database is not opened")
	}
	return db.MemberRepo().GetByChannelID(channelID)
}

// UpdateMemberStatus 更新成员状态
func (db *Database) UpdateMemberStatus(memberID string, status models.UserStatus) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.MemberRepo().UpdateStatus(memberID, status)
}

// RemoveMember 移除成员
func (db *Database) RemoveMember(memberID string) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.MemberRepo().Delete(memberID)
}

// ==================== 消息（Message） ====================

// SaveMessage 保存消息
func (db *Database) SaveMessage(message *models.Message) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	if err := db.MessageRepo().Create(message); err != nil {
		return err
	}
	// 更新统计（容错处理）
	_ = db.ChannelRepo().IncrementMessageCount(message.ChannelID)
	_ = db.MemberRepo().IncrementMessageCount(message.SenderID)
	return nil
}

// GetMessages 获取消息列表（分页）
func (db *Database) GetMessages(channelID string, limit, offset int) ([]*models.Message, error) {
	if db.channelDB == nil {
		return nil, fmt.Errorf("channel database is not opened")
	}
	return db.MessageRepo().GetByChannelID(channelID, limit, offset)
}

// SearchMessages 搜索消息（使用LIKE查询）
func (db *Database) SearchMessages(channelID, keyword string, limit, offset int) ([]*models.Message, error) {
	if db.channelDB == nil {
		return nil, fmt.Errorf("channel database is not opened")
	}

	if keyword == "" {
		return db.GetMessages(channelID, limit, offset)
	}

	// 使用 LIKE 搜索（content_text / sender_nickname / tags）
	like := "%" + keyword + "%"
	var messages []*models.Message
	err := db.channelDB.Where("channel_id = ? AND deleted = 0 AND (content_text LIKE ? OR sender_nickname LIKE ? OR tags LIKE ?)",
		channelID, like, like, like).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// DeleteMessage 删除消息（软删除）
func (db *Database) DeleteMessage(messageID, deletedBy string) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.MessageRepo().Delete(messageID, deletedBy)
}

// ==================== 文件（File） ====================

// SaveFile 保存文件
func (db *Database) SaveFile(file *models.File) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	if err := db.FileRepo().Create(file); err != nil {
		return err
	}
	// 更新统计（容错处理）
	_ = db.ChannelRepo().IncrementFileCount(file.ChannelID)
	_ = db.MemberRepo().IncrementFilesShared(file.SenderID)
	return nil
}

// GetFile 获取文件
func (db *Database) GetFile(fileID string) (*models.File, error) {
	if db.channelDB == nil {
		return nil, fmt.Errorf("channel database is not opened")
	}
	return db.FileRepo().GetByID(fileID)
}

// GetFiles 获取文件列表
func (db *Database) GetFiles(channelID string, limit, offset int) ([]*models.File, error) {
	if db.channelDB == nil {
		return nil, fmt.Errorf("channel database is not opened")
	}
	return db.FileRepo().GetByChannelID(channelID, limit, offset)
}

// ==================== 题目（Challenge） ====================

// CreateChallenge 创建题目
func (db *Database) CreateChallenge(ch *models.Challenge) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.ChallengeRepo().Create(ch)
}

// GetChallenges 获取题目列表
func (db *Database) GetChallenges(channelID string) ([]*models.Challenge, error) {
	if db.channelDB == nil {
		return nil, fmt.Errorf("channel database is not opened")
	}
	return db.ChallengeRepo().GetByChannelID(channelID)
}

// AssignChallenge 分配题目
func (db *Database) AssignChallenge(assignment *models.ChallengeAssignment) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.ChallengeRepo().AssignChallenge(assignment)
}

// SubmitFlag 提交 Flag
func (db *Database) SubmitFlag(submission *models.ChallengeSubmission) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.ChallengeRepo().SubmitFlag(submission)
}

// UpdateProgress 更新进度
func (db *Database) UpdateProgress(progress *models.ChallengeProgress) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.ChallengeRepo().UpdateProgress(progress)
}

// ==================== 审计日志（Audit） ====================

// SaveAuditLog 保存审计日志
func (db *Database) SaveAuditLog(log *models.AuditLog) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.AuditRepo().Log(log)
}

// GetAuditLogs 获取审计日志
func (db *Database) GetAuditLogs(channelID string, limit, offset int) ([]*models.AuditLog, error) {
	if db.channelDB == nil {
		return nil, fmt.Errorf("channel database is not opened")
	}
	return db.AuditRepo().GetByChannelID(channelID, limit, offset)
}

// ==================== 管理（禁言/置顶） ====================

// MuteMember 禁言成员
func (db *Database) MuteMember(record *models.MuteRecord) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.MemberRepo().MuteMember(record)
}

// UnmuteMember 解除禁言
func (db *Database) UnmuteMember(memberID, unmutedBy string) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.MemberRepo().UnmuteMember(memberID, unmutedBy)
}

// IsMuted 检查是否被禁言
func (db *Database) IsMuted(memberID string) (bool, error) {
	if db.channelDB == nil {
		return false, fmt.Errorf("channel database is not opened")
	}
	return db.MemberRepo().IsMuted(memberID)
}

// PinMessage 置顶消息
func (db *Database) PinMessage(channelID, messageID, pinnedBy, reason string) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.ChannelRepo().PinMessage(channelID, messageID, pinnedBy, reason)
}

// UnpinMessage 取消置顶
func (db *Database) UnpinMessage(channelID, messageID string) error {
	if db.channelDB == nil {
		return fmt.Errorf("channel database is not opened")
	}
	return db.ChannelRepo().UnpinMessage(channelID, messageID)
}

// GetPinnedMessages 获取置顶消息
func (db *Database) GetPinnedMessages(channelID string) ([]*models.PinnedMessage, error) {
	if db.channelDB == nil {
		return nil, fmt.Errorf("channel database is not opened")
	}
	return db.ChannelRepo().GetPinnedMessages(channelID)
}

// ==================== 缓存（Cache） ====================

// SaveCache 保存缓存（带过期时间）
func (db *Database) SaveCache(key string, value []byte, ttl time.Duration) error {
	if db.cacheDB == nil {
		return fmt.Errorf("cache database is not opened")
	}
	entry := &models.CacheEntry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
	return db.cacheDB.Save(entry).Error
}

// GetCache 获取缓存（返回值, 是否命中, 错误）
func (db *Database) GetCache(key string) ([]byte, bool, error) {
	if db.cacheDB == nil {
		return nil, false, fmt.Errorf("cache database is not opened")
	}
	var entry models.CacheEntry
	err := db.cacheDB.Where("key = ?", key).First(&entry).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if entry.IsExpired() {
		// 过期即视为未命中并尝试清理
		_ = db.cacheDB.Where("key = ?", key).Delete(&models.CacheEntry{}).Error
		return nil, false, nil
	}
	return entry.Value, true, nil
}

// CleanExpiredCache 清理过期缓存
func (db *Database) CleanExpiredCache() error {
	if db.cacheDB == nil {
		return fmt.Errorf("cache database is not opened")
	}
	now := time.Now()
	return db.cacheDB.Where("expires_at < ?", now).Delete(&models.CacheEntry{}).Error
}
