package client

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/events"
	"crosswire/internal/models"
	"crosswire/internal/transport"
)

// ReceiveManager 接收管理器
// 参考: docs/ARCHITECTURE.md - 3.1.3 客户端模块 - ReceiveManager
// 职责：监听广播帧、解密过滤、消息去重
type ReceiveManager struct {
	client *Client

	// 消息去重
	seenMessages map[string]time.Time // messageID -> timestamp
	seenMutex    sync.RWMutex
	maxSeenSize  int

	// 统计
	stats ReceiveStats
}

// ReceiveStats 接收统计
type ReceiveStats struct {
	TotalReceived     uint64
	ValidMessages     uint64
	DuplicateMessages uint64
	InvalidMessages   uint64
	DecryptFailures   uint64
	mutex             sync.RWMutex
}

// NewReceiveManager 创建接收管理器
func NewReceiveManager(client *Client) *ReceiveManager {
	return &ReceiveManager{
		client:       client,
		seenMessages: make(map[string]time.Time),
		maxSeenSize:  10000, // 最多记录10000条消息ID
	}
}

// Start 启动接收管理器
func (rm *ReceiveManager) Start() error {
	rm.client.logger.Info("[ReceiveManager] Starting receive manager...")

	// 订阅传输层消息
	if err := rm.client.transport.Subscribe(rm.handleTransportMessage); err != nil {
		return fmt.Errorf("failed to subscribe to transport: %w", err)
	}

	// 启动清理协程
	go rm.cleanupSeenMessages()

	rm.client.logger.Info("[ReceiveManager] Receive manager started")

	return nil
}

// Stop 停止接收管理器
func (rm *ReceiveManager) Stop() {
	rm.client.logger.Info("[ReceiveManager] Stopping receive manager...")

	// 清理已见消息记录
	rm.seenMutex.Lock()
	rm.seenMessages = make(map[string]time.Time)
	rm.seenMutex.Unlock()

	rm.client.logger.Info("[ReceiveManager] Receive manager stopped")
}

// handleTransportMessage 处理传输层消息
func (rm *ReceiveManager) handleTransportMessage(msg *transport.Message) {
	rm.stats.mutex.Lock()
	rm.stats.TotalReceived++
	rm.stats.mutex.Unlock()

	// 更新客户端统计
	rm.client.stats.mutex.Lock()
	rm.client.stats.TotalReceived++
	rm.client.stats.BytesReceived += uint64(len(msg.Payload))
	rm.client.stats.mutex.Unlock()

	// 1. 解密消息
	decrypted, err := rm.client.crypto.DecryptMessage(msg.Payload)
	if err != nil {
		rm.client.logger.Warn("[ReceiveManager] Failed to decrypt message: %v", err)
		rm.stats.mutex.Lock()
		rm.stats.DecryptFailures++
		rm.stats.mutex.Unlock()
		return
	}

	// 2. 根据消息类型处理
	switch msg.Type {
	case transport.MessageTypeAuth:
		rm.handleAuthMessage(decrypted)

	case transport.MessageTypeData:
		rm.handleDataMessage(decrypted)

	case transport.MessageTypeControl:
		rm.handleControlMessage(decrypted)

	default:
		rm.client.logger.Warn("[ReceiveManager] Unknown message type: %d", msg.Type)
		rm.stats.mutex.Lock()
		rm.stats.InvalidMessages++
		rm.stats.mutex.Unlock()
	}
}

// handleAuthMessage 处理认证消息
func (rm *ReceiveManager) handleAuthMessage(data []byte) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		rm.client.logger.Error("[ReceiveManager] Failed to unmarshal auth message: %v", err)
		return
	}

	msgType, ok := payload["type"].(string)
	if !ok {
		rm.client.logger.Warn("[ReceiveManager] Invalid auth message type")
		return
	}

	switch msgType {
	case "auth.join_response":
		rm.handleJoinResponse(payload)

	default:
		rm.client.logger.Warn("[ReceiveManager] Unknown auth message type: %s", msgType)
	}
}

// handleJoinResponse 处理加入响应
func (rm *ReceiveManager) handleJoinResponse(payload map[string]interface{}) {
	success, ok := payload["success"].(bool)
	if !ok || !success {
		errMsg, _ := payload["error"].(string)
		rm.client.logger.Error("[ReceiveManager] Join failed: %s", errMsg)

		// 发布加入失败事件
		rm.client.eventBus.Publish(events.EventSystemError, map[string]interface{}{
			"action": "join_failed",
			"error":  errMsg,
		})
		return
	}

	// 提取成员信息
	memberData, ok := payload["member"].(map[string]interface{})
	if !ok {
		rm.client.logger.Error("[ReceiveManager] Invalid member data in join response")
		return
	}

	memberID, ok := memberData["id"].(string)
	if !ok {
		rm.client.logger.Error("[ReceiveManager] Missing member ID in join response")
		return
	}

	// 设置成员ID
	rm.client.SetMemberID(memberID)

	// 更新连接时间
	rm.client.stats.mutex.Lock()
	rm.client.stats.ConnectedAt = time.Now()
	rm.client.stats.mutex.Unlock()

	rm.client.logger.Info("[ReceiveManager] Successfully joined channel as: %s", memberID)

	// 发布加入成功事件
	rm.client.eventBus.Publish(events.EventMemberJoined, &events.MemberEvent{
		Member:    nil, // TODO: 构造完整的Member对象
		Action:    "joined",
		ChannelID: rm.client.config.ChannelID,
	})

	// 触发同步
	if rm.client.syncManager != nil {
		go rm.client.syncManager.TriggerSync()
	}
}

// handleDataMessage 处理数据消息
func (rm *ReceiveManager) handleDataMessage(data []byte) {
	var msg models.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		rm.client.logger.Error("[ReceiveManager] Failed to unmarshal message: %v", err)
		rm.stats.mutex.Lock()
		rm.stats.InvalidMessages++
		rm.stats.mutex.Unlock()
		return
	}

	// 3. 检查消息去重
	if rm.isDuplicate(msg.ID) {
		rm.client.logger.Debug("[ReceiveManager] Duplicate message: %s", msg.ID)
		rm.stats.mutex.Lock()
		rm.stats.DuplicateMessages++
		rm.stats.mutex.Unlock()
		return
	}

	// 4. 标记为已见
	rm.markAsSeen(msg.ID)

	// 5. 验证消息（基本验证）
	if msg.ChannelID != rm.client.config.ChannelID {
		rm.client.logger.Warn("[ReceiveManager] Message from wrong channel: %s", msg.ChannelID)
		rm.stats.mutex.Lock()
		rm.stats.InvalidMessages++
		rm.stats.mutex.Unlock()
		return
	}

	// 过滤掉自己发送的消息（可选）
	if msg.SenderID == rm.client.memberID {
		rm.client.logger.Debug("[ReceiveManager] Ignoring own message: %s", msg.ID)
		return
	}

	// 6. 保存到数据库
	if err := rm.client.messageRepo.Create(&msg); err != nil {
		rm.client.logger.Error("[ReceiveManager] Failed to save message: %v", err)
	}

	// 7. 更新统计
	rm.stats.mutex.Lock()
	rm.stats.ValidMessages++
	rm.stats.mutex.Unlock()

	rm.client.stats.mutex.Lock()
	rm.client.stats.MessagesReceived++
	rm.client.stats.mutex.Unlock()

	// 8. 更新最后接收的消息ID
	rm.client.lastSeenMsgID = msg.ID

	// 9. 发布消息事件
	rm.client.eventBus.Publish(events.EventMessageReceived, &events.MessageEvent{
		Message:   &msg,
		ChannelID: msg.ChannelID,
		SenderID:  msg.SenderID,
	})

	rm.client.logger.Debug("[ReceiveManager] Message received: %s from %s", msg.ID, msg.SenderID)
}

// handleControlMessage 处理控制消息
func (rm *ReceiveManager) handleControlMessage(data []byte) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		rm.client.logger.Error("[ReceiveManager] Failed to unmarshal control message: %v", err)
		return
	}

	msgType, ok := payload["type"].(string)
	if !ok {
		rm.client.logger.Warn("[ReceiveManager] Invalid control message type")
		return
	}

	rm.client.logger.Debug("[ReceiveManager] Control message: %s", msgType)

	switch msgType {
	case "member.status":
		// 成员状态更新
		rm.handleMemberStatus(payload)

	case "member.joined":
		// 成员加入通知
		rm.handleMemberJoined(payload)

	case "member.left":
		// 成员离开通知
		rm.handleMemberLeft(payload)

	default:
		rm.client.logger.Debug("[ReceiveManager] Unknown control message: %s", msgType)
	}
}

// handleMemberStatus 处理成员状态更新
func (rm *ReceiveManager) handleMemberStatus(payload map[string]interface{}) {
	memberID, ok := payload["member_id"].(string)
	if !ok {
		return
	}

	status, ok := payload["status"].(string)
	if !ok {
		return
	}

	rm.client.logger.Debug("[ReceiveManager] Member status: %s -> %s", memberID, status)

	// 发布成员状态事件
	rm.client.eventBus.Publish(events.EventStatusChanged, &events.StatusEvent{
		MemberID:  memberID,
		ChannelID: rm.client.config.ChannelID,
	})
}

// handleMemberJoined 处理成员加入通知
func (rm *ReceiveManager) handleMemberJoined(payload map[string]interface{}) {
	rm.client.logger.Debug("[ReceiveManager] Member joined notification")

	// TODO: 解析成员信息并更新本地缓存

	rm.client.eventBus.Publish(events.EventMemberJoined, &events.MemberEvent{
		Action:    "joined",
		ChannelID: rm.client.config.ChannelID,
	})
}

// handleMemberLeft 处理成员离开通知
func (rm *ReceiveManager) handleMemberLeft(payload map[string]interface{}) {
	rm.client.logger.Debug("[ReceiveManager] Member left notification")

	// TODO: 解析成员信息并更新本地缓存

	rm.client.eventBus.Publish(events.EventMemberLeft, &events.MemberEvent{
		Action:    "left",
		ChannelID: rm.client.config.ChannelID,
	})
}

// isDuplicate 检查消息是否重复
func (rm *ReceiveManager) isDuplicate(messageID string) bool {
	rm.seenMutex.RLock()
	_, exists := rm.seenMessages[messageID]
	rm.seenMutex.RUnlock()
	return exists
}

// markAsSeen 标记消息为已见
func (rm *ReceiveManager) markAsSeen(messageID string) {
	rm.seenMutex.Lock()
	defer rm.seenMutex.Unlock()

	// 如果已满，删除最老的一半
	if len(rm.seenMessages) >= rm.maxSeenSize {
		rm.cleanupOldSeen()
	}

	rm.seenMessages[messageID] = time.Now()
}

// cleanupOldSeen 清理旧的已见消息记录（需要持有锁）
func (rm *ReceiveManager) cleanupOldSeen() {
	// 找出最老的一半并删除
	threshold := time.Now().Add(-1 * time.Hour)

	for msgID, t := range rm.seenMessages {
		if t.Before(threshold) {
			delete(rm.seenMessages, msgID)
		}
	}

	rm.client.logger.Debug("[ReceiveManager] Cleaned up seen messages, remaining: %d", len(rm.seenMessages))
}

// cleanupSeenMessages 定期清理已见消息记录
func (rm *ReceiveManager) cleanupSeenMessages() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rm.client.ctx.Done():
			return

		case <-ticker.C:
			rm.seenMutex.Lock()
			rm.cleanupOldSeen()
			rm.seenMutex.Unlock()
		}
	}
}

// GetStats 获取统计信息
func (rm *ReceiveManager) GetStats() ReceiveStats {
	rm.stats.mutex.RLock()
	defer rm.stats.mutex.RUnlock()

	return ReceiveStats{
		TotalReceived:     rm.stats.TotalReceived,
		ValidMessages:     rm.stats.ValidMessages,
		DuplicateMessages: rm.stats.DuplicateMessages,
		InvalidMessages:   rm.stats.InvalidMessages,
		DecryptFailures:   rm.stats.DecryptFailures,
	}
}
