<template>
  <a-badge v-if="badgeStatus" :status="badgeStatus" :offset="badgeOffset">
    <a-avatar :size="size" :src="src" :style="avatarStyle">
      <template v-if="!src">{{ initial }}</template>
    </a-avatar>
  </a-badge>
  <a-avatar v-else :size="size" :src="src" :style="avatarStyle">
    <template v-if="!src">{{ initial }}</template>
  </a-avatar>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, default: '' },
  src: { type: String, default: '' },
  size: { type: [Number, String], default: 32 },
  // status can be one of: online | busy | away | offline | success | error | warning | default
  status: { type: String, default: '' },
  badgeOffset: { type: Array, default: () => [0, 0] }
})

const initial = computed(() => (props.name && props.name.length > 0 ? props.name[0].toUpperCase() : '?'))

const palette = ['#1890ff', '#52c41a', '#faad14', '#f5222d', '#722ed1', '#13c2c2', '#eb2f96']
const colorIndex = computed(() => (props.name && props.name.length > 0 ? props.name.charCodeAt(0) % palette.length : 0))

const avatarStyle = computed(() => {
  if (props.src) return {}
  return { backgroundColor: palette[colorIndex.value], color: '#fff' }
})

const badgeStatus = computed(() => {
  if (!props.status) return ''
  const map = {
    online: 'success',
    busy: 'error',
    away: 'warning',
    offline: 'default',
    success: 'success',
    error: 'error',
    warning: 'warning',
    default: 'default'
  }
  return map[props.status] || ''
})
</script>

<style scoped>
</style>


