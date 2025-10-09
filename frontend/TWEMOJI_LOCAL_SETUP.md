# Twemoji 本地化配置指南

## 问题说明

当前项目使用 Unicode emoji 字符（如 😀），这依赖于用户系统字体，在不同平台显示效果可能不一致：
- Windows 10/11：Segoe UI Emoji
- macOS：Apple Color Emoji
- Linux：Noto Color Emoji 或其他

为了保证跨平台一致的 emoji 显示效果，建议使用 **Twemoji**（Twitter Emoji）本地 SVG 方案。

---

## 方案一：使用 twemoji JavaScript 库（推荐）

### 1. 安装 twemoji

```bash
cd frontend
npm install twemoji
```

### 2. 创建 emoji 渲染工具

创建 `frontend/src/utils/emoji.js`:

```javascript
import twemoji from 'twemoji'

/**
 * 将 Unicode emoji 转换为 Twemoji SVG
 * @param {string} text - 包含 emoji 的文本
 * @returns {string} - 转换后的 HTML
 */
export function renderEmoji(text) {
  return twemoji.parse(text, {
    folder: 'svg',
    ext: '.svg',
    base: 'https://cdn.jsdelivr.net/gh/twitter/twemoji@14.0.2/assets/'
  })
}

/**
 * 本地 SVG 模式（离线使用）
 * 需要先下载 SVG 文件到 public/emojis/twemoji/
 */
export function renderEmojiLocal(text) {
  return twemoji.parse(text, {
    folder: 'svg',
    ext: '.svg',
    base: '/emojis/twemoji/'
  })
}
```

### 3. 下载 Twemoji SVG 文件（离线环境）

```bash
# 克隆 twemoji 仓库
git clone --depth 1 https://github.com/twitter/twemoji.git temp-twemoji

# 复制 SVG 文件到项目
mkdir -p frontend/public/emojis/twemoji/svg
cp -r temp-twemoji/assets/svg/* frontend/public/emojis/twemoji/svg/

# 清理临时文件
rm -rf temp-twemoji
```

### 4. 修改组件使用 twemoji

修改 `frontend/src/components/EmojiPicker.vue`:

```vue
<template>
  <!-- 其他代码保持不变 -->
  <span
    v-for="emoji in getFilteredEmojis(category.key)"
    :key="emoji.char"
    class="emoji-item"
    :title="emoji.name"
    @click="selectEmoji(emoji)"
    v-html="renderEmoji(emoji.char)"
  ></span>
</template>

<script setup>
import { renderEmojiLocal as renderEmoji } from '@/utils/emoji'
// ... 其他导入
</script>

<style scoped>
/* 添加 twemoji 样式 */
.emoji-item :deep(img.emoji) {
  width: 1.5em;
  height: 1.5em;
  vertical-align: -0.1em;
}
</style>
```

---

## 方案二：纯 CSS 方案（最简单）

如果不想引入额外依赖，可以使用 CSS 字体回退：

修改 `frontend/src/styles/index.css`:

```css
/* Emoji 字体优化 */
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 
    'Helvetica Neue', Arial, 'Noto Sans', sans-serif, 
    'Apple Color Emoji', 'Segoe UI Emoji', 'Segoe UI Symbol', 
    'Noto Color Emoji';
}

.emoji-item,
.message-content {
  font-family: 'Apple Color Emoji', 'Segoe UI Emoji', 'Noto Color Emoji', 
    'Segoe UI Symbol', 'Twemoji Mozilla', 'EmojiOne Color', sans-serif;
}
```

---

## 方案三：使用 emoji-mart（功能最全）

如果需要更强大的 emoji 选择器，可以使用 `emoji-mart`:

```bash
npm install emoji-mart
```

然后替换现有的 `EmojiPicker.vue` 组件。

---

## 推荐方案

### 对于 CTF 离线环境：
**方案一** + 本地 SVG 文件（`renderEmojiLocal`）

### 对于在线环境：
**方案一** + CDN（`renderEmoji`）

### 最简方案：
**方案二**（纯 CSS，无额外依赖）

---

## 文件大小参考

- Twemoji SVG 全集：约 **15 MB**（3000+ 文件）
- twemoji npm 包：约 **100 KB**
- 常用 emoji SVG（200 个）：约 **1 MB**

如果只需要常用 emoji，可以只复制部分 SVG 文件以减小体积。

---

## 测试验证

1. 打开 emoji 选择器
2. 选择一个 emoji
3. 检查是否显示为统一的 Twitter 风格
4. 在不同操作系统上测试显示效果

---

## 常见问题

### Q: 为什么需要本地化 twemoji？
A: 避免 CDN 依赖，确保离线环境下 emoji 正常显示。

### Q: 15 MB SVG 文件会影响性能吗？
A: 不会。SVG 文件只在需要时按需加载，不会全部加载到内存。

### Q: 可以只下载部分 emoji 吗？
A: 可以。根据 `emoji-data.js` 中的 emoji 列表，只复制对应的 SVG 文件。

---

## 相关链接

- [Twemoji GitHub](https://github.com/twitter/twemoji)
- [Emoji Unicode 标准](https://unicode.org/emoji/charts/full-emoji-list.html)
- [CDN 备选](https://cdn.jsdelivr.net/gh/twitter/twemoji@14.0.2/assets/)

