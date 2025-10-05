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
	if err := s.initTransport(); err != nil {
		return fmt.Errorf("failed to init transport: %w", err)
	}

	// 加载或创建频道
	if err := s.channelManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize channel: %w", err)
	}

	// 启动传输层
	if err := s.transport.Start(); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	// 订阅传输层消息
	if err := s.transport.Subscribe(s.handleIncomingMessage); err != nil {
		return fmt.Errorf("failed to subscribe to transport: %w", err)
	}

	// 启动子模块
	s.wg.Add(1)
	go s.broadcastManager.Run()

	s.wg.Add(1)
	go s.messageRouter.Run()

	// 启动离线消息管理器
	if err := s.offlineManager.Start(); err != nil {
		return fmt.Errorf("failed to start offline manager: %w", err)
	}

	s.wg.Add(1)
	go s.statsReporter()

	// 发布服务信息（供客户端发现）
	if err := s.announceService(); err != nil {
		s.logger.Warn("[Server] Failed to announce service: %v", err)
		// 不阻止启动，只是警告
	} else {
		s.logger.Info("[Server] Service announced successfully")
	}

	s.logger.Info("[Server] Server started successfully")

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

	s.transport = t
	return nil
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
func (s *Server) GetStats() ServerStats {
	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()

	// 复制统计数据（避免复制锁）
	stats := ServerStats{
		StartTime:        s.startTime,
		TotalMembers:     s.channelManager.GetTotalCount(),
		OnlineMembers:    s.channelManager.GetOnlineCount(),
		TotalMessages:    s.stats.TotalMessages,
		TotalBroadcasts:  s.stats.TotalBroadcasts,
		TotalBytes:       s.stats.TotalBytes,
		DroppedMessages:  s.stats.DroppedMessages,
		RejectedMessages: s.stats.RejectedMessages,
	}

	return stats
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

// TODO: 实现以下功能
// - 消息持久化到离线队列
// - 消息确认（ACK）机制
// - 频率限制细节
// - 反垃圾消息
// - 成员踢出与封禁
// - 权限分级（管理员/普通成员）
