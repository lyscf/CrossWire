# CrossWire 前端开发环境设置

## 快速开始

### 1. 安装 Node.js

确保已安装 Node.js 18+ 版本：

```bash
node --version
# 应该显示 v18.x.x 或更高版本
```

如果没有安装，请从 [Node.js 官网](https://nodejs.org/) 下载安装。

### 2. 安装依赖

在 `frontend` 目录下运行：

```bash
cd frontend
npm install
```

这将安装以下核心依赖：
- Vue 3.4.21
- Ant Design Vue 4.1.2
- Pinia 2.1.7
- Vue Router 4.3.0
- Vite 5.1.6

### 3. 启动开发服务器

```bash
npm run dev
```

浏览器将自动打开 `http://localhost:5173`

### 4. 构建生产版本

```bash
npm run build
```

构建产物将输出到 `dist/` 目录。

## 与 Wails 集成

### 完整开发流程

1. **前端开发模式**（纯前端调试）：
   ```bash
   cd frontend
   npm run dev
   ```

2. **Wails 开发模式**（前后端联调）：
   ```bash
   # 在项目根目录
   wails dev
   ```
   
   Wails 会自动启动 Go 后端和前端 Vite 服务器。

3. **构建桌面应用**：
   ```bash
   wails build
   ```

## 目录说明

```
frontend/
├── src/
│   ├── views/           # 页面组件
│   │   ├── HomeView.vue        # 首页（模式选择）
│   │   ├── ServerView.vue      # 服务端配置
│   │   ├── ClientView.vue      # 客户端加入
│   │   └── ChatView.vue        # 聊天界面
│   │
│   ├── components/      # 可复用组件
│   │   ├── MessageList.vue     # 消息列表
│   │   ├── MessageInput.vue    # 消息输入框
│   │   └── MemberList.vue      # 成员列表
│   │
│   ├── stores/          # Pinia 状态管理
│   │   ├── app.js              # 应用状态
│   │   ├── channel.js          # 频道状态
│   │   ├── message.js          # 消息状态
│   │   ├── member.js           # 成员状态
│   │   └── file.js             # 文件状态
│   │
│   ├── router/          # 路由配置
│   ├── styles/          # 全局样式
│   └── assets/          # 静态资源
│
├── wailsjs/             # Wails 自动生成的 Go API 绑定
│   └── go/main/
│       ├── App.js              # Go App 方法的 JS 封装
│       └── App.d.ts            # TypeScript 类型定义
│
├── index.html           # HTML 模板
├── vite.config.js       # Vite 配置
└── package.json         # 依赖管理
```

## 开发技巧

### 1. 调用 Go 后端方法

```javascript
// 导入 Wails 生成的 API
import { StartServerMode } from '@/wailsjs/go/main/App'

// 调用 Go 方法
async function startServer() {
  try {
    const result = await StartServerMode(config)
    console.log('Server started:', result)
  } catch (error) {
    console.error('Failed to start server:', error)
  }
}
```

### 2. 监听运行时事件

```javascript
import { EventsOn } from '@/wailsjs/runtime/runtime'

// 监听后端事件
EventsOn('message:received', (message) => {
  console.log('New message:', message)
  // 更新状态
})
```

### 3. 使用 Pinia Store

```javascript
import { useAppStore } from '@/stores/app'
import { useMessageStore } from '@/stores/message'

const appStore = useAppStore()
const messageStore = useMessageStore()

// 读取状态
console.log(appStore.isConnected)

// 更新状态
appStore.setConnected(true)
messageStore.addMessage('main', newMessage)
```

### 4. 路由导航

```javascript
import { useRouter } from 'vue-router'

const router = useRouter()

// 编程式导航
router.push('/chat')
router.push({ name: 'server' })
```

## 常见问题

### Q1: npm install 失败

**解决方案：**
```bash
# 清除缓存
npm cache clean --force

# 使用国内镜像
npm config set registry https://registry.npmmirror.com

# 重新安装
npm install
```

### Q2: Vite 启动失败

**解决方案：**
```bash
# 删除 node_modules 和 lock 文件
rm -rf node_modules package-lock.json

# 重新安装
npm install
```

### Q3: Wails 找不到前端资源

**解决方案：**

确保 `wails.json` 中的前端配置正确：

```json
{
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "http://localhost:5173"
}
```

## 推荐开发工具

- **VSCode** + Vue Language Features (Volar) 插件
- **Vue DevTools** 浏览器扩展
- **ESLint** + **Prettier** 代码格式化

## 热重载

Wails 开发模式下支持热重载：
- 前端代码修改会自动刷新
- Go 代码修改会自动重新编译

## 下一步

参考 `README.md` 了解更多功能实现细节。

