package client

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/events"
	"crosswire/internal/models"
	"crosswire/internal/transport"

	"github.com/google/uuid"
)

// ChallengeManager CTF挑战客户端管理器
type ChallengeManager struct {
	client *Client

	// 挑战列表
	challenges      map[string]*models.Challenge
	challengesMutex sync.RWMutex

	// 我的提交记录
	submissions      map[string]*models.ChallengeSubmission
	submissionsMutex sync.RWMutex

	// 统计
	stats      ChallengeStats
	statsMutex sync.RWMutex
}

// ChallengeStats 挑战统计
type ChallengeStats struct {
	TotalChallenges    int
	SolvedChallenges   int
	TotalSubmissions   int
	CorrectSubmissions int
	mutex              sync.RWMutex
}

// NewChallengeManager 创建挑战管理器
func NewChallengeManager(client *Client) *ChallengeManager {
	return &ChallengeManager{
		client:      client,
		challenges:  make(map[string]*models.Challenge),
		submissions: make(map[string]*models.ChallengeSubmission),
	}
}

// Start 启动挑战管理器
func (cm *ChallengeManager) Start() error {
	cm.client.logger.Info("[ChallengeManager] Starting...")

	// 订阅挑战相关事件
	cm.client.eventBus.Subscribe(events.EventChallengeCreated, cm.handleChallengeCreated)
	cm.client.eventBus.Subscribe(events.EventChallengeAssigned, cm.handleChallengeAssigned)
	cm.client.eventBus.Subscribe(events.EventChallengeSolved, cm.handleChallengeSolved)

	// 加载本地挑战数据
	if err := cm.loadChallenges(); err != nil {
		cm.client.logger.Warn("[ChallengeManager] Failed to load challenges: %v", err)
	}

	cm.client.logger.Info("[ChallengeManager] Started successfully")
	return nil
}

// Stop 停止挑战管理器
func (cm *ChallengeManager) Stop() error {
	cm.client.logger.Info("[ChallengeManager] Stopping...")
	cm.client.logger.Info("[ChallengeManager] Stopped")
	return nil
}

// loadChallenges 从数据库加载挑战
func (cm *ChallengeManager) loadChallenges() error {
	cm.client.logger.Debug("[ChallengeManager] Loading challenges from database...")

	if cm.client == nil || cm.client.challengeRepo == nil {
		return fmt.Errorf("challenge repository not initialized")
	}

	// 读取当前频道的挑战列表
	channelID := cm.client.GetChannelID()
	challenges, err := cm.client.challengeRepo.GetByChannelID(channelID)
	if err != nil {
		return fmt.Errorf("failed to load challenges: %w", err)
	}

	// 写入内存缓存
	cm.challengesMutex.Lock()
	for _, ch := range challenges {
		if ch != nil {
			cm.challenges[ch.ID] = ch
		}
	}
	cm.challengesMutex.Unlock()

	// 计算统计
	cm.statsMutex.Lock()
	cm.stats.TotalChallenges = len(challenges)
	solved := 0
	for _, ch := range challenges {
		if ch != nil && len(ch.SolvedBy) > 0 {
			solved++
		}
	}
	cm.stats.SolvedChallenges = solved
	cm.statsMutex.Unlock()

	cm.client.logger.Info("[ChallengeManager] Loaded %d challenges from database", len(challenges))
	return nil
}

// GetChallenges 获取所有挑战
func (cm *ChallengeManager) GetChallenges() []*models.Challenge {
	cm.challengesMutex.RLock()
	defer cm.challengesMutex.RUnlock()

	challenges := make([]*models.Challenge, 0, len(cm.challenges))
	for _, challenge := range cm.challenges {
		challenges = append(challenges, challenge)
	}

	return challenges
}

// GetChallenge 获取指定挑战
func (cm *ChallengeManager) GetChallenge(challengeID string) (*models.Challenge, bool) {
	cm.challengesMutex.RLock()
	defer cm.challengesMutex.RUnlock()

	challenge, ok := cm.challenges[challengeID]
	return challenge, ok
}

// SubmitFlag 提交Flag
func (cm *ChallengeManager) SubmitFlag(challengeID string, flag string) error {
	cm.client.logger.Info("[ChallengeManager] SubmitFlag: challengeID=%s memberID=%s", challengeID, cm.client.memberID)

	// 检查挑战是否存在
	challenge, ok := cm.GetChallenge(challengeID)
	if !ok {
		return fmt.Errorf("challenge not found: %s", challengeID)
	}

	// 检查是否已解决
	if len(challenge.SolvedBy) > 0 {
		// 检查是否是当前用户解决的
		for _, memberID := range challenge.SolvedBy {
			if memberID == cm.client.memberID {
				return fmt.Errorf("challenge already solved by you")
			}
		}
	}

	// 构造提交消息
	submission := map[string]interface{}{
		"type":         "challenge.submit",
		"challenge_id": challengeID,
		"flag":         flag,
		"submitted_at": time.Now().Unix(),
	}

	payload, err := json.Marshal(submission)
	if err != nil {
		return fmt.Errorf("failed to marshal submission: %w", err)
	}

	encrypted, err := cm.client.crypto.EncryptMessage(payload)
	if err != nil {
		cm.client.logger.Error("[ChallengeManager] Encrypt submission failed: %v", err)
		return fmt.Errorf("failed to encrypt submission: %w", err)
	}

	// 发送提交
	msg := &transport.Message{
		ID:        uuid.New().String(),
		Type:      transport.MessageTypeData,
		SenderID:  cm.client.memberID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}

	if err := cm.client.transport.SendMessage(msg); err != nil {
		cm.client.logger.Error("[ChallengeManager] Send submission failed: %v", err)
		return fmt.Errorf("failed to send submission: %w", err)
	}

	// 记录提交（协作平台：所有提交都有效）
	cm.submissionsMutex.Lock()
	cm.submissions[challengeID] = &models.ChallengeSubmission{
		ID:          uuid.New().String(),
		ChallengeID: challengeID,
		MemberID:    cm.client.memberID,
		Flag:        flag,
		SubmittedAt: time.Now(),
	}
	cm.submissionsMutex.Unlock()

	// 更新统计
	cm.statsMutex.Lock()
	cm.stats.TotalSubmissions++
	cm.statsMutex.Unlock()

	cm.client.logger.Debug("[ChallengeManager] Flag submitted for challenge: %s", challengeID)

	return nil
}

// 提示功能已移除（协作平台不支持提示）

// GetSubmissions 获取我的所有提交记录
func (cm *ChallengeManager) GetSubmissions() []*models.ChallengeSubmission {
	cm.submissionsMutex.RLock()
	defer cm.submissionsMutex.RUnlock()

	submissions := make([]*models.ChallengeSubmission, 0, len(cm.submissions))
	for _, submission := range cm.submissions {
		submissions = append(submissions, submission)
	}

	return submissions
}

// GetChallengeSubmission 获取指定挑战的提交记录
func (cm *ChallengeManager) GetChallengeSubmission(challengeID string) (*models.ChallengeSubmission, bool) {
	cm.submissionsMutex.RLock()
	defer cm.submissionsMutex.RUnlock()

	submission, ok := cm.submissions[challengeID]
	return submission, ok
}

// GetStats 获取统计信息
func (cm *ChallengeManager) GetStats() ChallengeStats {
	cm.statsMutex.RLock()
	defer cm.statsMutex.RUnlock()

	return ChallengeStats{
		TotalChallenges:    cm.stats.TotalChallenges,
		SolvedChallenges:   cm.stats.SolvedChallenges,
		TotalSubmissions:   cm.stats.TotalSubmissions,
		CorrectSubmissions: cm.stats.CorrectSubmissions,
	}
}

// ===== 事件处理 =====

// handleChallengeCreated 处理挑战创建事件
func (cm *ChallengeManager) handleChallengeCreated(event *events.Event) {
	challengeEvent, ok := event.Data.(events.ChallengeEvent)
	if !ok {
		cm.client.logger.Warn("[ChallengeManager] Invalid challenge event data")
		return
	}

	challenge := challengeEvent.Challenge

	cm.challengesMutex.Lock()
	cm.challenges[challenge.ID] = challenge
	cm.challengesMutex.Unlock()

	cm.statsMutex.Lock()
	cm.stats.TotalChallenges++
	cm.statsMutex.Unlock()

	// 如果题目有关联的子频道，尝试同步子频道信息
	if challenge.SubChannelID != "" {
		go cm.syncSubChannel(challenge.SubChannelID)
	}

	cm.client.logger.Info("[ChallengeManager] New challenge created: %s (sub-channel: %s)",
		challenge.Title, challenge.SubChannelID)
}

// syncSubChannel 同步子频道信息（从服务端查询并保存到本地）
func (cm *ChallengeManager) syncSubChannel(subChannelID string) {
	// TODO: 实现从服务端同步子频道详细信息的逻辑
	// 当前子频道信息会在客户端调用 GetSubChannels() 时从本地数据库查询
	// 如果本地不存在，可以通过请求服务端获取完整的子频道信息
	cm.client.logger.Debug("[ChallengeManager] Sub-channel sync requested: %s (not yet implemented)", subChannelID)
}

// handleChallengeAssigned 处理挑战分配事件
func (cm *ChallengeManager) handleChallengeAssigned(event *events.Event) {
	challengeEvent, ok := event.Data.(events.ChallengeEvent)
	if !ok {
		cm.client.logger.Warn("[ChallengeManager] Invalid challenge event data")
		return
	}

	cm.challengesMutex.Lock()
	cm.challenges[challengeEvent.Challenge.ID] = challengeEvent.Challenge
	cm.challengesMutex.Unlock()

	cm.client.logger.Info("[ChallengeManager] Challenge assigned: %s", challengeEvent.Challenge.Title)
}

// handleChallengeSolved 处理挑战解决事件
func (cm *ChallengeManager) handleChallengeSolved(event *events.Event) {
	challengeEvent, ok := event.Data.(events.ChallengeEvent)
	if !ok {
		cm.client.logger.Warn("[ChallengeManager] Invalid challenge event data")
		return
	}

	// 更新挑战状态
	cm.challengesMutex.Lock()
	if challenge, ok := cm.challenges[challengeEvent.Challenge.ID]; ok {
		// 添加到已解决列表
		if challengeEvent.UserID != "" {
			challenge.SolvedBy = append(challenge.SolvedBy, challengeEvent.UserID)
		}
		challenge.SolvedAt = time.Now()
	}
	cm.challengesMutex.Unlock()

	// 更新提交记录（协作平台：无需验证正确性）
	cm.submissionsMutex.Lock()
	// 记录已提交（所有提交都有效）
	_ = cm.submissions[challengeEvent.Challenge.ID]
	cm.submissionsMutex.Unlock()

	// 更新统计
	cm.statsMutex.Lock()
	cm.stats.SolvedChallenges++
	cm.stats.CorrectSubmissions++
	cm.statsMutex.Unlock()

	cm.client.logger.Info("[ChallengeManager] Challenge solved: %s", challengeEvent.Challenge.Title)
}
