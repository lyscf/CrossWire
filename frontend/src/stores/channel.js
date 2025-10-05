import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useChannelStore = defineStore('channel', () => {
  // 当前频道信息
  const currentChannel = ref({
    id: '',
    name: '',
    createdAt: null,
    transportMode: '',
    maxMembers: 50,
    memberCount: 0
  })

  // 频道列表（包括主频道和题目频道）
  const channels = ref([
    {
      id: 'main',
      name: '主频道',
      type: 'main',
      unreadCount: 0
    }
  ])

  // 当前选中的频道 ID
  const selectedChannelId = ref('main')

  // Getters
  const selectedChannel = computed(() => {
    return channels.value.find(c => c.id === selectedChannelId.value)
  })

  const totalUnreadCount = computed(() => {
    return channels.value.reduce((sum, channel) => sum + channel.unreadCount, 0)
  })

  // Actions
  const setCurrentChannel = (channel) => {
    currentChannel.value = { ...currentChannel.value, ...channel }
  }

  const selectChannel = (channelId) => {
    selectedChannelId.value = channelId
    // 清除未读计数
    const channel = channels.value.find(c => c.id === channelId)
    if (channel) {
      channel.unreadCount = 0
    }
  }

  const addChannel = (channel) => {
    const exists = channels.value.find(c => c.id === channel.id)
    if (!exists) {
      channels.value.push({
        ...channel,
        unreadCount: 0
      })
    }
  }

  const removeChannel = (channelId) => {
    const index = channels.value.findIndex(c => c.id === channelId)
    if (index !== -1) {
      channels.value.splice(index, 1)
    }
  }

  const incrementUnreadCount = (channelId) => {
    const channel = channels.value.find(c => c.id === channelId)
    if (channel && channel.id !== selectedChannelId.value) {
      channel.unreadCount++
    }
  }

  const reset = () => {
    currentChannel.value = {
      id: '',
      name: '',
      createdAt: null,
      transportMode: '',
      maxMembers: 50,
      memberCount: 0
    }
    channels.value = [
      {
        id: 'main',
        name: '主频道',
        type: 'main',
        unreadCount: 0
      }
    ]
    selectedChannelId.value = 'main'
  }

  return {
    currentChannel,
    channels,
    selectedChannelId,
    selectedChannel,
    totalUnreadCount,
    setCurrentChannel,
    selectChannel,
    addChannel,
    removeChannel,
    incrementUnreadCount,
    reset
  }
})

