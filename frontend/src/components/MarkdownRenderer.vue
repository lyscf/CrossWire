<template>
  <div class="markdown-renderer">
    <div v-if="editable" class="markdown-editor">
      <a-tabs v-model:activeKey="activeTab" size="small">
        <a-tab-pane key="edit" tab="编辑">
          <a-textarea
            v-model:value="localContent"
            :rows="rows"
            placeholder="支持 Markdown 语法..."
            class="markdown-textarea"
          />
        </a-tab-pane>
        <a-tab-pane key="preview" tab="预览">
          <div class="markdown-content" v-html="renderedHtml"></div>
        </a-tab-pane>
      </a-tabs>

      <div class="markdown-toolbar">
        <a-space>
          <a-button type="text" size="small" title="粗体" @click="insertFormat('**', '**')">
            <BoldOutlined />
          </a-button>
          <a-button type="text" size="small" title="斜体" @click="insertFormat('*', '*')">
            <ItalicOutlined />
          </a-button>
          <a-button type="text" size="small" title="删除线" @click="insertFormat('~~', '~~')">
            <StrikethroughOutlined />
          </a-button>
          <a-divider type="vertical" />
          <a-button type="text" size="small" title="标题" @click="insertFormat('### ', '')">
            <FontSizeOutlined />
          </a-button>
          <a-button type="text" size="small" title="引用" @click="insertFormat('> ', '')">
            <span style="font-weight: bold">&gt;</span>
          </a-button>
          <a-button type="text" size="small" title="代码" @click="insertFormat('`', '`')">
            <CodeOutlined />
          </a-button>
          <a-button type="text" size="small" title="代码块" @click="insertCodeBlock">
            <FileCodeOutlined />
          </a-button>
          <a-divider type="vertical" />
          <a-button type="text" size="small" title="链接" @click="insertLink">
            <LinkOutlined />
          </a-button>
          <a-button type="text" size="small" title="图片" @click="insertImage">
            <PictureOutlined />
          </a-button>
          <a-button type="text" size="small" title="列表" @click="insertFormat('- ', '')">
            <UnorderedListOutlined />
          </a-button>
          <a-button type="text" size="small" title="有序列表" @click="insertFormat('1. ', '')">
            <OrderedListOutlined />
          </a-button>
        </a-space>

        <a-space>
          <span class="word-count">{{ wordCount }} 字</span>
          <a-button type="link" size="small" @click="togglePreview">
            <EyeOutlined /> {{ showPreview ? '隐藏' : '显示' }}预览
          </a-button>
        </a-space>
      </div>
    </div>

    <div v-else class="markdown-content" v-html="renderedHtml"></div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import {
  BoldOutlined,
  ItalicOutlined,
  StrikethroughOutlined,
  FontSizeOutlined,
  CodeOutlined,
  FileCodeOutlined,
  LinkOutlined,
  PictureOutlined,
  UnorderedListOutlined,
  OrderedListOutlined,
  EyeOutlined
} from '@ant-design/icons-vue'

const props = defineProps({
  content: {
    type: String,
    default: ''
  },
  editable: {
    type: Boolean,
    default: false
  },
  rows: {
    type: Number,
    default: 10
  }
})

const emit = defineEmits(['update:content'])

const localContent = ref(props.content)
const activeTab = ref('edit')
const showPreview = ref(true)

watch(() => props.content, (newVal) => {
  localContent.value = newVal
})

watch(localContent, (newVal) => {
  emit('update:content', newVal)
})

const wordCount = computed(() => {
  return localContent.value.length
})

// 简单的 Markdown 渲染（实际项目中应使用 markdown-it 库）
const renderedHtml = computed(() => {
  let html = localContent.value

  // 转义 HTML
  html = html
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  // 标题
  html = html.replace(/^### (.*$)/gim, '<h3>$1</h3>')
  html = html.replace(/^## (.*$)/gim, '<h2>$1</h2>')
  html = html.replace(/^# (.*$)/gim, '<h1>$1</h1>')

  // 粗体
  html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')

  // 斜体
  html = html.replace(/\*(.*?)\*/g, '<em>$1</em>')

  // 删除线
  html = html.replace(/~~(.*?)~~/g, '<del>$1</del>')

  // 行内代码
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>')

  // 代码块
  html = html.replace(/```(\w+)?\n([\s\S]*?)```/g, (match, lang, code) => {
    return `<pre><code class="language-${lang || 'text'}">${code.trim()}</code></pre>`
  })

  // 链接
  html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank">$1</a>')

  // 图片
  html = html.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1" />')

  // 引用
  html = html.replace(/^&gt; (.*$)/gim, '<blockquote>$1</blockquote>')

  // 无序列表
  html = html.replace(/^\- (.*$)/gim, '<li>$1</li>')
  html = html.replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')

  // 有序列表
  html = html.replace(/^\d+\. (.*$)/gim, '<li>$1</li>')

  // 段落
  html = html.replace(/\n\n/g, '</p><p>')
  html = '<p>' + html + '</p>'

  // 换行
  html = html.replace(/\n/g, '<br>')

  return html
})

const insertFormat = (before, after) => {
  const textarea = document.querySelector('.markdown-textarea textarea')
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const text = localContent.value
  const selectedText = text.substring(start, end)

  localContent.value =
    text.substring(0, start) +
    before +
    selectedText +
    after +
    text.substring(end)

  setTimeout(() => {
    textarea.focus()
    textarea.setSelectionRange(
      start + before.length,
      end + before.length
    )
  }, 0)
}

const insertCodeBlock = () => {
  insertFormat('\n```\n', '\n```\n')
}

const insertLink = () => {
  const url = prompt('请输入链接地址：')
  if (url) {
    insertFormat('[链接文本](', `${url})`)
  }
}

const insertImage = () => {
  const url = prompt('请输入图片地址：')
  if (url) {
    insertFormat('![图片描述](', `${url})`)
  }
}

const togglePreview = () => {
  showPreview.value = !showPreview.value
  activeTab.value = showPreview.value ? 'preview' : 'edit'
}
</script>

<style scoped>
.markdown-renderer {
  width: 100%;
}

.markdown-editor {
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  overflow: hidden;
}

.markdown-textarea {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
}

.markdown-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  border-top: 1px solid #f0f0f0;
  background-color: #fafafa;
}

.word-count {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.markdown-content {
  padding: 16px;
  line-height: 1.8;
  color: rgba(0, 0, 0, 0.85);
}

.markdown-content :deep(h1) {
  font-size: 28px;
  font-weight: 600;
  margin: 24px 0 16px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
}

.markdown-content :deep(h2) {
  font-size: 24px;
  font-weight: 600;
  margin: 20px 0 12px 0;
  padding-bottom: 6px;
  border-bottom: 1px solid #f0f0f0;
}

.markdown-content :deep(h3) {
  font-size: 20px;
  font-weight: 600;
  margin: 16px 0 8px 0;
}

.markdown-content :deep(p) {
  margin: 12px 0;
}

.markdown-content :deep(code) {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 0.9em;
  background-color: rgba(150, 150, 150, 0.1);
  padding: 2px 6px;
  border-radius: 2px;
  color: #c41d7f;
}

.markdown-content :deep(pre) {
  background-color: #f6f8fa;
  border: 1px solid #e1e4e8;
  border-radius: 4px;
  padding: 16px;
  overflow-x: auto;
  margin: 16px 0;
}

.markdown-content :deep(pre code) {
  background-color: transparent;
  padding: 0;
  color: inherit;
  font-size: 13px;
}

.markdown-content :deep(blockquote) {
  border-left: 4px solid #d9d9d9;
  padding-left: 16px;
  margin: 16px 0;
  color: rgba(0, 0, 0, 0.65);
}

.markdown-content :deep(ul),
.markdown-content :deep(ol) {
  padding-left: 24px;
  margin: 12px 0;
}

.markdown-content :deep(li) {
  margin: 6px 0;
}

.markdown-content :deep(a) {
  color: #1890ff;
  text-decoration: none;
}

.markdown-content :deep(a:hover) {
  text-decoration: underline;
}

.markdown-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 12px 0;
}

.markdown-content :deep(strong) {
  font-weight: 600;
}

.markdown-content :deep(em) {
  font-style: italic;
}

.markdown-content :deep(del) {
  text-decoration: line-through;
}
</style>

