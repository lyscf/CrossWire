package events

import (
	"time"
	
	"crosswire/internal/models"
)

// ===== 事件数据结构定义 =====

// MessageEvent 消息事件数据
type MessageEvent struct {
	Message  *models.Message
	ChannelID string
	SenderID  string
}

// MemberEvent 成员事件数据
type MemberEvent struct {
	Member    *models.Member
	ChannelID string
	Action    string // "joined", "left", "kicked", "muted", "unmuted"
	Reason    string // 可选的原因说明
}

// FileEvent 文件事件数据
type FileEvent struct {
	File      *models.File
	ChannelID string
	UploaderID string
	Progress  int // 0-100
}

// StatusEvent 状态变化事件数据
type StatusEvent struct {
	MemberID  string
	ChannelID string
	OldStatus models.UserStatus
	NewStatus models.UserStatus
}

// TypingEvent 输入状态事件数据
type TypingEvent struct {
	MemberID  string
	ChannelID string
	IsTyping  bool
}

// ChannelEvent 频道事件数据
type ChannelEvent struct {
	Channel *models.Channel
	Action  string // "created", "joined", "left", "updated"
	UserID  string
}

// SystemEvent 系统事件数据
type SystemEvent struct {
	Type    string      // "error", "connected", "disconnect", "reconnect"
	Message string
	Code    int
	Data    interface{}
}

// ChallengeEvent CTF挑战事件数据
type ChallengeEvent struct {
	Challenge *models.Challenge
	Action    string // "created", "assigned", "submitted", "solved", "hint"
	UserID    string
	ChannelID string
	ExtraData interface{}
}

// SubmissionEvent Flag提交事件数据
type SubmissionEvent struct {
	Submission  *models.ChallengeSubmission
	ChallengeID string
	UserID      string
	IsCorrect   bool
	Message     string
}

// ===== 辅助函数 =====

// NewMessageReceivedEvent 创建消息接收事件
func NewMessageReceivedEvent(msg *models.Message, channelID string) *Event {
	return &Event{
		Type: EventMessageReceived,
		Data: &MessageEvent{
			Message:   msg,
			ChannelID: channelID,
			SenderID:  msg.SenderID,
		},
		Timestamp: time.Now(),
		Source:    "message_handler",
	}
}

// NewMessageSentEvent 创建消息发送事件
func NewMessageSentEvent(msg *models.Message, channelID string) *Event {
	return &Event{
		Type: EventMessageSent,
		Data: &MessageEvent{
			Message:   msg,
			ChannelID: channelID,
			SenderID:  msg.SenderID,
		},
		Timestamp: time.Now(),
		Source:    "message_sender",
	}
}

// NewMemberJoinedEvent 创建成员加入事件
func NewMemberJoinedEvent(member *models.Member, channelID string) *Event {
	return &Event{
		Type: EventMemberJoined,
		Data: &MemberEvent{
			Member:    member,
			ChannelID: channelID,
			Action:    "joined",
		},
		Timestamp: time.Now(),
		Source:    "member_manager",
	}
}

// NewMemberLeftEvent 创建成员离开事件
func NewMemberLeftEvent(member *models.Member, channelID, reason string) *Event {
	return &Event{
		Type: EventMemberLeft,
		Data: &MemberEvent{
			Member:    member,
			ChannelID: channelID,
			Action:    "left",
			Reason:    reason,
		},
		Timestamp: time.Now(),
		Source:    "member_manager",
	}
}

// NewFileUploadedEvent 创建文件上传事件
func NewFileUploadedEvent(file *models.File, channelID, uploaderID string) *Event {
	return &Event{
		Type: EventFileUploaded,
		Data: &FileEvent{
			File:       file,
			ChannelID:  channelID,
			UploaderID: uploaderID,
			Progress:   100,
		},
		Timestamp: time.Now(),
		Source:    "file_manager",
	}
}

// NewFileProgressEvent 创建文件进度事件
func NewFileProgressEvent(file *models.File, channelID string, progress int) *Event {
	return &Event{
		Type: EventFileProgress,
		Data: &FileEvent{
			File:      file,
			ChannelID: channelID,
			Progress:  progress,
		},
		Timestamp: time.Now(),
		Source:    "file_manager",
	}
}

// NewStatusChangedEvent 创建状态变化事件
func NewStatusChangedEvent(memberID, channelID string, oldStatus, newStatus models.UserStatus) *Event {
	return &Event{
		Type: EventStatusChanged,
		Data: &StatusEvent{
			MemberID:  memberID,
			ChannelID: channelID,
			OldStatus: oldStatus,
			NewStatus: newStatus,
		},
		Timestamp: time.Now(),
		Source:    "member_manager",
	}
}

// NewTypingEvent 创建输入状态事件
func NewTypingEvent(memberID, channelID string, isTyping bool) *Event {
	eventType := EventTypingStart
	if !isTyping {
		eventType = EventTypingStop
	}
	
	return &Event{
		Type: eventType,
		Data: &TypingEvent{
			MemberID:  memberID,
			ChannelID: channelID,
			IsTyping:  isTyping,
		},
		Timestamp: time.Now(),
		Source:    "typing_manager",
	}
}

// NewChannelEvent 创建频道事件
func NewChannelEvent(eventType EventType, channel *models.Channel, userID, action string) *Event {
	return &Event{
		Type: eventType,
		Data: &ChannelEvent{
			Channel: channel,
			Action:  action,
			UserID:  userID,
		},
		Timestamp: time.Now(),
		Source:    "channel_manager",
	}
}

// NewSystemEvent 创建系统事件
func NewSystemEvent(eventType EventType, message string, code int, data interface{}) *Event {
	return &Event{
		Type: eventType,
		Data: &SystemEvent{
			Type:    string(eventType),
			Message: message,
			Code:    code,
			Data:    data,
		},
		Timestamp: time.Now(),
		Source:    "system",
	}
}

// NewChallengeEvent 创建挑战事件
func NewChallengeEvent(eventType EventType, challenge *models.Challenge, userID, channelID, action string, extra interface{}) *Event {
	return &Event{
		Type: eventType,
		Data: &ChallengeEvent{
			Challenge: challenge,
			Action:    action,
			UserID:    userID,
			ChannelID: channelID,
			ExtraData: extra,
		},
		Timestamp: time.Now(),
		Source:    "challenge_manager",
	}
}

// NewSubmissionEvent 创建提交事件
func NewSubmissionEvent(submission *models.ChallengeSubmission, isCorrect bool, message string) *Event {
	return &Event{
		Type: EventChallengeSubmitted,
		Data: &SubmissionEvent{
			Submission:  submission,
			ChallengeID: submission.ChallengeID,
			UserID:      submission.MemberID,
			IsCorrect:   isCorrect,
			Message:     message,
		},
		Timestamp: time.Now(),
		Source:    "challenge_manager",
	}
}

