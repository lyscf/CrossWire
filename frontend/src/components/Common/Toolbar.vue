<template>
  <div class="cw-toolbar" :style="toolbarStyle">
    <div class="cw-toolbar__left">
      <slot name="left" />
    </div>
    <div class="cw-toolbar__center">
      <slot name="center" />
    </div>
    <div class="cw-toolbar__right">
      <slot name="right" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  gap: { type: [String, Number], default: 'md' },
  height: { type: [String, Number], default: 64 }, // px
  paddingX: { type: [String, Number], default: '16px' },
  background: { type: String, default: '#fff' },
  border: { type: Boolean, default: false }
})

const computeGap = (value) => {
  if (typeof value === 'number') return `${value}px`
  if (typeof value === 'string') {
    if (value.endsWith('px') || value.endsWith('rem') || value.endsWith('%')) return value
    return `var(--spacing-${value})`
  }
  return 'var(--spacing-md)'
}

const normalizePx = (value) => {
  if (typeof value === 'number') return `${value}px`
  return value
}

const toolbarStyle = computed(() => ({
  gap: computeGap(props.gap),
  height: normalizePx(props.height),
  padding: `0 ${normalizePx(props.paddingX)}`,
  background: props.background,
  borderBottom: props.border ? '1px solid #f0f0f0' : undefined
}))
</script>

<style scoped>
.cw-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}
.cw-toolbar__left {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  flex-shrink: 0;
  min-width: 0;
}
.cw-toolbar__center {
  flex: 1;
  min-width: 0;
}
.cw-toolbar__right {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  flex-shrink: 0;
}
</style>


