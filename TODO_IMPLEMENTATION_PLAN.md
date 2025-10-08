# CrossWire TODO 实现计划

## 📅 文档日期: 2025-10-07
## 🎯 目标: 系统性完成所有待办功能

---

## 📊 TODO 统计

| 模块 | 总数 | P0 | P1 | P2 | P3 |
|------|------|----|----|----|----|
| 前端 - 文件管理 | 5 | 2 | 2 | 1 | 0 |
| 前端 - 通知系统 | 3 | 1 | 2 | 0 | 0 |
| 前端 - 题目系统 | 2 | 0 | 1 | 1 | 0 |
| 后端 - 用户配置 | 2 | 1 | 1 | 0 | 0 |
| 后端 - 文件管理 | 1 | 0 | 1 | 0 | 0 |
| 后端 - 系统功能 | 3 | 0 | 0 | 3 | 0 |
| 客户端 - 文件传输 | 4 | 0 | 2 | 2 | 0 |
| 客户端 - 其他 | 6 | 0 | 0 | 2 | 4 |
| **总计** | **26** | **4** | **9** | **9** | **4** |

**优先级说明**:
- P0: 阻塞性功能，必须立即实现
- P1: 重要功能，影响用户体验
- P2: 优化功能，可延后实现
- P3: 可选功能，有更好无害

---

## 🔴 P0 问题（阻塞性）- 4个

### 1. 前端文件上传实现 (P0)

**位置**: `frontend/src/components/MessageInput.vue:260`

**当前状态**: 
```javascript
const handleFileUpload = (file) => {
  console.log('Uploading file:', file.name)
  message.info(`正在上传文件: ${file.name}`)
  // TODO: 实现文件上传
  return false // 阻止自动上传
}
```

**实现方案**:
```javascript
import { uploadFile } from '@/api/app'

const handleFileUpload = async (file) => {
  // 验证文件
  if (file.size > 100 * 1024 * 1024) {
    message.error('文件大小不能超过 100MB')
    return false
  }

  try {
    message.loading({ content: `正在上传 ${file.name}...`, key: 'upload', duration: 0 })
    
    // 调用后端API上传
    await uploadFile({
      name: file.name,
      size: file.size,
      mime_type: file.type,
      path: file.path || ''
    })
    
    message.success({ content: `${file.name} 上传成功`, key: 'upload' })
  } catch (error) {
    message.error({ content: `上传失败: ${error.message}`, key: 'upload' })
  }
  
  return false // 阻止默认上传
}
```

**依赖**: 
- ✅ `uploadFile` API已存在
- ✅ 后端上传逻辑已实现

**预计工作量**: 0.5小时

---

### 2. 前端文件预览实现 (P0)

**位置**: `frontend/src/components/FileManager.vue:502`

**当前状态**: 
```javascript
const previewFile = (file) => {
  message.info(`正在打开预览: ${file.name}`)
  // TODO: 实现文件预览
}
```

**实现方案**:
```javascript
import FilePreview from './FilePreview.vue'

const showPreview = ref(false)
const previewingFile = ref(null)

const previewFile = async (file) => {
  // 根据文件类型判断是否支持预览
  const previewableTypes = ['image', 'pdf', 'text', 'code']
  
  if (!previewableTypes.includes(file.type)) {
    message.warning('该文件类型不支持预览，请下载后查看')
    return
  }

  // 对于需要下载内容的文件（如文本），先下载
  if (file.type === 'text' || file.type === 'code') {
    try {
      const content = await downloadFileContent(file.id)
      previewingFile.value = { ...file, content }
    } catch (error) {
      message.error('加载文件内容失败')
      return
    }
  } else {
    previewingFile.value = file
  }

  showPreview.value = true
}

// 模板中添加预览组件
// <FilePreview 
//   v-if="showPreview" 
//   :file="previewingFile" 
//   @close="showPreview = false" 
// />
```

**需要新增API**:
```javascript
// frontend/src/api/app.js
export async function getFileContent(fileId) {
  const res = await App.GetFileContent(fileId)
  return unwrap(res)
}
```

**预计工作量**: 1.5小时

---

### 3. 通知系统实现 (P0)

**位置**: `frontend/src/components/NotificationCenter.vue:166-179`

**当前状态**: 
```javascript
// TODO: 实现通知系统，通过WebSocket或EventBus接收实时通知
// TODO: 订阅事件总线的通知事件
```

**实现方案**:
```javascript
import { EventsOn } from '@/wailsjs/runtime/runtime'

const notifications = ref([])

onMounted(() => {
  // 订阅消息提及事件
  EventsOn('message:mention', (data) => {
    addNotification({
      id: Date.now().toString(),
      type: 'mention',
      title: `${data.sender_name} 提到了你`,
      description: data.content,
      timestamp: Date.now(),
      read: false,
      link: `/channel/${data.channel_id}`
    })
  })

  // 订阅题目分配事件
  EventsOn('challenge:assigned', (data) => {
    addNotification({
      id: Date.now().toString(),
      type: 'challenge',
      title: '新题目分配',
      description: `管理员为你分配了题目: ${data.challenge_title}`,
      timestamp: Date.now(),
      read: false,
      link: `/challenge/${data.challenge_id}`
    })
  })

  // 订阅Flag提交结果
  EventsOn('challenge:submitted', (data) => {
    addNotification({
      id: Date.now().toString(),
      type: 'flag',
      title: 'Flag已提交',
      description: `题目 "${data.challenge_title}" 的Flag已提交`,
      timestamp: Date.now(),
      read: false
    })
  })

  // 订阅系统通知
  EventsOn('system:notification', (data) => {
    addNotification({
      id: Date.now().toString(),
      type: 'system',
      title: data.title || '系统通知',
      description: data.message,
      timestamp: Date.now(),
      read: false
    })
  })
})

const addNotification = (notification) => {
  notifications.value.unshift(notification)
  
  // 限制通知数量
  if (notifications.value.length > 50) {
    notifications.value = notifications.value.slice(0, 50)
  }
  
  // 存储到LocalStorage（可选）
  localStorage.setItem('notifications', JSON.stringify(notifications.value))
}

// 从LocalStorage加载历史通知
const loadNotifications = () => {
  const stored = localStorage.getItem('notifications')
  if (stored) {
    try {
      notifications.value = JSON.parse(stored)
    } catch (error) {
      console.error('Failed to load notifications:', error)
    }
  }
}

onMounted(() => {
  loadNotifications()
  // ... 其他事件订阅
})
```

**后端需要添加事件发送**:
```go
// internal/app/event_handler.go 中添加
func (a *App) handleMessageWithMention(msg *models.Message) {
    // 解析@提及
    for _, mention := range msg.Mentions {
        runtime.EventsEmit(a.ctx, "message:mention", map[string]interface{}{
            "sender_name":  msg.SenderNickname,
            "content":      msg.ContentText,
            "channel_id":   msg.ChannelID,
            "message_id":   msg.ID,
            "mentioned_id": mention.UserID,
        })
    }
}
```

**预计工作量**: 3小时

---

### 4. 用户配置持久化 (P0)

**位置**: 
- `internal/app/app.go:64` - 加载用户配置
- `internal/app/app.go:128` - 保存用户配置

**当前状态**: 
```go
// TODO: 从数据库加载用户配置
// TODO: 将用户配置保存到数据库
```

**实现方案**:

#### 4.1 创建用户配置表

```go
// internal/models/user_config.go
type UserConfig struct {
    ID            string                 `gorm:"primaryKey" json:"id"`
    MemberID      string                 `gorm:"index;not null" json:"member_id"`
    Theme         string                 `json:"theme"`
    Language      string                 `json:"language"`
    Notifications NotificationSettings   `gorm:"type:json" json:"notifications"`
    CustomSettings map[string]interface{} `gorm:"type:json" json:"custom_settings"`
    CreatedAt     time.Time              `json:"created_at"`
    UpdatedAt     time.Time              `json:"updated_at"`
}
```

#### 4.2 实现加载逻辑

```go
// internal/app/app.go
func loadUserProfile(db *storage.Database, memberID string) *UserProfile {
    // 从数据库加载用户配置
    config, err := db.UserConfigRepo().GetByMemberID(memberID)
    if err != nil {
        // 如果不存在，返回默认配置
        return getDefaultUserProfile()
    }

    // 从Member表加载基本信息
    member, err := db.MemberRepo().GetByID(memberID)
    if err != nil {
        return getDefaultUserProfile()
    }

    return &UserProfile{
        Nickname:      member.Nickname,
        Avatar:        member.Avatar,
        Status:        string(member.Status),
        Theme:         config.Theme,
        Language:      config.Language,
        Notifications: config.Notifications,
    }
}

func getDefaultUserProfile() *UserProfile {
    return &UserProfile{
        Nickname: "User",
        Avatar:   "",
        Status:   "online",
        Theme:    "light",
        Language: "zh-CN",
        Notifications: NotificationSettings{
            Enabled:     true,
            Sound:       true,
            Desktop:     true,
            MentionOnly: false,
        },
    }
}
```

#### 4.3 实现保存逻辑

```go
// internal/app/app.go - Shutdown方法中
func (a *App) Shutdown(ctx context.Context) {
    // ... 其他关闭逻辑 ...

    // 保存用户配置
    if a.userProfile != nil {
        var memberID string
        if a.mode == ModeServer && a.server != nil {
            memberID = "server"
        } else if a.mode == ModeClient && a.client != nil {
            memberID = a.client.GetMemberID()
        }

        if memberID != "" {
            config := &models.UserConfig{
                MemberID:      memberID,
                Theme:         a.userProfile.Theme,
                Language:      a.userProfile.Language,
                Notifications: a.userProfile.Notifications,
                UpdatedAt:     time.Now(),
            }

            // 尝试更新，如果不存在则创建
            existing, _ := a.db.UserConfigRepo().GetByMemberID(memberID)
            if existing != nil {
                config.ID = existing.ID
                a.db.UserConfigRepo().Update(config)
            } else {
                config.ID = uuid.New().String()
                config.CreatedAt = time.Now()
                a.db.UserConfigRepo().Create(config)
            }
        }
    }

    // ... 其他关闭逻辑 ...
}
```

#### 4.4 需要新增Repository

```go
// internal/storage/user_config_repository.go
type UserConfigRepository struct {
    db *Database
}

func NewUserConfigRepository(db *Database) *UserConfigRepository {
    return &UserConfigRepository{db: db}
}

func (r *UserConfigRepository) GetByMemberID(memberID string) (*models.UserConfig, error) {
    var config models.UserConfig
    err := r.db.GetChannelDB().
        Where("member_id = ?", memberID).
        First(&config).Error
    return &config, err
}

func (r *UserConfigRepository) Create(config *models.UserConfig) error {
    return r.db.GetChannelDB().Create(config).Error
}

func (r *UserConfigRepository) Update(config *models.UserConfig) error {
    return r.db.GetChannelDB().Save(config).Error
}
```

**预计工作量**: 2小时

---

## 🟡 P1 问题（重要功能）- 9个

### 5. 文件分享链接生成 (P1)

**位置**: `frontend/src/components/FileManager.vue:507`

**实现方案**:
```javascript
const shareFile = async (file) => {
  try {
    // 生成分享链接
    const shareLink = await generateShareLink(file.id, {
      expires: 24 * 60 * 60, // 24小时过期
      password: null // 可选密码保护
    })
    
    // 复制到剪贴板
    await navigator.clipboard.writeText(shareLink.url)
    
    message.success({
      content: '分享链接已复制到剪贴板',
      description: `链接将在 ${shareLink.expiresIn} 小时后过期`
    })
  } catch (error) {
    message.error('生成分享链接失败')
  }
}
```

**需要后端API**:
```go
// internal/app/file_api.go
func (a *App) GenerateShareLink(fileID string, expiresIn int64, password string) Response {
    // 生成分享token
    token := uuid.New().String()
    
    shareLink := &models.ShareLink{
        ID:        uuid.New().String(),
        FileID:    fileID,
        Token:     token,
        CreatedBy: currentUserID,
        ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
        Password:  password, // 可选
    }
    
    if err := a.db.ShareLinkRepo().Create(shareLink); err != nil {
        return NewErrorResponse("create_error", "创建分享链接失败", err.Error())
    }
    
    // 生成URL
    url := fmt.Sprintf("crosswire://share/%s", token)
    
    return NewSuccessResponse(map[string]interface{}{
        "url":        url,
        "token":      token,
        "expires_at": shareLink.ExpiresAt.Unix(),
        "expires_in": expiresIn / 3600, // 小时
    })
}
```

**预计工作量**: 2小时

---

### 6. 批量文件下载 (P1)

**位置**: `frontend/src/components/FileManager.vue:548`

**实现方案**:
```javascript
const batchDownload = async () => {
  if (selectedFiles.value.length === 0) {
    message.warning('请先选择要下载的文件')
    return
  }

  message.loading({ 
    content: `正在下载 ${selectedFiles.value.length} 个文件...`, 
    key: 'batch-download',
    duration: 0
  })

  let successCount = 0
  let failCount = 0

  // 并发下载（限制并发数为3）
  const downloadPromises = []
  const concurrency = 3

  for (let i = 0; i < selectedFiles.value.length; i += concurrency) {
    const batch = selectedFiles.value.slice(i, i + concurrency)
    const batchPromises = batch.map(async file => {
      try {
        await downloadFileAPI(file.id)
        successCount++
      } catch (error) {
        failCount++
        console.error(`Failed to download ${file.name}:`, error)
      }
    })
    await Promise.all(batchPromises)
  }

  message.success({
    content: `下载完成: ${successCount} 成功，${failCount} 失败`,
    key: 'batch-download'
  })

  selectedFiles.value = []
}
```

**预计工作量**: 1小时

---

### 7. 题目进度API集成 (P1)

**位置**: `frontend/src/components/Challenge/ChallengeRoom.vue:281`

**实现方案**:
```javascript
const loadMembers = async () => {
  loading.value = true
  try {
    const members = await getMembers()
    
    // 并发获取每个成员的进度
    const memberProgressPromises = members.map(async (m) => {
      try {
        const progress = await getChallengeProgress(
          props.challenge.id,
          m.id || m.ID
        )
        return {
          id: m.id || m.ID,
          name: m.nickname || m.Nickname || 'Unknown',
          online: m.status === 'online',
          progress: progress?.progress || 0
        }
      } catch (error) {
        return {
          id: m.id || m.ID,
          name: m.nickname || m.Nickname || 'Unknown',
          online: m.status === 'online',
          progress: 0
        }
      }
    })

    roomMembers.value = await Promise.all(memberProgressPromises)
  } catch (error) {
    console.error('Failed to load members:', error)
  } finally {
    loading.value = false
  }
}
```

**需要新增API**:
```javascript
// frontend/src/api/app.js
export async function getChallengeProgress(challengeId, memberId) {
  const res = await App.GetChallengeProgress(challengeId, memberId)
  return unwrap(res)
}
```

**预计工作量**: 1小时

---

### 8. 消息Reactions加载 (P1)

**位置**: `internal/app/message_api.go:406`

**实现方案**:
```go
// internal/app/message_api.go - messageToDTO方法中
func (a *App) messageToDTO(message *models.Message) *MessageDTO {
    // ... 现有代码 ...

    // 加载reactions
    reactions := make([]MessageReaction, 0)
    dbReactions, err := a.db.MessageRepo().GetReactions(message.ID)
    if err == nil && len(dbReactions) > 0 {
        // 统计每个emoji的数量和用户列表
        emojiMap := make(map[string]*MessageReaction)
        
        for _, r := range dbReactions {
            if existing, ok := emojiMap[r.Emoji]; ok {
                existing.Count++
                existing.Users = append(existing.Users, r.UserID)
            } else {
                emojiMap[r.Emoji] = &MessageReaction{
                    Emoji: r.Emoji,
                    Count: 1,
                    Users: []string{r.UserID},
                }
            }
        }
        
        // 转换为数组
        for _, reaction := range emojiMap {
            reactions = append(reactions, *reaction)
        }
    }

    return &MessageDTO{
        // ... 现有字段 ...
        Reactions: reactions,
    }
}
```

**预计工作量**: 0.5小时

---

### 9. 提示解锁状态判断 (P1)

**位置**: `internal/app/challenge_api.go:354`

**实现方案**:
```go
// GetChallenge 方法中
func (a *App) GetChallenge(challengeID string) Response {
    // ... 获取challenge和hints ...

    // 获取当前用户ID
    var currentUserID string
    if a.mode == ModeServer {
        currentUserID = "server"
    } else if a.mode == ModeClient && a.client != nil {
        currentUserID = a.client.GetMemberID()
    }

    // 查询该用户已解锁的提示
    unlockedHints := make(map[string]bool)
    if currentUserID != "" {
        hints, err := a.db.ChallengeRepo().GetUnlockedHints(challengeID, currentUserID)
        if err == nil {
            for _, hintID := range hints {
                unlockedHints[hintID] = true
            }
        }
    }

    hintDTOs := make([]*HintDTO, 0)
    for _, hint := range hints {
        hintDTOs = append(hintDTOs, &HintDTO{
            ID:         hint.ID,
            ChallengeID: hint.ChallengeID,
            Content:    hint.Content,
            Cost:       hint.Cost,
            IsUnlocked: unlockedHints[hint.ID], // 根据当前用户判断
        })
    }

    // ...
}
```

**需要新增Repository方法**:
```go
// internal/storage/challenge_repository.go
func (r *ChallengeRepository) GetUnlockedHints(challengeID, userID string) ([]string, error) {
    var hints []models.ChallengeHint
    err := r.db.GetChannelDB().
        Where("challenge_id = ? AND unlocked_by LIKE ?", challengeID, "%"+userID+"%").
        Find(&hints).Error
    
    if err != nil {
        return nil, err
    }
    
    hintIDs := make([]string, len(hints))
    for i, h := range hints {
        hintIDs[i] = h.ID
    }
    
    return hintIDs, nil
}
```

**预计工作量**: 1小时

---

### 10. 文件接收逻辑 (P1)

**位置**: `internal/client/file_manager.go:691`

**实现方案**:
```go
// handleFileReceived 处理文件接收事件
func (fm *FileManager) handleFileReceived(event *events.Event) {
    fileEvent, ok := event.Data.(events.FileEvent)
    if !ok {
        fm.client.logger.Warn("[FileManager] Invalid file event data")
        return
    }

    if fileEvent.File != nil {
        fm.client.logger.Debug("[FileManager] File received: %s", fileEvent.File.ID)
        
        // 保存文件元数据到本地数据库
        if err := fm.client.fileRepo.Create(fileEvent.File); err != nil {
            fm.client.logger.Error("[FileManager] Failed to save file metadata: %v", err)
        }
        
        // 发布文件接收事件到前端
        fm.client.eventBus.Publish(events.EventFileProgress, events.FileEvent{
            File:       fileEvent.File,
            ChannelID:  fileEvent.ChannelID,
            UploaderID: fileEvent.UploaderID,
            Progress:   100,
        })
        
        fm.client.logger.Info("[FileManager] File metadata saved: %s", fileEvent.File.Filename)
    }
}
```

**预计工作量**: 1小时

---

### 11. 下载任务持久化 (P1)

**位置**: `internal/client/file_manager.go:962`

**实现方案**:
```go
// saveDownloadTaskState 保存下载任务状态到数据库
func (fm *FileManager) saveDownloadTaskState(task *FileDownloadTask) {
    // 序列化chunks（只保存已接收的分块索引）
    receivedChunks := make([]int, 0)
    task.chunksMutex.RLock()
    for chunkIndex := range task.chunks {
        receivedChunks = append(receivedChunks, chunkIndex)
    }
    task.chunksMutex.RUnlock()

    // 创建持久化记录
    downloadRecord := &models.DownloadTask{
        ID:             task.ID,
        FileID:         task.FileID,
        Filename:       task.Filename,
        SavePath:       task.SavePath,
        TotalChunks:    task.TotalChunks,
        ReceivedChunks: receivedChunks, // JSON数组
        Status:         string(task.Status),
        CreatedAt:      task.StartTime,
        UpdatedAt:      time.Now(),
    }

    // 保存到数据库
    existing, _ := fm.client.downloadTaskRepo.GetByID(task.ID)
    if existing != nil {
        fm.client.downloadTaskRepo.Update(downloadRecord)
    } else {
        fm.client.downloadTaskRepo.Create(downloadRecord)
    }

    fm.client.logger.Debug("[FileManager] Saved download task state to DB: %s (%d/%d chunks)",
        task.ID, len(receivedChunks), task.TotalChunks)
}

// loadDownloadTaskState 从数据库加载下载任务状态
func (fm *FileManager) loadDownloadTaskState(taskID string) (*FileDownloadTask, error) {
    record, err := fm.client.downloadTaskRepo.GetByID(taskID)
    if err != nil {
        return nil, fmt.Errorf("failed to load download task: %w", err)
    }

    // 重建任务
    task := &FileDownloadTask{
        ID:             record.ID,
        FileID:         record.FileID,
        Filename:       record.Filename,
        SavePath:       record.SavePath,
        TotalChunks:    record.TotalChunks,
        ReceivedChunks: len(record.ReceivedChunks),
        Status:         DownloadStatus(record.Status),
        StartTime:      record.CreatedAt,
        chunks:         make(map[int][]byte),
    }

    // TODO: 重新加载已接收的分块数据（如果保存了的话）
    // 这里可以选择不保存分块数据，只保存进度，断点续传时重新下载

    fm.client.logger.Debug("[FileManager] Loaded download task from DB: %s", taskID)
    return task, nil
}
```

**需要新增Model和Repository**:
```go
// internal/models/download_task.go
type DownloadTask struct {
    ID             string `gorm:"primaryKey"`
    FileID         string `gorm:"index"`
    Filename       string
    SavePath       string
    TotalChunks    int
    ReceivedChunks []int `gorm:"type:json"` // 已接收的分块索引
    Status         string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

**预计工作量**: 2小时

---

### 12. 查询待恢复上传任务 (P1)

**位置**: `internal/client/file_manager.go:984`

**实现方案**:
```go
// ListPendingUploads 列出所有待恢复的上传任务
func (fm *FileManager) ListPendingUploads() ([]*FileUploadTask, error) {
    fm.client.logger.Debug("[FileManager] Listing pending uploads from database...")

    // 从数据库查询所有未完成的文件记录
    files, err := fm.client.fileRepo.GetPendingUploads(fm.client.config.ChannelID)
    if err != nil {
        return nil, fmt.Errorf("failed to query pending uploads: %w", err)
    }

    tasks := make([]*FileUploadTask, 0, len(files))
    for _, file := range files {
        // 只恢复状态为Paused或Uploading的任务
        if file.UploadStatus != models.UploadStatusPaused && 
           file.UploadStatus != models.UploadStatusUploading {
            continue
        }

        // 检查文件是否仍然存在
        if _, err := os.Stat(file.StoragePath); os.IsNotExist(err) {
            fm.client.logger.Warn("[FileManager] File not found for pending upload: %s", file.StoragePath)
            continue
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

        // 重建分块状态
        for i := 0; i < file.UploadedChunks; i++ {
            task.chunkStatus[i] = true
        }

        tasks = append(tasks, task)
    }

    fm.client.logger.Info("[FileManager] Found %d pending upload tasks", len(tasks))
    return tasks, nil
}
```

**需要新增Repository方法**:
```go
// internal/storage/file_repository.go
func (r *FileRepository) GetPendingUploads(channelID string) ([]*models.File, error) {
    var files []*models.File
    err := r.db.GetChannelDB().
        Where("channel_id = ? AND upload_status IN ?", 
            channelID, 
            []models.UploadStatus{
                models.UploadStatusPaused,
                models.UploadStatusUploading,
            }).
        Find(&files).Error
    
    return files, err
}
```

**预计工作量**: 1.5小时

---

## 🟢 P2 问题（优化功能）- 9个

### 13. 最近频道列表 (P2)

**位置**: `internal/app/system_api.go:265`

**实现方案**:
```go
func (a *App) GetRecentChannels() Response {
    // 从数据库加载最近访问的频道
    var channels []*models.RecentChannel
    
    // 获取当前用户ID
    var userID string
    if a.mode == ModeServer {
        userID = "server"
    } else if a.mode == ModeClient && a.client != nil {
        userID = a.client.GetMemberID()
    }
    
    if userID == "" {
        return NewSuccessResponse(make([]*RecentChannel, 0))
    }
    
    // 查询最近访问记录（按最后访问时间排序）
    err := a.db.GetGlobalDB().
        Where("user_id = ?", userID).
        Order("last_accessed_at DESC").
        Limit(10).
        Find(&channels).Error
    
    if err != nil {
        a.logger.Warn("Failed to load recent channels: %v", err)
        return NewSuccessResponse(make([]*RecentChannel, 0))
    }
    
    // 转换为DTO
    recentChannels := make([]*RecentChannel, len(channels))
    for i, ch := range channels {
        recentChannels[i] = &RecentChannel{
            ID:            ch.ChannelID,
            Name:          ch.ChannelName,
            LastMessage:   ch.LastMessage,
            LastMessageAt: ch.LastMessageAt.Unix(),
            UnreadCount:   ch.UnreadCount,
        }
    }
    
    return NewSuccessResponse(recentChannels)
}

// UpdateRecentChannel 更新最近频道（在加入/发送消息时调用）
func (a *App) UpdateRecentChannel(channelID, channelName, lastMessage string) {
    // ... 实现更新逻辑
}
```

**需要新增Model**:
```go
type RecentChannel struct {
    ID              string    `gorm:"primaryKey"`
    UserID          string    `gorm:"index"`
    ChannelID       string
    ChannelName     string
    LastMessage     string
    LastMessageAt   time.Time
    LastAccessedAt  time.Time
    UnreadCount     int
    CreatedAt       time.Time
}
```

**预计工作量**: 1.5小时

---

### 14. 日志系统 (P2)

**位置**: 
- `internal/app/system_api.go:390` - 获取日志
- `internal/app/system_api.go:398` - 清空日志

**实现方案**:
```go
// GetLogs 获取日志
func (a *App) GetLogs(limit int) Response {
    // 从日志文件读取最近的日志
    logs, err := a.logger.GetRecentLogs(limit)
    if err != nil {
        return NewErrorResponse("read_error", "读取日志失败", err.Error())
    }
    
    // 转换为前端格式
    logEntries := make([]map[string]interface{}, len(logs))
    for i, log := range logs {
        logEntries[i] = map[string]interface{}{
            "timestamp": log.Timestamp.Unix(),
            "level":     log.Level,
            "message":   log.Message,
            "source":    log.Source,
        }
    }
    
    return NewSuccessResponse(logEntries)
}

// ClearLogs 清空日志
func (a *App) ClearLogs() Response {
    if err := a.logger.ClearLogs(); err != nil {
        return NewErrorResponse("clear_error", "清空日志失败", err.Error())
    }
    
    return NewSuccessResponse(map[string]interface{}{
        "message": "日志已清空",
    })
}
```

**需要在Logger中添加方法**:
```go
// internal/utils/logger.go
func (l *Logger) GetRecentLogs(limit int) ([]*LogEntry, error) {
    // 读取日志文件并解析
    // ...
}

func (l *Logger) ClearLogs() error {
    // 清空或轮转日志文件
    // ...
}
```

**预计工作量**: 2小时

---

### 15. 数据导入功能 (P2)

**位置**: `internal/app/system_api.go:355`

**实现方案**:
```go
func (a *App) ImportData(importPath string) Response {
    a.logger.Info("Importing data from: %s", importPath)

    // 打开ZIP文件
    zipReader, err := zip.OpenReader(importPath)
    if err != nil {
        return NewErrorResponse("file_error", "打开导入文件失败", err.Error())
    }
    defer zipReader.Close()

    // 读取并导入数据
    importedCount := 0
    
    for _, file := range zipReader.File {
        switch file.Name {
        case "messages.json":
            if count, err := a.importMessages(file); err == nil {
                importedCount += count
            } else {
                a.logger.Warn("Failed to import messages: %v", err)
            }
            
        case "files.json":
            if count, err := a.importFiles(file); err == nil {
                importedCount += count
            } else {
                a.logger.Warn("Failed to import files: %v", err)
            }
            
        case "members.json":
            if count, err := a.importMembers(file); err == nil {
                importedCount += count
            } else {
                a.logger.Warn("Failed to import members: %v", err)
            }
            
        case "challenges.json":
            if count, err := a.importChallenges(file); err == nil {
                importedCount += count
            } else {
                a.logger.Warn("Failed to import challenges: %v", err)
            }
        }
    }

    a.logger.Info("Data import completed: %d items imported", importedCount)

    return NewSuccessResponse(map[string]interface{}{
        "message": "数据导入成功",
        "count":   importedCount,
    })
}

// importMessages 导入消息（处理ID冲突）
func (a *App) importMessages(file *zip.File) (int, error) {
    // 读取JSON
    // 检查ID是否冲突
    // 批量插入或更新
    return 0, nil
}
```

**预计工作量**: 3小时

---

### 16. 技能标签解析 (P2)

**位置**: `frontend/src/components/Challenge/ChallengeAssign.vue:128`

**实现方案**:
```javascript
const loadMembers = async () => {
  loading.value = true
  try {
    const members = await getMembers()
    if (Array.isArray(members)) {
      availableMembers.value = members.map(m => ({
        id: m.id || m.ID,
        name: m.nickname || m.Nickname || 'Unknown',
        role: m.role || m.Role || 'member',
        skills: parseSkills(m.skills || m.Skills || []) // 解析技能标签
      }))
    }
  } catch (error) {
    console.error('Failed to load members:', error)
  } finally {
    loading.value = false
  }
}

const parseSkills = (skills) => {
  if (Array.isArray(skills) && skills.length > 0) {
    return skills.map(skill => {
      if (typeof skill === 'string') {
        // 简单字符串格式
        return {
          category: skill,
          level: 2,
          display: skill
        }
      } else if (typeof skill === 'object') {
        // 完整对象格式
        return {
          category: skill.category || skill.Category,
          level: skill.level || skill.Level || 2,
          display: `${skill.category} (${getLevelName(skill.level)})`
        }
      }
      return null
    }).filter(Boolean)
  }
  return []
}

const getLevelName = (level) => {
  const levels = {
    1: '初级',
    2: '中级',
    3: '高级',
    4: '专家'
  }
  return levels[level] || '未知'
}
```

**预计工作量**: 0.5小时

---

### 17-21. 客户端待实现功能 (P2)

这些是客户端的一些优化功能，包括：
- 生成临时用户密钥对
- 等待加入响应
- 根据msgType构造MessageContent
- 从数据库加载challenges
- 子频道同步
- 成员信息解析缓存
- 离线消息回复支持
- Transport元数据解析

**预计总工作量**: 6小时

---

## ⚪ P3 问题（可选功能）- 4个

### 22-25. 客户端文档中的TODO (P3)

这些是文档中标记的未来功能，包括：
- 断点续传增强
- 分块下载重试
- 带宽限制
- 文件缓存

**预计总工作量**: 8小时（可根据需求选择实现）

---

## 📅 实施时间表

### 第一阶段（1-2天）- P0问题
1. 文件上传实现 (0.5h)
2. 文件预览实现 (1.5h)
3. 通知系统实现 (3h)
4. 用户配置持久化 (2h)

**总计**: 7小时

### 第二阶段（3-5天）- P1问题
5. 文件分享 (2h)
6. 批量下载 (1h)
7. 题目进度API (1h)
8. Reactions加载 (0.5h)
9. 提示解锁状态 (1h)
10. 文件接收逻辑 (1h)
11. 下载任务持久化 (2h)
12. 待恢复上传任务 (1.5h)

**总计**: 10小时

### 第三阶段（6-10天）- P2问题
13-21. 各项优化功能

**总计**: 约18小时

### 第四阶段（可选）- P3问题
22-25. 可选增强功能

**总计**: 约8小时

---

## 🎯 优先级建议

**立即实施**:
1. P0 #1: 文件上传（核心功能）
2. P0 #3: 通知系统（用户体验）
3. P0 #4: 用户配置持久化（数据完整性）

**本周内完成**:
4. P0 #2: 文件预览
5. P1 #6: 批量下载
6. P1 #8: Reactions加载

**下周完成**:
7. P1 #5: 文件分享
8. P1 #11-12: 文件传输持久化

**后续迭代**:
- P2问题根据用户反馈决定优先级
- P3问题作为长期优化目标

---

## 📝 备注

1. **依赖关系**: 某些TODO之间存在依赖，需要按顺序实现
2. **测试要求**: 每个功能实现后需要编写单元测试和集成测试
3. **文档更新**: 实现后需要更新相应的README和API文档
4. **代码审查**: 建议每个P0/P1功能都进行代码审查
5. **性能考虑**: P2优化功能需要进行性能测试

---

**文档维护**: 实现一个TODO后，请在此文档中标记为已完成，并更新实际工作量。

**最后更新**: 2025-10-07
