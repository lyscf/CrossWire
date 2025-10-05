# CrossWire 项目状态报告

> 更新时间: 2025-10-05
> 版本: v0.1.0-alpha

---

## 📊 项目概览

**CrossWire** 是专为 CTF 线下赛设计的团队协作通讯工具，基于 **Wails2 (Go + Vue 3)** 技术栈开发。

- **技术栈**: Go 1.23 + Wails v2.10.2 + Vue 3 + GORM
- **数据库**: SQLite 3 + WAL 模式
- **加密**: AES-256-GCM + Ed25519 + X25519
- **传输**: ARP / HTTPS / mDNS (待实现)

---

## ✅ 已完成模块

### 1. 数据模型层 (Models) - 100%

**位置**: `internal/models/`

已完成所有数据模型定义:
- ✅ `types.go` - 基础类型和枚举
- ✅ `channel.go` - 频道模型
- ✅ `member.go` - 成员和禁言记录
- ✅ `message.go` - 消息、反应、输入状态
- ✅ `file.go` - 文件和分块
- ✅ `challenge.go` - CTF题目系统
- ✅ `audit.go` - 审计日志和用户配置

**特性**:
- 完整的GORM标签
- JSON序列化支持
- 外键关联
- GORM钩子函数
- 自定义类型(JSONField, StringArray等)

---

### 2. 数据存储层 (Storage) - 95%

**位置**: `internal/storage/`

已完成:
- ✅ `database.go` - 数据库管理器
  - SQLite配置优化(WAL模式, 缓存, mmap)
  - 多数据库支持(user.db, cache.db, channel/*.db)
  - 自动迁移

- ✅ `channel_repository.go` - 频道仓库
  - 基础CRUD
  - 统计信息
  - 置顶消息管理

- ✅ `member_repository.go` - 成员仓库
  - 成员管理
  - 状态更新
  - 禁言功能
  - 技能标签管理

- ✅ `message_repository.go` - 消息仓库
  - 消息CRUD
  - 分页查询
  - 表情回应
  - 输入状态

- ✅ `file_repository.go` - 文件仓库
  - 文件记录管理
  - 分块上传
  - 过期清理

- ✅ `challenge_repository.go` - 题目仓库
  - 题目管理
  - Flag验证(SHA256+盐值)
  - 分配和进度跟踪
  - 提交记录
  - 提示系统

- ✅ `audit_repository.go` - 审计日志仓库
  - 日志记录
  - 按类型/操作者查询
  - 自动清理

**待完成**:
- ⏳ FTS5全文搜索实现
- ⏳ 批量操作优化
- ⏳ 缓存策略

---

### 3. 加密模块 (Crypto) - 100%

**位置**: `internal/crypto/crypto.go`

已实现:
- ✅ AES-256-GCM加密/解密
- ✅ X25519密钥交换(ECDH)
- ✅ Ed25519数字签名
- ✅ Argon2id密钥派生
- ✅ 密码哈希和验证
- ✅ SHA256哈希
- ✅ 随机数生成

**特性**:
- 恒定时间比较(防时序攻击)
- 安全的随机数生成
- 标准化的密钥长度检查

**待完成**:
- ⏳ 密钥轮换
- ⏳ 密钥缓存管理
- ⏳ 文件分块加密

---

### 4. 工具模块 (Utils) - 90%

**位置**: `internal/utils/`

已完成:
- ✅ `logger.go` - 日志系统
  - 多级别日志(DEBUG/INFO/WARN/ERROR/FATAL)
  - 文件+控制台输出
  - 日期分割

- ✅ `validator.go` - 验证器
  - 昵称/频道名验证
  - 密码强度检查
  - IP/MAC地址验证
  - 题目参数验证
  - 角色/状态验证

**待完成**:
- ⏳ 日志轮转
- ⏳ 日志压缩
- ⏳ 结构化日志(JSON)

---

### 5. 应用层 (App) - 30%

**位置**: `internal/app/app.go`

已完成:
- ✅ 应用结构定义
- ✅ 生命周期管理(Startup/DomReady/Shutdown)
- ✅ 数据库初始化

**待完成 (有TODO注释)**:
- ⏳ 模式切换(服务端/客户端)
- ⏳ 频道管理API
- ⏳ 消息操作API
- ⏳ 文件操作API
- ⏳ 成员管理API
- ⏳ 题目管理API
- ⏳ 用户配置API

---

### 6. 主程序 (Main) - 80%

**位置**: `app.go`, `main.go`, `cmd/crosswire/main.go`

已完成:
- ✅ Wails桥接层
- ✅ 应用启动流程
- ✅ 数据目录初始化
- ✅ 生命周期钩子

---

## 🚧 待实现模块

### 1. 传输层 (Transport) - 0%

**计划位置**: `internal/transport/`

需要实现:
- ⏳ `transport.go` - 统一接口
- ⏳ `arp_transport.go` - ARP传输实现
- ⏳ `https_transport.go` - HTTPS/WebSocket
- ⏳ `mdns_transport.go` - mDNS传输
- ⏳ `factory.go` - 传输层工厂

---

### 2. 服务端 (Server) - 0%

**计划位置**: `internal/server/`

需要实现:
- ⏳ `server.go` - 服务端核心
- ⏳ `channel.go` - 频道管理
- ⏳ `broadcast.go` - 消息广播
- ⏳ `auth.go` - 认证授权

---

### 3. 客户端 (Client) - 0%

**计划位置**: `internal/client/`

需要实现:
- ⏳ `client.go` - 客户端核心
- ⏳ `connection.go` - 连接管理
- ⏳ `sync.go` - 消息同步
- ⏳ `cache.go` - 缓存管理

---

### 4. 事件总线 (EventBus) - 0%

**计划位置**: `internal/events/`

需要实现:
- ⏳ 事件订阅/发布
- ⏳ 事件类型定义
- ⏳ 异步事件处理

---

### 5. 前端应用 (Frontend) - 0%

**计划位置**: `frontend/`

需要实现:
- ⏳ Vue 3组件
- ⏳ Pinia状态管理
- ⏳ Naive UI集成
- ⏳ Wails API调用

---

## 📝 项目结构

```
CrossWire/
├── app.go                      # Wails桥接层 ✅
├── main.go                     # 主程序入口 ✅
├── go.mod                      # Go模块配置 ✅
├── go.sum                      # 依赖锁定 ✅
├── wails.json                  # Wails配置 ✅
├── Makefile                    # 构建脚本 ✅
├── PROJECT_STATUS.md           # 本文档 ✅
│
├── cmd/
│   └── crosswire/
│       └── main.go             # CLI入口(待定) ⏳
│
├── internal/
│   ├── app/
│   │   └── app.go              # 应用核心 ✅ (30%)
│   │
│   ├── models/                 # 数据模型 ✅ (100%)
│   │   ├── types.go            # ✅
│   │   ├── channel.go          # ✅
│   │   ├── member.go           # ✅
│   │   ├── message.go          # ✅
│   │   ├── file.go             # ✅
│   │   ├── challenge.go        # ✅
│   │   └── audit.go            # ✅
│   │
│   ├── storage/                # 存储层 ✅ (95%)
│   │   ├── database.go         # ✅
│   │   ├── channel_repository.go   # ✅
│   │   ├── member_repository.go    # ✅
│   │   ├── message_repository.go   # ✅
│   │   ├── file_repository.go      # ✅
│   │   ├── challenge_repository.go # ✅
│   │   └── audit_repository.go     # ✅
│   │
│   ├── crypto/                 # 加密模块 ✅ (100%)
│   │   └── crypto.go           # ✅
│   │
│   ├── utils/                  # 工具模块 ✅ (90%)
│   │   ├── logger.go           # ✅
│   │   └── validator.go        # ✅
│   │
│   ├── transport/              # 传输层 ⏳ (0%)
│   ├── server/                 # 服务端 ⏳ (0%)
│   ├── client/                 # 客户端 ⏳ (0%)
│   └── events/                 # 事件总线 ⏳ (0%)
│
├── frontend/                   # Vue前端 ⏳ (0%)
│
├── build/                      # 构建资源 ✅
│   ├── windows/
│   ├── darwin/
│   └── README.md
│
└── docs/                       # 文档 ✅
    ├── README.md
    ├── ARCHITECTURE.md
    ├── PROTOCOL.md
    ├── DATABASE.md
    ├── FEATURES.md
    ├── CHALLENGE_SYSTEM.md
    ├── ARP_BROADCAST_MODE.md
    └── CHANGELOG.md
```

---

## 🔧 技术栈

### 后端
- **语言**: Go 1.23
- **框架**: Wails v2.10.2
- **ORM**: GORM v1.31.0
- **数据库**: SQLite 3
- **加密**: golang.org/x/crypto

### 前端 (待实现)
- **框架**: Vue 3
- **状态管理**: Vite
- **UI库**: Ant Design Vue
- **构建工具**: Vite

---

## 📦 依赖包

```go
require (
	github.com/wailsapp/wails/v2 v2.10.2
	gorm.io/gorm v1.31.0
	gorm.io/driver/sqlite v1.6.0
	golang.org/x/crypto v0.33.0
)
```

---

## 🚀 快速开始

### 编译项目

```bash
# 开发模式
wails dev

# 生产构建
go build -o crosswire.exe .

# 使用Wails构建
wails build
```

### 项目初始化

```bash
# 下载依赖
go mod tidy

# 编译测试
go build .
```

---

## ✨ 核心特性

### 已实现
✅ 完整的数据模型定义  
✅ GORM ORM集成  
✅ SQLite优化配置  
✅ 多Repository模式  
✅ AES-256-GCM加密  
✅ Ed25519数字签名  
✅ X25519密钥交换  
✅ Argon2id密钥派生  
✅ CTF题目管理系统  
✅ Flag安全验证(SHA256)  
✅ 题目聊天室隔离  
✅ 审计日志系统  
✅ 多级别日志  
✅ 数据验证器  

### 待实现
⏳ ARP/HTTPS/mDNS传输  
⏳ 服务端/客户端逻辑  
⏳ 消息广播系统  
⏳ 事件驱动架构  
⏳ Vue 3前端界面  
⏳ FTS5全文搜索  
⏳ 文件分块传输  
⏳ 实时在线状态  
⏳ P2P发现机制  

---

## 📋 TODO 列表

### 高优先级
- [ ] 实现Transport传输层接口
- [ ] 实现Server服务端核心逻辑
- [ ] 实现Client客户端核心逻辑
- [ ] 实现EventBus事件总线
- [ ] 完善App层业务方法

### 中优先级
- [ ] 实现SQLite FTS5全文搜索
- [ ] 实现前端Vue应用基础框架
- [ ] 编写单元测试
- [ ] 实现文件分块传输
- [ ] 实现消息同步机制

### 低优先级
- [ ] 日志轮转和压缩
- [ ] 性能优化
- [ ] 文档完善
- [ ] 集成测试

---

## 🎯 下一步计划

1. **实现Transport传输层** (优先级: 最高)
   - 定义统一接口
   - 先实现HTTPS/WebSocket模式
   - 后续实现ARP和mDNS

2. **实现EventBus事件总线**
   - 订阅/发布机制
   - 异步事件处理
   - 与前端的事件桥接

3. **实现Server服务端逻辑**
   - 频道管理
   - 成员认证
   - 消息广播

4. **实现Client客户端逻辑**
   - 连接管理
   - 消息同步
   - 本地缓存

5. **完善App层API**
   - 导出给前端的完整API
   - 错误处理
   - 事件通知

6. **开始Vue前端开发**
   - 基础框架搭建
   - 核心组件开发
   - Wails API集成

---

## 📈 完成度

| 模块 | 完成度 | 状态 |
|-----|--------|------|
| 数据模型 (Models) | 100% | ✅ 已完成 |
| 存储层 (Storage) | 95% | ✅ 基本完成 |
| 加密模块 (Crypto) | 100% | ✅ 已完成 |
| 工具模块 (Utils) | 90% | ✅ 基本完成 |
| 应用层 (App) | 30% | 🚧 进行中 |
| 传输层 (Transport) | 0% | ⏳ 未开始 |
| 服务端 (Server) | 0% | ⏳ 未开始 |
| 客户端 (Client) | 0% | ⏳ 未开始 |
| 事件总线 (EventBus) | 0% | ⏳ 未开始 |
| 前端 (Frontend) | 0% | ⏳ 未开始 |
| **总体进度** | **≈35%** | 🚧 早期开发 |

---

## 🐛 已知问题

暂无

---

## 📝 注释说明

代码中使用TODO注释标注了待实现的功能:

```go
// TODO: 实现功能描述
// TODO: 优化建议
// TODO: 待完成的方法列表
```

搜索项目中的TODO可以快速定位需要继续开发的部分。

---

## 👥 开发建议

遵循"Cursor八荣八耻"编程准则:
1. ✅ 认真查询，拒绝瞎猜
2. ✅ 寻求确认，避免模糊执行
3. ✅ 人类确认，拒绝臆想业务
4. ✅ 复用现有，避免重复创造
5. ✅ 主动测试，确保质量
6. ✅ 遵循规范，维护架构
7. ✅ 诚实无知，拒绝假装
8. ✅ 谨慎重构，避免盲目修改

---

**项目仓库**: 待创建  
**文档**: `docs/` 目录  
**联系方式**: 待定

**最后更新**: 2025-10-05  
**维护者**: CrossWire Team

