# Events 事件总线系统

> CrossWire事件总线实现

---

## 📚 概述

EventBus是一个高性能、线程安全的事件发布-订阅系统，用于解耦系统各模块之间的通信。

**参考文档**: `docs/ARCHITECTURE.md` - 3.2.2 事件总线

---

## 🏗️ 架构设计

### 核心组件

```
┌─────────────────────────────────────────┐
│            EventBus                      │
├─────────────────────────────────────────┤
│  • 订阅管理 (Subscribers)                │
│  • 事件队列 (Event Queue)                │
│  • 工作协程 (Workers)                    │
│  • 统计信息 (Stats)                      │
└─────────────────────────────────────────┘
         │
         ├─→ 订阅者 (Subscriber 1)
         ├─→ 订阅者 (Subscriber 2)
         └─→ 订阅者 (Subscriber N)
```

### 事件流

```
Publisher → EventBus.Publish() → Event Queue → Worker → Handler 1
                                              ↓
                                              → Handler 2
                                              ↓
                                              → Handler N
```

---

## 📦 功能特性

### ✅ 已实现

- ✅ 发布-订阅模式
- ✅ 异步事件处理
- ✅ 事件过滤器
- ✅ 全局订阅（监听所有事件）
- ✅ 并发安全
- ✅ 超时保护
- ✅ Panic恢复
- ✅ 统计信息
- ✅ 多工作协程
- ✅ 事件队列
- ✅ 同步/异步发布

### ⏳ 待实现

- ⏳ 事件持久化
- ⏳ 事件重播
- ⏳ 优先级队列
- ⏳ 事件过滤器DSL
- ⏳ 事件链追踪

---

## 🎯 事件类型

### 消息相关

```go
EventMessageReceived  // 收到新消息
EventMessageSent      // 消息已发送
EventMessageDeleted   // 消息被删除
EventMessagePinned    // 消息被置顶
EventMessageEdited    // 消息被编辑
```

### 成员相关

```go
EventMemberJoined     // 成员加入
EventMemberLeft       // 成员离开
EventMemberKicked     // 成员被踢出
EventMemberMuted      // 成员被禁言
EventMemberUnmuted    // 成员解除禁言
```

### 状态相关

```go
EventStatusChanged    // 状态变化
EventTypingStart      // 开始输入
EventTypingStop       // 停止输入
```

### 文件相关

```go
EventFileUploaded     // 文件上传完成
EventFileDownloaded   // 文件下载完成
EventFileProgress     // 文件传输进度
```

### 频道相关

```go
EventChannelCreated   // 频道创建
EventChannelJoined    // 加入频道
EventChannelLeft      // 离开频道
EventChannelUpdated   // 频道信息更新
```

### 系统相关

```go
EventSystemError      // 系统错误
EventSystemConnected  // 连接成功
EventSystemDisconnect // 连接断开
EventSystemReconnect  // 重新连接
```

### CTF挑战相关

```go
EventChallengeCreated   // 题目创建
EventChallengeAssigned  // 题目分配
EventChallengeSubmitted // Flag提交
EventChallengeSolved    // 题目完成
EventChallengeHintUnlock // 提示解锁
```

---

## 💡 使用示例

### 1. 创建EventBus

```go
// 使用默认配置
eventBus := events.NewEventBus(nil)

// 使用自定义配置
config := &events.EventBusConfig{
    QueueSize:      2000,
    WorkerCount:    8,
    EnableAsync:    true,
    EnableStats:    true,
    HandlerTimeout: 3 * time.Second,
}
eventBus := events.NewEventBus(config)
```

### 2. 订阅事件

```go
// 订阅特定事件
subID := eventBus.Subscribe(events.EventMessageReceived, func(event *events.Event) {
    msgEvent := event.Data.(*events.MessageEvent)
    fmt.Printf("收到消息: %s\n", msgEvent.Message.Content)
})

// 带过滤器订阅
subID := eventBus.SubscribeWithFilter(
    events.EventMessageReceived,
    func(event *events.Event) {
        // 处理事件
    },
    func(event *events.Event) bool {
        // 只处理文本消息
        msgEvent := event.Data.(*events.MessageEvent)
        return msgEvent.Message.Type == models.MessageTypeText
    },
)

// 订阅所有事件
subID := eventBus.SubscribeAll(func(event *events.Event) {
    fmt.Printf("事件: %s, 时间: %s\n", event.Type, event.Timestamp)
})
```

### 3. 发布事件

```go
// 异步发布（默认）
eventBus.Publish(events.EventMessageReceived, &events.MessageEvent{
    Message:   msg,
    ChannelID: channelID,
    SenderID:  senderID,
})

// 同步发布（阻塞）
eventBus.PublishSync(events.EventMemberJoined, &events.MemberEvent{
    Member:    member,
    ChannelID: channelID,
    Action:    "joined",
})

// 带来源发布
eventBus.PublishWithSource(
    events.EventSystemError,
    &events.SystemEvent{Message: "Connection failed"},
    "transport_layer",
)
```

### 4. 使用辅助函数

```go
// 使用预定义的事件创建函数
event := events.NewMessageReceivedEvent(msg, channelID)
eventBus.Publish(event.Type, event.Data)

event := events.NewMemberJoinedEvent(member, channelID)
eventBus.Publish(event.Type, event.Data)

event := events.NewFileUploadedEvent(file, channelID, uploaderID)
eventBus.Publish(event.Type, event.Data)
```

### 5. 取消订阅

```go
// 取消单个订阅
success := eventBus.Unsubscribe(subID)

// 取消某类型的所有订阅
eventBus.UnsubscribeAll(events.EventMessageReceived)
```

### 6. 查询信息

```go
// 获取订阅者数量
count := eventBus.GetSubscriberCount(events.EventMessageReceived)

// 检查是否有订阅者
hasSubscribers := eventBus.HasSubscribers(events.EventFileUploaded)

// 获取统计信息
stats := eventBus.GetStats()
fmt.Printf("已发布: %d, 已处理: %d, 丢弃: %d\n",
    stats.EventsPublished,
    stats.EventsProcessed,
    stats.EventsDropped,
)
```

### 7. 关闭EventBus

```go
// 优雅关闭（等待队列处理完成）
eventBus.Close()
```

---

## 🔧 配置说明

### EventBusConfig

```go
type EventBusConfig struct {
    QueueSize      int           // 事件队列大小（默认: 1000）
    WorkerCount    int           // 工作协程数（默认: 4）
    EnableAsync    bool          // 启用异步处理（默认: true）
    EnableStats    bool          // 启用统计（默认: true）
    MaxRetries     int           // 最大重试次数（默认: 3）
    RetryDelay     time.Duration // 重试延迟（默认: 100ms）
    HandlerTimeout time.Duration // 处理超时（默认: 5s）
    EnableWildcard bool          // 启用通配符（默认: false）
}
```

### 默认配置

```go
DefaultConfig = &EventBusConfig{
    QueueSize:      1000,
    WorkerCount:    4,
    EnableAsync:    true,
    EnableStats:    true,
    MaxRetries:     3,
    RetryDelay:     100 * time.Millisecond,
    HandlerTimeout: 5 * time.Second,
    EnableWildcard: false,
}
```

---

## 📊 统计信息

### EventBusStats

```go
type EventBusStats struct {
    EventsPublished uint64        // 发布的事件数
    EventsProcessed uint64        // 处理的事件数
    EventsDropped   uint64        // 丢弃的事件数
    Subscriptions   int           // 订阅数
    HandlerErrors   uint64        // 处理错误数
    AverageLatency  time.Duration // 平均延迟
    StartTime       time.Time     // 启动时间
    LastEventTime   time.Time     // 最后事件时间
}
```

---

## 🎨 设计模式

### 1. 发布-订阅模式

解耦事件发布者和订阅者，发布者不需要知道谁在监听事件。

### 2. 异步处理

使用工作协程池异步处理事件，避免阻塞发布者。

### 3. 过滤器模式

支持自定义过滤器，灵活控制事件处理。

---

## 🔐 并发安全

### 保护机制

1. **读写锁**: 保护订阅者列表
2. **事件队列**: 使用channel实现线程安全队列
3. **统计锁**: 保护统计数据
4. **Panic恢复**: 防止单个处理函数崩溃影响整个系统
5. **超时控制**: 防止处理函数长时间阻塞

---

## 🚀 性能优化

### 1. 批量处理

使用事件队列批量处理，减少锁竞争。

### 2. 工作协程池

预创建多个工作协程，避免频繁创建销毁。

### 3. 异步发布

默认异步发布，不阻塞业务逻辑。

### 4. 订阅者拷贝

调用前拷贝订阅者列表，减少锁持有时间。

---

## 🧪 测试示例

```go
func TestEventBus(t *testing.T) {
    eb := events.NewEventBus(nil)
    defer eb.Close()
    
    // 测试订阅
    received := make(chan bool, 1)
    eb.Subscribe(events.EventMessageReceived, func(event *events.Event) {
        received <- true
    })
    
    // 测试发布
    eb.Publish(events.EventMessageReceived, &events.MessageEvent{})
    
    // 验证
    select {
    case <-received:
        t.Log("事件已处理")
    case <-time.After(1 * time.Second):
        t.Error("超时")
    }
}
```

---

## 📈 使用场景

### 1. 消息接收流程

```go
// Transport层接收到消息
transport.Subscribe(func(msg *transport.Message) {
    // 解密消息
    decrypted := crypto.Decrypt(msg.Payload)
    
    // 保存到数据库
    db.SaveMessage(decrypted)
    
    // 发布事件到前端
    eventBus.Publish(events.EventMessageReceived, &events.MessageEvent{
        Message:   decrypted,
        ChannelID: channelID,
    })
})
```

### 2. 成员管理

```go
// 成员加入
func (s *Server) OnMemberJoin(member *models.Member) {
    // 保存成员
    db.AddMember(member)
    
    // 发布事件
    eventBus.Publish(events.EventMemberJoined, &events.MemberEvent{
        Member:    member,
        ChannelID: s.channelID,
    })
}
```

### 3. 前端通知

```go
// App层订阅事件并通知前端
eventBus.Subscribe(events.EventMessageReceived, func(event *events.Event) {
    msgEvent := event.Data.(*events.MessageEvent)
    
    // 通过Wails发送到前端
    runtime.EventsEmit(ctx, "new:message", msgEvent)
})
```

---

## 📖 相关文档

- [ARCHITECTURE.md](../../docs/ARCHITECTURE.md) - 系统架构
- [PROTOCOL.md](../../docs/PROTOCOL.md) - 通信协议

---

**最后更新**: 2025-10-05  
**状态**: ✅ 完整实现并可用

