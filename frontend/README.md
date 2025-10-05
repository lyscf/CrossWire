# CrossWire Frontend

基于 Vue 3 + Vite + Ant Design Vue 的前端应用。

## 技术栈

- **Vue 3** - 渐进式 JavaScript 框架
- **Vite** - 下一代前端构建工具
- **Ant Design Vue** - 企业级 UI 组件库
- **Pinia** - Vue 3 状态管理
- **Vue Router** - 官方路由管理器
- **Day.js** - 轻量级日期处理库

## 项目结构

```
frontend/
├── src/
│   ├── assets/          # 静态资源
│   │   ├── images/      # 图片资源
│   │   └── fonts/       # 字体文件
│   ├── components/      # 可复用组件
│   │   ├── MessageList.vue
│   │   ├── MessageInput.vue
│   │   └── MemberList.vue
│   ├── views/           # 页面视图
│   │   ├── HomeView.vue    # 首页（模式选择）
│   │   ├── ServerView.vue  # 服务端配置页
│   │   ├── ClientView.vue  # 客户端加入页
│   │   └── ChatView.vue    # 聊天主界面
│   ├── stores/          # Pinia 状态管理
│   │   ├── app.js       # 应用全局状态
│   │   ├── channel.js   # 频道状态
│   │   ├── message.js   # 消息状态
│   │   ├── member.js    # 成员状态
│   │   └── file.js      # 文件状态
│   ├── router/          # 路由配置
│   │   └── index.js
│   ├── styles/          # 样式文件
│   │   ├── index.css
│   │   └── variables.css
│   ├── App.vue          # 根组件
│   └── main.js          # 应用入口
├── wailsjs/             # Wails 自动生成的 API
├── index.html           # HTML 模板
├── package.json         # 依赖配置
└── vite.config.js       # Vite 配置
```

## 开发指南

### 安装依赖

```bash
cd frontend
npm install
```

### 启动开发服务器

```bash
npm run dev
```

### 构建生产版本

```bash
npm run build
```

## 主要功能页面

### 1. 首页 (HomeView)

- 模式选择（服务端/客户端）
- 功能特性展示
- 美观的渐变背景

### 2. 服务端配置页 (ServerView)

- 频道创建配置
- 传输模式选择（ARP/HTTPS/mDNS）
- 网络接口选择
- 高级选项设置

### 3. 客户端加入页 (ClientView)

- 自动扫描局域网频道
- 手动输入服务器地址
- 二维码扫描（开发中）
- 用户信息填写

### 4. 聊天主界面 (ChatView)

- 频道列表侧边栏
- 消息列表显示
- 消息输入框
- 成员列表抽屉
- 置顶消息显示

## 核心组件

### MessageList

消息列表组件，支持：
- 系统消息显示
- 文本消息气泡
- 时间戳格式化
- 自动滚动到底部

### MessageInput

消息输入组件，支持：
- 多行文本输入
- 文件上传按钮
- 代码块插入
- 表情选择
- @提及功能
- Ctrl+Enter 发送

### MemberList

成员列表组件，支持：
- 在线/离线成员分组
- 成员状态标识
- 技能标签显示
- 当前任务显示
- 成员操作菜单

## 状态管理 (Pinia)

### appStore

全局应用状态：
- 运行模式（服务端/客户端）
- 连接状态
- 当前用户信息
- 应用设置

### channelStore

频道状态管理：
- 当前频道信息
- 频道列表
- 未读消息计数
- 频道切换

### messageStore

消息状态管理：
- 消息列表（按频道）
- 置顶消息
- 消息 CRUD 操作
- 消息搜索

### memberStore

成员状态管理：
- 成员列表
- 在线/离线状态
- 成员筛选
- 成员信息更新

### fileStore

文件状态管理：
- 文件列表
- 上传进度
- 文件操作

## 样式规范

- 使用 CSS 变量定义主题色
- 遵循 Ant Design 设计规范
- 响应式布局支持
- 自定义滚动条样式

## Wails 集成

前端通过 Wails Bridge 与 Go 后端通信：

```javascript
// 示例：调用后端方法
import { StartServerMode } from '../wailsjs/go/main/App'

await StartServerMode(config)
```

## 待实现功能

### ✅ 已完成
- [x] **@提及自动补全** - 完整的@提及系统，支持搜索、键盘导航和高亮
- [x] **全局搜索** - Ctrl+K 快捷键，支持搜索消息/题目/成员  
- [x] **通知中心** - 实时通知，支持分类和已读管理
- [x] **用户资料** - 完整的用户信息展示和编辑
- [x] **设置页面** - 通用/通知/网络/快捷键/高级设置
- [x] **文件管理器** - 双视图，支持上传/下载/预览/批量操作
- [x] **表情选择器** - 8大分类，360+ 表情
- [x] **Markdown渲染** - 编辑器和预览器

### 🔜 待增强（可选）
- [ ] 主题切换（暗色模式）
- [ ] 代码语法高亮（集成 Prism.js）
- [ ] LaTeX 数学公式支持
- [ ] 图表可视化

## 📚 项目文档

### 快速导航
- 📖 [项目说明](./README.md) - 项目介绍和使用指南（本文档）
- 🚀 [开发指南](./SETUP.md) - 环境搭建和开发流程
- 🎨 [设计规范](./DESIGN.md) - UI/UX 设计规范和配色系统

### 组件和架构
- 🧩 [组件文档](./COMPONENTS.md) - 所有组件的使用说明
- 🏗️ [架构文档](./ARCHITECTURE.md) - 前端架构和组件关系

### 项目管理
- 📊 [项目状态](./PROJECT_STATUS.md) - 实时开发进度
- ✅ [完成总结](./COMPLETE.md) - 已完成功能总结
- ❌ [缺少内容](./MISSING.md) - 待完成功能清单

### 使用指南
- 🎯 [功能说明](./FEATURES.md) - @提及功能详解
- 🚀 [高级组件](./ADVANCED_COMPONENTS.md) - 8个高级组件使用文档
- 🔧 [集成指南](./INTEGRATION_GUIDE.md) - 组件集成和配置
- 🎬 [演示指南](./DEMO_GUIDE.md) - 完整功能演示流程
- 📦 [离线部署](./OFFLINE_SETUP.md) - 离线CTF环境部署指南

### 其他
- ⚡ [优化记录](./OPTIMIZATION.md) - 性能优化历史
- ✔️ [质量检查](./CHECKLIST.md) - 代码质量清单
- 🎉 [完成总结](./COMPLETION_SUMMARY.md) - 项目完成总结

---

## 参考文档

- [Vue 3 文档](https://cn.vuejs.org/)
- [Ant Design Vue 文档](https://antdv.com/)
- [Pinia 文档](https://pinia.vuejs.org/zh/)
- [Vite 文档](https://cn.vitejs.dev/)
- [Wails 文档](https://wails.io/)

