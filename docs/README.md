# CrossWire 文档中心

> CTF 线下赛团队通讯系统 - 完整技术文档

---

## 📚 文档导航

### 核心文档

| 文档 | 说明 | 适合人群 |
|------|------|----------|
| [功能规格 (FEATURES.md)](FEATURES.md) | 详细的功能说明和使用指南 | 用户、产品经理、测试 |
| [系统架构 (ARCHITECTURE.md)](ARCHITECTURE.md) | 系统整体架构和模块设计 | 架构师、开发者 |
| [通信协议 (PROTOCOL.md)](PROTOCOL.md) | ARP/HTTPS/mDNS 协议详细规范 | 后端开发者、网络工程师 |
| [数据库设计 (DATABASE.md)](DATABASE.md) | 数据库结构和数据模型 | 后端开发者、DBA |
| **[题目管理系统 (CHALLENGE_SYSTEM.md)](CHALLENGE_SYSTEM.md)** | **CTF题目管理、分配、聊天室设计** | **后端开发者、产品经理** |

### 辅助文档

| 文档 | 说明 | 适合人群 |
|------|------|----------|
| [数据库表清单 (DATABASE_TABLES.md)](DATABASE_TABLES.md) | 所有数据库表的详细字段说明 | 开发者、DBA |
| **[ARP 广播模式 (ARP_BROADCAST_MODE.md)](ARP_BROADCAST_MODE.md)** | **ARP 传输层广播模式设计详解** | **后端开发者** |
| [更新日志 (CHANGELOG.md)](CHANGELOG.md) | 版本更新记录 | 所有人 |

---

## 🚀 快速开始

### 1. 了解项目

**CrossWire** 是专为 CTF 线下赛设计的团队协作通讯工具，支持：
- 🔀 ARP/HTTPS/mDNS 三种传输模式
- 🔐 端到端加密通信
- 👥 CTF 专属成员管理（技能标签、擅长领域）
- 📁 高效文件传输
- 💬 代码分享与高亮

**技术栈：** Golang + Wails2 + Vue 3

---

### 2. 阅读路线

#### 🎯 产品经理/用户
```
1. README.md (本文档)
   ↓
2. FEATURES.md (功能详解)
   ├─ 核心功能
   ├─ 使用流程
   └─ 界面设计
```

#### 🏗️ 架构师
```
1. README.md (本文档)
   ↓
2. ARCHITECTURE.md (系统架构)
   ├─ 分层架构
   ├─ 模块设计
   └─ 技术栈选型
   ↓
3. PROTOCOL.md (通信协议)
   └─ 传输层设计
```

#### 💻 前端开发者
```
1. ARCHITECTURE.md
   ├─ 前端架构
   └─ 目录结构
   ↓
2. FEATURES.md
   └─ 界面功能
```

#### 💻 后端开发者
```
1. ARCHITECTURE.md
   ├─ 后端架构
   └─ 模块设计
   ↓
2. PROTOCOL.md
   ├─ ARP 传输协议
   ├─ HTTPS 传输协议
   └─ mDNS 传输协议
   ↓
3. DATABASE.md
   ├─ Schema 设计
   ├─ 数据结构
   └─ 查询优化
```

---

## 📖 文档概览

### [FEATURES.md](FEATURES.md) - 功能规格文档

**内容目录：**
1. 核心功能
   - 双模式运行（服务端/客户端）
   - 传输模式选择（ARP/HTTPS/mDNS）
2. 用户功能
   - 注册与认证
   - 个人资料管理
   - 在线状态
3. 频道功能
   - 频道创建
   - 加入验证
   - 权限管理
4. 消息功能
   - 消息类型（文本/代码/文件/系统）
   - 发送与接收
   - 同步与历史
   - 消息交互（回复/置顶/删除）
5. 文件传输功能
   - 分块上传
   - 断点续传
   - 预览与缩略图
6. 成员管理功能
   - 成员列表
   - 踢出/禁言
   - 技能标签管理
7. 搜索与过滤功能
8. 界面功能

**适合人群：** 所有人

---

### [ARCHITECTURE.md](ARCHITECTURE.md) - 系统架构文档

**内容目录：**
1. 架构概述
   - 系统定位
   - 架构风格
2. 分层架构
   - 表示层（Vue）
   - 应用层（Go Backend）
   - 领域层（Business Logic）
   - 基础设施层（Transport/Storage）
3. 模块设计
   - 核心模块（App/Server/Client/Transport）
   - 辅助模块（Discovery/EventBus/Logger）
4. 部署架构
   - 单机部署
   - 网络拓扑
   - 数据流
5. 技术栈
6. 目录结构

**适合人群：** 架构师、开发者

---

### [PROTOCOL.md](PROTOCOL.md) - 通信协议文档

**内容目录：**
1. 协议概述
   - 协议栈架构
   - 协议特点
2. ARP 传输协议
   - 以太网帧结构
   - 通信流程（发现/认证/数据传输）
   - 可靠性保证（ACK/重传/流控）
3. HTTPS 传输协议
   - WebSocket 子协议
   - HTTP API（文件上传/下载/同步）
   - TLS 配置
4. mDNS 传输协议
   - 服务发现协议
   - 数据传输协议（服务实例名编码）
   - 性能优化
5. 应用层协议
   - 消息格式
   - 文件传输协议
   - 同步协议
6. 加密与安全
   - 密钥体系
   - 加密算法
   - 安全特性
7. 协议扩展

**适合人群：** 后端开发者、网络工程师

---

### [DATABASE.md](DATABASE.md) - 数据库设计文档

**内容目录：**
1. 数据库概述
   - 技术选型（SQLite）
   - 配置
   - 文件结构
2. 数据库 Schema
   - channels（频道表）
   - members（成员表）
   - messages（消息表）
   - messages_fts（全文搜索表）
   - files（文件表）
   - audit_logs（审计日志表）
   - mute_records（禁言记录表）
   - pinned_messages（置顶消息表）
   - user_config（用户配置表）
   - cache（本地缓存表）
3. 数据结构定义
   - Go 数据结构
   - JSON 序列化
4. 索引策略
   - 复合索引
   - 部分索引
   - 全文搜索优化
5. 查询优化
   - 常用查询
   - 批量操作
   - 缓存策略
6. 数据迁移
   - 版本管理
   - 迁移脚本
   - 自动迁移

**适合人群：** 后端开发者、DBA

---

### [CHALLENGE_SYSTEM.md](CHALLENGE_SYSTEM.md) - 题目管理系统文档 

**内容目录：**
1. 系统概述
   - 核心功能
   - 架构图
2. 数据库设计
   - challenges（题目表）
   - challenge_assignments（题目分配表）
   - challenge_progress（题目进度表）
   - challenge_submissions（提交记录表）
   - challenge_hints（题目提示表）
   - messages 表扩展（支持题目聊天室）
3. 功能设计
   - 题目创建/分配/提交
   - 进度跟踪
4. 聊天室设计
   - 主频道 vs 题目聊天室
   - 消息隔离
   - 权限控制
5. UI设计
   - 侧边栏布局
   - 题目详情页
   - 题目聊天室界面
6. API设计
   - 题目管理 API
   - 聊天室 API
   - Go 数据结构

**核心特性：**
- ✅ 管理员创建和分配CTF题目
- ✅ 每个题目自动创建独立聊天室
- ✅ 实时进度跟踪和Flag提交验证
- ✅ 分阶段提示系统
- ✅ 题目聊天室权限隔离

**适合人群：** 后端开发者、产品经理、前端开发者

---

## 🔍 常见问题

### Q: 为什么选择 ARP 作为传输方式？
**A:** ARP（实际上是原始以太网帧）工作在 OSI 第2层，采用服务器签名的广播模式，具有以下优势：
- ⚡ **极低延迟**：1-3ms，比 TCP/IP 快数倍
- 🔒 **隐蔽性高**：二层通信，难以被应用层防火墙检测
- 🚀 **高吞吐量**：50-100 MB/s 文件传输速度
- 🛡️ **安全可靠**：服务器 Ed25519 签名，防伪造和篡改
- 🎯 **实现优雅**：星型拓扑，客户端仅信任服务器

详见 [PROTOCOL.md - ARP 传输协议](PROTOCOL.md#2-arp-传输协议)

---

### Q: HTTPS 和 mDNS 模式分别适用于什么场景？
**A:** 
- **HTTPS 模式**：标准网络环境，跨网段通信，无需特殊权限，采用 TLS 1.3 加密
- **mDNS 模式**：极端受限网络（只有 UDP 5353 可通过），作为最后的 fallback
  - 采用服务器签名模式（类似 ARP）
  - 客户端 UDP 单播 → 服务器验证签名 → mDNS 组播
  - Ed25519 签名防止消息伪造

详见 [PROTOCOL.md - mDNS 传输协议](PROTOCOL.md#4-mdns-传输协议)

---

### Q: 数据是如何加密的？
**A:** CrossWire 采用**多层加密**：
1. **传输层**：TLS 1.3（HTTPS 模式）
2. **应用层**：X25519 端到端加密
3. **密钥交换**：X25519

详见 [PROTOCOL.md - 加密与安全](PROTOCOL.md#6-加密与安全)

---

### Q: 如何存储历史消息？
**A:** 使用 **SQLite** 本地数据库：
- 消息表支持全文搜索（FTS5）
- 文件可内联存储或引用文件系统
- 支持增量同步和离线缓存

详见 [DATABASE.md - 数据库 Schema](DATABASE.md#2-数据库-schema)

---

### Q: 成员的 CTF 技能信息存储在哪里？
**A:** 存储在 `members` 表的 JSON 字段中：
- `skills`: 技能标签（Web/Pwn/Reverse 等）
- `expertise`: 擅长领域（具体漏洞类型、工具）
- `current_task`: 当前正在解的题目

详见 [DATABASE.md - 成员表](DATABASE.md#22-成员表-members)

---

## 🛠️ 开发指南

### 环境要求

- **Go**: 1.21+
- **Node.js**: 18+
- **Wails**: 2.8+
- **操作系统**: Windows 10+, macOS 12+, Ubuntu 20.04+

### 克隆项目

```bash
git clone https://github.com/yourorg/crosswire.git
cd crosswire
```

### 安装依赖

```bash
# 后端依赖
cd backend
go mod download

# 前端依赖
cd ../frontend
npm install
```

### 启动开发服务器

```bash
# 在项目根目录
wails dev
```

### 构建生产版本

```bash
# Windows
wails build -platform windows/amd64

# Linux
wails build -platform linux/amd64

# macOS
wails build -platform darwin/amd64
```

详见 [ARCHITECTURE.md - 构建与打包](ARCHITECTURE.md#7-构建与打包)

---

## 📊 性能指标

| 指标 | ARP | HTTPS | mDNS |
|------|-----|-------|------|
| 消息延迟 | 1-3ms | 5-20ms | 200-1000ms |
| 文件速度 | 50-100 MB/s | 10-50 MB/s | 10-20 KB/s |
| 内存占用 | <200MB | <200MB | <200MB |
| CPU 占用 | <5% | <10% | <5% |

详见 [ARCHITECTURE.md - 性能指标](ARCHITECTURE.md#8-性能指标)

---

## 🔒 安全性

CrossWire 采用多层安全机制：

```
┌────────────────────────────────────┐
│  Transport Security (TLS 1.3)      │  HTTPS 模式
├────────────────────────────────────┤
│  Message Encryption (X25519)  │  所有模式
├────────────────────────────────────┤
│  Authentication (JWT + Challenge)  │  所有模式
├────────────────────────────────────┤
│  Authorization (RBAC)              │  所有模式
└────────────────────────────────────┘
```

详见 [ARCHITECTURE.md - 安全架构](ARCHITECTURE.md#9-安全架构)

---

## 📝 贡献指南

欢迎贡献代码和文档！请遵循以下步骤：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

---

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](../LICENSE) 文件。

---

## 📧 联系方式

- **Issue Tracker**: [GitHub Issues](https://github.com/yourorg/crosswire/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourorg/crosswire/discussions)
- **Email**: dev@crosswire.local

---

## 🙏 致谢

感谢以下开源项目：

- [Wails](https://wails.io/) - Go + Web 桌面应用框架
- [Vue.js](https://vuejs.org/) - 渐进式 JavaScript 框架
- [gopacket](https://github.com/google/gopacket) - Go 网络包处理库
- [hashicorp/mdns](https://github.com/hashicorp/mdns) - mDNS 实现

---

**最后更新：** 2025-10-05  
**文档版本：** 1.0.0
