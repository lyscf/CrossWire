package client

import (
	"encoding/json"
	"sync"
	"time"

	"crosswire/internal/models"
	"crosswire/internal/transport"
)

// SyncManager 同步管理器
// 参考: docs/ARCHITECTURE.md - 3.1.3 客户端模块 - SyncManager
// 职责：增量同步、离线消息、冲突解决
type SyncManager struct {
	client *Client

	// 同步状态
	lastSyncTime  time.Time
	lastMessageID string
	isSyncing     bool
	syncMutex     sync.Mutex
	pendingSync   bool

	// 统计
	stats SyncStats
}

// SyncStats 同步统计
type SyncStats struct {
	TotalSyncs       uint64
	SuccessfulSyncs  uint64
	FailedSyncs      uint64
	MessagesSynced   uint64
	MembersSynced    uint64
	LastSyncTime     time.Time
	LastSyncDuration time.Duration
	mutex            sync.RWMutex
}

// NewSyncManager 创建同步管理器
func NewSyncManager(client *Client) *SyncManager {
	return &SyncManager{
		client: client,
	}
}

// Start 启动同步管理器
func (sm *SyncManager) Start() error {
	sm.client.logger.Info("[SyncManager] Starting sync manager...")

	// 启动定期同步
	go sm.periodicSync()

	sm.client.logger.Info("[SyncManager] Sync manager started")

	return nil
}

// Stop 停止同步管理器
func (sm *SyncManager) Stop() {
	sm.client.logger.Info("[SyncManager] Stopping sync manager...")
	// 上下文取消会自动停止periodicSync

	sm.client.logger.Info("[SyncManager] Sync manager stopped")
}

// TriggerSync 触发一次同步
func (sm *SyncManager) TriggerSync() {
	sm.syncMutex.Lock()
	if sm.isSyncing {
		sm.pendingSync = true
		sm.syncMutex.Unlock()
		return
	}
	sm.isSyncing = true
	sm.syncMutex.Unlock()

	go sm.doSync()
}

// periodicSync 定期同步
func (sm *SyncManager) periodicSync() {
	ticker := time.NewTicker(sm.client.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.client.ctx.Done():
			return

		case <-ticker.C:
			sm.TriggerSync()
		}
	}
}

// doSync 执行同步
func (sm *SyncManager) doSync() {
	defer func() {
		sm.syncMutex.Lock()
		sm.isSyncing = false
		pending := sm.pendingSync
		sm.pendingSync = false
		sm.syncMutex.Unlock()

		if pending {
			go sm.doSync()
		}
	}()

	startTime := time.Now()

	sm.stats.mutex.Lock()
	sm.stats.TotalSyncs++
	sm.stats.mutex.Unlock()

	sm.client.logger.Debug("[SyncManager] Starting sync...")

	// 1. 构造同步请求
	syncReq := map[string]interface{}{
		"type":            "sync.request",
		"channel_id":      sm.client.config.ChannelID,
		"member_id":       sm.client.memberID,
		"last_message_id": sm.lastMessageID,
		"last_timestamp":  sm.lastSyncTime.Unix(),
		"limit":           sm.client.config.MaxSyncMessages,
	}

	// 2. 序列化并加密
	reqJSON, err := json.Marshal(syncReq)
	if err != nil {
		sm.client.logger.Error("[SyncManager] Failed to marshal sync request: %v", err)
		sm.stats.mutex.Lock()
		sm.stats.FailedSyncs++
		sm.stats.mutex.Unlock()
		return
	}

	reqData, err := sm.client.crypto.EncryptMessage(reqJSON)
	if err != nil {
		sm.client.logger.Error("[SyncManager] Failed to encrypt sync request: %v", err)
		sm.stats.mutex.Lock()
		sm.stats.FailedSyncs++
		sm.stats.mutex.Unlock()
		return
	}

	// 3. 发送请求
	msg := &transport.Message{
		Type:      transport.MessageTypeControl,
		SenderID:  sm.client.memberID,
		Payload:   reqData,
		Timestamp: time.Now(),
	}

	if err := sm.client.transport.SendMessage(msg); err != nil {
		sm.client.logger.Error("[SyncManager] Failed to send sync request: %v", err)
		sm.stats.mutex.Lock()
		sm.stats.FailedSyncs++
		sm.stats.mutex.Unlock()
		return
	}

	// 4. 等待响应（通过订阅机制，这里简化处理）
	// TODO: 实现请求-响应匹配机制
	// 暂时假设响应会通过receiveManager的handleSyncResponse处理

	// 5. 更新同步时间
	sm.lastSyncTime = time.Now()

	duration := time.Since(startTime)

	sm.stats.mutex.Lock()
	sm.stats.SuccessfulSyncs++
	sm.stats.LastSyncTime = sm.lastSyncTime
	sm.stats.LastSyncDuration = duration
	sm.stats.mutex.Unlock()

	sm.client.stats.mutex.Lock()
	sm.client.stats.SyncCount++
	sm.client.stats.LastSyncTime = sm.lastSyncTime
	sm.client.stats.mutex.Unlock()

	sm.client.logger.Debug("[SyncManager] Sync completed in %v", duration)
}

// HandleSyncResponse 处理同步响应
func (sm *SyncManager) HandleSyncResponse(data []byte) {
	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		sm.client.logger.Error("[SyncManager] Failed to unmarshal sync response: %v", err)
		return
	}

	// 1. 处理消息
	if messagesData, ok := response["messages"].([]interface{}); ok {
		sm.processSyncMessages(messagesData)
	}

	// 2. 处理成员
	if membersData, ok := response["members"].([]interface{}); ok {
		sm.processSyncMembers(membersData)
	}

	// 3. 检查是否有更多数据
	hasMore, _ := response["has_more"].(bool)
	if hasMore {
		// 触发下一次同步
		sm.client.logger.Debug("[SyncManager] More data available, triggering next sync")
		sm.TriggerSync()
	}

	sm.client.logger.Info("[SyncManager] Sync response processed")
}

// processSyncMessages 处理同步的消息
func (sm *SyncManager) processSyncMessages(messagesData []interface{}) {
	sm.client.logger.Debug("[SyncManager] Processing %d synced messages", len(messagesData))

	var syncedCount uint64

	for _, msgData := range messagesData {
		msgMap, ok := msgData.(map[string]interface{})
		if !ok {
			continue
		}

		// 转换为JSON再反序列化（简化处理）
		msgJSON, err := json.Marshal(msgMap)
		if err != nil {
			continue
		}

		var msg models.Message
		if err := json.Unmarshal(msgJSON, &msg); err != nil {
			sm.client.logger.Warn("[SyncManager] Failed to unmarshal message: %v", err)
			continue
		}

		// 检查是否已存在
		existing, err := sm.client.messageRepo.GetByID(msg.ID)
		if err == nil && existing != nil {
			// 已存在，检查是否需要更新（冲突解决）
			if sm.shouldUpdate(existing, &msg) {
				if err := sm.client.messageRepo.Update(&msg); err != nil {
					sm.client.logger.Warn("[SyncManager] Failed to update message: %v", err)
				} else {
					syncedCount++
				}
			}
		} else {
			// 不存在，插入
			if err := sm.client.messageRepo.Create(&msg); err != nil {
				sm.client.logger.Warn("[SyncManager] Failed to save message: %v", err)
			} else {
				syncedCount++
			}
		}

		// 更新最后消息ID
		if msg.Timestamp.After(sm.lastSyncTime) || sm.lastMessageID == "" {
			sm.lastMessageID = msg.ID
		}
	}

	sm.stats.mutex.Lock()
	sm.stats.MessagesSynced += syncedCount
	sm.stats.mutex.Unlock()

	sm.client.logger.Info("[SyncManager] Synced %d messages", syncedCount)
}

// processSyncMembers 处理同步的成员
func (sm *SyncManager) processSyncMembers(membersData []interface{}) {
	sm.client.logger.Debug("[SyncManager] Processing %d synced members", len(membersData))

	var syncedCount uint64

	for _, memberData := range membersData {
		memberMap, ok := memberData.(map[string]interface{})
		if !ok {
			continue
		}

		// 转换为JSON再反序列化
		memberJSON, err := json.Marshal(memberMap)
		if err != nil {
			continue
		}

		var member models.Member
		if err := json.Unmarshal(memberJSON, &member); err != nil {
			sm.client.logger.Warn("[SyncManager] Failed to unmarshal member: %v", err)
			continue
		}

		// 检查是否已存在
		existing, err := sm.client.memberRepo.GetByID(member.ID)
		if err == nil && existing != nil {
			// 更新
			if err := sm.client.memberRepo.Update(&member); err != nil {
				sm.client.logger.Warn("[SyncManager] Failed to update member: %v", err)
			} else {
				syncedCount++
			}
		} else {
			// 插入
			if err := sm.client.memberRepo.Create(&member); err != nil {
				sm.client.logger.Warn("[SyncManager] Failed to add member: %v", err)
			} else {
				syncedCount++
			}
		}
	}

	sm.stats.mutex.Lock()
	sm.stats.MembersSynced += syncedCount
	sm.stats.mutex.Unlock()

	sm.client.logger.Info("[SyncManager] Synced %d members", syncedCount)
}

// shouldUpdate 判断是否应该更新（冲突解决）
// 参考: docs/PROTOCOL.md - 5.3.2 冲突解决 - Last-Write-Wins
func (sm *SyncManager) shouldUpdate(local, remote *models.Message) bool {
	// 比较时间戳
	if !remote.Timestamp.Equal(local.Timestamp) {
		return remote.Timestamp.After(local.Timestamp)
	}

	// 时间戳相同，比较ID（字典序）
	return remote.ID > local.ID
}

// GetStats 获取统计信息
func (sm *SyncManager) GetStats() SyncStats {
	sm.stats.mutex.RLock()
	defer sm.stats.mutex.RUnlock()

	return SyncStats{
		TotalSyncs:       sm.stats.TotalSyncs,
		SuccessfulSyncs:  sm.stats.SuccessfulSyncs,
		FailedSyncs:      sm.stats.FailedSyncs,
		MessagesSynced:   sm.stats.MessagesSynced,
		MembersSynced:    sm.stats.MembersSynced,
		LastSyncTime:     sm.stats.LastSyncTime,
		LastSyncDuration: sm.stats.LastSyncDuration,
	}
}

// GetLastSyncTime 获取最后同步时间
func (sm *SyncManager) GetLastSyncTime() time.Time {
	return sm.lastSyncTime
}

// GetLastMessageID 获取最后消息ID
func (sm *SyncManager) GetLastMessageID() string {
	return sm.lastMessageID
}
