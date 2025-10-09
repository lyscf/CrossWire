#!/bin/bash
# Twemoji 本地化安装脚本

set -e

echo "📦 开始安装 Twemoji..."

# 1. 安装 npm 包
echo "1️⃣ 安装 twemoji npm 包..."
npm install twemoji

# 2. 下载 SVG 文件（可选，用于离线环境）
echo "2️⃣ 下载 Twemoji SVG 文件..."
if [ ! -d "public/emojis/twemoji" ]; then
  mkdir -p public/emojis/twemoji
  
  # 克隆仓库
  echo "   克隆 twemoji 仓库..."
  git clone --depth 1 https://github.com/twitter/twemoji.git temp-twemoji
  
  # 复制 SVG 文件
  echo "   复制 SVG 文件..."
  cp -r temp-twemoji/assets/svg public/emojis/twemoji/
  
  # 清理
  echo "   清理临时文件..."
  rm -rf temp-twemoji
  
  echo "   ✅ SVG 文件已复制到 public/emojis/twemoji/svg/"
else
  echo "   ⏭️  SVG 文件已存在，跳过下载"
fi

# 3. 创建工具函数
echo "3️⃣ 创建 emoji 工具函数..."
cat > src/utils/emoji.js << 'EOF'
import twemoji from 'twemoji'

/**
 * 将 Unicode emoji 转换为 Twemoji SVG（在线模式）
 * @param {string} text - 包含 emoji 的文本
 * @returns {string} - 转换后的 HTML
 */
export function renderEmoji(text) {
  if (!text) return ''
  return twemoji.parse(text, {
    folder: 'svg',
    ext: '.svg',
    base: 'https://cdn.jsdelivr.net/gh/twitter/twemoji@14.0.2/assets/'
  })
}

/**
 * 本地 SVG 模式（离线使用）
 * 需要先下载 SVG 文件到 public/emojis/twemoji/
 * @param {string} text - 包含 emoji 的文本
 * @returns {string} - 转换后的 HTML
 */
export function renderEmojiLocal(text) {
  if (!text) return ''
  return twemoji.parse(text, {
    folder: 'svg',
    ext: '.svg',
    base: '/emojis/twemoji/'
  })
}

/**
 * 自动选择模式（优先离线）
 */
export function renderEmojiAuto(text) {
  // 检测是否有本地 SVG 文件
  // 生产环境默认使用本地，开发环境使用 CDN
  if (import.meta.env.PROD) {
    return renderEmojiLocal(text)
  } else {
    return renderEmoji(text)
  }
}
EOF

echo "   ✅ 工具函数已创建: src/utils/emoji.js"

# 4. 提示下一步
echo ""
echo "✅ Twemoji 安装完成！"
echo ""
echo "📝 接下来需要手动修改以下文件："
echo ""
echo "1. frontend/src/components/EmojiPicker.vue"
echo "   - 导入: import { renderEmojiLocal } from '@/utils/emoji'"
echo "   - 修改: v-html=\"renderEmojiLocal(emoji.char)\""
echo ""
echo "2. frontend/src/components/MessageBubble.vue（如果有 emoji 显示）"
echo "   - 同样使用 v-html 和 renderEmojiLocal"
echo ""
echo "3. 添加 CSS 样式（在对应组件中）："
echo "   .emoji-item :deep(img.emoji) {"
echo "     width: 1.5em;"
echo "     height: 1.5em;"
echo "     vertical-align: -0.1em;"
echo "   }"
echo ""
echo "💡 提示："
echo "   - 离线环境：使用 renderEmojiLocal"
echo "   - 在线环境：使用 renderEmoji"
echo "   - 自动切换：使用 renderEmojiAuto"

