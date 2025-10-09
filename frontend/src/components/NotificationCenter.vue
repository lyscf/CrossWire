<template>
  <a-dropdown :trigger="['click']" placement="bottomRight">
    <a-badge :count="unreadCount" :offset="[-5, 5]">
      <a-button type="text" size="large" class="notification-button">
        <BellOutlined />
      </a-button>
    </a-badge>

    <template #overlay>
      <div class="notification-panel">
        <div class="notification-header">
          <h4>通知中心</h4>
          <a-space>
            <a-button type="link" size="small" @click="markAllRead">
              全部已读
            </a-button>
            <a-button type="link" size="small" @click="clearAll">
              清空
            </a-button>
          </a-space>
        </div>

        <a-tabs v-model:activeKey="activeTab" size="small">
          <a-tab-pane key="all" tab="全部">
            <div class="notification-list">
              <div
                v-for="notification in allNotifications"
                :key="notification.id"
                class="notification-item"
                :class="{ 'notification-unread': !notification.read }"
                @click="handleNotificationClick(notification)"
              >
                <div class="notification-icon">
                  <component
                    :is="getNotificationIcon(notification.type)"
                    :style="{ color: getNotificationColor(notification.type) }"
                  />
                </div>
                
                <div class="notification-content">
                  <div class="notification-title">
                    {{ notification.title }}
                  </div>
                  <div class="notification-desc">
                    {{ notification.description }}
                  </div>
                  <div class="notification-time">
                    {{ formatTime(notification.timestamp) }}
                  </div>
                </div>

                <a-button
                  type="text"
                  size="small"
                  class="notification-close"
                  @click.stop="removeNotification(notification.id)"
                >
                  <CloseOutlined />
                </a-button>
              </div>

              <a-empty
                v-if="allNotifications.length === 0"
                description="暂无通知"
                :image="Empty.PRESENTED_IMAGE_SIMPLE"
              />
            </div>
          </a-tab-pane>

          <a-tab-pane key="mention" :tab="`@提及 (${mentionNotifications.length})`">
            <div class="notification-list">
              <div
                v-for="notification in mentionNotifications"
                :key="notification.id"
                class="notification-item"
                :class="{ 'notification-unread': !notification.read }"
                @click="handleNotificationClick(notification)"
              >
                <div class="notification-icon">
                  <UserOutlined style="color: #1890ff" />
                </div>
                
                <div class="notification-content">
                  <div class="notification-title">
                    {{ notification.title }}
                  </div>
                  <div class="notification-desc">
                    {{ notification.description }}
                  </div>
                  <div class="notification-time">
                    {{ formatTime(notification.timestamp) }}
                  </div>
                </div>
              </div>

              <a-empty
                v-if="mentionNotifications.length === 0"
                description="暂无@提及"
                :image="Empty.PRESENTED_IMAGE_SIMPLE"
              />
            </div>
          </a-tab-pane>

          <a-tab-pane key="system" :tab="`系统 (${systemNotifications.length})`">
            <div class="notification-list">
              <div
                v-for="notification in systemNotifications"
                :key="notification.id"
                class="notification-item"
                @click="handleNotificationClick(notification)"
              >
                <div class="notification-icon">
                  <InfoCircleOutlined style="color: #52c41a" />
                </div>
                
                <div class="notification-content">
                  <div class="notification-title">
                    {{ notification.title }}
                  </div>
                  <div class="notification-desc">
                    {{ notification.description }}
                  </div>
                  <div class="notification-time">
                    {{ formatTime(notification.timestamp) }}
                  </div>
                </div>
              </div>

              <a-empty
                v-if="systemNotifications.length === 0"
                description="暂无系统通知"
                :image="Empty.PRESENTED_IMAGE_SIMPLE"
              />
            </div>
          </a-tab-pane>
        </a-tabs>
      </div>
    </template>
  </a-dropdown>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Empty } from 'ant-design-vue'
import { EventsOn } from '@/wailsjs/runtime/runtime'
import { getMyInfo } from '@/api/app'
import { useChallengeStore } from '@/stores/challenge'
import {
  BellOutlined,
  UserOutlined,
  TrophyOutlined,
  InfoCircleOutlined,
  CloseOutlined,
  MessageOutlined,
  FlagOutlined
} from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
// 可选：后续可接入通知历史 API
// import { getNotifications } from '@/api/app'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

const activeTab = ref('all')
// Stores
const challengeStore = useChallengeStore()


// 通知数据（通过Wails事件 EventsOn('app:event') 接收）
const notifications = ref([])
const myId = ref('')

const addNotification = (n) => {
  notifications.value.unshift(n)
  if (notifications.value.length > 50) {
    notifications.value = notifications.value.slice(0, 50)
  }
  try {
    localStorage.setItem('notifications', JSON.stringify(notifications.value))
  } catch {}
}

const loadNotifications = () => {
  try {
    const raw = localStorage.getItem('notifications')
    if (raw) notifications.value = JSON.parse(raw)
  } catch {}
}

// 组件挂载时初始化
onMounted(async () => {
  loadNotifications()
  try {
    const me = await getMyInfo()
    myId.value = me?.id || me?.ID || ''
  } catch {}

  // 订阅后端统一事件入口
  EventsOn('app:event', (evt) => {
    if (!evt) return
    const type = evt.type || evt.Type
    const data = evt.data || evt.Data || {}
    // 题目进度更新
    if (type === 'challenge:progress') {
      const ch = data.challenge || data.Challenge || {}
      const challengeId = ch.id || ch.ID || data.challenge_id || data.ChallengeID
      const extra = data.extra_data || data.ExtraData || {}
      const progressValue = Number(extra.progress || extra.Progress || 0)
      if (challengeId) {
        try {
          challengeStore.updateProgress(challengeId, progressValue)
          // 可选：通知提示
          addNotification({
            id: Date.now().toString(),
            type: 'system',
            title: '进度更新',
            description: `题目进度: ${progressValue}%`,
            timestamp: Date.now(),
            read: false
          })
        } catch {}
      }
      return
    }


    // 消息接收：检测@提及
    if (type === 'message:received') {
      const msg = data.message || data.Message || {}
      const mentions = msg.mentions || msg.Mentions || []
      const contentText = msg.content_text || msg.ContentText || msg.content?.text || ''
      if (myId.value && Array.isArray(mentions) && mentions.includes(myId.value)) {
        addNotification({
          id: Date.now().toString(),
          type: 'mention',
          title: `${msg.sender_nickname || '有人'} 提到了你`,
          description: contentText,
          timestamp: Date.now(),
          read: false,
          link: `/channel/${msg.channel_id || ''}`
        })
      }
      return
    }

    // 题目分配
    if (type === 'challenge:assigned') {
      const ch = data.challenge || data.Challenge || {}
      const title = ch.title || ch.Title || '新题目'
      addNotification({
        id: Date.now().toString(),
        type: 'challenge',
        title: '新题目分配',
        description: `管理员为你分配了题目: ${title}`,
        timestamp: Date.now(),
        read: false,
        link: `/challenges`
      })
      return
    }

    // Flag提交
    if (type === 'challenge:submitted') {
      addNotification({
        id: Date.now().toString(),
        type: 'flag',
        title: 'Flag已提交',
        description: data?.message || data?.Message || '已提交',
        timestamp: Date.now(),
        read: false
      })
      return
    }

    // 系统类
    if (type?.startsWith('system:')) {
      addNotification({
        id: Date.now().toString(),
        type: 'system',
        title: '系统通知',
        description: (data && (data.message || data.Message)) || type,
        timestamp: Date.now(),
        read: false
      })
    }
  })
})

const allNotifications = computed(() => notifications.value)

const mentionNotifications = computed(() => 
  notifications.value.filter(n => n.type === 'mention')
)

const systemNotifications = computed(() => 
  notifications.value.filter(n => n.type === 'system' || n.type === 'flag' || n.type === 'challenge')
)

const unreadCount = computed(() => 
  notifications.value.filter(n => !n.read).length
)

const getNotificationIcon = (type) => {
  const icons = {
    mention: UserOutlined,
    challenge: TrophyOutlined,
    flag: FlagOutlined,
    message: MessageOutlined,
    system: InfoCircleOutlined
  }
  return icons[type] || InfoCircleOutlined
}

const getNotificationColor = (type) => {
  const colors = {
    mention: '#1890ff',
    challenge: '#faad14',
    flag: '#52c41a',
    message: '#1890ff',
    system: '#52c41a'
  }
  return colors[type] || '#1890ff'
}

const formatTime = (timestamp) => {
  return dayjs(timestamp).fromNow()
}

const handleNotificationClick = (notification) => {
  // 标记为已读
  notification.read = true
  
  // 跳转到相关页面
  if (notification.link) {
    // router.push(notification.link)
    console.log('Navigate to:', notification.link)
  }
}

const markAllRead = () => {
  notifications.value.forEach(n => n.read = true)
}

const clearAll = () => {
  notifications.value = []
}

const removeNotification = (id) => {
  const index = notifications.value.findIndex(n => n.id === id)
  if (index > -1) {
    notifications.value.splice(index, 1)
  }
}
</script>

<style scoped>
.notification-button {
  display: flex;
  align-items: center;
  justify-content: center;
}

.notification-panel {
  width: 400px;
  background: white;
  border-radius: 4px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
}

.notification-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-md);
  border-bottom: 1px solid #f0f0f0;
}

.notification-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.notification-list {
  max-height: 400px;
  overflow-y: auto;
}

.notification-item {
  display: flex;
  gap: var(--spacing-sm);
  padding: var(--spacing-sm) var(--spacing-md);
  cursor: pointer;
  transition: background-color 0.2s;
  position: relative;
}

.notification-item:hover {
  background-color: #f5f5f5;
}

.notification-item:hover .notification-close {
  opacity: 1;
}

.notification-unread {
  background-color: #e6f7ff;
}

.notification-icon {
  flex-shrink: 0;
  font-size: 20px;
  margin-top: var(--spacing-xs);
}

.notification-content {
  flex: 1;
  min-width: 0;
}

.notification-title {
  font-size: 14px;
  font-weight: 500;
  color: rgba(0, 0, 0, 0.85);
  margin-bottom: var(--spacing-xs);
}

.notification-desc {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: var(--spacing-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-time {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.notification-close {
  position: absolute;
  top: 12px;
  right: 12px;
  opacity: 0;
  transition: opacity 0.2s;
}
</style>

