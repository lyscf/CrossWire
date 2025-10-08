# CrossWire 数据库设计文档

> CTF 线下赛通讯系统 - 数据库与数据结构
> 
> Version: 1.0.0  
> Date: 2025-10-05

---

## 📑 目录

- [1. 数据库概述](#1-数据库概述)
- [2. 数据库Schema](#2-数据库schema)
- [3. 数据结构定义](#3-数据结构定义)
- [4. 索引策略](#4-索引策略)
- [5. 查询优化](#5-查询优化)
- [6. 数据迁移](#6-数据迁移)

---

## 1. 数据库概述

### 1.1 技术选型

**SQLite 3+GORM**

**选择理由：**
- ✅ **零配置**：单文件数据库，无需安装
- ✅ **跨平台**：支持 Windows/Linux/macOS
- ✅ **轻量级**：适合桌面应用
- ✅ **ACID**：完整的事务支持
- ✅ **JSON**：支持 JSON1 扩展

**性能指标：**
- 读取：10,000+ ops/sec
- 写入：500-1,000 ops/sec（WAL 模式）
- 数据库大小限制：281 TB
- 单表行数限制：2^64

---

### 1.2 数据库配置

```sql
-- 启用 WAL 模式（Write-Ahead Logging）
PRAGMA journal_mode = WAL;

-- 设置缓存大小 (20MB)
PRAGMA cache_size = -20000;

-- 启用外键约束
PRAGMA foreign_keys = ON;

-- 设置同步模式（平衡性能与安全）
PRAGMA synchronous = NORMAL;

-- 设置临时文件位置
PRAGMA temp_store = MEMORY;

-- 启用 mmap（加速读取）
PRAGMA mmap_size = 268435456;  -- 256MB

-- 设置页面大小
PRAGMA page_size = 4096;

-- 启用自动清理
PRAGMA auto_vacuum = INCREMENTAL;
```

---

### 1.3 数据库文件结构

```
~/.crosswire/
├── channels/
│   ├── <channel-uuid>.db       # 频道主数据库
│   ├── <channel-uuid>.db-wal   # WAL 日志
│   └── <channel-uuid>.db-shm   # 共享内存
├── user.db                      # 用户配置
└── cache.db                     # 本地缓存
```

---

### 1.4 数据库表统计

CrossWire 共使用 **3 个数据库文件**，包含 **20 个表**：

#### 频道数据库 (`<channel-uuid>.db`)

| 序号 | 表名 | 类型 | 行数预估 | 说明 |
|------|------|------|----------|------|
| 1 | `channels` | 普通表 | 1 | 频道基本信息 |
| 2 | `members` | 普通表 | 10-50 | 频道成员信息 |
| 3 | `messages` | 普通表 | 10,000+ | 聊天消息记录（含题目聊天室） |
| 4 | `messages_fts` | 虚拟表 | - | 消息全文搜索索引 |
| 5 | `files` | 普通表 | 1,000+ | 文件元数据 |
| 6 | `audit_logs` | 普通表 | 1,000+ | 审计日志 |
| 7 | `mute_records` | 普通表 | 10-100 | 禁言记录 |
| 8 | `pinned_messages` | 普通表 | 5-20 | 置顶消息 |
| 9 | `file_chunks` | 普通表 | 10,000+ | 文件分块状态 |
| 10 | `message_reactions` | 普通表 | 5,000+ | 消息表情回应 |
| 11 | `typing_status` | 临时表 | 10-50 | 正在输入状态 |
| **12** | **`challenges`** | **普通表** | **50-200** | **CTF题目管理** |
| **13** | **`challenge_assignments`** | **普通表** | **100-500** | **题目分配关系** |
| **14** | **`challenge_progress`** | **普通表** | **500-2,000** | **题目进度记录** |
| **15** | **`challenge_submissions`** | **普通表** | **1,000-5,000** | **Flag提交记录** |
| **16** | **`challenge_hints`** | **普通表** | **100-300** | **题目提示** |

#### 用户数据库 (`user.db`)

| 序号 | 表名 | 类型 | 行数预估 | 说明 |
|------|------|------|----------|------|
| 17 | `user_profiles` | 普通表 | 1 | 用户个人资料 |
| 18 | `recent_channels` | 普通表 | 10-50 | 最近加入的频道 |
| 19 | `user_settings` | 普通表 | 1 | 用户配置 |

#### 缓存数据库 (`cache.db`)

| 序号 | 表名 | 类型 | 行数预估 | 说明 |
|------|------|------|----------|------|
| 20 | `cache_entries` | 普通表 | 1,000+ | 通用缓存 |

---

### 1.5 表关系图

```
频道数据库 (<channel-uuid>.db)
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  channels (1)                                           │
│    ├─→ members (N)                                      │
│    │    ├─→ mute_records (N)                            │
│    │    └─→ typing_status (N)                           │
│    │                                                     │
│    ├─→ messages (N)                                     │
│    │    ├─→ messages_fts (虚拟表)                       │
│    │    ├─→ message_reactions (N)                       │
│    │    ├─→ files (N)                                   │
│    │    │    └─→ file_chunks (N)                        │
│    │    └─→ pinned_messages (N)                         │
│    │                                                     │
│    └─→ audit_logs (N)                                   │
│                                                         │
└─────────────────────────────────────────────────────────┘

用户数据库 (user.db)
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  user_profiles (1)                                      │
│    ├─→ user_settings (1)                                │
│    └─→ recent_channels (N)                              │
│                                                         │
└─────────────────────────────────────────────────────────┘

缓存数据库 (cache.db)
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  cache_entries (独立)                                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## 2. 数据库 Schema

### 2.1 频道表 (channels)

**用途：** 存储频道的基本信息和配置

**主键：** `id` (UUID)

**外键：** 无

**字段详细说明：**

| 字段名 | 数据类型 | 约束 | 默认值 | 说明 | 示例值 |
|--------|----------|------|--------|------|--------|
| `id` | TEXT | PRIMARY KEY | - | 频道唯一标识 | `550e8400-e29b-41d4-a716-446655440000` |
| `name` | TEXT | NOT NULL | - | 频道名称 | `CTF-Team-Alpha` |
| `password_hash` | TEXT | NOT NULL | - | 密码 SHA256 哈希 | `5e884898da...` |
| `salt` | BLOB | NOT NULL | - | 密码盐值（32字节） | `\x3a5c...` |
| `created_at` | INTEGER | NOT NULL | - | 创建时间（Unix纳秒） | `1696512000000000000` |
| `creator_id` | TEXT | NOT NULL | - | 创建者用户ID | `user-uuid` |
| `max_members` | INTEGER | CHECK | 50 | 最大成员数 | `50` |
| `transport_mode` | TEXT | - | `'auto'` | 传输模式 | `'arp'`, `'https'`, `'mdns'`, `'auto'` |
| `port` | INTEGER | - | NULL | HTTPS端口 | `8443` |
| `interface` | TEXT | - | NULL | ARP网卡名 | `eth0`, `Wi-Fi` |
| `encryption_key` | BLOB | NOT NULL | - | AES-256密钥（加密存储） | `\x1a2b...` (32字节) |
| `key_version` | INTEGER | - | 1 | 密钥版本号 | `1` |
| `message_count` | INTEGER | - | 0 | 消息总数 | `12543` |
| `file_count` | INTEGER | - | 0 | 文件总数 | `87` |
| `total_traffic` | INTEGER | - | 0 | 总流量（字节） | `1073741824` |
| `metadata` | TEXT | - | NULL | 扩展元数据（JSON） | `{"theme":"dark"}` |
| `updated_at` | INTEGER | NOT NULL | - | 更新时间（Unix纳秒） | `1696512000000000000` |

**索引：**
```sql
CREATE INDEX idx_channels_created_at ON channels(created_at);
```

**SQL 定义：**
```sql
CREATE TABLE channels (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    password_hash   TEXT NOT NULL,
    salt            BLOB NOT NULL,
    created_at      INTEGER NOT NULL,
    creator_id      TEXT NOT NULL,
    max_members     INTEGER DEFAULT 50,
    transport_mode  TEXT DEFAULT 'auto',
    port            INTEGER,
    interface       TEXT,
    encryption_key  BLOB NOT NULL,
    key_version     INTEGER DEFAULT 1,
    message_count   INTEGER DEFAULT 0,
    file_count      INTEGER DEFAULT 0,
    total_traffic   INTEGER DEFAULT 0,
    metadata        TEXT,
    updated_at      INTEGER NOT NULL,
    CHECK(max_members > 0 AND max_members <= 100)
);

CREATE INDEX idx_channels_created_at ON channels(created_at);
```

---

### 2.2 成员表 (members)

**用途：** 存储频道成员信息、CTF技能、在线状态等

**主键：** `id` (UUID)

**外键：** `channel_id` → `channels(id)`

**字段详细说明：**

| 字段名 | 数据类型 | 约束 | 默认值 | 说明 | 示例值 |
|--------|----------|------|--------|------|--------|
| `id` | TEXT | PRIMARY KEY | - | 成员唯一标识 | `user-550e8400-e29b` |
| `channel_id` | TEXT | NOT NULL, FK | - | 所属频道ID | `channel-uuid` |
| `nickname` | TEXT | NOT NULL | - | 显示昵称 | `alice` |
| `avatar` | TEXT | - | NULL | 头像（Base64/URL） | `data:image/png;base64,...` |
| `role` | TEXT | NOT NULL, CHECK | - | 角色 | `'owner'`, `'admin'`, `'member'`, `'readonly'` |
| `status` | TEXT | CHECK | `'offline'` | 在线状态 | `'online'`, `'busy'`, `'away'`, `'offline'` |
| `public_key` | BLOB | - | NULL | RSA公钥（DER格式） | `\x30820122...` |
| `last_ip` | TEXT | - | NULL | 最后登录IP | `192.168.1.100` |
| `last_mac` | TEXT | - | NULL | 最后登录MAC地址 | `00:1A:2B:3C:4D:5E` |
| `skills` | TEXT | - | NULL | 技能标签（JSON数组） | `[{"category":"Web","level":4,"experience":150}]` |
| `expertise` | TEXT | - | NULL | 擅长领域（JSON数组） | `[{"name":"SQL注入","tools":["sqlmap"]}]` |
| `current_task` | TEXT | - | NULL | 当前任务（JSON对象） | `{"challenge":"Web-100","progress":60}` |
| `message_count` | INTEGER | - | 0 | 发送消息总数 | `543` |
| `files_shared` | INTEGER | - | 0 | 分享文件总数 | `12` |
| `online_time` | INTEGER | - | 0 | 累计在线时长（秒） | `3600` |
| `joined_at` | INTEGER | NOT NULL | - | 加入时间（Unix纳秒） | `1696512000000000000` |
| `last_seen` | INTEGER | NOT NULL | - | 最后活跃时间 | `1696512000000000000` |
| `last_heartbeat` | INTEGER | NOT NULL | - | 最后心跳时间 | `1696512000000000000` |
| `metadata` | TEXT | - | NULL | 扩展元数据（JSON） | `{"theme":"dark"}` |

**索引：**
```sql
CREATE INDEX idx_members_channel ON members(channel_id);
CREATE INDEX idx_members_status ON members(channel_id, status);
CREATE INDEX idx_members_last_seen ON members(last_seen);
```

**SQL 定义：**
```sql
CREATE TABLE members (
    id              TEXT PRIMARY KEY,
    channel_id      TEXT NOT NULL,
    nickname        TEXT NOT NULL,
    avatar          TEXT,
    role            TEXT NOT NULL,
    status          TEXT DEFAULT 'offline',
    public_key      BLOB,
    last_ip         TEXT,
    last_mac        TEXT,
    skills          TEXT,
    expertise       TEXT,
    current_task    TEXT,
    message_count   INTEGER DEFAULT 0,
    files_shared    INTEGER DEFAULT 0,
    online_time     INTEGER DEFAULT 0,
    joined_at       INTEGER NOT NULL,
    last_seen       INTEGER NOT NULL,
    last_heartbeat  INTEGER NOT NULL,
    metadata        TEXT,
    FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    CHECK(role IN ('owner', 'admin', 'member', 'readonly')),
    CHECK(status IN ('online', 'busy', 'away', 'offline'))
);

CREATE INDEX idx_members_channel ON members(channel_id);
CREATE INDEX idx_members_status ON members(channel_id, status);
CREATE INDEX idx_members_last_seen ON members(last_seen);
```

---

### 2.3 消息表 (messages)

```sql
CREATE TABLE messages (
    id              TEXT PRIMARY KEY,           -- UUID
    channel_id      TEXT NOT NULL,
    sender_id       TEXT NOT NULL,
    sender_nickname TEXT NOT NULL,              -- 冗余，加速查询
    
    -- 内容
    type            TEXT NOT NULL,               -- 'text', 'code', 'file', 'system'
    content         TEXT NOT NULL,               -- JSON 格式内容
    content_text    TEXT,                        -- 纯文本（用于全文搜索）
    
    -- 关系
    reply_to_id     TEXT,                        -- 回复的消息 ID
    thread_id       TEXT,                        -- 话题 ID
    
    -- 标签
    mentions        TEXT,                        -- JSON: ["user-id-1", "user-id-2"]
    tags            TEXT,                        -- JSON: ["web", "sqli"]
    
    -- 状态
    pinned          INTEGER DEFAULT 0,           -- 0=未置顶, 1=已置顶
    deleted         INTEGER DEFAULT 0,           -- 0=正常, 1=已删除
    deleted_by      TEXT,
    deleted_at      INTEGER,
    
    -- 时间
    timestamp       INTEGER NOT NULL,            -- Unix 纳秒
    edited_at       INTEGER,
    
    -- 加密
    encrypted       INTEGER DEFAULT 1,           -- 内容是否加密
    key_version     INTEGER DEFAULT 1,           -- 使用的密钥版本
    
    -- 元数据
    metadata        TEXT,                        -- JSON 扩展
    
    FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    FOREIGN KEY(sender_id) REFERENCES members(id) ON DELETE SET NULL,
    FOREIGN KEY(reply_to_id) REFERENCES messages(id) ON DELETE SET NULL,
    CHECK(type IN ('text', 'code', 'file', 'system', 'control'))
);

CREATE INDEX idx_messages_channel_time ON messages(channel_id, timestamp DESC);
CREATE INDEX idx_messages_sender ON messages(sender_id);
CREATE INDEX idx_messages_reply_to ON messages(reply_to_id);
CREATE INDEX idx_messages_pinned ON messages(channel_id, pinned) WHERE pinned = 1;
CREATE INDEX idx_messages_deleted ON messages(deleted) WHERE deleted = 0;
```

---


---

### 2.5 文件表 (files)

```sql
CREATE TABLE files (
    id              TEXT PRIMARY KEY,           -- UUID
    message_id      TEXT NOT NULL,              -- 关联消息
    channel_id      TEXT NOT NULL,
    sender_id       TEXT NOT NULL,
    
    -- 文件信息
    filename        TEXT NOT NULL,
    original_name   TEXT NOT NULL,              -- 原始文件名
    size            INTEGER NOT NULL,           -- 字节数
    mime_type       TEXT NOT NULL,
    
    -- 存储
    storage_type    TEXT NOT NULL,              -- 'inline', 'file', 'reference'
    storage_path    TEXT,                        -- 文件路径（storage_type='file'）
    data            BLOB,                        -- 内联数据（storage_type='inline'）
    
    -- 校验
    sha256          TEXT NOT NULL,
    checksum        TEXT NOT NULL,              -- CRC32
    
    -- 传输
    chunk_size      INTEGER DEFAULT 8192,
    total_chunks    INTEGER NOT NULL,
    uploaded_chunks INTEGER DEFAULT 0,
    upload_status   TEXT DEFAULT 'pending',     -- 'pending', 'uploading', 'completed', 'failed'
    
    -- 预览
    thumbnail       BLOB,                        -- 缩略图（图片/视频）
    preview_text    TEXT,                        -- 文本预览
    
    -- 时间
    uploaded_at     INTEGER NOT NULL,
    expires_at      INTEGER,                     -- 过期时间（可选）
    
    -- 加密
    encrypted       INTEGER DEFAULT 1,
    encryption_key  BLOB,                        -- 文件专用密钥（加密存储）
    
    -- 元数据
    metadata        TEXT,                        -- JSON 扩展
    
    FOREIGN KEY(message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    FOREIGN KEY(sender_id) REFERENCES members(id) ON DELETE SET NULL,
    CHECK(storage_type IN ('inline', 'file', 'reference')),
    CHECK(upload_status IN ('pending', 'uploading', 'completed', 'failed'))
);

CREATE INDEX idx_files_message ON files(message_id);
CREATE INDEX idx_files_channel ON files(channel_id);
CREATE INDEX idx_files_sender ON files(sender_id);
CREATE INDEX idx_files_uploaded_at ON files(uploaded_at DESC);
CREATE INDEX idx_files_expires ON files(expires_at) WHERE expires_at IS NOT NULL;
```

---

### 2.6 审计日志表 (audit_logs)

```sql
CREATE TABLE audit_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id      TEXT NOT NULL,
    
    -- 操作
    type            TEXT NOT NULL,               -- 'kick', 'mute', 'delete_message', 'pin', etc.
    operator_id     TEXT NOT NULL,               -- 操作者
    target_id       TEXT,                        -- 目标对象（用户/消息 ID）
    
    -- 详情
    reason          TEXT,
    details         TEXT,                        -- JSON 格式详细信息
    
    -- 时间
    timestamp       INTEGER NOT NULL,
    
    -- IP 追踪
    ip_address      TEXT,
    user_agent      TEXT,
    
    FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE
);

CREATE INDEX idx_audit_logs_channel ON audit_logs(channel_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_operator ON audit_logs(operator_id);
CREATE INDEX idx_audit_logs_type ON audit_logs(type);
```

---

### 2.7 禁言记录表 (mute_records)

```sql
CREATE TABLE mute_records (
    id              TEXT PRIMARY KEY,           -- UUID
    channel_id      TEXT NOT NULL,
    member_id       TEXT NOT NULL,
    
    -- 操作
    muted_by        TEXT NOT NULL,
    reason          TEXT,
    
    -- 时间
    muted_at        INTEGER NOT NULL,
    duration        INTEGER,                     -- 秒数，NULL=永久
    expires_at      INTEGER,                     -- 过期时间
    
    -- 状态
    active          INTEGER DEFAULT 1,           -- 0=已解除, 1=生效中
    unmuted_at      INTEGER,
    unmuted_by      TEXT,
    
    FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    FOREIGN KEY(member_id) REFERENCES members(id) ON DELETE CASCADE
);

CREATE INDEX idx_mute_records_member ON mute_records(member_id);
CREATE INDEX idx_mute_records_active ON mute_records(active) WHERE active = 1;
CREATE INDEX idx_mute_records_expires ON mute_records(expires_at) WHERE expires_at IS NOT NULL;
```

---

### 2.8 置顶消息表 (pinned_messages)

```sql
CREATE TABLE pinned_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id      TEXT NOT NULL,
    message_id      TEXT NOT NULL,
    
    -- 操作
    pinned_by       TEXT NOT NULL,
    reason          TEXT,
    
    -- 时间
    pinned_at       INTEGER NOT NULL,
    
    -- 顺序
    display_order   INTEGER DEFAULT 0,           -- 置顶顺序
    
    FOREIGN KEY(channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    FOREIGN KEY(message_id) REFERENCES messages(id) ON DELETE CASCADE,
    UNIQUE(channel_id, message_id)
);

CREATE INDEX idx_pinned_messages_channel ON pinned_messages(channel_id, display_order);
```

---

### 2.9 用户配置表 (user_config)

**独立数据库文件：`user.db`**

```sql
CREATE TABLE user_profiles (
    id              TEXT PRIMARY KEY,           -- UUID
    nickname        TEXT NOT NULL,
    avatar          TEXT,
    
    -- 密钥对
    private_key     BLOB NOT NULL,              -- RSA 私钥（加密存储）
    public_key      BLOB NOT NULL,              -- RSA 公钥
    
    -- CTF 信息
    skills          TEXT,                        -- JSON
    expertise       TEXT,                        -- JSON
    bio             TEXT,
    
    -- 设置
    theme           TEXT DEFAULT 'dark',
    language        TEXT DEFAULT 'zh-CN',
    auto_start      INTEGER DEFAULT 0,
    
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL
);

CREATE TABLE recent_channels (
    channel_id      TEXT PRIMARY KEY,
    channel_name    TEXT NOT NULL,
    server_address  TEXT,
    transport_mode  TEXT,
    last_joined     INTEGER NOT NULL,
    pinned          INTEGER DEFAULT 0
);

CREATE INDEX idx_recent_channels_last_joined ON recent_channels(last_joined DESC);
```

---

### 2.10 本地缓存表 (cache.db)

```sql
CREATE TABLE cache_entries (
    key             TEXT PRIMARY KEY,
    value           BLOB NOT NULL,
    expires_at      INTEGER NOT NULL,
    created_at      INTEGER NOT NULL
);

CREATE INDEX idx_cache_expires ON cache_entries(expires_at);

-- 定期清理过期缓存
CREATE TRIGGER cache_cleanup AFTER INSERT ON cache_entries BEGIN
    DELETE FROM cache_entries WHERE expires_at < unixepoch('now');
END;
```

---

## 3. 数据结构定义

### 3.1 Go 数据结构

#### 3.1.1 Channel 结构

```go
package models

type Channel struct {
    ID              string                 `json:"id" db:"id"`
    Name            string                 `json:"name" db:"name"`
    PasswordHash    string                 `json:"-" db:"password_hash"`
    Salt            []byte                 `json:"-" db:"salt"`
    CreatedAt       time.Time              `json:"created_at" db:"created_at"`
    CreatorID       string                 `json:"creator_id" db:"creator_id"`
    
    // 配置
    MaxMembers      int                    `json:"max_members" db:"max_members"`
    TransportMode   TransportMode          `json:"transport_mode" db:"transport_mode"`
    Port            int                    `json:"port,omitempty" db:"port"`
    Interface       string                 `json:"interface,omitempty" db:"interface"`
    
    // 加密
    EncryptionKey   []byte                 `json:"-" db:"encryption_key"`
    KeyVersion      int                    `json:"key_version" db:"key_version"`
    
    // 统计
    MessageCount    int64                  `json:"message_count" db:"message_count"`
    FileCount       int64                  `json:"file_count" db:"file_count"`
    TotalTraffic    uint64                 `json:"total_traffic" db:"total_traffic"`
    
    // 元数据
    Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
    UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
    
    // 运行时字段（不存数据库）
    Members         map[string]*Member     `json:"members,omitempty" db:"-"`
    OnlineCount     int                    `json:"online_count" db:"-"`
    PinnedMessages  []*PinnedMessage       `json:"pinned_messages,omitempty" db:"-"`
}

type TransportMode string

const (
    TransportARP   TransportMode = "arp"
    TransportHTTPS TransportMode = "https"
    TransportMDNS  TransportMode = "mdns"
    TransportAuto  TransportMode = "auto"
)
```

---

#### 3.1.2 Member 结构

```go
type Member struct {
    ID              string                 `json:"id" db:"id"`
    ChannelID       string                 `json:"channel_id" db:"channel_id"`
    Nickname        string                 `json:"nickname" db:"nickname"`
    Avatar          string                 `json:"avatar,omitempty" db:"avatar"`
    Role            Role                   `json:"role" db:"role"`
    Status          UserStatus             `json:"status" db:"status"`
    
    // 认证
    PublicKey       []byte                 `json:"-" db:"public_key"`
    LastIP          string                 `json:"last_ip,omitempty" db:"last_ip"`
    LastMAC         string                 `json:"last_mac,omitempty" db:"last_mac"`
    
    // CTF 相关
    Skills          []SkillTag             `json:"skills" db:"skills"`
    Expertise       []Expertise            `json:"expertise" db:"expertise"`
    CurrentTask     *CurrentTask           `json:"current_task,omitempty" db:"current_task"`
    
    // 统计
    MessageCount    int                    `json:"message_count" db:"message_count"`
    FilesShared     int                    `json:"files_shared" db:"files_shared"`
    OnlineTime      time.Duration          `json:"online_time" db:"online_time"`
    
    // 时间
    JoinedAt        time.Time              `json:"joined_at" db:"joined_at"`
    LastSeen        time.Time              `json:"last_seen" db:"last_seen"`
    LastHeartbeat   time.Time              `json:"last_heartbeat" db:"last_heartbeat"`
    
    // 元数据
    Metadata        map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

type Role string

const (
    RoleOwner    Role = "owner"
    RoleAdmin    Role = "admin"
    RoleMember   Role = "member"
    RoleReadOnly Role = "readonly"
)

type UserStatus string

const (
    StatusOnline  UserStatus = "online"
    StatusBusy    UserStatus = "busy"
    StatusAway    UserStatus = "away"
    StatusOffline UserStatus = "offline"
)

type SkillTag struct {
    Category   string    `json:"category"`   // "Web", "Pwn", "Reverse", "Crypto", "Misc"
    Level      int       `json:"level"`      // 1-5
    Experience int       `json:"experience"` // 题目数量
    LastUsed   time.Time `json:"last_used"`
}

type Expertise struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Tools       []string `json:"tools"`
    Notes       string   `json:"notes"`
}

type CurrentTask struct {
    Challenge   string    `json:"challenge"`
    StartTime   time.Time `json:"start_time"`
    Progress    int       `json:"progress"` // 0-100
    Notes       string    `json:"notes"`
    Teammates   []string  `json:"teammates"`
}
```

---

#### 3.1.3 Message 结构

```go
type Message struct {
    ID              string                 `json:"id" db:"id"`
    ChannelID       string                 `json:"channel_id" db:"channel_id"`
    SenderID        string                 `json:"sender_id" db:"sender_id"`
    SenderNickname  string                 `json:"sender_nickname" db:"sender_nickname"`
    
    // 内容
    Type            MessageType            `json:"type" db:"type"`
    Content         interface{}            `json:"content" db:"content"` // JSON
    ContentText     string                 `json:"content_text,omitempty" db:"content_text"`
    
    // 关系
    ReplyToID       string                 `json:"reply_to_id,omitempty" db:"reply_to_id"`
    ThreadID        string                 `json:"thread_id,omitempty" db:"thread_id"`
    
    // 标签
    Mentions        []string               `json:"mentions,omitempty" db:"mentions"`
    Tags            []string               `json:"tags,omitempty" db:"tags"`
    
    // 状态
    Pinned          bool                   `json:"pinned" db:"pinned"`
    Deleted         bool                   `json:"deleted" db:"deleted"`
    DeletedBy       string                 `json:"deleted_by,omitempty" db:"deleted_by"`
    DeletedAt       *time.Time             `json:"deleted_at,omitempty" db:"deleted_at"`
    
    // 时间
    Timestamp       time.Time              `json:"timestamp" db:"timestamp"`
    EditedAt        *time.Time             `json:"edited_at,omitempty" db:"edited_at"`
    
    // 加密
    Encrypted       bool                   `json:"encrypted" db:"encrypted"`
    KeyVersion      int                    `json:"key_version" db:"key_version"`
    
    // 元数据
    Metadata        map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

type MessageType string

const (
    MessageTypeText    MessageType = "text"
    MessageTypeCode    MessageType = "code"
    MessageTypeFile    MessageType = "file"
    MessageTypeSystem  MessageType = "system"
    MessageTypeControl MessageType = "control"
)

// 文本消息内容
type TextContent struct {
    Text    string   `json:"text"`
    Format  string   `json:"format"` // "plain", "markdown", "html"
    Mentions []string `json:"mentions,omitempty"`
    Tags    []string `json:"tags,omitempty"`
    ReplyTo string   `json:"reply_to,omitempty"`
}

// 代码消息内容
type CodeContent struct {
    Language    string `json:"language"`
    Code        string `json:"code"`
    Filename    string `json:"filename,omitempty"`
    Description string `json:"description,omitempty"`
    Highlighted bool   `json:"highlighted"`
}

// 文件消息内容
type FileContent struct {
    FileID    string `json:"file_id"`
    Filename  string `json:"filename"`
    Size      int64  `json:"size"`
    MimeType  string `json:"mime_type"`
    SHA256    string `json:"sha256"`
    Thumbnail string `json:"thumbnail,omitempty"` // Base64
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// 系统消息内容
type SystemContent struct {
    Event    string                 `json:"event"`
    ActorID  string                 `json:"actor_id"`
    TargetID string                 `json:"target_id,omitempty"`
    Reason   string                 `json:"reason,omitempty"`
    Extra    map[string]interface{} `json:"extra,omitempty"`
}
```

---

#### 3.1.4 File 结构

```go
type File struct {
    ID              string                 `json:"id" db:"id"`
    MessageID       string                 `json:"message_id" db:"message_id"`
    ChannelID       string                 `json:"channel_id" db:"channel_id"`
    SenderID        string                 `json:"sender_id" db:"sender_id"`
    
    // 文件信息
    Filename        string                 `json:"filename" db:"filename"`
    OriginalName    string                 `json:"original_name" db:"original_name"`
    Size            int64                  `json:"size" db:"size"`
    MimeType        string                 `json:"mime_type" db:"mime_type"`
    
    // 存储
    StorageType     StorageType            `json:"storage_type" db:"storage_type"`
    StoragePath     string                 `json:"storage_path,omitempty" db:"storage_path"`
    Data            []byte                 `json:"-" db:"data"`
    
    // 校验
    SHA256          string                 `json:"sha256" db:"sha256"`
    Checksum        string                 `json:"checksum" db:"checksum"`
    
    // 传输
    ChunkSize       int                    `json:"chunk_size" db:"chunk_size"`
    TotalChunks     int                    `json:"total_chunks" db:"total_chunks"`
    UploadedChunks  int                    `json:"uploaded_chunks" db:"uploaded_chunks"`
    UploadStatus    UploadStatus           `json:"upload_status" db:"upload_status"`
    
    // 预览
    Thumbnail       []byte                 `json:"thumbnail,omitempty" db:"thumbnail"`
    PreviewText     string                 `json:"preview_text,omitempty" db:"preview_text"`
    
    // 时间
    UploadedAt      time.Time              `json:"uploaded_at" db:"uploaded_at"`
    ExpiresAt       *time.Time             `json:"expires_at,omitempty" db:"expires_at"`
    
    // 加密
    Encrypted       bool                   `json:"encrypted" db:"encrypted"`
    EncryptionKey   []byte                 `json:"-" db:"encryption_key"`
    
    // 元数据
    Metadata        map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

type StorageType string

const (
    StorageInline    StorageType = "inline"    // 数据库中
    StorageFile      StorageType = "file"      // 文件系统
    StorageReference StorageType = "reference" // 外部引用
)

type UploadStatus string

const (
    UploadStatusPending    UploadStatus = "pending"
    UploadStatusUploading  UploadStatus = "uploading"
    UploadStatusCompleted  UploadStatus = "completed"
    UploadStatusFailed     UploadStatus = "failed"
)
```

---

### 3.2 JSON 序列化

#### 3.2.1 自定义 JSON 编码

```go
// Skills JSON 编码
func (s SkillTag) MarshalJSON() ([]byte, error) {
    return json.Marshal(map[string]interface{}{
        "category":   s.Category,
        "level":      s.Level,
        "experience": s.Experience,
        "last_used":  s.LastUsed.Unix(),
    })
}

func (s *SkillTag) UnmarshalJSON(data []byte) error {
    var temp map[string]interface{}
    if err := json.Unmarshal(data, &temp); err != nil {
        return err
    }
    
    s.Category = temp["category"].(string)
    s.Level = int(temp["level"].(float64))
    s.Experience = int(temp["experience"].(float64))
    s.LastUsed = time.Unix(int64(temp["last_used"].(float64)), 0)
    
    return nil
}

// Message Content 多态编码
func (m *Message) MarshalJSON() ([]byte, error) {
    type Alias Message
    
    // 根据类型编码 Content
    var contentJSON json.RawMessage
    switch m.Type {
    case MessageTypeText:
        contentJSON, _ = json.Marshal(m.Content.(*TextContent))
    case MessageTypeCode:
        contentJSON, _ = json.Marshal(m.Content.(*CodeContent))
    case MessageTypeFile:
        contentJSON, _ = json.Marshal(m.Content.(*FileContent))
    case MessageTypeSystem:
        contentJSON, _ = json.Marshal(m.Content.(*SystemContent))
    }
    
    return json.Marshal(&struct {
        *Alias
        Content json.RawMessage `json:"content"`
    }{
        Alias:   (*Alias)(m),
        Content: contentJSON,
    })
}
```

---

## 4. 索引策略

### 4.1 复合索引

```sql
-- 频道+时间范围查询（最常用）
CREATE INDEX idx_messages_channel_time ON messages(channel_id, timestamp DESC);

-- 发送者消息查询
CREATE INDEX idx_messages_sender_time ON messages(sender_id, timestamp DESC);

-- 回复链查询
CREATE INDEX idx_messages_thread ON messages(thread_id, timestamp ASC);

-- 标签查询（JSON 提取）
CREATE INDEX idx_messages_tags ON messages(json_extract(tags, '$[0]'));

-- 文件状态查询
CREATE INDEX idx_files_status ON files(channel_id, upload_status);
```

---

### 4.2 部分索引（节省空间）

```sql
-- 只索引未删除的消息
CREATE INDEX idx_messages_active ON messages(channel_id, timestamp DESC) 
WHERE deleted = 0;

-- 只索引置顶消息
CREATE INDEX idx_messages_pinned_only ON messages(channel_id, display_order) 
WHERE pinned = 1;

-- 只索引在线成员
CREATE INDEX idx_members_online ON members(channel_id, status) 
WHERE status != 'offline';

-- 只索引未过期的文件
CREATE INDEX idx_files_active ON files(channel_id, uploaded_at DESC) 
WHERE expires_at IS NULL OR expires_at > unixepoch('now');
```

---

### 4.3 全文搜索优化


---

## 5. 查询优化

### 5.1 常用查询

#### 5.1.1 获取最近消息（分页）

```sql
-- 使用 rowid 而不是 OFFSET（更快）
SELECT * FROM messages
WHERE channel_id = ? 
  AND deleted = 0
  AND rowid < ?  -- 上次查询的最小 rowid
ORDER BY timestamp DESC
LIMIT 50;
```

---

#### 5.1.2 获取成员列表（带在线状态）

```sql
SELECT 
    m.*,
    CASE 
        WHEN m.status = 'online' THEN 1
        WHEN m.status = 'busy' THEN 2
        WHEN m.status = 'away' THEN 3
        ELSE 4
    END as status_order
FROM members m
WHERE m.channel_id = ?
ORDER BY status_order ASC, m.joined_at ASC;
```

---

#### 5.1.3 全文搜索（多条件）

```sql
SELECT 
    m.*,
    snippet(messages_fts, 0, '<mark>', '</mark>', '...', 64) as snippet,
    bm25(messages_fts) as rank
FROM messages m
JOIN messages_fts ON m.rowid = messages_fts.rowid
WHERE m.channel_id = ?
  AND m.deleted = 0
  AND messages_fts MATCH ?  -- 搜索关键词
  AND m.timestamp BETWEEN ? AND ?  -- 时间范围
  AND (? IS NULL OR m.sender_id = ?)  -- 可选：发送者筛选
ORDER BY rank
LIMIT 50;
```

---

#### 5.1.4 统计查询（聚合）

```sql
-- 频道统计
SELECT 
    c.id,
    c.name,
    COUNT(DISTINCT m.id) as message_count,
    COUNT(DISTINCT mem.id) as member_count,
    SUM(CASE WHEN mem.status = 'online' THEN 1 ELSE 0 END) as online_count,
    COUNT(DISTINCT f.id) as file_count,
    SUM(f.size) as total_file_size
FROM channels c
LEFT JOIN messages m ON c.id = m.channel_id AND m.deleted = 0
LEFT JOIN members mem ON c.id = mem.channel_id
LEFT JOIN files f ON c.id = f.channel_id AND f.upload_status = 'completed'
WHERE c.id = ?
GROUP BY c.id;
```

---

### 5.2 批量操作优化

#### 5.2.1 批量插入消息

```go
func (db *Database) BatchInsertMessages(messages []*Message) error {
    tx, _ := db.Begin()
    defer tx.Rollback()
    
    stmt, _ := tx.Prepare(`
        INSERT INTO messages (
            id, channel_id, sender_id, sender_nickname,
            type, content, content_text, timestamp
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `)
    defer stmt.Close()
    
    for _, msg := range messages {
        contentJSON, _ := json.Marshal(msg.Content)
        _, err := stmt.Exec(
            msg.ID, msg.ChannelID, msg.SenderID, msg.SenderNickname,
            msg.Type, contentJSON, msg.ContentText, msg.Timestamp.UnixNano(),
        )
        if err != nil {
            return err
        }
    }
    
    return tx.Commit()
}
```

---

#### 5.2.2 批量更新成员状态

```sql
-- 使用 CASE WHEN 批量更新
UPDATE members
SET 
    status = CASE id
        WHEN 'user-1' THEN 'online'
        WHEN 'user-2' THEN 'busy'
        WHEN 'user-3' THEN 'away'
        ELSE status
    END,
    last_seen = unixepoch('now')
WHERE id IN ('user-1', 'user-2', 'user-3');
```

---

### 5.3 缓存策略

```go
type QueryCache struct {
    cache *lru.Cache
    ttl   time.Duration
}

type CacheEntry struct {
    Data      interface{}
    ExpiresAt time.Time
}

func (qc *QueryCache) Get(key string) (interface{}, bool) {
    if val, ok := qc.cache.Get(key); ok {
        entry := val.(*CacheEntry)
        if time.Now().Before(entry.ExpiresAt) {
            return entry.Data, true
        }
        qc.cache.Remove(key)
    }
    return nil, false
}

func (qc *QueryCache) Set(key string, data interface{}) {
    qc.cache.Add(key, &CacheEntry{
        Data:      data,
        ExpiresAt: time.Now().Add(qc.ttl),
    })
}

// 使用示例
var messageCache = &QueryCache{
    cache: lru.New(1000),
    ttl:   5 * time.Minute,
}

func GetMessage(id string) (*Message, error) {
    cacheKey := "message:" + id
    if cached, ok := messageCache.Get(cacheKey); ok {
        return cached.(*Message), nil
    }
    
    msg, err := db.QueryMessage(id)
    if err == nil {
        messageCache.Set(cacheKey, msg)
    }
    return msg, err
}
```

---

## 6. 数据迁移

### 6.1 版本管理

```sql
CREATE TABLE schema_version (
    version     INTEGER PRIMARY KEY,
    applied_at  INTEGER NOT NULL,
    description TEXT
);

INSERT INTO schema_version VALUES (1, unixepoch('now'), 'Initial schema');
```

---

### 6.2 迁移脚本

#### v1 → v2: 添加话题支持

```sql
-- migrations/002_add_threads.sql
BEGIN TRANSACTION;

-- 添加 thread_id 列
ALTER TABLE messages ADD COLUMN thread_id TEXT;

-- 创建索引
CREATE INDEX idx_messages_thread ON messages(thread_id, timestamp ASC);

-- 更新版本
INSERT INTO schema_version VALUES (2, unixepoch('now'), 'Add thread support');

COMMIT;
```

#### v2 → v3: 成员技能标签

```sql
-- migrations/003_member_skills.sql
BEGIN TRANSACTION;

-- 添加技能相关列
ALTER TABLE members ADD COLUMN skills TEXT;
ALTER TABLE members ADD COLUMN expertise TEXT;
ALTER TABLE members ADD COLUMN current_task TEXT;

-- 更新版本
INSERT INTO schema_version VALUES (3, unixepoch('now'), 'Add member CTF skills');

COMMIT;
```

---

### 6.3 自动迁移

```go
package database

import (
    "database/sql"
    "embed"
    "fmt"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (db *Database) Migrate() error {
    // 获取当前版本
    currentVersion := db.getCurrentVersion()
    
    // 读取所有迁移文件
    migrations, _ := migrationsFS.ReadDir("migrations")
    
    for _, migration := range migrations {
        version := extractVersionFromFilename(migration.Name())
        if version <= currentVersion {
            continue
        }
        
        // 执行迁移
        content, _ := migrationsFS.ReadFile("migrations/" + migration.Name())
        if err := db.executeMigration(string(content)); err != nil {
            return fmt.Errorf("migration %s failed: %w", migration.Name(), err)
        }
        
        log.Printf("Applied migration: %s", migration.Name())
    }
    
    return nil
}

func (db *Database) getCurrentVersion() int {
    var version int
    db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
    return version
}
```

---

## 7. CTF题目管理系统

### 7.1 功能概述

CrossWire 支持在频道内创建和管理 CTF 题目：

- ✅ **题目管理**：创建、编辑、分配 CTF 题目
- ✅ **独立聊天室**：每个题目自动创建独立聊天室
- ✅ **进度跟踪**：实时查看题目解答进度
- ✅ **Flag 提交**：验证和记录 Flag 提交
- ✅ **提示系统**：分阶段提供题目提示

### 7.2 新增表结构

#### 7.2.1 challenges（题目表）

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
    attachments     TEXT,                       -- JSON: ["file-id-1", "file-id-2"]
    tags            TEXT,                       -- JSON: ["sqli", "waf-bypass"]
    status          TEXT NOT NULL DEFAULT 'open',  -- open/solved/closed
    solved_by       TEXT,                       -- JSON: ["user-id-1"]
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

#### 7.2.2 challenge_assignments（题目分配表）

```sql
CREATE TABLE challenge_assignments (
    challenge_id    TEXT NOT NULL,
    member_id       TEXT NOT NULL,
    assigned_by     TEXT NOT NULL,
    assigned_at     INTEGER NOT NULL,
    role            TEXT NOT NULL DEFAULT 'member',  -- lead/member
    status          TEXT NOT NULL DEFAULT 'assigned', -- assigned/working/completed
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

#### 7.2.3 challenge_progress（进度跟踪表）

```sql
CREATE TABLE challenge_progress (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    challenge_id    TEXT NOT NULL,
    member_id       TEXT NOT NULL,
    progress        INTEGER NOT NULL DEFAULT 0,    -- 0-100
    status          TEXT NOT NULL DEFAULT 'not_started',
    summary         TEXT,
    findings        TEXT,                          -- JSON
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

#### 7.2.4 challenge_submissions（提交记录表）

```sql
CREATE TABLE challenge_submissions (
    id              TEXT PRIMARY KEY,
    challenge_id    TEXT NOT NULL,
    member_id       TEXT NOT NULL,
    flag            TEXT NOT NULL,                 -- 加密存储
    is_correct      INTEGER NOT NULL,
    submitted_at    INTEGER NOT NULL,
    ip_address      TEXT,
    response_time   INTEGER,
    metadata        TEXT,
    FOREIGN KEY(challenge_id) REFERENCES challenges(id) ON DELETE CASCADE,
    FOREIGN KEY(member_id) REFERENCES members(id) ON DELETE CASCADE,
    CHECK(is_correct IN (0, 1))
);

CREATE INDEX idx_submissions_challenge ON challenge_submissions(challenge_id);
CREATE INDEX idx_submissions_member ON challenge_submissions(member_id);
CREATE INDEX idx_submissions_time ON challenge_submissions(submitted_at DESC);
CREATE INDEX idx_submissions_correct ON challenge_submissions(challenge_id, is_correct) 
    WHERE is_correct = 1;
```

#### 7.2.5 challenge_hints（提示表）

```sql
CREATE TABLE challenge_hints (
    id              TEXT PRIMARY KEY,
    challenge_id    TEXT NOT NULL,
    order_num       INTEGER NOT NULL,
    content         TEXT NOT NULL,
    cost            INTEGER DEFAULT 0,
    unlocked_by     TEXT,                          -- JSON: ["user-id-1"]
    created_by      TEXT NOT NULL,
    created_at      INTEGER NOT NULL,
    FOREIGN KEY(challenge_id) REFERENCES challenges(id) ON DELETE CASCADE,
    FOREIGN KEY(created_by) REFERENCES members(id) ON DELETE SET NULL,
    UNIQUE(challenge_id, order_num)
);

CREATE INDEX idx_hints_challenge ON challenge_hints(challenge_id, order_num);
```

### 7.3 messages 表扩展

为支持题目聊天室，`messages` 表新增两个字段：

```sql
ALTER TABLE messages ADD COLUMN challenge_id TEXT;
ALTER TABLE messages ADD COLUMN room_type TEXT DEFAULT 'main';

-- 添加索引
CREATE INDEX idx_messages_challenge ON messages(challenge_id, timestamp DESC);
CREATE INDEX idx_messages_room_type ON messages(channel_id, room_type, timestamp DESC);

-- 添加外键约束（需要重建表）
-- FOREIGN KEY(challenge_id) REFERENCES challenges(id) ON DELETE CASCADE
-- CHECK(room_type IN ('main', 'challenge'))
```

**字段说明：**
- `room_type = 'main'`：主频道消息
- `room_type = 'challenge'`：题目聊天室消息
- `challenge_id`：题目聊天室时有值，主频道时为 NULL

### 7.4 Go 数据结构

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
    Flag          string    `json:"-" db:"flag"`
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

### 7.5 查询示例

#### 获取频道所有题目

```go
func (db *Database) GetChallenges(channelID string) ([]*Challenge, error) {
    query := `
        SELECT * FROM challenges 
        WHERE channel_id = ? 
        ORDER BY 
            CASE category 
                WHEN 'Web' THEN 1 
                WHEN 'Pwn' THEN 2 
                WHEN 'Reverse' THEN 3 
                WHEN 'Crypto' THEN 4 
                WHEN 'Misc' THEN 5 
                ELSE 6 
            END,
            points ASC
    `
    
    var challenges []*Challenge
    err := db.Select(&challenges, query, channelID)
    return challenges, err
}
```

#### 获取题目聊天室消息

```go
func (db *Database) GetChallengeMessages(challengeID string, limit int) ([]*Message, error) {
    query := `
        SELECT * FROM messages 
        WHERE challenge_id = ? 
          AND room_type = 'challenge'
          AND deleted = 0
        ORDER BY timestamp DESC 
        LIMIT ?
    `
    
    var messages []*Message
    err := db.Select(&messages, query, challengeID, limit)
    return messages, err
}
```

#### 获取成员题目进度

```go
func (db *Database) GetMemberProgress(challengeID, memberID string) (*ChallengeProgress, error) {
    query := `
        SELECT * FROM challenge_progress 
        WHERE challenge_id = ? 
          AND member_id = ?
        ORDER BY updated_at DESC 
        LIMIT 1
    `
    
    var progress ChallengeProgress
    err := db.Get(&progress, query, challengeID, memberID)
    return &progress, err
}
```

---

## 总结

CrossWire 数据库设计特点：

✅ **高性能**：合理的索引和查询优化  
✅ **安全存储**：加密敏感数据  
✅ **灵活扩展**：JSON 字段支持扩展  
✅ **事务完整性**：ACID 保证数据一致性  
✅ **版本迁移**：自动化数据库升级  

---

**相关文档：**
- [FEATURES.md](FEATURES.md) - 功能详细说明
- [PROTOCOL.md](PROTOCOL.md) - 通信协议规范
- [ARCHITECTURE.md](ARCHITECTURE.md) - 系统架构设计
