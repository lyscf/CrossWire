package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"time"

	"crosswire/internal/events"
	"crosswire/internal/models"
	"crosswire/internal/transport"

	"github.com/google/uuid"
)

// FileManager 文件传输管理器
type FileManager struct {
	client *Client
	ctx    context.Context
	cancel context.CancelFunc

	// 上传任务
	uploads      map[string]*FileUploadTask
	uploadsMutex sync.RWMutex

	// 下载任务
	downloads      map[string]*FileDownloadTask
	downloadsMutex sync.RWMutex

	// 统计信息
	stats      FileManagerStats
	statsMutex sync.RWMutex
}

// FileUploadTask 文件上传任务
type FileUploadTask struct {
	ID             string
	FilePath       string
	Filename       string
	Size           int64
	MimeType       string
	ChunkSize      int
	TotalChunks    int
	UploadedChunks int
	Status         models.UploadStatus
	SHA256         string
	StartTime      time.Time
	EndTime        *time.Time
	Error          error

	// 分块状态
	chunkStatus []bool
	mutex       sync.RWMutex

	// 进度回调
	OnProgress func(task *FileUploadTask)
}

// FileDownloadTask 文件下载任务
type FileDownloadTask struct {
	ID             string
	FileID         string
	Filename       string
	Size           int64
	SavePath       string
	ChunkSize      int
	TotalChunks    int
	ReceivedChunks int
	Status         DownloadStatus
	SHA256         string
	StartTime      time.Time
	EndTime        *time.Time
	Error          error

	// 分块数据
	chunks      map[int][]byte
	chunksMutex sync.RWMutex

	// 进度回调
	OnProgress func(task *FileDownloadTask)
}

// DownloadStatus 下载状态
type DownloadStatus string

const (
	DownloadStatusPending     DownloadStatus = "pending"
	DownloadStatusDownloading DownloadStatus = "downloading"
	DownloadStatusCompleted   DownloadStatus = "completed"
	DownloadStatusFailed      DownloadStatus = "failed"
)

// FileManagerStats 文件管理器统计
type FileManagerStats struct {
	TotalUploads        int64
	TotalDownloads      int64
	SuccessfulUploads   int64
	SuccessfulDownloads int64
	FailedUploads       int64
	FailedDownloads     int64
	BytesUploaded       int64
	BytesDownloaded     int64
	mutex               sync.RWMutex
}

// NewFileManager 创建文件管理器
func NewFileManager(client *Client) *FileManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &FileManager{
		client:    client,
		ctx:       ctx,
		cancel:    cancel,
		uploads:   make(map[string]*FileUploadTask),
		downloads: make(map[string]*FileDownloadTask),
	}
}

// Start 启动文件管理器
func (fm *FileManager) Start() error {
	fm.client.logger.Info("[FileManager] Starting...")

	// 订阅文件相关事件
	fm.client.eventBus.Subscribe(events.EventFileProgress, fm.handleFileReceived)

	fm.client.logger.Info("[FileManager] Started successfully")
	return nil
}

// Stop 停止文件管理器
func (fm *FileManager) Stop() error {
	fm.client.logger.Info("[FileManager] Stopping...")
	fm.cancel()

	// 取消所有进行中的任务
	fm.uploadsMutex.Lock()
	for _, task := range fm.uploads {
		if task.Status == models.UploadStatusUploading {
			task.Status = models.UploadStatusFailed
			task.Error = fmt.Errorf("cancelled")
		}
	}
	fm.uploadsMutex.Unlock()

	fm.downloadsMutex.Lock()
	for _, task := range fm.downloads {
		if task.Status == DownloadStatusDownloading {
			task.Status = DownloadStatusFailed
			task.Error = fmt.Errorf("cancelled")
		}
	}
	fm.downloadsMutex.Unlock()

	fm.client.logger.Info("[FileManager] Stopped")
	return nil
}

// UploadFile 上传文件
func (fm *FileManager) UploadFile(filePath string) (*FileUploadTask, error) {
	fm.client.logger.Info("[FileManager] Uploading file: %s", filePath)

	// 1. 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 2. 获取文件信息
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// 3. 计算文件哈希
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	// 重置文件指针
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	// 4. 创建上传任务
	chunkSize := fm.getOptimalChunkSize()
	totalChunks := int(math.Ceil(float64(stat.Size()) / float64(chunkSize)))

	task := &FileUploadTask{
		ID:             uuid.New().String(),
		FilePath:       filePath,
		Filename:       filepath.Base(filePath),
		Size:           stat.Size(),
		MimeType:       mime.TypeByExtension(filepath.Ext(filePath)),
		ChunkSize:      chunkSize,
		TotalChunks:    totalChunks,
		UploadedChunks: 0,
		Status:         models.UploadStatusPending,
		SHA256:         fileHash,
		StartTime:      time.Now(),
		chunkStatus:    make([]bool, totalChunks),
	}

	// 5. 注册任务
	fm.uploadsMutex.Lock()
	fm.uploads[task.ID] = task
	fm.uploadsMutex.Unlock()

	// 6. 更新统计
	fm.statsMutex.Lock()
	fm.stats.TotalUploads++
	fm.statsMutex.Unlock()

	// 7. 异步执行上传
	go fm.executeUpload(task, file)

	return task, nil
}

// executeUpload 执行文件上传
func (fm *FileManager) executeUpload(task *FileUploadTask, file *os.File) {
	defer file.Close()

	task.mutex.Lock()
	task.Status = models.UploadStatusUploading
	task.mutex.Unlock()

	fm.client.logger.Debug("[FileManager] Starting upload task: %s", task.ID)

	// 1. 发送文件元数据
	if err := fm.sendFileMetadata(task); err != nil {
		fm.pauseUpload(task, fmt.Errorf("failed to send metadata: %w", err))
		return
	}

	// 2. 分块上传
	buffer := make([]byte, task.ChunkSize)
	for chunkIndex := 0; chunkIndex < task.TotalChunks; chunkIndex++ {
		// 检查是否已上传
		task.mutex.RLock()
		alreadyUploaded := task.chunkStatus[chunkIndex]
		task.mutex.RUnlock()

		if alreadyUploaded {
			fm.client.logger.Debug("[FileManager] Skip chunk %d (already uploaded)", chunkIndex)
			continue
		}

		// 检查取消
		select {
		case <-fm.ctx.Done():
			fm.pauseUpload(task, fmt.Errorf("cancelled"))
			return
		default:
		}

		// 定位到正确的文件位置
		offset := int64(chunkIndex) * int64(task.ChunkSize)
		if _, err := file.Seek(offset, 0); err != nil {
			fm.pauseUpload(task, fmt.Errorf("failed to seek chunk %d: %w", chunkIndex, err))
			return
		}

		// 读取分块
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			fm.pauseUpload(task, fmt.Errorf("failed to read chunk %d: %w", chunkIndex, err))
			return
		}

		chunkData := buffer[:n]

		// 计算分块哈希
		chunkHash := sha256.Sum256(chunkData)

		// 加密分块
		encrypted, err := fm.client.crypto.EncryptMessage(chunkData)
		if err != nil {
			fm.pauseUpload(task, fmt.Errorf("failed to encrypt chunk %d: %w", chunkIndex, err))
			return
		}

		// 发送分块
		if err := fm.sendFileChunk(task, chunkIndex, encrypted, hex.EncodeToString(chunkHash[:])); err != nil {
			fm.pauseUpload(task, fmt.Errorf("failed to send chunk %d: %w", chunkIndex, err))
			return
		}

		// 更新状态
		task.mutex.Lock()
		task.chunkStatus[chunkIndex] = true
		task.UploadedChunks++
		task.mutex.Unlock()

		// 持久化任务状态（每10个分块保存一次）
		if chunkIndex%10 == 0 || chunkIndex == task.TotalChunks-1 {
			fm.saveUploadTaskState(task)
		}

		// 触发进度回调
		if task.OnProgress != nil {
			task.OnProgress(task)
		}

		fm.client.logger.Debug("[FileManager] Upload progress: %d/%d chunks", task.UploadedChunks, task.TotalChunks)
	}

	// 3. 发送完成消息
	if err := fm.sendFileComplete(task); err != nil {
		fm.pauseUpload(task, fmt.Errorf("failed to send complete: %w", err))
		return
	}

	// 4. 标记成功
	fm.completeUpload(task)
	fm.deleteUploadTaskState(task.ID) // 删除持久化状态
}

// sendFileMetadata 发送文件元数据
func (fm *FileManager) sendFileMetadata(task *FileUploadTask) error {
	metadata := map[string]interface{}{
		"type":         "file.metadata",
		"file_id":      task.ID,
		"filename":     task.Filename,
		"size":         task.Size,
		"mime_type":    task.MimeType,
		"sha256":       task.SHA256,
		"chunk_size":   task.ChunkSize,
		"total_chunks": task.TotalChunks,
		"timestamp":    time.Now().Unix(),
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	encrypted, err := fm.client.crypto.EncryptMessage(payload)
	if err != nil {
		return fmt.Errorf("failed to encrypt metadata: %w", err)
	}

	msg := &transport.Message{
		ID:        uuid.New().String(),
		Type:      transport.MessageTypeData,
		SenderID:  fm.client.memberID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}

	return fm.client.transport.SendMessage(msg)
}

// sendFileChunk 发送文件分块
func (fm *FileManager) sendFileChunk(task *FileUploadTask, chunkIndex int, data []byte, checksum string) error {
	chunk := map[string]interface{}{
		"type":         "file.chunk",
		"file_id":      task.ID,
		"chunk_index":  chunkIndex,
		"total_chunks": task.TotalChunks,
		"data":         data,
		"checksum":     checksum,
		"timestamp":    time.Now().Unix(),
	}

	payload, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk: %w", err)
	}

	encrypted, err := fm.client.crypto.EncryptMessage(payload)
	if err != nil {
		return fmt.Errorf("failed to encrypt chunk: %w", err)
	}

	msg := &transport.Message{
		ID:        uuid.New().String(),
		Type:      transport.MessageTypeData,
		SenderID:  fm.client.memberID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}

	return fm.client.transport.SendMessage(msg)
}

// sendFileComplete 发送完成消息
func (fm *FileManager) sendFileComplete(task *FileUploadTask) error {
	complete := map[string]interface{}{
		"type":      "file.complete",
		"file_id":   task.ID,
		"timestamp": time.Now().Unix(),
	}

	payload, err := json.Marshal(complete)
	if err != nil {
		return fmt.Errorf("failed to marshal complete: %w", err)
	}

	encrypted, err := fm.client.crypto.EncryptMessage(payload)
	if err != nil {
		return fmt.Errorf("failed to encrypt complete: %w", err)
	}

	msg := &transport.Message{
		ID:        uuid.New().String(),
		Type:      transport.MessageTypeControl,
		SenderID:  fm.client.memberID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}

	return fm.client.transport.SendMessage(msg)
}

// failUpload 标记上传失败
func (fm *FileManager) failUpload(task *FileUploadTask, err error) {
	task.mutex.Lock()
	task.Status = models.UploadStatusFailed
	task.Error = err
	now := time.Now()
	task.EndTime = &now
	task.mutex.Unlock()

	fm.statsMutex.Lock()
	fm.stats.FailedUploads++
	fm.statsMutex.Unlock()

	fm.client.logger.Error("[FileManager] Upload failed: %s - %v", task.ID, err)

	// 发布失败事件（使用系统错误事件）
	fm.client.eventBus.Publish(events.EventSystemError, events.SystemEvent{
		Type:    "file_upload_failed",
		Message: fmt.Sprintf("Upload failed for file %s: %v", task.ID, err),
		Data: map[string]string{
			"file_id":    task.ID,
			"channel_id": fm.client.config.ChannelID,
		},
	})
}

// completeUpload 标记上传完成
func (fm *FileManager) completeUpload(task *FileUploadTask) {
	task.mutex.Lock()
	task.Status = models.UploadStatusCompleted
	now := time.Now()
	task.EndTime = &now
	task.mutex.Unlock()

	fm.statsMutex.Lock()
	fm.stats.SuccessfulUploads++
	fm.stats.BytesUploaded += task.Size
	fm.statsMutex.Unlock()

	fm.client.logger.Info("[FileManager] Upload completed: %s (%d bytes)", task.ID, task.Size)

	// 发布完成事件
	fileRecord := &models.File{
		ID:        task.ID,
		ChannelID: fm.client.config.ChannelID,
		SenderID:  fm.client.memberID,
		Filename:  task.Filename,
		Size:      task.Size,
		SHA256:    task.SHA256,
	}
	fm.client.eventBus.Publish(events.EventFileUploaded, events.FileEvent{
		File:       fileRecord,
		ChannelID:  fm.client.config.ChannelID,
		UploaderID: fm.client.memberID,
		Progress:   100,
	})
}

// DownloadFile 下载文件
func (fm *FileManager) DownloadFile(fileID string, savePath string) (*FileDownloadTask, error) {
	fm.client.logger.Info("[FileManager] Downloading file: %s", fileID)

	// 1. 获取文件信息
	fileInfo, err := fm.client.fileRepo.GetByID(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 2. 创建下载任务
	task := &FileDownloadTask{
		ID:             uuid.New().String(),
		FileID:         fileID,
		Filename:       fileInfo.Filename,
		Size:           fileInfo.Size,
		SavePath:       savePath,
		ChunkSize:      fileInfo.ChunkSize,
		TotalChunks:    fileInfo.TotalChunks,
		ReceivedChunks: 0,
		Status:         DownloadStatusPending,
		SHA256:         fileInfo.SHA256,
		StartTime:      time.Now(),
		chunks:         make(map[int][]byte),
	}

	// 3. 注册任务
	fm.downloadsMutex.Lock()
	fm.downloads[task.ID] = task
	fm.downloadsMutex.Unlock()

	// 4. 更新统计
	fm.statsMutex.Lock()
	fm.stats.TotalDownloads++
	fm.statsMutex.Unlock()

	// 5. 异步执行下载
	go fm.executeDownload(task)

	return task, nil
}

// executeDownload 执行文件下载
func (fm *FileManager) executeDownload(task *FileDownloadTask) {
	task.Status = DownloadStatusDownloading
	fm.client.logger.Debug("[FileManager] Starting download task: %s", task.ID)

	// 1. 请求文件数据
	if err := fm.requestFileData(task); err != nil {
		fm.failDownload(task, fmt.Errorf("failed to request file: %w", err))
		return
	}

	// 2. 等待所有分块接收完成（由 handleFileChunk 处理）
	// 这里只是示例，实际需要实现等待机制
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-fm.ctx.Done():
			fm.failDownload(task, fmt.Errorf("cancelled"))
			return
		case <-timeout:
			fm.failDownload(task, fmt.Errorf("download timeout"))
			return
		case <-ticker.C:
			task.chunksMutex.RLock()
			received := len(task.chunks)
			task.chunksMutex.RUnlock()

			if received >= task.TotalChunks {
				// 所有分块接收完成
				if err := fm.assembleFile(task); err != nil {
					fm.failDownload(task, fmt.Errorf("failed to assemble file: %w", err))
					return
				}
				fm.completeDownload(task)
				return
			}
		}
	}
}

// requestFileData 请求文件数据
func (fm *FileManager) requestFileData(task *FileDownloadTask) error {
	request := map[string]interface{}{
		"type":      "file.request",
		"file_id":   task.FileID,
		"timestamp": time.Now().Unix(),
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	encrypted, err := fm.client.crypto.EncryptMessage(payload)
	if err != nil {
		return fmt.Errorf("failed to encrypt request: %w", err)
	}

	msg := &transport.Message{
		ID:        uuid.New().String(),
		Type:      transport.MessageTypeControl,
		SenderID:  fm.client.memberID,
		Payload:   encrypted,
		Timestamp: time.Now(),
	}

	return fm.client.transport.SendMessage(msg)
}

// assembleFile 组装文件
func (fm *FileManager) assembleFile(task *FileDownloadTask) error {
	fm.client.logger.Debug("[FileManager] Assembling file: %s", task.ID)

	// 1. 创建输出文件
	file, err := os.Create(task.SavePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 2. 按顺序写入分块
	task.chunksMutex.RLock()
	defer task.chunksMutex.RUnlock()

	hasher := sha256.New()
	for i := 0; i < task.TotalChunks; i++ {
		chunkData, ok := task.chunks[i]
		if !ok {
			return fmt.Errorf("missing chunk %d", i)
		}

		if _, err := file.Write(chunkData); err != nil {
			return fmt.Errorf("failed to write chunk %d: %w", i, err)
		}

		if _, err := hasher.Write(chunkData); err != nil {
			return fmt.Errorf("failed to hash chunk %d: %w", i, err)
		}
	}

	// 3. 验证哈希
	fileHash := hex.EncodeToString(hasher.Sum(nil))
	if fileHash != task.SHA256 {
		return fmt.Errorf("file hash mismatch: expected %s, got %s", task.SHA256, fileHash)
	}

	return nil
}

// failDownload 标记下载失败
func (fm *FileManager) failDownload(task *FileDownloadTask, err error) {
	task.Status = DownloadStatusFailed
	task.Error = err
	now := time.Now()
	task.EndTime = &now

	fm.statsMutex.Lock()
	fm.stats.FailedDownloads++
	fm.statsMutex.Unlock()

	fm.client.logger.Error("[FileManager] Download failed: %s - %v", task.ID, err)

	// 发布失败事件（使用系统错误事件）
	fm.client.eventBus.Publish(events.EventSystemError, events.SystemEvent{
		Type:    "file_download_failed",
		Message: fmt.Sprintf("Download failed for file %s: %v", task.FileID, err),
		Data: map[string]string{
			"file_id":    task.FileID,
			"channel_id": fm.client.config.ChannelID,
		},
	})
}

// completeDownload 标记下载完成
func (fm *FileManager) completeDownload(task *FileDownloadTask) {
	task.Status = DownloadStatusCompleted
	now := time.Now()
	task.EndTime = &now

	fm.statsMutex.Lock()
	fm.stats.SuccessfulDownloads++
	fm.stats.BytesDownloaded += task.Size
	fm.statsMutex.Unlock()

	fm.client.logger.Info("[FileManager] Download completed: %s (%d bytes)", task.ID, task.Size)

	// 发布完成事件
	fileRecord, _ := fm.client.fileRepo.GetByID(task.FileID)
	fm.client.eventBus.Publish(events.EventFileDownloaded, events.FileEvent{
		File:       fileRecord,
		ChannelID:  fm.client.config.ChannelID,
		UploaderID: "",
		Progress:   100,
	})
}

// handleFileReceived 处理文件接收事件
func (fm *FileManager) handleFileReceived(event *events.Event) {
	fileEvent, ok := event.Data.(events.FileEvent)
	if !ok {
		fm.client.logger.Warn("[FileManager] Invalid file event data")
		return
	}

	if fileEvent.File != nil {
		fm.client.logger.Debug("[FileManager] File received: %s", fileEvent.File.ID)
	}
	// TODO: 处理文件接收逻辑
}

// getOptimalChunkSize 获取最优分块大小
func (fm *FileManager) getOptimalChunkSize() int {
	mode := fm.client.transport.GetMode()
	switch mode {
	case models.TransportARP:
		return 1470 // 以太网 MTU
	case models.TransportHTTPS:
		return 64 * 1024 // 64KB
	case models.TransportMDNS:
		return 200 // 极小块
	default:
		return 32 * 1024 // 32KB
	}
}

// GetUploadTask 获取上传任务
func (fm *FileManager) GetUploadTask(taskID string) (*FileUploadTask, bool) {
	fm.uploadsMutex.RLock()
	defer fm.uploadsMutex.RUnlock()
	task, ok := fm.uploads[taskID]
	return task, ok
}

// GetDownloadTask 获取下载任务
func (fm *FileManager) GetDownloadTask(taskID string) (*FileDownloadTask, bool) {
	fm.downloadsMutex.RLock()
	defer fm.downloadsMutex.RUnlock()
	task, ok := fm.downloads[taskID]
	return task, ok
}

// GetStats 获取统计信息
func (fm *FileManager) GetStats() FileManagerStats {
	fm.statsMutex.RLock()
	defer fm.statsMutex.RUnlock()
	// 复制统计信息（避免复制锁）
	return FileManagerStats{
		TotalUploads:        fm.stats.TotalUploads,
		TotalDownloads:      fm.stats.TotalDownloads,
		SuccessfulUploads:   fm.stats.SuccessfulUploads,
		SuccessfulDownloads: fm.stats.SuccessfulDownloads,
		FailedUploads:       fm.stats.FailedUploads,
		FailedDownloads:     fm.stats.FailedDownloads,
		BytesUploaded:       fm.stats.BytesUploaded,
		BytesDownloaded:     fm.stats.BytesDownloaded,
	}
}

// GetProgress 获取上传进度
func (task *FileUploadTask) GetProgress() float64 {
	task.mutex.RLock()
	defer task.mutex.RUnlock()
	if task.TotalChunks == 0 {
		return 0
	}
	return float64(task.UploadedChunks) / float64(task.TotalChunks)
}

// GetProgress 获取下载进度
func (task *FileDownloadTask) GetProgress() float64 {
	task.chunksMutex.RLock()
	defer task.chunksMutex.RUnlock()
	if task.TotalChunks == 0 {
		return 0
	}
	return float64(len(task.chunks)) / float64(task.TotalChunks)
}

// ===== 断点续传功能 =====

// ResumeUpload 恢复上传任务
func (fm *FileManager) ResumeUpload(taskID string) error {
	fm.client.logger.Info("[FileManager] Resuming upload task: %s", taskID)

	// 1. 加载任务状态
	task, err := fm.loadUploadTaskState(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task state: %w", err)
	}

	// 2. 检查任务是否已完成
	if task.Status == models.UploadStatusCompleted {
		return fmt.Errorf("task already completed")
	}

	// 3. 打开文件
	file, err := os.Open(task.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	// 4. 注册任务
	fm.uploadsMutex.Lock()
	fm.uploads[task.ID] = task
	fm.uploadsMutex.Unlock()

	// 5. 继续执行上传
	go fm.executeUpload(task, file)

	return nil
}

// ResumeDownload 恢复下载任务
func (fm *FileManager) ResumeDownload(taskID string) error {
	fm.client.logger.Info("[FileManager] Resuming download task: %s", taskID)

	// 1. 加载任务状态
	task, err := fm.loadDownloadTaskState(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task state: %w", err)
	}

	// 2. 检查任务是否已完成
	if task.Status == DownloadStatusCompleted {
		return fmt.Errorf("task already completed")
	}

	// 3. 注册任务
	fm.downloadsMutex.Lock()
	fm.downloads[task.ID] = task
	fm.downloadsMutex.Unlock()

	// 4. 继续执行下载
	go fm.executeDownload(task)

	return nil
}

// pauseUpload 暂停上传（保存状态以便恢复）
func (fm *FileManager) pauseUpload(task *FileUploadTask, err error) {
	task.mutex.Lock()
	task.Status = models.UploadStatusPaused
	task.Error = err
	now := time.Now()
	task.EndTime = &now
	task.mutex.Unlock()

	// 保存任务状态
	fm.saveUploadTaskState(task)

	fm.client.logger.Warn("[FileManager] Upload paused: %s - %v", task.ID, err)

	// 发布暂停事件
	fm.client.eventBus.Publish(events.EventSystemError, events.SystemEvent{
		Type:    "file_upload_paused",
		Message: fmt.Sprintf("Upload paused for file %s: %v", task.ID, err),
		Data: map[string]string{
			"file_id":    task.ID,
			"channel_id": fm.client.config.ChannelID,
		},
	})
}

// pauseDownload 暂停下载（保存状态以便恢复）
func (fm *FileManager) pauseDownload(task *FileDownloadTask, err error) {
	task.Status = "paused"
	task.Error = err
	now := time.Now()
	task.EndTime = &now

	// 保存任务状态
	fm.saveDownloadTaskState(task)

	fm.client.logger.Warn("[FileManager] Download paused: %s - %v", task.ID, err)

	// 发布暂停事件
	fm.client.eventBus.Publish(events.EventSystemError, events.SystemEvent{
		Type:    "file_download_paused",
		Message: fmt.Sprintf("Download paused for file %s: %v", task.FileID, err),
		Data: map[string]string{
			"file_id":    task.FileID,
			"channel_id": fm.client.config.ChannelID,
		},
	})
}

// saveUploadTaskState 保存上传任务状态到数据库
func (fm *FileManager) saveUploadTaskState(task *FileUploadTask) {
	task.mutex.RLock()
	defer task.mutex.RUnlock()

	// 创建或更新文件记录
	file := &models.File{
		ID:             task.ID,
		MessageID:      task.ID, // 临时使用task ID
		ChannelID:      fm.client.config.ChannelID,
		SenderID:       fm.client.memberID,
		Filename:       task.Filename,
		OriginalName:   task.Filename,
		Size:           task.Size,
		MimeType:       task.MimeType,
		StorageType:    models.StorageFile,
		StoragePath:    task.FilePath,
		SHA256:         task.SHA256,
		ChunkSize:      task.ChunkSize,
		TotalChunks:    task.TotalChunks,
		UploadedChunks: task.UploadedChunks,
		UploadStatus:   task.Status,
		UploadedAt:     task.StartTime,
	}

	// 尝试更新，如果不存在则创建
	existing, err := fm.client.fileRepo.GetByID(task.ID)
	if err == nil && existing != nil {
		fm.client.fileRepo.Update(file)
	} else {
		fm.client.fileRepo.Create(file)
	}

	fm.client.logger.Debug("[FileManager] Saved upload task state: %s (%d/%d chunks)",
		task.ID, task.UploadedChunks, task.TotalChunks)
}

// loadUploadTaskState 从数据库加载上传任务状态
func (fm *FileManager) loadUploadTaskState(taskID string) (*FileUploadTask, error) {
	file, err := fm.client.fileRepo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load file record: %w", err)
	}

	// 重建任务
	task := &FileUploadTask{
		ID:             file.ID,
		FilePath:       file.StoragePath,
		Filename:       file.Filename,
		Size:           file.Size,
		MimeType:       file.MimeType,
		ChunkSize:      file.ChunkSize,
		TotalChunks:    file.TotalChunks,
		UploadedChunks: file.UploadedChunks,
		Status:         file.UploadStatus,
		SHA256:         file.SHA256,
		StartTime:      file.UploadedAt,
		chunkStatus:    make([]bool, file.TotalChunks),
	}

	// 重建分块状态（假设前N个分块已上传）
	for i := 0; i < file.UploadedChunks; i++ {
		task.chunkStatus[i] = true
	}

	fm.client.logger.Debug("[FileManager] Loaded upload task state: %s (%d/%d chunks)",
		taskID, task.UploadedChunks, task.TotalChunks)

	return task, nil
}

// deleteUploadTaskState 删除上传任务状态
func (fm *FileManager) deleteUploadTaskState(taskID string) {
	// 注意：这里不删除文件记录，因为上传完成后需要保留
	// 只是标记为不再需要恢复
	fm.client.logger.Debug("[FileManager] Upload task completed, state no longer needed: %s", taskID)
}

// saveDownloadTaskState 保存下载任务状态到内存（可扩展到数据库）
func (fm *FileManager) saveDownloadTaskState(task *FileDownloadTask) {
	// TODO: 可以扩展到数据库持久化
	fm.client.logger.Debug("[FileManager] Saved download task state: %s (%d/%d chunks)",
		task.ID, len(task.chunks), task.TotalChunks)
}

// loadDownloadTaskState 加载下载任务状态
func (fm *FileManager) loadDownloadTaskState(taskID string) (*FileDownloadTask, error) {
	// TODO: 从数据库加载
	fm.downloadsMutex.RLock()
	task, ok := fm.downloads[taskID]
	fm.downloadsMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// ListPendingUploads 列出所有待恢复的上传任务
func (fm *FileManager) ListPendingUploads() ([]*FileUploadTask, error) {
	// 从数据库查询所有未完成的上传任务
	// TODO: 需要在FileRepository添加查询方法
	fm.client.logger.Debug("[FileManager] Listing pending uploads...")

	tasks := make([]*FileUploadTask, 0)
	fm.uploadsMutex.RLock()
	for _, task := range fm.uploads {
		task.mutex.RLock()
		if task.Status == models.UploadStatusPaused || task.Status == models.UploadStatusUploading {
			tasks = append(tasks, task)
		}
		task.mutex.RUnlock()
	}
	fm.uploadsMutex.RUnlock()

	return tasks, nil
}

// CancelUpload 取消上传任务
func (fm *FileManager) CancelUpload(taskID string) error {
	fm.uploadsMutex.Lock()
	task, ok := fm.uploads[taskID]
	if !ok {
		fm.uploadsMutex.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}
	fm.uploadsMutex.Unlock()

	task.mutex.Lock()
	task.Status = models.UploadStatusFailed
	task.Error = fmt.Errorf("cancelled by user")
	task.mutex.Unlock()

	fm.deleteUploadTaskState(taskID)
	fm.client.logger.Info("[FileManager] Upload task cancelled: %s", taskID)

	return nil
}

// CancelDownload 取消下载任务
func (fm *FileManager) CancelDownload(taskID string) error {
	fm.downloadsMutex.Lock()
	task, ok := fm.downloads[taskID]
	if !ok {
		fm.downloadsMutex.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}
	fm.downloadsMutex.Unlock()

	task.Status = DownloadStatusFailed
	task.Error = fmt.Errorf("cancelled by user")

	fm.client.logger.Info("[FileManager] Download task cancelled: %s", taskID)

	return nil
}
