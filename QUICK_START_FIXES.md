# CrossWire 快速修复指南

> 立即可执行的修复任务
> 更新时间: 2025-10-07

---

## ✅ 已完成 (今天)

### 1. 服务端用户资料更新
- **问题**: Server用户修改资料后不生效
- **原因**: UpdateUserProfile只更新内存，未保存到数据库
- **修复**: 
  - ✅ 扩展UserProfile类型添加Email/Bio/Skills字段
  - ✅ 重写UpdateUserProfile保存到Member表
  - ✅ 支持服务端和客户端两种模式
  - ✅ 前端UserProfile组件集成API并添加调试日志
- **文件**: 
  - `internal/app/types.go`
  - `internal/app/system_api.go`
  - `frontend/src/components/UserProfile.vue`
  - `frontend/src/api/app.js`

### 2. 时间字段统一
- **问题**: `time.Time`字段使用`serializer:unixtime`导致类型不匹配
- **修复**: 
  - ✅ 移除所有`serializer:unixtime`标签
  - ✅ 使用GORM默认的DATETIME格式
  - ✅ 修复所有`&time.Now()`为`time.Now()`
  - ✅ 修复所有`t == nil`为`t.IsZero()`
- **影响文件**:
  - `internal/models/*.go` (所有模型)
  - `internal/storage/message_repository.go`
  - `internal/client/challenge_manager.go`
  - `internal/server/challenge_manager.go`

---

## 🔥 紧急 (本周内) - 3个任务

### 任务1: 验证题目聊天室消息隔离 ⏱️ 2小时

**问题**: 需要确认子频道消息不会泄露到主频道

**检查步骤**:
```bash
# 1. 启动服务端
wails dev

# 2. 创建题目
前端: 题目管理 → 创建题目 "Web-100"

# 3. 查看数据库
sqlite3 data/channels/<channel-id>.db
SELECT id, parent_channel_id, name FROM channels;
# 应该看到子频道的parent_channel_id指向主频道

# 4. 在子频道发送消息
前端: 题目频道 → Web-100 → 发送消息 "测试子频道消息"

# 5. 在主频道查看
前端: 主聊天室 → 检查消息列表
# 子频道消息不应出现在主频道

# 6. 验证数据库
SELECT id, channel_id, content FROM messages WHERE content LIKE '%测试%';
# 确认channel_id是子频道ID
```

**如果发现问题，修复方案**:
```go
// 文件: internal/storage/message_repository.go
func (r *MessageRepository) GetByChannelID(channelID string, limit, offset int) ([]*models.Message, error) {
    var messages []*models.Message
    err := r.db.GetChannelDB().
        Where("channel_id = ?", channelID).  // 严格匹配，不查询子频道
        Where("deleted = ?", false).
        Order("timestamp DESC").
        Limit(limit).
        Offset(offset).
        Find(&messages).Error
    return messages, err
}
```

**验证文件**: `internal/storage/message_repository.go`

---

### 任务2: 测试消息删除功能 ⏱️ 3小时

**现状**: `DeleteMessage` API已存在但未测试

**测试清单**:

```markdown
□ 1. 权限测试
   - [ ] 客户端普通成员尝试删除 → 应返回"permission_denied"
   - [ ] 服务端管理员删除 → 应成功
   - [ ] 发送者删除自己的消息 → 根据需求决定是否允许

□ 2. 软删除 vs 硬删除
   - [ ] 检查代码使用的是软删除(deleted=true)还是硬删除
   - [ ] 建议使用软删除保留审计记录

□ 3. 级联删除
   - [ ] 删除消息后reactions是否自动清理
   - [ ] 删除消息后pins是否自动清理

□ 4. 前端集成
   - [ ] ChatView添加删除按钮（悬停在消息上显示）
   - [ ] 仅管理员可见删除按钮
   - [ ] 点击后弹出确认对话框
   - [ ] 删除后消息列表自动更新

□ 5. 事件通知
   - [ ] 监听message:deleted事件
   - [ ] 所有客户端同步移除消息
```

**前端集成代码示例**:
```vue
<!-- frontend/src/components/MessageBubble.vue -->
<template>
  <div class="message-bubble">
    <!-- 消息内容 -->
    <div class="message-content">{{ message.content }}</div>
    
    <!-- 操作按钮（仅管理员） -->
    <div v-if="isAdmin" class="message-actions">
      <a-button 
        size="small" 
        danger 
        @click="handleDelete"
      >
        删除
      </a-button>
    </div>
  </div>
</template>

<script setup>
import { deleteMessage } from '@/api/app'
import { message as antMessage } from 'ant-design-vue'

const handleDelete = async () => {
  if (confirm('确定要删除这条消息吗？')) {
    try {
      await deleteMessage(props.message.id)
      antMessage.success('消息已删除')
      emit('deleted', props.message.id)
    } catch (error) {
      antMessage.error('删除失败: ' + error.message)
    }
  }
}
</script>
```

**API调用**:
```javascript
// frontend/src/api/app.js
export async function deleteMessage(messageID) {
  console.log('[API] Calling DeleteMessage:', messageID)
  const res = await App.DeleteMessage(messageID)
  console.log('[API] DeleteMessage response:', res)
  return unwrap(res)
}
```

---

### 任务3: 实现文件删除API ⏱️ 4小时

**当前缺失**: 没有`DeleteFile` API

**实施步骤**:

#### 1. 后端API (internal/app/file_api.go)

```go
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

	a.logger.Info("Deleting file: %s", fileID)

	// 从数据库获取文件信息
	file, err := a.db.FileRepo().GetByID(fileID)
	if err != nil {
		return NewErrorResponse("not_found", "文件不存在", err.Error())
	}

	// 权限检查：仅上传者或管理员可删除
	var currentUserID string
	if mode == ModeServer && srv != nil {
		currentUserID = "server"
	} else if mode == ModeClient && cli != nil {
		currentUserID = cli.GetMemberID()
	}

	// 检查权限
	if file.SenderID != currentUserID {
		// 检查是否为管理员
		member, _ := a.db.MemberRepo().GetByID(currentUserID)
		if member == nil || member.Role != models.RoleAdmin {
			return NewErrorResponse("permission_denied", "仅上传者或管理员可删除文件", "")
		}
	}

	// 删除数据库记录（级联删除chunks）
	if err := a.db.FileRepo().Delete(fileID); err != nil {
		return NewErrorResponse("delete_error", "删除文件记录失败", err.Error())
	}

	// 删除文件系统文件
	// TODO: 根据file.Path删除实际文件
	// os.Remove(file.Path)

	// 广播文件删除事件
	a.emitEvent(EventFileDeleted, map[string]interface{}{
		"file_id":  fileID,
		"filename": file.Filename,
	})

	a.logger.Info("File deleted: %s", fileID)

	return NewSuccessResponse(map[string]interface{}{
		"message": "文件已删除",
		"file_id": fileID,
	})
}
```

#### 2. 数据库Repository (internal/storage/file_repository.go)

```go
// Delete 删除文件（级联删除chunks）
func (r *FileRepository) Delete(fileID string) error {
	return r.db.GetChannelDB().Transaction(func(tx *gorm.DB) error {
		// 1. 删除file_chunks
		if err := tx.Where("file_id = ?", fileID).Delete(&models.FileChunk{}).Error; err != nil {
			return err
		}

		// 2. 删除files记录
		if err := tx.Where("id = ?", fileID).Delete(&models.File{}).Error; err != nil {
			return err
		}

		return nil
	})
}
```

#### 3. 前端集成 (frontend/src/components/FileList.vue)

```vue
<template>
  <a-list :data-source="files">
    <template #renderItem="{ item }">
      <a-list-item>
        <a-list-item-meta :title="item.filename">
          <template #description>
            {{ formatFileSize(item.size) }} • {{ formatDate(item.uploaded_at) }}
          </template>
        </a-list-item-meta>
        
        <template #actions>
          <a-button @click="handleDownload(item)">下载</a-button>
          <a-button 
            v-if="canDelete(item)" 
            danger 
            @click="handleDelete(item)"
          >
            删除
          </a-button>
        </template>
      </a-list-item>
    </template>
  </a-list>
</template>

<script setup>
import { deleteFile } from '@/api/app'
import { message } from 'ant-design-vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()

const canDelete = (file) => {
  // 上传者或管理员可删除
  return file.sender_id === userStore.userId || userStore.isAdmin
}

const handleDelete = async (file) => {
  if (confirm(`确定要删除文件"${file.filename}"吗？`)) {
    try {
      await deleteFile(file.id)
      message.success('文件已删除')
      emit('fileDeleted', file.id)
      // 刷新文件列表
      loadFiles()
    } catch (error) {
      message.error('删除失败: ' + error.message)
    }
  }
}
</script>
```

#### 4. API封装 (frontend/src/api/app.js)

```javascript
export async function deleteFile(fileID) {
  console.log('[API] Calling DeleteFile:', fileID)
  const res = await App.DeleteFile(fileID)
  console.log('[API] DeleteFile response:', res)
  return unwrap(res)
}
```

#### 5. 事件定义 (internal/app/event_handler.go)

```go
const (
	// ... 现有事件 ...
	EventFileDeleted         = "file:deleted"
)

func (a *App) subscribeFileEvents() {
	// ... 现有订阅 ...

	// 文件删除
	a.eventBus.Subscribe(events.EventFileDeleted, func(ev *events.Event) {
		a.emitEvent(EventFileDeleted, ev.Data)
	})
}
```

---

## ⚡ 重要 (两周内) - 5个任务

### 任务4: 消息编辑功能 ⏱️ 6小时
详见 `IMPLEMENTATION_PLAN.md` 第4节

### 任务5: 子频道权限隔离 ⏱️ 8小时
详见 `IMPLEMENTATION_PLAN.md` 第5节

### 任务6: 消息搜索优化 ⏱️ 4小时
详见 `IMPLEMENTATION_PLAN.md` 第6节

### 任务7: 技能标签等级UI ⏱️ 3小时
详见 `IMPLEMENTATION_PLAN.md` 第7节

### 任务8: 解题统计计算 ⏱️ 6小时
详见 `IMPLEMENTATION_PLAN.md` 第8节

---

## 📝 开发工作流

### 1. 选择任务
```bash
# 查看所有待办
cat TODO_FEATURE_CHECKLIST.md

# 选择一个P0或P1任务
# 标记为in_progress
```

### 2. 创建分支
```bash
git checkout -b fix/task-name
```

### 3. 实施修复
```bash
# 按照本文档的实施步骤操作
# 添加调试日志
# 编写单元测试
```

### 4. 测试验证
```bash
# 启动开发服务器
wails dev

# 手动测试功能
# 运行单元测试
go test ./internal/...

# 检查linter
golangci-lint run
```

### 5. 提交代码
```bash
git add .
git commit -m "fix: 任务描述"
git push origin fix/task-name
```

### 6. 更新TODO
```bash
# 标记任务为completed
# 更新TODO_FEATURE_CHECKLIST.md
```

---

## 🔍 调试技巧

### 后端调试
```go
// 添加详细日志
a.logger.Debug("[FunctionName] Input: %+v", input)
a.logger.Info("[FunctionName] Processing...")
a.logger.Error("[FunctionName] Error: %v", err)
```

### 前端调试
```javascript
// 控制台日志
console.log('[ComponentName] State:', state)
console.log('[API] Request:', request)
console.log('[API] Response:', response)

// Vue Devtools
// 安装浏览器扩展查看组件状态
```

### 数据库调试
```bash
# 连接数据库
sqlite3 data/channels/<channel-id>.db

# 查询消息
SELECT id, channel_id, sender_id, content, timestamp FROM messages ORDER BY timestamp DESC LIMIT 10;

# 查询文件
SELECT id, filename, size, upload_status FROM files;

# 查询成员
SELECT id, nickname, role, is_online FROM members;
```

---

## 📞 需要帮助?

如果遇到问题：

1. **查看文档**:
   - `TODO_FEATURE_CHECKLIST.md` - 功能清单
   - `IMPLEMENTATION_PLAN.md` - 详细方案
   - `docs/` - 架构和协议文档

2. **检查日志**:
   - 后端: `logs/crosswire_<date>.log`
   - 前端: 浏览器控制台
   - 数据库: SQLite错误信息

3. **代码搜索**:
   - 使用`grep`或IDE搜索相关函数
   - 查看类似功能的实现

4. **询问AI**:
   - 提供错误信息和相关代码
   - 描述预期行为和实际行为

---

**最后更新**: 2025-10-07  
**下次审查**: 每完成一个P0任务后更新


