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

// CreateChallenge 创建题目
func (cm *ChallengeManager) CreateChallenge(challenge *models.Challenge) error {
	if challenge == nil {
		return errors.New("challenge is nil")
	}

	// 设置频道ID
	challenge.ChannelID = cm.server.config.ChannelID

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

	// TODO: 添加Assign方法到ChallengeRepository
	// 暂时直接保存assignment
	_ = assignment // 临时避免未使用错误

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

	return nil
}

// HandleFlagSubmission 处理Flag提交
func (cm *ChallengeManager) HandleFlagSubmission(transportMsg *transport.Message) {
	cm.server.logger.Debug("[ChallengeManager] Flag submission from: %s", transportMsg.SenderID)

	// 解密消息
	decrypted, err := cm.server.crypto.DecryptMessage(transportMsg.Payload)
	if err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to decrypt submission: %v", err)
		return
	}

	// 反序列化提交
	var submission models.ChallengeSubmission
	if err := json.Unmarshal(decrypted, &submission); err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to unmarshal submission: %v", err)
		return
	}

	// 保存提交记录（所有提交都接受，无需验证）
	submission.SubmittedAt = time.Now()
	submission.IsCorrect = true // 不需要验证，全部标记为正确

	// 更新题目状态（添加到已解决列表）
	challenge, err := cm.server.challengeRepo.GetByID(submission.ChallengeID)
	if err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to get challenge: %v", err)
		cm.sendSubmissionResponse(transportMsg.SenderID, false, "Challenge not found", &submission)
		return
	}

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
		if challenge.SolvedAt == nil {
			now := time.Now()
			challenge.SolvedAt = &now
		}
		challenge.Status = "solved"

		if err := cm.server.challengeRepo.Update(challenge); err != nil {
			cm.server.logger.Error("[ChallengeManager] Failed to update challenge: %v", err)
		}
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

	cm.server.logger.Info("[ChallengeManager] Flag submitted and accepted: %s by %s (flag: %s)",
		submission.ChallengeID, submission.MemberID, submission.Flag)

	// 发布事件
	cm.server.eventBus.Publish(events.EventChallengeSolved, events.NewSubmissionEvent(&submission, true, "Flag accepted!"))

	// 广播解题消息
	cm.broadcastChallengeSolved(&submission)

	// 发送响应
	cm.sendSubmissionResponse(transportMsg.SenderID, true, "Flag submitted successfully!", &submission)
}

// SubmitFlag 提交Flag（直接接受，不验证）
// 参考: docs/CHALLENGE_SYSTEM.md - Flag提交流程
// 注意: 根据用户需求，FLAG不需要验证，所有提交都接受
func (cm *ChallengeManager) SubmitFlag(challengeID, memberID, flag string) error {
	// 获取题目
	challenge, err := cm.server.challengeRepo.GetByID(challengeID)
	if err != nil {
		return fmt.Errorf("challenge not found: %w", err)
	}

	if challenge.Status == "closed" {
		return fmt.Errorf("challenge is closed")
	}

	// 创建提交记录
	submission := &models.ChallengeSubmission{
		ChallengeID: challengeID,
		MemberID:    memberID,
		Flag:        flag,
		IsCorrect:   true, // 不验证，全部接受
		SubmittedAt: time.Now(),
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
		if challenge.SolvedAt == nil {
			now := time.Now()
			challenge.SolvedAt = &now
		}
		challenge.Status = "solved"

		if err := cm.server.challengeRepo.Update(challenge); err != nil {
			return fmt.Errorf("failed to update challenge: %w", err)
		}
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

	cm.server.logger.Info("[ChallengeManager] Flag submitted: %s by %s", challenge.Title, memberID)

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

// broadcastChallengeCreated 广播题目创建
func (cm *ChallengeManager) broadcastChallengeCreated(challenge *models.Challenge) {
	systemMsg := &models.Message{
		ID:        generateMessageID(),
		ChannelID: cm.server.config.ChannelID,
		SenderID:  "system",
		Type:      models.MessageTypeSystem,
		Timestamp: time.Now(),
	}

	// 设置系统消息内容（直接构造map）
	systemMsg.Content = models.MessageContent{
		"event":     "challenge_created",
		"actor_id":  "server",
		"target_id": challenge.ID,
		"extra": map[string]interface{}{
			"challenge_id": challenge.ID,
			"title":        challenge.Title,
			"category":     challenge.Category,
			"difficulty":   challenge.Difficulty,
			"points":       challenge.Points,
			"message":      fmt.Sprintf("New challenge created: %s [%s]", challenge.Title, challenge.Category),
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
			"message":      fmt.Sprintf("🎉 %s solved challenge: %s (Flag: %s)", member.Nickname, challenge.Title, submission.Flag),
		},
	}

	if err := cm.server.broadcastManager.Broadcast(systemMsg); err != nil {
		cm.server.logger.Error("[ChallengeManager] Failed to broadcast challenge solved: %v", err)
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

// UnlockHint 解锁提示
func (cm *ChallengeManager) UnlockHint(challengeID, memberID string, hintIndex int) error {
	// 获取所有提示
	hints, err := cm.server.challengeRepo.GetHints(challengeID)
	if err != nil {
		return fmt.Errorf("failed to get hints: %w", err)
	}

	if hintIndex < 0 || hintIndex >= len(hints) {
		return errors.New("invalid hint index")
	}

	hint := hints[hintIndex]

	// 解锁提示
	if err := cm.server.challengeRepo.UnlockHint(hint.ID, memberID); err != nil {
		return fmt.Errorf("failed to unlock hint: %w", err)
	}

	cm.server.logger.Info("[ChallengeManager] Hint unlocked: %s for member %s", hint.ID, memberID)

	// 发布事件
	if challenge, err := cm.server.challengeRepo.GetByID(challengeID); err == nil {
		cm.server.eventBus.Publish(events.EventChallengeHintUnlock, &events.ChallengeEvent{
			Challenge: challenge,
			Action:    "hint_unlocked",
			UserID:    memberID,
			ChannelID: cm.server.config.ChannelID,
			ExtraData: hint,
		})
	}

	return nil
}
