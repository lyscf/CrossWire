package server

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"crosswire/internal/events"
	"crosswire/internal/models"
	"crosswire/internal/transport"
)

// ChannelManager 频道管理器
// 参考: docs/ARCHITECTURE.md - 3.1.2 服务端模块 - ChannelManager
type ChannelManager struct {
	server *Server

	// 频道信息
	channel *models.Channel

	// 成员管理
	members      map[string]*models.Member // memberID -> Member
	membersMutex sync.RWMutex

	// 禁言记录
	muteRecords map[string]*models.MuteRecord // memberID -> MuteRecord
	muteMutex   sync.RWMutex
}

// NewChannelManager 创建频道管理器
func NewChannelManager(server *Server) *ChannelManager {
	return &ChannelManager{
		server:      server,
		members:     make(map[string]*models.Member),
		muteRecords: make(map[string]*models.MuteRecord),
	}
}

// Initialize 初始化频道
func (cm *ChannelManager) Initialize() error {
	// 尝试从数据库加载频道
	channel, err := cm.server.channelRepo.GetByID(cm.server.config.ChannelID)
	if err != nil {
		// 频道不存在，创建新频道
		channel = &models.Channel{
			ID:            cm.server.config.ChannelID,
			Name:          cm.server.config.ChannelName,
			MaxMembers:    cm.server.config.MaxMembers,
			TransportMode: cm.server.config.TransportMode,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := cm.server.channelRepo.Create(channel); err != nil {
			return fmt.Errorf("failed to create channel: %w", err)
		}

		cm.server.logger.Info("[ChannelManager] Created new channel: %s", channel.Name)
	} else {
		cm.server.logger.Info("[ChannelManager] Loaded existing channel: %s", channel.Name)
	}

	cm.channel = channel

	// 加载成员列表
	if err := cm.loadMembers(); err != nil {
		return fmt.Errorf("failed to load members: %w", err)
	}

	// 加载禁言记录
	if err := cm.loadMuteRecords(); err != nil {
		return fmt.Errorf("failed to load mute records: %w", err)
	}

	return nil
}

// loadMembers 加载成员列表
func (cm *ChannelManager) loadMembers() error {
	members, err := cm.server.memberRepo.GetByChannelID(cm.server.config.ChannelID)
	if err != nil {
		return err
	}

	cm.membersMutex.Lock()
	defer cm.membersMutex.Unlock()

	for _, member := range members {
		cm.members[member.ID] = member
	}

	cm.server.logger.Info("[ChannelManager] Loaded %d members", len(members))

	return nil
}

// loadMuteRecords 加载禁言记录
func (cm *ChannelManager) loadMuteRecords() error {
	// TODO: 添加GetMuteRecords方法到MemberRepository
	// 暂时使用空列表
	var records []*models.MuteRecord

	if len(records) > 0 {
		cm.muteMutex.Lock()
		defer cm.muteMutex.Unlock()

		now := time.Now()
		for _, record := range records {
			// 只加载未过期的禁言记录
			if record.ExpiresAt != nil && record.ExpiresAt.After(now) {
				cm.muteRecords[record.MemberID] = record
			}
		}

		cm.server.logger.Info("[ChannelManager] Loaded %d active mute records", len(cm.muteRecords))
	}

	return nil
}

// AddMember 添加成员
func (cm *ChannelManager) AddMember(member *models.Member) error {
	if member == nil {
		return errors.New("member is nil")
	}

	// 检查频道是否已满
	if cm.GetTotalCount() >= cm.channel.MaxMembers {
		return errors.New("channel is full")
	}

	// 检查成员是否已存在
	if cm.HasMember(member.ID) {
		return errors.New("member already exists")
	}

	// 设置频道ID
	member.ChannelID = cm.server.config.ChannelID

	// 保存到数据库
	if err := cm.server.memberRepo.Create(member); err != nil {
		return fmt.Errorf("failed to add member to database: %w", err)
	}

	// 添加到内存
	cm.membersMutex.Lock()
	cm.members[member.ID] = member
	cm.membersMutex.Unlock()

	cm.server.logger.Info("[ChannelManager] Member added: %s (%s)", member.Nickname, member.ID)

	// 发布事件
	cm.server.eventBus.Publish(events.EventMemberJoined, events.NewMemberJoinedEvent(member, cm.server.config.ChannelID))

	return nil
}

// RemoveMember 移除成员
func (cm *ChannelManager) RemoveMember(memberID string, reason string) error {
	member := cm.GetMemberByID(memberID)
	if member == nil {
		return errors.New("member not found")
	}

	// 从数据库删除
	if err := cm.server.memberRepo.Delete(memberID); err != nil {
		return fmt.Errorf("failed to delete member from database: %w", err)
	}

	// 从内存删除
	cm.membersMutex.Lock()
	delete(cm.members, memberID)
	cm.membersMutex.Unlock()

	cm.server.logger.Info("[ChannelManager] Member removed: %s (%s), reason: %s",
		member.Nickname, memberID, reason)

	// 发布事件
	cm.server.eventBus.Publish(events.EventMemberLeft, events.NewMemberLeftEvent(member, cm.server.config.ChannelID, reason))

	return nil
}

// GetMember 获取成员
func (cm *ChannelManager) GetMember(memberID string) (*models.Member, error) {
	member := cm.GetMemberByID(memberID)
	if member == nil {
		return nil, errors.New("member not found")
	}
	return member, nil
}

// GetMemberByID 根据ID获取成员
func (cm *ChannelManager) GetMemberByID(memberID string) *models.Member {
	cm.membersMutex.RLock()
	defer cm.membersMutex.RUnlock()

	return cm.members[memberID]
}

// GetMembers 获取所有成员
func (cm *ChannelManager) GetMembers() ([]*models.Member, error) {
	cm.membersMutex.RLock()
	defer cm.membersMutex.RUnlock()

	members := make([]*models.Member, 0, len(cm.members))
	for _, member := range cm.members {
		members = append(members, member)
	}

	return members, nil
}

// HasMember 检查成员是否存在
func (cm *ChannelManager) HasMember(memberID string) bool {
	cm.membersMutex.RLock()
	defer cm.membersMutex.RUnlock()

	_, exists := cm.members[memberID]
	return exists
}

// UpdateMemberStatus 更新成员状态
func (cm *ChannelManager) UpdateMemberStatus(memberID string, status models.UserStatus) error {
	member := cm.GetMemberByID(memberID)
	if member == nil {
		return errors.New("member not found")
	}

	oldStatus := member.Status
	member.Status = status
	member.LastSeen = time.Now()

	// 更新数据库
	if err := cm.server.memberRepo.UpdateStatus(memberID, status); err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}

	// 发布事件
	cm.server.eventBus.Publish(events.EventStatusChanged, events.NewStatusChangedEvent(
		memberID, cm.server.config.ChannelID, oldStatus, status))

	return nil
}

// MuteMember 禁言成员
func (cm *ChannelManager) MuteMember(memberID string, duration time.Duration, reason string) error {
	member := cm.GetMemberByID(memberID)
	if member == nil {
		return errors.New("member not found")
	}

	// 创建禁言记录
	expiresAt := time.Now().Add(duration)
	muteRecord := &models.MuteRecord{
		ChannelID: cm.server.config.ChannelID,
		MemberID:  memberID,
		Reason:    reason,
		MutedAt:   time.Now(),
		ExpiresAt: &expiresAt,
	}

	// 保存到数据库
	// TODO: 添加MuteMember方法到MemberRepository
	// 暂时只更新内存

	// 添加到内存
	cm.muteMutex.Lock()
	cm.muteRecords[memberID] = muteRecord
	cm.muteMutex.Unlock()

	cm.server.logger.Info("[ChannelManager] Member muted: %s, duration: %v, reason: %s",
		member.Nickname, duration, reason)

	// 发布事件
	cm.server.eventBus.Publish(events.EventMemberMuted, &events.MemberEvent{
		Member:    member,
		ChannelID: cm.server.config.ChannelID,
		Action:    "muted",
		Reason:    reason,
	})

	return nil
}

// UnmuteMember 解除禁言
func (cm *ChannelManager) UnmuteMember(memberID string) error {
	member := cm.GetMemberByID(memberID)
	if member == nil {
		return errors.New("member not found")
	}

	// 从数据库删除
	// TODO: 添加UnmuteMember方法到MemberRepository
	// 暂时只更新内存

	// 从内存删除
	cm.muteMutex.Lock()
	delete(cm.muteRecords, memberID)
	cm.muteMutex.Unlock()

	cm.server.logger.Info("[ChannelManager] Member unmuted: %s", member.Nickname)

	// 发布事件
	cm.server.eventBus.Publish(events.EventMemberUnmuted, &events.MemberEvent{
		Member:    member,
		ChannelID: cm.server.config.ChannelID,
		Action:    "unmuted",
	})

	return nil
}

// IsMuted 检查成员是否被禁言
func (cm *ChannelManager) IsMuted(memberID string) bool {
	cm.muteMutex.RLock()
	muteRecord, exists := cm.muteRecords[memberID]
	cm.muteMutex.RUnlock()

	if !exists {
		return false
	}

	// 检查是否过期
	if muteRecord.ExpiresAt != nil && time.Now().After(*muteRecord.ExpiresAt) {
		// 过期，自动解除禁言
		cm.muteMutex.Lock()
		delete(cm.muteRecords, memberID)
		cm.muteMutex.Unlock()
		return false
	}

	return true
}

// GetChannel 获取频道信息
func (cm *ChannelManager) GetChannel() (*models.Channel, error) {
	if cm.channel == nil {
		return nil, errors.New("channel not initialized")
	}
	return cm.channel, nil
}

// UpdateChannel 更新频道信息
func (cm *ChannelManager) UpdateChannel(updates map[string]interface{}) error {
	if cm.channel == nil {
		return errors.New("channel not initialized")
	}

	// 应用更新
	if name, ok := updates["name"].(string); ok {
		cm.channel.Name = name
	}
	if maxMembers, ok := updates["max_members"].(int); ok {
		cm.channel.MaxMembers = maxMembers
	}

	cm.channel.UpdatedAt = time.Now()

	// 保存到数据库
	if err := cm.server.channelRepo.Update(cm.channel); err != nil {
		return fmt.Errorf("failed to update channel: %w", err)
	}

	cm.server.logger.Info("[ChannelManager] Channel updated: %s", cm.channel.Name)

	// 发布事件
	cm.server.eventBus.Publish(events.EventChannelUpdated, events.NewChannelEvent(
		events.EventChannelUpdated, cm.channel, "", "updated"))

	return nil
}

// GetOnlineCount 获取在线成员数
func (cm *ChannelManager) GetOnlineCount() int {
	cm.membersMutex.RLock()
	defer cm.membersMutex.RUnlock()

	count := 0
	for _, member := range cm.members {
		if member.Status == models.StatusOnline || member.Status == models.StatusBusy {
			count++
		}
	}

	return count
}

// GetTotalCount 获取总成员数
func (cm *ChannelManager) GetTotalCount() int {
	cm.membersMutex.RLock()
	defer cm.membersMutex.RUnlock()

	return len(cm.members)
}

// HandleMemberStatus 处理成员状态变化
func (cm *ChannelManager) HandleMemberStatus(msg *transport.Message) {
	// TODO: 实现成员状态变化处理
	cm.server.logger.Debug("[ChannelManager] Member status update: %s", msg.SenderID)
}

// KickMember 踢出成员（主动移除）
// 参考: docs/ARCHITECTURE.md - 3.1.2 服务端模块 - ChannelManager
func (cm *ChannelManager) KickMember(memberID, reason string, kickedBy string) error {
	member := cm.GetMemberByID(memberID)
	if member == nil {
		return errors.New("member not found")
	}

	// 检查踢出者是否有权限（简化处理，实际需要检查角色）
	kicker := cm.GetMemberByID(kickedBy)
	if kicker == nil {
		return errors.New("kicker not found")
	}

	// 从数据库删除
	if err := cm.server.memberRepo.Delete(memberID); err != nil {
		return fmt.Errorf("failed to delete member from database: %w", err)
	}

	// 从内存删除
	cm.membersMutex.Lock()
	delete(cm.members, memberID)
	cm.membersMutex.Unlock()

	cm.server.logger.Info("[ChannelManager] Member kicked: %s by %s, reason: %s",
		member.Nickname, kicker.Nickname, reason)

	// 发布事件
	cm.server.eventBus.Publish(events.EventMemberKicked, events.NewMemberLeftEvent(member, cm.server.config.ChannelID, reason))

	// 通知被踢出的成员（通过特殊消息）
	cm.notifyMemberKicked(member, reason, kicker.Nickname)

	return nil
}

// BanMember 封禁成员（永久或长期）
func (cm *ChannelManager) BanMember(memberID string, reason string, bannedBy string, duration time.Duration) error {
	member := cm.GetMemberByID(memberID)
	if member == nil {
		return errors.New("member not found")
	}

	// 更新成员状态为封禁
	member.Status = models.StatusOffline
	member.LastSeen = time.Now()

	// 创建封禁记录（使用禁言记录结构，但时间更长）
	var expiresAt *time.Time
	if duration > 0 {
		expiry := time.Now().Add(duration)
		expiresAt = &expiry
	}

	banRecord := &models.MuteRecord{
		ChannelID: cm.server.config.ChannelID,
		MemberID:  memberID,
		Reason:    fmt.Sprintf("BANNED: %s", reason),
		MutedAt:   time.Now(),
		ExpiresAt: expiresAt,
	}

	// 保存封禁记录
	cm.muteMutex.Lock()
	cm.muteRecords[memberID] = banRecord
	cm.muteMutex.Unlock()

	// 更新数据库
	if err := cm.server.memberRepo.Update(member); err != nil {
		return fmt.Errorf("failed to update member: %w", err)
	}

	cm.server.logger.Info("[ChannelManager] Member banned: %s by %s, duration: %v, reason: %s",
		member.Nickname, bannedBy, duration, reason)

	// 发布事件
	cm.server.eventBus.Publish(events.EventMemberBanned, &events.MemberEvent{
		Member:    member,
		ChannelID: cm.server.config.ChannelID,
		Action:    "banned",
		Reason:    reason,
	})

	return nil
}

// UnbanMember 解封成员
func (cm *ChannelManager) UnbanMember(memberID string) error {
	member := cm.GetMemberByID(memberID)
	if member == nil {
		return errors.New("member not found")
	}

	// 移除封禁记录
	cm.muteMutex.Lock()
	delete(cm.muteRecords, memberID)
	cm.muteMutex.Unlock()

	cm.server.logger.Info("[ChannelManager] Member unbanned: %s", member.Nickname)

	// 发布事件
	cm.server.eventBus.Publish(events.EventMemberUnbanned, &events.MemberEvent{
		Member:    member,
		ChannelID: cm.server.config.ChannelID,
		Action:    "unbanned",
	})

	return nil
}

// IsBanned 检查成员是否被封禁
func (cm *ChannelManager) IsBanned(memberID string) bool {
	cm.muteMutex.RLock()
	record, exists := cm.muteRecords[memberID]
	cm.muteMutex.RUnlock()

	if !exists {
		return false
	}

	// 检查是否是封禁记录（以"BANNED:"开头）
	if len(record.Reason) < 7 || record.Reason[:7] != "BANNED:" {
		return false
	}

	// 检查是否过期
	if record.ExpiresAt != nil && time.Now().After(*record.ExpiresAt) {
		// 过期，自动解封
		cm.muteMutex.Lock()
		delete(cm.muteRecords, memberID)
		cm.muteMutex.Unlock()
		return false
	}

	return true
}

// UpdateMemberRole 更新成员角色
func (cm *ChannelManager) UpdateMemberRole(memberID string, role models.Role) error {
	member := cm.GetMemberByID(memberID)
	if member == nil {
		return errors.New("member not found")
	}

	oldRole := member.Role
	member.Role = role

	// 更新数据库
	if err := cm.server.memberRepo.Update(member); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	cm.server.logger.Info("[ChannelManager] Member role updated: %s from %s to %s",
		member.Nickname, oldRole, role)

	// 发布事件
	cm.server.eventBus.Publish(events.EventMemberRoleChanged, &events.MemberEvent{
		Member:    member,
		ChannelID: cm.server.config.ChannelID,
		Action:    "role_changed",
	})

	return nil
}

// GetMembersByRole 根据角色获取成员列表
func (cm *ChannelManager) GetMembersByRole(role models.Role) ([]*models.Member, error) {
	cm.membersMutex.RLock()
	defer cm.membersMutex.RUnlock()

	members := make([]*models.Member, 0)
	for _, member := range cm.members {
		if member.Role == role {
			members = append(members, member)
		}
	}

	return members, nil
}

// GetOnlineMembers 获取在线成员列表
func (cm *ChannelManager) GetOnlineMembers() ([]*models.Member, error) {
	cm.membersMutex.RLock()
	defer cm.membersMutex.RUnlock()

	members := make([]*models.Member, 0)
	for _, member := range cm.members {
		if member.Status == models.StatusOnline || member.Status == models.StatusBusy {
			members = append(members, member)
		}
	}

	return members, nil
}

// GetMutedMembers 获取被禁言的成员列表
func (cm *ChannelManager) GetMutedMembers() ([]*models.Member, error) {
	cm.muteMutex.RLock()
	mutedIDs := make([]string, 0, len(cm.muteRecords))
	for memberID := range cm.muteRecords {
		mutedIDs = append(mutedIDs, memberID)
	}
	cm.muteMutex.RUnlock()

	members := make([]*models.Member, 0)
	for _, memberID := range mutedIDs {
		if member := cm.GetMemberByID(memberID); member != nil {
			members = append(members, member)
		}
	}

	return members, nil
}

// notifyMemberKicked 通知成员被踢出
func (cm *ChannelManager) notifyMemberKicked(member *models.Member, reason, kickedBy string) {
	// 构造系统消息
	msg := &models.Message{
		ID:        fmt.Sprintf("kick-%s-%d", member.ID, time.Now().Unix()),
		ChannelID: cm.server.config.ChannelID,
		SenderID:  "server",
		Type:      models.MessageTypeSystem,
		Timestamp: time.Now(),
	}

	// 设置系统消息内容（直接构造map）
	msg.Content = models.MessageContent{
		"event":     "member_kicked",
		"actor_id":  kickedBy,
		"target_id": member.ID,
		"reason":    reason,
		"extra": map[string]interface{}{
			"member_id": member.ID,
			"nickname":  member.Nickname,
			"kicked_at": time.Now().Unix(),
		},
	}

	// 广播踢出通知
	if err := cm.server.broadcastManager.Broadcast(msg); err != nil {
		cm.server.logger.Error("[ChannelManager] Failed to broadcast kick notification: %v", err)
	}
}
