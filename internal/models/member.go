package models

import (
	"time"

	"gorm.io/gorm"
)

// Member 成员模型
type Member struct {
	ID            string         `gorm:"primaryKey;type:text" json:"id"`
	ChannelID     string         `gorm:"type:text;not null;index:idx_members_channel" json:"channel_id"`
	Nickname      string         `gorm:"type:text;not null" json:"nickname"`
	Avatar        string         `gorm:"type:text" json:"avatar,omitempty"`
	Role          Role           `gorm:"type:text;not null" json:"role"`
	Status        UserStatus     `gorm:"type:text;default:'offline';index:idx_members_status" json:"status"`
	PublicKey     []byte         `gorm:"type:blob" json:"-"`
	LastIP        string         `gorm:"type:text" json:"last_ip,omitempty"`
	LastMAC       string         `gorm:"type:text" json:"last_mac,omitempty"`
	Skills        SkillTags      `gorm:"type:text" json:"skills,omitempty"`
	Expertise     ExpertiseArray `gorm:"type:text" json:"expertise,omitempty"`
	CurrentTask   *CurrentTask   `gorm:"type:text" json:"current_task,omitempty"`
	MessageCount  int            `gorm:"type:integer;default:0" json:"message_count"`
	FilesShared   int            `gorm:"type:integer;default:0" json:"files_shared"`
	OnlineTime    int64          `gorm:"type:integer;default:0" json:"online_time"` // 秒
	JoinedAt      time.Time      `gorm:"type:integer;not null" json:"joined_at"`
	LastSeen      time.Time      `gorm:"type:integer;not null;index:idx_members_last_seen" json:"last_seen"`
	LastHeartbeat time.Time      `gorm:"type:integer;not null" json:"last_heartbeat"`
	Metadata      JSONField      `gorm:"type:text" json:"metadata,omitempty"`

	// 关联
	Channel *Channel `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (Member) TableName() string {
	return "members"
}

// BeforeCreate GORM 钩子
func (m *Member) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if m.JoinedAt.IsZero() {
		m.JoinedAt = now
	}
	if m.LastSeen.IsZero() {
		m.LastSeen = now
	}
	if m.LastHeartbeat.IsZero() {
		m.LastHeartbeat = now
	}
	return nil
}

// MuteRecord 禁言记录
type MuteRecord struct {
	ID        string     `gorm:"primaryKey;type:text" json:"id"`
	ChannelID string     `gorm:"type:text;not null;index:idx_mute_channel" json:"channel_id"`
	MemberID  string     `gorm:"type:text;not null;index:idx_mute_member" json:"member_id"`
	MutedBy   string     `gorm:"type:text;not null" json:"muted_by"`
	Reason    string     `gorm:"type:text" json:"reason,omitempty"`
	MutedAt   time.Time  `gorm:"type:integer;not null" json:"muted_at"`
	Duration  *int64     `gorm:"type:integer" json:"duration,omitempty"` // 秒数，NULL=永久
	ExpiresAt *time.Time `gorm:"type:integer;index:idx_mute_expires" json:"expires_at,omitempty"`
	Active    bool       `gorm:"type:integer;default:1;index:idx_mute_active" json:"active"`
	UnmutedAt *time.Time `gorm:"type:integer" json:"unmuted_at,omitempty"`
	UnmutedBy string     `gorm:"type:text" json:"unmuted_by,omitempty"`

	// 关联
	Channel *Channel `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"-"`
	Member  *Member  `gorm:"foreignKey:MemberID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (MuteRecord) TableName() string {
	return "mute_records"
}

// BeforeCreate GORM 钩子
func (m *MuteRecord) BeforeCreate(tx *gorm.DB) error {
	if m.MutedAt.IsZero() {
		m.MutedAt = time.Now()
	}
	return nil
}

// IsExpired 检查是否过期
func (m *MuteRecord) IsExpired() bool {
	if m.ExpiresAt == nil {
		return false // 永久禁言
	}
	return time.Now().After(*m.ExpiresAt)
}
