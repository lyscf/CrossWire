# 组件集成指南

## 🎯 概述

本文档说明如何在 CrossWire 前端中使用和集成各种高级组件。

---

## 📦 已集成组件

### ChatView（聊天主界面）

已完整集成以下组件：

#### 1. **顶部工具栏**
```vue
<div class="header-right">
  <!-- 全局搜索 -->
  <SearchBar />
  
  <!-- 通知中心 -->
  <NotificationCenter />
  
  <!-- 文件管理 -->
  <a-button @click="fileManagerVisible = true">
    <FileOutlined />
  </a-button>
  
  <!-- 用户菜单 -->
  <a-dropdown>
    <a-avatar>{{ currentUser.name[0] }}</a-avatar>
    <template #overlay>
      <a-menu>
        <a-menu-item @click="userProfileVisible = true">
          个人资料
        </a-menu-item>
        <a-menu-item @click="settingsVisible = true">
          设置
        </a-menu-item>
      </a-menu>
    </template>
  </a-dropdown>
</div>
```

#### 2. **消息输入区**

MessageInput 组件已集成：
- ✅ **@提及功能** - MentionSelector
- ✅ **表情选择** - EmojiPicker
- ✅ **文件上传** - 待后端对接
- ✅ **Markdown支持** - 可选启用

#### 3. **模态窗口/抽屉**

```vue
<!-- 文件管理器 -->
<FileManager v-model:open="fileManagerVisible" />

<!-- 用户资料 -->
<UserProfile
  v-model:open="userProfileVisible"
  :user-id="currentUser.id"
  :is-editable="true"
  @update="handleProfileUpdate"
/>

<!-- 设置 -->
<Settings
  v-model:open="settingsVisible"
  @save="handleSettingsSave"
/>
```

---

## 🚀 快速开始

### 1. 启动开发服务器

```bash
cd frontend
npm install
npm run dev
```

### 2. 测试功能

#### 全局搜索
- 按 `Ctrl+K` 打开搜索框
- 输入关键词搜索消息、题目、成员
- 使用 Tab 切换搜索类型

#### 通知中心
- 点击顶部铃铛图标
- 查看不同类型的通知
- 点击通知跳转到相关页面

#### 用户资料
- 点击右上角头像
- 选择"个人资料"
- 编辑并保存个人信息

#### 设置
- 点击头像 → 设置
- 配置通用、通知、网络等选项
- 查看快捷键列表

#### 文件管理
- 点击顶部文件图标
- 切换网格/列表视图
- 上传、下载、预览文件

#### @提及
- 在消息输入框输入 `@`
- 自动弹出成员选择器
- 使用键盘 ↑↓ 选择，Enter 确认

#### 表情
- 点击输入框工具栏的笑脸图标
- 选择分类或搜索表情
- 点击表情插入到消息

---

## 🔧 组件配置

### SearchBar

无需配置，直接使用：
```vue
<SearchBar />
```

自动连接到 Store：
- `messageStore` - 搜索消息
- `challengeStore` - 搜索题目
- `memberStore` - 搜索成员

### NotificationCenter

无需配置，直接使用：
```vue
<NotificationCenter />
```

通知数据结构：
```javascript
{
  id: '1',
  type: 'mention', // mention | challenge | flag | system
  title: '标题',
  description: '描述',
  timestamp: new Date(),
  read: false,
  link: '/chat?messageId=123'
}
```

### UserProfile

```vue
<UserProfile
  :open="showProfile"
  :user-id="userId"
  :is-editable="true"
  @update:open="showProfile = $event"
  @update="handleProfileUpdate"
/>
```

Props：
- `open` - 是否显示
- `userId` - 用户 ID（可选）
- `isEditable` - 是否可编辑（默认 false）

Events：
- `update:open` - 关闭事件
- `update` - 保存事件

### Settings

```vue
<Settings
  :open="showSettings"
  @update:open="showSettings = $event"
  @save="handleSave"
/>
```

保存的设置数据：
```javascript
{
  theme: 'light',
  language: 'zh-CN',
  fontSize: 14,
  notifications: {
    desktop: true,
    sound: true,
    types: ['mention', 'challenge']
  },
  network: {
    transport: 'arp',
    autoReconnect: true
  }
}
```

### FileManager

```vue
<FileManager
  :open="showFileManager"
  @update:open="showFileManager = $event"
/>
```

文件数据结构：
```javascript
{
  id: '1',
  name: 'exploit.py',
  type: 'code', // image | document | code | archive
  size: 2048,
  url: '/path/to/file',
  uploader: 'alice',
  uploadedAt: new Date()
}
```

### EmojiPicker

```vue
<EmojiPicker @select="handleEmojiSelect">
  <a-button>
    <SmileOutlined />
  </a-button>
</EmojiPicker>
```

选择事件：
```javascript
const handleEmojiSelect = (emoji) => {
  // emoji: '😀'
  messageContent.value += emoji
}
```

### MentionSelector

```vue
<MentionSelector
  :visible="showMention"
  :search-text="searchText"
  :members="members"
  :position="{ left: '10px', bottom: '50px' }"
  @select="handleMentionSelect"
  @cancel="closeMention"
/>
```

已在 MessageInput 中自动集成。

### MarkdownRenderer

```vue
<!-- 只读模式 -->
<MarkdownRenderer :content="markdown" />

<!-- 编辑模式 -->
<MarkdownRenderer
  v-model:content="markdown"
  :editable="true"
  :rows="10"
/>
```

---

## 🎨 样式定制

### 修改主题色

在 `App.vue` 中：

```javascript
const themeConfig = ref({
  token: {
    colorPrimary: '#1890ff', // 修改主色调
    borderRadius: 4,          // 修改圆角
    fontSize: 14              // 修改字体大小
  }
})
```

### 自定义组件样式

每个组件都支持通过 scoped CSS 覆盖样式：

```vue
<style scoped>
:deep(.search-bar) {
  width: 400px; /* 修改搜索框宽度 */
}

:deep(.notification-panel) {
  width: 500px; /* 修改通知面板宽度 */
}
</style>
```

---

## 📡 数据连接

### 连接到 Pinia Store

所有组件都设计为与 Pinia Store 配合使用：

```javascript
// stores/message.js
export const useMessageStore = defineStore('message', {
  state: () => ({
    messages: []
  }),
  actions: {
    addMessage(message) {
      this.messages.push(message)
    }
  }
})
```

在组件中使用：

```javascript
import { useMessageStore } from '@/stores/message'

const messageStore = useMessageStore()
const messages = computed(() => messageStore.messages)
```

### 与 Wails 后端对接

在需要后端交互的地方：

```javascript
import { SendMessage } from '@/wailsjs/go/main/App'

const sendMessage = async (content) => {
  try {
    await SendMessage(content)
    message.success('发送成功')
  } catch (error) {
    message.error('发送失败')
  }
}
```

---

## 🐛 常见问题

### Q1: 搜索结果为空？

**A:** 确保相关 Store 有数据：
```javascript
// 检查 Store
console.log(messageStore.messages)
console.log(challengeStore.challenges)
console.log(memberStore.members)
```

### Q2: @提及无法选中成员？

**A:** 检查成员数据格式：
```javascript
// 正确格式
{
  id: 'user1',
  name: 'alice',
  skills: ['Web'],
  status: 'online'
}
```

### Q3: 通知不显示？

**A:** 检查通知数据和设置：
```javascript
// 1. 检查通知数据是否存在
console.log(notifications.value)

// 2. 检查设置是否启用通知
console.log(settings.value.notifications)
```

### Q4: 文件上传失败？

**A:** 文件上传需要后端支持，当前为模拟数据。实际使用时需要：
```javascript
const handleUpload = async (file) => {
  const formData = new FormData()
  formData.append('file', file)
  
  // 调用 Wails API
  await UploadFile(formData)
}
```

### Q5: 表情显示为方框？

**A:** 确保系统支持 Unicode Emoji，或使用图片表情：
```javascript
// 可以替换为图片 URL
const emojiImages = {
  '😀': '/emojis/smile.png'
}
```

---

## 🔄 更新日志

### v1.0.0 (2025-10-05)
- ✅ 完成所有 8 个高级组件
- ✅ 集成到 ChatView
- ✅ 完善文档

---

## 📚 相关文档

- [高级组件文档](./ADVANCED_COMPONENTS.md) - 详细的组件 API
- [功能说明](./FEATURES.md) - @提及功能详解
- [项目状态](./PROJECT_STATUS.md) - 开发进度
- [架构文档](../docs/ARCHITECTURE.md) - 整体架构

---

## 🤝 贡献指南

如需添加新功能或修复 Bug：

1. 创建新分支
2. 编写代码和测试
3. 更新相关文档
4. 提交 Pull Request

---

**文档版本**: v1.0.0  
**更新时间**: 2025-10-05  
**维护者**: CrossWire Team
