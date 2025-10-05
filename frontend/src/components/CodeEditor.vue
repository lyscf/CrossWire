<template>
  <div class="code-editor">
    <div class="editor-header">
      <a-space>
        <a-select
          v-model:value="selectedLanguage"
          style="width: 150px"
          size="small"
        >
          <a-select-option value="javascript">JavaScript</a-select-option>
          <a-select-option value="python">Python</a-select-option>
          <a-select-option value="go">Go</a-select-option>
          <a-select-option value="c">C</a-select-option>
          <a-select-option value="cpp">C++</a-select-option>
          <a-select-option value="java">Java</a-select-option>
          <a-select-option value="php">PHP</a-select-option>
          <a-select-option value="bash">Bash</a-select-option>
        </a-select>
        
        <a-input
          v-model:value="fileName"
          placeholder="文件名"
          style="width: 200px"
          size="small"
        />
      </a-space>

      <a-space>
        <a-button size="small" @click="handleCopy">
          <CopyOutlined /> 复制
        </a-button>
        <a-button size="small" type="primary" @click="handleSend">
          <SendOutlined /> 发送
        </a-button>
      </a-space>
    </div>

    <div class="editor-content">
      <a-textarea
        v-model:value="code"
        :placeholder="`输入 ${selectedLanguage} 代码...`"
        :auto-size="{ minRows: 10, maxRows: 30 }"
        class="code-textarea"
      />
    </div>

    <div class="editor-footer">
      <span class="line-count">{{ lineCount }} 行</span>
      <span class="char-count">{{ code.length }} 字符</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { message } from 'ant-design-vue'
import {
  CopyOutlined,
  SendOutlined
} from '@ant-design/icons-vue'

const emit = defineEmits(['send'])

const code = ref('')
const selectedLanguage = ref('python')
const fileName = ref('script.py')

const lineCount = computed(() => {
  return code.value.split('\n').length
})

const handleCopy = async () => {
  try {
    await navigator.clipboard.writeText(code.value)
    message.success('代码已复制到剪贴板')
  } catch (error) {
    message.error('复制失败')
  }
}

const handleSend = () => {
  if (!code.value.trim()) {
    message.warning('请输入代码')
    return
  }

  emit('send', {
    type: 'code',
    language: selectedLanguage.value,
    filename: fileName.value,
    code: code.value
  })

  // 清空编辑器
  code.value = ''
  message.success('代码已发送')
}
</script>

<style scoped>
.code-editor {
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: white;
}

.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
  background-color: #fafafa;
}

.editor-content {
  padding: 12px;
}

.code-textarea {
  font-family: 'SFMono-Regular', 'Consolas', 'Liberation Mono', 'Menlo', 'Courier', monospace;
  font-size: 13px;
  line-height: 1.6;
  border: none;
  resize: none;
}

.code-textarea:focus {
  box-shadow: none;
}

.editor-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
  padding: 8px 12px;
  border-top: 1px solid #f0f0f0;
  background-color: #fafafa;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}
</style>

