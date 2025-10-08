# Flag明文存储迁移文档

## 🎯 修改目标

**当前问题**: 代码中存在大量竞赛平台的遗留逻辑，将Flag进行Hash加密存储，与协作平台的设计理念冲突。

**设计原则**:
- ✅ **协作平台（非竞赛平台）** - 不验证Flag正确性
- ✅ **Flag明文存储** - 所有成员可查看
- ✅ **Flag可修改** - 成员可以更新已提交的Flag
- ✅ **贡献度统计** - 关注成员参与度而非解题正确性

---

## 📋 发现的问题

### 问题1: Challenge模型中的FlagHash字段 ❌

**位置**: `internal/models/challenge.go:20`

**当前代码**:
```go
type Challenge struct {
    // ...
    FlagHash     string      `gorm:"type:text" json:"-"` // 不返回给前端
    // ...
}
```

**问题**:
- ❌ 使用Hash存储，前端无法查看
- ❌ `json:"-"` 隐藏字段，无法对成员可见
- ❌ 与协作平台设计冲突

**应该改为**:
```go
type Challenge struct {
    // ...
    Flag         string      `gorm:"type:text" json:"flag"` // 明文存储，对所有人可见
    // ...
}
```

---

### 问题2: ChallengeRepository的Hash相关方法 ❌

**位置**: `internal/storage/challenge_repository.go:94-106`

**当前代码**:
```go
// HashFlag 对Flag进行哈希（加盐）
func (r *ChallengeRepository) HashFlag(flag, challengeID string) string {
    h := sha256.New()
    h.Write([]byte(flag))
    h.Write([]byte(challengeID)) // 使用challengeID作为盐值
    return hex.EncodeToString(h.Sum(nil))
}

// VerifyFlag 验证Flag
func (r *ChallengeRepository) VerifyFlag(submitted, challengeID, storedHash string) bool {
    computedHash := r.HashFlag(submitted, challengeID)
    return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}
```

**问题**:
- ❌ 竞赛平台逻辑，不适用于协作平台
- ❌ 这两个方法在当前代码中**未被使用**
- ❌ 引入了不必要的依赖：`crypto/sha256`, `crypto/subtle`

**应该**:
- 🗑️ **直接删除这两个方法**
- 🗑️ 删除相关的import

---

### 问题3: 文档中的矛盾描述

#### docs/DATABASE.md ❌

**位置**: `docs/DATABASE.md:1290`

**当前内容**:
```sql
CREATE TABLE challenges (
    -- ...
    flag_hash       TEXT,                       -- SHA256，不存明文
    -- ...
);
```

**应该改为**:
```sql
CREATE TABLE challenges (
    -- ...
    flag            TEXT,                       -- Flag明文，所有人可见
    -- ...
);
```

---

#### docs/ARCHITECTURE.md ❌

**位置**: `docs/ARCHITECTURE.md:1226, 1231-1239`

**当前内容**:
```go
// ❌ 错误的设计
- ❌ **不存储明文Flag**：只存储 SHA256 哈希

func hashFlag(flag, challengeID string) string {
    h := sha256.New()
    h.Write([]byte(flag + challengeID))
    return hex.EncodeToString(h.Sum(nil))
}

func verifyFlag(submitted, challengeID, storedHash string) bool {
    computedHash := hashFlag(submitted, challengeID)
    return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}
```

**应该改为**:
```markdown
### Flag管理原则（协作平台）

- ✅ **明文存储Flag**：所有成员可查看
- ✅ **不验证Flag**：不判断正确与否
- ✅ **可修改Flag**：成员可以更新已提交的Flag
- ✅ **关注贡献度**：统计参与而非正确性

// 不需要hashFlag和verifyFlag方法
```

---

#### docs/CHALLENGE_SYSTEM.md ⚠️ 部分矛盾

**位置**: `docs/CHALLENGE_SYSTEM.md:389, 502`

**矛盾内容**:
```go
// 第389行：错误的设计
Challenge{
    FlagHash:    hashFlag(config.Flag),  // 不存储明文
}

// 第502行：正确的设计
Submission{
    Flag:        flag,  // 明文存储
}
```

**应该统一为**:
```go
Challenge{
    Flag:        config.Flag,  // 明文存储
}

Submission{
    Flag:        flag,  // 明文存储
}
```

---

#### internal/server/SERVER_TODO.md ❌

**位置**: `internal/server/SERVER_TODO.md:254, 272, 308`

**当前内容**:
```markdown
// 3. Hash Flag
// 2. Hash Flag 并对比
- [ ] 实现 SubmitFlag 方法（含 Flag Hash 对比）
```

**应该改为**:
```markdown
// 3. 存储Flag明文
// 2. 记录Flag提交（不验证）
- [ ] 实现 SubmitFlag 方法（明文存储，不验证）
```

---

## 🔧 详细修改清单

### 修改1: Challenge模型 (P0 - 必须)

**文件**: `internal/models/challenge.go`

**修改位置**: 第20行

**修改前**:
```go
type Challenge struct {
    ID           string      `gorm:"primaryKey;type:text" json:"id"`
    ChannelID    string      `gorm:"type:text;not null;index:idx_challenges_channel" json:"channel_id"`
    SubChannelID string      `gorm:"type:text" json:"sub_channel_id,omitempty"`
    Title        string      `gorm:"type:text;not null" json:"title"`
    Category     string      `gorm:"type:text;not null;index:idx_challenges_category" json:"category"`
    Difficulty   string      `gorm:"type:text;not null" json:"difficulty"`
    Points       int         `gorm:"type:integer;not null" json:"points"`
    Description  string      `gorm:"type:text;not null" json:"description"`
    FlagFormat   string      `gorm:"type:text" json:"flag_format,omitempty"`
    FlagHash     string      `gorm:"type:text" json:"-"` // 不返回给前端  ← 删除此行
    URL          string      `gorm:"type:text" json:"url,omitempty"`
    // ...
}
```

**修改后**:
```go
type Challenge struct {
    ID           string      `gorm:"primaryKey;type:text" json:"id"`
    ChannelID    string      `gorm:"type:text;not null;index:idx_challenges_channel" json:"channel_id"`
    SubChannelID string      `gorm:"type:text" json:"sub_channel_id,omitempty"`
    Title        string      `gorm:"type:text;not null" json:"title"`
    Category     string      `gorm:"type:text;not null;index:idx_challenges_category" json:"category"`
    Difficulty   string      `gorm:"type:text;not null" json:"difficulty"`
    Points       int         `gorm:"type:integer;not null" json:"points"`
    Description  string      `gorm:"type:text;not null" json:"description"`
    FlagFormat   string      `gorm:"type:text" json:"flag_format,omitempty"`
    Flag         string      `gorm:"type:text" json:"flag"` // 明文存储，对所有人可见  ← 新增此行
    URL          string      `gorm:"type:text" json:"url,omitempty"`
    // ...
}
```

**影响**:
- ✅ Flag对前端可见
- ✅ 明文存储
- ⚠️ 需要数据库迁移（如果有现有数据）

---

### 修改2: ChallengeRepository - 删除Hash方法 (P0 - 必须)

**文件**: `internal/storage/challenge_repository.go`

**修改位置**: 第1-10行 (imports), 第94-106行

**修改前**:
```go
package storage

import (
    "crypto/sha256"          // ← 删除
    "crypto/subtle"          // ← 删除
    "encoding/hex"           // ← 删除
    "time"

    "crosswire/internal/models"
)

// ChallengeRepository 题目数据仓库
type ChallengeRepository struct {
    db *Database
}

// ... 其他方法 ...

// HashFlag 对Flag进行哈希（加盐）           ← 删除整个方法
func (r *ChallengeRepository) HashFlag(flag, challengeID string) string {
    h := sha256.New()
    h.Write([]byte(flag))
    h.Write([]byte(challengeID)) // 使用challengeID作为盐值
    return hex.EncodeToString(h.Sum(nil))
}

// VerifyFlag 验证Flag                      ← 删除整个方法
func (r *ChallengeRepository) VerifyFlag(submitted, challengeID, storedHash string) bool {
    computedHash := r.HashFlag(submitted, challengeID)
    return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}
```

**修改后**:
```go
package storage

import (
    "time"

    "crosswire/internal/models"
)

// ChallengeRepository 题目数据仓库
type ChallengeRepository struct {
    db *Database
}

// ... 其他方法 ...

// HashFlag 和 VerifyFlag 方法已删除
// 协作平台不需要验证Flag，直接明文存储
```

**影响**:
- ✅ 删除不必要的依赖
- ✅ 简化代码逻辑
- ✅ 代码更符合协作平台设计

---

### 修改3: docs/DATABASE.md (P1 - 文档)

**文件**: `docs/DATABASE.md`

**修改位置**: 第1290行

**修改前**:
```sql
CREATE TABLE challenges (
    id              TEXT PRIMARY KEY,
    channel_id      TEXT NOT NULL,
    title           TEXT NOT NULL,
    category        TEXT NOT NULL,              -- Web/Pwn/Reverse/Crypto/Misc/Forensics
    difficulty      TEXT NOT NULL,              -- Easy/Medium/Hard/Insane
    points          INTEGER NOT NULL,
    description     TEXT NOT NULL,
    flag_format     TEXT,
    flag_hash       TEXT,                       -- SHA256，不存明文
    url             TEXT,
    -- ...
);
```

**修改后**:
```sql
CREATE TABLE challenges (
    id              TEXT PRIMARY KEY,
    channel_id      TEXT NOT NULL,
    title           TEXT NOT NULL,
    category        TEXT NOT NULL,              -- Web/Pwn/Reverse/Crypto/Misc/Forensics
    difficulty      TEXT NOT NULL,              -- Easy/Medium/Hard/Insane
    points          INTEGER NOT NULL,
    description     TEXT NOT NULL,
    flag_format     TEXT,
    flag            TEXT,                       -- Flag明文，所有人可见
    url             TEXT,
    -- ...
);
```

---

### 修改4: docs/ARCHITECTURE.md (P1 - 文档)

**文件**: `docs/ARCHITECTURE.md`

**修改位置**: 第1226行, 第1231-1239行

**需要删除的内容**:
```go
// 删除以下错误的设计说明
- ❌ **不存储明文Flag**：只存储 SHA256 哈希

// 删除以下方法示例
func hashFlag(flag, challengeID string) string {
    h := sha256.New()
    h.Write([]byte(flag + challengeID))
    return hex.EncodeToString(h.Sum(nil))
}

func verifyFlag(submitted, challengeID, storedHash string) bool {
    computedHash := hashFlag(submitted, challengeID)
    return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}
```

**新增内容**:
```markdown
### Flag管理原则（协作平台设计）

CrossWire是**协作平台，而非竞赛平台**，Flag管理遵循以下原则：

#### ✅ 明文存储
- Flag以**明文形式**存储在 `challenges.flag` 字段
- 对所有频道成员**完全可见**
- 便于团队协作和知识共享

#### ✅ 不验证正确性
- 系统**不判断**提交的Flag是否正确
- 不需要 `HashFlag()` 或 `VerifyFlag()` 方法
- 所有提交的Flag都被记录

#### ✅ 允许修改
- 成员可以**重复提交**Flag
- 可以**更新**之前提交的Flag
- 所有提交历史都被保留

#### ✅ 关注贡献度
- 统计成员的**参与度**（提交次数、消息数、文件分享数）
- 而非解题**正确性**（不计分、不排名）

#### 数据模型示例

```go
// Challenge 题目（包含Flag）
type Challenge struct {
    Flag         string `json:"flag"` // 明文存储，对所有人可见
    // ...
}

// ChallengeSubmission 提交记录
type ChallengeSubmission struct {
    Flag         string `json:"flag"` // 成员提交的Flag，明文可见
    // 无 IsCorrect 字段
    // ...
}
```
```

---

### 修改5: docs/CHALLENGE_SYSTEM.md (P1 - 文档)

**文件**: `docs/CHALLENGE_SYSTEM.md`

**修改位置**: 第389行

**修改前**:
```go
Challenge{
    Title:       config.Title,
    Category:    config.Category,
    Difficulty:  config.Difficulty,
    Points:      config.Points,
    Description: config.Description,
    FlagHash:    hashFlag(config.Flag),  // 不存储明文
    // ...
}
```

**修改后**:
```go
Challenge{
    Title:       config.Title,
    Category:    config.Category,
    Difficulty:  config.Difficulty,
    Points:      config.Points,
    Description: config.Description,
    Flag:        config.Flag,  // 明文存储，对所有人可见
    // ...
}
```

---

### 修改6: internal/server/SERVER_TODO.md (P2 - TODO文档)

**文件**: `internal/server/SERVER_TODO.md`

**修改位置**: 第254行, 第272行, 第308行

**需要搜索并替换**:
- "Hash Flag" → "存储Flag明文"
- "Flag Hash 对比" → "记录Flag提交"
- "含 Flag Hash 对比" → "明文存储，不验证"

**具体修改**:

**修改前**:
```markdown
### CreateChallenge 流程

1. 验证输入
2. 创建 Challenge 记录
3. Hash Flag            ← 改
4. 插入数据库

### SubmitFlag 流程

1. 验证输入
2. Hash Flag 并对比    ← 改
3. 记录提交

### TODO

- [ ] 实现 SubmitFlag 方法（含 Flag Hash 对比）  ← 改
```

**修改后**:
```markdown
### CreateChallenge 流程

1. 验证输入
2. 创建 Challenge 记录
3. 存储Flag明文
4. 插入数据库

### SubmitFlag 流程

1. 验证输入
2. 记录Flag提交（不验证正确性）
3. 更新成员贡献度统计

### TODO

- [ ] 实现 SubmitFlag 方法（明文存储，不验证）
```

---

## 🗄️ 数据库迁移

### 如果已有生产数据

**迁移SQL**:
```sql
-- 1. 添加新的 flag 字段
ALTER TABLE challenges ADD COLUMN flag TEXT;

-- 2. 如果有原始Flag数据，需要手动迁移
-- （因为 flag_hash 无法逆向还原为明文）
-- 可以考虑：
--   a) 重新输入原始Flag
--   b) 设置为空或默认值

-- 3. 删除旧的 flag_hash 字段
ALTER TABLE challenges DROP COLUMN flag_hash;
```

### 如果是新项目/无生产数据

**直接使用新的表结构即可**（GORM会自动创建）

---

## ✅ 验证清单

修改完成后，请逐项验证：

### 代码层面

- [ ] `internal/models/challenge.go` 中 `FlagHash` 字段已删除
- [ ] `internal/models/challenge.go` 中 `Flag` 字段已添加（明文，json可见）
- [ ] `internal/storage/challenge_repository.go` 中 `HashFlag` 方法已删除
- [ ] `internal/storage/challenge_repository.go` 中 `VerifyFlag` 方法已删除
- [ ] `internal/storage/challenge_repository.go` 中不必要的import已删除 (`crypto/sha256`, `crypto/subtle`, `encoding/hex`)
- [ ] `internal/app` 中没有调用 `HashFlag` 或 `VerifyFlag` 的代码
- [ ] `internal/server` 中没有调用 `HashFlag` 或 `VerifyFlag` 的代码

### 文档层面

- [ ] `docs/DATABASE.md` 中 `flag_hash` 已改为 `flag`
- [ ] `docs/ARCHITECTURE.md` 中删除了Hash相关的设计说明
- [ ] `docs/ARCHITECTURE.md` 中新增了协作平台的Flag管理原则
- [ ] `docs/CHALLENGE_SYSTEM.md` 中所有 `FlagHash` 都改为 `Flag`
- [ ] `internal/server/SERVER_TODO.md` 中Hash相关TODO已更新

### 功能验证

- [ ] 创建题目时，Flag以明文存储到数据库
- [ ] 前端获取题目详情时，可以看到Flag字段（`json:"flag"`）
- [ ] 提交Flag时，明文记录到 `challenge_submissions.flag`
- [ ] 所有成员都能查看已提交的Flag
- [ ] 不验证Flag正确性（无IsCorrect判断）

---

## 🎯 修改优先级

| 优先级 | 修改项 | 文件 | 预计时间 |
|--------|--------|------|----------|
| **P0** | 删除FlagHash字段 | `internal/models/challenge.go` | 2分钟 |
| **P0** | 添加Flag字段 | `internal/models/challenge.go` | 2分钟 |
| **P0** | 删除HashFlag方法 | `internal/storage/challenge_repository.go` | 2分钟 |
| **P0** | 删除VerifyFlag方法 | `internal/storage/challenge_repository.go` | 2分钟 |
| **P0** | 删除crypto imports | `internal/storage/challenge_repository.go` | 1分钟 |
| **P1** | 更新DATABASE.md | `docs/DATABASE.md` | 5分钟 |
| **P1** | 更新ARCHITECTURE.md | `docs/ARCHITECTURE.md` | 10分钟 |
| **P1** | 更新CHALLENGE_SYSTEM.md | `docs/CHALLENGE_SYSTEM.md` | 5分钟 |
| **P2** | 更新SERVER_TODO.md | `internal/server/SERVER_TODO.md` | 3分钟 |
| **总计** | | | **32分钟** |

---

## 🔍 潜在的其他文件

**需要额外检查**:
```bash
# 搜索所有包含 FlagHash 的文件
grep -r "FlagHash" .

# 搜索所有包含 HashFlag 的文件
grep -r "HashFlag" .

# 搜索所有包含 VerifyFlag 的文件
grep -r "VerifyFlag" .

# 搜索所有包含 "flag.*hash" 的文件（不区分大小写）
grep -ri "flag.*hash" .
```

**可能存在的其他位置**:
- `internal/app/challenge_api.go` - 创建题目的API
- `internal/client/challenge_manager.go` - 客户端题目管理
- `internal/server/challenge_manager.go` - 服务端题目管理
- `frontend/src/components/ChallengeView.vue` - 前端题目组件
- 测试文件（如果有）

---

## 📌 总结

**核心改动**:
1. ✅ `Challenge.FlagHash` → `Challenge.Flag`（明文，json可见）
2. 🗑️ 删除 `HashFlag()` 方法
3. 🗑️ 删除 `VerifyFlag()` 方法
4. 📝 更新所有相关文档

**设计理念**:
- 🤝 **协作**优先于竞赛
- 👀 **透明**优先于隐藏
- 📊 **贡献度**优先于正确性

**预计工作量**: 32分钟

**阻塞风险**: 低
- 这些方法当前未被使用
- 删除不会影响现有功能
- 只是清理遗留代码

---

**报告人**: AI Assistant  
**报告日期**: 2025-10-07  
**状态**: ⚠️ **待修复** - 发现竞赛平台遗留逻辑，需要清理
