package server

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/models"
	"crosswire/internal/transport"
)

// BroadcastManager 广播管理器
// 参考: docs/ARCHITECTURE.md - 3.1.2 服务端模块 - BroadcastManager
// 参考: docs/ARP_BROADCAST_MODE.md - 2. 服务器签名与广播
type BroadcastManager struct {
	server *Server

	// 广播队列
	broadcastQueue chan *BroadcastTask
	queueSize      int

	// 消息去重（防止接收自己的广播）
	sentMessages map[string]time.Time // messageID -> timestamp
	sentMutex    sync.RWMutex

	// ACK收集（可选）
	ackCollector map[string]map[string]bool // messageID -> memberID -> acked
	ackMutex     sync.RWMutex

	// 统计
	stats BroadcastStats
}

// BroadcastTask 广播任务
type BroadcastTask struct {
	Message   *models.Message
	Timestamp time.Time
	Retries   int
}

// BroadcastStats 广播统计
type BroadcastStats struct {
	TotalBroadcasts  uint64
	FailedBroadcasts uint64
	AverageLatency   time.Duration
	mutex            sync.RWMutex
}

// SignedPayload 签名的载荷
// 参考: docs/PROTOCOL.md - 2.2.3 消息广播（服务器签名模式）
type SignedPayload struct {
	Message   []byte `json:"message"`   // 加密的消息
	Signature []byte `json:"signature"` // 服务器签名
	Timestamp int64  `json:"timestamp"` // 时间戳
	ServerID  string `json:"server_id"` // 服务器ID
}

// NewBroadcastManager 创建广播管理器
func NewBroadcastManager(server *Server) *BroadcastManager {
	return &BroadcastManager{
		server:         server,
		broadcastQueue: make(chan *BroadcastTask, 100),
		queueSize:      100,
		sentMessages:   make(map[string]time.Time),
		ackCollector:   make(map[string]map[string]bool),
	}
}

// Run 运行广播管理器
func (bm *BroadcastManager) Run() {
	defer bm.server.wg.Done()

	// 定期清理过期的消息记录
	cleanupTicker := time.NewTicker(5 * time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-bm.server.ctx.Done():
			return

		case task := <-bm.broadcastQueue:
			bm.processBroadcastTask(task)

		case <-cleanupTicker.C:
			bm.cleanupSentMessages()
		}
	}
}

// Broadcast 广播消息
func (bm *BroadcastManager) Broadcast(msg *models.Message) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	// 添加到广播队列
	task := &BroadcastTask{
		Message:   msg,
		Timestamp: time.Now(),
		Retries:   0,
	}

	select {
	case bm.broadcastQueue <- task:
		return nil
	default:
		return fmt.Errorf("broadcast queue is full")
	}
}

// processBroadcastTask 处理广播任务
func (bm *BroadcastManager) processBroadcastTask(task *BroadcastTask) {
	startTime := time.Now()

	// 1. 序列化消息
	messageData, err := json.Marshal(task.Message)
	if err != nil {
		bm.server.logger.Error("[BroadcastManager] Failed to marshal message: %v", err)
		bm.stats.mutex.Lock()
		bm.stats.FailedBroadcasts++
		bm.stats.mutex.Unlock()
		return
	}

	// 2. 加密消息（使用频道密钥）
	encryptedData, err := bm.server.crypto.EncryptMessage(messageData)
	if err != nil {
		bm.server.logger.Error("[BroadcastManager] Failed to encrypt message: %v", err)
		bm.stats.mutex.Lock()
		bm.stats.FailedBroadcasts++
		bm.stats.mutex.Unlock()
		return
	}

	// 3. 服务器签名（如果启用）
	var signature []byte
	if bm.server.config.EnableSignature {
		signature = ed25519.Sign(bm.server.config.PrivateKey, encryptedData)
	}

	// 4. 构造签名载荷
	signedPayload := &SignedPayload{
		Message:   encryptedData,
		Signature: signature,
		Timestamp: time.Now().Unix(),
		ServerID:  bm.server.config.ChannelID,
	}

	payloadData, err := json.Marshal(signedPayload)
	if err != nil {
		bm.server.logger.Error("[BroadcastManager] Failed to marshal signed payload: %v", err)
		bm.stats.mutex.Lock()
		bm.stats.FailedBroadcasts++
		bm.stats.mutex.Unlock()
		return
	}

	// 5. 通过传输层广播
	transportMsg := &transport.Message{
		Type:      transport.MessageTypeData,
		SenderID:  bm.server.config.ChannelID,
		Payload:   payloadData,
		Timestamp: time.Now(),
	}

	if err := bm.server.transport.SendMessage(transportMsg); err != nil {
		bm.server.logger.Error("[BroadcastManager] Failed to broadcast message: %v", err)

		// 重试逻辑
		if task.Retries < 3 {
			task.Retries++
			time.Sleep(100 * time.Millisecond)
			bm.broadcastQueue <- task
		} else {
			bm.stats.mutex.Lock()
			bm.stats.FailedBroadcasts++
			bm.stats.mutex.Unlock()
		}
		return
	}

	// 6. 记录已发送消息（用于去重）
	bm.sentMutex.Lock()
	bm.sentMessages[task.Message.ID] = time.Now()
	bm.sentMutex.Unlock()

	// 7. 更新统计
	latency := time.Since(startTime)
	bm.stats.mutex.Lock()
	bm.stats.TotalBroadcasts++
	if bm.stats.AverageLatency == 0 {
		bm.stats.AverageLatency = latency
	} else {
		bm.stats.AverageLatency = (bm.stats.AverageLatency + latency) / 2
	}
	bm.stats.mutex.Unlock()

	// 更新服务器统计
	bm.server.stats.mutex.Lock()
	bm.server.stats.TotalBroadcasts++
	bm.server.stats.mutex.Unlock()

	bm.server.logger.Debug("[BroadcastManager] Message broadcasted: %s, latency: %v",
		task.Message.ID, latency)
}

// IsSentByMe 检查消息是否由自己发送（用于去重）
func (bm *BroadcastManager) IsSentByMe(messageID string) bool {
	bm.sentMutex.RLock()
	defer bm.sentMutex.RUnlock()

	_, exists := bm.sentMessages[messageID]
	return exists
}

// cleanupSentMessages 清理过期的消息记录
func (bm *BroadcastManager) cleanupSentMessages() {
	bm.sentMutex.Lock()
	defer bm.sentMutex.Unlock()

	expiry := time.Now().Add(-1 * time.Hour)
	for msgID, timestamp := range bm.sentMessages {
		if timestamp.Before(expiry) {
			delete(bm.sentMessages, msgID)
		}
	}

	bm.server.logger.Debug("[BroadcastManager] Cleaned up sent messages cache, remaining: %d",
		len(bm.sentMessages))
}

// RecordAck 记录ACK（可选功能）
func (bm *BroadcastManager) RecordAck(messageID, memberID string) {
	bm.ackMutex.Lock()
	defer bm.ackMutex.Unlock()

	if bm.ackCollector[messageID] == nil {
		bm.ackCollector[messageID] = make(map[string]bool)
	}

	bm.ackCollector[messageID][memberID] = true
}

// GetAckCount 获取ACK数量
func (bm *BroadcastManager) GetAckCount(messageID string) int {
	bm.ackMutex.RLock()
	defer bm.ackMutex.RUnlock()

	if acks, exists := bm.ackCollector[messageID]; exists {
		return len(acks)
	}

	return 0
}

// GetStats 获取统计信息
func (bm *BroadcastManager) GetStats() BroadcastStats {
	bm.stats.mutex.RLock()
	defer bm.stats.mutex.RUnlock()

	// 复制统计数据（避免复制锁）
	return BroadcastStats{
		TotalBroadcasts:  bm.stats.TotalBroadcasts,
		FailedBroadcasts: bm.stats.FailedBroadcasts,
		AverageLatency:   bm.stats.AverageLatency,
	}
}
