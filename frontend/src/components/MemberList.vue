<template>
  <div class="member-list">
    <!-- 在线成员 -->
    <div class="member-section">
      <div class="section-header">
        <span class="section-title">在线成员 ({{ onlineMembers.length }})</span>
      </div>

      <a-list :data-source="onlineMembers" :split="false">
        <template #renderItem="{ item }">
          <a-list-item class="member-item">
            <a-list-item-meta>
              <template #avatar>
                <a-badge :status="getStatusBadge(item.status)" :offset="[-5, 35]">
                  <a-avatar :style="{ backgroundColor: getAvatarColor(item.nickname) }">
                    {{ item.nickname[0] }}
                  </a-avatar>
                </a-badge>
              </template>

              <template #title>
                <div class="member-name">
                  {{ item.nickname }}
                  <a-tag v-if="item.role === '队长'" color="gold" size="small">
                    <CrownOutlined /> 队长
                  </a-tag>
                  <a-tag v-else-if="item.role === '管理员'" color="blue" size="small">
                    管理
                  </a-tag>
                </div>
              </template>

              <template #description>
                <div class="member-info">
                  <div class="member-skills">
                    <a-space :size="4">
                      <a-tag
                        v-for="skill in item.skills"
                        :key="skill"
                        size="small"
                        color="blue"
                      >
                        {{ skill }}
                      </a-tag>
                    </a-space>
                  </div>
                  <div v-if="item.currentTask" class="member-task">
                    <CodeOutlined /> 正在做: {{ item.currentTask }}
                  </div>
                  <div v-else class="member-status-text">
                    {{ getStatusText(item.status) }}
                  </div>
                </div>
              </template>
            </a-list-item-meta>

            <template #actions>
              <a-dropdown>
                <a-button type="text" size="small">
                  <EllipsisOutlined />
                </a-button>
                <template #overlay>
                  <a-menu>
                    <a-menu-item key="mention">
                      <UserAddOutlined /> 提及
                    </a-menu-item>
                    <a-menu-item key="viewProfile">
                      <ProfileOutlined /> 查看资料
                    </a-menu-item>
                  </a-menu>
                </template>
              </a-dropdown>
            </template>
          </a-list-item>
        </template>
      </a-list>
    </div>

    <!-- 离线成员 -->
    <div v-if="offlineMembers.length > 0" class="member-section">
      <a-divider />
      <div class="section-header">
        <span class="section-title">离线成员 ({{ offlineMembers.length }})</span>
      </div>

      <a-list :data-source="offlineMembers" :split="false">
        <template #renderItem="{ item }">
          <a-list-item class="member-item member-offline">
            <a-list-item-meta>
              <template #avatar>
                <a-avatar
                  :style="{ backgroundColor: '#d9d9d9', opacity: 0.6 }"
                >
                  {{ item.nickname[0] }}
                </a-avatar>
              </template>

              <template #title>
                <span style="color: #999">{{ item.nickname }}</span>
              </template>
            </a-list-item-meta>
          </a-list-item>
        </template>
      </a-list>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import {
  CrownOutlined,
  CodeOutlined,
  EllipsisOutlined,
  UserAddOutlined,
  ProfileOutlined
} from '@ant-design/icons-vue'

const props = defineProps({
  members: {
    type: Array,
    default: () => []
  }
})

const onlineMembers = computed(() => {
  return props.members.filter(m => m.status !== 'offline')
})

const offlineMembers = computed(() => {
  return props.members.filter(m => m.status === 'offline')
})

const getStatusBadge = (status) => {
  const statusMap = {
    online: 'success',
    busy: 'error',
    away: 'warning',
    offline: 'default'
  }
  return statusMap[status] || 'default'
}

const getStatusText = (status) => {
  const textMap = {
    online: '在线',
    busy: '忙碌中',
    away: '离开',
    offline: '离线'
  }
  return textMap[status] || '未知'
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
</script>

<style scoped>
.member-list {
  padding: 8px 0;
}

.member-section {
  margin-bottom: 16px;
}

.section-header {
  padding: 8px 0;
  margin-bottom: 8px;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.65);
  text-transform: uppercase;
}

.member-item {
  padding: 12px 8px;
  border-radius: 4px;
  transition: background-color 0.3s;
  cursor: pointer;
}

.member-item:hover {
  background-color: #f5f5f5;
}

.member-offline {
  opacity: 0.6;
}

.member-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 500;
}

.member-info {
  margin-top: 4px;
}

.member-skills {
  margin-bottom: 4px;
}

.member-task {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.65);
  margin-top: 4px;
}

.member-status-text {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}
</style>

