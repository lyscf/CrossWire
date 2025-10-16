package client

import (
	"crypto/sha256"
	"encoding/base64"
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

	// 订阅已由 client.initTransport 阶段完成，这里不重复订阅

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
	var decrypted []byte
	// 优先尝试解析服务器签名载荷（BroadcastManager 格式）
	var serverSigned struct {
		Message   []byte `json:"message"`
		Signature []byte `json:"signature"`
		Timestamp int64  `json:"timestamp"`
		ServerID  string `json:"server_id"`
	}
	if err := json.Unmarshal(msg.Payload, &serverSigned); err == nil && len(serverSigned.Message) > 0 {
		// 可选：此处可校验服务器签名（若已设置 server public key），当前先解密载荷
		plain, derr := rm.client.crypto.DecryptMessage(serverSigned.Message)
		if derr != nil {
			rm.client.logger.Warn("[ReceiveManager] Failed to decrypt signed payload: %v", derr)
			rm.stats.mutex.Lock()
			rm.stats.DecryptFailures++
			rm.stats.mutex.Unlock()
			return
		}
		decrypted = plain
	} else {
		// 回退：旧格式，直接解密传入负载
		plain, derr := rm.client.crypto.DecryptMessage(msg.Payload)
		if derr != nil {
			rm.client.logger.Warn("[ReceiveManager] Failed to decrypt message: %v", derr)
			rm.stats.mutex.Lock()
			rm.stats.DecryptFailures++
			rm.stats.mutex.Unlock()
			return
		}
		decrypted = plain
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

	// 同步更新客户端 ChannelID（若响应中提供）
	if chID, ok := payload["channel_id"].(string); ok && chID != "" && chID != rm.client.config.ChannelID {
		rm.client.logger.Info("[ReceiveManager] Updating client ChannelID from response: %s", chID)
		rm.client.config.ChannelID = chID

		// 重新打开正确的频道数据库并刷新仓库，确保后续成员与消息落到正确库
		if err := rm.client.db.OpenChannelDB(chID); err != nil {
			rm.client.logger.Error("[ReceiveManager] Failed to reopen channel DB for %s: %v", chID, err)
		} else {
			if err := rm.client.initRepositories(); err != nil {
				rm.client.logger.Error("[ReceiveManager] Failed to re-init repositories after channel switch: %v", err)
			}
		}
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

	// 若响应携带服务器公钥，则在 ARP 模式下启用广播验签
	if pk, ok := payload["server_public_key"].([]byte); ok && len(pk) > 0 {
		if rm.client.transport != nil && rm.client.transport.GetMode() == transport.TransportModeARP {
			rm.client.SetServerPublicKey(pk)
			rm.client.logger.Info("[ReceiveManager] Server public key received and set for ARP signature verification")
		}
	}

	// 更新连接时间
	rm.client.stats.mutex.Lock()
	rm.client.stats.ConnectedAt = time.Now()
	rm.client.stats.mutex.Unlock()

	rm.client.logger.Info("[ReceiveManager] Successfully joined channel as: %s", memberID)

	// 尝试构造成员对象
	member := &models.Member{ID: memberID, ChannelID: rm.client.config.ChannelID, Nickname: "", Status: models.StatusOnline}
	if nickname, ok := memberData["nickname"].(string); ok {
		member.Nickname = nickname
	}
	// 本地持久化我的成员信息，供 GetMyInfo 使用
	if existing, err := rm.client.memberRepo.GetByID(memberID); err == nil && existing != nil {
		existing.Nickname = member.Nickname
		existing.Status = models.StatusOnline
		existing.ChannelID = rm.client.config.ChannelID
		_ = rm.client.memberRepo.Update(existing)
	} else {
		// 初始化必要字段
		if member.JoinTime.IsZero() {
			member.JoinTime = time.Now()
		}
		if member.LastSeenAt.IsZero() {
			member.LastSeenAt = time.Now()
		}
		_ = rm.client.memberRepo.Create(member)
	}

	// 持久化 member_list（如果服务端提供）
	if list, ok := payload["member_list"].([]interface{}); ok {
		for _, it := range list {
			if m, ok2 := it.(map[string]interface{}); ok2 {
				mid, _ := m["id"].(string)
				nick, _ := m["nickname"].(string)
				roleStr, _ := m["role"].(string)
				statusStr, _ := m["status"].(string)
				if mid == "" {
					continue
				}
				rec := &models.Member{ID: mid, ChannelID: rm.client.config.ChannelID, Nickname: nick}
				if roleStr != "" {
					rec.Role = models.Role(roleStr)
				}
				if statusStr != "" {
					rec.Status = models.UserStatus(statusStr)
				}
				if exist, err := rm.client.memberRepo.GetByID(mid); err == nil && exist != nil {
					exist.Nickname = rec.Nickname
					if rec.Role != "" {
						exist.Role = rec.Role
					}
					if rec.Status != "" {
						exist.Status = rec.Status
					}
					_ = rm.client.memberRepo.Update(exist)
				} else {
					if rec.JoinedAt.IsZero() {
						rec.JoinedAt = time.Now()
					}
					if rec.LastSeenAt.IsZero() {
						rec.LastSeenAt = time.Now()
					}
					_ = rm.client.memberRepo.Create(rec)
				}
			}
		}
	}

	// 发布加入成功事件
	rm.client.eventBus.Publish(events.EventMemberJoined, &events.MemberEvent{
		Member:    member,
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
	// 1) 首先尝试解析为通用payload以识别文件相关消息
	var generic map[string]interface{}
	if err := json.Unmarshal(data, &generic); err == nil {
		if t, ok := generic["type"].(string); ok {
			switch t {
			case "file.metadata":
				rm.handleFileMetadata(generic)
				return
			case "file.chunk":
				rm.handleFileChunk(generic)
				return
			}
		}
	}

	// 2) 否则按普通聊天/系统消息处理（兼容旧格式）
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
	// 不再忽略自身消息：需要本地持久化以保证发送后立即可见

	// 5.5 识别内嵌系统事件：challenge_created / challenge_assigned / challenge_solved
	if msg.Type == models.MessageTypeSystem {
		if ev, ok := msg.Content["event"].(string); ok {
			switch ev {
			case "challenge_created":
				// 从extra构造Challenge最小字段
				extra, _ := msg.Content["extra"].(map[string]interface{})
				var ch models.Challenge
				if extra != nil {
					ch.ID, _ = extra["challenge_id"].(string)
					ch.ChannelID = rm.client.GetChannelID()
					ch.SubChannelID, _ = extra["sub_channel_id"].(string)
					ch.Title, _ = extra["title"].(string)
					ch.Category, _ = extra["category"].(string)
					ch.Difficulty, _ = extra["difficulty"].(string)
					if pts, ok2 := extra["points"].(float64); ok2 {
						ch.Points = int(pts)
					}
					if desc, ok2 := extra["message"].(string); ok2 {
						ch.Description = desc
					}
					ch.Status = "open"
					ch.CreatedBy = "server"
					ch.CreatedAt = time.Now()
					ch.UpdatedAt = time.Now()
				}
				if ch.ID != "" && rm.client.challengeRepo != nil {
					if existing, err := rm.client.challengeRepo.GetByID(ch.ID); err == nil && existing != nil {
						ch.CreatedAt = existing.CreatedAt
						_ = rm.client.challengeRepo.Update(&ch)
					} else {
						_ = rm.client.challengeRepo.Create(&ch)
					}
					rm.client.eventBus.Publish(events.EventChallengeCreated, &events.ChallengeEvent{
						Challenge: &ch,
						Action:    "created",
						UserID:    "server",
						ChannelID: rm.client.GetChannelID(),
						ExtraData: nil,
					})
					if ch.SubChannelID != "" {
						go rm.client.challengeManager.syncSubChannel(ch.SubChannelID)
					}
				}
			case "challenge_assigned":
				extra, _ := msg.Content["extra"].(map[string]interface{})
				challengeID, _ := extra["challenge_id"].(string)
				assigneeID, _ := extra["assignee_id"].(string)
				if challengeID != "" && rm.client.challengeRepo != nil {
					// 更新 AssignedTo + 写入分配关系
					ch, _ := rm.client.challengeRepo.GetByID(challengeID)
					if ch == nil {
						ch = &models.Challenge{ID: challengeID, ChannelID: rm.client.GetChannelID()}
					}
					exists := false
					for _, id := range ch.AssignedTo {
						if id == assigneeID {
							exists = true
							break
						}
					}
					if assigneeID != "" && !exists {
						ch.AssignedTo = append(ch.AssignedTo, assigneeID)
					}
					_ = rm.client.challengeRepo.Update(ch)
					if assigneeID != "" {
						_ = rm.client.challengeRepo.AssignChallenge(&models.ChallengeAssignment{
							ChallengeID: challengeID,
							MemberID:    assigneeID,
							AssignedBy:  "server",
							AssignedAt:  time.Now(),
							Status:      "assigned",
						})
					}
					rm.client.eventBus.Publish(events.EventChallengeAssigned, &events.ChallengeEvent{
						Challenge: ch,
						Action:    "assigned",
						UserID:    assigneeID,
						ChannelID: rm.client.GetChannelID(),
						ExtraData: nil,
					})
				}
			case "challenge_solved":
				extra, _ := msg.Content["extra"].(map[string]interface{})
				challengeID, _ := extra["challenge_id"].(string)
				solverName, _ := extra["nickname"].(string)
				// actor_id 也可能是 memberID
				actorID, _ := msg.Content["actor_id"].(string)
				if challengeID != "" && rm.client.challengeRepo != nil {
					ch, _ := rm.client.challengeRepo.GetByID(challengeID)
					if ch == nil {
						ch = &models.Challenge{ID: challengeID, ChannelID: rm.client.GetChannelID()}
					}
					if actorID != "" {
						ch.SolvedBy = append(ch.SolvedBy, actorID)
					}
					ch.Status = "solved"
					ch.SolvedAt = time.Now()
					// 若广播携带 flag，则回写到本地题目，确保刷新后仍可见
					if f, ok2 := extra["flag"].(string); ok2 && f != "" {
						ch.Flag = f
					}
					_ = rm.client.challengeRepo.Update(ch)
					if actorID != "" {
						_ = rm.client.challengeRepo.UpdateProgress(&models.ChallengeProgress{
							ChallengeID: challengeID,
							MemberID:    actorID,
							Status:      "solved",
							Progress:    100,
							UpdatedAt:   time.Now(),
						})
					}
					// 发布事件（使用 actorID 作为 UserID，solverName 仅作展示信息）
					rm.client.eventBus.Publish(events.EventChallengeSolved, &events.ChallengeEvent{
						Challenge: ch,
						Action:    "solved",
						UserID:    actorID,
						ChannelID: rm.client.GetChannelID(),
						ExtraData: map[string]string{"nickname": solverName},
					})
				}
			}
		}
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
	case "sync.response":
		// 同步响应：交给 SyncManager 处理
		rm.client.syncManager.HandleSyncResponse(data)

	case "member.status":
		// 成员状态更新
		rm.handleMemberStatus(payload)

	case "member.joined":
		// 成员加入通知
		rm.handleMemberJoined(payload)

	case "member.left":
		// 成员离开通知
		rm.handleMemberLeft(payload)

	case "file.request":
		// 文件下载请求
		rm.handleFileRequest(payload)

	case "file.complete":
		// 文件上传完成通知
		rm.handleFileComplete(payload)

	default:
		rm.client.logger.Debug("[ReceiveManager] Unknown control message: %s", msgType)
	}
}

// ===== 文件消息处理 =====

// handleFileMetadata 处理文件元数据
func (rm *ReceiveManager) handleFileMetadata(data map[string]interface{}) {
	fileID, _ := data["file_id"].(string)
	filename, _ := data["filename"].(string)
	sizeFloat, _ := data["size"].(float64)
	sha256Hex, _ := data["sha256"].(string)
	chunkSizeFloat, _ := data["chunk_size"].(float64)
	totalChunksFloat, _ := data["total_chunks"].(float64)

	size := int64(sizeFloat)
	chunkSize := int(chunkSizeFloat)
	totalChunks := int(totalChunksFloat)

	rm.client.logger.Info("[ReceiveManager] File metadata received: %s (%s, %d bytes, %d chunks)",
		fileID, filename, size, totalChunks)

	// 创建或更新文件记录
	fileRecord := &models.File{
		ID:             fileID,
		MessageID:      fileID,
		ChannelID:      rm.client.config.ChannelID,
		SenderID:       "",
		Filename:       filename,
		OriginalName:   filename,
		Size:           size,
		MimeType:       "application/octet-stream",
		StorageType:    models.StorageFile,
		StoragePath:    "",
		SHA256:         sha256Hex,
		Checksum:       "",
		ChunkSize:      chunkSize,
		TotalChunks:    totalChunks,
		UploadedChunks: 0,
		UploadStatus:   models.UploadStatusUploading,
		UploadedAt:     time.Now(),
		ExpiresAt:      time.Time{},
		Encrypted:      true,
	}

	if existing, err := rm.client.fileRepo.GetByID(fileID); err == nil && existing != nil {
		// 更新基本字段
		existing.Filename = filename
		existing.Size = size
		existing.SHA256 = sha256Hex
		existing.ChunkSize = chunkSize
		existing.TotalChunks = totalChunks
		existing.UploadStatus = models.UploadStatusUploading
		_ = rm.client.fileRepo.Update(existing)
	} else {
		_ = rm.client.fileRepo.Create(fileRecord)
	}

	// 发布上传开始/进度事件（0%）
	rm.client.eventBus.Publish(events.EventFileDownloadStarted, events.FileEvent{
		File:       fileRecord,
		ChannelID:  rm.client.config.ChannelID,
		UploaderID: "",
		Progress:   0,
	})
}

// handleFileChunk 处理文件分块
func (rm *ReceiveManager) handleFileChunk(data map[string]interface{}) {
	fileID, _ := data["file_id"].(string)
	chunkIndex := int(getFloat(data, "chunk_index"))
	totalChunks := int(getFloat(data, "total_chunks"))
	checksum, _ := data["checksum"].(string)

	// Base64 解码分块数据
	chunkDataB64, _ := data["data"].(string)
	chunkData, err := base64.StdEncoding.DecodeString(chunkDataB64)
	if err != nil {
		rm.client.logger.Error("[ReceiveManager] Failed to decode chunk data: %v", err)
		return
	}

	// 校验分块哈希
	actualChecksum := fmt.Sprintf("%x", sha256.Sum256(chunkData))
	if checksum != "" && actualChecksum != checksum {
		rm.client.logger.Error("[ReceiveManager] Chunk checksum mismatch: %s != %s", actualChecksum, checksum)
		return
	}

	// 查找对应下载任务
	task, ok := rm.client.fileManager.GetDownloadTaskByFileID(fileID)
	if !ok {
		rm.client.logger.Debug("[ReceiveManager] No download task for file %s, ignoring chunk", fileID)
		return
	}

	// 添加分块
	task.chunksMutex.Lock()
	if task.chunks == nil {
		task.chunks = make(map[int][]byte)
	}
	task.chunks[chunkIndex] = chunkData
	task.ReceivedChunks = len(task.chunks)
	task.chunksMutex.Unlock()

	// 触发回调
	if task.OnProgress != nil {
		task.OnProgress(task)
	}

	// 发布下载进度事件
	progress := int(float64(task.ReceivedChunks) / float64(maxInt(task.TotalChunks, 1)) * 100)
	var filePtr *models.File
	if f, err := rm.client.fileRepo.GetByID(fileID); err == nil {
		filePtr = f
	}

	rm.client.eventBus.Publish(events.EventFileDownloadProgress, events.FileEvent{
		File:       filePtr,
		ChannelID:  rm.client.config.ChannelID,
		UploaderID: "",
		Progress:   progress,
	})

	rm.client.logger.Debug("[ReceiveManager] File download progress: %s [%d/%d] (%d%%)",
		fileID, task.ReceivedChunks, totalChunks, progress)
}

// handleFileComplete 处理文件完成消息
func (rm *ReceiveManager) handleFileComplete(data map[string]interface{}) {
	fileID, _ := data["file_id"].(string)

	// 更新文件状态
	if file, err := rm.client.fileRepo.GetByID(fileID); err == nil && file != nil {
		file.UploadStatus = models.UploadStatusCompleted
		_ = rm.client.fileRepo.Update(file)
		rm.client.eventBus.Publish(events.EventFileDownloadCompleted, events.FileEvent{
			File:       file,
			ChannelID:  rm.client.config.ChannelID,
			UploaderID: "",
			Progress:   100,
		})
	}

	rm.client.logger.Info("[ReceiveManager] File upload completed (sender): %s", fileID)
}

// handleFileRequest 处理文件下载请求（对端请求我重新上传）
func (rm *ReceiveManager) handleFileRequest(data map[string]interface{}) {
	fileID, _ := data["file_id"].(string)
	requesterID, _ := data["requester_id"].(string)

	rm.client.logger.Info("[ReceiveManager] File request from %s: %s", requesterID, fileID)

	file, err := rm.client.fileRepo.GetByID(fileID)
	if err != nil || file == nil {
		rm.client.logger.Error("[ReceiveManager] Requested file not found: %s", fileID)
		return
	}

	if file.StoragePath == "" {
		rm.client.logger.Error("[ReceiveManager] File has no local path: %s", fileID)
		return
	}

	go func() {
		if _, err := rm.client.fileManager.UploadFile(file.StoragePath); err != nil {
			rm.client.logger.Error("[ReceiveManager] Failed to respond to file request: %v", err)
		}
	}()
}

// 工具函数
func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	// 解析成员信息并更新本地缓存
	memberID, _ := payload["member_id"].(string)
	nickname, _ := payload["nickname"].(string)
	roleStr, _ := payload["role"].(string)
	statusStr, _ := payload["status"].(string)

	member := &models.Member{
		ID:         memberID,
		ChannelID:  rm.client.config.ChannelID,
		Nickname:   nickname,
		Role:       models.Role(roleStr),
		Status:     models.UserStatus(statusStr),
		JoinedAt:   time.Now(),
		LastSeenAt: time.Now(),
	}
	// 尝试写入/更新数据库
	if memberID != "" {
		if existing, err := rm.client.memberRepo.GetByID(memberID); err == nil && existing != nil {
			existing.Nickname = member.Nickname
			existing.Role = member.Role
			existing.Status = member.Status
			_ = rm.client.memberRepo.Update(existing)
		} else {
			_ = rm.client.memberRepo.Create(member)
		}
	}

	rm.client.eventBus.Publish(events.EventMemberJoined, &events.MemberEvent{
		Member:    member,
		Action:    "joined",
		ChannelID: rm.client.config.ChannelID,
	})
}

// handleMemberLeft 处理成员离开通知
func (rm *ReceiveManager) handleMemberLeft(payload map[string]interface{}) {
	rm.client.logger.Debug("[ReceiveManager] Member left notification")
	// 解析成员信息并更新本地缓存
	memberID, _ := payload["member_id"].(string)
	if memberID != "" {
		// 标记离线
		_ = rm.client.memberRepo.UpdateStatus(memberID, models.StatusOffline)
	}

	rm.client.eventBus.Publish(events.EventMemberLeft, &events.MemberEvent{
		Member:    nil,
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
