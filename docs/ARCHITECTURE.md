# CrossWire 系统架构文档

> CTF 线下赛通讯系统 - 系统架构设计
> 
> Version: 1.0.0  
> Date: 2025-10-05

---

## 📑目录

- [1. 架构概述](#1-架构概述)
- [2. 分层架构](#2-分层架构)
- [3. 模块设计](#3-模块设计)
- [4. 部署架构](#4-部署架构)
- [5. 技术栈](#5-技术栈)
- [6. 目录结构](#6-目录结构)

---

## 1. 架构概述

### 1.1 系统定位

CrossWire 是一个**桌面端即时通讯应用**，采用 **Wails2 (Go + Vue)** 技术栈，支持 **ARP/HTTPS/mDNS** 三种传输模式的 CTF 团队协作工具。

### 1.2 架构风格

- **前后端分离**：Vue 3 前端 + Go 后端
- **事件驱动**：基于消息队列的异步通信
- **插件化传输**：可插拔的传输层实现
- **模块化设计**：高内聚低耦合的模块划分

### 1.3 核心特性

```
┌─────────────────────────────────────────┐
│            CrossWire 核心架构            │
├─────────────────────────────────────────┤
│ • 双模式运行（服务端/客户端）           │
│ • 三种传输（ARP/HTTPS/mDNS）            │
│ • 端到端加密（X25519）             │
│ • 实时同步（WebSocket/自定义协议）      │
│ • 离线缓存（SQLite）                    │
└─────────────────────────────────────────┘
```

---

## 2. 分层架构

### 2.1 整体分层

```
┌────────────────────────────────────────────────────────┐
│                  Presentation Layer (前端)              │
│  Vue 3 + Pinia + Naive UI                             │
│  - 用户界面渲染                                        │
│  - 状态管理                                            │
│  - 用户交互逻辑                                        │
└────────────────────────────────────────────────────────┘
                          ↕ Wails Bridge
┌────────────────────────────────────────────────────────┐
│                  Application Layer (应用层)             │
│  Go Backend                                            │
│  - 业务逻辑编排                                        │
│  - 权限控制                                            │
│  - 消息路由                                            │
└────────────────────────────────────────────────────────┘
                          ↕
┌────────────────────────────────────────────────────────┐
│                  Domain Layer (领域层)                  │
│  - 频道管理 (Channel)                                  │
│  - 成员管理 (Member)                                   │
│  - 消息处理 (Message)                                  │
│  - 文件传输 (File)                                     │
└────────────────────────────────────────────────────────┘
                          ↕
┌────────────────────────────────────────────────────────┐
│               Infrastructure Layer (基础设施层)         │
│  - 传输层 (Transport)                                  │
│  - 加密模块 (Crypto)                                   │
│  - 存储模块 (Storage)                                  │
│  - 服务发现 (Discovery)                                │
└────────────────────────────────────────────────────────┘
```

---

### 2.2 前端架构

```
frontend/
│
├─ 视图层 (Views)
│  ├─ HomeView      : 启动页（模式选择）
│  ├─ ServerView    : 服务端配置页
│  ├─ ClientView    : 客户端加入页
│  └─ ChatView      : 聊天主界面
│
├─ 组件层 (Components)
│  ├─ MessageList   : 消息列表
│  ├─ MessageInput  : 消息输入框
│  ├─ MemberList    : 成员列表
│  ├─ FilePreview   : 文件预览
│  └─ CodeEditor    : 代码编辑器
│
├─ 状态层 (Stores - Pinia)
│  ├─ appStore      : 应用全局状态
│  ├─ channelStore  : 频道状态
│  ├─ messageStore  : 消息状态
│  ├─ memberStore   : 成员状态
│  └─ fileStore     : 文件状态
│
└─ API 层 (Services)
   ├─ wailsAPI      : Wails 桥接
   ├─ messageAPI    : 消息相关 API
   ├─ fileAPI       : 文件相关 API
   └─ memberAPI     : 成员相关 API
```

---

### 2.3 后端架构

```
backend/
│
├─ 应用层 (App)
│  ├─ app.go           : Wails 主应用
│  ├─ lifecycle.go     : 生命周期管理
│  └─ events.go        : 事件处理
│
├─ 领域层 (Internal)
│  ├─ server/          : 服务端逻辑
│  │  ├─ server.go     : 服务端核心
│  │  ├─ channel.go    : 频道管理
│  │  ├─ broadcast.go  : 消息广播
│  │  └─ auth.go       : 认证授权
│  │
│  ├─ client/          : 客户端逻辑
│  │  ├─ client.go     : 客户端核心
│  │  ├─ connection.go : 连接管理
│  │  └─ sync.go       : 消息同步
│  │
│  └─ models/          : 数据模型
│     ├─ channel.go
│     ├─ member.go
│     ├─ message.go
│     └─ file.go
│
└─ 基础设施层 (Internal)
   ├─ transport/       : 传输层
   │  ├─ transport.go  : 统一接口
   │  ├─ arp.go        : ARP 实现
   │  ├─ https.go      : HTTPS 实现
   │  └─ mdns.go       : mDNS 实现
   │
   ├─ crypto/          : 加密模块
   │  ├─ aes.go        : AES 加密
   │  ├─ rsa.go        : RSA 密钥
   │  └─ keygen.go     : 密钥派生
   │
   ├─ storage/         : 存储模块
   │  ├─ database.go   : 数据库接口
   │  ├─ sqlite.go     : SQLite 实现
   │  └─ cache.go      : 缓存管理
   │
   └─ discovery/       : 服务发现
      ├─ mdns.go       : mDNS 发现
      └─ scan.go       : 局域网扫描
```

---

## 3. 模块设计

### 3.1 核心模块

#### 3.1.1 应用模块 (App)

**职责：**
- Wails 应用生命周期管理
- 前后端桥接
- 全局事件分发
- 模式切换（服务端/客户端）

**关键接口：**

```go
type App struct {
    ctx       context.Context
    mode      Mode
    server    *server.Server
    client    *client.Client
    db        *storage.Database
    eventBus  *EventBus
}

// 导出给前端的方法
func (a *App) StartServerMode(config ServerConfig) error
func (a *App) StartClientMode(config ClientConfig) error
func (a *App) SendMessage(content string) error
func (a *App) UploadFile(path string) error
func (a *App) GetMessages(limit int) ([]*models.Message, error)
```

---

#### 3.1.2 服务端模块 (Server)

**职责：**
- 管理频道
- 处理客户端连接
- 广播消息
- 权限控制

**架构：**

```
Server
  ├─ ChannelManager    : 频道管理器
  │   ├─ 创建/关闭频道
  │   ├─ 成员加入/离开
  │   └─ 权限验证
  │
  ├─ BroadcastManager  : 广播管理器（简化）
  │   ├─ 广播消息到所有成员
  │   ├─ 消息去重（防止接收自己的广播）
  │   └─ 可选ACK收集
  │
  ├─ MessageRouter     : 消息路由器
  │   ├─ 消息处理
  │   ├─ 消息持久化
  │   └─ 离线消息队列
  │
  ├─ AuthManager       : 认证管理器（简化）
  │   ├─ 密码验证
  │   ├─ 频道密钥分发
  │   └─ 权限检查
  │
  └─ ChallengeManager  : 题目管理器 
      ├─ 题目创建/编辑
      ├─ 题目分配
      ├─ Flag验证
      ├─ 进度跟踪
      └─ 聊天室隔离
```

**说明：**
- ❌ 移除 ConnectionManager（广播模式无需维护连接）
- ✅ 简化为 BroadcastManager
- ✅ 无需心跳检测（通过消息活跃度判断在线状态）
- ✅ 无需 JWT Token（通过频道密钥验证身份）

---

#### 3.1.3 客户端模块 (Client)

**职责：**
- 连接服务端
- 发送/接收消息
- 本地缓存
- 状态同步

**架构：**

```
Client
  ├─ ReceiveManager    : 接收管理（简化）
  │   ├─ 监听广播帧
  │   ├─ 解密过滤
  │   └─ 消息去重
  │
  ├─ SyncManager       : 同步管理
  │   ├─ 增量同步（首次加入时）
  │   ├─ 离线消息
  │   └─ 冲突解决
  │
  ├─ CacheManager      : 缓存管理
  │   ├─ 消息缓存
  │   ├─ 文件缓存
  │   └─ 成员缓存
  │
  └─ EventManager      : 事件管理
      ├─ 消息事件
      ├─ 状态事件
      └─ 系统事件
```

**说明：**
- ❌ 移除 ConnectionManager（广播模式无需连接管理）
- ✅ 简化为 ReceiveManager（仅监听和过滤）
- ✅ 无需心跳机制（通过消息活跃度判断）
- ✅ 无需断线重连（始终监听广播）

---

#### 3.1.4 传输模块 (Transport)

**职责：**
- 抽象传输层
- 实现多种传输方式
- 统一接口

**接口设计：**

```go
type Transport interface {
    // 生命周期
    Init(config *Config) error
    Start() error
    Stop() error
    
    // 连接
    Connect(target string) error
    Disconnect() error
    IsConnected() bool
    
    // 消息
    SendMessage(msg *Message) error
    ReceiveMessage() (*Message, error)
    Subscribe(handler MessageHandler) error
    
    // 文件
    SendFile(file *FileTransfer) error
    OnFileReceived(handler FileHandler) error
    
    // 发现
    Discover() ([]*PeerInfo, error)
    Announce(info *ServiceInfo) error
    
    // 元数据
    GetMode() TransportMode
    GetStats() *TransportStats
}
```

**实现：**
- `ARPTransport`：原始以太网帧
- `HTTPSTransport`：WebSocket over TLS
- `MDNSTransport`：服务发现隐蔽传输

---

#### 3.1.5 加密模块 (Crypto)

**职责：**
- 消息加密/解密
- 密钥管理
- 签名验证

**架构：**

```
Crypto Manager
  ├─ AESCipher        : X25519 加密
  ├─ RSACipher        : X25519 密钥交换
  ├─ KeyDerivation    : PBKDF2/Argon2 密钥派生
  ├─ SignatureManager : HMAC-SHA256 签名
  └─ KeyRotation      : 密钥轮换
```

---

#### 3.1.6 存储模块 (Storage)

**职责：**
- 数据持久化
- 查询优化
- 缓存管理

**接口设计：**

```go
type Database interface {
    // 频道
    CreateChannel(channel *models.Channel) error
    GetChannel(id string) (*models.Channel, error)
    
    // 成员
    AddMember(member *models.Member) error
    GetMembers(channelID string) ([]*models.Member, error)
    UpdateMemberStatus(id string, status models.UserStatus) error
    
    // 消息
    SaveMessage(msg *models.Message) error
    GetMessages(channelID string, limit, offset int) ([]*models.Message, error)
    SearchMessages(query *SearchQuery) ([]*models.Message, error)
    
    // 文件
    SaveFile(file *models.File) error
    GetFile(id string) (*models.File, error)
}
```

---

### 3.2 辅助模块

#### 3.2.1 服务发现 (Discovery)

```go
type Discovery struct {
    mdnsServer *mdns.Server
    scanner    *LocalScanner
}

func (d *Discovery) AnnounceService(info *ServiceInfo) error
func (d *Discovery) ScanLocalNetwork() ([]*ServerInfo, error)
func (d *Discovery) ResolveAddress(hostname string) (string, error)
```

---

#### 3.2.2 事件总线 (EventBus)

```go
type EventBus struct {
    subscribers map[EventType][]EventHandler
    mutex       sync.RWMutex
}

type EventType string

const (
    EventMessageReceived  EventType = "message:received"
    EventMemberJoined     EventType = "member:joined"
    EventMemberLeft       EventType = "member:left"
    EventFileUploaded     EventType = "file:uploaded"
    EventStatusChanged    EventType = "status:changed"
)

func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler)
func (eb *EventBus) Publish(eventType EventType, data interface{})
```

---

#### 3.2.3 日志模块 (Logger)

```go
type Logger struct {
    level      LogLevel
    output     io.Writer
    formatter  Formatter
}

func (l *Logger) Debug(format string, args ...interface{})
func (l *Logger) Info(format string, args ...interface{})
func (l *Logger) Warn(format string, args ...interface{})
func (l *Logger) Error(format string, args ...interface{})
```

---

## 4. 部署架构

### 4.1 单机部署

```
┌─────────────────────────────────────┐
│        User's Computer              │
│                                     │
│  ┌───────────────────────────────┐ │
│  │     CrossWire.exe             │ │
│  │  ┌─────────────────────────┐  │ │
│  │  │   Frontend (Embedded)   │  │ │
│  │  │   Vue 3 + Vite          │  │ │
│  │  └─────────────────────────┘  │ │
│  │            ↕ Wails Bridge     │ │
│  │  ┌─────────────────────────┐  │ │
│  │  │   Backend (Go)          │  │ │
│  │  │   - Server/Client Logic │  │ │
│  │  │   - Transport Layer     │  │ │
│  │  │   - Crypto Module       │  │ │
│  │  └─────────────────────────┘  │ │
│  └───────────────────────────────┘ │
│                                     │
│  ┌───────────────────────────────┐ │
│  │   SQLite Database             │ │
│  │   ~/.crosswire/               │ │
│  │   ├─ channels/*.db            │ │
│  │   ├─ user.db                  │ │
│  │   └─ cache.db                 │ │
│  └───────────────────────────────┘ │
└─────────────────────────────────────┘
         │
         │ Network (ARP/HTTPS/mDNS)
         ↓
   Other Clients
```

---

### 4.2 网络拓扑

#### 4.2.1 ARP 模式（局域网）

```
       Switch/Router
            │
    ┌───────┼───────┐
    │       │       │
┌───────┐ ┌───────┐ ┌───────┐
│Server │ │Client1│ │Client2│
│(队长) │ │(队员) │ │(队员) │
└───────┘ └───────┘ └───────┘
    │       │       │
    └───────┴───────┘
  原始以太网帧（Layer 2）
  EtherType: 0x88B5
```

---

#### 4.2.2 HTTPS 模式（跨网络）

```
        Internet/VPN
            │
    ┌───────┼───────┐
    │       │       │
┌───────┐ ┌───────┐ ┌───────┐
│Server │ │Client1│ │Client2│
│8443   │ │       │ │       │
└───────┘ └───────┘ └───────┘
    │       │       │
    └───────┴───────┘
  WebSocket over TLS 1.3
```

---

#### 4.2.3 mDNS 模式（受限网络）

```
   局域网（仅 UDP 5353 可通过）
            │
    ┌───────┼───────┐
    │       │       │
┌───────┐ ┌───────┐ ┌───────┐
│Server │ │Client1│ │Client2│
│       │ │       │ │       │
└───────┘ └───────┘ └───────┘
    │       │       │
    └───────┴───────┘
  mDNS 服务发现
  _crosswire._udp.local
```

---

### 4.3 数据流

#### 4.3.1 消息发送流程

```
Frontend (Vue)
    │
    │ 1. 用户输入消息
    ↓
Wails Bridge
    │
    │ 2. 调用 Go 方法
    ↓
App Layer
    │
    │ 3. 验证权限
    ↓
Domain Layer (Message Handler)
    │
    │ 4. 序列化消息
    ↓
Crypto Layer
    │
    │ 5. 加密消息
    ↓
Transport Layer
    │
    │ 6. 选择传输方式
    ├───┬───┬───
    │   │   │
    ARP HTTPS mDNS
    │   │   │
    └───┴───┴───
        │
        │ 7. 网络传输
        ↓
    Receiver
```

---

#### 4.3.2 消息接收流程

```
Transport Layer
    │
    │ 1. 接收网络数据
    ↓
Crypto Layer
    │
    │ 2. 解密验证
    ↓
Domain Layer
    │
    │ 3. 反序列化
    ↓
Storage Layer
    │
    │ 4. 持久化
    ↓
Event Bus
    │
    │ 5. 发布事件
    ↓
Frontend (Vue)
    │
    │ 6. 更新 UI
    ↓
User
```

---

## 5. 技术栈

### 5.1 前端技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| **Vue 3** | 3.4+ | 响应式 UI 框架 |
| **Pinia** | 2.1+ | 状态管理 |
| **Naive UI** | 2.38+ | UI 组件库 |
| **highlight.js** | 11.9+ | 代码高亮 |
| **markdown-it** | 14.0+ | Markdown 渲染 |
| **@vueuse/core** | 10.7+ | Vue 组合式工具 |
| **vite** | 5.0+ | 构建工具 |

---

### 5.2 后端技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| **Go** | 1.21+ | 后端语言 |
| **Wails** | 2.8+ | 桌面应用框架 |
| **gorilla/websocket** | 1.5+ | WebSocket 实现 |
| **google/gopacket** | 1.1+ | 原始数据包处理 |
| **hashicorp/mdns** | 1.0+ | mDNS 服务发现 |
| **mattn/go-sqlite3** | 1.14+ | SQLite 驱动 |
| **golang-jwt/jwt** | 5.2+ | JWT 认证 |
| **golang.org/x/crypto** | latest | 加密算法 |

---

### 5.3 开发工具

| 工具 | 用途 |
|------|------|
| **Git** | 版本控制 |
| **VS Code** | 代码编辑器 |
| **Wireshark** | 网络抓包分析 |
| **Postman** | API 测试 |
| **SQLite Browser** | 数据库查看 |

---

## 6. 目录结构

### 6.1 完整目录树

```
CrossWire/
├── frontend/                   # Vue 前端
│   ├── src/
│   │   ├── assets/             # 静态资源
│   │   │   ├── images/
│   │   │   ├── styles/
│   │   │   └── fonts/
│   │   │
│   │   ├── components/         # UI 组件
│   │   │   ├── chat/
│   │   │   │   ├── MessageList.vue
│   │   │   │   ├── MessageInput.vue
│   │   │   │   ├── MessageItem.vue
│   │   │   │   └── CodeBlock.vue
│   │   │   ├── member/
│   │   │   │   ├── MemberList.vue
│   │   │   │   ├── MemberCard.vue
│   │   │   │   └── MemberProfile.vue
│   │   │   ├── file/
│   │   │   │   ├── FileUpload.vue
│   │   │   │   ├── FilePreview.vue
│   │   │   │   └── FileProgress.vue
│   │   │   └── common/
│   │   │       ├── Button.vue
│   │   │       ├── Modal.vue
│   │   │       └── Loading.vue
│   │   │
│   │   ├── views/              # 页面视图
│   │   │   ├── HomeView.vue
│   │   │   ├── ServerView.vue
│   │   │   ├── ClientView.vue
│   │   │   ├── ChatView.vue
│   │   │   └── SettingsView.vue
│   │   │
│   │   ├── stores/             # Pinia 状态
│   │   │   ├── app.js
│   │   │   ├── channel.js
│   │   │   ├── message.js
│   │   │   ├── member.js
│   │   │   └── file.js
│   │   │
│   │   ├── api/                # API 层
│   │   │   ├── wails.js
│   │   │   ├── message.js
│   │   │   ├── file.js
│   │   │   └── member.js
│   │   │
│   │   ├── utils/              # 工具函数
│   │   │   ├── format.js
│   │   │   ├── validate.js
│   │   │   └── crypto.js
│   │   │
│   │   ├── App.vue             # 根组件
│   │   └── main.js             # 入口文件
│   │
│   ├── public/
│   ├── package.json
│   ├── vite.config.js
│   └── index.html
│
├── backend/                    # Go 后端
│   ├── cmd/
│   │   └── crosswire/
│   │       └── main.go         # 程序入口
│   │
│   ├── internal/
│   │   ├── app/                # 应用层
│   │   │   ├── app.go
│   │   │   ├── events.go
│   │   │   └── lifecycle.go
│   │   │
│   │   ├── server/             # 服务端
│   │   │   ├── server.go
│   │   │   ├── channel.go
│   │   │   ├── broadcast.go
│   │   │   ├── auth.go
│   │   │   └── connection.go
│   │   │
│   │   ├── client/             # 客户端
│   │   │   ├── client.go
│   │   │   ├── connection.go
│   │   │   ├── sync.go
│   │   │   └── cache.go
│   │   │
│   │   ├── transport/          # 传输层
│   │   │   ├── transport.go
│   │   │   ├── arp_transport.go
│   │   │   ├── https_transport.go
│   │   │   ├── mdns_transport.go
│   │   │   └── factory.go
│   │   │
│   │   ├── crypto/             # 加密模块
│   │   │   ├── aes.go
│   │   │   ├── rsa.go
│   │   │   ├── keygen.go
│   │   │   └── signature.go
│   │   │
│   │   ├── storage/            # 存储模块
│   │   │   ├── database.go
│   │   │   ├── sqlite.go
│   │   │   ├── cache.go
│   │   │   └── migrations/
│   │   │       ├── 001_init.sql
│   │   │       ├── 002_add_threads.sql
│   │   │       └── 003_member_skills.sql
│   │   │
│   │   ├── discovery/          # 服务发现
│   │   │   ├── mdns.go
│   │   │   └── scanner.go
│   │   │
│   │   ├── models/             # 数据模型
│   │   │   ├── channel.go
│   │   │   ├── member.go
│   │   │   ├── message.go
│   │   │   ├── file.go
│   │   │   └── types.go
│   │   │
│   │   └── utils/              # 工具函数
│   │       ├── logger.go
│   │       ├── validator.go
│   │       └── helpers.go
│   │
│   ├── pkg/                    # 可导出包
│   │   └── protocol/
│   │       ├── message.go
│   │       └── frame.go
│   │
│   ├── go.mod
│   └── go.sum
│
├── build/                      # 构建资源
│   ├── windows/
│   │   ├── icon.ico
│   │   └── wails.json
│   ├── darwin/
│   │   ├── icon.icns
│   │   └── Info.plist
│   └── linux/
│       ├── icon.png
│       └── crosswire.desktop
│
├── docs/                       # 文档
│   ├── ARCHITECTURE.md         # 架构文档
│   ├── PROTOCOL.md             # 协议文档
│   ├── DATABASE.md             # 数据库文档
│   ├── FEATURES.md             # 功能文档
│   ├── API.md                  # API 文档
│   └── images/                 # 文档图片
│
├── scripts/                    # 脚本
│   ├── build.sh
│   ├── dev.sh
│   └── test.sh
│
├── .gitignore
├── wails.json                  # Wails 配置
├── README.md
└── LICENSE
```

---

### 6.2 关键文件说明

| 文件 | 说明 |
|------|------|
| `cmd/crosswire/main.go` | Go 程序入口 |
| `internal/app/app.go` | Wails 应用主类 |
| `internal/transport/transport.go` | 传输层接口定义 |
| `internal/models/*.go` | 数据模型定义 |
| `frontend/src/main.js` | Vue 应用入口 |
| `frontend/src/App.vue` | Vue 根组件 |
| `wails.json` | Wails 配置文件 |
| `docs/*.md` | 项目文档 |

---

## 7. 构建与打包

### 7.1 开发模式

```bash
# 启动开发服务器（热重载）
wails dev

# 或使用脚本
./scripts/dev.sh
```

### 7.2 生产构建

```bash
# 构建 Windows 版本
wails build -platform windows/amd64

# 构建 Linux 版本
wails build -platform linux/amd64

# 构建 macOS 版本
wails build -platform darwin/amd64

# 构建全平台
wails build -platform windows/amd64,linux/amd64,darwin/amd64
```

### 7.3 输出文件

```
build/bin/
├── CrossWire-windows-amd64.exe
├── CrossWire-linux-amd64
└── CrossWire-darwin-amd64.app
```

---

## 8. 性能指标

### 8.1 目标性能

| 指标 | ARP 模式 | HTTPS 模式 | mDNS 模式 |
|------|----------|------------|-----------|
| 消息延迟 | <5ms | <50ms | <2s |
| 文件传输速度 | 50-100 MB/s | 10-50 MB/s | 10-20 KB/s |
| 内存占用 | <200MB | <200MB | <200MB |
| CPU 占用 | <5% | <10% | <5% |
| 数据库大小 | <100MB/万条消息 | <100MB/万条消息 | <100MB/万条消息 |

---

### 8.2 扩展性

- **并发连接数**：50 客户端/服务端
- **消息吞吐量**：1000 消息/秒
- **文件传输并发**：10 个文件同时传输
- **数据库容量**：10 万条消息

---

## 9. 安全架构

### 9.1 安全层次

```
┌────────────────────────────────────┐
│  Transport Layer Security (TLS)    │  HTTPS 模式
├────────────────────────────────────┤
│  Message Encryption (X25519)  │  所有模式
├────────────────────────────────────┤
│  Authentication (JWT + Challenge)  │  所有模式
├────────────────────────────────────┤
│  Authorization (RBAC)              │  所有模式
└────────────────────────────────────┘
```

---

### 9.2 安全机制

- **传输层**：TLS 1.3（HTTPS 模式）
- **应用层**：X25519 加密
- **认证**：Challenge-Response + JWT
- **授权**：基于角色的权限控制
- **签名**：HMAC-SHA256 消息签名
- **密钥管理**：定期轮换、安全存储

---

## 10. 监控与日志

### 10.1 日志级别

```go
const (
    LogLevelDebug LogLevel = iota
    LogLevelInfo
    LogLevelWarn
    LogLevelError
    LogLevelFatal
)
```

### 10.2 日志输出

```
logs/
├── app.log           # 应用日志
├── transport.log     # 传输层日志
├── error.log         # 错误日志
└── audit.log         # 审计日志
```

---

## 11. CTF题目管理系统

### 11.1 系统架构

```
题目管理系统
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  ChallengeManager (题目管理器)                          │
│  ├─ CreateChallenge()     : 创建题目                   │
│  ├─ AssignChallenge()     : 分配题目给成员             │
│  ├─ SubmitFlag()          : 提交Flag验证               │
│  ├─ UpdateProgress()      : 更新解题进度               │
│  └─ GetChallengeRoom()    : 获取题目聊天室             │
│                                                         │
│  RoomManager (聊天室管理器)                             │
│  ├─ CreateChallengeRoom() : 创建题目聊天室             │
│  ├─ SendToRoom()          : 发送消息到聊天室           │
│  ├─ CheckRoomAccess()     : 检查聊天室访问权限         │
│  └─ ListRooms()           : 列出所有聊天室             │
│                                                         │
│  ProgressTracker (进度跟踪器)                           │
│  ├─ UpdateProgress()      : 更新进度                   │
│  ├─ GetMemberProgress()   : 获取成员进度               │
│  └─ GetTeamProgress()     : 获取团队整体进度           │
│                                                         │
│  FlagValidator (Flag验证器)                             │
│  ├─ ValidateFlag()        : 验证Flag                   │
│  ├─ RecordSubmission()    : 记录提交历史               │
│  └─ GetSubmissions()      : 获取提交记录               │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### 11.2 数据流

#### 11.2.1 题目创建流程

```
管理员 → CreateChallenge()
         ↓
    验证权限
         ↓
    生成题目UUID
         ↓
    存储到数据库
         ↓
    自动创建题目聊天室
         ↓
    广播系统消息
         ↓
    返回题目信息
```

#### 11.2.2 题目分配流程

```
管理员 → AssignChallenge(challenge_id, member_ids[])
         ↓
    验证权限
         ↓
    批量创建分配记录
         ↓
    更新聊天室访问权限
         ↓
    通知被分配成员
         ↓
    广播系统消息
```

#### 11.2.3 Flag提交流程（协作平台：不验证，明文存储）

```
成员 → SubmitFlag(challenge_id, flag)
       ↓
   验证访问权限
       ↓
   记录提交（明文）
       ↓
   更新题目状态/进度
       ↓
   广播系统消息
       ↓
   返回结果
```

### 11.3 模块接口

#### 11.3.1 ChallengeManager

```go
package challenge

type ChallengeManager interface {
    // 题目管理
    CreateChallenge(config *ChallengeConfig) (*Challenge, error)
    UpdateChallenge(id string, updates *ChallengeUpdate) error
    DeleteChallenge(id string) error
    GetChallenge(id string) (*Challenge, error)
    ListChallenges(channelID string, filter *ChallengeFilter) ([]*Challenge, error)
    
    // 题目分配
    AssignChallenge(challengeID string, memberIDs []string, role string) error
    UnassignChallenge(challengeID string, memberID string) error
    GetAssignments(challengeID string) ([]*Assignment, error)
    GetMemberChallenges(memberID string) ([]*Challenge, error)
    
    // Flag管理
    SubmitFlag(challengeID, memberID, flag string) (*SubmissionResult, error)
    GetSubmissions(challengeID string) ([]*Submission, error)
    
    // 提示管理
    AddHint(challengeID string, content string, cost int) error
    UnlockHint(hintID, memberID string) error
    GetHints(challengeID string) ([]*Hint, error)
}
```

#### 11.3.2 RoomManager

```go
package challenge

type RoomManager interface {
    // 聊天室管理
    CreateChallengeRoom(challengeID string) error
    DeleteChallengeRoom(challengeID string) error
    
    // 消息管理
    SendToRoom(challengeID string, msg *Message) error
    GetRoomMessages(challengeID string, limit int) ([]*Message, error)
    
    // 权限管理
    CheckRoomAccess(challengeID, memberID string) bool
    UpdateRoomAccess(challengeID string, memberIDs []string) error
    
    // 查询
    ListRooms(channelID string) ([]*ChallengeRoom, error)
}
```

#### 11.3.3 ProgressTracker

```go
package challenge

type ProgressTracker interface {
    // 进度管理
    UpdateProgress(challengeID, memberID string, progress int, summary string) error
    GetMemberProgress(challengeID, memberID string) (*Progress, error)
    GetTeamProgress(challengeID string) ([]*Progress, error)
    
    // 统计
    GetOverallStats(channelID string) (*ChallengeStats, error)
    GetMemberStats(memberID string) (*MemberStats, error)
}
```

### 11.4 前端集成

#### 11.4.1 Vue 组件结构

```
frontend/src/components/Challenge/
├── ChallengeList.vue         # 题目列表
├── ChallengeCard.vue         # 题目卡片
├── ChallengeDetail.vue       # 题目详情
├── ChallengeCreate.vue       # 创建题目
├── ChallengeAssign.vue       # 分配题目
├── ChallengeProgress.vue     # 进度显示
├── ChallengeSubmit.vue       # 提交Flag
└── ChallengeRoom.vue         # 题目聊天室

frontend/src/views/
└── ChallengeView.vue         # 题目管理主视图

frontend/src/stores/
└── challengeStore.ts         # 题目状态管理
```

#### 11.4.2 API调用示例

```typescript
// stores/challengeStore.ts
import { defineStore } from 'pinia';

export const useChallengeStore = defineStore('challenge', {
  state: () => ({
    challenges: [],
    currentChallenge: null,
    assignments: [],
    progress: {},
  }),
  
  actions: {
    async createChallenge(config: ChallengeConfig) {
      const result = await window.go.app.CreateChallenge(config);
      if (result.success) {
        this.challenges.push(result.challenge);
      }
      return result;
    },
    
    async assignChallenge(challengeId: string, memberIds: string[]) {
      const result = await window.go.app.AssignChallenge(challengeId, memberIds);
      if (result.success) {
        this.assignments = result.assignments;
      }
      return result;
    },
    
    async submitFlag(challengeId: string, flag: string) {
      const result = await window.go.app.SubmitFlag(challengeId, flag);
      if (result.is_correct) {
        // 更新题目状态
        const challenge = this.challenges.find(c => c.id === challengeId);
        if (challenge) {
          challenge.status = 'solved';
        }
      }
      return result;
    },
    
    async updateProgress(challengeId: string, progress: number, summary: string) {
      const result = await window.go.app.UpdateProgress(challengeId, progress, summary);
      this.progress[challengeId] = result.progress;
      return result;
    },
  },
});
```

### 11.5 权限控制

| 操作 | Owner | Admin | Member |
|------|-------|-------|--------|
| 创建题目 | ✅ | ✅ | ❌ |
| 编辑题目 | ✅ | ✅ | ❌ |
| 删除题目 | ✅ | ✅ | ❌ |
| 分配题目 | ✅ | ✅ | ❌ |
| 查看所有题目 | ✅ | ✅ | ✅ (仅已分配) |
| 提交Flag | ✅ | ✅ | ✅ (仅已分配) |
| 更新进度 | ✅ | ✅ | ✅ (仅已分配) |
| 查看题目聊天室 | ✅ | ✅ | ✅ (仅已分配) |
| 添加提示 | ✅ | ✅ | ❌ |
| 解锁提示 | ✅ | ✅ | ✅ (仅已分配) |

### 11.6 安全考虑

#### 11.6.1 Flag存储

- ❌ **不存储明文Flag**：只存储 SHA256 哈希
- ✅ **提交记录加密**：历史提交的Flag加密存储
- ✅ **哈希加盐**：使用题目ID作为盐值

> 协作平台不进行Flag哈希或验证，以上示例已废弃。

#### 11.6.2 聊天室隔离

- ✅ **权限检查**：每次发送消息前验证权限
- ✅ **消息标记**：`room_type` 字段区分聊天室
- ✅ **访问控制**：只有被分配的成员才能访问

```go
func (s *Server) SendToChallengeRoom(challengeID string, msg *Message) error {
    // 获取有权限的成员
    assignments, _ := s.db.GetAssignments(challengeID)
    
    // 只发送给有权限的成员
    for _, assignment := range assignments {
        s.SendToClient(assignment.MemberID, msg)
    }
    
    return nil
}
```

---

## 总结

CrossWire 采用**分层架构 + 模块化设计**，实现了：

✅ **清晰的职责划分**：每层专注于特定功能  
✅ **高度可扩展性**：插件化传输层  
✅ **良好的可维护性**：模块解耦、接口抽象  
✅ **跨平台支持**：Wails 框架保证一致体验  
✅ **安全可靠**：多层加密 + 权限控制  

---

**相关文档：**
- [FEATURES.md](FEATURES.md) - 功能详细说明
- [PROTOCOL.md](PROTOCOL.md) - 通信协议规范
- [DATABASE.md](DATABASE.md) - 数据库设计