# 离线环境部署指南

## 🎯 目标

确保 CrossWire 前端在完全离线的 CTF 环境中正常运行。

---

## ✅ 已本地化内容

### 1. Emoji 表情 ✅
- **方案**：使用 Unicode Emoji（浏览器原生支持）
- **数据文件**：`src/assets/emojis/emoji-data.js`
- **持久化**：localStorage 保存最近使用
- **大小**：0 KB（无需额外资源）
- **状态**：✅ 已完成

### 2. UI 组件库 ✅
- **库**：Ant Design Vue
- **打包**：包含在 node_modules 中
- **状态**：✅ 已完成

### 3. 图标 ✅
- **库**：@ant-design/icons-vue
- **打包**：包含在 node_modules 中
- **状态**：✅ 已完成

### 4. 字体 ✅
- **字体**：Nunito（已本地化）
- **位置**：`src/assets/fonts/`
- **状态**：✅ 已完成

### 5. 所有依赖 ✅
- **管理**：npm/package.json
- **打包**：全部在 node_modules 中
- **状态**：✅ 已完成

---

## 📦 离线部署步骤

### 方法一：直接打包（推荐）

```bash
# 1. 在有网络的环境准备
cd frontend
npm install
npm run build

# 2. 打包整个 frontend 目录
tar -czf crosswire-frontend.tar.gz frontend/

# 3. 传输到离线环境
scp crosswire-frontend.tar.gz ctf-server:/path/to/

# 4. 在离线环境解压
tar -xzf crosswire-frontend.tar.gz
cd frontend

# 5. 启动（使用已安装的 node_modules）
npm run dev
```

### 方法二：仅打包 dist（更小）

```bash
# 1. 构建生产版本
cd frontend
npm run build

# 2. 打包 dist 目录
tar -czf crosswire-dist.tar.gz dist/

# 3. 在离线环境使用静态服务器
# 方式 A: 使用 Python
python3 -m http.server 5173 --directory dist

# 方式 B: 使用 Nginx
# 配置 nginx 指向 dist 目录

# 方式 C: 集成到 Wails 应用
# Wails 会自动服务 dist 目录
```

---

## 🔍 验证清单

部署后检查以下项目：

### 基础功能
- [ ] 应用能正常启动
- [ ] 所有页面可以访问
- [ ] 路由导航正常工作

### UI 组件
- [ ] Ant Design 组件正常显示
- [ ] 图标正确显示
- [ ] 自定义字体加载成功

### 高级功能
- [ ] Emoji 选择器可以打开
- [ ] Emoji 搜索功能正常
- [ ] 可以选择和插入 Emoji
- [ ] 最近使用的 Emoji 被保存

### 交互功能
- [ ] 搜索框工作正常 (Ctrl+K)
- [ ] 通知中心可以打开
- [ ] 文件管理器可以打开
- [ ] 用户资料可以编辑
- [ ] 设置页面可以访问
- [ ] @提及功能正常

---

## 🐛 常见问题

### Q1: Emoji 显示为方框？

**原因**：系统字体不支持该 Emoji  
**解决**：
- 方案 A：升级系统（安装最新的 Emoji 字体）
- 方案 B：使用 Twemoji SVG（见下文）

### Q2: 部分图标不显示？

**原因**：字体文件未正确加载  
**检查**：
```bash
# 检查字体文件是否存在
ls frontend/src/assets/fonts/

# 检查 index.css 中的字体引用
grep "@font-face" frontend/src/styles/index.css
```

### Q3: 应用启动后白屏？

**原因**：路由配置或构建问题  
**检查**：
```bash
# 检查控制台错误
打开浏览器开发者工具 → Console

# 检查构建输出
cat frontend/dist/index.html
```

### Q4: localStorage 不工作？

**原因**：浏览器隐私模式或权限问题  
**解决**：
- 使用正常模式（非隐私/无痕模式）
- 检查浏览器设置允许 localStorage

---

## 🎨 可选：启用 Twemoji SVG

如果需要统一的 Emoji 显示效果：

### 1. 下载 Twemoji（有网络环境）

```bash
# 完整版（~15MB）
git clone https://github.com/twitter/twemoji.git
cp -r twemoji/assets/svg frontend/public/emojis/svg/

# 或精选版（~2MB，200个常用）
# 参见 frontend/public/emojis/README.md
```

### 2. 创建 EmojiImage 组件

```vue
<!-- frontend/src/components/EmojiImage.vue -->
<template>
  <img
    v-if="hasSvg"
    :src="`/emojis/svg/${code}.svg`"
    :alt="char"
    :title="name"
    class="emoji-svg"
    @error="onError"
  />
  <span v-else class="emoji-unicode">{{ char }}</span>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  code: String,
  char: String,
  name: String
})

const hasSvg = ref(true)

const onError = () => {
  hasSvg.value = false // 降级到 Unicode
}
</script>

<style scoped>
.emoji-svg {
  width: 1.2em;
  height: 1.2em;
  vertical-align: -0.2em;
}
</style>
```

### 3. 在 EmojiPicker 中使用

```vue
<!-- 替换表情显示 -->
<EmojiImage
  :code="emoji.code"
  :char="emoji.char"
  :name="emoji.name"
/>
```

---

## 📊 体积对比

| 内容 | 大小 | 说明 |
|------|------|------|
| node_modules | ~100 MB | 全部依赖 |
| dist（构建后） | ~500 KB | 生产版本 |
| Emoji Unicode | 0 KB | 浏览器原生 |
| Emoji SVG（全） | ~15 MB | 2000+ 图标 |
| Emoji SVG（精选） | ~2 MB | 200 图标 |
| 总计（默认） | ~500 KB | 仅 dist |
| 总计（+ SVG） | ~2.5 MB | dist + 精选 SVG |

---

## 🚀 快速部署脚本

创建一个自动化部署脚本：

```bash
#!/bin/bash
# deploy-offline.sh

echo "=== CrossWire 离线部署工具 ==="

# 检查环境
if [ ! -d "frontend" ]; then
  echo "❌ 错误：请在项目根目录运行此脚本"
  exit 1
fi

# 1. 安装依赖（如果需要）
if [ ! -d "frontend/node_modules" ]; then
  echo "📦 安装依赖..."
  cd frontend
  npm install
  cd ..
fi

# 2. 构建生产版本
echo "🔨 构建生产版本..."
cd frontend
npm run build
cd ..

# 3. 创建离线包
echo "📦 创建离线包..."
PACKAGE_NAME="crosswire-offline-$(date +%Y%m%d).tar.gz"

tar -czf "$PACKAGE_NAME" \
  frontend/dist \
  frontend/package.json \
  frontend/package-lock.json \
  frontend/vite.config.js \
  frontend/index.html

echo "✅ 离线包已创建: $PACKAGE_NAME"
echo ""
echo "📝 部署说明："
echo "1. 将 $PACKAGE_NAME 传输到离线环境"
echo "2. 解压: tar -xzf $PACKAGE_NAME"
echo "3. 启动: cd frontend && npm run dev"
echo "   或使用静态服务器: python3 -m http.server 5173 --directory dist"
echo ""
echo "✅ 完成！"
```

使用方法：

```bash
chmod +x deploy-offline.sh
./deploy-offline.sh
```

---

## 📝 部署检查表

部署前确认：

- [ ] ✅ npm install 已完成
- [ ] ✅ npm run build 成功
- [ ] ✅ dist 目录已生成
- [ ] ✅ 所有文件打包完整
- [ ] ✅ 传输到目标环境
- [ ] ✅ 解压成功
- [ ] ✅ 应用启动正常
- [ ] ✅ 所有功能测试通过

---

## 🔗 相关文档

- [README.md](./README.md) - 项目说明
- [DEMO_GUIDE.md](./DEMO_GUIDE.md) - 功能演示
- [ADVANCED_COMPONENTS.md](./ADVANCED_COMPONENTS.md) - 组件文档
- [INTEGRATION_GUIDE.md](./INTEGRATION_GUIDE.md) - 集成指南
- [public/emojis/README.md](./public/emojis/README.md) - Emoji 详细说明

---

## ✅ 总结

CrossWire 前端已完全支持离线部署：

1. **零外部依赖** - 所有资源已本地化
2. **小体积** - 构建后仅 ~500KB
3. **完整功能** - 所有高级功能正常工作
4. **易部署** - 一键打包，简单部署

**适用场景**：
- ✅ 离线 CTF 比赛
- ✅ 内网隔离环境
- ✅ 无网络访问的场景
- ✅ 安全要求严格的环境

---

**文档版本**: v1.0.0  
**更新时间**: 2025-10-05  
**维护者**: CrossWire Team
