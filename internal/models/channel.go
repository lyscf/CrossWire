package models

import (
	"time"

	"gorm.io/gorm"
)

// Channel 频道模型
type Channel struct {
	ID              string        `gorm:"primaryKey;type:text" json:"id"`
	Name            string        `gorm:"type:text;not null" json:"name"`
	ParentChannelID string        `gorm:"type:text;index:idx_channels_parent" json:"parent_channel_id,omitempty"` // 如果为空则是主频道，否则是子频道
	PasswordHash    string        `gorm:"type:text;not null" json:"-"`
	Salt            []byte        `gorm:"type:blob;not null" json:"-"`
	CreatedAt       time.Time     `gorm:"not null" json:"created_at"`
	CreatorID       string        `gorm:"type:text;not null" json:"creator_id"`
	MaxMembers      int           `gorm:"type:integer;default:50" json:"max_members"`
	TransportMode   TransportMode `gorm:"type:text;default:'auto'" json:"transport_mode"`
	Port            int           `gorm:"type:integer" json:"port,omitempty"`
	Interface       string        `gorm:"type:text" json:"interface,omitempty"`
	EncryptionKey   []byte        `gorm:"type:blob;not null" json:"-"`
	KeyVersion      int           `gorm:"type:integer;default:1" json:"key_version"`
	MessageCount    int64         `gorm:"type:integer;default:0" json:"message_count"`
	FileCount       int64         `gorm:"type:integer;default:0" json:"file_count"`
	TotalTraffic    uint64        `gorm:"type:integer;default:0" json:"total_traffic"`
	Metadata        JSONField     `gorm:"type:text" json:"metadata,omitempty"`
	UpdatedAt       time.Time     `gorm:"not null" json:"updated_at"`

	// 运行时关联（不存储到数据库）
	Members        []*Member        `gorm:"-" json:"members,omitempty"`
	OnlineCount    int              `gorm:"-" json:"online_count"`
	PinnedMessages []*PinnedMessage `gorm:"-" json:"pinned_messages,omitempty"`
}

// TableName 指定表名
func (Channel) TableName() string {
	return "channels"
}

// BeforeCreate GORM 钩子：创建前
func (c *Channel) BeforeCreate(tx *gorm.DB) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now()
	}
	return nil
}

// BeforeUpdate GORM 钩子：更新前
func (c *Channel) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// PinnedMessage 置顶消息
type PinnedMessage struct {
	ID           int       `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID    string    `gorm:"type:text;not null;index:idx_pinned_channel" json:"channel_id"`
	MessageID    string    `gorm:"type:text;not null;uniqueIndex:idx_channel_message" json:"message_id"`
	PinnedBy     string    `gorm:"type:text;not null" json:"pinned_by"`
	Reason       string    `gorm:"type:text" json:"reason,omitempty"`
	PinnedAt     time.Time `gorm:"not null" json:"pinned_at"`
	DisplayOrder int       `gorm:"type:integer;default:0;index:idx_pinned_channel" json:"display_order"`

	// 关联
	Channel *Channel `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"-"`
	Message *Message `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (PinnedMessage) TableName() string {
	return "pinned_messages"
}

// BeforeCreate GORM 钩子
func (p *PinnedMessage) BeforeCreate(tx *gorm.DB) error {
	if p.PinnedAt.IsZero() {
		p.PinnedAt = time.Now()
	}
	return nil
}
