<template>
  <div class="cw-page" :class="{ 'cw-page--full': fullHeight }" :style="pageStyle">
    <div v-if="$slots.header" class="cw-page__header">
      <slot name="header" />
    </div>
    <div class="cw-page__content">
      <slot />
    </div>
    <div v-if="$slots.footer" class="cw-page__footer">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  padding: { type: [String, Number], default: 'lg' },
  gap: { type: [String, Number], default: 'md' },
  background: { type: String, default: '#fff' },
  fullHeight: { type: Boolean, default: true }
})

const computeSpace = (value) => {
  if (typeof value === 'number') return `${value}px`
  if (typeof value === 'string') {
    if (value.endsWith('px') || value.endsWith('rem') || value.endsWith('%')) return value
    return `var(--spacing-${value})`
  }
  return 'var(--spacing-md)'
}

const pageStyle = computed(() => ({
  '--cw-page-padding': computeSpace(props.padding),
  '--cw-page-gap': computeSpace(props.gap),
  background: props.background
}))
</script>

<style scoped>
.cw-page {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--cw-page-gap);
}
.cw-page--full {
  min-height: 100vh;
}
.cw-page__header {
  padding: var(--cw-page-padding);
  border-bottom: 1px solid #f0f0f0;
  background: inherit;
}
.cw-page__content {
  padding: var(--cw-page-padding);
  background: inherit;
  flex: 1;
}
.cw-page__footer {
  padding: var(--cw-page-padding);
  border-top: 1px solid #f0f0f0;
  background: inherit;
}
</style>


