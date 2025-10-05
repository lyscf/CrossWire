package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/events"
	"crosswire/internal/models"
	"crosswire/internal/transport"

	"github.com/google/uuid"
)

// OfflineQueue 离线消息队列
type OfflineQueue struct {
	client *Client
	ctx    context.Context
	cancel context.CancelFunc

	// 队列
	queue      []*QueuedMessage
	queueMutex sync.RWMutex

	// 统计
	stats      OfflineQueueStats
	statsMutex sync.RWMutex

	// 配置
	maxQueueSize int
	retryDelay   time.Duration
	maxRetries   int
}

// QueuedMessage 队列中的消息
type QueuedMessage struct {
	ID          string
	Content     string
	Type        models.MessageType
	ReplyTo     string
	QueuedAt    time.Time
	Retries     int
	LastAttempt time.Time
	Error       error
}

// OfflineQueueStats 离线队列统计
type OfflineQueueStats struct {
	TotalQueued  int64
	TotalSent    int64
	TotalFailed  int64
	CurrentSize  int
	LastSendTime time.Time
	mutex        sync.RWMutex
}

// NewOfflineQueue 创建离线消息队列
func NewOfflineQueue(client *Client) *OfflineQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &OfflineQueue{
		client:       client,
		ctx:          ctx,
		cancel:       cancel,
		queue:        make([]*QueuedMessage, 0),
		maxQueueSize: 1000,
		retryDelay:   5 * time.Second,
		maxRetries:   3,
	}
}

// Start 启动离线队列
func (oq *OfflineQueue) Start() error {
	oq.client.logger.Info("[OfflineQueue] Starting...")

	// 启动处理协程
	go oq.processLoop()

	oq.client.logger.Info("[OfflineQueue] Started successfully")
	return nil
}

// Stop 停止离线队列
func (oq *OfflineQueue) Stop() error {
	oq.client.logger.Info("[OfflineQueue] Stopping...")
	oq.cancel()
	oq.client.logger.Info("[OfflineQueue] Stopped")
	return nil
}

// Enqueue 将消息加入队列
func (oq *OfflineQueue) Enqueue(content string, msgType models.MessageType, replyTo string) error {
	oq.queueMutex.Lock()
	defer oq.queueMutex.Unlock()

	// 检查队列是否已满
	if len(oq.queue) >= oq.maxQueueSize {
		return fmt.Errorf("queue is full (max: %d)", oq.maxQueueSize)
	}

	msg := &QueuedMessage{
		ID:       uuid.New().String(),
		Content:  content,
		Type:     msgType,
		ReplyTo:  replyTo,
		QueuedAt: time.Now(),
		Retries:  0,
	}

	oq.queue = append(oq.queue, msg)

	// 更新统计
	oq.statsMutex.Lock()
	oq.stats.TotalQueued++
	oq.stats.CurrentSize = len(oq.queue)
	oq.statsMutex.Unlock()

	oq.client.logger.Debug("[OfflineQueue] Message queued: %s (queue size: %d)", msg.ID, len(oq.queue))

	// 发布事件
	oq.client.eventBus.Publish(events.EventSystemError, events.SystemEvent{
		Type:    "message_queued",
		Message: fmt.Sprintf("Message queued (offline): %s", msg.ID),
		Data: map[string]interface{}{
			"message_id": msg.ID,
			"queue_size": len(oq.queue),
		},
	})

	return nil
}

// processLoop 处理循环
func (oq *OfflineQueue) processLoop() {
	ticker := time.NewTicker(oq.retryDelay)
	defer ticker.Stop()

	for {
		select {
		case <-oq.ctx.Done():
			oq.client.logger.Debug("[OfflineQueue] Process loop stopped")
			return
		case <-ticker.C:
			oq.processQueue()
		}
	}
}

// processQueue 处理队列中的消息
func (oq *OfflineQueue) processQueue() {
	oq.queueMutex.Lock()

	if len(oq.queue) == 0 {
		oq.queueMutex.Unlock()
		return
	}

	// 获取第一条消息
	msg := oq.queue[0]
	oq.queueMutex.Unlock()

	// 检查客户端是否在线
	if !oq.client.IsRunning() {
		oq.client.logger.Debug("[OfflineQueue] Client offline, skipping send")
		return
	}

	// 尝试发送
	oq.client.logger.Debug("[OfflineQueue] Attempting to send message: %s (retry: %d/%d)",
		msg.ID, msg.Retries, oq.maxRetries)

	err := oq.sendMessage(msg)

	oq.queueMutex.Lock()
	defer oq.queueMutex.Unlock()

	if err != nil {
		// 发送失败
		msg.Retries++
		msg.LastAttempt = time.Now()
		msg.Error = err

		if msg.Retries >= oq.maxRetries {
			// 达到最大重试次数，移除消息
			oq.queue = oq.queue[1:]
			oq.statsMutex.Lock()
			oq.stats.TotalFailed++
			oq.stats.CurrentSize = len(oq.queue)
			oq.statsMutex.Unlock()

			oq.client.logger.Error("[OfflineQueue] Message failed after %d retries: %s - %v",
				oq.maxRetries, msg.ID, err)

			// 发布失败事件
			oq.client.eventBus.Publish(events.EventSystemError, events.SystemEvent{
				Type:    "message_send_failed",
				Message: fmt.Sprintf("Message failed after %d retries: %v", oq.maxRetries, err),
				Data: map[string]string{
					"message_id": msg.ID,
				},
			})
		} else {
			oq.client.logger.Warn("[OfflineQueue] Message send failed (retry %d/%d): %s - %v",
				msg.Retries, oq.maxRetries, msg.ID, err)
		}
	} else {
		// 发送成功，移除消息
		oq.queue = oq.queue[1:]
		oq.statsMutex.Lock()
		oq.stats.TotalSent++
		oq.stats.CurrentSize = len(oq.queue)
		oq.stats.LastSendTime = time.Now()
		oq.statsMutex.Unlock()

		oq.client.logger.Info("[OfflineQueue] Message sent successfully: %s", msg.ID)

		// 发布成功事件
		content := models.MessageContent{
			"text": msg.Content,
		}
		oq.client.eventBus.Publish(events.EventMessageSent, events.MessageEvent{
			Message: &models.Message{
				ID:      msg.ID,
				Content: content,
			},
			ChannelID: oq.client.config.ChannelID,
			SenderID:  oq.client.memberID,
		})
	}
}

// sendMessage 发送消息
func (oq *OfflineQueue) sendMessage(msg *QueuedMessage) error {
	// 构造消息
	content := models.MessageContent{
		"text": msg.Content,
	}
	message := &models.Message{
		ID:        msg.ID,
		ChannelID: oq.client.config.ChannelID,
		SenderID:  oq.client.memberID,
		Type:      msg.Type,
		Content:   content,
		ReplyTo:   nil, // TODO: 支持回复
		Timestamp: time.Now(),
	}

	// 序列化并加密
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	encrypted, err := oq.client.crypto.EncryptMessage(payload)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	// 发送
	transportMsg := &transport.Message{
		ID:        msg.ID,
		Type:      transport.MessageTypeData,
		SenderID:  oq.client.memberID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}

	return oq.client.transport.SendMessage(transportMsg)
}

// GetQueueSize 获取队列大小
func (oq *OfflineQueue) GetQueueSize() int {
	oq.queueMutex.RLock()
	defer oq.queueMutex.RUnlock()
	return len(oq.queue)
}

// GetQueuedMessages 获取队列中的消息列表
func (oq *OfflineQueue) GetQueuedMessages() []*QueuedMessage {
	oq.queueMutex.RLock()
	defer oq.queueMutex.RUnlock()

	messages := make([]*QueuedMessage, len(oq.queue))
	copy(messages, oq.queue)
	return messages
}

// Clear 清空队列
func (oq *OfflineQueue) Clear() {
	oq.queueMutex.Lock()
	defer oq.queueMutex.Unlock()

	oq.queue = make([]*QueuedMessage, 0)

	oq.statsMutex.Lock()
	oq.stats.CurrentSize = 0
	oq.statsMutex.Unlock()

	oq.client.logger.Info("[OfflineQueue] Queue cleared")
}

// GetStats 获取统计信息
func (oq *OfflineQueue) GetStats() OfflineQueueStats {
	oq.statsMutex.RLock()
	defer oq.statsMutex.RUnlock()

	return OfflineQueueStats{
		TotalQueued:  oq.stats.TotalQueued,
		TotalSent:    oq.stats.TotalSent,
		TotalFailed:  oq.stats.TotalFailed,
		CurrentSize:  oq.stats.CurrentSize,
		LastSendTime: oq.stats.LastSendTime,
	}
}

// TriggerSend 手动触发发送
func (oq *OfflineQueue) TriggerSend() {
	go oq.processQueue()
}

// SetMaxQueueSize 设置最大队列大小
func (oq *OfflineQueue) SetMaxQueueSize(size int) {
	oq.maxQueueSize = size
	oq.client.logger.Debug("[OfflineQueue] Max queue size set to: %d", size)
}

// SetRetryDelay 设置重试延迟
func (oq *OfflineQueue) SetRetryDelay(delay time.Duration) {
	oq.retryDelay = delay
	oq.client.logger.Debug("[OfflineQueue] Retry delay set to: %v", delay)
}

// SetMaxRetries 设置最大重试次数
func (oq *OfflineQueue) SetMaxRetries(retries int) {
	oq.maxRetries = retries
	oq.client.logger.Debug("[OfflineQueue] Max retries set to: %d", retries)
}
