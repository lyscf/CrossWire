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
          <a-menu-item key="main">
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

          <a-menu-item-group v-if="!collapsed" title="题目频道">
            <a-menu-item key="web-100">
              <template #icon>
                <CodeOutlined />
              </template>
              <span>Web-100</span>
              <a-badge :count="3" :offset="[10, 0]" />
            </a-menu-item>
            <a-menu-item key="pwn-200">
              <template #icon>
                <BugOutlined />
              </template>
              <span>Pwn-200</span>
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
          <div class="header-container">
            <div class="header-left">
              <h3 class="current-channel">主频道</h3>
              <a-tag color="green">
                <CheckCircleOutlined /> 已连接
              </a-tag>
            </div>

            <div class="header-center">
              <!-- 全局搜索 -->
              <SearchBar />
            </div>

            <div class="header-right">
              <a-space :size="8">
                <!-- 通知中心 -->
                <NotificationCenter />

                <!-- 文件管理 -->
                <a-tooltip title="文件管理">
                  <a-button 
                    type="text" 
                    :icon="h(FileOutlined)" 
                    @click="fileManagerVisible = true"
                  />
                </a-tooltip>

                <!-- 成员列表 -->
                <a-tooltip title="成员列表">
                  <a-button
                    type="text"
                    :icon="h(TeamOutlined)"
                    @click="memberDrawerVisible = true"
                  />
                </a-tooltip>

                <!-- 用户头像 -->
                <a-dropdown>
                  <a-avatar 
                    style="cursor: pointer; background-color: #1890ff"
                    @click="userProfileVisible = true"
                  >
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
            </div>
          </div>
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
                  <strong>admin:</strong> {{ pinnedMessages[0].content }}
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
import { ref, reactive, h, onMounted } from 'vue'
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

const router = useRouter()
const collapsed = ref(false)
const selectedChannels = ref(['main'])
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

// 模拟数据
const pinnedMessages = ref([
  {
    id: '1',
    content: '比赛规则：禁止攻击基础设施，发现漏洞请及时提交'
  }
])

const messages = ref([
  {
    id: '1',
    type: 'system',
    content: 'alice 加入了频道',
    timestamp: new Date(Date.now() - 3600000)
  },
  {
    id: '2',
    senderId: 'user1',
    senderName: 'alice',
    senderAvatar: '',
    content: '大家好！我擅长 Web 方向',
    timestamp: new Date(Date.now() - 3000000),
    type: 'text'
  },
  {
    id: '3',
    senderId: 'user2',
    senderName: 'bob',
    senderAvatar: '',
    content: '我来做 Pwn 题',
    timestamp: new Date(Date.now() - 2000000),
    type: 'text'
  }
])

const members = ref([
  {
    id: 'user1',
    nickname: 'alice',
    role: '队长',
    status: 'online',
    skills: ['Web', 'Crypto'],
    currentTask: 'Web-100'
  },
  {
    id: 'user2',
    nickname: 'bob',
    role: '队员',
    status: 'busy',
    skills: ['Pwn', 'Reverse'],
    currentTask: 'Pwn-200'
  },
  {
    id: 'user3',
    nickname: 'charlie',
    role: '队员',
    status: 'online',
    skills: ['Misc'],
    currentTask: null
  }
])

const handleSendMessage = (messageData) => {
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
  
  // TODO: 调用 Wails API 发送消息
  // await SendMessage(content, mentions)
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

onMounted(() => {
  // TODO: 初始化消息监听
  // InitializeMessageListener()
})
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

