package app

import (
	"crosswire/internal/events"
	"crosswire/internal/models"
	"encoding/base64"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// ==================== 文件操作 API ====================

// UploadFile 上传文件
func (a *App) UploadFile(req UploadFileRequest) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 验证文件路径
	if req.FilePath == "" {
		return NewErrorResponse("invalid_request", "文件路径不能为空", "")
	}

	// 检查文件是否存在
	fileInfo, err := os.Stat(req.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResponse("file_not_found", "文件不存在", req.FilePath)
		}
		return NewErrorResponse("file_error", "无法访问文件", err.Error())
	}

	// 检查是否为文件（非目录）
	if fileInfo.IsDir() {
		return NewErrorResponse("invalid_file", "不能上传目录", "")
	}

	a.logger.Info("Uploading file: %s (%d bytes)", req.FilePath, fileInfo.Size())

	// 上传文件
	var fileID string
	if mode == ModeClient && cli != nil {
		task, err2 := cli.UploadFile(req.FilePath)
		err = err2
		if task != nil {
			fileID = task.ID
		}
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("upload_error", "文件上传失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"file_id":  fileID,
		"filename": filepath.Base(req.FilePath),
		"size":     fileInfo.Size(),
		"message":  "文件上传已开始",
	})
}

// DownloadFile 下载文件
func (a *App) DownloadFile(req DownloadFileRequest) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 验证请求
	if req.FileID == "" {
		return NewErrorResponse("invalid_request", "文件ID不能为空", "")
	}

	if req.SavePath == "" {
		return NewErrorResponse("invalid_request", "保存路径不能为空", "")
	}

	// 检查保存目录是否存在
	saveDir := filepath.Dir(req.SavePath)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return NewErrorResponse("path_error", "无法创建保存目录", err.Error())
	}

	a.logger.Info("Downloading file: %s -> %s", req.FileID, req.SavePath)

	// 下载文件
	var err error
	if mode == ModeClient && cli != nil {
		_, err = cli.DownloadFile(req.FileID, req.SavePath)
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("download_error", "文件下载失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"file_id":   req.FileID,
		"save_path": req.SavePath,
		"message":   "文件下载已开始",
	})
}

// CancelUpload 取消文件上传
func (a *App) CancelUpload(fileID string) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 取消上传
	var err error
	if mode == ModeClient && cli != nil {
		err = cli.CancelUpload(fileID)
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("cancel_error", "取消上传失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "上传已取消",
	})
}

// CancelDownload 取消文件下载
func (a *App) CancelDownload(fileID string) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 取消下载
	var err error
	if mode == ModeClient && cli != nil {
		err = cli.CancelDownload(fileID)
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if err != nil {
		return NewErrorResponse("cancel_error", "取消下载失败", err.Error())
	}

	return NewSuccessResponse(map[string]interface{}{
		"message": "下载已取消",
	})
}

// GetFiles 获取文件列表
func (a *App) GetFiles(limit, offset int) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 获取当前频道ID
	var channelID string
	if a.mode == ModeServer && a.server != nil {
		ch, _ := a.server.GetChannel()
		channelID = ch.ID
	} else if a.mode == ModeClient && a.client != nil {
		channelID = a.client.GetChannelID()
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	// 从数据库获取文件列表
	files, err := a.db.FileRepo().GetByChannelID(channelID, limit, offset)
	if err != nil {
		return NewErrorResponse("db_error", "获取文件列表失败", err.Error())
	}

	// 转换为DTO
	fileDTOs := make([]*FileDTO, 0, len(files))
	for _, file := range files {
		dto := a.fileToDTO(file)
		fileDTOs = append(fileDTOs, dto)
	}

	return NewSuccessResponse(fileDTOs)
}

// GetFile 获取单个文件信息
func (a *App) GetFile(fileID string) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 从数据库获取文件
	file, err := a.db.FileRepo().GetByID(fileID)
	if err != nil {
		return NewErrorResponse("not_found", "文件不存在", err.Error())
	}

	dto := a.fileToDTO(file)
	return NewSuccessResponse(dto)
}

// GetFileContent 获取可预览的文件内容（文本/图片等）
// 文本类直接返回字符串；图片等返回 data URL（base64）
func (a *App) GetFileContent(fileID string) Response {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	file, err := a.db.FileRepo().GetByID(fileID)
	if err != nil || file == nil {
		return NewErrorResponse("not_found", "文件不存在", "")
	}

	// 优先使用存储路径
	var data []byte
	if file.StorageType == models.StorageFile && file.StoragePath != "" {
		b, err := os.ReadFile(file.StoragePath)
		if err != nil {
			return NewErrorResponse("read_error", "读取文件失败", err.Error())
		}
		data = b
	} else if len(file.Data) > 0 {
		data = file.Data
	} else {
		return NewErrorResponse("empty", "文件内容为空", "")
	}

	mt := file.MimeType
	if mt == "" {
		mt = mime.TypeByExtension(filepath.Ext(file.Filename))
	}
	if strings.HasPrefix(mt, "text/") || strings.Contains(mt, "json") || strings.Contains(mt, "xml") {
		return NewSuccessResponse(map[string]interface{}{
			"mode": "text",
			"mime": mt,
			"text": string(data),
			"name": file.Filename,
			"size": file.Size,
		})
	}

	// 其他类型转为 data URL（便于前端 img/pdf viewer 预览）
	b64 := base64.StdEncoding.EncodeToString(data)
	dataURL := "data:" + mt + ";base64," + b64
	return NewSuccessResponse(map[string]interface{}{
		"mode":    "dataurl",
		"mime":    mt,
		"dataUrl": dataURL,
		"name":    file.Filename,
		"size":    file.Size,
	})
}

// GetFileProgress 获取文件传输进度
func (a *App) GetFileProgress(fileID string) Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 获取进度
	var progress *FileTransferProgress
	var err error

	if mode == ModeClient && cli != nil {
		// 从客户端获取进度
		if uploadTask, ok := cli.GetUploadTask(fileID); ok && uploadTask != nil {
			progress = &FileTransferProgress{
				FileID:      fileID,
				FileName:    uploadTask.Filename,
				TotalSize:   uploadTask.Size,
				Transferred: int64(uploadTask.UploadedChunks) * int64(uploadTask.ChunkSize),
				Progress:    int(uploadTask.GetProgress() * 100),
				Speed:       0,
				Status:      string(uploadTask.Status),
			}
		} else {
			if downloadTask, ok := cli.GetDownloadTaskByFileID(fileID); ok && downloadTask != nil {
				progress = &FileTransferProgress{
					FileID:      fileID,
					FileName:    downloadTask.Filename,
					TotalSize:   downloadTask.Size,
					Transferred: int64(downloadTask.ReceivedChunks) * int64(downloadTask.ChunkSize),
					Progress:    int(downloadTask.GetProgress() * 100),
					Speed:       0,
					Status:      string(downloadTask.Status),
				}
			}
		}
	}

	if progress == nil {
		return NewErrorResponse("not_found", "未找到传输任务", "")
	}

	if err != nil {
		return NewErrorResponse("error", "获取进度失败", err.Error())
	}

	return NewSuccessResponse(progress)
}

// GetFileTransferStats 获取文件传输统计
func (a *App) GetFileTransferStats() Response {
	a.mu.RLock()
	mode := a.mode
	_ = a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	// 获取统计信息
	var stats map[string]interface{}

	if mode == ModeClient && cli != nil {
		clientStats := cli.GetFileManagerStats()
		stats = map[string]interface{}{
			"total_uploads":        clientStats.TotalUploads,
			"total_downloads":      clientStats.TotalDownloads,
			"successful_uploads":   clientStats.SuccessfulUploads,
			"successful_downloads": clientStats.SuccessfulDownloads,
			"failed_uploads":       clientStats.FailedUploads,
			"failed_downloads":     clientStats.FailedDownloads,
			"bytes_uploaded":       clientStats.BytesUploaded,
			"bytes_downloaded":     clientStats.BytesDownloaded,
		}
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	return NewSuccessResponse(stats)
}

// DeleteFile 删除文件
func (a *App) DeleteFile(fileID string) Response {
	a.mu.RLock()
	mode := a.mode
	srv := a.server
	cli := a.client
	a.mu.RUnlock()

	if !a.isRunning {
		return NewErrorResponse("not_running", "未连接到频道", "")
	}

	if fileID == "" {
		return NewErrorResponse("invalid_request", "文件ID不能为空", "")
	}

	a.logger.Info("Deleting file: %s", fileID)

	// 1. 读取文件
	file, err := a.db.FileRepo().GetByID(fileID)
	if err != nil || file == nil {
		return NewErrorResponse("not_found", "文件不存在", err.Error())
	}

	// 2. 权限检查：上传者或管理员
	var currentUserID string
	var isAdmin bool

	if mode == ModeServer && srv != nil {
		currentUserID = "server"
		isAdmin = true
	} else if mode == ModeClient && cli != nil {
		currentUserID = cli.GetMemberID()
		member, _ := a.db.MemberRepo().GetByID(currentUserID)
		isAdmin = (member != nil && member.Role == models.RoleAdmin)
	} else {
		return NewErrorResponse("invalid_mode", "无效的运行模式", "")
	}

	if file.SenderID != currentUserID && !isAdmin {
		return NewErrorResponse("permission_denied", "仅上传者或管理员可删除文件", "")
	}

	// 3. 处理物理文件
	switch file.StorageType {
	case models.StorageFile:
		if file.StoragePath != "" {
			if err := os.Remove(file.StoragePath); err != nil {
				a.logger.Warn("Failed to delete physical file: %v", err)
			}
		}
	default:
		// 其他存储类型由数据库删除处理
	}

	// 4. 删除数据库记录（级联删除分块）
	if err := a.db.FileRepo().Delete(fileID); err != nil {
		return NewErrorResponse("delete_error", "删除文件记录失败", err.Error())
	}

	// 5. 广播文件删除事件
	a.eventBus.Publish(events.EventFileDeleted, map[string]interface{}{
		"file_id":  fileID,
		"filename": file.Filename,
	})

	a.logger.Info("File deleted: %s", fileID)

	return NewSuccessResponse(map[string]interface{}{
		"message": "文件已删除",
		"file_id": fileID,
	})
}

// ==================== 辅助方法 ====================

// fileToDTO 转换文件模型为DTO
func (a *App) fileToDTO(file *models.File) *FileDTO {
	// 获取上传者信息
	uploaderName := "Unknown"

	return &FileDTO{
		ID:           file.ID,
		Name:         file.Filename,
		Size:         file.Size,
		MimeType:     file.MimeType,
		UploaderID:   file.SenderID,
		UploaderName: uploaderName,
		UploadStatus: file.UploadStatus,
		Progress:     int(float64(file.UploadedChunks) / float64(max(1, file.TotalChunks)) * 100),
		UploadTime:   file.UploadedAt.Unix(),
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
