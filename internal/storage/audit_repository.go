package storage

import (
	"crosswire/internal/models"
	"time"
)

// AuditRepository 审计日志仓库
type AuditRepository struct {
	db *Database
}

// NewAuditRepository 创建审计日志仓库
func NewAuditRepository(db *Database) *AuditRepository {
	return &AuditRepository{db: db}
}

// Log 记录审计日志
func (r *AuditRepository) Log(log *models.AuditLog) error {
	return r.db.GetChannelDB().Create(log).Error
}

// GetByChannelID 获取频道的审计日志
func (r *AuditRepository) GetByChannelID(channelID string, limit, offset int) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog
	err := r.db.GetChannelDB().Where("channel_id = ?", channelID).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// GetByType 按类型获取审计日志
func (r *AuditRepository) GetByType(channelID, logType string, limit, offset int) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog
	err := r.db.GetChannelDB().Where("channel_id = ? AND type = ?", channelID, logType).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// GetByOperator 按操作者获取审计日志
func (r *AuditRepository) GetByOperator(operatorID string, limit, offset int) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog
	err := r.db.GetChannelDB().Where("operator_id = ?", operatorID).
		Order("timestamp DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// GetByTimeRange 按时间范围获取审计日志
func (r *AuditRepository) GetByTimeRange(channelID string, start, end time.Time) ([]*models.AuditLog, error) {
	var logs []*models.AuditLog
	err := r.db.GetChannelDB().Where("channel_id = ? AND timestamp BETWEEN ? AND ?", channelID, start, end).
		Order("timestamp DESC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// CleanOldLogs 清理旧日志（保留最近N天）
func (r *AuditRepository) CleanOldLogs(days int) error {
	cutoff := time.Now().AddDate(0, 0, -days)
	return r.db.GetChannelDB().Where("timestamp < ?", cutoff).
		Delete(&models.AuditLog{}).Error
}

// Count 统计日志数量
func (r *AuditRepository) Count(channelID string) (int64, error) {
	var count int64
	err := r.db.GetChannelDB().Model(&models.AuditLog{}).
		Where("channel_id = ?", channelID).
		Count(&count).Error
	return count, err
}
