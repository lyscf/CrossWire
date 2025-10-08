package models

import (
	"time"

	"gorm.io/gorm"
)

// Member 成员模型
type Member struct {
	ID           string         `gorm:"primaryKey;type:text" json:"id"`
	ChannelID    string         `gorm:"type:text;not null;index:idx_members_channel" json:"channel_id"`
	Nickname     string         `gorm:"type:text;not null" json:"nickname"`
	Avatar       string         `gorm:"type:text" json:"avatar,omitempty"`
	Role         Role           `gorm:"type:text;not null" json:"role"`
	Status       UserStatus     `gorm:"type:text;default:'offline';index:idx_members_status" json:"status"`
	PublicKey    []byte         `gorm:"type:blob" json:"-"`
	LastIP       string         `gorm:"type:text" json:"last_ip,omitempty"`
	LastMAC      string         `gorm:"type:text" json:"last_mac,omitempty"`
	Skills       SkillTags      `gorm:"type:text" json:"skills,omitempty"`
	Expertise    ExpertiseArray `gorm:"type:text" json:"expertise,omitempty"`
	CurrentTask  *CurrentTask   `gorm:"type:text" json:"current_task,omitempty"`
	MessageCount int            `gorm:"type:integer;default:0" json:"message_count"`
	FilesShared  int            `gorm:"type:integer;default:0" json:"files_shared"`
	OnlineTime   int64          `gorm:"type:integer;default:0" json:"online_time"` // 秒

	// APP层使用的字段
	IsOnline   bool      `gorm:"type:integer;default:0;index:idx_members_online" json:"is_online"`
	IsMuted    bool      `gorm:"type:integer;default:0;index:idx_members_muted" json:"is_muted"`
	IsBanned   bool      `gorm:"type:integer;default:0;index:idx_members_banned" json:"is_banned"`
	JoinTime   time.Time `gorm:"not null" json:"join_time"`
	LastSeenAt time.Time `gorm:"not null" json:"last_seen_at"`

	// 原有时间字段
	JoinedAt      time.Time `gorm:"not null" json:"joined_at"`
	LastHeartbeat time.Time `gorm:"not null" json:"last_heartbeat"`
	Metadata      JSONField `gorm:"type:text" json:"metadata,omitempty"`

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
	if m.JoinTime.IsZero() {
		m.JoinTime = now
	}
	if m.LastSeenAt.IsZero() {
		m.LastSeenAt = now
	}
	if m.LastHeartbeat.IsZero() {
		m.LastHeartbeat = now
	}
	return nil
}

// AfterFind GORM 钩子 - 同步兼容字段
func (m *Member) AfterFind(tx *gorm.DB) error {
	// 同步 JoinTime（向后兼容）
	if m.JoinTime.IsZero() && !m.JoinedAt.IsZero() {
		m.JoinTime = m.JoinedAt
	}
	// 同步 JoinedAt（向后兼容）
	if m.JoinedAt.IsZero() && !m.JoinTime.IsZero() {
		m.JoinedAt = m.JoinTime
	}
	// 判断在线状态
	m.IsOnline = m.Status != StatusOffline && time.Since(m.LastHeartbeat) < 30*time.Second
	return nil
}

// MuteRecord 禁言记录
type MuteRecord struct {
	ID        string    `gorm:"primaryKey;type:text" json:"id"`
	ChannelID string    `gorm:"type:text;not null;index:idx_mute_channel" json:"channel_id"`
	MemberID  string    `gorm:"type:text;not null;index:idx_mute_member" json:"member_id"`
	MutedBy   string    `gorm:"type:text;not null" json:"muted_by"`
	Reason    string    `gorm:"type:text" json:"reason,omitempty"`
	MutedAt   time.Time `gorm:"not null" json:"muted_at"`
	Duration  *int64    `gorm:"type:integer" json:"duration,omitempty"` // 秒数，NULL=永久
	ExpiresAt time.Time `gorm:"index:idx_mute_expires" json:"expires_at,omitempty"`
	Active    bool      `gorm:"type:integer;default:1;index:idx_mute_active" json:"active"`
	UnmutedAt time.Time `json:"unmuted_at,omitempty"`
	UnmutedBy string    `gorm:"type:text" json:"unmuted_by,omitempty"`

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
	if m.ExpiresAt.IsZero() {
		return false // 永久禁言
	}
	return time.Now().After(m.ExpiresAt)
}
