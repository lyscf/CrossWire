<template>
  <div class="cw-empty">
    <a-empty :image="image" :description="description">
      <template #description>
        <div class="cw-empty__desc">{{ description }}</div>
      </template>
      <slot />
    </a-empty>
  </div>
</template>

<script setup>
const props = defineProps({
  description: { type: String, default: '暂无数据' },
  simple: { type: Boolean, default: true }
})

const image = props.simple ? undefined : undefined
</script>

<style scoped>
.cw-empty {
  padding: var(--spacing-md) 0;
}
.cw-empty__desc {
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
}
</style>


