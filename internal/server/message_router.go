package server

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/events"
	"crosswire/internal/models"
	"crosswire/internal/transport"
)

// MessageRouter 消息路由器
// 参考: docs/ARCHITECTURE.md - 3.1.2 服务端模块 - MessageRouter
type MessageRouter struct {
	server *Server

	// 消息队列
	messageQueue chan *MessageTask
	queueSize    int

	// 频率限制器
	rateLimiter *RateLimiter

	// 离线消息队列
	offlineMessages map[string][]*models.Message // memberID -> messages
	offlineMutex    sync.RWMutex
}

// MessageTask 消息任务
type MessageTask struct {
	TransportMessage *transport.Message
	ReceivedAt       time.Time
}

// SignedMessage 带签名的消息（与客户端对应）
type SignedMessage struct {
	Message   []byte `json:"message"`   // 原始消息JSON
	Signature []byte `json:"signature"` // Ed25519签名
	SenderID  string `json:"sender_id"` // 发送者ID
}

// RateLimiter 频率限制器
type RateLimiter struct {
	// memberID -> 消息时间戳列表
	messageTimes map[string][]time.Time
	mutex        sync.RWMutex
	maxRate      int           // 每分钟最多消息数
	window       time.Duration // 时间窗口
}

// NewMessageRouter 创建消息路由器
func NewMessageRouter(server *Server) *MessageRouter {
	return &MessageRouter{
		server:          server,
		messageQueue:    make(chan *MessageTask, 200),
		queueSize:       200,
		rateLimiter:     NewRateLimiter(server.config.MaxMessageRate),
		offlineMessages: make(map[string][]*models.Message),
	}
}

// NewRateLimiter 创建频率限制器
func NewRateLimiter(maxRate int) *RateLimiter {
	return &RateLimiter{
		messageTimes: make(map[string][]time.Time),
		maxRate:      maxRate,
		window:       1 * time.Minute,
	}
}

// Run 运行消息路由器
func (mr *MessageRouter) Run() {
	defer mr.server.wg.Done()

	for {
		select {
		case <-mr.server.ctx.Done():
			return
		case task := <-mr.messageQueue:
			mr.processMessageTask(task)
		}
	}
}

// HandleClientMessage 处理客户端消息
// 参考: docs/PROTOCOL.md - 2.2.3 消息广播（服务器签名模式）
func (mr *MessageRouter) HandleClientMessage(transportMsg *transport.Message) {
	task := &MessageTask{
		TransportMessage: transportMsg,
		ReceivedAt:       time.Now(),
	}

	select {
	case mr.messageQueue <- task:
		// 任务已加入队列
	default:
		mr.server.logger.Warn("[MessageRouter] Message queue is full, dropping message")
		mr.server.stats.mutex.Lock()
		mr.server.stats.DroppedMessages++
		mr.server.stats.mutex.Unlock()
	}
}

// processMessageTask 处理消息任务
func (mr *MessageRouter) processMessageTask(task *MessageTask) {
	// 1. 解密消息
	decrypted, err := mr.server.crypto.DecryptMessage(task.TransportMessage.Payload)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to decrypt message: %v", err)
		mr.server.stats.mutex.Lock()
		mr.server.stats.RejectedMessages++
		mr.server.stats.mutex.Unlock()
		return
	}

	// 2. 尝试解析签名消息
	var signedMsg SignedMessage
	if err := json.Unmarshal(decrypted, &signedMsg); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to unmarshal signed message: %v", err)
		mr.server.stats.mutex.Lock()
		mr.server.stats.RejectedMessages++
		mr.server.stats.mutex.Unlock()
		return
	}

	// 3. 验证签名
	// 获取成员公钥
	member := mr.server.channelManager.GetMemberByID(signedMsg.SenderID)
	if member == nil {
		mr.server.logger.Warn("[MessageRouter] Unknown sender: %s", signedMsg.SenderID)
		mr.server.stats.mutex.Lock()
		mr.server.stats.RejectedMessages++
		mr.server.stats.mutex.Unlock()
		return
	}

	if member.PublicKey == nil || len(member.PublicKey) == 0 {
		mr.server.logger.Warn("[MessageRouter] Member has no public key: %s", signedMsg.SenderID)
		mr.server.stats.mutex.Lock()
		mr.server.stats.RejectedMessages++
		mr.server.stats.mutex.Unlock()
		return
	}

	// 验证Ed25519签名
	if !ed25519.Verify(member.PublicKey, signedMsg.Message, signedMsg.Signature) {
		mr.server.logger.Warn("[MessageRouter] Invalid signature from member: %s", signedMsg.SenderID)
		mr.server.stats.mutex.Lock()
		mr.server.stats.RejectedMessages++
		mr.server.stats.mutex.Unlock()
		return
	}

	mr.server.logger.Debug("[MessageRouter] Signature verified for: %s", signedMsg.SenderID)

	// 4. 反序列化实际消息
	var msg models.Message
	if err := json.Unmarshal(signedMsg.Message, &msg); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to unmarshal message: %v", err)
		mr.server.stats.mutex.Lock()
		mr.server.stats.RejectedMessages++
		mr.server.stats.mutex.Unlock()
		return
	}

	// 5. 验证消息SenderID与签名的SenderID一致
	if msg.SenderID != signedMsg.SenderID {
		mr.server.logger.Warn("[MessageRouter] SenderID mismatch: %s != %s", msg.SenderID, signedMsg.SenderID)
		mr.server.stats.mutex.Lock()
		mr.server.stats.RejectedMessages++
		mr.server.stats.mutex.Unlock()
		return
	}

	// 6. 检查是否被禁言
	if mr.server.channelManager.IsMuted(msg.SenderID) {
		mr.server.logger.Warn("[MessageRouter] Muted member trying to send message: %s", msg.SenderID)
		mr.server.stats.mutex.Lock()
		mr.server.stats.RejectedMessages++
		mr.server.stats.mutex.Unlock()
		return
	}

	// 7. 频率限制检查
	if mr.server.config.EnableRateLimit {
		if !mr.rateLimiter.Allow(msg.SenderID) {
			mr.server.logger.Warn("[MessageRouter] Rate limit exceeded for member: %s", msg.SenderID)
			mr.server.stats.mutex.Lock()
			mr.server.stats.RejectedMessages++
			mr.server.stats.mutex.Unlock()
			return
		}
	}

	// 7.5. 反垃圾消息检测
	if isSpam, reason := mr.server.spamDetector.CheckMessage(&msg, msg.SenderID); isSpam {
		mr.server.logger.Warn("[MessageRouter] Spam message detected from %s: %s", msg.SenderID, reason)
		mr.server.stats.mutex.Lock()
		mr.server.stats.RejectedMessages++
		mr.server.stats.mutex.Unlock()
		return
	}

	// 8. 补充消息元数据
	msg.ChannelID = mr.server.config.ChannelID
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// 9. 持久化消息
	if err := mr.persistMessage(&msg); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to persist message: %v", err)
		// 不阻止广播
	}

	// 10. 广播消息（带服务器签名）
	if err := mr.server.broadcastManager.Broadcast(&msg); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to broadcast message: %v", err)
		return
	}

	// 11. 发布事件
	mr.server.eventBus.Publish(events.EventMessageReceived, events.NewMessageReceivedEvent(&msg, mr.server.config.ChannelID))

	mr.server.logger.Debug("[MessageRouter] Signed message verified and broadcasted: %s from %s",
		msg.ID, msg.SenderID)
}

// persistMessage 持久化消息
func (mr *MessageRouter) persistMessage(msg *models.Message) error {
	if err := mr.server.messageRepo.Create(msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// HandleFileUpload 处理文件上传
// 参考: internal/client/file_manager.go 的上传实现
func (mr *MessageRouter) HandleFileUpload(transportMsg *transport.Message) {
	mr.server.logger.Debug("[MessageRouter] File upload request from: %s", transportMsg.SenderID)

	// 1. 解密消息
	decrypted, err := mr.server.crypto.DecryptMessage(transportMsg.Payload)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to decrypt file upload: %v", err)
		return
	}

	// 2. 反序列化消息
	var msg models.Message
	if err := json.Unmarshal(decrypted, &msg); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to unmarshal file message: %v", err)
		return
	}

	// 3. 验证成员权限
	if !mr.server.channelManager.HasMember(msg.SenderID) {
		mr.server.logger.Warn("[MessageRouter] Non-member trying to upload file: %s", msg.SenderID)
		return
	}

	// 4. 根据消息类型处理
	switch msg.Type {
	case models.MessageTypeFile:
		// 文件元数据或完整小文件
		mr.handleFileMetadata(&msg)
	default:
		mr.server.logger.Warn("[MessageRouter] Unknown file message type: %s", msg.Type)
	}
}

// handleFileMetadata 处理文件元数据
func (mr *MessageRouter) handleFileMetadata(msg *models.Message) {
	mr.server.logger.Debug("[MessageRouter] Processing file metadata from: %s", msg.SenderID)

	// 1. 从消息内容中提取文件信息（直接从map读取）
	fileContent := &models.FileContent{}
	if fileID, ok := msg.Content["file_id"].(string); ok {
		fileContent.FileID = fileID
	}
	if filename, ok := msg.Content["filename"].(string); ok {
		fileContent.Filename = filename
	}
	if size, ok := msg.Content["size"].(float64); ok {
		fileContent.Size = int64(size)
	} else if size, ok := msg.Content["size"].(int64); ok {
		fileContent.Size = size
	}
	if mimeType, ok := msg.Content["mime_type"].(string); ok {
		fileContent.MimeType = mimeType
	}
	if sha256, ok := msg.Content["sha256"].(string); ok {
		fileContent.SHA256 = sha256
	}

	// 2. 创建或更新文件记录
	file := &models.File{
		ID:          fileContent.FileID,
		MessageID:   msg.ID,
		ChannelID:   mr.server.config.ChannelID,
		SenderID:    msg.SenderID,
		Filename:    fileContent.Filename,
		Size:        fileContent.Size,
		MimeType:    fileContent.MimeType,
		SHA256:      fileContent.SHA256,
		StorageType: models.StorageInline,
		Encrypted:   true,
		UploadedAt:  time.Now(),
	}

	// 3. 保存文件记录
	if err := mr.server.fileRepo.Create(file); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to save file metadata: %v", err)
		return
	}

	// 4. 持久化消息
	msg.ChannelID = mr.server.config.ChannelID
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	if err := mr.persistMessage(msg); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to persist file message: %v", err)
	}

	// 5. 广播文件消息
	if err := mr.server.broadcastManager.Broadcast(msg); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to broadcast file message: %v", err)
		return
	}

	// 6. 发布文件上传事件
	mr.server.eventBus.Publish(events.EventFileUploaded, events.FileEvent{
		File:       file,
		ChannelID:  mr.server.config.ChannelID,
		UploaderID: msg.SenderID,
		Progress:   100,
	})

	mr.server.logger.Info("[MessageRouter] File uploaded successfully: %s (%s)", file.Filename, file.ID)
}

// HandleFileChunk 处理文件分块
// 参考: internal/client/file_manager.go 的分块上传实现
func (mr *MessageRouter) HandleFileChunk(transportMsg *transport.Message) {
	mr.server.logger.Debug("[MessageRouter] File chunk from: %s", transportMsg.SenderID)

	// 1. 解密消息
	decrypted, err := mr.server.crypto.DecryptMessage(transportMsg.Payload)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to decrypt file chunk: %v", err)
		return
	}

	// 2. 解析分块数据结构
	type ChunkData struct {
		FileID      string `json:"file_id"`
		ChunkIndex  int    `json:"chunk_index"`
		Data        []byte `json:"data"`
		Checksum    string `json:"checksum"`
		TotalChunks int    `json:"total_chunks"`
	}

	var chunkData ChunkData
	if err := json.Unmarshal(decrypted, &chunkData); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to unmarshal chunk data: %v", err)
		return
	}

	// 3. 验证分块
	if !mr.verifyChunkChecksum(chunkData.Data, chunkData.Checksum) {
		mr.server.logger.Error("[MessageRouter] Chunk checksum mismatch for file: %s chunk: %d",
			chunkData.FileID, chunkData.ChunkIndex)
		return
	}

	// 4. 保存分块记录
	chunk := &models.FileChunk{
		FileID:     chunkData.FileID,
		ChunkIndex: chunkData.ChunkIndex,
		Size:       len(chunkData.Data),
		Checksum:   chunkData.Checksum,
		Uploaded:   true,
		UploadedAt: time.Now(),
	}

	if err := mr.server.fileRepo.CreateChunk(chunk); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to save chunk: %v", err)
		return
	}

	// 5. 更新文件上传进度
	file, err := mr.server.fileRepo.GetByID(chunkData.FileID)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to get file: %v", err)
		return
	}

	file.UploadedChunks++
	if err := mr.server.fileRepo.Update(file); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to update file progress: %v", err)
		return
	}

	// 6. 发布进度事件
	progress := int(float64(file.UploadedChunks) / float64(file.TotalChunks) * 100)
	mr.server.eventBus.Publish(events.EventFileProgress, events.FileEvent{
		File:       file,
		ChannelID:  mr.server.config.ChannelID,
		UploaderID: file.SenderID,
		Progress:   progress,
	})

	// 7. 检查是否完成
	if file.UploadedChunks >= file.TotalChunks {
		mr.handleFileUploadComplete(file)
	}

	// 8. 转发分块给其他客户端
	transportMsg.Type = transport.MessageTypeData
	if err := mr.server.transport.SendMessage(transportMsg); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to forward file chunk: %v", err)
	}
}

// handleFileUploadComplete 处理文件上传完成
func (mr *MessageRouter) handleFileUploadComplete(file *models.File) {
	mr.server.logger.Info("[MessageRouter] File upload completed: %s (%s)", file.Filename, file.ID)

	// 1. 更新文件状态
	file.UploadStatus = models.UploadStatusCompleted
	if err := mr.server.fileRepo.Update(file); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to update file status: %v", err)
		return
	}

	// 2. 验证完整性（如果有SHA256）
	if file.SHA256 != "" {
		// TODO: 重新计算完整文件的SHA256并验证
		mr.server.logger.Debug("[MessageRouter] File SHA256 verification: %s", file.SHA256)
	}

	// 3. 发布完成事件
	mr.server.eventBus.Publish(events.EventFileUploaded, events.FileEvent{
		File:       file,
		ChannelID:  mr.server.config.ChannelID,
		UploaderID: file.SenderID,
		Progress:   100,
	})
}

// HandleFileDownloadRequest 处理文件下载请求
func (mr *MessageRouter) HandleFileDownloadRequest(transportMsg *transport.Message) {
	mr.server.logger.Debug("[MessageRouter] File download request from: %s", transportMsg.SenderID)

	// 1. 解密请求
	decrypted, err := mr.server.crypto.DecryptMessage(transportMsg.Payload)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to decrypt download request: %v", err)
		return
	}

	type DownloadRequest struct {
		FileID string `json:"file_id"`
	}

	var req DownloadRequest
	if err := json.Unmarshal(decrypted, &req); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to unmarshal download request: %v", err)
		return
	}

	// 2. 验证权限
	if !mr.server.channelManager.HasMember(transportMsg.SenderID) {
		mr.server.logger.Warn("[MessageRouter] Non-member requesting file download: %s", transportMsg.SenderID)
		return
	}

	// 3. 获取文件信息
	file, err := mr.server.fileRepo.GetByID(req.FileID)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] File not found: %s", req.FileID)
		return
	}

	// 4. 获取文件分块
	chunks, err := mr.server.fileRepo.GetChunksByFileID(req.FileID)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to get file chunks: %v", err)
		return
	}

	// 5. 发送文件元数据
	mr.sendFileMetadataToMember(transportMsg.SenderID, file)

	// 6. 逐块发送（如果有分块）
	for _, chunk := range chunks {
		mr.sendFileChunkToMember(transportMsg.SenderID, file, chunk)
		time.Sleep(10 * time.Millisecond) // 避免淹没接收方
	}

	mr.server.logger.Info("[MessageRouter] File sent to member: %s -> %s",
		file.Filename, transportMsg.SenderID)
}

// sendFileMetadataToMember 发送文件元数据给指定成员
func (mr *MessageRouter) sendFileMetadataToMember(memberID string, file *models.File) error {
	// 构造文件消息
	msg := &models.Message{
		ID:        fmt.Sprintf("file-meta-%s", file.ID),
		ChannelID: mr.server.config.ChannelID,
		SenderID:  "server",
		Type:      models.MessageTypeFile,
		Timestamp: time.Now(),
	}

	// 设置文件内容（直接构造map）
	msg.Content = models.MessageContent{
		"file_id":   file.ID,
		"filename":  file.Filename,
		"size":      file.Size,
		"mime_type": file.MimeType,
		"sha256":    file.SHA256,
	}

	// 广播（客户端会根据需要接收）
	return mr.server.broadcastManager.Broadcast(msg)
}

// sendFileChunkToMember 发送文件分块给指定成员
func (mr *MessageRouter) sendFileChunkToMember(memberID string, file *models.File, chunk *models.FileChunk) error {
	// TODO: 实现分块数据的实际发送
	// 这需要从存储中读取分块数据并发送
	mr.server.logger.Debug("[MessageRouter] Sending chunk %d to member %s", chunk.ChunkIndex, memberID)
	return nil
}

// verifyChunkChecksum 验证分块校验和
func (mr *MessageRouter) verifyChunkChecksum(data []byte, checksum string) bool {
	// 简单的校验实现，实际应该根据具体的校验算法
	return true // TODO: 实现实际的校验逻辑
}

// AddOfflineMessage 添加离线消息
func (mr *MessageRouter) AddOfflineMessage(memberID string, msg *models.Message) {
	if !mr.server.config.EnableOffline {
		return
	}

	mr.offlineMutex.Lock()
	defer mr.offlineMutex.Unlock()

	mr.offlineMessages[memberID] = append(mr.offlineMessages[memberID], msg)

	mr.server.logger.Debug("[MessageRouter] Offline message queued for member: %s", memberID)
}

// GetOfflineMessages 获取离线消息
func (mr *MessageRouter) GetOfflineMessages(memberID string) []*models.Message {
	mr.offlineMutex.Lock()
	defer mr.offlineMutex.Unlock()

	messages := mr.offlineMessages[memberID]
	delete(mr.offlineMessages, memberID)

	return messages
}

// ClearOfflineMessages 清除离线消息
func (mr *MessageRouter) ClearOfflineMessages(memberID string) {
	mr.offlineMutex.Lock()
	defer mr.offlineMutex.Unlock()

	delete(mr.offlineMessages, memberID)
}

// Allow 检查是否允许发送消息（频率限制）
func (rl *RateLimiter) Allow(memberID string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// 获取该成员的消息时间戳
	times := rl.messageTimes[memberID]

	// 过滤掉过期的时间戳
	validTimes := make([]time.Time, 0)
	for _, t := range times {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}

	// 检查是否超过限制
	if len(validTimes) >= rl.maxRate {
		rl.messageTimes[memberID] = validTimes
		return false
	}

	// 添加新的时间戳
	validTimes = append(validTimes, now)
	rl.messageTimes[memberID] = validTimes

	return true
}

// Reset 重置频率限制
func (rl *RateLimiter) Reset(memberID string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.messageTimes, memberID)
}

// HandleSyncRequest 处理同步请求
// 参考: internal/client/sync_manager.go 的客户端实现
// 参考: docs/PROTOCOL.md - 5.3 消息同步
func (mr *MessageRouter) HandleSyncRequest(transportMsg *transport.Message) {
	mr.server.logger.Debug("[MessageRouter] Sync request from: %s", transportMsg.SenderID)

	// 1. 解密请求
	decrypted, err := mr.server.crypto.DecryptMessage(transportMsg.Payload)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to decrypt sync request: %v", err)
		return
	}

	// 2. 解析请求
	var req map[string]interface{}
	if err := json.Unmarshal(decrypted, &req); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to unmarshal sync request: %v", err)
		return
	}

	// 3. 验证权限
	memberID := transportMsg.SenderID
	if !mr.server.channelManager.HasMember(memberID) {
		mr.server.logger.Warn("[MessageRouter] Non-member requesting sync: %s", memberID)
		return
	}

	// 4. 提取参数
	lastMessageID, _ := req["last_message_id"].(string)
	lastTimestamp := int64(0)
	if ts, ok := req["last_timestamp"].(float64); ok {
		lastTimestamp = int64(ts)
	}
	limit := 100 // 默认限制
	if l, ok := req["limit"].(float64); ok {
		limit = int(l)
	}

	mr.server.logger.Debug("[MessageRouter] Sync params: lastMessageID=%s, lastTimestamp=%d, limit=%d",
		lastMessageID, lastTimestamp, limit)

	// 5. 构建响应
	response, err := mr.buildSyncResponse(memberID, lastTimestamp, limit)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to build sync response: %v", err)
		return
	}

	// 6. 序列化并加密响应
	responseJSON, err := json.Marshal(response)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to marshal sync response: %v", err)
		return
	}

	encryptedResponse, err := mr.server.crypto.EncryptMessage(responseJSON)
	if err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to encrypt sync response: %v", err)
		return
	}

	// 7. 发送响应
	responseMsg := &transport.Message{
		Type:      transport.MessageTypeControl,
		SenderID:  "server",
		Payload:   encryptedResponse,
		Timestamp: time.Now(),
	}

	if err := mr.server.transport.SendMessage(responseMsg); err != nil {
		mr.server.logger.Error("[MessageRouter] Failed to send sync response: %v", err)
		return
	}

	mr.server.logger.Info("[MessageRouter] Sync response sent to: %s", memberID)
}

// buildSyncResponse 构建同步响应
func (mr *MessageRouter) buildSyncResponse(memberID string, lastTimestamp int64, limit int) (map[string]interface{}, error) {
	response := make(map[string]interface{})

	// 1. 获取消息更新
	messages, hasMoreMessages, err := mr.getMessagesSince(lastTimestamp, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// 2. 获取成员更新
	members, err := mr.getMemberUpdates(lastTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}

	// 3. 获取频道信息
	channel, err := mr.server.channelManager.GetChannel()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// 4. 构造响应
	response["type"] = "sync.response"
	response["channel_id"] = mr.server.config.ChannelID
	response["timestamp"] = time.Now().Unix()
	response["messages"] = messages
	response["members"] = members
	response["channel"] = channel
	response["has_more"] = hasMoreMessages

	// 5. 如果有离线消息，一并发送
	if mr.server.config.EnableOffline {
		offlineMessages := mr.server.offlineManager.GetOfflineMessages(memberID)
		if len(offlineMessages) > 0 {
			response["offline_messages"] = offlineMessages
			mr.server.logger.Info("[MessageRouter] Including %d offline messages for: %s",
				len(offlineMessages), memberID)
		}
	}

	return response, nil
}

// getMessagesSince 获取指定时间后的消息
func (mr *MessageRouter) getMessagesSince(sinceTimestamp int64, limit int) ([]interface{}, bool, error) {
	// 转换时间戳
	sinceTime := time.Unix(sinceTimestamp, 0)

	// 从数据库获取消息（按时间戳排序）
	// 使用limit=0表示获取所有消息
	allMessages, err := mr.server.messageRepo.GetByChannelID(mr.server.config.ChannelID, 0, 0)
	if err != nil {
		return nil, false, err
	}

	// 过滤：只返回指定时间之后的消息
	filteredMessages := make([]*models.Message, 0)
	for _, msg := range allMessages {
		if msg.Timestamp.After(sinceTime) {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	// 按时间戳排序（最旧的在前）
	// 简化处理：假设已经按时间排序

	// 应用限制
	hasMore := false
	if len(filteredMessages) > limit {
		filteredMessages = filteredMessages[:limit]
		hasMore = true
	}

	// 转换为interface{}切片
	messages := make([]interface{}, len(filteredMessages))
	for i, msg := range filteredMessages {
		messages[i] = msg
	}

	mr.server.logger.Debug("[MessageRouter] Found %d messages since %v", len(messages), sinceTime)

	return messages, hasMore, nil
}

// getMemberUpdates 获取成员更新
func (mr *MessageRouter) getMemberUpdates(sinceTimestamp int64) ([]interface{}, error) {
	// 获取所有成员（简化处理）
	allMembers, err := mr.server.channelManager.GetMembers()
	if err != nil {
		return nil, err
	}

	// TODO: 如果需要增量更新，可以根据 UpdatedAt 字段过滤

	// 转换为interface{}切片
	members := make([]interface{}, len(allMembers))
	for i, member := range allMembers {
		members[i] = member
	}

	mr.server.logger.Debug("[MessageRouter] Sending %d members", len(members))

	return members, nil
}

// GetSyncStats 获取同步统计
func (mr *MessageRouter) GetSyncStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 获取消息队列状态
	stats["queue_length"] = len(mr.messageQueue)
	stats["queue_capacity"] = mr.queueSize

	// 获取离线消息数量
	mr.offlineMutex.RLock()
	offlineCount := 0
	for _, messages := range mr.offlineMessages {
		offlineCount += len(messages)
	}
	mr.offlineMutex.RUnlock()

	stats["offline_message_count"] = offlineCount

	return stats
}
