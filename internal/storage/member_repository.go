package storage

import (
	"time"

	"crosswire/internal/models"

	"gorm.io/gorm"
)

// MemberRepository 成员数据仓库
type MemberRepository struct {
	db *Database
}

// NewMemberRepository 创建成员仓库
func NewMemberRepository(db *Database) *MemberRepository {
	return &MemberRepository{db: db}
}

// Create 创建成员
func (r *MemberRepository) Create(member *models.Member) error {
	return r.db.GetChannelDB().Create(member).Error
}

// GetByID 根据ID获取成员
func (r *MemberRepository) GetByID(memberID string) (*models.Member, error) {
	var member models.Member
	err := r.db.GetChannelDB().Where("id = ?", memberID).First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// GetByChannelID 获取频道所有成员
func (r *MemberRepository) GetByChannelID(channelID string) ([]*models.Member, error) {
	var members []*models.Member
	err := r.db.GetChannelDB().Where("channel_id = ?", channelID).
		Order("joined_at ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

// GetOnlineMembers 获取在线成员
func (r *MemberRepository) GetOnlineMembers(channelID string) ([]*models.Member, error) {
	var members []*models.Member
	err := r.db.GetChannelDB().Where("channel_id = ? AND status != ?", channelID, models.StatusOffline).
		Order("status ASC, joined_at ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

// Update 更新成员信息
func (r *MemberRepository) Update(member *models.Member) error {
	return r.db.GetChannelDB().Save(member).Error
}

// UpdateStatus 更新成员状态
func (r *MemberRepository) UpdateStatus(memberID string, status models.UserStatus) error {
	return r.db.GetChannelDB().Model(&models.Member{}).
		Where("id = ?", memberID).
		Updates(map[string]interface{}{
			"status":    status,
			"last_seen": time.Now(),
		}).Error
}

// UpdateHeartbeat 更新心跳时间
func (r *MemberRepository) UpdateHeartbeat(memberID string) error {
	return r.db.GetChannelDB().Model(&models.Member{}).
		Where("id = ?", memberID).
		UpdateColumn("last_heartbeat", time.Now()).Error
}

// Delete 删除成员
func (r *MemberRepository) Delete(memberID string) error {
	return r.db.GetChannelDB().Where("id = ?", memberID).Delete(&models.Member{}).Error
}

// IncrementMessageCount 增加消息计数
func (r *MemberRepository) IncrementMessageCount(memberID string) error {
	return r.db.GetChannelDB().Model(&models.Member{}).
		Where("id = ?", memberID).
		UpdateColumn("message_count", gorm.Expr("message_count + ?", 1)).Error
}

// IncrementFilesShared 增加文件分享计数
func (r *MemberRepository) IncrementFilesShared(memberID string) error {
	return r.db.GetChannelDB().Model(&models.Member{}).
		Where("id = ?", memberID).
		UpdateColumn("files_shared", gorm.Expr("files_shared + ?", 1)).Error
}

// AddOnlineTime 增加在线时长
func (r *MemberRepository) AddOnlineTime(memberID string, seconds int64) error {
	return r.db.GetChannelDB().Model(&models.Member{}).
		Where("id = ?", memberID).
		UpdateColumn("online_time", gorm.Expr("online_time + ?", seconds)).Error
}

// UpdateSkills 更新技能标签
func (r *MemberRepository) UpdateSkills(memberID string, skills models.SkillTags) error {
	return r.db.GetChannelDB().Model(&models.Member{}).
		Where("id = ?", memberID).
		Update("skills", skills).Error
}

// UpdateCurrentTask 更新当前任务
func (r *MemberRepository) UpdateCurrentTask(memberID string, task *models.CurrentTask) error {
	return r.db.GetChannelDB().Model(&models.Member{}).
		Where("id = ?", memberID).
		Update("current_task", task).Error
}

// MuteMember 禁言成员
func (r *MemberRepository) MuteMember(record *models.MuteRecord) error {
	return r.db.GetChannelDB().Create(record).Error
}

// UnmuteMember 解除禁言
func (r *MemberRepository) UnmuteMember(memberID, unmutedBy string) error {
	now := time.Now()
	return r.db.GetChannelDB().Model(&models.MuteRecord{}).
		Where("member_id = ? AND active = ?", memberID, true).
		Updates(map[string]interface{}{
			"active":     false,
			"unmuted_at": now,
			"unmuted_by": unmutedBy,
		}).Error
}

// IsMuted 检查是否被禁言
func (r *MemberRepository) IsMuted(memberID string) (bool, error) {
	var count int64
	err := r.db.GetChannelDB().Model(&models.MuteRecord{}).
		Where("member_id = ? AND active = ?", memberID, true).
		Where("expires_at IS NULL OR expires_at > ?", time.Now()).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetMuteRecord 获取禁言记录
func (r *MemberRepository) GetMuteRecord(memberID string) (*models.MuteRecord, error) {
	var record models.MuteRecord
	err := r.db.GetChannelDB().Where("member_id = ? AND active = ?", memberID, true).
		Order("muted_at DESC").
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetMuteRecords 获取频道的有效禁言记录（active 且未过期）
func (r *MemberRepository) GetMuteRecords(channelID string) ([]*models.MuteRecord, error) {
	var records []*models.MuteRecord
	err := r.db.GetChannelDB().Where("channel_id = ? AND active = ?", channelID, true).
		Where("expires_at IS NULL OR expires_at > ?", time.Now()).
		Order("muted_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// TODO: 实现以下方法
// - GetMembersByRole() 按角色获取成员
// - SearchMembers() 搜索成员
// - GetMemberStats() 获取成员统计信息
// - UpdateMetadata() 更新成员元数据
