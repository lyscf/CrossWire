# Challenge API 简化说明

## 📝 用户需求

根据用户反馈，题目系统进行了简化：

1. ❌ **不需要验证 Flag** - 用户提交后直接接受
2. ❌ **不需要提交记录功能** - 无需查看历史提交
3. ❌ **不需要排行榜功能** - 无需排名统计
4. ✅ **用户提交后所有人可见** - 保留题目解决状态的可见性

## 🔧 已做的修改

### 1. Flag 提交简化

**修改文件**: `internal/server/challenge_manager.go`

```go
// SubmitFlag 直接接受所有Flag，不进行验证
func (cm *ChallengeManager) SubmitFlag(challengeID, memberID, flag string) error {
    // 不验证Flag，直接标记为已解决
    submission.IsCorrect = true
    // ...
}
```

**特点**:
- ✅ 所有提交的Flag都被接受
- ✅ 自动更新题目状态为"已解决"
- ✅ 将提交者添加到SolvedBy列表
- ✅ 广播解题消息，所有人可见

### 2. 禁用的功能

#### 2.1 提交记录功能

```go
// GetChallengeSubmissions 获取题目提交记录（已禁用）
func (a *App) GetChallengeSubmissions(challengeID string) Response {
    return NewErrorResponse("not_supported", "不支持提交记录功能", "")
}
```

#### 2.2 排行榜功能

```go
// GetLeaderboard 获取排行榜（已禁用）
func (a *App) GetLeaderboard() Response {
    return NewErrorResponse("not_supported", "不支持排行榜功能", "")
}
```

#### 2.3 统计功能

```go
// GetChallengeStats 获取题目统计信息（已禁用）
func (a *App) GetChallengeStats() Response {
    return NewErrorResponse("not_supported", "不支持统计功能", "")
}
```

#### 2.4 提示功能

```go
// AddHint 添加提示（已禁用）
func (a *App) AddHint(req AddHintRequest) Response {
    return NewErrorResponse("not_supported", "不支持提示功能", "")
}

// UnlockHint 解锁提示（已禁用）
func (a *App) UnlockHint(challengeID, hintID string) Response {
    return NewErrorResponse("not_supported", "不支持提示功能", "")
}
```

### 3. 保留的核心功能

#### ✅ 题目管理

- `CreateChallenge()` - 创建题目
- `GetChallenges()` - 获取题目列表
- `GetChallenge()` - 获取单个题目
- `UpdateChallenge()` - 更新题目
- `DeleteChallenge()` - 删除题目

#### ✅ 题目分配

- `AssignChallenge()` - 分配题目给成员

#### ✅ Flag提交（简化版）

- `SubmitFlag()` - 提交Flag（不验证，直接接受）
- 提交后自动广播，所有人可见

#### ✅ 进度更新

- `UpdateChallengeProgress()` - 更新题目进度

## 🎯 使用示例

### 创建题目

```javascript
// 前端调用
const response = await CreateChallenge({
    title: "Web题目示例",
    description: "找到隐藏的Flag",
    category: "Web",
    difficulty: "Easy",
    points: 100,
    flag: "flag{example}" // 可选，不会用于验证
});
```

### 提交 Flag

```javascript
// 前端调用
const response = await SubmitFlag({
    challenge_id: "challenge-id",
    flag: "任意内容" // 无需验证，直接接受
});

// 响应
{
    success: true,
    is_correct: true,
    message: "Flag已提交，服务器将接受",
    points: 100
}
```

### 查看题目状态

```javascript
// 获取题目列表
const challenges = await GetChallenges();

// 每个题目包含:
{
    id: "...",
    title: "...",
    is_solved: true,
    solved_by: ["member-1", "member-2"], // 所有解决者
    // ...
}
```

## 📊 数据流

```
用户提交Flag
    ↓
服务端直接接受
    ↓
更新题目状态（已解决）
    ↓
添加到SolvedBy列表
    ↓
广播解题消息
    ↓
所有客户端更新显示
```

## 🔄 与其他模块的关系

### Server 模块

- ✅ 实现了所有必要的包装方法
- ✅ 简化了Flag验证逻辑
- ❌ 移除了排行榜和统计方法

### Client 模块

- ✅ 可以提交Flag
- ✅ 接收解题广播
- ✅ 更新本地题目状态
- ❌ 不需要获取排行榜

### 数据库

- ✅ 保存题目信息
- ✅ 保存解决状态
- ❌ 不保存详细提交记录

## ⚠️ 注意事项

1. **安全性**: 由于不验证Flag，任何人都可以标记题目为已解决
2. **历史记录**: 无法查看谁在什么时候提交了什么Flag
3. **排名**: 无法生成基于分数的排行榜
4. **审计**: 没有详细的操作审计日志

这些都是根据用户需求有意简化的功能。

## 🚀 下一步

如果未来需要恢复这些功能：

1. **Flag验证**: 在 `SubmitFlag()` 中添加验证逻辑
2. **提交记录**: 恢复 `GetChallengeSubmissions()` 的实现
3. **排行榜**: 恢复 `GetLeaderboard()` 和相关计算逻辑
4. **统计**: 恢复 `GetChallengeStats()` 的实现

所有底层数据结构都已经准备好，只需要在API层恢复相应的查询和处理逻辑即可。

