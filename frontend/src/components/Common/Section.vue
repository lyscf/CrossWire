<template>
  <div class="cw-section" :style="sectionStyle">
    <div v-if="showHeader" class="cw-section__header">
      <h3 class="cw-section__title">{{ title }}</h3>
      <div class="cw-section__extra">
        <slot name="extra" />
      </div>
    </div>
    <div class="cw-section__body">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  title: { type: String, default: '' },
  padding: { type: [String, Number], default: 'md' },
  gap: { type: [String, Number], default: 'sm' },
  bordered: { type: Boolean, default: true },
  showHeader: { type: Boolean, default: true }
})

const computeSpace = (value) => {
  if (typeof value === 'number') return `${value}px`
  if (typeof value === 'string') {
    if (value.endsWith('px') || value.endsWith('rem') || value.endsWith('%')) return value
    return `var(--spacing-${value})`
  }
  return 'var(--spacing-md)'
}

const sectionStyle = computed(() => ({
  '--cw-section-padding': computeSpace(props.padding),
  '--cw-section-gap': computeSpace(props.gap),
  borderBottom: props.bordered ? '1px solid #f0f0f0' : 'none'
}))
</script>

<style scoped>
.cw-section {
  background: #fff;
}
.cw-section__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--cw-section-gap);
  padding: var(--cw-section-padding);
  border-bottom: 1px solid #f0f0f0;
}
.cw-section__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.cw-section__extra {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
}
.cw-section__body {
  padding: var(--cw-section-padding);
}
</style>


