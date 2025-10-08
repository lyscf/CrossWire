package app

import (
	"time"

	"crosswire/internal/models"
)

// ==================== 成员管理 API ====================

// GetMembers 获取成员列表（带批量统计优化）
func (a *App) GetMembers() Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 获取成员列表
	var members []*models.Member
	var err error
	if mode == ModeServer && a.server != nil {
		members, err = a.server.GetMembers()
	} else if mode == ModeClient && cli != nil {
		members, err = cli.GetMembers()
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}
	if err != nil {
		return NewErrorResponse("query_error", "获取成员失败", err.Error())
	}

	// 🔧 批量获取所有成员的贡献统计（性能优化：一次查询）
	contributionStatsMap, err := a.db.ChallengeRepo().GetAllMembersContributionStats()
	if err != nil {
		a.logger.Warn("[GetMembers] Failed to get contribution stats: %v", err)
		contributionStatsMap = make(map[string]int)
	}

	// 转换为DTO（使用预先计算的统计数据）
	memberDTOs := make([]*MemberDTO, 0, len(members))
	for _, member := range members {
		dto := a.memberToDTOWithStats(member, contributionStatsMap[member.ID])
		memberDTOs = append(memberDTOs, dto)
	}

	return NewSuccessResponse(memberDTOs)
}

// GetMember 获取单个成员信息
func (a *App) GetMember(memberID string) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		a.logger.Warn("[GetMember] Not running")
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	a.logger.Debug("[GetMember] Fetching member info for ID: %s", memberID)

	// 从数据库获取成员
	member, err := a.db.MemberRepo().GetByID(memberID)
	if err != nil {
		a.logger.Error("[GetMember] Failed to get member %s: %v", memberID, err)
		return NewErrorResponse("not_found", "成员不存在", err.Error())
	}

	a.logger.Info("[GetMember] Successfully retrieved member: %s (nickname: %s)",
		member.ID, member.Nickname)

	dto := a.memberToDTO(member)
	return NewSuccessResponse(dto)
}

// GetMyInfo 获取当前用户信息
func (a *App) GetMyInfo() Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		a.logger.Warn("[GetMyInfo] Not running")
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	var memberID string
	if mode == ModeServer && srv != nil {
		// 服务端使用固定的"server"成员ID
		memberID = "server"
		a.logger.Debug("[GetMyInfo] Server mode, using member ID: %s", memberID)
	} else if mode == ModeClient && cli != nil {
		memberID = cli.GetMemberID()
		a.logger.Debug("[GetMyInfo] Client mode, member ID: %s", memberID)
	} else {
		a.logger.Error("[GetMyInfo] Invalid mode: %s", mode)
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	// 从数据库获取成员信息
	a.logger.Debug("[GetMyInfo] Fetching member info for ID: %s", memberID)
	member, err := a.db.MemberRepo().GetByID(memberID)
	if err != nil {
		a.logger.Error("[GetMyInfo] Failed to get member info: %v", err)
		return NewErrorResponse("not_found", "获取用户信息失败", err.Error())
	}

	a.logger.Info("[GetMyInfo] Successfully retrieved member info: %s (nickname: %s, role: %s)",
		member.ID, member.Nickname, member.Role)

	dto := a.memberToDTO(member)
	return NewSuccessResponse(dto)
}

// UpdateMyStatus 更新我的状态
func (a *App) UpdateMyStatus(status models.UserStatus) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 更新状态
	var err error
	if mode == ModeClient && cli != nil {
		err = cli.UpdateStatus(status)
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("update_error", "更新状态失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "状态已更新",
		"status":  status,
	})
}

// UpdateMyProfile 更新我的资料
func (a *App) UpdateMyProfile(nickname, avatar string) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 更新资料
	var err error
	if mode == ModeClient && cli != nil {
		err = cli.UpdateProfile(nickname, avatar)
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("update_error", "更新资料失败", err.Error())
	}

	// 更新本地用户配置
	a.mu.Lock()
	if nickname != "" {
		a.userProfile.Nickname = nickname
	}
	if avatar != "" {
		a.userProfile.Avatar = avatar
	}
	a.mu.Unlock()

	return NewSuccessResponse(map[string]interface{}{
		"message":  "资料已更新",
		"nickname": nickname,
		"avatar":   avatar,
	})
}

// KickMember 踢出成员（仅服务端管理员）
func (a *App) KickMember(req KickMemberRequest) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端管理员可踢出成员", "")
	}

	reason := ""
	if req.Reason != nil {
		reason = *req.Reason
	}

	// 踢出成员
	if err := a.server.KickMember(req.MemberID, reason); err != nil {
		return NewErrorResponse("kick_error", "踢出成员失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "成员已被踢出",
	})
}

// BanMember 封禁成员（仅服务端管理员）
func (a *App) BanMember(req BanMemberRequest) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端管理员可封禁成员", "")
	}

	reason := ""
	if req.Reason != nil {
		reason = *req.Reason
	}

	duration := int64(0)
	if req.Duration != nil {
		duration = *req.Duration
	}

	// 封禁成员
	if err := a.server.BanMember(req.MemberID, reason, time.Duration(duration)*time.Second); err != nil {
		return NewErrorResponse("ban_error", "封禁成员失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "成员已被封禁",
	})
}

// UnbanMember 解封成员（仅服务端管理员）
func (a *App) UnbanMember(memberID string) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端管理员可解封成员", "")
	}

	// 解封成员
	if err := a.server.UnbanMember(memberID); err != nil {
		return NewErrorResponse("unban_error", "解封成员失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "成员已解封",
	})
}

// MuteMember 禁言成员（仅服务端管理员）
func (a *App) MuteMember(memberID string, duration int64) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端管理员可禁言成员", "")
	}

	// 禁言成员
	if err := a.server.MuteMember(memberID, time.Duration(duration)*time.Second, ""); err != nil {
		return NewErrorResponse("mute_error", "禁言成员失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "成员已被禁言",
	})
}

// UnmuteMember 解除禁言（仅服务端管理员）
func (a *App) UnmuteMember(memberID string) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端管理员可解除禁言", "")
	}

	// 解除禁言
	if err := srv.UnmuteMember(memberID); err != nil {
		return NewErrorResponse("unmute_error", "解除禁言失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "已解除禁言",
	})
}

// UpdateMemberRole 更新成员角色（仅服务端管理员）
func (a *App) UpdateMemberRole(memberID string, role models.MemberRole) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	a.mu.RUnlock()

	if mode != ModeServer || srv == nil {
		return NewErrorResponse("permission_denied", "仅服务端管理员可更新角色", "")
	}

	// 更新角色
	if err := a.server.UpdateMemberRole(memberID, role); err != nil {
		return NewErrorResponse("update_error", "更新角色失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "角色已更新",
		"role":    role,
	})
}

// ==================== 辅助方法 ====================

// memberToDTO 转换成员模型为DTO（单个查询，用于GetMember等单个成员查询）
func (a *App) memberToDTO(member *models.Member) *MemberDTO {
	// 获取该成员的参与题目数
	assignedCount, err := a.db.ChallengeRepo().CountAssignedToMember(member.ID)
	if err != nil {
		a.logger.Warn("[memberToDTO] Failed to count assigned challenges for %s: %v", member.ID, err)
		assignedCount = 0
	}

	return a.memberToDTOWithStats(member, assignedCount)
}

// memberToDTOWithStats 转换成员模型为DTO（使用预先计算的统计数据）
func (a *App) memberToDTOWithStats(member *models.Member, assignedCount int) *MemberDTO {
	dto := &MemberDTO{
		ID:           member.ID,
		Nickname:     member.Nickname,
		Avatar:       member.Avatar,
		Role:         member.Role,
		Status:       member.Status,
		IsOnline:     member.IsOnline,
		JoinTime:     member.JoinTime.Unix(),
		LastSeenAt:   member.LastSeenAt.Unix(),
		IsMuted:      member.IsMuted,
		IsBanned:     member.IsBanned,
		MessageCount: member.MessageCount,
		FilesShared:  member.FilesShared,
		OnlineTime:   member.OnlineTime,
	}

	// 提取Skills - 同时提供简单版本（类别名）与详细版本（含等级/经验）
	if len(member.Skills) > 0 {
		skills := make([]string, len(member.Skills))
		details := make([]SkillDetail, len(member.Skills))
		for i, skill := range member.Skills {
			skills[i] = skill.Category
			details[i] = SkillDetail{
				Category:   skill.Category,
				Level:      skill.Level,
				Experience: skill.Experience,
			}
		}
		dto.Skills = skills
		dto.SkillDetails = details
	}

	// 从Metadata提取email和bio
	if member.Metadata != nil {
		if email, ok := member.Metadata["email"].(string); ok {
			dto.Email = email
		}
		if bio, ok := member.Metadata["bio"].(string); ok {
			dto.Bio = bio
		}
	}

	// 🔧 统计参与的题目数（协作平台：分配给该成员的题目）
	dto.SolvedChallenges = assignedCount

	// 🔧 计算贡献度分数（协作平台）
	// 方案：消息数 * 1 + 文件数 * 5 + 参与题目数 * 10
	dto.TotalPoints = member.MessageCount + (member.FilesShared * 5) + (assignedCount * 10)

	return dto
}
