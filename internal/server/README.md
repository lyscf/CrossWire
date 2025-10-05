## Server 服务端模块

> CrossWire服务端核心实现

---

## 📚 概述

Server模块负责频道管理、消息广播、认证授权和CTF挑战管理。采用**服务器签名广播模式**，确保高安全性和可控性。

**参考文档**:
- `docs/ARCHITECTURE.md` - 3.1.2 服务端模块
- `docs/PROTOCOL.md` - 2.2.3 消息广播（服务器签名模式）
- `docs/ARP_BROADCAST_MODE.md` - 服务器签名广播模式设计
- `docs/CHALLENGE_SYSTEM.md` - CTF挑战系统

---

## 🏗️ 架构设计

### 核心架构

```
Server
  ├─ ChannelManager    : 频道管理器
  │   ├─ 创建/关闭频道
  │   ├─ 成员加入/离开
  │   └─ 权限验证
  │
  ├─ BroadcastManager  : 广播管理器
  │   ├─ 广播消息到所有成员
  │   ├─ 消息去重
  │   └─ ACK收集
  │
  ├─ MessageRouter     : 消息路由器
  │   ├─ 消息处理
  │   ├─ 消息持久化
  │   └─ 离线消息队列
  │
  ├─ AuthManager       : 认证管理器
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

---

## 🎯 服务器签名广播模式

### 设计原理

```
客户端 → 单播到服务器 → 服务器验证权限 → 服务器签名 → 广播到所有客户端
```

### 核心流程

#### 1. 消息发送（客户端 → 服务器）

```go
// 客户端单播给服务器
client.SendMessage(msg) → Server (验证权限)
```

#### 2. 服务器处理

```go
// 服务器验证、签名、广播
func (s *Server) HandleClientMessage(msg) {
    // 1. 解密消息
    decrypted := crypto.Decrypt(msg)
    
    // 2. 验证权限
    if !isValidMember(msg.SenderID) || isMuted(msg.SenderID) {
        return // 拒绝
    }
    
    // 3. 签名
    signature := ed25519.Sign(serverPrivateKey, decrypted)
    
    // 4. 广播
    broadcast(signedMessage)
}
```

#### 3. 客户端接收

```go
// 客户端验证签名后接受
func (c *Client) ReceiveLoop() {
    frame := receiveFrame()
    
    // 验证服务器签名（关键！）
    if !ed25519.Verify(serverPublicKey, frame.Message, frame.Signature) {
        return // 拒绝伪造消息
    }
    
    // 处理消息
    handleMessage(frame.Message)
}
```

---

## 📦 功能模块

### 1. ChannelManager - 频道管理器

**职责**:
- 频道创建与初始化
- 成员管理（加入/离开/踢出）
- 禁言管理
- 权限验证

**核心方法**:
```go
func (cm *ChannelManager) Initialize() error
func (cm *ChannelManager) AddMember(member *models.Member) error
func (cm *ChannelManager) RemoveMember(memberID, reason string) error
func (cm *ChannelManager) MuteMember(memberID string, duration time.Duration, reason string) error
func (cm *ChannelManager) IsMuted(memberID string) bool
```

**特性**:
- ✅ 自动加载频道和成员列表
- ✅ 禁言记录过期自动清理
- ✅ 在线成员统计
- ✅ 成员状态管理

---

### 2. BroadcastManager - 广播管理器

**职责**:
- 服务器签名消息广播
- 消息去重（防止接收自己的广播）
- ACK收集（可选）
- 广播统计

**核心方法**:
```go
func (bm *BroadcastManager) Broadcast(msg *models.Message) error
func (bm *BroadcastManager) IsSentByMe(messageID string) bool
func (bm *BroadcastManager) RecordAck(messageID, memberID string)
```

**广播流程**:
1. 序列化消息
2. 加密消息（频道密钥）
3. 服务器签名（Ed25519）
4. 构造签名载荷
5. 通过传输层广播
6. 记录已发送消息

---

### 3. MessageRouter - 消息路由器

**职责**:
- 接收并路由客户端消息
- 权限验证（成员身份/禁言状态）
- 频率限制
- 消息持久化
- 离线消息队列

**核心方法**:
```go
func (mr *MessageRouter) HandleClientMessage(transportMsg *transport.Message)
func (mr *MessageRouter) HandleFileUpload(transportMsg *transport.Message)
func (mr *MessageRouter) AddOfflineMessage(memberID string, msg *models.Message)
```

**验证流程**:
1. 解密消息
2. 验证发送者身份
3. 检查禁言状态
4. 频率限制检查
5. 持久化消息
6. 广播消息

---

### 4. AuthManager - 认证管理器

**职责**:
- 处理成员加入请求
- 密码验证
- 会话管理
- 权限检查

**核心方法**:
```go
func (am *AuthManager) HandleJoinRequest(transportMsg *transport.Message)
func (am *AuthManager) VerifySession(memberID string) bool
func (am *AuthManager) CheckPermission(memberID string, requiredRole models.Role) bool
```

**认证流程** (参考 `PROTOCOL.md` - 2.2.2):
1. 客户端发送加密的JOIN_REQUEST
2. 服务器解密并验证密码
3. 服务器验证昵称和频道容量
4. 创建成员和会话
5. 发送JOIN_RESPONSE（包含频道密钥、成员列表、服务器公钥）
6. 广播成员加入消息

---

### 5. ChallengeManager - 题目管理器

**职责**:
- CTF题目管理
- 题目分配
- Flag提交与验证
- 进度跟踪
- 提示解锁

**核心方法**:
```go
func (cm *ChallengeManager) CreateChallenge(challenge *models.Challenge) error
func (cm *ChallengeManager) AssignChallenge(challengeID, memberID, assignedBy string) error
func (cm *ChallengeManager) HandleFlagSubmission(transportMsg *transport.Message)
func (cm *ChallengeManager) VerifyFlag(challengeID, flag string) (bool, error)
func (cm *ChallengeManager) UnlockHint(challengeID, memberID string, hintIndex int) error
```

**Flag提交流程**:
1. 接收客户端提交
2. 解密并反序列化
3. 验证Flag
4. 保存提交记录
5. 更新进度
6. 发布事件
7. 广播解题消息（如果正确）

---

## 💡 使用示例

### 1. 创建并启动服务端

```go
// 配置
config := &server.ServerConfig{
    ChannelID:       "channel-uuid",
    ChannelPassword: "secure-password",
    ChannelName:     "My CTF Team",
    Description:     "Offline CTF Communication",
    MaxMembers:      50,
    TransportMode:   models.TransportModeARP,
    EnableSignature: true,
}

// 创建服务端
srv, err := server.NewServer(config, db, eventBus, logger)
if err != nil {
    log.Fatal(err)
}

// 启动
if err := srv.Start(); err != nil {
    log.Fatal(err)
}
defer srv.Stop()
```

### 2. 添加成员

```go
member := &models.Member{
    ID:       "member-uuid",
    Nickname: "Alice",
    Role:     models.RoleMember,
    Status:   models.StatusOnline,
}

if err := srv.AddMember(member); err != nil {
    log.Error("Failed to add member:", err)
}
```

### 3. 禁言成员

```go
// 禁言10分钟
duration := 10 * time.Minute
if err := srv.MuteMember(memberID, duration, "Spamming"); err != nil {
    log.Error("Failed to mute member:", err)
}
```

### 4. 创建CTF题目

```go
challenge := &models.Challenge{
    ID:          "chall-uuid",
    Title:       "Web Exploitation",
    Description: "Find the flag in the web app",
    Category:    "Web",
    Difficulty:  "Medium",
    Points:      300,
    Flag:        "flag{example}",
}

if err := srv.challengeManager.CreateChallenge(challenge); err != nil {
    log.Error("Failed to create challenge:", err)
}
```

### 5. 获取统计信息

```go
stats := srv.GetStats()
fmt.Printf("Members: %d/%d\n", stats.OnlineMembers, stats.TotalMembers)
fmt.Printf("Messages: %d\n", stats.TotalMessages)
fmt.Printf("Broadcasts: %d\n", stats.TotalBroadcasts)
```

---

## 🔒 安全机制

### 多层防护

1. **服务器签名验证**（第一道防线）
   - 所有广播消息必须有服务器Ed25519签名
   - 客户端拒绝无签名或签名无效的消息

2. **服务器权限控制**（第二道防线）
   - 验证成员身份
   - 验证禁言状态
   - 频率限制
   - 角色权限检查

3. **加密隔离**（第三道防线）
   - 不同频道使用不同密钥
   - 解密失败自动忽略

4. **防重放攻击**
   - 时间戳验证
   - Nonce检查
   - 消息去重

---

## 📊 配置说明

### ServerConfig

```go
type ServerConfig struct {
    // 频道配置
    ChannelID       string
    ChannelPassword string
    ChannelName     string
    Description     string
    MaxMembers      int
    
    // 传输配置
    TransportMode   models.TransportMode
    TransportConfig *transport.Config
    
    // 认证配置
    RequireAuth     bool
    AllowAnonymous  bool
    SessionTimeout  time.Duration
    
    // 消息配置
    MaxMessageSize  int
    MessageTTL      time.Duration
    EnableOffline   bool
    
    // 安全配置
    EnableRateLimit bool
    MaxMessageRate  int // 每分钟最多消息数
    EnableSignature bool // 是否启用服务器签名
    
    // 服务器密钥对
    PrivateKey ed25519.PrivateKey
    PublicKey  ed25519.PublicKey
}
```

### 默认配置

```go
DefaultServerConfig = &ServerConfig{
    MaxMembers:      100,
    RequireAuth:     true,
    AllowAnonymous:  false,
    SessionTimeout:  24 * time.Hour,
    MaxMessageSize:  10 * 1024 * 1024, // 10MB
    MessageTTL:      30 * 24 * time.Hour,
    EnableOffline:   true,
    EnableRateLimit: true,
    MaxMessageRate:  60, // 每分钟60条
    EnableSignature: true,
}
```

---

## 🚀 性能特性

### 1. 异步处理
- 广播队列：异步处理广播任务
- 消息队列：异步处理客户端消息
- 事件发布：异步通知各模块

### 2. 频率限制
- 滑动窗口算法
- 每成员独立限制
- 可配置限制阈值

### 3. 消息去重
- 基于消息ID的去重
- 定期清理过期记录
- 防止广播循环

---

## 📈 统计信息

### ServerStats

```go
type ServerStats struct {
    StartTime        time.Time
    TotalMembers     int
    OnlineMembers    int
    TotalMessages    uint64
    TotalBroadcasts  uint64
    TotalBytes       uint64
    DroppedMessages  uint64
    RejectedMessages uint64
}
```

---

## 🧪 测试示例

```go
func TestServer(t *testing.T) {
    // 创建服务端
    srv, err := server.NewServer(config, db, eventBus, logger)
    assert.NoError(t, err)
    
    // 启动
    err = srv.Start()
    assert.NoError(t, err)
    defer srv.Stop()
    
    // 添加成员
    member := &models.Member{
        ID:       "test-member",
        Nickname: "Test User",
    }
    err = srv.AddMember(member)
    assert.NoError(t, err)
    
    // 验证成员已添加
    members, err := srv.GetMembers()
    assert.NoError(t, err)
    assert.Len(t, members, 1)
}
```

---

## 📖 相关文档

- [ARCHITECTURE.md](../../docs/ARCHITECTURE.md) - 系统架构
- [PROTOCOL.md](../../docs/PROTOCOL.md) - 通信协议
- [ARP_BROADCAST_MODE.md](../../docs/ARP_BROADCAST_MODE.md) - 服务器签名广播模式
- [CHALLENGE_SYSTEM.md](../../docs/CHALLENGE_SYSTEM.md) - CTF挑战系统

---

## TODO

- [ ] 消息确认（ACK）机制完善
- [ ] 频道管理员权限细化
- [ ] 成员踢出与封禁功能
- [ ] 反垃圾消息检测
- [ ] 消息撤回功能
- [ ] 文件传输支持
- [ ] 离线消息队列优化

---

**最后更新**: 2025-10-05  
**状态**: ✅ 核心功能完整实现并可用

