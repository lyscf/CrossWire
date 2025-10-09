package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// TransportMode 传输模式
type TransportMode string

const (
	TransportARP   TransportMode = "arp"
	TransportHTTPS TransportMode = "https"
	TransportMDNS  TransportMode = "mdns"
	TransportAuto  TransportMode = "auto"
)

// Role 成员角色
type Role string

// MemberRole 成员角色别名（用于APP层）
type MemberRole = Role

const (
	RoleOwner     Role = "owner"
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
	RoleMember    Role = "member"
	RoleReadOnly  Role = "readonly"
)

// UserStatus 用户在线状态
type UserStatus string

const (
	StatusOnline  UserStatus = "online"
	StatusBusy    UserStatus = "busy"
	StatusAway    UserStatus = "away"
	StatusOffline UserStatus = "offline"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeCode     MessageType = "code"
	MessageTypeFile     MessageType = "file"
	MessageTypeSystem   MessageType = "system"
	MessageTypeControl  MessageType = "control"
	MessageTypeReaction MessageType = "reaction"
)

// StorageType 文件存储类型
type StorageType string

const (
	StorageInline    StorageType = "inline"    // 数据库中
	StorageFile      StorageType = "file"      // 文件系统
	StorageReference StorageType = "reference" // 外部引用
)

// UploadStatus 文件上传状态
type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusUploading UploadStatus = "uploading"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusFailed    UploadStatus = "failed"
	UploadStatusPaused    UploadStatus = "paused"
)

// JSONField 用于 GORM 的 JSON 字段类型
type JSONField map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (j *JSONField) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONField)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value 实现 driver.Valuer 接口
func (j JSONField) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// StringArray 用于 GORM 的字符串数组类型
type StringArray []string

// Scan 实现 sql.Scanner 接口
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// Value 实现 driver.Valuer 接口
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// SkillTag CTF 技能标签
type SkillTag struct {
	Category   string    `json:"category"`   // Web/Pwn/Reverse/Crypto/Misc
	Level      int       `json:"level"`      // 1-5
	Experience int       `json:"experience"` // 题目数量
	LastUsed   time.Time `json:"last_used"`
}

// SkillTags 技能标签数组
type SkillTags []SkillTag

// Scan 实现 sql.Scanner 接口
func (s *SkillTags) Scan(value interface{}) error {
	if value == nil {
		*s = []SkillTag{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// Value 实现 driver.Valuer 接口
func (s SkillTags) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Expertise 擅长领域
type Expertise struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
	Notes       string   `json:"notes"`
}

// ExpertiseArray 擅长领域数组
type ExpertiseArray []Expertise

// Scan 实现 sql.Scanner 接口
func (e *ExpertiseArray) Scan(value interface{}) error {
	if value == nil {
		*e = []Expertise{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, e)
}

// Value 实现 driver.Valuer 接口
func (e ExpertiseArray) Value() (driver.Value, error) {
	if e == nil {
		return nil, nil
	}
	return json.Marshal(e)
}

// CurrentTask 当前任务
type CurrentTask struct {
	Challenge string    `json:"challenge"`
	StartTime time.Time `json:"start_time"`
	Progress  int       `json:"progress"` // 0-100
	Notes     string    `json:"notes"`
	Teammates []string  `json:"teammates"`
}

// Scan 实现 sql.Scanner 接口
func (c *CurrentTask) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, c)
}

// Value 实现 driver.Valuer 接口
func (c CurrentTask) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// TextContent 文本消息内容
type TextContent struct {
	Text     string   `json:"text"`
	Format   string   `json:"format"` // plain/markdown/html
	Mentions []string `json:"mentions,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	ReplyTo  string   `json:"reply_to,omitempty"`
}

// CodeContent 代码消息内容
type CodeContent struct {
	Language    string `json:"language"`
	Code        string `json:"code"`
	Filename    string `json:"filename,omitempty"`
	Description string `json:"description,omitempty"`
	Highlighted bool   `json:"highlighted"`
}

// FileContent 文件消息内容
type FileContent struct {
	FileID    string     `json:"file_id"`
	Filename  string     `json:"filename"`
	Size      int64      `json:"size"`
	MimeType  string     `json:"mime_type"`
	SHA256    string     `json:"sha256"`
	Thumbnail string     `json:"thumbnail,omitempty"` // Base64
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// SystemContent 系统消息内容
type SystemContent struct {
	Event    string                 `json:"event"`
	ActorID  string                 `json:"actor_id"`
	TargetID string                 `json:"target_id,omitempty"`
	Reason   string                 `json:"reason,omitempty"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

// MessageContent 消息内容（存储为 JSON）
// 使用时需要根据 MessageType 解析为对应的具体类型
type MessageContent JSONField

// Scan implements sql.Scanner for MessageContent by delegating to JSONField
func (m *MessageContent) Scan(value interface{}) error {
	var jf JSONField
	if err := (&jf).Scan(value); err != nil {
		return err
	}
	*m = MessageContent(jf)
	return nil
}

// Value implements driver.Valuer for MessageContent by delegating to JSONField
func (m MessageContent) Value() (driver.Value, error) {
	jf := JSONField(m)
	return jf.Value()
}
