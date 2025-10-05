# CrossWire 下一步开发指南

> 继续开发CrossWire项目的详细步骤指南

---

## 📍 当前状态

项目已完成**基础架构层**的搭建:
- ✅ 完整的数据模型 (Models)
- ✅ 数据存储层 (Storage/Repositories)  
- ✅ 加密模块 (Crypto)
- ✅ 工具模块 (Logger/Validator)
- ✅ 项目可以成功编译

**总体进度**: 约 35%

---

## 🎯 优先级开发顺序

### 阶段1: 核心通信层 (2-3天)

#### 1.1 实现EventBus事件总线
**位置**: `internal/events/eventbus.go`

```go
package events

type EventType string
type EventHandler func(data interface{})

type EventBus struct {
    subscribers map[EventType][]EventHandler
    mu          sync.RWMutex
}

// 需要实现的方法:
// - NewEventBus()
// - Subscribe(eventType EventType, handler EventHandler)
// - Unsubscribe(eventType EventType, handler EventHandler)
// - Publish(eventType EventType, data interface{})
// - PublishAsync(eventType EventType, data interface{})
```

**事件类型定义**:
```go
const (
    EventMessageReceived  EventType = "message:received"
    EventMessageSent      EventType = "message:sent"
    EventMemberJoined     EventType = "member:joined"
    EventMemberLeft       EventType = "member:left"
    EventMemberStatusChanged EventType = "member:status_changed"
    EventFileUploaded     EventType = "file:uploaded"
    EventFileReceived     EventType = "file:received"
    EventChannelCreated   EventType = "channel:created"
    EventServerStarted    EventType = "server:started"
    EventClientConnected  EventType = "client:connected"
)
```

---

#### 1.2 实现Transport传输层接口
**位置**: `internal/transport/transport.go`

先实现统一接口:

```go
package transport

type Transport interface {
    // 生命周期
    Init(config *Config) error
    Start() error
    Stop() error
    
    // 消息收发
    Send(data []byte, target string) error
    Receive() ([]byte, error)
    SetMessageHandler(handler func([]byte, string))
    
    // 状态
    IsConnected() bool
    GetStats() *Stats
}

type Config struct {
    Mode      string // "arp", "https", "mdns"
    Interface string // 网卡名称 (ARP)
    Port      int    // 端口 (HTTPS)
    // ...
}
```

然后先实现**HTTPS模式** (最简单):

**位置**: `internal/transport/https_transport.go`

```go
package transport

import (
    "github.com/gorilla/websocket"
)

type HTTPSTransport struct {
    conn    *websocket.Conn
    config  *Config
    handler func([]byte, string)
}

// 实现Transport接口的所有方法
```

---

### 阶段2: 服务端实现 (2-3天)

#### 2.1 实现Server核心
**位置**: `internal/server/server.go`

```go
package server

type Server struct {
    channel     *models.Channel
    members     map[string]*models.Member
    transport   transport.Transport
    db          *storage.Database
    eventBus    *events.EventBus
    crypto      *crypto.Manager
    mu          sync.RWMutex
}

func NewServer(config *Config, db *storage.Database) *Server
func (s *Server) Start() error
func (s *Server) Stop() error
func (s *Server) BroadcastMessage(msg *models.Message) error
func (s *Server) HandleClientMessage(data []byte, clientID string)
```

---

#### 2.2 实现频道管理
**位置**: `internal/server/channel.go`

```go
func (s *Server) CreateChannel(name, password string) error
func (s *Server) AddMember(member *models.Member) error
func (s *Server) RemoveMember(memberID string) error
func (s *Server) GetOnlineMembers() []*models.Member
```

---

#### 2.3 实现认证管理
**位置**: `internal/server/auth.go`

```go
func (s *Server) AuthenticateMember(nickname, password string) (*models.Member, error)
func (s *Server) VerifyPassword(password string) bool
func (s *Server) GenerateMemberID(nickname string) string
```

---

### 阶段3: 客户端实现 (2-3天)

#### 3.1 实现Client核心
**位置**: `internal/client/client.go`

```go
package client

type Client struct {
    memberID    string
    channel     *models.Channel
    transport   transport.Transport
    db          *storage.Database
    eventBus    *events.EventBus
    crypto      *crypto.Manager
}

func NewClient(config *Config, db *storage.Database) *Client
func (c *Client) Connect(serverAddress string) error
func (c *Client) Disconnect() error
func (c *Client) SendMessage(content string, msgType models.MessageType) error
func (c *Client) HandleServerMessage(data []byte)
```

---

#### 3.2 实现消息同步
**位置**: `internal/client/sync.go`

```go
func (c *Client) SyncMessages() error
func (c *Client) SyncMembers() error
func (c *Client) SyncFiles() error
```

---

### 阶段4: 完善App层 (1-2天)

#### 4.1 实现模式切换
**位置**: `internal/app/app.go`

```go
type Mode string

const (
    ModeServer Mode = "server"
    ModeClient Mode = "client"
)

func (a *App) StartServerMode(config ServerConfig) error {
    a.mode = ModeServer
    a.server = server.NewServer(config, a.db)
    return a.server.Start()
}

func (a *App) StartClientMode(config ClientConfig) error {
    a.mode = ModeClient
    a.client = client.NewClient(config, a.db)
    return a.client.Connect(config.ServerAddress)
}
```

---

#### 4.2 实现消息API
**位置**: `internal/app/message.go`

```go
func (a *App) SendMessage(content string, msgType models.MessageType) error
func (a *App) GetMessages(limit, offset int) ([]*models.Message, error)
func (a *App) SearchMessages(query string) ([]*models.Message, error)
func (a *App) DeleteMessage(messageID string) error
func (a *App) PinMessage(messageID string) error
```

---

#### 4.3 实现文件API
**位置**: `internal/app/file.go`

```go
func (a *App) UploadFile(filePath string) error
func (a *App) DownloadFile(fileID, savePath string) error
func (a *App) GetFiles(limit, offset int) ([]*models.File, error)
```

---

#### 4.4 实现成员API
**位置**: `internal/app/member.go`

```go
func (a *App) GetMembers() ([]*models.Member, error)
func (a *App) UpdateMyStatus(status models.UserStatus) error
func (a *App) KickMember(memberID, reason string) error
func (a *App) MuteMember(memberID string, duration int64) error
```

---

#### 4.5 实现题目API
**位置**: `internal/app/challenge.go`

```go
func (a *App) CreateChallenge(challenge *models.Challenge) error
func (a *App) GetChallenges() ([]*models.Challenge, error)
func (a *App) AssignChallenge(challengeID string, memberIDs []string) error
func (a *App) SubmitFlag(challengeID, flag string) (*SubmissionResult, error)
func (a *App) UpdateProgress(challengeID string, progress int, summary string) error
```

---

### 阶段5: 前端开发 (5-7天)

#### 5.1 初始化Vue项目

```bash
cd frontend
npm create vite@latest . -- --template vue
npm install
npm install naive-ui pinia @vueuse/core
```

---

#### 5.2 实现核心组件

**目录结构**:
```
frontend/src/
├── App.vue
├── main.js
├── views/
│   ├── HomeView.vue        # 启动页
│   ├── ServerView.vue      # 服务端配置
│   ├── ClientView.vue      # 客户端加入
│   └── ChatView.vue        # 聊天界面
├── components/
│   ├── chat/
│   │   ├── MessageList.vue
│   │   ├── MessageInput.vue
│   │   └── MessageItem.vue
│   ├── member/
│   │   ├── MemberList.vue
│   │   └── MemberCard.vue
│   └── file/
│       ├── FileUpload.vue
│       └── FileList.vue
├── stores/
│   ├── app.js
│   ├── channel.js
│   ├── message.js
│   └── member.js
└── api/
    └── wails.js            # Wails API封装
```

---

#### 5.3 Wails API封装

**位置**: `frontend/src/api/wails.js`

```javascript
// 使用Wails的运行时生成的绑定
import * as App from '../../../wailsjs/go/main/App'

export const api = {
  // 模式切换
  startServerMode: App.StartServerMode,
  startClientMode: App.StartClientMode,
  
  // 消息
  sendMessage: App.SendMessage,
  getMessages: App.GetMessages,
  
  // 文件
  uploadFile: App.UploadFile,
  
  // 成员
  getMembers: App.GetMembers,
  
  // ...其他API
}
```

---

#### 5.4 Pinia状态管理

**位置**: `frontend/src/stores/message.js`

```javascript
import { defineStore } from 'pinia'
import { api } from '../api/wails'

export const useMessageStore = defineStore('message', {
  state: () => ({
    messages: [],
    loading: false
  }),
  
  actions: {
    async loadMessages() {
      this.loading = true
      try {
        this.messages = await api.getMessages(50, 0)
      } finally {
        this.loading = false
      }
    },
    
    async sendMessage(content, type = 'text') {
      await api.sendMessage(content, type)
      await this.loadMessages()
    }
  }
})
```

---

## 🛠️ 开发工具和命令

### 开发模式

```bash
# 启动Wails开发服务器（热重载）
wails dev

# 仅构建Go后端
go build .

# 运行测试
go test ./...
```

---

### 生成Wails绑定

```bash
# 生成Go到JS的绑定
wails generate module

# 生成TypeScript定义
wails generate bindings
```

---

### 编译打包

```bash
# 开发构建
wails build

# 生产构建
wails build -clean -upx

# 指定平台
wails build -platform windows/amd64
wails build -platform darwin/arm64
wails build -platform linux/amd64
```

---

## 📚 参考文档

### Wails文档
- [Wails官方文档](https://wails.io/docs/introduction)
- [Wails绑定](https://wails.io/docs/reference/runtime/intro)
- [Wails事件](https://wails.io/docs/reference/runtime/events)

### Go包文档
- [GORM文档](https://gorm.io/docs/)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto)

### 前端文档
- [Vue 3文档](https://vuejs.org/guide/introduction.html)


---

## 🧪 测试建议

### 单元测试

为每个模块编写测试:

```go
// internal/crypto/crypto_test.go
func TestAESEncrypt(t *testing.T) {
    m := NewManager()
    key := make([]byte, 32)
    plaintext := []byte("test message")
    
    ciphertext, err := m.AESEncrypt(plaintext, key)
    if err != nil {
        t.Fatal(err)
    }
    
    decrypted, err := m.AESDecrypt(ciphertext, key)
    if err != nil {
        t.Fatal(err)
    }
    
    if string(decrypted) != string(plaintext) {
        t.Errorf("decryption failed")
    }
}
```

---

### 集成测试

测试完整流程:

```go
// tests/integration_test.go
func TestServerClientCommunication(t *testing.T) {
    // 1. 启动服务端
    // 2. 启动客户端并连接
    // 3. 发送消息
    // 4. 验证消息接收
    // 5. 清理
}
```

---

## 📋 开发检查清单

### 每个新功能
- [ ] 是否有清晰的TODO注释?
- [ ] 是否有错误处理?
- [ ] 是否有日志记录?
- [ ] 是否有参数验证?
- [ ] 是否考虑了并发安全?
- [ ] 是否编写了单元测试?

### 提交前检查
- [ ] 代码能成功编译?
- [ ] 没有lint错误?
- [ ] 测试全部通过?
- [ ] 文档是否更新?
- [ ] TODO是否更新?

---

## 💡 开发技巧

1. **逐步实现**: 不要一次实现所有功能，先实现核心流程
2. **先简后繁**: 先实现HTTPS模式，再实现ARP/mDNS
3. **频繁测试**: 每实现一个模块就测试一次
4. **保持注释**: 用TODO标注未完成的部分
5. **参考文档**: 遇到问题先查看`docs/`目录下的设计文档

---

## 🎯 里程碑

### Milestone 1: 基础通信 (已完成35%)
- [x] 数据模型
- [x] 存储层
- [x] 加密模块
- [ ] 传输层
- [ ] 事件总线

### Milestone 2: 核心功能 (0%)
- [ ] 服务端实现
- [ ] 客户端实现
- [ ] App层API

### Milestone 3: 前端界面 (0%)
- [ ] Vue组件
- [ ] 状态管理
- [ ] Wails集成

### Milestone 4: 高级功能 (0%)
- [ ] 文件传输
- [ ] 题目系统
- [ ] 全文搜索

### Milestone 5: 优化完善 (0%)
- [ ] 性能优化
- [ ] 错误处理
- [ ] 测试覆盖
- [ ] 文档完善

---

## 📞 遇到问题?

1. 查看`docs/`目录下的设计文档
2. 搜索代码中的TODO注释
3. 检查`PROJECT_STATUS.md`了解当前进度
4. 参考Wails和Go的官方文档

---

**继续加油!** 🚀

每完成一个模块记得更新`PROJECT_STATUS.md`和TODO列表。

