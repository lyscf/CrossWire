package client

import (
	"crosswire/internal/events"
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
	lastSyncTime      time.Time
	lastMessageID     string
	lastMemberSync    time.Time
	lastChallengeSync time.Time
	// 请求关联
	lastRequestID string
	isSyncing     bool
	syncMutex     sync.Mutex
	pendingSync   bool

	// 自适应同步配置
	currentSyncInterval time.Duration
	minSyncInterval     time.Duration
	maxSyncInterval     time.Duration
	consecutiveFailures int
	lastFailureTime     time.Time

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
	// 获取配置的同步间隔
	baseInterval := client.config.SyncInterval
	if baseInterval == 0 {
		baseInterval = 30 * time.Second // 默认30秒
	}

	return &SyncManager{
		client: client,
		// 自适应同步配置
		currentSyncInterval: baseInterval,
		minSyncInterval:     5 * time.Second, // 最小5秒
		maxSyncInterval:     5 * time.Minute, // 最大5分钟
		consecutiveFailures: 0,
	}
}

// Start 启动同步管理器
func (sm *SyncManager) Start() error {
	sm.client.logger.Info("[SyncManager] Starting sync manager...")

	// 启动定期同步
	go sm.periodicSync()

	// 启动后立即触发一次同步，避免等待周期
	go sm.TriggerSync()

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

// ForceSync 强制立即同步（忽略当前间隔）
func (sm *SyncManager) ForceSync() {
	sm.client.logger.Info("[SyncManager] Force sync triggered")
	sm.TriggerSync()
}

// ResetSyncInterval 重置同步间隔到默认值
func (sm *SyncManager) ResetSyncInterval() {
	sm.syncMutex.Lock()
	defer sm.syncMutex.Unlock()

	baseInterval := sm.client.config.SyncInterval
	if baseInterval == 0 {
		baseInterval = 30 * time.Second
	}

	sm.currentSyncInterval = baseInterval
	sm.consecutiveFailures = 0
	sm.client.logger.Info("[SyncManager] Sync interval reset to %v", sm.currentSyncInterval)
}

// GetSyncStats 获取同步统计信息
func (sm *SyncManager) GetSyncStats() map[string]interface{} {
	sm.stats.mutex.RLock()
	defer sm.stats.mutex.RUnlock()

	sm.syncMutex.Lock()
	currentInterval := sm.currentSyncInterval
	consecutiveFailures := sm.consecutiveFailures
	sm.syncMutex.Unlock()

	return map[string]interface{}{
		"total_syncs":          sm.stats.TotalSyncs,
		"successful_syncs":     sm.stats.SuccessfulSyncs,
		"failed_syncs":         sm.stats.FailedSyncs,
		"messages_synced":      sm.stats.MessagesSynced,
		"members_synced":       sm.stats.MembersSynced,
		"last_sync_time":       sm.stats.LastSyncTime,
		"last_sync_duration":   sm.stats.LastSyncDuration,
		"current_interval":     currentInterval,
		"consecutive_failures": consecutiveFailures,
	}
}

// periodicSync 定期同步（自适应间隔）
func (sm *SyncManager) periodicSync() {
	for {
		select {
		case <-sm.client.ctx.Done():
			return

		case <-time.After(sm.getCurrentSyncInterval()):
			sm.TriggerSync()
		}
	}
}

// getCurrentSyncInterval 获取当前同步间隔
func (sm *SyncManager) getCurrentSyncInterval() time.Duration {
	sm.syncMutex.Lock()
	defer sm.syncMutex.Unlock()
	return sm.currentSyncInterval
}

// adjustSyncInterval 根据同步结果调整同步间隔
func (sm *SyncManager) adjustSyncInterval(success bool) {
	sm.syncMutex.Lock()
	defer sm.syncMutex.Unlock()

	if success {
		// 同步成功，重置失败计数，逐渐缩短间隔
		sm.consecutiveFailures = 0
		if sm.currentSyncInterval > sm.minSyncInterval {
			// 每次成功减少10%的间隔，但不少于最小值
			newInterval := time.Duration(float64(sm.currentSyncInterval) * 0.9)
			if newInterval < sm.minSyncInterval {
				newInterval = sm.minSyncInterval
			}
			sm.currentSyncInterval = newInterval
			sm.client.logger.Debug("[SyncManager] Sync interval decreased to %v", sm.currentSyncInterval)
		}
	} else {
		// 同步失败，增加失败计数，延长间隔
		sm.consecutiveFailures++
		sm.lastFailureTime = time.Now()

		// 根据连续失败次数指数退避
		multiplier := 1 << uint(sm.consecutiveFailures) // 2^n
		if multiplier > 16 {
			multiplier = 16 // 最大16倍
		}

		newInterval := time.Duration(multiplier) * sm.minSyncInterval
		if newInterval > sm.maxSyncInterval {
			newInterval = sm.maxSyncInterval
		}

		if newInterval != sm.currentSyncInterval {
			sm.currentSyncInterval = newInterval
			sm.client.logger.Warn("[SyncManager] Sync failed %d times, interval increased to %v",
				sm.consecutiveFailures, sm.currentSyncInterval)
		}
	}
}

// doSync 执行同步
func (sm *SyncManager) doSync() {
	success := false
	defer func() {
		// 根据同步结果调整间隔
		sm.adjustSyncInterval(success)

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
	reqID := generateRequestID()
	syncReq := map[string]interface{}{
		"type":            "sync.request",
		"channel_id":      sm.client.config.ChannelID,
		"member_id":       sm.client.memberID,
		"last_message_id": sm.lastMessageID,
		"last_timestamp":  sm.lastSyncTime.Unix(),
		"limit":           sm.client.config.MaxSyncMessages,
		"request_id":      reqID,
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

	// 标记同步成功（发送成功）
	success = true

	// 4. 记录本次请求ID，用于响应关联
	sm.syncMutex.Lock()
	sm.lastRequestID = reqID
	sm.syncMutex.Unlock()

	// 5. 此处不再用当前时间覆盖 lastSyncTime，
	//    让 processSyncMessages 基于消息时间推进水位
	duration := time.Since(startTime)

	sm.stats.mutex.Lock()
	sm.stats.SuccessfulSyncs++
	// 记录本次同步完成时间到统计，但不影响消息水位
	sm.stats.LastSyncTime = time.Now()
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

	// 关联ID校验（若存在）
	if rid, _ := response["request_id"].(string); rid != "" {
		sm.syncMutex.Lock()
		ok := (rid == sm.lastRequestID)
		sm.syncMutex.Unlock()
		if !ok {
			sm.client.logger.Debug("[SyncManager] Ignoring out-of-date sync response: %s", rid)
			return
		}
	}

	// 0. 处理主频道信息，确保本地有真实记录
	if chObj, ok := response["channel"].(map[string]interface{}); ok {
		sm.processSyncChannel(chObj)
	}

	// 1. 处理消息
	if messagesData, ok := response["messages"].([]interface{}); ok {
		sm.processSyncMessages(messagesData)
	}

	// 2. 处理成员
	if membersData, ok := response["members"].([]interface{}); ok {
		sm.processSyncMembers(membersData)
	}

	// 2.5 处理挑战（最小可用：全量覆盖/增量更新本地）
	if challengesData, ok := response["challenges"].([]interface{}); ok {
		sm.processSyncChallenges(challengesData)
	}

	// 2.6 处理子频道列表
	if subsData, ok := response["sub_channels"].([]interface{}); ok {
		sm.processSyncSubChannels(subsData)
	}

	// 3. 检查是否有更多数据
	hasMore, _ := response["has_more"].(bool)
	if hasMore {
		// 触发下一次同步（小延迟减少争用）
		sm.client.logger.Debug("[SyncManager] More data available, triggering next sync")
		time.AfterFunc(30*time.Millisecond, func() { sm.TriggerSync() })
	}

	sm.client.logger.Info("[SyncManager] Sync response processed")
}

// processSyncChannel 处理主频道信息（避免占位）
func (sm *SyncManager) processSyncChannel(chObj map[string]interface{}) {
	raw, err := json.Marshal(chObj)
	if err != nil {
		return
	}
	var ch models.Channel
	if err := json.Unmarshal(raw, &ch); err != nil {
		sm.client.logger.Warn("[SyncManager] Failed to unmarshal channel: %v", err)
		return
	}
	// 只接受当前配置频道
	if ch.ID == "" || ch.ID != sm.client.GetChannelID() {
		return
	}
	if existing, err := sm.client.channelRepo.GetByID(ch.ID); err == nil && existing != nil {
		ch.CreatedAt = existing.CreatedAt
		if err := sm.client.channelRepo.Update(&ch); err != nil {
			sm.client.logger.Warn("[SyncManager] Failed to update channel %s: %v", ch.ID, err)
		}
	} else {
		if err := sm.client.channelRepo.Create(&ch); err != nil {
			sm.client.logger.Warn("[SyncManager] Failed to create channel %s: %v", ch.ID, err)
		}
	}
}

// processSyncSubChannels 处理同步的子频道列表
func (sm *SyncManager) processSyncSubChannels(subsData []interface{}) {
	sm.client.logger.Debug("[SyncManager] Processing %d sub-channels", len(subsData))

	for _, chData := range subsData {
		m, ok := chData.(map[string]interface{})
		if !ok {
			continue
		}

		raw, err := json.Marshal(m)
		if err != nil {
			continue
		}

		var ch models.Channel
		if err := json.Unmarshal(raw, &ch); err != nil {
			sm.client.logger.Warn("[SyncManager] Failed to unmarshal sub-channel: %v", err)
			continue
		}

		// 只处理当前主频道下的子频道
		if ch.ParentChannelID == "" || ch.ParentChannelID != sm.client.GetChannelID() {
			continue
		}

		// 入库（存在则更新，不存在则创建）
		if existing, err := sm.client.channelRepo.GetByID(ch.ID); err == nil && existing != nil {
			ch.CreatedAt = existing.CreatedAt
			if err := sm.client.channelRepo.Update(&ch); err != nil {
				sm.client.logger.Warn("[SyncManager] Failed to update sub-channel %s: %v", ch.ID, err)
			}
		} else {
			if err := sm.client.channelRepo.Create(&ch); err != nil {
				sm.client.logger.Warn("[SyncManager] Failed to create sub-channel %s: %v", ch.ID, err)
			}
		}
	}
}

// processSyncMessages 处理同步的消息
func (sm *SyncManager) processSyncMessages(messagesData []interface{}) {
	sm.client.logger.Debug("[SyncManager] Processing %d synced messages", len(messagesData))

	var syncedCount uint64
	// 跟踪本批次最大时间戳与对应消息ID，用于推进水位
	maxTimestamp := sm.lastSyncTime
	maxMsgID := sm.lastMessageID

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
					// 发布更新事件供前端刷新
					sm.client.eventBus.Publish(events.EventMessageUpdated, &events.MessageEvent{
						Message:   &msg,
						ChannelID: msg.ChannelID,
						SenderID:  msg.SenderID,
					})
				}
			}
		} else {
			// 不存在，插入
			if err := sm.client.messageRepo.Create(&msg); err != nil {
				sm.client.logger.Warn("[SyncManager] Failed to save message: %v", err)
			} else {
				syncedCount++
				// 发布接收事件供前端刷新
				sm.client.eventBus.Publish(events.EventMessageReceived, &events.MessageEvent{
					Message:   &msg,
					ChannelID: msg.ChannelID,
					SenderID:  msg.SenderID,
				})
			}
		}

		// 推进水位：按时间戳最大（若相等按ID最大）
		if msg.Timestamp.After(maxTimestamp) || (msg.Timestamp.Equal(maxTimestamp) && msg.ID > maxMsgID) {
			maxTimestamp = msg.Timestamp
			maxMsgID = msg.ID
		}
	}

	// 批量处理完成后，统一更新水位与lastMessageID
	if maxTimestamp.After(sm.lastSyncTime) {
		sm.lastSyncTime = maxTimestamp
		sm.lastMessageID = maxMsgID
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

	// 追踪本批次增量水位（使用 LastSeenAt/LastHeartbeat/JoinedAt 最大值）
	maxTs := sm.lastMemberSync
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

		// 详细调试日志：即将同步的成员信息
		sm.client.logger.Debug(
			"[SyncManager] Upserting member id=%s channel_id=%s nickname=%s role=%s status=%s",
			member.ID, member.ChannelID, member.Nickname, string(member.Role), string(member.Status),
		)

		// 在写入前检查频道占位是否存在
		if member.ChannelID == "" {
			sm.client.logger.Warn("[SyncManager] Member %s missing channel_id; skipping", member.ID)
			continue
		}

		// 检查是否已存在
		existing, err := sm.client.memberRepo.GetByID(member.ID)
		if err == nil && existing != nil {
			// 更新
			if err := sm.client.memberRepo.Update(&member); err != nil {
				sm.client.logger.Warn("[SyncManager] Failed to update member id=%s channel_id=%s: %v", member.ID, member.ChannelID, err)
			} else {
				syncedCount++
			}
		} else {
			// 插入
			if err := sm.client.memberRepo.Create(&member); err != nil {
				// 失败时，补充更多上下文：确认频道是否存在
				_, chErr := sm.client.channelRepo.GetByID(member.ChannelID)
				if chErr != nil {
					sm.client.logger.Warn("[SyncManager] Failed to add member id=%s channel_id=%s: %v (channel missing: %v)", member.ID, member.ChannelID, err, chErr)
				} else {
					sm.client.logger.Warn("[SyncManager] Failed to add member id=%s channel_id=%s: %v (channel exists)", member.ID, member.ChannelID, err)
				}
			} else {
				syncedCount++
			}
		}

		// 推进成员水位
		t := member.JoinedAt
		if member.LastSeenAt.After(t) {
			t = member.LastSeenAt
		}
		if member.LastHeartbeat.After(t) {
			t = member.LastHeartbeat
		}
		if t.After(maxTs) {
			maxTs = t
		}
	}

	sm.stats.mutex.Lock()
	sm.stats.MembersSynced += syncedCount
	sm.stats.mutex.Unlock()

	// 更新成员增量水位
	if maxTs.After(sm.lastMemberSync) {
		sm.lastMemberSync = maxTs
	}

	sm.client.logger.Info("[SyncManager] Synced %d members", syncedCount)
}

// processSyncChallenges 处理同步的挑战数据
func (sm *SyncManager) processSyncChallenges(challengesData []interface{}) {
	sm.client.logger.Debug("[SyncManager] Processing %d synced challenges", len(challengesData))

	for _, chData := range challengesData {
		m, ok := chData.(map[string]interface{})
		if !ok {
			continue
		}

		// 转换为JSON再反序列化
		raw, err := json.Marshal(m)
		if err != nil {
			continue
		}
		var ch models.Challenge
		if err := json.Unmarshal(raw, &ch); err != nil {
			sm.client.logger.Warn("[SyncManager] Failed to unmarshal challenge: %v", err)
			continue
		}

		// 确保频道ID存在（兼容旧服务端或系统消息生成的占位记录）
		if ch.ChannelID == "" {
			ch.ChannelID = sm.client.GetChannelID()
		}

		// 详细日志：入库前打印关键信息
		sm.client.logger.Debug("[SyncManager] Upserting challenge id=%s title=%s points=%d status=%s", ch.ID, ch.Title, ch.Points, ch.Status)

		// 入库（存在则更新，不存在则创建）
		if existing, err := sm.client.challengeRepo.GetByID(ch.ID); err == nil && existing != nil {
			ch.CreatedAt = existing.CreatedAt // 保持创建时间
			if err := sm.client.challengeRepo.Update(&ch); err != nil {
				sm.client.logger.Warn("[SyncManager] Failed to update challenge id=%s: %v", ch.ID, err)
			} else {
				// 发布更新事件（直接使用 ChallengeEvent 作为数据）
				sm.client.eventBus.Publish(events.EventChallengeUpdated, &events.ChallengeEvent{
					Challenge: &ch,
					Action:    "updated",
					UserID:    "",
					ChannelID: sm.client.GetChannelID(),
					ExtraData: nil,
				})
				sm.client.logger.Debug("[SyncManager] Challenge updated id=%s", ch.ID)
			}
		} else {
			if err := sm.client.challengeRepo.Create(&ch); err != nil {
				sm.client.logger.Warn("[SyncManager] Failed to create challenge id=%s: %v", ch.ID, err)
			} else {
				// 发布创建事件（直接使用 ChallengeEvent 作为数据）
				sm.client.eventBus.Publish(events.EventChallengeCreated, &events.ChallengeEvent{
					Challenge: &ch,
					Action:    "created",
					UserID:    "",
					ChannelID: sm.client.GetChannelID(),
					ExtraData: nil,
				})
				sm.client.logger.Debug("[SyncManager] Challenge created id=%s", ch.ID)
			}
		}
	}
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

// generateRequestID 生成同步请求关联ID
func generateRequestID() string {
	return time.Now().Format("20060102150405.000000000")
}
