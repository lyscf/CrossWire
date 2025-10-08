package storage

import (
	"crosswire/internal/models"

	"gorm.io/gorm"
)

// ChannelRepository 频道数据仓库
type ChannelRepository struct {
	db *Database
}

// NewChannelRepository 创建频道仓库
func NewChannelRepository(db *Database) *ChannelRepository {
	return &ChannelRepository{db: db}
}

// Create 创建频道
func (r *ChannelRepository) Create(channel *models.Channel) error {
	return r.db.GetChannelDB().Create(channel).Error
}

// GetByID 根据ID获取频道
func (r *ChannelRepository) GetByID(channelID string) (*models.Channel, error) {
	var channel models.Channel
	err := r.db.GetChannelDB().Where("id = ?", channelID).First(&channel).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// Update 更新频道
func (r *ChannelRepository) Update(channel *models.Channel) error {
	return r.db.GetChannelDB().Save(channel).Error
}

// Delete 删除频道
func (r *ChannelRepository) Delete(channelID string) error {
	return r.db.GetChannelDB().Where("id = ?", channelID).Delete(&models.Channel{}).Error
}

// IncrementMessageCount 增加消息计数
func (r *ChannelRepository) IncrementMessageCount(channelID string) error {
	return r.db.GetChannelDB().Model(&models.Channel{}).
		Where("id = ?", channelID).
		UpdateColumn("message_count", gorm.Expr("message_count + ?", 1)).Error
}

// IncrementFileCount 增加文件计数
func (r *ChannelRepository) IncrementFileCount(channelID string) error {
	return r.db.GetChannelDB().Model(&models.Channel{}).
		Where("id = ?", channelID).
		UpdateColumn("file_count", gorm.Expr("file_count + ?", 1)).Error
}

// AddTraffic 添加流量统计
func (r *ChannelRepository) AddTraffic(channelID string, bytes uint64) error {
	return r.db.GetChannelDB().Model(&models.Channel{}).
		Where("id = ?", channelID).
		UpdateColumn("total_traffic", gorm.Expr("total_traffic + ?", bytes)).Error
}

// GetStats 获取频道统计信息
func (r *ChannelRepository) GetStats(channelID string) (map[string]interface{}, error) {
	var channel models.Channel
	err := r.db.GetChannelDB().Where("id = ?", channelID).First(&channel).Error
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"message_count": channel.MessageCount,
		"file_count":    channel.FileCount,
		"total_traffic": channel.TotalTraffic,
	}

	return stats, nil
}

// PinMessage 置顶消息
func (r *ChannelRepository) PinMessage(channelID, messageID, pinnedBy, reason string) error {
	// 获取当前最大的display_order
	var maxOrder int
	err := r.db.GetChannelDB().Model(&models.PinnedMessage{}).
		Where("channel_id = ?", channelID).
		Select("COALESCE(MAX(display_order), 0)").
		Scan(&maxOrder).Error
	if err != nil {
		return err
	}

	pinned := &models.PinnedMessage{
		ChannelID:    channelID,
		MessageID:    messageID,
		PinnedBy:     pinnedBy,
		Reason:       reason,
		DisplayOrder: maxOrder + 1,
	}

	return r.db.GetChannelDB().Create(pinned).Error
}

// UnpinMessage 取消置顶消息
func (r *ChannelRepository) UnpinMessage(channelID, messageID string) error {
	return r.db.GetChannelDB().Where("channel_id = ? AND message_id = ?", channelID, messageID).
		Delete(&models.PinnedMessage{}).Error
}

// GetPinnedMessages 获取置顶消息列表
func (r *ChannelRepository) GetPinnedMessages(channelID string) ([]*models.PinnedMessage, error) {
	var pinned []*models.PinnedMessage
	err := r.db.GetChannelDB().Where("channel_id = ?", channelID).
		Order("display_order ASC").
		Find(&pinned).Error
	if err != nil {
		return nil, err
	}
	return pinned, nil
}

// GetPinnedMessagesWithContent 获取带消息内容的置顶列表
func (r *ChannelRepository) GetPinnedMessagesWithContent(channelID string) ([]*struct {
	models.PinnedMessage
	ContentText    string `json:"content_text"`
	SenderID       string `json:"sender_id"`
	SenderNickname string `json:"sender_nickname"`
}, error) {
	type pinnedWith struct {
		models.PinnedMessage
		ContentText    string `json:"content_text"`
		SenderID       string `json:"sender_id"`
		SenderNickname string `json:"sender_nickname"`
	}

	var result []*pinnedWith
	err := r.db.GetChannelDB().
		Table("pinned_messages").
		Select("pinned_messages.*, messages.content_text AS content_text, messages.sender_id AS sender_id, messages.sender_nickname AS sender_nickname").
		Joins("INNER JOIN messages ON pinned_messages.message_id = messages.id").
		Where("pinned_messages.channel_id = ?", channelID).
		Order("pinned_messages.display_order ASC").
		Scan(&result).Error
	if err != nil {
		return nil, err
	}

	// Convert to anonymous struct slice to avoid leaking local type
	out := make([]*struct {
		models.PinnedMessage
		ContentText    string `json:"content_text"`
		SenderID       string `json:"sender_id"`
		SenderNickname string `json:"sender_nickname"`
	}, 0, len(result))
	for _, item := range result {
		out = append(out, &struct {
			models.PinnedMessage
			ContentText    string `json:"content_text"`
			SenderID       string `json:"sender_id"`
			SenderNickname string `json:"sender_nickname"`
		}{
			PinnedMessage:  item.PinnedMessage,
			ContentText:    item.ContentText,
			SenderID:       item.SenderID,
			SenderNickname: item.SenderNickname,
		})
	}
	return out, nil
}

// GetSubChannels 获取子频道列表（题目频道）
func (r *ChannelRepository) GetSubChannels(parentChannelID string) ([]*models.Channel, error) {
	var channels []*models.Channel
	err := r.db.GetChannelDB().Where("parent_channel_id = ?", parentChannelID).
		Order("created_at DESC").
		Find(&channels).Error
	if err != nil {
		return nil, err
	}
	return channels, nil
}

// TODO: 实现以下方法
// - UpdateMetadata() 更新频道元数据
// - GetChannelConfig() 获取频道配置
// - RotateEncryptionKey() 轮换加密密钥
