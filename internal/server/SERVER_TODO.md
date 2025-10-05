# CrossWire Server 模块完善 TODO 清单

> 基于客户端实现和文档规范，服务端需要完善的功能清单
>
> 创建时间: 2025-10-05
> 状态: 待完善

---

## 📋 总体概览

### 当前状态

| 模块 | 文件 | 完成度 | 状态 |
|------|------|--------|------|
| Server核心 | server.go | 70% | 🟡 待完善 |
| 频道管理 | channel_manager.go | 60% | 🟡 待完善 |
| 广播管理 | broadcast_manager.go | 50% | 🟡 待完善 |
| 消息路由 | message_router.go | 60% | 🟡 待完善 |
| 认证管理 | auth_manager.go | 70% | 🟡 待完善 |
| 挑战管理 | challenge_manager.go | 40% | 🔴 待实现 |

### 对比客户端功能

客户端已完成 **9个管理器，4129行代码**，实现了完整的功能。

服务端需要与之匹配的功能：
- ✅ 接收客户端消息并验证
- ✅ 签名并广播消息
- ✅ 文件传输支持（分块接收/发送）
- ✅ Challenge 完整实现
- ✅ 离线消息队列
- ✅ 成员管理（踢出、禁言、角色管理）
- ✅ 完整的统计和监控

---

## 🎯 优先级 P0 - 核心功能（必须立即完成）

### 1. ✅ 修复 Linter 错误

**文件:** `internal/server/auth_manager.go:83`

```go
// ❌ 错误: unknown field challenges
challenges: make(map[string]*AuthChallenge),

// ✅ 修复: AuthManager 结构中没有 challenges 字段，应该删除
```

**状态:** ✅ 已修复

---

### 2. 🔴 实现文件传输支持

**需求来源:**
- 客户端已实现 FileManager（上传/下载/断点续传）
- 服务端需要接收、验证并转发文件分块

**文件:** `internal/server/message_router.go`

**待实现功能:**

```go
// 文件分块处理
func (mr *MessageRouter) handleFileChunk(msg *models.Message) error {
    // 1. 验证文件元数据
    // 2. 接收分块
    // 3. 更新进度
    // 4. 完整性检查（最后一个分块）
    // 5. 广播文件分块
    return nil
}

// 文件上传完成处理
func (mr *MessageRouter) handleFileComplete(fileID string) error {
    // 1. 验证所有分块完整
    // 2. SHA256 校验
    // 3. 更新文件状态
    // 4. 广播上传完成事件
    return nil
}

// 文件下载请求处理
func (mr *MessageRouter) handleFileDownloadRequest(msg *models.Message) error {
    // 1. 验证权限
    // 2. 获取文件元数据
    // 3. 逐块发送给请求者
    return nil
}
```

**涉及的消息类型:**
- `MessageTypeFileMetadata` - 文件元数据
- `MessageTypeFileChunk` - 文件分块
- `MessageTypeFileComplete` - 上传完成
- `MessageTypeFileRequest` - 下载请求

**TODO 清单:**
- [ ] 在 `MessageRouter` 中添加文件分块处理
- [ ] 实现文件元数据验证
- [ ] 实现分块接收和转发
- [ ] 实现 SHA256 完整性校验
- [ ] 实现文件下载请求处理
- [ ] 添加文件传输统计（字节数、速度）

---

### 3. 🔴 完善消息签名逻辑

**需求来源:**
- 客户端已实现 SignatureVerifier
- 服务端需要对所有广播消息进行 Ed25519 签名

**文件:** `internal/server/broadcast_manager.go`

**当前问题:**
- 签名逻辑可能不完整
- 需要与客户端 SignatureVerifier 对应

**待完善:**

```go
// BroadcastMessage 广播消息（添加签名）
func (bm *BroadcastManager) BroadcastMessage(msg *models.Message) error {
    // 1. 序列化消息
    payload, err := json.Marshal(msg)
    if err != nil {
        return err
    }
    
    // 2. 加密消息
    encrypted, err := bm.server.crypto.EncryptMessage(payload)
    if err != nil {
        return err
    }
    
    // 3. 签名（关键！）
    signature := ed25519.Sign(bm.server.config.PrivateKey, encrypted)
    
    // 4. 构造 transport.Message
    transportMsg := &transport.Message{
        ID:        msg.ID,
        SenderID:  "server",
        Type:      transport.MessageTypeData,
        Payload:   encrypted,
        Signature: signature,  // 添加签名
        Timestamp: time.Now(),
    }
    
    // 5. 广播
    return bm.server.transport.SendMessage(transportMsg)
}
```

**TODO 清单:**
- [ ] 检查 BroadcastMessage 是否正确签名
- [ ] 确保签名包含在 transport.Message 中
- [ ] 验证签名格式与客户端一致
- [ ] 添加签名失败处理
- [ ] 添加签名统计

---

### 4. 🔴 实现离线消息队列

**需求来源:**
- 客户端已实现 OfflineQueue（自动重试）
- 服务端需要存储离线消息并在成员上线时推送

**文件:** 新建 `internal/server/offline_manager.go`

**功能设计:**

```go
type OfflineManager struct {
    server        *Server
    offlineQueue  map[string][]*models.Message // memberID -> messages
    mutex         sync.RWMutex
    maxQueueSize  int
}

// StoreOfflineMessage 存储离线消息
func (om *OfflineManager) StoreOfflineMessage(memberID string, msg *models.Message) error {
    om.mutex.Lock()
    defer om.mutex.Unlock()
    
    // 1. 检查队列大小
    queue := om.offlineQueue[memberID]
    if len(queue) >= om.maxQueueSize {
        // 删除最旧的消息
        queue = queue[1:]
    }
    
    // 2. 添加到队列
    queue = append(queue, msg)
    om.offlineQueue[memberID] = queue
    
    // 3. 持久化到数据库
    return om.server.messageRepo.Create(msg)
}

// DeliverOfflineMessages 投递离线消息
func (om *OfflineManager) DeliverOfflineMessages(memberID string) error {
    om.mutex.Lock()
    queue := om.offlineQueue[memberID]
    delete(om.offlineQueue, memberID)
    om.mutex.Unlock()
    
    // 批量发送
    for _, msg := range queue {
        // 发送给指定成员
        om.server.broadcastManager.SendToMember(memberID, msg)
    }
    
    return nil
}

// CleanupOldMessages 清理过期消息
func (om *OfflineManager) CleanupOldMessages(maxAge time.Duration) {
    // 定期清理
}
```

**TODO 清单:**
- [ ] 创建 OfflineManager 结构
- [ ] 实现消息存储到队列
- [ ] 实现消息持久化到数据库
- [ ] 实现成员上线时推送离线消息
- [ ] 实现过期消息清理
- [ ] 添加离线消息统计
- [ ] 集成到 Server.Start/Stop

---

### 5. 🔴 完善 Challenge 服务端逻辑

**需求来源:**
- 客户端已实现 ChallengeManager（获取、提交、请求提示）
- 服务端需要完整实现 Challenge 的创建、分配、验证

**文件:** `internal/server/challenge_manager.go`

**当前完成度:** 约 40%

**待完善功能:**

```go
// CreateChallenge 创建挑战
func (cm *ChallengeManager) CreateChallenge(challenge *models.Challenge) error {
    // 1. 验证权限（仅 Admin/Leader）
    // 2. 生成 Challenge ID
    // 3. Hash Flag
    // 4. 保存到数据库
    // 5. 广播创建事件
    return nil
}

// AssignChallenge 分配挑战
func (cm *ChallengeManager) AssignChallenge(challengeID, memberID string, assignedBy string) error {
    // 1. 验证权限
    // 2. 检查挑战是否存在
    // 3. 创建分配记录
    // 4. 广播分配事件
    return nil
}

// SubmitFlag 验证 Flag
func (cm *ChallengeManager) SubmitFlag(challengeID, memberID, flag string) error {
    // 1. 获取挑战
    // 2. Hash Flag 并对比
    // 3. 记录提交
    // 4. 更新统计
    // 5. 广播解决事件（如果正确）
    return nil
}

// UnlockHint 解锁提示
func (cm *ChallengeManager) UnlockHint(challengeID, memberID string, hintIndex int) error {
    // 1. 验证权限
    // 2. 扣除分数/积分
    // 3. 记录解锁
    // 4. 返回提示内容
    return nil
}

// UpdateProgress 更新进度
func (cm *ChallengeManager) UpdateProgress(challengeID, memberID string, progress int, summary string) error {
    // 1. 验证权限
    // 2. 更新进度记录
    // 3. 广播进度事件
    return nil
}

// GetLeaderboard 获取排行榜
func (cm *ChallengeManager) GetLeaderboard(channelID string) ([]*LeaderboardEntry, error) {
    // 1. 统计每个成员的解题数
    // 2. 计算总分
    // 3. 排序
    return nil, nil
}
```

**TODO 清单:**
- [ ] 实现 CreateChallenge 方法
- [ ] 实现 AssignChallenge 方法
- [ ] 实现 SubmitFlag 方法（含 Flag Hash 对比）
- [ ] 实现 UnlockHint 方法
- [ ] 实现 UpdateProgress 方法
- [ ] 实现 GetLeaderboard 方法
- [ ] 实现 GetChallengeStats 方法
- [ ] 添加 Challenge 事件广播
- [ ] 添加权限检查
- [ ] 添加数据库操作

---

## 🎯 优先级 P1 - 重要功能（需要尽快完成）

### 6. 🟡 完善成员管理功能

**文件:** `internal/server/channel_manager.go`

**待完善:**

```go
// KickMember 踢出成员
func (cm *ChannelManager) KickMember(memberID string, kickedBy string, reason string) error {
    // 1. 验证权限（仅 Admin）
    // 2. 检查不能踢自己
    // 3. 更新成员状态
    // 4. 广播踢出事件
    // 5. 记录审计日志
    return nil
}

// MuteMember 禁言成员
func (cm *ChannelManager) MuteMember(memberID string, duration time.Duration, mutedBy string) error {
    // 1. 验证权限（Admin/Leader）
    // 2. 更新成员状态
    // 3. 设置禁言过期时间
    // 4. 广播禁言事件
    return nil
}

// UnmuteMember 解除禁言
func (cm *ChannelManager) UnmuteMember(memberID string, unmutedBy string) error {
    // 1. 验证权限
    // 2. 更新成员状态
    // 3. 广播解除禁言事件
    return nil
}

// ChangeRole 修改角色
func (cm *ChannelManager) ChangeRole(memberID string, newRole models.Role, changedBy string) error {
    // 1. 验证权限（仅 Admin）
    // 2. 更新成员角色
    // 3. 广播角色变更事件
    return nil
}

// UpdateOnlineStatus 更新在线状态
func (cm *ChannelManager) UpdateOnlineStatus(memberID string, status models.UserStatus) error {
    // 1. 更新成员状态
    // 2. 更新最后活跃时间
    // 3. 广播状态变更
    return nil
}
```

**TODO 清单:**
- [ ] 实现 KickMember 方法
- [ ] 实现 MuteMember/UnmuteMember 方法
- [ ] 实现 ChangeRole 方法
- [ ] 实现 UpdateOnlineStatus 方法
- [ ] 添加权限验证辅助函数
- [ ] 添加审计日志记录
- [ ] 实现定期检查在线状态

---

### 7. 🟡 实现消息同步接口

**需求来源:**
- 客户端已实现 SyncManager（增量同步）
- 服务端需要提供同步接口

**文件:** `internal/server/message_router.go`

**待实现:**

```go
// HandleSyncRequest 处理同步请求
func (mr *MessageRouter) HandleSyncRequest(req *SyncRequest) (*SyncResponse, error) {
    // 1. 验证会话
    // 2. 获取指定时间戳后的消息
    // 3. 获取成员列表变更
    // 4. 获取文件元数据变更
    // 5. 返回增量数据
    return &SyncResponse{
        Messages:    messages,
        Members:     members,
        Files:       files,
        LastSync:    time.Now(),
        HasMore:     hasMore,
    }, nil
}

type SyncRequest struct {
    MemberID      string
    LastSyncTime  time.Time
    Limit         int
}

type SyncResponse struct {
    Messages      []*models.Message
    Members       []*models.Member
    Files         []*models.File
    LastSync      time.Time
    HasMore       bool
}
```

**TODO 清单:**
- [ ] 定义 SyncRequest/SyncResponse 结构
- [ ] 实现 HandleSyncRequest 方法
- [ ] 实现增量查询（基于时间戳）
- [ ] 实现分页支持
- [ ] 添加同步统计

---

### 8. 🟡 实现速率限制

**需求来源:**
- ServerConfig 中已有 EnableRateLimit 配置
- 需要防止消息洪水攻击

**文件:** `internal/server/message_router.go`

**待实现:**

```go
type RateLimiter struct {
    limits      map[string]*TokenBucket  // memberID -> bucket
    maxRate     int                      // 每分钟最大消息数
    mutex       sync.RWMutex
}

type TokenBucket struct {
    tokens       int
    lastRefill   time.Time
    maxTokens    int
    refillRate   int  // 每秒恢复的令牌数
}

// CheckRateLimit 检查速率限制
func (mr *MessageRouter) CheckRateLimit(memberID string) bool {
    mr.rateLimiter.mutex.Lock()
    defer mr.rateLimiter.mutex.Unlock()
    
    bucket := mr.rateLimiter.limits[memberID]
    if bucket == nil {
        bucket = &TokenBucket{
            tokens:     mr.rateLimiter.maxRate,
            lastRefill: time.Now(),
            maxTokens:  mr.rateLimiter.maxRate,
            refillRate: mr.rateLimiter.maxRate / 60,
        }
        mr.rateLimiter.limits[memberID] = bucket
    }
    
    // 补充令牌
    now := time.Now()
    elapsed := now.Sub(bucket.lastRefill).Seconds()
    tokensToAdd := int(elapsed * float64(bucket.refillRate))
    if tokensToAdd > 0 {
        bucket.tokens = min(bucket.tokens + tokensToAdd, bucket.maxTokens)
        bucket.lastRefill = now
    }
    
    // 消耗令牌
    if bucket.tokens > 0 {
        bucket.tokens--
        return true
    }
    
    return false  // 超过速率限制
}
```

**TODO 清单:**
- [ ] 创建 RateLimiter 结构
- [ ] 实现令牌桶算法
- [ ] 在 MessageRouter.RouteMessage 中集成
- [ ] 添加超限拒绝日志
- [ ] 添加速率限制统计
- [ ] 实现定期清理不活跃的令牌桶

---

### 9. 🟡 完善统计和监控

**文件:** `internal/server/server.go`

**待完善:**

```go
// GetDetailedStats 获取详细统计
func (s *Server) GetDetailedStats() *DetailedStats {
    return &DetailedStats{
        Server:    s.GetStats(),
        Channel:   s.channelManager.GetStats(),
        Broadcast: s.broadcastManager.GetStats(),
        Router:    s.messageRouter.GetStats(),
        Auth:      s.authManager.GetStats(),
        Challenge: s.challengeManager.GetStats(),
    }
}

type DetailedStats struct {
    Server    ServerStats
    Channel   ChannelStats
    Broadcast BroadcastStats
    Router    RouterStats
    Auth      AuthStats
    Challenge ChallengeStats
}
```

**TODO 清单:**
- [ ] 为每个管理器添加 GetStats 方法
- [ ] 实现 GetDetailedStats 聚合方法
- [ ] 添加更多统计指标（延迟、错误率）
- [ ] 实现统计导出（JSON/Prometheus）
- [ ] 添加实时监控事件

---

## 🎯 优先级 P2 - 优化和增强（可以延后）

### 10. 🟢 添加审计日志

**文件:** 新建 `internal/server/audit_manager.go`

**功能:**
- 记录所有关键操作（加入、踢出、禁言、角色变更）
- 记录所有 Challenge 操作（创建、提交、解决）
- 提供审计日志查询接口

**TODO 清单:**
- [ ] 创建 AuditManager 结构
- [ ] 实现日志记录方法
- [ ] 在各管理器中集成审计日志
- [ ] 实现日志查询接口
- [ ] 实现日志导出功能

---

### 11. 🟢 实现消息搜索

**需求:**
- 客户端可能需要搜索历史消息
- 使用 SQLite FTS5 全文搜索

**文件:** `internal/server/message_router.go`

**TODO 清单:**
- [ ] 配置 FTS5 虚拟表
- [ ] 实现 SearchMessages 方法
- [ ] 支持多种搜索模式（关键词、用户、时间范围）
- [ ] 添加搜索结果高亮

---

### 12. 🟢 实现广播确认机制（可选）

**文件:** `internal/server/broadcast_manager.go`

**功能:**
- 收集客户端 ACK
- 检测消息丢失
- 重传机制

**TODO 清单:**
- [ ] 定义 ACK 消息格式
- [ ] 实现 ACK 收集
- [ ] 实现超时检测
- [ ] 实现重传逻辑

---

### 13. 🟢 添加性能测试

**TODO 清单:**
- [ ] 编写基准测试（Benchmark）
- [ ] 测试消息吞吐量
- [ ] 测试并发连接
- [ ] 测试内存使用
- [ ] 优化性能瓶颈

---

### 14. 🟢 编写单元测试

**TODO 清单:**
- [ ] ChannelManager 单元测试
- [ ] BroadcastManager 单元测试
- [ ] MessageRouter 单元测试
- [ ] AuthManager 单元测试
- [ ] ChallengeManager 单元测试
- [ ] 集成测试

---

## 📝 代码质量检查

### Linter 错误修复

**当前错误:**
- [x] `internal/server/auth_manager.go:83` - unknown field challenges

**待检查:**
- [ ] 运行 `go vet`
- [ ] 运行 `golangci-lint`
- [ ] 修复所有警告

---

## 🔄 与客户端对应关系

### 功能对比表

| 功能 | 客户端 | 服务端 | 状态 |
|------|--------|--------|------|
| 消息收发 | ✅ ReceiveManager | ✅ MessageRouter | 🟡 待完善 |
| 文件传输 | ✅ FileManager | ❌ 未实现 | 🔴 P0 |
| 服务发现 | ✅ DiscoveryManager | ✅ Transport层 | ✅ 完成 |
| 离线队列 | ✅ OfflineQueue | ❌ 未实现 | 🔴 P0 |
| 签名验证 | ✅ SignatureVerifier | 🟡 部分实现 | 🟡 P0 |
| Challenge | ✅ ChallengeManager | 🟡 40%完成 | 🔴 P0 |
| 缓存管理 | ✅ CacheManager | N/A | N/A |
| 同步管理 | ✅ SyncManager | ❌ 未实现 | 🟡 P1 |

---

## 📊 预估工作量

### P0 - 核心功能（1-2天）
- ✅ 修复 Linter 错误：10分钟
- 🔴 实现文件传输：4小时
- 🔴 完善消息签名：1小时
- 🔴 实现离线消息队列：3小时
- 🔴 完善 Challenge 逻辑：6小时

**总计：约 14小时**

### P1 - 重要功能（1天）
- 完善成员管理：3小时
- 实现消息同步接口：2小时
- 实现速率限制：2小时
- 完善统计监控：2小时

**总计：约 9小时**

### P2 - 优化增强（1-2天）
- 审计日志：3小时
- 消息搜索：2小时
- 测试编写：8小时

**总计：约 13小时**

---

## 🚀 实施计划

### 第一阶段：核心功能（今天）

1. ✅ 修复 Linter 错误
2. 实现文件传输支持
3. 完善消息签名逻辑
4. 实现离线消息队列

### 第二阶段：Challenge 完善（今天）

5. 完善 Challenge 所有方法
6. 添加 Challenge 事件广播
7. 测试 Challenge 功能

### 第三阶段：重要功能（明天）

8. 完善成员管理
9. 实现消息同步接口
10. 实现速率限制
11. 完善统计监控

### 第四阶段：优化和测试（后续）

12. 添加审计日志
13. 实现消息搜索
14. 编写单元测试
15. 性能测试和优化

---

## ✅ 完成标准

服务端被认为"完成"的标准：

1. ✅ 所有 P0 功能实现
2. ✅ 所有 P1 功能实现
3. ✅ 无 Linter 错误
4. ✅ 通过编译
5. ✅ 与客户端功能匹配
6. ✅ 文档完整
7. ✅ 基本测试通过

---

## 📚 参考文档

- [docs/ARCHITECTURE.md](../../docs/ARCHITECTURE.md) - 架构设计
- [docs/PROTOCOL.md](../../docs/PROTOCOL.md) - 通信协议
- [docs/ARP_BROADCAST_MODE.md](../../docs/ARP_BROADCAST_MODE.md) - 广播模式
- [internal/client/README.md](../client/README.md) - 客户端实现
- [internal/client/FINAL_SUMMARY.md](../client/FINAL_SUMMARY.md) - 客户端总结

---

**创建时间:** 2025-10-05  
**预计完成:** 2-3天  
**当前进度:** 约 50%
