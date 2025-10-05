package models

import (
	"time"

	"gorm.io/gorm"
)

// AuditLog 审计日志
type AuditLog struct {
	ID         int       `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID  string    `gorm:"type:text;not null;index:idx_audit_channel" json:"channel_id"`
	Type       string    `gorm:"type:text;not null;index:idx_audit_type" json:"type"`
	OperatorID string    `gorm:"type:text;not null;index:idx_audit_operator" json:"operator_id"`
	TargetID   string    `gorm:"type:text" json:"target_id,omitempty"`
	Reason     string    `gorm:"type:text" json:"reason,omitempty"`
	Details    JSONField `gorm:"type:text" json:"details,omitempty"`
	Timestamp  time.Time `gorm:"type:integer;not null;index:idx_audit_timestamp" json:"timestamp"`
	IPAddress  string    `gorm:"type:text" json:"ip_address,omitempty"`
	UserAgent  string    `gorm:"type:text" json:"user_agent,omitempty"`

	// 关联
	Channel  *Channel `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"-"`
	Operator *Member  `gorm:"foreignKey:OperatorID" json:"-"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}

// BeforeCreate GORM 钩子
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.Timestamp.IsZero() {
		a.Timestamp = time.Now()
	}
	return nil
}

// UserProfile 用户配置（user.db）
type UserProfile struct {
	ID         string         `gorm:"primaryKey;type:text" json:"id"`
	Nickname   string         `gorm:"type:text;not null" json:"nickname"`
	Avatar     string         `gorm:"type:text" json:"avatar,omitempty"`
	PrivateKey []byte         `gorm:"type:blob;not null" json:"-"` // 加密存储
	PublicKey  []byte         `gorm:"type:blob;not null" json:"-"`
	Skills     SkillTags      `gorm:"type:text" json:"skills,omitempty"`
	Expertise  ExpertiseArray `gorm:"type:text" json:"expertise,omitempty"`
	Bio        string         `gorm:"type:text" json:"bio,omitempty"`
	Theme      string         `gorm:"type:text;default:'dark'" json:"theme"`
	Language   string         `gorm:"type:text;default:'zh-CN'" json:"language"`
	AutoStart  bool           `gorm:"type:integer;default:0" json:"auto_start"`
	CreatedAt  time.Time      `gorm:"type:integer;not null" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"type:integer;not null" json:"updated_at"`
}

// TableName 指定表名
func (UserProfile) TableName() string {
	return "user_profiles"
}

// BeforeCreate GORM 钩子
func (u *UserProfile) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM 钩子
func (u *UserProfile) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}

// RecentChannel 最近加入的频道
type RecentChannel struct {
	ChannelID     string        `gorm:"primaryKey;type:text" json:"channel_id"`
	ChannelName   string        `gorm:"type:text;not null" json:"channel_name"`
	ServerAddress string        `gorm:"type:text" json:"server_address,omitempty"`
	TransportMode TransportMode `gorm:"type:text" json:"transport_mode,omitempty"`
	LastJoined    time.Time     `gorm:"type:integer;not null;index:idx_recent_last_joined" json:"last_joined"`
	Pinned        bool          `gorm:"type:integer;default:0" json:"pinned"`
}

// TableName 指定表名
func (RecentChannel) TableName() string {
	return "recent_channels"
}

// CacheEntry 本地缓存（cache.db）
type CacheEntry struct {
	Key       string    `gorm:"primaryKey;type:text" json:"key"`
	Value     []byte    `gorm:"type:blob;not null" json:"-"`
	ExpiresAt time.Time `gorm:"type:integer;not null;index:idx_cache_expires" json:"expires_at"`
	CreatedAt time.Time `gorm:"type:integer;not null" json:"created_at"`
}

// TableName 指定表名
func (CacheEntry) TableName() string {
	return "cache_entries"
}

// IsExpired 检查是否过期
func (c *CacheEntry) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}
