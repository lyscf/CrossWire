# CrossWire 开发指南

## 开发环境设置

### 必需软件

1. **Go 1.21+**
   ```bash
   # 检查版本
   go version
   ```

2. **Node.js 18+** (用于前端)
   ```bash
   node --version
   npm --version
   ```

3. **Wails CLI 2.8+**
   ```bash
   # 安装 Wails
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   
   # 检查安装
   wails doctor
   ```

4. **Git**
   ```bash
   git --version
   ```

### 克隆项目

```bash
git clone <repository-url>
cd CrossWire
```

### 安装依赖

```bash
# 使用 Makefile
make deps

# 或手动安装
cd backend
go mod download

# TODO: 前端依赖
# cd frontend
# npm install
```

## 项目结构说明

### 后端结构 (backend/)

```
backend/
├── cmd/
│   └── crosswire/
│       └── main.go              # 程序入口
├── internal/
│   ├── app/                     # Wails 应用层
│   │   └── app.go               # 主应用类，导出给前端
│   ├── models/                  # GORM 数据模型
│   │   ├── types.go             # 类型定义
│   │   ├── channel.go           # 频道模型
│   │   ├── member.go            # 成员模型
│   │   ├── message.go           # 消息模型
│   │   ├── file.go              # 文件模型
│   │   ├── challenge.go         # 题目模型
│   │   └── audit.go             # 审计日志模型
│   ├── storage/                 # 数据库层
│   │   ├── database.go          # 数据库管理器
│   │   ├── channel_repository.go
│   │   ├── member_repository.go
│   │   └── message_repository.go
│   ├── crypto/                  # 加密模块
│   │   └── crypto.go
│   ├── transport/               # 传输层 (TODO)
│   ├── server/                  # 服务端 (TODO)
│   ├── client/                  # 客户端 (TODO)
│   └── utils/                   # 工具函数
│       ├── logger.go
│       └── validator.go
└── go.mod
```

### 数据模型说明

#### 核心模型

1. **Channel (频道)**
   - 存储频道配置、加密密钥、传输模式等
   - 关联：Members, Messages, Files

2. **Member (成员)**
   - 存储成员信息、技能标签、在线状态
   - 支持 CTF 技能管理

3. **Message (消息)**
   - 支持多种类型：文本、代码、文件、系统消息
   - 支持回复、话题、@提及
   - 支持题目聊天室

4. **File (文件)**
   - 支持分块上传、断点续传
   - 加密存储

5. **Challenge (题目)**
   - CTF 题目管理
   - 自动创建题目聊天室
   - Flag 哈希验证

## 开发工作流

### 1. 创建新功能分支

```bash
git checkout -b feature/your-feature-name
```

### 2. 编写代码

#### 添加新的数据模型

1. 在 `backend/internal/models/` 创建模型文件
2. 定义 GORM 模型结构
3. 实现 `TableName()` 方法
4. 添加 `BeforeCreate`/`BeforeUpdate` 钩子（如需要）
5. 在 `database.go` 的 `AutoMigrate` 中注册

#### 添加新的 Repository

1. 在 `backend/internal/storage/` 创建 repository 文件
2. 定义 Repository 结构体
3. 实现 CRUD 方法
4. 使用 GORM 查询构建器

#### 添加 API 方法

1. 在 `backend/internal/app/app.go` 添加导出方法
2. 方法签名格式：`func (a *App) MethodName(params...) (result, error)`
3. 添加详细注释
4. 实现业务逻辑
5. 处理错误

### 3. 测试

```bash
# 运行测试
make test

# 运行 linter
make lint

# 编译检查
make build-backend
```

### 4. 提交代码

```bash
git add .
git commit -m "feat: 添加新功能描述"
git push origin feature/your-feature-name
```

## 编码规范

### Go 代码规范

1. **命名规范**
   - 使用驼峰命名法 (camelCase/PascalCase)
   - 导出的标识符使用大写开头
   - 私有标识符使用小写开头
   - 接口名使用 `-er` 后缀 (如 `Repository`, `Handler`)

2. **注释规范**
   - 所有导出的函数、类型、常量都必须有注释
   - 注释以标识符名称开头
   - 使用完整的句子，以句号结尾

3. **错误处理**
   - 优先使用 `error` 返回值
   - 使用 `fmt.Errorf` 包装错误
   - 不要忽略错误

4. **TODO 注释**
   - 使用 `// TODO: 描述` 标记未完成功能
   - TODO 应该说明需要做什么，而不是为什么

### GORM 使用规范

1. **模型定义**
   ```go
   type Model struct {
       ID        string    `gorm:"primaryKey;type:text" json:"id"`
       CreatedAt time.Time `gorm:"type:integer;not null" json:"created_at"`
       // ...
   }
   ```

2. **查询构建**
   ```go
   // 好的做法
   db.Where("status = ?", "active").Order("created_at DESC").Find(&results)
   
   // 避免
   db.Where("status = 'active'").Find(&results) // SQL 注入风险
   ```

3. **事务处理**
   ```go
   err := db.Transaction(func(tx *gorm.DB) error {
       // 执行多个操作
       if err := tx.Create(&model).Error; err != nil {
           return err
       }
       return nil
   })
   ```

## 调试技巧

### 1. 启用调试日志

在 `storage/database.go` 中设置：
```go
gormConfig.Logger = logger.Default.LogMode(logger.Info)
```

### 2. 使用 Wails Dev Tools

```bash
wails dev
```

在浏览器中按 F12 打开开发者工具

### 3. 查看数据库

```bash
# 安装 SQLite 客户端
# Windows: https://www.sqlite.org/download.html
# Linux: sudo apt install sqlite3
# macOS: brew install sqlite

# 查看数据库
sqlite3 ~/.crosswire/channels/<channel-id>.db
```

## 常见问题

### Q: 如何添加新的传输模式？

A: 
1. 在 `transport/` 目录创建新的实现
2. 实现 `Transport` 接口
3. 在 `factory.go` 中注册

### Q: 如何添加新的消息类型？

A:
1. 在 `models/types.go` 添加类型常量
2. 定义内容结构体
3. 更新消息处理逻辑

### Q: 数据库迁移如何处理？

A: 使用 GORM 的 `AutoMigrate` 功能，它会自动创建表和添加字段（但不会删除）

## 下一步开发优先级

1. ✅ 数据模型（已完成）
2. ✅ 数据库层（已完成基础）
3. ✅ 加密模块（已完成基础）
4. 🔄 传输层实现
5. 🔄 服务端逻辑
6. 🔄 客户端逻辑
7. 🔄 前端界面
8. 🔄 题目管理系统

## 参考资源

- [Wails 文档](https://wails.io/docs/introduction)
- [GORM 文档](https://gorm.io/docs/)
- [Vue 3 文档](https://vuejs.org/)
- [Go 语言规范](https://go.dev/ref/spec)
- [项目文档](docs/)
