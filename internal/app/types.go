package app

import (
	"crosswire/internal/models"
)

// ==================== 运行模式 ====================

// Mode 运行模式
type Mode string

const (
	ModeIdle   Mode = "idle"   // 空闲
	ModeServer Mode = "server" // 服务端
	ModeClient Mode = "client" // 客户端
)

// ==================== 服务端配置 ====================

// ServerConfig 服务端配置
type ServerConfig struct {
	ChannelName      string               `json:"channel_name"`      // 频道名称
	Password         string               `json:"password"`          // 频道密码
	TransportMode    models.TransportMode `json:"transport_mode"`    // 传输模式
	NetworkInterface string               `json:"network_interface"` // 网络接口（ARP模式必填）
	ListenAddress    string               `json:"listen_address"`    // 监听地址（HTTPS模式）
	Port             int                  `json:"port"`              // 监听端口（HTTPS模式）
	MaxMembers       int                  `json:"max_members"`       // 最大成员数
	MaxFileSize      int64                `json:"max_file_size"`     // 最大文件大小（字节）
	EnableChallenge  bool                 `json:"enable_challenge"`  // 启用题目功能
	Description      string               `json:"description"`       // 频道描述
}

// ServerStatus 服务端状态
type ServerStatus struct {
	Running       bool          `json:"running"`
	ChannelID     string        `json:"channel_id"`
	ChannelName   string        `json:"channel_name"`
	TransportMode string        `json:"transport_mode"`
	MemberCount   int           `json:"member_count"`
	StartTime     int64         `json:"start_time"` // Unix timestamp
	NetworkStats  *NetworkStats `json:"network_stats"`
}

// ==================== 客户端配置 ====================

// ClientConfig 客户端配置
type ClientConfig struct {
    ChannelID        string               `json:"channel_id"`        // 频道ID（HTTPS: 通过/info获取）
	Password         string               `json:"password"`          // 频道密码
	TransportMode    models.TransportMode `json:"transport_mode"`    // 传输模式
	NetworkInterface string               `json:"network_interface"` // 网络接口（ARP模式）
	ServerAddress    string               `json:"server_address"`    // 服务器地址（HTTPS模式）
	Port             int                  `json:"port"`              // 服务器端口（HTTPS模式）
	Nickname         string               `json:"nickname"`          // 用户昵称
	Avatar           string               `json:"avatar"`            // 用户头像URL
	AutoReconnect    bool                 `json:"auto_reconnect"`    // 自动重连
}

// ClientStatus 客户端状态
type ClientStatus struct {
	Running       bool   `json:"running"`
	Connected     bool   `json:"connected"`
	ChannelID     string `json:"channel_id"`
	ChannelName   string `json:"channel_name"`
	MemberID      string `json:"member_id"`
	TransportMode string `json:"transport_mode"`
	ConnectTime   int64  `json:"connect_time"` // Unix timestamp
}

// ==================== 消息相关 ====================

// MessageDTO 消息数据传输对象
type MessageDTO struct {
	ID         string                `json:"id"`
	ChannelID  string                `json:"channel_id"`
	SenderID   string                `json:"sender_id"`
	SenderName string                `json:"sender_name"`
	Type       models.MessageType    `json:"type"`
	Content    models.MessageContent `json:"content"`
	Timestamp  int64                 `json:"timestamp"`
	EditedAt   int64                 `json:"edited_at"`
	IsDeleted  bool                  `json:"is_deleted"`
	IsPinned   bool                  `json:"is_pinned"`
	ReplyToID  *string               `json:"reply_to_id"`
	Reactions  []MessageReaction     `json:"reactions"`
}

// MessageReaction 聚合结构
type MessageReaction struct {
	Emoji   string   `json:"emoji"`
	UserIDs []string `json:"user_ids"`
	Count   int      `json:"count"`
}

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	Content   string             `json:"content"`
	Type      models.MessageType `json:"type"`
	ChannelID *string            `json:"channel_id,omitempty"`
	ReplyToID *string            `json:"reply_to_id,omitempty"`
}

// SendCodeRequest 发送代码消息请求
type SendCodeRequest struct {
	Code     string  `json:"code"`
	Language string  `json:"language"`
	Filename *string `json:"filename,omitempty"`
}

// SearchMessagesRequest 搜索消息请求
type SearchMessagesRequest struct {
	Query     string              `json:"query"`
	Type      *models.MessageType `json:"type,omitempty"`
	SenderID  *string             `json:"sender_id,omitempty"`
	StartTime *int64              `json:"start_time,omitempty"` // Unix timestamp
	EndTime   *int64              `json:"end_time,omitempty"`   // Unix timestamp
	Limit     int                 `json:"limit"`
	Offset    int                 `json:"offset"`
}

// PinMessageRequest 置顶消息请求
type PinMessageRequest struct {
	MessageID string `json:"message_id"`
	Reason    string `json:"reason"`
}

// PinnedMessageDTO 置顶消息传输对象
type PinnedMessageDTO struct {
	ID             int    `json:"id"`
	ChannelID      string `json:"channel_id"`
	MessageID      string `json:"message_id"`
	PinnedBy       string `json:"pinned_by"`
	Reason         string `json:"reason,omitempty"`
	PinnedAt       int64  `json:"pinned_at"`
	DisplayOrder   int    `json:"display_order"`
	ContentText    string `json:"content_text"`
	SenderID       string `json:"sender_id"`
	SenderNickname string `json:"sender_nickname"`
}

// ==================== 文件相关 ====================

// FileDTO 文件数据传输对象
type FileDTO struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	Size          int64               `json:"size"`
	MimeType      string              `json:"mime_type"`
	UploaderID    string              `json:"uploader_id"`
	UploaderName  string              `json:"uploader_name"`
	UploadStatus  models.UploadStatus `json:"upload_status"`
	Progress      int                 `json:"progress"`
	UploadTime    int64               `json:"upload_time"` // Unix timestamp
	ThumbnailPath *string             `json:"thumbnail_path,omitempty"`
	LocalPath     *string             `json:"local_path,omitempty"`
}

// UploadFileRequest 上传文件请求
type UploadFileRequest struct {
	FilePath    string  `json:"file_path"`
	Description *string `json:"description,omitempty"`
}

// DownloadFileRequest 下载文件请求
type DownloadFileRequest struct {
	FileID   string `json:"file_id"`
	SavePath string `json:"save_path"`
}

// FileTransferProgress 文件传输进度
type FileTransferProgress struct {
	FileID      string  `json:"file_id"`
	FileName    string  `json:"file_name"`
	TotalSize   int64   `json:"total_size"`
	Transferred int64   `json:"transferred"`
	Progress    int     `json:"progress"`
	Speed       int64   `json:"speed"` // 字节/秒
	Status      string  `json:"status"`
	Error       *string `json:"error,omitempty"`
}

// ==================== 成员相关 ====================

// MemberDTO 成员数据传输对象
type MemberDTO struct {
	ID               string            `json:"id"`
	Nickname         string            `json:"nickname"`
	Avatar           string            `json:"avatar"`
	Role             models.MemberRole `json:"role"`
	Status           models.UserStatus `json:"status"`
	IsOnline         bool              `json:"is_online"`
	JoinTime         int64             `json:"join_time"`    // Unix timestamp
	LastSeenAt       int64             `json:"last_seen_at"` // Unix timestamp
	IsMuted          bool              `json:"is_muted"`
	IsBanned         bool              `json:"is_banned"`
	Email            string            `json:"email,omitempty"`         // 邮箱（从Metadata提取）
	Bio              string            `json:"bio,omitempty"`           // 个人简介（从Metadata提取）
	Skills           []string          `json:"skills,omitempty"`        // 技能标签（仅类别名，向后兼容）
	SkillDetails     []SkillDetail     `json:"skill_details,omitempty"` // 技能详情（包含等级与经验）
	SolvedChallenges int               `json:"solved_challenges"`       // 已解决题目数
	TotalPoints      int               `json:"total_points"`            // 总积分
	Rank             int               `json:"rank,omitempty"`          // 排名
	MessageCount     int               `json:"message_count"`           // 消息数
	FilesShared      int               `json:"files_shared"`            // 共享文件数
	OnlineTime       int64             `json:"online_time"`             // 在线时长（秒）
}

// SkillDetail 技能详情（用于DTO与Profile传输）
type SkillDetail struct {
	Category   string `json:"category"`   // Web/Pwn/Reverse/Crypto/Misc
	Level      int    `json:"level"`      // 1-5
	Experience int    `json:"experience"` // 题目数量/经验
}

// UpdateMemberRequest 更新成员请求
type UpdateMemberRequest struct {
	MemberID string             `json:"member_id"`
	Role     *models.MemberRole `json:"role,omitempty"`
	IsMuted  *bool              `json:"is_muted,omitempty"`
}

// KickMemberRequest 踢出成员请求
type KickMemberRequest struct {
	MemberID string  `json:"member_id"`
	Reason   *string `json:"reason,omitempty"`
}

// BanMemberRequest 封禁成员请求
type BanMemberRequest struct {
	MemberID string  `json:"member_id"`
	Reason   *string `json:"reason,omitempty"`
	Duration *int64  `json:"duration,omitempty"` // 秒，nil表示永久
}

// ==================== 题目相关 ====================

// ChallengeDTO 题目数据传输对象
type ChallengeDTO struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	Difficulty   string   `json:"difficulty"`
	Points       int      `json:"points"`
	Flag         string   `json:"flag"`
	IsSolved     bool     `json:"is_solved"`
	SolvedBy     []string `json:"solved_by"`
	AssignedTo   []string `json:"assigned_to"`
	SubChannelID string   `json:"sub_channel_id,omitempty"` // 题目专属子频道ID
	CreatedAt    int64    `json:"created_at"`               // Unix timestamp
	UpdatedAt    int64    `json:"updated_at"`               // Unix timestamp
}

// SubChannelDTO 子频道数据传输对象
type SubChannelDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ParentChannelID string `json:"parent_channel_id"`
	MessageCount    int64  `json:"message_count"`
	OnlineCount     int    `json:"online_count"`
	CreatedAt       int64  `json:"created_at"` // Unix timestamp
}

// CreateChallengeRequest 创建题目请求
type CreateChallengeRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Difficulty  string `json:"difficulty"`
	Points      int    `json:"points"`
	Flag        string `json:"flag"`
}

// UpdateChallengeRequest 更新题目请求
type UpdateChallengeRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty"`
	Difficulty  *string `json:"difficulty,omitempty"`
	Points      *int    `json:"points,omitempty"`
	Flag        *string `json:"flag,omitempty"`
}

// SubmitFlagRequest 提交flag请求
type SubmitFlagRequest struct {
	ChallengeID string `json:"challenge_id"`
	Flag        string `json:"flag"`
}

// SubmitFlagResponse 提交flag响应（协作平台：不验证正确性）
type SubmitFlagResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Points  int    `json:"points,omitempty"` // 可选：用于贡献度统计
}

// UpdateProgressRequest 更新进度请求
type UpdateProgressRequest struct {
	ChallengeID string `json:"challenge_id"`
	Progress    int    `json:"progress"`
	Summary     string `json:"summary"`
}

// 提示相关类型已移除（协作平台不支持提示）

// LeaderboardEntry 排行榜条目
type LeaderboardEntry struct {
	MemberID    string `json:"member_id"`
	MemberName  string `json:"member_name"`
	TotalPoints int    `json:"total_points"`
	SolvedCount int    `json:"solved_count"`
	Rank        int    `json:"rank"`
}

// ==================== 系统相关 ====================

// NetworkInterface 网络接口
type NetworkInterface struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	MACAddress  string   `json:"mac_address"`
	IPAddresses []string `json:"ip_addresses"`
	IsUp        bool     `json:"is_up"`
	IsLoopback  bool     `json:"is_loopback"`
}

// NetworkStats 网络统计
type NetworkStats struct {
	BytesSent       int64   `json:"bytes_sent"`
	BytesReceived   int64   `json:"bytes_received"`
	PacketsSent     int64   `json:"packets_sent"`
	PacketsReceived int64   `json:"packets_received"`
	ErrorCount      int64   `json:"error_count"`
	AvgLatency      float64 `json:"avg_latency"` // 毫秒
}

// ConnectionTestResult 连接测试结果
type ConnectionTestResult struct {
	Success bool    `json:"success"`
	Latency float64 `json:"latency"` // 毫秒
	Message string  `json:"message"`
}

// UserProfile 用户配置
type UserProfile struct {
	Nickname      string               `json:"nickname"`
	Avatar        string               `json:"avatar"`
	Email         string               `json:"email"`
	Bio           string               `json:"bio"`
	Skills        []string             `json:"skills"`
	SkillDetails  []SkillDetail        `json:"skill_details,omitempty"`
	Status        models.UserStatus    `json:"status"`
	CustomStatus  string               `json:"custom_status"`
	Theme         string               `json:"theme"`
	Language      string               `json:"language"`
	Notifications NotificationSettings `json:"notifications"`
}

// NotificationSettings 通知设置
type NotificationSettings struct {
	Enabled     bool `json:"enabled"`
	Sound       bool `json:"sound"`
	Desktop     bool `json:"desktop"`
	MentionOnly bool `json:"mention_only"`
}

// RecentChannel 最近频道
type RecentChannel struct {
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	LastJoined  int64  `json:"last_joined"` // Unix timestamp
	Mode        Mode   `json:"mode"`        // server or client
}

// ExportOptions 导出选项
type ExportOptions struct {
	IncludeMessages   bool `json:"include_messages"`
	IncludeFiles      bool `json:"include_files"`
	IncludeChallenges bool `json:"include_challenges"`
	IncludeMembers    bool `json:"include_members"`
}

// ==================== 事件相关 ====================

// AppEvent 应用事件（发送到前端）
type AppEvent struct {
	Type      string      `json:"type"`
	Timestamp int64       `json:"timestamp"` // Unix timestamp
	Data      interface{} `json:"data"`
}

// EventType 事件类型常量
const (
	// 连接事件
	EventConnected    = "connected"
	EventDisconnected = "disconnected"
	EventReconnecting = "reconnecting"

	// 消息事件
	EventMessageReceived = "message:received"
	EventMessageSent     = "message:sent"
	EventMessageUpdated  = "message:updated"
	EventMessageDeleted  = "message:deleted"

	// 成员事件
	EventMemberJoined  = "member:joined"
	EventMemberLeft    = "member:left"
	EventMemberUpdated = "member:updated"
	EventMemberKicked  = "member:kicked"
	EventMemberBanned  = "member:banned"

	// 文件事件
	EventFileUploadStarted     = "file:upload:started"
	EventFileUploadProgress    = "file:upload:progress"
	EventFileUploadCompleted   = "file:upload:completed"
	EventFileUploadFailed      = "file:upload:failed"
	EventFileDownloadStarted   = "file:download:started"
	EventFileDownloadProgress  = "file:download:progress"
	EventFileDownloadCompleted = "file:download:completed"
	EventFileDownloadFailed    = "file:download:failed"
	EventFileDeleted           = "file:deleted"

	// 题目事件
	EventChallengeCreated  = "challenge:created"
	EventChallengeUpdated  = "challenge:updated"
	EventChallengeSolved   = "challenge:solved"
	EventChallengeAssigned = "challenge:assigned"
	EventChallengeProgress = "challenge:progress"

	// 系统事件
	EventError   = "error"
	EventWarning = "warning"
	EventInfo    = "info"
)

// ==================== 响应包装 ====================

// Response 通用响应
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ==================== 辅助函数 ====================

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) Response {
	return Response{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code, message, details string) Response {
	return Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}
