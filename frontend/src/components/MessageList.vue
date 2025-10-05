<template>
  <div class="message-list" ref="messageListRef">
    <div class="messages-container">
      <div
        v-for="message in messages"
        :key="message.id"
        class="message-wrapper"
      >
        <!-- 系统消息 -->
        <div v-if="message.type === 'system'" class="system-message">
          <a-divider>
            <span class="system-text">
              <InfoCircleOutlined /> {{ message.content }}
            </span>
          </a-divider>
        </div>

        <!-- 普通消息 -->
        <div
          v-else
          class="message-item"
          :class="{ 'message-mine': message.senderId === 'me' }"
        >
          <a-avatar
            v-if="message.senderId !== 'me'"
            :src="message.senderAvatar"
            class="message-avatar"
          >
            {{ message.senderName?.[0] }}
          </a-avatar>

          <div class="message-content-wrapper">
            <div v-if="message.senderId !== 'me'" class="message-header">
              <span class="sender-name">{{ message.senderName }}</span>
              <span class="message-time">{{ formatTime(message.timestamp) }}</span>
            </div>

            <div class="message-bubble">
              <div class="message-text" v-html="renderMessageWithMentions(message.content)"></div>
            </div>

            <div v-if="message.senderId === 'me'" class="message-header message-header-right">
              <span class="message-time">{{ formatTime(message.timestamp) }}</span>
            </div>
          </div>

          <a-avatar
            v-if="message.senderId === 'me'"
            class="message-avatar"
          >
            我
          </a-avatar>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick, computed } from 'vue'
import { InfoCircleOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { useMemberStore } from '@/stores/member'
import { useAppStore } from '@/stores/app'

const memberStore = useMemberStore()
const appStore = useAppStore()
const currentUsername = computed(() => appStore.currentUser?.username || 'me')

const props = defineProps({
  messages: {
    type: Array,
    default: () => []
  }
})

const messageListRef = ref(null)

const formatTime = (timestamp) => {
  const now = dayjs()
  const msgTime = dayjs(timestamp)
  
  if (now.diff(msgTime, 'day') === 0) {
    return msgTime.format('HH:mm')
  } else if (now.diff(msgTime, 'day') === 1) {
    return '昨天 ' + msgTime.format('HH:mm')
  } else {
    return msgTime.format('MM-DD HH:mm')
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messageListRef.value) {
      messageListRef.value.scrollTop = messageListRef.value.scrollHeight
    }
  })
}

watch(
  () => props.messages.length,
  () => {
    scrollToBottom()
  }
)

// 渲染带有 @提及 的消息
const renderMessageWithMentions = (content) => {
  if (!content) return ''
  
  // 转义 HTML 特殊字符
  const escapeHtml = (text) => {
    const div = document.createElement('div')
    div.textContent = text
    return div.innerHTML
  }
  
  // 替换 @用户名 为高亮样式
  const regex = /@(\w+)/g
  let result = escapeHtml(content)
  
  result = result.replace(regex, (match, username) => {
    // 检查是否是有效的成员
    const member = memberStore.members.find(m => m.name === username)
    const isMentioned = username === currentUsername.value
    
    if (member) {
      return `<span class="mention ${isMentioned ? 'mention-me' : ''}" title="@${username}">@${username}</span>`
    }
    return match
  })
  
  return result
}
</script>

<style scoped>
.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px;
}

.messages-container {
  max-width: 900px;
  margin: 0 auto;
}

.message-wrapper {
  margin-bottom: 20px;
}

.system-message {
  text-align: center;
}

.system-text {
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
}

.message-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.message-item.message-mine {
  flex-direction: row-reverse;
}

.message-avatar {
  flex-shrink: 0;
  background-color: #1890ff;
}

.message-content-wrapper {
  flex: 1;
  max-width: 60%;
}

.message-mine .message-content-wrapper {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.message-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.message-header-right {
  justify-content: flex-end;
}

.sender-name {
  font-size: 13px;
  font-weight: 500;
  color: rgba(0, 0, 0, 0.85);
}

.message-time {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.message-bubble {
  display: inline-block;
  padding: 10px 14px;
  border-radius: 4px;
  background-color: #f5f5f5;
  word-wrap: break-word;
  word-break: break-word;
}

.message-mine .message-bubble {
  background-color: #1890ff;
}

.message-text {
  font-size: 14px;
  line-height: 1.6;
  color: rgba(0, 0, 0, 0.85);
}

.message-mine .message-text {
  color: white;
}

/* @提及样式 */
.message-text :deep(.mention) {
  color: #1890ff;
  background-color: rgba(24, 144, 255, 0.1);
  padding: 2px 4px;
  border-radius: 2px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.message-text :deep(.mention:hover) {
  background-color: rgba(24, 144, 255, 0.2);
}

.message-text :deep(.mention-me) {
  color: #f5222d;
  background-color: rgba(245, 34, 45, 0.1);
}

.message-text :deep(.mention-me:hover) {
  background-color: rgba(245, 34, 45, 0.2);
}

/* 自己发送的消息中的 @提及样式 */
.message-mine .message-text :deep(.mention) {
  color: white;
  background-color: rgba(255, 255, 255, 0.2);
}

.message-mine .message-text :deep(.mention:hover) {
  background-color: rgba(255, 255, 255, 0.3);
}

.message-mine .message-text :deep(.mention-me) {
  background-color: rgba(255, 255, 255, 0.3);
}
</style>

