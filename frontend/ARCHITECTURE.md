# 前端架构概览

## 📐 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                        CrossWire Frontend                    │
│                  Vue 3 + Vite + Ant Design Vue              │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
    ┌───▼────┐           ┌───▼────┐           ┌───▼────┐
    │ Views  │           │Compone │           │ Stores │
    │  层    │───────────│  nts   │───────────│  层    │
    │(5个)   │           │  层    │           │ (6个)  │
    └───┬────┘           │(17个)  │           └───┬────┘
        │                └───┬────┘               │
        │                    │                    │
        └────────────────────┼────────────────────┘
                             │
                    ┌────────▼────────┐
                    │  Wails API 层   │
                    │   (Go Backend)  │
                    └─────────────────┘
```

---

## 📁 文件结构树

```
frontend/src/
│
├── 📄 main.js                 # 应用入口
├── 📄 App.vue                 # 根组件
│
├── 📂 views/                  # 页面组件 (5)
│   ├── HomeView.vue          # ✅ 首页 - 模式选择
│   ├── ServerView.vue        # ✅ 服务端配置
│   ├── ClientView.vue        # ✅ 客户端加入
│   ├── ChatView.vue          # ✅ 聊天主界面
│   └── ChallengeView.vue     # ✅ 题目管理
│
├── 📂 components/             # 组件库 (17)
│   │
│   ├── 💬 聊天组件 (3)
│   │   ├── MessageList.vue    # ✅ 消息列表
│   │   ├── MessageInput.vue   # ✅ 消息输入框
│   │   └── MemberList.vue     # ✅ 成员列表
│   │
│   ├── 🛠️ 工具组件 (2)
│   │   ├── FilePreview.vue    # ✅ 文件预览
│   │   └── CodeEditor.vue     # ✅ 代码编辑器
│   │
│   └── 🏆 题目管理组件 (8)
│       └── Challenge/
│           ├── ChallengeList.vue      # ✅ 题目列表
│           ├── ChallengeCard.vue      # ✅ 题目卡片
│           ├── ChallengeDetail.vue    # ✅ 题目详情
│           ├── ChallengeCreate.vue    # ✅ 创建题目
│           ├── ChallengeAssign.vue    # ✅ 高级分配
│           ├── ChallengeProgress.vue  # ✅ 进度可视化
│           ├── ChallengeSubmit.vue    # ✅ 提交表单
│           └── ChallengeRoom.vue      # ✅ 题目讨论室
│
├── 📂 stores/                 # 状态管理 (6)
│   ├── app.js                # ✅ 应用状态（模式、连接）
│   ├── channel.js            # ✅ 频道状态
│   ├── message.js            # ✅ 消息状态
│   ├── member.js             # ✅ 成员状态
│   ├── file.js               # ✅ 文件状态
│   └── challenge.js          # ✅ 题目状态
│
├── 📂 router/                 # 路由配置
│   └── index.js              # ✅ 路由定义 (5条)
│
├── 📂 styles/                 # 样式文件
│   ├── index.css             # ✅ 全局样式
│   └── variables.css         # ✅ CSS 变量
│
└── 📂 assets/                 # 静态资源
    ├── images/
    └── fonts/
```

---

## 🔄 页面流转图

```
                    ┌─────────────┐
                    │  HomeView   │ 首页
                    │  (/)        │
                    └──────┬──────┘
                           │
          ┌────────────────┴────────────────┐
          │                                 │
    ┌─────▼──────┐                   ┌─────▼──────┐
    │ServerView  │                   │ClientView  │
    │(/server)   │                   │(/client)   │
    └─────┬──────┘                   └─────┬──────┘
          │                                 │
          └────────────────┬────────────────┘
                           │
                    ┌──────▼──────┐
                    │  ChatView   │ 聊天主界面
                    │  (/chat)    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────────┐
                    │ChallengeView    │ 题目管理
                    │(/challenges)    │
                    └─────────────────┘
```

---

## 🧩 组件依赖关系

### ChatView 组件树
```
ChatView
├── MessageList           (消息区域)
│   └── MessageList.vue
├── MessageInput          (输入区域)
│   └── MessageInput.vue
└── MemberList           (成员侧边栏)
    └── MemberList.vue
```

### ChallengeView 组件树
```
ChallengeView
├── Sider
│   └── ChallengeList             (题目列表侧边栏)
│       └── ChallengeCard         (题目卡片)
│
└── Content
    ├── ChallengeDetail           (题目详情)
    │   ├── ChallengeAssign       (分配弹窗)
    │   ├── ChallengeSubmit       (提交弹窗)
    │   ├── ChallengeProgress     (进度弹窗)
    │   └── ChallengeRoom         (讨论室抽屉)
    │
    └── ChallengeCreate           (创建弹窗)
```

---

## 📊 状态管理流程

```
┌─────────────────────────────────────────────────────────┐
│                      Pinia Stores                       │
└─────────────────────────────────────────────────────────┘
         │            │            │            │
    ┌────▼───┐   ┌───▼────┐  ┌───▼────┐  ┌───▼─────┐
    │  App   │   │Channel │  │Message │  │ Member  │
    │ Store  │   │ Store  │  │ Store  │  │ Store   │
    └────┬───┘   └───┬────┘  └───┬────┘  └───┬─────┘
         │            │           │           │
         └────────────┴───────────┴───────────┘
                       │
              ┌────────▼─────────┐
              │  Vue Components  │
              └──────────────────┘
```

### Store 职责划分

| Store | 职责 | 主要数据 |
|-------|------|---------|
| **appStore** | 应用全局状态 | mode, connected, currentUser |
| **channelStore** | 频道管理 | channels, currentChannel |
| **messageStore** | 消息管理 | messages, pinnedMessages |
| **memberStore** | 成员管理 | members, onlineMembers |
| **fileStore** | 文件管理 | files, uploadProgress |
| **challengeStore** | 题目管理 | challenges, assignments |

---

## 🎨 主题系统

### 颜色变量
```css
/* Ant Design 标准配色 */
--primary-color: #1890ff      /* 主题蓝 */
--success-color: #52c41a      /* 成功绿 */
--warning-color: #faad14      /* 警告橙 */
--error-color: #ff4d4f        /* 错误红 */

/* 文本颜色 */
--text-primary: rgba(0,0,0,0.85)    /* 主要文本 */
--text-secondary: rgba(0,0,0,0.65)  /* 次要文本 */
--text-tertiary: rgba(0,0,0,0.45)   /* 辅助文本 */

/* 背景颜色 */
--bg-primary: #ffffff         /* 卡片白 */
--bg-secondary: #fafafa       /* 浅灰 */
--bg-tertiary: #f5f5f5        /* 背景灰 */
```

### 圆角规范
```css
--border-radius-base: 2px     /* 输入框、小组件 */
--border-radius-lg: 4px       /* 卡片、气泡 */
```

---

## 🔌 API 接口规范

### Wails API 调用示例
```javascript
// 从 stores 中调用
import { SendMessage } from '@/wailsjs/go/main/App'

export const messageStore = defineStore('message', {
  actions: {
    async sendMessage(content) {
      try {
        await SendMessage(content)
        // 更新本地状态
        this.messages.push(...)
      } catch (error) {
        console.error('发送失败', error)
      }
    }
  }
})
```

---

## 📦 组件 Props 和 Events

### MessageList
```typescript
Props: {
  messages: Array<Message>
}
Emits: {
  'pin': (messageId: string) => void
  'delete': (messageId: string) => void
}
```

### ChallengeCard
```typescript
Props: {
  challenge: Object<Challenge>
}
Emits: {
  'click': (challengeId: string) => void
}
```

### ChallengeAssign
```typescript
Props: {
  open: Boolean
  challenge: Object<Challenge>
}
Emits: {
  'update:open': (value: boolean) => void
  'assign': (data: AssignData) => void
}
```

---

## 🚀 数据流向

### 消息发送流程
```
1. 用户输入 (MessageInput)
         ↓
2. 调用 messageStore.sendMessage()
         ↓
3. Wails API: SendMessage()
         ↓
4. Go Backend 处理
         ↓
5. 广播消息
         ↓
6. 前端接收事件
         ↓
7. messageStore 更新
         ↓
8. MessageList 自动刷新 (响应式)
```

### 题目分配流程
```
1. 点击"分配题目"按钮 (ChallengeDetail)
         ↓
2. 打开 ChallengeAssign 弹窗
         ↓
3. 选择成员、设置优先级
         ↓
4. 提交分配 @assign 事件
         ↓
5. challengeStore.assignChallenge()
         ↓
6. Wails API: AssignChallenge()
         ↓
7. 更新本地状态
         ↓
8. 显示成功提示
```

---

## 🎯 核心功能映射

| 功能 | 页面/组件 | Store | API |
|------|----------|-------|-----|
| 创建频道 | ServerView | channelStore | CreateChannel |
| 加入频道 | ClientView | channelStore | JoinChannel |
| 发送消息 | MessageInput | messageStore | SendMessage |
| 上传文件 | MessageInput | fileStore | UploadFile |
| 创建题目 | ChallengeCreate | challengeStore | CreateChallenge |
| 分配题目 | ChallengeAssign | challengeStore | AssignChallenge |
| 提交Flag | ChallengeSubmit | challengeStore | SubmitFlag |
| 查看进度 | ChallengeProgress | challengeStore | GetProgress |
| 题目讨论 | ChallengeRoom | messageStore | SendMessage |

---

## 📱 响应式设计

### 布局断点
```css
/* 桌面端优先 */
大屏: > 1200px   (三栏布局)
中屏: 768-1200px (两栏布局)
小屏: < 768px    (单栏布局，抽屉侧边栏)
```

### 组件自适应
- **ChatView**: Drawer 模式显示成员列表
- **ChallengeView**: Drawer 模式显示题目列表
- **ChallengeRoom**: 全屏模式

---

## 🔒 权限控制（预留）

```javascript
// 权限检查示例
const canAssignChallenge = computed(() => {
  return appStore.currentUser?.role === 'admin' || 
         appStore.currentUser?.role === 'leader'
})
```

---

## 📈 性能优化策略

### 已实现
- ✅ 组件懒加载（路由级别）
- ✅ 计算属性缓存
- ✅ 事件委托
- ✅ CSS 变量减少重复

### 待实现
- [ ] 虚拟滚动（消息列表、题目列表）
- [ ] 图片懒加载
- [ ] 代码分割
- [ ] Service Worker

---

## 🧪 开发工具

### 调试工具
```bash
# Vue DevTools
chrome://extensions/

# 性能分析
npm run build -- --mode analyze
```

### 代码规范
```bash
# ESLint 检查
npm run lint

# 格式化
npm run format
```

---

## 📚 技术文档索引

1. **README.md** - 项目说明
2. **SETUP.md** - 环境搭建
3. **DESIGN.md** - 设计规范
4. **COMPONENTS.md** - 组件文档
5. **PROJECT_STATUS.md** - 项目状态
6. **COMPLETE.md** - 完成总结
7. **ARCHITECTURE.md** - 架构文档（本文档）

---

**更新时间**: 2025-10-05  
**架构版本**: v1.0.0

