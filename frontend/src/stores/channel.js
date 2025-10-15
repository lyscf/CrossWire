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

  // 主频道信息
  const mainChannel = ref({
    id: 'main',
    name: '主频道',
    type: 'main',
    unreadCount: 0
  })

  // 子频道列表（题目频道）
  const subChannels = ref([])

  // 当前选中的频道 ID
  const selectedChannelId = ref('main')

  // 所有频道列表（主频道 + 子频道）
  const channels = computed(() => {
    return [mainChannel.value, ...subChannels.value]
  })

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

  // 子频道管理
  const setSubChannels = (channelList) => {
    subChannels.value = channelList.map(ch => ({
      id: ch.id || ch.ID,
      name: ch.name || ch.Name,
      type: 'sub',
      message_count: ch.message_count || ch.MessageCount || 0,
      unreadCount: 0
    }))
  }

  const addSubChannel = (channel) => {
    const exists = subChannels.value.find(c => c.id === channel.id)
    if (!exists) {
      subChannels.value.push({
        id: channel.id,
        name: channel.name,
        type: 'sub',
        message_count: channel.message_count || 0,
        unreadCount: 0
      })
    }
  }

  const removeSubChannel = (channelId) => {
    const index = subChannels.value.findIndex(c => c.id === channelId)
    if (index !== -1) {
      subChannels.value.splice(index, 1)
    }
  }

  const updateSubChannel = (channelId, updates) => {
    const channel = subChannels.value.find(c => c.id === channelId)
    if (channel) {
      Object.assign(channel, updates)
    }
  }

  // 兼容性方法
  const addChannel = (channel) => {
    if (channel.type === 'main') {
      Object.assign(mainChannel.value, channel)
    } else {
      addSubChannel(channel)
    }
  }

  const removeChannel = (channelId) => {
    if (channelId === 'main') {
      // 主频道不能删除，只重置
      reset()
    } else {
      removeSubChannel(channelId)
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
    mainChannel.value = {
      id: 'main',
      name: '主频道',
      type: 'main',
      unreadCount: 0
    }
    subChannels.value = []
    selectedChannelId.value = 'main'
  }

  return {
    // 状态
    currentChannel,
    mainChannel,
    subChannels,
    channels,
    selectedChannelId,
    
    // Getters
    selectedChannel,
    totalUnreadCount,
    
    // Actions
    setCurrentChannel,
    selectChannel,
    setSubChannels,
    addSubChannel,
    removeSubChannel,
    updateSubChannel,
    
    // 兼容性方法
    addChannel,
    removeChannel,
    incrementUnreadCount,
    reset
  }
})

