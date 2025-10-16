package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"crosswire/internal/events"
	"crosswire/internal/models"
	"crosswire/internal/transport"
)

// ChallengeManager 题目管理器
// 参考: docs/ARCHITECTURE.md - 3.1.2 服务端模块 - ChallengeManager
// 参考: docs/CHALLENGE_SYSTEM.md
type ChallengeManager struct {
	server *Server
}

// NewChallengeManager 创建题目管理器
func NewChallengeManager(server *Server) *ChallengeManager {
	return &ChallengeManager{
		server: server,
	}
}

// CreateChallenge 创建题目（并创建对应的子频道）
func (cm *ChallengeManager) CreateChallenge(challenge *models.Challenge) error {
	if challenge == nil {
		return errors.New("challenge is nil")
	}

	// 设置频道ID
	challenge.ChannelID = cm.server.config.ChannelID

	// 创建题目专属子频道
	subChannel, err := cm.createSubChannel(challenge)
	if err != nil {
		return fmt.Errorf("failed to create sub-channel: %w", err)
	}
	challenge.SubChannelID = subChannel.ID
	cm.server.logger.Info("[ChallengeManager] Created sub-channel: %s for challenge: %s", subChannel.Name, challenge.Title)

	// 保存到数据库
	if err := cm.server.challengeRepo.Create(challenge); err != nil {
		return fmt.Errorf("failed to create challenge: %w", err)
	}

	cm.server.logger.Info("[ChallengeManager] Challenge created: %s (%s)", challenge.Title, challenge.ID)

	// 发布事件
	cm.server.eventBus.Publish(events.EventChallengeCreated, events.NewChallengeEvent(
		events.EventChallengeCreated, challenge, "", cm.server.config.ChannelID, "created", nil))

	// 广播题目创建消息
	cm.broadcastChallengeCreated(challenge)

	return nil
}

// createSubChannel 创建题目专属子频道
func (cm *ChallengeManager) createSubChannel(challenge *models.Challenge) (*models.Channel, error) {
	// 获取主频道
	parentChannel, err := cm.server.channelRepo.GetByID(cm.server.config.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent channel: %w", err)
	}

	// 创建子频道（继承主频道的密码和密钥）
	subChannel := &models.Channel{
		ID:              fmt.Sprintf("%s-sub-%s", cm.server.config.ChannelID, challenge.ID),
		Name:            fmt.Sprintf("%s [%s]", challenge.Title, challenge.Category),
		ParentChannelID: cm.server.config.ChannelID,
		PasswordHash:    parentChannel.PasswordHash,
		Salt:            parentChannel.Salt,
		EncryptionKey:   parentChannel.EncryptionKey,
		KeyVersion:      parentChannel.KeyVersion,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		CreatorID:       "server",
		MaxMembers:      parentChannel.MaxMembers,
		TransportMode:   parentChannel.TransportMode,
		Port:            parentChannel.Port,
		Interface:       parentChannel.Interface,
	}

	// 保存子频道
	if err := cm.server.channelRepo.Create(subChannel); err != nil {
		return nil, fmt.Errorf("failed to save sub-channel: %w", err)
	}

	return subChannel, nil
}

// AssignChallenge 分配题目
func (cm *ChallengeManager) AssignChallenge(challengeID, memberID, assignedBy string) error {
	// 获取题目
	challenge, err := cm.server.challengeRepo.GetByID(challengeID)
	if err != nil {
		return fmt.Errorf("failed to get challenge: %w", err)
	}

	// 创建分配记录
	assignment := &models.ChallengeAssignment{
		ChallengeID: challengeID,
		MemberID:    memberID,
		AssignedBy:  assignedBy,
		AssignedAt:  time.Now(),
		Status:      "assigned",
	}

	// 持久化分配记录
	if err := cm.server.challengeRepo.AssignChallenge(assignment); err != nil {
		return fmt.Errorf("failed to save assignment: %w", err)
	}

	// 更新 Challenge.AssignedTo（便于前端直接读取）
	alreadyAssigned := false
	for _, id := range challenge.AssignedTo {
		if id == memberID {
			alreadyAssigned = true
			break
		}
	}
	if !alreadyAssigned {
		challenge.AssignedTo = append(challenge.AssignedTo, memberID)
		if err := cm.server.challengeRepo.Update(challenge); err != nil {
			cm.server.logger.Warn("[ChallengeManager] Failed to update AssignedTo: %v", err)
		}
	}

	// 初始化进度
	progress := &models.ChallengeProgress{
		ChallengeID: challengeID,
		MemberID:    memberID,
		Status:      "in_progress",
		Progress:    0,
	}

	if err := cm.server.challengeRepo.UpdateProgress(progress); err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to initialize progress: %v", err)
	}

	cm.server.logger.Info("[ChallengeManager] Challenge assigned: %s to %s", challenge.Title, memberID)

	// 发布事件
	cm.server.eventBus.Publish(events.EventChallengeAssigned, events.NewChallengeEvent(
		events.EventChallengeAssigned, challenge, memberID, cm.server.config.ChannelID, "assigned", assignment))

	// 广播系统消息
	cm.broadcastChallengeAssigned(challenge, memberID, assignedBy)

	return nil
}

// HandleFlagSubmission 处理Flag提交
func (cm *ChallengeManager) HandleFlagSubmission(transportMsg *transport.Message) {
	cm.server.logger.Debug("[ChallengeManager] HandleFlagSubmission received: sender=%s ts=%v payload_len=%d", transportMsg.SenderID, transportMsg.Timestamp, len(transportMsg.Payload))

	// 解密消息
	decrypted, err := cm.server.crypto.DecryptMessage(transportMsg.Payload)
	if err != nil {
		cm.server.logger.Error("[ChallengeManager] Decrypt submission failed: %v", err)
		return
	}
	cm.server.logger.Debug("[ChallengeManager] Submission decrypted: bytes=%d", len(decrypted))

	// 反序列化提交
	var submission models.ChallengeSubmission
	if err := json.Unmarshal(decrypted, &submission); err != nil {
		cm.server.logger.Error("[ChallengeManager] Unmarshal submission failed: %v", err)
		return
	}
	cm.server.logger.Debug("[ChallengeManager] Parsed submission: challenge_id=%s member=%s id=%s flag_len=%d", submission.ChallengeID, transportMsg.SenderID, submission.ID, len(submission.Flag))

	// 保存提交记录（协作平台：所有提交都接受，无需验证）
	submission.SubmittedAt = time.Now()
	// 采用传输层的发送者ID作为提交成员ID（前端可能未包含 member_id）
	submission.MemberID = transportMsg.SenderID
	if submission.ID == "" {
		submission.ID = generateMessageID()
	}

	cm.server.logger.Debug("[ChallengeManager] Persisting submission: challenge=%s member=%s id=%s", submission.ChallengeID, submission.MemberID, submission.ID)

	// 持久化提交记录
	if err := cm.server.challengeRepo.SubmitFlag(&submission); err != nil {
		cm.server.logger.Error("[ChallengeManager] Persist submission failed: %v", err)
	}

	// 更新题目状态（添加到已解决列表）
	challenge, err := cm.server.challengeRepo.GetByID(submission.ChallengeID)
	if err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to get challenge: %v", err)
		cm.sendSubmissionResponse(transportMsg.SenderID, false, "Challenge not found", &submission)
		return
	}
	cm.server.logger.Debug("[ChallengeManager] Loaded challenge: title=%s status=%s solved_by=%d", challenge.Title, challenge.Status, len(challenge.SolvedBy))

	// 添加到SolvedBy列表（如果不存在）
	alreadySolved := false
	for _, solverID := range challenge.SolvedBy {
		if solverID == submission.MemberID {
			alreadySolved = true
			break
		}
	}

	if !alreadySolved {
		challenge.SolvedBy = append(challenge.SolvedBy, submission.MemberID)
		if challenge.SolvedAt.IsZero() {
			challenge.SolvedAt = time.Now()
		}
		challenge.Status = "solved"
		// 覆盖更新题目上的 Flag（协作平台：Flag 明文对所有人可见）
		challenge.Flag = submission.Flag

		cm.server.logger.Debug("[HandleFlagSubmission] Updating challenge: SolvedBy=%v Status=%s", challenge.SolvedBy, challenge.Status)
		if err := cm.server.challengeRepo.Update(challenge); err != nil {
			cm.server.logger.Error("[HandleFlagSubmission] Failed to update challenge: %v", err)
		} else {
			cm.server.logger.Info("[HandleFlagSubmission] Challenge updated successfully: %s now solved by %v", challenge.Title, challenge.SolvedBy)
		}
	} else {
		// 即使之前已标记 solved，也同步覆盖 Flag 以便前端刷新后可见
		if challenge.Flag != submission.Flag {
			challenge.Flag = submission.Flag
			if err := cm.server.challengeRepo.Update(challenge); err != nil {
				cm.server.logger.Error("[HandleFlagSubmission] Failed to update challenge flag: %v", err)
			} else {
				cm.server.logger.Info("[HandleFlagSubmission] Challenge flag updated for %s", challenge.Title)
			}
		}
		cm.server.logger.Debug("[HandleFlagSubmission] Member %s already solved challenge %s", submission.MemberID, challenge.Title)
	}

	// 更新进度
	progress := &models.ChallengeProgress{
		ChallengeID: submission.ChallengeID,
		MemberID:    submission.MemberID,
		Status:      "solved",
		Progress:    100,
		UpdatedAt:   time.Now(),
	}

	if err := cm.server.challengeRepo.UpdateProgress(progress); err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to update progress: %v", err)
	}

	cm.server.logger.Info("[ChallengeManager] Flag submitted: challenge=%s by=%s", submission.ChallengeID, submission.MemberID)

	// 发布事件（协作平台：所有提交都标记为成功）
	cm.server.eventBus.Publish(events.EventChallengeSolved, events.NewSubmissionEvent(&submission, true, "Flag accepted"))

	// 广播解题消息
	cm.broadcastChallengeSolved(&submission)

	// 发送响应（协作平台：总是返回成功）
	cm.sendSubmissionResponse(transportMsg.SenderID, true, "Flag 已接受!", &submission)
}

// SubmitFlag 提交Flag（直接接受，不验证）
// 参考: docs/CHALLENGE_SYSTEM.md - Flag提交流程
// 注意: 根据用户需求，FLAG不需要验证，所有提交都接受
func (cm *ChallengeManager) SubmitFlag(challengeID, memberID, flag string) error {
	cm.server.logger.Info("[ChallengeManager] SubmitFlag called: challengeID=%s memberID=%s", challengeID, memberID)
	// 获取题目
	challenge, err := cm.server.challengeRepo.GetByID(challengeID)
	if err != nil {
		return fmt.Errorf("challenge not found: %w", err)
	}

	if challenge.Status == "closed" {
		return fmt.Errorf("challenge is closed")
	}

	// 创建提交记录（协作平台：不验证，全部接受）
	submission := &models.ChallengeSubmission{
		ID:          generateMessageID(),
		ChallengeID: challengeID,
		MemberID:    memberID,
		Flag:        flag,
		SubmittedAt: time.Now(),
	}

	// 持久化提交记录
	if err := cm.server.challengeRepo.SubmitFlag(submission); err != nil {
		cm.server.logger.Error("[ChallengeManager] Persist submission failed: %v", err)
		return fmt.Errorf("failed to persist submission: %w", err)
	}

	// 更新题目状态
	alreadySolved := false
	for _, solverID := range challenge.SolvedBy {
		if solverID == memberID {
			alreadySolved = true
			break
		}
	}

	if !alreadySolved {
		challenge.SolvedBy = append(challenge.SolvedBy, memberID)
		if challenge.SolvedAt.IsZero() {
			challenge.SolvedAt = time.Now()
		}
		challenge.Status = "solved"
		// 覆盖更新题目上的 Flag（协作平台：Flag 明文对所有人可见）
		challenge.Flag = flag

		cm.server.logger.Debug("[ChallengeManager] Updating challenge: SolvedBy=%v Status=%s", challenge.SolvedBy, challenge.Status)
		if err := cm.server.challengeRepo.Update(challenge); err != nil {
			cm.server.logger.Error("[ChallengeManager] Failed to update challenge: %v", err)
			return fmt.Errorf("failed to update challenge: %w", err)
		}
		cm.server.logger.Info("[ChallengeManager] Challenge updated successfully: %s now solved by %v", challenge.Title, challenge.SolvedBy)
	} else {
		// 已解出情况下也同步覆盖 Flag，保证后续 GetChallenges 可见
		if challenge.Flag != flag {
			challenge.Flag = flag
			if err := cm.server.challengeRepo.Update(challenge); err != nil {
				cm.server.logger.Error("[ChallengeManager] Failed to update challenge flag: %v", err)
			}
		}
		cm.server.logger.Debug("[ChallengeManager] Member %s already solved challenge %s", memberID, challenge.Title)
	}

	// 更新进度
	progress := &models.ChallengeProgress{
		ChallengeID: challengeID,
		MemberID:    memberID,
		Status:      "solved",
		Progress:    100,
		UpdatedAt:   time.Now(),
	}

	if err := cm.server.challengeRepo.UpdateProgress(progress); err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to update progress: %v", err)
	}

	cm.server.logger.Info("[ChallengeManager] Flag submitted: %s by %s (submissionID=%s)", challenge.Title, memberID, submission.ID)

	// 发布事件
	cm.server.eventBus.Publish(events.EventChallengeSolved, events.NewSubmissionEvent(submission, true, "Flag accepted"))

	return nil
}

// sendSubmissionResponse 发送提交响应
func (cm *ChallengeManager) sendSubmissionResponse(to string, correct bool, message string, submission *models.ChallengeSubmission) {
	response := map[string]interface{}{
		"success":      correct,
		"message":      message,
		"challenge_id": submission.ChallengeID,
		"timestamp":    time.Now().Unix(),
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to marshal response: %v", err)
		return
	}

	encrypted, err := cm.server.crypto.EncryptMessage(responseData)
	if err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to encrypt response: %v", err)
		return
	}

	transportMsg := &transport.Message{
		Type:      transport.MessageTypeData,
		SenderID:  cm.server.config.ChannelID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}

	if err := cm.server.transport.SendMessage(transportMsg); err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to send response: %v", err)
	}
}

// broadcastChallengeCreated 广播题目创建（包含子频道信息）
func (cm *ChallengeManager) broadcastChallengeCreated(challenge *models.Challenge) {
	systemMsg := &models.Message{
		ID:        generateMessageID(),
		ChannelID: cm.server.config.ChannelID,
		SenderID:  "system",
		Type:      models.MessageTypeSystem,
		Timestamp: time.Now(),
	}

	// 设置系统消息内容（直接构造map），包含子频道ID
	systemMsg.Content = models.MessageContent{
		"event":     "challenge_created",
		"actor_id":  "server",
		"target_id": challenge.ID,
		"extra": map[string]interface{}{
			"challenge_id":   challenge.ID,
			"title":          challenge.Title,
			"category":       challenge.Category,
			"difficulty":     challenge.Difficulty,
			"points":         challenge.Points,
			"sub_channel_id": challenge.SubChannelID, // 添加子频道ID
			"message":        fmt.Sprintf("New challenge created: %s [%s]", challenge.Title, challenge.Category),
		},
	}

	if err := cm.server.broadcastManager.Broadcast(systemMsg); err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to broadcast challenge created: %v", err)
	}
}

// broadcastChallengeSolved 广播题目解决
func (cm *ChallengeManager) broadcastChallengeSolved(submission *models.ChallengeSubmission) {
	// 获取成员信息
	member := cm.server.channelManager.GetMemberByID(submission.MemberID)
	if member == nil {
		return
	}

	// 获取题目信息
	challenge, err := cm.server.challengeRepo.GetByID(submission.ChallengeID)
	if err != nil {
		return
	}

	systemMsg := &models.Message{
		ID:        generateMessageID(),
		ChannelID: cm.server.config.ChannelID,
		SenderID:  "system",
		Type:      models.MessageTypeSystem,
		Timestamp: time.Now(),
	}

	// 设置系统消息内容（直接构造map）
	systemMsg.Content = models.MessageContent{
		"event":     "challenge_solved",
		"actor_id":  submission.MemberID,
		"target_id": challenge.ID,
		"extra": map[string]interface{}{
			"challenge_id": challenge.ID,
			"nickname":     member.Nickname,
			"flag":         submission.Flag,
			"message":      fmt.Sprintf("🎉 %s 提交了挑战 %s 的 Flag: %s", member.Nickname, challenge.Title, submission.Flag),
		},
	}

	if err := cm.server.broadcastManager.Broadcast(systemMsg); err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to broadcast challenge solved: %v", err)
	}
}

// broadcastChallengeAssigned 广播题目分配系统消息
func (cm *ChallengeManager) broadcastChallengeAssigned(challenge *models.Challenge, memberID string, assignedBy string) {
	systemMsg := &models.Message{
		ID:        generateMessageID(),
		ChannelID: cm.server.config.ChannelID,
		SenderID:  "system",
		Type:      models.MessageTypeSystem,
		Timestamp: time.Now(),
	}

	// 设置系统消息内容
	systemMsg.Content = models.MessageContent{
		"event":     "challenge_assigned",
		"actor_id":  assignedBy,
		"target_id": challenge.ID,
		"extra": map[string]interface{}{
			"challenge_id": challenge.ID,
			"title":        challenge.Title,
			"assignee_id":  memberID,
			"message":      fmt.Sprintf("Challenge assigned: %s -> %s", challenge.Title, memberID),
		},
	}

	if err := cm.server.broadcastManager.Broadcast(systemMsg); err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to broadcast challenge assignment: %v", err)
	}
}

// GetChallengeProgress 获取题目进度
func (cm *ChallengeManager) GetChallengeProgress(challengeID, memberID string) (*models.ChallengeProgress, error) {
	return cm.server.challengeRepo.GetProgress(challengeID, memberID)
}

// UpdateProgress 更新题目进度
func (cm *ChallengeManager) UpdateProgress(challengeID, memberID string, progress int, summary string) error {
	progressData := &models.ChallengeProgress{
		ChallengeID: challengeID,
		MemberID:    memberID,
		Progress:    progress,
		Summary:     summary,
		UpdatedAt:   time.Now(),
	}

	if progress >= 100 {
		progressData.Status = "solved"
	} else if progress > 0 {
		progressData.Status = "in_progress"
	} else {
		progressData.Status = "not_started"
	}

	if err := cm.server.challengeRepo.UpdateProgress(progressData); err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}

	// 发布进度更新事件
	if challenge, err := cm.server.challengeRepo.GetByID(challengeID); err == nil {
		cm.server.eventBus.Publish(events.EventChallengeProgress, &events.ChallengeEvent{
			Challenge: challenge,
			Action:    "progress_updated",
			UserID:    memberID,
			ChannelID: cm.server.config.ChannelID,
			ExtraData: progressData,
		})
	}

	cm.server.logger.Debug("[ChallengeManager] Progress updated: %s for %s (%d%%)",
		challengeID, memberID, progress)

	return nil
}

// GetLeaderboard 获取排行榜
// 参考: docs/CHALLENGE_SYSTEM.md - 排行榜功能
func (cm *ChallengeManager) GetLeaderboard(channelID string) ([]*LeaderboardEntry, error) {
	// 获取所有题目
	challenges, err := cm.server.challengeRepo.GetByChannelID(channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenges: %w", err)
	}

	// 统计每个成员的解题数和分数
	leaderboard := make(map[string]*LeaderboardEntry)

	for _, challenge := range challenges {
		for _, memberID := range challenge.SolvedBy {
			if entry, exists := leaderboard[memberID]; exists {
				entry.SolvedCount++
				entry.TotalPoints += challenge.Points
			} else {
				member := cm.server.channelManager.GetMemberByID(memberID)
				if member != nil {
					leaderboard[memberID] = &LeaderboardEntry{
						MemberID:    memberID,
						Nickname:    member.Nickname,
						SolvedCount: 1,
						TotalPoints: challenge.Points,
					}
				}
			}
		}
	}

	// 转换为切片并排序
	entries := make([]*LeaderboardEntry, 0, len(leaderboard))
	for _, entry := range leaderboard {
		entries = append(entries, entry)
	}

	// 按分数排序（降序）
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].TotalPoints > entries[i].TotalPoints ||
				(entries[j].TotalPoints == entries[i].TotalPoints && entries[j].SolvedCount > entries[i].SolvedCount) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// 设置排名
	for i, entry := range entries {
		entry.Rank = i + 1
	}

	return entries, nil
}

// LeaderboardEntry 排行榜条目
type LeaderboardEntry struct {
	Rank        int    `json:"rank"`
	MemberID    string `json:"member_id"`
	Nickname    string `json:"nickname"`
	SolvedCount int    `json:"solved_count"`
	TotalPoints int    `json:"total_points"`
}

// GetStats 获取Challenge统计信息
func (cm *ChallengeManager) GetStats() ChallengeStats {
	challenges, _ := cm.server.challengeRepo.GetByChannelID(cm.server.config.ChannelID)

	stats := ChallengeStats{
		TotalChallenges: len(challenges),
	}

	for _, challenge := range challenges {
		if challenge.Status == "solved" {
			stats.SolvedChallenges++
		}
		stats.TotalSolves += len(challenge.SolvedBy)
	}

	return stats
}

// ChallengeStats Challenge统计
type ChallengeStats struct {
	TotalChallenges  int `json:"total_challenges"`
	SolvedChallenges int `json:"solved_challenges"`
	TotalSolves      int `json:"total_solves"`
}

// 提示功能已移除（协作平台不支持提示）
