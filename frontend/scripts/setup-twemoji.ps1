# Twemoji 本地化安装脚本（Windows PowerShell）

Write-Host "📦 开始安装 Twemoji..." -ForegroundColor Cyan

# 1. 安装 npm 包
Write-Host "`n1️⃣ 安装 twemoji npm 包..." -ForegroundColor Yellow
npm install twemoji

# 2. 下载 SVG 文件（可选，用于离线环境）
Write-Host "`n2️⃣ 下载 Twemoji SVG 文件..." -ForegroundColor Yellow

if (-not (Test-Path "public/emojis/twemoji")) {
    New-Item -ItemType Directory -Path "public/emojis/twemoji" -Force | Out-Null
    
    # 克隆仓库
    Write-Host "   克隆 twemoji 仓库..." -ForegroundColor Gray
    git clone --depth 1 https://github.com/twitter/twemoji.git temp-twemoji
    
    # 复制 SVG 文件
    Write-Host "   复制 SVG 文件..." -ForegroundColor Gray
    Copy-Item -Path "temp-twemoji/assets/svg" -Destination "public/emojis/twemoji/" -Recurse
    
    # 清理
    Write-Host "   清理临时文件..." -ForegroundColor Gray
    Remove-Item -Path "temp-twemoji" -Recurse -Force
    
    Write-Host "   ✅ SVG 文件已复制到 public/emojis/twemoji/svg/" -ForegroundColor Green
} else {
    Write-Host "   ⏭️  SVG 文件已存在，跳过下载" -ForegroundColor Gray
}

# 3. 创建工具函数
Write-Host "`n3️⃣ 创建 emoji 工具函数..." -ForegroundColor Yellow

$emojiUtilContent = @"
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
"@

if (-not (Test-Path "src/utils")) {
    New-Item -ItemType Directory -Path "src/utils" -Force | Out-Null
}

$emojiUtilContent | Out-File -FilePath "src/utils/emoji.js" -Encoding UTF8

Write-Host "   ✅ 工具函数已创建: src/utils/emoji.js" -ForegroundColor Green

# 4. 提示下一步
Write-Host "`n✅ Twemoji 安装完成！" -ForegroundColor Green
Write-Host "`n📝 接下来需要手动修改以下文件：" -ForegroundColor Cyan
Write-Host ""
Write-Host "1. frontend/src/components/EmojiPicker.vue" -ForegroundColor White
Write-Host "   - 导入: import { renderEmojiLocal } from '@/utils/emoji'" -ForegroundColor Gray
Write-Host "   - 修改: v-html=`"renderEmojiLocal(emoji.char)`"" -ForegroundColor Gray
Write-Host ""
Write-Host "2. frontend/src/components/MessageBubble.vue（如果有 emoji 显示）" -ForegroundColor White
Write-Host "   - 同样使用 v-html 和 renderEmojiLocal" -ForegroundColor Gray
Write-Host ""
Write-Host "3. 添加 CSS 样式（在对应组件中）：" -ForegroundColor White
Write-Host "   .emoji-item :deep(img.emoji) {" -ForegroundColor Gray
Write-Host "     width: 1.5em;" -ForegroundColor Gray
Write-Host "     height: 1.5em;" -ForegroundColor Gray
Write-Host "     vertical-align: -0.1em;" -ForegroundColor Gray
Write-Host "   }" -ForegroundColor Gray
Write-Host ""
Write-Host "💡 提示：" -ForegroundColor Cyan
Write-Host "   - 离线环境：使用 renderEmojiLocal" -ForegroundColor Gray
Write-Host "   - 在线环境：使用 renderEmoji" -ForegroundColor Gray
Write-Host "   - 自动切换：使用 renderEmojiAuto" -ForegroundColor Gray

