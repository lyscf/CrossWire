package server

import (
	"fmt"
	"sync"
	"time"

	"crosswire/internal/events"
	"crosswire/internal/models"
)

// OfflineManager 离线消息管理器
// 参考: internal/client/offline_queue.go 的客户端实现
// 参考: docs/ARCHITECTURE.md - 3.1.2 服务端模块 - MessageRouter
type OfflineManager struct {
	server *Server

	// 离线消息队列: memberID -> messages
	offlineQueue map[string][]*models.Message
	mutex        sync.RWMutex

	// 配置
	maxQueueSize int           // 每个成员最多存储的离线消息数
	maxAge       time.Duration // 离线消息最长保留时间

	// 统计
	stats OfflineManagerStats
}

// OfflineManagerStats 离线消息统计
type OfflineManagerStats struct {
	TotalQueued        uint64 // 总入队消息数
	TotalDelivered     uint64 // 总投递消息数
	TotalExpired       uint64 // 总过期消息数
	CurrentQueuedCount int    // 当前队列中的消息数
	mutex              sync.RWMutex
}

// NewOfflineManager 创建离线消息管理器
func NewOfflineManager(server *Server) *OfflineManager {
	return &OfflineManager{
		server:       server,
		offlineQueue: make(map[string][]*models.Message),
		maxQueueSize: 1000,               // 每个成员最多1000条离线消息
		maxAge:       7 * 24 * time.Hour, // 7天
	}
}

// Start 启动离线消息管理器
func (om *OfflineManager) Start() error {
	om.server.logger.Info("[OfflineManager] Starting...")

	// 启动定期清理过期消息
	om.server.wg.Add(1)
	go om.cleanupWorker()

	om.server.logger.Info("[OfflineManager] Started successfully")
	return nil
}

// Stop 停止离线消息管理器
func (om *OfflineManager) Stop() error {
	om.server.logger.Info("[OfflineManager] Stopping...")
	// cleanup worker 会通过 server.ctx.Done() 停止
	return nil
}

// StoreOfflineMessage 存储离线消息
// 参考: docs/ARCHITECTURE.md - MessageRouter 的离线消息队列功能
func (om *OfflineManager) StoreOfflineMessage(memberID string, msg *models.Message) error {
	if !om.server.config.EnableOffline {
		return fmt.Errorf("offline messages disabled")
	}

	om.mutex.Lock()
	defer om.mutex.Unlock()

	// 1. 获取成员的队列
	queue := om.offlineQueue[memberID]

	// 2. 检查队列大小限制
	if len(queue) >= om.maxQueueSize {
		// 删除最旧的消息
		om.server.logger.Warn("[OfflineManager] Queue full for member %s, dropping oldest message", memberID)
		queue = queue[1:]

		om.stats.mutex.Lock()
		om.stats.TotalExpired++
		om.stats.mutex.Unlock()
	}

	// 3. 添加到队列
	queue = append(queue, msg)
	om.offlineQueue[memberID] = queue

	// 4. 持久化到数据库（确保不丢失）
	if err := om.server.messageRepo.Create(msg); err != nil {
		om.server.logger.Error("[OfflineManager] Failed to persist offline message: %v", err)
		return fmt.Errorf("failed to persist message: %w", err)
	}

	// 5. 更新统计
	om.stats.mutex.Lock()
	om.stats.TotalQueued++
	om.stats.CurrentQueuedCount = om.calculateTotalQueued()
	om.stats.mutex.Unlock()

	om.server.logger.Debug("[OfflineManager] Message queued for offline member %s (queue size: %d)",
		memberID, len(queue))

	return nil
}

// DeliverOfflineMessages 投递离线消息给上线的成员
// 当成员重新上线时调用
func (om *OfflineManager) DeliverOfflineMessages(memberID string) error {
	om.mutex.Lock()
	queue := om.offlineQueue[memberID]
	delete(om.offlineQueue, memberID)
	om.mutex.Unlock()

	if len(queue) == 0 {
		om.server.logger.Debug("[OfflineManager] No offline messages for member: %s", memberID)
		return nil
	}

	om.server.logger.Info("[OfflineManager] Delivering %d offline messages to member: %s",
		len(queue), memberID)

	// 批量发送离线消息
	successCount := 0
	for _, msg := range queue {
		// 通过广播管理器发送（会签名）
		if err := om.server.broadcastManager.Broadcast(msg); err != nil {
			om.server.logger.Error("[OfflineManager] Failed to deliver offline message %s: %v",
				msg.ID, err)
			// 继续尝试发送其他消息
			continue
		}
		successCount++

		// 小延迟，避免淹没接收方
		time.Sleep(10 * time.Millisecond)
	}

	// 更新统计
	om.stats.mutex.Lock()
	om.stats.TotalDelivered += uint64(successCount)
	om.stats.CurrentQueuedCount = om.calculateTotalQueued()
	om.stats.mutex.Unlock()

	// 发布离线消息投递完成事件（使用系统连接事件作为通用系统消息）
	om.server.eventBus.Publish(events.EventSystemConnected, &events.SystemEvent{
		Type:    "offline_messages_delivered",
		Message: fmt.Sprintf("Delivered %d/%d offline messages", successCount, len(queue)),
		Data: map[string]interface{}{
			"member_id": memberID,
			"total":     len(queue),
			"delivered": successCount,
			"failed":    len(queue) - successCount,
		},
	})

	om.server.logger.Info("[OfflineManager] Delivered %d/%d offline messages to member: %s",
		successCount, len(queue), memberID)

	return nil
}

// GetOfflineMessageCount 获取成员的离线消息数量
func (om *OfflineManager) GetOfflineMessageCount(memberID string) int {
	om.mutex.RLock()
	defer om.mutex.RUnlock()

	return len(om.offlineQueue[memberID])
}

// GetOfflineMessages 获取成员的离线消息列表（不删除）
func (om *OfflineManager) GetOfflineMessages(memberID string) []*models.Message {
	om.mutex.RLock()
	defer om.mutex.RUnlock()

	queue := om.offlineQueue[memberID]
	// 返回副本
	messages := make([]*models.Message, len(queue))
	copy(messages, queue)

	return messages
}

// ClearOfflineMessages 清除指定成员的离线消息
func (om *OfflineManager) ClearOfflineMessages(memberID string) {
	om.mutex.Lock()
	defer om.mutex.Unlock()

	count := len(om.offlineQueue[memberID])
	delete(om.offlineQueue, memberID)

	om.stats.mutex.Lock()
	om.stats.CurrentQueuedCount = om.calculateTotalQueued()
	om.stats.mutex.Unlock()

	om.server.logger.Debug("[OfflineManager] Cleared %d offline messages for member: %s",
		count, memberID)
}

// CleanupOldMessages 清理过期的离线消息
func (om *OfflineManager) CleanupOldMessages(maxAge time.Duration) int {
	om.mutex.Lock()
	defer om.mutex.Unlock()

	expiredCount := 0
	cutoff := time.Now().Add(-maxAge)

	for memberID, queue := range om.offlineQueue {
		validMessages := make([]*models.Message, 0, len(queue))

		for _, msg := range queue {
			if msg.Timestamp.After(cutoff) {
				validMessages = append(validMessages, msg)
			} else {
				expiredCount++
			}
		}

		if len(validMessages) == 0 {
			delete(om.offlineQueue, memberID)
		} else {
			om.offlineQueue[memberID] = validMessages
		}
	}

	if expiredCount > 0 {
		om.stats.mutex.Lock()
		om.stats.TotalExpired += uint64(expiredCount)
		om.stats.CurrentQueuedCount = om.calculateTotalQueued()
		om.stats.mutex.Unlock()

		om.server.logger.Info("[OfflineManager] Cleaned up %d expired offline messages", expiredCount)
	}

	return expiredCount
}

// cleanupWorker 定期清理过期消息的工作协程
func (om *OfflineManager) cleanupWorker() {
	defer om.server.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // 每小时清理一次
	defer ticker.Stop()

	for {
		select {
		case <-om.server.ctx.Done():
			om.server.logger.Info("[OfflineManager] Cleanup worker stopped")
			return

		case <-ticker.C:
			om.CleanupOldMessages(om.maxAge)
		}
	}
}

// GetStats 获取统计信息
func (om *OfflineManager) GetStats() OfflineManagerStats {
	om.stats.mutex.RLock()
	defer om.stats.mutex.RUnlock()

	// 返回副本（避免复制mutex）
	return OfflineManagerStats{
		TotalQueued:        om.stats.TotalQueued,
		TotalDelivered:     om.stats.TotalDelivered,
		TotalExpired:       om.stats.TotalExpired,
		CurrentQueuedCount: om.stats.CurrentQueuedCount,
	}
}

// calculateTotalQueued 计算当前总队列消息数（需持有mutex）
func (om *OfflineManager) calculateTotalQueued() int {
	total := 0
	for _, queue := range om.offlineQueue {
		total += len(queue)
	}
	return total
}

// GetAllQueuedMembers 获取所有有离线消息的成员ID列表
func (om *OfflineManager) GetAllQueuedMembers() []string {
	om.mutex.RLock()
	defer om.mutex.RUnlock()

	members := make([]string, 0, len(om.offlineQueue))
	for memberID := range om.offlineQueue {
		members = append(members, memberID)
	}

	return members
}

// IsQueueFull 检查指定成员的队列是否已满
func (om *OfflineManager) IsQueueFull(memberID string) bool {
	om.mutex.RLock()
	defer om.mutex.RUnlock()

	return len(om.offlineQueue[memberID]) >= om.maxQueueSize
}
