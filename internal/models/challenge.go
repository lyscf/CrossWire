package models

import (
	"time"

	"gorm.io/gorm"
)

// Challenge CTF 题目
type Challenge struct {
	ID           string      `gorm:"primaryKey;type:text" json:"id"`
	ChannelID    string      `gorm:"type:text;not null;index:idx_challenges_channel" json:"channel_id"`
	SubChannelID string      `gorm:"type:text" json:"sub_channel_id,omitempty"` // 题目专属子频道ID
	Title        string      `gorm:"type:text;not null" json:"title"`
	Category     string      `gorm:"type:text;not null;index:idx_challenges_category" json:"category"`
	Difficulty   string      `gorm:"type:text;not null" json:"difficulty"`
	Points       int         `gorm:"type:integer;not null" json:"points"`
	Description  string      `gorm:"type:text;not null" json:"description"`
	FlagFormat   string      `gorm:"type:text" json:"flag_format,omitempty"`
	FlagHash     string      `gorm:"type:text" json:"-"` // 不返回给前端
	URL          string      `gorm:"type:text" json:"url,omitempty"`
	Attachments  StringArray `gorm:"type:text" json:"attachments,omitempty"`
	Tags         StringArray `gorm:"type:text" json:"tags,omitempty"`
	Status       string      `gorm:"type:text;not null;default:'open';index:idx_challenges_status" json:"status"`
	SolvedBy     StringArray `gorm:"type:text" json:"solved_by,omitempty"`
	SolvedAt     *time.Time  `gorm:"type:integer" json:"solved_at,omitempty"`
	AssignedTo   StringArray `gorm:"type:text" json:"assigned_to,omitempty"` // APP层使用：已分配给的成员ID列表
	CreatedBy    string      `gorm:"type:text;not null" json:"created_by"`
	CreatedAt    time.Time   `gorm:"type:integer;not null;index:idx_challenges_created_at" json:"created_at"`
	UpdatedAt    time.Time   `gorm:"type:integer;not null" json:"updated_at"`
	Metadata     JSONField   `gorm:"type:text" json:"metadata,omitempty"`

	// 关联
	Channel     *Channel               `gorm:"foreignKey:ChannelID;constraint:OnDelete:CASCADE" json:"-"`
	Creator     *Member                `gorm:"foreignKey:CreatedBy;constraint:OnDelete:SET NULL" json:"-"`
	Assignments []*ChallengeAssignment `gorm:"foreignKey:ChallengeID" json:"assignments,omitempty"`
	Progress    []*ChallengeProgress   `gorm:"foreignKey:ChallengeID" json:"progress,omitempty"`
	Submissions []*ChallengeSubmission `gorm:"foreignKey:ChallengeID" json:"submissions,omitempty"`
	Hints       []*ChallengeHint       `gorm:"foreignKey:ChallengeID" json:"hints,omitempty"`
}

// TableName 指定表名
func (Challenge) TableName() string {
	return "challenges"
}

// BeforeCreate GORM 钩子
func (c *Challenge) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM 钩子
func (c *Challenge) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// AfterFind GORM 钩子 - 同步AssignedTo字段
func (c *Challenge) AfterFind(tx *gorm.DB) error {
	// 从Assignments构建AssignedTo列表
	if len(c.Assignments) > 0 && len(c.AssignedTo) == 0 {
		assignedTo := make([]string, 0, len(c.Assignments))
		for _, assignment := range c.Assignments {
			assignedTo = append(assignedTo, assignment.MemberID)
		}
		c.AssignedTo = assignedTo
	}
	return nil
}

// ChallengeAssignment 题目分配
type ChallengeAssignment struct {
	ChallengeID string    `gorm:"type:text;not null;primaryKey;index:idx_assignments_challenge" json:"challenge_id"`
	MemberID    string    `gorm:"type:text;not null;primaryKey;index:idx_assignments_member" json:"member_id"`
	AssignedBy  string    `gorm:"type:text;not null" json:"assigned_by"`
	AssignedAt  time.Time `gorm:"type:integer;not null" json:"assigned_at"`
	Role        string    `gorm:"type:text;not null;default:'member'" json:"role"` // lead/member
	Status      string    `gorm:"type:text;not null;default:'assigned';index:idx_assignments_status" json:"status"`
	Notes       string    `gorm:"type:text" json:"notes,omitempty"`

	// 关联
	Challenge *Challenge `gorm:"foreignKey:ChallengeID;constraint:OnDelete:CASCADE" json:"-"`
	Member    *Member    `gorm:"foreignKey:MemberID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (ChallengeAssignment) TableName() string {
	return "challenge_assignments"
}

// BeforeCreate GORM 钩子
func (c *ChallengeAssignment) BeforeCreate(tx *gorm.DB) error {
	if c.AssignedAt.IsZero() {
		c.AssignedAt = time.Now()
	}
	return nil
}

// ChallengeProgress 题目进度
type ChallengeProgress struct {
	ID          int       `gorm:"primaryKey;autoIncrement" json:"id"`
	ChallengeID string    `gorm:"type:text;not null;index:idx_progress_challenge" json:"challenge_id"`
	MemberID    string    `gorm:"type:text;not null;index:idx_progress_member" json:"member_id"`
	Progress    int       `gorm:"type:integer;not null;default:0" json:"progress"` // 0-100
	Status      string    `gorm:"type:text;not null;default:'not_started'" json:"status"`
	Summary     string    `gorm:"type:text" json:"summary,omitempty"`
	Findings    string    `gorm:"type:text" json:"findings,omitempty"`
	Blockers    string    `gorm:"type:text" json:"blockers,omitempty"`
	UpdatedAt   time.Time `gorm:"type:integer;not null;index:idx_progress_updated" json:"updated_at"`
	Metadata    JSONField `gorm:"type:text" json:"metadata,omitempty"`

	// 关联
	Challenge *Challenge `gorm:"foreignKey:ChallengeID;constraint:OnDelete:CASCADE" json:"-"`
	Member    *Member    `gorm:"foreignKey:MemberID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (ChallengeProgress) TableName() string {
	return "challenge_progress"
}

// BeforeCreate GORM 钩子
func (c *ChallengeProgress) BeforeCreate(tx *gorm.DB) error {
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now()
	}
	return nil
}

// BeforeUpdate GORM 钩子
func (c *ChallengeProgress) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// ChallengeSubmission Flag 提交记录
type ChallengeSubmission struct {
	ID           string    `gorm:"primaryKey;type:text" json:"id"`
	ChallengeID  string    `gorm:"type:text;not null;index:idx_submissions_challenge" json:"challenge_id"`
	MemberID     string    `gorm:"type:text;not null;index:idx_submissions_member" json:"member_id"`
	Flag         string    `gorm:"type:text;not null" json:"-"` // 加密存储
	IsCorrect    bool      `gorm:"type:integer;not null;index:idx_submissions_correct" json:"is_correct"`
	SubmittedAt  time.Time `gorm:"type:integer;not null;index:idx_submissions_time" json:"submitted_at"`
	IPAddress    string    `gorm:"type:text" json:"ip_address,omitempty"`
	ResponseTime int       `gorm:"type:integer" json:"response_time,omitempty"` // 毫秒
	Metadata     JSONField `gorm:"type:text" json:"metadata,omitempty"`

	// 关联
	Challenge *Challenge `gorm:"foreignKey:ChallengeID;constraint:OnDelete:CASCADE" json:"-"`
	Member    *Member    `gorm:"foreignKey:MemberID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (ChallengeSubmission) TableName() string {
	return "challenge_submissions"
}

// BeforeCreate GORM 钩子
func (c *ChallengeSubmission) BeforeCreate(tx *gorm.DB) error {
	if c.SubmittedAt.IsZero() {
		c.SubmittedAt = time.Now()
	}
	return nil
}

// ChallengeHint 题目提示
type ChallengeHint struct {
	ID          string      `gorm:"primaryKey;type:text" json:"id"`
	ChallengeID string      `gorm:"type:text;not null;index:idx_hints_challenge" json:"challenge_id"`
	OrderNum    int         `gorm:"type:integer;not null;uniqueIndex:idx_challenge_order" json:"order_num"`
	Content     string      `gorm:"type:text;not null" json:"content"`
	Cost        int         `gorm:"type:integer;default:0" json:"cost"`
	UnlockedBy  StringArray `gorm:"type:text" json:"unlocked_by,omitempty"`
	CreatedBy   string      `gorm:"type:text;not null" json:"created_by"`
	CreatedAt   time.Time   `gorm:"type:integer;not null" json:"created_at"`

	// 关联
	Challenge *Challenge `gorm:"foreignKey:ChallengeID;constraint:OnDelete:CASCADE" json:"-"`
	Creator   *Member    `gorm:"foreignKey:CreatedBy;constraint:OnDelete:SET NULL" json:"-"`
}

// TableName 指定表名
func (ChallengeHint) TableName() string {
	return "challenge_hints"
}

// BeforeCreate GORM 钩子
func (c *ChallengeHint) BeforeCreate(tx *gorm.DB) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	return nil
}
