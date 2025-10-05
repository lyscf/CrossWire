<template>
  <div class="message-input" ref="messageInputRef">
    <div class="input-toolbar">
      <a-space>
        <a-tooltip title="发送文件">
          <a-upload
            :show-upload-list="false"
            :before-upload="handleFileUpload"
          >
            <a-button type="text" :icon="h(PaperClipOutlined)" />
          </a-upload>
        </a-tooltip>

        <a-tooltip title="代码块">
          <a-button
            type="text"
            :icon="h(CodeOutlined)"
            @click="insertCodeBlock"
          />
        </a-tooltip>

        <a-tooltip title="表情">
          <a-button type="text" :icon="h(SmileOutlined)" />
        </a-tooltip>

        <a-tooltip title="@提及 (输入 @ 触发)">
          <a-button
            type="text"
            :icon="h(UserOutlined)"
            @click="insertMention"
          />
        </a-tooltip>
      </a-space>
    </div>

    <div class="input-area">
      <a-textarea
        ref="textareaRef"
        v-model:value="messageContent"
        :auto-size="{ minRows: 1, maxRows: 5 }"
        placeholder="输入消息... (Ctrl+Enter 发送，输入 @ 提及成员)"
        :maxlength="5000"
        @keydown="handleKeyDown"
        @input="handleInput"
        class="message-textarea"
      />

      <div class="input-actions">
        <span class="char-count">{{ messageContent.length }}/5000</span>
        <a-button
          type="primary"
          :disabled="!messageContent.trim()"
          @click="handleSend"
        >
          <SendOutlined /> 发送
        </a-button>
      </div>
    </div>

    <!-- @提及选择器 -->
    <MentionSelector
      :visible="showMentionSelector"
      :search-text="mentionSearchText"
      :members="members"
      :position="mentionPosition"
      @select="handleMentionSelect"
      @cancel="closeMentionSelector"
    />
  </div>
</template>

<script setup>
import { ref, h, computed } from 'vue'
import { message } from 'ant-design-vue'
import {
  PaperClipOutlined,
  CodeOutlined,
  SmileOutlined,
  UserOutlined,
  SendOutlined
} from '@ant-design/icons-vue'
import MentionSelector from './MentionSelector.vue'
import { useMemberStore } from '@/stores/member'

const memberStore = useMemberStore()
const members = computed(() => memberStore.members)

const emit = defineEmits(['send'])

const messageInputRef = ref(null)
const textareaRef = ref(null)
const messageContent = ref('')

// @提及相关状态
const showMentionSelector = ref(false)
const mentionSearchText = ref('')
const mentionStartPos = ref(-1)
const mentionPosition = ref({ left: '0px', bottom: '0px' })

const handleSend = () => {
  if (!messageContent.value.trim()) {
    return
  }

  // 解析 @提及
  const mentions = parseMentions(messageContent.value)
  
  emit('send', {
    content: messageContent.value,
    mentions: mentions
  })
  
  messageContent.value = ''
  closeMentionSelector()
}

// 解析消息中的 @提及
const parseMentions = (text) => {
  const mentions = []
  const regex = /@(\w+)/g
  let match
  
  while ((match = regex.exec(text)) !== null) {
    const username = match[1]
    const member = members.value.find(m => m.name === username)
    if (member) {
      mentions.push({
        username,
        userId: member.id,
        position: match.index
      })
    }
  }
  
  return mentions
}

// 处理输入事件，检测 @ 符号
const handleInput = (e) => {
  const textarea = e.target
  const cursorPos = textarea.selectionStart
  const text = messageContent.value
  
  // 查找最近的 @ 符号
  let atPos = -1
  for (let i = cursorPos - 1; i >= 0; i--) {
    if (text[i] === '@') {
      // 检查 @ 前面是否是空格或开头
      if (i === 0 || text[i - 1] === ' ' || text[i - 1] === '\n') {
        atPos = i
        break
      }
    } else if (text[i] === ' ' || text[i] === '\n') {
      // 遇到空格或换行，停止搜索
      break
    }
  }
  
  if (atPos !== -1) {
    // 找到了 @，显示选择器
    mentionStartPos.value = atPos
    mentionSearchText.value = text.substring(atPos + 1, cursorPos)
    showMentionSelector.value = true
    updateMentionPosition(textarea)
  } else {
    // 没有找到 @，隐藏选择器
    closeMentionSelector()
  }
}

// 更新选择器位置
const updateMentionPosition = (textarea) => {
  if (!textarea) return
  
  const rect = textarea.getBoundingClientRect()
  
  // 简单定位：在输入框上方
  mentionPosition.value = {
    left: `${rect.left}px`,
    bottom: `${window.innerHeight - rect.top + 8}px`
  }
}

// 处理选中成员
const handleMentionSelect = (member) => {
  const textarea = textareaRef.value.resizableTextArea.textArea
  const before = messageContent.value.substring(0, mentionStartPos.value)
  const after = messageContent.value.substring(textarea.selectionStart)
  
  // 插入 @用户名 + 空格
  messageContent.value = before + `@${member.name} ` + after
  
  // 设置光标位置到插入内容后面
  setTimeout(() => {
    const newPos = before.length + member.name.length + 2
    textarea.setSelectionRange(newPos, newPos)
    textarea.focus()
  }, 0)
  
  closeMentionSelector()
}

// 关闭选择器
const closeMentionSelector = () => {
  showMentionSelector.value = false
  mentionSearchText.value = ''
  mentionStartPos.value = -1
}

const handleKeyDown = (e) => {
  // 如果选择器打开，某些按键由选择器处理
  if (showMentionSelector.value) {
    if (['ArrowDown', 'ArrowUp', 'Enter', 'Tab'].includes(e.key)) {
      // 这些按键由 MentionSelector 处理
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      closeMentionSelector()
      return
    }
  }
  
  // Ctrl+Enter 发送
  if (e.ctrlKey && e.key === 'Enter') {
    e.preventDefault()
    handleSend()
  }
}

const insertCodeBlock = () => {
  const codeTemplate = '\n```\n// 在此输入代码\n```\n'
  messageContent.value += codeTemplate
}

const insertMention = () => {
  const textarea = textareaRef.value.resizableTextArea.textArea
  const cursorPos = textarea.selectionStart
  
  // 在光标位置插入 @
  const before = messageContent.value.substring(0, cursorPos)
  const after = messageContent.value.substring(cursorPos)
  
  // 确保 @ 前面有空格（除非是开头）
  const needSpace = before.length > 0 && before[before.length - 1] !== ' ' && before[before.length - 1] !== '\n'
  messageContent.value = before + (needSpace ? ' @' : '@') + after
  
  // 设置光标位置并触发输入事件
  setTimeout(() => {
    const newPos = cursorPos + (needSpace ? 2 : 1)
    textarea.setSelectionRange(newPos, newPos)
    textarea.focus()
    handleInput({ target: textarea })
  }, 0)
}

const handleFileUpload = (file) => {
  console.log('Uploading file:', file.name)
  message.info(`正在上传文件: ${file.name}`)
  // TODO: 实现文件上传
  return false // 阻止自动上传
}
</script>

<style scoped>
.message-input {
  width: 100%;
}

.input-toolbar {
  margin-bottom: 12px;
}

.input-area {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.message-textarea {
  border-radius: 2px;
  font-size: 14px;
}

.input-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.char-count {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}
</style>

