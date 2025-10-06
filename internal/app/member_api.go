package app

import (
	"time"

	"crosswire/internal/models"
)

// ==================== 成员管理 API ====================

// GetMembers 获取成员列表
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

	// 转换为DTO
	memberDTOs := make([]*MemberDTO, 0, len(members))
	for _, member := range members {
		dto := a.memberToDTO(member)
		memberDTOs = append(memberDTOs, dto)
	}

	return NewSuccessResponse(memberDTOs)
}

// GetMember 获取单个成员信息
func (a *App) GetMember(memberID string) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 从数据库获取成员
	member, err := a.db.MemberRepo().GetByID(memberID)
	if err != nil {
		return NewErrorResponse("not_found", "成员不存在", err.Error())
	}

	dto := a.memberToDTO(member)
	return NewSuccessResponse(dto)
}

// GetMyInfo 获取当前用户信息
func (a *App) GetMyInfo() Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	var memberID string
	if mode == ModeServer && a.server != nil {
		// 服务端无本地成员ID，简化：返回错误
		return NewErrorResponse("invalid_mode", "服务端无本地成员", "")
	} else if mode == ModeClient && cli != nil {
		memberID = cli.GetMemberID()
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	// 从数据库获取成员信息
	member, err := a.db.MemberRepo().GetByID(memberID)
	if err != nil {
		return NewErrorResponse("not_found", "获取用户信息失败", err.Error())
	}

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

// memberToDTO 转换成员模型为DTO
func (a *App) memberToDTO(member *models.Member) *MemberDTO {
	return &MemberDTO{
		ID:         member.ID,
		Nickname:   member.Nickname,
		Avatar:     member.Avatar,
		Role:       member.Role,
		Status:     member.Status,
		IsOnline:   member.IsOnline,
		JoinTime:   member.JoinTime,
		LastSeenAt: member.LastSeenAt,
		IsMuted:    member.IsMuted,
		IsBanned:   member.IsBanned,
	}
}
