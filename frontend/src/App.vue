<template>
  <a-config-provider :theme="themeConfig">
    <router-view />
  </a-config-provider>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { theme } from 'ant-design-vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { useRouter } from 'vue-router'

const router = useRouter()

// Ant Design 标准主题配置
const themeConfig = ref({
  token: {
    colorPrimary: '#1890ff',
    colorSuccess: '#52c41a',
    colorWarning: '#faad14',
    colorError: '#ff4d4f',
    colorInfo: '#1890ff',
    fontSize: 14,
    borderRadius: 2,
    colorBgContainer: '#ffffff',
    colorBgLayout: '#f5f5f5',
  },
  algorithm: theme.defaultAlgorithm,
})

// 全局事件汇聚：对所有 app:event 进行分发，触发对应的 UI 更新
let unsubscribe = () => {}
onMounted(() => {
  try {
    unsubscribe = EventsOn('app:event', (evt) => {
      if (!evt) return
      const type = evt.type || evt.Type
      const data = evt.data || evt.Data || {}
      console.log('[App.vue] app:event', type, data && typeof data === 'object' ? Object.keys(data) : typeof data)

      // 统一派发到 window 级事件，便于各视图监听
      window.dispatchEvent(new CustomEvent('cw:app:event', { detail: { type, data } }))

      // 常见类型的快捷派发
      if (type === 'message:received') {
        window.dispatchEvent(new CustomEvent('cw:message:received', { detail: data }))
      } else if (type === 'member:joined' || type === 'member:left' || type === 'member:updated') {
        window.dispatchEvent(new CustomEvent('cw:member:update', { detail: { type, data } }))
      } else if (type.startsWith('file:')) {
        window.dispatchEvent(new CustomEvent('cw:file:event', { detail: { type, data } }))
      } else if (type.startsWith('challenge:')) {
        try {
          const ch = (data && (data.Challenge || data.challenge)) || data
          console.log('[App.vue] challenge payload', {
            id: ch?.ID || ch?.id,
            title: ch?.Title || ch?.title,
            status: ch?.Status || ch?.status
          })
        } catch (e) {}
        window.dispatchEvent(new CustomEvent('cw:challenge:event', { detail: { type, data } }))
      } else if (type === 'connected' || type === 'disconnected' || type === 'reconnecting') {
        window.dispatchEvent(new CustomEvent('cw:connection:event', { detail: { type, data } }))
      }
    })
  } catch {}
})

onUnmounted(() => {
  try { unsubscribe && unsubscribe() } catch {}
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

#app {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
}
</style>

