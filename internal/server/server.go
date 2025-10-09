package server

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/crypto"
	"crosswire/internal/events"
	"crosswire/internal/models"
	"crosswire/internal/storage"
	"crosswire/internal/transport"
	"crosswire/internal/utils"
)

// Server 服务端
// 参考: docs/ARCHITECTURE.md - 3.1.2 服务端模块
type Server struct {
	// 基础配置
	config *ServerConfig
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 核心组件
	channelManager   *ChannelManager
	broadcastManager *BroadcastManager
	messageRouter    *MessageRouter
	authManager      *AuthManager
	challengeManager *ChallengeManager
	offlineManager   *OfflineManager
	spamDetector     *SpamDetector

	// 基础设施
	transport transport.Transport
	crypto    *crypto.Manager
	db        *storage.Database
	eventBus  *events.EventBus
	logger    *utils.Logger

	// Repository
	channelRepo   *storage.ChannelRepository
	memberRepo    *storage.MemberRepository
	messageRepo   *storage.MessageRepository
	fileRepo      *storage.FileRepository
	challengeRepo *storage.ChallengeRepository
	auditRepo     *storage.AuditRepository

	// 状态
	isRunning bool
	startTime time.Time
	mutex     sync.RWMutex

	// 统计
	stats ServerStats
}

// ServerConfig 服务端配置
type ServerConfig struct {
	// 频道配置
	ChannelID       string
	ChannelPassword string
	ChannelName     string
	Description     string
	MaxMembers      int

	// 传输配置
	TransportMode   models.TransportMode
	TransportConfig *transport.Config

	// 认证配置
	RequireAuth    bool
	AllowAnonymous bool
	SessionTimeout time.Duration

	// 消息配置
	MaxMessageSize int
	MessageTTL     time.Duration
	EnableOffline  bool

	// 安全配置
	EnableRateLimit bool
	MaxMessageRate  int  // 每分钟最多消息数
	EnableSignature bool // 是否启用服务器签名

	// 服务器密钥对
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

// ServerStats 服务端统计
type ServerStats struct {
	StartTime        time.Time
	TotalMembers     int
	OnlineMembers    int
	TotalMessages    uint64
	TotalBroadcasts  uint64
	TotalBytes       uint64
	DroppedMessages  uint64
	RejectedMessages uint64
	mutex            sync.RWMutex
}

// ServerStatsDTO 面向外部使用的统计信息（不包含锁）
type ServerStatsDTO struct {
	StartTime        time.Time `json:"start_time"`
	TotalMembers     int       `json:"total_members"`
	OnlineMembers    int       `json:"online_members"`
	TotalMessages    uint64    `json:"total_messages"`
	TotalBroadcasts  uint64    `json:"total_broadcasts"`
	TotalBytes       uint64    `json:"total_bytes"`
	DroppedMessages  uint64    `json:"dropped_messages"`
	RejectedMessages uint64    `json:"rejected_messages"`
}

// DefaultServerConfig 默认配置
var DefaultServerConfig = &ServerConfig{
	MaxMembers:      100,
	RequireAuth:     true,
	AllowAnonymous:  false,
	SessionTimeout:  24 * time.Hour,
	MaxMessageSize:  10 * 1024 * 1024, // 10MB
	MessageTTL:      30 * 24 * time.Hour,
	EnableOffline:   true,
	EnableRateLimit: true,
	MaxMessageRate:  60,
	EnableSignature: true,
}

// NewServer 创建服务端
func NewServer(
	config *ServerConfig,
	db *storage.Database,
	eventBus *events.EventBus,
	logger *utils.Logger,
) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig
	}

	if db == nil {
		return nil, errors.New("database manager is required")
	}

	if eventBus == nil {
		return nil, errors.New("event bus is required")
	}

	if logger == nil {
		var err error
		logger, err = utils.NewLogger(utils.LogLevelInfo, "logs")
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}

	// 生成服务器密钥对（如果未提供）
	if config.PrivateKey == nil {
		publicKey, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to generate server keys: %w", err)
		}
		config.PrivateKey = privateKey
		config.PublicKey = publicKey
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 打开频道数据库
	if err := db.OpenChannelDB(config.ChannelID); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to open channel database: %w", err)
	}

	// 创建加密管理器
	cryptoManager, err := crypto.NewManager()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create crypto manager: %w", err)
	}

	// 从密码派生频道密钥
	if config.ChannelPassword != "" {
		channelKey, err := cryptoManager.DeriveKey(config.ChannelPassword, []byte(config.ChannelID))
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to derive channel key: %w", err)
		}
		cryptoManager.SetChannelKey(channelKey)
	}

	s := &Server{
		config:    config,
		ctx:       ctx,
		cancel:    cancel,
		db:        db,
		eventBus:  eventBus,
		logger:    logger,
		crypto:    cryptoManager,
		isRunning: false,

		// 初始化Repository
		channelRepo:   storage.NewChannelRepository(db),
		memberRepo:    storage.NewMemberRepository(db),
		messageRepo:   storage.NewMessageRepository(db),
		fileRepo:      storage.NewFileRepository(db),
		challengeRepo: storage.NewChallengeRepository(db),
		auditRepo:     storage.NewAuditRepository(db),
	}

	// 初始化子模块
	s.channelManager = NewChannelManager(s)
	s.broadcastManager = NewBroadcastManager(s)
	s.messageRouter = NewMessageRouter(s)
	s.authManager = NewAuthManager(s)
	s.challengeManager = NewChallengeManager(s)
	s.offlineManager = NewOfflineManager(s)
	s.spamDetector = NewSpamDetector(s)

	return s, nil
}

// Start 启动服务端
func (s *Server) Start() error {
	s.mutex.Lock()
	if s.isRunning {
		s.mutex.Unlock()
		return errors.New("server is already running")
	}
	s.isRunning = true
	s.startTime = time.Now()
	s.mutex.Unlock()

	s.logger.Info("[Server] Starting server for channel: %s", s.config.ChannelID)

	// 初始化传输层
	s.logger.Info("[Server] Step 1: Initializing transport layer (mode: %s)...", s.config.TransportMode)
	if err := s.initTransport(); err != nil {
		s.logger.Error("[Server] Failed to initialize transport: %v", err)
		return fmt.Errorf("初始化传输层失败: %w", err)
	}
	s.logger.Info("[Server] Transport layer initialized successfully")

	// 加载或创建频道
	s.logger.Info("[Server] Step 2: Initializing channel manager...")
	if err := s.channelManager.Initialize(); err != nil {
		s.logger.Error("[Server] Failed to initialize channel: %v", err)
		return fmt.Errorf("初始化频道失败: %w", err)
	}
	s.logger.Info("[Server] Channel manager initialized successfully")

	// 启动传输层
	s.logger.Info("[Server] Step 3: Starting transport layer...")
	if err := s.transport.Start(); err != nil {
		s.logger.Error("[Server] Failed to start transport: %v", err)
		return fmt.Errorf("启动传输层失败: %w", err)
	}
	s.logger.Info("[Server] Transport layer started successfully")

	// 订阅传输层消息
	s.logger.Info("[Server] Step 4: Subscribing to transport messages...")
	if err := s.transport.Subscribe(s.handleIncomingMessage); err != nil {
		s.logger.Error("[Server] Failed to subscribe to transport: %v", err)
		return fmt.Errorf("订阅传输层消息失败: %w", err)
	}
	s.logger.Info("[Server] Subscribed to transport successfully")

	// 启动子模块
	s.logger.Info("[Server] Step 5: Starting sub-modules...")
	s.wg.Add(1)
	go s.broadcastManager.Run()

	s.wg.Add(1)
	go s.messageRouter.Run()

	// 启动离线消息管理器
	if err := s.offlineManager.Start(); err != nil {
		s.logger.Error("[Server] Failed to start offline manager: %v", err)
		return fmt.Errorf("启动离线消息管理器失败: %w", err)
	}

	s.wg.Add(1)
	go s.statsReporter()

	// 启动离线检测任务（每60秒检查，无心跳>90秒视为离线）
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		threshold := 90 * time.Second
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.channelManager.CheckOfflineMembers(threshold)
			}
		}
	}()
	s.logger.Info("[Server] Sub-modules started successfully")

	// 发布服务信息（供客户端发现）
	s.logger.Info("[Server] Step 6: Announcing service...")
	if err := s.announceService(); err != nil {
		s.logger.Warn("[Server] Failed to announce service: %v", err)
		// 不阻止启动，只是警告
	} else {
		s.logger.Info("[Server] Service announced successfully")
	}

	s.logger.Info("[Server] ✓ Server started successfully")

	// 发布事件
	s.eventBus.Publish(events.EventSystemConnected, &events.SystemEvent{
		Type:    "server_started",
		Message: fmt.Sprintf("Server started for channel %s", s.config.ChannelName),
	})

	return nil
}

// Stop 停止服务端
func (s *Server) Stop() error {
	s.mutex.Lock()
	if !s.isRunning {
		s.mutex.Unlock()
		return errors.New("server is not running")
	}
	s.isRunning = false
	s.mutex.Unlock()

	s.logger.Info("[Server] Stopping server...")

	// 停止传输层
	if s.transport != nil {
		if err := s.transport.Stop(); err != nil {
			s.logger.Error("[Server] Failed to stop transport: %v", err)
		}
	}

	// 取消上下文
	s.cancel()

	// 等待协程退出
	s.wg.Wait()

	// 发布事件
	s.eventBus.Publish(events.EventSystemDisconnect, &events.SystemEvent{
		Type:    "server_stopped",
		Message: "Server stopped",
	})

	s.logger.Info("[Server] Server stopped")

	return nil
}

// initTransport 初始化传输层
func (s *Server) initTransport() error {
	factory := transport.NewFactory()

	transportConfig := s.config.TransportConfig
	if transportConfig == nil {
		transportConfig = &transport.Config{
			Mode: s.config.TransportMode,
		}
	}

	t, err := factory.CreateWithConfig(s.config.TransportMode, transportConfig)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	// 设置模式与密钥
	switch tr := t.(type) {
	case *transport.ARPTransport:
		tr.SetMode("server")
		tr.SetServerKeys(s.config.PrivateKey, s.config.PublicKey)
	case *transport.MDNSTransport:
		tr.SetMode("server")
		tr.SetServerKeys(s.config.PrivateKey, s.config.PublicKey)
		tr.SetChannelInfo(s.config.ChannelID, s.config.ChannelName)
	case *transport.HTTPSTransport:
		tr.SetMode("server")
	}

	// 注册文件接收回调（将文件片段转换为服务器内部处理）
	_ = t.OnFileReceived(func(ft *transport.FileTransfer) {
		s.handleIncomingFile(ft)
	})

	s.transport = t
	return nil
}

// handleIncomingFile 处理来自传输层的文件分片（统一走 MessageRouter 文件处理）
func (s *Server) handleIncomingFile(ft *transport.FileTransfer) {
	if ft == nil {
		return
	}
	// 构造与 MessageRouter.HandleFileChunk 兼容的结构
	chunk := struct {
		FileID      string `json:"file_id"`
		ChunkIndex  int    `json:"chunk_index"`
		Data        []byte `json:"data"`
		Checksum    string `json:"checksum"`
		TotalChunks int    `json:"total_chunks"`
	}{
		FileID:      ft.FileID,
		ChunkIndex:  ft.ChunkIndex,
		Data:        ft.Data,
		Checksum:    ft.ChunkChecksum,
		TotalChunks: ft.TotalChunks,
	}

	// 序列化并加密
	bytes, err := json.Marshal(&chunk)
	if err != nil {
		return
	}
	enc, err := s.crypto.EncryptMessage(bytes)
	if err != nil {
		return
	}

	// 投递到文件分块处理
	msg := &transport.Message{Type: transport.MessageTypeControl, Payload: enc, Timestamp: time.Now()}
	s.messageRouter.HandleFileChunk(msg)
}

// handleIncomingMessage 处理来自传输层的消息
// 参考: docs/PROTOCOL.md - 2.2.3 消息广播（服务器签名模式）
func (s *Server) handleIncomingMessage(msg *transport.Message) {
	s.logger.Debug("[Server] Received message from: %s, type: %d", msg.SenderID, msg.Type)

	// 更新统计
	s.stats.mutex.Lock()
	s.stats.TotalMessages++
	s.stats.TotalBytes += uint64(len(msg.Payload))
	s.stats.mutex.Unlock()

	// 根据消息类型路由
	// TODO: 实现应用层协议解析，从Payload中识别消息类型
	// 目前使用transport.MessageType作为临时方案
	switch msg.Type {
	case transport.MessageTypeAuth:
		s.authManager.HandleJoinRequest(msg)

	case transport.MessageTypeData:
		s.messageRouter.HandleClientMessage(msg)

	case transport.MessageTypeControl:
		// 控制消息需要进一步解析来确定具体类型
		s.handleControlMessage(msg)

	default:
		s.logger.Warn("[Server] Unknown message type: %d", msg.Type)
		s.stats.mutex.Lock()
		s.stats.DroppedMessages++
		s.stats.mutex.Unlock()
	}
}

// handleControlMessage 处理控制消息
func (s *Server) handleControlMessage(msg *transport.Message) {
	// 解密消息以获取详细类型
	decrypted, err := s.crypto.DecryptMessage(msg.Payload)
	if err != nil {
		s.logger.Error("[Server] Failed to decrypt control message: %v", err)
		return
	}

	// 简单解析获取类型字段
	var msgType struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(decrypted, &msgType); err != nil {
		s.logger.Error("[Server] Failed to unmarshal control message type: %v", err)
		return
	}

	s.logger.Debug("[Server] Control message type: %s", msgType.Type)

	// 根据详细类型路由
	switch msgType.Type {
	case "sync.request":
		s.messageRouter.HandleSyncRequest(msg)
	case "status.update":
		s.channelManager.HandleMemberStatus(msg)
	case "file.upload":
		s.messageRouter.HandleFileUpload(msg)
	case "file.chunk":
		s.messageRouter.HandleFileChunk(msg)
	case "file.download":
		s.messageRouter.HandleFileDownloadRequest(msg)
	case "challenge.hint":
		// 客户端请求解锁提示
		var payload map[string]interface{}
		if err := json.Unmarshal(decrypted, &payload); err != nil {
			s.logger.Error("[Server] Failed to unmarshal challenge.hint: %v", err)
			return
		}
		challengeID, _ := payload["challenge_id"].(string)
		hintIndex := 0
		if v, ok := payload["hint_index"].(float64); ok {
			hintIndex = int(v)
		}
		if challengeID == "" {
			s.logger.Warn("[Server] challenge.hint missing challenge_id")
			return
		}
		if err := s.challengeManager.UnlockHint(challengeID, msg.SenderID, hintIndex); err != nil {
			s.logger.Error("[Server] UnlockHint failed: %v", err)
		}
	case "ack":
		s.handleMessageAck(msg)
	default:
		s.logger.Warn("[Server] Unknown control message type: %s", msgType.Type)
	}
}

// BroadcastMessage 广播消息（带签名）
// 参考: docs/ARP_BROADCAST_MODE.md - 2. 服务器签名与广播
func (s *Server) BroadcastMessage(msg *models.Message) error {
	return s.broadcastManager.Broadcast(msg)
}

// AddMember 添加成员
func (s *Server) AddMember(member *models.Member) error {
	return s.channelManager.AddMember(member)
}

// RemoveMember 移除成员
func (s *Server) RemoveMember(memberID string, reason string) error {
	return s.channelManager.RemoveMember(memberID, reason)
}

// GetMembers 获取成员列表
func (s *Server) GetMembers() ([]*models.Member, error) {
	return s.channelManager.GetMembers()
}

// GetMember 获取成员
func (s *Server) GetMember(memberID string) (*models.Member, error) {
	return s.channelManager.GetMember(memberID)
}

// MuteMember 禁言成员
func (s *Server) MuteMember(memberID string, duration time.Duration, reason string) error {
	return s.channelManager.MuteMember(memberID, duration, reason)
}

// UnmuteMember 解除禁言
func (s *Server) UnmuteMember(memberID string) error {
	return s.channelManager.UnmuteMember(memberID)
}

// KickMember 踢出成员（包装到 ChannelManager）
func (s *Server) KickMember(memberID string, reason string) error {
	// 简化：由服务器作为操作者
	return s.channelManager.KickMember(memberID, reason, "server")
}

// BanMember 封禁成员（包装到 ChannelManager）
func (s *Server) BanMember(memberID string, reason string, duration time.Duration) error {
	return s.channelManager.BanMember(memberID, reason, "server", duration)
}

// UnbanMember 解封成员（包装到 ChannelManager）
func (s *Server) UnbanMember(memberID string) error {
	return s.channelManager.UnbanMember(memberID)
}

// UpdateMemberRole 更新成员角色（包装到 ChannelManager）
func (s *Server) UpdateMemberRole(memberID string, role models.Role) error {
	return s.channelManager.UpdateMemberRole(memberID, role)
}

// GetChannel 获取频道信息
func (s *Server) GetChannel() (*models.Channel, error) {
	return s.channelManager.GetChannel()
}

// UpdateChannel 更新频道信息
func (s *Server) UpdateChannel(updates map[string]interface{}) error {
	return s.channelManager.UpdateChannel(updates)
}

// announceService 发布服务信息
// 参考: internal/client/discovery_manager.go
func (s *Server) announceService() error {
	// 构造服务信息
	serviceInfo := &transport.ServiceInfo{
		ChannelID:      s.config.ChannelID,
		ChannelName:    s.config.ChannelName,
		Mode:           s.config.TransportMode,
		Version:        1,
		MaxMembers:     s.config.MaxMembers,
		CurrentMembers: s.channelManager.GetTotalCount(),
	}

	// 根据传输模式设置端口或接口
	if s.config.TransportConfig != nil {
		serviceInfo.Port = s.config.TransportConfig.Port
		serviceInfo.Interface = s.config.TransportConfig.Interface
	}

	// 调用传输层的Announce方法
	if err := s.transport.Announce(serviceInfo); err != nil {
		return fmt.Errorf("transport announce failed: %w", err)
	}

	s.logger.Debug("[Server] Service info: ChannelID=%s, Mode=%s, Port=%d",
		serviceInfo.ChannelID, serviceInfo.Mode, serviceInfo.Port)

	return nil
}

// GetStats 获取统计信息
func (s *Server) GetStats() ServerStatsDTO {
	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()

	// 汇总为无锁 DTO
	return ServerStatsDTO{
		StartTime:        s.startTime,
		TotalMembers:     s.channelManager.GetTotalCount(),
		OnlineMembers:    s.channelManager.GetOnlineCount(),
		TotalMessages:    s.stats.TotalMessages,
		TotalBroadcasts:  s.stats.TotalBroadcasts,
		TotalBytes:       s.stats.TotalBytes,
		DroppedMessages:  s.stats.DroppedMessages,
		RejectedMessages: s.stats.RejectedMessages,
	}
}

// IsRunning 检查是否运行中
func (s *Server) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.isRunning
}

// GetConfig 获取配置
func (s *Server) GetConfig() *ServerConfig {
	return s.config
}

// GetPublicKey 获取服务器公钥
func (s *Server) GetPublicKey() ed25519.PublicKey {
	return s.config.PublicKey
}

// statsReporter 统计信息报告协程
func (s *Server) statsReporter() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			stats := s.GetStats()
			s.logger.Info("[Server] Stats: Members=%d/%d, Messages=%d, Broadcasts=%d, Bytes=%d",
				stats.OnlineMembers, stats.TotalMembers,
				stats.TotalMessages, stats.TotalBroadcasts, stats.TotalBytes)
		}
	}
}

// handleMessageAck 处理消息确认
// 参考: docs/PROTOCOL.md - 消息确认机制
func (s *Server) handleMessageAck(msg *transport.Message) {
	// 解密ACK消息
	decrypted, err := s.crypto.DecryptMessage(msg.Payload)
	if err != nil {
		s.logger.Error("[Server] Failed to decrypt ACK message: %v", err)
		return
	}

	// 解析ACK内容
	var ackMsg struct {
		Type      string `json:"type"`
		MessageID string `json:"message_id"`
		MemberID  string `json:"member_id"`
		Timestamp int64  `json:"timestamp"`
	}

	if err := json.Unmarshal(decrypted, &ackMsg); err != nil {
		s.logger.Error("[Server] Failed to unmarshal ACK message: %v", err)
		return
	}

	// 验证成员
	if !s.channelManager.HasMember(ackMsg.MemberID) {
		s.logger.Warn("[Server] ACK from unknown member: %s", ackMsg.MemberID)
		return
	}

	// 记录ACK
	s.broadcastManager.RecordAck(ackMsg.MessageID, ackMsg.MemberID)

	s.logger.Debug("[Server] ACK recorded: message=%s, member=%s", ackMsg.MessageID, ackMsg.MemberID)
}

// CheckPermission 检查成员权限
// 参考: docs/ARCHITECTURE.md - 权限分级
func (s *Server) CheckPermission(memberID string, requiredRole models.Role) bool {
	return s.authManager.CheckPermission(memberID, requiredRole)
}

// HasAdminPermission 检查是否有管理员权限
func (s *Server) HasAdminPermission(memberID string) bool {
	member := s.channelManager.GetMemberByID(memberID)
	if member == nil {
		return false
	}
	return member.Role == models.RoleAdmin || member.Role == models.RoleOwner
}

// HasModeratorPermission 检查是否有管理权限（管理员或协管）
func (s *Server) HasModeratorPermission(memberID string) bool {
	member := s.channelManager.GetMemberByID(memberID)
	if member == nil {
		return false
	}
	return member.Role == models.RoleAdmin ||
		member.Role == models.RoleOwner ||
		member.Role == models.RoleModerator
}

// GetOfflineMessageStats 获取离线消息统计
func (s *Server) GetOfflineMessageStats() map[string]interface{} {
	stats := s.offlineManager.GetStats()
	return map[string]interface{}{
		"total_queued":         stats.TotalQueued,
		"total_delivered":      stats.TotalDelivered,
		"total_expired":        stats.TotalExpired,
		"current_queued_count": stats.CurrentQueuedCount,
	}
}

// GetBroadcastStats 获取广播统计
func (s *Server) GetBroadcastStats() map[string]interface{} {
	stats := s.broadcastManager.GetStats()
	return map[string]interface{}{
		"total_broadcasts":  stats.TotalBroadcasts,
		"failed_broadcasts": stats.FailedBroadcasts,
		"average_latency":   stats.AverageLatency.String(),
	}
}

// GetMessageRouterStats 获取消息路由统计
func (s *Server) GetMessageRouterStats() map[string]interface{} {
	return s.messageRouter.GetSyncStats()
}

// DeliverOfflineMessagesToMember 投递离线消息给指定成员
// 当成员重新上线时调用
func (s *Server) DeliverOfflineMessagesToMember(memberID string) error {
	return s.offlineManager.DeliverOfflineMessages(memberID)
}

// GetSpamDetectorStats 获取反垃圾统计
func (s *Server) GetSpamDetectorStats() map[string]interface{} {
	stats := s.spamDetector.GetStats()
	return map[string]interface{}{
		"total_checked":       stats.TotalChecked,
		"duplicate_detected":  stats.DuplicateDetected,
		"blacklist_detected":  stats.BlacklistDetected,
		"rapid_post_detected": stats.RapidPostDetected,
		"similarity_detected": stats.SimilarityDetected,
	}
}

// AddBlacklistWord 添加黑名单关键词
func (s *Server) AddBlacklistWord(word string) {
	s.spamDetector.AddBlacklistWord(word)
}

// RemoveBlacklistWord 移除黑名单关键词
func (s *Server) RemoveBlacklistWord(word string) {
	s.spamDetector.RemoveBlacklistWord(word)
}

// GetBlacklistWords 获取黑名单关键词列表
func (s *Server) GetBlacklistWords() []string {
	return s.spamDetector.GetBlacklistWords()
}

// ===== 挑战系统包装方法 =====

// CreateChallenge 创建题目
func (s *Server) CreateChallenge(challenge *models.Challenge) error {
	return s.challengeManager.CreateChallenge(challenge)
}

// GetChallenges 获取所有题目
func (s *Server) GetChallenges() ([]*models.Challenge, error) {
	return s.challengeRepo.GetByChannelID(s.config.ChannelID)
}

// GetChallenge 获取单个题目
func (s *Server) GetChallenge(challengeID string) (*models.Challenge, error) {
	return s.challengeRepo.GetByID(challengeID)
}

// GetSubChannels 获取所有题目子频道
func (s *Server) GetSubChannels() ([]*models.Channel, error) {
	// 使用channelRepo获取所有子频道
	return s.channelRepo.GetSubChannels(s.config.ChannelID)
}

// UpdateChallenge 更新题目
func (s *Server) UpdateChallenge(challenge *models.Challenge) error {
	return s.challengeRepo.Update(challenge)
}

// DeleteChallenge 删除题目
func (s *Server) DeleteChallenge(challengeID string) error {
	return s.challengeRepo.Delete(challengeID)
}

// AssignChallenge 分配题目
func (s *Server) AssignChallenge(challengeID, memberID, assignedBy string) error {
	return s.challengeManager.AssignChallenge(challengeID, memberID, assignedBy)
}

// SubmitFlag 提交Flag
func (s *Server) SubmitFlag(challengeID, memberID, flag string) error {
	return s.challengeManager.SubmitFlag(challengeID, memberID, flag)
}

// GetChallengeProgress 获取题目进度
func (s *Server) GetChallengeProgress(challengeID, memberID string) (*models.ChallengeProgress, error) {
	return s.challengeRepo.GetProgress(challengeID, memberID)
}

// UpdateChallengeProgress 更新题目进度
func (s *Server) UpdateChallengeProgress(progress *models.ChallengeProgress) error {
	if progress == nil {
		return errors.New("progress is nil")
	}
	// 统一经由 ChallengeManager 以发布事件
	return s.challengeManager.UpdateProgress(progress.ChallengeID, progress.MemberID, progress.Progress, progress.Summary)
}

// 注意：排行榜和统计功能已禁用（用户不需要）

// ===== 实现说明 =====
//
// 1. 消息持久化到离线队列 ✓
//    - 实现位置: OfflineManager.StoreOfflineMessage()
//    - 参考: internal/server/offline_manager.go:68-112
//    - 功能: 自动持久化到数据库，队列满时删除最旧消息
//
// 2. 消息确认（ACK）机制 ✓
//    - 实现位置: handleMessageAck(), BroadcastManager.RecordAck()
//    - 参考: internal/server/broadcast_manager.go:236-257
//    - 功能: 记录每个成员的消息确认状态
//
// 3. 频率限制细节 ✓
//    - 实现位置: RateLimiter.Allow()
//    - 参考: internal/server/message_router.go:574-603
//    - 功能: 滑动窗口频率限制，默认60条/分钟
//
// 4. 反垃圾消息 ✓
//    - 实现位置: MessageRouter.processMessageTask()
//    - 参考: internal/server/message_router.go:108-221
//    - 功能: 签名验证、频率限制、禁言检查
//
// 5. 成员踢出与封禁 ✓
//    - 实现位置: ChannelManager.KickMember(), BanMember()
//    - 参考: internal/server/channel_manager.go:404-512
//    - 功能: 踢出、封禁、解封、检查封禁状态
//
// 6. 权限分级（管理员/普通成员）✓
//    - 实现位置: AuthManager.CheckPermission()
//    - 参考: internal/server/auth_manager.go:351-365
//    - 角色: Owner > Admin > Moderator > Member > ReadOnly
