<template>
  <div class="challenge-room">
    <a-layout style="height: 100%">
      <!-- 题目信息头部 -->
      <a-layout-header class="room-header">
        <div class="header-left">
          <a-button type="text" @click="$emit('back')">
            <ArrowLeftOutlined /> 返回
          </a-button>
          <a-divider type="vertical" />
          <h3 class="room-title">
            {{ challenge.title }} - 讨论室
          </h3>
        </div>

        <div class="header-right">
          <a-space>
            <a-tag :color="getCategoryColor(challenge.category)">
              {{ challenge.category }}
            </a-tag>
            <a-tag>
              {{ challenge.points }} 分
            </a-tag>
            <a-button type="primary" @click="showSubmitModal = true">
              <FlagOutlined /> 提交 Flag
            </a-button>
          </a-space>
        </div>
      </a-layout-header>

      <!-- 主内容区 -->
      <a-layout-content class="room-content">
        <!-- 消息列表 -->
        <div class="messages-area">
          <div class="messages-container">
            <div
              v-for="msg in roomMessages"
              :key="msg.id"
              class="message-item"
              :class="{ 'message-mine': msg.senderId === 'me' }"
            >
              <a-avatar
                v-if="msg.senderId !== 'me'"
                :style="{ backgroundColor: getAvatarColor(msg.senderName) }"
                class="message-avatar"
              >
                {{ msg.senderName[0] }}
              </a-avatar>

              <div class="message-content-wrapper">
                <div v-if="msg.senderId !== 'me'" class="message-header">
                  <span class="sender-name">{{ msg.senderName }}</span>
                  <span class="message-time">{{ formatTime(msg.timestamp) }}</span>
                </div>

                <div class="message-bubble" :class="{ 'bubble-mine': msg.senderId === 'me' }">
                  <!-- 文本消息 -->
                  <div v-if="msg.type === 'text'" class="message-text">
                    {{ msg.content }}
                  </div>

                  <!-- 代码消息 -->
                  <div v-else-if="msg.type === 'code'" class="message-code">
                    <div class="code-header">
                      <span class="code-lang">{{ msg.language }}</span>
                      <a-button type="text" size="small" @click="copyCode(msg.code)">
                        <CopyOutlined /> 复制
                      </a-button>
                    </div>
                    <pre><code>{{ msg.code }}</code></pre>
                  </div>

                  <!-- 文件消息 -->
                  <div v-else-if="msg.type === 'file'" class="message-file">
                    <FileOutlined />
                    <span class="file-name">{{ msg.fileName }}</span>
                    <a-button type="link" size="small">下载</a-button>
                  </div>
                </div>

                <div v-if="msg.senderId === 'me'" class="message-header message-header-right">
                  <span class="message-time">{{ formatTime(msg.timestamp) }}</span>
                </div>
              </div>

              <a-avatar
                v-if="msg.senderId === 'me'"
                class="message-avatar"
                style="background-color: #1890ff"
              >
                我
              </a-avatar>
            </div>
          </div>
        </div>

        <!-- 输入框 -->
        <div class="input-area">
          <div class="input-toolbar">
            <a-space>
              <a-tooltip title="发送文件">
                <a-button type="text" size="small">
                  <PaperClipOutlined />
                </a-button>
              </a-tooltip>
              <a-tooltip title="代码块">
                <a-button type="text" size="small" @click="showCodeEditor = !showCodeEditor">
                  <CodeOutlined />
                </a-button>
              </a-tooltip>
              <a-tooltip title="表情">
                <a-button type="text" size="small">
                  <SmileOutlined />
                </a-button>
              </a-tooltip>
            </a-space>
          </div>

          <div class="input-box">
            <a-textarea
              v-model:value="messageInput"
              placeholder="输入消息... (Ctrl+Enter 发送)"
              :auto-size="{ minRows: 2, maxRows: 4 }"
              @keydown="handleKeyDown"
            />
            <a-button
              type="primary"
              @click="sendMessage"
              :disabled="!messageInput.trim()"
            >
              <SendOutlined /> 发送
            </a-button>
          </div>

          <!-- 代码编辑器 -->
          <div v-if="showCodeEditor" class="code-editor-box">
            <a-divider style="margin: 12px 0">代码分享</a-divider>
            <a-select
              v-model:value="codeLanguage"
              style="width: 150px; margin-bottom: 8px"
              size="small"
            >
              <a-select-option value="python">Python</a-select-option>
              <a-select-option value="javascript">JavaScript</a-select-option>
              <a-select-option value="bash">Bash</a-select-option>
              <a-select-option value="c">C</a-select-option>
            </a-select>
            <a-textarea
              v-model:value="codeInput"
              placeholder="输入代码..."
              :auto-size="{ minRows: 4, maxRows: 10 }"
              style="font-family: monospace"
            />
            <div style="margin-top: 8px; text-align: right">
              <a-space>
                <a-button size="small" @click="showCodeEditor = false">取消</a-button>
                <a-button type="primary" size="small" @click="sendCode">发送代码</a-button>
              </a-space>
            </div>
          </div>
        </div>
      </a-layout-content>

      <!-- 右侧边栏 - 参与成员 -->
      <a-layout-sider width="240" theme="light" class="room-sider">
        <div class="sider-header">
          <h4>参与成员</h4>
        </div>
        <a-list :data-source="roomMembers" size="small" :split="false">
          <template #renderItem="{ item }">
            <a-list-item class="member-item">
              <a-list-item-meta>
                <template #avatar>
                  <a-badge :status="item.online ? 'success' : 'default'" :offset="[-5, 30]">
                    <a-avatar :style="{ backgroundColor: getAvatarColor(item.name) }">
                      {{ item.name[0] }}
                    </a-avatar>
                  </a-badge>
                </template>
                <template #title>
                  <div class="member-name">{{ item.name }}</div>
                </template>
                <template #description>
                  <div class="member-progress">
                    进度: {{ item.progress }}%
                  </div>
                </template>
              </a-list-item-meta>
            </a-list-item>
          </template>
        </a-list>
      </a-layout-sider>
    </a-layout>

    <!-- 提交 Flag 弹窗 -->
    <ChallengeSubmit
      v-model:open="showSubmitModal"
      :challenge="challenge"
      @submit="handleSubmitFlag"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import {
  ArrowLeftOutlined,
  FlagOutlined,
  PaperClipOutlined,
  CodeOutlined,
  SmileOutlined,
  SendOutlined,
  CopyOutlined,
  FileOutlined
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import ChallengeSubmit from './ChallengeSubmit.vue'

const props = defineProps({
  challenge: {
    type: Object,
    required: true
  }
})

defineEmits(['back'])

const messageInput = ref('')
const showCodeEditor = ref(false)
const codeLanguage = ref('python')
const codeInput = ref('')
const showSubmitModal = ref(false)

// 模拟聊天室消息
const roomMessages = ref([
  {
    id: '1',
    senderId: 'user1',
    senderName: 'alice',
    type: 'text',
    content: '我发现了一个 SQL 注入点',
    timestamp: new Date(Date.now() - 1800000)
  },
  {
    id: '2',
    senderId: 'user2',
    senderName: 'bob',
    type: 'code',
    language: 'python',
    code: 'import requests\nurl = "http://target.com"\npayload = {"id": "1\' OR \'1\'=\'1"}',
    timestamp: new Date(Date.now() - 1200000)
  },
  {
    id: '3',
    senderId: 'user1',
    senderName: 'alice',
    type: 'text',
    content: '试试这个 payload，已经绕过了 WAF',
    timestamp: new Date(Date.now() - 600000)
  }
])

// 参与成员
const roomMembers = ref([
  { id: 'user1', name: 'alice', online: true, progress: 80 },
  { id: 'user2', name: 'bob', online: true, progress: 60 },
  { id: 'user3', name: 'charlie', online: false, progress: 30 }
])

const getCategoryColor = (category) => {
  const colors = {
    Web: 'blue',
    Pwn: 'red',
    Reverse: 'purple',
    Crypto: 'orange',
    Misc: 'green'
  }
  return colors[category] || 'default'
}

const getAvatarColor = (name) => {
  const colors = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1']
  const index = name.charCodeAt(0) % colors.length
  return colors[index]
}

const formatTime = (timestamp) => {
  return dayjs(timestamp).format('HH:mm')
}

const sendMessage = () => {
  if (!messageInput.value.trim()) return

  roomMessages.value.push({
    id: Date.now().toString(),
    senderId: 'me',
    senderName: '我',
    type: 'text',
    content: messageInput.value,
    timestamp: new Date()
  })

  messageInput.value = ''
}

const sendCode = () => {
  if (!codeInput.value.trim()) return

  roomMessages.value.push({
    id: Date.now().toString(),
    senderId: 'me',
    senderName: '我',
    type: 'code',
    language: codeLanguage.value,
    code: codeInput.value,
    timestamp: new Date()
  })

  codeInput.value = ''
  showCodeEditor.value = false
}

const handleKeyDown = (e) => {
  if (e.ctrlKey && e.key === 'Enter') {
    e.preventDefault()
    sendMessage()
  }
}

const copyCode = async (code) => {
  try {
    await navigator.clipboard.writeText(code)
    message.success('代码已复制')
  } catch (error) {
    message.error('复制失败')
  }
}

const handleSubmitFlag = (data) => {
  message.success('Flag 提交成功')
}
</script>

<style scoped>
.challenge-room {
  width: 100%;
  height: 600px;
}

.room-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: white;
  border-bottom: 1px solid #f0f0f0;
  padding: 0 24px;
  height: 64px;
}

.header-left {
  display: flex;
  align-items: center;
}

.room-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.room-content {
  background: white;
  display: flex;
  flex-direction: column;
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.message-item {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.message-item.message-mine {
  flex-direction: row-reverse;
}

.message-avatar {
  flex-shrink: 0;
}

.message-content-wrapper {
  flex: 1;
  max-width: 70%;
}

.message-header {
  display: flex;
  gap: 8px;
  margin-bottom: 4px;
  font-size: 13px;
}

.message-header-right {
  justify-content: flex-end;
}

.sender-name {
  font-weight: 500;
  color: rgba(0, 0, 0, 0.85);
}

.message-time {
  color: rgba(0, 0, 0, 0.45);
}

.message-bubble {
  display: inline-block;
  padding: 10px 14px;
  background-color: #f5f5f5;
  border-radius: 4px;
  word-wrap: break-word;
}

.bubble-mine {
  background-color: #1890ff;
}

.message-text {
  font-size: 14px;
  line-height: 1.6;
  color: rgba(0, 0, 0, 0.85);
}

.bubble-mine .message-text {
  color: white;
}

.message-code {
  min-width: 300px;
}

.code-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.code-lang {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.65);
}

.message-code pre {
  background-color: #fafafa;
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
  margin: 0;
}

.message-code code {
  font-family: 'Consolas', monospace;
  font-size: 13px;
}

.message-file {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-name {
  font-size: 14px;
}

.input-area {
  border-top: 1px solid #f0f0f0;
  padding: 16px;
  background-color: #fafafa;
}

.input-toolbar {
  margin-bottom: 12px;
}

.input-box {
  display: flex;
  gap: 8px;
}

.room-sider {
  background: white;
  border-left: 1px solid #f0f0f0;
}

.sider-header {
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.sider-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.member-item {
  padding: 12px 16px;
}

.member-name {
  font-size: 14px;
  font-weight: 500;
}

.member-progress {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.code-editor-box {
  margin-top: 12px;
}
</style>

