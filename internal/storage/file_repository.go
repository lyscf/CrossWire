package storage

import (
	"time"

	"crosswire/internal/models"
)

// FileRepository 文件数据仓库
type FileRepository struct {
	db *Database
}

// NewFileRepository 创建文件仓库
func NewFileRepository(db *Database) *FileRepository {
	return &FileRepository{db: db}
}

// Create 创建文件记录
func (r *FileRepository) Create(file *models.File) error {
	return r.db.GetChannelDB().Create(file).Error
}

// GetByID 根据ID获取文件
func (r *FileRepository) GetByID(fileID string) (*models.File, error) {
	var file models.File
	err := r.db.GetChannelDB().Where("id = ?", fileID).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByChannelID 获取频道所有文件
func (r *FileRepository) GetByChannelID(channelID string, limit, offset int) ([]*models.File, error) {
	var files []*models.File
	err := r.db.GetChannelDB().Where("channel_id = ?", channelID).
		Order("uploaded_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

// GetBySenderID 获取指定用户上传的文件
func (r *FileRepository) GetBySenderID(senderID string, limit, offset int) ([]*models.File, error) {
	var files []*models.File
	err := r.db.GetChannelDB().Where("sender_id = ?", senderID).
		Order("uploaded_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

// Update 更新文件记录
func (r *FileRepository) Update(file *models.File) error {
	return r.db.GetChannelDB().Save(file).Error
}

// UpdateUploadStatus 更新上传状态
func (r *FileRepository) UpdateUploadStatus(fileID string, status models.UploadStatus, uploadedChunks int) error {
	return r.db.GetChannelDB().Model(&models.File{}).
		Where("id = ?", fileID).
		Updates(map[string]interface{}{
			"upload_status":   status,
			"uploaded_chunks": uploadedChunks,
		}).Error
}

// Delete 删除文件记录
func (r *FileRepository) Delete(fileID string) error {
	return r.db.GetChannelDB().Where("id = ?", fileID).Delete(&models.File{}).Error
}

// GetExpiredFiles 获取过期的文件
func (r *FileRepository) GetExpiredFiles() ([]*models.File, error) {
	var files []*models.File
	now := time.Now()
	err := r.db.GetChannelDB().Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

// CleanExpiredFiles 清理过期文件
func (r *FileRepository) CleanExpiredFiles() error {
	now := time.Now()
	return r.db.GetChannelDB().Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Delete(&models.File{}).Error
}

// CreateChunk 创建文件分块记录
func (r *FileRepository) CreateChunk(chunk *models.FileChunk) error {
	return r.db.GetChannelDB().Create(chunk).Error
}

// UpdateChunk 更新文件分块
func (r *FileRepository) UpdateChunk(chunk *models.FileChunk) error {
	return r.db.GetChannelDB().Save(chunk).Error
}

// GetChunksByFileID 获取文件的所有分块
func (r *FileRepository) GetChunksByFileID(fileID string) ([]*models.FileChunk, error) {
	var chunks []*models.FileChunk
	err := r.db.GetChannelDB().Where("file_id = ?", fileID).
		Order("chunk_index ASC").
		Find(&chunks).Error
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

// GetPendingChunks 获取待上传的分块
func (r *FileRepository) GetPendingChunks(fileID string) ([]*models.FileChunk, error) {
	var chunks []*models.FileChunk
	err := r.db.GetChannelDB().Where("file_id = ? AND uploaded = ?", fileID, false).
		Order("chunk_index ASC").
		Find(&chunks).Error
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

// MarkChunkUploaded 标记分块为已上传
func (r *FileRepository) MarkChunkUploaded(chunkID int) error {
	return r.db.GetChannelDB().Model(&models.FileChunk{}).
		Where("id = ?", chunkID).
		Updates(map[string]interface{}{
			"uploaded":    true,
			"uploaded_at": time.Now(),
		}).Error
}

// GetUploadProgress 获取文件上传进度
func (r *FileRepository) GetUploadProgress(fileID string) (uploaded, total int, err error) {
	var file models.File
	err = r.db.GetChannelDB().Where("id = ?", fileID).First(&file).Error
	if err != nil {
		return 0, 0, err
	}
	return file.UploadedChunks, file.TotalChunks, nil
}

// SearchFiles 按名称/类型搜索文件
func (r *FileRepository) SearchFiles(channelID string, keyword string, mimeLike string, limit, offset int) ([]*models.File, error) {
	var files []*models.File
	q := r.db.GetChannelDB().Where("channel_id = ?", channelID)
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name LIKE ?", like)
	}
	if mimeLike != "" {
		like := mimeLike
		if like[len(like)-1] != '%' {
			like = like + "%"
		}
		q = q.Where("mime_type LIKE ?", like)
	}
	if limit > 0 {
		q = q.Limit(limit).Offset(offset)
	}
	if err := q.Order("uploaded_at DESC").Find(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}

// GetFilesByType 按MIME大类获取文件
func (r *FileRepository) GetFilesByType(channelID string, mimePrefix string, limit, offset int) ([]*models.File, error) {
	var files []*models.File
	like := mimePrefix
	if like == "" {
		like = "%"
	} else if like[len(like)-1] != '%' {
		like = like + "%"
	}
	err := r.db.GetChannelDB().Where("channel_id = ? AND mime_type LIKE ?", channelID, like).
		Order("uploaded_at DESC").
		Limit(limit).Offset(offset).Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}

// GetTotalSize 获取频道文件总大小
func (r *FileRepository) GetTotalSize(channelID string) (int64, error) {
	var total int64
	err := r.db.GetChannelDB().Model(&models.File{}).Where("channel_id = ?", channelID).
		Select("COALESCE(SUM(size),0)").Scan(&total).Error
	return total, err
}

// GetFileStats 获取文件统计信息
func (r *FileRepository) GetFileStats(channelID string) (map[string]interface{}, error) {
	var count int64
	if err := r.db.GetChannelDB().Model(&models.File{}).Where("channel_id = ?", channelID).Count(&count).Error; err != nil {
		return nil, err
	}
	totalSize, err := r.GetTotalSize(channelID)
	if err != nil {
		return nil, err
	}
	stats := map[string]interface{}{
		"count":      count,
		"total_size": totalSize,
	}
	return stats, nil
}

// GetPendingUploads 查询未完成的上传任务
func (r *FileRepository) GetPendingUploads(channelID string) ([]*models.File, error) {
	var files []*models.File
	err := r.db.GetChannelDB().Where("channel_id = ? AND (upload_status = ? OR upload_status = ?)",
		channelID, models.UploadStatusPaused, models.UploadStatusUploading).
		Order("uploaded_at DESC").
		Find(&files).Error
	if err != nil {
		return nil, err
	}
	return files, nil
}
