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
                <AvatarChip :name="item.nickname" :status="item.status" :badge-offset="[-5, 35]" />
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
                <AvatarChip :name="item.nickname" />
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
import AvatarChip from '@/components/Common/AvatarChip.vue'
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

</script>

<style scoped>
.member-list {
  padding: var(--spacing-sm) 0;
}

.member-section {
  margin-bottom: var(--spacing-md);
}

.section-header {
  padding: var(--spacing-sm) 0;
  margin-bottom: var(--spacing-sm);
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.65);
  text-transform: uppercase;
}

.member-item {
  padding: var(--spacing-sm) var(--spacing-sm);
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
  gap: var(--spacing-xs);
  font-size: 14px;
  font-weight: 500;
}

.member-info {
  margin-top: var(--spacing-xs);
}

.member-skills {
  margin-bottom: var(--spacing-xs);
}

.member-task {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.65);
  margin-top: var(--spacing-xs);
}

.member-status-text {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}
</style>

