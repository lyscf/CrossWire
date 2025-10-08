# Bug Report: 题目聊天室消息隔离问题

## 🐛 问题描述

**严重等级**: P0 - 阻塞性问题  
**影响范围**: 题目管理系统、消息隔离  
**发现时间**: 2025-10-07

**问题**: 客户端无法向子频道（题目聊天室）发送消息，所有消息都会发送到主频道，导致消息无法隔离。

---

## 🔍 根本原因

### 设计缺陷

系统在设计时考虑了子频道的概念，但**消息发送路径没有支持子频道**：

1. **数据库设计**（正确）:
   ```go
   // models/message.go
   type Message struct {
       ChannelID   string  // 消息所属频道（主频道或子频道）
       ChallengeID string  // 关联的题目ID
       RoomType    string  // "main" 或 "challenge"
       // ...
   }
   
   // models/channel.go
   type Channel struct {
       ID              string  // 主键
       ParentChannelID string  // 父频道ID（子频道时非空）
       // ...
   }
   ```

2. **查询隔离**（正确）:
   ```go
   // internal/storage/message_repository.go:35
   func (r *MessageRepository) GetByChannelID(channelID string, limit, offset int) {
       // WHERE channel_id = ? AND deleted = ?
       // 严格按 channel_id 过滤，理论上是隔离的
   }
   ```

3. **客户端发送**（❌ 错误）:
   ```go
   // internal/client/client.go:444
   func (c *Client) SendMessage(content string, msgType models.MessageType) error {
       msg := &models.Message{
           ChannelID: c.config.ChannelID,  // ❌ 硬编码主频道ID
           // ...
       }
   }
   ```
   
   **问题**: `c.config.ChannelID` 始终是连接时的主频道ID，无法动态指定子频道。

4. **App层API**（❌ 缺少参数）:
   ```go
   // internal/app/message_api.go:10
   func (a *App) SendMessage(req SendMessageRequest) Response {
       // SendMessageRequest 没有 channelID 字段
       cli.SendMessage(req.Content, req.Type)  // 无法传递频道信息
   }
   ```

5. **前端调用**（❌ 无法指定频道）:
   ```javascript
   // frontend/src/views/ChatView.vue:285
   await sendMessage(content, 'text')  // 没有 channelID 参数
   ```

---

## 🎯 影响

### 当前行为

```
用户场景：
1. 创建题目 "Web-100" → 自动创建子频道 "sub-web-100"
2. 切换到 "Web-100" 题目聊天室
3. 输入消息 "找到SQL注入点了"
4. 点击发送

实际结果：
- 消息的 channel_id = 主频道ID
- 消息显示在主频道
- 子频道是空的
- 消息没有隔离

预期结果：
- 消息的 channel_id = 子频道ID ("sub-web-100")
- 消息显示在题目聊天室
- 主频道看不到这条消息
- 消息完全隔离
```

### 数据库验证

```sql
-- 查询所有消息
SELECT id, channel_id, content, room_type, challenge_id 
FROM messages 
ORDER BY timestamp DESC 
LIMIT 10;

-- 预期：
-- channel_id   | room_type | challenge_id | content
-- -------------|-----------|--------------|------------------
-- main-id      | main      | NULL         | "普通聊天消息"
-- sub-web-100  | challenge | web-100      | "题目聊天室消息"

-- 实际：
-- channel_id   | room_type | challenge_id | content
-- -------------|-----------|--------------|------------------
-- main-id      | main      | NULL         | "普通聊天消息"
-- main-id      | main      | NULL         | "题目聊天室消息" (错误!)
```

---

## ✅ 解决方案

### 方案A: 修改消息发送链路（推荐）

**优点**: 架构清晰，支持多频道切换  
**缺点**: 需要修改多个层次  
**工时**: 6小时

#### 1. 修改数据类型

```go
// internal/app/types.go
type SendMessageRequest struct {
    Content   string             `json:"content"`
    Type      models.MessageType `json:"type"`
    ChannelID *string            `json:"channel_id,omitempty"` // 新增：可选的目标频道ID
    ReplyToID *string            `json:"reply_to_id,omitempty"`
}
```

#### 2. 修改App层API

```go
// internal/app/message_api.go
func (a *App) SendMessage(req SendMessageRequest) Response {
    // 确定目标频道ID
    targetChannelID := ""
    if req.ChannelID != nil && *req.ChannelID != "" {
        targetChannelID = *req.ChannelID
    } else {
        // 使用默认主频道ID
        if a.mode == ModeServer && a.server != nil {
            ch, _ := a.server.GetChannel()
            targetChannelID = ch.ID
        } else if a.mode == ModeClient && a.client != nil {
            targetChannelID = a.client.GetChannelID()
        }
    }
    
    // 发送消息
    if mode == ModeClient && cli != nil {
        err = cli.SendMessageToChannel(req.Content, req.Type, targetChannelID)
    }
    // ...
}
```

#### 3. 修改Client层

```go
// internal/client/client.go

// SendMessage 发送消息到默认频道（兼容旧代码）
func (c *Client) SendMessage(content string, msgType models.MessageType) error {
    return c.SendMessageToChannel(content, msgType, c.config.ChannelID)
}

// SendMessageToChannel 发送消息到指定频道（新方法）
func (c *Client) SendMessageToChannel(content string, msgType models.MessageType, channelID string) error {
    if !c.isRunning {
        return fmt.Errorf("client is not running")
    }
    
    // 构造消息
    msg := &models.Message{
        ID:        generateMessageID(),
        ChannelID: channelID,  // 使用传入的频道ID
        SenderID:  c.memberID,
        Type:      msgType,
        Timestamp: time.Now(),
    }
    
    // 如果是子频道消息，设置room_type和challenge_id
    if channelID != c.config.ChannelID {
        msg.RoomType = "challenge"
        // 从频道ID解析challenge_id（假设格式: {main-id}-sub-{challenge-id}）
        if strings.HasPrefix(channelID, c.config.ChannelID+"-sub-") {
            msg.ChallengeID = strings.TrimPrefix(channelID, c.config.ChannelID+"-sub-")
        }
    } else {
        msg.RoomType = "main"
    }
    
    // ... 后续序列化、签名、加密、发送逻辑 ...
}
```

#### 4. 修改前端

```vue
<!-- frontend/src/views/ChatView.vue -->
<script setup>
import { ref, computed } from 'vue'

// 当前选中的频道ID
const currentChannelID = ref('main')  // 默认主频道

// 选择频道时更新
const handleChannelSelect = (channelId) => {
  currentChannelID.value = channelId
  loadMessages(channelId)  // 加载该频道的消息
}

const handleSendMessage = async (messageData) => {
  const content = typeof messageData === 'string' ? messageData : messageData.content
  
  try {
    // 传递当前频道ID
    await sendMessage(content, 'text', currentChannelID.value)
  } catch (e) {
    message.error('发送失败: ' + (e.message || ''))
  }
}
</script>
```

```javascript
// frontend/src/api/app.js
export async function sendMessage(content, type = 'text', channelID = null) {
  console.log('[API] Calling SendMessage:', { content, type, channelID })
  const payload = {
    content,
    type,
    channel_id: channelID  // 传递频道ID
  }
  const res = await App.SendMessage(payload)
  console.log('[API] SendMessage response:', res)
  return unwrap(res)
}
```

---

### 方案B: 使用 challenge_id 过滤（简化方案）

**优点**: 改动小，快速修复  
**缺点**: 架构不清晰，只能用于题目聊天室  
**工时**: 2小时

#### 实施

1. 保持 `channel_id` 为主频道ID
2. 发送消息时设置 `challenge_id` 和 `room_type`
3. 查询消息时使用：
   ```go
   // 主频道消息
   WHERE channel_id = ? AND (room_type = 'main' OR room_type IS NULL)
   
   // 题目聊天室消息
   WHERE channel_id = ? AND room_type = 'challenge' AND challenge_id = ?
   ```

缺点：
- 违背了 channel_id 的设计初衷
- 子频道表 (channels) 成为冗余数据
- 无法支持其他类型的子频道

---

## 📝 推荐方案

**采用方案A**，理由：
1. 符合系统原有的设计意图（channel表已有parent_channel_id）
2. 扩展性强，未来可支持其他子频道类型
3. 前后端逻辑清晰，易于维护

**实施顺序**：
1. 后端：修改 Client.SendMessageToChannel
2. 后端：修改 App.SendMessage 支持 channelID 参数
3. 前端：ChatView 添加 currentChannelID 状态
4. 前端：sendMessage API 传递 channelID
5. 测试：创建题目→切换频道→发送消息→验证隔离

---

## 🧪 测试计划

### 测试用例1: 主频道消息隔离

```
1. 启动服务端
2. 客户端加入主频道
3. 在主频道发送消息 "主频道测试"
4. 验证：
   - 消息 channel_id = 主频道ID
   - 消息 room_type = "main"
   - 主频道可见此消息
```

### 测试用例2: 题目聊天室消息隔离

```
1. 创建题目 "Web-100"
2. 子频道ID: "main-sub-web-100"
3. 切换到 Web-100 聊天室
4. 发送消息 "子频道测试"
5. 验证：
   - 消息 channel_id = "main-sub-web-100"
   - 消息 room_type = "challenge"
   - 消息 challenge_id = "web-100"
   - 仅子频道可见此消息
   - 主频道不可见此消息
```

### 测试用例3: 频道切换

```
1. 在主频道发送消息A
2. 切换到题目聊天室
3. 发送消息B
4. 切换回主频道
5. 验证：
   - 主频道显示消息A，不显示消息B
   - 题目聊天室显示消息B，不显示消息A
```

---

## 📊 影响评估

| 模块 | 需要修改 | 文件数 | 代码行数 | 风险 |
|------|---------|--------|----------|------|
| 数据类型 | 是 | 1 | 5 | 低 |
| App层 | 是 | 1 | 20 | 中 |
| Client层 | 是 | 1 | 50 | 中 |
| 前端Vue | 是 | 2 | 30 | 低 |
| 前端API | 是 | 1 | 5 | 低 |
| **总计** | - | **6** | **110** | **中** |

---

## 🎯 完成标准

- [ ] 客户端可以发送消息到子频道
- [ ] 主频道消息不会出现在子频道
- [ ] 子频道消息不会出现在主频道
- [ ] 前端可以切换不同频道
- [ ] 数据库验证：channel_id 正确设置
- [ ] 单元测试通过
- [ ] 集成测试通过

---

**报告人**: AI Assistant  
**日期**: 2025-10-07  
**状态**: 待修复

