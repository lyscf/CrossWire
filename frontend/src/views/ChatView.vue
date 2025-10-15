<template>
  <div class="chat-container">
    <a-layout style="height: 100vh">
      <!-- 左侧边栏 - 频道列表 -->
      <a-layout-sider
        v-model:collapsed="collapsed"
        :trigger="null"
        collapsible
        width="240"
        theme="light"
        class="chat-sider"
      >
        <div class="channel-header">
          <h3 v-if="!collapsed" class="channel-title">
            {{ channelName }}
          </h3>
          <MenuFoldOutlined
            v-if="!collapsed"
            class="trigger"
            @click="() => (collapsed = !collapsed)"
          />
          <MenuUnfoldOutlined
            v-else
            class="trigger"
            @click="() => (collapsed = !collapsed)"
          />
        </div>

        <a-menu
          :selectedKeys="[currentChannelID]"
          mode="inline"
          theme="light"
          :inline-collapsed="collapsed"
        >
          <a-menu-item key="main" @click="handleChannelSelect('main')">
            <template #icon>
              <WechatOutlined />
            </template>
            <span>主频道</span>
          </a-menu-item>

          <!-- 子频道列表 -->
          <a-menu-item
            v-for="channel in channelStore.subChannels"
            :key="channel.id"
            @click="handleChannelSelect(channel.id)"
          >
            <template #icon>
              <CodeOutlined />
            </template>
            <span>{{ channel.name }}</span>
            <span v-if="channel.unreadCount > 0" class="unread-badge">
              {{ channel.unreadCount }}
            </span>
          </a-menu-item>

          <a-menu-divider />

          <a-menu-item key="challenges" @click="goToChallenges">
            <template #icon>
              <TrophyOutlined />
            </template>
            <span>题目管理</span>
          </a-menu-item>

        </a-menu>

        <div v-if="!collapsed" class="sider-footer">
          <a-space>
            <a-button type="text" size="small" @click="showSettings">
              <SettingOutlined />
            </a-button>
            <a-button type="text" size="small" danger @click="handleDisconnect">
              <PoweroffOutlined />
            </a-button>
          </a-space>
        </div>
      </a-layout-sider>

      <a-layout>
        <Headbar
          :current-channel-label="currentChannelLabel"
          :current-user="currentUser"
          :connected="true"
          @open-file-manager="fileManagerVisible = true"
          @open-member-drawer="memberDrawerVisible = true"
          @open-user-profile="userProfileVisible = true"
          @open-settings="settingsVisible = true"
          @disconnect="handleDisconnect"
        />

        <!-- 主内容区 -->
        <a-layout-content class="chat-content">
          <!-- 置顶消息 -->
          <div v-if="pinnedMessages.length > 0" class="pinned-messages">
            <a-alert type="info" closable>
              <template #icon>
                <PushpinOutlined />
              </template>
              <template #message>
                <div class="pinned-content">
                  <div v-for="item in pinnedMessages" :key="item.id" class="pinned-item">
                    <strong>{{ item.sender_nickname || 'admin' }}:</strong>
                    {{ item.content_text || item.content?.text || '' }}
                  </div>
                </div>
              </template>
            </a-alert>
          </div>

          <!-- 消息列表 + 加载更早 -->
          <div style="padding: 8px 0; text-align: center;" v-if="hasMore">
            <a-button size="small" :loading="loadingMore" @click="loadOlderMessages">加载更早消息</a-button>
          </div>
          <MessageList :messages="messages" :suppress-auto-scroll="loadingMore" />
        </a-layout-content>

        <!-- 底部输入框 -->
        <a-layout-footer class="chat-footer">
          <MessageInput @send="handleSendMessage" />
        </a-layout-footer>
      </a-layout>
    </a-layout>

    <!-- 成员列表抽屉 -->
    <a-drawer
      v-model:open="memberDrawerVisible"
      title="成员列表"
      placement="right"
      :width="360"
    >
      <MemberList :members="members" />
    </a-drawer>

    <!-- 文件管理器 -->
    <FileManager v-model:open="fileManagerVisible" />

    <!-- 用户资料 -->
    <UserProfile
      v-model:open="userProfileVisible"
      :user-id="currentUser.id"
      :is-editable="true"
      @update="handleProfileUpdate"
    />

    <!-- 设置 -->
    <Settings
      v-model:open="settingsVisible"
      @save="handleSettingsSave"
    />
  </div>
</template>

<script setup>
import { ref, reactive, h, onMounted, onUnmounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
// 事件由 App.vue 全局转发，这里监听 window 事件
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  WechatOutlined,
  CodeOutlined,
  TrophyOutlined,
  SettingOutlined,
  PoweroffOutlined,
  PushpinOutlined
} from '@ant-design/icons-vue'

import MessageList from '@/components/MessageList.vue'
import MessageInput from '@/components/MessageInput.vue'
import MemberList from '@/components/MemberList.vue'
import Headbar from '@/components/Headbar.vue'
import FileManager from '@/components/FileManager.vue'
import UserProfile from '@/components/UserProfile.vue'
import Settings from '@/components/Settings.vue'
import { useMemberStore } from '@/stores/member'
import { useChannelStore } from '@/stores/channel'

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)
const memberDrawerVisible = ref(false)
const fileManagerVisible = ref(false)
const userProfileVisible = ref(false)
const settingsVisible = ref(false)
const channelName = ref('CTF-Team-2025')

// 使用统一的频道状态管理
const channelStore = useChannelStore()
const memberStore = useMemberStore()

// 当前选中的频道ID和标签
const currentChannelID = computed(() => channelStore.selectedChannelId)
const currentChannelLabel = computed(() => {
  const channel = channelStore.selectedChannel
  return channel ? channel.name : '未知频道'
})

// 真实数据（从后端加载）
const pinnedMessages = ref([])
const messages = ref([])
const loadingMore = ref(false)
const hasMore = ref(true)
const pageSize = 200
const currentOffset = ref(0)
const members = ref([])
const lastEventAt = ref(Date.now())
let pollTimer = null

// 统一提取消息内容：兼容对象/字符串/不同字段
const extractContent = (m) => {
  try {
    let raw = m?.content ?? m?.Content
    if (typeof raw === 'string') {
      // 尝试解析JSON字符串
      try {
        const obj = JSON.parse(raw)
        if (obj && typeof obj === 'object') {
          return obj.text || obj.Text || obj.message || obj.Message || JSON.stringify(obj)
        }
      } catch {}
      return raw
    }
    if (raw && typeof raw === 'object') {
      return raw.text || raw.Text || raw.message || raw.Message || JSON.stringify(raw)
    }
    // 回退到扁平字段
    return m?.content_text || m?.ContentText || ''
  } catch {
    return ''
  }
}

import { 
  sendMessage, 
  getMessages, 
  getSubChannels, 
  getMembers,
  getMessagesByChannel,
  deleteMessage,
  pinMessage,
  unpinMessage,
  getPinnedMessages,
  reactToMessage,
  searchMessages
} from '@/api/app'

const handleSendMessage = async (messageData) => {
  const content = typeof messageData === 'string' ? messageData : messageData.content
  try {
    const channelParam = currentChannelID.value === 'main' ? null : currentChannelID.value
    await sendMessage(content, 'text', channelParam)
    // 发送成功后等待后端事件 message:received / message:updated 驱动刷新，避免同步拉取导致清空
  } catch (e) {
    message.error('发送失败: ' + (e.message || ''))
  }
}

const handleProfileUpdate = (updatedProfile) => {
  message.success('资料已更新')
  console.log('Updated profile:', updatedProfile)
}

const handleSettingsSave = (settings) => {
  message.success('设置已保存')
  console.log('Settings:', settings)
}

const showSettings = () => {
  settingsVisible.value = true
}

const goToChallenges = () => {
  router.push('/challenges')
}

const handleDisconnect = () => {
  Modal.confirm({
    title: '确认断开连接？',
    content: '断开后将退出频道，需要重新加入',
    onOk() {
      router.push('/')
    }
  })
}

onMounted(async () => {
  console.log('ChatView mounted, loading data from backend...')
  // 如果路由带有 channel 参数，切换到该子频道
  const ch = route.query.channel
  if (typeof ch === 'string' && ch) {
    channelStore.selectChannel(ch)
  }
  
  // 加载消息
  try {
    const list = await getMessages(pageSize, 0)
    console.log('Loaded messages:', list)
    if (Array.isArray(list)) {
      // 简单映射到本地结构
      list.forEach(m => messages.value.push({
        id: m.id || m.ID,
        senderId: m.sender_id || m.SenderID,
        senderName: m.sender_name || m.SenderName || m.sender_nickname || m.SenderNickname || 'user',
        content: extractContent(m),
        // 后端DTO时间字段为Unix秒，这里统一转毫秒
        timestamp: new Date(((m.edited_at || m.EditedAt || m.timestamp || m.Timestamp || 0) * 1000) || Date.now()),
        type: (m.type || m.Type || 'text')
      }))
    }
    
    if (messages.value.length === 0) {
      console.log('No messages found, showing welcome message')
      // 如果没有消息，显示欢迎消息
      messages.value.push({
        id: 'welcome',
        type: 'system',
        content: '欢迎来到 ' + channelName.value + ' 频道！',
        timestamp: new Date()
      })
    }
  } catch (e) {
    console.error('Failed to load messages:', e)
    message.warning('消息加载失败，显示本地消息')
  }
  
  // 加载成员列表
  try {
    const memberList = await getMembers()
    console.log('Loaded members:', memberList)
    if (Array.isArray(memberList)) {
      members.value = memberList.map(m => ({
        id: m.id || m.ID,
        nickname: m.nickname || m.Nickname || 'Unknown',
        role: m.role || m.Role || '队员',
        status: m.status || m.Status || 'offline',
        skills: m.skills || m.Skills || [],
        currentTask: m.current_task || m.CurrentTask || null
      }))

      // 同步到全局成员仓库，供 @提及 使用
      const storeMembers = members.value.map(m => ({
        id: m.id,
        name: m.nickname, // MentionSelector 依赖 name 字段
        nickname: m.nickname,
        role: (m.role || '').toString().toLowerCase(),
        status: (m.status || '').toString().toLowerCase(),
        online: (m.status || '').toString().toLowerCase() !== 'offline',
        skills: Array.isArray(m.skills) ? m.skills : [],
        task: m.currentTask || ''
      }))
      memberStore.setMembers(storeMembers)
    }
  } catch (e) {
    console.error('Failed to load members:', e)
    // 成员加载失败不影响使用
  }

  // 加载子频道列表（题目频道）
  try {
    const channelList = await getSubChannels()
    console.log('Loaded sub-channels:', channelList)
    if (Array.isArray(channelList)) {
      // 使用统一的频道状态管理
      channelStore.setSubChannels(channelList)
    }
  } catch (e) {
    console.error('Failed to load sub-channels:', e)
    // 子频道加载失败不影响使用（可能是客户端模式）
  }
  // 加载置顶消息
  try {
    const list = await getPinnedMessages()
    if (Array.isArray(list)) {
      pinnedMessages.value = list
    }
  } catch (e) {
    // ignore
  }

  // 监听全局转发事件：实时追加收到的消息与更新
  const handler = (ev) => {
    const detail = ev?.detail
    if (!detail || !detail.type) return
    if (detail.type !== 'message:received' && detail.type !== 'message:updated') return
    const data = detail.data || {}
    const m = data.message || data.Message || null
    if (!m) return
    const chId = m.channel_id || m.ChannelID || 'main'
    // 只处理当前频道
    const expectingMain = currentChannelID.value === 'main'
    if (!expectingMain && chId !== currentChannelID.value) return

    const normalized = {
      id: m.id || m.ID,
      senderId: m.sender_id || m.SenderID,
      senderName: m.sender_name || m.SenderName || m.sender_nickname || m.SenderNickname || 'user',
      content: extractContent(m),
      editedAt: (typeof m.edited_at === 'number' ? m.edited_at : (m.EditedAt || 0)),
      createdAt: (typeof m.timestamp === 'number' ? m.timestamp : (m.Timestamp || 0)),
      type: (m.type || m.Type || 'text')
    }

    const ts = new Date(((normalized.editedAt || normalized.createdAt) * 1000) || Date.now())
    const index = messages.value.findIndex(x => x.id === normalized.id)
    if (index >= 0) {
      // 更新已有消息
      const updated = { ...messages.value[index], senderName: normalized.senderName, content: normalized.content, type: normalized.type, timestamp: ts }
      const next = [...messages.value]
      next[index] = updated
      next.sort((a, b) => (a.timestamp?.getTime?.() || 0) - (b.timestamp?.getTime?.() || 0))
      messages.value = next
    } else {
      // 新增
      const next = [...messages.value, {
        id: normalized.id,
        senderId: normalized.senderId,
        senderName: normalized.senderName,
        content: normalized.content,
        type: normalized.type,
        timestamp: ts
      }]
      next.sort((a, b) => (a.timestamp?.getTime?.() || 0) - (b.timestamp?.getTime?.() || 0))
      messages.value = next
    }

    // 记录事件时间，用于轮询兜底判定
    lastEventAt.value = Date.now()
  }
  window.addEventListener('cw:app:event', handler)

  // 监听成员/状态/频道事件以刷新成员列表与频道标签
  const onMemberEvent = async (e) => {
    const detail = e?.detail || {}
    const t = detail.type
    if (!t) return
    if (t === 'member:joined' || t === 'member:left' || t === 'member:updated' || t === 'member:muted' || t === 'member:unmuted' || t === 'member:role_changed' || t === 'member:banned' || t === 'member:unbanned' || t === 'status:changed') {
      try {
        const memberList = await getMembers()
        if (Array.isArray(memberList)) {
          members.value = memberList.map(m => ({
            id: m.id || m.ID,
            nickname: m.nickname || m.Nickname || 'Unknown',
            role: m.role || m.Role || '队员',
            status: m.status || m.Status || 'offline',
            skills: m.skills || m.Skills || [],
            currentTask: m.current_task || m.CurrentTask || null
          }))
        }
      } catch {}
    }
  }
  window.addEventListener('cw:member:update', onMemberEvent)
  window.addEventListener('cw:connection:event', onMemberEvent)
  window.addEventListener('cw:app:event', onMemberEvent)

  // 启动轮询兜底：每5秒增量刷新一次，防止事件丢失导致不更新（尤其服务端前端）
  const refreshMessagesIncremental = async () => {
    try {
      const list = await getMessages(pageSize, 0)
      if (!Array.isArray(list)) return
      const incoming = list.map(m => ({
        id: m.id || m.ID,
        senderId: m.sender_id || m.SenderID,
        senderName: m.sender_name || m.SenderName || m.sender_nickname || m.SenderNickname || 'user',
        content: extractContent(m),
        editedAt: (typeof m.edited_at === 'number' ? m.edited_at : (m.EditedAt || 0)),
        createdAt: (typeof m.timestamp === 'number' ? m.timestamp : (m.Timestamp || 0)),
        type: (m.type || m.Type || 'text')
      }))
      // 合并到现有消息
      const map = new Map(messages.value.map(x => [x.id, x]))
      for (const n of incoming) {
        const ts = new Date(((n.editedAt || n.createdAt) * 1000) || Date.now())
        if (map.has(n.id)) {
          const prev = map.get(n.id)
          map.set(n.id, { ...prev, senderName: n.senderName, content: n.content, type: n.type, timestamp: ts })
        } else {
          map.set(n.id, { id: n.id, senderId: n.senderId, senderName: n.senderName, content: n.content, type: n.type, timestamp: ts })
        }
      }
      const merged = Array.from(map.values())
      merged.sort((a, b) => (a.timestamp?.getTime?.() || 0) - (b.timestamp?.getTime?.() || 0))
      messages.value = merged
    } catch {}
  }

  pollTimer = setInterval(async () => {
    // 若超过3秒未收到事件，则执行一次增量刷新
    if (Date.now() - lastEventAt.value > 3000) {
      await refreshMessagesIncremental()
    }
  }, 5000)

  unsubscribeEvent.value = () => {
    window.removeEventListener('cw:app:event', handler)
    window.removeEventListener('cw:member:update', onMemberEvent)
    window.removeEventListener('cw:connection:event', onMemberEvent)
    window.removeEventListener('cw:app:event', onMemberEvent)
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }
})

const unsubscribeEvent = ref(() => {})
onUnmounted(() => {
  try { unsubscribeEvent.value && unsubscribeEvent.value() } catch {}
})

const handleChannelSelect = async (channelId) => {
  // 使用统一的频道状态管理
  channelStore.selectChannel(channelId)
  // 重新加载该频道的消息
  try {
    await reloadChannelMessages()
  } catch (e) {
    message.warning('加载该频道消息失败')
  }
}

const reloadChannelMessages = async () => {
  let list = []
  const channelId = currentChannelID.value
  const useByChannel = channelId !== 'main'
  try {
    list = useByChannel ? await getMessagesByChannel(channelId, pageSize, 0) : await getMessages(pageSize, 0)
  } catch (e) {
    list = await getMessages(pageSize, 0)
  }
  messages.value = []
  if (Array.isArray(list)) {
    const normalized = list.map(m => ({
      id: m.id || m.ID,
      senderId: m.sender_id || m.SenderID,
      senderName: m.sender_name || m.SenderName || m.sender_nickname || m.SenderNickname || 'user',
      content: extractContent(m),
      editedAt: (typeof m.edited_at === 'number' ? m.edited_at : (m.EditedAt || 0)),
      createdAt: (typeof m.timestamp === 'number' ? m.timestamp : (m.Timestamp || 0)),
      type: (m.type || m.Type || 'text')
    }))
    // 前端再按 editedAt 升序兜底
    normalized.sort((a, b) => (a.editedAt || a.createdAt) - (b.editedAt || b.createdAt))
    messages.value = normalized.map(n => ({
      id: n.id,
      senderId: n.senderId,
      senderName: n.senderName,
      content: n.content,
      type: n.type,
      timestamp: new Date(((n.editedAt || n.createdAt) * 1000) || Date.now())
    }))
  }
}

// 加载更早消息（下一页）
const loadOlderMessages = async () => {
  if (loadingMore.value) return
  loadingMore.value = true
  try {
    const channelId = currentChannelID.value
    const useByChannel = channelId !== 'main'
    const nextOffset = (messages.value?.length || 0)
    let list = []
    try {
      list = useByChannel ? await getMessagesByChannel(channelId, pageSize, nextOffset) : await getMessages(pageSize, nextOffset)
    } catch (e) {
      list = []
    }
    // 规范化
    const normalized = (Array.isArray(list) ? list : []).map(m => ({
      id: m.id || m.ID,
      senderId: m.sender_id || m.SenderID,
      senderName: m.sender_name || m.SenderName || m.sender_nickname || m.SenderNickname || 'user',
      content: extractContent(m),
      editedAt: (typeof m.edited_at === 'number' ? m.edited_at : (m.EditedAt || 0)),
      createdAt: (typeof m.timestamp === 'number' ? m.timestamp : (m.Timestamp || 0)),
      type: (m.type || m.Type || 'text')
    }))
    // 合并去重（旧消息在前，保证时间升序）
    const merged = [...normalized, ...messages.value.map(n => ({
      id: n.id,
      senderId: n.senderId,
      senderName: n.senderName,
      content: n.content,
      type: n.type,
      editedAt: Math.floor((n.timestamp?.getTime?.() || Date.now()) / 1000),
      createdAt: Math.floor((n.timestamp?.getTime?.() || Date.now()) / 1000),
    }))]
    const uniqueMap = new Map()
    merged.forEach(n => {
      if (!uniqueMap.has(n.id)) uniqueMap.set(n.id, n)
    })
    const uniq = Array.from(uniqueMap.values())
    uniq.sort((a, b) => (a.editedAt || a.createdAt) - (b.editedAt || b.createdAt))
    messages.value = uniq.map(n => ({
      id: n.id,
      senderId: n.senderId,
      senderName: n.senderName,
      content: n.content,
      type: n.type,
      timestamp: new Date(((n.editedAt || n.createdAt) * 1000) || Date.now())
    }))
    // hasMore：如果返回数量小于pageSize，说明没有更多
    hasMore.value = (Array.isArray(list) && list.length === pageSize)
    currentOffset.value = nextOffset
  } finally {
    loadingMore.value = false
  }
}
</script>

<style scoped>
.chat-container {
  width: 100%;
  height: 100vh;
  background-color: #f5f5f5;
}

.chat-sider {
  background: white;
  border-right: 1px solid #f0f0f0;
}

.channel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
  height: 64px;
}

.channel-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.trigger {
  font-size: 18px;
  cursor: pointer;
  transition: color 0.3s;
}

.trigger:hover {
  color: #1890ff;
}

.sider-footer {
  position: absolute;
  bottom: 0;
  width: 100%;
  padding: 16px;
  border-top: 1px solid #f0f0f0;
  background: white;
}

.chat-header {
  background: white;
  padding: 0 16px;
  border-bottom: 1px solid #f0f0f0;
  height: 64px;
}

.header-container {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  height: 100%;
  max-width: 100%;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
  min-width: 0;
}

.header-center {
  flex: 1;
  max-width: 600px;
  min-width: 200px;
  padding: 0 16px;
}

.header-right {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.current-channel {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 响应式布局 */
@media (max-width: 1200px) {
  .header-container {
    gap: 12px;
  }
  
  .header-center {
    max-width: 400px;
    padding: 0 12px;
  }
  
  .chat-header {
    padding: 0 12px;
  }
}

@media (max-width: 992px) {
  .header-center {
    max-width: 300px;
    min-width: 150px;
    padding: 0 8px;
  }
  
  .current-channel {
    font-size: 16px;
  }
}

@media (max-width: 768px) {
  .header-container {
    gap: 8px;
  }
  
  .header-left {
    flex: 1;
    min-width: 0;
  }
  
  .header-center {
    position: absolute;
    left: 50%;
    transform: translateX(-50%);
    max-width: 250px;
    min-width: 120px;
    padding: 0;
    z-index: 10;
  }
  
  .header-right {
    position: relative;
    z-index: 11;
  }
  
  .chat-header {
    padding: 0 8px;
  }
}

@media (max-width: 576px) {
  .header-center {
    display: none; /* 在很小的屏幕上隐藏搜索框 */
  }
  
  .current-channel {
    font-size: 14px;
    max-width: 120px;
  }
}

.chat-content {
  background: white;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  padding: 0 16px;
}

.pinned-messages {
  padding: 16px;
  margin: 16px 0;
  border-bottom: 1px solid #f0f0f0;
}

.pinned-content {
  font-size: 14px;
}

.chat-footer {
  background: white;
  padding: 16px;
  border-top: 1px solid #f0f0f0;
}

/* 响应式内容区域 */
@media (max-width: 768px) {
  .chat-content {
    padding: 0 8px;
  }
  
  .pinned-messages {
    padding: 12px;
    margin: 12px 0;
  }
  
  .chat-footer {
    padding: 12px 8px;
  }
}

@media (max-width: 576px) {
  .chat-content {
    padding: 0 4px;
  }
  
  .pinned-messages {
    padding: 8px;
    margin: 8px 0;
  }
  
  .chat-footer {
    padding: 8px 4px;
  }
}
</style>

