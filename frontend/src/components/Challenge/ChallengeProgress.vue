<template>
  <div class="challenge-progress">
    <a-card title="解题进度" size="small">
      <!-- 整体进度 -->
      <div class="overall-progress">
        <div class="progress-header">
          <span class="progress-label">总体进度</span>
          <span class="progress-value">{{ overallProgress }}%</span>
        </div>
        <a-progress
          :percent="overallProgress"
          :status="overallProgress === 100 ? 'success' : 'active'"
          :stroke-color="{
            '0%': '#108ee9',
            '100%': '#52c41a'
          }"
        />
      </div>

      <a-divider />

      <!-- 成员进度列表 -->
      <div class="member-progress-list">
        <h4 class="section-title">成员进度</h4>
        <a-list :data-source="progressData" :split="false">
          <template #renderItem="{ item }">
            <a-list-item class="progress-item">
              <a-list-item-meta>
                <template #avatar>
                  <a-avatar :style="{ backgroundColor: getAvatarColor(item.memberName) }">
                    {{ item.memberName[0] }}
                  </a-avatar>
                </template>
                <template #title>
                  <div class="member-info">
                    <span class="member-name">{{ item.memberName }}</span>
                    <a-tag v-if="item.progress === 100" color="success" size="small">
                      <CheckCircleOutlined /> 已完成
                    </a-tag>
                    <a-tag v-else-if="item.progress > 0" color="processing" size="small">
                      <SyncOutlined :spin="true" /> 进行中
                    </a-tag>
                  </div>
                </template>
                <template #description>
                  <div class="progress-details">
                    <a-progress
                      :percent="item.progress"
                      :show-info="false"
                      size="small"
                      :status="item.progress === 100 ? 'success' : 'active'"
                    />
                    <div class="progress-meta">
                      <span class="progress-percent">{{ item.progress }}%</span>
                      <span class="progress-time">
                        更新于 {{ formatTime(item.updatedAt) }}
                      </span>
                    </div>
                    <div v-if="item.summary" class="progress-summary">
                      {{ item.summary }}
                    </div>
                  </div>
                </template>
              </a-list-item-meta>
            </a-list-item>
          </template>
        </a-list>
      </div>

      <a-divider />

      <!-- 进度时间线 -->
      <div class="progress-timeline">
        <h4 class="section-title">进度历史</h4>
        <a-timeline>
          <a-timeline-item
            v-for="event in timelineEvents"
            :key="event.id"
            :color="getTimelineColor(event.type)"
          >
            <template #dot>
              <component :is="getTimelineIcon(event.type)" />
            </template>
            <div class="timeline-content">
              <div class="timeline-header">
                <strong>{{ event.memberName }}</strong>
                <span class="timeline-action">{{ event.action }}</span>
              </div>
              <div class="timeline-detail">{{ event.detail }}</div>
              <div class="timeline-time">{{ formatTime(event.timestamp) }}</div>
            </div>
          </a-timeline-item>
        </a-timeline>
      </div>

      <!-- 统计信息 -->
      <div class="progress-stats">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-statistic
              title="参与人数"
              :value="progressData.length"
              :prefix="h(TeamOutlined)"
            />
          </a-col>
          <a-col :span="8">
            <a-statistic
              title="完成人数"
              :value="completedCount"
              :prefix="h(CheckCircleOutlined)"
              :value-style="{ color: '#52c41a' }"
            />
          </a-col>
          <a-col :span="8">
            <a-statistic
              title="平均进度"
              :value="averageProgress"
              suffix="%"
              :prefix="h(RiseOutlined)"
            />
          </a-col>
        </a-row>
      </div>
    </a-card>
  </div>
</template>

<script setup>
import { computed, h } from 'vue'
import {
  CheckCircleOutlined,
  SyncOutlined,
  TeamOutlined,
  RiseOutlined,
  ClockCircleOutlined,
  EditOutlined,
  FlagOutlined
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

const props = defineProps({
  progressData: {
    type: Array,
    default: () => []
  }
})

// 计算整体进度
const overallProgress = computed(() => {
  if (props.progressData.length === 0) return 0
  const total = props.progressData.reduce((sum, item) => sum + item.progress, 0)
  return Math.round(total / props.progressData.length)
})

// 完成人数
const completedCount = computed(() => {
  return props.progressData.filter(item => item.progress === 100).length
})

// 平均进度
const averageProgress = computed(() => {
  return overallProgress.value
})

// 时间线事件（从后端加载）
const timelineEvents = computed(() => [])

const formatTime = (timestamp) => {
  return dayjs(timestamp).fromNow()
}

const getAvatarColor = (name) => {
  const colors = [
    '#1890ff',
    '#52c41a',
    '#faad14',
    '#f5222d',
    '#722ed1',
    '#13c2c2',
    '#eb2f96'
  ]
  const index = name.charCodeAt(0) % colors.length
  return colors[index]
}

const getTimelineColor = (type) => {
  const colors = {
    update: 'blue',
    complete: 'green',
    start: 'gray'
  }
  return colors[type] || 'blue'
}

const getTimelineIcon = (type) => {
  const icons = {
    update: EditOutlined,
    complete: FlagOutlined,
    start: ClockCircleOutlined
  }
  return icons[type] || EditOutlined
}
</script>

<style scoped>
.challenge-progress {
  width: 100%;
}

.overall-progress {
  margin-bottom: 24px;
}

.progress-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.progress-label {
  font-size: 14px;
  font-weight: 500;
  color: rgba(0, 0, 0, 0.85);
}

.progress-value {
  font-size: 18px;
  font-weight: 600;
  color: #1890ff;
}

.section-title {
  margin: 0 0 16px 0;
  font-size: 14px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.85);
}

.progress-item {
  padding: 12px 0;
}

.member-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.member-name {
  font-weight: 500;
}

.progress-details {
  margin-top: 8px;
}

.progress-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 12px;
}

.progress-percent {
  color: rgba(0, 0, 0, 0.85);
  font-weight: 500;
}

.progress-time {
  color: rgba(0, 0, 0, 0.45);
}

.progress-summary {
  margin-top: 8px;
  padding: 8px;
  background-color: #fafafa;
  border-radius: 4px;
  font-size: 13px;
  color: rgba(0, 0, 0, 0.65);
}

.timeline-content {
  
}

.timeline-header {
  margin-bottom: 4px;
}

.timeline-action {
  margin-left: 8px;
  color: rgba(0, 0, 0, 0.65);
}

.timeline-detail {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 4px;
}

.timeline-time {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.progress-stats {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid #f0f0f0;
}
</style>

