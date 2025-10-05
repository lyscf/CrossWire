# CrossWire 数据库表完整清单

> 所有表的详细字段说明
> 
> Version: 1.0.0  
> Date: 2025-10-05

---

## 📊 数据库总览

**CrossWire 共有 20 个表，分布在 3 个数据库文件中：**

- **频道数据库** (`<channel-uuid>.db`): 16 个表
- **用户数据库** (`user.db`): 3 个表  
- **缓存数据库** (`cache.db`): 1 个表

**总字段数：** 约 280+ 个字段

---

## 📋 完整表清单

### 频道数据库表

| 表名 | 字段数 | 主要功能 | 关联表 |
|------|--------|----------|--------|
| [channels](#1-channels) | 16 | 频道基本信息 | - |
| [members](#2-members) | 19 | 成员信息与CTF技能 | channels |
| [messages](#3-messages) | 20 | 聊天消息（含题目聊天室） | channels, members, challenges |
| [messages_fts](#4-messages_fts) | 3 | 全文搜索索引 | messages |
| [files](#5-files) | 20 | 文件元数据 | messages, channels |
| [file_chunks](#6-file_chunks) | 9 | 文件分块状态 | files |
| [message_reactions](#7-message_reactions) | 6 | 消息表情回应 | messages, members |
| [typing_status](#8-typing_status) | 5 | 正在输入状态 | channels, members |
| [audit_logs](#9-audit_logs) | 9 | 操作审计日志 | channels |
| [mute_records](#10-mute_records) | 9 | 禁言记录 | channels, members |
| [pinned_messages](#11-pinned_messages) | 6 | 置顶消息 | channels, messages |
| [challenges](#12-challenges) | 18 | CTF题目管理 | channels, members |
| [challenge_assignments](#13-challenge_assignments) | 7 | 题目分配关系 | challenges, members |
| [challenge_progress](#14-challenge_progress) | 10 | 题目进度跟踪 | challenges, members |
| [challenge_submissions](#15-challenge_submissions) | 9 | Flag提交记录 | challenges, members |
| [challenge_hints](#16-challenge_hints) | 8 | 题目提示 | challenges, members |

### 用户数据库表

| 表名 | 字段数 | 主要功能 | 关联表 |
|------|--------|----------|--------|
| [user_profiles](#17-user_profiles) | 11 | 用户个人资料 | - |
| [user_settings](#18-user_settings) | 20+ | 用户配置 | user_profiles |
| [recent_channels](#19-recent_channels) | 6 | 最近访问频道 | - |

### 缓存数据库表

| 表名 | 字段数 | 主要功能 | 关联表 |
|------|--------|----------|--------|
| [cache_entries](#20-cache_entries) | 4 | 通用缓存 | - |

---

## 📝 表详细定义

### 1. channels（频道表）

**字段总数：** 16  
**主键：** `id`  
**外键：** 无  
**索引数：** 1

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | TEXT | PK | - | 频道UUID |
| name | TEXT | NOT NULL | - | 频道名称（3-50字符） |
| password_hash | TEXT | NOT NULL | - | SHA256密码哈希 |
| salt | BLOB | NOT NULL | - | 32字节随机盐值 |
| created_at | INTEGER | NOT NULL | - | 创建时间（Unix纳秒） |
| creator_id | TEXT | NOT NULL | - | 创建者UUID |
| max_members | INTEGER | CHECK(1-100) | 50 | 最大成员数 |
| transport_mode | TEXT | - | 'auto' | 传输模式（arp/https/mdns/auto） |
| port | INTEGER | - | NULL | HTTPS端口（1024-65535） |
| interface | TEXT | - | NULL | ARP网卡名称 |
| encryption_key | BLOB | NOT NULL | - | AES-256密钥（32字节，加密存储） |
| key_version | INTEGER | - | 1 | 密钥版本号 |
| message_count | INTEGER | - | 0 | 消息总数 |
| file_count | INTEGER | - | 0 | 文件总数 |
| total_traffic | INTEGER | - | 0 | 总流量（字节） |
| metadata | TEXT | - | NULL | JSON扩展字段 |
| updated_at | INTEGER | NOT NULL | - | 更新时间（Unix纳秒） |

---

### 2. members（成员表）

**字段总数：** 19  
**主键：** `id`  
**外键：** `channel_id` → `channels(id)`  
**索引数：** 3

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | TEXT | PK | - | 成员UUID |
| channel_id | TEXT | FK, NOT NULL | - | 所属频道ID |
| nickname | TEXT | NOT NULL | - | 昵称（3-20字符） |
| avatar | TEXT | - | NULL | 头像（Base64/URL，最大2MB） |
| role | TEXT | CHECK, NOT NULL | - | 角色（owner/admin/member/readonly） |
| status | TEXT | CHECK | 'offline' | 状态（online/busy/away/offline） |
| public_key | BLOB | - | NULL | X25519公钥（DER格式） |
| last_ip | TEXT | - | NULL | 最后登录IP |
| last_mac | TEXT | - | NULL | 最后登录MAC地址 |
| skills | TEXT | - | NULL | JSON数组：[{"category":"Web","level":4,"experience":150,"last_used":timestamp}] |
| expertise | TEXT | - | NULL | JSON数组：[{"name":"SQL注入","description":"...","tools":["sqlmap"],"notes":"..."}] |
| current_task | TEXT | - | NULL | JSON对象：{"challenge":"Web-100","start_time":timestamp,"progress":60,"notes":"...","teammates":["user-id"]} |
| message_count | INTEGER | - | 0 | 发送消息数 |
| files_shared | INTEGER | - | 0 | 分享文件数 |
| online_time | INTEGER | - | 0 | 在线时长（秒） |
| joined_at | INTEGER | NOT NULL | - | 加入时间 |
| last_seen | INTEGER | NOT NULL | - | 最后活跃时间 |
| last_heartbeat | INTEGER | NOT NULL | - | 最后心跳时间 |
| metadata | TEXT | - | NULL | JSON扩展字段 |

---

### 3. messages（消息表）

**字段总数：** 20  
**主键：** `id`  
**外键：** `channel_id` → `channels(id)`, `sender_id` → `members(id)`, `reply_to_id` → `messages(id)`, `challenge_id` → `challenges(id)`  
**索引数：** 7

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | TEXT | PK | - | 消息UUID |
| channel_id | TEXT | FK, NOT NULL | - | 所属频道ID |
| sender_id | TEXT | FK, NOT NULL | - | 发送者ID |
| sender_nickname | TEXT | NOT NULL | - | 发送者昵称（冗余，加速查询） |
| type | TEXT | CHECK, NOT NULL | - | 消息类型（text/code/file/system/control） |
| content | TEXT | NOT NULL | - | JSON格式内容 |
| content_text | TEXT | - | NULL | 纯文本内容（用于全文搜索） |
| reply_to_id | TEXT | FK | NULL | 回复的消息ID |
| thread_id | TEXT | - | NULL | 话题ID（用于消息分组） |
| mentions | TEXT | - | NULL | JSON数组：["user-id-1", "user-id-2"] |
| tags | TEXT | - | NULL | JSON数组：["web", "sqli", "xss"] |
| pinned | INTEGER | - | 0 | 是否置顶（0/1） |
| deleted | INTEGER | - | 0 | 是否删除（0/1） |
| deleted_by | TEXT | - | NULL | 删除操作者ID |
| deleted_at | INTEGER | - | NULL | 删除时间 |
| timestamp | INTEGER | NOT NULL | - | 发送时间（Unix纳秒） |
| edited_at | INTEGER | - | NULL | 编辑时间 |
| encrypted | INTEGER | - | 1 | 是否加密（0/1） |
| key_version | INTEGER | - | 1 | 使用的密钥版本 |
| **challenge_id** | **TEXT** | **FK** | **NULL** | **题目ID（题目聊天室专用）** |
| **room_type** | **TEXT** | **CHECK** | **'main'** | **聊天室类型（main/challenge）** |
| metadata | TEXT | - | NULL | JSON扩展字段 |

**content 字段格式示例：**

```json
// type = "text"
{
  "text": "消息内容",
  "format": "markdown",
  "mentions": ["user-id"],
  "tags": ["web"]
}

// type = "code"
{
  "language": "python",
  "code": "import pwn\n...",
  "filename": "exploit.py",
  "description": "Pwn-300 exp"
}

// type = "file"
{
  "file_id": "file-uuid",
  "filename": "payload.bin",
  "size": 2048,
  "mime_type": "application/octet-stream",
  "sha256": "hash..."
}
```

---

### 4. messages_fts（全文搜索表）

**字段总数：** 3  
**类型：** FTS5 虚拟表  
**关联表：** messages

| 字段名 | 类型 | 说明 |
|--------|------|------|
| content_text | TEXT | 消息纯文本内容 |
| sender_nickname | TEXT | 发送者昵称 |
| tags | TEXT | 标签（用于筛选） |

**特性：**
- 支持中英文全文搜索
- BM25 相关性排序
- 自动同步（通过触发器）

---

### 5. files（文件表）

**字段总数：** 20  
**主键：** `id`  
**外键：** `message_id` → `messages(id)`, `channel_id` → `channels(id)`, `sender_id` → `members(id)`  
**索引数：** 5

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | TEXT | PK | - | 文件UUID |
| message_id | TEXT | FK, NOT NULL | - | 关联消息ID |
| channel_id | TEXT | FK, NOT NULL | - | 所属频道ID |
| sender_id | TEXT | FK, NOT NULL | - | 上传者ID |
| filename | TEXT | NOT NULL | - | 存储文件名（UUID） |
| original_name | TEXT | NOT NULL | - | 原始文件名 |
| size | INTEGER | NOT NULL | - | 文件大小（字节） |
| mime_type | TEXT | NOT NULL | - | MIME类型 |
| storage_type | TEXT | CHECK, NOT NULL | - | 存储类型（inline/file/reference） |
| storage_path | TEXT | - | NULL | 文件路径（storage_type='file'） |
| data | BLOB | - | NULL | 内联数据（storage_type='inline'，<1MB） |
| sha256 | TEXT | NOT NULL | - | SHA256哈希 |
| checksum | TEXT | NOT NULL | - | CRC32校验 |
| chunk_size | INTEGER | - | 8192 | 分块大小（字节） |
| total_chunks | INTEGER | NOT NULL | - | 总块数 |
| uploaded_chunks | INTEGER | - | 0 | 已上传块数 |
| upload_status | TEXT | CHECK | 'pending' | 上传状态（pending/uploading/completed/failed） |
| thumbnail | BLOB | - | NULL | 缩略图（PNG，最大100KB） |
| preview_text | TEXT | - | NULL | 文本预览（前1000字符） |
| uploaded_at | INTEGER | NOT NULL | - | 上传时间 |
| expires_at | INTEGER | - | NULL | 过期时间（可选） |
| encrypted | INTEGER | - | 1 | 是否加密 |
| encryption_key | BLOB | - | NULL | 文件专用密钥（32字节，加密存储） |
| metadata | TEXT | - | NULL | JSON扩展字段 |

---

### 6. file_chunks（文件分块表）

**字段总数：** 9  
**主键：** `(file_id, chunk_index)` 复合主键  
**外键：** `file_id` → `files(id)`  
**索引数：** 2

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| file_id | TEXT | PK, FK, NOT NULL | - | 文件ID |
| chunk_index | INTEGER | PK, NOT NULL | - | 块索引（从0开始） |
| size | INTEGER | NOT NULL | - | 块大小（字节） |
| checksum | TEXT | NOT NULL | - | CRC32校验 |
| uploaded | INTEGER | - | 0 | 是否已上传（0/1） |
| uploaded_at | INTEGER | - | NULL | 上传时间 |
| retry_count | INTEGER | - | 0 | 重传次数 |
| last_error | TEXT | - | NULL | 最后错误信息 |
| metadata | TEXT | - | NULL | JSON扩展字段 |

**SQL 定义：**
```sql
CREATE TABLE file_chunks (
    file_id         TEXT NOT NULL,
    chunk_index     INTEGER NOT NULL,
    size            INTEGER NOT NULL,
    checksum        TEXT NOT NULL,
    uploaded        INTEGER DEFAULT 0,
    uploaded_at     INTEGER,
    retry_count     INTEGER DEFAULT 0,
    last_error      TEXT,
    metadata        TEXT,
    PRIMARY KEY (file_id, chunk_index),
    FOREIGN KEY(file_id) REFERENCES files(id) ON DELETE CASCADE,
    CHECK(uploaded IN (0, 1))
);

CREATE INDEX idx_file_chunks_status ON file_chunks(file_id, uploaded);
CREATE INDEX idx_file_chunks_retry ON file_chunks(file_id, retry_count) WHERE retry_count > 0;
```

---

### 7. message_reactions（消息表情回应表）

**字段总数：** 6  
**主键：** `(message_id, member_id, reaction)` 复合主键  
**外键：** `message_id` → `messages(id)`, `member_id` → `members(id)`  
**索引数：** 2

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| message_id | TEXT | PK, FK, NOT NULL | - | 消息ID |
| member_id | TEXT | PK, FK, NOT NULL | - | 成员ID |
| reaction | TEXT | PK, NOT NULL | - | 表情符号（👍/❤️/😂等） |
| created_at | INTEGER | NOT NULL | - | 添加时间 |
| updated_at | INTEGER | NOT NULL | - | 更新时间 |
| metadata | TEXT | - | NULL | JSON扩展字段 |

**SQL 定义：**
```sql
CREATE TABLE message_reactions (
    message_id      TEXT NOT NULL,
    member_id       TEXT NOT NULL,
    reaction        TEXT NOT NULL,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    metadata        TEXT,
    PRIMARY KEY (message_id, member_id, reaction),
    FOREIGN KEY(message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY(member_id) REFERENCES members(id) ON DELETE CASCADE
);

CREATE INDEX idx_reactions_message ON message_reactions(message_id);
CREATE INDEX idx_reactions_member ON message_reactions(member_id);
```

**使用示例：**
```sql
-- 添加表情
INSERT INTO message_reactions (message_id, member_id, reaction, created_at, updated_at)
VALUES ('msg-123', 'user-456', '👍', unixepoch('now'), unixepoch('now'));

-- 获取消息的所有表情及数量
SELECT reaction, COUNT(*) as count
FROM message_reactions
WHERE message_id = 'msg-123'
GROUP BY reaction;
```

---

### 8. typing_status（正在输入状态表）

**字段总数：** 5  
**主键：** `(channel_id, member_id)` 复合主键  
**外键：** `channel_id` → `channels(id)`, `member_id` → `members(id)`  
**类型：** 临时表（内存中，定期清理）

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| channel_id | TEXT | PK, FK, NOT NULL | - | 频道ID |
| member_id | TEXT | PK, FK, NOT NULL | - | 成员ID |
| member_nickname | TEXT | NOT NULL | - | 成员昵称（冗余） |
| started_at | INTEGER | NOT NULL | - | 开始输入时间 |
| expires_at | INTEGER | NOT NULL | - | 过期时间（started_at + 5秒） |

**SQL 定义：**
```sql
CREATE TABLE typing_status (
    channel_id      TEXT NOT NULL,
    member_id       TEXT NOT NULL,
    member_nickname TEXT NOT NULL,
    started_at      INTEGER NOT NULL,
    expires_at      INTEGER NOT NULL,
    PRIMARY KEY (channel_id, member_id),
    FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    FOREIGN KEY(member_id) REFERENCES members(id) ON DELETE CASCADE
);

CREATE INDEX idx_typing_expires ON typing_status(expires_at);

-- 自动清理过期记录
CREATE TRIGGER typing_status_cleanup AFTER INSERT ON typing_status BEGIN
    DELETE FROM typing_status WHERE expires_at < unixepoch('now');
END;
```

**特点：**
- 记录自动过期（5秒）
- 触发器自动清理
- 用于实时显示"XX正在输入..."

---

### 9. audit_logs（审计日志表）

**字段总数：** 9  
**主键：** `id` (自增)  
**外键：** `channel_id` → `channels(id)`  
**索引数：** 4

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | INTEGER | PK, AUTOINCREMENT | - | 日志ID |
| channel_id | TEXT | FK, NOT NULL | - | 频道ID |
| type | TEXT | NOT NULL | - | 操作类型（kick/mute/delete_message/pin/unpin/update_channel等） |
| operator_id | TEXT | NOT NULL | - | 操作者ID |
| target_id | TEXT | - | NULL | 目标对象ID（用户/消息ID） |
| reason | TEXT | - | NULL | 操作原因 |
| details | TEXT | - | NULL | 详细信息（JSON） |
| timestamp | INTEGER | NOT NULL | - | 操作时间 |
| ip_address | TEXT | - | NULL | 操作者IP |
| user_agent | TEXT | - | NULL | 操作者User-Agent |

---

### 10. mute_records（禁言记录表）

**字段总数：** 9  
**主键：** `id`  
**外键：** `channel_id` → `channels(id)`, `member_id` → `members(id)`  
**索引数：** 3

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | TEXT | PK | - | 记录UUID |
| channel_id | TEXT | FK, NOT NULL | - | 频道ID |
| member_id | TEXT | FK, NOT NULL | - | 成员ID |
| muted_by | TEXT | NOT NULL | - | 操作者ID |
| reason | TEXT | - | NULL | 禁言原因 |
| muted_at | INTEGER | NOT NULL | - | 禁言时间 |
| duration | INTEGER | - | NULL | 持续时长（秒），NULL=永久 |
| expires_at | INTEGER | - | NULL | 过期时间 |
| active | INTEGER | - | 1 | 是否生效（0/1） |
| unmuted_at | INTEGER | - | NULL | 解除时间 |
| unmuted_by | TEXT | - | NULL | 解除操作者ID |

---

### 11. pinned_messages（置顶消息表）

**字段总数：** 6  
**主键：** `id` (自增)  
**外键：** `channel_id` → `channels(id)`, `message_id` → `messages(id)`  
**索引数：** 1

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | INTEGER | PK, AUTOINCREMENT | - | 记录ID |
| channel_id | TEXT | FK, NOT NULL | - | 频道ID |
| message_id | TEXT | FK, NOT NULL, UNIQUE | - | 消息ID |
| pinned_by | TEXT | NOT NULL | - | 操作者ID |
| reason | TEXT | - | NULL | 置顶原因 |
| pinned_at | INTEGER | NOT NULL | - | 置顶时间 |
| display_order | INTEGER | - | 0 | 显示顺序（0=最上） |

---

### 12. challenges（题目表）

**字段总数：** 18  
**主键：** `id` (UUID)  
**外键：** `channel_id` → `channels(id)`, `created_by` → `members(id)`  
**索引数：** 4

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | TEXT | PK | - | 题目UUID |
| channel_id | TEXT | FK, NOT NULL | - | 所属频道ID |
| title | TEXT | NOT NULL | - | 题目标题 |
| category | TEXT | CHECK, NOT NULL | - | 题目分类（Web/Pwn/Reverse/Crypto/Misc/Forensics） |
| difficulty | TEXT | CHECK, NOT NULL | - | 难度等级（Easy/Medium/Hard/Insane） |
| points | INTEGER | CHECK, NOT NULL | - | 分值（1-1000） |
| description | TEXT | NOT NULL | - | 题目描述 |
| flag_format | TEXT | - | NULL | Flag格式说明 |
| flag | TEXT | - | NULL | **Flag明文（所有人可见）** |
| url | TEXT | - | NULL | 题目链接 |
| attachments | TEXT | - | NULL | 附件文件ID列表（JSON） |
| tags | TEXT | - | NULL | 标签（JSON数组） |
| status | TEXT | CHECK, NOT NULL | 'open' | 题目状态（open/solved/closed） |
| solved_by | TEXT | - | NULL | 解决者ID列表（JSON） |
| solved_at | INTEGER | - | NULL | 解决时间 |
| created_by | TEXT | FK, NOT NULL | - | 创建者ID |
| created_at | INTEGER | NOT NULL | - | 创建时间 |
| updated_at | INTEGER | NOT NULL | - | 更新时间 |
| metadata | TEXT | - | NULL | 扩展元数据（JSON） |

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

### 13. challenge_assignments（题目分配表）

**字段总数：** 7  
**主键：** `(challenge_id, member_id)` 复合主键  
**外键：** `challenge_id` → `challenges(id)`, `member_id` → `members(id)`  
**索引数：** 3

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| challenge_id | TEXT | PK, FK, NOT NULL | - | 题目ID |
| member_id | TEXT | PK, FK, NOT NULL | - | 成员ID |
| assigned_by | TEXT | FK, NOT NULL | - | 分配者ID |
| assigned_at | INTEGER | NOT NULL | - | 分配时间 |
| role | TEXT | CHECK, NOT NULL | 'member' | 角色（lead/member） |
| status | TEXT | CHECK, NOT NULL | 'assigned' | 状态（assigned/working/completed） |
| notes | TEXT | - | NULL | 备注 |

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

---

### 14. challenge_progress（题目进度表）

**字段总数：** 10  
**主键：** `id` (自增)  
**外键：** `challenge_id` → `challenges(id)`, `member_id` → `members(id)`  
**索引数：** 3

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | INTEGER | PK, AUTOINCREMENT | - | 记录ID |
| challenge_id | TEXT | FK, NOT NULL | - | 题目ID |
| member_id | TEXT | FK, NOT NULL | - | 成员ID |
| progress | INTEGER | CHECK, NOT NULL | 0 | 进度百分比（0-100） |
| status | TEXT | CHECK, NOT NULL | 'not_started' | 状态（not_started/in_progress/blocked/completed） |
| summary | TEXT | - | NULL | 进度摘要 |
| findings | TEXT | - | NULL | 发现内容（JSON） |
| blockers | TEXT | - | NULL | 阻塞问题 |
| updated_at | INTEGER | NOT NULL | - | 更新时间 |
| metadata | TEXT | - | NULL | 扩展元数据 |

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

### 15. challenge_submissions（提交记录表）

**字段总数：** 8  
**主键：** `id` (UUID)  
**外键：** `challenge_id` → `challenges(id)`, `member_id` → `members(id)`  
**索引数：** 3

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | TEXT | PK | - | 提交UUID |
| challenge_id | TEXT | FK, NOT NULL | - | 题目ID |
| member_id | TEXT | FK, NOT NULL | - | 提交者ID |
| flag | TEXT | NOT NULL | - | 提交的Flag明文 |
| action | TEXT | CHECK, NOT NULL | - | 操作类型（submit/update） |
| submitted_at | INTEGER | NOT NULL | - | 提交时间 |
| ip_address | TEXT | - | NULL | 提交IP |
| metadata | TEXT | - | NULL | 扩展元数据 |

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

### 16. challenge_hints（题目提示表）

**字段总数：** 8  
**主键：** `id` (UUID)  
**外键：** `challenge_id` → `challenges(id)`, `created_by` → `members(id)`  
**索引数：** 1

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | TEXT | PK | - | 提示UUID |
| challenge_id | TEXT | FK, NOT NULL | - | 题目ID |
| order_num | INTEGER | NOT NULL | - | 提示顺序（1,2,3...） |
| content | TEXT | NOT NULL | - | 提示内容 |
| cost | INTEGER | - | 0 | 解锁成本（可选） |
| unlocked_by | TEXT | - | NULL | 已解锁成员ID列表（JSON） |
| created_by | TEXT | FK, NOT NULL | - | 创建者ID |
| created_at | INTEGER | NOT NULL | - | 创建时间 |

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

### 17. user_profiles（用户资料表）

**数据库：** `user.db`  
**字段总数：** 11  
**主键：** `id`  
**外键：** 无

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | TEXT | PK | - | 用户UUID |
| nickname | TEXT | NOT NULL | - | 默认昵称 |
| avatar | TEXT | - | NULL | 默认头像 |
| private_key | BLOB | NOT NULL | - | **X25519私钥（32字节，加密存储）** |
| public_key | BLOB | NOT NULL | - | **X25519公钥（32字节）** |
| skills | TEXT | - | NULL | 技能标签（JSON） |
| expertise | TEXT | - | NULL | 擅长领域（JSON） |
| bio | TEXT | - | NULL | 个人简介 |
| theme | TEXT | - | 'dark' | 主题（dark/light/auto） |
| language | TEXT | - | 'zh-CN' | 语言 |
| auto_start | INTEGER | - | 0 | 开机自启动（0/1） |
| created_at | INTEGER | NOT NULL | - | 创建时间 |
| updated_at | INTEGER | NOT NULL | - | 更新时间 |

---

### 18. user_settings（用户配置表）

**数据库：** `user.db`  
**字段总数：** 20+  
**主键：** `user_id`  
**外键：** `user_id` → `user_profiles(id)`

| 字段名 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| user_id | TEXT | - | 用户ID（PK, FK） |
| theme | TEXT | 'dark' | 主题 |
| language | TEXT | 'zh-CN' | 语言 |
| font_size | INTEGER | 14 | 字体大小（10-24） |
| font_family | TEXT | 'system' | 字体 |
| enable_notifications | INTEGER | 1 | 启用通知 |
| notification_sound | INTEGER | 1 | 通知声音 |
| enable_mention_notification | INTEGER | 1 | @提及通知 |
| enable_file_notification | INTEGER | 1 | 文件上传通知 |
| auto_download_files | INTEGER | 0 | 自动下载文件 |
| max_auto_download_size | INTEGER | 10485760 | 自动下载最大值（10MB） |
| enable_code_highlight | INTEGER | 1 | 代码高亮 |
| enable_markdown | INTEGER | 1 | Markdown渲染 |
| enable_emoji | INTEGER | 1 | Emoji支持 |
| message_preview_lines | INTEGER | 3 | 消息预览行数 |
| show_typing_indicator | INTEGER | 1 | 显示输入状态 |
| send_on_enter | INTEGER | 1 | 回车发送（0=Ctrl+Enter） |
| enable_spell_check | INTEGER | 1 | 拼写检查 |
| hotkey_send | TEXT | 'Ctrl+Enter' | 发送快捷键 |
| hotkey_search | TEXT | 'Ctrl+K' | 搜索快捷键 |
| hotkey_file | TEXT | 'Ctrl+U' | 上传文件快捷键 |
| created_at | INTEGER | - | 创建时间 |
| updated_at | INTEGER | - | 更新时间 |

**SQL 定义：**
```sql
CREATE TABLE user_settings (
    user_id                     TEXT PRIMARY KEY,
    theme                       TEXT DEFAULT 'dark',
    language                    TEXT DEFAULT 'zh-CN',
    font_size                   INTEGER DEFAULT 14,
    font_family                 TEXT DEFAULT 'system',
    enable_notifications        INTEGER DEFAULT 1,
    notification_sound          INTEGER DEFAULT 1,
    enable_mention_notification INTEGER DEFAULT 1,
    enable_file_notification    INTEGER DEFAULT 1,
    auto_download_files         INTEGER DEFAULT 0,
    max_auto_download_size      INTEGER DEFAULT 10485760,
    enable_code_highlight       INTEGER DEFAULT 1,
    enable_markdown             INTEGER DEFAULT 1,
    enable_emoji                INTEGER DEFAULT 1,
    message_preview_lines       INTEGER DEFAULT 3,
    show_typing_indicator       INTEGER DEFAULT 1,
    send_on_enter               INTEGER DEFAULT 1,
    enable_spell_check          INTEGER DEFAULT 1,
    hotkey_send                 TEXT DEFAULT 'Ctrl+Enter',
    hotkey_search               TEXT DEFAULT 'Ctrl+K',
    hotkey_file                 TEXT DEFAULT 'Ctrl+U',
    created_at                  INTEGER NOT NULL,
    updated_at                  INTEGER NOT NULL,
    FOREIGN KEY(user_id) REFERENCES user_profiles(id) ON DELETE CASCADE,
    CHECK(font_size BETWEEN 10 AND 24),
    CHECK(theme IN ('dark', 'light', 'auto')),
    CHECK(enable_notifications IN (0, 1)),
    CHECK(send_on_enter IN (0, 1))
);
```

---

### 19. recent_channels（最近频道表）

**数据库：** `user.db`  
**字段总数：** 6  
**主键：** `channel_id`  
**外键：** 无

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| channel_id | TEXT | PK | - | 频道ID |
| channel_name | TEXT | NOT NULL | - | 频道名称 |
| server_address | TEXT | - | NULL | 服务器地址 |
| transport_mode | TEXT | - | NULL | 传输模式 |
| last_joined | INTEGER | NOT NULL | - | 最后加入时间 |
| pinned | INTEGER | - | 0 | 是否固定（0/1） |

---

### 20. cache_entries（缓存表）

**数据库：** `cache.db`  
**字段总数：** 4  
**主键：** `key`  
**外键：** 无

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| key | TEXT | PK | - | 缓存键 |
| value | BLOB | NOT NULL | - | 缓存值 |
| expires_at | INTEGER | NOT NULL | - | 过期时间 |
| created_at | INTEGER | NOT NULL | - | 创建时间 |

---

## 📈 数据库统计

### 表统计

- **总表数：** 15
- **普通表：** 13
- **虚拟表：** 1 (messages_fts)
- **临时表：** 1 (typing_status)

### 字段统计

- **总字段数：** ~200 字段
- **主键字段：** 15
- **外键字段：** 25
- **索引字段：** 40+

### 数据类型分布

- **TEXT:** 55%
- **INTEGER:** 40%
- **BLOB:** 5%

### 约束统计

- **NOT NULL:** 80+ 约束
- **CHECK:** 15+ 约束
- **FOREIGN KEY:** 15+ 外键
- **UNIQUE:** 5+ 唯一约束

---

## 🔗 表关系总结

### 一对多关系 (1:N)

```
channels → members
channels → messages
channels → files
channels → audit_logs
channels → mute_records
channels → pinned_messages
channels → typing_status

members → messages
members → files
members → message_reactions
members → mute_records
members → typing_status

messages → files
messages → pinned_messages
messages → message_reactions

files → file_chunks
```

### 自引用关系

```
messages.reply_to_id → messages.id  (回复消息)
messages.thread_id → messages.id     (话题分组)
```

---

## 💾 数据库大小估算

**单频道估算（假设50人，使用1年）：**

| 表 | 行数 | 平均行大小 | 总大小 |
|------|------|-----------|--------|
| channels | 1 | 500B | 500B |
| members | 50 | 2KB | 100KB |
| messages | 100,000 | 1KB | 100MB |
| messages_fts | - | - | 30MB |
| files | 1,000 | 2KB + 文件 | 2MB + 文件 |
| file_chunks | 10,000 | 200B | 2MB |
| message_reactions | 5,000 | 150B | 750KB |
| audit_logs | 1,000 | 500B | 500KB |
| 其他 | - | - | 1MB |
| **总计** | - | - | **~136MB + 文件** |

**注：**
- 实际大小取决于消息频率和文件数量
- SQLite WAL 模式会增加 10-20% 空间
- 定期 VACUUM 可优化空间

---

## 🎯 设计原则

### 1. 性能优化
- ✅ 合理的索引策略
- ✅ 冗余字段减少JOIN
- ✅ 部分索引节省空间
- ✅ FTS5 全文搜索

### 2. 数据完整性
- ✅ 外键约束
- ✅ CHECK 约束
- ✅ 触发器自动维护

### 3. 扩展性
- ✅ JSON metadata 字段
- ✅ 版本化设计
- ✅ 迁移脚本支持

### 4. 安全性
- ✅ 敏感数据加密存储
- ✅ 审计日志记录
- ✅ 软删除保留历史

---

**相关文档：**
- [DATABASE.md](DATABASE.md) - 数据库主文档
- [ARCHITECTURE.md](ARCHITECTURE.md) - 系统架构
- [PROTOCOL.md](PROTOCOL.md) - 通信协议
