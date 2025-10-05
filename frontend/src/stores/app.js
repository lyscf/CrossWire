import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  // 运行模式
  const mode = ref(null) // 'server' | 'client' | null
  const isConnected = ref(false)

  // 当前用户信息
  const currentUser = ref({
    id: '',
    nickname: '',
    role: '',
    skills: [],
    avatar: ''
  })

  // 应用设置
  const settings = ref({
    theme: 'light',
    fontSize: 14,
    notifications: true,
    soundEnabled: true
  })

  // Actions
  const setMode = (newMode) => {
    mode.value = newMode
  }

  const setConnected = (status) => {
    isConnected.value = status
  }

  const setCurrentUser = (user) => {
    currentUser.value = { ...currentUser.value, ...user }
  }

  const updateSettings = (newSettings) => {
    settings.value = { ...settings.value, ...newSettings }
  }

  const reset = () => {
    mode.value = null
    isConnected.value = false
    currentUser.value = {
      id: '',
      nickname: '',
      role: '',
      skills: [],
      avatar: ''
    }
  }

  return {
    mode,
    isConnected,
    currentUser,
    settings,
    setMode,
    setConnected,
    setCurrentUser,
    updateSettings,
    reset
  }
})

