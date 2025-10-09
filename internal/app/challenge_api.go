package app

import (
	"fmt"
	"time"

	"crosswire/internal/models"

	"github.com/google/uuid"
)

// generateID 生成唯一ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ==================== CTF题目管理 API ====================

// CreateChallenge 创建题目（仅服务端）
func (a *App) CreateChallenge(req CreateChallengeRequest) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端可创建题目", "")
	}

	// 验证请求
	if req.Title == "" {
		return NewErrorResponse("invalid_request", "题目标题不能为空", "")
	}

	// 协作平台：Flag可选且为明文

	a.logger.Info("Creating challenge: %s", req.Title)

	// 创建题目
	challenge := &models.Challenge{
		ID:          uuid.NewString(),
		Title:       req.Title,
		Category:    req.Category,
		Difficulty:  req.Difficulty,
		Description: req.Description,
		Points:      req.Points,
		Flag:        req.Flag,
		Status:      "open",
		CreatedBy:   "server",
	}

	err := srv.CreateChallenge(challenge)
	if err != nil {
		return NewErrorResponse("create_error", "创建题目失败", err.Error())
	}

	dto := a.challengeToDTO(challenge)
	return NewSuccessResponse(dto)
}

// GetChallenges 获取题目列表
func (a *App) GetChallenges() Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 获取题目列表
	var challenges []*models.Challenge
	var err error

	if mode == ModeServer && srv != nil {
		challenges, err = srv.GetChallenges()
	} else if mode == ModeClient && cli != nil {
		challenges = cli.GetChallenges()
		err = nil
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("query_error", "获取题目列表失败", err.Error())
	}

	// 转换为DTO
	challengeDTOs := make([]*ChallengeDTO, 0, len(challenges))
	for _, challenge := range challenges {
		dto := a.challengeToDTO(challenge)
		challengeDTOs = append(challengeDTOs, dto)
	}

	return NewSuccessResponse(challengeDTOs)
}

// GetChallenge 获取单个题目
func (a *App) GetChallenge(challengeID string) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 获取题目
	var challenge *models.Challenge
	var err error

	if mode == ModeServer && srv != nil {
		challenge, err = srv.GetChallenge(challengeID)
	} else if mode == ModeClient && cli != nil {
		var found bool
		challenge, found = cli.GetChallenge(challengeID)
		if !found {
			err = fmt.Errorf("challenge not found")
		}
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("not_found", "题目不存在", err.Error())
	}

	dto := a.challengeToDTO(challenge)
	return NewSuccessResponse(dto)
}

// UpdateChallenge 更新题目（仅服务端）
func (a *App) UpdateChallenge(challengeID string, req UpdateChallengeRequest) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端可更新题目", "")
	}

	a.logger.Info("Updating challenge: %s", challengeID)

	// 获取现有题目
	challenge, err := srv.GetChallenge(challengeID)
	if err != nil {
		return NewErrorResponse("not_found", "题目不存在", err.Error())
	}

	// 更新字段
	if req.Title != nil && *req.Title != "" {
		challenge.Title = *req.Title
	}
	if req.Description != nil && *req.Description != "" {
		challenge.Description = *req.Description
	}
	if req.Category != nil && *req.Category != "" {
		challenge.Category = *req.Category
	}
	if req.Difficulty != nil && *req.Difficulty != "" {
		challenge.Difficulty = *req.Difficulty
	}
	if req.Points != nil && *req.Points > 0 {
		challenge.Points = *req.Points
	}

	// 更新题目
	err = srv.UpdateChallenge(challenge)
	if err != nil {
		return NewErrorResponse("update_error", "更新题目失败", err.Error())
	}

	// 获取更新后的题目
	challenge, err = srv.GetChallenge(challengeID)
	if err != nil {
		return NewErrorResponse("query_error", "获取更新后的题目失败", err.Error())
	}

	dto := a.challengeToDTO(challenge)
	return NewSuccessResponse(dto)
}

// DeleteChallenge 删除题目（仅服务端）
func (a *App) DeleteChallenge(challengeID string) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端可删除题目", "")
	}

	a.logger.Info("Deleting challenge: %s", challengeID)

	// 删除题目
	if err := srv.DeleteChallenge(challengeID); err != nil {
		return NewErrorResponse("delete_error", "删除题目失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "题目已删除",
	})
}

// AssignChallenge 分配题目给成员（仅服务端）
func (a *App) AssignChallenge(challengeID string, memberIDs []string) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端可分配题目", "")
	}

	a.logger.Info("Assigning challenge %s to %d members", challengeID, len(memberIDs))

	// 分配题目给每个成员
	for _, memberID := range memberIDs {
		if err := srv.AssignChallenge(challengeID, memberID, "server"); err != nil {
			return NewErrorResponse("assign_error", "分配题目失败", err.Error())
		}
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "题目已分配",
	})
}

// SubmitFlag 提交flag
func (a *App) SubmitFlag(req SubmitFlagRequest) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 验证请求
	if req.ChallengeID == "" {
		return NewErrorResponse("invalid_request", "题目ID不能为空", "")
	}

	if req.Flag == "" {
		return NewErrorResponse("invalid_request", "flag不能为空", "")
	}

	a.logger.Info("Submitting flag for challenge: %s", req.ChallengeID)

	if mode == ModeServer && srv != nil {
		return NewErrorResponse("invalid_mode", "仅客户端可提交Flag", "")
	}
	if mode == ModeClient && cli != nil {
		if err := cli.SubmitFlag(req.ChallengeID, req.Flag); err != nil {
			return NewErrorResponse("submit_error", "提交失败", err.Error())
		}
		points := 0
		if ch, ok := cli.GetChallenge(req.ChallengeID); ok && ch != nil {
			points = ch.Points
		}
		return NewSuccessResponse(SubmitFlagResponse{
			Success: true,
			Message: "Flag已提交，协作记录已保存",
			Points:  points,
		})
	}
	return NewErrorResponse("invalid_mode", "无效的运行模式", "")
}

// UpdateChallengeProgress 更新题目进度
func (a *App) UpdateChallengeProgress(req UpdateProgressRequest) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 验证请求
	if req.ChallengeID == "" {
		return NewErrorResponse("invalid_request", "题目ID不能为空", "")
	}

	if req.Progress < 0 || req.Progress > 100 {
		return NewErrorResponse("invalid_request", "进度必须在0-100之间", "")
	}

	// 服务端：直接写入
	if mode == ModeServer && srv != nil {
		progress := &models.ChallengeProgress{
			ChallengeID: req.ChallengeID,
			MemberID:    "server",
			Progress:    req.Progress,
			Summary:     req.Summary,
		}
		if err := srv.UpdateChallengeProgress(progress); err != nil {
			return NewErrorResponse("update_error", "更新进度失败", err.Error())
		}
		return NewSuccessResponse(map[string]interface{}{
			"message":  "进度已更新",
			"progress": req.Progress,
		})
	}

	// 客户端：暂不实现（保留原行为）
	return NewErrorResponse("not_implemented", "客户端进度更新暂未实现", "")
}

// AddHint 添加提示（已禁用 - 不需要此功能）
func (a *App) AddHint(req AddHintRequest) Response {
	return NewErrorResponse("not_supported", "不支持提示功能", "")
}

// UnlockHint 解锁提示（客户端通过控制消息请求服务端）
func (a *App) UnlockHint(challengeID string, hintIndex int) Response {
	a.mu.RLock()
	mode := a.mode
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}
	if challengeID == "" || hintIndex < 0 {
		return NewErrorResponse("invalid_request", "challenge_id 或 hint_index 无效", "")
	}

	if mode != ModeClient || cli == nil {
		return NewErrorResponse("invalid_mode", "仅客户端可请求解锁提示", "")
	}

	// 通过客户端管理器请求提示
	if err := cli.RequestHint(challengeID, hintIndex); err != nil {
		return NewErrorResponse("request_error", "请求提示失败", err.Error())
	}
	return NewSuccessResponse(map[string]interface{}{
		"message": "已请求解锁提示",
	})
}

// GetLeaderboard 获取排行榜（已禁用 - 不需要此功能）
func (a *App) GetLeaderboard() Response {
	return NewErrorResponse("not_supported", "不支持排行榜功能", "")
}

// GetChallengeSubmissions 获取题目提交记录（已禁用 - 不需要此功能）
func (a *App) GetChallengeSubmissions(challengeID string) Response {
	return NewErrorResponse("not_supported", "不支持提交记录功能", "")
}

// GetChallengeStats 获取题目统计信息（已禁用 - 不需要此功能）
func (a *App) GetChallengeStats() Response {
	return NewErrorResponse("not_supported", "不支持统计功能", "")
}

// ==================== 辅助方法 ====================

// challengeToDTO 转换题目模型为DTO
func (a *App) challengeToDTO(challenge *models.Challenge) *ChallengeDTO {
	// 转换hints
	hints := make([]HintDTO, 0, len(challenge.Hints))
	for _, hint := range challenge.Hints {
		hints = append(hints, HintDTO{
			ID:         hint.ID,
			Content:    hint.Content,
			Cost:       hint.Cost,
			IsUnlocked: false, // TODO: 根据当前用户判断是否已解锁
		})
	}

	return &ChallengeDTO{
		ID:           challenge.ID,
		Title:        challenge.Title,
		Description:  challenge.Description,
		Category:     challenge.Category,
		Difficulty:   challenge.Difficulty,
		Points:       challenge.Points,
		Flag:         challenge.Flag,
		IsSolved:     len(challenge.SolvedBy) > 0,
		SolvedBy:     challenge.SolvedBy,
		Hints:        hints,
		AssignedTo:   challenge.AssignedTo,
		SubChannelID: challenge.SubChannelID,
		CreatedAt:    challenge.CreatedAt.Unix(),
		UpdatedAt:    challenge.UpdatedAt.Unix(),
	}
}

// GetChallengeProgress 获取某成员的题目进度
func (a *App) GetChallengeProgress(challengeID string, memberID string) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	if challengeID == "" || memberID == "" {
		return NewErrorResponse("invalid_request", "challengeID 与 memberID 不能为空", "")
	}

	if mode == ModeServer && srv != nil {
		progress, err := srv.GetChallengeProgress(challengeID, memberID)
		if err != nil || progress == nil {
			return NewSuccessResponse(map[string]interface{}{
				"challenge_id": challengeID,
				"member_id":    memberID,
				"progress":     0,
			})
		}
		return NewSuccessResponse(map[string]interface{}{
			"challenge_id": progress.ChallengeID,
			"member_id":    progress.MemberID,
			"progress":     progress.Progress,
			"summary":      progress.Summary,
			"updated_at":   progress.UpdatedAt.Unix(),
		})
	}

	if mode == ModeClient && cli != nil {
		pr, err := a.db.ChallengeRepo().GetProgress(challengeID, memberID)
		if err != nil || pr == nil {
			return NewSuccessResponse(map[string]interface{}{
				"challenge_id": challengeID,
				"member_id":    memberID,
				"progress":     0,
			})
		}
		return NewSuccessResponse(map[string]interface{}{
			"challenge_id": pr.ChallengeID,
			"member_id":    pr.MemberID,
			"progress":     pr.Progress,
			"summary":      pr.Summary,
			"updated_at":   pr.UpdatedAt.Unix(),
		})
	}

	return NewErrorResponse("invalid_mode", "无效的运行模式", "")
}
