# CrossWire 前端组件清单

## 页面组件 (Views)

### 1. HomeView.vue
- **路径**: `/`
- **功能**: 首页，模式选择（服务端/客户端）
- **特点**: 简洁卡片布局，功能特性展示

### 2. ServerView.vue
- **路径**: `/server`
- **功能**: 服务端配置页面
- **特点**: 表单配置，传输模式选择，网络接口选择

### 3. ClientView.vue
- **路径**: `/client`
- **功能**: 客户端加入页面
- **特点**: 三种加入方式（扫描/手动/二维码）

### 4. ChatView.vue
- **路径**: `/chat`
- **功能**: 聊天主界面
- **特点**: 三栏布局，消息列表，成员侧边栏

### 5. ChallengeView.vue ✨
- **路径**: `/challenges`
- **功能**: 题目管理主视图
- **特点**: 题目列表，题目详情，进度跟踪

## 聊天相关组件

### MessageList.vue
- **功能**: 消息列表展示
- **支持**: 系统消息、文本消息、气泡样式、时间格式化
- **特点**: 自动滚动到底部

### MessageInput.vue
- **功能**: 消息输入框
- **支持**: 文本输入、文件上传、代码块、@提及、表情
- **快捷键**: Ctrl+Enter 发送

### MemberList.vue
- **功能**: 成员列表展示
- **支持**: 在线/离线状态、技能标签、当前任务
- **特点**: 成员操作菜单

## 题目管理组件 ✨

### Challenge/ChallengeList.vue
- **功能**: 题目列表
- **支持**: 滚动加载、筛选

### Challenge/ChallengeCard.vue
- **功能**: 题目卡片
- **显示**: 标题、类型、难度、分值、状态、进度、分配信息
- **特点**: 点击选中，悬停高亮

### Challenge/ChallengeDetail.vue
- **功能**: 题目详情展示
- **支持**: 描述、分配信息、进度跟踪、提交历史
- **操作**: 分配题目、提交 Flag、更新进度

### Challenge/ChallengeCreate.vue
- **功能**: 创建题目弹窗
- **表单**: 标题、类型、难度、分值、描述、Flag、附件
- **特点**: 模态框形式

## 工具组件 ✨

### FilePreview.vue
- **功能**: 文件预览
- **支持**: 
  - 图片：jpg, png, gif, webp
  - 文本：txt, md, log, json
  - PDF：内嵌预览
  - 其他：显示文件信息
- **操作**: 下载、关闭

### CodeEditor.vue
- **功能**: 代码编辑器
- **支持**: 
  - 语言选择：Python, JavaScript, Go, C/C++, Java, PHP, Bash
  - 文件名设置
  - 代码复制
  - 代码发送
- **特点**: 行数统计、字符统计、等宽字体

## 状态管理 (Stores)

### appStore.js
- 应用全局状态
- 运行模式、连接状态、当前用户、设置

### channelStore.js
- 频道状态管理
- 当前频道、频道列表、选中频道、未读计数

### messageStore.js
- 消息状态管理
- 消息列表、置顶消息、CRUD 操作

### memberStore.js
- 成员状态管理
- 成员列表、在线状态、成员筛选

### fileStore.js
- 文件状态管理
- 文件列表、上传进度、文件操作

### challengeStore.js ✨
- 题目状态管理
- 题目列表、分配关系、进度跟踪、提交历史
- 统计功能：解决数量、总分、已得分

## 组件使用示例

### 使用 MessageList 组件

```vue
<template>
  <MessageList :messages="messages" />
</template>

<script setup>
import MessageList from '@/components/MessageList.vue'
import { ref } from 'vue'

const messages = ref([
  {
    id: '1',
    type: 'text',
    senderId: 'user1',
    senderName: 'alice',
    content: 'Hello!',
    timestamp: new Date()
  }
])
</script>
```

### 使用 ChallengeCard 组件

```vue
<template>
  <ChallengeCard
    :challenge="challenge"
    :selected="selectedId === challenge.id"
    @click="handleSelect(challenge.id)"
  />
</template>

<script setup>
import ChallengeCard from '@/components/Challenge/ChallengeCard.vue'

const challenge = {
  id: '1',
  title: 'SQL注入',
  category: 'Web',
  difficulty: 'Easy',
  points: 100,
  status: 'in_progress',
  progress: 60
}
</script>
```

### 使用 CodeEditor 组件

```vue
<template>
  <CodeEditor @send="handleSendCode" />
</template>

<script setup>
import CodeEditor from '@/components/CodeEditor.vue'

const handleSendCode = (codeData) => {
  console.log('Sending code:', codeData)
  // codeData = { type, language, filename, code }
}
</script>
```

## 组件目录结构

```
frontend/src/
├── views/                    # 页面组件
│   ├── HomeView.vue
│   ├── ServerView.vue
│   ├── ClientView.vue
│   ├── ChatView.vue
│   └── ChallengeView.vue    # 题目管理页面
│
├── components/               # 可复用组件
│   ├── MessageList.vue       # 消息列表
│   ├── MessageInput.vue      # 消息输入
│   ├── MemberList.vue        # 成员列表
│   ├── FilePreview.vue       # 文件预览
│   ├── CodeEditor.vue        # 代码编辑器
│   │
│   └── Challenge/            # 题目管理组件
│       ├── ChallengeList.vue
│       ├── ChallengeCard.vue
│       ├── ChallengeDetail.vue
│       └── ChallengeCreate.vue
│
└── stores/                   # Pinia 状态管理
    ├── app.js
    ├── channel.js
    ├── message.js
    ├── member.js
    ├── file.js
    └── challenge.js          # 题目状态
```

## 组件开发规范

### 1. 命名规范
- 组件名使用 PascalCase
- Props 使用 camelCase
- Events 使用 kebab-case

### 2. Props 定义
```javascript
defineProps({
  challenge: {
    type: Object,
    required: true
  },
  selected: {
    type: Boolean,
    default: false
  }
})
```

### 3. Events 定义
```javascript
defineEmits(['select', 'update', 'delete'])
```

### 4. 样式规范
- 使用 scoped 样式
- 遵循 Ant Design 设计规范
- 使用 CSS 变量

## 待开发组件

- [ ] SearchBar.vue - 全局搜索
- [ ] NotificationCenter.vue - 通知中心
- [ ] UserProfile.vue - 用户资料
- [ ] Settings.vue - 设置页面
- [ ] FileManager.vue - 文件管理器
- [ ] EmojiPicker.vue - 表情选择器
- [ ] MentionSelector.vue - @提及选择器
- [ ] MarkdownRenderer.vue - Markdown 渲染器
- [ ] Challenge/ChallengeAssign.vue - 题目分配
- [ ] Challenge/ChallengeProgress.vue - 进度显示
- [ ] Challenge/ChallengeSubmit.vue - 提交 Flag
- [ ] Challenge/ChallengeRoom.vue - 题目聊天室

## 组件依赖关系

```
ChatView
  ├── MessageList
  ├── MessageInput
  │   ├── FilePreview (modal)
  │   └── CodeEditor (modal)
  └── MemberList

ChallengeView
  ├── ChallengeList
  │   └── ChallengeCard
  ├── ChallengeDetail
  └── ChallengeCreate (modal)
```

## 更新日志

- **2025-10-05**: 创建基础聊天组件
- **2025-10-05**: 添加题目管理组件
- **2025-10-05**: 添加文件预览和代码编辑器

---

**文档版本**: 1.0.0  
**最后更新**: 2025-10-05

