package client

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
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

// Client 客户端核心
// 参考: docs/ARCHITECTURE.md - 3.1.3 客户端模块
type Client struct {
	config *Config

	// 核心组件
	transport transport.Transport
	crypto    *crypto.Manager
	db        *storage.Database
	eventBus  *events.EventBus
	logger    *utils.Logger

	// 仓库
	messageRepo   *storage.MessageRepository
	channelRepo   *storage.ChannelRepository
	memberRepo    *storage.MemberRepository
	fileRepo      *storage.FileRepository
	challengeRepo *storage.ChallengeRepository
	auditRepo     *storage.AuditRepository

	// 子管理器
	receiveManager    *ReceiveManager
	syncManager       *SyncManager
	cacheManager      *CacheManager
	fileManager       *FileManager
	discoveryManager  *DiscoveryManager
	offlineQueue      *OfflineQueue
	signatureVerifier *SignatureVerifier
	challengeManager  *ChallengeManager

	// 状态
	ctx           context.Context
	cancel        context.CancelFunc
	isRunning     bool
	mutex         sync.RWMutex
	startTime     time.Time
	memberID      string // 本地成员ID
	serverID      string // 服务端ID（频道ID）
	lastSeenMsgID string // 最后接收的消息ID

	// 密钥对（用于消息签名）
	privateKey []byte // Ed25519私钥
	publicKey  []byte // Ed25519公钥

	// 统计
	stats ClientStats
}

// Config 客户端配置
type Config struct {
	// 频道信息
	ChannelID       string
	ChannelPassword string

	// 用户信息
	Nickname string
	Avatar   string
	Role     models.Role

	// 传输配置
	TransportMode   models.TransportMode
	TransportConfig *transport.Config

	// 同步配置
	SyncInterval    time.Duration // 同步间隔
	MaxSyncMessages int           // 单次同步最大消息数

	// 缓存配置
	CacheSize     int           // 缓存大小（条数）
	CacheDuration time.Duration // 缓存有效期

	// 超时配置
	JoinTimeout time.Duration
	SyncTimeout time.Duration

	// 数据库路径
	DataDir string
}

// ClientStats 客户端统计信息
type ClientStats struct {
	StartTime        time.Time
	ConnectedAt      time.Time
	TotalReceived    uint64
	TotalSent        uint64
	MessagesReceived uint64
	MessagesSent     uint64
	FilesReceived    uint64
	FilesSent        uint64
	BytesReceived    uint64
	BytesSent        uint64
	SyncCount        uint64
	LastSyncTime     time.Time
	mutex            sync.RWMutex
}

// SignedMessage 带签名的消息（客户端发送使用）
type SignedMessage struct {
	Message   []byte `json:"message"`   // 原始消息JSON
	Signature []byte `json:"signature"` // Ed25519签名
	SenderID  string `json:"sender_id"` // 发送者ID
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		SyncInterval:    5 * time.Minute,
		MaxSyncMessages: 1000,
		CacheSize:       5000,
		CacheDuration:   24 * time.Hour,
		JoinTimeout:     30 * time.Second,
		SyncTimeout:     10 * time.Second,
		TransportMode:   models.TransportHTTPS,
	}
}

// NewClient 创建客户端
func NewClient(config *Config, db *storage.Database, eventBus *events.EventBus) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}

	if eventBus == nil {
		return nil, fmt.Errorf("eventBus is nil")
	}

	// 创建日志器
	logger, err := utils.NewLogger(utils.LogLevelDebug, config.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		config:    config,
		db:        db,
		eventBus:  eventBus,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		isRunning: false,
	}

	// 初始化加密管理器
	cryptoMgr, err := crypto.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto manager: %w", err)
	}
	c.crypto = cryptoMgr

	// 派生频道密钥
	channelKey, err := c.crypto.DeriveKey(config.ChannelPassword, []byte(config.ChannelID))
	if err != nil {
		return nil, fmt.Errorf("failed to derive channel key: %w", err)
	}
	c.crypto.SetChannelKey(channelKey)

	// 生成客户端密钥对（用于消息签名）
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}
	c.publicKey = publicKey
	c.privateKey = privateKey

	// 初始化仓库
	if err := c.initRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	// 初始化子管理器
	c.receiveManager = NewReceiveManager(c)
	c.syncManager = NewSyncManager(c)
	c.cacheManager = NewCacheManager(c)
	c.fileManager = NewFileManager(c)
	c.discoveryManager = NewDiscoveryManager(c)
	c.offlineQueue = NewOfflineQueue(c)
	c.signatureVerifier = NewSignatureVerifier(c)
	c.challengeManager = NewChallengeManager(c)

	c.logger.Info("[Client] Client created for channel: %s", config.ChannelID)

	return c, nil
}

// initRepositories 初始化仓库
func (c *Client) initRepositories() error {
	// 打开频道数据库
	if err := c.db.OpenChannelDB(c.config.ChannelID); err != nil {
		return fmt.Errorf("failed to open channel database: %w", err)
	}

	// 初始化仓库（传入整个Database对象）
	c.messageRepo = storage.NewMessageRepository(c.db)
	c.channelRepo = storage.NewChannelRepository(c.db)
	c.memberRepo = storage.NewMemberRepository(c.db)
	c.fileRepo = storage.NewFileRepository(c.db)
	c.challengeRepo = storage.NewChallengeRepository(c.db)
	c.auditRepo = storage.NewAuditRepository(c.db)

	return nil
}

// Start 启动客户端
func (c *Client) Start() error {
	c.mutex.Lock()
	if c.isRunning {
		c.mutex.Unlock()
		return fmt.Errorf("client is already running")
	}
	c.isRunning = true
	c.startTime = time.Now()
	c.mutex.Unlock()

	c.logger.Info("[Client] Starting client...")

	// 1. 初始化传输层
	if err := c.initTransport(); err != nil {
		return fmt.Errorf("failed to initialize transport: %w", err)
	}

	// 2. 加入频道（发送认证请求）
	if err := c.joinChannel(); err != nil {
		return fmt.Errorf("failed to join channel: %w", err)
	}

	// 3. 启动接收管理器
	if err := c.receiveManager.Start(); err != nil {
		return fmt.Errorf("failed to start receive manager: %w", err)
	}

	// 4. 启动同步管理器
	if err := c.syncManager.Start(); err != nil {
		return fmt.Errorf("failed to start sync manager: %w", err)
	}

	// 5. 启动缓存管理器
	if err := c.cacheManager.Start(); err != nil {
		return fmt.Errorf("failed to start cache manager: %w", err)
	}

	// 6. 启动文件管理器
	if err := c.fileManager.Start(); err != nil {
		return fmt.Errorf("failed to start file manager: %w", err)
	}

	// 7. 启动离线队列
	if err := c.offlineQueue.Start(); err != nil {
		return fmt.Errorf("failed to start offline queue: %w", err)
	}

	// 8. 启动挑战管理器
	if err := c.challengeManager.Start(); err != nil {
		return fmt.Errorf("failed to start challenge manager: %w", err)
	}

	c.logger.Info("[Client] Client started successfully")

	return nil
}

// Stop 停止客户端
func (c *Client) Stop() error {
	c.mutex.Lock()
	if !c.isRunning {
		c.mutex.Unlock()
		return nil
	}
	c.isRunning = false
	c.mutex.Unlock()

	c.logger.Info("[Client] Stopping client...")

	// 停止子管理器
	if c.challengeManager != nil {
		c.challengeManager.Stop()
	}
	if c.offlineQueue != nil {
		c.offlineQueue.Stop()
	}
	if c.fileManager != nil {
		c.fileManager.Stop()
	}
	if c.cacheManager != nil {
		c.cacheManager.Stop()
	}
	if c.syncManager != nil {
		c.syncManager.Stop()
	}
	if c.receiveManager != nil {
		c.receiveManager.Stop()
	}

	// 发送离开消息
	c.leaveChannel()

	// 停止传输层
	if c.transport != nil {
		c.transport.Stop()
	}

	// 取消上下文
	c.cancel()

	c.logger.Info("[Client] Client stopped")

	return nil
}

// initTransport 初始化传输层
func (c *Client) initTransport() error {
	c.logger.Debug("[Client] Initializing transport: %s", c.config.TransportMode)

	factory := transport.NewFactory()
	t, err := factory.CreateWithConfig(c.config.TransportMode, c.config.TransportConfig)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	c.transport = t

	if err := c.transport.Init(c.config.TransportConfig); err != nil {
		return fmt.Errorf("failed to initialize transport: %w", err)
	}

	if err := c.transport.Start(); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	return nil
}

// joinChannel 加入频道
func (c *Client) joinChannel() error {
	c.logger.Info("[Client] Joining channel: %s", c.config.ChannelID)

	// TODO: 生成临时的用户密钥对（用于接收加密的channel key）
	// 暂时跳过，直接使用密码派生的密钥

	// 构造加入请求
	joinReq := map[string]interface{}{
		"type":       "auth.join",
		"channel_id": c.config.ChannelID,
		"nickname":   c.config.Nickname,
		"avatar":     c.config.Avatar,
		"role":       c.config.Role,
		"public_key": c.publicKey, // 发送公钥用于验证签名
		"timestamp":  time.Now().Unix(),
	}

	// 序列化
	reqJSON, err := json.Marshal(joinReq)
	if err != nil {
		return fmt.Errorf("failed to marshal join request: %w", err)
	}

	reqData, err := c.crypto.EncryptMessage(reqJSON)
	if err != nil {
		return fmt.Errorf("failed to encrypt join request: %w", err)
	}

	// 发送
	msg := &transport.Message{
		Type:      transport.MessageTypeAuth,
		SenderID:  c.memberID,
		Payload:   reqData,
		Timestamp: time.Now(),
	}

	if err := c.transport.SendMessage(msg); err != nil {
		return fmt.Errorf("failed to send join request: %w", err)
	}

	c.logger.Info("[Client] Join request sent, waiting for response...")

	// TODO: 等待加入响应（通过receiveManager接收）
	// 暂时假设成功

	return nil
}

// leaveChannel 离开频道
func (c *Client) leaveChannel() {
	c.logger.Info("[Client] Leaving channel: %s", c.config.ChannelID)

	// 构造离开请求
	leaveReq := map[string]interface{}{
		"type":       "auth.leave",
		"channel_id": c.config.ChannelID,
		"member_id":  c.memberID,
		"timestamp":  time.Now().Unix(),
	}

	// 序列化并加密
	reqJSON, err := json.Marshal(leaveReq)
	if err != nil {
		c.logger.Error("[Client] Failed to marshal leave request: %v", err)
		return
	}

	reqData, err := c.crypto.EncryptMessage(reqJSON)
	if err != nil {
		c.logger.Error("[Client] Failed to encrypt leave request: %v", err)
		return
	}

	// 发送
	msg := &transport.Message{
		Type:      transport.MessageTypeControl,
		SenderID:  c.memberID,
		Payload:   reqData,
		Timestamp: time.Now(),
	}

	if err := c.transport.SendMessage(msg); err != nil {
		c.logger.Error("[Client] Failed to send leave request: %v", err)
	}
}

// SendMessage 发送消息
func (c *Client) SendMessage(content string, msgType models.MessageType) error {
	if !c.isRunning {
		return fmt.Errorf("client is not running")
	}

	// TODO: 根据msgType构造不同的MessageContent
	// 暂时简化处理

	// 构造消息
	msg := &models.Message{
		ID:        generateMessageID(),
		ChannelID: c.config.ChannelID,
		SenderID:  c.memberID,
		Type:      msgType,
		// Content:   content,
		Timestamp: time.Now(),
	}

	// 1. 序列化消息
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 2. 对消息进行签名
	signature := ed25519.Sign(c.privateKey, msgJSON)

	// 3. 构造带签名的消息
	signedMsg := &SignedMessage{
		Message:   msgJSON,
		Signature: signature,
		SenderID:  c.memberID,
	}

	// 4. 序列化签名消息
	signedJSON, err := json.Marshal(signedMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal signed message: %w", err)
	}

	// 5. 加密
	encrypted, err := c.crypto.EncryptMessage(signedJSON)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	// 6. 发送
	transportMsg := &transport.Message{
		Type:      transport.MessageTypeData,
		SenderID:  c.memberID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}

	if err := c.transport.SendMessage(transportMsg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// 更新统计
	c.stats.mutex.Lock()
	c.stats.TotalSent++
	c.stats.MessagesSent++
	c.stats.BytesSent += uint64(len(encrypted))
	c.stats.mutex.Unlock()

	c.logger.Debug("[Client] Signed message sent: %s", msg.ID)

	return nil
}

// GetMessages 获取消息列表
func (c *Client) GetMessages(limit int, offset int) ([]*models.Message, error) {
	return c.messageRepo.GetByChannelID(c.config.ChannelID, limit, offset)
}

// GetMembers 获取成员列表
func (c *Client) GetMembers() ([]*models.Member, error) {
	return c.memberRepo.GetByChannelID(c.config.ChannelID)
}

// UpdateStatus 更新本地成员在线状态并上报给服务器
func (c *Client) UpdateStatus(status models.UserStatus) error {
	c.mutex.RLock()
	running := c.isRunning
	c.mutex.RUnlock()
	if !running {
		return fmt.Errorf("client is not running")
	}
	if c.memberID == "" {
		return fmt.Errorf("member id is empty")
	}

	// 读取旧状态（容错处理）
	oldStatus := models.StatusOffline
	if member, err := c.memberRepo.GetByID(c.memberID); err == nil && member != nil {
		oldStatus = member.Status
	}

	// 更新本地数据库状态
	if err := c.memberRepo.UpdateStatus(c.memberID, status); err != nil {
		return fmt.Errorf("failed to update status in db: %w", err)
	}

	// 发送控制消息到服务器
	payload := map[string]interface{}{
		"type":       "status.update",
		"channel_id": c.config.ChannelID,
		"member_id":  c.memberID,
		"status":     status,
		"timestamp":  time.Now().Unix(),
	}
	reqJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	encrypted, err := c.crypto.EncryptMessage(reqJSON)
	if err != nil {
		return fmt.Errorf("failed to encrypt status update: %w", err)
	}

	msg := &transport.Message{
		Type:      transport.MessageTypeControl,
		SenderID:  c.memberID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}
	if err := c.transport.SendMessage(msg); err != nil {
		return fmt.Errorf("failed to send status update: %w", err)
	}

	// 发布本地事件
	c.eventBus.Publish(events.EventStatusChanged, events.NewStatusChangedEvent(
		c.memberID, c.config.ChannelID, oldStatus, status,
	))

	return nil
}

// UpdateProfile 更新本地成员资料（昵称/头像）
func (c *Client) UpdateProfile(nickname string, avatar string) error {
	if c.memberID == "" {
		return fmt.Errorf("member id is empty")
	}

	// 读取成员并更新字段
	member, err := c.memberRepo.GetByID(c.memberID)
	if err != nil {
		return fmt.Errorf("failed to load member: %w", err)
	}
	if nickname != "" {
		member.Nickname = nickname
	}
	if avatar != "" {
		member.Avatar = avatar
	}

	if err := c.memberRepo.Update(member); err != nil {
		return fmt.Errorf("failed to update profile in db: %w", err)
	}

	// 更新客户端配置（用于后续会话）
	if nickname != "" {
		c.config.Nickname = nickname
	}
	if avatar != "" {
		c.config.Avatar = avatar
	}

	// 发布成员更新事件
	c.eventBus.Publish(events.EventMemberUpdated, &events.MemberEvent{
		Member:    member,
		ChannelID: c.config.ChannelID,
		Action:    "updated",
	})

	return nil
}

// GetStats 获取统计信息
func (c *Client) GetStats() ClientStats {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()

	// 复制统计数据（避免复制锁）
	return ClientStats{
		StartTime:        c.stats.StartTime,
		ConnectedAt:      c.stats.ConnectedAt,
		TotalReceived:    c.stats.TotalReceived,
		TotalSent:        c.stats.TotalSent,
		MessagesReceived: c.stats.MessagesReceived,
		MessagesSent:     c.stats.MessagesSent,
		FilesReceived:    c.stats.FilesReceived,
		FilesSent:        c.stats.FilesSent,
		BytesReceived:    c.stats.BytesReceived,
		BytesSent:        c.stats.BytesSent,
		SyncCount:        c.stats.SyncCount,
		LastSyncTime:     c.stats.LastSyncTime,
	}
}

// IsRunning 检查是否运行中
func (c *Client) IsRunning() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isRunning
}

// ===== 辅助Getter =====

// GetChannelID 获取频道ID
func (c *Client) GetChannelID() string {
	if c.config == nil {
		return ""
	}
	return c.config.ChannelID
}

// IsConnected 检查传输连接状态
func (c *Client) IsConnected() bool {
	if c.transport == nil {
		return false
	}
	return c.transport.IsConnected()
}

// GetConnectTime 获取连接时间
func (c *Client) GetConnectTime() time.Time {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()
	return c.stats.ConnectedAt
}

// GetChannelInfo 获取频道信息（最小集）
func (c *Client) GetChannelInfo() *models.Channel {
	return &models.Channel{
		ID:            c.GetChannelID(),
		Name:          "",
		TransportMode: c.config.TransportMode,
	}
}

// ===== 文件传输相关 =====

// UploadFile 上传文件
func (c *Client) UploadFile(filePath string) (*FileUploadTask, error) {
	return c.fileManager.UploadFile(filePath)
}

// DownloadFile 下载文件
func (c *Client) DownloadFile(fileID string, savePath string) (*FileDownloadTask, error) {
	return c.fileManager.DownloadFile(fileID, savePath)
}

// GetUploadTask 获取上传任务
func (c *Client) GetUploadTask(taskID string) (*FileUploadTask, bool) {
	return c.fileManager.GetUploadTask(taskID)
}

// GetDownloadTask 获取下载任务
func (c *Client) GetDownloadTask(taskID string) (*FileDownloadTask, bool) {
	return c.fileManager.GetDownloadTask(taskID)
}

// GetFileManagerStats 获取文件管理器统计
func (c *Client) GetFileManagerStats() FileManagerStats {
	return c.fileManager.GetStats()
}

// ResumeUpload 恢复上传任务
func (c *Client) ResumeUpload(taskID string) error {
	return c.fileManager.ResumeUpload(taskID)
}

// ResumeDownload 恢复下载任务
func (c *Client) ResumeDownload(taskID string) error {
	return c.fileManager.ResumeDownload(taskID)
}

// ListPendingUploads 列出待恢复的上传任务
func (c *Client) ListPendingUploads() ([]*FileUploadTask, error) {
	return c.fileManager.ListPendingUploads()
}

// CancelUpload 取消上传任务
func (c *Client) CancelUpload(taskID string) error {
	return c.fileManager.CancelUpload(taskID)
}

// CancelDownload 取消下载任务
func (c *Client) CancelDownload(taskID string) error {
	return c.fileManager.CancelDownload(taskID)
}

// ===== 服务发现相关 =====

// Discover 扫描局域网中的服务器
func (c *Client) Discover(timeout time.Duration) ([]*DiscoveredServer, error) {
	return c.discoveryManager.Discover(timeout)
}

// DiscoverAuto 自动发现服务器（根据配置的传输模式）
func (c *Client) DiscoverAuto() ([]*DiscoveredServer, error) {
	return c.discoveryManager.DiscoverAuto()
}

// GetDiscoveredServers 获取已发现的服务器列表
func (c *Client) GetDiscoveredServers() []*DiscoveredServer {
	return c.discoveryManager.GetDiscoveredServers()
}

// GetServerByID 根据ID获取服务器
func (c *Client) GetServerByID(serverID string) (*DiscoveredServer, bool) {
	return c.discoveryManager.GetServerByID(serverID)
}

// ClearServers 清除已发现的服务器列表
func (c *Client) ClearServers() {
	c.discoveryManager.ClearServers()
}

// StartPeriodicDiscovery 启动定期扫描
func (c *Client) StartPeriodicDiscovery(interval time.Duration) {
	c.discoveryManager.StartPeriodicDiscovery(interval)
}

// StopPeriodicDiscovery 停止定期扫描
func (c *Client) StopPeriodicDiscovery() {
	c.discoveryManager.StopPeriodicDiscovery()
}

// ===== 离线队列相关 =====

// GetOfflineQueueSize 获取离线队列大小
func (c *Client) GetOfflineQueueSize() int {
	return c.offlineQueue.GetQueueSize()
}

// GetQueuedMessages 获取队列中的消息
func (c *Client) GetQueuedMessages() []*QueuedMessage {
	return c.offlineQueue.GetQueuedMessages()
}

// ClearOfflineQueue 清空离线队列
func (c *Client) ClearOfflineQueue() {
	c.offlineQueue.Clear()
}

// TriggerOfflineSend 手动触发离线消息发送
func (c *Client) TriggerOfflineSend() {
	c.offlineQueue.TriggerSend()
}

// GetOfflineQueueStats 获取离线队列统计
func (c *Client) GetOfflineQueueStats() OfflineQueueStats {
	return c.offlineQueue.GetStats()
}

// ===== 签名验证相关 =====

// SetServerPublicKey 设置服务器公钥
func (c *Client) SetServerPublicKey(publicKey []byte) {
	c.signatureVerifier.SetServerPublicKey(publicKey)
}

// VerifyMessage 验证消息签名
func (c *Client) VerifyMessage(msg *transport.Message) (bool, error) {
	return c.signatureVerifier.VerifyMessage(msg)
}

// GetSignatureStats 获取签名验证统计
func (c *Client) GetSignatureStats() SignatureStats {
	return c.signatureVerifier.GetStats()
}

// ===== Challenge相关 =====

// GetChallenges 获取所有挑战
func (c *Client) GetChallenges() []*models.Challenge {
	return c.challengeManager.GetChallenges()
}

// GetChallenge 获取指定挑战
func (c *Client) GetChallenge(challengeID string) (*models.Challenge, bool) {
	return c.challengeManager.GetChallenge(challengeID)
}

// SubmitFlag 提交Flag
func (c *Client) SubmitFlag(challengeID string, flag string) error {
	return c.challengeManager.SubmitFlag(challengeID, flag)
}

// RequestHint 请求提示
func (c *Client) RequestHint(challengeID string, hintIndex int) error {
	return c.challengeManager.RequestHint(challengeID, hintIndex)
}

// GetChallengeSubmissions 获取提交记录
func (c *Client) GetChallengeSubmissions() []*models.ChallengeSubmission {
	return c.challengeManager.GetSubmissions()
}

// GetChallengeStats 获取挑战统计
func (c *Client) GetChallengeStats() ChallengeStats {
	return c.challengeManager.GetStats()
}

// GetMemberID 获取本地成员ID
func (c *Client) GetMemberID() string {
	return c.memberID
}

// SetMemberID 设置本地成员ID（由加入响应后设置）
func (c *Client) SetMemberID(memberID string) {
	c.memberID = memberID
	c.logger.Info("[Client] Member ID set: %s", memberID)
}

// generateMessageID 生成消息ID
func generateMessageID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}
