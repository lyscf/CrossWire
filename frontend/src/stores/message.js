import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useMessageStore = defineStore('message', () => {
  // 消息列表 { channelId: [messages] }
  const messagesByChannel = ref({
    main: []
  })

  // 置顶消息 { channelId: [messages] }
  const pinnedMessagesByChannel = ref({
    main: []
  })

  // 当前正在编辑的消息
  const editingMessage = ref(null)

  // Getters
  const getChannelMessages = computed(() => (channelId) => {
    return messagesByChannel.value[channelId] || []
  })

  const getPinnedMessages = computed(() => (channelId) => {
    return pinnedMessagesByChannel.value[channelId] || []
  })

  // Actions
  const addMessage = (channelId, message) => {
    if (!messagesByChannel.value[channelId]) {
      messagesByChannel.value[channelId] = []
    }
    
    // 检查是否已存在（防止重复）
    const exists = messagesByChannel.value[channelId].find(m => m.id === message.id)
    if (!exists) {
      messagesByChannel.value[channelId].push(message)
    }
  }

  const addMessages = (channelId, messages) => {
    if (!messagesByChannel.value[channelId]) {
      messagesByChannel.value[channelId] = []
    }
    
    messages.forEach(message => {
      const exists = messagesByChannel.value[channelId].find(m => m.id === message.id)
      if (!exists) {
        messagesByChannel.value[channelId].push(message)
      }
    })
    
    // 按时间排序
    messagesByChannel.value[channelId].sort((a, b) => 
      new Date(a.timestamp) - new Date(b.timestamp)
    )
  }

  const updateMessage = (channelId, messageId, updates) => {
    const messages = messagesByChannel.value[channelId]
    if (messages) {
      const index = messages.findIndex(m => m.id === messageId)
      if (index !== -1) {
        messages[index] = { ...messages[index], ...updates }
      }
    }
  }

  const deleteMessage = (channelId, messageId) => {
    const messages = messagesByChannel.value[channelId]
    if (messages) {
      const index = messages.findIndex(m => m.id === messageId)
      if (index !== -1) {
        messages[index].deleted = true
        messages[index].content = '此消息已被删除'
      }
    }
  }

  const pinMessage = (channelId, message) => {
    if (!pinnedMessagesByChannel.value[channelId]) {
      pinnedMessagesByChannel.value[channelId] = []
    }
    
    // 最多 5 条置顶
    if (pinnedMessagesByChannel.value[channelId].length >= 5) {
      pinnedMessagesByChannel.value[channelId].shift()
    }
    
    pinnedMessagesByChannel.value[channelId].push(message)
  }

  const unpinMessage = (channelId, messageId) => {
    const pinned = pinnedMessagesByChannel.value[channelId]
    if (pinned) {
      const index = pinned.findIndex(m => m.id === messageId)
      if (index !== -1) {
        pinned.splice(index, 1)
      }
    }
  }

  const clearMessages = (channelId) => {
    if (channelId) {
      messagesByChannel.value[channelId] = []
      pinnedMessagesByChannel.value[channelId] = []
    } else {
      messagesByChannel.value = { main: [] }
      pinnedMessagesByChannel.value = { main: [] }
    }
  }

  const setEditingMessage = (message) => {
    editingMessage.value = message
  }

  return {
    messagesByChannel,
    pinnedMessagesByChannel,
    editingMessage,
    getChannelMessages,
    getPinnedMessages,
    addMessage,
    addMessages,
    updateMessage,
    deleteMessage,
    pinMessage,
    unpinMessage,
    clearMessages,
    setEditingMessage
  }
})

