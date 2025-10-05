<template>
  <div class="challenge-list">
    <a-list
      :data-source="challenges"
      :split="false"
    >
      <template #renderItem="{ item }">
        <ChallengeCard
          :challenge="item"
          :selected="selectedId === item.id"
          @click="$emit('select', item.id)"
        />
      </template>
    </a-list>
  </div>
</template>

<script setup>
import ChallengeCard from './ChallengeCard.vue'

defineProps({
  challenges: {
    type: Array,
    default: () => []
  },
  selectedId: {
    type: String,
    default: null
  }
})

defineEmits(['select'])
</script>

<style scoped>
.challenge-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}
</style>

