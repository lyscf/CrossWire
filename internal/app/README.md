# CrossWire APP Layer 应用层

## 📑 概述

APP层是前端（Vue）和后端业务逻辑之间的桥梁，通过 Wails 框架暴露给前端调用。所有API都遵循统一的响应格式和错误处理机制。

## 🏗️ 架构设计

```
┌─────────────────────────────────────┐
│         Frontend (Vue.js)           │
│    调用 window.go.main.App.*       │
└──────────────┬──────────────────────┘
               │ Wails Bridge
┌──────────────▼──────────────────────┐
│         APP Layer (internal/app)    │
│   - 统一的 Response 包装             │
│   - 参数验证和错误处理               │
│   - 模式管理 (Server/Client)         │
└──────────────┬──────────────────────┘
               │
     ┌─────────┴────────┐
     │                  │
┌────▼──────┐    ┌─────▼────┐
│  Server   │    │  Client  │
│  业务层   │    │  业务层  │
└───────────┘    └──────────┘
```

## 📂 文件结构

```
internal/app/
├── README.md            # 本文档
├── types.go             # 所有DTO类型定义
├── app.go               # 主应用类
├── server_api.go        # 服务端模式API
├── client_api.go        # 客户端模式API
├── message_api.go       # 消息操作API
├── file_api.go          # 文件操作API
├── member_api.go        # 成员管理API
├── challenge_api.go     # CTF题目管理API
├── system_api.go        # 系统功能API
└── event_handler.go     # 事件处理（后端->前端）
```

## 🔌 API 分类

### 1. 基础 API

#### `GetAppVersion() string`
获取应用版本号

#### `GetCurrentMode() string`
获取当前运行模式：`idle`、`server`、`client`

#### `IsRunning() bool`
检查是否正在运行

---

### 2. 服务端模式 API

#### `StartServerMode(config ServerConfig) Response`
启动服务端模式，创建频道

**请求参数 (ServerConfig):**
```typescript
{
  channel_name: string        // 频道名称
  password: string            // 频道密码（至少6位）
  transport_mode: string      // 传输模式: "arp", "https", "mdns"
  network_interface?: string  // 网络接口（ARP/mDNS模式必填）
  listen_address?: string     // 监听地址（HTTPS模式，默认0.0.0.0）
  port?: number               // 监听端口（HTTPS模式，默认8443）
  max_members?: number        // 最大成员数（默认100）
  max_file_size?: number      // 最大文件大小，字节（默认100MB）
  enable_challenge?: boolean  // 启用题目功能
  description?: string        // 频道描述
}
```

**响应:**
```typescript
{
  success: boolean
  data?: {
    running: boolean
    channel_id: string
    channel_name: string
    transport_mode: string
    member_count: number
    start_time: string
    network_stats: NetworkStats
  }
  error?: ErrorInfo
}
```

#### `StopServerMode() Response`
停止服务端模式

#### `GetServerStatus() Response`
获取服务端状态

---

### 3. 客户端模式 API

#### `StartClientMode(config ClientConfig) Response`
启动客户端模式并加入频道

**请求参数 (ClientConfig):**
```typescript
{
  password: string            // 频道密码
  transport_mode: string      // 传输模式
  network_interface?: string  // 网络接口（ARP/mDNS）
  server_address?: string     // 服务器地址（HTTPS）
  port?: number               // 服务器端口（HTTPS）
  nickname: string            // 用户昵称
  avatar?: string             // 头像URL
  auto_reconnect?: boolean    // 自动重连
}
```

#### `StopClientMode() Response`
停止客户端模式

#### `GetClientStatus() Response`
获取客户端状态

#### `DiscoverServers(timeout int) Response`
发现本地网络中的服务器（仅客户端模式）

#### `GetDiscoveredServers() Response`
获取已发现的服务器列表

---

### 4. 消息操作 API

#### `SendMessage(req SendMessageRequest) Response`
发送文本消息

**请求参数:**
```typescript
{
  content: string            // 消息内容
  type: string               // 消息类型: "text", "system", "file", "code"
  reply_to_id?: string       // 回复的消息ID
}
```

#### `SendCodeMessage(req SendCodeRequest) Response`
发送代码消息

**请求参数:**
```typescript
{
  code: string               // 代码内容
  language: string           // 编程语言
  filename?: string          // 文件名
}
```

#### `GetMessages(limit int, offset int) Response`
获取消息列表

**返回:** `MessageDTO[]`

#### `GetMessage(messageID string) Response`
获取单条消息

#### `SearchMessages(req SearchMessagesRequest) Response`
搜索消息

**请求参数:**
```typescript
{
  query: string              // 搜索关键词
  type?: string              // 消息类型过滤
  sender_id?: string         // 发送者过滤
  start_time?: string        // 开始时间
  end_time?: string          // 结束时间
  limit: number              // 返回数量
  offset: number             // 偏移量
}
```

#### `DeleteMessage(messageID string) Response`
删除消息（仅服务端）

#### `PinMessage(messageID string) Response`
置顶消息（仅服务端）

#### `UnpinMessage(messageID string) Response`
取消置顶（仅服务端）

#### `ReactToMessage(messageID string, emoji string) Response`
对消息添加反应

#### `RemoveReaction(messageID string, emoji string) Response`
移除消息反应

---

### 5. 文件操作 API

#### `UploadFile(req UploadFileRequest) Response`
上传文件

**请求参数:**
```typescript
{
  file_path: string          // 文件路径
  description?: string       // 文件描述
}
```

#### `DownloadFile(req DownloadFileRequest) Response`
下载文件

**请求参数:**
```typescript
{
  file_id: string            // 文件ID
  save_path: string          // 保存路径
}
```

#### `CancelUpload(fileID string) Response`
取消文件上传

#### `CancelDownload(fileID string) Response`
取消文件下载

#### `GetFiles(limit int, offset int) Response`
获取文件列表

**返回:** `FileDTO[]`

#### `GetFile(fileID string) Response`
获取单个文件信息

#### `GetFileProgress(fileID string) Response`
获取文件传输进度

**返回:**
```typescript
{
  file_id: string
  file_name: string
  total_size: number
  transferred: number
  progress: number         // 0-100
  speed: number            // 字节/秒
  status: string           // "uploading", "downloading", "completed", "failed"
  error?: string
}
```

#### `GetFileTransferStats() Response`
获取文件传输统计

---

### 6. 成员管理 API

#### `GetMembers() Response`
获取成员列表

**返回:** `MemberDTO[]`

#### `GetMember(memberID string) Response`
获取单个成员信息

#### `GetMyInfo() Response`
获取当前用户信息

#### `UpdateMyStatus(status string) Response`
更新我的状态：`online`、`away`、`busy`、`offline`

#### `UpdateMyProfile(nickname string, avatar string) Response`
更新我的资料

#### `KickMember(req KickMemberRequest) Response`
踢出成员（仅服务端管理员）

#### `BanMember(req BanMemberRequest) Response`
封禁成员（仅服务端管理员）

#### `UnbanMember(memberID string) Response`
解封成员（仅服务端管理员）

#### `MuteMember(memberID string, duration int64) Response`
禁言成员（仅服务端管理员），duration单位：秒

#### `UnmuteMember(memberID string) Response`
解除禁言（仅服务端管理员）

#### `UpdateMemberRole(memberID string, role string) Response`
更新成员角色（仅服务端管理员）
角色: `admin`、`moderator`、`member`

---

### 7. CTF题目管理 API

#### `CreateChallenge(req CreateChallengeRequest) Response`
创建题目（仅服务端）

**请求参数:**
```typescript
{
  title: string              // 题目标题
  description: string        // 题目描述
  category: string           // 分类: "web", "pwn", "reverse", "crypto", "misc"
  difficulty: string         // 难度: "easy", "medium", "hard"
  points: number             // 分数
  flags: string[]            // flag列表
}
```

#### `GetChallenges() Response`
获取题目列表

**返回:** `ChallengeDTO[]`

#### `GetChallenge(challengeID string) Response`
获取单个题目

#### `UpdateChallenge(challengeID string, req UpdateChallengeRequest) Response`
更新题目（仅服务端）

#### `DeleteChallenge(challengeID string) Response`
删除题目（仅服务端）

#### `AssignChallenge(challengeID string, memberIDs []string) Response`
分配题目给成员（仅服务端）

#### `SubmitFlag(req SubmitFlagRequest) Response`
提交flag

**请求参数:**
```typescript
{
  challenge_id: string       // 题目ID
  flag: string               // 提交的flag
}
```

**响应:**
```typescript
{
  success: boolean
  is_correct: boolean        // flag是否正确
  message: string            // 提示信息
  points?: number            // 获得的分数
}
```

#### `UpdateChallengeProgress(req UpdateProgressRequest) Response`
更新题目进度

**请求参数:**
```typescript
{
  challenge_id: string       // 题目ID
  progress: number           // 进度 0-100
  summary: string            // 进度说明
}
```

#### `AddHint(req AddHintRequest) Response`
添加提示（仅服务端）

**请求参数:**
```typescript
{
  challenge_id: string       // 题目ID
  content: string            // 提示内容
  cost: number               // 消耗的分数
}
```

#### `UnlockHint(challengeID string, hintID string) Response`
解锁提示

#### `GetLeaderboard() Response`
获取排行榜

**返回:** `LeaderboardEntry[]`

#### `GetChallengeSubmissions(challengeID string) Response`
获取题目提交记录

#### `GetChallengeStats() Response`
获取题目统计信息（仅服务端）

---

### 8. 系统功能 API

#### `GetNetworkInterfaces() Response`
获取网络接口列表

**返回:** `NetworkInterface[]`

#### `TestConnection(serverAddress string, mode string, timeout int) Response`
测试连接

#### `GetNetworkStats() Response`
获取网络统计

#### `GetUserProfile() Response`
获取用户配置

#### `UpdateUserProfile(profile UserProfile) Response`
更新用户配置

#### `GetRecentChannels() Response`
获取最近的频道

#### `ExportData(exportPath string, options ExportOptions) Response`
导出数据

**请求参数:**
```typescript
{
  include_messages: boolean
  include_files: boolean
  include_challenges: boolean
  include_members: boolean
}
```

#### `ImportData(importPath string) Response`
导入数据

#### `SelectFile(title string, filter string) Response`
选择文件（打开系统文件选择对话框）

#### `SelectDirectory(title string) Response`
选择目录

#### `SaveFileDialog(title string, defaultFilename string) Response`
保存文件对话框

---

## 📡 事件系统

后端事件会自动通过事件总线转发到前端，前端可以监听 `app:event` 事件：

```javascript
// 前端监听事件
window.runtime.EventsOn('app:event', (event) => {
  console.log('Event received:', event.type, event.data)
})
```

### 事件类型

**连接事件:**
- `connected` - 已连接
- `disconnected` - 已断开
- `reconnecting` - 重连中

**消息事件:**
- `message:received` - 接收到消息
- `message:sent` - 消息已发送
- `message:updated` - 消息已更新
- `message:deleted` - 消息已删除

**成员事件:**
- `member:joined` - 成员加入
- `member:left` - 成员离开
- `member:updated` - 成员信息更新
- `member:kicked` - 成员被踢出
- `member:banned` - 成员被封禁

**文件事件:**
- `file:upload:started` - 上传开始
- `file:upload:progress` - 上传进度
- `file:upload:completed` - 上传完成
- `file:upload:failed` - 上传失败
- `file:download:started` - 下载开始
- `file:download:progress` - 下载进度
- `file:download:completed` - 下载完成
- `file:download:failed` - 下载失败

**题目事件:**
- `challenge:created` - 题目创建
- `challenge:updated` - 题目更新
- `challenge:solved` - 题目解决
- `challenge:assigned` - 题目分配

**系统事件:**
- `error` - 错误
- `warning` - 警告
- `info` - 信息

---

## 🎯 响应格式

所有API都返回统一的 `Response` 结构：

```typescript
interface Response {
  success: boolean
  data?: any
  error?: {
    code: string
    message: string
    details?: string
  }
}
```

### 错误码

- `not_running` - 未运行
- `already_running` - 已在运行
- `invalid_config` - 配置无效
- `invalid_mode` - 无效的运行模式
- `permission_denied` - 权限不足
- `not_found` - 资源不存在
- `invalid_request` - 无效的请求
- `transport_error` - 传输层错误
- `file_error` - 文件错误
- `network_error` - 网络错误
- `db_error` - 数据库错误

---

## 🔒 权限控制

| 功能 | 服务端 | 客户端 | 说明 |
|------|--------|--------|------|
| 创建频道 | ✅ | ❌ | 仅服务端可创建 |
| 加入频道 | ❌ | ✅ | 仅客户端可加入 |
| 发送消息 | ✅ | ✅ | 都可以 |
| 删除消息 | ✅ | ❌ | 仅服务端 |
| 踢人/封禁 | ✅ | ❌ | 仅服务端管理员 |
| 创建题目 | ✅ | ❌ | 仅服务端 |
| 提交flag | ✅ | ✅ | 都可以 |
| 文件上传 | ✅ | ✅ | 都可以 |
| 查看统计 | ✅ | ❌ | 仅服务端 |

---

## 📝 前端调用示例

### Vue 3 Composition API

```vue
<script setup>
import { ref } from 'vue'

const status = ref(null)
const messages = ref([])

// 启动服务端
async function startServer() {
  const config = {
    channel_name: 'My CTF Team',
    password: '123456',
    transport_mode: 'https',
    port: 8443,
    enable_challenge: true
  }
  
  const result = await window.go.main.App.StartServerMode(config)
  if (result.success) {
    status.value = result.data
    console.log('Server started:', result.data)
  } else {
    console.error('Failed to start server:', result.error.message)
  }
}

// 发送消息
async function sendMessage(content) {
  const req = {
    content: content,
    type: 'text'
  }
  
  const result = await window.go.main.App.SendMessage(req)
  if (!result.success) {
    console.error('Failed to send message:', result.error.message)
  }
}

// 获取消息列表
async function loadMessages() {
  const result = await window.go.main.App.GetMessages(50, 0)
  if (result.success) {
    messages.value = result.data
  }
}

// 监听事件
window.runtime.EventsOn('app:event', (event) => {
  if (event.type === 'message:received') {
    messages.value.push(event.data)
  }
})
</script>
```

---

## 🚀 待实现功能

当前APP层已完成所有API定义，但依赖以下底层实现：

### Server层需要补充的方法：
- `GetChannelID()` - 获取频道ID
- `GetChannelInfo()` - 获取频道信息
- `GetMembers()` - 获取成员列表
- `GetMemberID()` - 获取服务端成员ID
- `SendMessage()` - 发送消息
- `DeleteMessage()` - 删除消息
- `PinMessage()` / `UnpinMessage()` - 置顶/取消置顶
- `AddReaction()` / `RemoveReaction()` - 添加/移除反应
- `UploadFile()` / `DownloadFile()` - 文件上传/下载
- `CancelUpload()` / `CancelDownload()` - 取消传输
- `GetUploadTask()` - 获取上传任务
- `GetFileManagerStats()` - 获取文件管理器统计
- `UpdateMyStatus()` / `UpdateMyProfile()` - 更新状态/资料
- `KickMember()` / `BanMember()` / `UnbanMember()` - 踢人/封禁/解封
- `MuteMember()` / `UnmuteMember()` - 禁言/解除禁言
- `UpdateMemberRole()` - 更新角色
- `CreateChallenge()` / `UpdateChallenge()` / `DeleteChallenge()` - 题目CRUD
- `GetChallenges()` / `GetChallenge()` - 获取题目
- `AssignChallenge()` - 分配题目
- `SubmitFlag()` - 提交flag
- `UpdateChallengeProgress()` - 更新进度
- `AddHint()` / `UnlockHint()` - 添加/解锁提示
- `GetLeaderboard()` / `GetChallengeStats()` - 排行榜/统计

### Client层需要补充的方法：
- `GetChannelID()` - 获取频道ID
- `GetChannelInfo()` - 获取频道信息
- `IsConnected()` - 是否已连接
- `GetConnectTime()` - 获取连接时间
- `GetMembers()` - 获取成员列表
- `SendMessage()` - 发送消息（调整参数）
- `AddReaction()` / `RemoveReaction()` - 添加/移除反应
- `UploadFile()` - 调整上传文件返回值
- `DownloadFile()` - 调整下载文件返回值
- `GetUploadTask()` / `GetDownloadTask()` - 获取传输任务
- `UpdateMyStatus()` / `UpdateMyProfile()` - 更新状态/资料
- `GetChallenges()` - 调整返回值
- `GetChallenge()` - 调整返回值
- `SubmitFlag()` - 调整返回值
- `UpdateProgress()` - 更新进度
- `RequestHint()` - 调整参数和返回值
- `GetChallengeSubmissions()` - 调整参数

### Database层需要补充的Repository：
- `MessageRepo()` - 消息仓库
- `FileRepo()` - 文件仓库
- `MemberRepo()` - 成员仓库
- `ChallengeRepo()` - 题目仓库
- `ChallengeSubmissionRepo()` - 题目提交仓库

### Transport层需要补充：
- 统一的配置结构（ARPConfig, HTTPSConfig, MDNSConfig）

### Models层需要补充的字段：
- `Member.IsOnline`, `JoinTime`, `LastSeenAt`, `IsMuted`, `IsBanned`
- `Message.IsEdited`, `IsDeleted`, `IsPinned`, `ReplyToID`改为指针类型
- `Challenge.AssignedTo`
- `MemberRole` 类型定义

---

## 八荣八耻规范 ✅

APP层严格遵循Go语言最佳实践：

✅ **以统一响应格式为荣**，避免返回不一致的数据结构  
✅ **以详细错误信息为荣**，避免返回含糊的错误提示  
✅ **以参数验证为荣**，避免将无效数据传递给底层  
✅ **以并发安全为荣**，所有状态访问都使用读写锁  
✅ **以日志记录为荣**，关键操作都有日志输出  
✅ **以事件驱动为荣**，使用事件总线解耦模块  
✅ **以清晰分层为荣**，APP层只做编排不含业务逻辑  
✅ **以文档完善为荣**，每个API都有详细说明和示例  

---

## 📚 相关文档

- [系统架构文档](../../docs/ARCHITECTURE.md)
- [通信协议文档](../../docs/PROTOCOL.md)
- [数据库设计文档](../../docs/DATABASE.md)
- [功能特性文档](../../docs/FEATURES.md)
- [前端对接指南](../../frontend/README.md)

---

**版本:** 1.0.0  
**更新日期:** 2025-10-06  
**作者:** CrossWire Team

