# 高级组件使用文档

## 📚 组件概览

CrossWire 前端实现了 8 个高级 UI 组件，覆盖搜索、通知、用户管理、设置、文件管理、表情和 Markdown 等核心功能。

---

## 1. SearchBar - 全局搜索

### 功能特性
- 🔍 **智能搜索**：同时搜索消息、题目、成员
- ⌨️ **快捷键**：`Ctrl+K` 快速打开
- 🎯 **分类展示**：Tab 切换不同类型
- 💡 **高亮匹配**：搜索词高亮显示
- 🚀 **快速导航**：点击结果直接跳转

### 使用方法
```vue
<SearchBar />
```

### 快捷键
- `Ctrl+K` - 打开搜索
- `Esc` - 关闭搜索
- `↑↓` - 导航结果
- `Enter` - 打开选中项

---

## 2. NotificationCenter - 通知中心

### 功能特性
- 🔔 **实时通知**：@提及、题目分配、系统消息
- 📊 **分类管理**：全部/提及/系统
- ✅ **批量操作**：全部已读、清空
- 🔗 **智能跳转**：点击通知直达相关页面
- 📈 **未读提示**：Badge 显示未读数量

### 使用方法
```vue
<NotificationCenter />
```

### 通知类型
| 类型 | 图标 | 颜色 | 说明 |
|------|------|------|------|
| mention | 👤 | 蓝色 | @提及通知 |
| challenge | 🏆 | 橙色 | 题目分配 |
| flag | 🚩 | 绿色 | Flag提交结果 |
| system | ℹ️ | 绿色 | 系统通知 |

---

## 3. UserProfile - 用户资料

### 功能特性
- 👤 **完整信息**：头像、昵称、邮箱、角色
- ✏️ **在线编辑**：直接修改个人资料
- 🏷️ **技能标签**：展示和管理技能
- 📊 **解题统计**：已解题目、积分、排名
- ⏱️ **活动时间线**：最近活动记录

### 使用方法
```vue
<UserProfile
  :open="showProfile"
  :userId="userId"
  :isEditable="true"
  @update:open="showProfile = $event"
  @update="handleProfileUpdate"
/>
```

### Props
| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| open | Boolean | false | 是否显示 |
| userId | String | null | 用户ID |
| isEditable | Boolean | false | 是否可编辑 |

---

## 4. Settings - 设置页面

### 功能特性
- 🎨 **通用设置**：主题、语言、字体大小
- 🔔 **通知设置**：桌面通知、声音、免打扰
- 🌐 **网络设置**：传输模式、自动重连、心跳
- ⚡ **快捷键**：所有快捷键查看
- 🛠️ **高级选项**：日志、缓存、数据导出

### 使用方法
```vue
<Settings
  :open="showSettings"
  @update:open="showSettings = $event"
  @save="handleSettingsSave"
/>
```

### 设置项
```javascript
{
  theme: 'light',  // 主题：light | dark | auto
  language: 'zh-CN',  // 语言
  fontSize: 14,  // 字体大小 12-18
  messageDensity: 'comfortable',  // 消息密度
  notifications: {
    desktop: true,  // 桌面通知
    sound: true,  // 声音提示
    types: ['mention', 'challenge'],  // 通知类型
    dndTime: null  // 免打扰时间
  },
  network: {
    transport: 'arp',  // 传输模式
    autoReconnect: true,  // 自动重连
    heartbeatInterval: 30  // 心跳间隔(秒)
  }
}
```

---

## 5. FileManager - 文件管理器

### 功能特性
- 📁 **双视图**：网格视图 / 列表视图
- 🔍 **智能搜索**：按文件名搜索
- 🏷️ **类型筛选**：图片、文档、代码、压缩包
- 📊 **排序功能**：时间、名称、大小
- 🎯 **批量操作**：批量下载、批量删除
- 👁️ **文件预览**：图片预览、详情展示

### 使用方法
```vue
<FileManager
  :open="showFileManager"
  @update:open="showFileManager = $event"
/>
```

### 支持的文件类型
- 🖼️ **图片**：jpg, png, gif, svg
- 📄 **文档**：txt, md, pdf, doc
- 💻 **代码**：py, js, go, c, cpp
- 📦 **压缩包**：zip, rar, 7z, tar

---

## 6. EmojiPicker - 表情选择器

### 功能特性
- 😀 **8 大分类**：笑脸、手势、动物、食物、旅行、物品、符号、旗帜
- 🔍 **搜索表情**：按名称搜索
- ⭐ **最近使用**：记录最近使用的表情
- 📱 **360+ 表情**：丰富的表情库

### 使用方法
```vue
<EmojiPicker @select="handleEmojiSelect">
  <a-button>
    <SmileOutlined />
  </a-button>
</EmojiPicker>
```

### 分类列表
| 分类 | 图标 | 数量 | 示例 |
|------|------|------|------|
| 笑脸 | 😀 | 60+ | 😀😃😄😁😆 |
| 手势 | 👋 | 30+ | 👋🤚✋👌✌️ |
| 动物 | 🐶 | 50+ | 🐶🐱🐭🐹🐰 |
| 食物 | 🍕 | 50+ | 🍎🍊🍋🍌🍉 |
| 旅行 | ✈️ | 30+ | 🚗🚕🚙✈️🚀 |
| 物品 | ⚽ | 40+ | ⚽🏀🎮📱💻 |
| 符号 | ❤️ | 40+ | ❤️💙💚💛💜 |
| 旗帜 | 🚩 | 20+ | 🇨🇳🇺🇸🇬🇧🇯🇵 |

---

## 7. MentionSelector - @提及选择器

### 功能特性
- 🎯 **智能触发**：输入 @ 自动弹出
- 🔍 **实时搜索**：边输入边过滤
- ⌨️ **键盘导航**：↑↓ Enter Esc Tab
- 👥 **成员信息**：在线状态、技能标签
- 🎨 **高亮显示**：消息中@高亮

### 使用方法
```vue
<MentionSelector
  :visible="showMentionSelector"
  :search-text="searchText"
  :members="members"
  :position="position"
  @select="handleSelect"
  @cancel="handleCancel"
/>
```

### 详细文档
参见 [FEATURES.md](./FEATURES.md)

---

## 8. MarkdownRenderer - Markdown 渲染器

### 功能特性
- ✏️ **编辑/预览**：双模式切换
- 🛠️ **工具栏**：快捷插入格式
- 🎨 **语法高亮**：代码块高亮
- ⚡ **实时渲染**：即时预览效果
- 📊 **字数统计**：实时显示字数

### 使用方法
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

### 支持的语法
- **标题**：`# H1` `## H2` `### H3`
- **粗体**：`**bold**`
- **斜体**：`*italic*`
- **删除线**：`~~strikethrough~~`
- **代码**：`` `code` ``
- **代码块**：` ```language\ncode\n``` `
- **链接**：`[text](url)`
- **图片**：`![alt](url)`
- **引用**：`> quote`
- **列表**：`- item` 或 `1. item`

### 工具栏按钮
| 按钮 | 功能 | 快捷键 |
|------|------|--------|
| **B** | 粗体 | - |
| *I* | 斜体 | - |
| ~~S~~ | 删除线 | - |
| H | 标题 | - |
| > | 引用 | - |
| `<>` | 行内代码 | - |
| { } | 代码块 | - |
| 🔗 | 链接 | - |
| 🖼️ | 图片 | - |
| • | 无序列表 | - |
| 1. | 有序列表 | - |

---

## 📦 组件导入

### 全部导入
```javascript
import SearchBar from '@/components/SearchBar.vue'
import NotificationCenter from '@/components/NotificationCenter.vue'
import UserProfile from '@/components/UserProfile.vue'
import Settings from '@/components/Settings.vue'
import FileManager from '@/components/FileManager.vue'
import EmojiPicker from '@/components/EmojiPicker.vue'
import MentionSelector from '@/components/MentionSelector.vue'
import MarkdownRenderer from '@/components/MarkdownRenderer.vue'
```

### 按需导入
```javascript
import { defineAsyncComponent } from 'vue'

const SearchBar = defineAsyncComponent(() =>
  import('@/components/SearchBar.vue')
)
```

---

## 🎯 使用示例

### 完整的工具栏
```vue
<template>
  <div class="app-toolbar">
    <!-- 搜索 -->
    <SearchBar />

    <!-- 通知 -->
    <NotificationCenter />

    <!-- 用户菜单 -->
    <a-dropdown>
      <a-avatar @click="showProfile = true">
        {{ username[0] }}
      </a-avatar>
      <template #overlay>
        <a-menu>
          <a-menu-item @click="showProfile = true">
            个人资料
          </a-menu-item>
          <a-menu-item @click="showSettings = true">
            设置
          </a-menu-item>
        </a-menu>
      </template>
    </a-dropdown>

    <!-- 用户资料 -->
    <UserProfile
      v-model:open="showProfile"
      :isEditable="true"
    />

    <!-- 设置 -->
    <Settings v-model:open="showSettings" />
  </div>
</template>
```

### 消息输入框集成
```vue
<template>
  <div class="message-input">
    <a-textarea v-model:value="message" />

    <!-- 工具栏 -->
    <div class="toolbar">
      <!-- 文件管理 -->
      <a-button @click="showFileManager = true">
        <PaperClipOutlined />
      </a-button>

      <!-- 表情 -->
      <EmojiPicker @select="insertEmoji">
        <a-button>
          <SmileOutlined />
        </a-button>
      </EmojiPicker>

      <!-- @提及 -->
      <a-button @click="insertMention">
        <UserOutlined />
      </a-button>
    </div>

    <FileManager v-model:open="showFileManager" />
    <MentionSelector
      :visible="showMention"
      @select="handleMention"
    />
  </div>
</template>
```

---

## 🎨 样式定制

所有组件都使用 Ant Design Vue 的主题系统，可以通过修改主题变量来定制样式：

```javascript
// main.js
import { theme } from 'ant-design-vue'

app.use(Antd, {
  theme: {
    token: {
      colorPrimary: '#1890ff',  // 主题色
      borderRadius: 4,  // 圆角
      fontSize: 14  // 字体大小
    }
  }
})
```

---

## 🔧 TypeScript 支持

```typescript
// 组件 Props 类型
interface SearchBarProps {
  // SearchBar 不需要 props
}

interface NotificationProps {
  // NotificationCenter 不需要 props
}

interface UserProfileProps {
  open: boolean
  userId?: string
  isEditable?: boolean
}

interface SettingsProps {
  open: boolean
}

interface FileManagerProps {
  open: boolean
}

interface EmojiPickerProps {
  // EmojiPicker 通过 slot 传入触发元素
}

interface MentionSelectorProps {
  visible: boolean
  searchText: string
  members: Member[]
  position: { left: string; bottom: string }
}

interface MarkdownRendererProps {
  content: string
  editable?: boolean
  rows?: number
}
```

---

## 📊 性能优化

### 懒加载
```javascript
// router/index.js
const routes = [
  {
    path: '/settings',
    component: () => import('@/views/SettingsView.vue')
  }
]
```

### 按需加载表情
```javascript
// EmojiPicker.vue
const emojiData = {
  smileys: () => import('@/data/emojis/smileys.json'),
  animals: () => import('@/data/emojis/animals.json')
  // ...
}
```

---

## 🐛 常见问题

### Q: 搜索结果为空？
A: 确保相关 Store 中有数据，如 messageStore、challengeStore、memberStore。

### Q: 通知不显示？
A: 检查是否在设置中启用了对应类型的通知。

### Q: @提及无法选择成员？
A: 确保 memberStore 中有成员数据，且成员对象包含 `id` 和 `name` 字段。

### Q: Markdown 渲染异常？
A: 当前使用简单的正则替换，建议在生产环境使用 `markdown-it` 库。

---

## 🚀 未来增强

1. **SearchBar**
   - [ ] 搜索历史记录
   - [ ] 高级搜索过滤器
   - [ ] 搜索建议

2. **NotificationCenter**
   - [ ] 浏览器桌面通知
   - [ ] 声音提示
   - [ ] 通知持久化

3. **EmojiPicker**
   - [ ] 自定义表情
   - [ ] 表情皮肤选择
   - [ ] 表情动画

4. **MarkdownRenderer**
   - [ ] 集成 markdown-it
   - [ ] 代码语法高亮
   - [ ] LaTeX 数学公式
   - [ ] 图表支持

---

**文档版本**: v1.0.0  
**更新时间**: 2025-10-05  
**维护者**: CrossWire Team

