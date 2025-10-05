# Server Lint 错误修复清单

## 核心架构问题

### 1. Database 类型不匹配
- **错误**: Server使用`storage.DatabaseManager`，但实际类型是`storage.Database`
- **修复**: 修改Server使用`*storage.Database`
- **影响文件**: `server.go`, 所有manager文件

### 2. Database方法调用错误
- **错误**: 调用`db.GetChannelDB(channelID)`，但实际方法不接受参数
- **正确用法**: 
  1. 先调用`db.OpenChannelDB(channelID)`打开频道数据库
  2. 然后用`db.GetChannelDB()`获取`*gorm.DB`
  3. 创建Repository：`repo := storage.NewXXXRepository(db)`
  4. 使用Repository方法操作数据

### 3. transport.Message 字段不匹配
- **错误**: 使用`msg.From`, `msg.To`
- **实际字段**: `msg.SenderID`, `msg.SenderMAC`, `msg.SenderAddr`
- **修复**: 全部改用`msg.SenderID`

### 4. transport.MessageType 类型错误
- **错误**: 使用字符串如`"auth.join"`
- **实际类型**: `MessageType byte`，应使用常量如`MessageTypeAuth`
- **修复**: 定义新的应用层消息类型常量

### 5. crypto.Manager 缺少方法
- **状态**: ✅ 已修复
- **已添加**: `EncryptMessage`, `DecryptMessage`, `GetChannelKey`, `SetChannelKey`

### 6. models 字段类型不匹配
- **时间戳**: 很多地方用`time.Now().Unix()`（int64），但字段类型是`time.Time`
- **Channel字段**: 使用了不存在的`Description`, `Mode`字段
- **MuteRecord**: `ExpiresAt`字段类型不匹配
- **Message.Content**: 类型是`MessageContent`，不是`string`

## 详细错误列表

### server.go (14个错误)
1. Line 38: `storage.DatabaseManager` 不存在 → 改为 `storage.Database`
2. Line 114: 同上
3. Line 131: `utils.NewLogger` 返回2个值 → 需要处理error
4. Line 147: `crypto.NewManager` 返回1个值 → 已修复为返回2个值
5. Line 281-285: Transport配置和创建方式错误
6. Line 297: `msg.From` 不存在 → 改为 `msg.SenderID`
7. Lines 307-319: MessageType使用字符串 → 需要定义应用层类型
8. Line 381, 386: 返回值复制锁 → 返回指针或拷贝非锁字段

### channel_manager.go (9个错误)
1. Lines 53-54: `Description`, `Mode` 字段不存在
2. Lines 56-57: 时间戳类型 `int64` vs `time.Time`
3. Line 257: 同上
4. Lines 288-289: `MutedAt`, `ExpiresAt` 类型不匹配
5. Line 366: 时间比较类型不匹配
6. Line 396: `Description` 字段不存在
7. Line 402: 时间戳类型不匹配
8. Line 447: `transport` 未定义

### broadcast_manager.go (4个错误)
1. Line 10: `events` 包导入但未使用
2. Line 130: `EncryptMessage` 方法 → 已修复
3. Lines 164-166: Message结构体字段错误
4. Line 266: 返回值复制锁

### message_router.go (4个错误)
1. Line 102: `DecryptMessage` 方法 → 已修复
2. Line 152: 不能将int转换为time.Time
3. Line 153: 时间戳类型不匹配
4. Line 192: `msg.From` 不存在

### auth_manager.go (14个错误)
1. Line 11: `events` 包导入但未使用
2-9. Lines 96-255: 多处`msg.From` → `msg.SenderID`
10. Line 99, 225: `DecryptMessage`, `EncryptMessage` → 已修复
11. Lines 144-145: 时间戳类型不匹配
12. Line 190: `GetChannelKey` → 已修复
13. Lines 253-255: Message结构体字段错误
14. Lines 273-274: Message.Content类型和时间戳类型

### challenge_manager.go (14个错误)
1. Line 77: 时间戳类型不匹配
2. Line 91: `StartedAt` 字段不存在
3-5. Lines 109-226: `msg.From` 和crypto方法
6. Line 134, 154: 时间戳类型不匹配
7. Lines 224-226: Message结构体字段错误
8. Lines 243-244, 276-277: Message.Content和时间戳类型
9. Line 323: `ChallengeID` 字段不存在于ChallengeEvent

## 修复策略

1. **第一步**: 修复基础类型（Database, crypto, utils）
2. **第二步**: 定义应用层消息类型系统
3. **第三步**: 修复Server使用Repository模式
4. **第四步**: 修复所有时间戳和字段类型不匹配
5. **第五步**: 修复transport.Message使用方式
6. **第六步**: 测试编译

