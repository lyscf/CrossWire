# Client 客户端模块

> CrossWire 客户端核心实现

## 📑 目录

- [概述](#概述)
- [模块架构](#模块架构)
- [核心组件](#核心组件)
- [使用示例](#使用示例)
- [配置说明](#配置说明)

---

## 概述

Client 模块负责客户端的核心功能，包括：

- **连接管理**：加入/离开频道
- **消息接收**：监听广播、解密、去重
- **消息同步**：增量同步、离线消息
- **本地缓存**：消息/成员/文件缓存

### 架构特点

根据服务器签名广播模式：

- ✅ 无需维护长连接（监听广播即可）
- ✅ 无需心跳机制（通过消息活跃度判断）
- ✅ 无需断线重连（始终监听）
- ✅ 简化的认证流程

---

## 模块架构

```
Client
  ├─ ReceiveManager    : 接收管理器
  │   ├─ 监听广播帧
  │   ├─ 解密过滤
  │   └─ 消息去重
  │
  ├─ SyncManager       : 同步管理器
  │   ├─ 增量同步
  │   ├─ 离线消息
  │   └─ 冲突解决
  │
  └─ CacheManager      : 缓存管理器
      ├─ 消息缓存
      ├─ 文件缓存
      └─ 成员缓存
```

---

## 核心组件

### 1. Client (client.go)

**职责**：客户端核心，协调各子管理器

**主要方法**：

```go
// 生命周期
func NewClient(config *Config, db *storage.Database, eventBus *events.EventBus) (*Client, error)
func (c *Client) Start() error
func (c *Client) Stop() error

// 频道操作
func (c *Client) joinChannel() error
func (c *Client) leaveChannel()

// 消息操作
func (c *Client) SendMessage(content string, msgType models.MessageType) error
func (c *Client) GetMessages(limit int, offset int) ([]*models.Message, error)

// 成员操作
func (c *Client) GetMembers() ([]*models.Member, error)

// 状态查询
func (c *Client) GetStats() ClientStats
func (c *Client) IsRunning() bool
```

**启动流程**：

```
1. 初始化传输层
2. 发送加入请求
3. 启动 ReceiveManager
4. 启动 SyncManager
5. 启动 CacheManager
```

---

### 2. ReceiveManager (receive_manager.go)

**职责**：接收和处理传输层消息

**核心功能**：

- **监听广播**：订阅传输层消息
- **解密过滤**：解密消息，验证频道ID
- **消息去重**：防止重复处理
- **事件分发**：发布消息到EventBus

**消息处理流程**：

```go
1. 接收 transport.Message
2. 解密 Payload
3. 根据 Type 路由：
   - MessageTypeAuth    -> handleAuthMessage
   - MessageTypeData    -> handleDataMessage
   - MessageTypeControl -> handleControlMessage
4. 去重检查
5. 保存到数据库
6. 发布事件
```

**去重机制**：

- 维护 `seenMessages` map
- 记录最近 10000 条消息ID
- 定期清理超过 1 小时的记录

---

### 3. SyncManager (sync_manager.go)

**职责**：同步离线消息和成员信息

**核心功能**：

- **定期同步**：按配置间隔自动同步
- **主动同步**：加入频道后立即同步
- **增量同步**：只同步新消息
- **冲突解决**：Last-Write-Wins 策略

**同步流程**：

```go
1. 构造 sync.request
   - last_message_id
   - last_timestamp
   - limit
2. 发送请求到服务端
3. 接收 sync.response
4. 处理消息列表
5. 处理成员列表
6. 更新 lastSyncTime
```

**冲突解决**：

```go
func shouldUpdate(local, remote *Message) bool {
    // 1. 比较时间戳
    if remote.Timestamp != local.Timestamp {
        return remote.Timestamp > local.Timestamp
    }
    // 2. 时间戳相同，比较ID
    return remote.ID > local.ID
}
```

---

### 4. CacheManager (cache_manager.go)

**职责**：管理本地缓存，提升性能

**核心功能**：

- **消息缓存**：最近 N 条消息（LRU）
- **成员缓存**：频道所有成员
- **文件缓存**：文件元数据
- **定期清理**：移除过期缓存

**缓存策略**：

```go
// 消息缓存
- 容量：config.CacheSize (默认 5000)
- 淘汰：LRU（最旧的消息）
- 过期：config.CacheDuration (默认 24 小时)

// 成员缓存
- 容量：无限制
- 淘汰：手动移除（成员离开时）

// 文件缓存
- 容量：无限制
- 淘汰：按需清理
```

**使用示例**：

```go
// 先查缓存，未命中再查数据库
msg, err := cacheManager.GetMessage(messageID)

// 主动放入缓存
cacheManager.PutMessage(msg)

// 批量获取
messages := cacheManager.GetMessages(100)
```

---

## 使用示例

### 创建和启动客户端

```go
package main

import (
    "crosswire/internal/client"
    "crosswire/internal/storage"
    "crosswire/internal/events"
    "crosswire/internal/models"
    "crosswire/internal/transport"
)

func main() {
    // 1. 准备配置
    config := &client.Config{
        ChannelID:       "channel-uuid-123",
        ChannelPassword: "my-secret-password",
        Nickname:        "Alice",
        Avatar:          "",
        Role:            models.RoleMember,
        TransportMode:   models.TransportHTTPS,
        TransportConfig: &transport.Config{
            ListenAddr: ":0",
            ServerAddr: "192.168.1.100:8443",
        },
        DataDir: "./data",
    }

    // 2. 初始化数据库
    db, err := storage.NewDatabase("./data")
    if err != nil {
        panic(err)
    }

    // 3. 创建事件总线
    eventBus := events.NewEventBus()

    // 4. 创建客户端
    c, err := client.NewClient(config, db, eventBus)
    if err != nil {
        panic(err)
    }

    // 5. 启动客户端
    if err := c.Start(); err != nil {
        panic(err)
    }

    defer c.Stop()
}
```

### 发送消息

```go
// 发送文本消息
err := client.SendMessage("Hello, world!", models.MessageTypeText)

// 发送代码消息
err := client.SendMessage("```go\nfunc main() {}\n```", models.MessageTypeCode)
```

### 订阅事件

```go
// 订阅消息接收事件
eventBus.Subscribe(events.EventTypeMessageReceived, func(data interface{}) {
    event := data.(*events.MessageEvent)
    fmt.Printf("New message: %s\n", event.Message.ID)
})

// 订阅成员加入事件
eventBus.Subscribe(events.EventTypeMemberJoined, func(data interface{}) {
    event := data.(*events.MemberEvent)
    fmt.Printf("Member joined: %s\n", event.Member.Nickname)
})
```

### 查询消息

```go
// 获取最近 100 条消息
messages, err := client.GetMessages(100, 0)

// 从缓存获取单条消息
msg, err := client.cacheManager.GetMessage(messageID)
```

### 查询成员

```go
// 获取所有成员
members, err := client.GetMembers()

// 从缓存获取成员
member, err := client.cacheManager.GetMember(memberID)
```

---

## 配置说明

### Config 结构

```go
type Config struct {
    // 频道信息
    ChannelID       string  // 频道UUID
    ChannelPassword string  // 频道密码

    // 用户信息
    Nickname string        // 昵称
    Avatar   string        // 头像URL
    Role     models.Role   // 角色

    // 传输配置
    TransportMode   models.TransportMode  // 传输模式
    TransportConfig *transport.Config     // 传输配置

    // 同步配置
    SyncInterval    time.Duration  // 同步间隔（默认：5分钟）
    MaxSyncMessages int            // 单次同步最大消息数（默认：1000）

    // 缓存配置
    CacheSize     int            // 缓存大小（默认：5000）
    CacheDuration time.Duration  // 缓存有效期（默认：24小时）

    // 超时配置
    JoinTimeout time.Duration  // 加入超时（默认：30秒）
    SyncTimeout time.Duration  // 同步超时（默认：10秒）

    // 数据库路径
    DataDir string  // 数据目录
}
```

### 默认配置

```go
config := client.DefaultConfig()
// 然后覆盖需要的字段
config.ChannelID = "my-channel"
config.ChannelPassword = "my-password"
config.Nickname = "Alice"
```

---

## 状态管理

### 客户端状态

```go
// 检查运行状态
if client.IsRunning() {
    // 客户端正在运行
}

// 获取成员ID
memberID := client.GetMemberID()
```

### 统计信息

```go
stats := client.GetStats()
fmt.Printf("Connected at: %v\n", stats.ConnectedAt)
fmt.Printf("Messages received: %d\n", stats.MessagesReceived)
fmt.Printf("Messages sent: %d\n", stats.MessagesSent)
fmt.Printf("Sync count: %d\n", stats.SyncCount)
fmt.Printf("Last sync: %v\n", stats.LastSyncTime)

// 接收管理器统计
receiveStats := client.receiveManager.GetStats()
fmt.Printf("Duplicate messages: %d\n", receiveStats.DuplicateMessages)

// 同步管理器统计
syncStats := client.syncManager.GetStats()
fmt.Printf("Synced messages: %d\n", syncStats.MessagesSynced)

// 缓存管理器统计
cacheStats := client.cacheManager.GetStats()
fmt.Printf("Cache hits: %d\n", cacheStats.CacheHits)
fmt.Printf("Cache misses: %d\n", cacheStats.CacheMisses)
```

---

## 事件系统

### 发布的事件

| 事件类型 | 数据类型 | 说明 |
|---------|---------|------|
| `EventTypeMessageReceived` | `*events.MessageEvent` | 接收到新消息 |
| `EventTypeMemberJoined` | `*events.MemberEvent` | 成员加入 |
| `EventTypeMemberLeft` | `*events.MemberEvent` | 成员离开 |
| `EventTypeMemberStatusChanged` | `*events.MemberEvent` | 成员状态变化 |
| `EventTypeError` | `map[string]interface{}` | 错误事件 |

### 订阅示例

```go
// 订阅多个事件
client.eventBus.Subscribe(events.EventTypeMessageReceived, handleMessage)
client.eventBus.Subscribe(events.EventTypeMemberJoined, handleMemberJoin)
client.eventBus.Subscribe(events.EventTypeError, handleError)
```

---

## TODO

- [ ] 实现文件传输功能
- [ ] 实现 Challenge 相关客户端逻辑
- [ ] 优化缓存淘汰策略
- [ ] 添加消息搜索功能
- [ ] 实现离线队列
- [ ] 添加网络状态检测
- [ ] 实现请求-响应匹配机制
- [ ] 添加详细的错误处理

---

## 参考文档

- [ARCHITECTURE.md](../../docs/ARCHITECTURE.md) - 系统架构
- [PROTOCOL.md](../../docs/PROTOCOL.md) - 通信协议
- [ARP_BROADCAST_MODE.md](../../docs/ARP_BROADCAST_MODE.md) - 广播模式设计

