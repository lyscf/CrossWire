<template>
  <div
    class="cw-stack"
    :class="{
      'cw-stack--row': directionComputed === 'row',
      'cw-stack--wrap': wrap
    }"
    :style="stackStyle"
  >
    <slot />
  </div>
  </template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  direction: { type: String, default: 'column' }, // 'row' | 'column'
  gap: { type: [String, Number], default: 'md' }, // xs|sm|md|lg|xl or px
  align: { type: String, default: 'center' },
  justify: { type: String, default: 'flex-start' },
  wrap: { type: Boolean, default: false }
})

const directionComputed = computed(() => (props.direction === 'row' ? 'row' : 'column'))

const computeGap = (value) => {
  if (typeof value === 'number') return `${value}px`
  if (typeof value === 'string') {
    if (value.endsWith('px') || value.endsWith('rem') || value.endsWith('%')) return value
    return `var(--spacing-${value})`
  }
  return 'var(--spacing-md)'
}

const stackStyle = computed(() => ({
  gap: computeGap(props.gap),
  alignItems: props.align,
  justifyContent: props.justify
}))
</script>

<style scoped>
.cw-stack {
  display: flex;
  flex-direction: column;
}
.cw-stack--row {
  flex-direction: row;
}
.cw-stack--wrap {
  flex-wrap: wrap;
}
</style>


