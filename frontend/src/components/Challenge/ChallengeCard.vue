<template>
  <a-list-item
    class="challenge-card"
    :class="{ 'challenge-card-selected': selected }"
  >
    <div class="card-content">
      <!-- 标题和类型 -->
      <div class="card-header">
        <h4 class="card-title">{{ challenge.title }}</h4>
        <a-tag :color="getCategoryColor(challenge.category)" size="small">
          {{ challenge.category }}
        </a-tag>
      </div>

      <!-- 分值和难度 -->
      <div class="card-meta">
        <a-space>
          <span class="meta-item">
            <TrophyOutlined /> {{ challenge.points }}分
          </span>
          <a-tag
            :color="getDifficultyColor(challenge.difficulty)"
            size="small"
          >
            {{ challenge.difficulty }}
          </a-tag>
        </a-space>
      </div>

      <!-- 状态和进度 -->
      <div class="card-status">
        <div class="status-text">
          <span :style="{ color: getStatusColor(challenge.status) }">
            {{ getStatusText(challenge.status) }}
          </span>
        </div>
        <div v-if="challenge.progress > 0" class="progress-bar">
          <a-progress
            :percent="challenge.progress"
            :show-info="false"
            size="small"
            :stroke-color="challenge.status === 'solved' ? '#52c41a' : '#1890ff'"
          />
        </div>
      </div>

      <!-- 分配的成员 -->
      <div v-if="challenge.assignedTo && challenge.assignedTo.length > 0" class="card-assigned">
        <a-avatar-group :max-count="3" size="small">
          <a-avatar
            v-for="member in challenge.assignedTo"
            :key="member"
            size="small"
          >
            {{ member[0] }}
          </a-avatar>
        </a-avatar-group>
      </div>
    </div>
  </a-list-item>
</template>

<script setup>
import { TrophyOutlined } from '@ant-design/icons-vue'

defineProps({
  challenge: {
    type: Object,
    required: true
  },
  selected: {
    type: Boolean,
    default: false
  }
})

const getCategoryColor = (category) => {
  const colors = {
    Web: 'blue',
    Pwn: 'red',
    Reverse: 'purple',
    Crypto: 'orange',
    Misc: 'green',
    Forensics: 'cyan'
  }
  return colors[category] || 'default'
}

const getDifficultyColor = (difficulty) => {
  const colors = {
    Easy: 'success',
    Medium: 'warning',
    Hard: 'error',
    Insane: 'purple'
  }
  return colors[difficulty] || 'default'
}

const getStatusColor = (status) => {
  const colors = {
    pending: 'rgba(0, 0, 0, 0.45)',
    in_progress: '#1890ff',
    solved: '#52c41a'
  }
  return colors[status] || 'rgba(0, 0, 0, 0.45)'
}

const getStatusText = (status) => {
  const texts = {
    pending: '待分配',
    in_progress: '进行中',
    solved: '已解决'
  }
  return texts[status] || '未知'
}
</script>

<style scoped>
.challenge-card {
  padding: 12px;
  margin-bottom: 8px;
  border-radius: 4px;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.3s;
}

.challenge-card:hover {
  background-color: #f5f5f5;
  border-color: #d9d9d9;
}

.challenge-card-selected {
  background-color: #e6f7ff;
  border-color: #1890ff;
}

.card-content {
  width: 100%;
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 8px;
  gap: 8px;
}

.card-title {
  flex: 1;
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
  line-height: 1.4;
}

.card-meta {
  margin-bottom: 8px;
}

.meta-item {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.65);
}

.card-status {
  margin-bottom: 8px;
}

.status-text {
  font-size: 12px;
  margin-bottom: 4px;
}

.progress-bar {
  margin-top: 4px;
}

.card-assigned {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #f0f0f0;
}
</style>

