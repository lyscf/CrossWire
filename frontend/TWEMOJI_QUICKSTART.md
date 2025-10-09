# Twemoji 快速启用指南

## 🚀 快速开始

### Windows（PowerShell）

```powershell
cd frontend
.\scripts\setup-twemoji.ps1
```

### Linux/macOS（Bash）

```bash
cd frontend
chmod +x scripts/setup-twemoji.sh
./scripts/setup-twemoji.sh
```

---

## 📝 手动配置（脚本执行后）

### 1. 修改 `EmojiPicker.vue`

```diff
<script setup>
+ import { renderEmojiLocal } from '@/utils/emoji'
  // ... 其他导入
</script>

<template>
  <span
    v-for="emoji in getFilteredEmojis(category.key)"
    :key="emoji.char"
    class="emoji-item"
    :title="emoji.name"
    @click="selectEmoji(emoji)"
-   {{ emoji.char }}
+   v-html="renderEmojiLocal(emoji.char)"
  ></span>
  
  <!-- 最近使用部分也需要修改 -->
  <span
    v-for="emoji in recentlyUsed"
    :key="emoji.char"
    class="emoji-item"
    @click="selectEmoji(emoji)"
-   {{ emoji.char }}
+   v-html="renderEmojiLocal(emoji.char)"
  ></span>
</template>

<style scoped>
+ /* Twemoji 样式 */
+ .emoji-item :deep(img.emoji) {
+   width: 1.5em;
+   height: 1.5em;
+   vertical-align: -0.1em;
+   cursor: pointer;
+ }
</style>
```

### 2. 如果消息气泡中也显示 emoji

修改 `MessageBubble.vue` 或类似组件：

```diff
<script setup>
+ import { renderEmojiLocal } from '@/utils/emoji'
+ import { computed } from 'vue'
+
+ const props = defineProps(['message'])
+
+ const renderedContent = computed(() => {
+   return renderEmojiLocal(props.message.content)
+ })
</script>

<template>
- <div class="message-text">{{ message.content }}</div>
+ <div class="message-text" v-html="renderedContent"></div>
</template>

<style scoped>
+ .message-text :deep(img.emoji) {
+   width: 1.2em;
+   height: 1.2em;
+   vertical-align: -0.2em;
+ }
</style>
```

---

## ✅ 验证

1. 运行开发服务器：`npm run dev`
2. 打开 emoji 选择器
3. 应该看到统一的 Twitter 风格 emoji
4. 在不同浏览器测试显示效果

---

## 📦 文件大小

- `twemoji` npm 包：~100 KB
- 本地 SVG 文件（全集）：~15 MB（3000+ emoji）
- 常用 200 个 emoji：~1 MB

如果想减小体积，可以只保留常用 emoji 的 SVG 文件。

---

## 🔧 故障排除

### Q: SVG 文件无法加载？
检查路径是否正确：`public/emojis/twemoji/svg/1f600.svg`

### Q: 显示仍然使用系统 emoji？
确保：
1. 已安装 `twemoji` npm 包
2. 已导入 `renderEmojiLocal`
3. 使用了 `v-html` 而不是 `{{}}`

### Q: emoji 显示太大/太小？
调整 CSS 中的 `width` 和 `height` 值

---

## 📚 更多选项

详细配置和其他方案请参考 `TWEMOJI_LOCAL_SETUP.md`

