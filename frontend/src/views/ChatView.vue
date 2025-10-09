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
          v-model:selectedKeys="selectedChannels"
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

          <a-menu-divider />

          <a-menu-item key="challenges" @click="goToChallenges">
            <template #icon>
              <TrophyOutlined />
            </template>
            <span>题目管理</span>
          </a-menu-item>

          <a-menu-divider />

          <!-- 题目频道列表 -->
          <a-menu-item-group v-if="!collapsed && subChannels.length > 0" title="题目频道">
            <a-menu-item v-for="channel in subChannels" :key="channel.id" @click="handleChannelSelect(channel.id)">
              <template #icon>
                <CodeOutlined />
              </template>
              <span>{{ channel.name }}</span>
              <a-badge v-if="channel.message_count > 0" :count="channel.message_count" :offset="[10, 0]" />
            </a-menu-item>
          </a-menu-item-group>
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
        <!-- 顶部栏 -->
        <a-layout-header class="chat-header">
          <Toolbar :gap="12" :height="64" padding-x="16px" background="#fff" :border="true">
            <template #left>
              <h3 class="current-channel">{{ currentChannelLabel }}</h3>
              <a-tag color="green">
                <CheckCircleOutlined /> 已连接
              </a-tag>
            </template>
            <template #center>
              <div class="header-center">
                <SearchBar />
              </div>
            </template>
            <template #right>
              <a-space :size="8">
                <NotificationCenter />
                <a-tooltip title="文件管理">
                  <a-button type="text" :icon="h(FileOutlined)" @click="fileManagerVisible = true" />
                </a-tooltip>
                <a-tooltip title="成员列表">
                  <a-button type="text" :icon="h(TeamOutlined)" @click="memberDrawerVisible = true" />
                </a-tooltip>
                <a-dropdown>
                  <a-avatar style="cursor: pointer; background-color: #1890ff" @click="userProfileVisible = true">
                    {{ currentUser.name[0].toUpperCase() }}
                  </a-avatar>
                  <template #overlay>
                    <a-menu>
                      <a-menu-item @click="userProfileVisible = true">
                        <UserOutlined /> 个人资料
                      </a-menu-item>
                      <a-menu-item @click="settingsVisible = true">
                        <SettingOutlined /> 设置
                      </a-menu-item>
                      <a-menu-divider />
                      <a-menu-item danger @click="handleDisconnect">
                        <PoweroffOutlined /> 断开连接
                      </a-menu-item>
                    </a-menu>
                  </template>
                </a-dropdown>
              </a-space>
            </template>
          </Toolbar>
        </a-layout-header>

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

          <!-- 消息列表 -->
          <MessageList :messages="messages" />
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
import { ref, reactive, h, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  WechatOutlined,
  CodeOutlined,
  BugOutlined,
  TrophyOutlined,
  SettingOutlined,
  PoweroffOutlined,
  CheckCircleOutlined,
  SearchOutlined,
  FileOutlined,
  TeamOutlined,
  PushpinOutlined,
  UserOutlined
} from '@ant-design/icons-vue'

import MessageList from '@/components/MessageList.vue'
import MessageInput from '@/components/MessageInput.vue'
import MemberList from '@/components/MemberList.vue'
import SearchBar from '@/components/SearchBar.vue'
import NotificationCenter from '@/components/NotificationCenter.vue'
import FileManager from '@/components/FileManager.vue'
import UserProfile from '@/components/UserProfile.vue'
import Settings from '@/components/Settings.vue'
import Toolbar from '@/components/Common/Toolbar.vue'

const router = useRouter()
const collapsed = ref(false)
const selectedChannels = ref(['main'])
const currentChannelID = ref('main')
const currentChannelLabel = computed(() => currentChannelID.value === 'main' ? '主频道' : (subChannels.value.find(ch => ch.id === currentChannelID.value)?.name || '子频道'))
const memberDrawerVisible = ref(false)
const fileManagerVisible = ref(false)
const userProfileVisible = ref(false)
const settingsVisible = ref(false)
const channelName = ref('CTF-Team-2025')

// 当前用户信息
const currentUser = ref({
  id: 'me',
  name: 'admin',
  email: 'admin@example.com',
  role: 'admin'
})

// 真实数据（从后端加载）
const pinnedMessages = ref([])
const messages = ref([])
const members = ref([])
const subChannels = ref([])

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
  // messageData 可能是字符串（旧版）或对象（包含 content 和 mentions）
  const content = typeof messageData === 'string' ? messageData : messageData.content
  const mentions = typeof messageData === 'object' ? messageData.mentions : []
  
  const newMessage = {
    id: Date.now().toString(),
    senderId: 'me',
    senderName: currentUser.value.name,
    senderAvatar: '',
    content: content,
    mentions: mentions,
    timestamp: new Date(),
    type: 'text'
  }
  
  messages.value.push(newMessage)
  
  try {
    const channelParam = currentChannelID.value === 'main' ? null : currentChannelID.value
    await sendMessage(content, 'text', channelParam)
  } catch (e) {
    // 发送失败提示
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
  
  // 加载消息
  try {
    const list = await getMessages(50, 0)
    console.log('Loaded messages:', list)
    if (Array.isArray(list)) {
      // 简单映射到本地结构
      list.forEach(m => messages.value.push({
        id: m.id || m.ID,
        senderId: m.sender_id || m.SenderID,
        senderName: m.sender_name || m.SenderName || 'user',
        content: (m.content && (m.content.text || m.content.Text)) || m.content_text || m.Content || '',
        timestamp: new Date(m.timestamp || m.Timestamp || Date.now()),
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
      subChannels.value = channelList.map(ch => ({
        id: ch.id || ch.ID,
        name: ch.name || ch.Name,
        message_count: ch.message_count || ch.MessageCount || 0
      }))
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
})

const handleChannelSelect = async (channelId) => {
  currentChannelID.value = channelId
  selectedChannels.value = [channelId]
  // 重新加载该频道的消息
  try {
    let list = []
    try {
      list = await getMessagesByChannel(channelId, 50, 0)
    } catch (e) {
      // 兼容：如果未生成绑定，回退到默认接口
      list = await getMessages(50, 0)
    }
    messages.value = []
    if (Array.isArray(list)) {
      list.forEach(m => messages.value.push({
        id: m.id || m.ID,
        senderId: m.sender_id || m.SenderID,
        senderName: m.sender_name || m.SenderName || 'user',
        content: (m.content && (m.content.text || m.content.Text)) || m.content_text || m.Content || '',
        timestamp: new Date(m.timestamp || m.Timestamp || Date.now()),
        type: (m.type || m.Type || 'text')
      }))
    }
  } catch (e) {
    message.warning('加载该频道消息失败')
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

