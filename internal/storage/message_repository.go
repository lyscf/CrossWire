package storage

import (
	"time"

	"crosswire/internal/models"
)

// MessageRepository 消息数据仓库
type MessageRepository struct {
	db *Database
}

// NewMessageRepository 创建消息仓库
func NewMessageRepository(db *Database) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create 创建消息
func (r *MessageRepository) Create(message *models.Message) error {
	if message == nil {
		return nil
	}
	// 当没有关联题目时，避免写入空字符串导致外键约束失败
	if message.ChallengeID == "" {
		return r.db.GetChannelDB().Omit("challenge_id").Create(message).Error
	}
	return r.db.GetChannelDB().Create(message).Error
}

// GetByID 根据ID获取消息
func (r *MessageRepository) GetByID(messageID string) (*models.Message, error) {
	var message models.Message
	err := r.db.GetChannelDB().Where("id = ?", messageID).First(&message).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

// GetByChannelID 获取频道消息（分页）
func (r *MessageRepository) GetByChannelID(channelID string, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	query := r.db.GetChannelDB().Where("channel_id = ? AND deleted = ?", channelID, false).
		// 为了分页从最新开始，按编辑时间倒序返回
		Order("edited_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetRecentMessages 获取最近的消息（使用rowid优化）
func (r *MessageRepository) GetRecentMessages(channelID string, limit int, beforeRowID *int64) ([]*models.Message, error) {
	query := r.db.GetChannelDB().Where("channel_id = ? AND deleted = ?", channelID, false)

	if beforeRowID != nil {
		query = query.Where("rowid < ?", *beforeRowID)
	}

	var messages []*models.Message
	// 最近消息同样按编辑时间倒序
	err := query.Order("edited_at DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// GetBySenderID 获取指定发送者的消息
func (r *MessageRepository) GetBySenderID(senderID string, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	err := r.db.GetChannelDB().Where("sender_id = ? AND deleted = ?", senderID, false).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// GetByThreadID 获取话题消息
func (r *MessageRepository) GetByThreadID(threadID string) ([]*models.Message, error) {
	var messages []*models.Message
	err := r.db.GetChannelDB().Where("thread_id = ? AND deleted = ?", threadID, false).
		Order("timestamp ASC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// GetChallengeMessages 获取题目聊天室消息
func (r *MessageRepository) GetChallengeMessages(challengeID string, limit int) ([]*models.Message, error) {
	var messages []*models.Message
	err := r.db.GetChannelDB().Where("challenge_id = ? AND room_type = ? AND deleted = ?", challengeID, "challenge", false).
		Order("timestamp DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// Update 更新消息
func (r *MessageRepository) Update(message *models.Message) error {
	return r.db.GetChannelDB().Save(message).Error
}

// Delete 删除消息（软删除）
func (r *MessageRepository) Delete(messageID, deletedBy string) error {
	now := time.Now()
	return r.db.GetChannelDB().Model(&models.Message{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"deleted":    true,
			"deleted_by": deletedBy,
			"deleted_at": now,
		}).Error
}

// HardDelete 硬删除消息
func (r *MessageRepository) HardDelete(messageID string) error {
	return r.db.GetChannelDB().Where("id = ?", messageID).Delete(&models.Message{}).Error
}

// SetPinned 设置置顶状态
func (r *MessageRepository) SetPinned(messageID string, pinned bool) error {
	return r.db.GetChannelDB().Model(&models.Message{}).
		Where("id = ?", messageID).
		Update("pinned", pinned).Error
}

// Count 统计消息数量
func (r *MessageRepository) Count(channelID string) (int64, error) {
	var count int64
	err := r.db.GetChannelDB().Model(&models.Message{}).
		Where("channel_id = ? AND deleted = ?", channelID, false).
		Count(&count).Error
	return count, err
}

// AddReaction 添加表情回应
func (r *MessageRepository) AddReaction(reaction *models.MessageReaction) error {
	return r.db.GetChannelDB().Create(reaction).Error
}

// RemoveReaction 移除表情回应
func (r *MessageRepository) RemoveReaction(messageID, userID, emoji string) error {
	return r.db.GetChannelDB().Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
		Delete(&models.MessageReaction{}).Error
}

// GetReactions 获取消息的所有表情回应
func (r *MessageRepository) GetReactions(messageID string) ([]*models.MessageReaction, error) {
	var reactions []*models.MessageReaction
	err := r.db.GetChannelDB().Where("message_id = ?", messageID).
		Order("created_at ASC").
		Find(&reactions).Error
	if err != nil {
		return nil, err
	}
	return reactions, nil
}

// SetTypingStatus 设置正在输入状态
func (r *MessageRepository) SetTypingStatus(channelID, userID string) error {
	// 先删除旧的状态
	r.db.GetChannelDB().Where("channel_id = ? AND user_id = ?", channelID, userID).
		Delete(&models.TypingStatus{})

	// 创建新状态
	typing := &models.TypingStatus{
		ChannelID: channelID,
		UserID:    userID,
		Timestamp: time.Now(),
	}
	return r.db.GetChannelDB().Create(typing).Error
}

// GetTypingUsers 获取正在输入的用户
func (r *MessageRepository) GetTypingUsers(channelID string) ([]*models.TypingStatus, error) {
	// 获取5秒内的输入状态
	cutoff := time.Now().Add(-5 * time.Second)

	var typing []*models.TypingStatus
	err := r.db.GetChannelDB().Where("channel_id = ? AND timestamp > ?", channelID, cutoff).
		Find(&typing).Error
	if err != nil {
		return nil, err
	}
	return typing, nil
}

// CleanExpiredTypingStatus 清理过期的输入状态
func (r *MessageRepository) CleanExpiredTypingStatus() error {
	cutoff := time.Now().Add(-10 * time.Second)
	return r.db.GetChannelDB().Where("timestamp < ?", cutoff).
		Delete(&models.TypingStatus{}).Error
}

// Search 搜索消息（使用LIKE查询）
func (r *MessageRepository) Search(channelID, keyword string, limit, offset int) ([]*models.Message, error) {
	if keyword == "" {
		return r.GetByChannelID(channelID, limit, offset)
	}

	// 使用 LIKE 搜索
	like := "%" + keyword + "%"
	var messages []*models.Message
	if err := r.db.GetChannelDB().Where("channel_id = ? AND deleted = 0 AND (content_text LIKE ? OR sender_nickname LIKE ? OR tags LIKE ?)",
		channelID, like, like, like).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetMessagesByTimeRange 按时间范围获取消息（含起止，按时间升序）
func (r *MessageRepository) GetMessagesByTimeRange(channelID string, start, end time.Time, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	q := r.db.GetChannelDB().Where("channel_id = ? AND deleted = ?", channelID, false)
	if !start.IsZero() {
		q = q.Where("timestamp >= ?", start)
	}
	if !end.IsZero() {
		q = q.Where("timestamp <= ?", end)
	}
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Order("timestamp ASC").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetMessagesByTag 按标签获取消息（LIKE 匹配）
func (r *MessageRepository) GetMessagesByTag(channelID string, tag string, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	like := "%" + tag + "%"
	err := r.db.GetChannelDB().Where("channel_id = ? AND deleted = 0 AND tags LIKE ?", channelID, like).
		Order("timestamp DESC").Limit(limit).Offset(offset).Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// GetMentionedMessages 获取@我的消息（mentions LIKE）
func (r *MessageRepository) GetMentionedMessages(channelID string, myID string, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	like := "%" + myID + "%"
	err := r.db.GetChannelDB().Where("channel_id = ? AND deleted = 0 AND mentions LIKE ?", channelID, like).
		Order("timestamp DESC").Limit(limit).Offset(offset).Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// BatchCreate 批量创建消息
func (r *MessageRepository) BatchCreate(messages []*models.Message) error {
	if len(messages) == 0 {
		return nil
	}
	return r.db.GetChannelDB().Create(&messages).Error
}

// GetMessageStats 获取消息统计信息（按天聚合）
func (r *MessageRepository) GetMessageStats(channelID string, from, to time.Time) (map[string]int64, error) {
	type row struct {
		Day string
		Cnt int64
	}
	var rows []row
	q := r.db.GetChannelDB().Table("messages").
		Select("strftime('%Y-%m-%d', timestamp) AS day, COUNT(1) AS cnt").
		Where("channel_id = ? AND deleted = 0", channelID)
	if !from.IsZero() {
		q = q.Where("timestamp >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("timestamp <= ?", to)
	}
	if err := q.Group("day").Order("day ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, r0 := range rows {
		out[r0.Day] = r0.Cnt
	}
	return out, nil
}
