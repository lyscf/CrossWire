package client

import (
	"fmt"
	"sync"
	"time"

	"crosswire/internal/models"
)

// CacheManager 缓存管理器
// 参考: docs/ARCHITECTURE.md - 3.1.3 客户端模块 - CacheManager
// 职责：消息缓存、文件缓存、成员缓存
type CacheManager struct {
	client *Client

	// 消息缓存
	messageCache map[string]*models.Message // messageID -> Message
	messageMutex sync.RWMutex
	maxMessages  int

	// 成员缓存
	memberCache map[string]*models.Member // memberID -> Member
	memberMutex sync.RWMutex

	// 文件缓存
	fileCache map[string]*models.File // fileID -> File
	fileMutex sync.RWMutex

	// 统计
	stats CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
	MessageCacheSize uint64
	MemberCacheSize  uint64
	FileCacheSize    uint64
	CacheHits        uint64
	CacheMisses      uint64
	mutex            sync.RWMutex
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(client *Client) *CacheManager {
	return &CacheManager{
		client:       client,
		messageCache: make(map[string]*models.Message),
		memberCache:  make(map[string]*models.Member),
		fileCache:    make(map[string]*models.File),
		maxMessages:  client.config.CacheSize,
	}
}

// Start 启动缓存管理器
func (cm *CacheManager) Start() error {
	cm.client.logger.Info("[CacheManager] Starting cache manager...")

	// 预加载最近的消息
	if err := cm.preloadMessages(); err != nil {
		cm.client.logger.Warn("[CacheManager] Failed to preload messages: %v", err)
	}

	// 预加载成员列表
	if err := cm.preloadMembers(); err != nil {
		cm.client.logger.Warn("[CacheManager] Failed to preload members: %v", err)
	}

	// 启动清理协程
	go cm.periodicCleanup()

	cm.client.logger.Info("[CacheManager] Cache manager started")

	return nil
}

// Stop 停止缓存管理器
func (cm *CacheManager) Stop() {
	cm.client.logger.Info("[CacheManager] Stopping cache manager...")

	// 清理缓存
	cm.messageMutex.Lock()
	cm.messageCache = make(map[string]*models.Message)
	cm.messageMutex.Unlock()

	cm.memberMutex.Lock()
	cm.memberCache = make(map[string]*models.Member)
	cm.memberMutex.Unlock()

	cm.fileMutex.Lock()
	cm.fileCache = make(map[string]*models.File)
	cm.fileMutex.Unlock()

	cm.client.logger.Info("[CacheManager] Cache manager stopped")
}

// preloadMessages 预加载最近的消息
func (cm *CacheManager) preloadMessages() error {
	cm.client.logger.Debug("[CacheManager] Preloading messages...")

	messages, err := cm.client.messageRepo.GetByChannelID(cm.client.config.ChannelID, cm.maxMessages, 0)
	if err != nil {
		return fmt.Errorf("failed to load messages: %w", err)
	}

	cm.messageMutex.Lock()
	defer cm.messageMutex.Unlock()

	for _, msg := range messages {
		cm.messageCache[msg.ID] = msg
	}

	cm.stats.mutex.Lock()
	cm.stats.MessageCacheSize = uint64(len(cm.messageCache))
	cm.stats.mutex.Unlock()

	cm.client.logger.Info("[CacheManager] Preloaded %d messages", len(messages))

	return nil
}

// preloadMembers 预加载成员列表
func (cm *CacheManager) preloadMembers() error {
	cm.client.logger.Debug("[CacheManager] Preloading members...")

	members, err := cm.client.memberRepo.GetByChannelID(cm.client.config.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to load members: %w", err)
	}

	cm.memberMutex.Lock()
	defer cm.memberMutex.Unlock()

	for _, member := range members {
		cm.memberCache[member.ID] = member
	}

	cm.stats.mutex.Lock()
	cm.stats.MemberCacheSize = uint64(len(cm.memberCache))
	cm.stats.mutex.Unlock()

	cm.client.logger.Info("[CacheManager] Preloaded %d members", len(members))

	return nil
}

// GetMessage 从缓存获取消息
func (cm *CacheManager) GetMessage(messageID string) (*models.Message, error) {
	// 先从缓存查询
	cm.messageMutex.RLock()
	if msg, exists := cm.messageCache[messageID]; exists {
		cm.messageMutex.RUnlock()

		cm.stats.mutex.Lock()
		cm.stats.CacheHits++
		cm.stats.mutex.Unlock()

		return msg, nil
	}
	cm.messageMutex.RUnlock()

	// 缓存未命中，从数据库查询
	cm.stats.mutex.Lock()
	cm.stats.CacheMisses++
	cm.stats.mutex.Unlock()

	msg, err := cm.client.messageRepo.GetByID(messageID)
	if err != nil {
		return nil, err
	}

	// 加入缓存
	cm.PutMessage(msg)

	return msg, nil
}

// PutMessage 将消息放入缓存
func (cm *CacheManager) PutMessage(msg *models.Message) {
	if msg == nil {
		return
	}

	cm.messageMutex.Lock()
	defer cm.messageMutex.Unlock()

	// 如果缓存已满，删除最旧的消息
	if len(cm.messageCache) >= cm.maxMessages {
		cm.evictOldestMessage()
	}

	cm.messageCache[msg.ID] = msg

	cm.stats.mutex.Lock()
	cm.stats.MessageCacheSize = uint64(len(cm.messageCache))
	cm.stats.mutex.Unlock()
}

// evictOldestMessage 移除最旧的消息（需要持有锁）
func (cm *CacheManager) evictOldestMessage() {
	var oldestID string
	var oldestTime time.Time

	for id, msg := range cm.messageCache {
		if oldestID == "" || msg.Timestamp.Before(oldestTime) {
			oldestID = id
			oldestTime = msg.Timestamp
		}
	}

	if oldestID != "" {
		delete(cm.messageCache, oldestID)
	}
}

// GetMember 从缓存获取成员
func (cm *CacheManager) GetMember(memberID string) (*models.Member, error) {
	// 先从缓存查询
	cm.memberMutex.RLock()
	if member, exists := cm.memberCache[memberID]; exists {
		cm.memberMutex.RUnlock()

		cm.stats.mutex.Lock()
		cm.stats.CacheHits++
		cm.stats.mutex.Unlock()

		return member, nil
	}
	cm.memberMutex.RUnlock()

	// 缓存未命中，从数据库查询
	cm.stats.mutex.Lock()
	cm.stats.CacheMisses++
	cm.stats.mutex.Unlock()

	member, err := cm.client.memberRepo.GetByID(memberID)
	if err != nil {
		return nil, err
	}

	// 加入缓存
	cm.PutMember(member)

	return member, nil
}

// PutMember 将成员放入缓存
func (cm *CacheManager) PutMember(member *models.Member) {
	if member == nil {
		return
	}

	cm.memberMutex.Lock()
	defer cm.memberMutex.Unlock()

	cm.memberCache[member.ID] = member

	cm.stats.mutex.Lock()
	cm.stats.MemberCacheSize = uint64(len(cm.memberCache))
	cm.stats.mutex.Unlock()
}

// RemoveMember 从缓存移除成员
func (cm *CacheManager) RemoveMember(memberID string) {
	cm.memberMutex.Lock()
	defer cm.memberMutex.Unlock()

	delete(cm.memberCache, memberID)

	cm.stats.mutex.Lock()
	cm.stats.MemberCacheSize = uint64(len(cm.memberCache))
	cm.stats.mutex.Unlock()
}

// GetFile 从缓存获取文件
func (cm *CacheManager) GetFile(fileID string) (*models.File, error) {
	// 先从缓存查询
	cm.fileMutex.RLock()
	if file, exists := cm.fileCache[fileID]; exists {
		cm.fileMutex.RUnlock()

		cm.stats.mutex.Lock()
		cm.stats.CacheHits++
		cm.stats.mutex.Unlock()

		return file, nil
	}
	cm.fileMutex.RUnlock()

	// 缓存未命中，从数据库查询
	cm.stats.mutex.Lock()
	cm.stats.CacheMisses++
	cm.stats.mutex.Unlock()

	file, err := cm.client.fileRepo.GetByID(fileID)
	if err != nil {
		return nil, err
	}

	// 加入缓存
	cm.PutFile(file)

	return file, nil
}

// PutFile 将文件放入缓存
func (cm *CacheManager) PutFile(file *models.File) {
	if file == nil {
		return
	}

	cm.fileMutex.Lock()
	defer cm.fileMutex.Unlock()

	cm.fileCache[file.ID] = file

	cm.stats.mutex.Lock()
	cm.stats.FileCacheSize = uint64(len(cm.fileCache))
	cm.stats.mutex.Unlock()
}

// GetMessages 批量获取消息
func (cm *CacheManager) GetMessages(limit int) []*models.Message {
	cm.messageMutex.RLock()
	defer cm.messageMutex.RUnlock()

	messages := make([]*models.Message, 0, limit)
	count := 0

	for _, msg := range cm.messageCache {
		if count >= limit {
			break
		}
		messages = append(messages, msg)
		count++
	}

	return messages
}

// GetMembers 获取所有缓存的成员
func (cm *CacheManager) GetMembers() []*models.Member {
	cm.memberMutex.RLock()
	defer cm.memberMutex.RUnlock()

	members := make([]*models.Member, 0, len(cm.memberCache))
	for _, member := range cm.memberCache {
		members = append(members, member)
	}

	return members
}

// periodicCleanup 定期清理过期缓存
func (cm *CacheManager) periodicCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cm.client.ctx.Done():
			return

		case <-ticker.C:
			cm.cleanupExpiredCache()
		}
	}
}

// cleanupExpiredCache 清理过期缓存
func (cm *CacheManager) cleanupExpiredCache() {
	cm.client.logger.Debug("[CacheManager] Cleaning up expired cache...")

	threshold := time.Now().Add(-cm.client.config.CacheDuration)

	// 清理过期消息
	cm.messageMutex.Lock()
	for id, msg := range cm.messageCache {
		if msg.Timestamp.Before(threshold) {
			delete(cm.messageCache, id)
		}
	}
	messageCount := len(cm.messageCache)
	cm.messageMutex.Unlock()

	cm.stats.mutex.Lock()
	cm.stats.MessageCacheSize = uint64(messageCount)
	cm.stats.mutex.Unlock()

	cm.client.logger.Debug("[CacheManager] Cleanup complete, %d messages remaining", messageCount)
}

// GetStats 获取统计信息
func (cm *CacheManager) GetStats() CacheStats {
	cm.stats.mutex.RLock()
	defer cm.stats.mutex.RUnlock()

	return CacheStats{
		MessageCacheSize: cm.stats.MessageCacheSize,
		MemberCacheSize:  cm.stats.MemberCacheSize,
		FileCacheSize:    cm.stats.FileCacheSize,
		CacheHits:        cm.stats.CacheHits,
		CacheMisses:      cm.stats.CacheMisses,
	}
}

// Clear 清空所有缓存
func (cm *CacheManager) Clear() {
	cm.messageMutex.Lock()
	cm.messageCache = make(map[string]*models.Message)
	cm.messageMutex.Unlock()

	cm.memberMutex.Lock()
	cm.memberCache = make(map[string]*models.Member)
	cm.memberMutex.Unlock()

	cm.fileMutex.Lock()
	cm.fileCache = make(map[string]*models.File)
	cm.fileMutex.Unlock()

	cm.stats.mutex.Lock()
	cm.stats.MessageCacheSize = 0
	cm.stats.MemberCacheSize = 0
	cm.stats.FileCacheSize = 0
	cm.stats.mutex.Unlock()

	cm.client.logger.Info("[CacheManager] Cache cleared")
}
