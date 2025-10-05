package models

import (
	"time"

	"gorm.io/gorm"
)

// File 文件模型
type File struct {
	ID             string       `gorm:"primaryKey;type:text" json:"id"`
	MessageID      string       `gorm:"type:text;not null;index:idx_files_message" json:"message_id"`
	ChannelID      string       `gorm:"type:text;not null;index:idx_files_channel" json:"channel_id"`
	SenderID       string       `gorm:"type:text;not null;index:idx_files_sender" json:"sender_id"`
	Filename       string       `gorm:"type:text;not null" json:"filename"`
	OriginalName   string       `gorm:"type:text;not null" json:"original_name"`
	Size           int64        `gorm:"type:integer;not null" json:"size"`
	MimeType       string       `gorm:"type:text;not null" json:"mime_type"`
	StorageType    StorageType  `gorm:"type:text;not null" json:"storage_type"`
	StoragePath    string       `gorm:"type:text" json:"storage_path,omitempty"`
	Data           []byte       `gorm:"type:blob" json:"-"`
	SHA256         string       `gorm:"type:text;not null" json:"sha256"`
	Checksum       string       `gorm:"type:text;not null" json:"checksum"`
	ChunkSize      int          `gorm:"type:integer;default:8192" json:"chunk_size"`
	TotalChunks    int          `gorm:"type:integer;not null" json:"total_chunks"`
	UploadedChunks int          `gorm:"type:integer;default:0" json:"uploaded_chunks"`
	UploadStatus   UploadStatus `gorm:"type:text;default:'pending'" json:"upload_status"`
	Thumbnail      []byte       `gorm:"type:blob" json:"thumbnail,omitempty"`
	PreviewText    string       `gorm:"type:text" json:"preview_text,omitempty"`
	UploadedAt     time.Time    `gorm:"type:integer;not null;index:idx_files_uploaded_at" json:"uploaded_at"`
	ExpiresAt      *time.Time   `gorm:"type:integer;index:idx_files_expires" json:"expires_at,omitempty"`
	Encrypted      bool         `gorm:"type:integer;default:1" json:"encrypted"`
	EncryptionKey  []byte       `gorm:"type:blob" json:"-"`
	Metadata       JSONField    `gorm:"type:text" json:"metadata,omitempty"`

	// 关联
	Message *Message `gorm:"foreignKey:MessageID;constraint:OnDelete:CASCADE" json:"-"`
	Channel *Channel `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"-"`
	Sender  *Member  `gorm:"foreignKey:SenderID;constraint:OnDelete:SET NULL" json:"-"`
}

// TableName 指定表名
func (File) TableName() string {
	return "files"
}

// BeforeCreate GORM 钩子
func (f *File) BeforeCreate(tx *gorm.DB) error {
	if f.UploadedAt.IsZero() {
		f.UploadedAt = time.Now()
	}
	return nil
}

// IsExpired 检查文件是否过期
func (f *File) IsExpired() bool {
	if f.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*f.ExpiresAt)
}

// FileChunk 文件分块状态
type FileChunk struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	FileID      string    `gorm:"type:text;not null;index:idx_chunks_file" json:"file_id"`
	ChunkIndex  int       `gorm:"type:integer;not null" json:"chunk_index"`
	Size        int       `gorm:"type:integer;not null" json:"size"`
	Checksum    string    `gorm:"type:text;not null" json:"checksum"`
	Uploaded    bool      `gorm:"type:integer;default:0" json:"uploaded"`
	UploadedAt  time.Time `gorm:"type:integer" json:"uploaded_at,omitempty"`
	RetryCount  int       `gorm:"type:integer;default:0" json:"retry_count"`
	LastAttempt time.Time `gorm:"type:integer" json:"last_attempt,omitempty"`

	// 关联
	File *File `gorm:"foreignKey:FileID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (FileChunk) TableName() string {
	return "file_chunks"
}
