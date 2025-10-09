<template>
  <div class="search-bar">
    <SearchInput
      v-model="searchText"
      placeholder="搜索消息、题目、成员... (Ctrl+K)"
      size="large"
      :show-dropdown="showResults && searchText.length > 0"
      global-shortcut="ctrl+k"
      :shortcuts="[
        { key: 'Esc', label: '关闭' },
        { key: '↑↓', label: '导航' },
        { key: 'Enter', label: '打开' }
      ]"
      :show-shortcuts="false"
      @focus="showResults = true"
      @change="handleSearch"
      class="search-input"
    >
      <template #dropdown>

        <div class="search-results-content">
      <a-tabs v-model:activeKey="activeTab" size="small">
        <!-- 消息搜索 -->
        <a-tab-pane key="messages" tab="消息">
          <template #tab>
            <MessageOutlined /> 消息 ({{ messageResults.length }})
          </template>
          <a-list
            :data-source="messageResults"
            size="small"
            :loading="searching"
          >
            <template #renderItem="{ item }">
              <a-list-item
                class="search-result-item"
                @click="goToMessage(item)"
              >
                <a-list-item-meta>
                  <template #avatar>
                    <AvatarChip :name="item.sender" />
                  </template>
                  <template #title>
                    <span class="sender-name">{{ item.sender }}</span>
                    <span class="message-time">{{ formatTime(item.timestamp) }}</span>
                  </template>
                  <template #description>
                    <div class="message-preview" v-html="highlightText(item.content)"></div>
                  </template>
                </a-list-item-meta>
              </a-list-item>
            </template>
            <template #locale>
              <a-empty description="未找到相关消息" />
            </template>
          </a-list>
        </a-tab-pane>

        <!-- 题目搜索 -->
        <a-tab-pane key="challenges" tab="题目">
          <template #tab>
            <TrophyOutlined /> 题目 ({{ challengeResults.length }})
          </template>
          <a-list
            :data-source="challengeResults"
            size="small"
            :loading="searching"
          >
            <template #renderItem="{ item }">
              <a-list-item
                class="search-result-item"
                @click="goToChallenge(item)"
              >
                <a-list-item-meta>
                  <template #title>
                    <span v-html="highlightText(item.title)"></span>
                    <a-tag :color="getCategoryColor(item.category)" size="small">
                      {{ item.category }}
                    </a-tag>
                  </template>
                  <template #description>
                    <div class="challenge-info">
                      <span>{{ item.points }} 分</span>
                      <a-divider type="vertical" />
                      <span v-html="highlightText(item.description)"></span>
                    </div>
                  </template>
                </a-list-item-meta>
              </a-list-item>
            </template>
            <template #locale>
              <a-empty description="未找到相关题目" />
            </template>
          </a-list>
        </a-tab-pane>

        <!-- 成员搜索 -->
        <a-tab-pane key="members" tab="成员">
          <template #tab>
            <UserOutlined /> 成员 ({{ memberResults.length }})
          </template>
          <a-list
            :data-source="memberResults"
            size="small"
            :loading="searching"
          >
            <template #renderItem="{ item }">
              <a-list-item
                class="search-result-item"
                @click="goToMember(item)"
              >
                <a-list-item-meta>
                  <template #avatar>
                    <AvatarChip :name="item.name" :status="item.online ? 'online' : 'offline'" />
                  </template>
                  <template #title>
                    <span v-html="highlightText(item.name)"></span>
                  </template>
                  <template #description>
                    <a-space size="small" wrap>
                      <a-tag
                        v-for="skill in item.skills"
                        :key="skill"
                        size="small"
                        :color="getSkillColor(skill)"
                      >
                        {{ skill }}
                      </a-tag>
                    </a-space>
                  </template>
                </a-list-item-meta>
              </a-list-item>
            </template>
            <template #locale>
              <a-empty description="未找到相关成员" />
            </template>
          </a-list>
        </a-tab-pane>
      </a-tabs>

        </div>
      </template>
    </SearchInput>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import {
  MessageOutlined,
  TrophyOutlined,
  UserOutlined
} from '@ant-design/icons-vue'
import SearchInput from '@/components/Common/SearchInput.vue'
import AvatarChip from '@/components/Common/AvatarChip.vue'
import { useMessageStore } from '@/stores/message'
import { useChallengeStore } from '@/stores/challenge'
import { useMemberStore } from '@/stores/member'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'

const router = useRouter()
const messageStore = useMessageStore()
const challengeStore = useChallengeStore()
const memberStore = useMemberStore()

const searchText = ref('')
const showResults = ref(false)
const activeTab = ref('messages')
const searching = ref(false)

// 搜索结果
const messageResults = computed(() => {
  if (!searchText.value) return []
  const query = searchText.value.toLowerCase()
  return messageStore.messages.filter(msg => 
    msg.content.toLowerCase().includes(query) ||
    msg.senderName.toLowerCase().includes(query)
  ).slice(0, 10)
})

const challengeResults = computed(() => {
  if (!searchText.value) return []
  const query = searchText.value.toLowerCase()
  return challengeStore.challenges.filter(ch => 
    ch.title.toLowerCase().includes(query) ||
    ch.description.toLowerCase().includes(query) ||
    ch.category.toLowerCase().includes(query)
  ).slice(0, 10)
})

const memberResults = computed(() => {
  if (!searchText.value) return []
  const query = searchText.value.toLowerCase()
  return memberStore.members.filter(m => 
    m.name.toLowerCase().includes(query) ||
    m.skills.some(s => s.toLowerCase().includes(query))
  ).slice(0, 10)
})

const handleSearch = () => {
  if (!searchText.value) return
  searching.value = true
  setTimeout(() => {
    searching.value = false
  }, 300)
}

// 高亮搜索文本
const highlightText = (text) => {
  if (!searchText.value || !text) return text
  
  const escapeHtml = (str) => {
    const div = document.createElement('div')
    div.textContent = str
    return div.innerHTML
  }
  
  const regex = new RegExp(`(${searchText.value})`, 'gi')
  return escapeHtml(text).replace(regex, '<mark>$1</mark>')
}

const formatTime = (timestamp) => {
  return dayjs(timestamp).format('MM-DD HH:mm')
}


const getCategoryColor = (category) => {
  const colors = {
    Web: 'blue',
    Pwn: 'red',
    Reverse: 'purple',
    Crypto: 'orange',
    Misc: 'green'
  }
  return colors[category] || 'default'
}

const getSkillColor = (skill) => {
  return getCategoryColor(skill)
}

// 导航到结果
const goToMessage = (message) => {
  router.push({ name: 'chat', query: { messageId: message.id } })
  closeSearch()
}

const goToChallenge = (challenge) => {
  router.push({ name: 'challenges', query: { id: challenge.id } })
  closeSearch()
}

const goToMember = (member) => {
  // 显示成员资料
  console.log('Show member profile:', member)
  closeSearch()
}

const closeSearch = () => {
  showResults.value = false
  searchText.value = ''
}
</script>

<style scoped>
.search-bar {
  position: relative;
  width: 100%;
  max-width: 600px;
  /* 移除上下 padding，避免与左侧标题/状态垂直不对齐 */
}

.search-input {
  width: 100%;
}

.search-results-content {
  max-height: 500px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.search-result-item {
  cursor: pointer;
  transition: background-color 0.2s;
}

.search-result-item:hover {
  background-color: #f5f5f5;
}

.sender-name {
  font-weight: 500;
  margin-right: var(--spacing-sm);
}

.message-time {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.message-preview {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.challenge-info {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.65);
}

:deep(mark) {
  background-color: #ffe58f;
  padding: 0 2px;
  border-radius: 2px;
}

/* 结果列表样式 */
.search-results-content :deep(.ant-tabs) {
  height: 100%;
}

.search-results-content :deep(.ant-tabs-content) {
  max-height: 400px;
  overflow-y: auto;
}
</style>

