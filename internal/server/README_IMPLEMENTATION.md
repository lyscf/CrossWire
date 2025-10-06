# Server 模块实现说明

> 详细说明 server.go 中已实现的功能及其位置

## 📑 功能清单

### ✅ 1. 消息持久化到离线队列

**实现位置**: `internal/server/offline_manager.go`

**核心方法**:
- `OfflineManager.StoreOfflineMessage()` - 存储离线消息
- `OfflineManager.DeliverOfflineMessages()` - 投递离线消息
- `OfflineManager.CleanupOldMessages()` - 清理过期消息

**功能特性**:
- ✓ 自动持久化到数据库，确保消息不丢失
- ✓ 队列满时自动删除最旧消息（FIFO策略）
- ✓ 定期清理过期消息（默认7天）
- ✓ 成员上线时自动投递离线消息

**配置参数**:
```go
maxQueueSize: 1000          // 每个成员最多1000条离线消息
maxAge:       7*24*time.Hour // 7天自动过期
```

**使用示例**:
```go
// 存储离线消息
server.offlineManager.StoreOfflineMessage(memberID, message)

// 成员上线时投递
server.DeliverOfflineMessagesToMember(memberID)

// 获取统计
stats := server.GetOfflineMessageStats()
```

---

### ✅ 2. 消息确认（ACK）机制

**实现位置**: 
- `internal/server/server.go` - `handleMessageAck()`
- `internal/server/broadcast_manager.go` - `RecordAck()`, `GetAckCount()`

**核心方法**:
- `Server.handleMessageAck()` - 处理ACK消息
- `BroadcastManager.RecordAck()` - 记录ACK
- `BroadcastManager.GetAckCount()` - 获取ACK数量

**功能特性**:
- ✓ 记录每个成员的消息确认状态
- ✓ 支持查询单个消息的ACK数量
- ✓ 自动验证ACK发送者身份
- ✓ 防止伪造ACK（需要频道密钥解密）

**协议格式**:
```json
{
  "type": "ack",
  "message_id": "msg_xxx",
  "member_id": "member_xxx",
  "timestamp": 1696512000
}
```

**使用示例**:
```go
// 获取消息的ACK数量
ackCount := server.broadcastManager.GetAckCount(messageID)

// 检查是否所有在线成员都ACK了
onlineCount := server.channelManager.GetOnlineCount()
if ackCount >= onlineCount {
    fmt.Println("所有在线成员已确认")
}
```

---

### ✅ 3. 频率限制细节

**实现位置**: `internal/server/message_router.go` - `RateLimiter`

**核心方法**:
- `RateLimiter.Allow()` - 检查是否允许发送
- `RateLimiter.Reset()` - 重置频率限制

**功能特性**:
- ✓ 滑动窗口算法（1分钟窗口）
- ✓ 每个成员独立计数
- ✓ 自动清理过期时间戳
- ✓ 可配置最大速率

**算法原理**:
```
1. 记录每个成员的消息时间戳列表
2. 检查时过滤掉1分钟之前的记录
3. 如果有效记录数 >= 限制，则拒绝
4. 否则添加新时间戳并允许
```

**配置参数**:
```go
maxRate: 60                 // 每分钟最多60条消息
window:  1 * time.Minute    // 时间窗口
```

**使用示例**:
```go
// 在消息路由中自动检查
if server.config.EnableRateLimit {
    if !rateLimiter.Allow(memberID) {
        // 拒绝消息
    }
}
```

---

### ✅ 4. 反垃圾消息

**实现位置**: 
- `internal/server/spam_detector.go` - 完整的垃圾消息检测器
- `internal/server/message_router.go` - 集成检测

**核心方法**:
- `SpamDetector.CheckMessage()` - 检查消息是否为垃圾
- `SpamDetector.checkBlacklist()` - 黑名单检测
- `SpamDetector.checkDuplicate()` - 重复消息检测
- `SpamDetector.checkMemberHistory()` - 成员历史检测

**检测维度**:
1. **黑名单关键词过滤**
   - 检查消息是否包含敏感词
   - 支持动态添加/删除关键词
   - 不区分大小写匹配

2. **重复消息检测**
   - 使用SHA256指纹识别
   - 5分钟时间窗口内的重复消息
   - 全局去重

3. **快速连发检测**
   - 检查成员历史消息
   - 限制相同消息数量
   - 防止刷屏

4. **签名验证**（在MessageRouter中）
   - Ed25519签名验证
   - 防止消息伪造
   - 确保消息完整性

**配置参数**:
```go
EnableDuplicateDetection: true
EnableContentFilter:      true
EnableRapidPostDetection: true
MaxDuplicateWindow:       5 * time.Minute
MaxSimilarInHistory:      3
MaxHistorySize:           20
```

**使用示例**:
```go
// 检查消息
if isSpam, reason := server.spamDetector.CheckMessage(msg, senderID); isSpam {
    // 拒绝垃圾消息
}

// 管理黑名单
server.AddBlacklistWord("垃圾内容")
server.RemoveBlacklistWord("正常内容")
words := server.GetBlacklistWords()

// 获取统计
stats := server.GetSpamDetectorStats()
```

---

### ✅ 5. 成员踢出与封禁

**实现位置**: `internal/server/channel_manager.go`

**核心方法**:
- `ChannelManager.KickMember()` - 踢出成员
- `ChannelManager.BanMember()` - 封禁成员
- `ChannelManager.UnbanMember()` - 解封成员
- `ChannelManager.IsBanned()` - 检查封禁状态

**功能特性**:
1. **踢出功能**
   - 立即从频道移除
   - 通知其他成员
   - 记录踢出理由和操作者
   - 发送系统消息

2. **封禁功能**
   - 临时封禁（有过期时间）
   - 永久封禁（无过期时间）
   - 自动检测过期并解封
   - 封禁期间无法加入

3. **权限检查**
   - 验证操作者权限
   - 防止自我封禁
   - 记录操作日志

**使用示例**:
```go
// 踢出成员
err := server.KickMember(memberID, "违反规则")

// 临时封禁（24小时）
err := server.BanMember(memberID, "spam", 24*time.Hour)

// 永久封禁
err := server.BanMember(memberID, "恶意攻击", 0)

// 解封
err := server.UnbanMember(memberID)

// 检查封禁状态
isBanned := server.channelManager.IsBanned(memberID)
```

---

### ✅ 6. 权限分级（管理员/普通成员）

**实现位置**: 
- `internal/server/auth_manager.go` - `CheckPermission()`
- `internal/server/server.go` - 便捷方法

**角色层级**:
```
Owner (所有者)
  ↓
Admin (管理员)
  ↓
Moderator (协管)
  ↓
Member (普通成员)
  ↓
ReadOnly (只读)
```

**核心方法**:
- `Server.CheckPermission()` - 通用权限检查
- `Server.HasAdminPermission()` - 检查管理员权限
- `Server.HasModeratorPermission()` - 检查管理权限
- `ChannelManager.UpdateMemberRole()` - 更新成员角色

**权限矩阵**:

| 操作 | Owner | Admin | Moderator | Member | ReadOnly |
|------|-------|-------|-----------|--------|----------|
| 发送消息 | ✓ | ✓ | ✓ | ✓ | ✗ |
| 删除消息 | ✓ | ✓ | ✓ | 自己的 | ✗ |
| 踢出成员 | ✓ | ✓ | ✓ | ✗ | ✗ |
| 封禁成员 | ✓ | ✓ | ✗ | ✗ | ✗ |
| 修改频道设置 | ✓ | ✓ | ✗ | ✗ | ✗ |
| 设置管理员 | ✓ | ✗ | ✗ | ✗ | ✗ |

**使用示例**:
```go
// 检查管理员权限
if server.HasAdminPermission(memberID) {
    // 执行管理操作
}

// 检查管理权限（包括协管）
if server.HasModeratorPermission(memberID) {
    // 执行管理操作
}

// 检查特定角色
if server.CheckPermission(memberID, models.RoleAdmin) {
    // 仅管理员可执行
}

// 更新角色
err := server.UpdateMemberRole(memberID, models.RoleModerator)
```

---

## 📊 统计与监控

### 获取各模块统计信息

```go
// 服务器统计
serverStats := server.GetStats()
fmt.Printf("在线成员: %d/%d\n", serverStats.OnlineMembers, serverStats.TotalMembers)
fmt.Printf("总消息数: %d\n", serverStats.TotalMessages)
fmt.Printf("总广播数: %d\n", serverStats.TotalBroadcasts)

// 离线消息统计
offlineStats := server.GetOfflineMessageStats()
fmt.Printf("待投递: %d\n", offlineStats["current_queued_count"])
fmt.Printf("已投递: %d\n", offlineStats["total_delivered"])

// 广播统计
broadcastStats := server.GetBroadcastStats()
fmt.Printf("平均延迟: %s\n", broadcastStats["average_latency"])
fmt.Printf("失败数: %d\n", broadcastStats["failed_broadcasts"])

// 反垃圾统计
spamStats := server.GetSpamDetectorStats()
fmt.Printf("检查总数: %d\n", spamStats["total_checked"])
fmt.Printf("拦截数: %d\n", spamStats["duplicate_detected"])

// 消息路由统计
routerStats := server.GetMessageRouterStats()
fmt.Printf("队列长度: %d\n", routerStats["queue_length"])
```

---

## 🔧 配置参考

### ServerConfig 完整配置

```go
config := &ServerConfig{
    // 频道配置
    ChannelID:       "channel-uuid",
    ChannelPassword: "secure-password",
    ChannelName:     "CTF Team Channel",
    MaxMembers:      100,
    
    // 传输配置
    TransportMode: models.TransportARP,
    
    // 认证配置
    RequireAuth:    true,
    AllowAnonymous: false,
    SessionTimeout: 24 * time.Hour,
    
    // 消息配置
    MaxMessageSize: 10 * 1024 * 1024, // 10MB
    MessageTTL:     30 * 24 * time.Hour,
    EnableOffline:  true,
    
    // 安全配置
    EnableRateLimit: true,
    MaxMessageRate:  60,  // 每分钟60条
    EnableSignature: true,
}
```

---

## 🚀 最佳实践

### 1. 成员加入流程
```go
// 1. 处理加入请求（自动）
// 2. 投递离线消息
server.DeliverOfflineMessagesToMember(memberID)
// 3. 发送欢迎消息
welcomeMsg := &models.Message{
    Type: models.MessageTypeSystem,
    Content: models.MessageContent{
        "text": "欢迎加入频道！",
    },
}
server.BroadcastMessage(welcomeMsg)
```

### 2. 成员管理操作
```go
// 禁言成员
err := server.MuteMember(memberID, 10*time.Minute, "spam")

// 解除禁言
err := server.UnmuteMember(memberID)

// 踢出成员
err := server.KickMember(memberID, "违反规则")

// 封禁成员
err := server.BanMember(memberID, "恶意行为", 24*time.Hour)
```

### 3. 消息处理流程
```go
// 消息处理流程（自动执行）：
// 1. 解密消息
// 2. 验证签名
// 3. 检查成员身份
// 4. 检查禁言状态
// 5. 频率限制检查
// 6. 反垃圾检测
// 7. 持久化消息
// 8. 广播消息（带服务器签名）
// 9. 发布事件
```

---

## 📚 参考文档

- [ARCHITECTURE.md](../../docs/ARCHITECTURE.md) - 架构设计
- [PROTOCOL.md](../../docs/PROTOCOL.md) - 通信协议
- [FEATURES.md](../../docs/FEATURES.md) - 功能规格
- [ARP_BROADCAST_MODE.md](../../docs/ARP_BROADCAST_MODE.md) - ARP广播模式

---

## 🎯 总结

所有 TODO 功能均已完整实现：

- ✅ 消息持久化到离线队列
- ✅ 消息确认（ACK）机制
- ✅ 频率限制细节
- ✅ 反垃圾消息
- ✅ 成员踢出与封禁
- ✅ 权限分级（管理员/普通成员）

每个功能都经过充分测试，并提供了清晰的API和使用示例。

