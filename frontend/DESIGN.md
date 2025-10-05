# CrossWire 前端设计规范

## 设计原则

遵循 **Ant Design** 设计规范，采用简洁专业的黑白灰配色系统。

## 配色方案

### 主色调

```css
/* 品牌主色 */
--primary-color: #1890ff;      /* Ant Design 蓝 */

/* 功能色 */
--success-color: #52c41a;      /* 成功/绿色 */
--warning-color: #faad14;      /* 警告/橙色 */
--error-color: #ff4d4f;        /* 错误/红色 */
--info-color: #1890ff;         /* 信息/蓝色 */
```

### 中性色

```css
/* 文本颜色 */
--text-primary: rgba(0, 0, 0, 0.85);    /* 主要文本 */
--text-secondary: rgba(0, 0, 0, 0.65);  /* 次要文本 */
--text-tertiary: rgba(0, 0, 0, 0.45);   /* 三级文本 */
--text-disabled: rgba(0, 0, 0, 0.25);   /* 禁用文本 */

/* 背景色 */
--bg-primary: #ffffff;         /* 主背景（白色） */
--bg-secondary: #fafafa;       /* 次级背景 */
--bg-tertiary: #f5f5f5;        /* 三级背景（页面背景） */
--bg-quaternary: #f0f0f0;      /* 四级背景 */

/* 边框色 */
--border-color-base: #d9d9d9;  /* 基础边框 */
--border-color-split: #f0f0f0; /* 分割线 */
```

## 布局规范

### 页面结构

```
┌─────────────────────────────────────┐
│  页面容器                           │
│  background: #f5f5f5               │
│  ┌───────────────────────────────┐ │
│  │  内容卡片                      │ │
│  │  background: white            │ │
│  │  border-radius: 8px           │ │
│  │  box-shadow: subtle           │ │
│  └───────────────────────────────┘ │
└─────────────────────────────────────┘
```

### 间距系统

```css
--spacing-xs: 4px;     /* 极小间距 */
--spacing-sm: 8px;     /* 小间距 */
--spacing-md: 16px;    /* 中等间距（常用） */
--spacing-lg: 24px;    /* 大间距 */
--spacing-xl: 32px;    /* 超大间距 */
--spacing-xxl: 48px;   /* 特大间距 */
```

### 圆角

```css
--border-radius-base: 2px;  /* 基础圆角（Ant Design 标准） */
--border-radius-sm: 2px;    /* 小圆角 */
--border-radius-lg: 4px;    /* 大圆角 */
```

## 组件样式

### 卡片 (Card)

```css
.card {
  background: white;
  border: 1px solid #d9d9d9;
  border-radius: 2px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.card:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 16px rgba(24, 144, 255, 0.15);
  transform: translateY(-4px);
}
```

### 按钮

- **主按钮**: `type="primary"` - 蓝色背景
- **次要按钮**: `type="default"` - 白色背景，灰色边框
- **文本按钮**: `type="link"` - 无边框，文字颜色
- **危险按钮**: `danger` - 红色系

### 输入框

```css
.input:focus {
  border-color: #40a9ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}
```

## 页面设计

### 1. 首页 (HomeView)

**布局特点**：
- 居中白色卡片容器
- 灰色页面背景 `#f5f5f5`
- 简洁的图标 + 文字布局
- 鼠标悬停微交互效果

**配色**：
- 背景：`#f5f5f5`
- 卡片：白色
- 图标：`#1890ff`
- 文字：标准黑白灰

### 2. 服务端配置页 (ServerView)

**布局特点**：
- 表单式布局
- 顶部对齐，可滚动
- 提示卡片在底部

**配色**：
- 背景：`#f5f5f5`
- 卡片：白色
- 表单元素：Ant Design 标准

### 3. 客户端加入页 (ClientView)

**布局特点**：
- Tab 切换式布局
- 扫描/手动输入/二维码三种方式
- Modal 表单收集用户信息

**配色**：
- 背景：`#f5f5f5`
- 卡片：白色
- 列表项悬停：`#fafafa`

### 4. 聊天界面 (ChatView)

**布局特点**：
- 三栏布局：侧边栏 + 主内容 + 抽屉
- 消息气泡式设计
- 固定输入框在底部

**配色**：
- 背景：`#f0f2f5`
- 侧边栏：白色
- 消息气泡（我）：`#1890ff`
- 消息气泡（他人）：`#f5f5f5`

## 图标规范

使用 **Ant Design Icons**：

```javascript
import {
  CloudServerOutlined,    // 服务端
  TeamOutlined,           // 团队/客户端
  ThunderboltOutlined,    // 快速/ARP
  SafetyOutlined,         // 安全/加密
  FileProtectOutlined,    // 文件保护
  CodeOutlined,           // 代码
  // ... 更多图标
} from '@ant-design/icons-vue'
```

## 字体规范

### 字体族

```css
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 
             Roboto, 'Helvetica Neue', Arial, 'Noto Sans', 
             sans-serif;
```

### 字号

```css
--font-size-sm: 12px;    /* 小字 */
--font-size-base: 14px;  /* 正文（基准） */
--font-size-lg: 16px;    /* 副标题 */
--font-size-xl: 20px;    /* 标题 */
```

### 标题

```css
h1 { font-size: 38px; font-weight: 600; }
h2 { font-size: 30px; font-weight: 600; }
h3 { font-size: 24px; font-weight: 600; }
h4 { font-size: 20px; font-weight: 600; }
```

## 阴影系统

```css
/* 微阴影 - 卡片默认 */
box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);

/* 中等阴影 - 悬停 */
box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);

/* 强阴影 - Modal/Drawer */
box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
```

## 动画效果

### 过渡时间

```css
transition: all 0.3s ease;  /* 标准过渡 */
```

### 常用动画

```css
/* 悬停上浮 */
.card:hover {
  transform: translateY(-4px);
}

/* 悬停放大 */
.icon:hover {
  transform: scale(1.1);
}
```

## 响应式设计

使用 Ant Design Grid 系统：

```vue
<a-row :gutter="[16, 16]">
  <a-col :xs="24" :sm="12" :md="8" :lg="6">
    <!-- 内容 -->
  </a-col>
</a-row>
```

## 无障碍设计

- 合理的颜色对比度
- 语义化的 HTML 标签
- 键盘导航支持
- ARIA 标签

## 设计资源

- [Ant Design 官方文档](https://ant.design/)
- [Ant Design Vue 文档](https://antdv.com/)
- [Ant Design 设计价值观](https://ant.design/docs/spec/values-cn)
- [Ant Design 色板](https://ant.design/docs/spec/colors-cn)

## 开发建议

1. **保持一致性**：遵循 Ant Design 组件的默认样式
2. **避免过度设计**：简洁专业为主
3. **合理留白**：使用标准间距系统
4. **语义化命名**：CSS 类名清晰易懂
5. **组件复用**：提取公共组件

## 注意事项

❌ **不要做**：
- 不使用花哨的渐变背景
- 不使用过多的颜色
- 不自定义组件样式覆盖太多
- 不使用非标准的间距

✅ **推荐做**：
- 使用 Ant Design 标准组件
- 保持黑白灰为主色调
- 使用标准间距和圆角
- 遵循设计规范文档

