package models

import (
	"time"

	"gorm.io/gorm"
)

// Message 消息模型
type Message struct {
	ID             string         `gorm:"primaryKey;type:text" json:"id"`
	ChannelID      string         `gorm:"type:text;not null;index:idx_messages_channel_time" json:"channel_id"`
	SenderID       string         `gorm:"type:text;not null;index:idx_messages_sender" json:"sender_id"`
	SenderNickname string         `gorm:"type:text;not null" json:"sender_nickname"`
	Type           MessageType    `gorm:"type:text;not null" json:"type"`
	Content        MessageContent `gorm:"type:text;not null" json:"content"`
	ContentText    string         `gorm:"type:text" json:"content_text,omitempty"` // 用于全文搜索
	ReplyToID      *string        `gorm:"type:text;index:idx_messages_reply_to" json:"reply_to_id,omitempty"`
	ThreadID       string         `gorm:"type:text;index:idx_messages_thread" json:"thread_id,omitempty"`
	Mentions       StringArray    `gorm:"type:text" json:"mentions,omitempty"`
	Tags           StringArray    `gorm:"type:text" json:"tags,omitempty"`

	// APP层使用的字段（兼容）
	IsDeleted bool `gorm:"type:integer;default:0;index:idx_messages_is_deleted" json:"is_deleted"`
	IsPinned  bool `gorm:"type:integer;default:0;index:idx_messages_is_pinned" json:"is_pinned"`

	// 原有字段
	Pinned     bool      `gorm:"type:integer;default:0;index:idx_messages_pinned" json:"pinned"`
	Deleted    bool      `gorm:"type:integer;default:0;index:idx_messages_deleted" json:"deleted"`
	DeletedBy  string    `gorm:"type:text" json:"deleted_by,omitempty"`
	DeletedAt  time.Time `gorm:"not null" json:"deleted_at"`
	EditedAt   time.Time `gorm:"not null" json:"edited_at"`
	Timestamp  time.Time `gorm:"not null;index:idx_messages_channel_time" json:"timestamp"`
	Encrypted  bool      `gorm:"type:integer;default:1" json:"encrypted"`
	KeyVersion int       `gorm:"type:integer;default:1" json:"key_version"`
	Metadata   JSONField `gorm:"type:text" json:"metadata,omitempty"`

	// 题目聊天室支持
	ChallengeID string `gorm:"type:text;index:idx_messages_challenge" json:"challenge_id,omitempty"`
	RoomType    string `gorm:"type:text;default:'main';index:idx_messages_room_type" json:"room_type"`

	// 关联
	Channel   *Channel   `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"-"`
	Sender    *Member    `gorm:"foreignKey:SenderID;constraint:OnDelete:SET NULL" json:"-"`
	ReplyTo   *Message   `gorm:"foreignKey:ReplyToID;constraint:OnDelete:SET NULL" json:"-"`
	Challenge *Challenge `gorm:"foreignKey:ChallengeID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (Message) TableName() string {
	return "messages"
}

// BeforeCreate GORM 钩子
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	if m.DeletedAt.IsZero() {
		m.DeletedAt = time.Now()
	}
	if m.EditedAt.IsZero() {
		// 初始化为创建时间，满足 NOT NULL 约束
		m.EditedAt = m.Timestamp
	}
	if m.RoomType == "" {
		m.RoomType = "main"
	}
	// 同步兼容字段
	m.IsDeleted = m.Deleted
	m.IsPinned = m.Pinned
	return nil
}

// AfterFind GORM 钩子 - 同步兼容字段
func (m *Message) AfterFind(tx *gorm.DB) error {
	m.IsDeleted = m.Deleted
	m.IsPinned = m.Pinned
	return nil
}

// BeforeUpdate GORM 钩子
func (m *Message) BeforeUpdate(tx *gorm.DB) error {
	// 同步兼容字段
	m.IsDeleted = m.Deleted
	m.IsPinned = m.Pinned
	if m.EditedAt.IsZero() {
		m.EditedAt = time.Now()
	}
	return nil
}

// MessageReaction 消息表情回应
type MessageReaction struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	MessageID string    `gorm:"type:text;not null;index:idx_reactions_message" json:"message_id"`
	UserID    string    `gorm:"type:text;not null" json:"user_id"`
	Emoji     string    `gorm:"type:text;not null" json:"emoji"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`

	// 关联
	Message *Message `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (MessageReaction) TableName() string {
	return "message_reactions"
}

// BeforeCreate GORM 钩子
func (r *MessageReaction) BeforeCreate(tx *gorm.DB) error {
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	return nil
}

// TypingStatus 正在输入状态
type TypingStatus struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID string    `gorm:"type:text;not null;index:idx_typing_channel" json:"channel_id"`
	UserID    string    `gorm:"type:text;not null" json:"user_id"`
	Timestamp time.Time `gorm:"not null" json:"timestamp"`

	// 关联
	Channel *Channel `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (TypingStatus) TableName() string {
	return "typing_status"
}

// IsExpired 检查是否过期（超过5秒）
func (t *TypingStatus) IsExpired() bool {
	return time.Since(t.Timestamp) > 5*time.Second
}
