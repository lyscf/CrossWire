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
	ReplyToID      string         `gorm:"type:text;index:idx_messages_reply_to" json:"reply_to_id,omitempty"`
	ThreadID       string         `gorm:"type:text;index:idx_messages_thread" json:"thread_id,omitempty"`
	Mentions       StringArray    `gorm:"type:text" json:"mentions,omitempty"`
	Tags           StringArray    `gorm:"type:text" json:"tags,omitempty"`
	Pinned         bool           `gorm:"type:integer;default:0;index:idx_messages_pinned" json:"pinned"`
	Deleted        bool           `gorm:"type:integer;default:0;index:idx_messages_deleted" json:"deleted"`
	DeletedBy      string         `gorm:"type:text" json:"deleted_by,omitempty"`
	DeletedAt      *time.Time     `gorm:"type:integer" json:"deleted_at,omitempty"`
	Timestamp      time.Time      `gorm:"type:integer;not null;index:idx_messages_channel_time" json:"timestamp"`
	EditedAt       *time.Time     `gorm:"type:integer" json:"edited_at,omitempty"`
	Encrypted      bool           `gorm:"type:integer;default:1" json:"encrypted"`
	KeyVersion     int            `gorm:"type:integer;default:1" json:"key_version"`
	Metadata       JSONField      `gorm:"type:text" json:"metadata,omitempty"`

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
	if m.RoomType == "" {
		m.RoomType = "main"
	}
	return nil
}

// MessageReaction 消息表情回应
type MessageReaction struct {
	ID        int       `gorm:"primaryKey;autoIncrement" json:"id"`
	MessageID string    `gorm:"type:text;not null;index:idx_reactions_message" json:"message_id"`
	UserID    string    `gorm:"type:text;not null" json:"user_id"`
	Emoji     string    `gorm:"type:text;not null" json:"emoji"`
	CreatedAt time.Time `gorm:"type:integer;not null" json:"created_at"`

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
	Timestamp time.Time `gorm:"type:integer;not null" json:"timestamp"`

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
