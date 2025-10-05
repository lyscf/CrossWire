# Emoji 资源说明

## 📦 本地化 Emoji

为了支持离线 CTF 环境，CrossWire 使用完全本地化的 Emoji 系统。

---

## 🎯 使用方案

### 方案一：Unicode Emoji（默认）

**优点**：
- 无需额外资源
- 体积小
- 浏览器原生支持

**缺点**：
- 不同平台显示效果不同
- 部分老旧系统可能不支持

**当前状态**：✅ 已实现

### 方案二：Twemoji SVG（推荐）

如需统一的显示效果，可以下载 Twemoji 资源包：

```bash
# 1. 下载 Twemoji
git clone https://github.com/twitter/twemoji.git

# 2. 复制 SVG 文件到项目
cp -r twemoji/assets/svg frontend/public/emojis/svg/

# 3. 复制完成后大小约 15MB（2000+ SVG 文件）
```

**Twemoji 官方链接**：
- GitHub: https://github.com/twitter/twemoji
- CDN: https://twemoji.maxcdn.com/v/latest/svg/

---

## 📁 目录结构

```
frontend/public/emojis/
├── README.md           # 本文档
├── svg/                # SVG 格式表情（可选）
│   ├── 1f600.svg      # 😀
│   ├── 1f603.svg      # 😃
│   ├── 1f604.svg      # 😄
│   └── ...            # 2000+ 文件
├── png/                # PNG 格式表情（可选）
│   ├── 72x72/         # 小尺寸
│   └── 144x144/       # 大尺寸
└── sprite.png          # Sprite Sheet（可选）
```

---

## 🔧 集成方式

### 当前实现（Unicode）

```javascript
// src/assets/emojis/emoji-data.js
export const EMOJI_DATA = {
  smileys: [
    { code: '1f600', char: '😀', name: '笑脸' }
  ]
}
```

### 启用 SVG（可选）

如果下载了 Twemoji SVG 文件，可以使用：

```vue
<!-- 创建 EmojiImage 组件 -->
<template>
  <img
    v-if="useSvg"
    :src="`/emojis/svg/${code}.svg`"
    :alt="char"
    :title="name"
    class="emoji-svg"
  />
  <span v-else class="emoji-unicode">
    {{ char }}
  </span>
</template>

<script setup>
defineProps({
  code: String,  // 如：'1f600'
  char: String,  // 如：'😀'
  name: String,  // 如：'笑脸'
  useSvg: {
    type: Boolean,
    default: false  // 默认使用 Unicode
  }
})
</script>

<style scoped>
.emoji-svg {
  width: 1.2em;
  height: 1.2em;
  vertical-align: -0.2em;
}

.emoji-unicode {
  font-family: "Apple Color Emoji", "Segoe UI Emoji", "NotoColorEmoji";
}
</style>
```

---

## 📊 资源大小对比

| 方案 | 大小 | 优点 | 缺点 |
|------|------|------|------|
| Unicode | 0 KB | 无需资源 | 跨平台不一致 |
| SVG (全部) | ~15 MB | 统一显示 | 体积较大 |
| SVG (精选) | ~2 MB | 平衡方案 | 需要筛选 |
| PNG Sprite | ~5 MB | 加载快 | 固定尺寸 |

---

## 🎨 精选 Emoji 列表

如果只想包含常用 Emoji，可以仅下载这些文件（约 200 个）：

### 表情（50个）
```
1f600.svg (😀) 1f603.svg (😃) 1f604.svg (😄) 1f601.svg (😁)
1f606.svg (😆) 1f605.svg (😅) 1f923.svg (🤣) 1f602.svg (😂)
1f642.svg (🙂) 1f609.svg (😉) 1f60a.svg (😊) 1f607.svg (😇)
1f970.svg (🥰) 1f60d.svg (😍) 1f929.svg (🤩) 1f618.svg (😘)
1f60b.svg (😋) 1f61b.svg (😛) 1f61c.svg (😜) 1f92a.svg (🤪)
1f911.svg (🤑) 1f917.svg (🤗) 1f92d.svg (🤭) 1f92b.svg (🤫)
1f914.svg (🤔) 1f928.svg (🤨) 1f610.svg (😐) 1f636.svg (😶)
1f60f.svg (😏) 1f644.svg (🙄) 1f62c.svg (😬) 1f925.svg (🤥)
1f634.svg (😴) 1f637.svg (😷) 1f912.svg (🤒) 1f915.svg (🤕)
1f635.svg (😵) 1f92f.svg (🤯) 1f920.svg (🤠) 1f973.svg (🥳)
1f60e.svg (😎) 1f913.svg (🤓) ...
```

### 手势（30个）
```
1f44b.svg (👋) 1f44c.svg (👌) 270c.svg (✌️) 1f44d.svg (👍)
1f44e.svg (👎) 1f44a.svg (👊) 1f44f.svg (👏) 1f64c.svg (🙌)
1f450.svg (👐) 1f91d.svg (🤝) 1f64f.svg (🙏) ...
```

### 符号（30个）
```
2764.svg (❤️) 1f499.svg (💙) 1f49a.svg (💚) 1f49b.svg (💛)
1f49c.svg (💜) 1f494.svg (💔) 2b50.svg (⭐) 1f31f.svg (🌟)
2728.svg (✨) 26a1.svg (⚡) 1f525.svg (🔥) 1f4af.svg (💯)
2705.svg (✅) 274c.svg (❌) 2757.svg (❗) 2753.svg (❓)
26a0.svg (⚠️) 1f6ab.svg (🚫) ...
```

### 动物（20个）
```
1f436.svg (🐶) 1f431.svg (🐱) 1f42d.svg (🐭) 1f439.svg (🐹)
1f430.svg (🐰) 1f98a.svg (🦊) 1f43b.svg (🐻) 1f43c.svg (🐼)
1f981.svg (🦁) 1f42f.svg (🐯) ...
```

### 食物（20个）
```
1f34e.svg (🍎) 1f34c.svg (🍌) 1f349.svg (🍉) 1f34a.svg (🍊)
1f347.svg (🍇) 1f353.svg (🍓) 1f35e.svg (🍞) 1f354.svg (🍔)
1f35f.svg (🍟) 1f355.svg (🍕) ...
```

### 活动（20个）
```
26bd.svg (⚽) 1f3c0.svg (🏀) 1f3c8.svg (🏈) 26be.svg (⚾)
1f3be.svg (🎾) 1f3d0.svg (🏐) 1f3af.svg (🎯) 1f3ae.svg (🎮)
1f3a4.svg (🎤) 1f3a7.svg (🎧) ...
```

### 旅行（20个）
```
1f697.svg (🚗) 1f695.svg (🚕) 1f699.svg (🚙) 1f68c.svg (🚌)
2708.svg (✈️) 1f680.svg (🚀) 1f6f8.svg (🛸) ...
```

### 物品（20个）
```
1f4f1.svg (📱) 1f4bb.svg (💻) 1f5a5.svg (🖥️) 1f4f7.svg (📷)
1f4a1.svg (💡) 1f512.svg (🔒) 1f513.svg (🔓) 1f511.svg (🔑)
1f528.svg (🔨) 1f527.svg (🔧) ...
```

---

## 🚀 下载脚本

创建一个脚本自动下载精选 Emoji：

```bash
#!/bin/bash
# download-emojis.sh

# 创建目录
mkdir -p frontend/public/emojis/svg

# 定义精选 Emoji 代码列表
EMOJIS=(
  "1f600" "1f603" "1f604" "1f601" "1f606" "1f605" "1f923" "1f602"
  "1f44b" "1f44c" "270c" "1f44d" "1f44e" "1f44a" "1f44f" "1f64c"
  "2764" "1f499" "1f49a" "1f49b" "1f49c" "1f494" "2b50" "1f31f"
  "1f436" "1f431" "1f42d" "1f439" "1f430" "1f98a" "1f43b" "1f43c"
  "1f34e" "1f34c" "1f349" "1f34a" "1f347" "1f353" "1f35e" "1f354"
  "26bd" "1f3c0" "1f3c8" "26be" "1f3be" "1f3d0" "1f3af" "1f3ae"
  "1f697" "1f695" "1f699" "1f68c" "2708" "1f680" "1f6f8" "1f681"
  "1f4f1" "1f4bb" "1f5a5" "1f4f7" "1f4a1" "1f512" "1f513" "1f511"
  # 添加更多...
)

# 下载每个 Emoji
BASE_URL="https://raw.githubusercontent.com/twitter/twemoji/master/assets/svg"

for code in "${EMOJIS[@]}"; do
  echo "Downloading $code.svg..."
  curl -s "$BASE_URL/$code.svg" -o "frontend/public/emojis/svg/$code.svg"
done

echo "✅ 完成！已下载 ${#EMOJIS[@]} 个表情"
```

---

## 🔗 相关资源

### Emoji 数据源
- **Twemoji**: https://github.com/twitter/twemoji
- **Noto Emoji**: https://github.com/googlefonts/noto-emoji
- **OpenMoji**: https://openmoji.org/
- **EmojiOne**: https://www.joypixels.com/

### Emoji 工具
- **Emoji Picker**: https://github.com/missive/emoji-mart
- **Emoji Regex**: https://github.com/mathiasbynens/emoji-regex
- **Unicode Emoji List**: https://unicode.org/emoji/charts/full-emoji-list.html

---

## 📝 使用建议

### 开发环境
- 使用 Unicode（无需下载）
- 快速迭代，开发效率高

### 生产环境（有网络）
- 使用 CDN 的 Twemoji
- 无需打包，减小体积

### 生产环境（离线 CTF）
- 下载精选 200 个 Emoji SVG（~2MB）
- 打包到应用中
- 统一显示效果

---

## ✅ 当前状态

- ✅ Unicode Emoji 数据文件已创建
- ✅ EmojiPicker 组件已更新
- ✅ 支持搜索和分类
- ✅ localStorage 持久化最近使用
- ⏳ Twemoji SVG 下载（可选）
- ⏳ EmojiImage 组件（可选）

---

**建议**：对于离线 CTF 环境，当前的 Unicode 方案已经足够。如果需要统一显示效果，再考虑下载 Twemoji SVG。

**文档版本**: v1.0.0  
**更新时间**: 2025-10-05
